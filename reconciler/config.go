package reconciler

import (
	"github.com/3scale-ops/basereconciler/reconciler/resource"
	"k8s.io/apimachinery/pkg/runtime/schema"
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

func (opt ReconcilerConfig) ResourceConfigForGVK(gvk schema.GroupVersionKind) resource.ReconcilerConfig {
	return opt.ResourceReconcilerConfig[gvk.String()]
}
