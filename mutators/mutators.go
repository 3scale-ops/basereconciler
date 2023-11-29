package mutators

import (
	"context"
	"fmt"

	"github.com/3scale-ops/basereconciler/reconciler/resource"
	"github.com/3scale-ops/basereconciler/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// reconcileDeploymentReplicas reconciles the number of replicas of a Deployment
func ReconcileDeploymentReplicas(enforce bool) resource.TemplateMutationFunction {
	return func(ctx context.Context, cl client.Client, desired client.Object) error {
		if enforce {
			// Let the value in the template
			// override the runtime value
			return nil
		}

		live := &appsv1.Deployment{}
		if err := cl.Get(ctx, util.ObjectKey(desired), live); err != nil {
			if errors.IsNotFound(err) {
				return nil
			}
			return fmt.Errorf("unable to retrieve live object: %w", err)
		}

		// override the value in the template with the
		// runtime value
		desired.(*appsv1.Deployment).Spec.Replicas = live.Spec.Replicas
		return nil
	}
}

func ReconcileServiceNodePorts() resource.TemplateMutationFunction {
	return func(ctx context.Context, cl client.Client, desired client.Object) error {

		svc := desired.(*corev1.Service)
		live := &corev1.Service{}
		if err := cl.Get(ctx, util.ObjectKey(desired), live); err != nil {
			if errors.IsNotFound(err) {
				return nil
			}
			return fmt.Errorf("unable to retrieve live object: %w", err)
		}

		// // Set runtime values in the resource:
		// "/spec/clusterIP", "/spec/clusterIPs", "/spec/ipFamilies", "/spec/ipFamilyPolicy", "/spec/ports/*/nodePort"
		svc.Spec.ClusterIP = live.Spec.ClusterIP
		svc.Spec.ClusterIPs = live.Spec.ClusterIPs
		svc.Spec.IPFamilies = live.Spec.IPFamilies
		svc.Spec.IPFamilyPolicy = live.Spec.IPFamilyPolicy

		// For services that are not ClusterIP we need to populate the runtime values
		// of NodePort for each port
		if svc.Spec.Type != corev1.ServiceTypeClusterIP {
			for idx, port := range svc.Spec.Ports {
				runtimePort := findPort(port.Port, port.Protocol, live.Spec.Ports)
				if runtimePort != nil {
					svc.Spec.Ports[idx].NodePort = runtimePort.NodePort
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
