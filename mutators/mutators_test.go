package mutators

import (
	"context"
	"reflect"
	"testing"

	"github.com/3scale-ops/basereconciler/util"
	"github.com/google/go-cmp/cmp"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestReconcileDeploymentReplicas(t *testing.T) {
	type args struct {
		enforce bool
		ctx     context.Context
		cl      client.Client
		desired *appsv1.Deployment
	}
	tests := []struct {
		name    string
		args    args
		want    *appsv1.Deployment
		wantErr bool
	}{
		{
			name: "Enforces number of replicas",
			args: args{
				enforce: true,
				ctx:     context.TODO(),
				cl: fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(
					&appsv1.Deployment{
						ObjectMeta: metav1.ObjectMeta{Name: "deployment", Namespace: "ns"},
						Spec:       appsv1.DeploymentSpec{Replicas: util.Pointer[int32](10)},
					}).Build(),
				desired: &appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{Name: "deployment", Namespace: "ns"},
					Spec:       appsv1.DeploymentSpec{Replicas: util.Pointer[int32](2)},
				},
			},
			want: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: "deployment", Namespace: "ns"},
				Spec:       appsv1.DeploymentSpec{Replicas: util.Pointer[int32](2)},
			},
			wantErr: false,
		},
		{
			name: "Sets live replicas in template",
			args: args{
				enforce: false,
				ctx:     context.TODO(),
				cl: fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(
					&appsv1.Deployment{
						ObjectMeta: metav1.ObjectMeta{Name: "deployment", Namespace: "ns"},
						Spec:       appsv1.DeploymentSpec{Replicas: util.Pointer[int32](10)},
					}).Build(),
				desired: &appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{Name: "deployment", Namespace: "ns"},
					Spec:       appsv1.DeploymentSpec{Replicas: util.Pointer[int32](2)},
				},
			},
			want: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: "deployment", Namespace: "ns"},
				Spec:       appsv1.DeploymentSpec{Replicas: util.Pointer[int32](10)},
			},
			wantErr: false,
		},
		{
			name: "No error if deployment not found",
			args: args{
				enforce: false,
				ctx:     context.TODO(),
				cl:      fake.NewClientBuilder().WithScheme(scheme.Scheme).Build(),
				desired: &appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{Name: "deployment", Namespace: "ns"},
					Spec:       appsv1.DeploymentSpec{Replicas: util.Pointer[int32](2)},
				},
			},
			want: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: "deployment", Namespace: "ns"},
				Spec:       appsv1.DeploymentSpec{Replicas: util.Pointer[int32](2)},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ReconcileDeploymentReplicas(tt.args.enforce)(tt.args.ctx, tt.args.cl, tt.args.desired); (err != nil) != tt.wantErr {
				t.Errorf("ReconcileDeploymentReplicas() error = %v, wantErr %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(tt.args.desired, tt.want); len(diff) > 0 {
				t.Errorf("ReconcileDeploymentReplicas() = diff %s", diff)
			}
		})
	}
}

func Test_ReconcileServiceNodePorts(t *testing.T) {
	type args struct {
		ctx     context.Context
		cl      client.Client
		desired *corev1.Service
	}
	tests := []struct {
		name    string
		args    args
		want    *corev1.Service
		wantErr bool
	}{
		{
			name: "Populates the runtime fields",
			args: args{
				ctx: context.TODO(),
				cl: fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(
					&corev1.Service{
						ObjectMeta: metav1.ObjectMeta{
							Name: "service", Namespace: "ns",
						},
						Spec: corev1.ServiceSpec{
							Type:       corev1.ServiceTypeLoadBalancer,
							ClusterIP:  "1.1.1.1",
							ClusterIPs: []string{"1.1.1.1"},
							Ports: []corev1.ServicePort{{
								Name: "port", Port: 80, TargetPort: intstr.FromInt(80), Protocol: corev1.ProtocolTCP, NodePort: 3333}},
						},
					}).Build(),
				desired: &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{Name: "service", Namespace: "ns"},
					Spec: corev1.ServiceSpec{
						Type: corev1.ServiceTypeLoadBalancer,
						Ports: []corev1.ServicePort{{
							Name: "port", Port: 80, TargetPort: intstr.FromInt(80), Protocol: corev1.ProtocolTCP}},
					},
				},
			},
			want: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name: "service", Namespace: "ns",
				},
				Spec: corev1.ServiceSpec{
					Type:       corev1.ServiceTypeLoadBalancer,
					ClusterIP:  "1.1.1.1",
					ClusterIPs: []string{"1.1.1.1"},
					Ports: []corev1.ServicePort{{
						Name: "port", Port: 80, TargetPort: intstr.FromInt(80), Protocol: corev1.ProtocolTCP, NodePort: 3333}},
				},
			},
			wantErr: false,
		},
		{
			name: "Populates the runtime fields (template adds a new port)",
			args: args{
				ctx: context.TODO(),
				cl: fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(
					&corev1.Service{
						ObjectMeta: metav1.ObjectMeta{
							Name: "service", Namespace: "ns",
						},
						Spec: corev1.ServiceSpec{
							Type:       corev1.ServiceTypeLoadBalancer,
							ClusterIP:  "1.1.1.1",
							ClusterIPs: []string{"1.1.1.1"},
							Ports: []corev1.ServicePort{{
								Name: "port", Port: 80, TargetPort: intstr.FromInt(80), Protocol: corev1.ProtocolTCP, NodePort: 3333}},
						},
					}).Build(),
				desired: &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{Name: "service", Namespace: "ns"},
					Spec: corev1.ServiceSpec{
						Type: corev1.ServiceTypeLoadBalancer,
						Ports: []corev1.ServicePort{
							{Name: "port", Port: 80, TargetPort: intstr.FromInt(80), Protocol: corev1.ProtocolTCP},
							{Name: "port", Port: 8080, TargetPort: intstr.FromInt(8080), Protocol: corev1.ProtocolTCP}},
					},
				},
			},
			want: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name: "service", Namespace: "ns",
				},
				Spec: corev1.ServiceSpec{
					Type:       corev1.ServiceTypeLoadBalancer,
					ClusterIP:  "1.1.1.1",
					ClusterIPs: []string{"1.1.1.1"},
					Ports: []corev1.ServicePort{
						{Name: "port", Port: 80, TargetPort: intstr.FromInt(80), Protocol: corev1.ProtocolTCP, NodePort: 3333},
						{Name: "port", Port: 8080, TargetPort: intstr.FromInt(8080), Protocol: corev1.ProtocolTCP},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Populates the runtime fields (template removes a port)",
			args: args{
				ctx: context.TODO(),
				cl: fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(
					&corev1.Service{
						ObjectMeta: metav1.ObjectMeta{
							Name: "service", Namespace: "ns",
						},
						Spec: corev1.ServiceSpec{
							Type:       corev1.ServiceTypeLoadBalancer,
							ClusterIP:  "1.1.1.1",
							ClusterIPs: []string{"1.1.1.1"},
							Ports: []corev1.ServicePort{
								{Name: "port", Port: 80, TargetPort: intstr.FromInt(80), Protocol: corev1.ProtocolTCP, NodePort: 3333},
								{Name: "port", Port: 8080, TargetPort: intstr.FromInt(8080), Protocol: corev1.ProtocolTCP, NodePort: 3334},
							},
						},
					}).Build(),
				desired: &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{Name: "service", Namespace: "ns"},
					Spec: corev1.ServiceSpec{
						Type: corev1.ServiceTypeLoadBalancer,
						Ports: []corev1.ServicePort{{
							Name: "port", Port: 80, TargetPort: intstr.FromInt(80), Protocol: corev1.ProtocolTCP}},
					},
				},
			},
			want: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name: "service", Namespace: "ns",
				},
				Spec: corev1.ServiceSpec{
					Type:       corev1.ServiceTypeLoadBalancer,
					ClusterIP:  "1.1.1.1",
					ClusterIPs: []string{"1.1.1.1"},
					Ports: []corev1.ServicePort{
						{Name: "port", Port: 80, TargetPort: intstr.FromInt(80), Protocol: corev1.ProtocolTCP, NodePort: 3333},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Populates the runtime fields (does not fail with ClusterIP service)",
			args: args{
				ctx: context.TODO(),
				cl: fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(
					&corev1.Service{
						ObjectMeta: metav1.ObjectMeta{
							Name: "service", Namespace: "ns",
						},
						Spec: corev1.ServiceSpec{
							Type:       corev1.ServiceTypeClusterIP,
							ClusterIP:  "1.1.1.1",
							ClusterIPs: []string{"1.1.1.1"},
							Ports: []corev1.ServicePort{{
								Name: "port", Port: 80, TargetPort: intstr.FromInt(80), Protocol: corev1.ProtocolTCP}},
						},
					}).Build(),
				desired: &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{Name: "service", Namespace: "ns"},
					Spec: corev1.ServiceSpec{
						Type: corev1.ServiceTypeClusterIP,
						Ports: []corev1.ServicePort{{
							Name: "port", Port: 80, TargetPort: intstr.FromInt(80), Protocol: corev1.ProtocolTCP}},
					},
				},
			},
			want: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name: "service", Namespace: "ns",
				},
				Spec: corev1.ServiceSpec{
					Type:       corev1.ServiceTypeClusterIP,
					ClusterIP:  "1.1.1.1",
					ClusterIPs: []string{"1.1.1.1"},
					Ports: []corev1.ServicePort{{
						Name: "port", Port: 80, TargetPort: intstr.FromInt(80), Protocol: corev1.ProtocolTCP}},
				},
			},
			wantErr: false,
		},
		{
			name: "Populates the runtime fields (does not fail if Service not found)",
			args: args{
				ctx: context.TODO(),
				cl:  fake.NewClientBuilder().WithScheme(scheme.Scheme).Build(),
				desired: &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{Name: "service", Namespace: "ns"},
					Spec: corev1.ServiceSpec{
						Type: corev1.ServiceTypeClusterIP,
						Ports: []corev1.ServicePort{{
							Name: "port", Port: 80, TargetPort: intstr.FromInt(80), Protocol: corev1.ProtocolTCP}},
					},
				},
			},
			want: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name: "service", Namespace: "ns",
				},
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeClusterIP,
					Ports: []corev1.ServicePort{{
						Name: "port", Port: 80, TargetPort: intstr.FromInt(80), Protocol: corev1.ProtocolTCP}},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ReconcileServiceNodePorts()(tt.args.ctx, tt.args.cl, tt.args.desired); (err != nil) != tt.wantErr {
				t.Errorf("ReconcileServiceNodePorts() error = %v, wantErr %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(tt.args.desired, tt.want); len(diff) > 0 {
				t.Errorf("ReconcileServiceNodePorts() = diff %s", diff)
			}
		})
	}
}

func Test_findPort(t *testing.T) {
	type args struct {
		pNumber   int32
		pProtocol corev1.Protocol
		ports     []corev1.ServicePort
	}
	tests := []struct {
		name string
		args args
		want *corev1.ServicePort
	}{
		{
			name: "Fount",
			args: args{
				pNumber:   80,
				pProtocol: "TCP",
				ports: []corev1.ServicePort{
					{
						Name:       "http",
						Protocol:   "TCP",
						Port:       80,
						TargetPort: intstr.FromInt(8080),
					},
					{
						Name:       "https",
						Protocol:   "TCP",
						Port:       443,
						TargetPort: intstr.FromInt(8443),
					},
				},
			},
			want: &corev1.ServicePort{
				Name:       "http",
				Protocol:   "TCP",
				Port:       80,
				TargetPort: intstr.FromInt(8080),
			},
		},
		{
			name: "Not fount",
			args: args{
				pNumber:   80,
				pProtocol: "TCP",
				ports: []corev1.ServicePort{{
					Name:       "https",
					Protocol:   "TCP",
					Port:       443,
					TargetPort: intstr.FromInt(8443),
				}},
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := findPort(tt.args.pNumber, tt.args.pProtocol, tt.args.ports); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("findPort() = %v, want %v", got, tt.want)
			}
		})
	}
}
