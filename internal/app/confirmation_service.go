package app

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/jessevaughan/increasex/internal/config"
	"github.com/jessevaughan/increasex/internal/util"
)

type ConfirmationService struct {
	ttl time.Duration
}

type confirmationPayload struct {
	Operation   string `json:"operation"`
	PayloadHash string `json:"payload_hash"`
	Profile     string `json:"profile"`
	Environment string `json:"environment"`
	IssuedAt    int64  `json:"issued_at"`
}

func NewConfirmationService() ConfirmationService {
	return ConfirmationService{ttl: 5 * time.Minute}
}

func (s ConfirmationService) Generate(operation string, session Session, payload any) (string, error) {
	key, err := confirmationKey()
	if err != nil {
		return "", err
	}
	body := confirmationPayload{
		Operation:   operation,
		PayloadHash: hashPayload(payload),
		Profile:     session.ProfileName,
		Environment: session.Environment,
		IssuedAt:    time.Now().UTC().Unix(),
	}
	plain, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	sig := sign(key, plain)
	return base64.RawURLEncoding.EncodeToString(plain) + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}

func (s ConfirmationService) Verify(token, operation string, session Session, payload any) error {
	if token == "" {
		return util.NewError(util.CodeConfirmationRequired, "confirmation token is required", nil, false)
	}
	key, err := confirmationKey()
	if err != nil {
		return err
	}
	parts := splitToken(token)
	if len(parts) != 2 {
		return util.NewError(util.CodeConfirmationInvalid, "malformed confirmation token", nil, false)
	}
	plain, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return util.NewError(util.CodeConfirmationInvalid, "malformed confirmation token", nil, false)
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return util.NewError(util.CodeConfirmationInvalid, "malformed confirmation token", nil, false)
	}
	if !hmac.Equal(sig, sign(key, plain)) {
		return util.NewError(util.CodeConfirmationInvalid, "invalid confirmation token signature", nil, false)
	}
	var parsed confirmationPayload
	if err := json.Unmarshal(plain, &parsed); err != nil {
		return util.NewError(util.CodeConfirmationInvalid, "malformed confirmation token payload", nil, false)
	}
	if parsed.Operation != operation || parsed.Profile != session.ProfileName || parsed.Environment != session.Environment {
		return util.NewError(util.CodeConfirmationInvalid, "confirmation token context mismatch", nil, false)
	}
	if parsed.PayloadHash != hashPayload(payload) {
		return util.NewError(util.CodeConfirmationInvalid, "confirmation token payload mismatch", nil, false)
	}
	if time.Since(time.Unix(parsed.IssuedAt, 0)) > s.ttl {
		return util.NewError(util.CodeConfirmationInvalid, "confirmation token has expired", nil, false)
	}
	return nil
}

func hashPayload(payload any) string {
	raw, _ := json.Marshal(payload)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func sign(key, data []byte) []byte {
	m := hmac.New(sha256.New, key)
	_, _ = m.Write(data)
	return m.Sum(nil)
}

func splitToken(token string) []string {
	for i := 0; i < len(token); i++ {
		if token[i] == '.' {
			return []string{token[:i], token[i+1:]}
		}
	}
	return nil
}

func confirmationKey() ([]byte, error) {
	dir, err := config.ConfigDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "confirmation.key")
	if data, err := os.ReadFile(path); err == nil && len(data) > 0 {
		return data, nil
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	data := make([]byte, 32)
	if _, err := rand.Read(data); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return nil, err
	}
	return data, nil
}

var errUnsupportedOperation = errors.New("unsupported operation")
