package resources

import (
	"context"
	"fmt"

	"github.com/3scale-ops/basereconciler/property"
	"github.com/3scale-ops/basereconciler/reconciler"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var _ reconciler.Resource = ServiceAccountTemplate{}

// ServiceAccountTemplate has methods to generate and reconcile a ServiceAccount
type ServiceAccountTemplate struct {
	Template  func() *corev1.ServiceAccount
	IsEnabled bool
}

// Build returns a ServiceAccount resource
func (sat ServiceAccountTemplate) Build(ctx context.Context, cl client.Client) (client.Object, error) {
	return sat.Template().DeepCopy(), nil
}

// Enabled indicates if the resource should be present or not
func (sat ServiceAccountTemplate) Enabled() bool {
	return sat.IsEnabled
}

// ResourceReconciler implements a generic reconciler for ServiceAccount resources
func (sat ServiceAccountTemplate) ResourceReconciler(ctx context.Context, cl client.Client, obj client.Object) error {
	logger := log.FromContext(ctx, "kind", "ServiceAccount", "resource", obj.GetName())

	needsUpdate := false
	desired := obj.(*corev1.ServiceAccount)

	instance := &corev1.ServiceAccount{}
	err := cl.Get(ctx, types.NamespacedName{Name: desired.GetName(), Namespace: desired.GetNamespace()}, instance)
	if err != nil {
		if errors.IsNotFound(err) {

			if sat.Enabled() {
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
	if !sat.Enabled() {
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
	)

	if needsUpdate {
		err := cl.Update(ctx, instance)
		if err != nil {
			return err
		}
		logger.Info("Resource updated")
	}

	return nil
}
