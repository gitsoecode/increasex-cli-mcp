package cli

import (
	"github.com/gitsoecode/increasex-cli-mcp/internal/app"
	increasex "github.com/gitsoecode/increasex-cli-mcp/internal/increase"
	"github.com/gitsoecode/increasex-cli-mcp/internal/ui"
	"github.com/spf13/cobra"
)

var (
	promptCardString  = ui.PromptString
	promptCardBool    = promptBool
	selectCardAccount = chooseAccount
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
			}
			if len(cards) == 0 {
				actions = []ui.Option{
					{Label: "Create a new card", Value: "create"},
				}
			}
			for {
				action, err := promptSelectNavigation("Card actions", actions, navBack, navExit)
				if err != nil {
					return bubbleNavigation(cmd, err)
				}
				if action == "create" {
					return invokeCommand(cmd, newCardsCreateCmd(ctx))
				}
				if len(cards) == 0 {
					return nil
				}
				selected, err := chooseCard(cards, "Select a card")
				if err != nil {
					if isNavigateBack(err) {
						continue
					}
					return bubbleNavigation(cmd, err)
				}
				switch action {
				case "retrieve":
					return invokeCommand(cmd, newCardsRetrieveCmd(ctx), "--card-id", selected)
				case "details":
					return invokeCommand(cmd, newCardsDetailsCmd(ctx), "--card-id", selected)
				case "iframe":
					return invokeCommand(cmd, newCardsCreateDetailsIframeCmd(ctx), "--card-id", selected)
				case "update_pin":
					return invokeCommand(cmd, newCardsUpdatePINCmd(ctx), "--card-id", selected)
				}
			}
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
					return bubbleNavigation(cmd, err)
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
					return bubbleNavigation(cmd, err)
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
	var openInBrowser bool
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
				step := 0
				for step < 2 {
					switch step {
					case 0:
						input.CardID, err = chooseCard(cards, "Select a card")
						if err != nil {
							return bubbleNavigation(cmd, err)
						}
					case 1:
						if input.PhysicalCardID != "" {
							step++
							continue
						}
						input.PhysicalCardID, err = promptStringNavigation("Physical card id (optional)", false)
						if err != nil {
							if isNavigateBack(err) {
								step = 0
								input.CardID = ""
								continue
							}
							return bubbleNavigation(cmd, err)
						}
					}
					step++
				}
			}
			data, requestID, err := ctx.Services.CreateCardDetailsIframe(cmd.Context(), api, input)
			if ctx.Options.JSON {
				return printEnvelopeJSON(data, requestID, err)
			}
			if err != nil {
				return err
			}
			if err := printJSON(data); err != nil {
				return err
			}
			if openInBrowser && !ctx.Options.JSON {
				return openBrowserURL(data.IframeURL)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&input.CardID, "card-id", "", "card id")
	cmd.Flags().StringVar(&input.PhysicalCardID, "physical-card-id", "", "physical card id")
	cmd.Flags().BoolVar(&openInBrowser, "open", false, "open the iframe URL in your browser")
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
				step := 0
				var cards []app.CardSummary
				if input.CardID == "" {
					cards, _, err = ctx.Services.ListCards(cmd.Context(), api, "", "", "", 25)
					if err != nil {
						return err
					}
				}
				for step < 2 {
					switch step {
					case 0:
						if input.CardID != "" {
							step++
							continue
						}
						input.CardID, err = chooseCard(cards, "Select a card")
						if err != nil {
							return bubbleNavigation(cmd, err)
						}
					case 1:
						if input.PIN != "" {
							step++
							continue
						}
						input.PIN, err = promptStringNavigation("New PIN", true)
						if err != nil {
							if isNavigateBack(err) {
								step = 0
								input.CardID = ""
								continue
							}
							return bubbleNavigation(cmd, err)
						}
					}
					step++
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
					confirmed, err := promptConfirmationNavigation("Update this card PIN?")
					if err != nil || !confirmed {
						return bubbleNavigation(cmd, err)
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
			billingAddress := cardBillingAddressFromFlags(billingCity, billingLine1, billingLine2, billingPostalCode, billingState)
			digitalWallet := cardDigitalWalletFromFlags(walletEmail, walletPhone, walletProfileID)
			if isInteractiveRequested(ctx.Options) {
				if err := promptCreateCardInput(cmd, ctx, api, &input, &billingAddress, &digitalWallet); err != nil {
					return err
				}
			}
			input.BillingAddress = billingAddress
			input.DigitalWallet = digitalWallet
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
					confirmed, err := promptConfirmationNavigation("Create this card?")
					if err != nil || !confirmed {
						return bubbleNavigation(cmd, err)
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

func promptCreateCardInput(cmd *cobra.Command, ctx *Context, api *increasex.Client, input *app.CreateCardInput, billingAddress **app.BillingAddressInput, digitalWallet **app.DigitalWalletInput) error {
	accounts, _, err := ctx.Services.ListAccounts(cmd.Context(), api, "open", 25, "")
	if err != nil {
		return err
	}
	return promptCreateCardFields(accounts, input, billingAddress, digitalWallet)
}

func promptCreateCardFields(accounts []app.AccountSummary, input *app.CreateCardInput, billingAddress **app.BillingAddressInput, digitalWallet **app.DigitalWalletInput) error {
	var err error
	if input.AccountID == "" {
		input.AccountID, err = selectCardAccount(accounts, "Select source account")
		if err != nil {
			return err
		}
	}
	if input.Description == "" {
		input.Description, err = promptCardString("Description (optional)", false)
		if err != nil {
			return err
		}
	}
	if input.CardProgram == "" {
		input.CardProgram, err = promptCardString("Card program (optional)", false)
		if err != nil {
			return err
		}
	}
	if input.EntityID == "" {
		input.EntityID, err = promptCardString("Entity id (optional)", false)
		if err != nil {
			return err
		}
	}
	if err := promptCardBillingAddressInput(billingAddress); err != nil {
		return err
	}
	return promptCardDigitalWalletInput(digitalWallet)
}

func cardBillingAddressFromFlags(city, line1, line2, postalCode, state string) *app.BillingAddressInput {
	if line1 != "" || city != "" || postalCode != "" || state != "" || line2 != "" {
		return &app.BillingAddressInput{
			City:       city,
			Line1:      line1,
			Line2:      line2,
			PostalCode: postalCode,
			State:      state,
		}
	}
	return nil
}

func cardDigitalWalletFromFlags(email, phone, profileID string) *app.DigitalWalletInput {
	if email != "" || phone != "" || profileID != "" {
		return &app.DigitalWalletInput{
			DigitalCardProfileID: profileID,
			Email:                email,
			Phone:                phone,
		}
	}
	return nil
}

func promptCardBillingAddressInput(input **app.BillingAddressInput) error {
	if *input == nil {
		include, err := promptCardBool("Add billing address?", "Add billing address", "Skip billing address")
		if err != nil || !include {
			return err
		}
		*input = &app.BillingAddressInput{}
	}
	var err error
	if (*input).Line1 == "" {
		(*input).Line1, err = promptCardString("Billing address line 1", true)
		if err != nil {
			return err
		}
	}
	if (*input).Line2 == "" {
		(*input).Line2, err = promptCardString("Billing address line 2 (optional)", false)
		if err != nil {
			return err
		}
	}
	if (*input).City == "" {
		(*input).City, err = promptCardString("Billing city", true)
		if err != nil {
			return err
		}
	}
	if (*input).State == "" {
		(*input).State, err = promptCardString("Billing state", true)
		if err != nil {
			return err
		}
	}
	if (*input).PostalCode == "" {
		(*input).PostalCode, err = promptCardString("Billing postal code", true)
		if err != nil {
			return err
		}
	}
	return nil
}

func promptCardDigitalWalletInput(input **app.DigitalWalletInput) error {
	if *input == nil {
		include, err := promptCardBool("Add digital wallet details?", "Add digital wallet details", "Skip digital wallet details")
		if err != nil || !include {
			return err
		}
		*input = &app.DigitalWalletInput{}
	}
	var err error
	if (*input).DigitalCardProfileID == "" {
		(*input).DigitalCardProfileID, err = promptCardString("Digital card profile id (optional)", false)
		if err != nil {
			return err
		}
	}
	if (*input).Email == "" {
		(*input).Email, err = promptCardString("Wallet email (optional)", false)
		if err != nil {
			return err
		}
	}
	if (*input).Phone == "" {
		(*input).Phone, err = promptCardString("Wallet phone (optional)", false)
		if err != nil {
			return err
		}
	}
	if (*input).DigitalCardProfileID == "" && (*input).Email == "" && (*input).Phone == "" {
		*input = nil
	}
	return nil
}
