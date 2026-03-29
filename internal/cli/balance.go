package cli

import (
	"fmt"

	"github.com/jessevaughan/increasex/internal/util"
	"github.com/spf13/cobra"
)

func newBalanceCmd(ctx *Context) *cobra.Command {
	var accountID string
	cmd := &cobra.Command{
		Use:   "balance",
		Short: "Retrieve an account balance",
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
			if accountID == "" {
				return fmt.Errorf("account-id is required")
			}
			balance, requestID, err := ctx.Services.GetBalance(cmd.Context(), api, accountID)
			if ctx.Options.JSON {
				return printEnvelopeJSON(balance, requestID, err)
			}
			if err != nil {
				return err
			}
			fmt.Printf("account_id: %s\ncurrent_balance: %s\navailable_balance: %s\n", balance.AccountID, util.FormatUSDMinor(balance.CurrentBalance), util.FormatUSDMinor(balance.AvailableBalance))
			return nil
		},
	}
	cmd.Flags().StringVar(&accountID, "account-id", "", "account id")
	return cmd
}
