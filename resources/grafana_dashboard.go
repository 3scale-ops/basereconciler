package resources

import (
	"context"
	"fmt"

	"github.com/3scale-ops/basereconciler/reconciler"
	grafanav1alpha1 "github.com/grafana-operator/grafana-operator/v4/api/integreatly/v1alpha1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var _ reconciler.Resource = GrafanaDashboardTemplate{}

// GrafanaDashboardTemplate has methods to generate and reconcile a GrafanaDashboard
type GrafanaDashboardTemplate struct {
	Template  func() *grafanav1alpha1.GrafanaDashboard
	IsEnabled bool
}

// Build returns a GrafanaDashboard resource
func (gdt GrafanaDashboardTemplate) Build(ctx context.Context, cl client.Client) (client.Object, error) {
	return gdt.Template().DeepCopy(), nil
}

// Enabled indicates if the resource should be present or not
func (gdt GrafanaDashboardTemplate) Enabled() bool {
	return gdt.IsEnabled
}

// ResourceReconciler implements a generic reconciler for GrafanaDashboard resources
func (gdt GrafanaDashboardTemplate) ResourceReconciler(ctx context.Context, cl client.Client, obj client.Object) error {
	logger := log.FromContext(ctx, "kind", "GrafanaDashboard", "resource", obj.GetName())

	needsUpdate := false
	desired := obj.(*grafanav1alpha1.GrafanaDashboard)

	instance := &grafanav1alpha1.GrafanaDashboard{}
	err := cl.Get(ctx, types.NamespacedName{Name: desired.GetName(), Namespace: desired.GetNamespace()}, instance)
	if err != nil {
		if errors.IsNotFound(err) {

			if gdt.Enabled() {
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
	if !gdt.Enabled() {
		err := cl.Delete(ctx, instance)
		if err != nil {
			return fmt.Errorf("unable to delete object: " + err.Error())
		}
		logger.Info("resource deleted")
		return nil
	}

	/* Reconcile metadata */
	if !equality.Semantic.DeepEqual(instance.GetLabels(), desired.GetLabels()) {
		instance.ObjectMeta.Labels = desired.GetLabels()
		needsUpdate = true
	}

	/* Reconcile the spec */
	if !equality.Semantic.DeepEqual(instance.Spec, desired.Spec) {
		instance.Spec = desired.Spec
		needsUpdate = true
	}

	if needsUpdate {
		err := cl.Update(ctx, instance)
		if err != nil {
			return err
		}
		logger.Info("resource updated")
	}

	return nil
}
