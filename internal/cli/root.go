package cli

import (
	"github.com/jessevaughan/increasex/internal/app"
	"github.com/jessevaughan/increasex/internal/ui"
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
}

type Context struct {
	Services app.Services
	Options  *RootOptions
}

func NewRootCmd() *cobra.Command {
	cobra.EnableCommandSorting = false

	options := &RootOptions{}
	ctx := &Context{
		Services: app.NewServices(),
		Options:  options,
	}

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

	cmd.AddCommand(
		newAccountsCmd(ctx),
		newBalanceCmd(ctx),
		newTransactionsCmd(ctx),
		newTransferCmd(ctx),
		newExternalAccountsCmd(ctx),
		newCardsCmd(ctx),
		newAuthCmd(ctx),
		newMCPCmd(ctx),
	)
	return cmd
}

func runRootMenu(cmd *cobra.Command, ctx *Context) error {
	for {
		choice, err := ui.PromptSelect("IncreaseX", []ui.Option{
			{Label: "Accounts", Value: "accounts", Description: "List accounts and common account actions"},
			{Label: "Balance", Value: "balance", Description: "Retrieve an account balance"},
			{Label: "Transactions", Value: "transactions", Description: "List recent transactions"},
			{Label: "Transfer", Value: "transfer", Description: "Create, review, approve, or cancel transfers"},
			{Label: "External Accounts", Value: "external_accounts", Description: "Manage stored external destinations"},
			{Label: "Cards", Value: "cards", Description: "Retrieve cards, details, and card actions"},
			{Label: "Authentication", Value: "auth", Description: "Log in, export, and inspect credentials"},
			{Label: "Exit", Value: "exit", Description: "Return to the shell"},
		})
		if err != nil {
			return err
		}
		switch choice {
		case "accounts":
			if err := invokeCommand(cmd, newAccountsCmd(ctx)); err != nil {
				return err
			}
		case "balance":
			if err := invokeCommand(cmd, newBalanceCmd(ctx)); err != nil {
				return err
			}
		case "transactions":
			if err := invokeCommand(cmd, newTransactionsCmd(ctx)); err != nil {
				return err
			}
		case "transfer":
			if err := runTransferMenu(cmd, ctx); err != nil {
				return err
			}
		case "external_accounts":
			if err := invokeCommand(cmd, newExternalAccountsCmd(ctx)); err != nil {
				return err
			}
		case "cards":
			if err := invokeCommand(cmd, newCardsCmd(ctx)); err != nil {
				return err
			}
		case "auth":
			if err := invokeCommand(cmd, newAuthStatusCmd(ctx)); err != nil {
				return err
			}
		case "exit":
			return nil
		}
	}
}
