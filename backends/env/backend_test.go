package env

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.yandex/confetti/internal/ptr"
)

func TestFromVar(t *testing.T) {
	t.Run("empty_env", func(t *testing.T) {
		t.Cleanup(os.Clearenv)

		key := time.Now().String()
		setter := FromVar(key)

		var target string
		err := setter(context.Background(), &target)
		assert.NoError(t, err)
		assert.Empty(t, target)
	})

	t.Run("success", func(t *testing.T) {
		t.Cleanup(os.Clearenv)

		key := "TEST_" + strconv.Itoa(int(time.Now().UnixNano()))
		err := os.Setenv(key, "42")
		assert.NoError(t, err)
		setter := FromVar(key)

		t.Run("string", func(t *testing.T) {
			var target string
			err = setter(context.Background(), &target)
			assert.Equal(t, "42", target)
			assert.NoError(t, err)
		})
		t.Run("int", func(t *testing.T) {
			var target int
			err = setter(context.Background(), &target)
			assert.Equal(t, int(42), target)
			assert.NoError(t, err)
		})
		t.Run("uint", func(t *testing.T) {
			var target uint
			err = setter(context.Background(), &target)
			assert.Equal(t, uint(42), target)
			assert.NoError(t, err)
		})
		t.Run("float", func(t *testing.T) {
			var target float32
			err = setter(context.Background(), &target)
			assert.Equal(t, float32(42), target)
			assert.NoError(t, err)
		})
		t.Run("bool", func(t *testing.T) {
			key := "TEST_" + strconv.Itoa(int(time.Now().UnixNano())) + "_BOOL"
			err := os.Setenv(key, "true")
			assert.NoError(t, err)
			setter := FromVar(key)

			var target bool
			err = setter(context.Background(), &target)
			assert.True(t, target)
			assert.NoError(t, err)
		})
		t.Run("complex", func(t *testing.T) {
			key := "TEST_" + strconv.Itoa(int(time.Now().UnixNano())) + "_COMPLEX"
			err := os.Setenv(key, "42i")
			assert.NoError(t, err)
			setter := FromVar(key)

			var target complex64
			err = setter(context.Background(), &target)
			assert.Equal(t, complex64(42i), target)
			assert.NoError(t, err)
		})
		t.Run("duration", func(t *testing.T) {
			key := "TEST_" + strconv.Itoa(int(time.Now().UnixNano())) + "_DURATION"
			err := os.Setenv(key, "42ns")
			assert.NoError(t, err)
			setter := FromVar(key)

			var target time.Duration
			err = setter(context.Background(), &target)
			assert.Equal(t, time.Duration(42), target)
			assert.NoError(t, err)
		})
		t.Run("text_unmarshaler", func(t *testing.T) {
			key := "TEST_" + strconv.Itoa(int(time.Now().UnixNano())) + "_TEXT_UNMARSHALER"
			err := os.Setenv(key, "debug")
			assert.NoError(t, err)
			setter := FromVar(key)

			var target Level
			err = setter(context.Background(), &target)
			assert.Equal(t, DebugLevel, target)
			assert.NoError(t, err)
		})
	})

	t.Run("bad_value", func(t *testing.T) {
		key := "TEST_" + strconv.Itoa(int(time.Now().UnixNano()))
		err := os.Setenv(key, "_*_")
		assert.NoError(t, err)
		setter := FromVar(key)

		t.Run("int", func(t *testing.T) {
			var target int
			err = setter(context.Background(), &target)
			assert.Empty(t, target)
			assert.EqualError(t, err, "cannot convert value '_*_' to int: strconv.ParseInt: parsing \"_*_\": invalid syntax")
		})
		t.Run("uint", func(t *testing.T) {
			var target uint
			err = setter(context.Background(), &target)
			assert.Empty(t, target)
			assert.EqualError(t, err, "cannot convert value '_*_' to uint: strconv.ParseUint: parsing \"_*_\": invalid syntax")
		})
		t.Run("float", func(t *testing.T) {
			var target float32
			err = setter(context.Background(), &target)
			assert.Empty(t, target)
			assert.EqualError(t, err, "cannot convert value '_*_' to float: strconv.ParseFloat: parsing \"_*_\": invalid syntax")
		})
		t.Run("bool", func(t *testing.T) {
			var target bool
			err = setter(context.Background(), &target)
			assert.Empty(t, target)
			assert.EqualError(t, err, "cannot convert value '_*_' to bool: strconv.ParseBool: parsing \"_*_\": invalid syntax")
		})
		t.Run("complex", func(t *testing.T) {
			var target complex64
			err = setter(context.Background(), &target)
			assert.Empty(t, target)
			assert.EqualError(t, err, "cannot convert value '_*_' to complex: strconv.ParseComplex: parsing \"_*_\": invalid syntax")
		})
	})
}

func TestFromVarNumericOverflowPreservesTarget(t *testing.T) {
	t.Run("float32", func(t *testing.T) {
		t.Setenv("CONFETTI_TEST_NUMERIC_BIT_WIDTH", "1e39")
		target := float32(12.5)

		err := FromVar("CONFETTI_TEST_NUMERIC_BIT_WIDTH")(t.Context(), &target)

		assert.ErrorContains(t, err, "cannot convert value '1e39' to float")
		assert.Equal(t, float32(12.5), target)
	})

	t.Run("complex64", func(t *testing.T) {
		t.Setenv("CONFETTI_TEST_NUMERIC_BIT_WIDTH", "1e39+1e39i")
		target := complex64(1 + 2i)

		err := FromVar("CONFETTI_TEST_NUMERIC_BIT_WIDTH")(t.Context(), &target)

		assert.ErrorContains(t, err, "cannot convert value '1e39+1e39i' to complex")
		assert.Equal(t, complex64(1+2i), target)
	})
}

func TestFromEnviron(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		t.Cleanup(os.Clearenv)

		type profile struct {
			User     string  `env:"USER"`
			Password string  `env:"-"`
			Group    string  `env:"GROUP"`
			HomeDir  string  `env:"HOME_DIR"`
			State    *string `env:"STATE"`
		}

		assert.NoError(t, os.Setenv("USER", "volozh"))
		assert.NoError(t, os.Setenv("PASSWORD", "root"))
		assert.NoError(t, os.Setenv("GROUP", "root"))
		assert.NoError(t, os.Setenv("HOME_DIR", "yandex"))
		assert.NoError(t, os.Setenv("STATE", "active"))

		var target profile

		setter := FromEnviron()
		err := setter(context.Background(), &target)
		assert.NoError(t, err)

		expected := profile{
			User:    "volozh",
			Group:   "root",
			HomeDir: "yandex",
			State:   ptr.T("active"),
		}
		assert.Equal(t, expected, target)
	})

	t.Run("custom_tag", func(t *testing.T) {
		t.Cleanup(os.Clearenv)

		type profile struct {
			User     string `environ:"USER"`
			Password string `env:"-"`
			Group    string `environ:"GROUP"`
			HomeDir  string `environ:"HOME_DIR"`
		}

		assert.NoError(t, os.Setenv("USER", "volozh"))
		assert.NoError(t, os.Setenv("PASSWORD", "root"))
		assert.NoError(t, os.Setenv("GROUP", "root"))
		assert.NoError(t, os.Setenv("HOME_DIR", "yandex"))

		var target profile

		setter := FromEnviron(Tag("environ"))
		err := setter(context.Background(), &target)
		assert.NoError(t, err)

		expected := profile{
			User:    "volozh",
			Group:   "root",
			HomeDir: "yandex",
		}
		assert.Equal(t, expected, target)
	})
	t.Run("unexported_tagged_scalar", func(t *testing.T) {
		type profile struct {
			Visible string `environ:"VISIBLE"`
			hidden  string `environ:"HIDDEN"`
		}

		t.Setenv("VISIBLE", "visible")
		t.Setenv("HIDDEN", "hidden")

		target := profile{hidden: "unchanged"}
		setter := FromEnviron(Tag("environ"))

		err := setter(t.Context(), &target)

		assert.NoError(t, err)
		assert.Equal(t, "visible", target.Visible)
		assert.Equal(t, "unchanged", target.hidden)
	})
	t.Run("unexported_nested_struct", func(t *testing.T) {
		type nested struct {
			Child string `env:"HIDDEN_CHILD"`
		}
		type profile struct {
			Visible string `env:"VISIBLE"`
			hidden  nested `env:"HIDDEN"`
		}

		t.Setenv("VISIBLE", "visible")
		t.Setenv("HIDDEN", "hidden")
		t.Setenv("HIDDEN_CHILD", "hidden child")

		target := profile{hidden: nested{Child: "unchanged"}}
		setter := FromEnviron()

		err := setter(t.Context(), &target)

		assert.NoError(t, err)
		assert.Equal(t, "visible", target.Visible)
		assert.Equal(t, "unchanged", target.hidden.Child)
	})
	t.Run("nested_struct", func(t *testing.T) {
		t.Cleanup(os.Clearenv)

		type mounts struct {
			Root string `env:"MOUNT_ROOT"`
			Swap string `env:"MOUNT_SWAP"`
		}

		type profile struct {
			User     string `env:"USER"`
			Password string `env:"-"`
			Group    string `env:"GROUP"`
			HomeDir  string `env:"HOME_DIR"`
			Mounts   mounts
		}

		assert.NoError(t, os.Setenv("USER", "volozh"))
		assert.NoError(t, os.Setenv("PASSWORD", "root"))
		assert.NoError(t, os.Setenv("GROUP", "root"))
		assert.NoError(t, os.Setenv("HOME_DIR", "yandex"))

		assert.NoError(t, os.Setenv("MOUNT_ROOT", "/"))
		assert.NoError(t, os.Setenv("MOUNT_SWAP", "/swap"))

		var target profile

		setter := FromEnviron()
		err := setter(context.Background(), &target)
		assert.NoError(t, err)

		expected := profile{
			User:    "volozh",
			Group:   "root",
			HomeDir: "yandex",
			Mounts: mounts{
				Root: "/",
				Swap: "/swap",
			},
		}
		assert.Equal(t, expected, target)
	})
	t.Run("nested_struct_pointer", func(t *testing.T) {
		t.Cleanup(os.Clearenv)

		type mounts struct {
			Root string `env:"MOUNT_ROOT"`
			Swap string `env:"MOUNT_SWAP"`
		}

		type profile struct {
			User     string `env:"USER"`
			Password string `env:"-"`
			Group    string `env:"GROUP"`
			HomeDir  string `env:"HOME_DIR"`
			Mounts   *mounts
		}

		assert.NoError(t, os.Setenv("USER", "volozh"))
		assert.NoError(t, os.Setenv("PASSWORD", "root"))
		assert.NoError(t, os.Setenv("GROUP", "root"))
		assert.NoError(t, os.Setenv("HOME_DIR", "yandex"))

		assert.NoError(t, os.Setenv("MOUNT_ROOT", "/"))
		assert.NoError(t, os.Setenv("MOUNT_SWAP", "/swap"))

		var target profile

		setter := FromEnviron()
		err := setter(context.Background(), &target)
		assert.NoError(t, err)

		expected := profile{
			User:    "volozh",
			Group:   "root",
			HomeDir: "yandex",
			Mounts: &mounts{
				Root: "/",
				Swap: "/swap",
			},
		}
		assert.Equal(t, expected, target)
	})
	t.Run("nested_struct_pointer_when_missing_env", func(t *testing.T) {
		t.Cleanup(os.Clearenv)

		type mounts struct {
			Root string `env:"MOUNT_ROOT"`
			Swap string `env:"MOUNT_SWAP"`
		}

		type profile struct {
			User     string `env:"USER"`
			Password string `env:"-"`
			Group    string `env:"GROUP"`
			HomeDir  string `env:"HOME_DIR"`
			Mounts   *mounts
		}

		assert.NoError(t, os.Setenv("USER", "volozh"))
		assert.NoError(t, os.Setenv("PASSWORD", "root"))
		assert.NoError(t, os.Setenv("GROUP", "root"))
		assert.NoError(t, os.Setenv("HOME_DIR", "yandex"))

		var target profile

		setter := FromEnviron()
		err := setter(context.Background(), &target)
		assert.NoError(t, err)

		expected := profile{
			User:    "volozh",
			Group:   "root",
			HomeDir: "yandex",
			Mounts:  nil,
		}
		assert.Equal(t, expected, target)
	})
	t.Run("nested_struct_with_prefix", func(t *testing.T) {
		t.Cleanup(os.Clearenv)

		type volumes struct {
			Point string `env:"POINT"`
			Size  uint64 `env:"SIZE"`
		}

		type mounts struct {
			Root volumes `env:"ROOT_MOUNT"`
			Swap volumes `env:"SWAP_MOUNT"`
		}

		type profile struct {
			User     string `env:"USER"`
			Password string `env:"-"`
			Group    string `env:"GROUP"`
			HomeDir  string `env:"HOME_DIR"`
			Mounts   mounts
		}

		assert.NoError(t, os.Setenv("USER", "volozh"))
		assert.NoError(t, os.Setenv("PASSWORD", "root"))
		assert.NoError(t, os.Setenv("GROUP", "root"))
		assert.NoError(t, os.Setenv("HOME_DIR", "yandex"))

		assert.NoError(t, os.Setenv("ROOT_MOUNT_POINT", "/"))
		assert.NoError(t, os.Setenv("ROOT_MOUNT_SIZE", "5368709120"))
		assert.NoError(t, os.Setenv("SWAP_MOUNT_POINT", "/swap"))
		assert.NoError(t, os.Setenv("SWAP_MOUNT_SIZE", "1073741824"))

		var target profile

		setter := FromEnviron(RecursiveKeys)
		err := setter(context.Background(), &target)
		assert.NoError(t, err)

		expected := profile{
			User:    "volozh",
			Group:   "root",
			HomeDir: "yandex",
			Mounts: mounts{
				Root: volumes{
					Point: "/",
					Size:  5_368_709_120,
				},
				Swap: volumes{
					Point: "/swap",
					Size:  1_073_741_824,
				},
			},
		}
		assert.Equal(t, expected, target)
	})
	t.Run("nested_struct_pointer_with_prefix", func(t *testing.T) {
		t.Cleanup(os.Clearenv)

		type volumes struct {
			Point string `env:"POINT"`
			Size  uint64 `env:"SIZE"`
		}

		type mounts struct {
			Root *volumes `env:"ROOT_MOUNT"`
			Swap *volumes `env:"SWAP_MOUNT"`
		}

		type profile struct {
			User     string `env:"USER"`
			Password string `env:"-"`
			Group    string `env:"GROUP"`
			HomeDir  string `env:"HOME_DIR"`
			Mounts   *mounts
		}

		assert.NoError(t, os.Setenv("USER", "volozh"))
		assert.NoError(t, os.Setenv("PASSWORD", "root"))
		assert.NoError(t, os.Setenv("GROUP", "root"))
		assert.NoError(t, os.Setenv("HOME_DIR", "yandex"))

		assert.NoError(t, os.Setenv("ROOT_MOUNT_POINT", "/"))
		assert.NoError(t, os.Setenv("ROOT_MOUNT_SIZE", "5368709120"))
		assert.NoError(t, os.Setenv("SWAP_MOUNT_POINT", "/swap"))
		assert.NoError(t, os.Setenv("SWAP_MOUNT_SIZE", "1073741824"))

		var target profile

		setter := FromEnviron(RecursiveKeys)
		err := setter(context.Background(), &target)
		assert.NoError(t, err)

		expected := profile{
			User:    "volozh",
			Group:   "root",
			HomeDir: "yandex",
			Mounts: &mounts{
				Root: &volumes{
					Point: "/",
					Size:  5_368_709_120,
				},
				Swap: &volumes{
					Point: "/swap",
					Size:  1_073_741_824,
				},
			},
		}
		assert.Equal(t, expected, target)
	})

	t.Run("pointer", func(t *testing.T) {
		t.Cleanup(os.Clearenv)

		type test struct {
			Empty      *int
			Default    *int `env:"DEFAULT"`
			NonDefault *int `env:"NON_DEFAULT"`
		}

		assert.NoError(t, os.Setenv("DEFAULT", "0"))
		assert.NoError(t, os.Setenv("NON_DEFAULT", "1"))

		var target test

		setter := FromEnviron()
		err := setter(context.Background(), &target)
		assert.NoError(t, err)

		expected := test{
			Default:    new(int),
			NonDefault: new(int),
		}

		*expected.Default = 0
		*expected.NonDefault = 1
		assert.Equal(t, expected, target)
	})

	t.Run("double_pointer", func(t *testing.T) {
		t.Cleanup(os.Clearenv)

		type test struct {
			Empty      **int
			Default    **int `env:"DEFAULT"`
			NonDefault **int `env:"NON_DEFAULT"`
		}

		assert.NoError(t, os.Setenv("DEFAULT", "0"))
		assert.NoError(t, os.Setenv("NON_DEFAULT", "1"))

		var target test

		setter := FromEnviron()
		err := setter(context.Background(), &target)
		assert.NoError(t, err)

		expected := test{
			Default:    new(*int),
			NonDefault: new(*int),
		}
		*expected.Default = new(int)
		*expected.NonDefault = new(int)

		**expected.Default = 0
		**expected.NonDefault = 1
		assert.Equal(t, expected, target)
	})
}

func TestFromEnvironPrefixOnRootField(t *testing.T) {
	t.Setenv("PASSWORD", "unprefixed")
	t.Setenv("APP_PASSWORD", "root")
	var cfg struct {
		Password string `env:"PASSWORD"`
	}

	err := FromEnviron(Prefix("APP"))(t.Context(), &cfg)

	assert.NoError(t, err)
	assert.Equal(t, "root", cfg.Password)
}

func TestFromEnvironSkipsUntaggedScalars(t *testing.T) {
	t.Run("root", func(t *testing.T) {
		t.Setenv("APP_", "ignored")
		t.Setenv("APP_VALUE", "configured")
		cfg := struct {
			Untagged string
			Value    string `env:"VALUE"`
		}{Untagged: "initial"}

		err := FromEnviron(Prefix("APP"))(t.Context(), &cfg)

		assert.NoError(t, err)
		assert.Equal(t, "initial", cfg.Untagged)
		assert.Equal(t, "configured", cfg.Value)
	})

	t.Run("root with custom tag", func(t *testing.T) {
		t.Setenv("APP_", "ignored")
		t.Setenv("APP_VALUE", "configured")
		cfg := struct {
			Untagged string
			Value    string `environ:"VALUE"`
		}{Untagged: "initial"}

		err := FromEnviron(Prefix("APP"), Tag("environ"))(t.Context(), &cfg)

		assert.NoError(t, err)
		assert.Equal(t, "initial", cfg.Untagged)
		assert.Equal(t, "configured", cfg.Value)
	})

	t.Run("below tagged parent", func(t *testing.T) {
		t.Setenv("APP::PARENT::", "ignored")
		t.Setenv("APP::PARENT::VALUE", "configured")
		cfg := struct {
			Parent struct {
				Untagged string
				Value    string `env:"VALUE"`
			} `env:"PARENT"`
		}{Parent: struct {
			Untagged string
			Value    string `env:"VALUE"`
		}{Untagged: "initial"}}

		err := FromEnviron(Prefix("APP"), RecursiveKeys, RecursiveKeyGlue("::"))(t.Context(), &cfg)

		assert.NoError(t, err)
		assert.Equal(t, "initial", cfg.Parent.Untagged)
		assert.Equal(t, "configured", cfg.Parent.Value)
	})
}

func TestFromEnvironPrefixUnderTaggedStruct(t *testing.T) {
	t.Setenv("VALUE", "unprefixed")
	t.Setenv("NESTED_VALUE", "parent-only")
	t.Setenv("APP_VALUE", "static")
	t.Setenv("APP_NESTED_VALUE", "recursive")
	var cfg struct {
		Nested struct {
			Value string `env:"VALUE"`
		} `env:"NESTED"`
	}

	err := FromEnviron(Prefix("APP"))(t.Context(), &cfg)

	assert.NoError(t, err)
	assert.Equal(t, "static", cfg.Nested.Value)
}

func TestFromEnvironPrefixUnderUntaggedStructs(t *testing.T) {
	t.Setenv("VALUE", "unprefixed")
	t.Setenv("APP_VALUE", "static")
	var cfg struct {
		Outer struct {
			Inner struct {
				Value string `env:"VALUE"`
			}
		}
	}

	err := FromEnviron(Prefix("APP"))(t.Context(), &cfg)

	assert.NoError(t, err)
	assert.Equal(t, "static", cfg.Outer.Inner.Value)
}

func TestFromEnvironPrefixWithRecursiveKeys(t *testing.T) {
	t.Setenv("APPNESTEDVALUE", "no-glue")
	t.Setenv("APP_NESTED_VALUE", "recursive")
	var cfg struct {
		Nested struct {
			Value string `env:"VALUE"`
		} `env:"NESTED"`
	}

	err := FromEnviron(Prefix("APP"), RecursiveKeys)(t.Context(), &cfg)

	assert.NoError(t, err)
	assert.Equal(t, "recursive", cfg.Nested.Value)
}

func TestFromEnvironRecursiveKeyGlueOptionOrder(t *testing.T) {
	testCases := []struct {
		name string
		opts []EnvironOpt
	}{
		{name: "before_recursive_keys", opts: []EnvironOpt{Prefix("APP"), RecursiveKeyGlue("::"), RecursiveKeys}},
		{name: "after_recursive_keys", opts: []EnvironOpt{Prefix("APP"), RecursiveKeys, RecursiveKeyGlue("::")}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Setenv("APP_NESTED_VALUE", "default-glue")
			t.Setenv("APP::NESTED::VALUE", "custom")
			var cfg struct {
				Nested struct {
					Value string `env:"VALUE"`
				} `env:"NESTED"`
			}

			err := FromEnviron(testCase.opts...)(t.Context(), &cfg)

			assert.NoError(t, err)
			assert.Equal(t, "custom", cfg.Nested.Value)
		})
	}
}

func TestFromEnviron_map_slice(t *testing.T) {
	type nested struct {
		Value int `env:"VALUE"`
	}

	t.Run("map_nested_struct", func(t *testing.T) {
		t.Cleanup(os.Clearenv)

		var cfg struct {
			Password string `env:"PASSWORD"`
			Map      map[string]nested
		}
		cfg.Map = map[string]nested{
			"first":  {},
			"second": {},
		}

		assert.NoError(t, os.Setenv("APP_PASSWORD", "root"))
		assert.NoError(t, os.Setenv("APP_VALUE", "123"))

		setter := FromEnviron(RecursiveKeys, Prefix("APP"))
		assert.NoError(t, setter(context.Background(), &cfg))

		assert.Equal(t, "root", cfg.Password)
		assert.Equal(t, 123, cfg.Map["first"].Value)
		assert.Equal(t, 123, cfg.Map["second"].Value)
	})

	t.Run("pointer_map_nested_struct", func(t *testing.T) {
		t.Cleanup(os.Clearenv)

		var cfg struct {
			Password string `env:"PASSWORD"`
			Map      *map[string]nested
		}
		cfg.Map = new(map[string]nested)
		*cfg.Map = map[string]nested{
			"first":  {},
			"second": {},
		}

		assert.NoError(t, os.Setenv("APP_PASSWORD", "root"))
		assert.NoError(t, os.Setenv("APP_VALUE", "123"))

		setter := FromEnviron(RecursiveKeys, Prefix("APP"))
		assert.NoError(t, setter(context.Background(), &cfg))

		assert.Equal(t, "root", cfg.Password)
		assert.Equal(t, 123, (*cfg.Map)["first"].Value)
		assert.Equal(t, 123, (*cfg.Map)["second"].Value)
	})

	t.Run("map_pointer_nested_struct", func(t *testing.T) {
		t.Cleanup(os.Clearenv)

		var cfg struct {
			Password string `env:"PASSWORD"`
			Map      map[string]*nested
		}
		cfg.Map = map[string]*nested{
			"first":  nil,
			"second": nil,
		}

		assert.NoError(t, os.Setenv("APP_PASSWORD", "root"))
		assert.NoError(t, os.Setenv("APP_VALUE", "123"))

		setter := FromEnviron(RecursiveKeys, Prefix("APP"))
		assert.NoError(t, setter(context.Background(), &cfg))

		assert.Equal(t, "root", cfg.Password)
		assert.Equal(t, 123, cfg.Map["first"].Value)
		assert.Equal(t, 123, cfg.Map["second"].Value)
	})

	t.Run("slice_nested_struct", func(t *testing.T) {
		t.Cleanup(os.Clearenv)

		var cfg struct {
			Password string `env:"PASSWORD"`
			Slice    []nested
		}
		cfg.Slice = make([]nested, 2)

		assert.NoError(t, os.Setenv("APP_PASSWORD", "root"))
		assert.NoError(t, os.Setenv("APP_VALUE", "123"))

		setter := FromEnviron(RecursiveKeys, Prefix("APP"))
		assert.NoError(t, setter(context.Background(), &cfg))

		assert.Equal(t, "root", cfg.Password)
		assert.Equal(t, 123, cfg.Slice[0].Value)
		assert.Equal(t, 123, cfg.Slice[1].Value)
	})

	t.Run("pointer_slice_nested_struct", func(t *testing.T) {
		t.Cleanup(os.Clearenv)

		var cfg struct {
			Password string `env:"PASSWORD"`
			Slice    *[]nested
		}
		cfg.Slice = &[]nested{{}, {}}

		assert.NoError(t, os.Setenv("APP_PASSWORD", "root"))
		assert.NoError(t, os.Setenv("APP_VALUE", "123"))

		setter := FromEnviron(RecursiveKeys, Prefix("APP"))
		assert.NoError(t, setter(context.Background(), &cfg))

		assert.Equal(t, "root", cfg.Password)
		assert.Equal(t, 123, (*cfg.Slice)[0].Value)
		assert.Equal(t, 123, (*cfg.Slice)[1].Value)
	})

	t.Run("slice_pointer_nested_struct", func(t *testing.T) {
		t.Cleanup(os.Clearenv)

		var cfg struct {
			Password string `env:"PASSWORD"`
			Slice    []*nested
		}
		cfg.Slice = make([]*nested, 2)

		assert.NoError(t, os.Setenv("APP_PASSWORD", "root"))
		assert.NoError(t, os.Setenv("APP_VALUE", "123"))

		setter := FromEnviron(RecursiveKeys, Prefix("APP"))
		assert.NoError(t, setter(context.Background(), &cfg))

		assert.Equal(t, "root", cfg.Password)
		assert.Equal(t, 123, cfg.Slice[0].Value)
		assert.Equal(t, 123, cfg.Slice[1].Value)
	})
}
