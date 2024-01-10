package reconciler

import (
	"reflect"
	"sync"

	"github.com/3scale-ops/basereconciler/util"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

type ReconcilerWithTypeTracker interface {
	reconcile.Reconciler
	BuildTypeTracker(ctrl controller.Controller)
}

func SetupWithDynamicTypeWatches(r ReconcilerWithTypeTracker, bldr *builder.Builder) error {
	ctrl, err := bldr.Build(r)
	if err != nil {
		return err
	}
	r.BuildTypeTracker(ctrl)
	return nil
}

type typeTracker struct {
	seenTypes []schema.GroupVersionKind
	ctrl      controller.Controller
	mu        sync.Mutex
}

func (tt *typeTracker) trackType(gvk schema.GroupVersionKind) bool {
	if !util.ContainsBy(tt.seenTypes, func(x schema.GroupVersionKind) bool {
		return reflect.DeepEqual(x, gvk)
	}) {
		tt.mu.Lock()
		defer tt.mu.Unlock()
		tt.seenTypes = append(tt.seenTypes, gvk)
		return true
	}
	return false
}

func (r *Reconciler) watchOwned(gvk schema.GroupVersionKind, owner client.Object) error {
	o, err := util.NewObjectFromGVK(gvk, r.Scheme)
	if err != nil {
		return err
	}
	r.typeTracker.mu.Lock()
	defer r.typeTracker.mu.Unlock()
	err = r.typeTracker.ctrl.Watch(&source.Kind{Type: o}, &handler.EnqueueRequestForOwner{OwnerType: owner, IsController: true})
	if err != nil {
		return err
	}
	return nil
}

// Reconciler is expected to be overwriten
func (r *Reconciler) BuildTypeTracker(ctrl controller.Controller) {
	r.typeTracker = typeTracker{
		seenTypes: []schema.GroupVersionKind{},
		ctrl:      ctrl,
	}
}
