package reconciler

import (
	"context"
	"fmt"
	"time"

	"github.com/3scale-ops/basereconciler/resource"
	"github.com/3scale-ops/basereconciler/util"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type action string

const (
	ContinueAction         action = "Continue"
	ReturnAction           action = "Return"
	ReturnAndRequeueAction action = "ReturnAndRequeue"
)

type Result struct {
	Action       action
	RequeueAfter time.Duration
	Error        error
}

func (result Result) ShouldReturn() bool {
	return result.Action == ReturnAction || result.Action == ReturnAndRequeueAction || result.Error != nil
}

func (result Result) Values() (ctrl.Result, error) {

	return ctrl.Result{
			Requeue:      func() bool { return result.Action == ReturnAndRequeueAction }(),
			RequeueAfter: result.RequeueAfter,
		},
		result.Error
}

var options = struct {
	finalizer         *string
	finalizationLogic []finalizationFunction
}{
	finalizer:         nil,
	finalizationLogic: []finalizationFunction{},
}

// LifecycleOption is an interface that defines options that can be passed to
// the reconciler's ManageResourceLifecycle() function
type LifecycleOption interface {
	applyToLifecycleOptions()
}

type finalizer string

func (f finalizer) applyToLifecycleOptions() {
	options.finalizer = util.Pointer(string(f))
}
func WithFinalizer(f string) finalizer {
	return finalizer(f)
}

type finalizationFunction func(context.Context, client.Client) error

func (fn finalizationFunction) applyToLifecycleOptions() {
	options.finalizationLogic = append(options.finalizationLogic, fn)
}

func WithFinalizationFunc(fn func(context.Context, client.Client) error) finalizationFunction {
	return fn
}

// Reconciler computes a list of resources that it needs to keep in place
type Reconciler struct {
	client.Client
	Log         logr.Logger
	Scheme      *runtime.Scheme
	gvk         schema.GroupVersionKind
	typeTracker typeTracker
}

// NewFromManager returns a new Reconciler from a controller-runtime manager.Manager
func NewFromManager(mgr manager.Manager) *Reconciler {
	return &Reconciler{Client: mgr.GetClient(), Scheme: mgr.GetScheme(), Log: logr.Discard()}
}

func (r *Reconciler) WithGVK(apiVersion, kind string) *Reconciler {
	r.gvk = schema.FromAPIVersionAndKind(apiVersion, kind)
	return r
}

// WithLogger sets the Reconciler logger
func (r *Reconciler) WithLogger(logger logr.Logger) *Reconciler {
	r.Log = logger
	return r
}

// Logger returns the Reconciler logger and a copy of the context that also includes the logger inside to pass it around easily.
func (r *Reconciler) Logger(ctx context.Context, keysAndValues ...interface{}) (context.Context, logr.Logger) {
	var logger logr.Logger
	if !r.Log.IsZero() {
		// get the logger configured in the Reconciler
		logger = r.Log.WithValues(keysAndValues...)
	} else {
		// try to get a logger from the context
		logger = logr.FromContextOrDiscard(ctx).WithValues(keysAndValues...)
	}
	return logr.NewContext(ctx, logger), logger
}

// ManageResourceLifecycle manages the lifecycle of the resource, from initialization to
// finalization and deletion.
// The behaviour can be modified depending on the options passed to the function:
//   - finalizer: if a non-nil finalizer is passed to the function, it will ensure that the
//     custom resource has a finalizer in place, updasting it if required.
//   - cleanupFns: variadic parameter that allows passing cleanup functions that will be
//     run when the custom resource is being deleted. Only works with a non-nil finalizer, otherwise
//     the custom resource will be immediately deleted and the functions won't run.
func (r *Reconciler) ManageResourceLifecycle(ctx context.Context, req reconcile.Request, obj client.Object,
	opts ...LifecycleOption) Result {

	for _, o := range opts {
		o.applyToLifecycleOptions()
	}

	ctx, logger := r.Logger(ctx)
	err := r.Client.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, obj)
	if err != nil {
		if errors.IsNotFound(err) {
			// Return and don't requeue
			return Result{Action: ReturnAction}
		}
		return Result{Error: err}
	}

	if util.IsBeingDeleted(obj) {

		// finalizer logic is only triggered if the controller
		// sets a finalizer and the finalizer is still present in the
		// resource
		if options.finalizer != nil && controllerutil.ContainsFinalizer(obj, *options.finalizer) {

			err := r.Finalize(ctx, options.finalizationLogic, logger)
			if err != nil {
				logger.Error(err, "unable to delete instance")
				return Result{Error: err}
			}
			controllerutil.RemoveFinalizer(obj, *options.finalizer)
			err = r.Client.Update(ctx, obj)
			if err != nil {
				logger.Error(err, "unable to update instance")
				return Result{Error: err}
			}

		}
		// object being deleted, return without doing anything
		// and stop the reconcile loop
		return Result{Action: ReturnAction}
	}

	if ok := r.IsInitialized(obj, options.finalizer); !ok {
		err := r.Client.Update(ctx, obj)
		if err != nil {
			logger.Error(err, "unable to initialize instance")
			return Result{Error: err}
		}
		return Result{Action: ReturnAndRequeueAction}
	}
	return Result{Action: ContinueAction}
}

// IsInitialized can be used to check if instance is correctly initialized.
// Returns false if it isn't.
func (r *Reconciler) IsInitialized(obj client.Object, finalizer *string) bool {
	ok := true
	if finalizer != nil && !controllerutil.ContainsFinalizer(obj, *finalizer) {
		controllerutil.AddFinalizer(obj, *finalizer)
		ok = false
	}

	return ok
}

// Finalize contains finalization logic for the Reconciler
func (r *Reconciler) Finalize(ctx context.Context, fns []finalizationFunction, log logr.Logger) error {
	// Call any cleanup functions passed
	for _, fn := range fns {
		err := fn(ctx, r.Client)
		if err != nil {
			return err
		}
	}
	return nil
}

// ReconcileOwnedResources handles generalized resource reconcile logic for a controller:
//
//   - Takes a list of templates and calls resource.CreateOrUpdate on each one of them. The templates
//     need to implement the resource.TemplateInterface interface. Users can take advantage of the generic
//     resource.Template[T] struct that the resource package provides, which already implements the
//     resource.TemplateInterface.
//   - Each template is added to the list of managed resources if resource.CreateOrUpdate returns with no error
//   - If the resource pruner is enabled any resource owned by the custom resource not present in the list of managed
//     resources is deleted. The resource pruner must be enabled in the global config (see package config) and also not
//     explicitely disabled in the resource by the '<annotations-domain>/prune: true/false' annotation.
func (r *Reconciler) ReconcileOwnedResources(ctx context.Context, owner client.Object, list []resource.TemplateInterface) Result {
	managedResources := []corev1.ObjectReference{}

	for _, template := range list {
		ref, err := resource.CreateOrUpdate(ctx, r.Client, r.Scheme, owner, template)
		if err != nil {
			return Result{Error: fmt.Errorf("unable to CreateOrUpdate resource: %w", err)}
		}
		if ref != nil {
			managedResources = append(managedResources, *ref)
			r.typeTracker.trackType(schema.FromAPIVersionAndKind(ref.APIVersion, ref.Kind))
		}
	}

	if isPrunerEnabled(owner) {
		if err := r.pruneOrphaned(ctx, owner, managedResources); err != nil {

			return Result{Error: fmt.Errorf("unable to prune orphaned resources: %w", err)}
		}
	}

	return Result{Action: ContinueAction}
}

// SecretEventHandler returns an EventHandler for the specific client.ObjectList
// list object passed as parameter
// TODO: generalize this to watch any object type
func (r *Reconciler) SecretEventHandler(ol client.ObjectList, logger logr.Logger) handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(
		func(o client.Object) []reconcile.Request {
			if err := r.Client.List(context.TODO(), ol); err != nil {
				logger.Error(err, "unable to retrieve the list of resources")
				return []reconcile.Request{}
			}
			items := util.GetItems(ol)
			if len(items) == 0 {
				return []reconcile.Request{}
			}

			// This is a bit undiscriminate as we don't have a way to detect which
			// resources are interested in the event, so we just wake them all up
			// TODO: pass a function that can decide if the event is of interest for a given resource
			req := make([]reconcile.Request, 0, len(items))
			for _, item := range items {
				req = append(req, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(item)})
			}
			return req
		},
	)
}
