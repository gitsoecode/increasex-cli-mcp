package cli

import (
	"fmt"
	"strings"

	"github.com/gitsoecode/increasex-cli-mcp/internal/auth"
	"github.com/gitsoecode/increasex-cli-mcp/internal/config"
	"github.com/gitsoecode/increasex-cli-mcp/internal/ui"
	"github.com/spf13/cobra"
)

func newAuthCmd(ctx *Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage authentication",
		RunE: func(cmd *cobra.Command, args []string) error {
			if terminalMenuRequested(ctx.Options) {
				return runAuthMenu(cmd, ctx)
			}
			return cmd.Help()
		},
	}
	cmd.AddCommand(
		newAuthLoginCmd(ctx),
		newAuthUseCmd(ctx),
		newAuthExportCmd(ctx),
		newAuthWhoamiCmd(ctx),
		newAuthLogoutCmd(ctx),
		newAuthStatusCmd(ctx),
	)
	return cmd
}

func runAuthMenu(cmd *cobra.Command, ctx *Context) error {
	for {
		choice, err := promptSelectNavigation("Authentication", []ui.Option{
			{Label: "Status", Value: "status", Description: "Show stored profile status"},
			{Label: "Switch", Value: "use", Description: "Change the active stored profile without re-entering an API key"},
			{Label: "Login", Value: "login", Description: "Store credentials or print environment exports"},
			{Label: "Export", Value: "export", Description: "Print shell export commands with the raw API key after confirmation"},
			{Label: "Who am I", Value: "whoami", Description: "Validate auth and show the active profile and entity context"},
			{Label: "Logout", Value: "logout", Description: "Remove stored credentials for the selected profile"},
		}, navBack, navExit)
		if err != nil {
			return err
		}
		switch choice {
		case "status":
			if err := invokeCommand(cmd, newAuthStatusCmd(ctx)); err != nil {
				if isNavigateExit(err) {
					return err
				}
				return err
			}
		case "use":
			if err := invokeCommand(cmd, newAuthUseCmd(ctx)); err != nil {
				if isNavigateExit(err) {
					return err
				}
				return err
			}
		case "login":
			if err := invokeCommand(cmd, newAuthLoginCmd(ctx)); err != nil {
				if isNavigateExit(err) {
					return err
				}
				return err
			}
		case "export":
			if err := invokeCommand(cmd, newAuthExportCmd(ctx)); err != nil {
				if isNavigateExit(err) {
					return err
				}
				return err
			}
		case "whoami":
			if err := invokeCommand(cmd, newAuthWhoamiCmd(ctx)); err != nil {
				if isNavigateExit(err) {
					return err
				}
				return err
			}
		case "logout":
			if err := invokeCommand(cmd, newAuthLogoutCmd(ctx)); err != nil {
				if isNavigateExit(err) {
					return err
				}
				return err
			}
		case "back", "exit":
			return nil
		}
	}
}

func newAuthUseCmd(ctx *Context) *cobra.Command {
	return &cobra.Command{
		Use:     "use [profile]",
		Aliases: []string{"switch"},
		Short:   "Switch the active stored profile",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			profiles, err := ctx.Services.ListAuthProfiles()
			if err != nil {
				if ctx.Options.JSON {
					return printEnvelopeJSON(nil, "", err)
				}
				return err
			}
			if len(profiles) == 0 {
				err := fmt.Errorf("no stored profiles found; run `increasex auth login` first")
				if ctx.Options.JSON {
					return printEnvelopeJSON(nil, "", err)
				}
				return err
			}

			profile := ""
			if len(args) > 0 {
				profile = strings.TrimSpace(args[0])
			}
			if profile == "" {
				if isInteractiveRequested(ctx.Options) {
					selected, err := promptSelectNavigation("Switch profile", buildAuthProfileOptions(profiles), navBack, navExit)
					if err != nil {
						return bubbleNavigation(cmd, err)
					}
					profile = selected
				} else {
					err := fmt.Errorf("profile is required; available profiles: %s", strings.Join(authProfileNames(profiles), ", "))
					if ctx.Options.JSON {
						return printEnvelopeJSON(nil, "", err)
					}
					return err
				}
			}

			alreadyDefault := false
			for _, item := range profiles {
				if item.Profile.Name == profile {
					alreadyDefault = item.IsDefault
					break
				}
			}

			summary, err := ctx.Services.UseAuthProfile(profile)
			if ctx.Options.JSON {
				return printEnvelopeJSON(summary, "", err)
			}
			if err != nil {
				return err
			}
			printAuthProfileSummary(summary, alreadyDefault)
			return nil
		},
	}
}

func newAuthLoginCmd(ctx *Context) *cobra.Command {
	var profile, env, apiKey, storage string
	var printEnv bool
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Store credentials or print a shell export snippet",
		RunE: func(cmd *cobra.Command, args []string) error {
			if env == "" && isInteractiveRequested(ctx.Options) {
				selected, err := promptSelectNavigation("Environment", []ui.Option{{Label: "Sandbox", Value: config.EnvSandbox}, {Label: "Production", Value: config.EnvProduction}}, navBack, navExit)
				if err != nil {
					return bubbleNavigation(cmd, err)
				}
				env = selected
			}
			if isInteractiveRequested(ctx.Options) && !cmd.Flags().Changed("name") {
				value, err := promptStringNavigation(loginProfilePromptLabel(env), false)
				if err != nil {
					return bubbleNavigation(cmd, err)
				}
				profile = strings.TrimSpace(value)
				if profile == "" {
					profile = defaultLoginProfileName(env)
				}
			}
			if apiKey == "" && isInteractiveRequested(ctx.Options) {
				value, err := promptStringNavigation("Increase API key", true)
				if err != nil {
					return bubbleNavigation(cmd, err)
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
				selected, err := promptSelectNavigation("Storage mode", []ui.Option{
					{Label: "Automatic (Recommended)", Value: string(config.StorageModeAuto), Description: "Save a durable local credential and mirror to Keychain when available"},
					{Label: "File only", Value: string(config.StorageModeFile), Description: "Best for agents and MCP across terminal sessions"},
					{Label: "Keychain only", Value: string(config.StorageModeKeychain), Description: "Store only in Keychain"},
				}, navBack, navExit)
				if err != nil {
					return bubbleNavigation(cmd, err)
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
	var confirm bool
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Print shell export commands from the resolved credential (includes the raw API key)",
		Long: strings.TrimSpace(`
Print shell export commands from the resolved credential.

Warning: this command prints the raw API key to stdout. Shell history, terminal
scrollback, screenshots, or wrappers that capture stdout may expose it.
`),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !ctx.Options.JSON {
				fmt.Fprintln(cmd.ErrOrStderr(), authExportWarningText())
			}
			if !confirm && !ctx.Options.Yes {
				if isInteractiveRequested(ctx.Options) {
					confirmed, err := promptConfirmationNavigation(authExportConfirmationPrompt())
					if err != nil || !confirmed {
						return bubbleNavigation(cmd, err)
					}
				} else {
					return fmt.Errorf("auth export prints a secret to stdout; rerun with --confirm if you intend to expose it")
				}
			}
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
	cmd.Flags().BoolVar(&confirm, "confirm", false, "confirm that printing the raw API key to stdout is intentional")
	return cmd
}

func newAuthWhoamiCmd(ctx *Context) *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Validate auth and show the active profile and entity context",
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

func authExportWarningText() string {
	return "Warning: auth export prints your raw API key to stdout."
}

func authExportConfirmationPrompt() string {
	return "Export raw API credentials to stdout?"
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

func printAuthProfileSummary(summary auth.ProfileSummary, alreadyDefault bool) {
	message := "Active profile updated"
	if alreadyDefault {
		message = "Profile already active"
	}
	printKeyValues(map[string]any{
		"message":                       message,
		"name":                          summary.Profile.Name,
		"environment":                   summary.Profile.Environment,
		"storage_mode":                  summary.Profile.StorageMode,
		"is_default":                    summary.IsDefault,
		"file_credential_available":     summary.FileCredentialAvailable,
		"keychain_credential_available": summary.KeychainCredentialAvail,
		"preferred_runtime_source":      summary.PreferredRuntimeSource,
		"mcp_ready":                     summary.MCPReady,
		"credential_error":              summary.CredentialError,
		"warnings":                      strings.Join(summary.Warnings, "; "),
	})
}

func buildAuthProfileOptions(profiles []auth.ProfileSummary) []ui.Option {
	options := make([]ui.Option, 0, len(profiles))
	for _, profile := range profiles {
		label := profile.Profile.Name
		if profile.IsDefault {
			label += " (active)"
		}
		descriptionParts := []string{profile.Profile.Environment, string(profile.Profile.StorageMode)}
		switch {
		case profile.MCPReady:
			descriptionParts = append(descriptionParts, "mcp-ready")
		case profile.KeychainCredentialAvail:
			descriptionParts = append(descriptionParts, "keychain-only")
		default:
			descriptionParts = append(descriptionParts, "missing-credentials")
		}
		options = append(options, ui.Option{
			Label:       label,
			Value:       profile.Profile.Name,
			Description: strings.Join(descriptionParts, " | "),
			Search: strings.Join([]string{
				profile.Profile.Name,
				profile.Profile.Environment,
				string(profile.Profile.StorageMode),
				profile.PreferredRuntimeSource,
				strings.Join(profile.Warnings, " "),
			}, " "),
		})
	}
	return options
}

func authProfileNames(profiles []auth.ProfileSummary) []string {
	names := make([]string, 0, len(profiles))
	for _, profile := range profiles {
		names = append(names, profile.Profile.Name)
	}
	return names
}

func loginProfilePromptLabel(env string) string {
	return fmt.Sprintf("Profile name (blank for %s)", defaultLoginProfileName(env))
}

func defaultLoginProfileName(env string) string {
	switch strings.ToLower(strings.TrimSpace(env)) {
	case config.EnvSandbox:
		return "sandbox"
	case config.EnvProduction:
		return "prod"
	default:
		return "default"
	}
}

func shellQuote(value string) string {
	value = strings.ReplaceAll(value, `'`, `'"'"'`)
	return "'" + value + "'"
}
