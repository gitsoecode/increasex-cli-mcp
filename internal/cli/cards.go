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
			if !isInteractiveRequested(ctx.Options) {
				return nil
			}
			actions := []ui.Option{
				{Label: "Retrieve masked details", Value: "retrieve"},
				{Label: "Retrieve card details", Value: "details"},
				{Label: "Create details iframe", Value: "iframe"},
				{Label: "Create a new card", Value: "create"},
				{Label: "Update PIN", Value: "update_pin"},
				{Label: "Back", Value: "back"},
			}
			if len(cards) == 0 {
				actions = []ui.Option{
					{Label: "Create a new card", Value: "create"},
					{Label: "Back", Value: "back"},
				}
			}
			action, err := ui.PromptSelect("Card actions", actions)
			if err != nil {
				return err
			}
			switch action {
			case "retrieve":
				selected, err := chooseCard(cards, "Select a card")
				if err != nil {
					return err
				}
				return invokeCommand(cmd, newCardsRetrieveCmd(ctx), "--card-id", selected)
			case "details":
				selected, err := chooseCard(cards, "Select a card")
				if err != nil {
					return err
				}
				return invokeCommand(cmd, newCardsDetailsCmd(ctx), "--card-id", selected)
			case "iframe":
				selected, err := chooseCard(cards, "Select a card")
				if err != nil {
					return err
				}
				return invokeCommand(cmd, newCardsCreateDetailsIframeCmd(ctx), "--card-id", selected)
			case "create":
				return invokeCommand(cmd, newCardsCreateCmd(ctx))
			case "update_pin":
				selected, err := chooseCard(cards, "Select a card")
				if err != nil {
					return err
				}
				return invokeCommand(cmd, newCardsUpdatePINCmd(ctx), "--card-id", selected)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&accountID, "account-id", "", "account id")
	cmd.Flags().StringVar(&status, "status", "", "status filter")
	cmd.Flags().StringVar(&cursor, "cursor", "", "page cursor")
	cmd.Flags().Int64Var(&limit, "limit", 20, "maximum cards to return")
	cmd.AddCommand(
		newCardsCreateCmd(ctx),
		newCardsRetrieveCmd(ctx),
		newCardsDetailsCmd(ctx),
		newCardsCreateDetailsIframeCmd(ctx),
		newCardsUpdatePINCmd(ctx),
	)
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

func newCardsRetrieveCmd(ctx *Context) *cobra.Command {
	var cardID string
	cmd := &cobra.Command{
		Use:     "retrieve",
		Short:   "Retrieve masked card details",
		Aliases: []string{"get"},
		RunE: func(cmd *cobra.Command, args []string) error {
			_, api, err := ctx.resolve(cmd.Context())
			if err != nil {
				return printEnvelopeJSON(nil, "", err)
			}
			if cardID == "" && isInteractiveRequested(ctx.Options) {
				cards, _, err := ctx.Services.ListCards(cmd.Context(), api, "", "", "", 25)
				if err != nil {
					return err
				}
				cardID, err = chooseCard(cards, "Select a card")
				if err != nil {
					return err
				}
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

func newCardsDetailsCmd(ctx *Context) *cobra.Command {
	var cardID string
	cmd := &cobra.Command{
		Use:   "details",
		Short: "Retrieve card details",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, api, err := ctx.resolve(cmd.Context())
			if err != nil {
				return printEnvelopeJSON(nil, "", err)
			}
			if cardID == "" && isInteractiveRequested(ctx.Options) {
				cards, _, err := ctx.Services.ListCards(cmd.Context(), api, "", "", "", 25)
				if err != nil {
					return err
				}
				cardID, err = chooseCard(cards, "Select a card")
				if err != nil {
					return err
				}
			}
			card, requestID, err := ctx.Services.RetrieveSensitiveCardDetails(cmd.Context(), api, cardID)
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

func newCardsCreateDetailsIframeCmd(ctx *Context) *cobra.Command {
	var input app.CreateCardDetailsIframeInput
	cmd := &cobra.Command{
		Use:   "create-details-iframe",
		Short: "Create a card details iframe",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, api, err := ctx.resolve(cmd.Context())
			if err != nil {
				return printEnvelopeJSON(nil, "", err)
			}
			if input.CardID == "" && isInteractiveRequested(ctx.Options) {
				cards, _, err := ctx.Services.ListCards(cmd.Context(), api, "", "", "", 25)
				if err != nil {
					return err
				}
				input.CardID, err = chooseCard(cards, "Select a card")
				if err != nil {
					return err
				}
			}
			if input.PhysicalCardID == "" && isInteractiveRequested(ctx.Options) {
				input.PhysicalCardID, err = ui.PromptString("Physical card id (optional)", false)
				if err != nil {
					return err
				}
			}
			data, requestID, err := ctx.Services.CreateCardDetailsIframe(cmd.Context(), api, input)
			if ctx.Options.JSON {
				return printEnvelopeJSON(data, requestID, err)
			}
			if err != nil {
				return err
			}
			return printJSON(data)
		},
	}
	cmd.Flags().StringVar(&input.CardID, "card-id", "", "card id")
	cmd.Flags().StringVar(&input.PhysicalCardID, "physical-card-id", "", "physical card id")
	return cmd
}

func newCardsUpdatePINCmd(ctx *Context) *cobra.Command {
	var input app.UpdateCardPINInput
	var dryRun bool
	cmd := &cobra.Command{
		Use:   "update-pin",
		Short: "Preview or update a card PIN",
		RunE: func(cmd *cobra.Command, args []string) error {
			session, api, err := ctx.resolve(cmd.Context())
			if err != nil {
				return printEnvelopeJSON(nil, "", err)
			}
			if isInteractiveRequested(ctx.Options) {
				if input.CardID == "" {
					cards, _, err := ctx.Services.ListCards(cmd.Context(), api, "", "", "", 25)
					if err != nil {
						return err
					}
					input.CardID, err = chooseCard(cards, "Select a card")
					if err != nil {
						return err
					}
				}
				if input.PIN == "" {
					input.PIN, err = ui.PromptString("New PIN", true)
					if err != nil {
						return err
					}
				}
			}
			input.DryRun = &dryRun
			if dryRun {
				preview, err := ctx.Services.PreviewUpdateCardPIN(*session, input)
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
				preview, err := ctx.Services.PreviewUpdateCardPIN(*session, input)
				if err != nil {
					return err
				}
				if !ctx.Options.Yes {
					printPreview(preview)
					confirmed, err := ui.Confirm("Update this card PIN?")
					if err != nil || !confirmed {
						return err
					}
				}
				input.ConfirmationToken = preview.ConfirmationToken
			}
			data, requestID, err := ctx.Services.ExecuteUpdateCardPIN(cmd.Context(), api, *session, input)
			if ctx.Options.JSON {
				return printEnvelopeJSON(data, requestID, err)
			}
			if err != nil {
				return err
			}
			return printJSON(data)
		},
	}
	cmd.Flags().StringVar(&input.CardID, "card-id", "", "card id")
	cmd.Flags().StringVar(&input.PIN, "pin", "", "new PIN")
	cmd.Flags().StringVar(&input.ConfirmationToken, "confirmation-token", "", "confirmation token")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview only")
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
