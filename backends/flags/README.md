# flags

`flags` is a `confetti` backend to load data from command line arguments.

## Usage

To load value from single argument use `From` function:

```go
package main

import (
    "context"

    "golang.yandex/confetti"
    "golang.yandex/confetti/backends/flags"
)

func main() {
    var gopath, goroot string

    _ = confetti.NewLoader().
        Load(context.Background(),
            confetti.To(&gopath, flags.From("p")),
            confetti.To(&goroot, flags.From("r")),
        )
}
```

`From` accepts multiple functional options to fine tune loading behaviour:

- `Usage` - allows to supply custom string as an example of flag usage.
- `FlagSet(*flag.FlagSet)` - sets a custom flag set. You can fine tune help output and parse errors with it.

To load multiple arguments to struct use `FromArgs` function in conjunction with struct tags:

```go
package main

import (
    "context"

    "golang.yandex/confetti"
    "golang.yandex/confetti/backends/flags"
)

type goSetup struct {
    GoPath string `flag:"g,usage=path to directory with packages sources"`
    GoRoot string `flag:"r,usage=path to directory with Go binaries"`
}

func main() {
    var setup goSetup

    _ = confetti.NewLoader().
        Load(context.Background(), confetti.To(&setup, flags.FromArgs()))
}
```

`FromArgs` accepts multiple functional options to fine tune loading behaviour:

- `Tag` - overrides struct tag name to look argument name in. Default: `"flag"`.
- `Name` - sets name of flag set to register arguments to. Default: `os.Args[0]`.
- `FlagsSet` - sets custom FlagSet. You can fine tune various params, like help printer or parsing errors processing, with it.

### Custom value decoding

You can process loaded values with your custom logic if your type implements one of given interfaces:

- `flag.Value`
- `sql.Scanner`
- `encoding.TextUnmarshaler`
- `encoding.BinaryUnmarshaler`
