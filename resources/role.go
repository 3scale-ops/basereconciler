package resources

import (
	"context"
	"fmt"

	"github.com/3scale-ops/basereconciler/reconciler"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var _ reconciler.Resource = RoleTemplate{}

// RoleTemplate has methods to generate and reconcile a Role
type RoleTemplate struct {
	Template  func() *rbacv1.Role
	IsEnabled bool
}

// Build returns a Role resource
func (rt RoleTemplate) Build(ctx context.Context, cl client.Client) (client.Object, error) {
	return rt.Template().DeepCopy(), nil
}

// Enabled indicates if the resource should be present or not
func (rt RoleTemplate) Enabled() bool {
	return rt.IsEnabled
}

// ResourceReconciler implements a generic reconciler for Role resources
func (rt RoleTemplate) ResourceReconciler(ctx context.Context, cl client.Client, obj client.Object) error {
	logger := log.FromContext(ctx, "kind", "Role", "resource", obj.GetName())

	needsUpdate := false
	desired := obj.(*rbacv1.Role)

	instance := &rbacv1.Role{}
	err := cl.Get(ctx, types.NamespacedName{Name: desired.GetName(), Namespace: desired.GetNamespace()}, instance)
	if err != nil {
		if errors.IsNotFound(err) {

			if rt.Enabled() {
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
	if !rt.Enabled() {
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
	if !equality.Semantic.DeepEqual(instance.GetAnnotations(), desired.GetAnnotations()) {
		instance.ObjectMeta.Annotations = desired.GetAnnotations()
		needsUpdate = true
	}

	/* Reconcile the rules */
	if !equality.Semantic.DeepEqual(instance.Rules, desired.Rules) {
		instance.Rules = desired.Rules
		needsUpdate = true
	}

	if needsUpdate {
		err := cl.Update(ctx, instance)
		if err != nil {
			return err
		}
		logger.Info("Resource updated")
	}

	return nil
}
