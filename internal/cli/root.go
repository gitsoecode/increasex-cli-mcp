package cli

import (
	"github.com/gitsoecode/increasex-cli-mcp/internal/app"
	"github.com/gitsoecode/increasex-cli-mcp/internal/ui"
	"github.com/spf13/cobra"
)

type RootOptions struct {
	Profile     string
	Environment string
	JSON        bool
	Interactive bool
	Yes         bool
	Debug       bool
	APIKey      string
	Advanced    bool
}

type Context struct {
	Services app.Services
	Options  *RootOptions
}

func NewRootCmd() *cobra.Command {
	options := &RootOptions{}
	ctx := &Context{
		Services: app.NewServices(),
		Options:  options,
	}
	return newRootCmdWithContext(ctx)
}

func newRootCmdWithContext(ctx *Context) *cobra.Command {
	cobra.EnableCommandSorting = false

	options := ctx.Options
	cmd := &cobra.Command{
		Use:           "increasex",
		Short:         "Increase CLI wrapper and local MCP server",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if terminalMenuRequested(options) {
				return runRootMenu(cmd, ctx)
			}
			return cmd.Help()
		},
	}

	cmd.PersistentFlags().StringVar(&options.Profile, "profile", "", "profile name")
	cmd.PersistentFlags().StringVar(&options.Environment, "env", "", "environment override: sandbox or production")
	cmd.PersistentFlags().BoolVar(&options.JSON, "json", false, "emit machine-readable JSON")
	cmd.PersistentFlags().BoolVar(&options.Interactive, "interactive", false, "force interactive mode")
	cmd.PersistentFlags().BoolVar(&options.Yes, "yes", false, "auto-confirm write operations")
	cmd.PersistentFlags().BoolVar(&options.Debug, "debug", false, "enable debug output")
	cmd.PersistentFlags().StringVar(&options.APIKey, "api-key", "", "explicit API key override")
	cmd.PersistentFlags().BoolVarP(&options.Advanced, "advanced", "a", false, "disable terminal menus and use command-driven mode")

	cmd.AddCommand(
		newAccountsCmd(ctx),
		newAccountNumbersCmd(ctx),
		newBalanceCmd(ctx),
		newTransactionsCmd(ctx),
		newTransferCmd(ctx),
		newExternalAccountsCmd(ctx),
		newCardsCmd(ctx),
		newAuthCmd(ctx),
		newMCPCmd(ctx),
	)
	applySilentBehavior(cmd)
	return cmd
}

func applySilentBehavior(cmd *cobra.Command) {
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	for _, child := range cmd.Commands() {
		applySilentBehavior(child)
	}
}

func runRootMenu(cmd *cobra.Command, ctx *Context) error {
	for {
		choice, err := promptSelectNavigation("IncreaseX", []ui.Option{
			{Label: "Accounts", Value: "accounts", Description: "List accounts and common account actions"},
			{Label: "Account Numbers", Value: "account_numbers", Description: "Discover, create, and disable account numbers"},
			{Label: "Balance", Value: "balance", Description: "Retrieve an account balance"},
			{Label: "Transactions", Value: "transactions", Description: "List recent transactions"},
			{Label: "Transfer", Value: "transfer", Description: "Create, review, approve, or cancel transfers"},
			{Label: "External Accounts", Value: "external_accounts", Description: "Manage stored external destinations"},
			{Label: "Cards", Value: "cards", Description: "Retrieve cards, details, and card actions"},
			{Label: "Authentication", Value: "auth", Description: "Log in, export, and inspect credentials"},
			{Label: "Advanced", Value: "advanced", Description: "Type an increasex command directly"},
		}, navExit)
		if err != nil {
			return err
		}
		switch choice {
		case "accounts":
			if err := invokeCommand(cmd, newAccountsCmd(ctx)); err != nil {
				if isNavigateExit(err) {
					return nil
				}
				return err
			}
		case "account_numbers":
			if err := invokeCommand(cmd, newAccountNumbersCmd(ctx)); err != nil {
				if isNavigateExit(err) {
					return nil
				}
				return err
			}
		case "balance":
			if err := invokeCommand(cmd, newBalanceCmd(ctx)); err != nil {
				if isNavigateExit(err) {
					return nil
				}
				return err
			}
		case "transactions":
			if err := invokeCommand(cmd, newTransactionsCmd(ctx)); err != nil {
				if isNavigateExit(err) {
					return nil
				}
				return err
			}
		case "transfer":
			if err := runTransferMenu(cmd, ctx); err != nil {
				if isNavigateExit(err) {
					return nil
				}
				return err
			}
		case "external_accounts":
			if err := invokeCommand(cmd, newExternalAccountsCmd(ctx)); err != nil {
				if isNavigateExit(err) {
					return nil
				}
				return err
			}
		case "cards":
			if err := invokeCommand(cmd, newCardsCmd(ctx)); err != nil {
				if isNavigateExit(err) {
					return nil
				}
				return err
			}
		case "auth":
			if err := runAuthMenu(cmd, ctx); err != nil {
				if isNavigateExit(err) {
					return nil
				}
				return err
			}
		case "advanced":
			if err := runAdvancedPrompt(cmd, ctx); err != nil {
				if isNavigateExit(err) {
					return nil
				}
				return err
			}
		case "exit":
			return nil
		}
	}
}

func runAdvancedPrompt(cmd *cobra.Command, ctx *Context) error {
	for {
		input, err := promptStringNavigation("Enter increasex command", false)
		if err != nil {
			if isNavigateBack(err) {
				return nil
			}
			if isNavigateExit(err) {
				return err
			}
			return err
		}

		args, err := parseAdvancedCommand(input)
		if err != nil {
			printCLIError(err)
			continue
		}
		if len(args) == 0 {
			return nil
		}

		advancedCtx := &Context{
			Services: ctx.Services,
			Options:  cloneRootOptions(ctx.Options),
		}
		advancedCtx.Options.Advanced = true

		root := newRootCmdWithContext(advancedCtx)
		root.SetContext(cmd.Context())
		root.SetArgs(args)
		return root.ExecuteContext(cmd.Context())
	}
}

func cloneRootOptions(options *RootOptions) *RootOptions {
	if options == nil {
		return &RootOptions{}
	}
	cloned := *options
	return &cloned
}
