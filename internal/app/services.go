package app

import (
	"context"
	"fmt"
	"sort"
	"strings"

	increase "github.com/Increase/increase-go"
	"github.com/jessevaughan/increasex/internal/auth"
	increasex "github.com/jessevaughan/increasex/internal/increase"
	"github.com/jessevaughan/increasex/internal/util"
)

type Services struct {
	auth    auth.Service
	confirm ConfirmationService
}

func NewServices() Services {
	return Services{
		auth:    auth.NewService(),
		confirm: NewConfirmationService(),
	}
}

func (s Services) ResolveSession(ctx context.Context, input auth.ResolveInput) (*Session, *increasex.Client, error) {
	resolved, err := s.auth.Resolve(input)
	if err != nil {
		return nil, nil, err
	}
	session := &Session{
		ProfileName: resolved.ProfileName,
		Environment: resolved.Environment,
		TokenSource: resolved.TokenSource,
	}
	return session, increasex.NewClient(resolved.Token, resolved.Environment), nil
}

func (s Services) Login(input auth.LoginInput) (auth.LoginResult, error) {
	return s.auth.SaveLogin(input)
}

func (s Services) Logout(profileName string) error {
	return s.auth.Logout(profileName)
}

func (s Services) AuthStatus(profileName string) (auth.StatusResult, error) {
	return s.auth.Status(profileName)
}

func (s Services) Export(input auth.ResolveInput) (map[string]string, error) {
	return s.auth.Export(input)
}

func (s Services) WhoAmI(ctx context.Context, api *increasex.Client, session Session) (map[string]any, string, error) {
	result, err := api.ListAccounts(ctx, increase.AccountListParams{Limit: increase.Int(1)})
	if err != nil {
		return nil, "", err
	}
	identity := ""
	if len(result.Data) > 0 {
		identity = result.Data[0].EntityID
	}
	return map[string]any{
		"active_profile": session.ProfileName,
		"environment":    session.Environment,
		"token_source":   session.TokenSource,
		"identity":       identity,
		"mcp_ready":      true,
	}, result.RequestID, nil
}

func (s Services) ListAccounts(ctx context.Context, api *increasex.Client, status string, limit int64, cursor string) ([]AccountSummary, string, error) {
	params := increase.AccountListParams{}
	if status != "" {
		params.Status = increase.F(increase.AccountListParamsStatus{
			In: increase.F([]increase.AccountListParamsStatusIn{increase.AccountListParamsStatusIn(status)}),
		})
	}
	if limit > 0 {
		params.Limit = increase.Int(limit)
	}
	if cursor != "" {
		params.Cursor = increase.String(cursor)
	}
	result, err := api.ListAccounts(ctx, params)
	if err != nil {
		return nil, "", err
	}
	accounts := make([]AccountSummary, 0, len(result.Data))
	for _, account := range result.Data {
		accounts = append(accounts, normalizeAccount(account))
	}
	return accounts, result.RequestID, nil
}

func (s Services) ResolveAccount(ctx context.Context, api *increasex.Client, query string, limit int64) ([]map[string]any, string, error) {
	if limit <= 0 {
		limit = 10
	}
	accounts, requestID, err := s.ListAccounts(ctx, api, "", 100, "")
	if err != nil {
		return nil, "", err
	}
	type candidate struct {
		AccountSummary
		Score int
	}
	queryLower := strings.ToLower(strings.TrimSpace(query))
	candidates := []candidate{}
	for _, account := range accounts {
		score := accountScore(account, queryLower)
		if score <= 0 {
			continue
		}
		candidates = append(candidates, candidate{AccountSummary: account, Score: score})
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Score == candidates[j].Score {
			return candidates[i].Name < candidates[j].Name
		}
		return candidates[i].Score > candidates[j].Score
	})
	out := []map[string]any{}
	for i, match := range candidates {
		if int64(i) >= limit {
			break
		}
		out = append(out, map[string]any{
			"id":        match.ID,
			"name":      match.Name,
			"status":    match.Status,
			"entity_id": match.EntityID,
			"score":     match.Score,
		})
	}
	return out, requestID, nil
}

func accountScore(account AccountSummary, query string) int {
	if query == "" {
		return 1
	}
	name := strings.ToLower(account.Name)
	switch {
	case account.ID == query:
		return 100
	case name == query:
		return 90
	case strings.Contains(name, query):
		return 70
	case strings.Contains(strings.ToLower(account.EntityID), query):
		return 50
	case strings.Contains(strings.ToLower(account.InformationalEntityID), query):
		return 40
	default:
		return 0
	}
}

func (s Services) GetBalance(ctx context.Context, api *increasex.Client, accountID string) (*BalanceSummary, string, error) {
	result, err := api.GetBalance(ctx, accountID)
	if err != nil {
		return nil, "", err
	}
	return &BalanceSummary{
		AccountID:        result.Data.AccountID,
		CurrentBalance:   result.Data.CurrentBalance,
		AvailableBalance: result.Data.AvailableBalance,
	}, result.RequestID, nil
}

func (s Services) ListRecentTransactions(ctx context.Context, api *increasex.Client, accountID, since, cursor string, limit int64, categories []string) ([]TransactionSummary, string, error) {
	params := increase.TransactionListParams{}
	if accountID != "" {
		params.AccountID = increase.String(accountID)
	}
	if limit > 0 {
		params.Limit = increase.Int(limit)
	}
	if cursor != "" {
		params.Cursor = increase.String(cursor)
	}
	if since != "" {
		parsed, err := increasex.ParseSince(since)
		if err != nil {
			return nil, "", util.NewError(util.CodeValidationError, "since must be RFC3339", nil, false)
		}
		params.CreatedAt = increase.F(increase.TransactionListParamsCreatedAt{
			OnOrAfter: increase.F(parsed),
		})
	}
	if len(categories) > 0 {
		values := make([]increase.TransactionListParamsCategoryIn, 0, len(categories))
		for _, category := range categories {
			values = append(values, increase.TransactionListParamsCategoryIn(category))
		}
		params.Category = increase.F(increase.TransactionListParamsCategory{
			In: increase.F(values),
		})
	}
	result, err := api.ListTransactions(ctx, params)
	if err != nil {
		return nil, "", err
	}
	transactions := make([]TransactionSummary, 0, len(result.Data))
	for _, txn := range result.Data {
		transactions = append(transactions, normalizeTransaction(txn))
	}
	return transactions, result.RequestID, nil
}

func (s Services) ListCards(ctx context.Context, api *increasex.Client, accountID, status, cursor string, limit int64) ([]CardSummary, string, error) {
	params := increase.CardListParams{}
	if accountID != "" {
		params.AccountID = increase.String(accountID)
	}
	if cursor != "" {
		params.Cursor = increase.String(cursor)
	}
	if limit > 0 {
		params.Limit = increase.Int(limit)
	}
	result, err := api.ListCards(ctx, params)
	if err != nil {
		return nil, "", err
	}
	cards := make([]CardSummary, 0, len(result.Data))
	for _, card := range result.Data {
		if status != "" && string(card.Status) != status {
			continue
		}
		cards = append(cards, normalizeCard(card))
	}
	return cards, result.RequestID, nil
}

func (s Services) RetrieveCardDetails(ctx context.Context, api *increasex.Client, cardID string) (*CardSummary, string, error) {
	result, err := api.GetCard(ctx, cardID)
	if err != nil {
		return nil, "", err
	}
	card := normalizeCard(*result.Data)
	card.BillingDetails = map[string]any{
		"line1":       maskLine(result.Data.BillingAddress.Line1),
		"line2":       maskLine(result.Data.BillingAddress.Line2),
		"city":        result.Data.BillingAddress.City,
		"state":       result.Data.BillingAddress.State,
		"postal_code": result.Data.BillingAddress.PostalCode,
	}
	return &card, result.RequestID, nil
}

func maskLine(value string) string {
	if value == "" {
		return ""
	}
	if len(value) <= 4 {
		return "***"
	}
	return value[:2] + strings.Repeat("*", len(value)-2)
}

func effectiveConfirmationPayload(input any) map[string]any {
	effective := CloneMap(input)
	delete(effective, "dry_run")
	delete(effective, "confirmation_token")
	return effective
}

func (s Services) PreviewCreateAccount(session Session, input CreateAccountInput) (*PreviewResult, error) {
	effective := effectiveConfirmationPayload(input)
	token, err := s.confirm.Generate("create_account", session, effective)
	if err != nil {
		return nil, err
	}
	return &PreviewResult{
		Mode:              "preview",
		Summary:           fmt.Sprintf("Create account %q in %s", input.Name, session.Environment),
		ConfirmationToken: token,
		Details:           effective,
	}, nil
}

func (s Services) ExecuteCreateAccount(ctx context.Context, api *increasex.Client, session Session, input CreateAccountInput) (any, string, error) {
	if err := s.confirm.Verify(input.ConfirmationToken, "create_account", session, effectiveConfirmationPayload(input)); err != nil {
		return nil, "", err
	}
	params := increase.AccountNewParams{Name: increase.String(input.Name)}
	if input.EntityID != "" {
		params.EntityID = increase.String(input.EntityID)
	}
	if input.InformationalEntityID != "" {
		params.InformationalEntityID = increase.String(input.InformationalEntityID)
	}
	if input.ProgramID != "" {
		params.ProgramID = increase.String(input.ProgramID)
	}
	result, err := api.CreateAccount(ctx, params, input.IdempotencyKey)
	if err != nil {
		return nil, "", err
	}
	return map[string]any{"mode": "executed", "account": normalizeAccount(*result.Data)}, result.RequestID, nil
}

func (s Services) PreviewCloseAccount(session Session, input CloseAccountInput) (*PreviewResult, error) {
	effective := effectiveConfirmationPayload(input)
	token, err := s.confirm.Generate("close_account", session, effective)
	if err != nil {
		return nil, err
	}
	return &PreviewResult{
		Mode:              "preview",
		Summary:           fmt.Sprintf("Close account %s", input.AccountID),
		ConfirmationToken: token,
		Details:           effective,
	}, nil
}

func (s Services) ExecuteCloseAccount(ctx context.Context, api *increasex.Client, session Session, input CloseAccountInput) (any, string, error) {
	if err := s.confirm.Verify(input.ConfirmationToken, "close_account", session, effectiveConfirmationPayload(input)); err != nil {
		return nil, "", err
	}
	result, err := api.CloseAccount(ctx, input.AccountID, input.IdempotencyKey)
	if err != nil {
		return nil, "", err
	}
	return map[string]any{
		"mode":       "executed",
		"account_id": result.Data.ID,
		"status":     string(result.Data.Status),
		"closed_at":  util.RFC3339OrEmpty(result.Data.ClosedAt),
	}, result.RequestID, nil
}

func (s Services) PreviewCreateAccountNumber(session Session, input CreateAccountNumberInput) (*PreviewResult, error) {
	effective := effectiveConfirmationPayload(input)
	token, err := s.confirm.Generate("create_account_number", session, effective)
	if err != nil {
		return nil, err
	}
	return &PreviewResult{
		Mode:              "preview",
		Summary:           fmt.Sprintf("Create account number %q for %s", input.Name, input.AccountID),
		ConfirmationToken: token,
		Details:           effective,
	}, nil
}

func (s Services) ExecuteCreateAccountNumber(ctx context.Context, api *increasex.Client, session Session, input CreateAccountNumberInput) (any, string, error) {
	if err := s.confirm.Verify(input.ConfirmationToken, "create_account_number", session, effectiveConfirmationPayload(input)); err != nil {
		return nil, "", err
	}
	params := increase.AccountNumberNewParams{
		AccountID: increase.String(input.AccountID),
		Name:      increase.String(input.Name),
	}
	if input.InboundACH != nil && input.InboundACH.DebitStatus != "" {
		params.InboundACH = increase.F(increase.AccountNumberNewParamsInboundACH{
			DebitStatus: increase.F(increase.AccountNumberNewParamsInboundACHDebitStatus(input.InboundACH.DebitStatus)),
		})
	}
	if input.InboundChecks != nil && input.InboundChecks.Status != "" {
		params.InboundChecks = increase.F(increase.AccountNumberNewParamsInboundChecks{
			Status: increase.F(increase.AccountNumberNewParamsInboundChecksStatus(input.InboundChecks.Status)),
		})
	}
	result, err := api.CreateAccountNumber(ctx, params, input.IdempotencyKey)
	if err != nil {
		return nil, "", err
	}
	return map[string]any{
		"mode":                 "executed",
		"id":                   result.Data.ID,
		"account_id":           result.Data.AccountID,
		"routing_number":       result.Data.RoutingNumber,
		"account_number_last4": util.MaskAccountNumber(result.Data.AccountNumber),
		"created_at":           util.RFC3339OrEmpty(result.Data.CreatedAt),
	}, result.RequestID, nil
}

func (s Services) PreviewInternalTransfer(session Session, input MoveMoneyInternalInput) (*PreviewResult, error) {
	effective := effectiveConfirmationPayload(input)
	if input.Description == "" {
		effective["description"] = generatedInternalDescription(input)
	}
	token, err := s.confirm.Generate("move_money_internal", session, effective)
	if err != nil {
		return nil, err
	}
	return &PreviewResult{
		Mode:              "preview",
		Summary:           fmt.Sprintf("Transfer %s from %s to %s", util.FormatUSDMinor(input.AmountCents), input.FromAccountID, input.ToAccountID),
		ConfirmationToken: token,
		Details:           effective,
	}, nil
}

func generatedInternalDescription(input MoveMoneyInternalInput) string {
	return fmt.Sprintf("Transfer %s from %s to %s", util.FormatUSDMinor(input.AmountCents), input.FromAccountID, input.ToAccountID)
}

func (s Services) ExecuteInternalTransfer(ctx context.Context, api *increasex.Client, session Session, input MoveMoneyInternalInput) (any, string, error) {
	effective := effectiveConfirmationPayload(input)
	description := input.Description
	if description == "" {
		description = generatedInternalDescription(input)
		effective["description"] = description
	}
	if err := s.confirm.Verify(input.ConfirmationToken, "move_money_internal", session, effective); err != nil {
		return nil, "", err
	}
	params := increase.AccountTransferNewParams{
		AccountID:            increase.String(input.FromAccountID),
		DestinationAccountID: increase.String(input.ToAccountID),
		Amount:               increase.Int(input.AmountCents),
		Description:          increase.String(description),
	}
	if input.RequireApproval != nil {
		params.RequireApproval = increase.Bool(*input.RequireApproval)
	}
	result, err := api.CreateInternalTransfer(ctx, params, input.IdempotencyKey)
	if err != nil {
		return nil, "", err
	}
	return map[string]any{
		"mode":            "executed",
		"transfer_id":     result.Data.ID,
		"status":          string(result.Data.Status),
		"from_account_id": result.Data.AccountID,
		"to_account_id":   result.Data.DestinationAccountID,
		"amount_cents":    result.Data.Amount,
		"created_at":      util.RFC3339OrEmpty(result.Data.CreatedAt),
	}, result.RequestID, nil
}

func (s Services) PreviewCreateCard(session Session, input CreateCardInput) (*PreviewResult, error) {
	effective := effectiveConfirmationPayload(input)
	token, err := s.confirm.Generate("create_card", session, effective)
	if err != nil {
		return nil, err
	}
	return &PreviewResult{
		Mode:              "preview",
		Summary:           fmt.Sprintf("Create card for account %s", input.AccountID),
		ConfirmationToken: token,
		Details:           effective,
	}, nil
}

func (s Services) ExecuteCreateCard(ctx context.Context, api *increasex.Client, session Session, input CreateCardInput) (any, string, error) {
	if err := s.confirm.Verify(input.ConfirmationToken, "create_card", session, effectiveConfirmationPayload(input)); err != nil {
		return nil, "", err
	}
	var result increasex.APIResult[*increase.Card]
	var err error
	if input.CardProgram != "" {
		body := map[string]any{"account_id": input.AccountID}
		if input.Description != "" {
			body["description"] = input.Description
		}
		if input.EntityID != "" {
			body["entity_id"] = input.EntityID
		}
		body["card_program"] = input.CardProgram
		if input.BillingAddress != nil {
			body["billing_address"] = CloneMap(input.BillingAddress)
		}
		if input.DigitalWallet != nil {
			body["digital_wallet"] = CloneMap(input.DigitalWallet)
		}
		result, err = api.CreateCardRaw(ctx, body, input.IdempotencyKey)
	} else {
		params := increase.CardNewParams{AccountID: increase.String(input.AccountID)}
		if input.Description != "" {
			params.Description = increase.String(input.Description)
		}
		if input.EntityID != "" {
			params.EntityID = increase.String(input.EntityID)
		}
		if input.BillingAddress != nil {
			params.BillingAddress = increase.F(increase.CardNewParamsBillingAddress{
				City:       increase.String(input.BillingAddress.City),
				Line1:      increase.String(input.BillingAddress.Line1),
				PostalCode: increase.String(input.BillingAddress.PostalCode),
				State:      increase.String(input.BillingAddress.State),
				Line2:      increase.String(input.BillingAddress.Line2),
			})
		}
		if input.DigitalWallet != nil {
			params.DigitalWallet = increase.F(increase.CardNewParamsDigitalWallet{
				DigitalCardProfileID: increase.String(input.DigitalWallet.DigitalCardProfileID),
				Email:                increase.String(input.DigitalWallet.Email),
				Phone:                increase.String(input.DigitalWallet.Phone),
			})
		}
		result, err = api.CreateCard(ctx, params, input.IdempotencyKey)
	}
	if err != nil {
		return nil, "", err
	}
	return map[string]any{"mode": "executed", "card": normalizeCard(*result.Data)}, result.RequestID, nil
}

func (s Services) PreviewExternalACH(session Session, input ACHTransferInput) (*PreviewResult, error) {
	effective := effectiveConfirmationPayload(input)
	token, err := s.confirm.Generate("move_money_external_ach", session, effective)
	if err != nil {
		return nil, err
	}
	return &PreviewResult{
		Mode:              "preview",
		Summary:           fmt.Sprintf("Send ACH transfer %s", util.FormatUSDMinor(input.AmountCents)),
		ConfirmationToken: token,
		Details:           effective,
	}, nil
}

func (s Services) ExecuteExternalACH(ctx context.Context, api *increasex.Client, session Session, input ACHTransferInput) (any, string, error) {
	if err := s.confirm.Verify(input.ConfirmationToken, "move_money_external_ach", session, effectiveConfirmationPayload(input)); err != nil {
		return nil, "", err
	}
	if input.ExternalAccountID == "" && (input.AccountNumber == "" || input.RoutingNumber == "") {
		return nil, "", util.NewError(util.CodeValidationError, "provide external_account_id or account_number and routing_number", nil, false)
	}
	params := increase.ACHTransferNewParams{
		AccountID:           increase.String(input.AccountID),
		Amount:              increase.Int(input.AmountCents),
		StatementDescriptor: increase.String(input.StatementDescriptor),
	}
	if input.AccountNumber != "" {
		params.AccountNumber = increase.String(input.AccountNumber)
	}
	if input.RoutingNumber != "" {
		params.RoutingNumber = increase.String(input.RoutingNumber)
	}
	if input.ExternalAccountID != "" {
		params.ExternalAccountID = increase.String(input.ExternalAccountID)
	}
	if input.Funding != "" {
		params.Funding = increase.F(increase.ACHTransferNewParamsFunding(input.Funding))
	}
	if input.DestinationAccountHolder != "" {
		params.DestinationAccountHolder = increase.F(increase.ACHTransferNewParamsDestinationAccountHolder(input.DestinationAccountHolder))
	}
	if input.IndividualID != "" {
		params.IndividualID = increase.String(input.IndividualID)
	}
	if input.IndividualName != "" {
		params.IndividualName = increase.String(input.IndividualName)
	}
	if input.CompanyName != "" {
		params.CompanyName = increase.String(input.CompanyName)
	}
	if input.CompanyEntryDescription != "" {
		params.CompanyEntryDescription = increase.String(input.CompanyEntryDescription)
	}
	if input.CompanyDescriptiveDate != "" {
		params.CompanyDescriptiveDate = increase.String(input.CompanyDescriptiveDate)
	}
	if input.CompanyDiscretionaryData != "" {
		params.CompanyDiscretionaryData = increase.String(input.CompanyDiscretionaryData)
	}
	if input.RequireApproval != nil {
		params.RequireApproval = increase.Bool(*input.RequireApproval)
	}
	result, err := api.CreateACHTransfer(ctx, params, input.IdempotencyKey)
	if err != nil {
		return nil, "", err
	}
	return map[string]any{
		"mode":        "executed",
		"rail":        "ach",
		"transfer_id": result.Data.ID,
		"status":      string(result.Data.Status),
		"created_at":  util.RFC3339OrEmpty(result.Data.CreatedAt),
	}, result.RequestID, nil
}

func (s Services) PreviewExternalRTP(session Session, input RTPTransferInput) (*PreviewResult, error) {
	effective := effectiveConfirmationPayload(input)
	token, err := s.confirm.Generate("move_money_external_rtp", session, effective)
	if err != nil {
		return nil, err
	}
	return &PreviewResult{
		Mode:              "preview",
		Summary:           fmt.Sprintf("Send RTP transfer %s", util.FormatUSDMinor(input.AmountCents)),
		ConfirmationToken: token,
		Details:           effective,
	}, nil
}

func (s Services) ExecuteExternalRTP(ctx context.Context, api *increasex.Client, session Session, input RTPTransferInput) (any, string, error) {
	if err := s.confirm.Verify(input.ConfirmationToken, "move_money_external_rtp", session, effectiveConfirmationPayload(input)); err != nil {
		return nil, "", err
	}
	params := increase.RealTimePaymentsTransferNewParams{
		Amount:                            increase.Int(input.AmountCents),
		CreditorName:                      increase.String(input.CreditorName),
		UnstructuredRemittanceInformation: increase.String(input.RemittanceInformation),
		SourceAccountNumberID:             increase.String(input.SourceAccountNumberID),
	}
	if input.DebtorName != "" {
		params.DebtorName = increase.String(input.DebtorName)
	}
	if input.DestinationAccountNumber != "" {
		params.AccountNumber = increase.String(input.DestinationAccountNumber)
	}
	if input.DestinationRoutingNumber != "" {
		params.RoutingNumber = increase.String(input.DestinationRoutingNumber)
	}
	if input.ExternalAccountID != "" {
		params.ExternalAccountID = increase.String(input.ExternalAccountID)
	}
	if input.UltimateCreditorName != "" {
		params.UltimateCreditorName = increase.String(input.UltimateCreditorName)
	}
	if input.UltimateDebtorName != "" {
		params.UltimateDebtorName = increase.String(input.UltimateDebtorName)
	}
	if input.RequireApproval != nil {
		params.RequireApproval = increase.Bool(*input.RequireApproval)
	}
	result, err := api.CreateRTPTransfer(ctx, params, input.IdempotencyKey)
	if err != nil {
		return nil, "", err
	}
	return map[string]any{
		"mode":        "executed",
		"rail":        "rtp",
		"transfer_id": result.Data.ID,
		"status":      string(result.Data.Status),
		"created_at":  util.RFC3339OrEmpty(result.Data.CreatedAt),
	}, result.RequestID, nil
}

func (s Services) PreviewExternalFedNow(session Session, input FedNowTransferInput) (*PreviewResult, error) {
	effective := effectiveConfirmationPayload(input)
	token, err := s.confirm.Generate("move_money_external_fednow", session, effective)
	if err != nil {
		return nil, err
	}
	return &PreviewResult{
		Mode:              "preview",
		Summary:           fmt.Sprintf("Send FedNow transfer %s", util.FormatUSDMinor(input.AmountCents)),
		ConfirmationToken: token,
		Details:           effective,
	}, nil
}

func (s Services) ExecuteExternalFedNow(ctx context.Context, api *increasex.Client, session Session, input FedNowTransferInput) (any, string, error) {
	if err := s.confirm.Verify(input.ConfirmationToken, "move_money_external_fednow", session, effectiveConfirmationPayload(input)); err != nil {
		return nil, "", err
	}
	if input.ExternalAccountID == "" && (input.AccountNumber == "" || input.RoutingNumber == "") {
		return nil, "", util.NewError(util.CodeValidationError, "provide external_account_id or account_number and routing_number", nil, false)
	}
	params := increase.FednowTransferNewParams{
		AccountNumber:                     increase.String(input.AccountNumber),
		Amount:                            increase.Int(input.AmountCents),
		CreditorName:                      increase.String(input.CreditorName),
		DebtorName:                        increase.String(input.DebtorName),
		ExternalAccountID:                 increase.String(input.ExternalAccountID),
		RoutingNumber:                     increase.String(input.RoutingNumber),
		SourceAccountNumberID:             increase.String(input.SourceAccountNumberID),
		UnstructuredRemittanceInformation: increase.String(input.UnstructuredRemittanceInformation),
	}
	if input.CreditorAddress != nil {
		params.CreditorAddress = increase.F(increase.FednowTransferNewParamsCreditorAddress{
			Line1:      increase.String(input.CreditorAddress.Line1),
			City:       increase.String(input.CreditorAddress.City),
			State:      increase.String(input.CreditorAddress.State),
			PostalCode: increase.String(input.CreditorAddress.PostalCode),
		})
	}
	if input.DebtorAddress != nil {
		params.DebtorAddress = increase.F(increase.FednowTransferNewParamsDebtorAddress{
			Line1:      increase.String(input.DebtorAddress.Line1),
			City:       increase.String(input.DebtorAddress.City),
			State:      increase.String(input.DebtorAddress.State),
			PostalCode: increase.String(input.DebtorAddress.PostalCode),
		})
	}
	if input.RequireApproval != nil {
		params.RequireApproval = increase.Bool(*input.RequireApproval)
	}
	result, err := api.CreateFedNowTransfer(ctx, params, input.IdempotencyKey)
	if err != nil {
		return nil, "", err
	}
	return map[string]any{
		"mode":        "executed",
		"rail":        "fednow",
		"transfer_id": result.Data.ID,
		"status":      result.Data.Status,
		"created_at":  util.RFC3339OrEmpty(result.Data.CreatedAt),
	}, result.RequestID, nil
}

func (s Services) PreviewExternalWire(session Session, input WireTransferInput) (*PreviewResult, error) {
	effective := effectiveConfirmationPayload(input)
	token, err := s.confirm.Generate("move_money_external_wire", session, effective)
	if err != nil {
		return nil, err
	}
	return &PreviewResult{
		Mode:              "preview",
		Summary:           fmt.Sprintf("Send wire transfer %s", util.FormatUSDMinor(input.AmountCents)),
		ConfirmationToken: token,
		Details:           effective,
	}, nil
}

func (s Services) ExecuteExternalWire(ctx context.Context, api *increasex.Client, session Session, input WireTransferInput) (any, string, error) {
	if err := s.confirm.Verify(input.ConfirmationToken, "move_money_external_wire", session, effectiveConfirmationPayload(input)); err != nil {
		return nil, "", err
	}
	if input.ExternalAccountID == "" && (input.AccountNumber == "" || input.RoutingNumber == "") {
		return nil, "", util.NewError(util.CodeValidationError, "provide external_account_id or account_number and routing_number", nil, false)
	}
	params := increase.WireTransferNewParams{
		AccountID: increase.String(input.AccountID),
		Amount:    increase.Int(input.AmountCents),
		Creditor: increase.F(increase.WireTransferNewParamsCreditor{
			Name: increase.String(input.BeneficiaryName),
		}),
		Remittance: increase.F(increase.WireTransferNewParamsRemittance{
			Category: increase.F(increase.WireTransferNewParamsRemittanceCategoryUnstructured),
			Unstructured: increase.F(increase.WireTransferNewParamsRemittanceUnstructured{
				Message: increase.String(input.MessageToRecipient),
			}),
		}),
	}
	if input.AccountNumber != "" {
		params.AccountNumber = increase.String(input.AccountNumber)
	}
	if input.RoutingNumber != "" {
		params.RoutingNumber = increase.String(input.RoutingNumber)
	}
	if input.ExternalAccountID != "" {
		params.ExternalAccountID = increase.String(input.ExternalAccountID)
	}
	if input.BeneficiaryAddressLine1 != "" {
		params.Creditor = increase.F(increase.WireTransferNewParamsCreditor{
			Name: increase.String(input.BeneficiaryName),
			Address: increase.F(increase.WireTransferNewParamsCreditorAddress{
				Unstructured: increase.F(increase.WireTransferNewParamsCreditorAddressUnstructured{
					Line1: increase.String(input.BeneficiaryAddressLine1),
					Line2: increase.String(input.BeneficiaryAddressLine2),
					Line3: increase.String(input.BeneficiaryAddressLine3),
				}),
			}),
		})
	}
	if input.OriginatorName != "" {
		params.Debtor = increase.F(increase.WireTransferNewParamsDebtor{
			Name: increase.String(input.OriginatorName),
			Address: increase.F(increase.WireTransferNewParamsDebtorAddress{
				Unstructured: increase.F(increase.WireTransferNewParamsDebtorAddressUnstructured{
					Line1: increase.String(input.OriginatorAddressLine1),
					Line2: increase.String(input.OriginatorAddressLine2),
					Line3: increase.String(input.OriginatorAddressLine3),
				}),
			}),
		})
	}
	if input.RequireApproval != nil {
		params.RequireApproval = increase.Bool(*input.RequireApproval)
	}
	result, err := api.CreateWireTransfer(ctx, params, input.IdempotencyKey)
	if err != nil {
		return nil, "", err
	}
	return map[string]any{
		"mode":        "executed",
		"rail":        "wire",
		"transfer_id": result.Data.ID,
		"status":      string(result.Data.Status),
		"created_at":  util.RFC3339OrEmpty(result.Data.CreatedAt),
	}, result.RequestID, nil
}

func normalizeAccount(account increase.Account) AccountSummary {
	return AccountSummary{
		ID:                    account.ID,
		Name:                  account.Name,
		Status:                string(account.Status),
		EntityID:              account.EntityID,
		InformationalEntityID: account.InformationalEntityID,
		ProgramID:             account.ProgramID,
		CreatedAt:             util.RFC3339OrEmpty(account.CreatedAt),
	}
}

func normalizeTransaction(txn increase.Transaction) TransactionSummary {
	direction := "credit"
	if txn.Amount < 0 {
		direction = "debit"
	}
	return TransactionSummary{
		ID:                  txn.ID,
		AccountID:           txn.AccountID,
		AmountCents:         txn.Amount,
		Direction:           direction,
		Description:         txn.Description,
		Type:                string(txn.Type),
		CreatedAt:           util.RFC3339OrEmpty(txn.CreatedAt),
		RouteID:             txn.RouteID,
		RouteType:           string(txn.RouteType),
		CounterpartySummary: txn.Description,
	}
}

func normalizeCard(card increase.Card) CardSummary {
	return CardSummary{
		ID:              card.ID,
		AccountID:       card.AccountID,
		Last4:           util.MaskLast4(card.Last4),
		Status:          string(card.Status),
		Description:     card.Description,
		EntityID:        card.EntityID,
		ExpirationMonth: card.ExpirationMonth,
		ExpirationYear:  card.ExpirationYear,
		CreatedAt:       util.RFC3339OrEmpty(card.CreatedAt),
	}
}
