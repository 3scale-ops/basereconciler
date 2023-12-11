package resource

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/3scale-ops/basereconciler/config"
	"github.com/3scale-ops/basereconciler/util"
	"github.com/go-logr/logr"
	"github.com/nsf/jsondiff"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// CreateOrUpdate cretes or updates resources. The function receives several paremters:
//   - ctx: the context
//   - cl: the kubernetes API client
//   - scheme: the kubernetes API scheme
//   - owner: the object that owns the resource. Used to set the OwnerReference in the resource
//   - template: the struct that describes how the resource needs to be reconciled. It must implement
//     the TemplateInterface interface. When template.GetEnsureProperties is not set or an empty list, this
//     function will lookup for configuration in the global configuration (see pacakge config).
func CreateOrUpdate(ctx context.Context, cl client.Client, scheme *runtime.Scheme,
	owner client.Object, template TemplateInterface) (*corev1.ObjectReference, error) {

	desired, err := template.Build(ctx, cl, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to build template: %w", err)
	}

	key := util.ObjectKey(desired)
	gvk, err := apiutil.GVKForObject(desired, scheme)
	if err != nil {
		return nil, err
	}
	logger := logr.FromContextOrDiscard(ctx).WithValues("gvk", gvk, "resource", desired.GetName())

	live, err := util.NewObjectFromGVK(gvk, scheme)
	if err != nil {
		return nil, wrapError("unable to create object from GVK", key, gvk, err)
	}
	err = cl.Get(ctx, key, live)
	if err != nil {
		if errors.IsNotFound(err) {
			if template.Enabled() {
				if err := controllerutil.SetControllerReference(owner, desired, scheme); err != nil {
					return nil, wrapError("unable to set controller reference", key, gvk, err)
				}
				err = cl.Create(ctx, desired)
				if err != nil {
					return nil, wrapError("unable to create resource", key, gvk, err)
				}
				logger.Info("resource created")
				return util.ObjectReference(desired, gvk), nil

			} else {
				return nil, nil
			}
		}
		return nil, wrapError("unable to get resource", key, gvk, err)
	}

	/* Delete and return if not enabled */
	if !template.Enabled() {
		err := cl.Delete(ctx, live)
		if err != nil {
			return nil, wrapError("unable to delete object", key, gvk, err)
		}
		logger.Info("resource deleted")
		return nil, nil
	}

	ensure, ignore, err := reconcilerConfig(template, gvk)
	if err != nil {
		return nil, wrapError("unable to retrieve config for resource reconciler", key, gvk, err)
	}

	// normalizedLive is a struct that will be populated with only the reconciled
	// properties and their respective live values. It will be used to compare it with
	// the desire and determine in an update is required.
	normalizedLive, err := util.NewObjectFromGVK(gvk, scheme)
	if err != nil {
		return nil, wrapError("unable to create object from GVK", key, gvk, err)
	}
	normalizedLive.SetName(desired.GetName())
	normalizedLive.SetNamespace(desired.GetNamespace())

	// convert to unstructured
	u_desired, err := runtime.DefaultUnstructuredConverter.ToUnstructured(desired)
	if err != nil {
		return nil, wrapError("unable to convert to unstructured", key, gvk, err)

	}

	u_live, err := runtime.DefaultUnstructuredConverter.ToUnstructured(live)
	if err != nil {
		return nil, wrapError("unable to convert to unstructured", key, gvk, err)
	}

	u_normalizedLive, err := runtime.DefaultUnstructuredConverter.ToUnstructured(normalizedLive)
	if err != nil {
		return nil, wrapError("unable to convert to unstructured", key, gvk, err)
	}

	// reconcile properties
	for _, property := range ensure {
		if err := property.reconcile(u_live, u_desired, u_normalizedLive, logger); err != nil {
			return nil, wrapError(fmt.Sprintf("unable to reconcile property %s", property), key, gvk, err)
		}
	}

	// ignore properties
	for _, property := range ignore {
		for _, m := range []map[string]any{u_live, u_desired, u_normalizedLive} {
			if err := property.ignore(m); err != nil {
				return nil, wrapError(fmt.Sprintf("unable to ignore property %s", property), key, gvk, err)
			}
		}
	}

	// do the comparison using structs so "equality.Semantic" can be used
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u_normalizedLive, normalizedLive); err != nil {
		return nil, wrapError("unable to convert from unstructured", key, gvk, err)
	}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u_desired, desired); err != nil {
		return nil, wrapError("unable to convert from unstructured", key, gvk, err)
	}
	if !equality.Semantic.DeepEqual(normalizedLive, desired) {
		logger.V(1).Info("resource update required", "diff", printfDiff(normalizedLive, desired))
		err := cl.Update(ctx, client.Object(&unstructured.Unstructured{Object: u_live}))
		if err != nil {
			return nil, wrapError("unable to update resource", key, gvk, err)
		}
		logger.Info("Resource updated")
	}

	return util.ObjectReference(live, gvk), nil
}

func printfDiff(a, b client.Object) string {
	ajson, err := json.Marshal(a)
	if err != nil {
		return fmt.Errorf("unable to log differences: %w", err).Error()
	}
	bjson, err := json.Marshal(b)
	if err != nil {
		return fmt.Errorf("unable to log differences: %w", err).Error()
	}
	opts := jsondiff.DefaultJSONOptions()
	opts.SkipMatches = true
	opts.Indent = "\t"
	_, diff := jsondiff.Compare(ajson, bjson, &opts)
	return diff
}

func wrapError(msg string, key types.NamespacedName, gvk schema.GroupVersionKind, err error) error {
	return fmt.Errorf("%s %s/%s/%s: %w", msg, gvk.Kind, key.Name, key.Namespace, err)
}

func reconcilerConfig(template TemplateInterface, gvk schema.GroupVersionKind) ([]Property, []Property, error) {

	if len(template.GetEnsureProperties()) == 0 {
		cfg, err := config.GetDefaultReconcileConfigForGVK(gvk)
		if err != nil {
			return nil, nil, err
		}
		ensure := util.ConvertStringSlice[string, Property](cfg.EnsureProperties)
		ignore := util.ConvertStringSlice[string, Property](cfg.IgnoreProperties)
		return ensure, ignore, nil
	}

	return template.GetEnsureProperties(), template.GetIgnoreProperties(), nil
}
