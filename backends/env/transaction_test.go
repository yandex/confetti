package env

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	ref "golang.yandex/confetti/internal/reflect"
)

func TestFromVarPointerSafety(t *testing.T) {
	const key = "CONFETTI_ENV_POINTER"

	t.Run("typed nil target with absent variable", func(t *testing.T) {
		var target *int

		err := FromVar(key)(t.Context(), target)

		assert.NoError(t, err)
		assert.Nil(t, target)
	})

	t.Run("typed nil target", func(t *testing.T) {
		t.Setenv(key, "42")
		var target *int

		err := FromVar(key)(t.Context(), target)

		assert.Error(t, err)
		assert.Nil(t, target)
	})

	t.Run("nil pointer success", func(t *testing.T) {
		t.Setenv(key, "42")
		var target *int

		err := FromVar(key)(t.Context(), &target)

		if assert.NoError(t, err) && assert.NotNil(t, target) {
			assert.Equal(t, 42, *target)
		}
	})

	t.Run("nil pointer conversion error", func(t *testing.T) {
		t.Setenv(key, "invalid")
		var target *int

		err := FromVar(key)(t.Context(), &target)

		assert.Error(t, err)
		assert.Nil(t, target)
	})

	t.Run("existing pointer conversion error", func(t *testing.T) {
		t.Setenv(key, "invalid")
		original := new(int)
		*original = 7
		target := original

		err := FromVar(key)(t.Context(), &target)

		assert.Error(t, err)
		assert.Same(t, original, target)
		assert.Equal(t, 7, *target)
	})

	t.Run("existing pointer success", func(t *testing.T) {
		t.Setenv(key, "42")
		original := new(int)
		*original = 7
		target := original

		err := FromVar(key)(t.Context(), &target)

		assert.NoError(t, err)
		assert.Same(t, original, target)
		assert.Equal(t, 42, *target)
	})

	t.Run("multiple pointer success", func(t *testing.T) {
		t.Setenv(key, "42")
		var target **int

		err := FromVar(key)(t.Context(), &target)

		if assert.NoError(t, err) && assert.NotNil(t, target) && assert.NotNil(t, *target) {
			assert.Equal(t, 42, **target)
		}
	})

	t.Run("unsupported pointer", func(t *testing.T) {
		t.Setenv(key, "value")
		var target *struct{}

		err := FromVar(key)(t.Context(), &target)

		assert.NoError(t, err)
		assert.Nil(t, target)
	})
}

func TestFromEnvironPointerSafety(t *testing.T) {
	tests := []struct {
		name       string
		value      string
		initial    int
		hasInitial bool
		wantError  bool
		want       int
	}{
		{name: "nil pointer conversion error", value: "invalid", wantError: true},
		{name: "existing pointer conversion error", value: "invalid", initial: 7, hasInitial: true, wantError: true, want: 7},
		{name: "nil pointer success", value: "42", want: 42},
		{name: "existing pointer success", value: "42", initial: 7, hasInitial: true, want: 42},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("CONFETTI_ENV_VALUE", test.value)

			type config struct {
				Value *int `env:"CONFETTI_ENV_VALUE"`
			}

			var original *int
			if test.hasInitial {
				original = new(int)
				*original = test.initial
			}
			target := config{Value: original}

			err := FromEnviron()(t.Context(), &target)

			if test.wantError {
				assert.Error(t, err)
				assert.Same(t, original, target.Value)
				if original != nil {
					assert.Equal(t, test.want, *target.Value)
				}
				return
			}

			if assert.NoError(t, err) && assert.NotNil(t, target.Value) {
				assert.Equal(t, test.want, *target.Value)
			}
			if original != nil {
				assert.Same(t, original, target.Value)
			}
		})
	}
}

func TestFromVarDecoderErrorsAreTransactional(t *testing.T) {
	const key = "CONFETTI_ENV_DECODER"
	t.Setenv(key, "value")

	t.Run("sql scanner", func(t *testing.T) {
		original := &transactionalScanner{value: "initial"}
		target := original

		err := FromVar(key)(t.Context(), &target)

		assert.ErrorIs(t, err, errTransactionalDecoder)
		assert.Same(t, original, target)
		assert.Equal(t, "initial", target.value)
	})

	t.Run("text unmarshaler", func(t *testing.T) {
		original := &transactionalText{value: "initial"}
		target := original

		err := FromVar(key)(t.Context(), &target)

		assert.ErrorIs(t, err, errTransactionalDecoder)
		assert.Same(t, original, target)
		assert.Equal(t, "initial", target.value)
	})

	t.Run("binary unmarshaler", func(t *testing.T) {
		original := &transactionalBinary{value: "initial"}
		target := original

		err := FromVar(key)(t.Context(), &target)

		assert.ErrorIs(t, err, errTransactionalDecoder)
		assert.Same(t, original, target)
		assert.Equal(t, "initial", target.value)
	})

	t.Run("nil text unmarshaler", func(t *testing.T) {
		var target *transactionalText

		err := FromVar(key)(t.Context(), &target)

		assert.ErrorIs(t, err, errTransactionalDecoder)
		assert.Nil(t, target)
	})

	t.Run("skip nested", func(t *testing.T) {
		var target *transactionalSkip

		err := FromVar(key)(t.Context(), &target)

		assert.NoError(t, err)
		assert.Nil(t, target)
	})

	t.Run("successful decoder uses current value", func(t *testing.T) {
		original := transactionalMerge("initial")
		target := &original

		err := FromVar(key)(t.Context(), &target)

		assert.NoError(t, err)
		assert.Same(t, &original, target)
		assert.Equal(t, transactionalMerge("initial:value"), *target)
	})
}

func TestFromEnvironDecoderErrorsAreTransactional(t *testing.T) {
	t.Setenv("CONFETTI_ENV_DECODER", "value")

	t.Run("sql scanner", func(t *testing.T) {
		type config struct {
			Value *transactionalScanner `env:"CONFETTI_ENV_DECODER"`
		}
		original := &transactionalScanner{value: "initial"}
		target := config{Value: original}

		err := FromEnviron()(t.Context(), &target)

		assert.ErrorIs(t, err, errTransactionalDecoder)
		assert.Same(t, original, target.Value)
		assert.Equal(t, "initial", target.Value.value)
	})

	t.Run("text unmarshaler", func(t *testing.T) {
		type config struct {
			Value *transactionalText `env:"CONFETTI_ENV_DECODER"`
		}
		original := &transactionalText{value: "initial"}
		target := config{Value: original}

		err := FromEnviron()(t.Context(), &target)

		assert.ErrorIs(t, err, errTransactionalDecoder)
		assert.Same(t, original, target.Value)
		assert.Equal(t, "initial", target.Value.value)
	})

	t.Run("binary unmarshaler", func(t *testing.T) {
		type config struct {
			Value *transactionalBinary `env:"CONFETTI_ENV_DECODER"`
		}
		original := &transactionalBinary{value: "initial"}
		target := config{Value: original}

		err := FromEnviron()(t.Context(), &target)

		assert.ErrorIs(t, err, errTransactionalDecoder)
		assert.Same(t, original, target.Value)
		assert.Equal(t, "initial", target.Value.value)
	})
}

func TestFromEnvironCommitsDeferredZeroValues(t *testing.T) {
	type deepest struct {
		Int         int               `env:"INT"`
		Bool        bool              `env:"BOOL"`
		Text        string            `env:"TEXT"`
		InvalidInt  int               `env:"INVALID_INT"`
		Skip        transactionalSkip `env:"SKIP"`
		Unsupported struct{}          `env:"UNSUPPORTED"`
	}
	type middle struct {
		Deepest *deepest `env:"DEEPEST"`
	}
	type config struct {
		Middle *middle `env:"MIDDLE"`
	}

	tests := []struct {
		name          string
		key           string
		value         string
		wantError     bool
		wantAllocated bool
	}{
		{name: "zero int", key: "MIDDLE_DEEPEST_INT", value: "0", wantAllocated: true},
		{name: "false bool", key: "MIDDLE_DEEPEST_BOOL", value: "false", wantAllocated: true},
		{name: "empty string", key: "MIDDLE_DEEPEST_TEXT", value: "", wantAllocated: true},
		{name: "absent variable"},
		{name: "conversion error", key: "MIDDLE_DEEPEST_INVALID_INT", value: "invalid", wantError: true},
		{name: "skip nested", key: "MIDDLE_DEEPEST_SKIP", value: "value"},
		{name: "unsupported type", key: "MIDDLE_DEEPEST_UNSUPPORTED", value: "value"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.key != "" {
				t.Setenv(test.key, test.value)
			}

			var target config
			err := FromEnviron(RecursiveKeys)(t.Context(), &target)

			if test.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			if test.wantAllocated {
				assert.NotNil(t, target.Middle)
				if target.Middle != nil {
					assert.NotNil(t, target.Middle.Deepest)
				}
			} else {
				assert.Nil(t, target.Middle)
			}
		})
	}
}

var errTransactionalDecoder = errors.New("transactional decoder error")

type transactionalScanner struct {
	value string
}

func (v *transactionalScanner) Scan(any) error {
	v.value = "changed"
	return errTransactionalDecoder
}

type transactionalText struct {
	value string
}

func (v *transactionalText) UnmarshalText([]byte) error {
	v.value = "changed"
	return errTransactionalDecoder
}

type transactionalBinary struct {
	value string
}

func (v *transactionalBinary) UnmarshalBinary([]byte) error {
	v.value = "changed"
	return errTransactionalDecoder
}

type transactionalSkip struct {
	value string
}

func (v *transactionalSkip) UnmarshalText([]byte) error {
	v.value = "changed"
	return ref.ErrSkipNested
}

type transactionalMerge string

func (v *transactionalMerge) UnmarshalText(value []byte) error {
	*v += transactionalMerge(":" + string(value))
	return nil
}
