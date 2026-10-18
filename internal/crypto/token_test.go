package crypto_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/barthjs/photopi/internal/crypto"
)

const purpose = "test-purpose"

var key = []byte("test-signing-key")

func TestGenerateToken(t *testing.T) {
	t.Parallel()

	t.Run("returns a token with its hash", func(t *testing.T) {
		t.Parallel()

		token, hash, err := crypto.GenerateToken(32)

		require.NoError(t, err)
		assert.NotEmpty(t, token)
		assert.Len(t, hash, 32)
		assert.Equal(t, hash, crypto.HashToken(token), "HashToken must reproduce the hash from GenerateToken")
	})
}

func TestSHA256Hex(t *testing.T) {
	t.Parallel()

	t.Run("returns the hex digest", func(t *testing.T) {
		t.Parallel()

		// echo -n "hello world" | sha256sum
		expected := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"

		digest := crypto.SHA256Hex([]byte("hello world"))

		assert.Equal(t, expected, digest)
	})
}

func TestGenerateRandomCode(t *testing.T) {
	t.Parallel()

	t.Run("returns a code of the length from the alphabet", func(t *testing.T) {
		t.Parallel()

		code, err := crypto.GenerateRandomCode(10)

		require.NoError(t, err)
		assert.Len(t, code, 10)
		for _, char := range code {
			assert.True(t, strings.ContainsRune("ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789", char), "invalid character %c", char)
		}
	})
}

func TestSignToken(t *testing.T) {
	t.Parallel()

	t.Run("returns the padded id with the signature", func(t *testing.T) {
		t.Parallel()

		token := crypto.SignToken(key, purpose, 42)

		id, signature, found := strings.Cut(token, ".")
		assert.True(t, found)
		assert.Equal(t, "000042", id, "the id is zero-padded to six digits")
		assert.Len(t, signature, 32, "128 bits as hex")
	})

	t.Run("returns the same token for the same id", func(t *testing.T) {
		t.Parallel()

		token := crypto.SignToken(key, purpose, 7)

		assert.Equal(t, crypto.SignToken(key, purpose, 7), token)
	})

	t.Run("returns another token for another id", func(t *testing.T) {
		t.Parallel()

		token := crypto.SignToken(key, purpose, 7)

		assert.NotEqual(t, crypto.SignToken(key, purpose, 8), token)
	})
}

func TestVerifyToken(t *testing.T) {
	t.Parallel()

	valid := []struct {
		name string
		id   int
	}{
		{
			name: "returns a small id",
			id:   42,
		},
		{
			name: "returns a zero id",
			id:   0,
		},
		{
			name: "returns an id wider than the padding",
			id:   12345678,
		},
		{
			name: "returns a negative id",
			id:   -3,
		},
	}

	for _, tt := range valid {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			token := crypto.SignToken(key, purpose, tt.id)

			id, ok := crypto.VerifyToken(key, purpose, token)

			assert.True(t, ok, "VerifyToken(%q) rejected a valid token", token)
			assert.Equal(t, tt.id, id)
		})
	}

	token := crypto.SignToken(key, purpose, 42)
	signature := token[len("000042."):]
	tampered := map[string]string{
		"rejects a different id":                 "000043." + signature,
		"rejects a different key":                crypto.SignToken([]byte("other-key"), purpose, 42),
		"rejects a different purpose":            crypto.SignToken(key, "other-purpose", 42),
		"rejects a truncated signature":          token[:len(token)-2],
		"rejects a non-numeric id":               "abc." + signature,
		"rejects a non-hex signature":            "000042.zz" + signature[2:],
		"rejects garbage":                        "not-a-token",
		"rejects an empty token":                 "",
		"rejects a token without separator":      "000042",
		"rejects a separator only":               ".",
		"rejects the signature of another token": "000042." + crypto.SignToken(key, purpose, 1)[len("000001."):],
	}

	for name, tamperedToken := range tampered {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, ok := crypto.VerifyToken(key, purpose, tamperedToken)

			assert.False(t, ok, "VerifyToken(%q) accepted a tampered token", tamperedToken)
		})
	}
}
