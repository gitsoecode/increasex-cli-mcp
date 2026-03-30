package app

import (
	"context"
	"fmt"
	"strings"

	increase "github.com/Increase/increase-go"
	increasex "github.com/gitsoecode/increasex-cli-mcp/internal/increase"
	"github.com/gitsoecode/increasex-cli-mcp/internal/util"
)

func (s Services) ListExternalAccounts(ctx context.Context, api *increasex.Client, status, cursor string, limit int64) ([]ExternalAccountSummary, string, error) {
	params := increase.ExternalAccountListParams{}
	if cursor != "" {
		params.Cursor = increase.String(cursor)
	}
	if limit > 0 {
		params.Limit = increase.Int(limit)
	}
	if status != "" {
		params.Status = increase.F(increase.ExternalAccountListParamsStatus{
			In: increase.F([]increase.ExternalAccountListParamsStatusIn{increase.ExternalAccountListParamsStatusIn(status)}),
		})
	}
	result, err := api.ListExternalAccounts(ctx, params)
	if err != nil {
		return nil, "", err
	}
	items := make([]ExternalAccountSummary, 0, len(result.Data))
	for _, account := range result.Data {
		items = append(items, normalizeExternalAccount(account))
	}
	return items, result.RequestID, nil
}

func (s Services) RetrieveExternalAccount(ctx context.Context, api *increasex.Client, externalAccountID string) (*ExternalAccountSummary, string, error) {
	result, err := api.GetExternalAccount(ctx, externalAccountID)
	if err != nil {
		return nil, "", err
	}
	summary := normalizeExternalAccount(*result.Data)
	return &summary, result.RequestID, nil
}

func (s Services) PreviewCreateExternalAccount(session Session, input CreateExternalAccountInput) (*PreviewResult, error) {
	if err := validateCreateExternalAccountInput(input); err != nil {
		return nil, err
	}
	effective := effectiveConfirmationPayload(input)
	token, err := s.confirm.Generate("create_external_account", session, effective)
	if err != nil {
		return nil, err
	}
	return &PreviewResult{
		Mode:              "preview",
		Summary:           fmt.Sprintf("Create external account %q", input.Description),
		ConfirmationToken: token,
		Details:           effective,
	}, nil
}

func (s Services) ExecuteCreateExternalAccount(ctx context.Context, api *increasex.Client, session Session, input CreateExternalAccountInput) (any, string, error) {
	if err := validateCreateExternalAccountInput(input); err != nil {
		return nil, "", err
	}
	if err := s.confirm.Verify(input.ConfirmationToken, "create_external_account", session, effectiveConfirmationPayload(input)); err != nil {
		return nil, "", err
	}
	params := increase.ExternalAccountNewParams{
		AccountNumber: increase.String(input.AccountNumber),
		Description:   increase.String(input.Description),
		RoutingNumber: increase.String(input.RoutingNumber),
	}
	if input.AccountHolder != "" {
		params.AccountHolder = increase.F(increase.ExternalAccountNewParamsAccountHolder(input.AccountHolder))
	}
	if input.Funding != "" {
		params.Funding = increase.F(increase.ExternalAccountNewParamsFunding(input.Funding))
	}
	result, err := api.CreateExternalAccount(ctx, params, input.IdempotencyKey)
	if err != nil {
		return nil, "", err
	}
	return map[string]any{
		"mode":             "executed",
		"external_account": normalizeExternalAccount(*result.Data),
	}, result.RequestID, nil
}

func (s Services) PreviewUpdateExternalAccount(session Session, input UpdateExternalAccountInput) (*PreviewResult, error) {
	if err := validateUpdateExternalAccountInput(input); err != nil {
		return nil, err
	}
	effective := effectiveConfirmationPayload(input)
	token, err := s.confirm.Generate("update_external_account", session, effective)
	if err != nil {
		return nil, err
	}
	return &PreviewResult{
		Mode:              "preview",
		Summary:           fmt.Sprintf("Update external account %s", input.ExternalAccountID),
		ConfirmationToken: token,
		Details:           effective,
	}, nil
}

func (s Services) ExecuteUpdateExternalAccount(ctx context.Context, api *increasex.Client, session Session, input UpdateExternalAccountInput) (any, string, error) {
	if err := validateUpdateExternalAccountInput(input); err != nil {
		return nil, "", err
	}
	if err := s.confirm.Verify(input.ConfirmationToken, "update_external_account", session, effectiveConfirmationPayload(input)); err != nil {
		return nil, "", err
	}
	params := increase.ExternalAccountUpdateParams{}
	if input.AccountHolder != "" {
		params.AccountHolder = increase.F(increase.ExternalAccountUpdateParamsAccountHolder(input.AccountHolder))
	}
	if input.Description != "" {
		params.Description = increase.String(input.Description)
	}
	if input.Funding != "" {
		params.Funding = increase.F(increase.ExternalAccountUpdateParamsFunding(input.Funding))
	}
	if input.Status != "" {
		params.Status = increase.F(increase.ExternalAccountUpdateParamsStatus(input.Status))
	}
	result, err := api.UpdateExternalAccount(ctx, input.ExternalAccountID, params, input.IdempotencyKey)
	if err != nil {
		return nil, "", err
	}
	return map[string]any{
		"mode":             "executed",
		"external_account": normalizeExternalAccount(*result.Data),
	}, result.RequestID, nil
}

func (s Services) RetrieveSensitiveCardDetails(ctx context.Context, api *increasex.Client, cardID string) (*CardDetailsSummary, string, error) {
	cardResult, err := api.GetCard(ctx, cardID)
	if err != nil {
		return nil, "", err
	}
	result, err := api.GetCardDetails(ctx, cardID)
	if err != nil {
		return nil, "", err
	}
	card := normalizeSensitiveCard(*cardResult.Data)
	card.BillingDetails = normalizeCardBillingDetails(cardResult.Data.BillingAddress, false)
	card.ExpirationMonth = result.Data.ExpirationMonth
	card.ExpirationYear = result.Data.ExpirationYear
	card.PrimaryAccountNumber = result.Data.PrimaryAccountNumber
	card.VerificationCode = result.Data.VerificationCode
	card.PIN = result.Data.Pin
	requestID := result.RequestID
	if strings.TrimSpace(requestID) == "" {
		requestID = cardResult.RequestID
	}
	return &card, requestID, nil
}

func (s Services) CreateCardDetailsIframe(ctx context.Context, api *increasex.Client, input CreateCardDetailsIframeInput) (*CardDetailsIframeResult, string, error) {
	params := increase.CardNewDetailsIframeParams{}
	if input.PhysicalCardID != "" {
		params.PhysicalCardID = increase.String(input.PhysicalCardID)
	}
	result, err := api.CreateCardDetailsIframe(ctx, input.CardID, params)
	if err != nil {
		return nil, "", err
	}
	return &CardDetailsIframeResult{
		CardID:    input.CardID,
		IframeURL: result.Data.IframeURL,
		ExpiresAt: util.RFC3339OrEmpty(result.Data.ExpiresAt),
	}, result.RequestID, nil
}

func (s Services) PreviewUpdateCardPIN(session Session, input UpdateCardPINInput) (*PreviewResult, error) {
	effective := effectiveConfirmationPayload(input)
	effective["pin"] = "****"
	token, err := s.confirm.Generate("update_card_pin", session, effective)
	if err != nil {
		return nil, err
	}
	return &PreviewResult{
		Mode:              "preview",
		Summary:           fmt.Sprintf("Update PIN for card %s", input.CardID),
		ConfirmationToken: token,
		Details:           effective,
	}, nil
}

func (s Services) ExecuteUpdateCardPIN(ctx context.Context, api *increasex.Client, session Session, input UpdateCardPINInput) (any, string, error) {
	effective := effectiveConfirmationPayload(input)
	effective["pin"] = "****"
	if err := s.confirm.Verify(input.ConfirmationToken, "update_card_pin", session, effective); err != nil {
		return nil, "", err
	}
	result, err := api.UpdateCardPIN(ctx, input.CardID, increase.CardUpdatePinParams{
		Pin: increase.String(input.PIN),
	})
	if err != nil {
		return nil, "", err
	}
	return map[string]any{
		"mode":         "executed",
		"card_id":      result.Data.CardID,
		"pin_updated":  true,
		"details_type": string(result.Data.Type),
	}, result.RequestID, nil
}

func (s Services) ListTransfers(ctx context.Context, api *increasex.Client, input ListTransfersInput) ([]TransferSummary, string, error) {
	switch input.Rail {
	case "account":
		return s.listAccountTransfers(ctx, api, input)
	case "ach":
		return s.listACHTransfers(ctx, api, input)
	case "real_time_payments":
		return s.listRTPTransfers(ctx, api, input)
	case "fednow":
		return s.listFedNowTransfers(ctx, api, input)
	case "wire":
		return s.listWireTransfers(ctx, api, input)
	default:
		return nil, "", util.NewError(util.CodeValidationError, "unsupported rail", map[string]any{"rail": input.Rail}, false)
	}
}

func (s Services) ListTransferQueue(ctx context.Context, api *increasex.Client, rail string, limit int64) ([]TransferSummary, string, error) {
	items, requestID, err := s.ListTransfers(ctx, api, ListTransfersInput{
		Rail:   rail,
		Status: "pending_approval",
		Limit:  limit,
	})
	if err != nil {
		return nil, "", err
	}
	filtered := make([]TransferSummary, 0, len(items))
	for _, item := range items {
		if item.Status == "pending_approval" {
			filtered = append(filtered, item)
		}
	}
	return filtered, requestID, nil
}

func (s Services) RetrieveTransfer(ctx context.Context, api *increasex.Client, rail, transferID string) (*TransferSummary, string, error) {
	switch rail {
	case "account":
		result, err := api.GetInternalTransfer(ctx, transferID)
		if err != nil {
			return nil, "", err
		}
		summary := normalizeAccountTransfer(*result.Data)
		return &summary, result.RequestID, nil
	case "ach":
		result, err := api.GetACHTransfer(ctx, transferID)
		if err != nil {
			return nil, "", err
		}
		summary := normalizeACHTransfer(*result.Data)
		return &summary, result.RequestID, nil
	case "real_time_payments":
		result, err := api.GetRTPTransfer(ctx, transferID)
		if err != nil {
			return nil, "", err
		}
		summary := normalizeRTPTransfer(*result.Data)
		return &summary, result.RequestID, nil
	case "fednow":
		result, err := api.GetFedNowTransfer(ctx, transferID)
		if err != nil {
			return nil, "", err
		}
		summary := normalizeFedNowTransfer(*result.Data)
		return &summary, result.RequestID, nil
	case "wire":
		result, err := api.GetWireTransfer(ctx, transferID)
		if err != nil {
			return nil, "", err
		}
		summary := normalizeWireTransfer(*result.Data)
		return &summary, result.RequestID, nil
	default:
		return nil, "", util.NewError(util.CodeValidationError, "unsupported rail", map[string]any{"rail": rail}, false)
	}
}

func (s Services) PreviewApproveTransfer(session Session, input TransferActionInput) (*PreviewResult, error) {
	if err := validateTransferActionInput(input); err != nil {
		return nil, err
	}
	effective := effectiveConfirmationPayload(input)
	token, err := s.confirm.Generate("approve_transfer", session, effective)
	if err != nil {
		return nil, err
	}
	return &PreviewResult{
		Mode:              "preview",
		Summary:           fmt.Sprintf("Approve %s transfer %s", input.Rail, input.TransferID),
		ConfirmationToken: token,
		Details:           effective,
	}, nil
}

func (s Services) ExecuteApproveTransfer(ctx context.Context, api *increasex.Client, session Session, input TransferActionInput) (any, string, error) {
	if err := validateTransferActionInput(input); err != nil {
		return nil, "", err
	}
	if err := s.confirm.Verify(input.ConfirmationToken, "approve_transfer", session, effectiveConfirmationPayload(input)); err != nil {
		return nil, "", err
	}
	switch input.Rail {
	case "account":
		result, err := api.ApproveInternalTransfer(ctx, input.TransferID)
		if err != nil {
			return nil, "", err
		}
		return map[string]any{"mode": "executed", "transfer": normalizeAccountTransfer(*result.Data)}, result.RequestID, nil
	case "ach":
		result, err := api.ApproveACHTransfer(ctx, input.TransferID)
		if err != nil {
			return nil, "", err
		}
		return map[string]any{"mode": "executed", "transfer": normalizeACHTransfer(*result.Data)}, result.RequestID, nil
	case "real_time_payments":
		result, err := api.ApproveRTPTransfer(ctx, input.TransferID)
		if err != nil {
			return nil, "", err
		}
		return map[string]any{"mode": "executed", "transfer": normalizeRTPTransfer(*result.Data)}, result.RequestID, nil
	case "fednow":
		result, err := api.ApproveFedNowTransfer(ctx, input.TransferID)
		if err != nil {
			return nil, "", err
		}
		return map[string]any{"mode": "executed", "transfer": normalizeFedNowTransfer(*result.Data)}, result.RequestID, nil
	case "wire":
		result, err := api.ApproveWireTransfer(ctx, input.TransferID)
		if err != nil {
			return nil, "", err
		}
		return map[string]any{"mode": "executed", "transfer": normalizeWireTransfer(*result.Data)}, result.RequestID, nil
	default:
		return nil, "", util.NewError(util.CodeValidationError, "unsupported rail", map[string]any{"rail": input.Rail}, false)
	}
}

func (s Services) PreviewCancelTransfer(session Session, input TransferActionInput) (*PreviewResult, error) {
	if err := validateTransferActionInput(input); err != nil {
		return nil, err
	}
	effective := effectiveConfirmationPayload(input)
	token, err := s.confirm.Generate("cancel_transfer", session, effective)
	if err != nil {
		return nil, err
	}
	return &PreviewResult{
		Mode:              "preview",
		Summary:           fmt.Sprintf("Cancel %s transfer %s", input.Rail, input.TransferID),
		ConfirmationToken: token,
		Details:           effective,
	}, nil
}

func (s Services) ExecuteCancelTransfer(ctx context.Context, api *increasex.Client, session Session, input TransferActionInput) (any, string, error) {
	if err := validateTransferActionInput(input); err != nil {
		return nil, "", err
	}
	if err := s.confirm.Verify(input.ConfirmationToken, "cancel_transfer", session, effectiveConfirmationPayload(input)); err != nil {
		return nil, "", err
	}
	switch input.Rail {
	case "account":
		result, err := api.CancelInternalTransfer(ctx, input.TransferID)
		if err != nil {
			return nil, "", err
		}
		return map[string]any{"mode": "executed", "transfer": normalizeAccountTransfer(*result.Data)}, result.RequestID, nil
	case "ach":
		result, err := api.CancelACHTransfer(ctx, input.TransferID)
		if err != nil {
			return nil, "", err
		}
		return map[string]any{"mode": "executed", "transfer": normalizeACHTransfer(*result.Data)}, result.RequestID, nil
	case "real_time_payments":
		result, err := api.CancelRTPTransfer(ctx, input.TransferID)
		if err != nil {
			return nil, "", err
		}
		return map[string]any{"mode": "executed", "transfer": normalizeRTPTransfer(*result.Data)}, result.RequestID, nil
	case "fednow":
		result, err := api.CancelFedNowTransfer(ctx, input.TransferID)
		if err != nil {
			return nil, "", err
		}
		return map[string]any{"mode": "executed", "transfer": normalizeFedNowTransfer(*result.Data)}, result.RequestID, nil
	case "wire":
		result, err := api.CancelWireTransfer(ctx, input.TransferID)
		if err != nil {
			return nil, "", err
		}
		return map[string]any{"mode": "executed", "transfer": normalizeWireTransfer(*result.Data)}, result.RequestID, nil
	default:
		return nil, "", util.NewError(util.CodeValidationError, "unsupported rail", map[string]any{"rail": input.Rail}, false)
	}
}

func (s Services) listAccountTransfers(ctx context.Context, api *increasex.Client, input ListTransfersInput) ([]TransferSummary, string, error) {
	params := increase.AccountTransferListParams{}
	if input.AccountID != "" {
		params.AccountID = increase.String(input.AccountID)
	}
	if input.Cursor != "" {
		params.Cursor = increase.String(input.Cursor)
	}
	if input.Limit > 0 {
		params.Limit = increase.Int(input.Limit)
	}
	if input.Since != "" {
		parsed, err := increasex.ParseSince(input.Since)
		if err != nil {
			return nil, "", util.NewError(util.CodeValidationError, "since must be RFC3339", nil, false)
		}
		params.CreatedAt = increase.F(increase.AccountTransferListParamsCreatedAt{
			OnOrAfter: increase.F(parsed),
		})
	}
	result, err := api.ListInternalTransfers(ctx, params)
	if err != nil {
		return nil, "", err
	}
	items := make([]TransferSummary, 0, len(result.Data))
	for _, transfer := range result.Data {
		items = append(items, normalizeAccountTransfer(transfer))
	}
	return filterTransferSummaries(items, input), result.RequestID, nil
}

func (s Services) listACHTransfers(ctx context.Context, api *increasex.Client, input ListTransfersInput) ([]TransferSummary, string, error) {
	params := increase.ACHTransferListParams{}
	if input.AccountID != "" {
		params.AccountID = increase.String(input.AccountID)
	}
	if input.Cursor != "" {
		params.Cursor = increase.String(input.Cursor)
	}
	if input.Limit > 0 {
		params.Limit = increase.Int(input.Limit)
	}
	if input.ExternalAccountID != "" {
		params.ExternalAccountID = increase.String(input.ExternalAccountID)
	}
	if input.Status != "" {
		params.Status = increase.F(increase.ACHTransferListParamsStatus{
			In: increase.F([]increase.ACHTransferListParamsStatusIn{increase.ACHTransferListParamsStatusIn(input.Status)}),
		})
	}
	if input.Since != "" {
		parsed, err := increasex.ParseSince(input.Since)
		if err != nil {
			return nil, "", util.NewError(util.CodeValidationError, "since must be RFC3339", nil, false)
		}
		params.CreatedAt = increase.F(increase.ACHTransferListParamsCreatedAt{
			OnOrAfter: increase.F(parsed),
		})
	}
	result, err := api.ListACHTransfers(ctx, params)
	if err != nil {
		return nil, "", err
	}
	items := make([]TransferSummary, 0, len(result.Data))
	for _, transfer := range result.Data {
		items = append(items, normalizeACHTransfer(transfer))
	}
	return filterTransferSummaries(items, input), result.RequestID, nil
}

func (s Services) listRTPTransfers(ctx context.Context, api *increasex.Client, input ListTransfersInput) ([]TransferSummary, string, error) {
	params := increase.RealTimePaymentsTransferListParams{}
	if input.Cursor != "" {
		params.Cursor = increase.String(input.Cursor)
	}
	if input.Limit > 0 {
		params.Limit = increase.Int(input.Limit)
	}
	if input.ExternalAccountID != "" {
		params.ExternalAccountID = increase.String(input.ExternalAccountID)
	}
	if input.Status != "" {
		params.Status = increase.F(increase.RealTimePaymentsTransferListParamsStatus{
			In: increase.F([]increase.RealTimePaymentsTransferListParamsStatusIn{increase.RealTimePaymentsTransferListParamsStatusIn(input.Status)}),
		})
	}
	if input.Since != "" {
		parsed, err := increasex.ParseSince(input.Since)
		if err != nil {
			return nil, "", util.NewError(util.CodeValidationError, "since must be RFC3339", nil, false)
		}
		params.CreatedAt = increase.F(increase.RealTimePaymentsTransferListParamsCreatedAt{
			OnOrAfter: increase.F(parsed),
		})
	}
	result, err := api.ListRTPTransfers(ctx, params)
	if err != nil {
		return nil, "", err
	}
	items := make([]TransferSummary, 0, len(result.Data))
	for _, transfer := range result.Data {
		items = append(items, normalizeRTPTransfer(transfer))
	}
	return filterTransferSummaries(items, input), result.RequestID, nil
}

func (s Services) listFedNowTransfers(ctx context.Context, api *increasex.Client, input ListTransfersInput) ([]TransferSummary, string, error) {
	params := increase.FednowTransferListParams{}
	if input.AccountID != "" {
		params.AccountID = increase.String(input.AccountID)
	}
	if input.Cursor != "" {
		params.Cursor = increase.String(input.Cursor)
	}
	if input.Limit > 0 {
		params.Limit = increase.Int(input.Limit)
	}
	if input.ExternalAccountID != "" {
		params.ExternalAccountID = increase.String(input.ExternalAccountID)
	}
	if input.Status != "" {
		params.Status = increase.F(increase.FednowTransferListParamsStatus{
			In: increase.F([]increase.FednowTransferListParamsStatusIn{increase.FednowTransferListParamsStatusIn(input.Status)}),
		})
	}
	if input.Since != "" {
		parsed, err := increasex.ParseSince(input.Since)
		if err != nil {
			return nil, "", util.NewError(util.CodeValidationError, "since must be RFC3339", nil, false)
		}
		params.CreatedAt = increase.F(increase.FednowTransferListParamsCreatedAt{
			OnOrAfter: increase.F(parsed),
		})
	}
	result, err := api.ListFedNowTransfers(ctx, params)
	if err != nil {
		return nil, "", err
	}
	items := make([]TransferSummary, 0, len(result.Data))
	for _, transfer := range result.Data {
		items = append(items, normalizeFedNowTransfer(transfer))
	}
	return filterTransferSummaries(items, input), result.RequestID, nil
}

func (s Services) listWireTransfers(ctx context.Context, api *increasex.Client, input ListTransfersInput) ([]TransferSummary, string, error) {
	params := increase.WireTransferListParams{}
	if input.AccountID != "" {
		params.AccountID = increase.String(input.AccountID)
	}
	if input.Cursor != "" {
		params.Cursor = increase.String(input.Cursor)
	}
	if input.Limit > 0 {
		params.Limit = increase.Int(input.Limit)
	}
	if input.ExternalAccountID != "" {
		params.ExternalAccountID = increase.String(input.ExternalAccountID)
	}
	if input.Status != "" {
		params.Status = increase.F(increase.WireTransferListParamsStatus{
			In: increase.F([]increase.WireTransferListParamsStatusIn{increase.WireTransferListParamsStatusIn(input.Status)}),
		})
	}
	if input.Since != "" {
		parsed, err := increasex.ParseSince(input.Since)
		if err != nil {
			return nil, "", util.NewError(util.CodeValidationError, "since must be RFC3339", nil, false)
		}
		params.CreatedAt = increase.F(increase.WireTransferListParamsCreatedAt{
			OnOrAfter: increase.F(parsed),
		})
	}
	result, err := api.ListWireTransfers(ctx, params)
	if err != nil {
		return nil, "", err
	}
	items := make([]TransferSummary, 0, len(result.Data))
	for _, transfer := range result.Data {
		items = append(items, normalizeWireTransfer(transfer))
	}
	return filterTransferSummaries(items, input), result.RequestID, nil
}

func normalizeExternalAccount(account increase.ExternalAccount) ExternalAccountSummary {
	return ExternalAccountSummary{
		ID:                  account.ID,
		Description:         account.Description,
		AccountHolder:       string(account.AccountHolder),
		Funding:             string(account.Funding),
		RoutingNumber:       account.RoutingNumber,
		AccountNumberMasked: util.MaskAccountNumber(account.AccountNumber),
		Status:              string(account.Status),
		CreatedAt:           util.RFC3339OrEmpty(account.CreatedAt),
	}
}

func normalizeAccountTransfer(transfer increase.AccountTransfer) TransferSummary {
	return TransferSummary{
		Rail:                 "account",
		ID:                   transfer.ID,
		AccountID:            transfer.AccountID,
		AmountCents:          transfer.Amount,
		Status:               string(transfer.Status),
		CreatedAt:            util.RFC3339OrEmpty(transfer.CreatedAt),
		PendingTransactionID: transfer.PendingTransactionID,
		Counterparty:         transfer.DestinationAccountID,
	}
}

func normalizeACHTransfer(transfer increase.ACHTransfer) TransferSummary {
	return TransferSummary{
		Rail:                 "ach",
		ID:                   transfer.ID,
		AccountID:            transfer.AccountID,
		AmountCents:          transfer.Amount,
		Status:               string(transfer.Status),
		CreatedAt:            util.RFC3339OrEmpty(transfer.CreatedAt),
		ExternalAccountID:    transfer.ExternalAccountID,
		PendingTransactionID: transfer.PendingTransactionID,
		Counterparty:         firstNonEmpty(transfer.IndividualName, transfer.CompanyName, transfer.StatementDescriptor),
	}
}

func normalizeRTPTransfer(transfer increase.RealTimePaymentsTransfer) TransferSummary {
	return TransferSummary{
		Rail:                 "real_time_payments",
		ID:                   transfer.ID,
		AccountID:            transfer.AccountID,
		AmountCents:          transfer.Amount,
		Status:               string(transfer.Status),
		CreatedAt:            util.RFC3339OrEmpty(transfer.CreatedAt),
		ExternalAccountID:    transfer.ExternalAccountID,
		PendingTransactionID: transfer.PendingTransactionID,
		Counterparty:         transfer.CreditorName,
	}
}

func normalizeFedNowTransfer(transfer increase.FednowTransfer) TransferSummary {
	return TransferSummary{
		Rail:                 "fednow",
		ID:                   transfer.ID,
		AccountID:            transfer.AccountID,
		AmountCents:          transfer.Amount,
		Status:               string(transfer.Status),
		CreatedAt:            util.RFC3339OrEmpty(transfer.CreatedAt),
		ExternalAccountID:    transfer.ExternalAccountID,
		PendingTransactionID: transfer.PendingTransactionID,
		Counterparty:         transfer.CreditorName,
	}
}

func normalizeWireTransfer(transfer increase.WireTransfer) TransferSummary {
	return TransferSummary{
		Rail:                 "wire",
		ID:                   transfer.ID,
		AccountID:            transfer.AccountID,
		AmountCents:          transfer.Amount,
		Status:               string(transfer.Status),
		CreatedAt:            util.RFC3339OrEmpty(transfer.CreatedAt),
		ExternalAccountID:    transfer.ExternalAccountID,
		PendingTransactionID: transfer.PendingTransactionID,
		Counterparty:         transfer.Creditor.Name,
	}
}

func filterTransferSummaries(items []TransferSummary, input ListTransfersInput) []TransferSummary {
	if input.Status == "" && input.ExternalAccountID == "" {
		return items
	}
	filtered := make([]TransferSummary, 0, len(items))
	for _, item := range items {
		if input.Status != "" && !strings.EqualFold(item.Status, input.Status) {
			continue
		}
		if input.ExternalAccountID != "" && item.ExternalAccountID != input.ExternalAccountID {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func validateCreateExternalAccountInput(input CreateExternalAccountInput) error {
	fields := []util.FieldError{}
	if strings.TrimSpace(input.Description) == "" {
		fields = append(fields, util.FieldError{Field: "description", Message: "is required"})
	}
	if strings.TrimSpace(input.RoutingNumber) == "" {
		fields = append(fields, util.FieldError{Field: "routing_number", Message: "is required"})
	}
	if strings.TrimSpace(input.AccountNumber) == "" {
		fields = append(fields, util.FieldError{Field: "account_number", Message: "is required"})
	}
	if value := strings.TrimSpace(input.AccountHolder); value != "" && !isAllowedValue(value, []string{"business", "individual", "unknown"}) {
		fields = append(fields, util.FieldError{Field: "account_holder", Message: "expected one of business, individual, or unknown"})
	}
	if value := strings.TrimSpace(input.Funding); value != "" && !isAllowedValue(value, []string{"checking", "savings", "general_ledger", "other"}) {
		fields = append(fields, util.FieldError{Field: "funding", Message: "expected one of checking, savings, general_ledger, or other"})
	}
	if len(fields) == 0 {
		return nil
	}
	return &util.ErrorDetail{
		Code:    util.CodeValidationError,
		Message: "Please correct the highlighted external account fields.",
		Fields:  fields,
	}
}

func validateUpdateExternalAccountInput(input UpdateExternalAccountInput) error {
	fields := []util.FieldError{}
	if strings.TrimSpace(input.ExternalAccountID) == "" {
		fields = append(fields, util.FieldError{Field: "external_account_id", Message: "is required"})
	}
	if value := strings.TrimSpace(input.AccountHolder); value != "" && !isAllowedValue(value, []string{"business", "individual"}) {
		fields = append(fields, util.FieldError{Field: "account_holder", Message: "expected one of business or individual"})
	}
	if value := strings.TrimSpace(input.Funding); value != "" && !isAllowedValue(value, []string{"checking", "savings", "general_ledger", "other"}) {
		fields = append(fields, util.FieldError{Field: "funding", Message: "expected one of checking, savings, general_ledger, or other"})
	}
	if value := strings.TrimSpace(input.Status); value != "" && !isAllowedValue(value, []string{"active", "archived"}) {
		fields = append(fields, util.FieldError{Field: "status", Message: "expected one of active or archived"})
	}
	if len(fields) == 0 {
		return nil
	}
	return &util.ErrorDetail{
		Code:    util.CodeValidationError,
		Message: "Please correct the highlighted external account fields.",
		Fields:  fields,
	}
}

func validateCreateCardInput(input CreateCardInput) error {
	fields := []util.FieldError{}
	if strings.TrimSpace(input.AccountID) == "" {
		fields = append(fields, util.FieldError{Field: "account_id", Message: "is required"})
	}
	if len(strings.TrimSpace(input.Description)) > 200 {
		fields = append(fields, util.FieldError{Field: "description", Message: "must be 200 characters or fewer"})
	}
	if input.BillingAddress != nil {
		if strings.TrimSpace(input.BillingAddress.Line1) == "" {
			fields = append(fields, util.FieldError{Field: "billing_address.line1", Message: "is required"})
		}
		if strings.TrimSpace(input.BillingAddress.City) == "" {
			fields = append(fields, util.FieldError{Field: "billing_address.city", Message: "is required"})
		}
		if strings.TrimSpace(input.BillingAddress.State) == "" {
			fields = append(fields, util.FieldError{Field: "billing_address.state", Message: "is required"})
		}
		if strings.TrimSpace(input.BillingAddress.PostalCode) == "" {
			fields = append(fields, util.FieldError{Field: "billing_address.postal_code", Message: "is required"})
		}
	}
	if input.DigitalWallet != nil &&
		strings.TrimSpace(input.DigitalWallet.Email) == "" &&
		strings.TrimSpace(input.DigitalWallet.Phone) == "" &&
		strings.TrimSpace(input.DigitalWallet.DigitalCardProfileID) == "" {
		fields = append(fields, util.FieldError{Field: "digital_wallet", Message: "provide email, phone, or digital_card_profile_id"})
	}
	if len(fields) == 0 {
		return nil
	}
	return &util.ErrorDetail{
		Code:    util.CodeValidationError,
		Message: "Please correct the highlighted card fields.",
		Fields:  fields,
	}
}

func isAllowedValue(value string, allowed []string) bool {
	for _, option := range allowed {
		if value == option {
			return true
		}
	}
	return false
}
