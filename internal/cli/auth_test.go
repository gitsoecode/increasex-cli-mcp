package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gitsoecode/increasex-cli-mcp/internal/app"
	"github.com/gitsoecode/increasex-cli-mcp/internal/auth"
	"github.com/gitsoecode/increasex-cli-mcp/internal/config"
	"github.com/gitsoecode/increasex-cli-mcp/internal/ui"
)

func TestAuthExportSafetyMessaging(t *testing.T) {
	if got := authExportWarningText(); got != "Warning: auth export prints your raw API key to stdout." {
		t.Fatalf("authExportWarningText() = %q", got)
	}
	if got := authExportConfirmationPrompt(); got != "Export raw API credentials to stdout?" {
		t.Fatalf("authExportConfirmationPrompt() = %q", got)
	}
}

func TestAuthExportCommandHasConfirmFlag(t *testing.T) {
	cmd := newAuthExportCmd(&Context{Options: &RootOptions{}})
	flag := cmd.Flags().Lookup("confirm")
	if flag == nil {
		t.Fatal("auth export command should expose --confirm")
	}
	if flag.DefValue != "false" {
		t.Fatalf("--confirm default = %q, want false", flag.DefValue)
	}
}

func TestAuthUseCommandHasSwitchAlias(t *testing.T) {
	cmd := newAuthUseCmd(&Context{Options: &RootOptions{}})
	if len(cmd.Aliases) != 1 || cmd.Aliases[0] != "switch" {
		t.Fatalf("Aliases = %v, want [switch]", cmd.Aliases)
	}
}

func TestAuthLoginInteractiveDefaultsProfileNameFromEnvironment(t *testing.T) {
	for _, tc := range []struct {
		name   string
		env    string
		want   string
		apiKey string
	}{
		{name: "sandbox", env: config.EnvSandbox, want: "sandbox", apiKey: "sandbox-token"},
		{name: "production", env: config.EnvProduction, want: "prod", apiKey: "prod-token"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := authTestContext(t)
			ctx.Options.Interactive = true

			originalSelect := runPromptSelect
			originalString := runPromptString
			t.Cleanup(func() {
				runPromptSelect = originalSelect
				runPromptString = originalString
			})

			runPromptSelect = func(label string, options []ui.Option) (string, error) {
				if label != "Environment" {
					t.Fatalf("label = %q, want Environment", label)
				}
				return tc.env, nil
			}

			promptCalls := 0
			runPromptString = func(label string, required bool) (string, error) {
				promptCalls++
				switch promptCalls {
				case 1:
					if !strings.Contains(label, tc.want) {
						t.Fatalf("label = %q, want suggested profile %q", label, tc.want)
					}
					return "", nil
				case 2:
					if label != "Increase API key" {
						t.Fatalf("label = %q, want Increase API key", label)
					}
					return tc.apiKey, nil
				default:
					t.Fatalf("unexpected prompt call %d", promptCalls)
					return "", nil
				}
			}

			cmd := newAuthLoginCmd(ctx)
			cmd.SetContext(context.Background())
			if err := cmd.Execute(); err != nil {
				t.Fatalf("Execute() error = %v", err)
			}

			cfg, err := config.Load()
			if err != nil {
				t.Fatalf("config.Load() error = %v", err)
			}
			if cfg.DefaultProfile != tc.want {
				t.Fatalf("DefaultProfile = %q, want %q", cfg.DefaultProfile, tc.want)
			}
			if _, ok := cfg.Profiles[tc.want]; !ok {
				t.Fatalf("Profiles = %#v, want stored profile %q", cfg.Profiles, tc.want)
			}
		})
	}
}

func TestAuthUseCommandUpdatesDefaultProfile(t *testing.T) {
	ctx := authTestContext(t)
	for _, input := range []auth.LoginInput{
		{ProfileName: "sandbox", Environment: config.EnvSandbox, APIKey: "sandbox-token", StorageMode: config.StorageModeFile},
		{ProfileName: "prod", Environment: config.EnvProduction, APIKey: "prod-token", StorageMode: config.StorageModeFile},
	} {
		if _, err := ctx.Services.Login(input); err != nil {
			t.Fatalf("Login(%q) error = %v", input.ProfileName, err)
		}
	}

	cmd := newAuthUseCmd(ctx)
	cmd.SetContext(context.Background())
	cmd.SetArgs([]string{"sandbox"})

	output := captureStdout(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
	})

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}
	if cfg.DefaultProfile != "sandbox" {
		t.Fatalf("DefaultProfile = %q, want sandbox", cfg.DefaultProfile)
	}
	if !strings.Contains(output, "Active profile updated") {
		t.Fatalf("output = %q, want update confirmation", output)
	}
}

func TestAuthUseCommandWithoutArgumentListsProfiles(t *testing.T) {
	ctx := authTestContext(t)
	for _, input := range []auth.LoginInput{
		{ProfileName: "sandbox", Environment: config.EnvSandbox, APIKey: "sandbox-token", StorageMode: config.StorageModeFile},
		{ProfileName: "prod", Environment: config.EnvProduction, APIKey: "prod-token", StorageMode: config.StorageModeFile},
	} {
		if _, err := ctx.Services.Login(input); err != nil {
			t.Fatalf("Login(%q) error = %v", input.ProfileName, err)
		}
	}

	cmd := newAuthUseCmd(ctx)
	cmd.SetContext(context.Background())
	cmd.SetArgs(nil)

	oldStdin := os.Stdin
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	os.Stdin = reader
	t.Cleanup(func() {
		os.Stdin = oldStdin
		reader.Close()
		writer.Close()
	})

	err = cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want profile requirement")
	}
	if !strings.Contains(err.Error(), "available profiles: prod, sandbox") {
		t.Fatalf("Execute() error = %v, want available profile names", err)
	}
}

func TestAuthUseCommandJSONOutputIncludesSelectedProfile(t *testing.T) {
	ctx := authTestContext(t)
	ctx.Options.JSON = true
	for _, input := range []auth.LoginInput{
		{ProfileName: "sandbox", Environment: config.EnvSandbox, APIKey: "sandbox-token", StorageMode: config.StorageModeFile},
		{ProfileName: "prod", Environment: config.EnvProduction, APIKey: "prod-token", StorageMode: config.StorageModeFile},
	} {
		if _, err := ctx.Services.Login(input); err != nil {
			t.Fatalf("Login(%q) error = %v", input.ProfileName, err)
		}
	}

	cmd := newAuthUseCmd(ctx)
	cmd.SetContext(context.Background())
	cmd.SetArgs([]string{"sandbox"})

	output := captureStdout(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
	})

	if !strings.Contains(output, `"ok": true`) {
		t.Fatalf("output = %q, want success envelope", output)
	}
	if !strings.Contains(output, `"name": "sandbox"`) {
		t.Fatalf("output = %q, want selected profile name", output)
	}
	if !strings.Contains(output, `"is_default": true`) {
		t.Fatalf("output = %q, want default marker", output)
	}
}

func TestRunAuthMenuOffersSwitchAction(t *testing.T) {
	original := runPromptSelect
	t.Cleanup(func() { runPromptSelect = original })

	runPromptSelect = func(label string, options []ui.Option) (string, error) {
		if label != "Authentication" {
			t.Fatalf("label = %q, want Authentication", label)
		}
		for _, option := range options {
			if option.Value == "use" && strings.Contains(option.Description, "active stored profile") {
				return "__nav_exit", nil
			}
		}
		t.Fatalf("options = %#v, want switch action", options)
		return "", nil
	}

	err := runAuthMenu(newAuthCmd(&Context{Options: &RootOptions{}}), &Context{Options: &RootOptions{}})
	if !isNavigateExit(err) {
		t.Fatalf("runAuthMenu() error = %v, want exit navigation", err)
	}
}

func authTestContext(t *testing.T) *Context {
	t.Helper()
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tempHome, ".config"))
	return &Context{
		Services: app.NewServices(),
		Options:  &RootOptions{Advanced: true},
	}
}
