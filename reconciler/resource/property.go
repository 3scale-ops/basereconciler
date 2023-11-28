package resource

import (
	"fmt"

	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	"github.com/ohler55/ojg/jp"
	"k8s.io/apimachinery/pkg/api/equality"
)

type PropertyDelta int

const (
	MissingInBoth                     PropertyDelta = 0
	MissingFromDesiredPresentInTarget PropertyDelta = 1
	PresentInDesiredMissingFromTarget PropertyDelta = 2
	PresentInBoth                     PropertyDelta = 3
)

type Property string

func (p Property) JSONPath() string { return string(p) }

func (p Property) Reconcile(target, desired map[string]any, logger logr.Logger) (bool, error) {
	expr, err := jp.ParseString(p.JSONPath())
	if err != nil {
		return false, fmt.Errorf("unable to parse JSONPath '%s': %w", p.JSONPath(), err)
	}

	desiredVal := expr.Get(desired)
	val := expr.Get(target)
	if len(desiredVal) > 1 || len(val) > 1 {
		return false, fmt.Errorf("multi-valued JSONPath (%s) not supported when reconciling properties", p.JSONPath())
	}

	// if p.checkMetadataEquality(val, desiredVal) {
	// 	return false, nil
	// }

	// if p == "metadata.annotations" {
	// 	spew.Printf("\n#### metadata.annotations: \n")
	// 	// spew.Printf("\t %+v\n", val)
	// 	// spew.Printf("\t %+v\n", desiredVal)
	// 	spew.Dump(val)
	// 	spew.Dump(desiredVal)
	// }

	switch delta(len(desiredVal), len(val)) {

	case MissingInBoth:
		// nothing to do
		return false, nil

	case MissingFromDesiredPresentInTarget:
		// delete property from target
		logger.V(1).Info("differences detected", "op", "delete", "path", p.JSONPath(), "diff", cmp.Diff(val[0], nil))
		if err := expr.Del(target); err != nil {
			return false, fmt.Errorf("usable to delete JSONPath '%s'", p.JSONPath())
		}
		return true, nil

	case PresentInDesiredMissingFromTarget:
		// add property to target
		logger.V(1).Info("differences detected", "op", "add", "path", p.JSONPath(), "diff", cmp.Diff(nil, desiredVal[0]))
		if err := expr.Set(target, desiredVal[0]); err != nil {
			return false, fmt.Errorf("usable to add value '%v' in JSONPath '%s'", desiredVal[0], p.JSONPath())
		}
		return true, nil

	case PresentInBoth:
		// replace property in target if values differ
		if !equality.Semantic.DeepEqual(desiredVal[0], val[0]) {
			logger.V(1).Info("differences detected", "op", "replace", "path", p.JSONPath(), "diff", cmp.Diff(val[0], desiredVal[0]))
			if err := expr.Set(target, desiredVal[0]); err != nil {
				return false, fmt.Errorf("usable to replace value '%v' in JSONPath '%s'", desiredVal[0], p.JSONPath())
			}
			return true, nil
		}

	}

	return false, nil
}

func delta(a, b int) PropertyDelta {
	return PropertyDelta(a<<1 + b)
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

// func (p Property) checkMetadataEquality(a, b []any) bool {
// 	if p == "metadata.annotations" || p == "metadata.labels" {
// 		return emptyMap(a) && emptyMap(b)
// 	}
// 	if p == "metadata.finalizers" {
// 		return emptyList(a) && emptyList(b)
// 	}
// 	return false
// }

// func emptyMap(a []any) bool {
// 	return reflect.DeepEqual(a, []any{map[string]any{}}) || reflect.DeepEqual(a, []any{})
// }

// func emptyList(a []any) bool {
// 	return reflect.DeepEqual(a, []any{[]any{}}) || reflect.DeepEqual(a, []any{})
// }
