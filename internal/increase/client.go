package increasex

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	increase "github.com/increase/increase-go"
	"github.com/increase/increase-go/option"
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

func (c *Client) CreateInternalTransfer(ctx context.Context, params increase.AccountTransferNewParams, idempotencyKey string) (APIResult[*increase.AccountTransfer], error) {
	var resp *http.Response
	transfer, err := c.raw.AccountTransfers.New(ctx, params, c.requestOptions(idempotencyKey, &resp)...)
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

func (c *Client) CreateRTPTransfer(ctx context.Context, params increase.RealTimePaymentsTransferNewParams, idempotencyKey string) (APIResult[*increase.RealTimePaymentsTransfer], error) {
	var resp *http.Response
	transfer, err := c.raw.RealTimePaymentsTransfers.New(ctx, params, c.requestOptions(idempotencyKey, &resp)...)
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

func (c *Client) CreateFedNowTransfer(ctx context.Context, params FedNowTransferNewParams, idempotencyKey string) (APIResult[*FedNowTransfer], error) {
	var resp *http.Response
	var transfer FedNowTransfer
	if err := c.raw.Post(ctx, "fednow_transfers", params, &transfer, c.requestOptions(idempotencyKey, &resp)...); err != nil {
		return APIResult[*FedNowTransfer]{}, err
	}
	return APIResult[*FedNowTransfer]{Data: &transfer, RequestID: requestIDFrom(resp)}, nil
}

func ParseSince(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, value)
}
