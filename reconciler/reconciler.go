package reconciler

import (
	"context"
	"fmt"
	"reflect"
	"strconv"

	"github.com/3scale-ops/basereconciler/util"
	externalsecretsv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/go-logr/logr"
	grafanav1alpha1 "github.com/grafana-operator/grafana-operator/v4/api/integreatly/v1alpha1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type ReconcilerManagedTypes []client.ObjectList

func (mts ReconcilerManagedTypes) Register(mt client.ObjectList) ReconcilerManagedTypes {
	mts = append(mts, mt)
	return mts
}

func NewManagedTypes() ReconcilerManagedTypes {
	return ReconcilerManagedTypes{}
}

type ReconcilerOptions struct {
	ManagedTypes      ReconcilerManagedTypes
	AnnotationsDomain string
	ResourcePruner    bool
}

var Config ReconcilerOptions = ReconcilerOptions{
	AnnotationsDomain: "basereconciler.3cale.net",
	ResourcePruner:    true,
	ManagedTypes: ReconcilerManagedTypes{
		&corev1.ServiceList{},
		&corev1.ConfigMapList{},
		&appsv1.DeploymentList{},
		&appsv1.StatefulSetList{},
		&externalsecretsv1beta1.ExternalSecretList{},
		&grafanav1alpha1.GrafanaDashboardList{},
		&autoscalingv2.HorizontalPodAutoscalerList{},
		&policyv1.PodDisruptionBudgetList{},
		&monitoringv1.PodMonitorList{},
		&rbacv1.RoleBindingList{},
		&rbacv1.RoleList{},
		&corev1.ServiceAccountList{},
		&pipelinev1beta1.PipelineList{},
		&pipelinev1beta1.TaskList{},
	},
}

type Resource interface {
	Build(ctx context.Context, cl client.Client) (client.Object, error)
	Enabled() bool
	ResourceReconciler(context.Context, client.Client, client.Object) error
}

// Reconciler computes a list of resources that it needs to keep in place
type Reconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func NewFromManager(mgr manager.Manager) Reconciler {
	return Reconciler{Client: mgr.GetClient(), Scheme: mgr.GetScheme()}
}

// GetInstance tries to retrieve the custom resource instance and perform some standard
// tasks like initialization and cleanup when required.
func (r *Reconciler) GetInstance(ctx context.Context, key types.NamespacedName,
	instance client.Object, finalizer *string, cleanupFns []func()) error {
	logger := log.FromContext(ctx)

	err := r.Client.Get(ctx, key, instance)
	if err != nil {
		return err
	}

	if util.IsBeingDeleted(instance) {

		// finalizer logic is only triggered if the controller
		// sets a finalizer, otherwise there's notihng to be done
		if finalizer != nil {

			if !controllerutil.ContainsFinalizer(instance, *finalizer) {
				return nil
			}
			err := r.ManageCleanupLogic(instance, cleanupFns, logger)
			if err != nil {
				logger.Error(err, "unable to delete instance")
				return err
			}
			controllerutil.RemoveFinalizer(instance, *finalizer)
			err = r.Client.Update(ctx, instance)
			if err != nil {
				logger.Error(err, "unable to update instance")
				return err
			}
		}
		return nil
	}

	if ok := r.IsInitialized(instance, finalizer); !ok {
		err := r.Client.Update(ctx, instance)
		if err != nil {
			logger.Error(err, "unable to initialize instance")
			return err
		}
		return nil
	}
	return nil
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

// ManageCleanupLogic contains finalization logic for the LockedResourcesReconciler
// Functionality can be extended by passing extra cleanup functions
func (r *Reconciler) ManageCleanupLogic(instance client.Object, fns []func(), log logr.Logger) error {

	// Call any cleanup functions passed
	for _, fn := range fns {
		fn()
	}

	return nil
}

// ReconcileOwnedResources handles generalized resource reconcile logic for
// all controllers
func (r *Reconciler) ReconcileOwnedResources(ctx context.Context, owner client.Object, resources []Resource) error {

	managedResources := []corev1.ObjectReference{}

	for _, res := range resources {

		object, err := res.Build(ctx, r.Client)
		if err != nil {
			return err
		}

		if err := controllerutil.SetControllerReference(owner, object, r.Scheme); err != nil {
			return err
		}

		if err := res.ResourceReconciler(ctx, r.Client, object); err != nil {
			return err
		}

		managedResources = append(managedResources, corev1.ObjectReference{
			Namespace: object.GetNamespace(),
			Name:      object.GetName(),
			Kind:      reflect.TypeOf(object).Elem().Name(),
		})
	}

	if IsPrunerEnabled(owner) {
		if err := r.PruneOrphaned(ctx, owner, managedResources); err != nil {
			return err
		}
	}

	return nil
}

func IsPrunerEnabled(owner client.Object) bool {
	// prune is active by default
	prune := true

	// get the per resource switch (annotation)
	if value, ok := owner.GetAnnotations()[fmt.Sprintf("%s/prune", Config.AnnotationsDomain)]; ok {
		var err error
		prune, err = strconv.ParseBool(value)
		if err != nil {
			prune = true
		}
	}

	return prune && Config.ResourcePruner
}

func (r *Reconciler) PruneOrphaned(ctx context.Context, owner client.Object, managed []corev1.ObjectReference) error {
	logger := log.FromContext(ctx)

	for _, lType := range Config.ManagedTypes {

		err := r.Client.List(ctx, lType, client.InNamespace(owner.GetNamespace()))
		if err != nil {
			return err
		}

		for _, obj := range util.GetItems(lType) {

			kind := reflect.TypeOf(obj).Elem().Name()
			if isOwned(owner, obj) && !util.IsBeingDeleted(obj) && !isManaged(util.ObjectKey(obj), kind, managed) {

				err := r.Client.Delete(ctx, obj)
				if err != nil {
					return err
				}
				logger.Info("resource deleted", "kind", reflect.TypeOf(obj).Elem().Name(), "resource", obj.GetName())
			}
		}
	}
	return nil
}

func isOwned(owner client.Object, owned client.Object) bool {
	refs := owned.GetOwnerReferences()
	for _, ref := range refs {
		if ref.Kind == owner.GetObjectKind().GroupVersionKind().Kind && ref.Name == owner.GetName() {
			return true
		}
	}
	return false
}

func isManaged(key types.NamespacedName, kind string, managed []corev1.ObjectReference) bool {

	for _, m := range managed {
		if m.Name == key.Name && m.Namespace == key.Namespace && m.Kind == kind {
			return true
		}
	}
	return false
}

// SecretEventHandler returns an EventHandler for the specific client.ObjectList
// list object passed as parameter
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

			return []reconcile.Request{{NamespacedName: util.ObjectKey(items[0])}}
		},
	)
}
