package resource

import (
	"context"
	"fmt"

	"github.com/3scale-ops/basereconciler/util"
	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Reconcile implements a generic reconciler for resources
func (t Template[T]) Reconcile(ctx context.Context, cl client.Client, s *runtime.Scheme, owner client.Object) error {

	desired, err := t.Build(ctx, cl, nil)
	if err != nil {
		return err
	}

	gvk, err := apiutil.GVKForObject(desired, s)
	if err != nil {
		return err
	}
	logger := log.FromContext(ctx, "gvk", gvk, "resource", desired.GetName())

	instance, err := util.NewFromGVK(gvk, s)
	if err != nil {
		return err
	}
	err = cl.Get(ctx, util.ObjectKey(desired), instance)
	if err != nil {
		if errors.IsNotFound(err) {
			if t.Enabled() {
				if err := controllerutil.SetControllerReference(owner, desired, s); err != nil {
					return err
				}
				err = cl.Create(ctx, desired)
				if err != nil {
					return fmt.Errorf("unable to create object: %w", err)
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
	if !t.Enabled() {
		err := cl.Delete(ctx, instance)
		if err != nil {
			return fmt.Errorf("unable to delete object: %w", err)
		}
		logger.Info("resource deleted")
		return nil
	}

	needsUpdate := false
	cfg := t.ReconcilerConfig()
	diff, err := util.NewFromGVK(gvk, s)
	if err != nil {
		return err
	}
	diff.SetName(desired.GetName())
	diff.SetNamespace(desired.GetNamespace())

	// convert to unstructured
	udesired, err := runtime.DefaultUnstructuredConverter.ToUnstructured(desired)
	if err != nil {
		return fmt.Errorf("unable to convert desired to unstructured: %w", err)
	}

	uinstance, err := runtime.DefaultUnstructuredConverter.ToUnstructured(instance)
	if err != nil {
		return fmt.Errorf("unable to convert instance to unstructured: %w", err)
	}

	udiff, err := runtime.DefaultUnstructuredConverter.ToUnstructured(diff)
	if err != nil {
		return fmt.Errorf("unable to convert diff to unstructured: %w", err)
	}

	// reconcile properties
	for _, property := range cfg.ReconcileProperties {
		changed, err := property.Reconcile(uinstance, udesired, udiff, logger)
		if err != nil {
			return err
		}
		needsUpdate = needsUpdate || changed
	}

	// ignore properties
	for _, property := range cfg.IgnoreProperties {
		if err := property.Ignore(uinstance, udesired, udiff, logger); err != nil {
			return err
		}
	}

	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(udiff, diff); err != nil {
		return fmt.Errorf("unable to convert diff from unstructured: %w", err)
	}

	if !equality.Semantic.DeepEqual(diff, desired) {
		logger.Info("resource required update", "diff", cmp.Diff(diff, desired))
		err := cl.Update(ctx, client.Object(&unstructured.Unstructured{Object: uinstance}))
		if err != nil {
			return err
		}
		logger.Info("Resource updated")
	}

	return nil
}
