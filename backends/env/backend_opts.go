package env

type EnvironOpt func(*environOpts)

// RecursiveKeys is used in FromEnviron function to enable complex keys construction.
// It remembers parent struct key as parser recursively descends deeper and uses it as key prefix for child values.
//
// Example:
//
//	type Config struct {
//		RemoteHost string `env:"APP_REMOTE_HOST"`
//		Credentials Credentials `env:"APP_CREDENTIALS"`
//	}
//
//	type Credentials struct {
//		Username string `env:"USERNAME"`
//		Password string `env:"PASSWORD"`
//	}
//
// In this example, if RecursiveKeys option passed to FromEnviron function,
// struct will be filled using values from 3 environment variables:
// - APP_REMOTE_HOST
// - APP_CREDENTIALS_USERNAME
// - APP_CREDENTIALS_PASSWORD
func RecursiveKeys(opts *environOpts) {
	opts.constructPrefix = true
}

// RecursiveKeyGlue sets custom string to join complex key parts.
// It must be used in conjunction with RecursiveKeys option.
func RecursiveKeyGlue(glue string) EnvironOpt {
	return func(opts *environOpts) {
		opts.prefixGlue = glue
	}
}

// Tag sets struct tag to look key in.
func Tag(tag string) EnvironOpt {
	return func(opts *environOpts) {
		opts.tag = tag
	}
}

// Prefix sets prefix for every env lookup.
func Prefix(p string) EnvironOpt {
	return func(opts *environOpts) {
		opts.prefix = p
	}
}
