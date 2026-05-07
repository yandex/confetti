// Package pem provides methods to load various cryptographic data to custom Go cryptographic primitives
package pem

import (
	"context"
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
)

// PrivateKey represents a private key using an unspecified algorithm.
// All private key types in the standard library implement the following interface.
type PrivateKey interface {
	Equal(crypto.PrivateKey) bool
}

// PublicKey represents a public key using an unspecified algorithm.
// All public key types in the standard library implement the following interface.
type PublicKey interface {
	Equal(crypto.PublicKey) bool
}

// types for reflection checks
var (
	privateKeyType     = reflect.TypeFor[PrivateKey]()
	publicKeyType      = reflect.TypeFor[PublicKey]()
	certificateType    = reflect.TypeFor[x509.Certificate]()
	certificatePtrType = reflect.TypeFor[*x509.Certificate]()
)

// FromString loads value from PEM string to given target
func FromString(s string) func(context.Context, any) error {
	return func(ctx context.Context, target any) error {
		return FromReader(strings.NewReader(s))(ctx, target)
	}
}

// FromFile loads values from file to given target
func FromFile(path string) func(context.Context, any) error {
	return func(ctx context.Context, target any) error {
		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("cannot open file: %w", err)
		}
		defer f.Close()
		return FromReader(f)(ctx, target)
	}
}

// FromReader loads values from reader to given target
func FromReader(r io.Reader) func(context.Context, any) error {
	return func(ctx context.Context, target any) error {
		if _, err := targetType(target); err != nil {
			return err
		}

		b, err := io.ReadAll(r)
		if err != nil {
			return fmt.Errorf("cannot read PEM bytes: %w", err)
		}

		var block *pem.Block
		var blocks []*pem.Block
		for len(b) > 0 {
			block, b = pem.Decode(b)
			if block == nil {
				break
			}
			blocks = append(blocks, block)
		}

		return FromBlocks(blocks...)(ctx, target)
	}
}

// FromBlocks loads values from given PEM blocks to given target
func FromBlocks(blocks ...*pem.Block) func(context.Context, any) error {
	pemChain := newChain(blocks...)

	return func(ctx context.Context, target any) error {
		if len(blocks) == 0 {
			return nil
		}

		typ, err := targetType(target)
		if err != nil {
			return err
		}

		if len(pemChain.Blocks()) == 0 {
			return nil
		}

		// check target's crypto type
		pemtype := pemType(typ)

		// process crypto compatible type
		if pemtype != nil {
			// process single value target
			typeBlocks := getBlocksByType(pemChain, pemtype)
			if len(typeBlocks) == 0 {
				return nil
			}
			// try set first compatible block to target
			value, err := decodeValue(typeBlocks[0], typ)
			if err != nil {
				return err
			}
			setDestination(reflect.ValueOf(target), value)
			return nil
		}

		if typ.Kind() == reflect.Slice {
			value, changed, err := decodeSlice(typ, pemChain)
			if err != nil {
				return err
			}
			if changed {
				setDestination(reflect.ValueOf(target), value)
			}
		}

		if typ.Kind() == reflect.Struct {
			assignments, err := decodeStruct(typ, pemChain)
			if err != nil {
				return err
			}
			if len(assignments) > 0 {
				assignments, err = prepareStructAssignments(reflect.ValueOf(target), assignments)
				if err != nil {
					return err
				}
				setStructFields(reflect.ValueOf(target), assignments)
			}
		}

		return nil
	}
}

func targetType(target any) (reflect.Type, error) {
	value := reflect.ValueOf(target)
	if !value.IsValid() || value.Kind() != reflect.Ptr || value.IsNil() {
		return nil, errors.New("target must be a non-nil pointer")
	}

	for {
		switch value.Kind() {
		case reflect.Ptr:
			if value.IsNil() {
				if !value.CanSet() {
					return nil, errors.New("target must be a settable pointer")
				}
				return indirectType(value.Type()), nil
			}
			value = value.Elem()
		case reflect.Interface:
			if value.IsNil() {
				return value.Type(), nil
			}

			elem := value.Elem()
			if elem.Kind() != reflect.Ptr || elem.IsNil() {
				return elem.Type(), nil
			}
			value = elem
		default:
			if !value.CanSet() {
				return nil, errors.New("target must be a settable pointer")
			}
			return value.Type(), nil
		}
	}
}

func indirectType(typ reflect.Type) reflect.Type {
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	return typ
}

func decodeSlice(sliceType reflect.Type, pemChain chain) (reflect.Value, bool, error) {
	elemType := sliceType.Elem()

	// encode PEM chain to slice of bytes
	if elemType.Kind() == reflect.Uint8 {
		encoded, err := pemChain.MarshalBinary()
		if err != nil {
			return reflect.Value{}, false, fmt.Errorf("cannot encode PEM chain bytes: %w", err)
		}
		value := reflect.New(sliceType).Elem()
		value.SetBytes(encoded)
		return value, true, nil
	}

	// check target's crypto type
	pemtype := pemType(elemType)
	if pemtype == nil {
		return reflect.Value{}, false, nil
	}

	typeBlocks := getBlocksByType(pemChain, pemtype)
	if len(typeBlocks) == 0 {
		return reflect.Value{}, false, nil
	}

	blocksCount := len(typeBlocks)
	target := reflect.MakeSlice(sliceType, blocksCount, blocksCount)
	for i, block := range typeBlocks {
		value, err := decodeValue(block, elemType)
		if err != nil {
			return reflect.Value{}, false, fmt.Errorf("cannot set PEM value to slice at index %d: %w", i, err)
		}
		target.Index(i).Set(value)
	}

	return target, true, nil
}

type fieldAssignment struct {
	index []int
	path  string
	value reflect.Value
}

func decodeStruct(structType reflect.Type, pemChain chain) ([]fieldAssignment, error) {
	ancestors := map[reflect.Type]bool{structType: true}
	return collectStructAssignments(structType, nil, nil, pemChain, ancestors)
}

func collectStructAssignments(
	structType reflect.Type,
	index []int,
	path []string,
	pemChain chain,
	ancestors map[reflect.Type]bool,
) ([]fieldAssignment, error) {
	var assignments []fieldAssignment
	for i := range structType.NumField() {
		field := structType.Field(i)
		if !field.IsExported() {
			continue
		}

		fieldIndex := appendIndex(index, field.Index)
		fieldPath := append(path, field.Name)
		fieldAssignments, err := decodeStructField(field.Type, fieldIndex, fieldPath, pemChain, ancestors)
		if err != nil {
			return nil, err
		}
		assignments = append(assignments, fieldAssignments...)
	}
	return assignments, nil
}

func decodeStructField(
	fieldType reflect.Type,
	index []int,
	path []string,
	pemChain chain,
	ancestors map[reflect.Type]bool,
) ([]fieldAssignment, error) {
	value, changed, err := decodeFieldValue(fieldType, pemChain)
	if err != nil {
		return nil, fmt.Errorf("cannot decode PEM field %s: %w", strings.Join(path, "."), err)
	}
	if changed {
		return []fieldAssignment{{index: index, path: strings.Join(path, "."), value: value}}, nil
	}

	nestedType, ok := nestedStructType(fieldType)
	if !ok || ancestors[nestedType] {
		return nil, nil
	}

	ancestors[nestedType] = true
	assignments, err := collectStructAssignments(nestedType, index, path, pemChain, ancestors)
	delete(ancestors, nestedType)
	return assignments, err
}

func decodeFieldValue(fieldType reflect.Type, pemChain chain) (reflect.Value, bool, error) {
	targetType := indirectType(fieldType)
	pemtype := pemType(targetType)
	if pemtype != nil {
		return decodeDirectField(targetType, pemtype, pemChain)
	}

	if targetType.Kind() != reflect.Slice {
		return reflect.Value{}, false, nil
	}
	return decodeSlice(targetType, pemChain)
}

func decodeDirectField(fieldType, pemtype reflect.Type, pemChain chain) (reflect.Value, bool, error) {
	typeBlocks := getBlocksByType(pemChain, pemtype)
	if len(typeBlocks) == 0 {
		return reflect.Value{}, false, nil
	}

	value, err := decodeValue(typeBlocks[0], fieldType)
	if err != nil {
		return reflect.Value{}, false, err
	}
	for _, block := range typeBlocks[1:] {
		if _, err := parseValue(block); err != nil {
			return reflect.Value{}, false, err
		}
	}
	return value, true, nil
}

func nestedStructType(typ reflect.Type) (reflect.Type, bool) {
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return nil, false
	}
	return typ, true
}

func appendIndex(index, fieldIndex []int) []int {
	result := make([]int, 0, len(index)+len(fieldIndex))
	result = append(result, index...)
	return append(result, fieldIndex...)
}

func prepareStructAssignments(target reflect.Value, assignments []fieldAssignment) ([]fieldAssignment, error) {
	prepared := make([]fieldAssignment, len(assignments))
	copy(prepared, assignments)

	for i := range prepared {
		field, ok := structField(target, prepared[i].index)
		if !ok || field.Kind() != reflect.Interface || field.IsNil() {
			continue
		}

		value := field.Elem()
		if value.Kind() == reflect.Ptr && value.IsNil() {
			continue
		}

		fieldType := value.Type()
		if value.Kind() == reflect.Ptr {
			fieldType = indirectType(fieldType)
		}
		converted, ok := valueForType(prepared[i].value, fieldType)
		if !ok {
			return nil, fmt.Errorf("cannot decode PEM field %s: cannot set PEM value to %s", prepared[i].path, fieldType)
		}
		prepared[i].value = converted
	}

	return prepared, nil
}

func structField(target reflect.Value, index []int) (reflect.Value, bool) {
	target, ok := structTarget(target)
	if !ok {
		return reflect.Value{}, false
	}

	for _, fieldIndex := range index[:len(index)-1] {
		if target.Kind() != reflect.Struct {
			return reflect.Value{}, false
		}
		target = target.Field(fieldIndex)
		for target.Kind() == reflect.Ptr {
			if target.IsNil() {
				return reflect.Value{}, false
			}
			target = target.Elem()
		}
	}
	if target.Kind() != reflect.Struct {
		return reflect.Value{}, false
	}
	return target.Field(index[len(index)-1]), true
}

func structTarget(target reflect.Value) (reflect.Value, bool) {
	for {
		switch target.Kind() {
		case reflect.Ptr:
			if target.IsNil() {
				return reflect.Value{}, false
			}
			target = target.Elem()
		case reflect.Interface:
			if target.IsNil() {
				return reflect.Value{}, false
			}
			target = target.Elem()
		default:
			return target, true
		}
	}
}

func setStructFields(target reflect.Value, assignments []fieldAssignment) {
	for {
		switch target.Kind() {
		case reflect.Ptr:
			if target.IsNil() {
				target.Set(reflect.New(target.Type().Elem()))
			}
			target = target.Elem()
		case reflect.Interface:
			value := target.Elem()
			if value.Kind() == reflect.Ptr {
				target = value
				continue
			}

			copy := reflect.New(value.Type()).Elem()
			copy.Set(value)
			setStructAssignments(copy, assignments)
			target.Set(copy)
			return
		default:
			setStructAssignments(target, assignments)
			return
		}
	}
}

func setStructAssignments(target reflect.Value, assignments []fieldAssignment) {
	for _, assignment := range assignments {
		setStructField(target, assignment)
	}
}

func setStructField(target reflect.Value, assignment fieldAssignment) {
	for _, index := range assignment.index[:len(assignment.index)-1] {
		target = target.Field(index)
		for target.Kind() == reflect.Ptr {
			if target.IsNil() {
				target.Set(reflect.New(target.Type().Elem()))
			}
			target = target.Elem()
		}
	}
	setDestination(target.Field(assignment.index[len(assignment.index)-1]), assignment.value)
}

func decodeValue(block *pem.Block, targetType reflect.Type) (reflect.Value, error) {
	value, err := parseValue(block)
	if err != nil {
		return reflect.Value{}, err
	}

	setValue, ok := valueForType(reflect.ValueOf(value), targetType)
	if !ok {
		return reflect.Value{}, fmt.Errorf("cannot set PEM value to %s", targetType)
	}
	return setValue, nil
}

func parseValue(block *pem.Block) (any, error) {
	var value any
	var err error

	switch block.Type {
	case "PRIVATE KEY":
		value, err = x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("cannot parse PKCS8 private key: %w", err)
		}
	case "RSA PRIVATE KEY":
		value, err = x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("cannot parse PKCS1 private key: %w", err)
		}
	case "EC PRIVATE KEY":
		value, err = x509.ParseECPrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("cannot parse EC private key: %w", err)
		}
	case "PUBLIC KEY", "EC PUBLIC KEY":
		value, err = x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("cannot parse PKIX public key: %w", err)
		}
	case "RSA PUBLIC KEY":
		value, err = x509.ParsePKCS1PublicKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("cannot parse PKCS1 public key: %w", err)
		}
	case "CERTIFICATE":
		value, err = x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("cannot parse PEM certificate: %w", err)
		}
	default:
		return nil, nil
	}

	return value, nil
}

func valueForType(value reflect.Value, targetType reflect.Type) (reflect.Value, bool) {
	for {
		if value.Type().AssignableTo(targetType) {
			return value, true
		}
		if value.Kind() != reflect.Ptr {
			return reflect.Value{}, false
		}
		value = value.Elem()
	}
}

func setDestination(target reflect.Value, value reflect.Value) {
	for {
		switch target.Kind() {
		case reflect.Ptr:
			if target.IsNil() {
				target.Set(reflect.New(target.Type().Elem()))
			}
			target = target.Elem()
		case reflect.Interface:
			if target.IsNil() {
				target.Set(value)
				return
			}

			elem := target.Elem()
			if elem.Kind() != reflect.Ptr || elem.IsNil() {
				target.Set(value)
				return
			}
			target = elem
		default:
			target.Set(value)
			return
		}
	}
}

// getBlocksByType returns appropriate PEM blocks for target type
func getBlocksByType(c chain, typ reflect.Type) []*pem.Block {
	switch typ {
	case privateKeyType:
		return c.Private()
	case publicKeyType:
		return c.Public()
	case certificateType:
		return c.Certificates()
	}
	return nil
}

// pemType converts reflect.Type to PEM type (private key, public key or certificate)
func pemType(typ reflect.Type) reflect.Type {
	typPtr := typ
	if typ.Kind() != reflect.Pointer {
		typPtr = reflect.PointerTo(typ)
	}

	if typ.Implements(privateKeyType) || typPtr.Implements(privateKeyType) {
		return privateKeyType
	}
	if typ.Implements(publicKeyType) || typPtr.Implements(publicKeyType) {
		return publicKeyType
	}
	if typ == certificateType || typPtr == certificatePtrType {
		return certificateType
	}

	return nil
}
