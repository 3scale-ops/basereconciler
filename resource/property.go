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

func (p Property) Reconcile(u_live, u_desired, u_normalizedLive map[string]any, logger logr.Logger) error {
	expr, err := jp.ParseString(p.JSONPath())
	if err != nil {
		return fmt.Errorf("unable to parse JSONPath '%s': %w", p.JSONPath(), err)
	}

	desiredVal := expr.Get(u_desired)
	liveVal := expr.Get(u_live)
	if len(desiredVal) > 1 || len(liveVal) > 1 {
		return fmt.Errorf("multi-valued JSONPath (%s) not supported when reconciling properties", p.JSONPath())
	}

	// store the live value for later comparison in u_normalizedLive
	if len(liveVal) != 0 {
		if err := expr.Set(u_normalizedLive, liveVal[0]); err != nil {
			return fmt.Errorf("usable to add value '%v' in JSONPath '%s'", liveVal[0], p.JSONPath())
		}
	}

	switch delta(len(desiredVal), len(liveVal)) {

	case MissingInBoth:
		// nothing to do
		return nil

	case MissingFromDesiredPresentInLive:
		// delete property from u_live
		if err := expr.Del(u_live); err != nil {
			return fmt.Errorf("usable to delete JSONPath '%s'", p.JSONPath())
		}
		return nil

	case PresentInDesiredMissingFromLive:
		// add property to u_live
		if err := expr.Set(u_live, desiredVal[0]); err != nil {
			return fmt.Errorf("usable to add value '%v' in JSONPath '%s'", desiredVal[0], p.JSONPath())
		}
		return nil

	case PresentInBoth:
		// replace property in u_live if values differ
		if !equality.Semantic.DeepEqual(desiredVal[0], liveVal[0]) {
			if err := expr.Set(u_live, desiredVal[0]); err != nil {
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
		return fmt.Errorf("unable to parse delete JSONPath '%s' from unstructured: %w", p.JSONPath(), err)
	}
	return nil
}
