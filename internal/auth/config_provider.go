package auth

import (
	"time"

	"github.com/jessevaughan/increasex/internal/config"
)

type FileStore struct{}

func (s FileStore) Save(profile config.Profile, token string) (config.Profile, error) {
	credentials, err := config.LoadCredentials()
	if err != nil {
		return profile, err
	}
	credentials.Profiles[profile.Name] = config.CredentialEntry{
		Profile:     profile.Name,
		Environment: profile.Environment,
		APIKey:      token,
		UpdatedAt:   time.Now().UTC().Format(time.RFC3339),
		Source:      string(config.NormalizeStorageMode(profile.StorageMode)),
	}
	profile.ConfigToken = ""
	return profile, config.SaveCredentials(credentials)
}

func (s FileStore) Load(profile config.Profile) (*StoredCredential, error) {
	credentials, err := config.LoadCredentials()
	if err != nil {
		return nil, err
	}
	entry, ok := credentials.Profiles[profile.Name]
	if !ok || entry.APIKey == "" {
		return nil, nil
	}
	return &StoredCredential{Token: entry.APIKey, Source: string(config.StorageModeFile)}, nil
}

func (s FileStore) Delete(profile config.Profile) error {
	credentials, err := config.LoadCredentials()
	if err != nil {
		return err
	}
	delete(credentials.Profiles, profile.Name)
	return config.SaveCredentials(credentials)
}
