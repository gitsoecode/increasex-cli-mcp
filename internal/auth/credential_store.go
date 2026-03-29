package auth

import "github.com/jessevaughan/increasex/internal/config"

type StoredCredential struct {
	Token  string
	Source string
}

type CredentialStore interface {
	Save(profile config.Profile, token string) (config.Profile, error)
	Load(profile config.Profile) (*StoredCredential, error)
	Delete(profile config.Profile) error
}
