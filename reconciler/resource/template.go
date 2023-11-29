package resource

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type TemplateInterface interface {
	Build(ctx context.Context, cl client.Client, o client.Object) (client.Object, error)
	Enabled() bool
	ReconcilerConfig() ReconcilerConfig
}

type TemplateBuilderFunction[T client.Object] func(client.Object) (T, error)

type TemplateMutationFunction func(context.Context, client.Client, client.Object, client.Object) error

type ReconcilerConfig struct {
	ReconcileProperties []Property
	IgnoreProperties    []Property
}

type Template[T client.Object] struct {
	TemplateBuilder   TemplateBuilderFunction[T]
	TemplateMutations []TemplateMutationFunction
	IsEnabled         bool
	EnsureProperties  []Property
	IgnoreProperties  []Property
}

// Build returns a T resource by executing its template function
func (t Template[T]) Build(ctx context.Context, cl client.Client, o client.Object) (client.Object, error) {
	return t.TemplateBuilder(o)
}

// Enabled indicates if the resource should be present or not
func (t Template[T]) Enabled() bool {
	return t.IsEnabled
}

// Enabled indicates if the resource should be present or not
func (t Template[T]) ReconcilerConfig() ReconcilerConfig {
	// TODO: return a set of defaults if not set
	return ReconcilerConfig{
		ReconcileProperties: t.EnsureProperties,
		IgnoreProperties:    t.IgnoreProperties,
	}
}

func (t Template[T]) ChainTemplateBuilder(mutation TemplateBuilderFunction[T]) Template[T] {

	fn := t.TemplateBuilder
	t.TemplateBuilder = func(in client.Object) (T, error) {
		o, err := fn(in)
		if err != nil {
			return o, err
		}
		return mutation(o)
	}

	return t
}

func (t Template[T]) C(mutation TemplateBuilderFunction[T]) Template[T] {
	return t.ChainTemplateBuilder(mutation)
}
