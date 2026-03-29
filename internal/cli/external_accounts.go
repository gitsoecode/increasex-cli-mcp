package cli

import (
	"fmt"

	"github.com/jessevaughan/increasex/internal/app"
	"github.com/jessevaughan/increasex/internal/ui"
	"github.com/spf13/cobra"
)

func newExternalAccountsCmd(ctx *Context) *cobra.Command {
	var status, cursor string
	var limit int64
	cmd := &cobra.Command{
		Use:   "external-accounts",
		Short: "List and manage stored external accounts",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, api, err := ctx.resolve(cmd.Context())
			if err != nil {
				return printEnvelopeJSON(nil, "", err)
			}
			accounts, requestID, err := ctx.Services.ListExternalAccounts(cmd.Context(), api, status, cursor, limit)
			if ctx.Options.JSON {
				return printEnvelopeJSON(map[string]any{"external_accounts": accounts}, requestID, err)
			}
			if err != nil {
				return err
			}
			printExternalAccounts(accounts)
			if !isInteractiveRequested(ctx.Options) {
				return nil
			}
			options := []ui.Option{
				{Label: "Retrieve an external account", Value: "retrieve"},
				{Label: "Create an external account", Value: "create"},
				{Label: "Update an external account", Value: "update"},
				{Label: "Back", Value: "back"},
			}
			if len(accounts) == 0 {
				options = []ui.Option{
					{Label: "Create an external account", Value: "create"},
					{Label: "Back", Value: "back"},
				}
			}
			action, err := ui.PromptSelect("External account actions", options)
			if err != nil {
				return err
			}
			switch action {
			case "retrieve":
				selected, err := chooseExternalAccount(accounts, "Select an external account")
				if err != nil {
					return err
				}
				return invokeCommand(cmd, newExternalAccountsRetrieveCmd(ctx), "--external-account-id", selected)
			case "create":
				return invokeCommand(cmd, newExternalAccountsCreateCmd(ctx))
			case "update":
				selected, err := chooseExternalAccount(accounts, "Select an external account")
				if err != nil {
					return err
				}
				return invokeCommand(cmd, newExternalAccountsUpdateCmd(ctx), "--external-account-id", selected)
			default:
				return nil
			}
		},
	}
	cmd.Flags().StringVar(&status, "status", "", "status filter")
	cmd.Flags().StringVar(&cursor, "cursor", "", "page cursor")
	cmd.Flags().Int64Var(&limit, "limit", 20, "maximum external accounts to return")
	cmd.AddCommand(
		newExternalAccountsRetrieveCmd(ctx),
		newExternalAccountsCreateCmd(ctx),
		newExternalAccountsUpdateCmd(ctx),
	)
	return cmd
}

func newExternalAccountsRetrieveCmd(ctx *Context) *cobra.Command {
	var externalAccountID string
	cmd := &cobra.Command{
		Use:     "retrieve",
		Short:   "Retrieve a stored external account",
		Aliases: []string{"get"},
		RunE: func(cmd *cobra.Command, args []string) error {
			_, api, err := ctx.resolve(cmd.Context())
			if err != nil {
				return printEnvelopeJSON(nil, "", err)
			}
			if externalAccountID == "" && isInteractiveRequested(ctx.Options) {
				accounts, _, err := ctx.Services.ListExternalAccounts(cmd.Context(), api, "", "", 25)
				if err != nil {
					return err
				}
				externalAccountID, err = chooseExternalAccount(accounts, "Select an external account")
				if err != nil {
					return err
				}
			}
			if externalAccountID == "" {
				return fmt.Errorf("external-account-id is required")
			}
			account, requestID, err := ctx.Services.RetrieveExternalAccount(cmd.Context(), api, externalAccountID)
			if ctx.Options.JSON {
				return printEnvelopeJSON(account, requestID, err)
			}
			if err != nil {
				return err
			}
			return printJSON(account)
		},
	}
	cmd.Flags().StringVar(&externalAccountID, "external-account-id", "", "external account id")
	return cmd
}

func newExternalAccountsCreateCmd(ctx *Context) *cobra.Command {
	var input app.CreateExternalAccountInput
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Preview or create an external account",
		RunE: func(cmd *cobra.Command, args []string) error {
			session, api, err := ctx.resolve(cmd.Context())
			if err != nil {
				return printEnvelopeJSON(nil, "", err)
			}
			if isInteractiveRequested(ctx.Options) {
				if input.Description == "" {
					input.Description, err = ui.PromptString("Description", true)
					if err != nil {
						return err
					}
				}
				if input.RoutingNumber == "" {
					input.RoutingNumber, err = ui.PromptString("Routing number", true)
					if err != nil {
						return err
					}
				}
				if input.AccountNumber == "" {
					input.AccountNumber, err = ui.PromptString("Account number", true)
					if err != nil {
						return err
					}
				}
				if input.AccountHolder == "" {
					input.AccountHolder, err = ui.PromptString("Account holder (optional)", false)
					if err != nil {
						return err
					}
				}
				if input.Funding == "" {
					input.Funding, err = ui.PromptString("Funding (optional)", false)
					if err != nil {
						return err
					}
				}
			}
			input.DryRun = &dryRun
			if dryRun {
				preview, err := ctx.Services.PreviewCreateExternalAccount(*session, input)
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
				preview, err := ctx.Services.PreviewCreateExternalAccount(*session, input)
				if err != nil {
					return err
				}
				if !ctx.Options.Yes {
					printPreview(preview)
					confirmed, err := ui.Confirm("Create this external account?")
					if err != nil || !confirmed {
						return err
					}
				}
				input.ConfirmationToken = preview.ConfirmationToken
			}
			data, requestID, err := ctx.Services.ExecuteCreateExternalAccount(cmd.Context(), api, *session, input)
			if ctx.Options.JSON {
				return printEnvelopeJSON(data, requestID, err)
			}
			if err != nil {
				return err
			}
			return printJSON(data)
		},
	}
	cmd.Flags().StringVar(&input.Description, "description", "", "external account description")
	cmd.Flags().StringVar(&input.RoutingNumber, "routing-number", "", "routing number")
	cmd.Flags().StringVar(&input.AccountNumber, "account-number", "", "account number")
	cmd.Flags().StringVar(&input.AccountHolder, "account-holder", "", "account holder")
	cmd.Flags().StringVar(&input.Funding, "funding", "", "funding type")
	cmd.Flags().StringVar(&input.IdempotencyKey, "idempotency-key", "", "idempotency key")
	cmd.Flags().StringVar(&input.ConfirmationToken, "confirmation-token", "", "confirmation token")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview only")
	return cmd
}

func newExternalAccountsUpdateCmd(ctx *Context) *cobra.Command {
	var input app.UpdateExternalAccountInput
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Preview or update an external account",
		RunE: func(cmd *cobra.Command, args []string) error {
			session, api, err := ctx.resolve(cmd.Context())
			if err != nil {
				return printEnvelopeJSON(nil, "", err)
			}
			if isInteractiveRequested(ctx.Options) {
				if input.ExternalAccountID == "" {
					accounts, _, err := ctx.Services.ListExternalAccounts(cmd.Context(), api, "", "", 25)
					if err != nil {
						return err
					}
					input.ExternalAccountID, err = chooseExternalAccount(accounts, "Select an external account")
					if err != nil {
						return err
					}
				}
				if input.Description == "" {
					input.Description, err = ui.PromptString("Updated description (optional)", false)
					if err != nil {
						return err
					}
				}
				if input.AccountHolder == "" {
					input.AccountHolder, err = ui.PromptString("Updated account holder (optional)", false)
					if err != nil {
						return err
					}
				}
				if input.Funding == "" {
					input.Funding, err = ui.PromptString("Updated funding (optional)", false)
					if err != nil {
						return err
					}
				}
				if input.Status == "" {
					input.Status, err = ui.PromptString("Updated status (optional)", false)
					if err != nil {
						return err
					}
				}
			}
			input.DryRun = &dryRun
			if dryRun {
				preview, err := ctx.Services.PreviewUpdateExternalAccount(*session, input)
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
				preview, err := ctx.Services.PreviewUpdateExternalAccount(*session, input)
				if err != nil {
					return err
				}
				if !ctx.Options.Yes {
					printPreview(preview)
					confirmed, err := ui.Confirm("Update this external account?")
					if err != nil || !confirmed {
						return err
					}
				}
				input.ConfirmationToken = preview.ConfirmationToken
			}
			data, requestID, err := ctx.Services.ExecuteUpdateExternalAccount(cmd.Context(), api, *session, input)
			if ctx.Options.JSON {
				return printEnvelopeJSON(data, requestID, err)
			}
			if err != nil {
				return err
			}
			return printJSON(data)
		},
	}
	cmd.Flags().StringVar(&input.ExternalAccountID, "external-account-id", "", "external account id")
	cmd.Flags().StringVar(&input.Description, "description", "", "updated external account description")
	cmd.Flags().StringVar(&input.AccountHolder, "account-holder", "", "updated account holder")
	cmd.Flags().StringVar(&input.Funding, "funding", "", "updated funding type")
	cmd.Flags().StringVar(&input.Status, "status", "", "updated status")
	cmd.Flags().StringVar(&input.IdempotencyKey, "idempotency-key", "", "idempotency key")
	cmd.Flags().StringVar(&input.ConfirmationToken, "confirmation-token", "", "confirmation token")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview only")
	return cmd
}
