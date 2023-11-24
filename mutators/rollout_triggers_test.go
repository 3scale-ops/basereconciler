package mutators

import (
	"context"
	"testing"

	"github.com/3scale-ops/basereconciler/util"
	"github.com/google/go-cmp/cmp"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestRolloutTrigger_GetHash(t *testing.T) {
	type fields struct {
		Name          string
		ConfigMapName *string
		SecretName    *string
	}
	type args struct {
		ctx       context.Context
		cl        client.Client
		namespace string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "Secret hash",
			fields: fields{
				Name:       "secret",
				SecretName: util.Pointer("secret"),
			},
			args: args{
				ctx: context.TODO(),
				cl: fake.NewClientBuilder().WithObjects(
					&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "secret", Namespace: "ns"},
						Data: map[string][]byte{"key": []byte("data")},
					},
				).Build(),
				namespace: "ns",
			},
			want:    util.Hash(map[string][]byte{"key": []byte("data")}),
			wantErr: false,
		},
		{
			name: "ConfigMap hash",
			fields: fields{
				Name:          "cm",
				ConfigMapName: util.Pointer("cm"),
			},
			args: args{
				ctx: context.TODO(),
				cl: fake.NewClientBuilder().WithObjects(
					&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "cm", Namespace: "ns"},
						Data: map[string]string{"key": "data"},
					},
				).Build(),
				namespace: "ns",
			},
			want:    util.Hash(map[string]string{"key": "data"}),
			wantErr: false,
		},
		{
			name: "Returns '' if secret does not exist",
			fields: fields{
				Name:       "secret",
				SecretName: util.Pointer("secret"),
			},
			args: args{
				ctx:       context.TODO(),
				cl:        fake.NewClientBuilder().Build(),
				namespace: "ns",
			},
			want:    "",
			wantErr: false,
		},
		{
			name: "Returns '' if cm does not exist",
			fields: fields{
				Name:          "secret",
				ConfigMapName: util.Pointer("secret"),
			},
			args: args{
				ctx:       context.TODO(),
				cl:        fake.NewClientBuilder().Build(),
				namespace: "ns",
			},
			want:    "",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rt := RolloutTrigger{
				Name:          tt.fields.Name,
				ConfigMapName: tt.fields.ConfigMapName,
				SecretName:    tt.fields.SecretName,
			}
			got, err := rt.GetHash(tt.args.ctx, tt.args.cl, tt.args.namespace)
			if (err != nil) != tt.wantErr {
				t.Errorf("RolloutTrigger.GetHash() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("RolloutTrigger.GetHash() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRolloutTrigger_GetAnnotationKey(t *testing.T) {
	type fields struct {
		Name          string
		ConfigMapName *string
		SecretName    *string
	}
	type args struct {
		annotationsDomain string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rt := RolloutTrigger{
				Name:          tt.fields.Name,
				ConfigMapName: tt.fields.ConfigMapName,
				SecretName:    tt.fields.SecretName,
			}
			if got := rt.GetAnnotationKey(tt.args.annotationsDomain); got != tt.want {
				t.Errorf("RolloutTrigger.GetAnnotationKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRolloutTrigger_AddToDeployment(t *testing.T) {
	type fields struct {
		Name          string
		ConfigMapName *string
		SecretName    *string
	}
	type args struct {
		annotationsDomain string
		ctx               context.Context
		cl                client.Client
		instance          client.Object
		desired           client.Object
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    client.Object
		wantErr bool
	}{
		{
			name: "Adds rollout annotation to Deplotment's pods",
			fields: fields{
				Name:          "cm",
				ConfigMapName: util.Pointer("cm"),
			},
			args: args{
				annotationsDomain: "example.com",
				ctx:               context.TODO(),
				cl: fake.NewClientBuilder().WithObjects(
					&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "cm", Namespace: "ns"},
						Data: map[string]string{"key": "data"}},
					&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "cm", Namespace: "ns"}},
				).Build(),
				instance: nil,
				desired:  &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "cm", Namespace: "ns"}},
			},
			want: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: "cm", Namespace: "ns"},
				Spec: appsv1.DeploymentSpec{Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{"example.com/cm.configmap-hash": util.Hash(map[string]string{"key": "data"})},
				}}},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trigger := RolloutTrigger{
				Name:          tt.fields.Name,
				ConfigMapName: tt.fields.ConfigMapName,
				SecretName:    tt.fields.SecretName,
			}
			err := trigger.AddToDeployment(tt.args.annotationsDomain)(tt.args.ctx, tt.args.cl, tt.args.instance, tt.args.desired)
			if (err != nil) != tt.wantErr {
				t.Errorf("RolloutTrigger.AddToDeployment() error = %v, wantErr %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(tt.args.desired, tt.want, util.IgnoreProperty("ResourceVersion")); len(diff) > 0 {
				t.Errorf("RolloutTrigger.AddToDeployment() diff = %v", diff)
			}
		})
	}
}

func TestRolloutTrigger_AddToStatefulSet(t *testing.T) {
	type fields struct {
		Name          string
		ConfigMapName *string
		SecretName    *string
	}
	type args struct {
		annotationsDomain string
		ctx               context.Context
		cl                client.Client
		instance          client.Object
		desired           client.Object
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    client.Object
		wantErr bool
	}{
		{
			name: "Adds rollout annotation to Deplotment's pods",
			fields: fields{
				Name:          "cm",
				ConfigMapName: util.Pointer("cm"),
			},
			args: args{
				annotationsDomain: "example.com",
				ctx:               context.TODO(),
				cl: fake.NewClientBuilder().WithObjects(
					&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "cm", Namespace: "ns"},
						Data: map[string]string{"key": "data"}},
					&appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: "cm", Namespace: "ns"}},
				).Build(),
				instance: nil,
				desired:  &appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: "cm", Namespace: "ns"}},
			},
			want: &appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{Name: "cm", Namespace: "ns"},
				Spec: appsv1.StatefulSetSpec{Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{"example.com/cm.configmap-hash": util.Hash(map[string]string{"key": "data"})},
				}}},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trigger := RolloutTrigger{
				Name:          tt.fields.Name,
				ConfigMapName: tt.fields.ConfigMapName,
				SecretName:    tt.fields.SecretName,
			}
			err := trigger.AddToStatefulSet(tt.args.annotationsDomain)(tt.args.ctx, tt.args.cl, tt.args.instance, tt.args.desired)
			if (err != nil) != tt.wantErr {
				t.Errorf("RolloutTrigger.AddToStatefulSet() error = %v, wantErr %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(tt.args.desired, tt.want, util.IgnoreProperty("ResourceVersion")); len(diff) > 0 {
				t.Errorf("RolloutTrigger.AddToStatefulSet() diff = %v", diff)
			}
		})
	}
}