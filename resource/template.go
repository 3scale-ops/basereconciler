package resource

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type TemplateInterface interface {
	Build(ctx context.Context, cl client.Client, o client.Object) (client.Object, error)
	Enabled() bool
	GetEnsureProperties() []Property
	GetIgnoreProperties() []Property
}

type TemplateBuilderFunction[T client.Object] func(client.Object) (T, error)

type TemplateMutationFunction func(context.Context, client.Client, client.Object) error

type Template[T client.Object] struct {
	TemplateBuilder   TemplateBuilderFunction[T]
	TemplateMutations []TemplateMutationFunction
	IsEnabled         bool
	EnsureProperties  []Property
	IgnoreProperties  []Property
}

// Build returns a T resource by executing its template function
func (t Template[T]) Build(ctx context.Context, cl client.Client, o client.Object) (client.Object, error) {
	o, err := t.TemplateBuilder(o)
	if err != nil {
		return nil, err
	}
	for _, fn := range t.TemplateMutations {
		if err := fn(ctx, cl, o); err != nil {
			return nil, err
		}
	}
	return o, nil
}

// Enabled indicates if the resource should be present or not
func (t Template[T]) Enabled() bool {
	return t.IsEnabled
}

// GetEnsureProperties returns the list of properties that should be reconciled
func (t Template[T]) GetEnsureProperties() []Property {
	return t.EnsureProperties
}

// GetIgnoreProperties returns the list of properties that should be ignored
func (t Template[T]) GetIgnoreProperties() []Property {
	return t.IgnoreProperties
}

// func (t Template[T]) GVK() (schema.GroupVersionKind, error) {
// 	o, err := t.TemplateBuilder(o)
// 	if err != nil {
// 		return schema.GroupVersionKind{}, err
// 	}
// 	gvk, err := apiutil.GVKForObject(o, scheme)
// 	if err != nil {
// 		return nil, err
// 	}
// }

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
