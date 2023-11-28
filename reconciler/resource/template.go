package resource

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type TemplateInterface interface {
	Build(ctx context.Context, cl client.Client, o client.Object) (client.Object, error)
	Enabled() bool
	// ReconcilerConfig() ReconcilerConfig
	Reconcile(context.Context, client.Client, *runtime.Scheme, client.Object) error
}

type TemplateBuilderFunction[T client.Object] func(client.Object) (T, error)

type MutationFunction func(context.Context, client.Client, client.Object, client.Object) error

type ReconcilerConfig struct {
	ReconcileProperties []Property
	IgnoreProperties    []Property
	Mutations           []MutationFunction
}

type Template[T client.Object] struct {
	Builder             TemplateBuilderFunction[T]
	IsEnabled           bool
	MutatorFns          []MutationFunction
	ReconcileProperties []Property
	IgnoreProperties    []Property
}

// Build returns a T resource by executing its template function
func (t Template[T]) Build(ctx context.Context, cl client.Client, o client.Object) (client.Object, error) {
	return t.Builder(o)
}

// Enabled indicates if the resource should be present or not
func (t Template[T]) Enabled() bool {
	return t.IsEnabled
}

// Enabled indicates if the resource should be present or not
func (t Template[T]) ReconcilerConfig() ReconcilerConfig {
	// TODO: return a set of defaults if not set
	return ReconcilerConfig{
		ReconcileProperties: t.ReconcileProperties,
		IgnoreProperties:    t.IgnoreProperties,
		Mutations:           t.MutatorFns,
	}
}

func (t Template[T]) MutateTemplate(mutation TemplateBuilderFunction[T]) Template[T] {

	fn := t.Builder
	t.Builder = func(in client.Object) (T, error) {
		o, err := fn(in)
		if err != nil {
			return o, err
		}
		return mutation(o)
	}

	return t
}

func (t Template[T]) M(mutation TemplateBuilderFunction[T]) Template[T] {
	return t.MutateTemplate(mutation)
}
