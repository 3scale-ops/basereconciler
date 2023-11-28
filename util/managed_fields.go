package util

import (
	"bytes"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/structured-merge-diff/v4/fieldpath"
)

// EmptyFields represents a set with no paths
// It looks like metav1.Fields{Raw: []byte("{}")}
var EmptyFields = func() metav1.FieldsV1 {
	f, err := SetToFields(*fieldpath.NewSet())
	if err != nil {
		panic("should never happen")
	}
	return f
}()

// SetToFields creates a tree of fields from an input set of paths
func SetToFields(s fieldpath.Set) (f metav1.FieldsV1, err error) {
	f.Raw, err = s.ToJSON()
	return f, err
}

// FieldsToSet creates a set paths from an input trie of fields
func FieldsToSet(f metav1.FieldsV1) (s fieldpath.Set, err error) {
	err = s.FromJSON(bytes.NewReader(f.Raw))
	return s, err
}

func DecodeVersionedSet(encodedVersionedSet *metav1.ManagedFieldsEntry) (versionedSet fieldpath.VersionedSet, err error) {
	fields := EmptyFields
	if encodedVersionedSet.FieldsV1 != nil {
		fields = *encodedVersionedSet.FieldsV1
	}
	set, err := FieldsToSet(fields)
	if err != nil {
		return nil, fmt.Errorf("error decoding set: %v", err)
	}
	return fieldpath.NewVersionedSet(&set, fieldpath.APIVersion(encodedVersionedSet.APIVersion), encodedVersionedSet.Operation == metav1.ManagedFieldsOperationApply), nil
}
