package reflect

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

var ErrSkipNested = errors.New("skip nested")

type Node struct {
	Value reflect.Value

	HasStructField bool
	StructField    reflect.StructField

	commit               func()
	collectionDescendant bool
	fieldPath            FieldPath
}

// FieldPath identifies a declared struct field route.
type FieldPath struct {
	steps []fieldPathStep
}

type fieldPathStep struct {
	structType reflect.Type
	fieldIndex int
}

// Equal reports if two paths identify the same declared struct field route.
func (p FieldPath) Equal(other FieldPath) bool {
	if len(p.steps) != len(other.steps) {
		return false
	}
	for i := range p.steps {
		if p.steps[i] != other.steps[i] {
			return false
		}
	}
	return true
}

// IsZero reports if the traversal did not collect this field path.
func (p FieldPath) IsZero() bool {
	return len(p.steps) == 0
}

func (p FieldPath) append(structType reflect.Type, fieldIndex int) FieldPath {
	steps := make([]fieldPathStep, len(p.steps)+1)
	copy(steps, p.steps)
	steps[len(p.steps)] = fieldPathStep{structType: structType, fieldIndex: fieldIndex}
	return FieldPath{steps: steps}
}

func (n *Node) String() string {
	builder := strings.Builder{}
	builder.WriteString(fmt.Sprintf("{Value: %+v", n.Value.String()))
	builder.WriteString(fmt.Sprintf(", HasStructField: %t", n.HasStructField))
	if n.HasStructField {
		builder.WriteString(fmt.Sprintf(", StructField: %+v", n.StructField))
	}
	builder.WriteString("}")
	return builder.String()
}

func (n *Node) Elem() Node {
	return Node{
		Value:                n.Value.Elem(),
		HasStructField:       n.HasStructField,
		StructField:          n.StructField,
		commit:               n.commit,
		collectionDescendant: n.collectionDescendant,
		fieldPath:            n.fieldPath,
	}
}

// IsCollectionDescendant reports if the node descends from a collection.
func (n Node) IsCollectionDescendant() bool {
	return n.collectionDescendant
}

// FieldPath reports the declared struct field route for the node.
func (n Node) FieldPath() FieldPath {
	return n.fieldPath
}

// Commit saves deferred pointer values for this node.
func (n Node) Commit() {
	if n.commit != nil {
		n.commit()
	}
}

// HasDeferredCommit reports if the node contains a deferred pointer value.
func (n Node) HasDeferredCommit() bool {
	return n.commit != nil
}

func StructFieldNode(value reflect.Value, field reflect.StructField) Node {
	return Node{
		Value:          value,
		HasStructField: true,
		StructField:    field,
	}
}

type CallbackFunc[T any] func(*TraverseContext[T], Node) error

type TraverseContext[T any] struct {
	Data T

	ctx               context.Context
	collectFieldPaths bool
}

// TraverseOption configures a traversal context.
type TraverseOption func(*traverseOptions)

type traverseOptions struct {
	collectFieldPaths bool
}

// WithFieldPathMetadata collects collection ancestry and declared field paths.
func WithFieldPathMetadata() TraverseOption {
	return func(options *traverseOptions) {
		options.collectFieldPaths = true
	}
}

func NewTraverseContext[T any](ctx context.Context, data T, opts ...TraverseOption) TraverseContext[T] {
	options := traverseOptions{}
	for _, opt := range opts {
		opt(&options)
	}
	return TraverseContext[T]{
		Data:              data,
		collectFieldPaths: options.collectFieldPaths,
		ctx:               ctx,
	}
}

func (t *TraverseContext[T]) Context() context.Context {
	return t.ctx
}

func TraverseStruct[T any](tctx TraverseContext[T], target reflect.Value, callback CallbackFunc[T]) error {
	if !target.IsValid() || target.Kind() != reflect.Ptr || target.Elem().Kind() != reflect.Struct {
		return errors.New("target must be a pointer to struct")
	}
	return Traverse(tctx, Node{Value: target}, callback)
}

func getUnwrappedType(target Node) reflect.Type {
	t := target.Value.Type()
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

func Traverse[T any](tctx TraverseContext[T], target Node, callback CallbackFunc[T]) (err error) {
	if !target.Value.IsValid() {
		return errors.New("invalid reflect.Value")
	}
	switch getUnwrappedType(target).Kind() {
	case reflect.Array, reflect.Slice:
		err = processSlice(tctx, target, callback)
	case reflect.Struct:
		err = processStruct(tctx, target, callback)
	case reflect.Map:
		err = processMap(tctx, target, callback)
	default:
		err = callback(&tctx, target)
	}
	if errors.Is(err, ErrSkipNested) {
		return nil
	}
	return err
}

func processSlice[T any](tctx TraverseContext[T], target Node, callback CallbackFunc[T]) error {
	err := callback(&tctx, target)
	if err != nil {
		return err
	}
	newTarget := recursiveElem(target)
	if !newTarget.Value.IsValid() {
		return nil
	}
	for i := 0; i < newTarget.Value.Len(); i++ {
		node := Node{Value: newTarget.Value.Index(i), commit: newTarget.commit}
		if tctx.collectFieldPaths {
			node.collectionDescendant = true
			node.fieldPath = newTarget.fieldPath
		}
		err = Traverse(tctx, node, callback)
		if err != nil {
			return err
		}
	}
	return nil
}

func processMap[T any](tctx TraverseContext[T], target Node, callback CallbackFunc[T]) error {
	err := callback(&tctx, target)
	if err != nil {
		return err
	}
	newTarget := recursiveElem(target)
	if !newTarget.Value.IsValid() {
		return nil
	}
	for _, key := range newTarget.Value.MapKeys() {
		mapValue := newTarget.Value.MapIndex(key)
		// make new value because mapValue is unaddressable
		newMapValue := reflect.New(mapValue.Type()).Elem()
		newMapValue.Set(mapValue)
		commit := func() {
			newTarget.Value.SetMapIndex(key, newMapValue)
			newTarget.Commit()
		}
		node := Node{Value: newMapValue, commit: commit}
		if tctx.collectFieldPaths {
			node.collectionDescendant = true
			node.fieldPath = newTarget.fieldPath
		}
		err = Traverse(tctx, node, callback)
		if err != nil {
			return err
		}
		newTarget.Value.SetMapIndex(key, newMapValue)
	}
	return nil
}

func processStruct[T any](tctx TraverseContext[T], target Node, callback CallbackFunc[T]) error {
	err := callback(&tctx, target)
	if err != nil {
		return err
	}
	s := recursiveElemWithNew(target)
	for i := 0; i < s.Node.Value.NumField(); i++ {
		field := StructFieldNode(s.Node.Value.Field(i), s.Node.Value.Type().Field(i))
		field.commit = s.commitFunc()
		if tctx.collectFieldPaths {
			field.collectionDescendant = s.Node.collectionDescendant
			field.fieldPath = s.Node.fieldPath.append(s.Node.Value.Type(), i)
		}
		err = Traverse(tctx, field, callback)
		if err != nil {
			return err
		}
	}
	s.setIsNotZero()
	return nil
}

func recursiveElem(node Node) Node {
	val := node.Value
	for val.Kind() == reflect.Ptr || val.Kind() == reflect.Interface {
		val = val.Elem()
	}
	return Node{
		Value:                val,
		HasStructField:       node.HasStructField,
		StructField:          node.StructField,
		commit:               node.commit,
		collectionDescendant: node.collectionDescendant,
		fieldPath:            node.fieldPath,
	}
}

type setter struct {
	Node        Node
	callNew     bool
	pointToSet  reflect.Value
	valueForSet reflect.Value
}

func (s *setter) commitFunc() func() {
	if !s.callNew && !s.Node.HasDeferredCommit() {
		return nil
	}
	return s.commit
}

func (s *setter) commit() {
	if s.callNew {
		s.pointToSet.Set(s.valueForSet)
	}
	s.Node.Commit()
}

func (s *setter) setIsNotZero() {
	if !s.callNew || s.Node.Value.IsZero() {
		return
	}
	s.commit()
}

func recursiveElemWithNew(node Node) setter {
	var s setter
	val := node.Value
	for val.Kind() == reflect.Ptr {
		// set zero value of proper type
		if val.IsNil() {
			if s.callNew {
				val.Set(reflect.New(val.Type().Elem()))
			} else {
				s.callNew = true
				s.pointToSet = val
				val = reflect.New(val.Type().Elem())
				s.valueForSet = val
			}
		}
		// unwrap value
		val = val.Elem()
	}
	s.Node = Node{
		Value:                val,
		HasStructField:       node.HasStructField,
		StructField:          node.StructField,
		commit:               node.commit,
		collectionDescendant: node.collectionDescendant,
		fieldPath:            node.fieldPath,
	}
	return s
}
