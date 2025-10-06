package tests

import (
	"testing"
	"time"

	"github.com/dbehnke/allstar-nexus/backend/auth"
)

func TestJWTGenerateParse(t *testing.T) {
	secret := "test-secret"
	tok, err := auth.GenerateJWT("user@example.com", "user", time.Minute, secret)
	if err != nil {
		t.Fatalf("generate error: %v", err)
	}
	email, role, exp, err := auth.ParseJWT(tok, secret)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if email != "user@example.com" || role != "user" {
		t.Fatalf("unexpected claims: %s %s", email, role)
	}
	if time.Until(exp) <= 0 {
		t.Fatalf("expiry not in future")
	}
}

func TestJWTBadSignature(t *testing.T) {
	secret := "test-secret"
	tok, _ := auth.GenerateJWT("x@y.z", "user", time.Minute, secret)
	// modify last char to corrupt signature
	if tok[len(tok)-1] == 'a' {
		tok = tok[:len(tok)-1] + "b"
	} else {
		tok = tok[:len(tok)-1] + "a"
	}
	_, _, _, err := auth.ParseJWT(tok, secret)
	if err == nil {
		t.Fatalf("expected error for bad signature")
	}
}
