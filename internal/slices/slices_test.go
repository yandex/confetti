package slices_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.yandex/confetti/internal/slices"
)

type testSrc struct{}

func (testSrc) Int63() int64 {
	return 1 << 62
}

func (testSrc) Seed(int64) {}

func TestMap(t *testing.T) {
	t.Run("double", func(t *testing.T) {
		s := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
		res := slices.Map(s, func(v int) int {
			return v * 2
		})
		expected := []int{2, 4, 6, 8, 10, 12, 14, 16, 18, 20}
		assert.Equal(t, expected, res)
	})
	t.Run("custom_type", func(t *testing.T) {
		type ohMyInts []int
		type ohMyFloats []float32

		s := ohMyInts{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
		res := slices.Map(s, func(v int) float32 {
			return float32(v) / 2
		})
		expected := ohMyFloats{0.5, 1, 1.5, 2, 2.5, 3, 3.5, 4, 4.5, 5}
		assert.Equal(t, expected, ohMyFloats(res))
	})
}

func TestShuffle(t *testing.T) {
	original := []int{1, 2, 3, 4}
	input := []int{1, 2, 3, 4}
	expected := []int{1, 4, 2, 3}

	result := slices.Shuffle(input, testSrc{})

	assert.Equal(t, expected, result)
	assert.Equal(t, expected, input)
	assert.NotEqual(t, original, input)
}
