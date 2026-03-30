package auth

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gitsoecode/increasex-cli-mcp/internal/config"
)

func TestResolvePrefersExplicitThenEnvThenStoredProfile(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tempHome, ".config"))
	t.Setenv("INCREASE_API_KEY", "")
	t.Setenv("INCREASEX_ENV", "")
	t.Setenv("INCREASEX_PROFILE", "")

	service := NewService()
	result, err := service.SaveLogin(LoginInput{
		ProfileName: "default",
		Environment: config.EnvSandbox,
		APIKey:      "stored-token",
		StorageMode: config.StorageModeFile,
	})
	if err != nil {
		t.Fatalf("SaveLogin() error = %v", err)
	}
	if !result.FileSaved || !result.MCPReady {
		t.Fatalf("SaveLogin() result = %#v, want file_saved and mcp_ready", result)
	}

	t.Setenv("INCREASE_API_KEY", "env-token")
	resolved, err := service.Resolve(ResolveInput{})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolved.Token != "env-token" || resolved.TokenSource != "env" {
		t.Fatalf("Resolve() env precedence = %#v, want env token", resolved)
	}

	resolved, err = service.Resolve(ResolveInput{APIKey: "flag-token"})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolved.Token != "flag-token" || resolved.TokenSource != "flag" {
		t.Fatalf("Resolve() explicit precedence = %#v, want flag token", resolved)
	}

	t.Setenv("INCREASE_API_KEY", "")
	resolved, err = service.Resolve(ResolveInput{})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolved.Token != "stored-token" || resolved.TokenSource != string(config.StorageModeFile) {
		t.Fatalf("Resolve() stored precedence = %#v, want file token", resolved)
	}
}

func TestSaveLoginAutoWritesFileAndToleratesKeychainFailure(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tempHome, ".config"))

	original := runSecurityCommand
	t.Cleanup(func() {
		runSecurityCommand = original
	})
	runSecurityCommand = func(args ...string) ([]byte, error) {
		if len(args) > 0 && args[0] == "add-generic-password" {
			return []byte("keychain unavailable"), errors.New("exit status 1")
		}
		return []byte("security: SecKeychainSearchCopyNext: The specified item could not be found in the keychain.\n"), errors.New("exit status 44")
	}

	service := NewService()
	result, err := service.SaveLogin(LoginInput{
		ProfileName: "default",
		Environment: config.EnvSandbox,
		APIKey:      "stored-token",
		StorageMode: config.StorageModeAuto,
	})
	if err != nil {
		t.Fatalf("SaveLogin() error = %v", err)
	}
	if !result.FileSaved || !result.MCPReady {
		t.Fatalf("SaveLogin() result = %#v, want file-backed MCP readiness", result)
	}
	if result.KeychainMirrored {
		t.Fatalf("SaveLogin() keychain_mirrored = true, want false")
	}
	if len(result.Warnings) == 0 {
		t.Fatalf("SaveLogin() warnings = %#v, want keychain warning", result.Warnings)
	}

	resolved, err := service.Resolve(ResolveInput{})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolved.Token != "stored-token" || resolved.TokenSource != string(config.StorageModeFile) {
		t.Fatalf("Resolve() = %#v, want file-backed token", resolved)
	}
}

func TestLogoutRemovesProfileAndCredentialFile(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tempHome, ".config"))

	service := NewService()
	_, err := service.SaveLogin(LoginInput{
		ProfileName: "default",
		Environment: config.EnvSandbox,
		APIKey:      "stored-token",
		StorageMode: config.StorageModeFile,
	})
	if err != nil {
		t.Fatalf("SaveLogin() error = %v", err)
	}

	if err := service.Logout("default"); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}
	if _, ok := cfg.Profiles["default"]; ok {
		t.Fatal("expected default profile to be removed")
	}
	credentials, err := config.LoadCredentials()
	if err != nil {
		t.Fatalf("config.LoadCredentials() error = %v", err)
	}
	if _, ok := credentials.Profiles["default"]; ok {
		t.Fatal("expected default credential to be removed")
	}
	if _, err := os.Stat(filepath.Join(tempHome, ".config")); err != nil && !os.IsNotExist(err) {
		t.Fatalf("unexpected config stat error: %v", err)
	}
}

func TestResolveNormalizesStoredQuotedToken(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tempHome, ".config"))
	t.Setenv("INCREASE_API_KEY", "")
	t.Setenv("INCREASEX_ENV", "")
	t.Setenv("INCREASEX_PROFILE", "")

	service := NewService()
	_, err := service.SaveLogin(LoginInput{
		ProfileName: "default",
		Environment: config.EnvSandbox,
		APIKey:      "'stored-token'",
		StorageMode: config.StorageModeFile,
	})
	if err != nil {
		t.Fatalf("SaveLogin() error = %v", err)
	}

	resolved, err := service.Resolve(ResolveInput{})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolved.Token != "stored-token" {
		t.Fatalf("Resolve() token = %q, want stored-token", resolved.Token)
	}
}

func TestResolveNormalizesEnvQuotedToken(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tempHome, ".config"))
	t.Setenv("INCREASE_API_KEY", "\"env-token\"")
	t.Setenv("INCREASEX_ENV", "")
	t.Setenv("INCREASEX_PROFILE", "")

	resolved, err := NewService().Resolve(ResolveInput{})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolved.Token != "env-token" {
		t.Fatalf("Resolve() token = %q, want env-token", resolved.Token)
	}
}

func TestStatusReportsFileBackedMCPReady(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tempHome, ".config"))

	service := NewService()
	_, err := service.SaveLogin(LoginInput{
		ProfileName: "default",
		Environment: config.EnvSandbox,
		APIKey:      "stored-token",
		StorageMode: config.StorageModeFile,
	})
	if err != nil {
		t.Fatalf("SaveLogin() error = %v", err)
	}

	status, err := service.Status("default")
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if !status.FileCredentialAvailable || !status.MCPReady {
		t.Fatalf("Status() = %#v, want file-backed MCP ready", status)
	}
	if status.PreferredRuntimeSource != string(config.StorageModeFile) {
		t.Fatalf("Status() preferred source = %q, want file", status.PreferredRuntimeSource)
	}
}

func TestResolveMigratesLegacyConfigToken(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tempHome, ".config"))
	t.Setenv("INCREASE_API_KEY", "")
	t.Setenv("INCREASEX_ENV", "")
	t.Setenv("INCREASEX_PROFILE", "")

	cfg := config.DefaultConfig()
	cfg.Profiles["default"] = config.Profile{
		Name:        "default",
		Environment: config.EnvSandbox,
		StorageMode: config.StorageModeConfig,
		ConfigToken: "legacy-token",
	}
	cfg.DefaultProfile = "default"
	if err := config.Save(cfg); err != nil {
		t.Fatalf("config.Save() error = %v", err)
	}

	service := NewService()
	resolved, err := service.Resolve(ResolveInput{})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if resolved.Token != "legacy-token" {
		t.Fatalf("Resolve() token = %q, want legacy-token", resolved.Token)
	}
	credentials, err := config.LoadCredentials()
	if err != nil {
		t.Fatalf("config.LoadCredentials() error = %v", err)
	}
	entry, ok := credentials.Profiles["default"]
	if !ok || entry.APIKey != "legacy-token" {
		t.Fatalf("credentials entry = %#v, want migrated legacy token", entry)
	}
	cfg, err = config.Load()
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}
	if cfg.Profiles["default"].ConfigToken != "" {
		t.Fatalf("ConfigToken = %q, want cleared", cfg.Profiles["default"].ConfigToken)
	}
	if cfg.Profiles["default"].StorageMode != config.StorageModeFile {
		t.Fatalf("StorageMode = %q, want file", cfg.Profiles["default"].StorageMode)
	}
}

func TestCredentialsFilePermissions(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tempHome, ".config"))

	service := NewService()
	_, err := service.SaveLogin(LoginInput{
		ProfileName: "default",
		Environment: config.EnvSandbox,
		APIKey:      "stored-token",
		StorageMode: config.StorageModeFile,
	})
	if err != nil {
		t.Fatalf("SaveLogin() error = %v", err)
	}

	dir, err := config.ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir() error = %v", err)
	}
	credentialsPath, err := config.CredentialsPath()
	if err != nil {
		t.Fatalf("CredentialsPath() error = %v", err)
	}
	dirInfo, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("os.Stat(config dir) error = %v", err)
	}
	if dirInfo.Mode().Perm() != 0o700 {
		t.Fatalf("config dir perms = %#o, want 0700", dirInfo.Mode().Perm())
	}
	fileInfo, err := os.Stat(credentialsPath)
	if err != nil {
		t.Fatalf("os.Stat(credentials file) error = %v", err)
	}
	if fileInfo.Mode().Perm() != 0o600 {
		t.Fatalf("credentials file perms = %#o, want 0600", fileInfo.Mode().Perm())
	}
}

func TestExportUsesResolvedCredential(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tempHome, ".config"))

	service := NewService()
	_, err := service.SaveLogin(LoginInput{
		ProfileName: "default",
		Environment: config.EnvSandbox,
		APIKey:      "stored-token",
		StorageMode: config.StorageModeFile,
	})
	if err != nil {
		t.Fatalf("SaveLogin() error = %v", err)
	}

	exports, err := service.Export(ResolveInput{})
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	if exports["INCREASE_API_KEY"] != "stored-token" {
		t.Fatalf("INCREASE_API_KEY = %q, want stored-token", exports["INCREASE_API_KEY"])
	}
	if exports["INCREASEX_PROFILE"] != "default" || exports["INCREASEX_ENV"] != config.EnvSandbox {
		t.Fatalf("exports = %#v, want profile/env", exports)
	}
}

func TestStatusWarnsWhenOnlyKeychainIsAvailable(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tempHome, ".config"))

	original := runSecurityCommand
	t.Cleanup(func() {
		runSecurityCommand = original
	})
	runSecurityCommand = func(args ...string) ([]byte, error) {
		switch args[0] {
		case "add-generic-password", "find-generic-password":
			return []byte("increasex-base64:c3RvcmVkLXRva2Vu"), nil
		case "delete-generic-password":
			return []byte(""), nil
		default:
			return nil, nil
		}
	}

	service := NewService()
	_, err := service.SaveLogin(LoginInput{
		ProfileName: "default",
		Environment: config.EnvSandbox,
		APIKey:      "stored-token",
		StorageMode: config.StorageModeKeychain,
	})
	if err != nil {
		t.Fatalf("SaveLogin() error = %v", err)
	}

	status, err := service.Status("default")
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if status.MCPReady {
		t.Fatalf("Status().MCPReady = true, want false without file credential")
	}
	if !status.KeychainCredentialAvail {
		t.Fatalf("Status() = %#v, want keychain credential available", status)
	}
	if len(status.Warnings) == 0 || !strings.Contains(status.Warnings[0], "MCP durability") {
		t.Fatalf("Status().Warnings = %#v, want MCP durability warning", status.Warnings)
	}
}

func TestListProfilesReturnsSortedSummariesWithDefaultMarker(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tempHome, ".config"))

	service := NewService()
	for _, input := range []LoginInput{
		{ProfileName: "sandbox", Environment: config.EnvSandbox, APIKey: "sandbox-token", StorageMode: config.StorageModeFile},
		{ProfileName: "prod", Environment: config.EnvProduction, APIKey: "prod-token", StorageMode: config.StorageModeFile},
	} {
		if _, err := service.SaveLogin(input); err != nil {
			t.Fatalf("SaveLogin(%q) error = %v", input.ProfileName, err)
		}
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}
	cfg.DefaultProfile = "sandbox"
	if err := config.Save(cfg); err != nil {
		t.Fatalf("config.Save() error = %v", err)
	}

	profiles, err := service.ListProfiles()
	if err != nil {
		t.Fatalf("ListProfiles() error = %v", err)
	}
	if len(profiles) != 2 {
		t.Fatalf("len(profiles) = %d, want 2", len(profiles))
	}
	if profiles[0].Profile.Name != "prod" || profiles[1].Profile.Name != "sandbox" {
		t.Fatalf("profiles order = %#v, want alphabetical ordering", profiles)
	}
	if profiles[0].IsDefault {
		t.Fatal("profiles[0].IsDefault = true, want false")
	}
	if !profiles[1].IsDefault {
		t.Fatal("profiles[1].IsDefault = false, want true")
	}
	if !profiles[1].FileCredentialAvailable || !profiles[1].MCPReady {
		t.Fatalf("profiles[1] = %#v, want file-backed MCP readiness", profiles[1])
	}
}

func TestUseProfileUpdatesDefaultProfile(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tempHome, ".config"))

	service := NewService()
	for _, input := range []LoginInput{
		{ProfileName: "sandbox", Environment: config.EnvSandbox, APIKey: "sandbox-token", StorageMode: config.StorageModeFile},
		{ProfileName: "prod", Environment: config.EnvProduction, APIKey: "prod-token", StorageMode: config.StorageModeFile},
	} {
		if _, err := service.SaveLogin(input); err != nil {
			t.Fatalf("SaveLogin(%q) error = %v", input.ProfileName, err)
		}
	}

	summary, err := service.UseProfile("sandbox")
	if err != nil {
		t.Fatalf("UseProfile() error = %v", err)
	}
	if !summary.IsDefault || summary.Profile.Name != "sandbox" {
		t.Fatalf("UseProfile() = %#v, want sandbox marked default", summary)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}
	if cfg.DefaultProfile != "sandbox" {
		t.Fatalf("DefaultProfile = %q, want sandbox", cfg.DefaultProfile)
	}
}

func TestUseProfileRejectsUnknownProfile(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tempHome, ".config"))

	service := NewService()
	if _, err := service.UseProfile("missing"); err == nil || !strings.Contains(err.Error(), "profile not found") {
		t.Fatalf("UseProfile() error = %v, want profile not found", err)
	}
}

func TestUseProfileRejectsProfilesWithoutReadableCredentials(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tempHome, ".config"))

	cfg := config.DefaultConfig()
	cfg.Profiles["empty"] = config.Profile{
		Name:        "empty",
		Environment: config.EnvSandbox,
		StorageMode: config.StorageModeFile,
	}
	cfg.DefaultProfile = "empty"
	if err := config.Save(cfg); err != nil {
		t.Fatalf("config.Save() error = %v", err)
	}

	service := NewService()
	if _, err := service.UseProfile("empty"); err == nil || !strings.Contains(err.Error(), "no readable stored credentials") {
		t.Fatalf("UseProfile() error = %v, want missing credential error", err)
	}
}

func TestUseProfileAllowsKeychainOnlyProfiles(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tempHome, ".config"))

	original := runSecurityCommand
	t.Cleanup(func() {
		runSecurityCommand = original
	})
	runSecurityCommand = func(args ...string) ([]byte, error) {
		switch args[0] {
		case "add-generic-password", "find-generic-password":
			return []byte("increasex-base64:c3RvcmVkLXRva2Vu"), nil
		case "delete-generic-password":
			return []byte(""), nil
		default:
			return nil, nil
		}
	}

	service := NewService()
	if _, err := service.SaveLogin(LoginInput{
		ProfileName: "sandbox",
		Environment: config.EnvSandbox,
		APIKey:      "stored-token",
		StorageMode: config.StorageModeKeychain,
	}); err != nil {
		t.Fatalf("SaveLogin() error = %v", err)
	}

	summary, err := service.UseProfile("sandbox")
	if err != nil {
		t.Fatalf("UseProfile() error = %v", err)
	}
	if !summary.KeychainCredentialAvail {
		t.Fatalf("UseProfile() = %#v, want keychain credential available", summary)
	}
	if summary.MCPReady {
		t.Fatalf("UseProfile().MCPReady = true, want false for keychain-only profile")
	}
	if len(summary.Warnings) == 0 || !strings.Contains(summary.Warnings[0], "MCP durability") {
		t.Fatalf("UseProfile().Warnings = %#v, want MCP durability warning", summary.Warnings)
	}
}

func TestUseProfileIsNoOpForAlreadyActiveProfile(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tempHome, ".config"))

	service := NewService()
	if _, err := service.SaveLogin(LoginInput{
		ProfileName: "sandbox",
		Environment: config.EnvSandbox,
		APIKey:      "stored-token",
		StorageMode: config.StorageModeFile,
	}); err != nil {
		t.Fatalf("SaveLogin() error = %v", err)
	}

	summary, err := service.UseProfile("sandbox")
	if err != nil {
		t.Fatalf("UseProfile() error = %v", err)
	}
	if !summary.IsDefault {
		t.Fatalf("UseProfile() = %#v, want active profile to remain default", summary)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}
	if cfg.DefaultProfile != "sandbox" {
		t.Fatalf("DefaultProfile = %q, want sandbox", cfg.DefaultProfile)
	}
}
