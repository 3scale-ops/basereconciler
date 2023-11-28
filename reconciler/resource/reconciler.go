package resource

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/3scale-ops/basereconciler/util"
	"github.com/nsf/jsondiff"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type MutationFunction func(context.Context, client.Client, client.Object, client.Object) error

type ReconcilerConfig struct {
	ReconcileProperties []Property
	IgnoreProperties    []Property
	Mutations           []MutationFunction
}

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

	// convert to unstructured
	udesired, err := runtime.DefaultUnstructuredConverter.ToUnstructured(desired)
	if err != nil {
		return fmt.Errorf("unable to convert desired to unstructured: %w", err)
	}

	uinstance, err := runtime.DefaultUnstructuredConverter.ToUnstructured(instance)
	if err != nil {
		return fmt.Errorf("unable to convert instance to unstructured: %w", err)
	}

	// calculate normalized diff
	unormalized := util.IntersectMap(udesired, uinstance)
	normalized, err := util.NewFromGVK(gvk, s)
	if err != nil {
		return err
	}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(unormalized, normalized)
	if err != nil {
		return fmt.Errorf("unable to convert from unstructured: %w", err)
	}

	if equality.Semantic.DeepEqual(normalized, desired) {
		// nothing to do
		return nil
	} else {
		fmt.Printf("%+v\n", unormalized)
		fmt.Printf("%+v\n", udesired)
	}

	if diff, err := printDiff(unormalized, udesired); err != nil {
		logger.Error(err, "update required, but unable to log diff")
	} else {
		logger.Info("update required", "diff", diff)
	}

	// // test autodetect managed fields by walking the template
	// jp.Walk(udesired, func(path jp.Expr, value any) {
	// 	logger.Info("autodetect managed fields", "path", path.String())
	// }, true)

	// if err := controllerutil.SetControllerReference(owner, object, r.Scheme); err != nil {
	// 	return err
	// }

	// // ignore properties
	// for _, property := range cfg.IgnoreProperties {
	// 	if err := property.Ignore(uinstance, udesired, logger); err != nil {
	// 		return err
	// 	}
	// }
	// // reconcile properties
	// for _, property := range cfg.ReconcileProperties {
	// 	changed, err := property.Reconcile(uinstance, udesired, logger)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	needsUpdate = needsUpdate || changed
	// }

	// if needsUpdate {
	// 	err := cl.Update(ctx, client.Object(&unstructured.Unstructured{Object: uinstance}))
	// 	if err != nil {
	// 		return err
	// 	}
	// 	logger.Info("Resource updated")
	// }

	return nil
}

func printDiff(a, b map[string]any) (string, error) {
	ja, err := json.Marshal(a)
	if err != nil {
		return "", fmt.Errorf("unable to log differences: %w", err)
	}
	jb, err := json.Marshal(b)
	if err != nil {
		return "", fmt.Errorf("unable to log differences: %w", err)
	}

	opts := jsondiff.DefaultJSONOptions()
	opts.SkipMatches = true
	opts.Indent = "\t"
	_, diff := jsondiff.Compare(ja, jb, &opts)
	return diff, nil
}
