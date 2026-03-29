package cli

import (
	"fmt"

	"github.com/jessevaughan/increasex/internal/app"
	"github.com/jessevaughan/increasex/internal/ui"
	"github.com/spf13/cobra"
)

func newAccountsCmd(ctx *Context) *cobra.Command {
	var status, cursor string
	var limit int64
	cmd := &cobra.Command{
		Use:   "accounts",
		Short: "List accounts",
		RunE: func(cmd *cobra.Command, args []string) error {
			session, api, err := ctx.resolve(cmd.Context())
			if err != nil {
				return printEnvelopeJSON(nil, "", err)
			}
			_ = session
			accounts, requestID, err := ctx.Services.ListAccounts(cmd.Context(), api, status, limit, cursor)
			if ctx.Options.JSON {
				return printEnvelopeJSON(map[string]any{"accounts": accounts}, requestID, err)
			}
			if err != nil {
				return err
			}
			printAccounts(accounts)
			if isInteractiveRequested(ctx.Options) && len(accounts) > 0 {
				accountID, err := chooseAccount(accounts, "Account actions")
				if err != nil {
					return err
				}
				action, err := ui.PromptSelect("Choose action", []ui.Option{
					{Label: "Get balance", Value: "balance"},
					{Label: "Recent transactions", Value: "transactions"},
					{Label: "Create account number", Value: "create_number"},
					{Label: "Close account", Value: "close"},
				})
				if err != nil {
					return err
				}
				switch action {
				case "balance":
					balance, _, err := ctx.Services.GetBalance(cmd.Context(), api, accountID)
					if err != nil {
						return err
					}
					printKeyValues(map[string]any{
						"account_id":        balance.AccountID,
						"current_balance":   balance.CurrentBalance,
						"available_balance": balance.AvailableBalance,
					})
				case "transactions":
					items, _, err := ctx.Services.ListRecentTransactions(cmd.Context(), api, accountID, "", "", 10, nil)
					if err != nil {
						return err
					}
					printTransactions(items)
				case "create_number":
					name, err := ui.PromptString("Account number name", true)
					if err != nil {
						return err
					}
					preview, err := ctx.Services.PreviewCreateAccountNumber(*session, app.CreateAccountNumberInput{
						AccountID: accountID,
						Name:      name,
					})
					if err != nil {
						return err
					}
					printPreview(preview)
				case "close":
					preview, err := ctx.Services.PreviewCloseAccount(*session, app.CloseAccountInput{
						AccountID: accountID,
					})
					if err != nil {
						return err
					}
					printPreview(preview)
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&status, "status", "", "filter by status")
	cmd.Flags().StringVar(&cursor, "cursor", "", "page cursor")
	cmd.Flags().Int64Var(&limit, "limit", 20, "maximum accounts to return")
	cmd.AddCommand(
		newAccountsCreateCmd(ctx),
		newAccountsCloseCmd(ctx),
		newAccountsCreateNumberCmd(ctx),
	)
	return cmd
}

func newAccountsCreateCmd(ctx *Context) *cobra.Command {
	var input app.CreateAccountInput
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Preview or create an account",
		RunE: func(cmd *cobra.Command, args []string) error {
			session, api, err := ctx.resolve(cmd.Context())
			if err != nil {
				return printEnvelopeJSON(nil, "", err)
			}
			if input.Name == "" && isInteractiveRequested(ctx.Options) {
				value, err := ui.PromptString("Account name", true)
				if err != nil {
					return err
				}
				input.Name = value
			}
			input.DryRun = &dryRun
			if dryRun {
				preview, err := ctx.Services.PreviewCreateAccount(*session, input)
				if ctx.Options.JSON {
					return printEnvelopeJSON(preview, "", err)
				}
				if err != nil {
					return err
				}
				printPreview(preview)
				return nil
			}
			if input.ConfirmationToken == "" {
				preview, err := ctx.Services.PreviewCreateAccount(*session, input)
				if err != nil {
					return err
				}
				if !ctx.Options.Yes {
					printPreview(preview)
					confirmed, err := ui.Confirm("Create this account?")
					if err != nil || !confirmed {
						return err
					}
				}
				input.ConfirmationToken = preview.ConfirmationToken
			}
			data, requestID, err := ctx.Services.ExecuteCreateAccount(cmd.Context(), api, *session, input)
			if ctx.Options.JSON {
				return printEnvelopeJSON(data, requestID, err)
			}
			if err != nil {
				return err
			}
			fmt.Println("Account created")
			return printJSON(data)
		},
	}
	cmd.Flags().StringVar(&input.Name, "name", "", "account name")
	cmd.Flags().StringVar(&input.EntityID, "entity-id", "", "entity id")
	cmd.Flags().StringVar(&input.InformationalEntityID, "informational-entity-id", "", "informational entity id")
	cmd.Flags().StringVar(&input.ProgramID, "program-id", "", "program id")
	cmd.Flags().StringVar(&input.IdempotencyKey, "idempotency-key", "", "idempotency key")
	cmd.Flags().StringVar(&input.ConfirmationToken, "confirmation-token", "", "confirmation token")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview only")
	return cmd
}

func newAccountsCloseCmd(ctx *Context) *cobra.Command {
	var input app.CloseAccountInput
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "close",
		Short: "Preview or close an account",
		RunE: func(cmd *cobra.Command, args []string) error {
			session, api, err := ctx.resolve(cmd.Context())
			if err != nil {
				return printEnvelopeJSON(nil, "", err)
			}
			input.DryRun = &dryRun
			if dryRun {
				preview, err := ctx.Services.PreviewCloseAccount(*session, input)
				if ctx.Options.JSON {
					return printEnvelopeJSON(preview, "", err)
				}
				if err != nil {
					return err
				}
				printPreview(preview)
				return nil
			}
			if input.ConfirmationToken == "" {
				preview, err := ctx.Services.PreviewCloseAccount(*session, input)
				if err != nil {
					return err
				}
				if !ctx.Options.Yes {
					printPreview(preview)
					confirmed, err := ui.Confirm("Close this account?")
					if err != nil || !confirmed {
						return err
					}
				}
				input.ConfirmationToken = preview.ConfirmationToken
			}
			data, requestID, err := ctx.Services.ExecuteCloseAccount(cmd.Context(), api, *session, input)
			if ctx.Options.JSON {
				return printEnvelopeJSON(data, requestID, err)
			}
			if err != nil {
				return err
			}
			return printJSON(data)
		},
	}
	cmd.Flags().StringVar(&input.AccountID, "account-id", "", "account id")
	cmd.Flags().StringVar(&input.IdempotencyKey, "idempotency-key", "", "idempotency key")
	cmd.Flags().StringVar(&input.ConfirmationToken, "confirmation-token", "", "confirmation token")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview only")
	return cmd
}

func newAccountsCreateNumberCmd(ctx *Context) *cobra.Command {
	var input app.CreateAccountNumberInput
	var dryRun bool
	var achDebitStatus, checksStatus string
	cmd := &cobra.Command{
		Use:   "create-number",
		Short: "Preview or create an account number",
		RunE: func(cmd *cobra.Command, args []string) error {
			session, api, err := ctx.resolve(cmd.Context())
			if err != nil {
				return printEnvelopeJSON(nil, "", err)
			}
			if achDebitStatus != "" {
				input.InboundACH = &app.InboundACHInput{DebitStatus: achDebitStatus}
			}
			if checksStatus != "" {
				input.InboundChecks = &app.InboundChecksInput{Status: checksStatus}
			}
			input.DryRun = &dryRun
			if dryRun {
				preview, err := ctx.Services.PreviewCreateAccountNumber(*session, input)
				if ctx.Options.JSON {
					return printEnvelopeJSON(preview, "", err)
				}
				if err != nil {
					return err
				}
				printPreview(preview)
				return nil
			}
			if input.ConfirmationToken == "" {
				preview, err := ctx.Services.PreviewCreateAccountNumber(*session, input)
				if err != nil {
					return err
				}
				if !ctx.Options.Yes {
					printPreview(preview)
					confirmed, err := ui.Confirm("Create this account number?")
					if err != nil || !confirmed {
						return err
					}
				}
				input.ConfirmationToken = preview.ConfirmationToken
			}
			data, requestID, err := ctx.Services.ExecuteCreateAccountNumber(cmd.Context(), api, *session, input)
			if ctx.Options.JSON {
				return printEnvelopeJSON(data, requestID, err)
			}
			if err != nil {
				return err
			}
			return printJSON(data)
		},
	}
	cmd.Flags().StringVar(&input.AccountID, "account-id", "", "account id")
	cmd.Flags().StringVar(&input.Name, "name", "", "account number name")
	cmd.Flags().StringVar(&achDebitStatus, "ach-debit-status", "", "inbound ACH debit status")
	cmd.Flags().StringVar(&checksStatus, "checks-status", "", "inbound checks status")
	cmd.Flags().StringVar(&input.IdempotencyKey, "idempotency-key", "", "idempotency key")
	cmd.Flags().StringVar(&input.ConfirmationToken, "confirmation-token", "", "confirmation token")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview only")
	return cmd
}
