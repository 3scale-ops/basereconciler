package resource

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestTemplate_ChainTemplateBuilder(t *testing.T) {

	podTemplate := &Template[*corev1.Pod]{
		TemplateBuilder: func(client.Object) (*corev1.Pod, error) {
			return &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "pod", Namespace: "ns"},
			}, nil
		},
	}

	podTemplate.ChainTemplateBuilder(func(o client.Object) (*corev1.Pod, error) {
		o.SetAnnotations(map[string]string{"key": "value"})
		return o.(*corev1.Pod), nil
	})

	got, _ := podTemplate.Build(context.TODO(), fake.NewClientBuilder().Build(), nil)
	want := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "pod", Namespace: "ns", Annotations: map[string]string{"key": "value"}},
	}

	if diff := cmp.Diff(got, want); len(diff) > 0 {
		t.Errorf("(Template).ChainTemplateBuilder() diff = %v", diff)
	}
}
