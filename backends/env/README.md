# env

`env` is a `confetti` backend to load data from environment variables.

## Usage

To load value from single variable use `FromVar` function:

```go
package main

import (
    "context"

    "golang.yandex/confetti"
    "golang.yandex/confetti/backends/env"
)

func main() {
    var gopath, goroot string

    _ = confetti.NewLoader().
        Load(context.Background(),
            confetti.To(&gopath, env.FromVar("GOPATH")),
            confetti.To(&goroot, env.FromVar("GOROOT")),
        )
}
```

To load multiple variables to struct use `FromEnviron` function in conjunction with struct tags:

```go
package main

import (
    "context"

    "golang.yandex/confetti"
    "golang.yandex/confetti/backends/env"
)

type goSetup struct {
    GoPath string `env:"GOPATH"`
    GoRoot string `env:"GOROOT"`
}

func main() {
    var setup goSetup

    _ = confetti.NewLoader().
        Load(context.Background(), confetti.To(&setup, env.FromEnviron()))
}
```

`FromEnviron` accepts multiple functional options to fine tune loading behaviour:

- `Tag` - overrides struct tag name to look env var name in. Default: `"env"`.
- `Prefix` - sets prefix to be added to all env var names fetched from struct tags.
- `RecursiveKeys` - enables keys concatenation for folded struct tags.
- `RecursiveKeyGlue` - sets string to be used to concat recursive keys: Default: `"_"`.

### Custom value decoding

You can process loaded values with your custom logic if your type implements one of given interfaces:

- `sql.Scanner`
- `encoding.TextUnmarshaler`
- `encoding.BinaryUnmarshaler`
