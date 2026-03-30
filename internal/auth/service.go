package auth

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/gitsoecode/increasex-cli-mcp/internal/config"
	"github.com/gitsoecode/increasex-cli-mcp/internal/util"
)

type ResolveInput struct {
	ProfileName string
	Environment string
	APIKey      string
}

type ResolvedAuth struct {
	ProfileName string
	Environment string
	Token       string
	TokenSource string
}

type LoginResult struct {
	Profile           config.Profile `json:"profile"`
	FileSaved         bool           `json:"file_saved"`
	KeychainMirrored  bool           `json:"keychain_mirrored"`
	KeychainAvailable bool           `json:"keychain_available"`
	Warnings          []string       `json:"warnings,omitempty"`
	MCPReady          bool           `json:"mcp_ready"`
}

type StatusResult struct {
	Profile                 config.Profile `json:"profile"`
	FileCredentialAvailable bool           `json:"file_credential_available"`
	KeychainCredentialAvail bool           `json:"keychain_credential_available"`
	PreferredRuntimeSource  string         `json:"preferred_runtime_source,omitempty"`
	MCPReady                bool           `json:"mcp_ready"`
	CredentialError         string         `json:"credential_error,omitempty"`
	Warnings                []string       `json:"warnings,omitempty"`
}

type ProfileSummary struct {
	Profile                 config.Profile `json:"profile"`
	IsDefault               bool           `json:"is_default"`
	FileCredentialAvailable bool           `json:"file_credential_available"`
	KeychainCredentialAvail bool           `json:"keychain_credential_available"`
	PreferredRuntimeSource  string         `json:"preferred_runtime_source,omitempty"`
	MCPReady                bool           `json:"mcp_ready"`
	CredentialError         string         `json:"credential_error,omitempty"`
	Warnings                []string       `json:"warnings,omitempty"`
}

type LoginInput struct {
	ProfileName string
	Environment string
	APIKey      string
	StorageMode config.StorageMode
}

type Service struct {
	keychain KeychainStore
	file     FileStore
}

func NewService() Service {
	return Service{
		keychain: KeychainStore{},
		file:     FileStore{},
	}
}

func (s Service) SaveLogin(input LoginInput) (LoginResult, error) {
	cfg, err := config.Load()
	if err != nil {
		return LoginResult{}, err
	}
	apiKey := normalizeAPIKey(input.APIKey)
	name := strings.TrimSpace(input.ProfileName)
	if name == "" {
		name = "default"
	}
	storageMode := config.NormalizeStorageMode(input.StorageMode)
	profile := config.Profile{
		Name:        name,
		Environment: normalizeEnv(input.Environment),
		StorageMode: storageMode,
	}
	result := LoginResult{
		Profile:           profile,
		KeychainAvailable: true,
	}
	switch storageMode {
	case config.StorageModeAuto:
		profile, err = s.file.Save(profile, apiKey)
		if err != nil {
			return LoginResult{}, err
		}
		result.FileSaved = true
		result.MCPReady = true
		if mirroredProfile, mirrorErr := s.keychain.Save(profile, apiKey); mirrorErr != nil {
			result.KeychainAvailable = false
			result.Warnings = append(result.Warnings, fmt.Sprintf("Keychain mirror unavailable: %v", mirrorErr))
		} else {
			profile = mirroredProfile
			result.KeychainMirrored = true
		}
	case config.StorageModeFile:
		profile, err = s.file.Save(profile, apiKey)
		result.FileSaved = err == nil
		result.MCPReady = err == nil
	case config.StorageModeKeychain:
		profile, err = s.keychain.Save(profile, apiKey)
		result.KeychainMirrored = err == nil
		result.KeychainAvailable = err == nil
	default:
		err = fmt.Errorf("unsupported storage mode %q", storageMode)
	}
	if err != nil {
		return LoginResult{}, err
	}
	profile.StorageMode = storageMode
	cfg.Profiles[name] = profile
	cfg.DefaultProfile = name
	if err := config.Save(cfg); err != nil {
		return LoginResult{}, err
	}
	result.Profile = profile
	return result, nil
}

func (s Service) Resolve(input ResolveInput) (*ResolvedAuth, error) {
	env := normalizeEnv(firstNonEmpty(input.Environment, os.Getenv("INCREASEX_ENV")))
	if apiKey := normalizeAPIKey(input.APIKey); apiKey != "" {
		return &ResolvedAuth{
			ProfileName: firstNonEmpty(input.ProfileName, os.Getenv("INCREASEX_PROFILE"), "default"),
			Environment: env,
			Token:       apiKey,
			TokenSource: "flag",
		}, nil
	}
	if envToken := normalizeAPIKey(LoadEnvToken()); envToken != "" {
		return &ResolvedAuth{
			ProfileName: firstNonEmpty(input.ProfileName, os.Getenv("INCREASEX_PROFILE"), "default"),
			Environment: env,
			Token:       envToken,
			TokenSource: "env",
		}, nil
	}

	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	profileName := firstNonEmpty(input.ProfileName, os.Getenv("INCREASEX_PROFILE"), cfg.DefaultProfile, "default")
	profile, ok := cfg.Profiles[profileName]
	if !ok {
		return nil, util.NewError(util.CodeAuthError, "no credentials found for the selected profile", map[string]any{
			"profile": profileName,
		}, false)
	}
	if input.Environment != "" {
		profile.Environment = normalizeEnv(input.Environment)
	}
	if profile.Environment == "" {
		profile.Environment = env
	}
	profile.StorageMode = config.NormalizeStorageMode(profile.StorageMode)

	if credential, err := s.loadFileCredential(profile); err != nil {
		return nil, err
	} else if credential != nil {
		return &ResolvedAuth{
			ProfileName: profileName,
			Environment: profile.Environment,
			Token:       normalizeAPIKey(credential.Token),
			TokenSource: credential.Source,
		}, nil
	}

	if profile.StorageMode == config.StorageModeAuto || profile.StorageMode == config.StorageModeKeychain {
		cred, err := s.keychain.Load(profile)
		if err != nil {
			return nil, err
		}
		if cred != nil {
			token := normalizeAPIKey(cred.Token)
			if profile.StorageMode == config.StorageModeAuto {
				if _, fileErr := s.file.Save(profile, token); fileErr == nil {
					cfgProfile := cfg.Profiles[profileName]
					cfgProfile.StorageMode = config.StorageModeAuto
					cfgProfile.ConfigToken = ""
					cfg.Profiles[profileName] = cfgProfile
					_ = config.Save(cfg)
				}
			}
			return &ResolvedAuth{
				ProfileName: profileName,
				Environment: profile.Environment,
				Token:       token,
				TokenSource: cred.Source,
			}, nil
		}
	}

	return nil, util.NewError(util.CodeAuthError, "no stored credentials are available for the selected profile", map[string]any{
		"profile": profileName,
	}, false)
}

func (s Service) Logout(profileName string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	name := firstNonEmpty(profileName, cfg.DefaultProfile, "default")
	profile, ok := cfg.Profiles[name]
	if !ok {
		return nil
	}
	if err := s.file.Delete(profile); err != nil {
		return err
	}
	if err := s.keychain.Delete(profile); err != nil {
		return err
	}
	delete(cfg.Profiles, name)
	if cfg.DefaultProfile == name {
		cfg.DefaultProfile = "default"
	}
	return config.Save(cfg)
}

func (s Service) Status(profileName string) (StatusResult, error) {
	cfg, err := config.Load()
	if err != nil {
		return StatusResult{}, err
	}
	name := firstNonEmpty(profileName, cfg.DefaultProfile, "default")
	profile, ok := cfg.Profiles[name]
	if !ok {
		return StatusResult{}, util.NewError(util.CodeAuthError, "profile not found", map[string]any{"profile": name}, false)
	}
	return s.statusForProfile(profile)
}

func (s Service) ListProfiles() ([]ProfileSummary, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(cfg.Profiles))
	for name := range cfg.Profiles {
		names = append(names, name)
	}
	sort.Strings(names)

	summaries := make([]ProfileSummary, 0, len(names))
	for _, name := range names {
		profile := cfg.Profiles[name]
		summary, err := s.summaryForProfile(profile, name == cfg.DefaultProfile)
		if err != nil {
			return nil, err
		}
		summaries = append(summaries, summary)
	}
	return summaries, nil
}

func (s Service) UseProfile(profileName string) (ProfileSummary, error) {
	name := strings.TrimSpace(profileName)
	if name == "" {
		return ProfileSummary{}, util.NewError(util.CodeValidationError, "profile name is required", nil, false)
	}

	cfg, err := config.Load()
	if err != nil {
		return ProfileSummary{}, err
	}
	profile, ok := cfg.Profiles[name]
	if !ok {
		return ProfileSummary{}, util.NewError(util.CodeAuthError, "profile not found", map[string]any{"profile": name}, false)
	}
	summary, err := s.summaryForProfile(profile, name == cfg.DefaultProfile)
	if err != nil {
		return ProfileSummary{}, err
	}
	if !summary.FileCredentialAvailable && !summary.KeychainCredentialAvail {
		return ProfileSummary{}, util.NewError(util.CodeAuthError, "no readable stored credentials are available for the selected profile", map[string]any{
			"profile": name,
		}, false)
	}
	if summary.IsDefault {
		return summary, nil
	}
	cfg.DefaultProfile = name
	if err := config.Save(cfg); err != nil {
		return ProfileSummary{}, err
	}
	summary.IsDefault = true
	return summary, nil
}

func (s Service) statusForProfile(profile config.Profile) (StatusResult, error) {
	profile.StorageMode = config.NormalizeStorageMode(profile.StorageMode)
	result := StatusResult{Profile: profile}
	fileCredential, fileErr := s.loadFileCredential(profile)
	if fileErr != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("file credential check failed: %v", fileErr))
	}
	if fileCredential != nil && normalizeAPIKey(fileCredential.Token) != "" {
		result.FileCredentialAvailable = true
	}
	keychainCredential, keychainErr := s.keychain.Load(profile)
	if keychainErr != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("keychain credential check failed: %v", keychainErr))
	}
	if keychainCredential != nil && normalizeAPIKey(keychainCredential.Token) != "" {
		result.KeychainCredentialAvail = true
	}
	switch {
	case result.FileCredentialAvailable:
		result.PreferredRuntimeSource = string(config.StorageModeFile)
		result.MCPReady = true
	case result.KeychainCredentialAvail:
		result.PreferredRuntimeSource = string(config.StorageModeKeychain)
		result.Warnings = append(result.Warnings, "keychain credential is available, but MCP durability depends on a file credential")
	default:
		result.CredentialError = "no readable stored credentials are available for the selected profile"
	}
	return result, nil
}

func (s Service) summaryForProfile(profile config.Profile, isDefault bool) (ProfileSummary, error) {
	status, err := s.statusForProfile(profile)
	if err != nil {
		return ProfileSummary{}, err
	}
	return ProfileSummary{
		Profile:                 status.Profile,
		IsDefault:               isDefault,
		FileCredentialAvailable: status.FileCredentialAvailable,
		KeychainCredentialAvail: status.KeychainCredentialAvail,
		PreferredRuntimeSource:  status.PreferredRuntimeSource,
		MCPReady:                status.MCPReady,
		CredentialError:         status.CredentialError,
		Warnings:                append([]string(nil), status.Warnings...),
	}, nil
}

func (s Service) Export(input ResolveInput) (map[string]string, error) {
	resolved, err := s.Resolve(input)
	if err != nil {
		return nil, err
	}
	return map[string]string{
		"INCREASE_API_KEY":  resolved.Token,
		"INCREASEX_ENV":     resolved.Environment,
		"INCREASEX_PROFILE": resolved.ProfileName,
	}, nil
}

func (s Service) loadFileCredential(profile config.Profile) (*StoredCredential, error) {
	credential, err := s.file.Load(profile)
	if err != nil {
		return nil, err
	}
	if credential != nil {
		return credential, nil
	}
	if normalizeAPIKey(profile.ConfigToken) == "" {
		return nil, nil
	}
	migratedProfile := profile
	migratedProfile.StorageMode = config.StorageModeFile
	if _, err := s.file.Save(migratedProfile, normalizeAPIKey(profile.ConfigToken)); err != nil {
		return nil, err
	}

	cfg, err := config.Load()
	if err == nil {
		cfgProfile, ok := cfg.Profiles[profile.Name]
		if ok {
			cfgProfile.ConfigToken = ""
			cfgProfile.StorageMode = config.StorageModeFile
			cfg.Profiles[profile.Name] = cfgProfile
			_ = config.Save(cfg)
		}
	}
	return &StoredCredential{Token: normalizeAPIKey(profile.ConfigToken), Source: "migrated_from_config"}, nil
}

func normalizeAPIKey(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 {
		if (value[0] == '\'' && value[len(value)-1] == '\'') || (value[0] == '"' && value[len(value)-1] == '"') {
			value = strings.TrimSpace(value[1 : len(value)-1])
		}
	}
	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func normalizeEnv(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", config.EnvSandbox:
		return config.EnvSandbox
	case config.EnvProduction:
		return config.EnvProduction
	default:
		return config.EnvSandbox
	}
}
