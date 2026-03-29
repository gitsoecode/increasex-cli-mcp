package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

const (
	EnvSandbox    = "sandbox"
	EnvProduction = "production"
)

type StorageMode string

const (
	StorageModeAuto     StorageMode = "auto"
	StorageModeFile     StorageMode = "file"
	StorageModeKeychain StorageMode = "keychain"
	StorageModeConfig   StorageMode = "config"
)

type Profile struct {
	Name            string      `json:"name"`
	Environment     string      `json:"environment"`
	StorageMode     StorageMode `json:"storage_mode"`
	KeychainAccount string      `json:"keychain_account,omitempty"`
	// ConfigToken is retained for migration from the original inline secret storage.
	ConfigToken        string `json:"config_token,omitempty"`
	LastKnownEntity    string `json:"last_known_entity,omitempty"`
	LastKnownRequestID string `json:"last_known_request_id,omitempty"`
}

type CredentialEntry struct {
	Profile     string `json:"profile"`
	Environment string `json:"environment"`
	APIKey      string `json:"api_key"`
	UpdatedAt   string `json:"updated_at"`
	Source      string `json:"source,omitempty"`
}

type CredentialsFile struct {
	Profiles map[string]CredentialEntry `json:"profiles"`
}

type File struct {
	DefaultProfile string             `json:"default_profile"`
	Profiles       map[string]Profile `json:"profiles"`
}

func DefaultConfig() File {
	return File{
		DefaultProfile: "default",
		Profiles:       map[string]Profile{},
	}
}

func ConfigDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "increasex"), nil
}

func ConfigPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func CredentialsPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "credentials.json"), nil
}

func Load() (File, error) {
	path, err := ConfigPath()
	if err != nil {
		return File{}, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return DefaultConfig(), nil
	}
	if err != nil {
		return File{}, err
	}
	cfg := DefaultConfig()
	if err := json.Unmarshal(data, &cfg); err != nil {
		return File{}, err
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]Profile{}
	}
	if cfg.DefaultProfile == "" {
		cfg.DefaultProfile = "default"
	}
	for name, profile := range cfg.Profiles {
		profile.StorageMode = NormalizeStorageMode(profile.StorageMode)
		if profile.Name == "" {
			profile.Name = name
		}
		if profile.Environment == "" {
			profile.Environment = EnvSandbox
		}
		cfg.Profiles[name] = profile
	}
	return cfg, nil
}

func Save(cfg File) error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func DefaultCredentialsFile() CredentialsFile {
	return CredentialsFile{
		Profiles: map[string]CredentialEntry{},
	}
}

func LoadCredentials() (CredentialsFile, error) {
	path, err := CredentialsPath()
	if err != nil {
		return CredentialsFile{}, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return DefaultCredentialsFile(), nil
	}
	if err != nil {
		return CredentialsFile{}, err
	}
	credentials := DefaultCredentialsFile()
	if err := json.Unmarshal(data, &credentials); err != nil {
		return CredentialsFile{}, err
	}
	if credentials.Profiles == nil {
		credentials.Profiles = map[string]CredentialEntry{}
	}
	return credentials, nil
}

func SaveCredentials(credentials CredentialsFile) error {
	path, err := CredentialsPath()
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(credentials, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func NormalizeStorageMode(mode StorageMode) StorageMode {
	switch mode {
	case "", StorageModeAuto:
		return StorageModeAuto
	case StorageModeFile, StorageModeConfig:
		return StorageModeFile
	case StorageModeKeychain:
		return StorageModeKeychain
	default:
		return StorageModeAuto
	}
}
