package confetti

import (
	"context"
)

// ValueSetter receives target to set value to
type ValueSetter func(ctx context.Context, target any) error

// Loader holds options and context of config processing
type Loader struct{}

// NewLoader returns new Loader instance with given options
func NewLoader(opts ...LoaderOpt) *Loader {
	var l Loader
	for _, opt := range opts {
		opt(&l)
	}
	return &l
}

// Load calls all loaders and fills targets with values
func (l *Loader) Load(ctx context.Context, loaders ...func(context.Context) error) error {
	for _, loader := range loaders {
		if err := loader(ctx); err != nil {
			return err
		}
	}
	return nil
}

// To returns new anonymous loader function for given target
func To(target any, setters ...ValueSetter) func(context.Context) error {
	return func(ctx context.Context) error {
		for _, setter := range setters {
			if err := setter(ctx, target); err != nil {
				return err
			}
		}
		return nil
	}
}

// Fill is a shorthand for (NewLoader).Load(To()) methods call.
// It is useful for loading simple values to basic variables
func Fill(ctx context.Context, target any, setters ...ValueSetter) error {
	return NewLoader().Load(ctx, To(target, setters...))
}
