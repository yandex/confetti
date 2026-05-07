package reflect

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testTraverseContext = TraverseContext[[]Node]

func newTestTraverseContext(ctx context.Context) testTraverseContext {
	return NewTraverseContext(ctx, []Node{})
}

func recursiveElemWithNewInplace(node Node) Node {
	val := node.Value
	for val.Kind() == reflect.Ptr {
		// set zero value of proper type
		if val.IsNil() {
			val.Set(reflect.New(val.Type().Elem()))
		}
		// unwrap value
		val = val.Elem()
	}
	return Node{
		Value:          val,
		HasStructField: node.HasStructField,
		StructField:    node.StructField,
	}
}

func callback(ctx *testTraverseContext, field Node) error {
	ctx.Data = append(ctx.Data, field)

	if !field.HasStructField {
		return nil
	}
	tag, ok := field.StructField.Tag.Lookup("test")
	if !ok {
		return nil
	}

	field = recursiveElemWithNewInplace(field)

	if !field.Value.IsValid() {
		return nil
	}
	if field.Value.Kind() != reflect.String {
		return nil
	}
	field.Value.SetString(tag)

	return nil
}

func TestTraverseStruct(t *testing.T) {
	t.Run("s", func(t *testing.T) {
		var target struct {
			V ***int
		}
		tv := reflect.ValueOf(&target)

		recursiveElemWithNew(Node{Value: tv.Elem().Field(0)})

	})

	t.Run("not_struct", func(t *testing.T) {
		var target int
		tv := reflect.ValueOf(target)

		err := TraverseStruct(newTestTraverseContext(context.Background()), tv, callback)
		assert.Error(t, err)
	})

	type Empty struct {
		Name2 string
	}
	type Nested struct {
		Name2 string `test:"Skinderev"`
	}
	type NestedPtr struct {
		Name2 *string `test:"Skinderev"`
	}

	t.Run("basic", func(t *testing.T) {
		type TargetType struct {
			FirstName string `test:"Oleg"`
			LastName  string `test:"Skinderev"`
		}
		var target TargetType

		tv := reflect.ValueOf(&target)

		err := TraverseStruct(newTestTraverseContext(context.Background()), tv, callback)
		assert.NoError(t, err)

		expected := TargetType{
			FirstName: "Oleg",
			LastName:  "Skinderev",
		}
		assert.Equal(t, expected, target)

	})

	t.Run("basic_pointer", func(t *testing.T) {
		type TargetType struct {
			Name  *string `test:"Oleg"`
			Name2 *string `test:"Skinderev"`
		}
		var target TargetType

		tv := reflect.ValueOf(&target)

		err := TraverseStruct(newTestTraverseContext(context.Background()), tv, callback)
		assert.NoError(t, err)

		expected := TargetType{
			Name:  new(string),
			Name2: new(string),
		}
		*expected.Name = "Oleg"
		*expected.Name2 = "Skinderev"
		assert.Equal(t, expected, target)

	})

	t.Run("nested_struct", func(t *testing.T) {
		type TargetType struct {
			Name   string `test:"Oleg"`
			Nested Nested
		}
		var target TargetType

		tv := reflect.ValueOf(&target)

		err := TraverseStruct(newTestTraverseContext(context.Background()), tv, callback)
		assert.NoError(t, err)

		expected := TargetType{
			Name: "Oleg",
			Nested: Nested{
				Name2: "Skinderev",
			},
		}
		assert.Equal(t, expected, target)
	})

	t.Run("pointer_nested_struct", func(t *testing.T) {
		type TargetType struct {
			Name   string `test:"Oleg"`
			Nested *Nested
		}
		var target TargetType

		tv := reflect.ValueOf(&target)

		err := TraverseStruct(newTestTraverseContext(context.Background()), tv, callback)
		assert.NoError(t, err)

		expected := TargetType{
			Name:   "Oleg",
			Nested: new(Nested),
		}
		expected.Nested.Name2 = "Skinderev"

		assert.Equal(t, expected, target)
	})

	t.Run("nested_struct_with_pointer", func(t *testing.T) {
		type TargetType struct {
			Name   string `test:"Oleg"`
			Nested NestedPtr
		}
		var target TargetType

		tv := reflect.ValueOf(&target)

		err := TraverseStruct(newTestTraverseContext(context.Background()), tv, callback)
		assert.NoError(t, err)

		expected := TargetType{
			Name: "Oleg",
			Nested: NestedPtr{
				Name2: new(string),
			},
		}
		*expected.Nested.Name2 = "Skinderev"
		assert.Equal(t, expected, target)
	})

	t.Run("pointer_nested_struct_with_pointer", func(t *testing.T) {
		type TargetType struct {
			Name   *string `test:"Oleg"`
			Nested *NestedPtr
		}
		var target TargetType

		tv := reflect.ValueOf(&target)

		err := TraverseStruct(newTestTraverseContext(context.Background()), tv, callback)
		assert.NoError(t, err)

		expected := TargetType{
			Name:   new(string),
			Nested: new(NestedPtr),
		}
		*expected.Name = "Oleg"
		expected.Nested.Name2 = new(string)
		*expected.Nested.Name2 = "Skinderev"
		assert.Equal(t, expected, target)
	})

	t.Run("nested_struct_pointer_empty", func(t *testing.T) {
		type Empty struct {
			Name2 string
		}

		type TargetType struct {
			Name   *string `test:"Oleg"`
			Nested *Empty
		}
		var target TargetType

		tv := reflect.ValueOf(&target)

		err := TraverseStruct(newTestTraverseContext(context.Background()), tv, callback)
		assert.NoError(t, err)

		expected := TargetType{
			Name:   new(string),
			Nested: nil,
		}
		*expected.Name = "Oleg"
		assert.Equal(t, expected, target)
	})

	t.Run("map_nested_struct", func(t *testing.T) {
		type TargetType struct {
			Name string `test:"Oleg"`
			Map  map[string]Nested
		}
		var target TargetType

		target.Map = map[string]Nested{
			"key1": {},
			"key2": {},
		}
		tv := reflect.ValueOf(&target)

		err := TraverseStruct(newTestTraverseContext(context.Background()), tv, callback)
		assert.NoError(t, err)

		expected := TargetType{
			Name: "Oleg",
		}
		expected.Map = map[string]Nested{
			"key1": {"Skinderev"},
			"key2": {"Skinderev"},
		}

		assert.Equal(t, expected, target)
	})

	t.Run("map_nested_struct_with_pointer", func(t *testing.T) {
		type TargetType struct {
			Name *string `test:"Oleg"`
			Map  map[string]NestedPtr
		}
		var target TargetType

		target.Map = map[string]NestedPtr{
			"key1": {},
			"key2": {},
		}
		tv := reflect.ValueOf(&target)

		err := TraverseStruct(newTestTraverseContext(context.Background()), tv, callback)
		assert.NoError(t, err)

		expected := TargetType{
			Name: new(string),
			Map: map[string]NestedPtr{
				"key1": {new(string)},
				"key2": {new(string)},
			},
		}

		*expected.Name = "Oleg"
		*expected.Map["key1"].Name2 = "Skinderev"
		*expected.Map["key2"].Name2 = "Skinderev"

		assert.Equal(t, expected, target)
	})

	t.Run("slice_nested_struct", func(t *testing.T) {
		type TargetType struct {
			Name  string `test:"Oleg"`
			Slice []Nested
		}
		var target TargetType

		target.Slice = make([]Nested, 2)
		tv := reflect.ValueOf(&target)

		err := TraverseStruct(newTestTraverseContext(context.Background()), tv, callback)
		assert.NoError(t, err)

		expected := TargetType{
			Name: "Oleg",
		}
		expected.Slice = []Nested{{"Skinderev"}, {"Skinderev"}}
		assert.Equal(t, expected, target)
	})

	t.Run("slice_pointer_nested_struct", func(t *testing.T) {
		type TargetType struct {
			Name  string `test:"Oleg"`
			Slice []*Nested
		}
		var target TargetType

		target.Slice = make([]*Nested, 2)
		tv := reflect.ValueOf(&target)

		err := TraverseStruct(newTestTraverseContext(context.Background()), tv, callback)
		assert.NoError(t, err)

		expected := TargetType{
			Name: "Oleg",
		}
		expected.Slice = []*Nested{{"Skinderev"}, {"Skinderev"}}
		assert.Equal(t, expected, target)
	})

	t.Run("slice_nested_struct_with_pointer", func(t *testing.T) {
		type TargetType struct {
			Name  *string `test:"Oleg"`
			Slice []NestedPtr
		}
		var target TargetType

		target.Slice = make([]NestedPtr, 2)
		tv := reflect.ValueOf(&target)

		err := TraverseStruct(newTestTraverseContext(context.Background()), tv, callback)
		assert.NoError(t, err)

		expected := TargetType{
			Name: new(string),
			Slice: []NestedPtr{
				{new(string)},
				{new(string)},
			},
		}

		*expected.Name = "Oleg"
		*expected.Slice[0].Name2 = "Skinderev"
		*expected.Slice[1].Name2 = "Skinderev"
		assert.Equal(t, expected, target)
	})

	t.Run("pointer_slice_nested_struct", func(t *testing.T) {
		type TargetType struct {
			Name  string `test:"Oleg"`
			Slice *[]Nested
		}
		var target TargetType

		target.Slice = new([]Nested)
		*target.Slice = make([]Nested, 2)
		tv := reflect.ValueOf(&target)

		err := TraverseStruct(newTestTraverseContext(context.Background()), tv, callback)
		assert.NoError(t, err)

		expected := TargetType{
			Name: "Oleg",
		}
		expected.Slice = new([]Nested)
		*expected.Slice = []Nested{{"Skinderev"}, {"Skinderev"}}
		assert.Equal(t, expected, target)
	})

	t.Run("default_value_pointer", func(t *testing.T) {
		type TargetType struct {
			FirstName *string `test:""`
			LastName  *string `test:""`
		}
		var target TargetType

		tv := reflect.ValueOf(&target)

		err := TraverseStruct(newTestTraverseContext(context.Background()), tv, callback)
		assert.NoError(t, err)

		expected := TargetType{
			FirstName: new(string),
			LastName:  new(string),
		}
		assert.Equal(t, expected, target)

	})

	t.Run("nested_struct_double_pointer", func(t *testing.T) {
		type TargetType struct {
			Name   *string `test:"Oleg"`
			Nested **NestedPtr
		}
		var target TargetType

		tv := reflect.ValueOf(&target)

		err := TraverseStruct(newTestTraverseContext(context.Background()), tv, callback)
		assert.NoError(t, err)

		expected := TargetType{
			Name:   new(string),
			Nested: new(*NestedPtr),
		}
		*expected.Name = "Oleg"
		*expected.Nested = new(NestedPtr)
		(*expected.Nested).Name2 = new(string)
		*(*expected.Nested).Name2 = "Skinderev"
		assert.Equal(t, expected, target)
	})

	t.Run("nested_struct_double_pointer_empty", func(t *testing.T) {
		type TargetType struct {
			Name   *string `test:"Oleg"`
			Nested **Empty
		}
		var target TargetType

		tv := reflect.ValueOf(&target)

		err := TraverseStruct(newTestTraverseContext(context.Background()), tv, callback)
		assert.NoError(t, err)

		expected := TargetType{
			Name:   new(string),
			Nested: nil,
		}
		*expected.Name = "Oleg"
		assert.Equal(t, expected, target)
	})

}

func TestNodeFieldPathMetadata(t *testing.T) {
	type item struct {
		Value string `test:"path"`
	}
	type nested struct {
		Items []*item
	}
	type group struct {
		Nested *nested
	}
	type config struct {
		Direct string `test:"path"`
		First  []group
		Second []group
		Mapped map[string][]*item
	}

	cfg := config{
		First: []group{
			{Nested: &nested{Items: []*item{{Value: "first-one"}}}},
			{Nested: &nested{Items: []*item{{Value: "first-two"}}}},
		},
		Second: []group{{Nested: &nested{Items: []*item{{Value: "second"}}}}},
		Mapped: map[string][]*item{
			"one": {{Value: "mapped-one"}},
			"two": {{Value: "mapped-two"}},
		},
	}
	t.Run("absent_by_default", func(t *testing.T) {
		paths, err := collectTestPaths(t, &cfg)

		assert.NoError(t, err)
		for _, path := range paths {
			assert.False(t, path.IsCollectionDescendant())
			assert.True(t, path.FieldPath().IsZero())
		}
	})

	t.Run("present_when_requested", func(t *testing.T) {
		paths, err := collectTestPaths(t, &cfg, WithFieldPathMetadata())

		assert.NoError(t, err)
		assert.False(t, paths[""].IsCollectionDescendant())
		assert.True(t, paths["first-one"].IsCollectionDescendant())
		assert.True(t, paths["first-two"].IsCollectionDescendant())
		assert.True(t, paths["second"].IsCollectionDescendant())
		assert.True(t, paths["mapped-one"].IsCollectionDescendant())
		assert.True(t, paths["mapped-two"].IsCollectionDescendant())
		assert.False(t, paths[""].FieldPath().IsZero())
		assert.True(t, paths["first-one"].FieldPath().Equal(paths["first-two"].FieldPath()))
		assert.False(t, paths["first-one"].FieldPath().Equal(paths["second"].FieldPath()))
		assert.True(t, paths["mapped-one"].FieldPath().Equal(paths["mapped-two"].FieldPath()))
		assert.False(t, paths["first-one"].FieldPath().Equal(paths["mapped-one"].FieldPath()))
	})
}

func collectTestPaths(t *testing.T, target any, opts ...TraverseOption) (map[string]Node, error) {
	t.Helper()
	paths := make(map[string]Node)
	err := TraverseStruct(
		NewTraverseContext(t.Context(), []Node{}, opts...),
		reflect.ValueOf(target),
		func(_ *testTraverseContext, field Node) error {
			if field.HasStructField && field.StructField.Tag.Get("test") == "path" {
				paths[field.Value.String()] = field
			}
			return nil
		},
	)
	return paths, err
}

func TestTraverseDeferredMapCommit(t *testing.T) {
	t.Run("multiple_values_commit_independently", func(t *testing.T) {
		type item struct {
			Value string `test:"deferred"`
		}
		type config struct {
			Items map[string]item
		}

		cfg := config{Items: map[string]item{
			"first":  {Value: "first"},
			"second": {Value: "second"},
		}}
		deferred := make(map[string]Node)

		err := TraverseStruct(newTestTraverseContext(t.Context()), reflect.ValueOf(&cfg), func(_ *testTraverseContext, field Node) error {
			if field.StructField.Tag.Get("test") == "deferred" {
				deferred[field.Value.String()] = field
			}
			return nil
		})

		assert.NoError(t, err)
		deferred["first"].Value.SetString("")
		deferred["first"].Commit()
		assert.Equal(t, "", cfg.Items["first"].Value)
		assert.Equal(t, "second", cfg.Items["second"].Value)

		deferred["second"].Value.SetString("loaded")
		deferred["second"].Commit()
		deferred["second"].Value.SetString("reloaded")
		deferred["second"].Commit()
		assert.Equal(t, "reloaded", cfg.Items["second"].Value)
	})

	t.Run("struct_value", func(t *testing.T) {
		type item struct {
			Value string `test:"deferred"`
		}
		type config struct {
			Items map[string]item
		}

		cfg := config{Items: map[string]item{"item": {Value: "original"}}}
		var deferred Node

		err := TraverseStruct(newTestTraverseContext(t.Context()), reflect.ValueOf(&cfg), func(_ *testTraverseContext, field Node) error {
			if field.StructField.Tag.Get("test") == "deferred" {
				deferred = field
			}
			return nil
		})

		assert.NoError(t, err)
		deferred.Value.SetString("loaded")
		deferred.Commit()
		assert.Equal(t, "loaded", cfg.Items["item"].Value)
	})

	t.Run("nil_pointer_value", func(t *testing.T) {
		type item struct {
			Value string `test:"deferred"`
		}
		type config struct {
			Items map[string]*item
		}

		cfg := config{Items: map[string]*item{"item": nil}}
		var deferred Node

		err := TraverseStruct(newTestTraverseContext(t.Context()), reflect.ValueOf(&cfg), func(_ *testTraverseContext, field Node) error {
			if field.StructField.Tag.Get("test") == "deferred" {
				deferred = field
			}
			return nil
		})

		assert.NoError(t, err)
		deferred.Value.SetString("loaded")
		deferred.Commit()
		if assert.NotNil(t, cfg.Items["item"]) {
			assert.Equal(t, "loaded", cfg.Items["item"].Value)
		}
	})

	t.Run("multiple_pointer_levels", func(t *testing.T) {
		type item struct {
			Value string `test:"deferred"`
		}
		type config struct {
			Items map[string]***item
		}

		cfg := config{Items: map[string]***item{"item": nil}}
		var deferred Node

		err := TraverseStruct(newTestTraverseContext(t.Context()), reflect.ValueOf(&cfg), func(_ *testTraverseContext, field Node) error {
			if field.StructField.Tag.Get("test") == "deferred" {
				deferred = field
			}
			return nil
		})

		assert.NoError(t, err)
		deferred.Value.SetString("loaded")
		deferred.Commit()
		if assert.NotNil(t, cfg.Items["item"]) {
			if assert.NotNil(t, *cfg.Items["item"]) {
				if assert.NotNil(t, **cfg.Items["item"]) {
					assert.Equal(t, "loaded", (***cfg.Items["item"]).Value)
				}
			}
		}
	})

	t.Run("nested_map_below_pointer", func(t *testing.T) {
		type item struct {
			Value string `test:"deferred"`
		}
		type group struct {
			Items map[string]item
		}
		type config struct {
			Groups map[string]*group
		}

		cfg := config{Groups: map[string]*group{
			"group": {Items: map[string]item{"item": {Value: "original"}}},
		}}
		var deferred Node

		err := TraverseStruct(newTestTraverseContext(t.Context()), reflect.ValueOf(&cfg), func(_ *testTraverseContext, field Node) error {
			if field.StructField.Tag.Get("test") == "deferred" {
				deferred = field
			}
			return nil
		})

		assert.NoError(t, err)
		deferred.Value.SetString("loaded")
		deferred.Commit()
		assert.Equal(t, "loaded", cfg.Groups["group"].Items["item"].Value)
	})
}
