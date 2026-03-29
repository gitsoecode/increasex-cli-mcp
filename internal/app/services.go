package app

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	increase "github.com/Increase/increase-go"
	"github.com/gitsoecode/increasex-cli-mcp/internal/auth"
	increasex "github.com/gitsoecode/increasex-cli-mcp/internal/increase"
	"github.com/gitsoecode/increasex-cli-mcp/internal/util"
)

type Services struct {
	auth    auth.Service
	confirm ConfirmationService
	now     func() time.Time
}

func NewServices() Services {
	return Services{
		auth:    auth.NewService(),
		confirm: NewConfirmationService(),
		now:     time.Now,
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
		"entity_id":      identity,
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

func (s Services) ListAccountNumbers(ctx context.Context, api *increasex.Client, accountID, status string, limit int64, cursor string) ([]AccountNumberSummary, string, error) {
	params := increase.AccountNumberListParams{}
	if accountID != "" {
		params.AccountID = increase.String(accountID)
	}
	if status != "" {
		params.Status = increase.F(increase.AccountNumberListParamsStatus{
			In: increase.F([]increase.AccountNumberListParamsStatusIn{increase.AccountNumberListParamsStatusIn(status)}),
		})
	}
	if limit > 0 {
		params.Limit = increase.Int(limit)
	}
	if cursor != "" {
		params.Cursor = increase.String(cursor)
	}
	result, err := api.ListAccountNumbers(ctx, params)
	if err != nil {
		return nil, "", err
	}
	numbers := make([]AccountNumberSummary, 0, len(result.Data))
	for _, number := range result.Data {
		numbers = append(numbers, normalizeAccountNumber(number))
	}
	numbers = s.attachAccountNamesToAccountNumbers(ctx, api, numbers)
	return numbers, result.RequestID, nil
}

func (s Services) RetrieveAccountNumber(ctx context.Context, api *increasex.Client, accountNumberID string) (*AccountNumberDetails, string, error) {
	result, err := api.GetAccountNumber(ctx, accountNumberID)
	if err != nil {
		return nil, "", err
	}
	details := normalizeAccountNumberDetails(*result.Data)
	details = s.attachAccountNameToAccountNumberDetails(ctx, api, details)
	return &details, result.RequestID, nil
}

func (s Services) RetrieveSensitiveAccountNumberDetails(ctx context.Context, api *increasex.Client, accountNumberID string) (*AccountNumberDetails, string, error) {
	result, err := api.GetAccountNumber(ctx, accountNumberID)
	if err != nil {
		return nil, "", err
	}
	details := normalizeSensitiveAccountNumberDetails(*result.Data)
	details = s.attachAccountNameToAccountNumberDetails(ctx, api, details)
	return &details, result.RequestID, nil
}

func (s Services) attachAccountNamesToAccountNumbers(ctx context.Context, api *increasex.Client, numbers []AccountNumberSummary) []AccountNumberSummary {
	accountNames := s.resolveAccountNames(ctx, api, collectAccountNumberAccountIDs(numbers))
	if len(accountNames) == 0 {
		return numbers
	}
	enriched := make([]AccountNumberSummary, 0, len(numbers))
	for _, number := range numbers {
		enriched = append(enriched, enrichAccountNumberSummary(number, accountNames))
	}
	return enriched
}

func (s Services) attachAccountNameToAccountNumberDetails(ctx context.Context, api *increasex.Client, details AccountNumberDetails) AccountNumberDetails {
	accountNames := s.resolveAccountNames(ctx, api, []string{details.AccountID})
	return enrichAccountNumberDetails(details, accountNames)
}

func (s Services) resolveAccountNames(ctx context.Context, api *increasex.Client, accountIDs []string) map[string]string {
	names := map[string]string{}
	for _, accountID := range uniqueNonEmptyStrings(accountIDs) {
		result, err := api.GetAccount(ctx, accountID)
		if err != nil || result.Data == nil {
			continue
		}
		if strings.TrimSpace(result.Data.Name) != "" {
			names[accountID] = result.Data.Name
		}
	}
	return names
}

func collectAccountNumberAccountIDs(numbers []AccountNumberSummary) []string {
	ids := make([]string, 0, len(numbers))
	for _, number := range numbers {
		ids = append(ids, number.AccountID)
	}
	return ids
}

func uniqueNonEmptyStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	sort.Strings(out)
	return out
}

func enrichAccountNumberSummary(summary AccountNumberSummary, accountNames map[string]string) AccountNumberSummary {
	if accountName := strings.TrimSpace(accountNames[summary.AccountID]); accountName != "" {
		summary.AccountName = accountName
	}
	return summary
}

func enrichAccountNumberDetails(details AccountNumberDetails, accountNames map[string]string) AccountNumberDetails {
	details.AccountNumberSummary = enrichAccountNumberSummary(details.AccountNumberSummary, accountNames)
	return details
}

func (s Services) ListPrograms(ctx context.Context, api *increasex.Client, limit int64, cursor string) ([]ProgramSummary, string, error) {
	params := increase.ProgramListParams{}
	if limit > 0 {
		params.Limit = increase.Int(limit)
	}
	if cursor != "" {
		params.Cursor = increase.String(cursor)
	}
	result, err := api.ListPrograms(ctx, params)
	if err != nil {
		return nil, "", err
	}
	programs := make([]ProgramSummary, 0, len(result.Data))
	for _, program := range result.Data {
		programs = append(programs, normalizeProgram(program))
	}
	return programs, result.RequestID, nil
}

func (s Services) RetrieveProgram(ctx context.Context, api *increasex.Client, programID string) (*ProgramSummary, string, error) {
	result, err := api.GetProgram(ctx, programID)
	if err != nil {
		return nil, "", err
	}
	summary := normalizeProgram(*result.Data)
	return &summary, result.RequestID, nil
}

func (s Services) ListDigitalCardProfiles(ctx context.Context, api *increasex.Client, status, idempotencyKey, cursor string, limit int64) ([]DigitalCardProfileSummary, string, error) {
	params := increase.DigitalCardProfileListParams{}
	if cursor != "" {
		params.Cursor = increase.String(cursor)
	}
	if idempotencyKey != "" {
		params.IdempotencyKey = increase.String(idempotencyKey)
	}
	if limit > 0 {
		params.Limit = increase.Int(limit)
	}
	if status != "" {
		params.Status = increase.F(increase.DigitalCardProfileListParamsStatus{
			In: increase.F([]increase.DigitalCardProfileListParamsStatusIn{increase.DigitalCardProfileListParamsStatusIn(status)}),
		})
	}
	result, err := api.ListDigitalCardProfiles(ctx, params)
	if err != nil {
		return nil, "", err
	}
	profiles := make([]DigitalCardProfileSummary, 0, len(result.Data))
	for _, profile := range result.Data {
		profiles = append(profiles, normalizeDigitalCardProfile(profile))
	}
	return profiles, result.RequestID, nil
}

func (s Services) RetrieveDigitalCardProfile(ctx context.Context, api *increasex.Client, digitalCardProfileID string) (*DigitalCardProfileSummary, string, error) {
	result, err := api.GetDigitalCardProfile(ctx, digitalCardProfileID)
	if err != nil {
		return nil, "", err
	}
	summary := normalizeDigitalCardProfile(*result.Data)
	return &summary, result.RequestID, nil
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

func (s Services) ListEvents(ctx context.Context, api *increasex.Client, input ListEventsInput) ([]EventSummary, string, error) {
	params := increase.EventListParams{}
	if input.AssociatedObjectID != "" {
		params.AssociatedObjectID = increase.String(input.AssociatedObjectID)
	}
	if input.Cursor != "" {
		params.Cursor = increase.String(input.Cursor)
	}
	if input.Limit > 0 {
		params.Limit = increase.Int(input.Limit)
	}
	if len(input.Categories) > 0 {
		values := make([]increase.EventListParamsCategoryIn, 0, len(input.Categories))
		for _, category := range input.Categories {
			values = append(values, increase.EventListParamsCategoryIn(category))
		}
		params.Category = increase.F(increase.EventListParamsCategory{
			In: increase.F(values),
		})
	}
	since, until, err := parseOptionalRFC3339Bounds(input.TimeRange.Since, input.TimeRange.Until, "since", "until")
	if err != nil {
		return nil, "", err
	}
	if since != nil || until != nil {
		createdAt := increase.EventListParamsCreatedAt{}
		if since != nil {
			createdAt.OnOrAfter = increase.F(*since)
		}
		if until != nil {
			createdAt.OnOrBefore = increase.F(*until)
		}
		params.CreatedAt = increase.F(createdAt)
	}
	result, err := api.ListEvents(ctx, params)
	if err != nil {
		return nil, "", err
	}
	events := make([]EventSummary, 0, len(result.Data))
	for _, event := range result.Data {
		events = append(events, normalizeEvent(event))
	}
	return events, result.RequestID, nil
}

func (s Services) RetrieveEvent(ctx context.Context, api *increasex.Client, eventID string) (*EventSummary, string, error) {
	result, err := api.GetEvent(ctx, eventID)
	if err != nil {
		return nil, "", err
	}
	summary := normalizeEvent(*result.Data)
	return &summary, result.RequestID, nil
}

func (s Services) ListDocuments(ctx context.Context, api *increasex.Client, input ListDocumentsInput) ([]DocumentSummary, string, error) {
	since, until, err := parseOptionalRFC3339Bounds(input.TimeRange.Since, input.TimeRange.Until, "since", "until")
	if err != nil {
		return nil, "", err
	}
	params := increasex.DocumentListParams{
		Cursor:         input.Cursor,
		EntityID:       input.EntityID,
		Categories:     input.Categories,
		IdempotencyKey: input.IdempotencyKey,
		Limit:          input.Limit,
	}
	if since != nil {
		params.CreatedOnOrAfter = since
	}
	if until != nil {
		params.CreatedOnOrBefore = until
	}
	result, err := api.ListDocuments(ctx, params)
	if err != nil {
		return nil, "", err
	}
	documents := make([]DocumentSummary, 0, len(result.Data))
	for _, document := range result.Data {
		documents = append(documents, normalizeDocument(document))
	}
	return documents, result.RequestID, nil
}

func (s Services) RetrieveDocument(ctx context.Context, api *increasex.Client, documentID string) (*DocumentSummary, string, error) {
	result, err := api.GetDocument(ctx, documentID)
	if err != nil {
		return nil, "", err
	}
	summary := normalizeDocument(*result.Data)
	return &summary, result.RequestID, nil
}

func (s Services) ListRecentTransactions(ctx context.Context, api *increasex.Client, input ListTransactionsInput) ([]TransactionSummary, string, error) {
	params, err := s.buildTransactionListParams(input)
	if err != nil {
		return nil, "", err
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

func (s Services) buildTransactionListParams(input ListTransactionsInput) (increase.TransactionListParams, error) {
	params := increase.TransactionListParams{}
	if input.AccountID != "" {
		params.AccountID = increase.String(input.AccountID)
	}
	if input.Limit > 0 {
		params.Limit = increase.Int(input.Limit)
	}
	if input.Cursor != "" {
		params.Cursor = increase.String(input.Cursor)
	}
	createdAt, err := s.resolveTransactionCreatedAt(input.TimeRange)
	if err != nil {
		return increase.TransactionListParams{}, err
	}
	params.CreatedAt = increase.F(createdAt)
	if len(input.Categories) > 0 {
		values := make([]increase.TransactionListParamsCategoryIn, 0, len(input.Categories))
		for _, category := range input.Categories {
			values = append(values, increase.TransactionListParamsCategoryIn(category))
		}
		params.Category = increase.F(increase.TransactionListParamsCategory{
			In: increase.F(values),
		})
	}
	return params, nil
}

func (s Services) resolveTransactionCreatedAt(input TransactionTimeRangeInput) (increase.TransactionListParamsCreatedAt, error) {
	since, until, err := s.resolveTransactionTimeRange(input)
	if err != nil {
		return increase.TransactionListParamsCreatedAt{}, err
	}
	createdAt := increase.TransactionListParamsCreatedAt{}
	if since != nil {
		createdAt.OnOrAfter = increase.F(*since)
	}
	if until != nil {
		createdAt.OnOrBefore = increase.F(*until)
	}
	return createdAt, nil
}

func (s Services) resolveTransactionTimeRange(input TransactionTimeRangeInput) (*time.Time, *time.Time, error) {
	sinceText := strings.TrimSpace(input.Since)
	untilText := strings.TrimSpace(input.Until)
	period := strings.TrimSpace(input.Period)

	if sinceText != "" || untilText != "" {
		period = ""
	}

	var since, until *time.Time
	if period != "" {
		resolvedSince, resolvedUntil, err := s.resolveTransactionPeriodPreset(period)
		if err != nil {
			return nil, nil, err
		}
		since = &resolvedSince
		until = &resolvedUntil
	}

	if sinceText != "" {
		parsed, err := increasex.ParseRFC3339(sinceText)
		if err != nil {
			return nil, nil, util.NewError(util.CodeValidationError, "since must be RFC3339", nil, false)
		}
		value := parsed.UTC()
		since = &value
	}
	if untilText != "" {
		parsed, err := increasex.ParseRFC3339(untilText)
		if err != nil {
			return nil, nil, util.NewError(util.CodeValidationError, "until must be RFC3339", nil, false)
		}
		value := parsed.UTC()
		until = &value
	}

	if since == nil && until == nil {
		now := s.currentTime().UTC()
		value := now.AddDate(0, 0, -30)
		since = &value
	}
	if since != nil && until != nil && since.After(*until) {
		return nil, nil, util.NewError(util.CodeValidationError, "since must be before or equal to until", nil, false)
	}
	return since, until, nil
}

func (s Services) resolveTransactionPeriodPreset(period string) (time.Time, time.Time, error) {
	now := s.currentTime().UTC()
	switch period {
	case "last-7d":
		return now.AddDate(0, 0, -7), now, nil
	case "last-30d":
		return now.AddDate(0, 0, -30), now, nil
	case "current-month":
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		return start, now, nil
	case "previous-month":
		currentMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		previousMonthStart := currentMonthStart.AddDate(0, -1, 0)
		previousMonthEnd := currentMonthStart.Add(-time.Second)
		return previousMonthStart, previousMonthEnd, nil
	default:
		return time.Time{}, time.Time{}, util.NewError(util.CodeValidationError, "period must be one of last-7d, last-30d, current-month, previous-month", nil, false)
	}
}

func parseOptionalRFC3339Bounds(sinceText, untilText, sinceLabel, untilLabel string) (*time.Time, *time.Time, error) {
	sinceText = strings.TrimSpace(sinceText)
	untilText = strings.TrimSpace(untilText)

	var since, until *time.Time
	if sinceText != "" {
		parsed, err := increasex.ParseRFC3339(sinceText)
		if err != nil {
			return nil, nil, util.NewError(util.CodeValidationError, fmt.Sprintf("%s must be RFC3339", sinceLabel), nil, false)
		}
		value := parsed.UTC()
		since = &value
	}
	if untilText != "" {
		parsed, err := increasex.ParseRFC3339(untilText)
		if err != nil {
			return nil, nil, util.NewError(util.CodeValidationError, fmt.Sprintf("%s must be RFC3339", untilLabel), nil, false)
		}
		value := parsed.UTC()
		until = &value
	}
	if since != nil && until != nil && since.After(*until) {
		return nil, nil, util.NewError(util.CodeValidationError, fmt.Sprintf("%s must be before or equal to %s", sinceLabel, untilLabel), nil, false)
	}
	return since, until, nil
}

func (s Services) currentTime() time.Time {
	if s.now != nil {
		return s.now()
	}
	return time.Now()
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
	if err := validateCreateAccountNumberInput(input); err != nil {
		return nil, err
	}
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
	if err := validateCreateAccountNumberInput(input); err != nil {
		return nil, "", err
	}
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

func (s Services) PreviewDisableAccountNumber(session Session, input DisableAccountNumberInput) (*PreviewResult, error) {
	if err := validateDisableAccountNumberInput(input); err != nil {
		return nil, err
	}
	effective := effectiveConfirmationPayload(input)
	token, err := s.confirm.Generate("disable_account_number", session, effective)
	if err != nil {
		return nil, err
	}
	return &PreviewResult{
		Mode:              "preview",
		Summary:           fmt.Sprintf("Disable account number %s", input.AccountNumberID),
		ConfirmationToken: token,
		Details:           effective,
	}, nil
}

func (s Services) ExecuteDisableAccountNumber(ctx context.Context, api *increasex.Client, session Session, input DisableAccountNumberInput) (any, string, error) {
	if err := validateDisableAccountNumberInput(input); err != nil {
		return nil, "", err
	}
	if err := s.confirm.Verify(input.ConfirmationToken, "disable_account_number", session, effectiveConfirmationPayload(input)); err != nil {
		return nil, "", err
	}
	result, err := api.UpdateAccountNumber(ctx, input.AccountNumberID, increase.AccountNumberUpdateParams{
		Status: increase.F(increase.AccountNumberUpdateParamsStatusDisabled),
	}, input.IdempotencyKey)
	if err != nil {
		return nil, "", err
	}
	return map[string]any{
		"mode":           "executed",
		"account_number": normalizeAccountNumber(*result.Data),
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
		Summary:           transferPreviewSummary("account", input.RequireApproval, util.FormatUSDMinor(input.AmountCents)),
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
		"outcome":         transferOutcome(input.RequireApproval),
		"message":         transferOutcomeMessage("account", input.RequireApproval),
		"transfer_id":     result.Data.ID,
		"status":          string(result.Data.Status),
		"from_account_id": result.Data.AccountID,
		"to_account_id":   result.Data.DestinationAccountID,
		"amount_cents":    result.Data.Amount,
		"created_at":      util.RFC3339OrEmpty(result.Data.CreatedAt),
	}, result.RequestID, nil
}

func (s Services) PreviewCreateCard(session Session, input CreateCardInput) (*PreviewResult, error) {
	if err := validateCreateCardInput(input); err != nil {
		return nil, err
	}
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
	if err := validateCreateCardInput(input); err != nil {
		return nil, "", err
	}
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
		Summary:           transferPreviewSummary("ach", input.RequireApproval, util.FormatUSDMinor(input.AmountCents)),
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
		"outcome":     transferOutcome(input.RequireApproval),
		"message":     transferOutcomeMessage("ach", input.RequireApproval),
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
		Summary:           transferPreviewSummary("real_time_payments", input.RequireApproval, util.FormatUSDMinor(input.AmountCents)),
		ConfirmationToken: token,
		Details:           effective,
	}, nil
}

func (s Services) ExecuteExternalRTP(ctx context.Context, api *increasex.Client, session Session, input RTPTransferInput) (any, string, error) {
	if err := s.confirm.Verify(input.ConfirmationToken, "move_money_external_rtp", session, effectiveConfirmationPayload(input)); err != nil {
		return nil, "", err
	}
	if strings.TrimSpace(input.SourceAccountNumberID) == "" {
		return nil, "", sourceAccountNumberRequiredError("source_account_number_id")
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
		"outcome":     transferOutcome(input.RequireApproval),
		"message":     transferOutcomeMessage("real_time_payments", input.RequireApproval),
		"rail":        "real_time_payments",
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
		Summary:           transferPreviewSummary("fednow", input.RequireApproval, util.FormatUSDMinor(input.AmountCents)),
		ConfirmationToken: token,
		Details:           effective,
	}, nil
}

func (s Services) ExecuteExternalFedNow(ctx context.Context, api *increasex.Client, session Session, input FedNowTransferInput) (any, string, error) {
	if err := s.confirm.Verify(input.ConfirmationToken, "move_money_external_fednow", session, effectiveConfirmationPayload(input)); err != nil {
		return nil, "", err
	}
	if strings.TrimSpace(input.SourceAccountNumberID) == "" {
		return nil, "", sourceAccountNumberRequiredError("source_account_number_id")
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
		"outcome":     transferOutcome(input.RequireApproval),
		"message":     transferOutcomeMessage("fednow", input.RequireApproval),
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
		Summary:           transferPreviewSummary("wire", input.RequireApproval, util.FormatUSDMinor(input.AmountCents)),
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
	if input.SourceAccountNumberID != "" {
		params.SourceAccountNumberID = increase.String(input.SourceAccountNumberID)
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
		"outcome":     transferOutcome(input.RequireApproval),
		"message":     transferOutcomeMessage("wire", input.RequireApproval),
		"rail":        "wire",
		"transfer_id": result.Data.ID,
		"status":      string(result.Data.Status),
		"created_at":  util.RFC3339OrEmpty(result.Data.CreatedAt),
	}, result.RequestID, nil
}

func transferPreviewSummary(rail string, requireApproval *bool, amount string) string {
	if isApprovalRequired(requireApproval) {
		return fmt.Sprintf("Queue %s transfer %s for approval", transferRailLabel(rail), amount)
	}
	return fmt.Sprintf("Create %s transfer %s", transferRailLabel(rail), amount)
}

func transferOutcome(requireApproval *bool) string {
	if isApprovalRequired(requireApproval) {
		return "queued_for_approval"
	}
	return "submitted"
}

func transferOutcomeMessage(rail string, requireApproval *bool) string {
	if isApprovalRequired(requireApproval) {
		return fmt.Sprintf("%s transfer queued for approval", transferRailLabel(rail))
	}
	return fmt.Sprintf("%s transfer submitted", transferRailLabel(rail))
}

func transferRailLabel(rail string) string {
	switch rail {
	case "account":
		return "account"
	case "ach":
		return "ACH"
	case "real_time_payments":
		return "Real-Time Payments"
	case "fednow":
		return "FedNow"
	case "wire":
		return "wire"
	default:
		return rail
	}
}

func isApprovalRequired(requireApproval *bool) bool {
	return requireApproval != nil && *requireApproval
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

func normalizeAccountNumber(number increase.AccountNumber) AccountNumberSummary {
	return AccountNumberSummary{
		ID:                  number.ID,
		AccountID:           number.AccountID,
		Name:                number.Name,
		RoutingNumber:       number.RoutingNumber,
		AccountNumberMasked: util.MaskAccountNumber(number.AccountNumber),
		Status:              string(number.Status),
		CreatedAt:           util.RFC3339OrEmpty(number.CreatedAt),
		InboundACH: &InboundACHInput{
			DebitStatus: string(number.InboundACH.DebitStatus),
		},
		InboundChecks: &InboundChecksInput{
			Status: string(number.InboundChecks.Status),
		},
	}
}

func normalizeAccountNumberDetails(number increase.AccountNumber) AccountNumberDetails {
	return AccountNumberDetails{
		AccountNumberSummary: normalizeAccountNumber(number),
		IdempotencyKey:       number.IdempotencyKey,
	}
}

func normalizeSensitiveAccountNumberDetails(number increase.AccountNumber) AccountNumberDetails {
	details := normalizeAccountNumberDetails(number)
	details.AccountNumber = number.AccountNumber
	return details
}

func normalizeProgram(program increase.Program) ProgramSummary {
	summary := ProgramSummary{
		ID:                          program.ID,
		Name:                        program.Name,
		Bank:                        string(program.Bank),
		BillingAccountID:            program.BillingAccountID,
		DefaultDigitalCardProfileID: program.DefaultDigitalCardProfileID,
		InterestRate:                program.InterestRate,
		CreatedAt:                   util.RFC3339OrEmpty(program.CreatedAt),
		UpdatedAt:                   util.RFC3339OrEmpty(program.UpdatedAt),
	}
	if program.Lending.MaximumExtendableCredit != 0 {
		summary.MaximumExtendableCredit = program.Lending.MaximumExtendableCredit
	}
	return summary
}

func normalizeDigitalCardProfile(profile increase.DigitalCardProfile) DigitalCardProfileSummary {
	return DigitalCardProfileSummary{
		ID:                    profile.ID,
		Description:           profile.Description,
		CardDescription:       profile.CardDescription,
		IssuerName:            profile.IssuerName,
		Status:                string(profile.Status),
		AppIconFileID:         profile.AppIconFileID,
		BackgroundImageFileID: profile.BackgroundImageFileID,
		ContactEmail:          profile.ContactEmail,
		ContactPhone:          profile.ContactPhone,
		ContactWebsite:        profile.ContactWebsite,
		TextColor: DigitalCardProfileTextColorSummary{
			Red:   profile.TextColor.Red,
			Green: profile.TextColor.Green,
			Blue:  profile.TextColor.Blue,
		},
		CreatedAt: util.RFC3339OrEmpty(profile.CreatedAt),
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

func normalizeEvent(event increase.Event) EventSummary {
	return EventSummary{
		ID:                   event.ID,
		AssociatedObjectID:   event.AssociatedObjectID,
		AssociatedObjectType: event.AssociatedObjectType,
		Category:             string(event.Category),
		CreatedAt:            util.RFC3339OrEmpty(event.CreatedAt),
	}
}

func normalizeDocument(document increasex.Document) DocumentSummary {
	return DocumentSummary{
		ID:                        document.ID,
		Category:                  document.Category,
		EntityID:                  document.EntityID,
		FileID:                    document.FileID,
		IdempotencyKey:            document.IdempotencyKey,
		CreatedAt:                 util.RFC3339OrEmpty(document.CreatedAt),
		AccountVerificationLetter: document.AccountVerificationLetter,
		FundingInstructions:       document.FundingInstructions,
	}
}

func validateCreateAccountNumberInput(input CreateAccountNumberInput) error {
	fields := []util.FieldError{}
	if strings.TrimSpace(input.AccountID) == "" {
		fields = append(fields, util.FieldError{Field: "account_id", Message: "is required"})
	}
	if strings.TrimSpace(input.Name) == "" {
		fields = append(fields, util.FieldError{Field: "name", Message: "is required"})
	}
	if len(fields) == 0 {
		return nil
	}
	return &util.ErrorDetail{
		Code:    util.CodeValidationError,
		Message: "Please correct the highlighted account number fields.",
		Fields:  fields,
	}
}

func validateDisableAccountNumberInput(input DisableAccountNumberInput) error {
	if strings.TrimSpace(input.AccountNumberID) != "" {
		return nil
	}
	return &util.ErrorDetail{
		Code:    util.CodeValidationError,
		Message: "Please select an account number to disable.",
		Fields: []util.FieldError{
			{Field: "account_number_id", Message: "is required"},
		},
	}
}

func sourceAccountNumberRequiredError(field string) error {
	return &util.ErrorDetail{
		Code:    util.CodeValidationError,
		Message: "source_account_number_id is required for this transfer rail. Use list_account_numbers to discover one or create_account_number to mint a new one.",
		Details: map[string]any{
			"discovery_tool": "list_account_numbers",
			"create_tool":    "create_account_number",
		},
		Fields: []util.FieldError{
			{Field: field, Message: "is required"},
		},
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
