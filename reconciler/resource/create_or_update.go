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

	live, err := util.NewFromGVK(gvk, scheme)
	if err != nil {
		return nil, err
	}
	err = cl.Get(ctx, util.ObjectKey(desired), live)
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
				return util.ObjectReference(live, gvk), nil

			} else {
				return nil, nil
			}
		}
		return nil, err
	}

	/* Delete and return if not enabled */
	if !template.Enabled() {
		err := cl.Delete(ctx, live)
		if err != nil {
			return nil, fmt.Errorf("unable to delete object: %w", err)
		}
		logger.Info("resource deleted")
		return nil, nil
	}

	cfg := template.ReconcilerConfig()

	// normalizedLive is a struct that will be populated with only the reconciled
	// properties and their respective live values. It will be used to compare it with
	// the desire and determine in an update is required.
	normalizedLive, err := util.NewFromGVK(gvk, scheme)
	if err != nil {
		return nil, err
	}
	normalizedLive.SetName(desired.GetName())
	normalizedLive.SetNamespace(desired.GetNamespace())

	// convert to unstructured
	u_desired, err := runtime.DefaultUnstructuredConverter.ToUnstructured(desired)
	if err != nil {
		return nil, fmt.Errorf("unable to convert desired to unstructured: %w", err)
	}

	u_live, err := runtime.DefaultUnstructuredConverter.ToUnstructured(live)
	if err != nil {
		return nil, fmt.Errorf("unable to convert live to unstructured: %w", err)
	}

	u_normzalizedLive, err := runtime.DefaultUnstructuredConverter.ToUnstructured(normalizedLive)
	if err != nil {
		return nil, fmt.Errorf("unable to convert normalizedLive to unstructured: %w", err)
	}

	// reconcile properties
	for _, property := range cfg.ReconcileProperties {
		if err := property.Reconcile(u_live, u_desired, u_normzalizedLive, logger); err != nil {
			return nil, err
		}
	}

	// ignore properties
	for _, property := range cfg.IgnoreProperties {
		for _, m := range []map[string]any{u_live, u_desired, u_normzalizedLive} {
			if err := property.Ignore(m); err != nil {
				return nil, err
			}
		}
	}

	// do the comparison using structs so "equality.Semantic" can be used
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u_normzalizedLive, normalizedLive); err != nil {
		return nil, fmt.Errorf("unable to convert diff from unstructured: %w", err)
	}
	if !equality.Semantic.DeepEqual(normalizedLive, desired) {
		logger.V(1).Info("resource update required", "diff", printfDiff(normalizedLive, desired))
		err := cl.Update(ctx, client.Object(&unstructured.Unstructured{Object: u_live}))
		if err != nil {
			return nil, err
		}
		logger.Info("Resource updated")
	}

	return util.ObjectReference(live, gvk), nil
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
