package reconciler

import (
	"context"
	"reflect"
	"testing"

	"github.com/3scale-ops/basereconciler/config"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestReconciler_pruneOrphaned(t *testing.T) {
	type fields struct {
		Client    client.Client
		Scheme    *runtime.Scheme
		seenTypes []schema.GroupVersionKind
	}
	type args struct {
		ctx     context.Context
		owner   client.Object
		managed []corev1.ObjectReference
	}
	type check struct {
		absent bool
		obj    client.Object
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []check
		wantErr bool
	}{
		{
			name: "Prunes resource 1",
			fields: fields{
				Client: fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(
					&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
						Name: "deploy", Namespace: "ns",
						OwnerReferences: []metav1.OwnerReference{{APIVersion: "v1", Kind: "ServiceAccount", Name: "owner"}}}},
					&autoscalingv2.HorizontalPodAutoscaler{ObjectMeta: metav1.ObjectMeta{
						Name: "hpa", Namespace: "ns",
						OwnerReferences: []metav1.OwnerReference{{APIVersion: "v1", Kind: "ServiceAccount", Name: "owner"}}}},
					&policyv1.PodDisruptionBudget{ObjectMeta: metav1.ObjectMeta{
						Name: "pdb", Namespace: "ns",
						OwnerReferences: []metav1.OwnerReference{{APIVersion: "v1", Kind: "ServiceAccount", Name: "owner"}}}},
				).Build(),
				Scheme: scheme.Scheme,
				seenTypes: []schema.GroupVersionKind{
					schema.FromAPIVersionAndKind("autoscaling/v2", "HorizontalPodAutoscaler"),
					schema.FromAPIVersionAndKind("policy/v1", "PodDisruptionBudget"),
				},
			},
			args: args{
				ctx: context.TODO(),
				owner: &corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{Name: "owner", Namespace: "ns"},
				},
				managed: []corev1.ObjectReference{
					{Namespace: "ns", Name: "deploy", Kind: "Deployment", APIVersion: "apps/v1"},
				},
			},
			want: []check{
				{absent: false, obj: &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "deploy", Namespace: "ns"}}},
				{absent: true, obj: &autoscalingv2.HorizontalPodAutoscaler{ObjectMeta: metav1.ObjectMeta{Name: "hpa", Namespace: "ns"}}},
				{absent: true, obj: &policyv1.PodDisruptionBudget{ObjectMeta: metav1.ObjectMeta{Name: "pdb", Namespace: "ns"}}},
			},
			wantErr: false,
		},
		{
			name: "Prunes resource 2",
			fields: fields{
				Client: fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(
					&corev1.Secret{ObjectMeta: metav1.ObjectMeta{
						Name: "aaaa", Namespace: "ns",
						OwnerReferences: []metav1.OwnerReference{{APIVersion: "v1", Kind: "ServiceAccount", Name: "owner"}}}},
					&corev1.Secret{ObjectMeta: metav1.ObjectMeta{
						Name: "bbbb", Namespace: "ns",
						OwnerReferences: []metav1.OwnerReference{{APIVersion: "v1", Kind: "ServiceAccount", Name: "owner"}}}},
					&corev1.Secret{ObjectMeta: metav1.ObjectMeta{
						Name: "cccc", Namespace: "ns",
						OwnerReferences: []metav1.OwnerReference{{APIVersion: "v1", Kind: "ServiceAccount", Name: "owner"}}}},
					&corev1.Secret{ObjectMeta: metav1.ObjectMeta{
						Name: "dddd", Namespace: "ns",
						OwnerReferences: []metav1.OwnerReference{{APIVersion: "v1", Kind: "ServiceAccount", Name: "owner"}}}},
				).Build(),
				Scheme: scheme.Scheme,
				seenTypes: []schema.GroupVersionKind{
					schema.FromAPIVersionAndKind("v1", "Secret"),
				},
			},
			args: args{
				ctx: context.TODO(),
				owner: &corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{Name: "owner", Namespace: "ns"},
				},
				managed: []corev1.ObjectReference{
					{Namespace: "ns", Name: "aaaa", Kind: "Secret", APIVersion: "v1"},
					{Namespace: "ns", Name: "bbbb", Kind: "Secret", APIVersion: "v1"},
					{Namespace: "ns", Name: "cccc", Kind: "Secret", APIVersion: "v1"},
				},
			},
			want: []check{
				{absent: false, obj: &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "aaaa", Namespace: "ns"}}},
				{absent: false, obj: &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "bbbb", Namespace: "ns"}}},
				{absent: false, obj: &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "cccc", Namespace: "ns"}}},
				{absent: true, obj: &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "dddd", Namespace: "ns"}}},
			},
			wantErr: false,
		},
		{
			name: "Does nothing",
			fields: fields{
				Client: fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(
					&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
						Name: "deploy", Namespace: "ns",
						OwnerReferences: []metav1.OwnerReference{{APIVersion: "v1", Kind: "ServiceAccount", Name: "owner"}}}},
					&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{
						Name: "sa", Namespace: "ns",
						OwnerReferences: []metav1.OwnerReference{{APIVersion: "v1", Kind: "ServiceAccount", Name: "owner"}}}},
					&autoscalingv2.HorizontalPodAutoscaler{ObjectMeta: metav1.ObjectMeta{
						Name: "hpa", Namespace: "ns",
						OwnerReferences: []metav1.OwnerReference{{APIVersion: "v1", Kind: "ServiceAccount", Name: "owner"}}}},
					&policyv1.PodDisruptionBudget{ObjectMeta: metav1.ObjectMeta{
						Name: "pdb", Namespace: "ns",
						OwnerReferences: []metav1.OwnerReference{{APIVersion: "v1", Kind: "ServiceAccount", Name: "owner"}}}},
				).Build(),
				Scheme: scheme.Scheme,
				seenTypes: []schema.GroupVersionKind{
					schema.FromAPIVersionAndKind("v1", "ServiceAccount"),
					schema.FromAPIVersionAndKind("apps/v1", "Deployment"),
					schema.FromAPIVersionAndKind("autoscaling/v2", "HorizontalPodAutoscaler"),
					schema.FromAPIVersionAndKind("policy/v1", "PodDisruptionBudget"),
				},
			},
			args: args{
				ctx: context.TODO(),
				owner: &corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{Name: "owner", Namespace: "ns"},
				},
				managed: []corev1.ObjectReference{
					{Namespace: "ns", Name: "deploy", Kind: "Deployment", APIVersion: "apps/v1"},
					{Namespace: "ns", Name: "sa", Kind: "ServiceAccount", APIVersion: "v1"},
					{Namespace: "ns", Name: "hpa", Kind: "HorizontalPodAutoscaler", APIVersion: "autoscaling/v2"},
					{Namespace: "ns", Name: "pdb", Kind: "PodDisruptionBudget", APIVersion: "policy/v1"},
				},
			},
			want: []check{
				{absent: false, obj: &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "deploy", Namespace: "ns"}}},
				{absent: false, obj: &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "sa", Namespace: "ns"}}},
				{absent: false, obj: &autoscalingv2.HorizontalPodAutoscaler{ObjectMeta: metav1.ObjectMeta{Name: "hpa", Namespace: "ns"}}},
				{absent: false, obj: &policyv1.PodDisruptionBudget{ObjectMeta: metav1.ObjectMeta{Name: "pdb", Namespace: "ns"}}},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reconciler{
				Client:      tt.fields.Client,
				Scheme:      tt.fields.Scheme,
				typeTracker: typeTracker{seenTypes: tt.fields.seenTypes},
			}
			if err := r.pruneOrphaned(tt.args.ctx, tt.args.owner, tt.args.managed); (err != nil) != tt.wantErr {
				t.Errorf("Reconciler.pruneOrphaned() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			for _, check := range tt.want {
				err := tt.fields.Client.Get(tt.args.ctx, client.ObjectKeyFromObject(check.obj), check.obj)
				if (err != nil && errors.IsNotFound(err)) != check.absent {
					t.Errorf("Reconciler.pruneOrphaned()  want %s to be absent=%v", check.obj.GetName(), check.absent)
				}
			}
		})
	}
}

func Test_isPrunerEnabled(t *testing.T) {
	type args struct {
		owner client.Object
	}
	tests := []struct {
		name    string
		args    args
		preExec func()
		want    bool
	}{
		{
			name: "Returns true",
			args: args{
				owner: &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "ns"},
				},
			},
			preExec: func() {},
			want:    true,
		},
		{
			name: "Disabled by annotation",
			args: args{
				owner: &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "ns",
						Annotations: map[string]string{"example.com/prune": "false"},
					},
				},
			},
			preExec: func() { config.SetAnnotationsDomain("example.com") },
			want:    false,
		},
		{
			name: "Disabled by global config",
			args: args{
				owner: &corev1.Service{},
			},
			preExec: func() { config.DisableResourcePruner() },
			want:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.preExec()
			if got := isPrunerEnabled(tt.args.owner); got != tt.want {
				t.Errorf("isPrunerEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_typeTracker_trackType(t *testing.T) {
	type fields struct {
		seenTypes []schema.GroupVersionKind
	}
	type args struct {
		gvk schema.GroupVersionKind
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []schema.GroupVersionKind
	}{
		{
			name: "Adds the type",
			fields: fields{
				seenTypes: []schema.GroupVersionKind{
					{Group: "", Version: "v1", Kind: "ServiceAccount"},
				},
			},
			args: args{
				gvk: schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Service"},
			},
			want: []schema.GroupVersionKind{
				{Group: "", Version: "v1", Kind: "ServiceAccount"},
				{Group: "", Version: "v1", Kind: "Service"},
			},
		},
		{
			name: "Does nothing",
			fields: fields{
				seenTypes: []schema.GroupVersionKind{
					{Group: "", Version: "v1", Kind: "Service"},
				},
			},
			args: args{
				gvk: schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Service"},
			},
			want: []schema.GroupVersionKind{
				{Group: "", Version: "v1", Kind: "Service"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracker := &typeTracker{
				seenTypes: tt.fields.seenTypes,
			}
			tracker.trackType(tt.args.gvk)
			if !reflect.DeepEqual(tracker.seenTypes, tt.want) {
				t.Errorf("(*typeTracker).trackType() = %v, want %v", tracker.seenTypes, tt.want)
			}
		})
	}
}
