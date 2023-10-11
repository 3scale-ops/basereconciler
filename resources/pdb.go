package resources

import (
	"context"
	"fmt"

	"github.com/3scale-ops/basereconciler/property"
	"github.com/3scale-ops/basereconciler/reconciler"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var _ reconciler.Resource = PodDisruptionBudgetTemplate{}

// PodDisruptionBudgetTemplate has methods to generate and reconcile a PodDisruptionBudget
type PodDisruptionBudgetTemplate struct {
	Template  func() *policyv1.PodDisruptionBudget
	IsEnabled bool
}

// Build returns a PodDisruptionBudget resource
func (pdbt PodDisruptionBudgetTemplate) Build(ctx context.Context, cl client.Client) (client.Object, error) {
	return pdbt.Template().DeepCopy(), nil
}

// Enabled indicates if the resource should be present or not
func (pdbt PodDisruptionBudgetTemplate) Enabled() bool {
	return pdbt.IsEnabled
}

// ResourceReconciler implements a generic reconciler for PodDisruptionBudget resources
func (pdbt PodDisruptionBudgetTemplate) ResourceReconciler(ctx context.Context, cl client.Client, obj client.Object) error {
	logger := log.FromContext(ctx, "kind", "PodDisruptionBudget", "resource", obj.GetName())

	needsUpdate := false
	desired := obj.(*policyv1.PodDisruptionBudget)

	instance := &policyv1.PodDisruptionBudget{}
	err := cl.Get(ctx, types.NamespacedName{Name: desired.GetName(), Namespace: desired.GetNamespace()}, instance)
	if err != nil {
		if errors.IsNotFound(err) {

			if pdbt.Enabled() {
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
	if !pdbt.Enabled() {
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
		property.NewChangeSet[intstr.IntOrString]("spec.maxUnavailable", instance.Spec.MaxUnavailable, desired.Spec.MaxUnavailable),
		property.NewChangeSet[intstr.IntOrString]("spec.minAvailable", instance.Spec.MinAvailable, desired.Spec.MinAvailable),
		property.NewChangeSet[metav1.LabelSelector]("spec.selector", instance.Spec.Selector, desired.Spec.Selector),
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
