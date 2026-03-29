package cli

import (
	"github.com/jessevaughan/increasex/internal/app"
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
	}

	cmd.PersistentFlags().StringVar(&options.Profile, "profile", "", "profile name")
	cmd.PersistentFlags().StringVar(&options.Environment, "env", "", "environment override: sandbox or production")
	cmd.PersistentFlags().BoolVar(&options.JSON, "json", false, "emit machine-readable JSON")
	cmd.PersistentFlags().BoolVar(&options.Interactive, "interactive", false, "force interactive mode")
	cmd.PersistentFlags().BoolVar(&options.Yes, "yes", false, "auto-confirm write operations")
	cmd.PersistentFlags().BoolVar(&options.Debug, "debug", false, "enable debug output")
	cmd.PersistentFlags().StringVar(&options.APIKey, "api-key", "", "explicit API key override")

	cmd.AddCommand(
		newAuthCmd(ctx),
		newAccountsCmd(ctx),
		newBalanceCmd(ctx),
		newTransactionsCmd(ctx),
		newTransferCmd(ctx),
		newCardsCmd(ctx),
		newMCPCmd(ctx),
	)
	return cmd
}
