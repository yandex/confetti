package pem

import (
	"bytes"
	"context"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFromReader_mixedChain(t *testing.T) {
	t.Run("with_key", func(t *testing.T) {
		raw := []byte(`lol
kek
cheburek
-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA2jq+Hk0es6wCgCwzK/Tj
0txkOw5bsHVxf8NpYWmOI4NavIue1lmmBvfZmog8nDh0TTcs3pxs/YMpeDdK9W/P
Jx9y5BCMC9rL8W7dWyVUX7jjT4QiNTR+w+YIHw8hNF0KuiVbgHdYiPEaM8K5t/9l
/tspTE4jW27fmebZGjePsKaw3uj1d9/YKWfHBFKGzkmxvalDjt85d4V2wFcZpbNK
T8FHvG6//dJBzKm8wL5wCIhYkthRtbPdK0Jj0n8KItvdqZS/Lc8FEJZUeoQyVI3W
EM4pil6C3wagNf7PwRUhjEesTSO+3EHbUYeWIWSahJVHbPHzDa4IIwC/h3IBTH5n
8wIDAQAB
-----END PUBLIC KEY-----`)

		var target *rsa.PublicKey
		setter := FromReader(bytes.NewReader(raw))
		err := setter(context.Background(), &target)
		assert.NoError(t, err)
		assert.NotNil(t, target)
	})

	t.Run("without_key", func(t *testing.T) {
		raw := []byte(`
			lol
			kek
			cheburek
		`)

		var target *rsa.PublicKey
		setter := FromReader(bytes.NewReader(raw))
		err := setter(context.Background(), &target)
		assert.NoError(t, err)
		assert.Nil(t, target)
	})
}

func TestFromBlocks_single(t *testing.T) {
	testCases := []struct {
		name       string
		keygen     func() (PrivateKey, PublicKey)
		privTarget PrivateKey
		pubTarget  PublicKey
	}{
		{
			name: "rsa",
			keygen: func() (PrivateKey, PublicKey) {
				privkey, err := rsa.GenerateKey(rand.Reader, 2048)
				require.NoError(t, err)
				return privkey, &privkey.PublicKey
			},
			privTarget: new(rsa.PrivateKey),
			pubTarget:  new(rsa.PublicKey),
		},
		{
			name: "ecdsa",
			keygen: func() (PrivateKey, PublicKey) {
				privkey, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
				require.NoError(t, err)
				return privkey, &privkey.PublicKey
			},
			privTarget: new(ecdsa.PrivateKey),
			pubTarget:  new(ecdsa.PublicKey),
		},
		{
			name: "ed25519",
			keygen: func() (PrivateKey, PublicKey) {
				pubkey, privkey, err := ed25519.GenerateKey(rand.Reader)
				require.NoError(t, err)
				return privkey, pubkey
			},
			privTarget: new(ed25519.PrivateKey),
			pubTarget:  new(ed25519.PublicKey),
		},
		{
			name: "ecdh",
			keygen: func() (PrivateKey, PublicKey) {
				privkey, err := ecdh.X25519().GenerateKey(rand.Reader)
				require.NoError(t, err)
				return privkey, privkey.PublicKey()
			},
			privTarget: new(ecdh.PrivateKey),
			pubTarget:  new(ecdh.PublicKey),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			privkey, pubkey := tc.keygen()
			c, err := newTestChain(privkey, pubkey, nil)
			require.NoError(t, err)

			setter := FromBlocks(c.blocks...)

			err = setter(context.Background(), &tc.privTarget)
			assert.NoError(t, err)
			assert.True(t, tc.privTarget.Equal(privkey), "private keys are not equal")

			err = setter(context.Background(), &tc.pubTarget)
			assert.NoError(t, err)
			assert.True(t, tc.pubTarget.Equal(pubkey), "public keys are not equal")
		})
	}
}

func TestFromBlocks_keyLabelParsers(t *testing.T) {
	rsaPrivateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	ecdsaPrivateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	pkcs8PrivateKey, err := x509.MarshalPKCS8PrivateKey(rsaPrivateKey)
	require.NoError(t, err)
	ecPrivateKey, err := x509.MarshalECPrivateKey(ecdsaPrivateKey)
	require.NoError(t, err)
	pkixPublicKey, err := x509.MarshalPKIXPublicKey(&rsaPrivateKey.PublicKey)
	require.NoError(t, err)
	pkixECPublicKey, err := x509.MarshalPKIXPublicKey(&ecdsaPrivateKey.PublicKey)
	require.NoError(t, err)

	testCases := []struct {
		name       string
		block      *pem.Block
		privateKey PrivateKey
		publicKey  PublicKey
	}{
		{
			name:       "PKCS8_private_key",
			block:      &pem.Block{Type: "PRIVATE KEY", Bytes: pkcs8PrivateKey},
			privateKey: rsaPrivateKey,
		},
		{
			name:       "PKCS1_RSA_private_key",
			block:      &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(rsaPrivateKey)},
			privateKey: rsaPrivateKey,
		},
		{
			name:       "SEC1_EC_private_key",
			block:      &pem.Block{Type: "EC PRIVATE KEY", Bytes: ecPrivateKey},
			privateKey: ecdsaPrivateKey,
		},
		{
			name:      "PKIX_public_key",
			block:     &pem.Block{Type: "PUBLIC KEY", Bytes: pkixPublicKey},
			publicKey: &rsaPrivateKey.PublicKey,
		},
		{
			name:      "PKIX_EC_public_key",
			block:     &pem.Block{Type: "EC PUBLIC KEY", Bytes: pkixECPublicKey},
			publicKey: &ecdsaPrivateKey.PublicKey,
		},
		{
			name:      "PKCS1_RSA_public_key",
			block:     &pem.Block{Type: "RSA PUBLIC KEY", Bytes: x509.MarshalPKCS1PublicKey(&rsaPrivateKey.PublicKey)},
			publicKey: &rsaPrivateKey.PublicKey,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.privateKey != nil {
				var target PrivateKey

				err := FromBlocks(tc.block)(t.Context(), &target)

				require.NoError(t, err)
				require.NotNil(t, target)
				assert.True(t, target.Equal(tc.privateKey))
				return
			}

			var target PublicKey

			err := FromBlocks(tc.block)(t.Context(), &target)

			require.NoError(t, err)
			require.NotNil(t, target)
			assert.True(t, target.Equal(tc.publicKey))
		})
	}
}

func TestFromBlocks_struct(t *testing.T) {
	privateKey, certificate, blocks := testStructPEMBlocks(t)

	t.Run("flat_fields", func(t *testing.T) {
		type targetType struct {
			PrivateKey  *rsa.PrivateKey
			PublicKey   *rsa.PublicKey
			Certificate *x509.Certificate
			PublicKeys  []*rsa.PublicKey
			PEM         []byte
		}

		var target targetType
		err := FromBlocks(blocks...)(t.Context(), &target)

		require.NoError(t, err)
		require.NotNil(t, target.PrivateKey)
		assert.True(t, target.PrivateKey.Equal(privateKey))
		require.NotNil(t, target.PublicKey)
		assert.True(t, target.PublicKey.Equal(&privateKey.PublicKey))
		require.NotNil(t, target.Certificate)
		assert.True(t, target.Certificate.Equal(certificate))
		require.Len(t, target.PublicKeys, 1)
		assert.True(t, target.PublicKeys[0].Equal(&privateKey.PublicKey))

		expectedPEM, err := newChain(blocks...).MarshalBinary()
		require.NoError(t, err)
		assert.Equal(t, expectedPEM, target.PEM)
	})

	t.Run("nested_struct", func(t *testing.T) {
		type certificates struct {
			PublicKey    *rsa.PublicKey
			Certificates []*x509.Certificate
		}
		type targetType struct {
			Certificates certificates
		}

		var target targetType
		err := FromBlocks(blocks...)(t.Context(), &target)

		require.NoError(t, err)
		require.NotNil(t, target.Certificates.PublicKey)
		assert.True(t, target.Certificates.PublicKey.Equal(&privateKey.PublicKey))
		require.Len(t, target.Certificates.Certificates, 1)
		assert.True(t, target.Certificates.Certificates[0].Equal(certificate))
	})

	t.Run("nil_nested_pointer", func(t *testing.T) {
		type keys struct {
			PublicKey *rsa.PublicKey
		}
		type targetType struct {
			Keys *keys
		}

		var unmatched targetType
		err := FromBlocks(blocks[0])(t.Context(), &unmatched)
		require.NoError(t, err)
		assert.Nil(t, unmatched.Keys)

		var matched targetType
		err = FromBlocks(blocks[1])(t.Context(), &matched)
		require.NoError(t, err)
		require.NotNil(t, matched.Keys)
		assert.True(t, matched.Keys.PublicKey.Equal(&privateKey.PublicKey))
	})

	t.Run("unsupported_and_unexported_fields", func(t *testing.T) {
		type targetType struct {
			PublicKey *rsa.PublicKey
			Pool      *x509.CertPool
			private   *rsa.PublicKey
			Name      string
		}

		private := &rsa.PublicKey{N: big.NewInt(7), E: 3}
		pool := x509.NewCertPool()
		target := targetType{
			Pool:    pool,
			private: private,
			Name:    "preserve",
		}
		err := FromBlocks(blocks[1])(t.Context(), &target)

		require.NoError(t, err)
		require.NotNil(t, target.PublicKey)
		assert.True(t, target.PublicKey.Equal(&privateKey.PublicKey))
		assert.Same(t, pool, target.Pool)
		assert.Same(t, private, target.private)
		assert.Equal(t, "preserve", target.Name)
	})

	t.Run("unmatched_blocks_preserve_target", func(t *testing.T) {
		type targetType struct {
			PublicKey *rsa.PublicKey
		}

		original := &rsa.PublicKey{N: big.NewInt(7), E: 3}
		target := targetType{PublicKey: original}
		err := FromBlocks(blocks[2])(t.Context(), &target)

		require.NoError(t, err)
		assert.Same(t, original, target.PublicKey)
	})
}

func TestFromBlocks_structAtomic(t *testing.T) {
	type nested struct {
		PublicKeys []*rsa.PublicKey
	}
	type targetType struct {
		PublicKey *rsa.PublicKey
		Nested    nested
	}

	originalPublicKey := &rsa.PublicKey{N: big.NewInt(7), E: 3}
	originalNestedKey := &rsa.PublicKey{N: big.NewInt(11), E: 3}
	target := targetType{
		PublicKey: originalPublicKey,
		Nested: nested{
			PublicKeys: []*rsa.PublicKey{originalNestedKey},
		},
	}
	blocks := []*pem.Block{
		testRSAPublicKeyBlock(t),
		{Type: "PUBLIC KEY", Bytes: []byte("malformed")},
	}

	err := FromBlocks(blocks...)(t.Context(), &target)

	require.ErrorContains(t, err, "PublicKey")
	assert.Same(t, originalPublicKey, target.PublicKey)
	require.Len(t, target.Nested.PublicKeys, 1)
	assert.Same(t, originalNestedKey, target.Nested.PublicKeys[0])
}

func TestFromBlocks_structMalformedNestedField(t *testing.T) {
	type nested struct {
		PublicKeys []*rsa.PublicKey
	}
	type targetType struct {
		Nested nested
	}

	original := &rsa.PublicKey{N: big.NewInt(7), E: 3}
	target := targetType{Nested: nested{PublicKeys: []*rsa.PublicKey{original}}}
	blocks := []*pem.Block{
		testRSAPublicKeyBlock(t),
		{Type: "PUBLIC KEY", Bytes: []byte("malformed")},
	}

	err := FromBlocks(blocks...)(t.Context(), &target)

	require.ErrorContains(t, err, "Nested.PublicKeys")
	require.Len(t, target.Nested.PublicKeys, 1)
	assert.Same(t, original, target.Nested.PublicKeys[0])
}

func TestFromBlocks_structInterfaceTarget(t *testing.T) {
	type targetType struct {
		PublicKey *rsa.PublicKey
	}

	block := testRSAPublicKeyBlock(t)
	contained := targetType{}
	var target any = contained

	err := FromBlocks(block)(t.Context(), &target)

	require.NoError(t, err)
	actual, ok := target.(targetType)
	require.True(t, ok)
	require.NotNil(t, actual.PublicKey)
	assert.Equal(t, big.NewInt(3), actual.PublicKey.N)
	assert.Equal(t, 65537, actual.PublicKey.E)
}

func TestFromBlocks_structPointerChain(t *testing.T) {
	type targetType struct {
		PublicKey **rsa.PublicKey
	}

	block := testRSAPublicKeyBlock(t)
	originalKey := &rsa.PublicKey{N: big.NewInt(7), E: 3}
	inner := originalKey
	originalPointer := &inner
	target := targetType{PublicKey: originalPointer}

	err := FromBlocks(block)(t.Context(), &target)

	require.NoError(t, err)
	assert.Same(t, originalPointer, target.PublicKey)
	assert.Same(t, originalKey, *target.PublicKey)
	assert.Equal(t, big.NewInt(3), (**target.PublicKey).N)
	assert.Equal(t, 65537, (**target.PublicKey).E)
}

func TestFromBlocks_structPointerToSlice(t *testing.T) {
	type targetType struct {
		PublicKeys *[]*rsa.PublicKey
	}

	block := testRSAPublicKeyBlock(t)
	originalKey := &rsa.PublicKey{N: big.NewInt(7), E: 3}
	keys := []*rsa.PublicKey{originalKey}
	originalPointer := &keys
	target := targetType{PublicKeys: originalPointer}

	err := FromBlocks(block)(t.Context(), &target)

	require.NoError(t, err)
	assert.Same(t, originalPointer, target.PublicKeys)
	require.Len(t, *target.PublicKeys, 1)
	assert.Equal(t, big.NewInt(3), (*target.PublicKeys)[0].N)
	assert.Equal(t, 65537, (*target.PublicKeys)[0].E)
}

func TestFromBlocks_structInterfaceField(t *testing.T) {
	type targetType struct {
		PublicKey PublicKey
	}

	block := testRSAPublicKeyBlock(t)
	original := &rsa.PublicKey{N: big.NewInt(7), E: 3}
	target := targetType{PublicKey: original}

	err := FromBlocks(block)(t.Context(), &target)

	require.NoError(t, err)
	assert.Same(t, original, target.PublicKey)
	assert.Equal(t, big.NewInt(3), original.N)
	assert.Equal(t, 65537, original.E)
}

func TestFromBlocks_recursiveStruct(t *testing.T) {
	type targetType struct {
		PublicKey *rsa.PublicKey
		Next      *targetType
	}

	block := testRSAPublicKeyBlock(t)
	nextKey := &rsa.PublicKey{N: big.NewInt(7), E: 3}
	next := &targetType{PublicKey: nextKey}
	target := targetType{Next: next}

	err := FromBlocks(block)(t.Context(), &target)

	require.NoError(t, err)
	require.NotNil(t, target.PublicKey)
	assert.Equal(t, big.NewInt(3), target.PublicKey.N)
	assert.Same(t, next, target.Next)
	assert.Same(t, nextKey, target.Next.PublicKey)
}

func TestFromBlocks_targetSafety(t *testing.T) {
	validBlock := testRSAPublicKeyBlock(t)
	malformedBlock := &pem.Block{Type: "PUBLIC KEY", Bytes: []byte("invalid")}
	unmatchedBlock := &pem.Block{Type: "CERTIFICATE"}

	testCases := []struct {
		name   string
		blocks []*pem.Block
		test   func(*testing.T, func(context.Context, any) error)
	}{
		{
			name: "zero_blocks_are_noop",
			test: func(t *testing.T, setter func(context.Context, any) error) {
				assert.NoError(t, setter(t.Context(), rsa.PublicKey{}))
			},
		},
		{
			name:   "nil_direct_target",
			blocks: []*pem.Block{validBlock},
			test: func(t *testing.T, setter func(context.Context, any) error) {
				assert.Error(t, setter(t.Context(), nil))
			},
		},
		{
			name:   "typed_nil_direct_target",
			blocks: []*pem.Block{validBlock},
			test: func(t *testing.T, setter func(context.Context, any) error) {
				var target *rsa.PublicKey
				assert.Error(t, setter(t.Context(), target))
			},
		},
		{
			name:   "non_pointer_direct_target",
			blocks: []*pem.Block{validBlock},
			test: func(t *testing.T, setter func(context.Context, any) error) {
				assert.Error(t, setter(t.Context(), rsa.PublicKey{}))
			},
		},
		{
			name:   "unmatched_block_preserves_nil_destination",
			blocks: []*pem.Block{unmatchedBlock},
			test: func(t *testing.T, setter func(context.Context, any) error) {
				var target *rsa.PublicKey
				assert.NoError(t, setter(t.Context(), &target))
				assert.Nil(t, target)
			},
		},
		{
			name:   "unsupported_target_preserves_nil_destination",
			blocks: []*pem.Block{validBlock},
			test: func(t *testing.T, setter func(context.Context, any) error) {
				var target *x509.CertPool
				assert.NoError(t, setter(t.Context(), &target))
				assert.Nil(t, target)
			},
		},
		{
			name:   "malformed_block_preserves_nil_destination",
			blocks: []*pem.Block{malformedBlock},
			test: func(t *testing.T, setter func(context.Context, any) error) {
				var target *rsa.PublicKey
				assert.Error(t, setter(t.Context(), &target))
				assert.Nil(t, target)
			},
		},
		{
			name:   "malformed_block_preserves_existing_destination",
			blocks: []*pem.Block{malformedBlock},
			test: func(t *testing.T, setter func(context.Context, any) error) {
				target := &rsa.PublicKey{N: big.NewInt(7), E: 65537}
				original := target
				originalValue := *target

				assert.Error(t, setter(t.Context(), &target))
				assert.Same(t, original, target)
				assert.Equal(t, originalValue, *target)
			},
		},
		{
			name:   "successful_pointer_chain",
			blocks: []*pem.Block{validBlock},
			test: func(t *testing.T, setter func(context.Context, any) error) {
				var target **rsa.PublicKey

				assert.NoError(t, setter(t.Context(), &target))
				require.NotNil(t, target)
				require.NotNil(t, *target)
				assert.Equal(t, big.NewInt(3), (*target).N)
				assert.Equal(t, 65537, (*target).E)
			},
		},
		{
			name:   "success_preserves_existing_pointer",
			blocks: []*pem.Block{validBlock},
			test: func(t *testing.T, setter func(context.Context, any) error) {
				target := &rsa.PublicKey{N: big.NewInt(7), E: 3}
				original := target

				assert.NoError(t, setter(t.Context(), &target))
				assert.Same(t, original, target)
				assert.Equal(t, big.NewInt(3), target.N)
				assert.Equal(t, 65537, target.E)
			},
		},
		{
			name:   "nil_interface_target",
			blocks: []*pem.Block{validBlock},
			test: func(t *testing.T, setter func(context.Context, any) error) {
				var target PublicKey

				assert.NoError(t, setter(t.Context(), &target))
				key, ok := target.(*rsa.PublicKey)
				require.True(t, ok)
				assert.Equal(t, big.NewInt(3), key.N)
			},
		},
		{
			name:   "typed_nil_interface_target",
			blocks: []*pem.Block{validBlock},
			test: func(t *testing.T, setter func(context.Context, any) error) {
				var key *rsa.PublicKey
				var target any = key

				assert.NoError(t, setter(t.Context(), &target))
				key, ok := target.(*rsa.PublicKey)
				require.True(t, ok)
				assert.Equal(t, big.NewInt(3), key.N)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.test(t, FromBlocks(tc.blocks...))
		})
	}
}

func TestFromReader_typedNilTarget(t *testing.T) {
	block := testRSAPublicKeyBlock(t)
	var target *rsa.PublicKey

	err := FromReader(bytes.NewReader(pem.EncodeToMemory(block)))(t.Context(), target)

	assert.Error(t, err)
	assert.Nil(t, target)
}

func TestFromBlocks_certificate(t *testing.T) {
	cert := newTestCertificate()

	certPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	assert.NoError(t, err)

	caBytes, err := x509.CreateCertificate(rand.Reader, cert, cert, &certPrivKey.PublicKey, certPrivKey)
	assert.NoError(t, err)
	cert.Raw = caBytes

	certpem := &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	}

	var target x509.Certificate
	setter := FromBlocks(certpem)
	err = setter(context.Background(), &target)
	assert.NoError(t, err)

	assert.True(t, target.Equal(cert), "certificates are not equal")
}

func TestFromBlocks_slice(t *testing.T) {
	// generate blocks
	var blocks []*pem.Block
	for i := 0; i < 5; i++ {
		privkey, err := rsa.GenerateKey(rand.Reader, 4096)
		assert.NoError(t, err)

		cert := newTestCertificate()
		c, err := newTestChain(privkey, &privkey.PublicKey, cert)
		require.NoError(t, err)

		blocks = append(blocks, c.blocks...)
	}

	t.Run("bytes", func(t *testing.T) {
		var target []byte
		setter := FromBlocks(blocks...)
		err := setter(context.Background(), &target)
		assert.NoError(t, err)

		expected, _ := newChain(blocks...).MarshalBinary()
		assert.Equal(t, expected, target)
	})
	t.Run("typed", func(t *testing.T) {
		var target []*rsa.PublicKey
		setter := FromBlocks(blocks...)
		err := setter(context.Background(), &target)
		assert.NoError(t, err)

		var expected []*rsa.PublicKey
		for _, block := range newChain(blocks...).Public() {
			value, err := x509.ParsePKIXPublicKey(block.Bytes)
			require.NoError(t, err)
			expected = append(expected, value.(*rsa.PublicKey))
		}

		assert.Equal(t, expected, target)
	})
}

func TestFromBlocks_typedSliceAtomic(t *testing.T) {
	valid := testRSAPublicKeyBlock(t)
	malformed := &pem.Block{Type: "PUBLIC KEY", Bytes: []byte("malformed")}
	sentinel := &rsa.PublicKey{N: big.NewInt(1), E: 3}

	testCases := []struct {
		name          string
		blocks        []*pem.Block
		target        []*rsa.PublicKey
		wantErr       bool
		wantLen       int
		wantNil       bool
		wantUnchanged bool
	}{
		{
			name:    "mixed_valid_and_malformed_preserves_nil",
			blocks:  []*pem.Block{valid, malformed},
			wantErr: true,
			wantNil: true,
		},
		{
			name:          "mixed_valid_and_malformed_preserves_existing",
			blocks:        []*pem.Block{valid, malformed},
			target:        []*rsa.PublicKey{sentinel},
			wantErr:       true,
			wantUnchanged: true,
		},
		{
			name:    "nil_block_is_ignored",
			blocks:  []*pem.Block{nil, valid},
			wantLen: 1,
		},
		{
			name:          "nil_blocks_are_noop",
			blocks:        []*pem.Block{nil},
			target:        []*rsa.PublicKey{sentinel},
			wantLen:       1,
			wantUnchanged: true,
		},
		{
			name:          "unmatched_and_unsupported_blocks_are_noops",
			blocks:        []*pem.Block{{Type: "CERTIFICATE", Bytes: []byte("malformed")}, {Type: "UNKNOWN"}},
			target:        []*rsa.PublicKey{sentinel},
			wantLen:       1,
			wantUnchanged: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			target := tc.target
			err := FromBlocks(tc.blocks...)(t.Context(), &target)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			if tc.wantUnchanged {
				require.Len(t, target, 1)
				assert.Same(t, sentinel, target[0])
				return
			}
			if tc.wantNil {
				assert.Nil(t, target)
				return
			}
			assert.Len(t, target, tc.wantLen)
		})
	}
}

func TestFromBlocks_pointerToSlice(t *testing.T) {
	valid := testRSAPublicKeyBlock(t)
	malformed := &pem.Block{Type: "PUBLIC KEY", Bytes: []byte("malformed")}

	testCases := []struct {
		name    string
		blocks  []*pem.Block
		wantErr bool
		wantNil bool
	}{
		{
			name:    "malformed_block_does_not_allocate",
			blocks:  []*pem.Block{valid, malformed},
			wantErr: true,
			wantNil: true,
		},
		{
			name:    "unmatched_blocks_do_not_allocate",
			blocks:  []*pem.Block{{Type: "UNKNOWN"}},
			wantNil: true,
		},
		{
			name:   "valid_block_allocates_and_sets",
			blocks: []*pem.Block{valid},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var target *[]*rsa.PublicKey
			err := FromBlocks(tc.blocks...)(t.Context(), &target)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			if tc.wantNil {
				assert.Nil(t, target)
				return
			}
			require.NotNil(t, target)
			assert.Len(t, *target, 1)
		})
	}
}

func TestFromBlocks_sliceValuedKeys(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	chain, err := newTestChain(privateKey, publicKey, nil)
	require.NoError(t, err)

	t.Run("public_keys", func(t *testing.T) {
		var target []ed25519.PublicKey

		err := FromBlocks(chain.blocks...)(t.Context(), &target)

		require.NoError(t, err)
		require.Len(t, target, 1)
		assert.Equal(t, publicKey, target[0])
	})

	t.Run("private_keys", func(t *testing.T) {
		var target []ed25519.PrivateKey

		err := FromBlocks(chain.blocks...)(t.Context(), &target)

		require.NoError(t, err)
		require.Len(t, target, 1)
		assert.Equal(t, privateKey, target[0])
	})

	t.Run("public_key_error_is_atomic", func(t *testing.T) {
		sentinel := ed25519.PublicKey{1, 2, 3}
		target := []ed25519.PublicKey{sentinel}
		blocks := append(slices.Clone(chain.Public()), &pem.Block{Type: "PUBLIC KEY", Bytes: []byte("malformed")})

		err := FromBlocks(blocks...)(t.Context(), &target)

		require.Error(t, err)
		assert.Equal(t, []ed25519.PublicKey{sentinel}, target)
	})

	t.Run("private_key_error_is_atomic", func(t *testing.T) {
		sentinel := ed25519.PrivateKey{1, 2, 3}
		target := []ed25519.PrivateKey{sentinel}
		blocks := append(slices.Clone(chain.Private()), &pem.Block{Type: "PRIVATE KEY", Bytes: []byte("malformed")})

		err := FromBlocks(blocks...)(t.Context(), &target)

		require.Error(t, err)
		assert.Equal(t, []ed25519.PrivateKey{sentinel}, target)
	})
}

func TestFromBlocks_pointerToBytes(t *testing.T) {
	block := testRSAPublicKeyBlock(t)
	expected, err := newChain(block).MarshalBinary()
	require.NoError(t, err)

	t.Run("valid_block_allocates", func(t *testing.T) {
		var target *[]byte

		err := FromBlocks(block)(t.Context(), &target)

		require.NoError(t, err)
		require.NotNil(t, target)
		assert.Equal(t, expected, *target)
	})

	t.Run("valid_block_preserves_pointer", func(t *testing.T) {
		value := []byte("old")
		target := &value
		original := target

		err := FromBlocks(block)(t.Context(), &target)

		require.NoError(t, err)
		assert.Same(t, original, target)
		assert.Equal(t, expected, *target)
	})

	t.Run("empty_blocks_do_not_allocate", func(t *testing.T) {
		var target *[]byte

		err := FromBlocks()(t.Context(), &target)

		require.NoError(t, err)
		assert.Nil(t, target)
	})

	t.Run("unknown_blocks_do_not_allocate", func(t *testing.T) {
		var target *[]byte

		err := FromBlocks(&pem.Block{Type: "UNKNOWN"})(t.Context(), &target)

		require.NoError(t, err)
		assert.Nil(t, target)
	})
}

func TestPEM_detectType(t *testing.T) {
	testCases := []struct {
		name     string
		target   reflect.Value
		expected reflect.Type
	}{
		{"rsa_public_key", reflect.ValueOf(new(rsa.PublicKey)), publicKeyType},
		{"ecda_public_key", reflect.ValueOf(new(ecdsa.PublicKey)), publicKeyType},
		{"ed25519_public_key", reflect.ValueOf(new(ed25519.PublicKey)), publicKeyType},
		{"ecdh_public_key", reflect.ValueOf(new(ecdh.PublicKey)), publicKeyType},

		{"rsa_private_key", reflect.ValueOf(new(rsa.PrivateKey)), privateKeyType},
		{"ecda_private_key", reflect.ValueOf(new(ecdsa.PrivateKey)), privateKeyType},
		{"ed25519_private_key", reflect.ValueOf(new(ed25519.PrivateKey)), privateKeyType},
		{"ecdh_private_key", reflect.ValueOf(new(ecdh.PrivateKey)), privateKeyType},

		{"certificate", reflect.ValueOf(x509.Certificate{}), certificateType},

		{"certificate_pool", reflect.ValueOf(new(x509.CertPool)), nil},
		{"int", reflect.ValueOf(42), nil},
		{"float", reflect.ValueOf(42.0), nil},
		{"bool", reflect.ValueOf(true), nil},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, pemType(tc.target.Type()))
		})
	}
}

func testRSAPublicKeyBlock(t *testing.T) *pem.Block {
	t.Helper()

	bytes, err := x509.MarshalPKIXPublicKey(&rsa.PublicKey{N: big.NewInt(3), E: 65537})
	require.NoError(t, err)

	return &pem.Block{Type: "PUBLIC KEY", Bytes: bytes}
}

func testStructPEMBlocks(t *testing.T) (*rsa.PrivateKey, *x509.Certificate, []*pem.Block) {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	certificate := newTestCertificate()
	chain, err := newTestChain(privateKey, &privateKey.PublicKey, certificate)
	require.NoError(t, err)
	return privateKey, certificate, chain.blocks
}

func newTestChain(privkey PrivateKey, pubkey PublicKey, cert *x509.Certificate) (*chain, error) {
	var blocks []*pem.Block

	if privkey != nil {
		privder, err := x509.MarshalPKCS8PrivateKey(privkey)
		if err != nil {
			return nil, fmt.Errorf("cannot encode private key to PEM: %w", err)
		}
		blocks = append(blocks, &pem.Block{Type: "PRIVATE KEY", Bytes: privder})
	}

	if pubkey != nil {
		pubder, err := x509.MarshalPKIXPublicKey(pubkey)
		if err != nil {
			return nil, fmt.Errorf("cannot encode public key to PEM: %w", err)
		}
		blocks = append(blocks, &pem.Block{Type: "PUBLIC KEY", Bytes: pubder})
	}

	if cert != nil {
		certb, err := x509.CreateCertificate(rand.Reader, cert, cert, pubkey, privkey)
		if err != nil {
			return nil, fmt.Errorf("cannot create certificate bytes: %w", err)
		}
		cert.Raw = certb

		blocks = append(blocks, &pem.Block{Type: "CERTIFICATE", Bytes: certb})
	}

	c := newChain(blocks...)
	return &c, nil
}

func newTestCertificate() *x509.Certificate {
	return &x509.Certificate{
		SerialNumber: big.NewInt(2019),
		Subject: pkix.Name{
			Organization:  []string{"Company, INC."},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{"San Francisco"},
			StreetAddress: []string{"Golden Gate Bridge"},
			PostalCode:    []string{"94016"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
}
