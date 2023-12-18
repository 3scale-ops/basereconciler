package util

import (
	"context"
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestGetItems(t *testing.T) {
	type args struct {
		list client.ObjectList
	}
	tests := []struct {
		name string
		args args
		want []client.Object
	}{
		{
			name: "Returns items of a corev1.ServiceList as []client.Object",
			args: args{
				list: &corev1.ServiceList{
					Items: []corev1.Service{
						{ObjectMeta: metav1.ObjectMeta{Name: "one"}},
						{ObjectMeta: metav1.ObjectMeta{Name: "two"}},
					},
				},
			},
			want: []client.Object{
				&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "one"}},
				&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "two"}},
			},
		},
		{
			name: "Returns items of a corev1.PodList as []client.Object",
			args: args{
				list: &corev1.PodList{
					Items: []corev1.Pod{
						{ObjectMeta: metav1.ObjectMeta{Name: "one"}},
						{ObjectMeta: metav1.ObjectMeta{Name: "two"}},
					},
				},
			},
			want: []client.Object{
				&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "one"}},
				&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "two"}},
			},
		}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetItems(tt.args.list); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetItems() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewObjectFromGVK(t *testing.T) {
	type args struct {
		gvk schema.GroupVersionKind
		s   *runtime.Scheme
	}
	tests := []struct {
		name    string
		args    args
		want    client.Object
		wantErr bool
	}{
		{
			name: "Returns an object of the given gvk",
			args: args{
				gvk: schema.GroupVersionKind{
					Group:   "",
					Version: "v1",
					Kind:    "Service",
				},
				s: scheme.Scheme,
			},
			want:    &corev1.Service{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewObjectFromGVK(tt.args.gvk, tt.args.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewObjectFromGVK() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewObjectFromGVK() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewObjectListFromGVK(t *testing.T) {
	type args struct {
		gvk schema.GroupVersionKind
		s   *runtime.Scheme
	}
	tests := []struct {
		name    string
		args    args
		want    client.ObjectList
		wantErr bool
	}{
		{
			name: "Returns a list when given an Object gvk",
			args: args{
				gvk: schema.GroupVersionKind{
					Group:   "",
					Version: "v1",
					Kind:    "Service",
				},
				s: scheme.Scheme,
			},
			want:    &corev1.ServiceList{},
			wantErr: false,
		},
		{
			name: "Returns a list when given an ObjectList gvk",
			args: args{
				gvk: schema.GroupVersionKind{
					Group:   "",
					Version: "v1",
					Kind:    "ServiceList",
				},
				s: scheme.Scheme,
			},
			want:    &corev1.ServiceList{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewObjectListFromGVK(tt.args.gvk, tt.args.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewFromGVK() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewFromGVK() = %v, want %v", got, tt.want)
			}
		})
	}
}

type testResource struct {
	*corev1.Service
}

func (o *testResource) Default() {
	o.SetLabels(map[string]string{"key": "value"})
}
func TestResourceDefaulter(t *testing.T) {
	type args struct {
		o *testResource
	}
	tests := []struct {
		name  string
		args  args
		check func(client.Object) bool
	}{
		{
			name: "Calling the ResourceDefaulter applies defaults",
			args: args{
				o: &testResource{
					Service: &corev1.Service{ObjectMeta: metav1.ObjectMeta{
						Name: "test", Namespace: "ns",
					}},
				},
			},
			check: func(o client.Object) bool {
				if v, ok := o.GetLabels()["key"]; ok {
					if v == "value" {
						return true
					}
				}
				return false
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = ResourceDefaulter(tt.args.o)(context.TODO(), nil, tt.args.o)
			if !tt.check(tt.args.o) {
				t.Errorf("ResourceDefaulter() got %v", tt.args.o)
			}
		})
	}
}
