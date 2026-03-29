package cli

import (
	"fmt"

	"github.com/jessevaughan/increasex/internal/app"
	increasex "github.com/jessevaughan/increasex/internal/increase"
	"github.com/jessevaughan/increasex/internal/ui"
	"github.com/spf13/cobra"
)

func newTransferCmd(ctx *Context) *cobra.Command {
	cmd := &cobra.Command{Use: "transfer", Short: "Preview and execute transfers"}
	cmd.AddCommand(newTransferInternalCmd(ctx), newTransferExternalCmd(ctx))
	return cmd
}

func newTransferInternalCmd(ctx *Context) *cobra.Command {
	var input app.MoveMoneyInternalInput
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "internal",
		Short: "Move money between Increase accounts",
		RunE: func(cmd *cobra.Command, args []string) error {
			session, api, err := ctx.resolve(cmd.Context())
			if err != nil {
				return printEnvelopeJSON(nil, "", err)
			}
			if isInteractiveRequested(ctx.Options) {
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
					value, err := ui.PromptString("Amount in cents", true)
					if err != nil {
						return err
					}
					fmt.Sscan(value, &input.AmountCents)
				}
				if input.Description == "" {
					value, err := ui.PromptString("Description (optional)", false)
					if err != nil {
						return err
					}
					input.Description = value
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
					confirmed, err := ui.Confirm("Execute this transfer?")
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
	cmd.Flags().StringVar(&input.IdempotencyKey, "idempotency-key", "", "idempotency key")
	cmd.Flags().StringVar(&input.ConfirmationToken, "confirmation-token", "", "confirmation token")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview only")
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
		Short: "Send money over ACH, RTP, FedNow, or wire",
		RunE: func(cmd *cobra.Command, args []string) error {
			session, api, err := ctx.resolve(cmd.Context())
			if err != nil {
				return printEnvelopeJSON(nil, "", err)
			}
			if rail == "" && isInteractiveRequested(ctx.Options) {
				rail, err = ui.PromptSelect("Rail", []ui.Option{
					{Label: "ACH", Value: "ach"},
					{Label: "RTP", Value: "rtp"},
					{Label: "FedNow", Value: "fednow"},
					{Label: "Wire", Value: "wire"},
				})
				if err != nil {
					return err
				}
			}
			switch rail {
			case "ach":
				achInput.DryRun = &dryRun
				return runExternalACH(cmd, ctx, *session, api, achInput)
			case "rtp":
				rtpInput.DryRun = &dryRun
				return runExternalRTP(cmd, ctx, *session, api, rtpInput)
			case "fednow":
				fednowInput.DryRun = &dryRun
				return runExternalFedNow(cmd, ctx, *session, api, fednowInput)
			case "wire":
				wireInput.DryRun = &dryRun
				return runExternalWire(cmd, ctx, *session, api, wireInput)
			default:
				return fmt.Errorf("unsupported rail %q", rail)
			}
		},
	}
	cmd.Flags().StringVar(&rail, "rail", "", "rail: ach, rtp, fednow, wire")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview only")

	cmd.Flags().StringVar(&achInput.AccountID, "account-id", "", "ACH source account id")
	cmd.Flags().Int64Var(&achInput.AmountCents, "amount-cents", 0, "amount in minor units")
	cmd.Flags().StringVar(&achInput.StatementDescriptor, "statement-descriptor", "", "statement descriptor")
	cmd.Flags().StringVar(&achInput.AccountNumber, "account-number", "", "destination account number")
	cmd.Flags().StringVar(&achInput.RoutingNumber, "routing-number", "", "routing number")
	cmd.Flags().StringVar(&achInput.ExternalAccountID, "external-account-id", "", "external account id")
	cmd.Flags().StringVar(&achInput.IndividualName, "individual-name", "", "recipient individual name")
	cmd.Flags().StringVar(&achInput.CompanyName, "company-name", "", "company name")
	cmd.Flags().StringVar(&achInput.IdempotencyKey, "idempotency-key", "", "idempotency key")
	cmd.Flags().StringVar(&achInput.ConfirmationToken, "confirmation-token", "", "confirmation token")

	cmd.Flags().StringVar(&rtpInput.CreditorName, "creditor-name", "", "creditor name")
	cmd.Flags().StringVar(&rtpInput.RemittanceInformation, "remittance-information", "", "remittance information")
	cmd.Flags().StringVar(&rtpInput.SourceAccountNumberID, "source-account-number-id", "", "source account number id")
	cmd.Flags().StringVar(&rtpInput.DestinationAccountNumber, "destination-account-number", "", "destination account number")
	cmd.Flags().StringVar(&rtpInput.DestinationRoutingNumber, "destination-routing-number", "", "destination routing number")
	cmd.Flags().StringVar(&rtpInput.ExternalAccountID, "rtp-external-account-id", "", "external account id")
	cmd.Flags().Int64Var(&rtpInput.AmountCents, "rtp-amount-cents", 0, "RTP amount in minor units")
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
	cmd.Flags().StringVar(&fednowInput.IdempotencyKey, "fednow-idempotency-key", "", "idempotency key")
	cmd.Flags().StringVar(&fednowInput.ConfirmationToken, "fednow-confirmation-token", "", "confirmation token")

	cmd.Flags().StringVar(&wireInput.AccountID, "wire-account-id", "", "wire source account id")
	cmd.Flags().Int64Var(&wireInput.AmountCents, "wire-amount-cents", 0, "wire amount in minor units")
	cmd.Flags().StringVar(&wireInput.BeneficiaryName, "wire-beneficiary-name", "", "beneficiary name")
	cmd.Flags().StringVar(&wireInput.MessageToRecipient, "wire-message-to-recipient", "", "message to recipient")
	cmd.Flags().StringVar(&wireInput.AccountNumber, "wire-account-number", "", "destination account number")
	cmd.Flags().StringVar(&wireInput.RoutingNumber, "wire-routing-number", "", "routing number")
	cmd.Flags().StringVar(&wireInput.ExternalAccountID, "wire-external-account-id", "", "external account id")
	cmd.Flags().StringVar(&wireInput.IdempotencyKey, "wire-idempotency-key", "", "idempotency key")
	cmd.Flags().StringVar(&wireInput.ConfirmationToken, "wire-confirmation-token", "", "confirmation token")

	return cmd
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
			confirmed, err := ui.Confirm("Execute this RTP transfer?")
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
