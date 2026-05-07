package flags

import (
	"database/sql"
	"encoding"
	"errors"
	"flag"
	"fmt"
	"reflect"

	ref "golang.yandex/confetti/internal/reflect"
)

// ErrUnsupported reports that the target does not support a flag value.
var ErrUnsupported = errors.New("flag value not set")

// flagScanner is a wrapper for custom config scanner to work with flags
type flagScanner struct {
	target reflect.Value
	commit func()
}

type getterFlagScanner struct {
	flagScanner
}

func newFlagScanner(target reflect.Value, commit func()) flag.Value {
	scanner := flagScanner{target: target, commit: commit}
	terminalPointerType := reflect.PointerTo(terminalType(target.Type()))
	if terminalPointerType.Implements(reflect.TypeFor[flag.Getter]()) {
		return getterFlagScanner{flagScanner: scanner}
	}
	return scanner
}

func (f getterFlagScanner) Get() any {
	value, ok := pointerTerminal(f.target)
	if !ok {
		return reflect.New(terminalType(f.target.Type())).Interface().(flag.Getter).Get()
	}
	return value.Addr().Interface().(flag.Getter).Get()
}

func (f flagScanner) String() string {
	if value, ok := pointerTerminalFlagValue(f.target); ok {
		return value.String()
	}

	value := unwrapValue(f.target)
	if value.IsValid() {
		return fmt.Sprint(value.Interface())
	}
	if !f.target.IsValid() {
		return ""
	}
	return fmt.Sprint(reflect.Zero(terminalType(f.target.Type())).Interface())
}

func pointerTerminalFlagValue(target reflect.Value) (flag.Value, bool) {
	if !target.IsValid() {
		return nil, false
	}

	terminalType := terminalType(target.Type())
	if !reflect.PointerTo(terminalType).Implements(reflect.TypeFor[flag.Value]()) {
		return nil, false
	}

	terminal, ok := pointerTerminal(target)
	if !ok {
		return reflect.TypeAssert[flag.Value](reflect.New(terminalType))
	}
	if !terminal.CanAddr() {
		return nil, false
	}
	return reflect.TypeAssert[flag.Value](terminal.Addr())
}

func (f flagScanner) IsBoolFlag() bool {
	target, ok := pointerTerminal(f.target)
	if !ok {
		target = reflect.New(terminalType(f.target.Type())).Elem()
	}
	return isBoolFlagValue(target)
}

func (f flagScanner) Set(val string) error {
	value := reflect.New(terminalType(f.target.Type())).Elem()
	if current, ok := pointerTerminal(f.target); ok {
		value.Set(current)
	}

	err := decodeFlagValue(value, val)
	if errors.Is(err, ErrUnsupported) {
		return nil
	}
	if err != nil {
		return err
	}

	setPointerTarget(f.target, value)
	f.commitValue()
	return nil
}

func allocatePointer(target reflect.Value) reflect.Value {
	for target.Kind() == reflect.Ptr {
		if target.IsNil() {
			target.Set(reflect.New(target.Type().Elem()))
		}
		target = target.Elem()
	}
	return target
}

func setPointerTarget(target, value reflect.Value) {
	allocatePointer(target).Set(value)
}

func pointerTerminal(target reflect.Value) (reflect.Value, bool) {
	for target.Kind() == reflect.Ptr {
		if target.IsNil() {
			return reflect.Value{}, false
		}
		target = target.Elem()
	}
	return target, true
}

func terminalType(target reflect.Type) reflect.Type {
	for target.Kind() == reflect.Ptr {
		target = target.Elem()
	}
	return target
}

// get pointer to value to check interfaces implementation properly
func decodeFlagValue(target reflect.Value, val string) error {
	valuePtr := target.Addr().Interface()

	switch value := valuePtr.(type) {
	case flag.Value:
		return value.Set(val)
	case sql.Scanner:
		return value.Scan(val)
	case encoding.TextUnmarshaler:
		return value.UnmarshalText([]byte(val))
	case encoding.BinaryUnmarshaler:
		return value.UnmarshalBinary([]byte(val))
	}

	switch target.Kind() {
	case reflect.String,
		reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int,
		reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint,
		reflect.Float32, reflect.Float64,
		reflect.Bool,
		reflect.Complex64, reflect.Complex128:
		return ref.SetValue(target, val)
	default:
		return ErrUnsupported
	}
}

func valueUsesCustomFlagDecoder(value any) bool {
	switch value.(type) {
	case flag.Value, sql.Scanner, encoding.TextUnmarshaler, encoding.BinaryUnmarshaler:
		return true
	default:
		return false
	}
}

func isBoolFlagValue(target reflect.Value) bool {
	value := target.Addr().Interface()
	if valueUsesCustomFlagDecoder(value) {
		boolFlag, ok := value.(interface{ IsBoolFlag() bool })
		return ok && boolFlag.IsBoolFlag()
	}
	return target.Kind() == reflect.Bool
}

func (f flagScanner) commitValue() {
	if f.commit != nil {
		f.commit()
	}
}

type deferredFlagValue struct {
	value  flag.Value
	commit func()
}

type getterDeferredFlagValue struct {
	deferredFlagValue
	getter flag.Getter
}

type collectionFlagValue struct {
	values []flag.Value
}

type getterCollectionFlagValue struct {
	collectionFlagValue
	getter flag.Getter
}

type collectionValues interface {
	collectionFlagValues() []flag.Value
}

func newDeferredFlagValue(value flag.Value, commit func()) flag.Value {
	deferred := deferredFlagValue{value: value, commit: commit}
	if getter, ok := value.(flag.Getter); ok {
		return getterDeferredFlagValue{deferredFlagValue: deferred, getter: getter}
	}
	return deferred
}

func newCollectionFlagValue(values []flag.Value) flag.Value {
	collection := collectionFlagValue{values: values}
	if getter, ok := collectionGetter(values); ok {
		return getterCollectionFlagValue{collectionFlagValue: collection, getter: getter}
	}
	return collection
}

func collectionGetter(values []flag.Value) (flag.Getter, bool) {
	if len(values) == 0 {
		return nil, false
	}
	getter, ok := values[0].(flag.Getter)
	if !ok {
		return nil, false
	}
	for _, value := range values[1:] {
		if _, ok := value.(flag.Getter); !ok {
			return nil, false
		}
	}
	return getter, true
}

func collectionFlagValues(value flag.Value) ([]flag.Value, bool) {
	collection, ok := value.(collectionValues)
	if !ok {
		return nil, false
	}
	return collection.collectionFlagValues(), true
}

func (v collectionFlagValue) collectionFlagValues() []flag.Value {
	return v.values
}

func (v collectionFlagValue) String() string {
	if len(v.values) == 0 {
		return ""
	}
	return v.values[0].String()
}

func (v collectionFlagValue) Set(value string) error {
	for _, target := range v.values {
		if err := target.Set(value); err != nil {
			return err
		}
	}
	return nil
}

func (v collectionFlagValue) IsBoolFlag() bool {
	if len(v.values) == 0 {
		return false
	}
	for _, value := range v.values {
		boolFlag, ok := value.(interface{ IsBoolFlag() bool })
		if !ok || !boolFlag.IsBoolFlag() {
			return false
		}
	}
	return true
}

func (v getterCollectionFlagValue) Get() any {
	return v.getter.Get()
}

func (f deferredFlagValue) String() string {
	return f.value.String()
}

func (f deferredFlagValue) Set(val string) error {
	if err := f.value.Set(val); err != nil {
		return err
	}
	if f.commit != nil {
		f.commit()
	}
	return nil
}

func (f deferredFlagValue) IsBoolFlag() bool {
	boolFlag, ok := f.value.(interface{ IsBoolFlag() bool })
	return ok && boolFlag.IsBoolFlag()
}

func (f getterDeferredFlagValue) Get() any {
	return f.getter.Get()
}

func unwrapValue(val reflect.Value) reflect.Value {
	for val.Kind() == reflect.Ptr || val.Kind() == reflect.Interface {
		val = val.Elem()
	}
	return val
}
