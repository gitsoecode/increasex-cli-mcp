package increasex

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	increase "github.com/Increase/increase-go"
	"github.com/Increase/increase-go/option"
	"github.com/jessevaughan/increasex/internal/config"
	"github.com/jessevaughan/increasex/internal/util"
)

type Client struct {
	raw *increase.Client
}

func NewClient(apiKey, env string) *Client {
	opts := []option.RequestOption{option.WithAPIKey(apiKey)}
	if env == config.EnvSandbox {
		opts = append(opts, option.WithEnvironmentSandbox())
	}
	return &Client{raw: increase.NewClient(opts...)}
}

type APIResult[T any] struct {
	Data      T
	RequestID string
}

func (c *Client) requestOptions(idempotencyKey string, response **http.Response) []option.RequestOption {
	opts := []option.RequestOption{}
	if response != nil {
		opts = append(opts, option.WithResponseInto(response))
	}
	if idempotencyKey != "" {
		opts = append(opts, option.WithHeader("Idempotency-Key", idempotencyKey))
	}
	return opts
}

func requestIDFrom(resp *http.Response) string {
	if resp == nil {
		return ""
	}
	if rid := resp.Header.Get("X-Request-Id"); rid != "" {
		return rid
	}
	return resp.Header.Get("X-Request-ID")
}

func WrapError(err error) *util.ErrorDetail {
	if err == nil {
		return nil
	}
	var apiErr *increase.Error
	if errors.As(err, &apiErr) {
		details := map[string]any{
			"type":   string(apiErr.Type),
			"reason": string(apiErr.Reason),
		}
		if apiErr.Detail != "" {
			details["detail"] = apiErr.Detail
		}
		switch apiErr.Status {
		case increase.ErrorStatus401, increase.ErrorStatus403:
			return util.NewError(util.CodeAuthError, apiErr.Title, details, false)
		case increase.ErrorStatus404:
			return util.NewError(util.CodeNotFound, apiErr.Title, details, false)
		case increase.ErrorStatus409:
			return util.NewError(util.CodeIdempotencyConflict, apiErr.Title, details, false)
		case increase.ErrorStatus429:
			return util.NewError(util.CodeRateLimited, apiErr.Title, details, true)
		case increase.ErrorStatus400:
			return util.NewError(util.CodeValidationError, apiErr.Title, details, false)
		default:
			return util.NewError(util.CodeAPIError, apiErr.Title, details, apiErr.Status >= 500)
		}
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return util.NewError(util.CodeNetworkError, err.Error(), nil, true)
	}
	if strings.Contains(strings.ToLower(err.Error()), "confirmation") {
		return util.NewError(util.CodeConfirmationInvalid, err.Error(), nil, false)
	}
	return util.NewError(util.CodeUnknownError, err.Error(), nil, false)
}

func (c *Client) ListAccounts(ctx context.Context, params increase.AccountListParams) (APIResult[[]increase.Account], error) {
	var resp *http.Response
	page, err := c.raw.Accounts.List(ctx, params, c.requestOptions("", &resp)...)
	if err != nil {
		return APIResult[[]increase.Account]{}, err
	}
	return APIResult[[]increase.Account]{Data: page.Data, RequestID: requestIDFrom(resp)}, nil
}

func (c *Client) GetAccount(ctx context.Context, accountID string) (APIResult[*increase.Account], error) {
	var resp *http.Response
	account, err := c.raw.Accounts.Get(ctx, accountID, c.requestOptions("", &resp)...)
	if err != nil {
		return APIResult[*increase.Account]{}, err
	}
	return APIResult[*increase.Account]{Data: account, RequestID: requestIDFrom(resp)}, nil
}

func (c *Client) GetBalance(ctx context.Context, accountID string) (APIResult[*increase.BalanceLookup], error) {
	var resp *http.Response
	balance, err := c.raw.Accounts.Balance(ctx, accountID, increase.AccountBalanceParams{}, c.requestOptions("", &resp)...)
	if err != nil {
		return APIResult[*increase.BalanceLookup]{}, err
	}
	return APIResult[*increase.BalanceLookup]{Data: balance, RequestID: requestIDFrom(resp)}, nil
}

func (c *Client) CloseAccount(ctx context.Context, accountID, idempotencyKey string) (APIResult[*increase.Account], error) {
	var resp *http.Response
	account, err := c.raw.Accounts.Close(ctx, accountID, c.requestOptions(idempotencyKey, &resp)...)
	if err != nil {
		return APIResult[*increase.Account]{}, err
	}
	return APIResult[*increase.Account]{Data: account, RequestID: requestIDFrom(resp)}, nil
}

func (c *Client) CreateAccount(ctx context.Context, params increase.AccountNewParams, idempotencyKey string) (APIResult[*increase.Account], error) {
	var resp *http.Response
	account, err := c.raw.Accounts.New(ctx, params, c.requestOptions(idempotencyKey, &resp)...)
	if err != nil {
		return APIResult[*increase.Account]{}, err
	}
	return APIResult[*increase.Account]{Data: account, RequestID: requestIDFrom(resp)}, nil
}

func (c *Client) CreateAccountNumber(ctx context.Context, params increase.AccountNumberNewParams, idempotencyKey string) (APIResult[*increase.AccountNumber], error) {
	var resp *http.Response
	number, err := c.raw.AccountNumbers.New(ctx, params, c.requestOptions(idempotencyKey, &resp)...)
	if err != nil {
		return APIResult[*increase.AccountNumber]{}, err
	}
	return APIResult[*increase.AccountNumber]{Data: number, RequestID: requestIDFrom(resp)}, nil
}

func (c *Client) ListTransactions(ctx context.Context, params increase.TransactionListParams) (APIResult[[]increase.Transaction], error) {
	var resp *http.Response
	page, err := c.raw.Transactions.List(ctx, params, c.requestOptions("", &resp)...)
	if err != nil {
		return APIResult[[]increase.Transaction]{}, err
	}
	return APIResult[[]increase.Transaction]{Data: page.Data, RequestID: requestIDFrom(resp)}, nil
}

func (c *Client) ListCards(ctx context.Context, params increase.CardListParams) (APIResult[[]increase.Card], error) {
	var resp *http.Response
	page, err := c.raw.Cards.List(ctx, params, c.requestOptions("", &resp)...)
	if err != nil {
		return APIResult[[]increase.Card]{}, err
	}
	return APIResult[[]increase.Card]{Data: page.Data, RequestID: requestIDFrom(resp)}, nil
}

func (c *Client) GetCard(ctx context.Context, cardID string) (APIResult[*increase.Card], error) {
	var resp *http.Response
	card, err := c.raw.Cards.Get(ctx, cardID, c.requestOptions("", &resp)...)
	if err != nil {
		return APIResult[*increase.Card]{}, err
	}
	return APIResult[*increase.Card]{Data: card, RequestID: requestIDFrom(resp)}, nil
}

func (c *Client) CreateCard(ctx context.Context, params increase.CardNewParams, idempotencyKey string) (APIResult[*increase.Card], error) {
	var resp *http.Response
	card, err := c.raw.Cards.New(ctx, params, c.requestOptions(idempotencyKey, &resp)...)
	if err != nil {
		return APIResult[*increase.Card]{}, err
	}
	return APIResult[*increase.Card]{Data: card, RequestID: requestIDFrom(resp)}, nil
}

func (c *Client) CreateCardRaw(ctx context.Context, body map[string]any, idempotencyKey string) (APIResult[*increase.Card], error) {
	var resp *http.Response
	var card increase.Card
	if err := c.raw.Post(ctx, "cards", body, &card, c.requestOptions(idempotencyKey, &resp)...); err != nil {
		return APIResult[*increase.Card]{}, err
	}
	return APIResult[*increase.Card]{Data: &card, RequestID: requestIDFrom(resp)}, nil
}

func (c *Client) GetCardDetails(ctx context.Context, cardID string) (APIResult[*increase.CardDetails], error) {
	var resp *http.Response
	details, err := c.raw.Cards.Details(ctx, cardID, c.requestOptions("", &resp)...)
	if err != nil {
		return APIResult[*increase.CardDetails]{}, err
	}
	return APIResult[*increase.CardDetails]{Data: details, RequestID: requestIDFrom(resp)}, nil
}

func (c *Client) CreateCardDetailsIframe(ctx context.Context, cardID string, params increase.CardNewDetailsIframeParams) (APIResult[*increase.CardIframeURL], error) {
	var resp *http.Response
	iframe, err := c.raw.Cards.NewDetailsIframe(ctx, cardID, params, c.requestOptions("", &resp)...)
	if err != nil {
		return APIResult[*increase.CardIframeURL]{}, err
	}
	return APIResult[*increase.CardIframeURL]{Data: iframe, RequestID: requestIDFrom(resp)}, nil
}

func (c *Client) UpdateCardPIN(ctx context.Context, cardID string, params increase.CardUpdatePinParams) (APIResult[*increase.CardDetails], error) {
	var resp *http.Response
	details, err := c.raw.Cards.UpdatePin(ctx, cardID, params, c.requestOptions("", &resp)...)
	if err != nil {
		return APIResult[*increase.CardDetails]{}, err
	}
	return APIResult[*increase.CardDetails]{Data: details, RequestID: requestIDFrom(resp)}, nil
}

func (c *Client) ListExternalAccounts(ctx context.Context, params increase.ExternalAccountListParams) (APIResult[[]increase.ExternalAccount], error) {
	var resp *http.Response
	page, err := c.raw.ExternalAccounts.List(ctx, params, c.requestOptions("", &resp)...)
	if err != nil {
		return APIResult[[]increase.ExternalAccount]{}, err
	}
	return APIResult[[]increase.ExternalAccount]{Data: page.Data, RequestID: requestIDFrom(resp)}, nil
}

func (c *Client) GetExternalAccount(ctx context.Context, externalAccountID string) (APIResult[*increase.ExternalAccount], error) {
	var resp *http.Response
	account, err := c.raw.ExternalAccounts.Get(ctx, externalAccountID, c.requestOptions("", &resp)...)
	if err != nil {
		return APIResult[*increase.ExternalAccount]{}, err
	}
	return APIResult[*increase.ExternalAccount]{Data: account, RequestID: requestIDFrom(resp)}, nil
}

func (c *Client) CreateExternalAccount(ctx context.Context, params increase.ExternalAccountNewParams, idempotencyKey string) (APIResult[*increase.ExternalAccount], error) {
	var resp *http.Response
	account, err := c.raw.ExternalAccounts.New(ctx, params, c.requestOptions(idempotencyKey, &resp)...)
	if err != nil {
		return APIResult[*increase.ExternalAccount]{}, err
	}
	return APIResult[*increase.ExternalAccount]{Data: account, RequestID: requestIDFrom(resp)}, nil
}

func (c *Client) UpdateExternalAccount(ctx context.Context, externalAccountID string, params increase.ExternalAccountUpdateParams, idempotencyKey string) (APIResult[*increase.ExternalAccount], error) {
	var resp *http.Response
	account, err := c.raw.ExternalAccounts.Update(ctx, externalAccountID, params, c.requestOptions(idempotencyKey, &resp)...)
	if err != nil {
		return APIResult[*increase.ExternalAccount]{}, err
	}
	return APIResult[*increase.ExternalAccount]{Data: account, RequestID: requestIDFrom(resp)}, nil
}

func (c *Client) CreateInternalTransfer(ctx context.Context, params increase.AccountTransferNewParams, idempotencyKey string) (APIResult[*increase.AccountTransfer], error) {
	var resp *http.Response
	transfer, err := c.raw.AccountTransfers.New(ctx, params, c.requestOptions(idempotencyKey, &resp)...)
	if err != nil {
		return APIResult[*increase.AccountTransfer]{}, err
	}
	return APIResult[*increase.AccountTransfer]{Data: transfer, RequestID: requestIDFrom(resp)}, nil
}

func (c *Client) ListInternalTransfers(ctx context.Context, params increase.AccountTransferListParams) (APIResult[[]increase.AccountTransfer], error) {
	var resp *http.Response
	page, err := c.raw.AccountTransfers.List(ctx, params, c.requestOptions("", &resp)...)
	if err != nil {
		return APIResult[[]increase.AccountTransfer]{}, err
	}
	return APIResult[[]increase.AccountTransfer]{Data: page.Data, RequestID: requestIDFrom(resp)}, nil
}

func (c *Client) GetInternalTransfer(ctx context.Context, transferID string) (APIResult[*increase.AccountTransfer], error) {
	var resp *http.Response
	transfer, err := c.raw.AccountTransfers.Get(ctx, transferID, c.requestOptions("", &resp)...)
	if err != nil {
		return APIResult[*increase.AccountTransfer]{}, err
	}
	return APIResult[*increase.AccountTransfer]{Data: transfer, RequestID: requestIDFrom(resp)}, nil
}

func (c *Client) ApproveInternalTransfer(ctx context.Context, transferID string) (APIResult[*increase.AccountTransfer], error) {
	var resp *http.Response
	transfer, err := c.raw.AccountTransfers.Approve(ctx, transferID, c.requestOptions("", &resp)...)
	if err != nil {
		return APIResult[*increase.AccountTransfer]{}, err
	}
	return APIResult[*increase.AccountTransfer]{Data: transfer, RequestID: requestIDFrom(resp)}, nil
}

func (c *Client) CancelInternalTransfer(ctx context.Context, transferID string) (APIResult[*increase.AccountTransfer], error) {
	var resp *http.Response
	transfer, err := c.raw.AccountTransfers.Cancel(ctx, transferID, c.requestOptions("", &resp)...)
	if err != nil {
		return APIResult[*increase.AccountTransfer]{}, err
	}
	return APIResult[*increase.AccountTransfer]{Data: transfer, RequestID: requestIDFrom(resp)}, nil
}

func (c *Client) CreateACHTransfer(ctx context.Context, params increase.ACHTransferNewParams, idempotencyKey string) (APIResult[*increase.ACHTransfer], error) {
	var resp *http.Response
	transfer, err := c.raw.ACHTransfers.New(ctx, params, c.requestOptions(idempotencyKey, &resp)...)
	if err != nil {
		return APIResult[*increase.ACHTransfer]{}, err
	}
	return APIResult[*increase.ACHTransfer]{Data: transfer, RequestID: requestIDFrom(resp)}, nil
}

func (c *Client) ListACHTransfers(ctx context.Context, params increase.ACHTransferListParams) (APIResult[[]increase.ACHTransfer], error) {
	var resp *http.Response
	page, err := c.raw.ACHTransfers.List(ctx, params, c.requestOptions("", &resp)...)
	if err != nil {
		return APIResult[[]increase.ACHTransfer]{}, err
	}
	return APIResult[[]increase.ACHTransfer]{Data: page.Data, RequestID: requestIDFrom(resp)}, nil
}

func (c *Client) GetACHTransfer(ctx context.Context, transferID string) (APIResult[*increase.ACHTransfer], error) {
	var resp *http.Response
	transfer, err := c.raw.ACHTransfers.Get(ctx, transferID, c.requestOptions("", &resp)...)
	if err != nil {
		return APIResult[*increase.ACHTransfer]{}, err
	}
	return APIResult[*increase.ACHTransfer]{Data: transfer, RequestID: requestIDFrom(resp)}, nil
}

func (c *Client) ApproveACHTransfer(ctx context.Context, transferID string) (APIResult[*increase.ACHTransfer], error) {
	var resp *http.Response
	transfer, err := c.raw.ACHTransfers.Approve(ctx, transferID, c.requestOptions("", &resp)...)
	if err != nil {
		return APIResult[*increase.ACHTransfer]{}, err
	}
	return APIResult[*increase.ACHTransfer]{Data: transfer, RequestID: requestIDFrom(resp)}, nil
}

func (c *Client) CancelACHTransfer(ctx context.Context, transferID string) (APIResult[*increase.ACHTransfer], error) {
	var resp *http.Response
	transfer, err := c.raw.ACHTransfers.Cancel(ctx, transferID, c.requestOptions("", &resp)...)
	if err != nil {
		return APIResult[*increase.ACHTransfer]{}, err
	}
	return APIResult[*increase.ACHTransfer]{Data: transfer, RequestID: requestIDFrom(resp)}, nil
}

func (c *Client) CreateRTPTransfer(ctx context.Context, params increase.RealTimePaymentsTransferNewParams, idempotencyKey string) (APIResult[*increase.RealTimePaymentsTransfer], error) {
	var resp *http.Response
	transfer, err := c.raw.RealTimePaymentsTransfers.New(ctx, params, c.requestOptions(idempotencyKey, &resp)...)
	if err != nil {
		return APIResult[*increase.RealTimePaymentsTransfer]{}, err
	}
	return APIResult[*increase.RealTimePaymentsTransfer]{Data: transfer, RequestID: requestIDFrom(resp)}, nil
}

func (c *Client) ListRTPTransfers(ctx context.Context, params increase.RealTimePaymentsTransferListParams) (APIResult[[]increase.RealTimePaymentsTransfer], error) {
	var resp *http.Response
	page, err := c.raw.RealTimePaymentsTransfers.List(ctx, params, c.requestOptions("", &resp)...)
	if err != nil {
		return APIResult[[]increase.RealTimePaymentsTransfer]{}, err
	}
	return APIResult[[]increase.RealTimePaymentsTransfer]{Data: page.Data, RequestID: requestIDFrom(resp)}, nil
}

func (c *Client) GetRTPTransfer(ctx context.Context, transferID string) (APIResult[*increase.RealTimePaymentsTransfer], error) {
	var resp *http.Response
	transfer, err := c.raw.RealTimePaymentsTransfers.Get(ctx, transferID, c.requestOptions("", &resp)...)
	if err != nil {
		return APIResult[*increase.RealTimePaymentsTransfer]{}, err
	}
	return APIResult[*increase.RealTimePaymentsTransfer]{Data: transfer, RequestID: requestIDFrom(resp)}, nil
}

func (c *Client) ApproveRTPTransfer(ctx context.Context, transferID string) (APIResult[*increase.RealTimePaymentsTransfer], error) {
	var resp *http.Response
	transfer, err := c.raw.RealTimePaymentsTransfers.Approve(ctx, transferID, c.requestOptions("", &resp)...)
	if err != nil {
		return APIResult[*increase.RealTimePaymentsTransfer]{}, err
	}
	return APIResult[*increase.RealTimePaymentsTransfer]{Data: transfer, RequestID: requestIDFrom(resp)}, nil
}

func (c *Client) CancelRTPTransfer(ctx context.Context, transferID string) (APIResult[*increase.RealTimePaymentsTransfer], error) {
	var resp *http.Response
	transfer, err := c.raw.RealTimePaymentsTransfers.Cancel(ctx, transferID, c.requestOptions("", &resp)...)
	if err != nil {
		return APIResult[*increase.RealTimePaymentsTransfer]{}, err
	}
	return APIResult[*increase.RealTimePaymentsTransfer]{Data: transfer, RequestID: requestIDFrom(resp)}, nil
}

func (c *Client) CreateWireTransfer(ctx context.Context, params increase.WireTransferNewParams, idempotencyKey string) (APIResult[*increase.WireTransfer], error) {
	var resp *http.Response
	transfer, err := c.raw.WireTransfers.New(ctx, params, c.requestOptions(idempotencyKey, &resp)...)
	if err != nil {
		return APIResult[*increase.WireTransfer]{}, err
	}
	return APIResult[*increase.WireTransfer]{Data: transfer, RequestID: requestIDFrom(resp)}, nil
}

func (c *Client) ListWireTransfers(ctx context.Context, params increase.WireTransferListParams) (APIResult[[]increase.WireTransfer], error) {
	var resp *http.Response
	page, err := c.raw.WireTransfers.List(ctx, params, c.requestOptions("", &resp)...)
	if err != nil {
		return APIResult[[]increase.WireTransfer]{}, err
	}
	return APIResult[[]increase.WireTransfer]{Data: page.Data, RequestID: requestIDFrom(resp)}, nil
}

func (c *Client) GetWireTransfer(ctx context.Context, transferID string) (APIResult[*increase.WireTransfer], error) {
	var resp *http.Response
	transfer, err := c.raw.WireTransfers.Get(ctx, transferID, c.requestOptions("", &resp)...)
	if err != nil {
		return APIResult[*increase.WireTransfer]{}, err
	}
	return APIResult[*increase.WireTransfer]{Data: transfer, RequestID: requestIDFrom(resp)}, nil
}

func (c *Client) ApproveWireTransfer(ctx context.Context, transferID string) (APIResult[*increase.WireTransfer], error) {
	var resp *http.Response
	transfer, err := c.raw.WireTransfers.Approve(ctx, transferID, c.requestOptions("", &resp)...)
	if err != nil {
		return APIResult[*increase.WireTransfer]{}, err
	}
	return APIResult[*increase.WireTransfer]{Data: transfer, RequestID: requestIDFrom(resp)}, nil
}

func (c *Client) CancelWireTransfer(ctx context.Context, transferID string) (APIResult[*increase.WireTransfer], error) {
	var resp *http.Response
	transfer, err := c.raw.WireTransfers.Cancel(ctx, transferID, c.requestOptions("", &resp)...)
	if err != nil {
		return APIResult[*increase.WireTransfer]{}, err
	}
	return APIResult[*increase.WireTransfer]{Data: transfer, RequestID: requestIDFrom(resp)}, nil
}

func (c *Client) CreateFedNowTransfer(ctx context.Context, params increase.FednowTransferNewParams, idempotencyKey string) (APIResult[*increase.FednowTransfer], error) {
	var resp *http.Response
	transfer, err := c.raw.FednowTransfers.New(ctx, params, c.requestOptions(idempotencyKey, &resp)...)
	if err != nil {
		return APIResult[*increase.FednowTransfer]{}, err
	}
	return APIResult[*increase.FednowTransfer]{Data: transfer, RequestID: requestIDFrom(resp)}, nil
}

func (c *Client) ListFedNowTransfers(ctx context.Context, params increase.FednowTransferListParams) (APIResult[[]increase.FednowTransfer], error) {
	var resp *http.Response
	page, err := c.raw.FednowTransfers.List(ctx, params, c.requestOptions("", &resp)...)
	if err != nil {
		return APIResult[[]increase.FednowTransfer]{}, err
	}
	return APIResult[[]increase.FednowTransfer]{Data: page.Data, RequestID: requestIDFrom(resp)}, nil
}

func (c *Client) GetFedNowTransfer(ctx context.Context, transferID string) (APIResult[*increase.FednowTransfer], error) {
	var resp *http.Response
	transfer, err := c.raw.FednowTransfers.Get(ctx, transferID, c.requestOptions("", &resp)...)
	if err != nil {
		return APIResult[*increase.FednowTransfer]{}, err
	}
	return APIResult[*increase.FednowTransfer]{Data: transfer, RequestID: requestIDFrom(resp)}, nil
}

func (c *Client) ApproveFedNowTransfer(ctx context.Context, transferID string) (APIResult[*increase.FednowTransfer], error) {
	var resp *http.Response
	transfer, err := c.raw.FednowTransfers.Approve(ctx, transferID, c.requestOptions("", &resp)...)
	if err != nil {
		return APIResult[*increase.FednowTransfer]{}, err
	}
	return APIResult[*increase.FednowTransfer]{Data: transfer, RequestID: requestIDFrom(resp)}, nil
}

func (c *Client) CancelFedNowTransfer(ctx context.Context, transferID string) (APIResult[*increase.FednowTransfer], error) {
	var resp *http.Response
	transfer, err := c.raw.FednowTransfers.Cancel(ctx, transferID, c.requestOptions("", &resp)...)
	if err != nil {
		return APIResult[*increase.FednowTransfer]{}, err
	}
	return APIResult[*increase.FednowTransfer]{Data: transfer, RequestID: requestIDFrom(resp)}, nil
}

func ParseSince(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, value)
}
