package resources

import (
	"context"

	"github.com/3scale-ops/basereconciler/reconciler/resource"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// reconcileDeploymentReplicas reconciles the number of replicas of a Deployment
func ReconcileDeploymentReplicas(enforce bool) resource.MutationFunction {
	return func(ctx context.Context, cl client.Client, instance client.Object, desired client.Object) error {
		if enforce {
			// Let the value in the template
			// override the runtime value
			return nil
		}

		idep := instance.(*appsv1.Deployment)
		ddep := desired.(*appsv1.Deployment)

		// override the value in the template with the
		// runtime value
		ddep.Spec.Replicas = idep.Spec.Replicas
		return nil
	}
}

func ReconcileServiceNodePorts() resource.MutationFunction {
	return func(ctx context.Context, cl client.Client, instance client.Object, desired client.Object) error {

		isvc := instance.(*corev1.Service)
		dsvc := desired.(*corev1.Service)

		// // Set runtime values in the resource:
		// // "/spec/clusterIP", "/spec/clusterIPs", "/spec/ipFamilies", "/spec/ipFamilyPolicy", "/spec/ports/*/nodePort"
		// dsvc.Spec.ClusterIP = isvc.Spec.ClusterIP
		// dsvc.Spec.ClusterIPs = isvc.Spec.ClusterIPs
		// dsvc.Spec.IPFamilies = isvc.Spec.IPFamilies
		// dsvc.Spec.IPFamilyPolicy = isvc.Spec.IPFamilyPolicy

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
