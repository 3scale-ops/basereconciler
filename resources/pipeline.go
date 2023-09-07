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

var _ reconciler.Resource = PipelineTemplate{}

// PipelineTemplate has methods to generate and reconcile a Pipeline
type PipelineTemplate struct {
	Template  func() *pipelinev1beta1.Pipeline
	IsEnabled bool
}

// Build returns a Pipeline resource
func (pt PipelineTemplate) Build(ctx context.Context, cl client.Client) (client.Object, error) {
	return pt.Template().DeepCopy(), nil
}

// Enabled indicates if the resource should be present or not
func (pt PipelineTemplate) Enabled() bool {
	return pt.IsEnabled
}

// ResourceReconciler implements a generic reconciler for Pipeline resources
func (pt PipelineTemplate) ResourceReconciler(ctx context.Context, cl client.Client, obj client.Object) error {
	logger := log.FromContext(ctx, "kind", "Pipeline", "resource", obj.GetName())

	needsUpdate := false
	desired := obj.(*pipelinev1beta1.Pipeline)

	instance := &pipelinev1beta1.Pipeline{}
	err := cl.Get(ctx, types.NamespacedName{Name: desired.GetName(), Namespace: desired.GetNamespace()}, instance)
	if err != nil {
		if errors.IsNotFound(err) {

			if pt.Enabled() {
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
	if !pt.Enabled() {
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

	if needsUpdate {
		err := cl.Update(ctx, instance)
		if err != nil {
			return err
		}
		logger.Info("Resource updated")
	}

	return nil
}
