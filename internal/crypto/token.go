// Package crypto wraps standard cryptographic primitives behind small functions with fixed parameters.
package crypto

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
)

const (
	// tokenSignatureLen is the number of raw HMAC bytes kept in a signed token (128 bits).
	tokenSignatureLen = 16
)

// GenerateToken returns a cryptographically random, URL-safe token together with its SHA-256 hash.
func GenerateToken(numBytes int) (token string, hash []byte, err error) {
	b := make([]byte, numBytes)
	if _, err = rand.Read(b); err != nil {
		return "", nil, fmt.Errorf("generating token: %w", err)
	}
	token = base64.RawURLEncoding.EncodeToString(b)
	sum := sha256.Sum256([]byte(token))
	return token, sum[:], nil
}

// HashToken returns the SHA-256 hash of a token, matching the hash produced by GenerateToken.
func HashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

// SHA256Hex returns the lowercase hex-encoded SHA-256 digest of data.
func SHA256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// GenerateRandomCode generates a random alphanumeric code of the given length.
// Bytes are drawn from crypto/rand and values in the final, partial block of 256
// are rejected and re-drawn, so every character in the charset is equally likely.
func GenerateRandomCode(length int) (string, error) {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	threshold := 256 - (256 % len(charset))

	code := make([]byte, length)
	buf := make([]byte, 1)
	for i := range code {
		for {
			if _, err := rand.Read(buf); err != nil {
				return "", err
			}
			if int(buf[0]) < threshold {
				code[i] = charset[int(buf[0])%len(charset)]
				break
			}
		}
	}
	return string(code), nil
}

// SignToken signs id under purpose with a key and returns an unguessable token embedding it,
// of the form "<id>.<hex signature>". purpose domain-separates tokens so the same key cannot be reused to forge
// a token signed for a different purpose. The id itself is not secret, only forging a valid signature without
// the key is meant to be infeasible.
func SignToken(key []byte, purpose string, id int) string {
	return fmt.Sprintf("%06d.%s", id, hex.EncodeToString(signID(key, purpose, id)))
}

// VerifyToken checks a token produced by SignToken against a key and purpose and returns the id it was signed for.
// ok is false if the token is malformed or its signature does not match, in which case id is meaningless.
func VerifyToken(key []byte, purpose string, token string) (id int, ok bool) {
	idPart, sigPart, found := strings.Cut(token, ".")
	if !found {
		return 0, false
	}

	id, err := strconv.Atoi(idPart)
	if err != nil {
		return 0, false
	}

	signature, err := hex.DecodeString(sigPart)
	if err != nil {
		return 0, false
	}

	if !hmac.Equal(signature, signID(key, purpose, id)) {
		return 0, false
	}

	return id, true
}

// signID computes the truncated HMAC-SHA256 of id under a key, domain-separated by purpose.
func signID(key []byte, purpose string, id int) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = fmt.Fprintf(mac, "%s:%d", purpose, id)
	return mac.Sum(nil)[:tokenSignatureLen]
}
