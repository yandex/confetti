package pem

import (
	"bytes"
	"encoding/pem"
	"fmt"
)

// chain represents chain of PEM blocks sorted by types
type chain struct {
	blocks  []*pem.Block
	private int
	public  int
	cert    int
}

// newChain parses given PEM blocks into chain
func newChain(blocks ...*pem.Block) chain {
	var priv, pub, cert []*pem.Block
	for _, b := range blocks {
		if b == nil {
			continue
		}
		switch b.Type {
		case "PRIVATE KEY", "RSA PRIVATE KEY", "EC PRIVATE KEY":
			priv = append(priv, b)
		case "PUBLIC KEY", "RSA PUBLIC KEY", "EC PUBLIC KEY":
			pub = append(pub, b)
		case "CERTIFICATE":
			cert = append(cert, b)
		}
	}

	var c chain
	c.blocks = append(c.blocks, priv...)
	c.blocks = append(c.blocks, pub...)
	c.blocks = append(c.blocks, cert...)

	c.private, c.public, c.cert = len(priv), len(pub), len(cert)
	return c
}

// Blocks returns all blocks from chain
func (c chain) Blocks() []*pem.Block {
	return c.blocks
}

// Private returns all private blocks from chain
func (c chain) Private() []*pem.Block {
	return c.blocks[:c.private]
}

// Public returns all public blocks from chain
func (c chain) Public() []*pem.Block {
	return c.blocks[c.private : c.private+c.public]
}

// Certificates returns all certificate blocks from chain
func (c chain) Certificates() []*pem.Block {
	return c.blocks[c.private+c.public:]
}

// MarshalBinary encodes all PEM blocks to bytes
func (c chain) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer

	for _, block := range c.blocks {
		if err := pem.Encode(&buf, block); err != nil {
			return nil, fmt.Errorf("cannot encode %s block: %w", block.Type, err)
		}
	}

	return buf.Bytes(), nil
}
