# json

`json` is a `confetti` backend to load data from JSON source.

Basically it is just a thin wrapper around `encoding/json`.

## Usage

To load JSON data from file use `FromFile` function:

```go
package main

import (
    "context"

    "golang.yandex/confetti"
    cjson "golang.yandex/confetti/backends/json"
)

type goSetup struct {
    GoPath string `json:"gopath"`
    GoRoot string `json:"goroot"`
}

func main() {
    var setup goSetup

    _ = confetti.NewLoader().
        Load(context.Background(), confetti.To(&setup, cjson.FromFile("data.json")))
}
```

There are also `FromString` and `FromReader` function that loads JSON from given raw string and `io.Reader` respectively.
