# basereconciler

Basereconciler is an attempt to create a reconciler that can be imported an used in any controller-runtime based controller to perform the most common tasks a controller usually performs. It's a bunch of code that it's typically written again and again for every and each controller and that can be abstracted to work in a more generic way to avoid the repetition and improve code mantainability.
At the moment basereconciler can perform the following tasks:

* **Get the custom resource and perform some common tasks on it**:
  * Management of resource finalizer: some custom resources required more complex finalization logic. For this to happen a finalizer must be in place. Basereconciler can keep this finalizer in place and remove it when necessary during resource finalization.
  * Management of finalization logic: it checks if the resource is being finalized and executed the finalization logic passed to it if that is the case. When all finalization logic is completed it removes the finalizer on the custom resource.
* **Reconcile resources owned by the custom resource**: basreconciler can keep the owned resources of a custom resource in it's desired state. It works for any resource type, and only requires that the user configures how each specific resource type has to be configured. The resource reconciler only works in "update mode" right now, so any operation to transition a given resource from its live state to its desired state will be an Update. We might add a "patch mode" in the future.
* **Reconcile custom resource status**: if the custom resource implements a certain interface, basereconciler can also be in charge of reconciling the status.
* **Resource pruner**: when the reconciler stops seeing a certain resource, owned by the custom resource, it will prune them as it understands that the resource is no logner required. The resource pruner can be disabled globally or enabled/disabled on a per resource basis based on an annotation.

## Basic Usage

The following example is a kubebuilder bootstrapped controller that uses basereconciler to reconcile several resources owned by a custom resource. Explanations inline in the code.

```go
package controllers

import (
	"context"

	"github.com/3scale-ops/basereconciler/config"
	"github.com/3scale-ops/basereconciler/mutators"
	"github.com/3scale-ops/basereconciler/reconciler"
	"github.com/3scale-ops/basereconciler/resource"
	"github.com/3scale-ops/basereconciler/util"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	webappv1 "my.domain/guestbook/api/v1"
)

// Use the init function to configure the behavior of the controller. In this case we use
// "SetDefaultReconcileConfigForGVK" to specify the paths that need to be reconciled/ignored
// for each resource type. Check the "github.com/3scale-ops/basereconciler/config" for more
// configuration options
func init() {
	config.SetDefaultReconcileConfigForGVK(
		schema.FromAPIVersionAndKind("v1", "Service"),
		config.ReconcileConfigForGVK{
			EnsureProperties: []string{
				"metadata.annotations",
				"metadata.labels",
				"spec",
			},
		})
	config.SetDefaultReconcileConfigForGVK(
		schema.FromAPIVersionAndKind("apps/v1", "Deployment"),
		config.ReconcileConfigForGVK{
			EnsureProperties: []string{
				"metadata.annotations",
				"metadata.labels",
				"spec",
			},
			IgnoreProperties: []string{
				"metadata.annotations['deployment.kubernetes.io/revision']",
			},
		})
}

// GuestbookReconciler reconciles a Guestbook object
type GuestbookReconciler struct {
	*reconciler.Reconciler
}

// +kubebuilder:rbac:groups=webapp.my.domain,resources=guestbooks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=webapp.my.domain,resources=guestbooks/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="core",namespace=placeholder,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="apps",namespace=placeholder,resources=deployments,verbs=get;list;watch;create;update;patch;delete

func (r *GuestbookReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// configure the logger for the controller. The function also returns a modified
	// copy of the context that includes the logger so it's easily passed around to other functions.
	ctx, logger := r.Logger(ctx, "guestbook", req.NamespacedName)

	// GetInstance will take care of retrieveing the custom resoure from the API. It is also in charge of the resource
	// finalization logic if there is one. In this example, we are configuring a finalizer in our custom resource and passing
	// a finalization function that will casuse a log line to show when the resource is being deleted.
	guestbook := &webappv1.Guestbook{}
	result := r.GetInstance(ctx, req, guestbook, util.Pointer("guestbook-finalizer"), func() { logger.Info("finalizing resource") })
	if result.IsReturnAndRequeue() {
		return result.Values()
	}

	// ReconcileOwnedResources creates/updates/deletes the resoures that our custom resource owns.
	// It is a list of templates, in this case generated from the base of an object we provide.
	// Modifiers can be added to the template to get live values from the k8s API, like in this example
	// with the Service. Check the documentation of the "github.com/3scale-ops/basereconciler/resource"
	// for more information on building templates.
	result = r.ReconcileOwnedResources(ctx, guestbook, []resource.TemplateInterface{

		resource.NewTemplateFromObjectFunction[*appsv1.Deployment](
			func() *appsv1.Deployment {
				return &appsv1.Deployment{
					// define your object here
				}
			}),

		resource.NewTemplateFromObjectFunction[*corev1.Service](
			func() *corev1.Service {
				return &corev1.Service{
					// define your object here
				}
			}).
			// Retrieve the live values that kube-controller-manager sets
			// in the Service spec to avoid overwrting them
			WithMutation(mutators.SetServiceLiveValues()).
			// There are some useful mutations in the "github.com/3scale-ops/basereconciler/mutators"
			// package or you can pass your own mutation functions
			WithMutation(func(ctx context.Context, cl client.Client, desired client.Object) error {
				// your mutation logic here
				return nil
			}).
			// The templates are reconciled using the global config defined in the init() function
			// but in this case we are passing a custom config that will apply
			// only to the reconciliation of this template
			WithEnsureProperties([]resource.Property{"spec"}).
			WithIgnoreProperties([]resource.Property{"spec.clusterIP", "spec.clusterIPs"}),
	})

	if result.IsReturnAndRequeue() {
		return result.Values()
	}

	return ctrl.Result{}, nil
}

func (r *GuestbookReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&webappv1.Guestbook{}).
		// add the watches for the specific resource types that the
		// custom resource owns to watch for changes on those
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Complete(r)
}
```

Then you just need to register the controller with the controller-runtime manager and you are all set!

```go
[...]
	if err = (&controllers.GuestbookReconciler{
		Reconciler: reconciler.NewFromManager(mgr).WithLogger(ctrl.Log.WithName("controllers").WithName("Guestbook")),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Guestbook")
		os.Exit(1)
	}
[...]
```