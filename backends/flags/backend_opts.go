package flags

import (
	"flag"
)

// FlagOpt is a callback function for single flag definition
type FlagOpt func(*flagOpts)

// flagOpts holds flag definition options
type flagOpts struct {
	name  string
	usage string
	fset  *flag.FlagSet
}

// Usage sets flag usage string
func Usage(usage string) FlagOpt {
	return func(s *flagOpts) {
		s.usage = usage
	}
}

// FlagSet sets a flag set as a base for a single flag definition.
func FlagSet(fset *flag.FlagSet) FlagOpt {
	return func(s *flagOpts) {
		s.fset = fset
	}
}

// FlagSetOpt is a callback function for flag set definition
type FlagSetOpt func(*flagSetOpts)

// flagSetOpts holds flag set definition options
type flagSetOpts struct {
	name string
	tag  string
	fset *flag.FlagSet
}

// Name specifies flag set name
func Name(name string) FlagSetOpt {
	return func(s *flagSetOpts) {
		s.name = name
	}
}

// Tag sets name of struct tag containing flag names
func Tag(tag string) FlagSetOpt {
	return func(s *flagSetOpts) {
		s.tag = tag
	}
}

// FlagsSet sets flag set as base to work with multiple flags definitions
func FlagsSet(fset *flag.FlagSet) FlagSetOpt {
	return func(fso *flagSetOpts) {
		fso.fset = fset
	}
}
