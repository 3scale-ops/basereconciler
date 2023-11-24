package resource

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type MutationFunction func(context.Context, client.Client, client.Object, client.Object) error

type ReconcilerConfig struct {
	ReconcileProperties []Property
	IgnoreProperties    []Property
	Mutations           []MutationFunction
}

// Reconcile implements a generic reconciler for resources
func (cfg ReconcilerConfig) Reconcile(ctx context.Context, cl client.Client, s *runtime.Scheme,
	desired client.Object, enabled bool) error {

	gvk, err := apiutil.GVKForObject(desired, s)
	if err != nil {
		return err
	}
	logger := log.FromContext(ctx, "gvk", gvk, "resource", desired.GetName())
	needsUpdate := false

	ro, err := s.New(gvk)
	if err != nil {
		return err
	}
	instance, ok := ro.(client.Object)
	if !ok {
		return fmt.Errorf("runtime object %T does not implement client.Object", ro)
	}

	err = cl.Get(ctx, types.NamespacedName{Name: desired.GetName(), Namespace: desired.GetNamespace()}, instance)
	if err != nil {
		if errors.IsNotFound(err) {

			if enabled {
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
	if !enabled {
		err := cl.Delete(ctx, instance)
		if err != nil {
			return fmt.Errorf("unable to delete object: %w", err)
		}
		logger.Info("resource deleted")
		return nil
	}

	for _, fn := range cfg.Mutations {
		if err := fn(ctx, cl, instance, desired); err != nil {
			return err
		}
	}

	// convert to unstructured
	udesired, err := runtime.DefaultUnstructuredConverter.ToUnstructured(desired)
	if err != nil {
		return fmt.Errorf("unable to convert desired to unstructured: %w", err)
	}

	uinstance, err := runtime.DefaultUnstructuredConverter.ToUnstructured(instance)
	if err != nil {
		return fmt.Errorf("unable to convert instance to unstructured: %w", err)
	}

	// ignore properties
	for _, property := range cfg.IgnoreProperties {
		if err := property.Ignore(uinstance, udesired, logger); err != nil {
			return err
		}
	}
	// reconcile properties
	for _, property := range cfg.ReconcileProperties {
		changed, err := property.Reconcile(uinstance, udesired, logger)
		if err != nil {
			return err
		}
		needsUpdate = needsUpdate || changed
	}

	if needsUpdate {
		err := cl.Update(ctx, client.Object(&unstructured.Unstructured{Object: uinstance}))
		if err != nil {
			return err
		}
		logger.Info("Resource updated")
	}

	return nil
}
