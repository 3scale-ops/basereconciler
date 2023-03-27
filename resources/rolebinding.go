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

var _ reconciler.Resource = RoleBindingTemplate{}

// RoleBindingTemplate has methods to generate and reconcile a RoleBinding
type RoleBindingTemplate struct {
	Template  func() *rbacv1.RoleBinding
	IsEnabled bool
}

// Build returns a RoleBinding resource
func (rbt RoleBindingTemplate) Build(ctx context.Context, cl client.Client) (client.Object, error) {
	return rbt.Template().DeepCopy(), nil
}

// Enabled indicates if the resource should be present or not
func (rbt RoleBindingTemplate) Enabled() bool {
	return rbt.IsEnabled
}

// ResourceReconciler implements a generic reconciler for RoleBinding resources
func (rbt RoleBindingTemplate) ResourceReconciler(ctx context.Context, cl client.Client, obj client.Object) error {
	logger := log.FromContext(ctx, "kind", "RoleBinding", "resource", obj.GetName())

	needsUpdate := false
	desired := obj.(*rbacv1.RoleBinding)

	instance := &rbacv1.RoleBinding{}
	err := cl.Get(ctx, types.NamespacedName{Name: desired.GetName(), Namespace: desired.GetNamespace()}, instance)
	if err != nil {
		if errors.IsNotFound(err) {

			if rbt.Enabled() {
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
	if !rbt.Enabled() {
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

	/* Reconcile the roleref */
	if !equality.Semantic.DeepEqual(instance.RoleRef, desired.RoleRef) {
		instance.RoleRef = desired.RoleRef
		needsUpdate = true
	}

	/* Reconcile the subjects */
	if !equality.Semantic.DeepEqual(instance.Subjects, desired.Subjects) {
		instance.Subjects = desired.Subjects
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
