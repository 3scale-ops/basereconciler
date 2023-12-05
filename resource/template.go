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

// TemplateBuilderFunction is a function that returns a k8s API object (client.Object) when
// called. TemplateBuilderFunction has no access to cluster live info
type TemplateBuilderFunction[T client.Object] func(client.Object) (T, error)

// func NewBuilderFunctionFromObject[T client.Object](o T) TemplateBuilderFunction[T] {
// 	return func(client.Object) (T, error) {
// 		return o, nil
// 	}
// }

// TemplateMutationFunction represents mutation functions that require an API client, generally
// because they need to retrieve live cluster information to mutate the object
type TemplateMutationFunction func(context.Context, client.Client, client.Object) error

type Template[T client.Object] struct {
	// TemplateBuilder is the function that is used as the basic
	// tempalte for the object. It is called by Build() to create the
	// object.
	TemplateBuilder TemplateBuilderFunction[T]
	// TemplateMutations are functions that are called during Build() after
	// TemplateBuilder has ben invoked, to perform mutations on the object that require
	// access to an API client.
	TemplateMutations []TemplateMutationFunction
	// IsEnabled specifies whether the resourse describe by this Template should
	// exists or not
	IsEnabled bool
	// EnsureProperties are the properties from the desired object that should be enforced
	// to the live object. The syntax is jsonpath.
	EnsureProperties []Property
	// IgnoreProperties are the properties from the live object that should not trigger
	// updates. This is used to ignore nested properties within the "EnsuredProperties". The
	// syntax is jsonpath.
	IgnoreProperties []Property
}

func NewTemplate[T client.Object](tb TemplateBuilderFunction[T],
	enabled bool, mutations ...TemplateMutationFunction) *Template[T] {
	return &Template[T]{
		TemplateBuilder:   tb,
		TemplateMutations: mutations,
		IsEnabled:         enabled,
		EnsureProperties:  []Property{},
		IgnoreProperties:  []Property{},
	}
}

func NewTemplateFromObjectFunction[T client.Object](fn func() T,
	enabled bool, mutations ...TemplateMutationFunction) *Template[T] {
	return &Template[T]{
		TemplateBuilder:   func(client.Object) (T, error) { return fn(), nil },
		TemplateMutations: mutations,
		IsEnabled:         enabled,
		EnsureProperties:  []Property{},
		IgnoreProperties:  []Property{},
	}
}

// Build returns a T resource by executing its template function
func (t *Template[T]) Build(ctx context.Context, cl client.Client, o client.Object) (client.Object, error) {
	o, err := t.TemplateBuilder(o)
	if err != nil {
		return nil, err
	}
	for _, fn := range t.TemplateMutations {
		if err := fn(ctx, cl, o); err != nil {
			return nil, err
		}
	}
	return o.DeepCopyObject().(client.Object), nil
}

// Enabled indicates if the resource should be present or not
func (t *Template[T]) Enabled() bool {
	return t.IsEnabled
}

// GetEnsureProperties returns the list of properties that should be reconciled
func (t *Template[T]) GetEnsureProperties() []Property {
	return t.EnsureProperties
}

// GetIgnoreProperties returns the list of properties that should be ignored
func (t *Template[T]) GetIgnoreProperties() []Property {
	return t.IgnoreProperties
}

// Apply chains template functions to make them composable
func (t *Template[T]) Apply(mutation TemplateBuilderFunction[T]) *Template[T] {

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

func (t *Template[T]) Chain(mutation TemplateBuilderFunction[T]) *Template[T] {
	return t.Apply(mutation)
}
