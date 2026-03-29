package cli

import (
	"fmt"
	"strings"

	"github.com/jessevaughan/increasex/internal/auth"
	"github.com/jessevaughan/increasex/internal/config"
	"github.com/jessevaughan/increasex/internal/ui"
	"github.com/spf13/cobra"
)

func newAuthCmd(ctx *Context) *cobra.Command {
	cmd := &cobra.Command{Use: "auth", Short: "Manage authentication"}
	cmd.AddCommand(
		newAuthLoginCmd(ctx),
		newAuthExportCmd(ctx),
		newAuthWhoamiCmd(ctx),
		newAuthLogoutCmd(ctx),
		newAuthStatusCmd(ctx),
	)
	return cmd
}

func newAuthLoginCmd(ctx *Context) *cobra.Command {
	var profile, env, apiKey, storage string
	var printEnv bool
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Store credentials or print a shell export snippet",
		RunE: func(cmd *cobra.Command, args []string) error {
			if env == "" && isInteractiveRequested(ctx.Options) {
				selected, err := ui.PromptSelect("Environment", []ui.Option{{Label: "Sandbox", Value: config.EnvSandbox}, {Label: "Production", Value: config.EnvProduction}})
				if err != nil {
					return err
				}
				env = selected
			}
			if apiKey == "" && isInteractiveRequested(ctx.Options) {
				value, err := ui.PromptString("Increase API key", true)
				if err != nil {
					return err
				}
				apiKey = value
			}
			if profile == "" {
				profile = "default"
			}
			if env == "" {
				env = ctx.Options.Environment
			}
			if printEnv {
				fmt.Printf("export INCREASE_API_KEY=%s\n", shellQuote(apiKey))
				fmt.Printf("export INCREASEX_ENV=%s\n", shellQuote(env))
				fmt.Printf("export INCREASEX_PROFILE=%s\n", shellQuote(profile))
				return nil
			}
			if storage == "" && isInteractiveRequested(ctx.Options) {
				selected, err := ui.PromptSelect("Storage mode", []ui.Option{
					{Label: "Automatic (Recommended)", Value: string(config.StorageModeAuto), Description: "Save a durable local credential and mirror to Keychain when available"},
					{Label: "File only", Value: string(config.StorageModeFile), Description: "Best for agents and MCP across terminal sessions"},
					{Label: "Keychain only", Value: string(config.StorageModeKeychain), Description: "Store only in Keychain"},
				})
				if err != nil {
					return err
				}
				storage = selected
			}
			if apiKey == "" {
				return fmt.Errorf("api key is required")
			}
			loginResult, err := ctx.Services.Login(auth.LoginInput{
				ProfileName: profile,
				Environment: env,
				APIKey:      apiKey,
				StorageMode: config.StorageMode(storage),
			})
			if ctx.Options.JSON {
				return printEnvelopeJSON(loginResult, "", err)
			}
			if err != nil {
				return err
			}
			printKeyValues(map[string]any{
				"profile":            loginResult.Profile.Name,
				"environment":        loginResult.Profile.Environment,
				"storage_mode":       loginResult.Profile.StorageMode,
				"file_saved":         loginResult.FileSaved,
				"keychain_mirrored":  loginResult.KeychainMirrored,
				"keychain_available": loginResult.KeychainAvailable,
				"mcp_ready":          loginResult.MCPReady,
				"warnings":           strings.Join(loginResult.Warnings, "; "),
			})
			return nil
		},
	}
	cmd.Flags().StringVar(&profile, "name", "default", "profile name")
	cmd.Flags().StringVar(&env, "env", "", "environment")
	cmd.Flags().StringVar(&apiKey, "api-key", "", "Increase API key")
	cmd.Flags().StringVar(&storage, "storage", string(config.StorageModeAuto), "storage mode: auto, file, or keychain")
	cmd.Flags().BoolVar(&printEnv, "print-env", false, "print shell export commands instead of storing credentials")
	return cmd
}

func newAuthExportCmd(ctx *Context) *cobra.Command {
	return &cobra.Command{
		Use:   "export",
		Short: "Print shell export commands from the resolved credential",
		RunE: func(cmd *cobra.Command, args []string) error {
			exports, err := ctx.Services.Export(auth.ResolveInput{
				ProfileName: ctx.Options.Profile,
				Environment: ctx.Options.Environment,
				APIKey:      ctx.Options.APIKey,
			})
			if ctx.Options.JSON {
				return printEnvelopeJSON(exports, "", err)
			}
			if err != nil {
				return err
			}
			keys := []string{"INCREASE_API_KEY", "INCREASEX_ENV", "INCREASEX_PROFILE"}
			for _, key := range keys {
				value, ok := exports[key]
				if !ok {
					continue
				}
				fmt.Printf("export %s=%s\n", key, shellQuote(value))
			}
			return nil
		},
	}
}

func newAuthWhoamiCmd(ctx *Context) *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Validate auth and show active identity context",
		RunE: func(cmd *cobra.Command, args []string) error {
			session, api, err := ctx.resolve(cmd.Context())
			if err != nil {
				return printEnvelopeJSON(nil, "", err)
			}
			data, requestID, err := ctx.Services.WhoAmI(cmd.Context(), api, *session)
			if ctx.Options.JSON {
				return printEnvelopeJSON(data, requestID, err)
			}
			if err != nil {
				return err
			}
			printKeyValues(data)
			return nil
		},
	}
}

func newAuthLogoutCmd(ctx *Context) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove stored credentials for the selected profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			err := ctx.Services.Logout(ctx.Options.Profile)
			if ctx.Options.JSON {
				return printEnvelopeJSON(map[string]any{"logged_out": true}, "", err)
			}
			if err != nil {
				return err
			}
			fmt.Println("Logged out")
			return nil
		},
	}
}

func newAuthStatusCmd(ctx *Context) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show stored profile status",
		RunE: func(cmd *cobra.Command, args []string) error {
			status, err := ctx.Services.AuthStatus(ctx.Options.Profile)
			if ctx.Options.JSON {
				return printEnvelopeJSON(status, "", err)
			}
			if err != nil {
				return err
			}
			printKeyValues(map[string]any{
				"name":                          status.Profile.Name,
				"environment":                   status.Profile.Environment,
				"storage_mode":                  status.Profile.StorageMode,
				"file_credential_available":     status.FileCredentialAvailable,
				"keychain_credential_available": status.KeychainCredentialAvail,
				"preferred_runtime_source":      status.PreferredRuntimeSource,
				"mcp_ready":                     status.MCPReady,
				"credential_error":              status.CredentialError,
				"warnings":                      strings.Join(status.Warnings, "; "),
			})
			return nil
		},
	}
}

func shellQuote(value string) string {
	value = strings.ReplaceAll(value, `'`, `'"'"'`)
	return "'" + value + "'"
}
