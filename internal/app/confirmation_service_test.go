package app

import (
	"path/filepath"
	"testing"
	"time"
)

func TestConfirmationTokenRoundTrip(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tempHome, ".config"))
	service := NewConfirmationService()
	session := Session{ProfileName: "default", Environment: "sandbox"}
	payload := map[string]any{"account_id": "account_123", "amount_cents": 100}

	token, err := service.Generate("move_money_internal", session, payload)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if err := service.Verify(token, "move_money_internal", session, payload); err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
}

func TestConfirmationTokenRejectsChangedPayload(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tempHome, ".config"))
	service := NewConfirmationService()
	session := Session{ProfileName: "default", Environment: "sandbox"}
	original := map[string]any{"account_id": "account_123", "amount_cents": 100}
	modified := map[string]any{"account_id": "account_123", "amount_cents": 200}

	token, err := service.Generate("move_money_internal", session, original)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if err := service.Verify(token, "move_money_internal", session, modified); err == nil {
		t.Fatal("Verify() expected payload mismatch error")
	}
}

func TestConfirmationTokenExpires(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tempHome, ".config"))
	service := ConfirmationService{ttl: time.Nanosecond}
	session := Session{ProfileName: "default", Environment: "sandbox"}
	payload := map[string]any{"name": "Example"}

	token, err := service.Generate("create_account", session, payload)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	time.Sleep(time.Millisecond)
	if err := service.Verify(token, "create_account", session, payload); err == nil {
		t.Fatal("Verify() expected expiration error")
	}
}
