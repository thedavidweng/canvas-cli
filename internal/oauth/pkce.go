// Package oauth contains protocol helpers shared by Canvas OAuth commands.
package oauth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

// PKCE is the verifier and S256 challenge for one authorization request.
type PKCE struct {
	Verifier  string
	Challenge string
}

// NewPKCE creates a verifier with 256 bits of entropy and its RFC 7636 S256 challenge.
func NewPKCE() (PKCE, error) {
	verifierBytes := make([]byte, 32)
	if _, err := rand.Read(verifierBytes); err != nil {
		return PKCE{}, err
	}
	verifier := base64.RawURLEncoding.EncodeToString(verifierBytes)
	return PKCE{Verifier: verifier, Challenge: Challenge(verifier)}, nil
}

// Challenge returns the RFC 7636 S256 challenge for verifier.
func Challenge(verifier string) string {
	digest := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(digest[:])
}
