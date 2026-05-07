# confetti

`confetti` is a library to load data from multiple various sources to custom targets.

## Usage

Load single string from environment variable:

```go
package main

import (
    "context"

    "golang.yandex/confetti"
    "golang.yandex/confetti/backends/env"
)

func main() {
    var username string

    err := confetti.NewLoader().
        Load(context.Background(),
            confetti.To(&username, env.FromVar("USER")),
        )
    // ...
}
```

or use a `Fill` shorthand:

```go
package main

import (
    "context"

    "golang.yandex/confetti"
    "golang.yandex/confetti/backends/env"
)

func main() {
    var username string

    err := confetti.Fill(context.Background(), &username, env.FromVar("USER"))
    // ...
}
```

You can use multiple value setters in chain to load data with override.

Load string from environment variable or command line argument:

```go
package main

import (
    "context"

    "golang.yandex/confetti"
    "golang.yandex/confetti/backends/env"
    "golang.yandex/confetti/backends/flags"
)

func main() {
    var username string

    err := confetti.NewLoader().
        Load(context.Background(),
            confetti.To(&username, env.FromVar("USER"), flags.From("u")),
        )
    // ...
}
```

Note that command line argument has higher priority due to loading order.

You can load data to custom struct using struct tags:

```go
package main

import (
    "context"

    "golang.yandex/confetti"
    "golang.yandex/confetti/backends/env"
    "golang.yandex/confetti/backends/flags"
)

type dbconf struct {
    User     string `env:"DB_USER" flag:"u"`
    Password string `env:"DB_PASSWORD" flag:"P"`
    Host     string `env:"DB_HOST" flag:"H"`
    Port     string `env:"DB_PORT" flag:"p"`
}

func main() {
    var cfg dbconf

    err := confetti.NewLoader().
        Load(context.Background(),
            confetti.To(&cfg, env.FromEnviron(), flags.FromArgs()),
        )
    // ...
}
```

### Available backends

- [env](backends/env) - loads data from environment variables
- [flags](backends/flags) - loads data from command line arguments
- [pem](backends/pem) - loads data from PEM files or other popular sources
- [json](/backends/json) - loads data from JSON strings, files or other popular sources

### Write your own backend

In a nutshell a simplest backend is just a function of given signature:

```go
type ValueSetter func(ctx context.Context, target any) error
```

Suppose you want to implement backend that fills value with IP address data from `/etc/hosts`:

```go
package hosts

import (
    "context"
    "errors"
    "fmt"
    "reflect"
)

// ForHost sets IP address of given host to target based on /etc/hosts file content
func ForHost(host string) func(context.Context, any) error {
    return func(_ context.Context, target any) error {
        fv := reflect.ValueOf(target)

        if fv.Kind() != reflect.Ptr {
            return fmt.Errorf("target for host '%s' must be a pointer", host)
        }

        fv = fv.Elem()
        if fv.Kind() != reflect.String {
            return fmt.Errorf("target for host '%s' must be a pointer to string", host)
        }

        val, err := findIPByHost(host)
        if errors.Is(err, ErrNotFound) {
            return nil
        }
        if err != nil {
            return fmt.Errorf("cannot find IP address for host '%s' in /etc/hosts: %w", err)
        }

        fv.SetString(val)
        return nil
    }
}
```
