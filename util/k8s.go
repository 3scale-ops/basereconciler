package util

import (
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ObjectKey(o client.Object) types.NamespacedName {
	return types.NamespacedName{Name: o.GetName(), Namespace: o.GetNamespace()}
}

// this is an ugly function to retrieve the list of Items from a
// client.ObjectList because the interface doesn't have a GetItems
// method
func GetItems(list client.ObjectList) []client.Object {
	items := []client.Object{}
	values := reflect.ValueOf(list).Elem().FieldByName("Items")
	for i := 0; i < values.Len(); i++ {
		item := values.Index(i)
		if item.Kind() == reflect.Pointer {
			items = append(items, item.Interface().(client.Object))
		} else {
			items = append(items, item.Addr().Interface().(client.Object))
		}
	}

	return items
}

func IsBeingDeleted(o client.Object) bool {
	return !o.GetDeletionTimestamp().IsZero()
}

func NewFromGVK(gvk schema.GroupVersionKind, s *runtime.Scheme) (client.Object, error) {
	o, err := s.New(gvk)
	if err != nil {
		return nil, err
	}
	new, ok := o.(client.Object)
	if !ok {
		return nil, fmt.Errorf("runtime object %T does not implement client.Object", o)
	}
	return new, nil
}
