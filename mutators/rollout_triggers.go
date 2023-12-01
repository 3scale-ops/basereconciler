package mutators

import (
	"context"
	"fmt"

	"github.com/3scale-ops/basereconciler/config"
	"github.com/3scale-ops/basereconciler/resource"
	"github.com/3scale-ops/basereconciler/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// RolloutTrigger defines a configuration source that should trigger a
// rollout whenever the data within that configuration source changes
type RolloutTrigger struct {
	Name          string
	ConfigMapName *string
	SecretName    *string
}

// GetHash returns the hash of the data container in the RolloutTrigger
// config source
func (rt RolloutTrigger) GetHash(ctx context.Context, cl client.Client, namespace string) (string, error) {

	if rt.SecretName != nil {
		secret := &corev1.Secret{}
		key := types.NamespacedName{Name: *rt.SecretName, Namespace: namespace}
		if err := cl.Get(ctx, key, secret); err != nil {
			if errors.IsNotFound(err) {
				return "", nil
			}
			return "", err
		}
		return util.Hash(secret.Data), nil

	} else if rt.ConfigMapName != nil {
		cm := &corev1.ConfigMap{}
		key := types.NamespacedName{Name: *rt.ConfigMapName, Namespace: namespace}
		if err := cl.Get(ctx, key, cm); err != nil {
			if errors.IsNotFound(err) {
				return "", nil
			}
			return "", err
		}
		return util.Hash(cm.Data), nil

	} else {
		return "", fmt.Errorf("empty rollout trigger")
	}
}

// GetAnnotationKey returns the annotation key to be used in the Pods that read
// from the config source defined in the RolloutTrigger
func (rt RolloutTrigger) GetAnnotationKey(annotationsDomain string) string {
	if rt.SecretName != nil {
		return fmt.Sprintf("%s/%s.%s", string(annotationsDomain), rt.Name, "secret-hash")
	}
	return fmt.Sprintf("%s/%s.%s", string(annotationsDomain), rt.Name, "configmap-hash")
}

// reconcileRolloutTriggers modifies the Deployment with the appropriate rollout triggers (annotations)
func (trigger RolloutTrigger) AddToDeployment(params ...string) resource.TemplateMutationFunction {
	var domain string
	if len(params) == 0 {
		domain = config.GetAnnotationsDomain()
	} else {
		domain = params[0]
	}
	return func(ctx context.Context, cl client.Client, desired client.Object) error {

		ddep := desired.(*appsv1.Deployment)
		if ddep.Spec.Template.ObjectMeta.Annotations == nil {
			ddep.Spec.Template.ObjectMeta.Annotations = map[string]string{}
		}
		hash, err := trigger.GetHash(ctx, cl, ddep.GetNamespace())
		if err != nil {
			return err
		}
		ddep.Spec.Template.ObjectMeta.Annotations[trigger.GetAnnotationKey(domain)] = hash
		return nil
	}
}

// reconcileRolloutTriggers modifies the StatefulSet with the appropriate rollout triggers (annotations)
func (trigger RolloutTrigger) AddToStatefulSet(params ...string) resource.TemplateMutationFunction {
	var domain string
	if len(params) == 0 {
		domain = config.GetAnnotationsDomain()
	} else {
		domain = params[0]
	}
	return func(ctx context.Context, cl client.Client, desired client.Object) error {

		dss := desired.(*appsv1.StatefulSet)
		if dss.Spec.Template.ObjectMeta.Annotations == nil {
			dss.Spec.Template.ObjectMeta.Annotations = map[string]string{}
		}
		hash, err := trigger.GetHash(ctx, cl, dss.GetNamespace())
		if err != nil {
			return err
		}
		dss.Spec.Template.ObjectMeta.Annotations[trigger.GetAnnotationKey(domain)] = hash
		return nil
	}
}
