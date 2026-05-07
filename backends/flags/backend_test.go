package flags

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.yandex/confetti/internal/ptr"
)

type flagValue string

func (f *flagValue) String() string {
	return string(*f)
}

func (f *flagValue) Set(s string) error {
	*f = flagValue(strings.ReplaceAll(s, "*", ""))
	return nil
}

type customStringFlagValue string

func (f *customStringFlagValue) String() string {
	return "custom:" + string(*f)
}

func (f *customStringFlagValue) Set(value string) error {
	*f = customStringFlagValue(value)
	return nil
}

type pointerStringerValue string

func (f *pointerStringerValue) String() string {
	return "custom:" + string(*f)
}

type getterFlagValue string

func (f *getterFlagValue) String() string {
	return string(*f)
}

func (f *getterFlagValue) Set(value string) error {
	*f = getterFlagValue(value)
	return nil
}

func (f *getterFlagValue) Get() any {
	return string(*f)
}

type getterBoolFlagValue bool

func (f *getterBoolFlagValue) String() string {
	return strconv.FormatBool(bool(*f))
}

func (f *getterBoolFlagValue) Set(value string) error {
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return err
	}
	*f = getterBoolFlagValue(parsed)
	return nil
}

func (*getterBoolFlagValue) IsBoolFlag() bool {
	return true
}

func (f *getterBoolFlagValue) Get() any {
	return bool(*f)
}

type boolFlagValue bool

func (f *boolFlagValue) String() string {
	return strconv.FormatBool(bool(*f))
}

func (f *boolFlagValue) Set(s string) error {
	value, err := strconv.ParseBool(s)
	if err != nil {
		return err
	}
	*f = boolFlagValue(value)
	return nil
}

func (*boolFlagValue) IsBoolFlag() bool {
	return true
}

type explicitBoolFlagValue bool

func (f *explicitBoolFlagValue) String() string {
	return strconv.FormatBool(bool(*f))
}

func (f *explicitBoolFlagValue) Set(value string) error {
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return err
	}
	*f = explicitBoolFlagValue(parsed)
	return nil
}

type conditionalBoolFlagValue struct {
	allowShorthand bool
	value          bool
}

func (f *conditionalBoolFlagValue) String() string {
	return strconv.FormatBool(f.value)
}

func (f *conditionalBoolFlagValue) Set(value string) error {
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return err
	}
	f.value = parsed
	return nil
}

func (f *conditionalBoolFlagValue) IsBoolFlag() bool {
	return f.allowShorthand
}

type scannerValue string

func (f *scannerValue) Scan(value any) error {
	text, ok := value.(string)
	if !ok {
		return errors.New("scanner value must be a string")
	}
	*f = scannerValue(text)
	return nil
}

type textValue string

func (f *textValue) UnmarshalText(value []byte) error {
	*f = textValue(value)
	return nil
}

type binaryValue string

func (f *binaryValue) UnmarshalBinary(value []byte) error {
	*f = binaryValue(value)
	return nil
}

type failingScannerValue string

func (f *failingScannerValue) Scan(any) error {
	*f = "changed"
	return errors.New("scanner value error")
}

type failingTextValue string

func (f *failingTextValue) UnmarshalText([]byte) error {
	*f = "changed"
	return errors.New("text value error")
}

type failingBinaryValue string

func (f *failingBinaryValue) UnmarshalBinary([]byte) error {
	*f = "changed"
	return errors.New("binary value error")
}

var errFlagValue = errors.New("flag value error")

type failingFlagValue string

func (f *failingFlagValue) String() string {
	return string(*f)
}

func (*failingFlagValue) Set(string) error {
	return errFlagValue
}

type collectionErrorValue struct {
	value string
	fail  bool
}

func (f *collectionErrorValue) String() string {
	return f.value
}

func (f *collectionErrorValue) Set(value string) error {
	if f.fail {
		return errFlagValue
	}
	f.value = value
	return nil
}

type mergeTextValue string

func (f *mergeTextValue) UnmarshalText(value []byte) error {
	*f += mergeTextValue(":" + string(value))
	return nil
}

type decoderPriorityValue string

func (f *decoderPriorityValue) Scan(any) error {
	*f = "scanner"
	return nil
}

func (f *decoderPriorityValue) UnmarshalText([]byte) error {
	*f = "text"
	return nil
}

func (f *decoderPriorityValue) UnmarshalBinary([]byte) error {
	*f = "binary"
	return nil
}

type textDecoderPriorityValue string

func (f *textDecoderPriorityValue) UnmarshalText([]byte) error {
	*f = "text"
	return nil
}

func (f *textDecoderPriorityValue) UnmarshalBinary([]byte) error {
	*f = "binary"
	return nil
}

type flagDecoderPriorityValue string

func (f *flagDecoderPriorityValue) String() string {
	return string(*f)
}

func (f *flagDecoderPriorityValue) Set(string) error {
	*f = "flag"
	return nil
}

func (f *flagDecoderPriorityValue) Scan(any) error {
	*f = "scanner"
	return nil
}

func (f *flagDecoderPriorityValue) UnmarshalText([]byte) error {
	*f = "text"
	return nil
}

func (f *flagDecoderPriorityValue) UnmarshalBinary([]byte) error {
	*f = "binary"
	return nil
}

type directFlagValue struct {
	value *string
}

func (f directFlagValue) String() string {
	return *f.value
}

func (f directFlagValue) Set(value string) error {
	*f.value = value
	return nil
}

type unsupportedValue struct {
	Value string
}

func swapOsArgs(args ...string) (revert func()) {
	argsCopy := slices.Clone(os.Args)
	os.Args = args
	return func() { os.Args = argsCopy }
}

func TestUsage(t *testing.T) {
	var opts flagOpts
	usage := "test"
	Usage(usage)(&opts)
	assert.Equal(t, "test", opts.usage)
}

func TestFlagSet(t *testing.T) {
	fset := flag.NewFlagSet("app", flag.ContinueOnError)
	var opts flagOpts

	FlagSet(fset)(&opts)

	assert.Same(t, fset, opts.fset)
}

func TestCollectionFlagValueIsBoolFlag(t *testing.T) {
	boolValue := boolFlagValue(false)
	textValue := flagValue("text")

	testCases := []struct {
		name   string
		values []flag.Value
		want   bool
	}{
		{name: "empty", want: false},
		{name: "all_boolean", values: []flag.Value{&boolValue, &boolValue}, want: true},
		{name: "mixed", values: []flag.Value{&boolValue, &textValue}, want: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			value := collectionFlagValue{values: tc.values}

			assert.Equal(t, tc.want, value.IsBoolFlag())
		})
	}
}

func TestFromPointerTargets(t *testing.T) {
	testCases := []struct {
		name string
		args []string
		run  func(*testing.T)
	}{
		{
			name: "int",
			args: []string{"app", "-value", "42"},
			run: func(t *testing.T) {
				var value *int
				err := From("value")(t.Context(), &value)

				assert.NoError(t, err)
				assert.NotNil(t, value)
				assert.Equal(t, 42, *value)
			},
		},
		{
			name: "double_int",
			args: []string{"app", "-value", "42"},
			run: func(t *testing.T) {
				var value **int
				err := From("value")(t.Context(), &value)

				assert.NoError(t, err)
				assert.NotNil(t, value)
				assert.NotNil(t, *value)
				assert.Equal(t, 42, **value)
			},
		},
		{
			name: "boolean",
			args: []string{"app", "-value"},
			run: func(t *testing.T) {
				var value *bool
				err := From("value")(t.Context(), &value)

				assert.NoError(t, err)
				assert.NotNil(t, value)
				assert.True(t, *value)
			},
		},
		{
			name: "invalid_nil",
			args: []string{"app", "-value", "invalid"},
			run: func(t *testing.T) {
				fset := flag.NewFlagSet("app", flag.ContinueOnError)
				fset.SetOutput(io.Discard)

				var value *int
				err := From("value", FlagSet(fset))(t.Context(), &value)

				assert.Error(t, err)
				assert.Nil(t, value)
			},
		},
		{
			name: "invalid_existing",
			args: []string{"app", "-value", "invalid"},
			run: func(t *testing.T) {
				fset := flag.NewFlagSet("app", flag.ContinueOnError)
				fset.SetOutput(io.Discard)

				existing := 7
				value := &existing
				err := From("value", FlagSet(fset))(t.Context(), &value)

				assert.Error(t, err)
				assert.Same(t, &existing, value)
				assert.Equal(t, 7, *value)
			},
		},
		{
			name: "invalid_double_nil",
			args: []string{"app", "-value", "invalid"},
			run: func(t *testing.T) {
				fset := flag.NewFlagSet("app", flag.ContinueOnError)
				fset.SetOutput(io.Discard)

				var value **int
				err := From("value", FlagSet(fset))(t.Context(), &value)

				assert.Error(t, err)
				assert.Nil(t, value)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reset := swapOsArgs(tc.args...)
			defer reset()

			tc.run(t)
		})
	}
}

func TestFromPointerTargetDecoders(t *testing.T) {
	testCases := []struct {
		name string
		args []string
		run  func(*testing.T)
	}{
		{
			name: "flag_value",
			args: []string{"app", "-value", "f*u"},
			run: func(t *testing.T) {
				var value *flagValue
				err := From("value")(t.Context(), &value)

				assert.NoError(t, err)
				assert.Equal(t, flagValue("fu"), *value)
			},
		},
		{
			name: "sql_scanner",
			args: []string{"app", "-value", "loaded"},
			run: func(t *testing.T) {
				var value *scannerValue
				err := From("value")(t.Context(), &value)

				assert.NoError(t, err)
				assert.Equal(t, scannerValue("loaded"), *value)
			},
		},
		{
			name: "text_unmarshaler",
			args: []string{"app", "-value", "loaded"},
			run: func(t *testing.T) {
				var value *textValue
				err := From("value")(t.Context(), &value)

				assert.NoError(t, err)
				assert.Equal(t, textValue("loaded"), *value)
			},
		},
		{
			name: "binary_unmarshaler",
			args: []string{"app", "-value", "loaded"},
			run: func(t *testing.T) {
				var value *binaryValue
				err := From("value")(t.Context(), &value)

				assert.NoError(t, err)
				assert.Equal(t, binaryValue("loaded"), *value)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reset := swapOsArgs(tc.args...)
			defer reset()

			tc.run(t)
		})
	}

	t.Run("failing_text_unmarshaler", func(t *testing.T) {
		reset := swapOsArgs("app", "-value", "loaded")
		defer reset()

		existing := failingTextValue("original")
		value := &existing
		fset := flag.NewFlagSet("app", flag.ContinueOnError)
		fset.SetOutput(io.Discard)

		err := From("value", FlagSet(fset))(t.Context(), &value)

		assert.ErrorContains(t, err, "text value error")
		assert.Same(t, &existing, value)
		assert.Equal(t, failingTextValue("original"), *value)
	})

	t.Run("merge_text_unmarshaler", func(t *testing.T) {
		reset := swapOsArgs("app", "-value", "loaded")
		defer reset()

		existing := mergeTextValue("original")
		value := &existing

		err := From("value")(t.Context(), &value)

		assert.NoError(t, err)
		assert.Same(t, &existing, value)
		assert.Equal(t, mergeTextValue("original:loaded"), *value)
	})
}

func TestFromDecoderErrorsPreserveTargets(t *testing.T) {
	testCases := []struct {
		name      string
		newTarget func() (any, func(*testing.T))
	}{
		{
			name: "sql_scanner_nonzero",
			newTarget: func() (any, func(*testing.T)) {
				value := failingScannerValue("original")
				return &value, func(t *testing.T) {
					assert.Equal(t, failingScannerValue("original"), value)
				}
			},
		},
		{
			name: "text_unmarshaler_nonzero",
			newTarget: func() (any, func(*testing.T)) {
				value := failingTextValue("original")
				return &value, func(t *testing.T) {
					assert.Equal(t, failingTextValue("original"), value)
				}
			},
		},
		{
			name: "binary_unmarshaler_nonzero",
			newTarget: func() (any, func(*testing.T)) {
				value := failingBinaryValue("original")
				return &value, func(t *testing.T) {
					assert.Equal(t, failingBinaryValue("original"), value)
				}
			},
		},
		{
			name: "sql_scanner_nil_pointer",
			newTarget: func() (any, func(*testing.T)) {
				var value *failingScannerValue
				return &value, func(t *testing.T) {
					assert.Nil(t, value)
				}
			},
		},
		{
			name: "text_unmarshaler_nil_pointer",
			newTarget: func() (any, func(*testing.T)) {
				var value *failingTextValue
				return &value, func(t *testing.T) {
					assert.Nil(t, value)
				}
			},
		},
		{
			name: "binary_unmarshaler_nil_pointer",
			newTarget: func() (any, func(*testing.T)) {
				var value *failingBinaryValue
				return &value, func(t *testing.T) {
					assert.Nil(t, value)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reset := swapOsArgs("app", "-value", "loaded")
			defer reset()

			fset := flag.NewFlagSet("app", flag.ContinueOnError)
			fset.SetOutput(io.Discard)
			target, verify := tc.newTarget()

			err := From("value", FlagSet(fset))(t.Context(), target)

			assert.Error(t, err)
			verify(t)
		})
	}
}

func TestFromDirectFlagValuePreservesIdentity(t *testing.T) {
	reset := swapOsArgs("app", "-value", "f*u")
	defer reset()

	fset := flag.NewFlagSet("app", flag.ContinueOnError)
	value := flagValue("original")

	err := From("value", FlagSet(fset))(t.Context(), &value)

	assert.NoError(t, err)
	assert.Equal(t, flagValue("fu"), value)
	assert.Same(t, &value, fset.Lookup("value").Value)
}

func TestCustomBooleanFlagShorthand(t *testing.T) {
	t.Run("direct", func(t *testing.T) {
		t.Run("requires_explicit_value", func(t *testing.T) {
			reset := swapOsArgs("app", "-enabled")
			defer reset()

			var enabled *explicitBoolFlagValue
			fset := flag.NewFlagSet("app", flag.ContinueOnError)
			fset.SetOutput(io.Discard)

			err := From("enabled", FlagSet(fset))(t.Context(), &enabled)

			assert.ErrorContains(t, err, "flag needs an argument: -enabled")
			assert.Nil(t, enabled)
		})

		t.Run("explicit_value_is_consumed_and_applied", func(t *testing.T) {
			reset := swapOsArgs("app", "-enabled", "false")
			defer reset()

			var enabled *explicitBoolFlagValue

			err := From("enabled")(t.Context(), &enabled)

			if assert.NotNil(t, enabled) {
				assert.False(t, bool(*enabled))
			}
			assert.NoError(t, err)
		})

		t.Run("is_bool_flag_allows_shorthand", func(t *testing.T) {
			reset := swapOsArgs("app", "-enabled")
			defer reset()

			var enabled *boolFlagValue

			err := From("enabled")(t.Context(), &enabled)

			if assert.NotNil(t, enabled) {
				assert.True(t, bool(*enabled))
			}
			assert.NoError(t, err)
		})

		t.Run("existing_pointer_chain_uses_live_bool_receiver", func(t *testing.T) {
			reset := swapOsArgs("app", "-enabled")
			defer reset()

			terminal := conditionalBoolFlagValue{allowShorthand: true}
			inner := &terminal
			enabled := &inner
			fset := flag.NewFlagSet("app", flag.ContinueOnError)
			fset.SetOutput(io.Discard)

			err := From("enabled", FlagSet(fset))(t.Context(), &enabled)

			assert.NoError(t, err)
			if assert.NotNil(t, enabled) && assert.NotNil(t, *enabled) {
				assert.Same(t, inner, *enabled)
				assert.True(t, (**enabled).value)
			}
		})
	})

	t.Run("tagged", func(t *testing.T) {
		t.Run("requires_explicit_value", func(t *testing.T) {
			reset := swapOsArgs("app", "-enabled")
			defer reset()

			type config struct {
				Enabled **explicitBoolFlagValue `flag:"enabled"`
			}

			var cfg config
			fset := flag.NewFlagSet("app", flag.ContinueOnError)
			fset.SetOutput(io.Discard)

			err := FromArgs(FlagsSet(fset))(t.Context(), &cfg)

			assert.ErrorContains(t, err, "flag needs an argument: -enabled")
			assert.Nil(t, cfg.Enabled)
		})

		t.Run("explicit_value_is_consumed_and_applied", func(t *testing.T) {
			reset := swapOsArgs("app", "-enabled", "false")
			defer reset()

			type config struct {
				Enabled **explicitBoolFlagValue `flag:"enabled"`
			}

			var cfg config

			err := FromArgs()(t.Context(), &cfg)

			if assert.NotNil(t, cfg.Enabled) && assert.NotNil(t, *cfg.Enabled) {
				assert.False(t, bool(**cfg.Enabled))
			}
			assert.NoError(t, err)
		})

		t.Run("is_bool_flag_allows_shorthand", func(t *testing.T) {
			reset := swapOsArgs("app", "-enabled")
			defer reset()

			type config struct {
				Enabled **boolFlagValue `flag:"enabled"`
			}

			var cfg config

			err := FromArgs()(t.Context(), &cfg)

			if assert.NotNil(t, cfg.Enabled) && assert.NotNil(t, *cfg.Enabled) {
				assert.True(t, bool(**cfg.Enabled))
			}
			assert.NoError(t, err)
		})
	})
}

func TestFlagGetter(t *testing.T) {
	t.Run("direct_pointer_chain", func(t *testing.T) {
		reset := swapOsArgs("app", "-value", "loaded")
		defer reset()

		var value **getterFlagValue
		fset := flag.NewFlagSet("app", flag.ContinueOnError)
		fset.SetOutput(io.Discard)

		err := From("value", FlagSet(fset))(t.Context(), &value)

		assert.NoError(t, err)
		getter, ok := fset.Lookup("value").Value.(flag.Getter)
		if assert.True(t, ok) {
			assert.Equal(t, "loaded", getter.Get())
		}
		if assert.NotNil(t, value) && assert.NotNil(t, *value) {
			assert.Equal(t, getterFlagValue("loaded"), **value)
		}
	})

	t.Run("direct_pointer_chain_nil_before_set", func(t *testing.T) {
		reset := swapOsArgs("app", "-other", "ignored")
		defer reset()

		var value **getterFlagValue
		fset := flag.NewFlagSet("app", flag.ContinueOnError)
		fset.SetOutput(io.Discard)

		err := From("value", FlagSet(fset))(t.Context(), &value)

		assert.NoError(t, err)
		getter, ok := fset.Lookup("value").Value.(flag.Getter)
		if assert.True(t, ok) {
			assert.Equal(t, "", getter.Get())
		}
		assert.Nil(t, value)
		assert.NoError(t, fset.Set("value", "loaded"))
		if assert.NotNil(t, value) && assert.NotNil(t, *value) {
			assert.Equal(t, getterFlagValue("loaded"), **value)
		}
		if assert.True(t, ok) {
			assert.Equal(t, "loaded", getter.Get())
		}
	})

	t.Run("direct_pointer_chain_existing", func(t *testing.T) {
		reset := swapOsArgs("app", "-other", "ignored")
		defer reset()

		terminal := getterFlagValue("default")
		inner := &terminal
		outer := &inner
		fset := flag.NewFlagSet("app", flag.ContinueOnError)
		fset.SetOutput(io.Discard)

		err := From("value", FlagSet(fset))(t.Context(), &outer)

		assert.NoError(t, err)
		getter, ok := fset.Lookup("value").Value.(flag.Getter)
		if assert.True(t, ok) {
			assert.Equal(t, "default", getter.Get())
		}
		assert.NoError(t, fset.Set("value", "loaded"))
		assert.Same(t, inner, *outer)
		assert.Equal(t, getterFlagValue("loaded"), **outer)
		replacement := getterFlagValue("replacement")
		replacementInner := &replacement
		outer = &replacementInner
		if assert.True(t, ok) {
			assert.Equal(t, "replacement", getter.Get())
		}
	})

	t.Run("tagged_pointer_chain_nil_before_set", func(t *testing.T) {
		reset := swapOsArgs("app", "-other", "ignored")
		defer reset()

		type config struct {
			Value **getterFlagValue `flag:"value"`
		}

		var cfg config
		fset := flag.NewFlagSet("app", flag.ContinueOnError)
		fset.SetOutput(io.Discard)

		err := FromArgs(FlagsSet(fset))(t.Context(), &cfg)

		assert.NoError(t, err)
		getter, ok := fset.Lookup("value").Value.(flag.Getter)
		if assert.True(t, ok) {
			assert.Equal(t, "", getter.Get())
		}
		assert.Nil(t, cfg.Value)
		assert.NoError(t, fset.Set("value", "loaded"))
		if assert.NotNil(t, cfg.Value) && assert.NotNil(t, *cfg.Value) {
			assert.Equal(t, getterFlagValue("loaded"), **cfg.Value)
		}
		replacement := getterFlagValue("replacement")
		replacementInner := &replacement
		cfg.Value = &replacementInner
		if assert.True(t, ok) {
			assert.Equal(t, "replacement", getter.Get())
		}
	})

	t.Run("tagged_map_pointer_chains", func(t *testing.T) {
		reset := swapOsArgs("app", "-value", "loaded")
		defer reset()

		type item struct {
			Value **getterFlagValue `flag:"value"`
		}
		type config struct {
			Items map[string]item
		}

		cfg := config{Items: map[string]item{
			"first":  {},
			"second": {},
		}}
		fset := flag.NewFlagSet("app", flag.ContinueOnError)
		fset.SetOutput(io.Discard)

		err := FromArgs(FlagsSet(fset))(t.Context(), &cfg)

		assert.NoError(t, err)
		getter, ok := fset.Lookup("value").Value.(flag.Getter)
		if assert.True(t, ok) {
			assert.Equal(t, "loaded", getter.Get())
		}
		for _, item := range cfg.Items {
			if assert.NotNil(t, item.Value) && assert.NotNil(t, *item.Value) {
				assert.Equal(t, getterFlagValue("loaded"), **item.Value)
			}
		}
	})

	t.Run("non_getter_pointer_chain", func(t *testing.T) {
		reset := swapOsArgs("app", "-other", "ignored")
		defer reset()

		var value **flagValue
		fset := flag.NewFlagSet("app", flag.ContinueOnError)
		fset.SetOutput(io.Discard)

		err := From("value", FlagSet(fset))(t.Context(), &value)

		assert.NoError(t, err)
		_, ok := fset.Lookup("value").Value.(flag.Getter)
		assert.False(t, ok)
		assert.NoError(t, fset.Set("value", "loaded"))
		if assert.NotNil(t, value) && assert.NotNil(t, *value) {
			assert.Equal(t, flagValue("loaded"), **value)
		}
	})

	t.Run("tagged_non_getter_pointer_chain", func(t *testing.T) {
		reset := swapOsArgs("app", "-other", "ignored")
		defer reset()

		type config struct {
			Value **flagValue `flag:"value"`
		}

		var cfg config
		fset := flag.NewFlagSet("app", flag.ContinueOnError)
		fset.SetOutput(io.Discard)

		err := FromArgs(FlagsSet(fset))(t.Context(), &cfg)

		assert.NoError(t, err)
		_, ok := fset.Lookup("value").Value.(flag.Getter)
		assert.False(t, ok)
		assert.NoError(t, fset.Set("value", "loaded"))
		if assert.NotNil(t, cfg.Value) && assert.NotNil(t, *cfg.Value) {
			assert.Equal(t, flagValue("loaded"), **cfg.Value)
		}
	})

	t.Run("direct_baseline", func(t *testing.T) {
		reset := swapOsArgs("app", "-value", "loaded")
		defer reset()

		fset := flag.NewFlagSet("app", flag.ContinueOnError)
		value := getterFlagValue("default")

		err := From("value", FlagSet(fset))(t.Context(), &value)

		assert.NoError(t, err)
		getter, ok := fset.Lookup("value").Value.(flag.Getter)
		if assert.True(t, ok) {
			assert.Equal(t, "loaded", getter.Get())
		}
		assert.Same(t, &value, fset.Lookup("value").Value)
	})

	t.Run("deferred_map_values", func(t *testing.T) {
		reset := swapOsArgs("app", "-value", "loaded")
		defer reset()

		type item struct {
			Value getterFlagValue `flag:"value"`
		}
		type config struct {
			Items map[string]item
		}

		fset := flag.NewFlagSet("app", flag.ContinueOnError)
		fset.SetOutput(io.Discard)
		cfg := config{Items: map[string]item{
			"first":  {Value: "default"},
			"second": {Value: "default"},
		}}

		err := FromArgs(FlagsSet(fset))(t.Context(), &cfg)

		assert.NoError(t, err)
		assert.Equal(t, "default", fset.Lookup("value").DefValue)
		assert.Equal(t, getterFlagValue("loaded"), cfg.Items["first"].Value)
		assert.Equal(t, getterFlagValue("loaded"), cfg.Items["second"].Value)
		getter, ok := fset.Lookup("value").Value.(flag.Getter)
		if assert.True(t, ok) {
			assert.Equal(t, "loaded", getter.Get())
		}
	})

	t.Run("nested_collection_values", func(t *testing.T) {
		reset := swapOsArgs("app", "-value", "loaded")
		defer reset()

		type item struct {
			Value getterFlagValue `flag:"value"`
		}
		type group struct {
			Items map[string][]item
		}
		type config struct {
			Groups map[string]group
		}

		cfg := config{Groups: map[string]group{
			"group": {Items: map[string][]item{
				"items": {{Value: "first"}, {Value: "second"}},
			}},
		}}
		fset := flag.NewFlagSet("app", flag.ContinueOnError)
		fset.SetOutput(io.Discard)

		err := FromArgs(FlagsSet(fset))(t.Context(), &cfg)

		assert.NoError(t, err)
		assert.Equal(t, getterFlagValue("loaded"), cfg.Groups["group"].Items["items"][0].Value)
		assert.Equal(t, getterFlagValue("loaded"), cfg.Groups["group"].Items["items"][1].Value)
		getter, ok := fset.Lookup("value").Value.(flag.Getter)
		if assert.True(t, ok) {
			assert.Equal(t, "loaded", getter.Get())
		}
	})

	t.Run("nil_custom_pointer", func(t *testing.T) {
		reset := swapOsArgs("app", "-value", "loaded")
		defer reset()

		type config struct {
			Value *getterFlagValue `flag:"value"`
		}

		fset := flag.NewFlagSet("app", flag.ContinueOnError)
		fset.SetOutput(io.Discard)
		var cfg config

		err := FromArgs(FlagsSet(fset))(t.Context(), &cfg)

		assert.NoError(t, err)
		if assert.NotNil(t, cfg.Value) {
			assert.Equal(t, getterFlagValue("loaded"), *cfg.Value)
		}
		getter, ok := fset.Lookup("value").Value.(flag.Getter)
		if assert.True(t, ok) {
			assert.Equal(t, "loaded", getter.Get())
		}
	})

	t.Run("non_getter_deferred_value", func(t *testing.T) {
		value := flagValue("default")
		deferred := newDeferredFlagValue(&value, nil)

		_, ok := deferred.(flag.Getter)
		assert.False(t, ok)
	})

	t.Run("getter_collection", func(t *testing.T) {
		first := getterFlagValue("first")
		second := getterFlagValue("second")
		value := newCollectionFlagValue([]flag.Value{&first, &second})
		fset := flag.NewFlagSet("app", flag.ContinueOnError)
		fset.Var(value, "value", "")

		assert.Equal(t, "first", value.String())
		assert.Equal(t, "first", fset.Lookup("value").DefValue)
		getter, ok := fset.Lookup("value").Value.(flag.Getter)
		if assert.True(t, ok) {
			assert.Equal(t, "first", getter.Get())
		}
	})

	t.Run("non_getter_collection", func(t *testing.T) {
		first := flagValue("first")
		second := flagValue("second")
		value := newCollectionFlagValue([]flag.Value{&first, &second})

		_, ok := value.(flag.Getter)
		assert.False(t, ok)
	})

	t.Run("mixed_collection", func(t *testing.T) {
		getter := getterFlagValue("getter")
		nonGetter := flagValue("non-getter")
		definitions := addCollectionFlagDefinition(nil, flagDefinition{
			name:  "value",
			value: &getter,
		})
		definitions = addCollectionFlagDefinition(definitions, flagDefinition{
			name:  "value",
			value: &nonGetter,
		})

		assert.Len(t, definitions, 1)
		_, ok := definitions[0].value.(flag.Getter)
		assert.False(t, ok)
	})

	t.Run("getter_wrappers_preserve_bool_shorthand", func(t *testing.T) {
		first := getterBoolFlagValue(false)
		second := getterBoolFlagValue(false)
		value := newCollectionFlagValue([]flag.Value{
			newDeferredFlagValue(&first, nil),
			newDeferredFlagValue(&second, nil),
		})
		fset := flag.NewFlagSet("app", flag.ContinueOnError)
		fset.SetOutput(io.Discard)
		fset.Var(value, "enabled", "")

		err := fset.Parse([]string{"-enabled"})

		assert.NoError(t, err)
		assert.True(t, bool(first))
		assert.True(t, bool(second))
		boolFlag, ok := fset.Lookup("enabled").Value.(interface{ IsBoolFlag() bool })
		if assert.True(t, ok) {
			assert.True(t, boolFlag.IsBoolFlag())
		}
		getter, ok := fset.Lookup("enabled").Value.(flag.Getter)
		if assert.True(t, ok) {
			assert.Equal(t, true, getter.Get())
		}
	})

	t.Run("getter_scanner_preserves_bool_shorthand", func(t *testing.T) {
		reset := swapOsArgs("app", "-enabled")
		defer reset()

		var enabled **getterBoolFlagValue
		fset := flag.NewFlagSet("app", flag.ContinueOnError)
		fset.SetOutput(io.Discard)

		err := From("enabled", FlagSet(fset))(t.Context(), &enabled)

		assert.NoError(t, err)
		if assert.NotNil(t, enabled) && assert.NotNil(t, *enabled) {
			assert.True(t, bool(**enabled))
		}
		getter, ok := fset.Lookup("enabled").Value.(flag.Getter)
		if assert.True(t, ok) {
			assert.Equal(t, true, getter.Get())
		}
	})
}

func TestFromDecoderPrecedence(t *testing.T) {
	testCases := []struct {
		name      string
		newTarget func() (any, func(*testing.T))
	}{
		{
			name: "flag_value_before_other_decoders",
			newTarget: func() (any, func(*testing.T)) {
				var value *flagDecoderPriorityValue
				return &value, func(t *testing.T) {
					assert.Equal(t, flagDecoderPriorityValue("flag"), *value)
				}
			},
		},
		{
			name: "sql_scanner_before_text_and_binary",
			newTarget: func() (any, func(*testing.T)) {
				var value *decoderPriorityValue
				return &value, func(t *testing.T) {
					assert.Equal(t, decoderPriorityValue("scanner"), *value)
				}
			},
		},
		{
			name: "text_before_binary",
			newTarget: func() (any, func(*testing.T)) {
				var value *textDecoderPriorityValue
				return &value, func(t *testing.T) {
					assert.Equal(t, textDecoderPriorityValue("text"), *value)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reset := swapOsArgs("app", "-value", "loaded")
			defer reset()

			target, verify := tc.newTarget()
			err := From("value")(t.Context(), target)

			assert.NoError(t, err)
			verify(t)
		})
	}
}

func TestFromDecoderPointerChains(t *testing.T) {
	t.Run("error_preserves_existing_chain", func(t *testing.T) {
		reset := swapOsArgs("app", "-value", "loaded")
		defer reset()

		value := failingTextValue("original")
		inner := &value
		outer := &inner
		originalInner := inner
		originalOuter := outer
		fset := flag.NewFlagSet("app", flag.ContinueOnError)
		fset.SetOutput(io.Discard)

		err := From("value", FlagSet(fset))(t.Context(), &outer)

		assert.ErrorContains(t, err, "text value error")
		assert.Same(t, originalOuter, outer)
		assert.Same(t, originalInner, *outer)
		assert.Equal(t, failingTextValue("original"), **outer)
	})

	t.Run("error_does_not_allocate_partial_chain", func(t *testing.T) {
		reset := swapOsArgs("app", "-value", "loaded")
		defer reset()

		var inner *failingTextValue
		outer := &inner
		originalOuter := outer
		fset := flag.NewFlagSet("app", flag.ContinueOnError)
		fset.SetOutput(io.Discard)

		err := From("value", FlagSet(fset))(t.Context(), &outer)

		assert.ErrorContains(t, err, "text value error")
		assert.Same(t, originalOuter, outer)
		assert.Nil(t, *outer)
	})

	t.Run("success_preserves_existing_chain", func(t *testing.T) {
		reset := swapOsArgs("app", "-value", "loaded")
		defer reset()

		value := mergeTextValue("original")
		inner := &value
		outer := &inner
		originalInner := inner
		originalOuter := outer

		err := From("value")(t.Context(), &outer)

		assert.NoError(t, err)
		assert.Same(t, originalOuter, outer)
		assert.Same(t, originalInner, *outer)
		assert.Equal(t, mergeTextValue("original:loaded"), **outer)
	})
}

func TestUnsupportedTypesRemainNoOps(t *testing.T) {
	t.Run("from", func(t *testing.T) {
		reset := swapOsArgs("app", "-value", "loaded")
		defer reset()

		var value *unsupportedValue
		err := From("value")(t.Context(), &value)

		assert.NoError(t, err)
		assert.Nil(t, value)
	})

	t.Run("from_args", func(t *testing.T) {
		reset := swapOsArgs("app", "-value", "loaded")
		defer reset()

		type config struct {
			Value *unsupportedValue `flag:"value"`
		}
		var cfg config

		err := FromArgs()(t.Context(), &cfg)

		assert.NoError(t, err)
		assert.Nil(t, cfg.Value)
	})
}

func TestFromInvalidTargets(t *testing.T) {
	reset := swapOsArgs("app", "-value", "42")
	defer reset()

	testCases := []struct {
		name   string
		target any
	}{
		{name: "nil", target: nil},
		{name: "nil_int", target: (*int)(nil)},
		{name: "nil_double_int", target: (**int)(nil)},
		{name: "nil_flag_value", target: (*flagValue)(nil)},
		{name: "non_pointer", target: 42},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fset := flag.NewFlagSet("app", flag.ContinueOnError)
			fset.SetOutput(io.Discard)
			var err error

			assert.NotPanics(t, func() {
				err = From("value", FlagSet(fset))(t.Context(), tc.target)
			})
			assert.ErrorContains(t, err, "flag target")
		})
	}

	t.Run("direct_flag_value", func(t *testing.T) {
		var value string
		target := directFlagValue{value: &value}

		err := From("value")(t.Context(), target)

		assert.NoError(t, err)
		assert.Equal(t, "42", value)
	})
}

func TestFromPointerTargetPartialMutation(t *testing.T) {
	reset := swapOsArgs("app", "-value", "42", "-value", "invalid")
	defer reset()

	fset := flag.NewFlagSet("app", flag.ContinueOnError)
	fset.SetOutput(io.Discard)
	var value *int

	err := From("value", FlagSet(fset))(t.Context(), &value)

	assert.Error(t, err)
	assert.NotNil(t, value)
	assert.Equal(t, 42, *value)
}

func TestFlagValuePointerChainString(t *testing.T) {
	t.Run("direct_initialized", func(t *testing.T) {
		reset := swapOsArgs("app", "-other", "ignored")
		defer reset()

		terminal := customStringFlagValue("default")
		inner := &terminal
		outer := &inner
		fset := flag.NewFlagSet("app", flag.ContinueOnError)
		fset.SetOutput(io.Discard)

		err := From("value", FlagSet(fset))(t.Context(), &outer)

		assert.NoError(t, err)
		assert.Equal(t, "custom:default", fset.Lookup("value").DefValue)
		assert.NoError(t, fset.Set("value", "loaded"))
		assert.Equal(t, customStringFlagValue("loaded"), **outer)
		assert.Equal(t, "custom:loaded", fset.Lookup("value").Value.String())

		replacement := customStringFlagValue("replacement")
		replacementInner := &replacement
		outer = &replacementInner
		assert.Equal(t, "custom:replacement", fset.Lookup("value").Value.String())
	})

	t.Run("direct_nil", func(t *testing.T) {
		reset := swapOsArgs("app", "-other", "ignored")
		defer reset()

		var value **customStringFlagValue
		fset := flag.NewFlagSet("app", flag.ContinueOnError)
		fset.SetOutput(io.Discard)

		err := From("value", FlagSet(fset))(t.Context(), &value)

		assert.NoError(t, err)
		assert.Equal(t, "custom:", fset.Lookup("value").DefValue)
		assert.Nil(t, value)
		assert.NoError(t, fset.Set("value", "loaded"))
		if assert.NotNil(t, value) && assert.NotNil(t, *value) {
			assert.Equal(t, customStringFlagValue("loaded"), **value)
		}
		assert.Equal(t, "custom:loaded", fset.Lookup("value").Value.String())
	})

	t.Run("direct_non_flag_value", func(t *testing.T) {
		reset := swapOsArgs("app", "-other", "ignored")
		defer reset()

		terminal := pointerStringerValue("default")
		inner := &terminal
		outer := &inner
		fset := flag.NewFlagSet("app", flag.ContinueOnError)
		fset.SetOutput(io.Discard)

		err := From("value", FlagSet(fset))(t.Context(), &outer)

		assert.NoError(t, err)
		assert.Equal(t, "default", fset.Lookup("value").DefValue)
	})

	t.Run("tagged_nil", func(t *testing.T) {
		reset := swapOsArgs("app", "-other", "ignored")
		defer reset()

		type config struct {
			Value **customStringFlagValue `flag:"value"`
		}

		var cfg config
		fset := flag.NewFlagSet("app", flag.ContinueOnError)
		fset.SetOutput(io.Discard)

		err := FromArgs(FlagsSet(fset))(t.Context(), &cfg)

		assert.NoError(t, err)
		assert.Equal(t, "custom:", fset.Lookup("value").DefValue)
		assert.Nil(t, cfg.Value)
		assert.NoError(t, fset.Set("value", "loaded"))
		if assert.NotNil(t, cfg.Value) && assert.NotNil(t, *cfg.Value) {
			assert.Equal(t, customStringFlagValue("loaded"), **cfg.Value)
		}
		assert.Equal(t, "custom:loaded", fset.Lookup("value").Value.String())
	})

	t.Run("tagged_initialized", func(t *testing.T) {
		reset := swapOsArgs("app", "-other", "ignored")
		defer reset()

		type config struct {
			Value **customStringFlagValue `flag:"value"`
		}

		terminal := customStringFlagValue("default")
		inner := &terminal
		cfg := config{Value: &inner}
		fset := flag.NewFlagSet("app", flag.ContinueOnError)
		fset.SetOutput(io.Discard)

		err := FromArgs(FlagsSet(fset))(t.Context(), &cfg)

		assert.NoError(t, err)
		assert.Equal(t, "custom:default", fset.Lookup("value").DefValue)
		assert.NoError(t, fset.Set("value", "loaded"))
		assert.Equal(t, customStringFlagValue("loaded"), **cfg.Value)
		assert.Equal(t, "custom:loaded", fset.Lookup("value").Value.String())

		replacement := customStringFlagValue("replacement")
		replacementInner := &replacement
		cfg.Value = &replacementInner
		assert.Equal(t, "custom:replacement", fset.Lookup("value").Value.String())
	})
}

func TestFlagDefaultValues(t *testing.T) {
	reset := swapOsArgs("app", "-unrelated")
	defer reset()

	type namedInteger int
	type namedBoolean bool
	type namedString string

	t.Run("direct_targets", func(t *testing.T) {
		testCases := []struct {
			name      string
			newTarget func() any
			want      string
		}{
			{
				name: "integer",
				newTarget: func() any {
					value := 42
					return &value
				},
				want: "42",
			},
			{
				name: "boolean",
				newTarget: func() any {
					value := true
					return &value
				},
				want: "true",
			},
			{
				name: "duration",
				newTarget: func() any {
					value := time.Second
					return &value
				},
				want: "1s",
			},
			{
				name: "string",
				newTarget: func() any {
					value := "default"
					return &value
				},
				want: "default",
			},
			{
				name: "nil_pointer",
				newTarget: func() any {
					var value *int
					return &value
				},
				want: "0",
			},
			{
				name: "named_integer",
				newTarget: func() any {
					value := namedInteger(42)
					return &value
				},
				want: "42",
			},
			{
				name: "named_boolean",
				newTarget: func() any {
					value := namedBoolean(true)
					return &value
				},
				want: "true",
			},
			{
				name: "named_string",
				newTarget: func() any {
					value := namedString("default")
					return &value
				},
				want: "default",
			},
			{
				name: "nil_pointer_chain",
				newTarget: func() any {
					var value **int
					return &value
				},
				want: "0",
			},
			{
				name: "pointer_chain",
				newTarget: func() any {
					value := 42
					inner := &value
					outer := &inner
					return &outer
				},
				want: "42",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				fset := flag.NewFlagSet("app", flag.ContinueOnError)
				fset.SetOutput(io.Discard)

				err := From(tc.name, FlagSet(fset))(t.Context(), tc.newTarget())

				assert.NoError(t, err)
				assert.Equal(t, tc.want, fset.Lookup(tc.name).DefValue)
			})
		}
	})

	t.Run("from_args_fields", func(t *testing.T) {
		type config struct {
			Integer  int           `flag:"integer"`
			Boolean  bool          `flag:"boolean"`
			Duration time.Duration `flag:"duration"`
			String   string        `flag:"string"`
			Pointer  *int          `flag:"pointer"`
		}

		fset := flag.NewFlagSet("app", flag.ContinueOnError)
		fset.SetOutput(io.Discard)
		cfg := config{
			Integer:  42,
			Boolean:  true,
			Duration: time.Second,
			String:   "default",
		}

		err := FromArgs(FlagsSet(fset))(t.Context(), &cfg)

		assert.NoError(t, err)
		assert.Equal(t, "42", fset.Lookup("integer").DefValue)
		assert.Equal(t, "true", fset.Lookup("boolean").DefValue)
		assert.Equal(t, "1s", fset.Lookup("duration").DefValue)
		assert.Equal(t, "default", fset.Lookup("string").DefValue)
		assert.Equal(t, "0", fset.Lookup("pointer").DefValue)
		assert.Nil(t, cfg.Pointer)
	})

	t.Run("print_defaults", func(t *testing.T) {
		fset := flag.NewFlagSet("app", flag.ContinueOnError)
		var output bytes.Buffer
		fset.SetOutput(&output)
		value := 42

		err := From("value", FlagSet(fset))(t.Context(), &value)

		assert.NoError(t, err)
		assert.NotPanics(t, fset.PrintDefaults)
		assert.Contains(t, output.String(), "(default 42)")
		assert.NotContains(t, output.String(), "panic calling String method")
		assert.NotContains(t, output.String(), "<int Value>")
	})
}

func TestFrom(t *testing.T) {
	t.Run("empty_flag", func(t *testing.T) {
		reset := swapOsArgs("app")
		defer reset()

		target := 5
		err := From("test")(context.Background(), &target)
		assert.NoError(t, err)
		assert.Equal(t, target, 5)
	})

	t.Run("success", func(t *testing.T) {
		t.Run("string", func(t *testing.T) {
			reset := swapOsArgs("app", "-test", "42")
			defer reset()

			var target string
			err := From("test")(context.Background(), &target)
			assert.Equal(t, "42", target)
			assert.NoError(t, err)
		})
		t.Run("int", func(t *testing.T) {
			reset := swapOsArgs("app", "-test", "42")
			defer reset()

			var target int
			err := From("test")(context.Background(), &target)
			assert.Equal(t, 42, target)
			assert.NoError(t, err)
		})
		t.Run("uint", func(t *testing.T) {
			reset := swapOsArgs("app", "-test", "42")
			defer reset()

			var target uint
			err := From("test")(context.Background(), &target)
			assert.Equal(t, uint(42), target)
			assert.NoError(t, err)
		})
		t.Run("float", func(t *testing.T) {
			reset := swapOsArgs("app", "-test", "42")
			defer reset()

			var target float32
			err := From("test")(context.Background(), &target)
			assert.Equal(t, float32(42), target)
			assert.NoError(t, err)
		})
		t.Run("bool", func(t *testing.T) {
			reset := swapOsArgs("app", "-test")
			defer reset()

			var target bool
			err := From("test")(context.Background(), &target)
			assert.True(t, target)
			assert.NoError(t, err)
		})
		t.Run("complex", func(t *testing.T) {
			reset := swapOsArgs("app", "-test", "42i")
			defer reset()

			var target complex64
			err := From("test")(context.Background(), &target)
			assert.Equal(t, complex64(42i), target)
			assert.NoError(t, err)
		})
		t.Run("duration", func(t *testing.T) {
			reset := swapOsArgs("app", "-test", "42ns")
			defer reset()

			var target time.Duration
			err := From("test")(context.Background(), &target)
			assert.Equal(t, time.Duration(42), target)
			assert.NoError(t, err)
		})
		t.Run("field_value", func(t *testing.T) {
			reset := swapOsArgs("app", "-test", "f*u")
			defer reset()

			var target flagValue
			err := From("test")(context.Background(), &target)
			assert.Equal(t, flagValue("fu"), target)
			assert.NoError(t, err)
		})
		t.Run("text_unmarshaler", func(t *testing.T) {
			uid := "3f98346a-91e3-46c3-802e-fc771d3734fe"
			reset := swapOsArgs("app", "-test", uid)
			defer reset()

			var target string
			err := From("test")(context.Background(), &target)
			assert.Equal(t, uid, target)
			assert.NoError(t, err)
		})
	})

	t.Run("two_values", func(t *testing.T) {
		reset := swapOsArgs("app", "-host", "localhost", "-port", "8080")
		defer reset()

		var port, host string

		err := From("host")(context.Background(), &host)
		assert.Equal(t, "localhost", host)
		assert.NoError(t, err)

		err = From("port")(context.Background(), &port)
		assert.Equal(t, "8080", port)
		assert.NoError(t, err)
	})

	t.Run("registered_flag_arguments", func(t *testing.T) {
		t.Run("missing_value", func(t *testing.T) {
			reset := swapOsArgs("app", "-path")
			defer reset()

			var path string
			err := From("path")(t.Context(), &path)

			assert.EqualError(t, err, "flag needs an argument: -path")
			assert.Empty(t, path)
		})

		t.Run("negative_value", func(t *testing.T) {
			reset := swapOsArgs("app", "-port", "-1")
			defer reset()

			var port int
			err := From("port")(t.Context(), &port)

			assert.NoError(t, err)
			assert.Equal(t, -1, port)
		})

		t.Run("dash_prefixed_string", func(t *testing.T) {
			reset := swapOsArgs("app", "-path", "-literal")
			defer reset()

			var path string
			err := From("path")(t.Context(), &path)

			assert.NoError(t, err)
			assert.Equal(t, "-literal", path)
		})

		t.Run("terminator_value", func(t *testing.T) {
			reset := swapOsArgs("app", "-path", "--")
			defer reset()

			var path string
			err := From("path")(t.Context(), &path)

			assert.NoError(t, err)
			assert.Equal(t, "--", path)
		})

		t.Run("boolean_adjacency", func(t *testing.T) {
			reset := swapOsArgs("app", "-enabled", "-unknown", "value")
			defer reset()

			var enabled bool
			err := From("enabled")(t.Context(), &enabled)

			assert.NoError(t, err)
			assert.True(t, enabled)
		})

		t.Run("equal_syntax", func(t *testing.T) {
			reset := swapOsArgs("app", "--port=-1")
			defer reset()

			var port int
			err := From("port")(t.Context(), &port)

			assert.NoError(t, err)
			assert.Equal(t, -1, port)
		})

		t.Run("registered_help_flag", func(t *testing.T) {
			reset := swapOsArgs("app", "-h", "-literal")
			defer reset()

			var value string
			err := From("h")(t.Context(), &value)

			assert.NoError(t, err)
			assert.Equal(t, "-literal", value)
		})
	})

	t.Run("custom_flag_set_scope", func(t *testing.T) {
		reset := swapOsArgs("app", "-other", "changed", "-value", "loaded")
		defer reset()

		other := "default"
		fset := flag.NewFlagSet("app", flag.ContinueOnError)
		fset.SetOutput(io.Discard)
		fset.StringVar(&other, "other", other, "")

		var value string
		err := From("value", FlagSet(fset))(t.Context(), &value)

		assert.NoError(t, err)
		assert.Equal(t, "loaded", value)
		assert.Equal(t, "default", other)
	})

	t.Run("custom_flag_set_excluded_flag_owns_dash_prefixed_value", func(t *testing.T) {
		reset := swapOsArgs("app", "-other", "-value")
		defer reset()

		other := "default"
		fset := flag.NewFlagSet("app", flag.ContinueOnError)
		fset.SetOutput(io.Discard)
		fset.StringVar(&other, "other", other, "")

		var value string
		err := From("value", FlagSet(fset))(t.Context(), &value)

		assert.NoError(t, err)
		assert.Empty(t, value)
		assert.Equal(t, "default", other)
	})

	t.Run("custom_flag_set_registered_help_scope", func(t *testing.T) {
		testCases := []struct {
			name string
			args []string
			flag string
		}{
			{
				name: "short_help",
				args: []string{"app", "-h", "ignored", "-value", "loaded"},
				flag: "h",
			},
			{
				name: "long_help",
				args: []string{"app", "--help", "ignored", "-value", "loaded"},
				flag: "help",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				reset := swapOsArgs(tc.args...)
				defer reset()

				ignored := "default"
				fset := flag.NewFlagSet("app", flag.ContinueOnError)
				fset.SetOutput(io.Discard)
				fset.StringVar(&ignored, tc.flag, ignored, "")

				var value string
				err := From("value", FlagSet(fset))(t.Context(), &value)

				assert.NoError(t, err)
				assert.Equal(t, "default", ignored)
				assert.Equal(t, "loaded", value)
			})
		}
	})

	t.Run("bad_value", func(t *testing.T) {
		reset := swapOsArgs("app", "-test", "***")
		defer reset()

		newFlagSet := func() *flag.FlagSet {
			fset := flag.NewFlagSet("app", flag.ContinueOnError)
			fset.SetOutput(io.Discard)
			return fset
		}

		t.Run("int", func(t *testing.T) {
			var target int
			err := From("test", FlagSet(newFlagSet()))(t.Context(), &target)
			assert.Empty(t, target)
			assert.Contains(t, err.Error(), "invalid value \"***\" for flag -test")
		})
		t.Run("uint", func(t *testing.T) {
			var target uint
			err := From("test", FlagSet(newFlagSet()))(t.Context(), &target)
			assert.Empty(t, target)
			assert.Contains(t, err.Error(), "invalid value \"***\" for flag -test")
		})
		t.Run("float", func(t *testing.T) {
			var target float32
			err := From("test", FlagSet(newFlagSet()))(t.Context(), &target)
			assert.Empty(t, target)
			assert.Contains(t, err.Error(), "invalid value \"***\" for flag -test")
		})
		t.Run("complex", func(t *testing.T) {
			var target complex64
			err := From("test", FlagSet(newFlagSet()))(t.Context(), &target)
			assert.Empty(t, target)
			assert.Contains(t, err.Error(), "invalid value \"***\" for flag -test")
		})
		t.Run("overflow", func(t *testing.T) {
			reset := swapOsArgs("app", "-test", "1489")
			defer reset()

			var target int8
			err := From("test")(context.Background(), &target)
			assert.Equal(t, int8(0), target)
			assert.EqualError(t, err, `invalid value "1489" for flag -test: value 1489 overflows int8`)
		})
	})

	t.Run("custom_flag_set", func(t *testing.T) {
		reset := swapOsArgs("app", "-test", "***")
		defer reset()

		var output bytes.Buffer
		fset := flag.NewFlagSet("custom", flag.ContinueOnError)
		fset.SetOutput(&output)

		var target int
		err := From("test", FlagSet(fset))(t.Context(), &target)

		assert.Empty(t, target)
		assert.Equal(t, "custom", fset.Name())
		assert.EqualError(t, err, `invalid value "***" for flag -test: cannot convert value '***' to int: strconv.ParseInt: parsing "***": invalid syntax`)
		assert.Contains(t, output.String(), `invalid value "***" for flag -test`)
	})

	t.Run("custom_flag_set_usage", func(t *testing.T) {
		reset := swapOsArgs("app", "-h")
		defer reset()

		usageCalled := false
		fset := flag.NewFlagSet("custom", flag.ContinueOnError)
		fset.Usage = func() {
			usageCalled = true
		}

		var target string
		err := From("test", FlagSet(fset))(t.Context(), &target)

		assert.ErrorIs(t, err, flag.ErrHelp)
		assert.True(t, usageCalled)
	})

	t.Run("nil_flag_set", func(t *testing.T) {
		reset := swapOsArgs("app", "-test", "42")
		defer reset()

		var target int
		err := From("test", FlagSet(nil))(t.Context(), &target)

		assert.NoError(t, err)
		assert.Equal(t, 42, target)
	})
}

func TestFromArgsDecoderErrorsPreserveFields(t *testing.T) {
	testCases := []struct {
		name      string
		newTarget func() (any, func(*testing.T))
	}{
		{
			name: "sql_scanner_nonzero",
			newTarget: func() (any, func(*testing.T)) {
				type config struct {
					Before string              `flag:"before"`
					Value  failingScannerValue `flag:"value"`
				}
				cfg := config{Value: "original"}
				return &cfg, func(t *testing.T) {
					assert.Equal(t, "loaded", cfg.Before)
					assert.Equal(t, failingScannerValue("original"), cfg.Value)
				}
			},
		},
		{
			name: "text_unmarshaler_nonzero",
			newTarget: func() (any, func(*testing.T)) {
				type config struct {
					Before string           `flag:"before"`
					Value  failingTextValue `flag:"value"`
				}
				cfg := config{Value: "original"}
				return &cfg, func(t *testing.T) {
					assert.Equal(t, "loaded", cfg.Before)
					assert.Equal(t, failingTextValue("original"), cfg.Value)
				}
			},
		},
		{
			name: "binary_unmarshaler_nonzero",
			newTarget: func() (any, func(*testing.T)) {
				type config struct {
					Before string             `flag:"before"`
					Value  failingBinaryValue `flag:"value"`
				}
				cfg := config{Value: "original"}
				return &cfg, func(t *testing.T) {
					assert.Equal(t, "loaded", cfg.Before)
					assert.Equal(t, failingBinaryValue("original"), cfg.Value)
				}
			},
		},
		{
			name: "sql_scanner_nil_pointer",
			newTarget: func() (any, func(*testing.T)) {
				type config struct {
					Before string               `flag:"before"`
					Value  *failingScannerValue `flag:"value"`
				}
				var cfg config
				return &cfg, func(t *testing.T) {
					assert.Equal(t, "loaded", cfg.Before)
					assert.Nil(t, cfg.Value)
				}
			},
		},
		{
			name: "text_unmarshaler_nil_pointer",
			newTarget: func() (any, func(*testing.T)) {
				type config struct {
					Before string            `flag:"before"`
					Value  *failingTextValue `flag:"value"`
				}
				var cfg config
				return &cfg, func(t *testing.T) {
					assert.Equal(t, "loaded", cfg.Before)
					assert.Nil(t, cfg.Value)
				}
			},
		},
		{
			name: "binary_unmarshaler_nil_pointer",
			newTarget: func() (any, func(*testing.T)) {
				type config struct {
					Before string              `flag:"before"`
					Value  *failingBinaryValue `flag:"value"`
				}
				var cfg config
				return &cfg, func(t *testing.T) {
					assert.Equal(t, "loaded", cfg.Before)
					assert.Nil(t, cfg.Value)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reset := swapOsArgs("app", "-before", "loaded", "-value", "invalid")
			defer reset()

			fset := flag.NewFlagSet("app", flag.ContinueOnError)
			fset.SetOutput(io.Discard)
			target, verify := tc.newTarget()

			err := FromArgs(FlagsSet(fset))(t.Context(), target)

			assert.Error(t, err)
			verify(t)
		})
	}
}

func TestFromArgsDecoderSuccessPreservesPointerIdentity(t *testing.T) {
	reset := swapOsArgs("app", "-value", "loaded")
	defer reset()

	type config struct {
		Value *mergeTextValue `flag:"value"`
	}

	existing := mergeTextValue("original")
	cfg := config{Value: &existing}

	err := FromArgs()(t.Context(), &cfg)

	assert.NoError(t, err)
	assert.Same(t, &existing, cfg.Value)
	assert.Equal(t, mergeTextValue("original:loaded"), *cfg.Value)
}

func TestFromNumericOverflowPreservesTarget(t *testing.T) {
	t.Run("float32", func(t *testing.T) {
		reset := swapOsArgs("app", "-test", "1e39")
		defer reset()

		fset := flag.NewFlagSet("app", flag.ContinueOnError)
		fset.SetOutput(io.Discard)
		target := float32(12.5)
		err := From("test", FlagSet(fset))(t.Context(), &target)

		assert.ErrorContains(t, err, "cannot convert value '1e39' to float")
		assert.Equal(t, float32(12.5), target)
	})

	t.Run("complex64", func(t *testing.T) {
		reset := swapOsArgs("app", "-test", "1e39+1e39i")
		defer reset()

		fset := flag.NewFlagSet("app", flag.ContinueOnError)
		fset.SetOutput(io.Discard)
		target := complex64(1 + 2i)
		err := From("test", FlagSet(fset))(t.Context(), &target)

		assert.ErrorContains(t, err, "cannot convert value '1e39+1e39i' to complex")
		assert.Equal(t, complex64(1+2i), target)
	})
}

func TestFlagRegistration(t *testing.T) {
	reset := swapOsArgs("app", "-value", "loaded")
	defer reset()

	t.Run("from", func(t *testing.T) {
		testCases := []struct {
			name       string
			flagName   string
			registered bool
			want       string
		}{
			{name: "empty", want: "name is empty"},
			{name: "leading_dash", flagName: "-value", want: "name starts with '-'"},
			{name: "equals", flagName: "value=test", want: "name contains '='"},
			{name: "custom_set_collision", flagName: "value", registered: true, want: "already exists in FlagSet"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				fset := flag.NewFlagSet("app", flag.ContinueOnError)
				fset.SetOutput(io.Discard)
				if tc.registered {
					fset.String(tc.flagName, "original", "")
				}
				var target string
				var err error

				assert.NotPanics(t, func() {
					err = From(tc.flagName, FlagSet(fset))(t.Context(), &target)
				})

				assert.ErrorContains(t, err, tc.want)
				if tc.registered {
					assert.Equal(t, "original", fset.Lookup(tc.flagName).DefValue)
				} else {
					assert.Nil(t, fset.Lookup(tc.flagName))
				}
			})
		}
	})

	t.Run("from_args", func(t *testing.T) {
		type nested struct {
			Value string `flag:"value"`
		}

		testCases := []struct {
			name       string
			target     any
			registered bool
			want       string
			absentName string
		}{
			{
				name: "sibling_duplicates",
				target: &struct {
					First  string `flag:"value"`
					Second string `flag:"value"`
				}{},
				want:       "duplicate flag name \"value\"",
				absentName: "value",
			},
			{
				name: "nested_duplicates",
				target: &struct {
					First  nested
					Second nested
				}{},
				want:       "duplicate flag name \"value\"",
				absentName: "value",
			},
			{
				name: "custom_set_collision",
				target: &struct {
					First    string `flag:"first"`
					Conflict string `flag:"value"`
				}{},
				registered: true,
				want:       "flag name \"value\" already exists in FlagSet",
				absentName: "first",
			},
			{
				name: "leading_dash",
				target: &struct {
					Value string `flag:"-value"`
				}{},
				want:       "name starts with '-'",
				absentName: "-value",
			},
			{
				name: "equals",
				target: &struct {
					Value string `flag:"value=test"`
				}{},
				want:       "name contains '='",
				absentName: "value=test",
			},
			{
				name: "empty",
				target: &struct {
					Value string `flag:",usage=value"`
				}{},
				want: "name is empty",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				fset := flag.NewFlagSet("app", flag.ContinueOnError)
				fset.SetOutput(io.Discard)
				if tc.registered {
					fset.String("value", "original", "")
				}
				var err error

				assert.NotPanics(t, func() {
					err = FromArgs(FlagsSet(fset))(t.Context(), tc.target)
				})

				assert.ErrorContains(t, err, tc.want)
				if tc.registered {
					assert.Equal(t, "original", fset.Lookup("value").DefValue)
				}
				if tc.absentName != "" {
					assert.Nil(t, fset.Lookup(tc.absentName))
				}
			})
		}
	})
}

func TestFromArgs(t *testing.T) {
	t.Run("fail", func(t *testing.T) {
		reset := swapOsArgs("app", "-r")
		defer reset()

		t.Run("non_pointer", func(t *testing.T) {
			var target string
			err := FromArgs()(context.Background(), &target)
			assert.Empty(t, target)
			assert.EqualError(t, err, "cannot collect flags from struct tags: target must be a pointer to struct")
		})
		t.Run("pointer_to_non_struct", func(t *testing.T) {
			var target *string
			err := FromArgs()(context.Background(), &target)
			assert.Empty(t, target)
			assert.EqualError(t, err, "cannot collect flags from struct tags: target must be a pointer to struct")
		})
	})

	t.Run("overflow", func(t *testing.T) {
		reset := swapOsArgs("app", "-u", "admin", "-P", "root", "-a", "-p", "8080", "-v", "-l", "1489")
		defer reset()

		type remote struct {
			Hostname string `flag:"h"`
			Port     int16  `flag:"p"`
		}

		type config struct {
			Username        string `flag:"u"`
			Password        string `flag:"P"`
			Level           int8   `flag:"l"`
			Verbose         bool   `flag:"v"`
			UploadArtifacts bool   `flag:"a"`
			Remote          remote
		}

		cfg := config{
			Username: "nobody",
			Password: "nopass",
			Level:    2,
			Remote: remote{
				Hostname: "localhost",
				Port:     80,
			},
		}

		err := FromArgs()(context.Background(), &cfg)
		assert.EqualError(t, err, `invalid value "1489" for flag -l: value 1489 overflows int8`)

		// config expected to be corrupted
		expected := config{
			Username:        "admin",
			Password:        "root",
			Level:           2,
			Verbose:         true,
			UploadArtifacts: true,
			Remote: remote{
				Hostname: "localhost",
				Port:     8080,
			},
		}
		assert.Equal(t, expected, cfg)
	})

	t.Run("success", func(t *testing.T) {
		reset := swapOsArgs("app", "-u", "admin", "-P", "root", "-a", "-H", "localhost", "-p", "8080", "-v", "-l", "4")
		defer reset()

		type remote struct {
			Hostname string `flag:"H"`
			Port     *int16 `flag:"p"`
		}

		type config struct {
			Username        string `flag:"u"`
			Password        string `flag:"P"`
			Level           int8   `flag:"l"`
			Verbose         bool   `flag:"v"`
			UploadArtifacts bool   `flag:"a"`
			Remote          remote
		}

		cfg := config{
			Username: "nobody",
			Password: "nopass",
			Level:    2,
		}

		err := FromArgs()(context.Background(), &cfg)
		assert.NoError(t, err)

		expected := config{
			Username:        "admin",
			Password:        "root",
			Level:           4,
			Verbose:         true,
			UploadArtifacts: true,
			Remote: remote{
				Hostname: "localhost",
				Port:     ptr.T[int16](8080),
			},
		}
		assert.Equal(t, expected, cfg)
	})

	t.Run("registered_flag_arguments", func(t *testing.T) {
		t.Run("dash_prefixed_values_and_boolean_adjacency", func(t *testing.T) {
			reset := swapOsArgs("app", "-port", "-1", "-path", "-literal", "-enabled", "-unknown", "value")
			defer reset()

			type config struct {
				Port    int    `flag:"port"`
				Path    string `flag:"path"`
				Enabled bool   `flag:"enabled"`
			}

			var cfg config
			err := FromArgs()(t.Context(), &cfg)

			assert.NoError(t, err)
			assert.Equal(t, config{Port: -1, Path: "-literal", Enabled: true}, cfg)
		})

		t.Run("equal_syntax", func(t *testing.T) {
			reset := swapOsArgs("app", "--port=-1", "--path=-literal", "--enabled=false")
			defer reset()

			type config struct {
				Port    int    `flag:"port"`
				Path    string `flag:"path"`
				Enabled bool   `flag:"enabled"`
			}

			cfg := config{Enabled: true}
			err := FromArgs()(t.Context(), &cfg)

			assert.NoError(t, err)
			assert.Equal(t, config{Port: -1, Path: "-literal"}, cfg)
		})

		t.Run("registered_help_flags", func(t *testing.T) {
			reset := swapOsArgs("app", "-h", "short", "--help", "long")
			defer reset()

			type config struct {
				Short string `flag:"h"`
				Long  string `flag:"help"`
			}

			var cfg config
			err := FromArgs()(t.Context(), &cfg)

			assert.NoError(t, err)
			assert.Equal(t, config{Short: "short", Long: "long"}, cfg)
		})

		t.Run("implicit_help_flag", func(t *testing.T) {
			reset := swapOsArgs("app", "--help")
			defer reset()

			fset := flag.NewFlagSet("app", flag.ContinueOnError)
			fset.Usage = func() {}

			type config struct {
				Value string `flag:"value"`
			}

			var cfg config
			err := FromArgs(FlagsSet(fset))(t.Context(), &cfg)

			assert.ErrorIs(t, err, flag.ErrHelp)
		})

		t.Run("implicit_help_preserves_order", func(t *testing.T) {
			reset := swapOsArgs("app", "-before", "loaded", "--help", "-after", "ignored")
			defer reset()

			fset := flag.NewFlagSet("app", flag.ContinueOnError)
			fset.SetOutput(io.Discard)
			fset.Usage = func() {}

			type config struct {
				Before string `flag:"before"`
				After  string `flag:"after"`
			}

			var cfg config
			err := FromArgs(FlagsSet(fset))(t.Context(), &cfg)

			assert.ErrorIs(t, err, flag.ErrHelp)
			assert.Equal(t, config{Before: "loaded"}, cfg)
		})

		t.Run("undefined_help_value", func(t *testing.T) {
			reset := swapOsArgs("app", "--help=value")
			defer reset()

			fset := flag.NewFlagSet("app", flag.ContinueOnError)
			fset.SetOutput(io.Discard)

			type config struct {
				Value string `flag:"value"`
			}

			var cfg config
			err := FromArgs(FlagsSet(fset))(t.Context(), &cfg)

			assert.ErrorIs(t, err, flag.ErrHelp)
		})
	})

	t.Run("several_nested_structs", func(t *testing.T) {
		reset := swapOsArgs(
			"app",
			"-H", "localhost",
			"-p", "80",
			"-u", "tilt",
			"-P", "qwerty",
			"-l", "2",
		)
		defer reset()

		type remote struct {
			Hostname string `flag:"H"`
			Port     int16  `flag:"p"`
		}

		type user struct {
			Username string `flag:"u"`
			Password string `flag:"P"`
		}

		type logger struct {
			Level int8 `flag:"l"`
		}

		type config struct {
			Remote remote
			User   user
			Logger logger
		}

		cfg := config{}

		err := FromArgs()(context.Background(), &cfg)
		assert.NoError(t, err)

		expect := config{
			Remote: remote{
				Hostname: "localhost",
				Port:     80,
			},
			User: user{
				Username: "tilt",
				Password: "qwerty",
			},
			Logger: logger{
				Level: 2,
			},
		}
		assert.Equal(t, expect, cfg)
	})

	t.Run("nil_nested_struct", func(t *testing.T) {
		type remote struct {
			Port int `flag:"port"`
		}

		type config struct {
			Remote *remote
		}

		testCases := []struct {
			name       string
			args       []string
			wantRemote *remote
		}{
			{
				name: "absent_child_flag",
				args: []string{"app", "-unknown", "value"},
			},
			{
				name:       "nonzero_child_flag",
				args:       []string{"app", "-port", "8080"},
				wantRemote: &remote{Port: 8080},
			},
			{
				name:       "zero_child_flag",
				args:       []string{"app", "-port", "0"},
				wantRemote: &remote{},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				reset := swapOsArgs(tc.args...)
				defer reset()

				var cfg config
				err := FromArgs()(t.Context(), &cfg)

				assert.NoError(t, err)
				assert.Equal(t, tc.wantRemote, cfg.Remote)
			})
		}
	})

	t.Run("invalid_first_nested_value", func(t *testing.T) {
		reset := swapOsArgs("app", "-port", "invalid")
		defer reset()

		type remote struct {
			Port int `flag:"port"`
		}

		type config struct {
			Remote *remote
		}

		var cfg config
		err := FromArgs()(t.Context(), &cfg)

		assert.Error(t, err)
		assert.Nil(t, cfg.Remote)
	})

	t.Run("nested_custom_flag_value", func(t *testing.T) {
		reset := swapOsArgs("app", "-value", "f*u")
		defer reset()

		type remote struct {
			Value flagValue `flag:"value"`
		}

		type config struct {
			Remote *remote
		}

		var cfg config
		err := FromArgs()(t.Context(), &cfg)

		assert.NoError(t, err)
		assert.Equal(t, &remote{Value: "fu"}, cfg.Remote)
	})

	t.Run("nested_custom_boolean_flag_value", func(t *testing.T) {
		reset := swapOsArgs("app", "-enabled")
		defer reset()

		type remote struct {
			Enabled boolFlagValue `flag:"enabled"`
		}

		type config struct {
			Remote *remote
		}

		var cfg config
		err := FromArgs()(t.Context(), &cfg)

		assert.NoError(t, err)
		assert.Equal(t, &remote{Enabled: true}, cfg.Remote)
	})

	t.Run("nested_failing_custom_flag_value", func(t *testing.T) {
		reset := swapOsArgs("app", "-value", "invalid")
		defer reset()

		type remote struct {
			Value failingFlagValue `flag:"value"`
		}

		type config struct {
			Remote *remote
		}

		var cfg config
		err := FromArgs()(t.Context(), &cfg)

		assert.ErrorContains(t, err, errFlagValue.Error())
		assert.Nil(t, cfg.Remote)
	})

	t.Run("nil_custom_flag_value", func(t *testing.T) {
		t.Run("success", func(t *testing.T) {
			reset := swapOsArgs("app", "-value", "f*u")
			defer reset()

			type config struct {
				Value *flagValue `flag:"value"`
			}

			var cfg config
			err := FromArgs()(t.Context(), &cfg)

			assert.NoError(t, err)
			assert.NotNil(t, cfg.Value)
			assert.Equal(t, flagValue("fu"), *cfg.Value)
		})

		t.Run("absent", func(t *testing.T) {
			reset := swapOsArgs("app", "-unknown", "value")
			defer reset()

			type config struct {
				Value *flagValue `flag:"value"`
			}

			var cfg config
			err := FromArgs()(t.Context(), &cfg)

			assert.NoError(t, err)
			assert.Nil(t, cfg.Value)
		})

		t.Run("failure", func(t *testing.T) {
			reset := swapOsArgs("app", "-value", "invalid")
			defer reset()

			fset := flag.NewFlagSet("app", flag.ContinueOnError)
			fset.SetOutput(io.Discard)
			type config struct {
				Value *failingFlagValue `flag:"value"`
			}

			var cfg config
			err := FromArgs(FlagsSet(fset))(t.Context(), &cfg)

			assert.ErrorContains(t, err, errFlagValue.Error())
			assert.Nil(t, cfg.Value)
		})

		t.Run("boolean", func(t *testing.T) {
			reset := swapOsArgs("app", "-enabled")
			defer reset()

			type config struct {
				Enabled *boolFlagValue `flag:"enabled"`
			}

			var cfg config
			err := FromArgs()(t.Context(), &cfg)

			assert.NoError(t, err)
			assert.NotNil(t, cfg.Enabled)
			assert.True(t, bool(*cfg.Enabled))
		})

		t.Run("nested_success", func(t *testing.T) {
			reset := swapOsArgs("app", "-value", "f*u")
			defer reset()

			type remote struct {
				Value *flagValue `flag:"value"`
			}
			type config struct {
				Remote *remote
			}

			var cfg config
			err := FromArgs()(t.Context(), &cfg)

			assert.NoError(t, err)
			assert.Equal(t, &remote{Value: ptr.T(flagValue("fu"))}, cfg.Remote)
		})

		t.Run("nested_absent", func(t *testing.T) {
			reset := swapOsArgs("app", "-unknown", "value")
			defer reset()

			type remote struct {
				Value *flagValue `flag:"value"`
			}
			type config struct {
				Remote *remote
			}

			var cfg config
			err := FromArgs()(t.Context(), &cfg)

			assert.NoError(t, err)
			assert.Nil(t, cfg.Remote)
		})

		t.Run("nested_failure", func(t *testing.T) {
			reset := swapOsArgs("app", "-value", "invalid")
			defer reset()

			fset := flag.NewFlagSet("app", flag.ContinueOnError)
			fset.SetOutput(io.Discard)
			type remote struct {
				Value *failingFlagValue `flag:"value"`
			}
			type config struct {
				Remote *remote
			}

			var cfg config
			err := FromArgs(FlagsSet(fset))(t.Context(), &cfg)

			assert.ErrorContains(t, err, errFlagValue.Error())
			assert.Nil(t, cfg.Remote)
		})
	})

	t.Run("skipped_subtrees", func(t *testing.T) {
		testCases := []struct {
			name string
			run  func(*testing.T)
		}{
			{
				name: "unexported_tagged_scalar",
				run: func(t *testing.T) {
					reset := swapOsArgs("app", "-visible", "loaded", "-hidden", "blocked")
					defer reset()

					type config struct {
						Visible string `flag:"visible"`
						hidden  string `flag:"hidden"`
					}

					cfg := config{hidden: "original"}
					assert.NotPanics(t, func() {
						err := FromArgs()(t.Context(), &cfg)
						assert.NoError(t, err)
					})

					assert.Equal(t, "loaded", cfg.Visible)
					assert.Equal(t, "original", cfg.hidden)
				},
			},
			{
				name: "unexported_nested_struct_with_custom_tag",
				run: func(t *testing.T) {
					reset := swapOsArgs("app", "-visible", "loaded", "-hidden", "blocked")
					defer reset()

					type nested struct {
						Value string `config:"hidden"`
					}
					type config struct {
						Visible string `config:"visible"`
						hidden  nested
					}

					cfg := config{hidden: nested{Value: "original"}}
					assert.NotPanics(t, func() {
						err := FromArgs(Tag("config"))(t.Context(), &cfg)
						assert.NoError(t, err)
					})

					assert.Equal(t, "loaded", cfg.Visible)
					assert.Equal(t, "original", cfg.hidden.Value)
				},
			},
			{
				name: "value_struct_and_siblings",
				run: func(t *testing.T) {
					reset := swapOsArgs("app", "-visible", "loaded", "-allowed", "loaded", "-hidden", "blocked")
					defer reset()

					type allowed struct {
						Value string `flag:"allowed"`
					}
					type hidden struct {
						Value string `flag:"hidden"`
					}
					type config struct {
						Visible string `flag:"visible"`
						Allowed allowed
						Hidden  hidden `flag:"-"`
					}

					cfg := config{Hidden: hidden{Value: "original"}}
					assert.NotPanics(t, func() {
						err := FromArgs()(t.Context(), &cfg)
						assert.NoError(t, err)
					})

					assert.Equal(t, "loaded", cfg.Visible)
					assert.Equal(t, "loaded", cfg.Allowed.Value)
					assert.Equal(t, "original", cfg.Hidden.Value)
				},
			},
			{
				name: "pointer_struct",
				run: func(t *testing.T) {
					reset := swapOsArgs("app", "-visible", "loaded", "-hidden", "blocked")
					defer reset()

					type nested struct {
						Value string `flag:"hidden"`
					}
					type config struct {
						Visible string  `flag:"visible"`
						Hidden  *nested `flag:"-"`
					}

					hidden := &nested{Value: "original"}
					cfg := config{Hidden: hidden}
					assert.NotPanics(t, func() {
						err := FromArgs()(t.Context(), &cfg)
						assert.NoError(t, err)
					})

					assert.Equal(t, "loaded", cfg.Visible)
					assert.Same(t, hidden, cfg.Hidden)
					assert.Equal(t, "original", cfg.Hidden.Value)
				},
			},
			{
				name: "nil_pointer_struct",
				run: func(t *testing.T) {
					reset := swapOsArgs("app", "-visible", "loaded", "-hidden", "blocked")
					defer reset()

					type nested struct {
						Value string `flag:"hidden"`
					}
					type config struct {
						Visible string  `flag:"visible"`
						Hidden  *nested `flag:"-"`
					}

					var cfg config
					assert.NotPanics(t, func() {
						err := FromArgs()(t.Context(), &cfg)
						assert.NoError(t, err)
					})

					assert.Equal(t, "loaded", cfg.Visible)
					assert.Nil(t, cfg.Hidden)
				},
			},
			{
				name: "custom_tag",
				run: func(t *testing.T) {
					reset := swapOsArgs("app", "-visible", "loaded", "-hidden", "blocked")
					defer reset()

					type nested struct {
						Value string `config:"hidden"`
					}
					type config struct {
						Visible string `config:"visible"`
						Hidden  nested `config:"-"`
					}

					cfg := config{Hidden: nested{Value: "original"}}
					assert.NotPanics(t, func() {
						err := FromArgs(Tag("config"))(t.Context(), &cfg)
						assert.NoError(t, err)
					})

					assert.Equal(t, "loaded", cfg.Visible)
					assert.Equal(t, "original", cfg.Hidden.Value)
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, tc.run)
		}
	})

	t.Run("tagged_pointer_containers_skip_nested_tags", func(t *testing.T) {
		testCases := []struct {
			name string
			run  func(*testing.T)
		}{
			{
				name: "array",
				run: func(t *testing.T) {
					reset := swapOsArgs("app", "-items", "ignored", "-nested", "loaded")
					defer reset()

					type item struct {
						Value string `flag:"nested"`
					}
					type config struct {
						Items *[1]item `flag:"items"`
					}

					items := &[1]item{{Value: "original"}}
					cfg := config{Items: items}
					fset := flag.NewFlagSet("app", flag.ContinueOnError)
					fset.SetOutput(io.Discard)

					err := FromArgs(FlagsSet(fset))(t.Context(), &cfg)

					assert.NoError(t, err)
					assert.NotNil(t, fset.Lookup("items"))
					assert.Nil(t, fset.Lookup("nested"))
					assert.Equal(t, "original", cfg.Items[0].Value)
				},
			},
			{
				name: "slice",
				run: func(t *testing.T) {
					reset := swapOsArgs("app", "-items", "ignored", "-nested", "loaded")
					defer reset()

					type item struct {
						Value string `flag:"nested"`
					}
					type config struct {
						Items *[]item `flag:"items"`
					}

					items := []item{{Value: "original"}}
					cfg := config{Items: &items}
					fset := flag.NewFlagSet("app", flag.ContinueOnError)
					fset.SetOutput(io.Discard)

					err := FromArgs(FlagsSet(fset))(t.Context(), &cfg)

					assert.NoError(t, err)
					assert.NotNil(t, fset.Lookup("items"))
					assert.Nil(t, fset.Lookup("nested"))
					assert.Equal(t, "original", (*cfg.Items)[0].Value)
				},
			},
			{
				name: "map",
				run: func(t *testing.T) {
					reset := swapOsArgs("app", "-items", "ignored", "-nested", "loaded")
					defer reset()

					type item struct {
						Value string `flag:"nested"`
					}
					type config struct {
						Items *map[string]item `flag:"items"`
					}

					items := map[string]item{"item": {Value: "original"}}
					cfg := config{Items: &items}
					fset := flag.NewFlagSet("app", flag.ContinueOnError)
					fset.SetOutput(io.Discard)

					err := FromArgs(FlagsSet(fset))(t.Context(), &cfg)

					assert.NoError(t, err)
					assert.NotNil(t, fset.Lookup("items"))
					assert.Nil(t, fset.Lookup("nested"))
					assert.Equal(t, "original", (*cfg.Items)["item"].Value)
				},
			},
			{
				name: "multiple_pointer_levels",
				run: func(t *testing.T) {
					reset := swapOsArgs("app", "-items", "ignored", "-nested", "loaded")
					defer reset()

					type item struct {
						Value string `flag:"nested"`
					}
					type config struct {
						Items **[]item `flag:"items"`
					}

					items := []item{{Value: "original"}}
					itemsPtr := &items
					cfg := config{Items: &itemsPtr}
					fset := flag.NewFlagSet("app", flag.ContinueOnError)
					fset.SetOutput(io.Discard)

					err := FromArgs(FlagsSet(fset))(t.Context(), &cfg)

					assert.NoError(t, err)
					assert.NotNil(t, fset.Lookup("items"))
					assert.Nil(t, fset.Lookup("nested"))
					assert.Equal(t, "original", (**cfg.Items)[0].Value)
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, tc.run)
		}
	})

	t.Run("untagged_pointer_container_traverses_nested_tags", func(t *testing.T) {
		reset := swapOsArgs("app", "-nested", "loaded")
		defer reset()

		type item struct {
			Value string `flag:"nested"`
		}
		type config struct {
			Items *[]item
		}

		items := []item{{Value: "original"}}
		cfg := config{Items: &items}
		fset := flag.NewFlagSet("app", flag.ContinueOnError)
		fset.SetOutput(io.Discard)

		err := FromArgs(FlagsSet(fset))(t.Context(), &cfg)

		assert.NoError(t, err)
		assert.NotNil(t, fset.Lookup("nested"))
		assert.Equal(t, "loaded", (*cfg.Items)[0].Value)
	})

	t.Run("untagged_collections_share_flags", func(t *testing.T) {
		t.Run("array", func(t *testing.T) {
			reset := swapOsArgs("app", "-value", "0")
			defer reset()

			type item struct {
				Value int `flag:"value,usage=all values"`
			}
			type config struct {
				Items [2]item
			}

			cfg := config{Items: [2]item{{Value: 7}, {Value: 9}}}
			fset := flag.NewFlagSet("app", flag.ContinueOnError)
			fset.SetOutput(io.Discard)
			err := FromArgs(FlagsSet(fset))(t.Context(), &cfg)

			assert.NoError(t, err)
			assert.Equal(t, "7", fset.Lookup("value").DefValue)
			assert.Equal(t, "all values", fset.Lookup("value").Usage)
			assert.Equal(t, config{}, cfg)
		})

		t.Run("pointer_wrapped_slice_boolean", func(t *testing.T) {
			reset := swapOsArgs("app", "-enabled")
			defer reset()

			type item struct {
				Enabled bool `flag:"enabled"`
			}
			type config struct {
				Items *[]item
			}

			items := []item{{}, {}}
			cfg := config{Items: &items}
			err := FromArgs()(t.Context(), &cfg)

			assert.NoError(t, err)
			assert.Equal(t, []item{{Enabled: true}, {Enabled: true}}, *cfg.Items)
		})

		t.Run("map_values_use_custom_flag_value", func(t *testing.T) {
			reset := swapOsArgs("app", "-value", "*loaded*")
			defer reset()

			type item struct {
				Value flagValue `flag:"value"`
			}
			type config struct {
				Items map[string]item
			}

			cfg := config{Items: map[string]item{
				"first":  {Value: "first"},
				"second": {Value: "second"},
			}}
			err := FromArgs()(t.Context(), &cfg)

			assert.NoError(t, err)
			assert.Equal(t, flagValue("loaded"), cfg.Items["first"].Value)
			assert.Equal(t, flagValue("loaded"), cfg.Items["second"].Value)
		})

		t.Run("custom_boolean_value_accepts_shorthand", func(t *testing.T) {
			reset := swapOsArgs("app", "-enabled")
			defer reset()

			type item struct {
				Enabled boolFlagValue `flag:"enabled"`
			}
			type config struct {
				Items []item
			}

			cfg := config{Items: []item{{}, {}}}
			err := FromArgs()(t.Context(), &cfg)

			assert.NoError(t, err)
			assert.Equal(t, config{Items: []item{{Enabled: true}, {Enabled: true}}}, cfg)
		})

		t.Run("setter_error_partially_applies_collection_values", func(t *testing.T) {
			reset := swapOsArgs("app", "-value", "1")
			defer reset()

			type item struct {
				Value collectionErrorValue `flag:"value"`
			}
			type config struct {
				Items []item
			}

			cfg := config{
				Items: []item{
					{Value: collectionErrorValue{value: "original"}},
					{Value: collectionErrorValue{value: "original", fail: true}},
				},
			}
			fset := flag.NewFlagSet("app", flag.ContinueOnError)
			fset.SetOutput(io.Discard)

			err := FromArgs(FlagsSet(fset))(t.Context(), &cfg)

			assert.ErrorContains(t, err, errFlagValue.Error())
			assert.Equal(t, "1", cfg.Items[0].Value.value)
			assert.Equal(t, "original", cfg.Items[1].Value.value)
		})

		t.Run("nested_collections_commit_all_values", func(t *testing.T) {
			reset := swapOsArgs("app", "-value", "0")
			defer reset()

			type item struct {
				Value int `flag:"value"`
			}
			type group struct {
				Items map[string][]*item
			}
			type config struct {
				Groups map[string]group
			}

			cfg := config{Groups: map[string]group{
				"first":  {Items: map[string][]*item{"items": {{Value: 1}, {Value: 2}}}},
				"second": {Items: map[string][]*item{"items": {{Value: 3}}}},
			}}
			fset := flag.NewFlagSet("app", flag.ContinueOnError)
			fset.SetOutput(io.Discard)

			err := FromArgs(FlagsSet(fset))(t.Context(), &cfg)

			assert.NoError(t, err)
			assert.Equal(t, 0, cfg.Groups["first"].Items["items"][0].Value)
			assert.Equal(t, 0, cfg.Groups["first"].Items["items"][1].Value)
			assert.Equal(t, 0, cfg.Groups["second"].Items["items"][0].Value)
		})

		t.Run("empty_slice_and_map_register_no_flag", func(t *testing.T) {
			reset := swapOsArgs("app", "-value", "loaded")
			defer reset()

			type item struct {
				Value string `flag:"value"`
			}
			type config struct {
				Slice []item
				Map   map[string]item
			}

			var cfg config
			fset := flag.NewFlagSet("app", flag.ContinueOnError)
			fset.SetOutput(io.Discard)
			err := FromArgs(FlagsSet(fset))(t.Context(), &cfg)

			assert.NoError(t, err)
			assert.Nil(t, fset.Lookup("value"))
			assert.Equal(t, config{}, cfg)
		})

		t.Run("ordinary_field_collision_returns_duplicate_error", func(t *testing.T) {
			reset := swapOsArgs("app", "-value", "loaded")
			defer reset()

			type item struct {
				Value string `flag:"value"`
			}
			type config struct {
				Value string `flag:"value"`
				Items []item
			}

			cfg := config{Items: []item{{}, {}}}
			fset := flag.NewFlagSet("app", flag.ContinueOnError)
			fset.SetOutput(io.Discard)

			err := FromArgs(FlagsSet(fset))(t.Context(), &cfg)

			assert.EqualError(t, err, "cannot register flags from struct tags: duplicate flag name \"value\"")
			assert.Nil(t, fset.Lookup("value"))
		})

		t.Run("distinct_fields_in_collection_return_duplicate_error", func(t *testing.T) {
			reset := swapOsArgs("app", "-value", "loaded")
			defer reset()

			type item struct {
				First  string `flag:"value"`
				Second string `flag:"value"`
			}
			type config struct {
				Items []item
			}

			cfg := config{Items: []item{{}, {}}}
			fset := flag.NewFlagSet("app", flag.ContinueOnError)
			fset.SetOutput(io.Discard)

			err := FromArgs(FlagsSet(fset))(t.Context(), &cfg)

			assert.EqualError(t, err, "cannot register flags from struct tags: duplicate flag name \"value\"")
			assert.Nil(t, fset.Lookup("value"))
		})

		t.Run("distinct_collections_return_duplicate_error", func(t *testing.T) {
			reset := swapOsArgs("app", "-value", "loaded")
			defer reset()

			type item struct {
				Value string `flag:"value"`
			}
			type config struct {
				First  []item
				Second []item
			}

			cfg := config{First: []item{{}}, Second: []item{{}}}
			fset := flag.NewFlagSet("app", flag.ContinueOnError)
			fset.SetOutput(io.Discard)

			err := FromArgs(FlagsSet(fset))(t.Context(), &cfg)

			assert.EqualError(t, err, "cannot register flags from struct tags: duplicate flag name \"value\"")
			assert.Nil(t, fset.Lookup("value"))
		})
	})

	t.Run("untagged_map_values_commit_deferred_flags", func(t *testing.T) {
		t.Run("struct_value_with_zero", func(t *testing.T) {
			reset := swapOsArgs("app", "-value", "0")
			defer reset()

			type item struct {
				Value int `flag:"value"`
			}
			type config struct {
				Items map[string]item
			}

			cfg := config{Items: map[string]item{"item": {Value: 7}}}
			fset := flag.NewFlagSet("app", flag.ContinueOnError)
			fset.SetOutput(io.Discard)

			err := FromArgs(FlagsSet(fset))(t.Context(), &cfg)

			assert.NoError(t, err)
			assert.Equal(t, 0, cfg.Items["item"].Value)
		})

		t.Run("non_nil_pointer_value", func(t *testing.T) {
			reset := swapOsArgs("app", "-value", "0")
			defer reset()

			type item struct {
				Value int `flag:"value"`
			}
			type config struct {
				Items map[string]*item
			}

			cfg := config{Items: map[string]*item{"item": {Value: 7}}}
			fset := flag.NewFlagSet("app", flag.ContinueOnError)
			fset.SetOutput(io.Discard)

			err := FromArgs(FlagsSet(fset))(t.Context(), &cfg)

			assert.NoError(t, err)
			if assert.NotNil(t, cfg.Items["item"]) {
				assert.Equal(t, 0, cfg.Items["item"].Value)
			}
		})

		t.Run("nil_pointer_value_with_zero", func(t *testing.T) {
			reset := swapOsArgs("app", "-value", "0")
			defer reset()

			type item struct {
				Value int `flag:"value"`
			}
			type config struct {
				Items map[string]*item
			}

			cfg := config{Items: map[string]*item{"item": nil}}
			fset := flag.NewFlagSet("app", flag.ContinueOnError)
			fset.SetOutput(io.Discard)

			err := FromArgs(FlagsSet(fset))(t.Context(), &cfg)

			assert.NoError(t, err)
			if assert.NotNil(t, cfg.Items["item"]) {
				assert.Equal(t, 0, cfg.Items["item"].Value)
			}
		})

		t.Run("absent_flag_does_not_allocate_nil_pointer", func(t *testing.T) {
			reset := swapOsArgs("app", "-unknown", "value")
			defer reset()

			type item struct {
				Value int `flag:"value"`
			}
			type config struct {
				Items map[string]*item
			}

			cfg := config{Items: map[string]*item{"item": nil}}
			fset := flag.NewFlagSet("app", flag.ContinueOnError)
			fset.SetOutput(io.Discard)

			err := FromArgs(FlagsSet(fset))(t.Context(), &cfg)

			assert.NoError(t, err)
			assert.Nil(t, cfg.Items["item"])
		})

		t.Run("multiple_pointer_levels", func(t *testing.T) {
			reset := swapOsArgs("app", "-value", "0")
			defer reset()

			type item struct {
				Value int `flag:"value"`
			}
			type config struct {
				Items map[string]***item
			}

			cfg := config{Items: map[string]***item{"item": nil}}
			fset := flag.NewFlagSet("app", flag.ContinueOnError)
			fset.SetOutput(io.Discard)

			err := FromArgs(FlagsSet(fset))(t.Context(), &cfg)

			assert.NoError(t, err)
			if assert.NotNil(t, cfg.Items["item"]) {
				if assert.NotNil(t, *cfg.Items["item"]) {
					if assert.NotNil(t, **cfg.Items["item"]) {
						assert.Equal(t, 0, (***cfg.Items["item"]).Value)
					}
				}
			}
		})

		t.Run("parse_error_preserves_struct_value", func(t *testing.T) {
			reset := swapOsArgs("app", "-value", "invalid")
			defer reset()

			type item struct {
				Value int `flag:"value"`
			}
			type config struct {
				Items map[string]item
			}

			cfg := config{Items: map[string]item{"item": {Value: 7}}}
			fset := flag.NewFlagSet("app", flag.ContinueOnError)
			fset.SetOutput(io.Discard)

			err := FromArgs(FlagsSet(fset))(t.Context(), &cfg)

			assert.Error(t, err)
			assert.Equal(t, 7, cfg.Items["item"].Value)
		})

		t.Run("decoder_error_does_not_allocate_nil_pointer", func(t *testing.T) {
			reset := swapOsArgs("app", "-value", "invalid")
			defer reset()

			type item struct {
				Value failingTextValue `flag:"value"`
			}
			type config struct {
				Items map[string]*item
			}

			cfg := config{Items: map[string]*item{"item": nil}}
			fset := flag.NewFlagSet("app", flag.ContinueOnError)
			fset.SetOutput(io.Discard)

			err := FromArgs(FlagsSet(fset))(t.Context(), &cfg)

			assert.ErrorContains(t, err, "text value error")
			assert.Nil(t, cfg.Items["item"])
		})
	})

	t.Run("tagged_direct_and_named_containers_skip_nested_tags", func(t *testing.T) {
		reset := swapOsArgs(
			"app",
			"-array", "ignored",
			"-slice", "ignored",
			"-map", "ignored",
			"-named-array", "ignored",
			"-named-slice", "ignored",
			"-named-map", "ignored",
			"-pointer-named", "ignored",
			"-nil-pointer-named", "ignored",
			"-nested", "loaded",
		)
		defer reset()

		type item struct {
			Value string `flag:"nested"`
		}
		type namedArray [1]item
		type namedSlice []item
		type namedMap map[string]item
		type config struct {
			Array           [1]item         `flag:"array"`
			Slice           []item          `flag:"slice"`
			Map             map[string]item `flag:"map"`
			NamedArray      namedArray      `flag:"named-array"`
			NamedSlice      namedSlice      `flag:"named-slice"`
			NamedMap        namedMap        `flag:"named-map"`
			PointerNamed    *namedSlice     `flag:"pointer-named"`
			NilPointerNamed *namedSlice     `flag:"nil-pointer-named"`
		}

		pointerNamed := namedSlice{{Value: "original"}}
		cfg := config{
			Array:        [1]item{{Value: "original"}},
			Slice:        []item{{Value: "original"}},
			Map:          map[string]item{"item": {Value: "original"}},
			NamedArray:   namedArray{{Value: "original"}},
			NamedSlice:   namedSlice{{Value: "original"}},
			NamedMap:     namedMap{"item": {Value: "original"}},
			PointerNamed: &pointerNamed,
		}
		fset := flag.NewFlagSet("app", flag.ContinueOnError)
		fset.SetOutput(io.Discard)

		err := FromArgs(FlagsSet(fset))(t.Context(), &cfg)

		assert.NoError(t, err)
		assert.Nil(t, fset.Lookup("nested"))
		assert.Equal(t, "original", cfg.Array[0].Value)
		assert.Equal(t, "original", cfg.Slice[0].Value)
		assert.Equal(t, "original", cfg.Map["item"].Value)
		assert.Equal(t, "original", cfg.NamedArray[0].Value)
		assert.Equal(t, "original", cfg.NamedSlice[0].Value)
		assert.Equal(t, "original", cfg.NamedMap["item"].Value)
		assert.Equal(t, "original", (*cfg.PointerNamed)[0].Value)
		assert.Nil(t, cfg.NilPointerNamed)
	})

	t.Run("direct_custom_flag_value_uses_original_value", func(t *testing.T) {
		reset := swapOsArgs("app", "-value", "f*u")
		defer reset()

		fset := flag.NewFlagSet("app", flag.ContinueOnError)
		fset.SetOutput(io.Discard)

		type config struct {
			Value flagValue `flag:"value"`
		}

		var cfg config
		err := FromArgs(FlagsSet(fset))(t.Context(), &cfg)

		assert.NoError(t, err)
		assert.Equal(t, flagValue("fu"), cfg.Value)
		assert.Same(t, &cfg.Value, fset.Lookup("value").Value)
	})

	t.Run("partial_nested_value", func(t *testing.T) {
		reset := swapOsArgs("app", "-z", "1", "-a", "invalid")
		defer reset()

		type remote struct {
			Valid   int `flag:"z"`
			Invalid int `flag:"a"`
		}

		type config struct {
			Remote *remote
		}

		var cfg config
		err := FromArgs()(t.Context(), &cfg)

		assert.Error(t, err)
		assert.Equal(t, &remote{Valid: 1}, cfg.Remote)
	})

	t.Run("multiple_nested_pointer_levels", func(t *testing.T) {
		type remote struct {
			Port int `flag:"port"`
		}

		type config struct {
			Remote **remote
		}

		testCases := []struct {
			name string
			args []string
			port int
		}{
			{
				name: "nonzero_child_flag",
				args: []string{"app", "-port", "8080"},
				port: 8080,
			},
			{
				name: "zero_child_flag",
				args: []string{"app", "-port", "0"},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				reset := swapOsArgs(tc.args...)
				defer reset()

				var cfg config
				err := FromArgs()(t.Context(), &cfg)

				assert.NoError(t, err)
				assert.NotNil(t, cfg.Remote)
				assert.NotNil(t, *cfg.Remote)
				assert.Equal(t, tc.port, (*cfg.Remote).Port)
			})
		}
	})

	t.Run("pointer_terminal_values", func(t *testing.T) {
		type config struct {
			Enabled **bool `flag:"enabled"`
			Port    **int  `flag:"port"`
		}

		testCases := []struct {
			name string
			args []string
			run  func(*testing.T, config, error)
		}{
			{
				name: "boolean",
				args: []string{"app", "-enabled"},
				run: func(t *testing.T, cfg config, err error) {
					assert.NoError(t, err)
					assert.NotNil(t, cfg.Enabled)
					assert.NotNil(t, *cfg.Enabled)
					assert.True(t, **cfg.Enabled)
				},
			},
			{
				name: "valid_int",
				args: []string{"app", "-port", "42"},
				run: func(t *testing.T, cfg config, err error) {
					assert.NoError(t, err)
					assert.NotNil(t, cfg.Port)
					assert.NotNil(t, *cfg.Port)
					assert.Equal(t, 42, **cfg.Port)
				},
			},
			{
				name: "invalid_int",
				args: []string{"app", "-port", "invalid"},
				run: func(t *testing.T, cfg config, err error) {
					assert.Error(t, err)
					assert.Nil(t, cfg.Port)
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				reset := swapOsArgs(tc.args...)
				defer reset()

				var cfg config
				err := FromArgs()(t.Context(), &cfg)

				tc.run(t, cfg, err)
			})
		}
	})

	t.Run("invalid_nested_pointer_terminal", func(t *testing.T) {
		reset := swapOsArgs("app", "-port", "invalid")
		defer reset()

		type remote struct {
			Port **int `flag:"port"`
		}

		type config struct {
			Remote *remote
		}

		var cfg config
		err := FromArgs()(t.Context(), &cfg)

		assert.Error(t, err)
		assert.Nil(t, cfg.Remote)
	})
}

func newExtractionFlagSet() *flag.FlagSet {
	fset := flag.NewFlagSet("app", flag.ContinueOnError)
	fset.String("host", "", "")
	fset.String("port", "", "")
	fset.String("path", "", "")
	fset.Bool("v", false, "")
	fset.Bool("t", false, "")
	fset.Bool("enabled", false, "")
	fset.String("h", "", "")
	fset.String("help", "", "")
	return fset
}

func TestExtractRegistered(t *testing.T) {
	testCases := []struct {
		name     string
		args     []string
		expected []string
	}{
		{"nil", nil, nil},
		{"empty", []string{}, nil},
		{"registered_flags_keep_order", []string{"-port", "1", "-v", "-host", "localhost", "--port=2"}, []string{"-port", "1", "-v", "-host", "localhost", "--port=2"}},
		{"dash_prefixed_values", []string{"-port", "-1", "-path", "-literal"}, []string{"-port", "-1", "-path", "-literal"}},
		{"boolean_does_not_consume_next_flag", []string{"-v", "-unknown", "value", "-port", "1"}, []string{"-v", "-port", "1"}},
		{"equal_syntax", []string{"-port=-1", "--path=-literal", "--enabled=false"}, []string{"-port=-1", "--path=-literal", "--enabled=false"}},
		{"registered_help_flags", []string{"-h", "short", "--help", "long"}, []string{"-h", "short", "--help", "long"}},
		{"implicit_help_flag", []string{"-port", "1", "--unknown", "value", "-h"}, []string{"-port", "1", "-h"}},
		{"implicit_help_position", []string{"-port", "1", "--help", "-port", "2"}, []string{"-port", "1", "--help", "-port", "2"}},
		{"undefined_help_value", []string{"--help=value"}, []string{"--help=value"}},
		{"termination", []string{"-port", "1", "--", "-path", "ignored", "-h"}, []string{"-port", "1"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fset := newExtractionFlagSet()
			if strings.Contains(tc.name, "help") && tc.name != "registered_help_flags" {
				fset = flag.NewFlagSet("app", flag.ContinueOnError)
				fset.String("port", "", "")
			}

			res := extractRegistered(fset, nil, tc.args...)
			assert.Equal(t, tc.expected, res)
		})
	}

	t.Run("disallowed_registered_help_flags", func(t *testing.T) {
		fset := newExtractionFlagSet()
		names := map[string]struct{}{"port": {}}

		res := extractRegistered(fset, names, "-h", "ignored", "--help", "ignored", "-port", "1")

		assert.Equal(t, []string{"-port", "1"}, res)
	})

	t.Run("allow_list_assigns_values_before_filtering", func(t *testing.T) {
		testCases := []struct {
			name     string
			names    map[string]struct{}
			args     []string
			expected []string
		}{
			{
				name:     "excluded_non_boolean_owns_dash_prefixed_value",
				names:    map[string]struct{}{"port": {}},
				args:     []string{"-host", "-port"},
				expected: nil,
			},
			{
				name:     "allowed_non_boolean_preserves_dash_prefixed_value",
				names:    map[string]struct{}{"host": {}},
				args:     []string{"-host", "-literal"},
				expected: []string{"-host", "-literal"},
			},
			{
				name:     "excluded_boolean_does_not_own_next_token",
				names:    map[string]struct{}{"port": {}},
				args:     []string{"-v", "-port", "1"},
				expected: []string{"-port", "1"},
			},
			{
				name:     "excluded_inline_value_does_not_own_next_token",
				names:    map[string]struct{}{"port": {}},
				args:     []string{"-host=value", "-port", "1"},
				expected: []string{"-port", "1"},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				res := extractRegistered(newExtractionFlagSet(), tc.names, tc.args...)

				assert.Equal(t, tc.expected, res)
			})
		}
	})
}

func BenchmarkExtractRegistered(b *testing.B) {
	args := []string{"-host", "localhost", "-port", "8080", "-host", "127.0.0.1", "-v", "-t", "-path=/tmp/1.txt", "-named", "-m", "true", "/home/matthew/password.txt"}
	fset := newExtractionFlagSet()

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		extractRegistered(fset, nil, args...)
	}
}

func TestParseFlagTag(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected flagStructTag
	}{
		{"empty", "", flagStructTag{}},
		{"name_only", "u", flagStructTag{name: "u"}},
		{"name_and_usage", "u,usage=admin username", flagStructTag{name: "u", usage: "admin username"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := parseFlagTag(tc.input)
			assert.Equal(t, tc.expected, res)
		})
	}
}
