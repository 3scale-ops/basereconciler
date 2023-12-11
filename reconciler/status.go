package reconciler

import (
	"context"

	"github.com/3scale-ops/basereconciler/status"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ReconcileStatus can reconcile the status of a custom resource when the resource implements
// the status.ObjectWithAppStatus interface. It is specifically targeted for the status of custom
// resources that deploy Deployments/StatefulSets, as it can aggregate the status of those into the
// status of the custom resource. It also accepts functions with signature "func() bool" that can
// reconcile the status of the custom resource and return whether update is required or not.
func (r *Reconciler) ReconcileStatus(ctx context.Context, instance status.ObjectWithAppStatus,
	deployments, statefulsets []types.NamespacedName, mutators ...func() bool) error {
	logger := log.FromContext(ctx)
	update := false
	status := instance.GetStatus()

	// Aggregate the status of all Deployments owned
	// by this instance
	for _, key := range deployments {
		deployment := &appsv1.Deployment{}
		deploymentStatus := status.GetDeploymentStatus(key)
		if err := r.Client.Get(ctx, key, deployment); err != nil {
			return err
		}

		if !equality.Semantic.DeepEqual(deploymentStatus, deployment.Status) {
			status.SetDeploymentStatus(key, &deployment.Status)
			update = true
		}
	}

	// Aggregate the status of all StatefulSets owned
	// by this instance
	for _, key := range statefulsets {
		sts := &appsv1.StatefulSet{}
		stsStatus := status.GetStatefulSetStatus(key)
		if err := r.Client.Get(ctx, key, sts); err != nil {
			return err
		}

		if !equality.Semantic.DeepEqual(stsStatus, sts.Status) {
			status.SetStatefulSetStatus(key, &sts.Status)
			update = true
		}
	}

	// TODO: calculate health

	// call mutators
	for _, fn := range mutators {
		if fn() {
			update = true
		}
	}

	if update {
		if err := r.Client.Status().Update(ctx, instance); err != nil {
			logger.Error(err, "unable to update status")
			return err
		}
	}

	return nil
}
