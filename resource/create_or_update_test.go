package resource

import (
	"context"
	"reflect"
	"testing"

	"github.com/3scale-ops/basereconciler/util"
	"github.com/google/go-cmp/cmp"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCreateOrUpdate(t *testing.T) {
	type args struct {
		ctx      context.Context
		cl       client.Client
		scheme   *runtime.Scheme
		owner    client.Object
		template TemplateInterface
	}
	tests := []struct {
		name          string
		args          args
		want          *corev1.ObjectReference
		wantErr       bool
		wantObject    client.Object
		wantObjectErr func(error) bool
	}{
		{
			name: "Reconciles properties and applies mutations",
			args: args{
				ctx: context.TODO(),
				cl: fake.NewClientBuilder().WithObjects(
					&corev1.Service{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "service",
							Namespace: "ns",
						},
						Spec: corev1.ServiceSpec{
							Type:                  corev1.ServiceTypeLoadBalancer,
							ExternalTrafficPolicy: corev1.ServiceExternalTrafficPolicyTypeCluster,
							SessionAffinity:       corev1.ServiceAffinityNone,
							Ports: []corev1.ServicePort{
								{Name: "port1", Port: 80, TargetPort: intstr.FromInt(80), Protocol: corev1.ProtocolTCP, NodePort: 33333},
								{Name: "port2", Port: 8080, TargetPort: intstr.FromInt(8080), Protocol: corev1.ProtocolTCP, NodePort: 33334},
							},
							Selector: map[string]string{"selector": "deployment"},
						},
					}).Build(),
				scheme: scheme.Scheme,
				owner:  &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "owner", Namespace: "ns"}},
				template: &Template[*corev1.Service]{
					TemplateBuilder: func(client.Object) (*corev1.Service, error) {
						return &corev1.Service{
							ObjectMeta: metav1.ObjectMeta{
								Name:        "service",
								Namespace:   "ns",
								Annotations: map[string]string{"key": "value"},
							},
							Spec: corev1.ServiceSpec{
								Type:                  corev1.ServiceTypeLoadBalancer,
								InternalTrafficPolicy: util.Pointer(corev1.ServiceInternalTrafficPolicyLocal),
								Ports: []corev1.ServicePort{{
									Name: "port1", Port: 90, TargetPort: intstr.FromInt(90), Protocol: corev1.ProtocolTCP}},
							},
						}, nil
					},
					TemplateMutations: []TemplateMutationFunction{
						func(ctx context.Context, cl client.Client, o client.Object) error {
							o.(*corev1.Service).Spec.Ports[0].NodePort = 33333
							return nil
						},
					},
					IsEnabled:        true,
					EnsureProperties: []Property{"metadata.annotations", "spec.selector", "spec.ports", "spec.internalTrafficPolicy"},
					IgnoreProperties: []Property{},
				},
			},
			want: &corev1.ObjectReference{
				Kind:       "Service",
				Namespace:  "ns",
				Name:       "service",
				APIVersion: "v1",
			},
			wantErr: false,
			wantObject: &corev1.Service{
				TypeMeta: metav1.TypeMeta{Kind: "Service", APIVersion: "v1"},
				ObjectMeta: metav1.ObjectMeta{
					Name:        "service",
					Namespace:   "ns",
					Annotations: map[string]string{"key": "value"},
				},
				Spec: corev1.ServiceSpec{
					Type:                  corev1.ServiceTypeLoadBalancer,
					ExternalTrafficPolicy: corev1.ServiceExternalTrafficPolicyTypeCluster,
					InternalTrafficPolicy: util.Pointer(corev1.ServiceInternalTrafficPolicyLocal),
					SessionAffinity:       corev1.ServiceAffinityNone,
					Ports: []corev1.ServicePort{{
						Name: "port1", Port: 90, TargetPort: intstr.FromInt(90), Protocol: corev1.ProtocolTCP, NodePort: 33333}},
				},
			},
		},
		{
			name: "Ignores properties",
			args: args{
				ctx: context.TODO(),
				cl: fake.NewClientBuilder().WithObjects(
					&corev1.Service{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "service",
							Namespace: "ns",
						},
						Spec: corev1.ServiceSpec{
							Type:                  corev1.ServiceTypeLoadBalancer,
							ExternalTrafficPolicy: corev1.ServiceExternalTrafficPolicyTypeCluster,
							InternalTrafficPolicy: util.Pointer(corev1.ServiceInternalTrafficPolicyCluster),
							SessionAffinity:       corev1.ServiceAffinityNone,
							Ports: []corev1.ServicePort{
								{Name: "port1", Port: 80, TargetPort: intstr.FromInt(80), Protocol: corev1.ProtocolTCP, NodePort: 33333},
								{Name: "port2", Port: 8080, TargetPort: intstr.FromInt(8080), Protocol: corev1.ProtocolTCP, NodePort: 33334},
							},
							Selector: map[string]string{"selector": "deployment"},
						},
					}).Build(),
				scheme: scheme.Scheme,
				owner:  &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "owner", Namespace: "ns"}},
				template: &Template[*corev1.Service]{
					TemplateBuilder: func(client.Object) (*corev1.Service, error) {
						return &corev1.Service{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "service",
								Namespace: "ns",
							},
							Spec: corev1.ServiceSpec{
								Type:                  corev1.ServiceTypeLoadBalancer,
								ExternalTrafficPolicy: corev1.ServiceExternalTrafficPolicyTypeCluster,
								SessionAffinity:       corev1.ServiceAffinityNone,
								Ports: []corev1.ServicePort{
									{Name: "port1", Port: 80, TargetPort: intstr.FromInt(80), Protocol: corev1.ProtocolTCP, NodePort: 33333},
									{Name: "port2", Port: 8080, TargetPort: intstr.FromInt(8080), Protocol: corev1.ProtocolTCP, NodePort: 33334},
								},
								Selector: map[string]string{"selector": "deployment"},
							},
						}, nil
					},
					IsEnabled:        true,
					EnsureProperties: []Property{"metadata.annotations", "spec"},
					IgnoreProperties: []Property{"spec.internalTrafficPolicy"},
				},
			},
			want: &corev1.ObjectReference{
				Kind:       "Service",
				Namespace:  "ns",
				Name:       "service",
				APIVersion: "v1",
			},
			wantErr: false,
			wantObject: &corev1.Service{
				TypeMeta: metav1.TypeMeta{Kind: "Service", APIVersion: "v1"},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "service",
					Namespace: "ns",
				},
				Spec: corev1.ServiceSpec{
					Type:                  corev1.ServiceTypeLoadBalancer,
					ExternalTrafficPolicy: corev1.ServiceExternalTrafficPolicyTypeCluster,
					InternalTrafficPolicy: util.Pointer(corev1.ServiceInternalTrafficPolicyCluster),
					SessionAffinity:       corev1.ServiceAffinityNone,
					Ports: []corev1.ServicePort{
						{Name: "port1", Port: 80, TargetPort: intstr.FromInt(80), Protocol: corev1.ProtocolTCP, NodePort: 33333},
						{Name: "port2", Port: 8080, TargetPort: intstr.FromInt(8080), Protocol: corev1.ProtocolTCP, NodePort: 33334},
					},
					Selector: map[string]string{"selector": "deployment"},
				},
			},
		},
		{
			name: "Creates object",
			args: args{
				ctx:    context.TODO(),
				cl:     fake.NewClientBuilder().Build(),
				scheme: scheme.Scheme,
				owner:  &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "owner", Namespace: "ns"}},
				template: &Template[*corev1.Service]{
					TemplateBuilder: func(client.Object) (*corev1.Service, error) {
						return &corev1.Service{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "service",
								Namespace: "ns",
							},
							Spec: corev1.ServiceSpec{
								Type:                  corev1.ServiceTypeLoadBalancer,
								ExternalTrafficPolicy: corev1.ServiceExternalTrafficPolicyTypeCluster,
								SessionAffinity:       corev1.ServiceAffinityNone,
								Ports: []corev1.ServicePort{
									{Name: "port1", Port: 80, TargetPort: intstr.FromInt(80), Protocol: corev1.ProtocolTCP, NodePort: 33333},
									{Name: "port2", Port: 8080, TargetPort: intstr.FromInt(8080), Protocol: corev1.ProtocolTCP, NodePort: 33334},
								},
								Selector: map[string]string{"selector": "deployment"},
							},
						}, nil
					},
					IsEnabled: true,
				},
			},
			want: &corev1.ObjectReference{
				Kind:       "Service",
				Namespace:  "ns",
				Name:       "service",
				APIVersion: "v1",
			},
			wantErr: false,
			wantObject: &corev1.Service{
				TypeMeta: metav1.TypeMeta{Kind: "Service", APIVersion: "v1"},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "service",
					Namespace: "ns",
					OwnerReferences: []metav1.OwnerReference{{
						APIVersion:         "v1",
						Kind:               "ServiceAccount",
						Name:               "owner",
						Controller:         util.Pointer(true),
						BlockOwnerDeletion: util.Pointer(true),
					}},
				},
				Spec: corev1.ServiceSpec{
					Type:                  corev1.ServiceTypeLoadBalancer,
					ExternalTrafficPolicy: corev1.ServiceExternalTrafficPolicyTypeCluster,
					SessionAffinity:       corev1.ServiceAffinityNone,
					Ports: []corev1.ServicePort{
						{Name: "port1", Port: 80, TargetPort: intstr.FromInt(80), Protocol: corev1.ProtocolTCP, NodePort: 33333},
						{Name: "port2", Port: 8080, TargetPort: intstr.FromInt(8080), Protocol: corev1.ProtocolTCP, NodePort: 33334},
					},
					Selector: map[string]string{"selector": "deployment"},
				},
			},
		},
		{
			name: "Object is disabled and being deleted, do nothing",
			args: args{
				ctx:    context.TODO(),
				cl:     fake.NewClientBuilder().Build(),
				scheme: scheme.Scheme,
				owner:  &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "owner", Namespace: "ns"}},
				template: &Template[*corev1.Service]{
					TemplateBuilder: func(client.Object) (*corev1.Service, error) {
						return &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "service", Namespace: "ns"}}, nil
					},
					IsEnabled: false,
				},
			},
			want:          nil,
			wantErr:       false,
			wantObject:    &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "service", Namespace: "ns"}},
			wantObjectErr: errors.IsNotFound,
		},
		{
			name: "Object is disabled, delete",
			args: args{
				ctx:    context.TODO(),
				cl:     fake.NewClientBuilder().WithObjects(&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "service", Namespace: "ns"}}).Build(),
				scheme: scheme.Scheme,
				owner:  &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "owner", Namespace: "ns"}},
				template: &Template[*corev1.Service]{
					TemplateBuilder: func(client.Object) (*corev1.Service, error) {
						return &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "service", Namespace: "ns"}}, nil
					},
					IsEnabled: false,
				},
			},
			want:          nil,
			wantErr:       false,
			wantObject:    &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "service", Namespace: "ns"}},
			wantObjectErr: errors.IsNotFound,
		},
		{
			name: "Create a Deployment",
			args: args{
				ctx:    context.TODO(),
				cl:     fake.NewClientBuilder().Build(),
				scheme: scheme.Scheme,
				owner:  &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "owner", Namespace: "ns"}},
				template: &Template[*appsv1.Deployment]{
					TemplateBuilder: func(client.Object) (*appsv1.Deployment, error) {
						return &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "deployment", Namespace: "ns"}}, nil
					},
					IsEnabled: true,
				},
			},
			want: &corev1.ObjectReference{
				Kind:       "Deployment",
				Namespace:  "ns",
				Name:       "deployment",
				APIVersion: "apps/v1",
			},
			wantErr: false,
			wantObject: &appsv1.Deployment{
				TypeMeta: metav1.TypeMeta{Kind: "Deployment", APIVersion: "apps/v1"},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "deployment",
					Namespace: "ns",
					OwnerReferences: []metav1.OwnerReference{{
						APIVersion:         "v1",
						Kind:               "ServiceAccount",
						Name:               "owner",
						Controller:         util.Pointer(true),
						BlockOwnerDeletion: util.Pointer(true),
					}},
				},
			},
			wantObjectErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref, err := CreateOrUpdate(tt.args.ctx, tt.args.cl, tt.args.scheme, tt.args.owner, tt.args.template)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateOrUpdate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(ref, tt.want, util.IgnoreProperty("ResourceVersion"), util.IgnoreProperty("UID")); len(diff) > 0 {
				t.Errorf("CreateOrUpdate() ref diff = %v", diff)
				return
			}
			if tt.wantObjectErr == nil {
				o, _ := util.NewObjectFromGVK(schema.FromAPIVersionAndKind(ref.APIVersion, ref.Kind), tt.args.scheme)
				tt.args.cl.Get(context.TODO(), types.NamespacedName{Name: tt.wantObject.GetName(), Namespace: tt.wantObject.GetNamespace()}, o)
				if diff := cmp.Diff(o, tt.wantObject, util.IgnoreProperty("ResourceVersion")); len(diff) > 0 {
					t.Errorf("CreateOrUpdate() object diff = %v", diff)
				}
			} else {
				o := tt.wantObject.DeepCopyObject().(client.Object)
				err := tt.args.cl.Get(context.TODO(), types.NamespacedName{Name: tt.wantObject.GetName(), Namespace: tt.wantObject.GetNamespace()}, o)
				if !tt.wantObjectErr(err) {
					t.Errorf("CreateOrUpdate() got err retrieving object = %v", err)
				}
			}
		})
	}
}

func Test_reconcilerConfig(t *testing.T) {
	type args struct {
		template TemplateInterface
		gvk      schema.GroupVersionKind
	}
	tests := []struct {
		name    string
		args    args
		want    []Property
		want1   []Property
		wantErr bool
	}{
		{
			name: "Returns default config",
			args: args{
				template: &Template[*corev1.Pod]{},
				gvk:      schema.FromAPIVersionAndKind("v1", "Pod"),
			},
			want:    []Property{"metadata.annotations", "metadata.labels", "spec"},
			want1:   []Property{},
			wantErr: false,
		},
		{
			name: "Returns explicit config",
			args: args{
				template: &Template[*corev1.Pod]{
					EnsureProperties: []Property{"a.b.c"},
					IgnoreProperties: []Property{"x"},
				},
				gvk: schema.FromAPIVersionAndKind("v1", "Pod"),
			},
			want:    []Property{"a.b.c"},
			want1:   []Property{"x"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := reconcilerConfig(tt.args.template, tt.args.gvk)
			if (err != nil) != tt.wantErr {
				t.Errorf("reconcilerConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("reconcilerConfig() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("reconcilerConfig() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
