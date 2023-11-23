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

var _ reconciler.Resource = StatefulSetTemplate{}

// StatefulSet specifies a StatefulSet resource and its rollout triggers
type StatefulSetTemplate struct {
	Template        func() *appsv1.StatefulSet
	RolloutTriggers []RolloutTrigger
	IsEnabled       bool
}

func (sst StatefulSetTemplate) Build(ctx context.Context, cl client.Client) (client.Object, error) {

	ss := sst.Template()

	if err := sst.reconcileRolloutTriggers(ctx, cl, ss); err != nil {
		return nil, err
	}

	return ss.DeepCopy(), nil
}

func (sst StatefulSetTemplate) Enabled() bool {
	return sst.IsEnabled
}

func (sts StatefulSetTemplate) ResourceReconciler(ctx context.Context, cl client.Client, obj client.Object) error {
	logger := log.FromContext(ctx, "kind", "StatefulSet", "resource", obj.GetName())

	needsUpdate := false
	desired := obj.(*appsv1.StatefulSet)

	instance := &appsv1.StatefulSet{}
	err := cl.Get(ctx, types.NamespacedName{Name: desired.GetName(), Namespace: desired.GetNamespace()}, instance)
	if err != nil {
		if errors.IsNotFound(err) {

			if sts.Enabled() {
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
	if !sts.Enabled() {
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
		property.NewChangeSet[map[string]string]("metadata.annotations", &instance.ObjectMeta.Annotations, &desired.ObjectMeta.Annotations),
		property.NewChangeSet[int32]("spec.minReadySeconds", &instance.Spec.MinReadySeconds, &desired.Spec.MinReadySeconds),
		property.NewChangeSet[appsv1.StatefulSetPersistentVolumeClaimRetentionPolicy]("spec.persistentVolumeClaimRetentionPolicy", instance.Spec.PersistentVolumeClaimRetentionPolicy, desired.Spec.PersistentVolumeClaimRetentionPolicy),
		property.NewChangeSet[int32]("spec.replicas", instance.Spec.Replicas, desired.Spec.Replicas),
		property.NewChangeSet[metav1.LabelSelector]("spec.selector", instance.Spec.Selector, desired.Spec.Selector),
		property.NewChangeSet[string]("spec.serviceName", &instance.Spec.ServiceName, &desired.Spec.ServiceName),
		property.NewChangeSet[appsv1.StatefulSetUpdateStrategy]("spec.updateStrategy", &instance.Spec.UpdateStrategy, &desired.Spec.UpdateStrategy),
		property.NewChangeSet[[]corev1.PersistentVolumeClaim]("spec.volumeClaimTemplates", &instance.Spec.VolumeClaimTemplates, &desired.Spec.VolumeClaimTemplates),
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

// reconcileRolloutTriggers modifies the StatefulSet with the appropriate rollout triggers (annotations)
func (sst StatefulSetTemplate) reconcileRolloutTriggers(ctx context.Context, cl client.Client, ss *appsv1.StatefulSet) error {

	if ss.Spec.Template.ObjectMeta.Annotations == nil {
		ss.Spec.Template.ObjectMeta.Annotations = map[string]string{}
	}

	for _, trigger := range sst.RolloutTriggers {
		hash, err := trigger.GetHash(ctx, cl, ss.GetNamespace())
		if err != nil {
			return err
		}
		ss.Spec.Template.ObjectMeta.Annotations[trigger.GetAnnotationKey()] = hash
	}

	return nil
}
