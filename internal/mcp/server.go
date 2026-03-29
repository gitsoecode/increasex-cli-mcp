package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/jessevaughan/increasex/internal/app"
	"github.com/jessevaughan/increasex/internal/auth"
	increasex "github.com/jessevaughan/increasex/internal/increase"
	"github.com/jessevaughan/increasex/internal/util"
)

type Options struct {
	Profile     string
	Environment string
	APIKey      string
	Debug       bool
}

type Server struct {
	services app.Services
	options  Options
	reader   *bufio.Reader
	writer   io.Writer
	mode     transportMode
}

type transportMode string

const (
	transportModeUnknown       transportMode = ""
	transportModeContentLength transportMode = "content-length"
	transportModeJSONLine      transportMode = "json-line"
)

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type initializeParams struct {
	ProtocolVersion string `json:"protocolVersion"`
}

type response struct {
	JSONRPC string     `json:"jsonrpc"`
	ID      any        `json:"id,omitempty"`
	Result  any        `json:"result,omitempty"`
	Error   *respError `json:"error,omitempty"`
}

type respError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type toolCallParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

type toolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

func NewServer(services app.Services, options Options) *Server {
	return &Server{
		services: services,
		options:  options,
		reader:   bufio.NewReader(os.Stdin),
		writer:   os.Stdout,
	}
}

func (s *Server) Serve(ctx context.Context) error {
	s.debugf("starting stdio MCP server")
	for {
		payload, err := s.readFrame()
		if err == io.EOF {
			s.debugf("stdin closed")
			return nil
		}
		if err != nil {
			s.debugf("readFrame error: %v", err)
			return err
		}
		var req request
		if err := json.Unmarshal(payload, &req); err != nil {
			if s.options.Debug {
				fmt.Fprintln(os.Stderr, "mcp decode error:", err)
			}
			continue
		}
		s.debugf("received method=%s", req.Method)
		resp := s.handle(ctx, req)
		if resp == nil {
			continue
		}
		if err := s.writeFrame(resp); err != nil {
			s.debugf("writeFrame error: %v", err)
			return err
		}
	}
}

func (s *Server) handle(ctx context.Context, req request) *response {
	switch req.Method {
	case "initialize":
		protocolVersion := "2024-11-05"
		if len(req.Params) > 0 {
			var params initializeParams
			if err := json.Unmarshal(req.Params, &params); err == nil && strings.TrimSpace(params.ProtocolVersion) != "" {
				protocolVersion = params.ProtocolVersion
			}
		}
		s.debugf("initialize protocolVersion=%s", protocolVersion)
		return &response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]any{
				"protocolVersion": protocolVersion,
				"serverInfo": map[string]any{
					"name":    "increasex",
					"version": "0.1.0",
				},
				"capabilities": map[string]any{
					"tools": map[string]any{
						"listChanged": false,
					},
				},
			},
		}
	case "notifications/initialized":
		return nil
	case "ping":
		return &response{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{}}
	case "tools/list":
		return &response{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{"tools": s.tools()}}
	case "tools/call":
		var params toolCallParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return &response{JSONRPC: "2.0", ID: req.ID, Error: &respError{Code: -32602, Message: err.Error()}}
		}
		result, isErr := s.callTool(ctx, params.Name, params.Arguments)
		return &response{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{
			"content":           []map[string]any{{"type": "text", "text": toJSONString(result)}},
			"structuredContent": result,
			"isError":           isErr,
		}}
	default:
		if req.ID == nil || strings.HasPrefix(req.Method, "notifications/") {
			return nil
		}
		return &response{JSONRPC: "2.0", ID: req.ID, Error: &respError{Code: -32601, Message: "method not found"}}
	}
}

func (s *Server) callTool(ctx context.Context, name string, args map[string]any) (any, bool) {
	session, api, err := s.services.ResolveSession(ctx, auth.ResolveInput{
		ProfileName: s.options.Profile,
		Environment: s.options.Environment,
		APIKey:      s.options.APIKey,
	})
	if err != nil {
		return util.Fail(increasex.WrapError(err), ""), true
	}
	switch name {
	case "list_accounts":
		status := asString(args["status"])
		limit := asInt64(args["limit"], 20)
		cursor := asString(args["cursor"])
		accounts, requestID, err := s.services.ListAccounts(ctx, api, status, limit, cursor)
		return envelope(map[string]any{"accounts": accounts}, requestID, err)
	case "resolve_account":
		query := asString(args["query"])
		limit := asInt64(args["limit"], 10)
		matches, requestID, err := s.services.ResolveAccount(ctx, api, query, limit)
		return envelope(map[string]any{"matches": matches}, requestID, err)
	case "get_balance":
		result, requestID, err := s.services.GetBalance(ctx, api, asString(args["account_id"]))
		return envelope(result, requestID, err)
	case "list_recent_transactions":
		result, requestID, err := s.services.ListRecentTransactions(
			ctx,
			api,
			asString(args["account_id"]),
			asString(args["since"]),
			asString(args["cursor"]),
			asInt64(args["limit"], 20),
			asStringSlice(args["categories"]),
		)
		return envelope(map[string]any{"transactions": result}, requestID, err)
	case "list_cards":
		result, requestID, err := s.services.ListCards(
			ctx,
			api,
			asString(args["account_id"]),
			asString(args["status"]),
			asString(args["cursor"]),
			asInt64(args["limit"], 20),
		)
		return envelope(map[string]any{"cards": result}, requestID, err)
	case "retrieve_card_details":
		result, requestID, err := s.services.RetrieveCardDetails(ctx, api, asString(args["card_id"]))
		return envelope(result, requestID, err)
	case "retrieve_card_sensitive_details":
		result, requestID, err := s.services.RetrieveSensitiveCardDetails(ctx, api, asString(args["card_id"]))
		return envelope(result, requestID, err)
	case "create_card_details_iframe":
		result, requestID, err := s.services.CreateCardDetailsIframe(ctx, api, app.CreateCardDetailsIframeInput{
			CardID:         asString(args["card_id"]),
			PhysicalCardID: asString(args["physical_card_id"]),
		})
		return envelope(result, requestID, err)
	case "list_external_accounts":
		result, requestID, err := s.services.ListExternalAccounts(
			ctx,
			api,
			asString(args["status"]),
			asString(args["cursor"]),
			asInt64(args["limit"], 20),
		)
		return envelope(map[string]any{"external_accounts": result}, requestID, err)
	case "retrieve_external_account":
		result, requestID, err := s.services.RetrieveExternalAccount(ctx, api, asString(args["external_account_id"]))
		return envelope(result, requestID, err)
	case "create_account":
		var input app.CreateAccountInput
		decodeArgs(args, &input)
		return s.handleWrite(ctx, *session, api, app.IsDryRun(input.DryRun), input.ConfirmationToken,
			func() (*app.PreviewResult, error) { return s.services.PreviewCreateAccount(*session, input) },
			func() (any, string, error) { return s.services.ExecuteCreateAccount(ctx, api, *session, input) })
	case "close_account":
		var input app.CloseAccountInput
		decodeArgs(args, &input)
		return s.handleWrite(ctx, *session, api, app.IsDryRun(input.DryRun), input.ConfirmationToken,
			func() (*app.PreviewResult, error) { return s.services.PreviewCloseAccount(*session, input) },
			func() (any, string, error) { return s.services.ExecuteCloseAccount(ctx, api, *session, input) })
	case "create_account_number":
		var input app.CreateAccountNumberInput
		decodeArgs(args, &input)
		return s.handleWrite(ctx, *session, api, app.IsDryRun(input.DryRun), input.ConfirmationToken,
			func() (*app.PreviewResult, error) { return s.services.PreviewCreateAccountNumber(*session, input) },
			func() (any, string, error) { return s.services.ExecuteCreateAccountNumber(ctx, api, *session, input) })
	case "create_account_transfer", "move_money_internal":
		var input app.MoveMoneyInternalInput
		decodeArgs(args, &input)
		return s.handleWrite(ctx, *session, api, app.IsDryRun(input.DryRun), input.ConfirmationToken,
			func() (*app.PreviewResult, error) { return s.services.PreviewInternalTransfer(*session, input) },
			func() (any, string, error) { return s.services.ExecuteInternalTransfer(ctx, api, *session, input) })
	case "create_card":
		var input app.CreateCardInput
		decodeArgs(args, &input)
		return s.handleWrite(ctx, *session, api, app.IsDryRun(input.DryRun), input.ConfirmationToken,
			func() (*app.PreviewResult, error) { return s.services.PreviewCreateCard(*session, input) },
			func() (any, string, error) { return s.services.ExecuteCreateCard(ctx, api, *session, input) })
	case "create_ach_transfer", "move_money_external_ach":
		var input app.ACHTransferInput
		decodeArgs(args, &input)
		return s.handleWrite(ctx, *session, api, app.IsDryRun(input.DryRun), input.ConfirmationToken,
			func() (*app.PreviewResult, error) { return s.services.PreviewExternalACH(*session, input) },
			func() (any, string, error) { return s.services.ExecuteExternalACH(ctx, api, *session, input) })
	case "create_real_time_payments_transfer", "move_money_external_rtp":
		var input app.RTPTransferInput
		decodeArgs(args, &input)
		return s.handleWrite(ctx, *session, api, app.IsDryRun(input.DryRun), input.ConfirmationToken,
			func() (*app.PreviewResult, error) { return s.services.PreviewExternalRTP(*session, input) },
			func() (any, string, error) { return s.services.ExecuteExternalRTP(ctx, api, *session, input) })
	case "create_fednow_transfer", "move_money_external_fednow":
		var input app.FedNowTransferInput
		decodeArgs(args, &input)
		return s.handleWrite(ctx, *session, api, app.IsDryRun(input.DryRun), input.ConfirmationToken,
			func() (*app.PreviewResult, error) { return s.services.PreviewExternalFedNow(*session, input) },
			func() (any, string, error) { return s.services.ExecuteExternalFedNow(ctx, api, *session, input) })
	case "create_wire_transfer", "move_money_external_wire":
		var input app.WireTransferInput
		decodeArgs(args, &input)
		return s.handleWrite(ctx, *session, api, app.IsDryRun(input.DryRun), input.ConfirmationToken,
			func() (*app.PreviewResult, error) { return s.services.PreviewExternalWire(*session, input) },
			func() (any, string, error) { return s.services.ExecuteExternalWire(ctx, api, *session, input) })
	case "create_external_account":
		var input app.CreateExternalAccountInput
		decodeArgs(args, &input)
		return s.handleWrite(ctx, *session, api, app.IsDryRun(input.DryRun), input.ConfirmationToken,
			func() (*app.PreviewResult, error) { return s.services.PreviewCreateExternalAccount(*session, input) },
			func() (any, string, error) { return s.services.ExecuteCreateExternalAccount(ctx, api, *session, input) })
	case "update_external_account":
		var input app.UpdateExternalAccountInput
		decodeArgs(args, &input)
		return s.handleWrite(ctx, *session, api, app.IsDryRun(input.DryRun), input.ConfirmationToken,
			func() (*app.PreviewResult, error) { return s.services.PreviewUpdateExternalAccount(*session, input) },
			func() (any, string, error) { return s.services.ExecuteUpdateExternalAccount(ctx, api, *session, input) })
	case "update_card_pin":
		var input app.UpdateCardPINInput
		decodeArgs(args, &input)
		return s.handleWrite(ctx, *session, api, app.IsDryRun(input.DryRun), input.ConfirmationToken,
			func() (*app.PreviewResult, error) { return s.services.PreviewUpdateCardPIN(*session, input) },
			func() (any, string, error) { return s.services.ExecuteUpdateCardPIN(ctx, api, *session, input) })
	case "list_transfers":
		result, requestID, err := s.services.ListTransfers(ctx, api, app.ListTransfersInput{
			Rail:              asString(args["rail"]),
			AccountID:         asString(args["account_id"]),
			ExternalAccountID: asString(args["external_account_id"]),
			Status:            asString(args["status"]),
			Since:             asString(args["since"]),
			Cursor:            asString(args["cursor"]),
			Limit:             asInt64(args["limit"], 20),
		})
		return envelope(map[string]any{"transfers": result}, requestID, err)
	case "retrieve_transfer":
		result, requestID, err := s.services.RetrieveTransfer(ctx, api, asString(args["rail"]), asString(args["transfer_id"]))
		return envelope(result, requestID, err)
	case "list_transfer_queue":
		result, requestID, err := s.services.ListTransferQueue(ctx, api, asString(args["rail"]), asInt64(args["limit"], 20))
		return envelope(map[string]any{"transfers": result}, requestID, err)
	case "approve_transfer":
		var input app.TransferActionInput
		decodeArgs(args, &input)
		return s.handleWrite(ctx, *session, api, app.IsDryRun(input.DryRun), input.ConfirmationToken,
			func() (*app.PreviewResult, error) { return s.services.PreviewApproveTransfer(*session, input) },
			func() (any, string, error) { return s.services.ExecuteApproveTransfer(ctx, api, *session, input) })
	case "cancel_transfer":
		var input app.TransferActionInput
		decodeArgs(args, &input)
		return s.handleWrite(ctx, *session, api, app.IsDryRun(input.DryRun), input.ConfirmationToken,
			func() (*app.PreviewResult, error) { return s.services.PreviewCancelTransfer(*session, input) },
			func() (any, string, error) { return s.services.ExecuteCancelTransfer(ctx, api, *session, input) })
	case "describe_capabilities":
		return envelope(map[string]any{
			"accounts":          []string{"list_accounts", "resolve_account", "get_balance", "create_account", "close_account", "create_account_number"},
			"transactions":      []string{"list_recent_transactions"},
			"cards":             []string{"list_cards", "retrieve_card_details", "retrieve_card_sensitive_details", "create_card_details_iframe", "create_card", "update_card_pin"},
			"external_accounts": []string{"list_external_accounts", "retrieve_external_account", "create_external_account", "update_external_account"},
			"transfers":         []string{"create_account_transfer", "create_ach_transfer", "create_real_time_payments_transfer", "create_fednow_transfer", "create_wire_transfer", "list_transfers", "retrieve_transfer", "list_transfer_queue", "approve_transfer", "cancel_transfer"},
		}, "", nil)
	default:
		return util.Fail(util.NewError(util.CodeValidationError, "unknown tool", map[string]any{"tool": name}, false), ""), true
	}
}

func (s *Server) handleWrite(ctx context.Context, session app.Session, api *increasex.Client, dryRun bool, confirmationToken string, preview func() (*app.PreviewResult, error), execute func() (any, string, error)) (any, bool) {
	if dryRun || strings.TrimSpace(confirmationToken) == "" {
		result, err := preview()
		return envelope(result, "", err)
	}
	data, requestID, err := execute()
	return envelope(data, requestID, err)
}

func envelope(data any, requestID string, err error) (any, bool) {
	if err != nil {
		return util.Fail(increasex.WrapError(err), requestID), true
	}
	return util.Ok(data, requestID), false
}

func (s *Server) tools() []toolDefinition {
	return []toolDefinition{
		{Name: "describe_capabilities", Description: "Help: grouped overview of supported accounts, cards, external accounts, and transfer tools", InputSchema: objectSchema(map[string]any{})},
		{Name: "list_accounts", Description: "Accounts: list Increase accounts", InputSchema: objectSchema(map[string]any{"status": stringSchema(), "limit": intSchema(), "cursor": stringSchema()})},
		{Name: "resolve_account", Description: "Accounts: resolve an account by fuzzy name or id", InputSchema: requiredSchema(map[string]any{"query": stringSchema(), "limit": intSchema()}, "query")},
		{Name: "get_balance", Description: "Accounts: retrieve an account balance", InputSchema: requiredSchema(map[string]any{"account_id": stringSchema()}, "account_id")},
		{Name: "create_account", Description: "Accounts: preview or create an account", InputSchema: objectSchema(map[string]any{"name": stringSchema(), "entity_id": stringSchema(), "informational_entity_id": stringSchema(), "program_id": stringSchema(), "idempotency_key": stringSchema(), "dry_run": boolSchema(), "confirmation_token": stringSchema()})},
		{Name: "close_account", Description: "Accounts: preview or close an account", InputSchema: objectSchema(map[string]any{"account_id": stringSchema(), "idempotency_key": stringSchema(), "dry_run": boolSchema(), "confirmation_token": stringSchema()})},
		{Name: "create_account_number", Description: "Accounts: preview or create an account number", InputSchema: objectSchema(map[string]any{"account_id": stringSchema(), "name": stringSchema(), "inbound_ach": objectSchema(map[string]any{"debit_status": stringSchema()}), "inbound_checks": objectSchema(map[string]any{"status": stringSchema()}), "idempotency_key": stringSchema(), "dry_run": boolSchema(), "confirmation_token": stringSchema()})},
		{Name: "list_recent_transactions", Description: "Transactions: list recent transactions", InputSchema: objectSchema(map[string]any{"account_id": stringSchema(), "since": stringSchema(), "limit": intSchema(), "cursor": stringSchema(), "categories": arraySchema(stringSchema())})},
		{Name: "list_cards", Description: "Cards: list cards", InputSchema: objectSchema(map[string]any{"account_id": stringSchema(), "status": stringSchema(), "limit": intSchema(), "cursor": stringSchema()})},
		{Name: "retrieve_card_details", Description: "Cards: retrieve masked card details", InputSchema: requiredSchema(map[string]any{"card_id": stringSchema()}, "card_id")},
		{Name: "retrieve_card_sensitive_details", Description: "Cards: retrieve sensitive card details", InputSchema: requiredSchema(map[string]any{"card_id": stringSchema()}, "card_id")},
		{Name: "create_card_details_iframe", Description: "Cards: create details iframe for a card", InputSchema: requiredSchema(map[string]any{"card_id": stringSchema(), "physical_card_id": stringSchema()}, "card_id")},
		{Name: "create_card", Description: "Cards: preview or create a card", InputSchema: objectSchema(map[string]any{"account_id": stringSchema(), "description": stringSchema(), "card_program": stringSchema(), "entity_id": stringSchema(), "billing_address": objectSchema(map[string]any{"city": stringSchema(), "line1": stringSchema(), "line2": stringSchema(), "postal_code": stringSchema(), "state": stringSchema()}), "digital_wallet": objectSchema(map[string]any{"digital_card_profile_id": stringSchema(), "email": stringSchema(), "phone": stringSchema()}), "idempotency_key": stringSchema(), "dry_run": boolSchema(), "confirmation_token": stringSchema()})},
		{Name: "update_card_pin", Description: "Cards: preview or update a card PIN", InputSchema: objectSchema(map[string]any{"card_id": stringSchema(), "pin": stringSchema(), "dry_run": boolSchema(), "confirmation_token": stringSchema()})},
		{Name: "list_external_accounts", Description: "External accounts: list stored external accounts", InputSchema: objectSchema(map[string]any{"status": stringSchema(), "cursor": stringSchema(), "limit": intSchema()})},
		{Name: "retrieve_external_account", Description: "External accounts: retrieve one external account", InputSchema: requiredSchema(map[string]any{"external_account_id": stringSchema()}, "external_account_id")},
		{Name: "create_external_account", Description: "External accounts: preview or create an external account", InputSchema: objectSchema(map[string]any{"account_number": stringSchema(), "description": stringSchema(), "routing_number": stringSchema(), "account_holder": stringSchema(), "funding": stringSchema(), "idempotency_key": stringSchema(), "dry_run": boolSchema(), "confirmation_token": stringSchema()})},
		{Name: "update_external_account", Description: "External accounts: preview or update an external account", InputSchema: objectSchema(map[string]any{"external_account_id": stringSchema(), "account_holder": stringSchema(), "description": stringSchema(), "funding": stringSchema(), "status": stringSchema(), "idempotency_key": stringSchema(), "dry_run": boolSchema(), "confirmation_token": stringSchema()})},
		{Name: "list_transfers", Description: "Transfers: list transfers by rail and status", InputSchema: requiredSchema(map[string]any{"rail": stringSchema(), "account_id": stringSchema(), "external_account_id": stringSchema(), "status": stringSchema(), "since": stringSchema(), "cursor": stringSchema(), "limit": intSchema()}, "rail")},
		{Name: "retrieve_transfer", Description: "Transfers: retrieve a transfer by rail and id", InputSchema: requiredSchema(map[string]any{"rail": stringSchema(), "transfer_id": stringSchema()}, "rail", "transfer_id")},
		{Name: "list_transfer_queue", Description: "Transfers: list pending approval queue entries for a rail", InputSchema: requiredSchema(map[string]any{"rail": stringSchema(), "limit": intSchema()}, "rail")},
		{Name: "approve_transfer", Description: "Transfers: preview or approve a pending transfer", InputSchema: objectSchema(map[string]any{"rail": stringSchema(), "transfer_id": stringSchema(), "dry_run": boolSchema(), "confirmation_token": stringSchema()})},
		{Name: "cancel_transfer", Description: "Transfers: preview or cancel a pending transfer", InputSchema: objectSchema(map[string]any{"rail": stringSchema(), "transfer_id": stringSchema(), "dry_run": boolSchema(), "confirmation_token": stringSchema()})},
		{Name: "create_account_transfer", Description: "Transfers: preview or create an account transfer", InputSchema: objectSchema(map[string]any{"from_account_id": stringSchema(), "to_account_id": stringSchema(), "amount_cents": intSchema(), "description": stringSchema(), "require_approval": boolSchema(), "idempotency_key": stringSchema(), "dry_run": boolSchema(), "confirmation_token": stringSchema()})},
		{Name: "create_ach_transfer", Description: "Transfers: preview or create an ACH transfer", InputSchema: objectSchema(map[string]any{"account_id": stringSchema(), "amount_cents": intSchema(), "statement_descriptor": stringSchema(), "account_number": stringSchema(), "routing_number": stringSchema(), "external_account_id": stringSchema(), "funding": stringSchema(), "destination_account_holder": stringSchema(), "individual_id": stringSchema(), "individual_name": stringSchema(), "company_name": stringSchema(), "company_entry_description": stringSchema(), "company_descriptive_date": stringSchema(), "company_discretionary_data": stringSchema(), "require_approval": boolSchema(), "idempotency_key": stringSchema(), "dry_run": boolSchema(), "confirmation_token": stringSchema()})},
		{Name: "create_real_time_payments_transfer", Description: "Transfers: preview or create a Real-Time Payments transfer", InputSchema: objectSchema(map[string]any{"amount_cents": intSchema(), "creditor_name": stringSchema(), "remittance_information": stringSchema(), "source_account_number_id": stringSchema(), "debtor_name": stringSchema(), "destination_account_number": stringSchema(), "destination_routing_number": stringSchema(), "external_account_id": stringSchema(), "ultimate_creditor_name": stringSchema(), "ultimate_debtor_name": stringSchema(), "require_approval": boolSchema(), "idempotency_key": stringSchema(), "dry_run": boolSchema(), "confirmation_token": stringSchema()})},
		{Name: "create_fednow_transfer", Description: "Transfers: preview or create a FedNow transfer", InputSchema: objectSchema(map[string]any{"account_id": stringSchema(), "amount_cents": intSchema(), "creditor_name": stringSchema(), "debtor_name": stringSchema(), "source_account_number_id": stringSchema(), "unstructured_remittance_information": stringSchema(), "account_number": stringSchema(), "routing_number": stringSchema(), "external_account_id": stringSchema(), "creditor_address": objectSchema(map[string]any{"city": stringSchema(), "line1": stringSchema(), "line2": stringSchema(), "postal_code": stringSchema(), "state": stringSchema()}), "debtor_address": objectSchema(map[string]any{"city": stringSchema(), "line1": stringSchema(), "line2": stringSchema(), "postal_code": stringSchema(), "state": stringSchema()}), "require_approval": boolSchema(), "idempotency_key": stringSchema(), "dry_run": boolSchema(), "confirmation_token": stringSchema()})},
		{Name: "create_wire_transfer", Description: "Transfers: preview or create a wire transfer", InputSchema: objectSchema(map[string]any{"account_id": stringSchema(), "amount_cents": intSchema(), "beneficiary_name": stringSchema(), "message_to_recipient": stringSchema(), "account_number": stringSchema(), "routing_number": stringSchema(), "external_account_id": stringSchema(), "beneficiary_address_line1": stringSchema(), "beneficiary_address_line2": stringSchema(), "beneficiary_address_line3": stringSchema(), "originator_name": stringSchema(), "originator_address_line1": stringSchema(), "originator_address_line2": stringSchema(), "originator_address_line3": stringSchema(), "require_approval": boolSchema(), "idempotency_key": stringSchema(), "dry_run": boolSchema(), "confirmation_token": stringSchema()})},
		{Name: "move_money_internal", Description: "Compatibility alias for create_account_transfer", InputSchema: objectSchema(map[string]any{"from_account_id": stringSchema(), "to_account_id": stringSchema(), "amount_cents": intSchema(), "description": stringSchema(), "require_approval": boolSchema(), "idempotency_key": stringSchema(), "dry_run": boolSchema(), "confirmation_token": stringSchema()})},
		{Name: "move_money_external_ach", Description: "Compatibility alias for create_ach_transfer", InputSchema: objectSchema(map[string]any{"account_id": stringSchema(), "amount_cents": intSchema(), "statement_descriptor": stringSchema(), "account_number": stringSchema(), "routing_number": stringSchema(), "external_account_id": stringSchema(), "funding": stringSchema(), "destination_account_holder": stringSchema(), "individual_id": stringSchema(), "individual_name": stringSchema(), "company_name": stringSchema(), "company_entry_description": stringSchema(), "company_descriptive_date": stringSchema(), "company_discretionary_data": stringSchema(), "require_approval": boolSchema(), "idempotency_key": stringSchema(), "dry_run": boolSchema(), "confirmation_token": stringSchema()})},
		{Name: "move_money_external_rtp", Description: "Compatibility alias for create_real_time_payments_transfer", InputSchema: objectSchema(map[string]any{"amount_cents": intSchema(), "creditor_name": stringSchema(), "remittance_information": stringSchema(), "source_account_number_id": stringSchema(), "debtor_name": stringSchema(), "destination_account_number": stringSchema(), "destination_routing_number": stringSchema(), "external_account_id": stringSchema(), "ultimate_creditor_name": stringSchema(), "ultimate_debtor_name": stringSchema(), "require_approval": boolSchema(), "idempotency_key": stringSchema(), "dry_run": boolSchema(), "confirmation_token": stringSchema()})},
		{Name: "move_money_external_fednow", Description: "Compatibility alias for create_fednow_transfer", InputSchema: objectSchema(map[string]any{"account_id": stringSchema(), "amount_cents": intSchema(), "creditor_name": stringSchema(), "debtor_name": stringSchema(), "source_account_number_id": stringSchema(), "unstructured_remittance_information": stringSchema(), "account_number": stringSchema(), "routing_number": stringSchema(), "external_account_id": stringSchema(), "creditor_address": objectSchema(map[string]any{"city": stringSchema(), "line1": stringSchema(), "line2": stringSchema(), "postal_code": stringSchema(), "state": stringSchema()}), "debtor_address": objectSchema(map[string]any{"city": stringSchema(), "line1": stringSchema(), "line2": stringSchema(), "postal_code": stringSchema(), "state": stringSchema()}), "require_approval": boolSchema(), "idempotency_key": stringSchema(), "dry_run": boolSchema(), "confirmation_token": stringSchema()})},
		{Name: "move_money_external_wire", Description: "Compatibility alias for create_wire_transfer", InputSchema: objectSchema(map[string]any{"account_id": stringSchema(), "amount_cents": intSchema(), "beneficiary_name": stringSchema(), "message_to_recipient": stringSchema(), "account_number": stringSchema(), "routing_number": stringSchema(), "external_account_id": stringSchema(), "beneficiary_address_line1": stringSchema(), "beneficiary_address_line2": stringSchema(), "beneficiary_address_line3": stringSchema(), "originator_name": stringSchema(), "originator_address_line1": stringSchema(), "originator_address_line2": stringSchema(), "originator_address_line3": stringSchema(), "require_approval": boolSchema(), "idempotency_key": stringSchema(), "dry_run": boolSchema(), "confirmation_token": stringSchema()})},
	}
}

func stringSchema() map[string]any { return map[string]any{"type": "string"} }
func intSchema() map[string]any    { return map[string]any{"type": "integer"} }
func boolSchema() map[string]any   { return map[string]any{"type": "boolean"} }

func arraySchema(item any) map[string]any {
	return map[string]any{"type": "array", "items": item}
}

func objectSchema(properties map[string]any) map[string]any {
	return map[string]any{"type": "object", "properties": properties}
}

func requiredSchema(properties map[string]any, required ...string) map[string]any {
	schema := objectSchema(properties)
	schema["required"] = required
	return schema
}

func decodeArgs(args map[string]any, out any) {
	raw, _ := json.Marshal(args)
	_ = json.Unmarshal(raw, out)
}

func asString(value any) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	default:
		return fmt.Sprint(v)
	}
}

func asInt64(value any, fallback int64) int64 {
	if value == nil {
		return fallback
	}
	switch v := value.(type) {
	case float64:
		return int64(v)
	case int64:
		return v
	case int:
		return int64(v)
	case json.Number:
		i, _ := v.Int64()
		return i
	default:
		return fallback
	}
}

func asStringSlice(value any) []string {
	if value == nil {
		return nil
	}
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, asString(item))
	}
	return out
}

func toJSONString(value any) string {
	raw, _ := json.Marshal(value)
	return string(raw)
}

func (s *Server) readFrame() ([]byte, error) {
	switch s.mode {
	case transportModeJSONLine:
		return s.readJSONLineFrame()
	case transportModeContentLength:
		return s.readContentLengthFrame("")
	}

	for {
		line, err := s.reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return nil, err
		}

		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if err == io.EOF {
				return nil, io.EOF
			}
			continue
		}

		if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
			s.mode = transportModeJSONLine
			return finishJSONLine(trimmed, err)
		}

		s.mode = transportModeContentLength
		return s.readContentLengthFrame(line)
	}
}

func (s *Server) readJSONLineFrame() ([]byte, error) {
	for {
		line, err := s.reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return nil, err
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if err == io.EOF {
				return nil, io.EOF
			}
			continue
		}
		return finishJSONLine(trimmed, err)
	}
}

func finishJSONLine(line string, err error) ([]byte, error) {
	if err == io.EOF && strings.TrimSpace(line) == "" {
		return nil, io.EOF
	}
	return []byte(line), nil
}

func (s *Server) readContentLengthFrame(firstLine string) ([]byte, error) {
	length := 0
	sawHeader := false
	if strings.TrimSpace(firstLine) != "" {
		sawHeader = true
		if parsed, ok := parseContentLengthHeader(strings.TrimSpace(firstLine)); ok {
			length = parsed
		}
	}
	for {
		line, err := s.reader.ReadString('\n')
		if err != nil {
			if err == io.EOF && !sawHeader {
				return nil, io.EOF
			}
			return nil, err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			if !sawHeader {
				continue
			}
			break
		}
		sawHeader = true
		if parsed, ok := parseContentLengthHeader(line); ok {
			length = parsed
		}
	}
	if length <= 0 {
		return nil, fmt.Errorf("missing Content-Length header")
	}
	body := make([]byte, length)
	if _, err := io.ReadFull(s.reader, body); err != nil {
		return nil, err
	}
	return body, nil
}

func (s *Server) writeFrame(value any) error {
	body, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if s.mode == transportModeJSONLine {
		if _, err := s.writer.Write(body); err != nil {
			return err
		}
		_, err = s.writer.Write([]byte("\n"))
		return err
	}
	if _, err := fmt.Fprintf(s.writer, "Content-Length: %d\r\n\r\n", len(body)); err != nil {
		return err
	}
	_, err = s.writer.Write(body)
	return err
}

func parseContentLengthHeader(line string) (int, bool) {
	key, value, ok := strings.Cut(line, ":")
	if !ok || !strings.EqualFold(strings.TrimSpace(key), "Content-Length") {
		return 0, false
	}
	length, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0, false
	}
	return length, true
}

func (s *Server) debugf(format string, args ...any) {
	if !s.options.Debug && os.Getenv("INCREASEX_MCP_DEBUG") == "" {
		return
	}
	fmt.Fprintf(os.Stderr, "increasex mcp: "+format+"\n", args...)
}
