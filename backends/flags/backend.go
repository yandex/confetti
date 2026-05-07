package flags

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"reflect"
	"strings"

	refl "golang.yandex/confetti/internal/reflect"
)

// From loads value from a single flag to variable by given name
func From(name string, opts ...FlagOpt) func(context.Context, any) error {
	if len(os.Args) < 2 {
		return func(_ context.Context, _ any) error {
			return nil
		}
	}

	fo := flagOpts{
		name: name,
	}
	for _, opt := range opts {
		opt(&fo)
	}
	if fo.fset == nil {
		fset := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
		fset.Usage = func() { printHelp(fset) }
		fo.fset = fset
	}

	cmdArgs := os.Args[1:]

	return func(_ context.Context, target any) error {
		if err := validateFromTarget(target); err != nil {
			return err
		}

		definition := directFlagDefinition(fo.name, fo.usage, target)
		if err := registerFlagDefinitions(fo.fset, []flagDefinition{definition}); err != nil {
			return fmt.Errorf("cannot register flag %q: %w", fo.name, err)
		}
		args := extractRegistered(fo.fset, map[string]struct{}{fo.name: {}}, cmdArgs...)
		return fo.fset.Parse(args)
	}
}

func validateFromTarget(target any) error {
	value := reflect.ValueOf(target)
	if !value.IsValid() {
		return errors.New("flag target must be a non-nil pointer or flag.Value")
	}
	if value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return errors.New("flag target must be a non-nil pointer")
		}
		return nil
	}
	if _, ok := target.(flag.Value); ok {
		return nil
	}
	return errors.New("flag target must be a pointer or flag.Value")
}

// FromArgs loads values from os.Args to given struct target
func FromArgs(opts ...FlagSetOpt) func(context.Context, any) error {
	if len(os.Args) < 2 {
		return func(_ context.Context, _ any) error {
			return nil
		}
	}

	fo := flagSetOpts{
		name: os.Args[0],
		tag:  "flag",
	}
	for _, opt := range opts {
		opt(&fo)
	}

	if fo.fset == nil {
		fset := flag.NewFlagSet(fo.name, flag.ContinueOnError)
		fset.Usage = func() { printHelp(fset) }
		fo.fset = fset
	}

	return func(ctx context.Context, target any) error {
		definitions, err := collectFlagDefinitions(ctx, reflect.ValueOf(target), fo.tag)
		if err != nil {
			return fmt.Errorf("cannot collect flags from struct tags: %w", err)
		}
		if err := registerFlagDefinitions(fo.fset, definitions); err != nil {
			return fmt.Errorf("cannot register flags from struct tags: %w", err)
		}

		cmdArgs := os.Args[1:]

		args := extractRegistered(fo.fset, nil, cmdArgs...)

		if err := fo.fset.Parse(args); err != nil {
			return err
		}

		return nil
	}
}

type flagDefinition struct {
	name           string
	usage          string
	value          flag.Value
	collectionPath refl.FieldPath
}

func directFlagDefinition(name, usage string, target any) flagDefinition {
	if value, ok := target.(flag.Value); ok {
		return flagDefinition{name: name, usage: usage, value: value}
	}
	return flagDefinition{
		name:  name,
		usage: usage,
		value: newFlagScanner(reflect.ValueOf(target), nil),
	}
}

func collectFlagDefinitions(ctx context.Context, target reflect.Value, tag string) ([]flagDefinition, error) {
	definitions := make([]flagDefinition, 0)
	err := refl.TraverseStruct(
		refl.NewTraverseContext(ctx, struct{}{}, refl.WithFieldPathMetadata()),
		target,
		func(_ *refl.TraverseContext[struct{}], field refl.Node) error {
			definition, ok, traverseNext := fieldFlagDefinition(field, tag)
			if ok {
				if field.IsCollectionDescendant() {
					definition.collectionPath = field.FieldPath()
					definitions = addCollectionFlagDefinition(definitions, definition)
				} else {
					definitions = append(definitions, definition)
				}
			}
			return traverseNext
		},
	)
	return definitions, err
}

func addCollectionFlagDefinition(definitions []flagDefinition, definition flagDefinition) []flagDefinition {
	for i := range definitions {
		if definitions[i].name != definition.name || !definitions[i].collectionPath.Equal(definition.collectionPath) {
			continue
		}

		values, ok := collectionFlagValues(definitions[i].value)
		if !ok {
			continue
		}
		values = append(values, definition.value)
		definitions[i].value = newCollectionFlagValue(values)
		return definitions
	}

	definition.value = newCollectionFlagValue([]flag.Value{definition.value})
	definitions = append(definitions, definition)
	return definitions
}

func fieldFlagDefinition(field refl.Node, tag string) (flagDefinition, bool, error) {
	if !field.HasStructField {
		return flagDefinition{}, false, nil
	}
	if !field.StructField.IsExported() {
		return flagDefinition{}, false, refl.ErrSkipNested
	}

	flagTag := field.StructField.Tag.Get(tag)
	if flagTag == "-" {
		return flagDefinition{}, false, refl.ErrSkipNested
	}
	if flagTag == "" {
		return flagDefinition{}, false, nil
	}

	fk := field.Value.Type().Kind()
	var traverseNext error
	terminalKind := terminalType(field.Value.Type()).Kind()
	if terminalKind == reflect.Array || terminalKind == reflect.Slice || terminalKind == reflect.Map {
		traverseNext = refl.ErrSkipNested
	}

	tagData := parseFlagTag(flagTag)
	definition := flagDefinition{name: tagData.name, usage: tagData.usage}
	definition.value = fieldFlagValue(field, fk)
	return definition, true, traverseNext
}

func fieldFlagValue(field refl.Node, kind reflect.Kind) flag.Value {
	target := field.Value
	if kind != reflect.Ptr && target.CanAddr() {
		target = target.Addr()
	}
	if value, ok := deferredNilFlagValue(target, field); ok {
		return value
	}
	if value, ok := target.Interface().(flag.Value); ok {
		if field.HasDeferredCommit() {
			return newDeferredFlagValue(value, field.Commit)
		}
		return value
	}
	return newFlagScanner(field.Value, field.Commit)
}

func registerFlagDefinitions(fset *flag.FlagSet, definitions []flagDefinition) error {
	names := make(map[string]struct{}, len(definitions))
	for _, definition := range definitions {
		if err := validateFlagName(definition.name); err != nil {
			return fmt.Errorf("invalid flag name %q: %w", definition.name, err)
		}
		if _, ok := names[definition.name]; ok {
			return fmt.Errorf("duplicate flag name %q", definition.name)
		}
		if fset.Lookup(definition.name) != nil {
			return fmt.Errorf("flag name %q already exists in FlagSet", definition.name)
		}
		names[definition.name] = struct{}{}
	}

	for _, definition := range definitions {
		fset.Var(definition.value, definition.name, definition.usage)
	}
	return nil
}

func validateFlagName(name string) error {
	if name == "" {
		return errors.New("name is empty")
	}
	if strings.HasPrefix(name, "-") {
		return errors.New("name starts with '-'")
	}
	if strings.Contains(name, "=") {
		return errors.New("name contains '='")
	}
	return nil
}

func deferredNilFlagValue(target reflect.Value, field refl.Node) (flag.Value, bool) {
	if target.Kind() != reflect.Ptr || !target.IsNil() || !target.Type().Implements(reflect.TypeFor[flag.Value]()) {
		return nil, false
	}

	value := reflect.New(target.Type().Elem())
	return newDeferredFlagValue(
		value.Interface().(flag.Value),
		func() {
			target.Set(value)
			field.Commit()
		},
	), true
}

func extractRegistered(fset *flag.FlagSet, names map[string]struct{}, args ...string) (res []string) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			break
		}

		name, hasValue, ok := parseFlagArgument(arg)
		if !ok {
			continue
		}

		f := fset.Lookup(name)
		if f == nil {
			if name == "h" || name == "help" {
				res = append(res, arg)
			}
			continue
		}

		value := ""
		consumesValue := !hasValue && !isBoolFlag(f) && i < len(args)-1
		if consumesValue {
			i++
			value = args[i]
		}

		if names != nil {
			if _, ok := names[name]; !ok {
				continue
			}
		}

		res = append(res, arg)
		if consumesValue {
			res = append(res, value)
		}
	}

	return res
}

func parseFlagArgument(arg string) (name string, hasValue, ok bool) {
	if len(arg) < 2 || arg[0] != '-' {
		return "", false, false
	}

	name = arg[1:]
	if name[0] == '-' {
		name = name[1:]
	}
	if name == "" {
		return "", false, false
	}

	name, _, hasValue = strings.Cut(name, "=")
	return name, hasValue, true
}

func isBoolFlag(f *flag.Flag) bool {
	boolFlag, ok := f.Value.(interface{ IsBoolFlag() bool })
	return ok && boolFlag.IsBoolFlag()
}

const (
	flagTagUsagePrefix = "usage="
)

type flagStructTag struct {
	name  string
	usage string
}

func parseFlagTag(tag string) flagStructTag {
	if tag == "" {
		return flagStructTag{}
	}

	parts := strings.Split(tag, ",")

	res := flagStructTag{name: parts[0]}
	for _, part := range parts {
		// parse flag usage from tag
		if strings.HasPrefix(part, flagTagUsagePrefix) {
			res.usage = part[len(flagTagUsagePrefix):]
		}
	}

	return res
}

// printHelp prints help message to standard error
func printHelp(f *flag.FlagSet) {
	f.VisitAll(func(fl *flag.Flag) {
		var b strings.Builder
		fmt.Fprintf(&b, "  -%s", fl.Name) // Two spaces before -; see next two comments.
		name, usage := flag.UnquoteUsage(fl)
		if len(name) > 0 {
			b.WriteString(" ")
			b.WriteString(name)
		}
		b.WriteString("\t")
		b.WriteString(strings.ReplaceAll(usage, "\n", "\n    \t"))

		_, _ = fmt.Fprint(f.Output(), b.String(), "\n")
	})
}
