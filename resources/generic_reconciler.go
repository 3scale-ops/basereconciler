package resources

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	"github.com/ohler55/ojg/jp"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type MutationFunction func(context.Context, client.Client, client.Object, client.Object) error

type ResourceReconcilerConfig struct {
	ReconcileProperties []Property
	IgnoreProperties    []Property
}

type ObjectPointer[T any] interface {
	*T
	client.Object
}

// ResourceReconciler implements a generic reconciler for resources
func ResourceReconciler[T any, objectPtr ObjectPointer[T]](ctx context.Context, cl client.Client,
	desired objectPtr, enabled bool, cfg ResourceReconcilerConfig, mutationFns ...MutationFunction) error {

	logger := log.FromContext(ctx, "kind", desired.GetObjectKind(), "resource", desired.GetName())
	needsUpdate := false

	var instance objectPtr = new(T)
	err := cl.Get(ctx, types.NamespacedName{Name: desired.GetName(), Namespace: desired.GetNamespace()}, instance)
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

	for _, fn := range mutationFns {
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

	switch v := any(desired).(type) {

	case *T:
		for _, property := range cfg.IgnoreProperties {
			if err := property.Ignore(uinstance, udesired, logger); err != nil {
				return err
			}
		}
		for _, property := range cfg.ReconcileProperties {
			changed, err := property.Reconcile(uinstance, udesired, logger)
			if err != nil {
				return err
			}
			needsUpdate = needsUpdate || changed
		}

		if err = runtime.DefaultUnstructuredConverter.FromUnstructured(uinstance, instance); err != nil {
			return fmt.Errorf("unable to convert unstructured to instance: %w", err)
		}

	default:
		return fmt.Errorf(fmt.Sprintf("unable to reconcile resource type '%T'", v))
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

type Property string

func (p Property) JSONPath() string { return string(p) }

func (p Property) Reconcile(instance, desired map[string]any, logger logr.Logger) (bool, error) {
	expr, err := jp.ParseString(p.JSONPath())
	if err != nil {
		return false, fmt.Errorf("unable to parse JSONPath '%s': %w", p.JSONPath(), err)
	}

	desiredVal := expr.Get(desired)
	instanceVal := expr.Get(instance)
	if len(desiredVal) > 1 || len(instanceVal) > 1 {
		return false, fmt.Errorf("multi-valued JSONPath (%s) not supported when reconciling properties", p.JSONPath())
	}

	if len(desiredVal) != 0 && (len(instanceVal) == 0 || !equality.Semantic.DeepEqual(desiredVal[0], instanceVal[0])) {
		logger.V(1).Info("differences detected", "path", p.JSONPath(), "diff", cmp.Diff(instanceVal, desiredVal))
		if err := expr.Set(instance, desiredVal[0]); err != nil {
			return false, fmt.Errorf("usable to set value '%v' in JSONPath '%s'", desiredVal[0], p.JSONPath())
		}
		return true, nil
	}

	return false, nil
}

func (p Property) Ignore(instance, desired map[string]any, logger logr.Logger) error {
	expr, err := jp.ParseString(p.JSONPath())
	if err != nil {
		return fmt.Errorf("unable to parse JSONPath '%s': %w", p.JSONPath(), err)
	}

	err = expr.Del(desired)
	if err != nil {
		return fmt.Errorf("unable to parse delete JSONPath '%s' from unstructured desired: %w", p.JSONPath(), err)
	}
	err = expr.Del(instance)
	if err != nil {
		return fmt.Errorf("unable to parse delete JSONPath '%s' from unstructured instance: %w", p.JSONPath(), err)
	}

	return nil
}
