package resources

import (
	"context"
	"fmt"

	"github.com/3scale-ops/basereconciler/property"
	"github.com/3scale-ops/basereconciler/reconciler"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var _ reconciler.Resource = DeploymentTemplate{}

// DeploymentTemplate specifies a Deployment resource and its rollout triggers
type DeploymentTemplate struct {
	Template        func() *appsv1.Deployment
	RolloutTriggers []RolloutTrigger
	EnforceReplicas bool
	IsEnabled       bool
}

func (dt DeploymentTemplate) Build(ctx context.Context, cl client.Client) (client.Object, error) {

	dep := dt.Template()

	if err := dt.reconcileDeploymentReplicas(ctx, cl, dep); err != nil {
		return nil, err
	}

	if err := dt.reconcileRolloutTriggers(ctx, cl, dep); err != nil {
		return nil, err
	}

	return dep.DeepCopy(), nil
}

func (dt DeploymentTemplate) Enabled() bool {
	return dt.IsEnabled
}

func (dep DeploymentTemplate) ResourceReconciler(ctx context.Context, cl client.Client, obj client.Object) error {
	logger := log.FromContext(ctx, "kind", "Deployment", "resource", obj.GetName())

	needsUpdate := false
	desired := obj.(*appsv1.Deployment)

	instance := &appsv1.Deployment{}
	err := cl.Get(ctx, types.NamespacedName{Name: desired.GetName(), Namespace: desired.GetNamespace()}, instance)
	if err != nil {
		if errors.IsNotFound(err) {

			if dep.Enabled() {
				err = cl.Create(ctx, desired)
				if err != nil {
					return fmt.Errorf("unable to create object: " + err.Error())
				}
				logger.Info("resource created")
				return nil

			} else {
				return nil
			}
		}

		return err
	}

	/* Delete and return if not enabled */
	if !dep.Enabled() {
		err := cl.Delete(ctx, instance)
		if err != nil {
			return fmt.Errorf("unable to delete object: " + err.Error())
		}
		logger.Info("resource deleted")
		return nil
	}

	/* Ensure the resource is in its desired state */
	needsUpdate = property.EnsureDesired(logger,
		property.NewChangeSet[map[string]string]("metadata.labels", &instance.ObjectMeta.Labels, &desired.ObjectMeta.Labels),
		property.NewChangeSet[map[string]string]("metadata.annotations", &instance.ObjectMeta.Annotations, &desired.ObjectMeta.Annotations), // property.IgnoreNested(`metadata.annotations['deployment.kubernetes.io/revision']`),

		property.NewChangeSet[int32]("spec.minReadySeconds", &instance.Spec.MinReadySeconds, &desired.Spec.MinReadySeconds),
		property.NewChangeSet[int32]("spec.replicas", instance.Spec.Replicas, desired.Spec.Replicas),
		property.NewChangeSet[metav1.LabelSelector]("spec.selector", instance.Spec.Selector, desired.Spec.Selector),
		property.NewChangeSet[appsv1.DeploymentStrategy]("spec.strategy", &instance.Spec.Strategy, &desired.Spec.Strategy),
		property.NewChangeSet[map[string]string]("spec.template.metadata.labels", &instance.Spec.Template.ObjectMeta.Labels, &desired.Spec.Template.ObjectMeta.Labels),
		property.NewChangeSet[map[string]string]("spec.template.metadata.annotations", &instance.Spec.Template.ObjectMeta.Annotations, &desired.Spec.Template.ObjectMeta.Annotations),
		property.NewChangeSet[corev1.PodSpec]("spec.template.spec", &instance.Spec.Template.Spec, &desired.Spec.Template.Spec), // property.IgnoreNested("spec.template.spec.dnsPolicy"),
		// property.IgnoreNested("spec.template.spec.schedulerName"),

	)

	if needsUpdate {
		err := cl.Update(ctx, instance)
		if err != nil {
			return err
		}
		logger.Info("resource updated")
	}

	return nil
}

// reconcileDeploymentReplicas reconciles the number of replicas of a Deployment
func (dt DeploymentTemplate) reconcileDeploymentReplicas(ctx context.Context, cl client.Client, dep *appsv1.Deployment) error {

	if dt.EnforceReplicas {
		// Let the value in the template
		// override the runtime value
		return nil
	}

	key := types.NamespacedName{
		Name:      dep.GetName(),
		Namespace: dep.GetNamespace(),
	}
	instance := &appsv1.Deployment{}
	err := cl.Get(ctx, key, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}

	// override the value in the template with the
	// runtime value
	dep.Spec.Replicas = instance.Spec.Replicas
	return nil
}

// reconcileRolloutTriggers modifies the Deployment with the appropriate rollout triggers (annotations)
func (dt DeploymentTemplate) reconcileRolloutTriggers(ctx context.Context, cl client.Client, dep *appsv1.Deployment) error {

	if dep.Spec.Template.ObjectMeta.Annotations == nil {
		dep.Spec.Template.ObjectMeta.Annotations = map[string]string{}
	}

	for _, trigger := range dt.RolloutTriggers {
		hash, err := trigger.GetHash(ctx, cl, dep.GetNamespace())
		if err != nil {
			return err
		}
		dep.Spec.Template.ObjectMeta.Annotations[trigger.GetAnnotationKey()] = hash
	}

	return nil
}
