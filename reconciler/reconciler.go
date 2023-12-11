package reconciler

import (
	"context"
	"fmt"

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

// Reconciler computes a list of resources that it needs to keep in place
type Reconciler struct {
	client.Client
	Log         logr.Logger
	Scheme      *runtime.Scheme
	typeTracker typeTracker
}

// NewFromManager returns a new Reconciler from a controller-runtime manager.Manager
func NewFromManager(mgr manager.Manager) *Reconciler {
	return &Reconciler{Client: mgr.GetClient(), Scheme: mgr.GetScheme(), Log: logr.Discard()}
}

func (r *Reconciler) WithLogger(logger logr.Logger) *Reconciler {
	r.Log = logger
	return r
}

func (r *Reconciler) GetLogger(ctx context.Context) logr.Logger {
	if logger, err := logr.FromContext(ctx); err != nil {
		return r.Log
	} else {
		return logger
	}
}

func (r *Reconciler) SetLogger(ctx *context.Context, keysAndValues ...interface{}) logr.Logger {
	logger := r.GetLogger(*ctx).WithValues(keysAndValues)
	*ctx = logr.NewContext(*ctx, logger)
	return logger
}

// GetInstance tries to retrieve the custom resource instance and perform some standard
// tasks like initialization and cleanup. The behaviour can be modified depending on the
// parameters passed to the function:
//   - finalizer: if a non-nil finalizer is passed to the function, it will ensure that the
//     custom resource has a finalizer in place, updasting it if required.
//   - cleanupFns: variadic parameter that allows passing cleanup functions that will be
//     run when the custom resource is being deleted. Only works with a non-nil finalizer, otherwise
//     the custom resource will be immediately deleted and the functions won't run.
func (r *Reconciler) GetInstance(ctx context.Context, key types.NamespacedName,
	instance client.Object, finalizer *string, cleanupFns ...func()) (*ctrl.Result, error) {
	logger := logr.FromContextOrDiscard(ctx)

	err := r.Client.Get(ctx, key, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Return and don't requeue
			return &ctrl.Result{}, nil
		}
		return &ctrl.Result{}, err
	}

	if util.IsBeingDeleted(instance) {

		// finalizer logic is only triggered if the controller
		// sets a finalizer, otherwise there's notihng to be done
		if finalizer != nil {

			if !controllerutil.ContainsFinalizer(instance, *finalizer) {
				return &ctrl.Result{}, nil
			}
			err := r.ManageCleanupLogic(instance, cleanupFns, logger)
			if err != nil {
				logger.Error(err, "unable to delete instance")
				result, err := ctrl.Result{}, err
				return &result, err
			}
			controllerutil.RemoveFinalizer(instance, *finalizer)
			err = r.Client.Update(ctx, instance)
			if err != nil {
				logger.Error(err, "unable to update instance")
				result, err := ctrl.Result{}, err
				return &result, err
			}

		}
		return &ctrl.Result{}, nil
	}

	if ok := r.IsInitialized(instance, finalizer); !ok {
		err := r.Client.Update(ctx, instance)
		if err != nil {
			logger.Error(err, "unable to initialize instance")
			result, err := ctrl.Result{}, err
			return &result, err
		}
		return &ctrl.Result{}, nil
	}
	return nil, nil
}

// IsInitialized can be used to check if instance is correctly initialized.
// Returns false if it isn't.
func (r *Reconciler) IsInitialized(instance client.Object, finalizer *string) bool {
	ok := true
	if finalizer != nil && !controllerutil.ContainsFinalizer(instance, *finalizer) {
		controllerutil.AddFinalizer(instance, *finalizer)
		ok = false
	}

	return ok
}

// ManageCleanupLogic contains finalization logic for the Reconciler
func (r *Reconciler) ManageCleanupLogic(instance client.Object, fns []func(), log logr.Logger) error {
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
func (r *Reconciler) ReconcileOwnedResources(ctx context.Context, owner client.Object, list []resource.TemplateInterface) error {
	managedResources := []corev1.ObjectReference{}

	for _, template := range list {
		ref, err := resource.CreateOrUpdate(ctx, r.Client, r.Scheme, owner, template)
		if err != nil {
			return fmt.Errorf("unable to CreateOrUpdate resource: %w", err)
		}
		if ref != nil {
			managedResources = append(managedResources, *ref)
			r.typeTracker.trackType(schema.FromAPIVersionAndKind(ref.APIVersion, ref.Kind))
		}
	}

	if isPrunerEnabled(owner) {
		if err := r.pruneOrphaned(ctx, owner, managedResources); err != nil {
			return fmt.Errorf("unable to prune orphaned resources: %w", err)
		}
	}

	return nil
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
				req = append(req, reconcile.Request{NamespacedName: util.ObjectKey(item)})
			}
			return req
		},
	)
}
