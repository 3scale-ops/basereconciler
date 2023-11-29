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

	"github.com/3scale-ops/basereconciler/mutators"
	"github.com/3scale-ops/basereconciler/reconciler"
	"github.com/3scale-ops/basereconciler/reconciler/resource"
	"github.com/3scale-ops/basereconciler/test/api/v1alpha1"
	"github.com/3scale-ops/basereconciler/util"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// func init() {
// 	reconciler.Config.AnnotationsDomain = "example.com"
// 	reconciler.Config.ResourcePruner = true
// 	reconciler.Config.ManagedTypes = reconciler.NewManagedTypes().
// 		Register(&corev1.ServiceList{}).
// 		Register(&appsv1.DeploymentList{}).
// 		Register(&autoscalingv2.HorizontalPodAutoscalerList{}).
// 		Register(&policyv1.PodDisruptionBudgetList{})
// }

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

	err = r.ReconcileOwnedResources(ctx, instance, []resource.TemplateInterface{
		resource.Template[*appsv1.Deployment]{
			TemplateBuilder: deployment(req.Namespace),
			IsEnabled:       true,
			EnsureProperties: []resource.Property{
				"metadata.annotations",
				"metadata.labels",
				"spec.minReadySeconds",
				"spec.replicas",
				"spec.selector",
				"spec.strategy",
				"spec.template.metadata",
				"spec.template.spec",
			},
			IgnoreProperties: []resource.Property{
				"metadata.annotations['deployment.kubernetes.io/revision']",
				"spec.template.spec.dnsPolicy",
				"spec.template.spec.schedulerName",
				"spec.template.spec.restartPolicy",
				"spec.template.spec.securityContext",
				"spec.template.spec.terminationGracePeriodSeconds",
				"spec.template.spec.containers[*].terminationMessagePath",
				"spec.template.spec.containers[*].terminationMessagePolicy",
			},
			TemplateMutations: []resource.TemplateMutationFunction{
				mutators.ReconcileDeploymentReplicas(true),
				mutators.RolloutTrigger{
					Name:       "secret",
					SecretName: pointer.String("secret"),
				}.AddToDeployment("example.com"),
			},
		},

		// resource.Template[*corev1.Service]{
		// 	Builder:   service(req.Namespace, instance.Spec.ServiceAnnotations),
		// 	IsEnabled: true,
		// 	ReconcileProperties: []resource.Property{
		// 		"metadata.annotations",
		// 		"metadata.labels",
		// 		"spec.type",
		// 		"spec.selector",
		// 		"spec.ports",
		// 	},
		// 	IgnoreProperties: []resource.Property{
		// 		"spec.clusterIP",
		// 		"spec.clusterIPs",
		// 		"spec.ipFamilies",
		// 		"spec.IpFamilyPolicy",
		// 	},
		// 	MutatorFns: []resource.MutationFunction{
		// 		mutators.ReconcileServiceNodePorts(),
		// 	},
		// },

		// resource.Template[*autoscalingv2.HorizontalPodAutoscaler]{
		// 	Builder:   hpa(req.Namespace),
		// 	IsEnabled: instance.Spec.HPA != nil && *instance.Spec.HPA,
		// 	ReconcileProperties: []resource.Property{
		// 		"metadata.annotations",
		// 		"metadata.labels",
		// 		"spec.scaleTargetRef",
		// 		"spec.minReplicas",
		// 		"spec.maxReplicas",
		// 		"spec.metrics",
		// 	},
		// },
		// resource.Template[*policyv1.PodDisruptionBudget]{
		// 	Builder:   pdb(req.Namespace),
		// 	IsEnabled: instance.Spec.PDB != nil && *instance.Spec.PDB,
		// 	ReconcileProperties: []resource.Property{
		// 		"metadata.annotations",
		// 		"metadata.labels",
		// 		"spec.maxUnavailable",
		// 		"spec.minAvailable",
		// 		"spec.selector",
		// 	},
		// },
	})

	if err != nil {
		logger.Error(err, "unable to reconcile owned resources")
		return ctrl.Result{}, err
	}

	// reconcile the status
	// err = r.ReconcileStatus(ctx, instance,
	// 	[]types.NamespacedName{{Name: "deployment", Namespace: instance.GetNamespace()}}, nil)
	// if err != nil {
	// 	return ctrl.Result{}, err
	// }

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
		Watches(&source.Kind{Type: &corev1.Secret{TypeMeta: metav1.TypeMeta{Kind: "Secret"}}},
			r.SecretEventHandler(&v1alpha1.TestList{}, r.Log)).
		Complete(r)
}

func deployment(namespace string) resource.TemplateBuilderFunction[*appsv1.Deployment] {
	return func(client.Object) (*appsv1.Deployment, error) {
		dep := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "deployment",
				Namespace: namespace,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: pointer.Int32(1),
				Strategy: appsv1.DeploymentStrategy{
					Type: appsv1.RollingUpdateDeploymentStrategyType,
					RollingUpdate: &appsv1.RollingUpdateDeployment{
						MaxSurge:       util.Pointer(intstr.FromString("25%")),
						MaxUnavailable: util.Pointer(intstr.FromString("25%"))},
				},
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
								Name:            "container",
								Image:           "example.com:latest",
								ImagePullPolicy: corev1.PullAlways,
								Resources:       corev1.ResourceRequirements{},
							},
						},
					},
				},
			},
		}

		return dep, nil
	}
}

func service(namespace string, annotations map[string]string) resource.TemplateBuilderFunction[*corev1.Service] {
	return func(client.Object) (*corev1.Service, error) {
		return &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "service",
				Namespace:   namespace,
				Annotations: annotations,
			},
			Spec: corev1.ServiceSpec{
				Type: corev1.ServiceTypeLoadBalancer,
				Ports: []corev1.ServicePort{{
					Name: "port", Port: 80, TargetPort: intstr.FromInt(80), Protocol: corev1.ProtocolTCP}},
				Selector: map[string]string{"selector": "deployment"},
			},
		}, nil
	}
}

func hpa(namespace string) resource.TemplateBuilderFunction[*autoscalingv2.HorizontalPodAutoscaler] {
	return func(client.Object) (*autoscalingv2.HorizontalPodAutoscaler, error) {
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
				Metrics: []autoscalingv2.MetricSpec{{
					Type: autoscalingv2.ResourceMetricSourceType,
					Resource: &autoscalingv2.ResourceMetricSource{
						Name: "cpu",
						Target: autoscalingv2.MetricTarget{
							Type:               autoscalingv2.UtilizationMetricType,
							AverageUtilization: util.Pointer(int32(90)),
						},
					},
				}},
			},
		}, nil
	}
}

func pdb(namespace string) resource.TemplateBuilderFunction[*policyv1.PodDisruptionBudget] {
	return func(client.Object) (*policyv1.PodDisruptionBudget, error) {

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
		}, nil
	}
}
