package reflect

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetValueFloatBitWidths(t *testing.T) {
	t.Run("float32_overflow_preserves_target", func(t *testing.T) {
		target := float32(12.5)
		err := SetValue(reflect.ValueOf(&target).Elem(), "1e39")

		assert.EqualError(t, err, `cannot convert value '1e39' to float: strconv.ParseFloat: parsing "1e39": value out of range`)
		assert.Equal(t, float32(12.5), target)
	})

	t.Run("float64_accepts_same_magnitude", func(t *testing.T) {
		target := float64(-1)
		err := SetValue(reflect.ValueOf(&target).Elem(), "1e39")

		assert.NoError(t, err)
		assert.Equal(t, float64(1e39), target)
	})
}

func TestSetValueComplexBitWidths(t *testing.T) {
	t.Run("complex64_overflow_preserves_target", func(t *testing.T) {
		target := complex64(1 + 2i)
		err := SetValue(reflect.ValueOf(&target).Elem(), "1e39+1e39i")

		assert.EqualError(t, err, `cannot convert value '1e39+1e39i' to complex: strconv.ParseComplex: parsing "1e39+1e39i": value out of range`)
		assert.Equal(t, complex64(1+2i), target)
	})

	t.Run("complex128_accepts_same_magnitude", func(t *testing.T) {
		target := complex128(-1 - 2i)
		err := SetValue(reflect.ValueOf(&target).Elem(), "1e39+1e39i")

		assert.NoError(t, err)
		assert.Equal(t, complex128(1e39+1e39i), target)
	})
}

func TestSetValueIntegerBitWidths(t *testing.T) {
	t.Run("int64_min", func(t *testing.T) {
		var target int64
		err := SetValue(reflect.ValueOf(&target).Elem(), "-9223372036854775808")

		assert.NoError(t, err)
		assert.Equal(t, int64(-1<<63), target)
	})

	t.Run("int64_max", func(t *testing.T) {
		var target int64
		err := SetValue(reflect.ValueOf(&target).Elem(), "9223372036854775807")

		assert.NoError(t, err)
		assert.Equal(t, int64(1<<63-1), target)
	})

	t.Run("uint64_max", func(t *testing.T) {
		var target uint64
		err := SetValue(reflect.ValueOf(&target).Elem(), "18446744073709551615")

		assert.NoError(t, err)
		assert.Equal(t, uint64(1<<64-1), target)
	})
}

func TestSetValueIntegerTargetOverflow(t *testing.T) {
	t.Run("int8", func(t *testing.T) {
		target := int8(12)
		err := SetValue(reflect.ValueOf(&target).Elem(), "128")

		assert.EqualError(t, err, "value 128 overflows int8")
		assert.Equal(t, int8(12), target)
	})

	t.Run("uint8", func(t *testing.T) {
		target := uint8(12)
		err := SetValue(reflect.ValueOf(&target).Elem(), "256")

		assert.EqualError(t, err, "value 256 overflows uint8")
		assert.Equal(t, uint8(12), target)
	})
}
