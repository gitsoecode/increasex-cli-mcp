package cli

import (
	"fmt"
	"strings"

	"github.com/jessevaughan/increasex/internal/app"
	increasex "github.com/jessevaughan/increasex/internal/increase"
	"github.com/jessevaughan/increasex/internal/ui"
	"github.com/spf13/cobra"
)

func newTransferCmd(ctx *Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transfer",
		Short: "Create and manage transfers",
		RunE: func(cmd *cobra.Command, args []string) error {
			if terminalMenuRequested(ctx.Options) {
				return runTransferMenu(cmd, ctx)
			}
			return cmd.Help()
		},
	}
	cmd.AddCommand(
		newTransferInternalCmd(ctx),
		newTransferExternalCmd(ctx),
		newTransferListCmd(ctx),
		newTransferRetrieveCmd(ctx),
		newTransferQueueCmd(ctx),
		newTransferApproveCmd(ctx),
		newTransferCancelCmd(ctx),
	)
	return cmd
}

func runTransferMenu(cmd *cobra.Command, ctx *Context) error {
	for {
		choice, err := ui.PromptSelect("Transfers", []ui.Option{
			{Label: "Create an account transfer", Value: "internal", Description: "Move funds between Increase accounts"},
			{Label: "Create an external transfer", Value: "external", Description: "Send ACH, Real-Time Payments, FedNow, or wire"},
			{Label: "List transfers", Value: "list", Description: "Review recent transfers by rail"},
			{Label: "Retrieve a transfer", Value: "retrieve", Description: "Inspect one transfer by rail and id"},
			{Label: "View approval queue", Value: "queue", Description: "List pending approval transfers"},
			{Label: "Approve a transfer", Value: "approve", Description: "Approve a pending transfer"},
			{Label: "Cancel a transfer", Value: "cancel", Description: "Cancel a transfer"},
			{Label: "Back", Value: "back", Description: "Return to the previous menu"},
			{Label: "Exit", Value: "exit", Description: "Return to the shell"},
		})
		if err != nil {
			return err
		}
		switch choice {
		case "internal":
			if err := invokeCommand(cmd, newTransferInternalCmd(ctx)); err != nil {
				return err
			}
		case "external":
			if err := invokeCommand(cmd, newTransferExternalCmd(ctx)); err != nil {
				return err
			}
		case "list":
			if err := invokeCommand(cmd, newTransferListCmd(ctx)); err != nil {
				return err
			}
		case "retrieve":
			if err := invokeCommand(cmd, newTransferRetrieveCmd(ctx)); err != nil {
				return err
			}
		case "queue":
			if err := invokeCommand(cmd, newTransferQueueCmd(ctx)); err != nil {
				return err
			}
		case "approve":
			if err := invokeCommand(cmd, newTransferApproveCmd(ctx)); err != nil {
				return err
			}
		case "cancel":
			if err := invokeCommand(cmd, newTransferCancelCmd(ctx)); err != nil {
				return err
			}
		case "back", "exit":
			return nil
		}
	}
}

func newTransferInternalCmd(ctx *Context) *cobra.Command {
	var input app.MoveMoneyInternalInput
	var dryRun bool
	cmd := &cobra.Command{
		Use:     "internal",
		Aliases: []string{"account"},
		Short:   "Preview or create an account transfer",
		RunE: func(cmd *cobra.Command, args []string) error {
			session, api, err := ctx.resolve(cmd.Context())
			if err != nil {
				return printEnvelopeJSON(nil, "", err)
			}
			if isInteractiveRequested(ctx.Options) {
				if err := promptInternalTransferInput(cmd, ctx, api, &input); err != nil {
					return err
				}
			}
			input.DryRun = &dryRun
			if dryRun {
				preview, err := ctx.Services.PreviewInternalTransfer(*session, input)
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
				preview, err := ctx.Services.PreviewInternalTransfer(*session, input)
				if err != nil {
					return err
				}
				if !ctx.Options.Yes {
					printPreview(preview)
					confirmed, err := ui.Confirm("Execute this account transfer?")
					if err != nil || !confirmed {
						return err
					}
				}
				input.ConfirmationToken = preview.ConfirmationToken
			}
			data, requestID, err := ctx.Services.ExecuteInternalTransfer(cmd.Context(), api, *session, input)
			if ctx.Options.JSON {
				return printEnvelopeJSON(data, requestID, err)
			}
			if err != nil {
				return err
			}
			return printJSON(data)
		},
	}
	cmd.Flags().StringVar(&input.FromAccountID, "from-account-id", "", "source account id")
	cmd.Flags().StringVar(&input.ToAccountID, "to-account-id", "", "destination account id")
	cmd.Flags().Int64Var(&input.AmountCents, "amount-cents", 0, "amount in minor units")
	cmd.Flags().StringVar(&input.Description, "description", "", "transfer description")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview only")
	cmd.Flags().Bool("require-approval", false, "queue this transfer for approval")
	cmd.Flags().StringVar(&input.IdempotencyKey, "idempotency-key", "", "idempotency key")
	cmd.Flags().StringVar(&input.ConfirmationToken, "confirmation-token", "", "confirmation token")
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("require-approval") {
			value, err := cmd.Flags().GetBool("require-approval")
			if err != nil {
				return err
			}
			input.RequireApproval = boolPtr(value)
		}
		return nil
	}
	return cmd
}

func newTransferExternalCmd(ctx *Context) *cobra.Command {
	var rail string
	var dryRun bool
	var achInput app.ACHTransferInput
	var rtpInput app.RTPTransferInput
	var fednowInput app.FedNowTransferInput
	var wireInput app.WireTransferInput
	cmd := &cobra.Command{
		Use:   "external",
		Short: "Preview or create an external transfer",
		RunE: func(cmd *cobra.Command, args []string) error {
			session, api, err := ctx.resolve(cmd.Context())
			if err != nil {
				return printEnvelopeJSON(nil, "", err)
			}
			if rail == "" && isInteractiveRequested(ctx.Options) {
				rail, err = promptExternalRail()
				if err != nil {
					return err
				}
			}
			rail = normalizeTransferRail(rail)
			switch rail {
			case "ach":
				if isInteractiveRequested(ctx.Options) {
					if err := promptACHTransferInput(cmd, ctx, api, &achInput); err != nil {
						return err
					}
				}
				achInput.DryRun = &dryRun
				return runExternalACH(cmd, ctx, *session, api, achInput)
			case "real_time_payments":
				if isInteractiveRequested(ctx.Options) {
					if err := promptRTPTransferInput(cmd, ctx, api, &rtpInput); err != nil {
						return err
					}
				}
				rtpInput.DryRun = &dryRun
				return runExternalRTP(cmd, ctx, *session, api, rtpInput)
			case "fednow":
				if isInteractiveRequested(ctx.Options) {
					if err := promptFedNowTransferInput(cmd, ctx, api, &fednowInput); err != nil {
						return err
					}
				}
				fednowInput.DryRun = &dryRun
				return runExternalFedNow(cmd, ctx, *session, api, fednowInput)
			case "wire":
				if isInteractiveRequested(ctx.Options) {
					if err := promptWireTransferInput(cmd, ctx, api, &wireInput); err != nil {
						return err
					}
				}
				wireInput.DryRun = &dryRun
				return runExternalWire(cmd, ctx, *session, api, wireInput)
			default:
				return fmt.Errorf("rail is required: ach, real_time_payments, fednow, or wire")
			}
		},
	}
	cmd.Flags().StringVar(&rail, "rail", "", "rail: ach, real_time_payments, fednow, wire")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview only")

	cmd.Flags().StringVar(&achInput.AccountID, "account-id", "", "ACH source account id")
	cmd.Flags().Int64Var(&achInput.AmountCents, "amount-cents", 0, "ACH amount in minor units")
	cmd.Flags().StringVar(&achInput.StatementDescriptor, "statement-descriptor", "", "statement descriptor")
	cmd.Flags().StringVar(&achInput.AccountNumber, "account-number", "", "destination account number")
	cmd.Flags().StringVar(&achInput.RoutingNumber, "routing-number", "", "routing number")
	cmd.Flags().StringVar(&achInput.ExternalAccountID, "external-account-id", "", "external account id")
	cmd.Flags().StringVar(&achInput.IndividualName, "individual-name", "", "recipient individual name")
	cmd.Flags().StringVar(&achInput.CompanyName, "company-name", "", "company name")
	cmd.Flags().Bool("require-approval", false, "queue this ACH transfer for approval")
	cmd.Flags().StringVar(&achInput.IdempotencyKey, "idempotency-key", "", "idempotency key")
	cmd.Flags().StringVar(&achInput.ConfirmationToken, "confirmation-token", "", "confirmation token")

	cmd.Flags().StringVar(&rtpInput.CreditorName, "rtp-creditor-name", "", "creditor name")
	cmd.Flags().StringVar(&rtpInput.RemittanceInformation, "rtp-remittance-information", "", "remittance information")
	cmd.Flags().StringVar(&rtpInput.SourceAccountNumberID, "rtp-source-account-number-id", "", "source account number id")
	cmd.Flags().StringVar(&rtpInput.DestinationAccountNumber, "rtp-destination-account-number", "", "destination account number")
	cmd.Flags().StringVar(&rtpInput.DestinationRoutingNumber, "rtp-destination-routing-number", "", "destination routing number")
	cmd.Flags().StringVar(&rtpInput.ExternalAccountID, "rtp-external-account-id", "", "external account id")
	cmd.Flags().StringVar(&rtpInput.DebtorName, "rtp-debtor-name", "", "debtor name")
	cmd.Flags().Int64Var(&rtpInput.AmountCents, "rtp-amount-cents", 0, "Real-Time Payments amount in minor units")
	cmd.Flags().Bool("rtp-require-approval", false, "queue this Real-Time Payments transfer for approval")
	cmd.Flags().StringVar(&rtpInput.IdempotencyKey, "rtp-idempotency-key", "", "idempotency key")
	cmd.Flags().StringVar(&rtpInput.ConfirmationToken, "rtp-confirmation-token", "", "confirmation token")

	cmd.Flags().StringVar(&fednowInput.AccountID, "fednow-account-id", "", "FedNow source account id")
	cmd.Flags().Int64Var(&fednowInput.AmountCents, "fednow-amount-cents", 0, "FedNow amount in minor units")
	cmd.Flags().StringVar(&fednowInput.CreditorName, "fednow-creditor-name", "", "creditor name")
	cmd.Flags().StringVar(&fednowInput.DebtorName, "fednow-debtor-name", "", "debtor name")
	cmd.Flags().StringVar(&fednowInput.SourceAccountNumberID, "fednow-source-account-number-id", "", "source account number id")
	cmd.Flags().StringVar(&fednowInput.AccountNumber, "fednow-account-number", "", "destination account number")
	cmd.Flags().StringVar(&fednowInput.RoutingNumber, "fednow-routing-number", "", "routing number")
	cmd.Flags().StringVar(&fednowInput.ExternalAccountID, "fednow-external-account-id", "", "external account id")
	cmd.Flags().StringVar(&fednowInput.UnstructuredRemittanceInformation, "fednow-remittance", "", "unstructured remittance info")
	cmd.Flags().Bool("fednow-require-approval", false, "queue this FedNow transfer for approval")
	cmd.Flags().StringVar(&fednowInput.IdempotencyKey, "fednow-idempotency-key", "", "idempotency key")
	cmd.Flags().StringVar(&fednowInput.ConfirmationToken, "fednow-confirmation-token", "", "confirmation token")

	cmd.Flags().StringVar(&wireInput.AccountID, "wire-account-id", "", "wire source account id")
	cmd.Flags().Int64Var(&wireInput.AmountCents, "wire-amount-cents", 0, "wire amount in minor units")
	cmd.Flags().StringVar(&wireInput.BeneficiaryName, "wire-beneficiary-name", "", "beneficiary name")
	cmd.Flags().StringVar(&wireInput.MessageToRecipient, "wire-message-to-recipient", "", "message to recipient")
	cmd.Flags().StringVar(&wireInput.AccountNumber, "wire-account-number", "", "destination account number")
	cmd.Flags().StringVar(&wireInput.RoutingNumber, "wire-routing-number", "", "routing number")
	cmd.Flags().StringVar(&wireInput.ExternalAccountID, "wire-external-account-id", "", "external account id")
	cmd.Flags().Bool("wire-require-approval", false, "queue this wire transfer for approval")
	cmd.Flags().StringVar(&wireInput.IdempotencyKey, "wire-idempotency-key", "", "idempotency key")
	cmd.Flags().StringVar(&wireInput.ConfirmationToken, "wire-confirmation-token", "", "confirmation token")

	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("require-approval") {
			value, err := cmd.Flags().GetBool("require-approval")
			if err != nil {
				return err
			}
			achInput.RequireApproval = boolPtr(value)
		}
		if cmd.Flags().Changed("rtp-require-approval") {
			value, err := cmd.Flags().GetBool("rtp-require-approval")
			if err != nil {
				return err
			}
			rtpInput.RequireApproval = boolPtr(value)
		}
		if cmd.Flags().Changed("fednow-require-approval") {
			value, err := cmd.Flags().GetBool("fednow-require-approval")
			if err != nil {
				return err
			}
			fednowInput.RequireApproval = boolPtr(value)
		}
		if cmd.Flags().Changed("wire-require-approval") {
			value, err := cmd.Flags().GetBool("wire-require-approval")
			if err != nil {
				return err
			}
			wireInput.RequireApproval = boolPtr(value)
		}
		return nil
	}
	return cmd
}

func newTransferListCmd(ctx *Context) *cobra.Command {
	input := app.ListTransfersInput{Limit: 20}
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List transfers by rail",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, api, err := ctx.resolve(cmd.Context())
			if err != nil {
				return printEnvelopeJSON(nil, "", err)
			}
			if input.Rail == "" && isInteractiveRequested(ctx.Options) {
				input.Rail, err = promptRail("Transfer rail")
				if err != nil {
					return err
				}
			}
			input.Rail = normalizeTransferRail(input.Rail)
			items, requestID, err := ctx.Services.ListTransfers(cmd.Context(), api, input)
			if ctx.Options.JSON {
				return printEnvelopeJSON(map[string]any{"transfers": items}, requestID, err)
			}
			if err != nil {
				return err
			}
			printTransfers("Transfers", items)
			if !isInteractiveRequested(ctx.Options) || len(items) == 0 {
				return nil
			}
			action, err := ui.PromptSelect("Transfer actions", []ui.Option{
				{Label: "Retrieve a transfer", Value: "retrieve"},
				{Label: "Approve a pending transfer", Value: "approve"},
				{Label: "Cancel a transfer", Value: "cancel"},
				{Label: "Back", Value: "back"},
			})
			if err != nil {
				return err
			}
			selected, err := chooseTransfer(items, "Select a transfer")
			if err != nil {
				return err
			}
			transfer, ok := findTransferSummary(items, selected)
			if !ok {
				return fmt.Errorf("selected transfer not found")
			}
			switch action {
			case "retrieve":
				return invokeCommand(cmd, newTransferRetrieveCmd(ctx), "--rail", transfer.Rail, "--transfer-id", transfer.ID)
			case "approve":
				return invokeCommand(cmd, newTransferApproveCmd(ctx), "--rail", transfer.Rail, "--transfer-id", transfer.ID)
			case "cancel":
				return invokeCommand(cmd, newTransferCancelCmd(ctx), "--rail", transfer.Rail, "--transfer-id", transfer.ID)
			default:
				return nil
			}
		},
	}
	cmd.Flags().StringVar(&input.Rail, "rail", "", "rail: account, ach, real_time_payments, fednow, wire")
	cmd.Flags().StringVar(&input.AccountID, "account-id", "", "account id filter")
	cmd.Flags().StringVar(&input.ExternalAccountID, "external-account-id", "", "external account id filter")
	cmd.Flags().StringVar(&input.Status, "status", "", "status filter")
	cmd.Flags().StringVar(&input.Since, "since", "", "RFC3339 lower bound")
	cmd.Flags().StringVar(&input.Cursor, "cursor", "", "page cursor")
	cmd.Flags().Int64Var(&input.Limit, "limit", 20, "maximum transfers to return")
	return cmd
}

func newTransferRetrieveCmd(ctx *Context) *cobra.Command {
	var rail, transferID string
	var limit int64
	cmd := &cobra.Command{
		Use:   "retrieve",
		Short: "Retrieve a transfer by rail and id",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, api, err := ctx.resolve(cmd.Context())
			if err != nil {
				return printEnvelopeJSON(nil, "", err)
			}
			if rail == "" && isInteractiveRequested(ctx.Options) {
				rail, err = promptRail("Transfer rail")
				if err != nil {
					return err
				}
			}
			rail = normalizeTransferRail(rail)
			if transferID == "" && isInteractiveRequested(ctx.Options) {
				items, _, err := ctx.Services.ListTransfers(cmd.Context(), api, app.ListTransfersInput{
					Rail:  rail,
					Limit: limit,
				})
				if err != nil {
					return err
				}
				transferID, err = chooseTransfer(items, "Select a transfer")
				if err != nil {
					return err
				}
			}
			result, requestID, err := ctx.Services.RetrieveTransfer(cmd.Context(), api, rail, transferID)
			if ctx.Options.JSON {
				return printEnvelopeJSON(result, requestID, err)
			}
			if err != nil {
				return err
			}
			return printJSON(result)
		},
	}
	cmd.Flags().StringVar(&rail, "rail", "", "rail: account, ach, real_time_payments, fednow, wire")
	cmd.Flags().StringVar(&transferID, "transfer-id", "", "transfer id")
	cmd.Flags().Int64Var(&limit, "limit", 20, "maximum transfers to inspect when prompting")
	return cmd
}

func newTransferQueueCmd(ctx *Context) *cobra.Command {
	var rail string
	var limit int64
	cmd := &cobra.Command{
		Use:   "queue",
		Short: "List pending approval transfers",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, api, err := ctx.resolve(cmd.Context())
			if err != nil {
				return printEnvelopeJSON(nil, "", err)
			}
			if rail == "" && isInteractiveRequested(ctx.Options) {
				rail, err = promptRail("Transfer rail")
				if err != nil {
					return err
				}
			}
			rail = normalizeTransferRail(rail)
			items, requestID, err := ctx.Services.ListTransferQueue(cmd.Context(), api, rail, limit)
			if ctx.Options.JSON {
				return printEnvelopeJSON(map[string]any{"transfers": items}, requestID, err)
			}
			if err != nil {
				return err
			}
			printTransfers("Pending Approval Queue", items)
			if !isInteractiveRequested(ctx.Options) || len(items) == 0 {
				return nil
			}
			action, err := ui.PromptSelect("Queue actions", []ui.Option{
				{Label: "Approve a transfer", Value: "approve"},
				{Label: "Cancel a transfer", Value: "cancel"},
				{Label: "Back", Value: "back"},
			})
			if err != nil {
				return err
			}
			selected, err := chooseTransfer(items, "Select a transfer")
			if err != nil {
				return err
			}
			if action == "approve" {
				return invokeCommand(cmd, newTransferApproveCmd(ctx), "--rail", rail, "--transfer-id", selected)
			}
			if action == "cancel" {
				return invokeCommand(cmd, newTransferCancelCmd(ctx), "--rail", rail, "--transfer-id", selected)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&rail, "rail", "", "rail: account, ach, real_time_payments, fednow, wire")
	cmd.Flags().Int64Var(&limit, "limit", 20, "maximum transfers to return")
	return cmd
}

func newTransferApproveCmd(ctx *Context) *cobra.Command {
	var input app.TransferActionInput
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "approve",
		Short: "Preview or approve a pending transfer",
		RunE: func(cmd *cobra.Command, args []string) error {
			session, api, err := ctx.resolve(cmd.Context())
			if err != nil {
				return printEnvelopeJSON(nil, "", err)
			}
			if err := promptTransferAction(cmd, ctx, api, &input, true); err != nil {
				return err
			}
			input.DryRun = &dryRun
			if dryRun {
				preview, err := ctx.Services.PreviewApproveTransfer(*session, input)
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
				preview, err := ctx.Services.PreviewApproveTransfer(*session, input)
				if err != nil {
					return err
				}
				if !ctx.Options.Yes {
					printPreview(preview)
					confirmed, err := ui.Confirm("Approve this transfer?")
					if err != nil || !confirmed {
						return err
					}
				}
				input.ConfirmationToken = preview.ConfirmationToken
			}
			data, requestID, err := ctx.Services.ExecuteApproveTransfer(cmd.Context(), api, *session, input)
			if ctx.Options.JSON {
				return printEnvelopeJSON(data, requestID, err)
			}
			if err != nil {
				return err
			}
			return printJSON(data)
		},
	}
	cmd.Flags().StringVar(&input.Rail, "rail", "", "rail: account, ach, real_time_payments, fednow, wire")
	cmd.Flags().StringVar(&input.TransferID, "transfer-id", "", "transfer id")
	cmd.Flags().StringVar(&input.ConfirmationToken, "confirmation-token", "", "confirmation token")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview only")
	return cmd
}

func newTransferCancelCmd(ctx *Context) *cobra.Command {
	var input app.TransferActionInput
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "cancel",
		Short: "Preview or cancel a transfer",
		RunE: func(cmd *cobra.Command, args []string) error {
			session, api, err := ctx.resolve(cmd.Context())
			if err != nil {
				return printEnvelopeJSON(nil, "", err)
			}
			if err := promptTransferAction(cmd, ctx, api, &input, false); err != nil {
				return err
			}
			input.DryRun = &dryRun
			if dryRun {
				preview, err := ctx.Services.PreviewCancelTransfer(*session, input)
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
				preview, err := ctx.Services.PreviewCancelTransfer(*session, input)
				if err != nil {
					return err
				}
				if !ctx.Options.Yes {
					printPreview(preview)
					confirmed, err := ui.Confirm("Cancel this transfer?")
					if err != nil || !confirmed {
						return err
					}
				}
				input.ConfirmationToken = preview.ConfirmationToken
			}
			data, requestID, err := ctx.Services.ExecuteCancelTransfer(cmd.Context(), api, *session, input)
			if ctx.Options.JSON {
				return printEnvelopeJSON(data, requestID, err)
			}
			if err != nil {
				return err
			}
			return printJSON(data)
		},
	}
	cmd.Flags().StringVar(&input.Rail, "rail", "", "rail: account, ach, real_time_payments, fednow, wire")
	cmd.Flags().StringVar(&input.TransferID, "transfer-id", "", "transfer id")
	cmd.Flags().StringVar(&input.ConfirmationToken, "confirmation-token", "", "confirmation token")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview only")
	return cmd
}

func promptInternalTransferInput(cmd *cobra.Command, ctx *Context, api *increasex.Client, input *app.MoveMoneyInternalInput) error {
	accounts, _, err := ctx.Services.ListAccounts(cmd.Context(), api, "open", 25, "")
	if err != nil {
		return err
	}
	if input.FromAccountID == "" {
		input.FromAccountID, err = chooseAccount(accounts, "Select source account")
		if err != nil {
			return err
		}
	}
	if input.ToAccountID == "" {
		input.ToAccountID, err = chooseAccount(accounts, "Select destination account")
		if err != nil {
			return err
		}
	}
	if input.AmountCents == 0 {
		input.AmountCents, err = promptInt64("Amount in cents", true)
		if err != nil {
			return err
		}
	}
	if input.Description == "" {
		input.Description, err = ui.PromptString("Description (optional)", false)
		if err != nil {
			return err
		}
	}
	if input.RequireApproval == nil {
		requireApproval, err := promptBool("Transfer handling", "Queue for approval", "Transfer without approval")
		if err != nil {
			return err
		}
		input.RequireApproval = boolPtr(requireApproval)
	}
	return nil
}

func promptACHTransferInput(cmd *cobra.Command, ctx *Context, api *increasex.Client, input *app.ACHTransferInput) error {
	accounts, _, err := ctx.Services.ListAccounts(cmd.Context(), api, "open", 25, "")
	if err != nil {
		return err
	}
	if input.AccountID == "" {
		input.AccountID, err = chooseAccount(accounts, "Select source account")
		if err != nil {
			return err
		}
	}
	if input.AmountCents == 0 {
		input.AmountCents, err = promptInt64("Amount in cents", true)
		if err != nil {
			return err
		}
	}
	if input.StatementDescriptor == "" {
		input.StatementDescriptor, err = ui.PromptString("Statement descriptor", true)
		if err != nil {
			return err
		}
	}
	if err := promptExternalDestination(cmd, ctx, api, &input.ExternalAccountID, &input.AccountNumber, &input.RoutingNumber); err != nil {
		return err
	}
	if input.IndividualName == "" && input.CompanyName == "" {
		payeeType, err := ui.PromptSelect("Recipient type", []ui.Option{
			{Label: "Individual", Value: "individual"},
			{Label: "Company", Value: "company"},
			{Label: "Skip", Value: "skip"},
		})
		if err != nil {
			return err
		}
		switch payeeType {
		case "individual":
			input.IndividualName, err = ui.PromptString("Individual name", true)
			if err != nil {
				return err
			}
		case "company":
			input.CompanyName, err = ui.PromptString("Company name", true)
			if err != nil {
				return err
			}
		}
	}
	if input.RequireApproval == nil {
		requireApproval, err := promptBool("Transfer handling", "Queue for approval", "Transfer without approval")
		if err != nil {
			return err
		}
		input.RequireApproval = boolPtr(requireApproval)
	}
	return nil
}

func promptRTPTransferInput(cmd *cobra.Command, ctx *Context, api *increasex.Client, input *app.RTPTransferInput) error {
	if input.AmountCents == 0 {
		value, err := promptInt64("Amount in cents", true)
		if err != nil {
			return err
		}
		input.AmountCents = value
	}
	if input.CreditorName == "" {
		value, err := ui.PromptString("Creditor name", true)
		if err != nil {
			return err
		}
		input.CreditorName = value
	}
	if input.SourceAccountNumberID == "" {
		value, err := ui.PromptString("Source account number id", true)
		if err != nil {
			return err
		}
		input.SourceAccountNumberID = value
	}
	if input.RemittanceInformation == "" {
		value, err := ui.PromptString("Remittance information (optional)", false)
		if err != nil {
			return err
		}
		input.RemittanceInformation = value
	}
	if input.DebtorName == "" {
		value, err := ui.PromptString("Debtor name (optional)", false)
		if err != nil {
			return err
		}
		input.DebtorName = value
	}
	if err := promptExternalDestination(cmd, ctx, api, &input.ExternalAccountID, &input.DestinationAccountNumber, &input.DestinationRoutingNumber); err != nil {
		return err
	}
	if input.RequireApproval == nil {
		requireApproval, err := promptBool("Transfer handling", "Queue for approval", "Transfer without approval")
		if err != nil {
			return err
		}
		input.RequireApproval = boolPtr(requireApproval)
	}
	return nil
}

func promptFedNowTransferInput(cmd *cobra.Command, ctx *Context, api *increasex.Client, input *app.FedNowTransferInput) error {
	accounts, _, err := ctx.Services.ListAccounts(cmd.Context(), api, "open", 25, "")
	if err != nil {
		return err
	}
	if input.AccountID == "" {
		input.AccountID, err = chooseAccount(accounts, "Select source account")
		if err != nil {
			return err
		}
	}
	if input.AmountCents == 0 {
		input.AmountCents, err = promptInt64("Amount in cents", true)
		if err != nil {
			return err
		}
	}
	if input.CreditorName == "" {
		input.CreditorName, err = ui.PromptString("Creditor name", true)
		if err != nil {
			return err
		}
	}
	if input.DebtorName == "" {
		input.DebtorName, err = ui.PromptString("Debtor name", true)
		if err != nil {
			return err
		}
	}
	if input.SourceAccountNumberID == "" {
		input.SourceAccountNumberID, err = ui.PromptString("Source account number id", true)
		if err != nil {
			return err
		}
	}
	if input.UnstructuredRemittanceInformation == "" {
		input.UnstructuredRemittanceInformation, err = ui.PromptString("Unstructured remittance information (optional)", false)
		if err != nil {
			return err
		}
	}
	if err := promptExternalDestination(cmd, ctx, api, &input.ExternalAccountID, &input.AccountNumber, &input.RoutingNumber); err != nil {
		return err
	}
	if input.RequireApproval == nil {
		requireApproval, err := promptBool("Transfer handling", "Queue for approval", "Transfer without approval")
		if err != nil {
			return err
		}
		input.RequireApproval = boolPtr(requireApproval)
	}
	return nil
}

func promptWireTransferInput(cmd *cobra.Command, ctx *Context, api *increasex.Client, input *app.WireTransferInput) error {
	accounts, _, err := ctx.Services.ListAccounts(cmd.Context(), api, "open", 25, "")
	if err != nil {
		return err
	}
	if input.AccountID == "" {
		input.AccountID, err = chooseAccount(accounts, "Select source account")
		if err != nil {
			return err
		}
	}
	if input.AmountCents == 0 {
		input.AmountCents, err = promptInt64("Amount in cents", true)
		if err != nil {
			return err
		}
	}
	if input.BeneficiaryName == "" {
		input.BeneficiaryName, err = ui.PromptString("Beneficiary name", true)
		if err != nil {
			return err
		}
	}
	if input.MessageToRecipient == "" {
		input.MessageToRecipient, err = ui.PromptString("Message to recipient (optional)", false)
		if err != nil {
			return err
		}
	}
	if err := promptExternalDestination(cmd, ctx, api, &input.ExternalAccountID, &input.AccountNumber, &input.RoutingNumber); err != nil {
		return err
	}
	if input.RequireApproval == nil {
		requireApproval, err := promptBool("Transfer handling", "Queue for approval", "Transfer without approval")
		if err != nil {
			return err
		}
		input.RequireApproval = boolPtr(requireApproval)
	}
	return nil
}

func promptExternalDestination(cmd *cobra.Command, ctx *Context, api *increasex.Client, externalAccountID, accountNumber, routingNumber *string) error {
	if strings.TrimSpace(*externalAccountID) != "" || (strings.TrimSpace(*accountNumber) != "" && strings.TrimSpace(*routingNumber) != "") {
		return nil
	}
	choice, err := ui.PromptSelect("Destination", []ui.Option{
		{Label: "Use a stored external account", Value: "stored"},
		{Label: "Enter bank details manually", Value: "manual"},
	})
	if err != nil {
		return err
	}
	if choice == "stored" {
		externalAccounts, _, err := ctx.Services.ListExternalAccounts(cmd.Context(), api, "active", "", 25)
		if err != nil {
			return err
		}
		if len(externalAccounts) == 0 {
			fmt.Println(mutedStyle.Render("No active external accounts found, switching to manual entry."))
		} else {
			selected, err := chooseExternalAccount(externalAccounts, "Select an external account")
			if err != nil {
				return err
			}
			*externalAccountID = selected
			return nil
		}
	}
	value, err := ui.PromptString("Routing number", true)
	if err != nil {
		return err
	}
	*routingNumber = value
	value, err = ui.PromptString("Account number", true)
	if err != nil {
		return err
	}
	*accountNumber = value
	return nil
}

func promptTransferAction(cmd *cobra.Command, ctx *Context, api *increasex.Client, input *app.TransferActionInput, pendingOnly bool) error {
	if !isInteractiveRequested(ctx.Options) {
		input.Rail = normalizeTransferRail(input.Rail)
		return nil
	}
	var err error
	if input.Rail == "" {
		input.Rail, err = promptRail("Transfer rail")
		if err != nil {
			return err
		}
	}
	input.Rail = normalizeTransferRail(input.Rail)
	if input.TransferID != "" {
		return nil
	}
	var items []app.TransferSummary
	if pendingOnly {
		items, _, err = ctx.Services.ListTransferQueue(cmd.Context(), api, input.Rail, 20)
	} else {
		items, _, err = ctx.Services.ListTransfers(cmd.Context(), api, app.ListTransfersInput{
			Rail:  input.Rail,
			Limit: 20,
		})
	}
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return fmt.Errorf("no transfers available for rail %s", input.Rail)
	}
	input.TransferID, err = chooseTransfer(items, "Select a transfer")
	return err
}

func promptExternalRail() (string, error) {
	value, err := ui.PromptSelect("Transfer rail", []ui.Option{
		{Label: "ACH", Value: "ach"},
		{Label: "Real-Time Payments", Value: "real_time_payments"},
		{Label: "FedNow", Value: "fednow"},
		{Label: "Wire", Value: "wire"},
	})
	if err != nil {
		return "", err
	}
	return value, nil
}

func normalizeTransferRail(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "internal":
		return "account"
	case "account_transfer":
		return "account"
	case "rtp":
		return "real_time_payments"
	case "real-time-payments":
		return "real_time_payments"
	default:
		return strings.TrimSpace(strings.ToLower(value))
	}
}

func findTransferSummary(items []app.TransferSummary, transferID string) (app.TransferSummary, bool) {
	for _, item := range items {
		if item.ID == transferID {
			return item, true
		}
	}
	return app.TransferSummary{}, false
}

func runExternalACH(cmd *cobra.Command, ctx *Context, session app.Session, api any, input app.ACHTransferInput) error {
	client := api.(*increasex.Client)
	if app.IsDryRun(input.DryRun) {
		preview, err := ctx.Services.PreviewExternalACH(session, input)
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
		preview, err := ctx.Services.PreviewExternalACH(session, input)
		if err != nil {
			return err
		}
		if !ctx.Options.Yes {
			printPreview(preview)
			confirmed, err := ui.Confirm("Execute this ACH transfer?")
			if err != nil || !confirmed {
				return err
			}
		}
		input.ConfirmationToken = preview.ConfirmationToken
	}
	data, requestID, err := ctx.Services.ExecuteExternalACH(cmd.Context(), client, session, input)
	if ctx.Options.JSON {
		return printEnvelopeJSON(data, requestID, err)
	}
	if err != nil {
		return err
	}
	return printJSON(data)
}

func runExternalRTP(cmd *cobra.Command, ctx *Context, session app.Session, api any, input app.RTPTransferInput) error {
	client := api.(*increasex.Client)
	if app.IsDryRun(input.DryRun) {
		preview, err := ctx.Services.PreviewExternalRTP(session, input)
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
		preview, err := ctx.Services.PreviewExternalRTP(session, input)
		if err != nil {
			return err
		}
		if !ctx.Options.Yes {
			printPreview(preview)
			confirmed, err := ui.Confirm("Execute this Real-Time Payments transfer?")
			if err != nil || !confirmed {
				return err
			}
		}
		input.ConfirmationToken = preview.ConfirmationToken
	}
	data, requestID, err := ctx.Services.ExecuteExternalRTP(cmd.Context(), client, session, input)
	if ctx.Options.JSON {
		return printEnvelopeJSON(data, requestID, err)
	}
	if err != nil {
		return err
	}
	return printJSON(data)
}

func runExternalFedNow(cmd *cobra.Command, ctx *Context, session app.Session, api any, input app.FedNowTransferInput) error {
	client := api.(*increasex.Client)
	if app.IsDryRun(input.DryRun) {
		preview, err := ctx.Services.PreviewExternalFedNow(session, input)
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
		preview, err := ctx.Services.PreviewExternalFedNow(session, input)
		if err != nil {
			return err
		}
		if !ctx.Options.Yes {
			printPreview(preview)
			confirmed, err := ui.Confirm("Execute this FedNow transfer?")
			if err != nil || !confirmed {
				return err
			}
		}
		input.ConfirmationToken = preview.ConfirmationToken
	}
	data, requestID, err := ctx.Services.ExecuteExternalFedNow(cmd.Context(), client, session, input)
	if ctx.Options.JSON {
		return printEnvelopeJSON(data, requestID, err)
	}
	if err != nil {
		return err
	}
	return printJSON(data)
}

func runExternalWire(cmd *cobra.Command, ctx *Context, session app.Session, api any, input app.WireTransferInput) error {
	client := api.(*increasex.Client)
	if app.IsDryRun(input.DryRun) {
		preview, err := ctx.Services.PreviewExternalWire(session, input)
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
		preview, err := ctx.Services.PreviewExternalWire(session, input)
		if err != nil {
			return err
		}
		if !ctx.Options.Yes {
			printPreview(preview)
			confirmed, err := ui.Confirm("Execute this wire transfer?")
			if err != nil || !confirmed {
				return err
			}
		}
		input.ConfirmationToken = preview.ConfirmationToken
	}
	data, requestID, err := ctx.Services.ExecuteExternalWire(cmd.Context(), client, session, input)
	if ctx.Options.JSON {
		return printEnvelopeJSON(data, requestID, err)
	}
	if err != nil {
		return err
	}
	return printJSON(data)
}
