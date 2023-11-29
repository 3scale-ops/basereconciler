package reconciler

import (
	"github.com/3scale-ops/basereconciler/reconciler/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ReconcilerManagedTypes []client.ObjectList

func (mts ReconcilerManagedTypes) Register(mt client.ObjectList) ReconcilerManagedTypes {
	mts = append(mts, mt)
	return mts
}

func NewManagedTypes() ReconcilerManagedTypes {
	return ReconcilerManagedTypes{}
}

type ReconcilerConfig struct {
	ManagedTypes             ReconcilerManagedTypes
	AnnotationsDomain        string
	ResourcePruner           bool
	ResourceReconcilerConfig map[string]resource.ReconcilerConfig
}

// func (opt ReconcilerConfig) ResourceConfigForGVK(gvk schema.GroupVersionKind) resource.ReconcilerConfig {
// 	return opt.ResourceReconcilerConfig[gvk.String()]
// }

var Config ReconcilerConfig = ReconcilerConfig{
	AnnotationsDomain: "basereconciler.3cale.net",
	ResourcePruner:    true,
	ManagedTypes:      ReconcilerManagedTypes{
		// &corev1.ServiceList{},
		// &corev1.ConfigMapList{},
		// &appsv1.DeploymentList{},
		// &appsv1.StatefulSetList{},
		// &externalsecretsv1beta1.ExternalSecretList{},
		// &grafanav1alpha1.GrafanaDashboardList{},
		// &autoscalingv2.HorizontalPodAutoscalerList{},
		// &policyv1.PodDisruptionBudgetList{},
		// &monitoringv1.PodMonitorList{},
		// &rbacv1.RoleBindingList{},
		// &rbacv1.RoleList{},
		// &corev1.ServiceAccountList{},
		// &pipelinev1beta1.PipelineList{},
		// &pipelinev1beta1.TaskList{},
	},
}
