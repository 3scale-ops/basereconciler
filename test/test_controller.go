/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package test

import (
	"context"
	"time"

	"github.com/3scale-ops/basereconciler/reconciler"
	"github.com/3scale-ops/basereconciler/resources"
	"github.com/3scale-ops/basereconciler/test/api/v1alpha1"
	externalsecretsv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/go-logr/logr"
	grafanav1alpha1 "github.com/grafana-operator/grafana-operator/v4/api/integreatly/v1alpha1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

func init() {
	reconciler.Config.AnnotationsDomain = "example.com"
	reconciler.Config.ResourcePruner = true
	reconciler.Config.ManagedTypes = reconciler.NewManagedTypes().
		Register(&corev1.ServiceList{}).
		Register(&appsv1.DeploymentList{}).
		Register(&externalsecretsv1beta1.ExternalSecretList{}).
		Register(&grafanav1alpha1.GrafanaDashboardList{}).
		Register(&autoscalingv2.HorizontalPodAutoscalerList{}).
		Register(&policyv1.PodDisruptionBudgetList{}).
		Register(&monitoringv1.PodMonitorList{})
}

// Reconciler reconciles a Test object
// +kubebuilder:object:generate=false
type Reconciler struct {
	reconciler.Reconciler
	Log logr.Logger
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Log.WithValues("name", req.Name, "namespace", req.Namespace)
	ctx = log.IntoContext(ctx, logger)

	instance := &v1alpha1.Test{}
	key := types.NamespacedName{Name: req.Name, Namespace: req.Namespace}
	result, err := r.GetInstance(ctx, key, instance, nil, nil)
	if result != nil || err != nil {
		return *result, err
	}

	err = r.ReconcileOwnedResources(ctx, instance, []reconciler.Resource{
		resources.DeploymentTemplate{
			Template: deployment(req.Namespace),
			RolloutTriggers: []resources.RolloutTrigger{{
				Name:       "secret",
				SecretName: pointer.String("secret"),
			}},
			EnforceReplicas: true,
			IsEnabled:       true,
		},
		resources.ExternalSecretTemplate{
			Template:  externalSecret(req.Namespace),
			IsEnabled: true,
		},
		resources.ServiceTemplate{
			Template:  service(req.Namespace, instance.Spec.ServiceAnnotations),
			IsEnabled: true,
		},
		resources.PodDisruptionBudgetTemplate{
			Template:  pdb(req.Namespace),
			IsEnabled: instance.Spec.PDB != nil && *instance.Spec.PDB,
		},
		resources.HorizontalPodAutoscalerTemplate{
			Template:  hpa(req.Namespace),
			IsEnabled: instance.Spec.HPA != nil && *instance.Spec.HPA,
		},
		resources.PodMonitorTemplate{
			Template:  podmonitor(req.Namespace),
			IsEnabled: instance.Spec.HPA != nil && *instance.Spec.HPA,
		},
		resources.GrafanaDashboardTemplate{
			Template:  dashboard(req.Namespace),
			IsEnabled: instance.Spec.HPA != nil && *instance.Spec.HPA,
		},
	})

	if err != nil {
		logger.Error(err, "unable to reconcile owned resources")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Test{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&policyv1.PodDisruptionBudget{}).
		Owns(&autoscalingv2.HorizontalPodAutoscaler{}).
		Owns(&externalsecretsv1beta1.ExternalSecret{}).
		Watches(&source.Kind{Type: &corev1.Secret{TypeMeta: metav1.TypeMeta{Kind: "Secret"}}},
			r.SecretEventHandler(&v1alpha1.TestList{}, r.Log)).
		Complete(r)
}

func deployment(namespace string) func() *appsv1.Deployment {
	return func() *appsv1.Deployment {
		dep := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "deployment",
				Namespace: namespace,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: pointer.Int32Ptr(1),
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"selector": "deployment"},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{"selector": "deployment"},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:      "container",
								Image:     "example.com:latest",
								Resources: corev1.ResourceRequirements{},
							},
						},
					},
				},
			},
		}

		return dep
	}
}

func service(namespace string, annotations map[string]string) func() *corev1.Service {
	return func() *corev1.Service {
		return &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "service",
				Namespace:   namespace,
				Annotations: annotations,
			},
			Spec: corev1.ServiceSpec{
				Type:                  corev1.ServiceTypeLoadBalancer,
				ExternalTrafficPolicy: corev1.ServiceExternalTrafficPolicyTypeCluster,
				SessionAffinity:       corev1.ServiceAffinityNone,
				Ports: []corev1.ServicePort{{
					Name: "port", Port: 80, TargetPort: intstr.FromInt(80), Protocol: corev1.ProtocolTCP}},
				Selector: map[string]string{"selector": "deployment"},
			},
		}
	}
}

func externalSecret(namespace string) func() *externalsecretsv1beta1.ExternalSecret {

	return func() *externalsecretsv1beta1.ExternalSecret {
		return &externalsecretsv1beta1.ExternalSecret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "secret",
				Namespace: namespace,
			},
			Spec: externalsecretsv1beta1.ExternalSecretSpec{
				SecretStoreRef:  externalsecretsv1beta1.SecretStoreRef{Name: "vault-mgmt", Kind: "ClusterSecretStore"},
				Target:          externalsecretsv1beta1.ExternalSecretTarget{Name: "secret"},
				RefreshInterval: &metav1.Duration{Duration: 60 * time.Second},
				Data: []externalsecretsv1beta1.ExternalSecretData{
					{
						SecretKey: "KEY",
						RemoteRef: externalsecretsv1beta1.ExternalSecretDataRemoteRef{
							Key:      "vault-path",
							Property: "vault-key",
						},
					},
				},
			},
		}
	}
}

func hpa(namespace string) func() *autoscalingv2.HorizontalPodAutoscaler {
	return func() *autoscalingv2.HorizontalPodAutoscaler {
		return &autoscalingv2.HorizontalPodAutoscaler{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "hpa",
				Namespace: namespace,
			},
			Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
				ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
					APIVersion: appsv1.SchemeGroupVersion.String(),
					Kind:       "Deployment",
					Name:       "deployment",
				},
				MinReplicas: pointer.Int32(1),
				MaxReplicas: 1,
			},
		}
	}
}

func pdb(namespace string) func() *policyv1.PodDisruptionBudget {
	return func() *policyv1.PodDisruptionBudget {

		return &policyv1.PodDisruptionBudget{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pdb",
				Namespace: namespace,
			},
			Spec: policyv1.PodDisruptionBudgetSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"selector": "deployment"},
				},
				MinAvailable: intstr.ValueOrDefault(nil, intstr.FromInt(1)),
			},
		}
	}
}

func podmonitor(namespace string) func() *monitoringv1.PodMonitor {
	return func() *monitoringv1.PodMonitor {

		return &monitoringv1.PodMonitor{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pm",
				Namespace: namespace,
			},
			Spec: monitoringv1.PodMonitorSpec{
				JobLabel:            "job",
				PodMetricsEndpoints: []monitoringv1.PodMetricsEndpoint{},
				Selector: metav1.LabelSelector{
					MatchLabels: map[string]string{"selector": "deployment"},
				},
			},
		}
	}
}

func dashboard(namespace string) func() *grafanav1alpha1.GrafanaDashboard {
	return func() *grafanav1alpha1.GrafanaDashboard {

		return &grafanav1alpha1.GrafanaDashboard{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dashboard",
				Namespace: namespace,
			},
			Spec: grafanav1alpha1.GrafanaDashboardSpec{
				Json: "{}",
			},
		}
	}
}
