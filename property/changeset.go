package property

import (
	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/api/equality"
)

type ChangeSet[T any] struct {
	path    string
	current *T
	desired *T
}

func NewChangeSet[T any](path string, current *T, desired *T) *ChangeSet[T] {
	return &ChangeSet[T]{path: path, current: current, desired: desired}
}

// EnsureDesired checks if two structs are equal. If they are not, current is overwriten
// with the value of desired. Bool flag is returned to indicate if the value of current was changed.
func (set *ChangeSet[T]) EnsureDesired(logger logr.Logger) bool {

	if equality.Semantic.DeepEqual(set.current, set.desired) {
		return false
	}

	logger.V(1).Info("differences detected", "path", set.path, "diff", cmp.Diff(set.current, set.desired))
	if set.desired == nil {
		set.current = nil
	} else {
		*set.current = *set.desired
	}

	return true
}

type ReconcilableProperty interface {
	EnsureDesired(logger logr.Logger) bool
}

func EnsureDesired(logger logr.Logger, changeSets ...ReconcilableProperty) bool {
	changed := false

	for _, set := range changeSets {

		if set.EnsureDesired(logger) {
			changed = true
		}
	}

	return changed
}
