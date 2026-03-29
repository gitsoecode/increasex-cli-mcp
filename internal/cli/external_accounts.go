package cli

import (
	"fmt"

	"github.com/gitsoecode/increasex-cli-mcp/internal/app"
	"github.com/gitsoecode/increasex-cli-mcp/internal/ui"
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
			}
			if len(accounts) == 0 {
				options = []ui.Option{
					{Label: "Create an external account", Value: "create"},
				}
			}
			for {
				action, err := promptSelectNavigation("External account actions", options, navBack, navExit)
				if err != nil {
					return bubbleNavigation(cmd, err)
				}
				if action == "create" {
					return invokeCommand(cmd, newExternalAccountsCreateCmd(ctx))
				}
				if len(accounts) == 0 {
					return nil
				}
				selected, err := chooseExternalAccount(accounts, "Select an external account")
				if err != nil {
					if isNavigateBack(err) {
						continue
					}
					return bubbleNavigation(cmd, err)
				}
				switch action {
				case "retrieve":
					return invokeCommand(cmd, newExternalAccountsRetrieveCmd(ctx), "--external-account-id", selected)
				case "update":
					return invokeCommand(cmd, newExternalAccountsUpdateCmd(ctx), "--external-account-id", selected)
				default:
					return nil
				}
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
					return bubbleNavigation(cmd, err)
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
				if err := promptCreateExternalAccountInput(&input); err != nil {
					return bubbleNavigation(cmd, err)
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
					confirmed, err := promptConfirmationNavigation("Create this external account?")
					if err != nil || !confirmed {
						return bubbleNavigation(cmd, err)
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
				accounts, _, err := ctx.Services.ListExternalAccounts(cmd.Context(), api, "", "", 25)
				if err != nil {
					return err
				}
				if err := promptUpdateExternalAccountInput(accounts, &input); err != nil {
					return bubbleNavigation(cmd, err)
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
					confirmed, err := promptConfirmationNavigation("Update this external account?")
					if err != nil || !confirmed {
						return bubbleNavigation(cmd, err)
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

func promptOptionalExternalAccountHolderCreate() (string, error) {
	return promptOptionalSelect("Account holder (optional)", []ui.Option{
		{Label: "Business", Value: "business"},
		{Label: "Individual", Value: "individual"},
		{Label: "Unknown", Value: "unknown"},
	})
}

func promptOptionalExternalAccountHolderUpdate() (string, error) {
	return promptOptionalSelect("Updated account holder (optional)", []ui.Option{
		{Label: "Business", Value: "business"},
		{Label: "Individual", Value: "individual"},
	})
}

func promptOptionalExternalAccountFunding() (string, error) {
	return promptOptionalSelect("Funding (optional)", []ui.Option{
		{Label: "Checking", Value: "checking"},
		{Label: "Savings", Value: "savings"},
		{Label: "General ledger", Value: "general_ledger"},
		{Label: "Other", Value: "other"},
	})
}

func promptOptionalExternalAccountStatus() (string, error) {
	return promptOptionalSelect("Updated status (optional)", []ui.Option{
		{Label: "Active", Value: "active"},
		{Label: "Archived", Value: "archived"},
	})
}

func promptOptionalSelect(label string, options []ui.Option) (string, error) {
	items := append([]ui.Option{{Label: "Skip", Value: "", Description: "Leave this unset"}}, options...)
	return promptSelectNavigation(label, items, navBack, navExit)
}

func promptCreateExternalAccountInput(input *app.CreateExternalAccountInput) error {
	step := 0
	for step < 5 {
		switch step {
		case 0:
			if input.Description != "" {
				step++
				continue
			}
			value, err := promptStringNavigation("Description", true)
			if err != nil {
				if isNavigateBack(err) {
					return err
				}
				return err
			}
			input.Description = value
		case 1:
			if input.RoutingNumber != "" {
				step++
				continue
			}
			value, err := promptStringNavigation("Routing number", true)
			if err != nil {
				if isNavigateBack(err) {
					step = max(0, step-1)
					continue
				}
				return err
			}
			input.RoutingNumber = value
		case 2:
			if input.AccountNumber != "" {
				step++
				continue
			}
			value, err := promptStringNavigation("Account number", true)
			if err != nil {
				if isNavigateBack(err) {
					step = max(0, step-1)
					continue
				}
				return err
			}
			input.AccountNumber = value
		case 3:
			if input.AccountHolder != "" {
				step++
				continue
			}
			value, err := promptOptionalExternalAccountHolderCreate()
			if err != nil {
				if isNavigateBack(err) {
					step = max(0, step-1)
					continue
				}
				return err
			}
			input.AccountHolder = value
		case 4:
			if input.Funding != "" {
				step++
				continue
			}
			value, err := promptOptionalExternalAccountFunding()
			if err != nil {
				if isNavigateBack(err) {
					step = max(0, step-1)
					continue
				}
				return err
			}
			input.Funding = value
		}
		step++
	}
	return nil
}

func promptUpdateExternalAccountInput(accounts []app.ExternalAccountSummary, input *app.UpdateExternalAccountInput) error {
	step := 0
	for step < 5 {
		switch step {
		case 0:
			if input.ExternalAccountID != "" {
				step++
				continue
			}
			value, err := chooseExternalAccount(accounts, "Select an external account")
			if err != nil {
				if isNavigateBack(err) {
					return err
				}
				return err
			}
			input.ExternalAccountID = value
		case 1:
			if input.Description != "" {
				step++
				continue
			}
			value, err := promptStringNavigation("Updated description (optional)", false)
			if err != nil {
				if isNavigateBack(err) {
					step = max(0, step-1)
					continue
				}
				return err
			}
			input.Description = value
		case 2:
			if input.AccountHolder != "" {
				step++
				continue
			}
			value, err := promptOptionalExternalAccountHolderUpdate()
			if err != nil {
				if isNavigateBack(err) {
					step = max(0, step-1)
					continue
				}
				return err
			}
			input.AccountHolder = value
		case 3:
			if input.Funding != "" {
				step++
				continue
			}
			value, err := promptOptionalExternalAccountFunding()
			if err != nil {
				if isNavigateBack(err) {
					step = max(0, step-1)
					continue
				}
				return err
			}
			input.Funding = value
		case 4:
			if input.Status != "" {
				step++
				continue
			}
			value, err := promptOptionalExternalAccountStatus()
			if err != nil {
				if isNavigateBack(err) {
					step = max(0, step-1)
					continue
				}
				return err
			}
			input.Status = value
		}
		step++
	}
	return nil
}
