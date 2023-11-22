package config

import (
	"fmt"
	"reflect"

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

func GetAnnotationsDomain() string       { return config.annotationsDomain }
func SetAnnotationsDomain(domain string) { config.annotationsDomain = domain }

func EnableResourcePruner()         { config.resourcePruner = true }
func DisableResourcePruner()        { config.resourcePruner = false }
func IsResourcePrunerEnabled() bool { return config.resourcePruner }

func GetDefaultReconcileConfigForGVK(gvk schema.GroupVersionKind) (ReconcileConfigForGVK, error) {
	if cfg, ok := config.defaultResourceReconcileConfig[gvk.String()]; ok {
		return cfg, nil
	} else if defcfg, ok := config.defaultResourceReconcileConfig["*"]; ok {
		return defcfg, nil
	} else {
		return ReconcileConfigForGVK{}, fmt.Errorf("no config registered for gvk %s", gvk)
	}
}
func SetDefaultReconcileConfigForGVK(gvk schema.GroupVersionKind, cfg ReconcileConfigForGVK) {
	if reflect.DeepEqual(gvk, schema.GroupVersionKind{}) {
		config.defaultResourceReconcileConfig["*"] = cfg
	} else {
		config.defaultResourceReconcileConfig[gvk.String()] = cfg
	}
}
