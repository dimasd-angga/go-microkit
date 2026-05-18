package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestSigner_RoundTrip(t *testing.T) {
	s := NewSigner("test-secret-do-not-use", time.Minute, time.Hour)
	uid := uuid.New()

	pair, err := s.Issue(uid, "a@b.com", RoleUser)
	if err != nil {
		t.Fatal(err)
	}
	if pair.Access == "" || pair.Refresh == "" {
		t.Fatal("expected non-empty tokens")
	}

	c, err := s.Verify(pair.Access)
	if err != nil {
		t.Fatalf("verify access: %v", err)
	}
	if c.UserID != uid {
		t.Errorf("uid mismatch: %v != %v", c.UserID, uid)
	}
	if c.Email != "a@b.com" {
		t.Errorf("email mismatch: got %q", c.Email)
	}
}

func TestSigner_RejectsBadSecret(t *testing.T) {
	a := NewSigner("secret-a", time.Minute, time.Hour)
	b := NewSigner("secret-b", time.Minute, time.Hour)

	pair, _ := a.Issue(uuid.New(), "x@y.com", RoleUser)
	if _, err := b.Verify(pair.Access); err == nil {
		t.Fatal("expected verify failure with wrong secret")
	}
}
