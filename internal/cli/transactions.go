package cli

import "github.com/spf13/cobra"

func newTransactionsCmd(ctx *Context) *cobra.Command {
	var accountID, since, cursor string
	var limit int64
	var categories []string
	cmd := &cobra.Command{
		Use:   "transactions",
		Short: "List recent transactions",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, api, err := ctx.resolve(cmd.Context())
			if err != nil {
				return printEnvelopeJSON(nil, "", err)
			}
			if accountID == "" && isInteractiveRequested(ctx.Options) {
				accounts, _, err := ctx.Services.ListAccounts(cmd.Context(), api, "", 25, "")
				if err != nil {
					return err
				}
				accountID, err = chooseAccount(accounts, "Select an account")
				if err != nil {
					return err
				}
			}
			items, requestID, err := ctx.Services.ListRecentTransactions(cmd.Context(), api, accountID, since, cursor, limit, categories)
			if ctx.Options.JSON {
				return printEnvelopeJSON(map[string]any{"transactions": items}, requestID, err)
			}
			if err != nil {
				return err
			}
			printTransactions(items)
			return nil
		},
	}
	cmd.Flags().StringVar(&accountID, "account-id", "", "account id")
	cmd.Flags().StringVar(&since, "since", "", "RFC3339 lower bound")
	cmd.Flags().StringVar(&cursor, "cursor", "", "page cursor")
	cmd.Flags().Int64Var(&limit, "limit", 20, "maximum transactions to return")
	cmd.Flags().StringSliceVar(&categories, "category", nil, "transaction categories")
	return cmd
}
