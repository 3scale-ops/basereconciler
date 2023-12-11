// Package config provides global configuration for the basereconciler. The package
// provides some barebones configuration, but in most cases the user will want to
// tailor this configuration to the needs and requirements of the specific controller/s.
package config

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

type ReconcileConfigForGVK struct {
	EnsureProperties []string
	IgnoreProperties []string
}

var config = struct {
	annotationsDomain              string
	resourcePruner                 bool
	defaultResourceReconcileConfig map[string]ReconcileConfigForGVK
}{
	annotationsDomain: "basereconciler.3cale.net",
	resourcePruner:    true,
	defaultResourceReconcileConfig: map[string]ReconcileConfigForGVK{
		"*": {
			EnsureProperties: []string{
				"metadata.annotations",
				"metadata.labels",
				"spec",
			},
			IgnoreProperties: []string{},
		},
	},
}

// GetAnnotationsDomain returns the globally configured annotations domain. The annotations
// domain is used for the rollout trigger annotations (see the mutators package) and the resource
// finalizers
func GetAnnotationsDomain() string { return config.annotationsDomain }

// SetAnnotationsDomain globally configures the annotations domain. The annotations
// domain is used for the rollout trigger annotations (see the mutators package) and the resource
// finalizers
func SetAnnotationsDomain(domain string) { config.annotationsDomain = domain }

// EnableResourcePruner enables the resource pruner. The resource pruner keeps track of
// the owned resources of a given custom resource and deletes all that are not present in the list
// of managed resoures to reconcile.
func EnableResourcePruner() { config.resourcePruner = true }

// DisableResourcePruner disables the resource pruner. The resource pruner keeps track of
// the owned resources of a given custom resource and deletes all that are not present in the list
// of managed resoures to reconcile.
func DisableResourcePruner() { config.resourcePruner = false }

// IsResourcePrunerEnabled returs a boolean indicating wheter the resource pruner is enabled or not.
func IsResourcePrunerEnabled() bool { return config.resourcePruner }

// GetDefaultReconcileConfigForGVK returns the default configuration that instructs basereconciler how to reconcile
// a given kubernetes GVK (GroupVersionKind). This default config will be used if the "resource.Template" object (see
// the resource package) does not specify a configuration itself.
// When the passed GVK does not match any of the configured, this function returns the "wildcard", which is a default
// set of basic reconclie rules that the reconciler will try to use when no other configuration is available.
func GetDefaultReconcileConfigForGVK(gvk schema.GroupVersionKind) (ReconcileConfigForGVK, error) {
	if cfg, ok := config.defaultResourceReconcileConfig[gvk.String()]; ok {
		return cfg, nil
	} else if defcfg, ok := config.defaultResourceReconcileConfig["*"]; ok {
		return defcfg, nil
	} else {
		return ReconcileConfigForGVK{}, fmt.Errorf("no config registered for gvk %s", gvk)
	}
}

// SetDefaultReconcileConfigForGVK sets the default configuration that instructs basereconciler how to reconcile
// a given kubernetes GVK (GroupVersionKind). This default config will be used if the "resource.Template" object (see
// the resource package) does not specify a configuration itself.
// If the passed GVK is an empty one ("schema.GroupVersionKind{}"), the function will set the wildcard instead, which
// is a default set of basic reconclie rules that the reconciler will try to use when no other configuration is available.
func SetDefaultReconcileConfigForGVK(gvk schema.GroupVersionKind, cfg ReconcileConfigForGVK) {
	if gvk.Empty() {
		config.defaultResourceReconcileConfig["*"] = cfg
	} else {
		config.defaultResourceReconcileConfig[gvk.String()] = cfg
	}
}
