package resources

import (
	"context"
	"fmt"

	"github.com/3scale-ops/basereconciler/property"
	"github.com/3scale-ops/basereconciler/reconciler"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
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

	/* Ensure the resource is in its desired state */
	needsUpdate = property.EnsureDesired(logger,
		property.NewChangeSet[map[string]string]("metadata.labels", &instance.ObjectMeta.Labels, &desired.ObjectMeta.Labels),
		property.NewChangeSet[map[string]string]("metadata.annotations", &instance.ObjectMeta.Annotations, &desired.ObjectMeta.Annotations),
		property.NewChangeSet[string]("spec.displayName", &instance.Spec.DisplayName, &desired.Spec.DisplayName),
		property.NewChangeSet[string]("spec.description", &instance.Spec.Description, &desired.Spec.Description),
		property.NewChangeSet[pipelinev1beta1.ParamSpecs]("spec.params", &instance.Spec.Params, &desired.Spec.Params),
		property.NewChangeSet[[]pipelinev1beta1.Step]("spec.steps", &instance.Spec.Steps, &desired.Spec.Steps),
		property.NewChangeSet[pipelinev1beta1.StepTemplate]("spec.stepTemplate", instance.Spec.StepTemplate, desired.Spec.StepTemplate),
		property.NewChangeSet[[]corev1.Volume]("spec.volumes", &instance.Spec.Volumes, &desired.Spec.Volumes),
		property.NewChangeSet[[]pipelinev1beta1.Sidecar]("spec.sidecars", &instance.Spec.Sidecars, &desired.Spec.Sidecars),
		property.NewChangeSet[[]pipelinev1beta1.WorkspaceDeclaration]("spec.workspaces", &instance.Spec.Workspaces, &desired.Spec.Workspaces),
		property.NewChangeSet[[]pipelinev1beta1.TaskResult]("spec.results", &instance.Spec.Results, &desired.Spec.Results),
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
