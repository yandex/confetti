package json

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
)

// FromString loads values from JSON string to given struct
func FromString(s string) func(context.Context, any) error {
	return func(ctx context.Context, target any) error {
		return FromReader(bytes.NewBufferString(s))(ctx, target)
	}
}

// FromFile loads values from file to given struct
func FromFile(path string) func(context.Context, any) error {
	return func(ctx context.Context, target any) error {
		fd, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("cannot open JSON file: %w", err)
		}
		defer fd.Close()
		return FromReader(fd)(ctx, target)
	}
}

// FromReader loads values from reader to given struct
func FromReader(r io.Reader) func(context.Context, any) error {
	return func(ctx context.Context, target any) error {
		tv := reflect.ValueOf(target)
		if !tv.IsValid() || tv.Kind() != reflect.Ptr || tv.Elem().Kind() != reflect.Struct {
			return errors.New("target must be a pointer to struct")
		}
		return json.NewDecoder(r).
			Decode(target)
	}
}
