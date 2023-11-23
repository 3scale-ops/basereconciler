package resources

import (
	"context"

	"github.com/3scale-ops/basereconciler/reconciler"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ reconciler.Resource = ServiceTemplate{}

// ServiceTemplate has methods to generate and reconcile a Service
type ServiceTemplate struct {
	Template  func() *corev1.Service
	IsEnabled bool
}

// Build returns a Service resource
func (st ServiceTemplate) Build(ctx context.Context, cl client.Client) (client.Object, error) {
	return st.Template().DeepCopy(), nil
}

// Enabled indicates if the resource should be present or not
func (dt ServiceTemplate) Enabled() bool {
	return dt.IsEnabled
}

// ResourceReconciler implements a generic reconciler for Service resources
func (st ServiceTemplate) ResourceReconciler(ctx context.Context, cl client.Client, obj client.Object) error {

	return ResourceReconciler[corev1.Service, *corev1.Service](
		ctx, cl, obj.(*corev1.Service), st.Enabled(),
		ResourceReconcilerConfig{
			ReconcileProperties: []Property{
				"metadata.labels",
				"metadata.annotations",
				"spec.type",
				"spec.ports",
				"spec.selector",
			},
			// IgnoreProperties: []Property{
			// 	"spec.ports[*].nodePort",
			// },
		},
		populateServiceSpecRuntimeValues(),
	)

}

func populateServiceSpecRuntimeValues() MutationFunction {

	return func(ctx context.Context, cl client.Client, instance client.Object, desired client.Object) error {

		isvc := instance.(*corev1.Service)
		dsvc := desired.(*corev1.Service)

		// Set runtime values in the resource:
		// "/spec/clusterIP", "/spec/clusterIPs", "/spec/ipFamilies", "/spec/ipFamilyPolicy", "/spec/ports/*/nodePort"
		dsvc.Spec.ClusterIP = isvc.Spec.ClusterIP
		dsvc.Spec.ClusterIPs = isvc.Spec.ClusterIPs
		dsvc.Spec.IPFamilies = isvc.Spec.IPFamilies
		dsvc.Spec.IPFamilyPolicy = isvc.Spec.IPFamilyPolicy

		// For services that are not ClusterIP we need to populate the runtime values
		// of NodePort for each port
		if dsvc.Spec.Type != corev1.ServiceTypeClusterIP {
			for idx, port := range dsvc.Spec.Ports {
				runtimePort := findPort(port.Port, port.Protocol, isvc.Spec.Ports)
				if runtimePort != nil {
					dsvc.Spec.Ports[idx].NodePort = runtimePort.NodePort
				}
			}
		}
		return nil
	}
}

func findPort(pNumber int32, pProtocol corev1.Protocol, ports []corev1.ServicePort) *corev1.ServicePort {
	// Ports within a svc are uniquely identified by
	// the "port" and "protocol" fields. This is documented in
	// k8s API reference
	for _, port := range ports {
		if pNumber == port.Port && pProtocol == port.Protocol {
			return &port
		}
	}
	// not found
	return nil
}
