package pem

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	mathrand "math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"golang.yandex/confetti/internal/slices"
)

func TestNewChain(t *testing.T) {
	privkey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	pubkey := &privkey.PublicKey

	privder, err := x509.MarshalPKCS8PrivateKey(privkey)
	require.NoError(t, err)
	privpem := &pem.Block{Type: "PRIVATE KEY", Bytes: privder}

	pubder, err := x509.MarshalPKIXPublicKey(pubkey)
	require.NoError(t, err)
	pubpem := &pem.Block{Type: "PUBLIC KEY", Bytes: pubder}

	cert := newTestCertificate()
	certder, err := x509.CreateCertificate(rand.Reader, cert, cert, pubkey, privkey)
	require.NoError(t, err)
	cert.Raw = certder
	certpem := &pem.Block{Type: "CERTIFICATE", Bytes: certder}

	blocks := []*pem.Block{privpem, pubpem, certpem}
	slices.Shuffle(blocks, mathrand.NewSource(time.Now().UnixNano()))

	c := newChain(blocks...)

	expected := []*pem.Block{privpem, pubpem, certpem}
	assert.Equal(t, expected, c.blocks)
}

func TestChain_Getters(t *testing.T) {
	privkey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	pubkey := &privkey.PublicKey
	cert := newTestCertificate()

	c, err := newTestChain(privkey, pubkey, cert)
	require.NoError(t, err)

	privder, err := x509.MarshalPKCS8PrivateKey(privkey)
	require.NoError(t, err)
	privpem := &pem.Block{Type: "PRIVATE KEY", Bytes: privder}

	pubder, err := x509.MarshalPKIXPublicKey(pubkey)
	require.NoError(t, err)
	pubpem := &pem.Block{Type: "PUBLIC KEY", Bytes: pubder}

	certder, err := x509.CreateCertificate(rand.Reader, cert, cert, pubkey, privkey)
	require.NoError(t, err)
	cert.Raw = certder
	certpem := &pem.Block{Type: "CERTIFICATE", Bytes: certder}

	t.Run("private", func(t *testing.T) {
		assert.Equal(t, []*pem.Block{privpem}, c.Private())
	})
	t.Run("public", func(t *testing.T) {
		assert.Equal(t, []*pem.Block{pubpem}, c.Public())
	})
	t.Run("certificates", func(t *testing.T) {
		assert.Equal(t, []*pem.Block{certpem}, c.Certificates())
	})
	t.Run("blocks", func(t *testing.T) {
		assert.Equal(t, []*pem.Block{privpem, pubpem, certpem}, c.Blocks())
	})
}
