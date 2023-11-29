package resource

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/3scale-ops/basereconciler/util"
	"github.com/nsf/jsondiff"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// CreateOrUpdate cretes or updates resources
func CreateOrUpdate(ctx context.Context, cl client.Client, scheme *runtime.Scheme, owner client.Object, template TemplateInterface) (*corev1.ObjectReference, error) {

	desired, err := template.Build(ctx, cl, nil)
	if err != nil {
		return nil, err
	}

	gvk, err := apiutil.GVKForObject(desired, scheme)
	if err != nil {
		return nil, err
	}
	logger := log.FromContext(ctx, "gvk", gvk, "resource", desired.GetName())

	instance, err := util.NewFromGVK(gvk, scheme)
	if err != nil {
		return nil, err
	}
	err = cl.Get(ctx, util.ObjectKey(desired), instance)
	if err != nil {
		if errors.IsNotFound(err) {
			if template.Enabled() {
				if err := controllerutil.SetControllerReference(owner, desired, scheme); err != nil {
					return nil, err
				}
				err = cl.Create(ctx, desired)
				if err != nil {
					return nil, fmt.Errorf("unable to create object: %w", err)
				}
				logger.Info("resource created")
				return util.ObjectReference(instance, gvk), nil

			} else {
				return nil, nil
			}
		}
		return nil, err
	}

	/* Delete and return if not enabled */
	if !template.Enabled() {
		err := cl.Delete(ctx, instance)
		if err != nil {
			return nil, fmt.Errorf("unable to delete object: %w", err)
		}
		logger.Info("resource deleted")
		return nil, nil
	}

	cfg := template.ReconcilerConfig()
	diff, err := util.NewFromGVK(gvk, scheme)
	if err != nil {
		return nil, err
	}
	diff.SetName(desired.GetName())
	diff.SetNamespace(desired.GetNamespace())

	// convert to unstructured
	udesired, err := runtime.DefaultUnstructuredConverter.ToUnstructured(desired)
	if err != nil {
		return nil, fmt.Errorf("unable to convert desired to unstructured: %w", err)
	}

	uinstance, err := runtime.DefaultUnstructuredConverter.ToUnstructured(instance)
	if err != nil {
		return nil, fmt.Errorf("unable to convert instance to unstructured: %w", err)
	}

	udiff, err := runtime.DefaultUnstructuredConverter.ToUnstructured(diff)
	if err != nil {
		return nil, fmt.Errorf("unable to convert diff to unstructured: %w", err)
	}

	// reconcile properties
	for _, property := range cfg.ReconcileProperties {
		if err := property.Reconcile(uinstance, udesired, udiff, logger); err != nil {
			return nil, err
		}
	}

	// ignore properties
	for _, property := range cfg.IgnoreProperties {
		for _, m := range []map[string]any{uinstance, udesired, udiff} {
			if err := property.Ignore(m); err != nil {
				return nil, err
			}
		}
	}

	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(udiff, diff); err != nil {
		return nil, fmt.Errorf("unable to convert diff from unstructured: %w", err)
	}
	if !equality.Semantic.DeepEqual(diff, desired) {
		logger.V(1).Info("resource update required", "diff", printfDiff(diff, desired))
		err := cl.Update(ctx, client.Object(&unstructured.Unstructured{Object: uinstance}))
		if err != nil {
			return nil, err
		}
		logger.Info("Resource updated")
	}

	return util.ObjectReference(instance, gvk), nil
}

func printfDiff(a, b client.Object) string {
	ajson, err := json.Marshal(a)
	if err != nil {
		return fmt.Errorf("unsable to log differences: %w", err).Error()
	}
	bjson, err := json.Marshal(b)
	if err != nil {
		return fmt.Errorf("unsable to log differences: %w", err).Error()
	}
	opts := jsondiff.DefaultJSONOptions()
	opts.SkipMatches = true
	opts.Indent = "\t"
	_, diff := jsondiff.Compare(ajson, bjson, &opts)
	return diff
}
