package auth

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"github.com/jessevaughan/increasex/internal/config"
)

func TestKeychainStoreSaveVerifiesRoundTrip(t *testing.T) {
	original := runSecurityCommand
	t.Cleanup(func() {
		runSecurityCommand = original
	})

	var savedPassword string
	runSecurityCommand = func(args ...string) ([]byte, error) {
		switch args[0] {
		case "add-generic-password":
			savedPassword = args[len(args)-1]
			return []byte(""), nil
		case "find-generic-password":
			return []byte(savedPassword), nil
		default:
			t.Fatalf("unexpected keychain command: %v", args)
			return nil, nil
		}
	}

	profile, err := KeychainStore{}.Save(config.Profile{
		Name:        "default",
		Environment: config.EnvSandbox,
	}, "test-token")
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if profile.KeychainAccount != "default:sandbox" {
		t.Fatalf("Save() keychain account = %q, want default:sandbox", profile.KeychainAccount)
	}
	if !strings.HasPrefix(savedPassword, keychainEncodingPrefix) {
		t.Fatalf("Save() stored password prefix = %q, want %q", savedPassword, keychainEncodingPrefix)
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(savedPassword, keychainEncodingPrefix))
	if err != nil {
		t.Fatalf("DecodeString() error = %v", err)
	}
	if string(decoded) != "test-token" {
		t.Fatalf("decoded password = %q, want test-token", string(decoded))
	}
}

func TestKeychainStoreLoadReturnsNilWhenItemMissing(t *testing.T) {
	original := runSecurityCommand
	t.Cleanup(func() {
		runSecurityCommand = original
	})

	runSecurityCommand = func(args ...string) ([]byte, error) {
		return []byte("security: SecKeychainSearchCopyNext: The specified item could not be found in the keychain.\n"), errors.New("exit status 44")
	}

	credential, err := KeychainStore{}.Load(config.Profile{Name: "default", Environment: config.EnvSandbox})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if credential != nil {
		t.Fatalf("Load() credential = %#v, want nil", credential)
	}
}

func TestKeychainStoreSaveFailsWhenVerificationDoesNotMatch(t *testing.T) {
	original := runSecurityCommand
	t.Cleanup(func() {
		runSecurityCommand = original
	})

	runSecurityCommand = func(args ...string) ([]byte, error) {
		switch args[0] {
		case "add-generic-password":
			return []byte(""), nil
		case "find-generic-password":
			return []byte(keychainEncodingPrefix + base64.StdEncoding.EncodeToString([]byte("wrong-token"))), nil
		default:
			t.Fatalf("unexpected keychain command: %v", args)
			return nil, nil
		}
	}

	_, err := KeychainStore{}.Save(config.Profile{Name: "default", Environment: config.EnvSandbox}, "expected-token")
	if err == nil || !strings.Contains(err.Error(), "verification failed") {
		t.Fatalf("Save() error = %v, want verification failure", err)
	}
}
