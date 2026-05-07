package json

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFromString(t *testing.T) {
	t.Run("non_struct", func(t *testing.T) {
		var target float32
		setter := FromString(`{"test": "shimba"}`)
		err := setter(context.Background(), &target)
		assert.EqualError(t, err, "target must be a pointer to struct")
	})
	t.Run("single_json", func(t *testing.T) {
		type config struct {
			Test string `json:"test"`
		}

		var target config
		setter := FromString(`{"test": "shimba"}`)

		err := setter(context.Background(), &target)
		assert.NoError(t, err)

		expected := config{
			Test: "shimba",
		}
		assert.Equal(t, expected, target)
	})
	t.Run("override_json", func(t *testing.T) {
		type config struct {
			User     string `json:"user"`
			Password string `json:"password"`
			Host     string `json:"host"`
		}

		var target config

		setter := FromString(`{"user": "nobody", "host": "publichost.mycloud.com"}`)
		err := setter(context.Background(), &target)
		assert.NoError(t, err)

		setter = FromString(`{"user": "admin", "password": "lookenTooken", "host": "securedhost.mycloud.com"}`)
		err = setter(context.Background(), &target)
		assert.NoError(t, err)

		expected := config{
			User:     "admin",
			Password: "lookenTooken",
			Host:     "securedhost.mycloud.com",
		}
		assert.Equal(t, expected, target)
	})
}

func TestFromReader(t *testing.T) {
	t.Run("non_struct", func(t *testing.T) {
		var target float32
		setter := FromString(`{"test": "shimba"}`)
		err := setter(context.Background(), &target)
		assert.EqualError(t, err, "target must be a pointer to struct")
	})
	t.Run("single_json", func(t *testing.T) {
		type config struct {
			Test string `json:"test"`
		}

		var target config
		setter := FromReader(bytes.NewBufferString(`{"test": "shimba"}`))

		err := setter(context.Background(), &target)
		assert.NoError(t, err)

		expected := config{
			Test: "shimba",
		}
		assert.Equal(t, expected, target)
	})
	t.Run("override_json", func(t *testing.T) {
		type config struct {
			User     string `json:"user"`
			Password string `json:"password"`
			Host     string `json:"host"`
		}

		var target config

		setter := FromReader(bytes.NewBufferString(`{"user": "nobody", "host": "publichost.mycloud.com"}`))
		err := setter(context.Background(), &target)
		assert.NoError(t, err)

		setter = FromReader(bytes.NewBufferString(`{"user": "admin", "password": "lookenTooken", "host": "securedhost.mycloud.com"}`))
		err = setter(context.Background(), &target)
		assert.NoError(t, err)

		expected := config{
			User:     "admin",
			Password: "lookenTooken",
			Host:     "securedhost.mycloud.com",
		}
		assert.Equal(t, expected, target)
	})
}
