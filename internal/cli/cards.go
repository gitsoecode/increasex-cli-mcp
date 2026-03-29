package cli

import (
	"github.com/jessevaughan/increasex/internal/app"
	"github.com/jessevaughan/increasex/internal/ui"
	"github.com/spf13/cobra"
)

func newCardsCmd(ctx *Context) *cobra.Command {
	var accountID, status, cursor string
	var limit int64
	cmd := &cobra.Command{
		Use:   "cards",
		Short: "List cards",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, api, err := ctx.resolve(cmd.Context())
			if err != nil {
				return printEnvelopeJSON(nil, "", err)
			}
			cards, requestID, err := ctx.Services.ListCards(cmd.Context(), api, accountID, status, cursor, limit)
			if ctx.Options.JSON {
				return printEnvelopeJSON(map[string]any{"cards": cards}, requestID, err)
			}
			if err != nil {
				return err
			}
			printCards(cards)
			if isInteractiveRequested(ctx.Options) && len(cards) > 0 {
				cardID, err := ui.PromptSelect("Card actions", []ui.Option{
					{Label: "Retrieve card details", Value: "get"},
					{Label: "Create a new card", Value: "create"},
				})
				if err != nil {
					return err
				}
				if cardID == "get" {
					selected, err := ui.PromptSelect("Select a card", buildCardOptions(cards))
					if err != nil {
						return err
					}
					card, _, err := ctx.Services.RetrieveCardDetails(cmd.Context(), api, selected)
					if err != nil {
						return err
					}
					return printJSON(card)
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&accountID, "account-id", "", "account id")
	cmd.Flags().StringVar(&status, "status", "", "status filter")
	cmd.Flags().StringVar(&cursor, "cursor", "", "page cursor")
	cmd.Flags().Int64Var(&limit, "limit", 20, "maximum cards to return")
	cmd.AddCommand(newCardsCreateCmd(ctx), newCardsGetCmd(ctx))
	return cmd
}

func buildCardOptions(cards []app.CardSummary) []ui.Option {
	options := make([]ui.Option, 0, len(cards))
	for _, card := range cards {
		options = append(options, ui.Option{
			Label: card.ID + " " + card.Last4,
			Value: card.ID,
		})
	}
	return options
}

func newCardsGetCmd(ctx *Context) *cobra.Command {
	var cardID string
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Retrieve masked card details",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, api, err := ctx.resolve(cmd.Context())
			if err != nil {
				return printEnvelopeJSON(nil, "", err)
			}
			card, requestID, err := ctx.Services.RetrieveCardDetails(cmd.Context(), api, cardID)
			if ctx.Options.JSON {
				return printEnvelopeJSON(card, requestID, err)
			}
			if err != nil {
				return err
			}
			return printJSON(card)
		},
	}
	cmd.Flags().StringVar(&cardID, "card-id", "", "card id")
	return cmd
}

func newCardsCreateCmd(ctx *Context) *cobra.Command {
	var input app.CreateCardInput
	var dryRun bool
	var billingCity, billingLine1, billingLine2, billingPostalCode, billingState string
	var walletEmail, walletPhone, walletProfileID string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Preview or create a card",
		RunE: func(cmd *cobra.Command, args []string) error {
			session, api, err := ctx.resolve(cmd.Context())
			if err != nil {
				return printEnvelopeJSON(nil, "", err)
			}
			if billingLine1 != "" || billingCity != "" || billingPostalCode != "" || billingState != "" {
				input.BillingAddress = &app.BillingAddressInput{
					City:       billingCity,
					Line1:      billingLine1,
					Line2:      billingLine2,
					PostalCode: billingPostalCode,
					State:      billingState,
				}
			}
			if walletEmail != "" || walletPhone != "" || walletProfileID != "" {
				input.DigitalWallet = &app.DigitalWalletInput{
					DigitalCardProfileID: walletProfileID,
					Email:                walletEmail,
					Phone:                walletPhone,
				}
			}
			input.DryRun = &dryRun
			if dryRun {
				preview, err := ctx.Services.PreviewCreateCard(*session, input)
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
				preview, err := ctx.Services.PreviewCreateCard(*session, input)
				if err != nil {
					return err
				}
				if !ctx.Options.Yes {
					printPreview(preview)
					confirmed, err := ui.Confirm("Create this card?")
					if err != nil || !confirmed {
						return err
					}
				}
				input.ConfirmationToken = preview.ConfirmationToken
			}
			data, requestID, err := ctx.Services.ExecuteCreateCard(cmd.Context(), api, *session, input)
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
	cmd.Flags().StringVar(&input.Description, "description", "", "card description")
	cmd.Flags().StringVar(&input.CardProgram, "card-program", "", "card program")
	cmd.Flags().StringVar(&input.EntityID, "entity-id", "", "entity id")
	cmd.Flags().StringVar(&input.IdempotencyKey, "idempotency-key", "", "idempotency key")
	cmd.Flags().StringVar(&input.ConfirmationToken, "confirmation-token", "", "confirmation token")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview only")
	cmd.Flags().StringVar(&billingCity, "billing-city", "", "billing city")
	cmd.Flags().StringVar(&billingLine1, "billing-line1", "", "billing address line 1")
	cmd.Flags().StringVar(&billingLine2, "billing-line2", "", "billing address line 2")
	cmd.Flags().StringVar(&billingPostalCode, "billing-postal-code", "", "billing postal code")
	cmd.Flags().StringVar(&billingState, "billing-state", "", "billing state")
	cmd.Flags().StringVar(&walletEmail, "wallet-email", "", "digital wallet email")
	cmd.Flags().StringVar(&walletPhone, "wallet-phone", "", "digital wallet phone")
	cmd.Flags().StringVar(&walletProfileID, "wallet-profile-id", "", "digital card profile id")
	return cmd
}
