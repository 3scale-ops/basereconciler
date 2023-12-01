package config

import (
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestGetDefaultReconcileConfigForGVK(t *testing.T) {
	type args struct {
		gvk schema.GroupVersionKind
	}
	tests := []struct {
		name    string
		prepare func()
		args    args
		want    ReconcileConfigForGVK
		wantErr bool
	}{
		{
			name:    "Returns the wildcard config",
			prepare: func() {},
			args: args{
				gvk: schema.GroupVersionKind{},
			},
			want: ReconcileConfigForGVK{
				EnsureProperties: []string{
					"metadata.annotations",
					"metadata.labels",
					"spec",
				},
				IgnoreProperties: []string{},
			},
			wantErr: false,
		},
		{
			name: "Returns config for a GVK",
			prepare: func() {
				config.defaultResourceReconcileConfig["apps/v1, Kind=Deployment"] = ReconcileConfigForGVK{
					EnsureProperties: []string{"a.b.c"},
					IgnoreProperties: []string{"x.y"},
				}
			},
			args: args{
				gvk: schema.FromAPIVersionAndKind("apps/v1", "Deployment"),
			},
			want: ReconcileConfigForGVK{
				EnsureProperties: []string{"a.b.c"},
				IgnoreProperties: []string{"x.y"},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare()
			got, err := GetDefaultReconcileConfigForGVK(tt.args.gvk)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetDefaultReconcileConfigForGVK() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetDefaultReconcileConfigForGVK() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetDefaultReconcileConfigForGVK(t *testing.T) {
	type args struct {
		gvk schema.GroupVersionKind
		cfg ReconcileConfigForGVK
	}
	tests := []struct {
		name  string
		args  args
		check func() ReconcileConfigForGVK
	}{
		{
			name: "Sets the wildcard config",
			args: args{
				gvk: schema.GroupVersionKind{},
				cfg: ReconcileConfigForGVK{
					EnsureProperties: []string{"a.b.c"},
					IgnoreProperties: []string{"x.y.z"},
				},
			},
			check: func() ReconcileConfigForGVK {
				return config.defaultResourceReconcileConfig["*"]
			},
		},
		{
			name: "Sets config for the given GVK",
			args: args{
				gvk: schema.FromAPIVersionAndKind("apps/v1", "StatefulSet"),
				cfg: ReconcileConfigForGVK{
					EnsureProperties: []string{"a.b.c"},
					IgnoreProperties: []string{"x.y.z"},
				},
			},
			check: func() ReconcileConfigForGVK {
				return config.defaultResourceReconcileConfig["apps/v1, Kind=StatefulSet"]
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetDefaultReconcileConfigForGVK(tt.args.gvk, tt.args.cfg)
			if got := tt.check(); !reflect.DeepEqual(got, tt.args.cfg) {
				t.Errorf("SetDefaultReconcileConfigForGVK() = %v, want %v", got, tt.args.cfg)
			}
		})
	}
}
