package reconciler

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/3scale-ops/basereconciler/resource"
	"github.com/3scale-ops/basereconciler/util"
	"github.com/go-logr/logr"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

func TestResult_ShouldReturn(t *testing.T) {
	type fields struct {
		Action       action
		RequeueAfter time.Duration
		Error        error
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name:   "Empty result returns false",
			fields: fields{},
			want:   false,
		},
		{
			name: "continueAction returns false",
			fields: fields{
				Action: ContinueAction,
				Error:  nil,
			},
			want: false,
		},
		{
			name: "returnAction returns true",
			fields: fields{
				Action: ReturnAction,
			},
			want: true,
		},
		{
			name: "Error returns true",
			fields: fields{
				Error: fmt.Errorf("error"),
			},
			want: true,
		},
		{
			name: "returnAndRequeueAction true returns true",
			fields: fields{
				Action: ReturnAndRequeueAction,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Result{
				Action:       tt.fields.Action,
				RequeueAfter: tt.fields.RequeueAfter,
				Error:        tt.fields.Error,
			}
			if got := result.ShouldReturn(); got != tt.want {
				t.Errorf("Result.ShouldReturn() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResult_Values(t *testing.T) {
	type fields struct {
		Action       action
		RequeueAfter time.Duration
		Error        error
	}
	tests := []struct {
		name   string
		fields fields
		want1  ctrl.Result
		want2  error
	}{
		{
			name: "Returns expected results",
			fields: fields{
				Action:       "",
				RequeueAfter: 0,
				Error:        errors.New("error"),
			},
			want1: reconcile.Result{},
			want2: errors.New("error"),
		},
		{
			name: "Returns expected results, with 'RequeueAfter'",
			fields: fields{
				Action:       "",
				RequeueAfter: 60 * time.Second,
				Error:        nil,
			},
			want1: reconcile.Result{RequeueAfter: 60 * time.Second},
			want2: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Result{
				Action:       tt.fields.Action,
				RequeueAfter: tt.fields.RequeueAfter,
				Error:        tt.fields.Error,
			}
			got1, got2 := result.Values()
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("Result.Values() = %v, want %v", got1, tt.want1)
			}
			if tt.want2 != nil && (got2.Error() != tt.want2.Error()) {
				t.Errorf("Result.Values() = %v, want %v", got2, tt.want2)
			}
		})
	}
}

func TestReconciler_ManageResourceLifecycle(t *testing.T) {
	type fields struct {
		Client client.Client
		Log    logr.Logger
		Scheme *runtime.Scheme
	}
	type args struct {
		req  reconcile.Request
		obj  client.Object
		opts []lifecycleOption
	}
	tests := []struct {
		name       string
		fields     fields
		args       args
		want       Result
		wantObject client.Object
	}{
		{
			name: "Gets the resource",
			fields: fields{
				Client: fake.NewClientBuilder().WithObjects(
					&corev1.Service{ObjectMeta: metav1.ObjectMeta{
						Name: "my-resource", Namespace: "ns",
					}},
				).Build(),
				Log:    logr.Discard(),
				Scheme: scheme.Scheme,
			},
			args: args{
				req:  reconcile.Request{NamespacedName: types.NamespacedName{Name: "my-resource", Namespace: "ns"}},
				obj:  &corev1.Service{},
				opts: []lifecycleOption{},
			},
			want: Result{
				Action:       ContinueAction,
				RequeueAfter: 0,
				Error:        nil,
			},
			wantObject: nil,
		},
		{
			name: "Resource not found",
			fields: fields{
				Client: fake.NewClientBuilder().Build(),
				Log:    logr.Discard(),
				Scheme: scheme.Scheme,
			},
			args: args{
				req:  reconcile.Request{NamespacedName: types.NamespacedName{Name: "my-resource", Namespace: "ns"}},
				obj:  &corev1.Service{},
				opts: []lifecycleOption{},
			},
			want: Result{
				Action:       ReturnAction,
				RequeueAfter: 0,
				Error:        nil,
			},
			wantObject: nil,
		},
		{
			name: "Resource being deleted",
			fields: fields{
				Client: fake.NewClientBuilder().WithObjects(
					&corev1.Service{ObjectMeta: metav1.ObjectMeta{
						Name: "my-resource", Namespace: "ns",
						DeletionTimestamp: &metav1.Time{Time: time.Now()},
						Finalizers:        []string{"finalizer"},
					}},
				).Build(),
				Log:    logr.Discard(),
				Scheme: scheme.Scheme,
			},
			args: args{
				req:  reconcile.Request{NamespacedName: types.NamespacedName{Name: "my-resource", Namespace: "ns"}},
				obj:  &corev1.Service{},
				opts: []lifecycleOption{},
			},
			want: Result{
				Action:       ReturnAction,
				RequeueAfter: 0,
				Error:        nil,
			},
			wantObject: nil,
		},
		{
			name: "Adds finalizer",
			fields: fields{
				Client: fake.NewClientBuilder().WithObjects(
					&corev1.Service{ObjectMeta: metav1.ObjectMeta{
						Name: "my-resource", Namespace: "ns",
					}},
				).Build(),
				Log:    logr.Discard(),
				Scheme: scheme.Scheme,
			},
			args: args{
				req:  reconcile.Request{NamespacedName: types.NamespacedName{Name: "my-resource", Namespace: "ns"}},
				obj:  &corev1.Service{},
				opts: []lifecycleOption{WithFinalizer("finalizer")},
			},
			want: Result{
				Action:       ReturnAndRequeueAction,
				RequeueAfter: 0,
				Error:        nil,
			},
			wantObject: &corev1.Service{ObjectMeta: metav1.ObjectMeta{
				Name: "my-resource", Namespace: "ns",
				Finalizers: []string{"finalizer"},
			}},
		},
		{
			name: "Executes finalization logic",
			fields: fields{
				Client: fake.NewClientBuilder().WithObjects(
					&corev1.Service{ObjectMeta: metav1.ObjectMeta{
						Name: "my-resource", Namespace: "ns",
						Finalizers:        []string{"finalizer"},
						DeletionTimestamp: &metav1.Time{Time: time.Now()},
					}},
				).Build(),
				Log:    logr.Discard(),
				Scheme: scheme.Scheme,
			},
			args: args{
				req: reconcile.Request{NamespacedName: types.NamespacedName{Name: "my-resource", Namespace: "ns"}},
				obj: &corev1.Service{},
				opts: []lifecycleOption{
					WithFinalizer("finalizer"),
					WithFinalizationFunc(func(ctx context.Context, c client.Client) error {
						o := &corev1.ServiceAccount{}
						o.SetName("test-finalization-logic")
						o.SetNamespace("ns")
						c.Create(ctx, o)
						return nil
					}),
				},
			},
			want: Result{
				Action:       ReturnAction,
				RequeueAfter: 0,
				Error:        nil,
			},
			wantObject: &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{
				Name: "test-finalization-logic", Namespace: "ns",
			}},
		},
		{
			name: "Removes the finalizer",
			fields: fields{
				Client: fake.NewClientBuilder().WithObjects(
					&corev1.Service{ObjectMeta: metav1.ObjectMeta{
						Name: "my-resource", Namespace: "ns",
						Finalizers:        []string{"finalizer"},
						DeletionTimestamp: &metav1.Time{Time: time.Now()},
					}},
				).Build(),
				Log:    logr.Discard(),
				Scheme: scheme.Scheme,
			},
			args: args{
				req: reconcile.Request{NamespacedName: types.NamespacedName{Name: "my-resource", Namespace: "ns"}},
				obj: &corev1.Service{},
				opts: []lifecycleOption{
					WithFinalizer("finalizer"),
				},
			},
			want: Result{
				Action:       ReturnAction,
				RequeueAfter: 0,
				Error:        nil,
			},
			wantObject: &corev1.Service{ObjectMeta: metav1.ObjectMeta{
				Name: "my-resource", Namespace: "ns",
				DeletionTimestamp: &metav1.Time{Time: time.Now()},
			}},
		},
		{
			name: "Adds the finalizer",
			fields: fields{
				Client: fake.NewClientBuilder().WithObjects(
					&corev1.Service{ObjectMeta: metav1.ObjectMeta{
						Name: "my-resource", Namespace: "ns",
					}},
				).Build(),
				Log:    logr.Discard(),
				Scheme: scheme.Scheme,
			},
			args: args{
				req: reconcile.Request{NamespacedName: types.NamespacedName{Name: "my-resource", Namespace: "ns"}},
				obj: &corev1.Service{},
				opts: []lifecycleOption{
					WithFinalizer("finalizer"),
				},
			},
			want: Result{
				Action:       ReturnAndRequeueAction,
				RequeueAfter: 0,
				Error:        nil,
			},
			wantObject: &corev1.Service{ObjectMeta: metav1.ObjectMeta{
				Name: "my-resource", Namespace: "ns",
				Finalizers: []string{"finalizer"},
			}},
		},
		{
			name: "Runs initialization logic",
			fields: fields{
				Client: fake.NewClientBuilder().WithObjects(
					&corev1.Service{ObjectMeta: metav1.ObjectMeta{
						Name: "my-resource", Namespace: "ns",
					}},
				).Build(),
				Log:    logr.Discard(),
				Scheme: scheme.Scheme,
			},
			args: args{
				req: reconcile.Request{NamespacedName: types.NamespacedName{Name: "my-resource", Namespace: "ns"}},
				obj: &corev1.Service{},
				opts: []lifecycleOption{
					WithInitializationFunc(func(ctx context.Context, c client.Client, o client.Object) error {
						o.SetLabels(map[string]string{"initialized": "yes"})
						return nil
					}),
				},
			},
			want: Result{
				Action:       ReturnAndRequeueAction,
				RequeueAfter: 0,
				Error:        nil,
			},
			wantObject: &corev1.Service{ObjectMeta: metav1.ObjectMeta{
				Name: "my-resource", Namespace: "ns",
				Labels: map[string]string{"initialized": "yes"},
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reconciler{
				Client: tt.fields.Client,
				Log:    tt.fields.Log,
				Scheme: tt.fields.Scheme,
			}
			got := r.ManageResourceLifecycle(context.TODO(), tt.args.req, tt.args.obj, tt.args.opts...)
			if diff := cmp.Diff(got, tt.want); len(diff) > 0 {
				t.Errorf("Reconciler.ManageResourceLifecycle() diff = %v", diff)
			}
			if tt.wantObject != nil {
				o := tt.wantObject.DeepCopyObject().(client.Object)
				tt.fields.Client.Get(context.TODO(), types.NamespacedName{Name: tt.wantObject.GetName(), Namespace: tt.wantObject.GetNamespace()}, o)
				if diff := cmp.Diff(o, tt.wantObject,
					util.IgnoreProperty("ResourceVersion"),
					util.IgnoreProperty("DeletionTimestamp"),
					util.IgnoreProperty("Kind"),
					util.IgnoreProperty("APIVersion")); len(diff) > 0 {
					t.Errorf("Reconciler.ManageResourceLifecycle() diff = %v", diff)
				}
			}
		})
	}
}

type testController struct {
	reconcile.Reconciler
}

func (c *testController) Watch(src source.Source) error {
	return nil
}
func (c *testController) Start(ctx context.Context) error { return nil }
func (c *testController) GetLogger() logr.Logger          { return logr.Discard() }

func TestReconciler_ReconcileOwnedResources(t *testing.T) {

	type fields struct {
		Client    client.Client
		Log       logr.Logger
		Scheme    *runtime.Scheme
		SeenTypes []schema.GroupVersionKind
		mgr       manager.Manager
	}
	type args struct {
		owner client.Object
		list  []resource.TemplateInterface
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   Result
	}{
		{
			name: "Creates owned resources",
			fields: fields{
				Client:    fake.NewClientBuilder().Build(),
				Log:       logr.Discard(),
				Scheme:    scheme.Scheme,
				SeenTypes: []schema.GroupVersionKind{},
				mgr:       func() manager.Manager { mgr, _ := ctrl.NewManager(&rest.Config{}, ctrl.Options{}); return mgr }(),
			},
			args: args{
				owner: &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "owner", Namespace: "ns"}},
				list: []resource.TemplateInterface{
					resource.NewTemplateFromObjectFunction[*corev1.Service](
						func() *corev1.Service {
							return &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "service", Namespace: "ns"}}
						}),
					resource.NewTemplateFromObjectFunction[*corev1.ConfigMap](
						func() *corev1.ConfigMap {
							return &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "cm", Namespace: "ns"}}
						}),
				},
			},
			want: Result{
				Action:       ReturnAndRequeueAction,
				RequeueAfter: 0,
				Error:        nil,
			},
		},
		{
			name: "Updates owned resources and does not add new watches",
			fields: fields{
				Client: fake.NewClientBuilder().WithObjects(
					&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "owner", Namespace: "ns"}},
					&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "service", Namespace: "ns"}},
					&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "cm", Namespace: "ns"}},
				).Build(),
				Log:    logr.Discard(),
				Scheme: scheme.Scheme,
				SeenTypes: []schema.GroupVersionKind{
					{Group: "", Version: "v1", Kind: "Service"},
					{Group: "", Version: "v1", Kind: "ConfigMap"},
				},
				mgr: func() manager.Manager { mgr, _ := ctrl.NewManager(&rest.Config{}, ctrl.Options{}); return mgr }(),
			},
			args: args{
				owner: &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "owner", Namespace: "ns"}},
				list: []resource.TemplateInterface{
					resource.NewTemplateFromObjectFunction[*corev1.Service](
						func() *corev1.Service {
							return &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "service", Namespace: "ns", Labels: map[string]string{"key": "value"}}}
						}),
					resource.NewTemplateFromObjectFunction[*corev1.ConfigMap](
						func() *corev1.ConfigMap {
							return &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "cm", Namespace: "ns"}}
						}),
				},
			},
			want: Result{
				Action:       ContinueAction,
				RequeueAfter: 0,
				Error:        nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reconciler{
				Client: tt.fields.Client,
				Scheme: tt.fields.Scheme,
				typeTracker: typeTracker{
					seenTypes: tt.fields.SeenTypes,
					ctrl:      &testController{},
				},
				mgr: tt.fields.mgr,
			}
			got := r.ReconcileOwnedResources(context.TODO(), tt.args.owner, tt.args.list)
			if diff := cmp.Diff(got, tt.want); len(diff) > 0 {
				t.Errorf("Reconciler.ReconcileOwnedResources() = %v, want %v", got, tt.want)

			}
		})
	}
}
