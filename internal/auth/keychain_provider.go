package auth

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/gitsoecode/increasex-cli-mcp/internal/config"
)

const keychainService = "increasex"
const keychainEncodingPrefix = "increasex-base64:"

var runSecurityCommand = func(args ...string) ([]byte, error) {
	return exec.Command("/usr/bin/security", args...).CombinedOutput()
}

type KeychainStore struct{}

func keychainAccount(profile config.Profile) string {
	if profile.KeychainAccount != "" {
		return profile.KeychainAccount
	}
	return fmt.Sprintf("%s:%s", profile.Name, profile.Environment)
}

func (s KeychainStore) Save(profile config.Profile, token string) (config.Profile, error) {
	account := keychainAccount(profile)
	encoded := keychainEncodingPrefix + base64.StdEncoding.EncodeToString([]byte(token))
	if out, err := runSecurityCommand("add-generic-password", "-U", "-s", keychainService, "-a", account, "-w", encoded); err != nil {
		return profile, fmt.Errorf("save keychain credential: %w: %s", err, strings.TrimSpace(string(out)))
	}
	profile.KeychainAccount = account
	credential, err := s.Load(profile)
	if err != nil {
		return profile, err
	}
	if credential == nil || credential.Token != token {
		return profile, errors.New("keychain credential verification failed after save")
	}
	profile.ConfigToken = ""
	return profile, nil
}

func (s KeychainStore) Load(profile config.Profile) (*StoredCredential, error) {
	account := keychainAccount(profile)
	out, err := runSecurityCommand("find-generic-password", "-s", keychainService, "-wa", account)
	if isKeychainNotFound(out, err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load keychain credential: %w: %s", err, strings.TrimSpace(string(out)))
	}
	value := strings.TrimSpace(string(out))
	if strings.HasPrefix(value, keychainEncodingPrefix) {
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(value, keychainEncodingPrefix))
		if err != nil {
			return nil, fmt.Errorf("decode keychain credential: %w", err)
		}
		value = string(decoded)
	}
	return &StoredCredential{Token: value, Source: string(config.StorageModeKeychain)}, nil
}

func (s KeychainStore) Delete(profile config.Profile) error {
	account := keychainAccount(profile)
	out, err := runSecurityCommand("delete-generic-password", "-s", keychainService, "-a", account)
	if isKeychainNotFound(out, err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("delete keychain credential: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func isKeychainNotFound(out []byte, err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(string(out))
	return strings.Contains(message, "could not be found") ||
		strings.Contains(message, "item could not be found") ||
		strings.Contains(message, "the specified item could not be found")
}
