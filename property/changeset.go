package property

import (
	"fmt"
	"strings"

	"github.com/3scale-ops/basereconciler/util"
	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	"github.com/ohler55/ojg/jp"
	"k8s.io/apimachinery/pkg/api/equality"
	// metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ReconcilableProperty interface {
	Apply(logr.Logger) bool
}

func EnsureDesired(logger logr.Logger, changeSets ...ReconcilableProperty) bool {
	changed := false

	for _, set := range changeSets {

		if set.Apply(logger) {
			changed = true
		}
	}

	return changed
}

type IgnoreNested string

type ChangeSet[T any] struct {
	path    string
	current *T
	desired *T
	ignore  []IgnoreNested
}

func NewChangeSet[T any](path string, current *T, desired *T, ignore ...IgnoreNested) *ChangeSet[T] {
	return &ChangeSet[T]{path: path, current: current, desired: desired, ignore: ignore}
}

// Apply checks if two structs are equal. If they are not, current is overwriten
// with the value of desired. Bool flag is returned to indicate if the value of current was changed.
func (set *ChangeSet[T]) Apply(logger logr.Logger) bool {

	for _, jsonpath := range set.ignore {
		if err := set.removeMatchingProperties(string(jsonpath)); err != nil {
			// log the error but keep going, this is not a critical error
			// most likely the jsonpath expression is not correct
			logger.Error(err, fmt.Sprintf("unable to ignore expression '%s' in ChangeSet", jsonpath), "path", set.path)
		}
	}

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

func (set *ChangeSet[T]) removeMatchingProperties(jsonpath string) error {

	relativeJSONPath := strings.TrimPrefix(strings.TrimPrefix(jsonpath, set.path), ".")

	expr, err := jp.ParseString(relativeJSONPath)
	if err != nil {
		return err
	}

	a := util.MustStructToMap(set.current)
	b := util.MustStructToMap(set.desired)

	err = expr.Del(a)
	if err != nil {
		return err
	}
	err = expr.Del(b)
	if err != nil {
		return err
	}

	moda := new(T)
	modb := new(T)
	util.MustMapToStruct(a, moda)
	util.MustMapToStruct(b, modb)
	set.current = moda
	set.desired = modb

	return nil
}
