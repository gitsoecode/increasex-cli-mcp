package cli

import (
	"github.com/gitsoecode/increasex-cli-mcp/internal/app"
	"github.com/gitsoecode/increasex-cli-mcp/internal/ui"
	"github.com/spf13/cobra"
)

func newAccountNumbersCmd(ctx *Context) *cobra.Command {
	var accountID, status, cursor string
	var limit int64
	cmd := &cobra.Command{
		Use:   "account-numbers",
		Short: "Discover and manage account numbers",
		RunE: func(cmd *cobra.Command, args []string) error {
			session, api, err := ctx.resolve(cmd.Context())
			if err != nil {
				return printEnvelopeJSON(nil, "", err)
			}
			_ = session
			numbers, requestID, err := ctx.Services.ListAccountNumbers(cmd.Context(), api, accountID, status, limit, cursor)
			if ctx.Options.JSON {
				return printEnvelopeJSON(map[string]any{"account_numbers": numbers}, requestID, err)
			}
			if err != nil {
				return err
			}
			printAccountNumbers(numbers)
			if !isInteractiveRequested(ctx.Options) {
				return nil
			}
			for {
				options := []ui.Option{
					{Label: "Inspect an account number", Value: "inspect"},
					{Label: "Create an account number", Value: "create"},
				}
				if len(numbers) > 0 {
					options = append(options, ui.Option{Label: "Disable an account number", Value: "disable"})
				}
				action, err := promptSelectNavigation("Account numbers menu", options, navBack, navExit)
				if err != nil {
					return bubbleNavigation(cmd, err)
				}
				switch action {
				case "create":
					args := []string{}
					if accountID != "" {
						args = append(args, "--account-id", accountID)
					}
					if err := invokeCommand(cmd, newAccountNumbersCreateCmd(ctx), args...); err != nil {
						return bubbleNavigation(cmd, err)
					}
				case "inspect":
					if len(numbers) == 0 {
						continue
					}
					selected, err := chooseAccountNumber(numbers, "Select an account number")
					if err != nil {
						if isNavigateBack(err) {
							continue
						}
						return bubbleNavigation(cmd, err)
					}
					if err := invokeCommand(cmd, newAccountNumbersGetCmd(ctx), "--account-number-id", selected, "--show-sensitive"); err != nil {
						return bubbleNavigation(cmd, err)
					}
					return nil
				case "disable":
					if len(numbers) == 0 {
						continue
					}
					selected, err := chooseAccountNumber(numbers, "Select an account number")
					if err != nil {
						if isNavigateBack(err) {
							continue
						}
						return bubbleNavigation(cmd, err)
					}
					if err := invokeCommand(cmd, newAccountNumbersDisableCmd(ctx), "--account-number-id", selected); err != nil {
						return bubbleNavigation(cmd, err)
					}
				}
				numbers, _, err = ctx.Services.ListAccountNumbers(cmd.Context(), api, accountID, status, limit, cursor)
				if err != nil {
					return err
				}
				printAccountNumbers(numbers)
			}
		},
	}
	cmd.Flags().StringVar(&accountID, "account-id", "", "filter by parent account id")
	cmd.Flags().StringVar(&status, "status", "", "filter by status")
	cmd.Flags().StringVar(&cursor, "cursor", "", "page cursor")
	cmd.Flags().Int64Var(&limit, "limit", 20, "maximum account numbers to return")
	cmd.AddCommand(
		newAccountNumbersGetCmd(ctx),
		newAccountNumbersCreateCmd(ctx),
		newAccountNumbersDisableCmd(ctx),
	)
	return cmd
}

func newAccountNumbersGetCmd(ctx *Context) *cobra.Command {
	var accountNumberID string
	var showSensitive bool
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Retrieve an account number",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, api, err := ctx.resolve(cmd.Context())
			if err != nil {
				return printEnvelopeJSON(nil, "", err)
			}
			if accountNumberID == "" && isInteractiveRequested(ctx.Options) {
				numbers, _, err := ctx.Services.ListAccountNumbers(cmd.Context(), api, "", "", 100, "")
				if err != nil {
					return err
				}
				accountNumberID, err = chooseAccountNumber(numbers, "Select an account number")
				if err != nil {
					return bubbleNavigation(cmd, err)
				}
			}
			var (
				number    *app.AccountNumberDetails
				requestID string
			)
			if showSensitive {
				number, requestID, err = ctx.Services.RetrieveSensitiveAccountNumberDetails(cmd.Context(), api, accountNumberID)
			} else {
				number, requestID, err = ctx.Services.RetrieveAccountNumber(cmd.Context(), api, accountNumberID)
			}
			if ctx.Options.JSON {
				return printEnvelopeJSON(number, requestID, err)
			}
			if err != nil {
				return err
			}
			printAccountNumberDetails(number)
			return nil
		},
	}
	cmd.Flags().StringVar(&accountNumberID, "account-number-id", "", "account number id")
	cmd.Flags().BoolVar(&showSensitive, "show-sensitive", false, "show full account number for this account number")
	return cmd
}

func newAccountNumbersCreateCmd(ctx *Context) *cobra.Command {
	var input app.CreateAccountNumberInput
	var dryRun bool
	var achDebitStatus, checksStatus string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Preview or create an account number",
		RunE: func(cmd *cobra.Command, args []string) error {
			session, api, err := ctx.resolve(cmd.Context())
			if err != nil {
				return printEnvelopeJSON(nil, "", err)
			}
			if input.AccountID == "" && isInteractiveRequested(ctx.Options) {
				accounts, _, err := ctx.Services.ListAccounts(cmd.Context(), api, "open", 100, "")
				if err != nil {
					return err
				}
				input.AccountID, err = chooseAccount(accounts, "Select a parent account")
				if err != nil {
					return bubbleNavigation(cmd, err)
				}
			}
			if input.Name == "" && isInteractiveRequested(ctx.Options) {
				input.Name, err = promptStringNavigation("Account number name", true)
				if err != nil {
					return bubbleNavigation(cmd, err)
				}
			}
			if achDebitStatus == "" && isInteractiveRequested(ctx.Options) {
				achDebitStatus, err = promptOptionalSelect("Inbound ACH debit status (optional)", []ui.Option{
					{Label: "Allowed", Value: "allowed"},
					{Label: "Blocked", Value: "blocked"},
				})
				if err != nil {
					return bubbleNavigation(cmd, err)
				}
			}
			if checksStatus == "" && isInteractiveRequested(ctx.Options) {
				checksStatus, err = promptOptionalSelect("Inbound checks status (optional)", []ui.Option{
					{Label: "Allowed", Value: "allowed"},
					{Label: "Check transfers only", Value: "check_transfers_only"},
				})
				if err != nil {
					return bubbleNavigation(cmd, err)
				}
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
					confirmed, err := promptConfirmationNavigation("Create this account number?")
					if err != nil || !confirmed {
						return bubbleNavigation(cmd, err)
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
	cmd.Flags().StringVar(&input.AccountID, "account-id", "", "parent account id")
	cmd.Flags().StringVar(&input.Name, "name", "", "account number name")
	cmd.Flags().StringVar(&achDebitStatus, "ach-debit-status", "", "inbound ACH debit status")
	cmd.Flags().StringVar(&checksStatus, "checks-status", "", "inbound checks status")
	cmd.Flags().StringVar(&input.IdempotencyKey, "idempotency-key", "", "idempotency key")
	cmd.Flags().StringVar(&input.ConfirmationToken, "confirmation-token", "", "confirmation token")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview only")
	return cmd
}

func newAccountNumbersDisableCmd(ctx *Context) *cobra.Command {
	var input app.DisableAccountNumberInput
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "disable",
		Short: "Preview or disable an account number",
		RunE: func(cmd *cobra.Command, args []string) error {
			session, api, err := ctx.resolve(cmd.Context())
			if err != nil {
				return printEnvelopeJSON(nil, "", err)
			}
			if input.AccountNumberID == "" && isInteractiveRequested(ctx.Options) {
				numbers, _, err := ctx.Services.ListAccountNumbers(cmd.Context(), api, "", "active", 100, "")
				if err != nil {
					return err
				}
				input.AccountNumberID, err = chooseAccountNumber(numbers, "Select an account number to disable")
				if err != nil {
					return bubbleNavigation(cmd, err)
				}
			}
			input.DryRun = &dryRun
			if dryRun {
				preview, err := ctx.Services.PreviewDisableAccountNumber(*session, input)
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
				preview, err := ctx.Services.PreviewDisableAccountNumber(*session, input)
				if err != nil {
					return err
				}
				if !ctx.Options.Yes {
					printPreview(preview)
					confirmed, err := promptConfirmationNavigation("Disable this account number?")
					if err != nil || !confirmed {
						return bubbleNavigation(cmd, err)
					}
				}
				input.ConfirmationToken = preview.ConfirmationToken
			}
			data, requestID, err := ctx.Services.ExecuteDisableAccountNumber(cmd.Context(), api, *session, input)
			if ctx.Options.JSON {
				return printEnvelopeJSON(data, requestID, err)
			}
			if err != nil {
				return err
			}
			return printJSON(data)
		},
	}
	cmd.Flags().StringVar(&input.AccountNumberID, "account-number-id", "", "account number id")
	cmd.Flags().StringVar(&input.IdempotencyKey, "idempotency-key", "", "idempotency key")
	cmd.Flags().StringVar(&input.ConfirmationToken, "confirmation-token", "", "confirmation token")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview only")
	return cmd
}
