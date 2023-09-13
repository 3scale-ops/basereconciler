package resources

import (
	"context"
	"fmt"

	"github.com/3scale-ops/basereconciler/reconciler"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var _ reconciler.Resource = TaskTemplate{}

// TaskTemplate has methods to generate and reconcile a Task
type TaskTemplate struct {
	Template  func() *pipelinev1beta1.Task
	IsEnabled bool
}

// Build returns a Task resource
func (tt TaskTemplate) Build(ctx context.Context, cl client.Client) (client.Object, error) {
	return tt.Template().DeepCopy(), nil
}

// Enabled indicates if the resource should be present or not
func (tt TaskTemplate) Enabled() bool {
	return tt.IsEnabled
}

// ResourceReconciler implements a generic reconciler for Task resources
func (tt TaskTemplate) ResourceReconciler(ctx context.Context, cl client.Client, obj client.Object) error {
	logger := log.FromContext(ctx, "kind", "Task", "resource", obj.GetName())

	needsUpdate := false
	desired := obj.(*pipelinev1beta1.Task)

	instance := &pipelinev1beta1.Task{}
	err := cl.Get(ctx, types.NamespacedName{Name: desired.GetName(), Namespace: desired.GetNamespace()}, instance)
	if err != nil {
		if errors.IsNotFound(err) {

			if tt.Enabled() {
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
	if !tt.Enabled() {
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

	/* Reconcile spec */

	if instance.Spec.DisplayName != desired.Spec.DisplayName {
		instance.Spec.DisplayName = desired.Spec.DisplayName
		needsUpdate = true
	}

	if instance.Spec.Description != desired.Spec.Description {
		instance.Spec.Description = desired.Spec.Description
		needsUpdate = true
	}

	if !equality.Semantic.DeepEqual(instance.Spec.Params, desired.Spec.Params) {
		instance.Spec.Params = desired.Spec.Params
		needsUpdate = true
	}

	if !equality.Semantic.DeepEqual(instance.Spec.Steps, desired.Spec.Steps) {
		instance.Spec.Steps = desired.Spec.Steps
		needsUpdate = true
	}

	if !equality.Semantic.DeepEqual(instance.Spec.StepTemplate, desired.Spec.StepTemplate) {
		instance.Spec.StepTemplate = desired.Spec.StepTemplate
		needsUpdate = true
	}

	if !equality.Semantic.DeepEqual(instance.Spec.Volumes, desired.Spec.Volumes) {
		instance.Spec.Volumes = desired.Spec.Volumes
		needsUpdate = true
	}

	if !equality.Semantic.DeepEqual(instance.Spec.Sidecars, desired.Spec.Sidecars) {
		instance.Spec.Sidecars = desired.Spec.Sidecars
		needsUpdate = true
	}

	if !equality.Semantic.DeepEqual(instance.Spec.Workspaces, desired.Spec.Workspaces) {
		instance.Spec.Workspaces = desired.Spec.Workspaces
		needsUpdate = true
	}

	if !equality.Semantic.DeepEqual(instance.Spec.Volumes, desired.Spec.Volumes) {
		instance.Spec.Volumes = desired.Spec.Volumes
		needsUpdate = true
	}

	if !equality.Semantic.DeepEqual(instance.Spec.Results, desired.Spec.Results) {
		instance.Spec.Results = desired.Spec.Results
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
