package oauth

import (
	"regexp"
	"testing"
)

func TestChallengeRFC7636Example(t *testing.T) {
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	const want = "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"
	if got := Challenge(verifier); got != want {
		t.Fatalf("Challenge() = %q, want %q", got, want)
	}
}

func TestNewPKCEUsesURLSafeVerifierAndMatchesChallenge(t *testing.T) {
	pkce, err := NewPKCE()
	if err != nil {
		t.Fatalf("NewPKCE(): %v", err)
	}
	if len(pkce.Verifier) != 43 {
		t.Fatalf("verifier length = %d, want 43", len(pkce.Verifier))
	}
	if !regexp.MustCompile(`^[A-Za-z0-9_-]+$`).MatchString(pkce.Verifier) {
		t.Fatalf("verifier contains a non-URL-safe character: %q", pkce.Verifier)
	}
	if pkce.Challenge != Challenge(pkce.Verifier) {
		t.Fatal("challenge does not match verifier")
	}
}
