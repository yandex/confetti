# pem

`pem` is a `confetti` backend to load various cryptographic data from PEM encoded sources.

Basically it is just a wrapper around `crypto`.

## Usage

To load PEM data from file use `FromFile` function:

```go
package main

import (
    "context"
    "crypto/rsa"

    "golang.yandex/confetti"
    pem "golang.yandex/confetti/backends/pem"
)

func main() {
    var pubkey *rsa.PublicKey

    _ = confetti.NewLoader().
        Load(context.Background(), confetti.To(&pubkey, pem.FromFile("pubkey.pem")))
}
```

There are also `FromString` and `FromReader` function that loads PEM data from given raw string and `io.Reader` respectively.

Special function `FromBlocks` allows to load PEM data from previously parsed PEM blocks.
