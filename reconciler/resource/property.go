package resource

import (
	"fmt"

	"github.com/go-logr/logr"
	"github.com/ohler55/ojg/jp"
	"k8s.io/apimachinery/pkg/api/equality"
)

type PropertyDelta int

const (
	MissingInBoth                   PropertyDelta = 0
	MissingFromDesiredPresentInLive PropertyDelta = 1
	PresentInDesiredMissingFromLive PropertyDelta = 2
	PresentInBoth                   PropertyDelta = 3
)

type Property string

func (p Property) JSONPath() string { return string(p) }

func (p Property) Reconcile(live, desired, diff map[string]any, logger logr.Logger) error {
	expr, err := jp.ParseString(p.JSONPath())
	if err != nil {
		return fmt.Errorf("unable to parse JSONPath '%s': %w", p.JSONPath(), err)
	}

	desiredVal := expr.Get(desired)
	liveVal := expr.Get(live)
	if len(desiredVal) > 1 || len(liveVal) > 1 {
		return fmt.Errorf("multi-valued JSONPath (%s) not supported when reconciling properties", p.JSONPath())
	}

	// store the live value for later comparison
	if len(liveVal) != 0 {
		if err := expr.Set(diff, liveVal[0]); err != nil {
			return fmt.Errorf("usable to add value '%v' in JSONPath '%s'", liveVal[0], p.JSONPath())
		}
	}

	switch delta(len(desiredVal), len(liveVal)) {

	case MissingInBoth:
		// nothing to do
		return nil

	case MissingFromDesiredPresentInLive:
		// delete property from target
		// logger.V(1).Info("differences detected", "op", "delete", "path", p.JSONPath(), "diff", cmp.Diff(liveVal[0], nil))
		if err := expr.Del(live); err != nil {
			return fmt.Errorf("usable to delete JSONPath '%s'", p.JSONPath())
		}
		return nil

	case PresentInDesiredMissingFromLive:
		// add property to target
		// logger.V(1).Info("differences detected", "op", "add", "path", p.JSONPath(), "diff", cmp.Diff(nil, desiredVal[0]))
		if err := expr.Set(live, desiredVal[0]); err != nil {
			return fmt.Errorf("usable to add value '%v' in JSONPath '%s'", desiredVal[0], p.JSONPath())
		}
		return nil

	case PresentInBoth:
		// replace property in target if values differ
		if !equality.Semantic.DeepEqual(desiredVal[0], liveVal[0]) {
			// logger.V(1).Info("differences detected", "op", "replace", "path", p.JSONPath(), "diff", cmp.Diff(liveVal[0], desiredVal[0]))
			if err := expr.Set(live, desiredVal[0]); err != nil {
				return fmt.Errorf("usable to replace value '%v' in JSONPath '%s'", desiredVal[0], p.JSONPath())
			}
			return nil
		}

	}

	return nil
}

func delta(a, b int) PropertyDelta {
	return PropertyDelta(a<<1 + b)
}

func (p Property) Ignore(m map[string]any) error {
	expr, err := jp.ParseString(p.JSONPath())
	if err != nil {
		return fmt.Errorf("unable to parse JSONPath '%s': %w", p.JSONPath(), err)
	}
	if err = expr.Del(m); err != nil {
		return fmt.Errorf("unable to parse delete JSONPath '%s' from unstructured desired: %w", p.JSONPath(), err)
	}
	return nil
}
