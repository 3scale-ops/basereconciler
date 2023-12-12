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

type Result struct {
	Requeue      bool
	RequeueAfter time.Duration
	Error        error
}

func (result Result) IsReturnAndRequeue() bool {
	return result.Requeue || result.Error != nil
}

func (result Result) Values() (ctrl.Result, error) {
	return ctrl.Result{
			Requeue:      result.Requeue,
			RequeueAfter: result.RequeueAfter,
		},
		result.Error
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

// GetInstance tries to retrieve the custom resource instance and perform some standard
// tasks like initialization and cleanup. The behaviour can be modified depending on the
// parameters passed to the function:
//   - finalizer: if a non-nil finalizer is passed to the function, it will ensure that the
//     custom resource has a finalizer in place, updasting it if required.
//   - cleanupFns: variadic parameter that allows passing cleanup functions that will be
//     run when the custom resource is being deleted. Only works with a non-nil finalizer, otherwise
//     the custom resource will be immediately deleted and the functions won't run.
func (r *Reconciler) GetInstance(ctx context.Context, req reconcile.Request, obj client.Object,
	finalizer *string, cleanupFns ...func()) Result {

	ctx, logger := r.Logger(ctx)
	err := r.Client.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, obj)
	if err != nil {
		if errors.IsNotFound(err) {
			// Return and don't requeue
			return Result{Requeue: false, Error: nil}
		}
		return Result{Requeue: false, Error: err}
	}

	if util.IsBeingDeleted(obj) {

		// finalizer logic is only triggered if the controller
		// sets a finalizer and the finalizer is still present in the
		// resource
		if finalizer != nil && controllerutil.ContainsFinalizer(obj, *finalizer) {

			err := r.ManageCleanupLogic(obj, cleanupFns, logger)
			if err != nil {
				logger.Error(err, "unable to delete instance")
				return Result{Requeue: false, Error: err}
			}
			controllerutil.RemoveFinalizer(obj, *finalizer)
			err = r.Client.Update(ctx, obj)
			if err != nil {
				logger.Error(err, "unable to update instance")
				return Result{Requeue: false, Error: err}
			}

		}
		// no finalizer, just return without doing anything
		return Result{Requeue: false, Error: nil}
	}

	if ok := r.IsInitialized(obj, finalizer); !ok {
		err := r.Client.Update(ctx, obj)
		if err != nil {
			logger.Error(err, "unable to initialize instance")
			return Result{Requeue: false, Error: err}
		}
		return Result{Requeue: true, Error: nil}
	}
	return Result{Requeue: false, Error: nil}
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

// ManageCleanupLogic contains finalization logic for the Reconciler
func (r *Reconciler) ManageCleanupLogic(obj client.Object, fns []func(), log logr.Logger) error {
	// Call any cleanup functions passed
	for _, fn := range fns {
		fn()
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
			return Result{
				Requeue: false,
				Error:   fmt.Errorf("unable to CreateOrUpdate resource: %w", err),
			}
		}
		if ref != nil {
			managedResources = append(managedResources, *ref)
			r.typeTracker.trackType(schema.FromAPIVersionAndKind(ref.APIVersion, ref.Kind))
		}
	}

	if isPrunerEnabled(owner) {
		if err := r.pruneOrphaned(ctx, owner, managedResources); err != nil {

			return Result{
				Requeue: false,
				Error:   fmt.Errorf("unable to prune orphaned resources: %w", err),
			}
		}
	}

	return Result{Requeue: false, Error: nil}
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
