package reconciler

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Test_isManaged(t *testing.T) {
	type args struct {
		key     types.NamespacedName
		kind    string
		managed []corev1.ObjectReference
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Returns true",
			args: args{
				key:  types.NamespacedName{Name: "system-recaptcha", Namespace: "ns"},
				kind: "Secret",
				managed: []corev1.ObjectReference{
					{Namespace: "ns", Name: "system-recaptcha", Kind: "Secret"},
					{Namespace: "ns", Name: "system-smtp", Kind: "Secret"},
					{Namespace: "ns", Name: "system-zync", Kind: "Secret"},
					{Namespace: "ns", Name: "system", Kind: "Secret"},
					{Namespace: "ns", Name: "system-app", Kind: "Deployment"},
					{Namespace: "ns", Name: "system-app", Kind: "ServiceAccount"},
					{Namespace: "ns", Name: "system-app", Kind: "HorizontalPodAutoscaler"},
					{Namespace: "ns", Name: "system-app", Kind: "PodDisruptionBudget"},
					{Namespace: "ns", Name: "system-app", Kind: "PodMonitor"},
				},
			},
			want: true,
		},
		{
			name: "Returns false",
			args: args{
				key:  types.NamespacedName{Name: "system-recaptcha", Namespace: "ns"},
				kind: "Secret",
				managed: []corev1.ObjectReference{
					{Namespace: "ns", Name: "system-smtp", Kind: "Secret"},
					{Namespace: "ns", Name: "system-zync", Kind: "Secret"},
					{Namespace: "ns", Name: "system", Kind: "Secret"},
					{Namespace: "ns", Name: "system-app", Kind: "Deployment"},
					{Namespace: "ns", Name: "system-app", Kind: "ServiceAccount"},
					{Namespace: "ns", Name: "system-app", Kind: "HorizontalPodAutoscaler"},
					{Namespace: "ns", Name: "system-app", Kind: "PodDisruptionBudget"},
					{Namespace: "ns", Name: "system-app", Kind: "PodMonitor"},
				},
			},
			want: false,
		},
		{
			name: "Returns false",
			args: args{
				key:  types.NamespacedName{Name: "system-app", Namespace: "ns"},
				kind: "Role",
				managed: []corev1.ObjectReference{
					{Namespace: "ns", Name: "system-smtp", Kind: "Secret"},
					{Namespace: "ns", Name: "system-zync", Kind: "Secret"},
					{Namespace: "ns", Name: "system", Kind: "Secret"},
					{Namespace: "ns", Name: "system-app", Kind: "Deployment"},
					{Namespace: "ns", Name: "system-app", Kind: "ServiceAccount"},
					{Namespace: "ns", Name: "system-app", Kind: "HorizontalPodAutoscaler"},
					{Namespace: "ns", Name: "system-app", Kind: "PodDisruptionBudget"},
					{Namespace: "ns", Name: "system-app", Kind: "PodMonitor"},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isManaged(tt.args.key, tt.args.kind, tt.args.managed); got != tt.want {
				t.Errorf("isManaged() = %v, want %v", got, tt.want)
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
			preExec: func() { Config.AnnotationsDomain = "example.com" },
			want:    false,
		},
		{
			name: "Disabled by config",
			args: args{
				owner: &corev1.Service{},
			},
			preExec: func() { Config.ResourcePruner = false },
			want:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.preExec()
			if got := IsPrunerEnabled(tt.args.owner); got != tt.want {
				t.Errorf("isPrunerEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}
