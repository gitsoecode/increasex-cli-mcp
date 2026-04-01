package app

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/gitsoecode/increasex-cli-mcp/internal/util"
)

func TestRetrieveTransferResolvesEventIDForAccountTransfer(t *testing.T) {
	services := NewServices()
	paths := []string{}
	api := newTestIncreaseClient(t, func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/events/event_123":
			w.Header().Set("X-Request-Id", "req_event")
			_, _ = w.Write([]byte(`{
				"id":"event_123",
				"associated_object_id":"account_transfer_123",
				"associated_object_type":"account_transfer",
				"category":"account_transfer.created",
				"created_at":"2026-03-29T12:00:00Z",
				"type":"event"
			}`))
		case "/account_transfers/account_transfer_123":
			w.Header().Set("X-Request-Id", "req_transfer")
			_, _ = w.Write([]byte(`{
				"id":"account_transfer_123",
				"account_id":"account_from",
				"amount":5000,
				"created_at":"2026-03-29T12:00:01Z",
				"description":"Ops funding",
				"destination_account_id":"account_to",
				"destination_transaction_id":"transaction_dest",
				"pending_transaction_id":"pending_transaction_123",
				"status":"complete",
				"transaction_id":"transaction_source",
				"type":"account_transfer"
			}`))
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	})

	result, requestID, err := services.RetrieveTransfer(context.Background(), api, RetrieveTransferInput{
		EventID: "event_123",
	})
	if err != nil {
		t.Fatalf("RetrieveTransfer() error = %v", err)
	}
	if requestID != "req_transfer" {
		t.Fatalf("RetrieveTransfer() requestID = %q, want %q", requestID, "req_transfer")
	}
	if got := strings.Join(paths, ","); got != "/events/event_123,/account_transfers/account_transfer_123" {
		t.Fatalf("RetrieveTransfer() paths = %q, want event lookup followed by transfer lookup", got)
	}
	if result.Rail != "account" {
		t.Fatalf("RetrieveTransfer() rail = %q, want %q", result.Rail, "account")
	}
	if result.DestinationAccountID != "account_to" {
		t.Fatalf("RetrieveTransfer() destination_account_id = %q, want %q", result.DestinationAccountID, "account_to")
	}
	if result.Description != "Ops funding" {
		t.Fatalf("RetrieveTransfer() description = %q, want %q", result.Description, "Ops funding")
	}
}

func TestRetrieveTransferInfersRailFromTransferID(t *testing.T) {
	services := NewServices()
	api := newTestIncreaseClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/account_transfers/account_transfer_123" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Request-Id", "req_transfer")
		_, _ = w.Write([]byte(`{
			"id":"account_transfer_123",
			"account_id":"account_from",
			"amount":5000,
			"created_at":"2026-03-29T12:00:01Z",
			"description":"Ops funding",
			"destination_account_id":"account_to",
			"destination_transaction_id":"transaction_dest",
			"status":"complete",
			"transaction_id":"transaction_source",
			"type":"account_transfer"
		}`))
	})

	result, _, err := services.RetrieveTransfer(context.Background(), api, RetrieveTransferInput{
		TransferID: "account_transfer_123",
	})
	if err != nil {
		t.Fatalf("RetrieveTransfer() error = %v", err)
	}
	if result.Rail != "account" {
		t.Fatalf("RetrieveTransfer() rail = %q, want %q", result.Rail, "account")
	}
	if result.DestinationAccountID != "account_to" {
		t.Fatalf("RetrieveTransfer() destination_account_id = %q, want %q", result.DestinationAccountID, "account_to")
	}
}

func TestRetrieveTransferRejectsMismatchedRailAndTransferID(t *testing.T) {
	services := NewServices()
	api := newTestIncreaseClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected HTTP request %s", r.URL.Path)
	})

	_, _, err := services.RetrieveTransfer(context.Background(), api, RetrieveTransferInput{
		Rail:       "wire",
		TransferID: "account_transfer_123",
	})
	if err == nil {
		t.Fatal("RetrieveTransfer() error = nil, want validation error")
	}
	var detail *util.ErrorDetail
	if !errors.As(err, &detail) {
		t.Fatalf("RetrieveTransfer() error = %T, want *util.ErrorDetail", err)
	}
	if detail.Code != util.CodeValidationError {
		t.Fatalf("RetrieveTransfer() code = %q, want %q", detail.Code, util.CodeValidationError)
	}
	if detail.Message != "rail does not match the transfer_id prefix" {
		t.Fatalf("RetrieveTransfer() message = %q, want mismatch guidance", detail.Message)
	}
}

func TestRetrieveTransferRejectsNonTransferEvent(t *testing.T) {
	services := NewServices()
	api := newTestIncreaseClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/events/event_123" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Request-Id", "req_event")
		_, _ = w.Write([]byte(`{
			"id":"event_123",
			"associated_object_id":"account_123",
			"associated_object_type":"account",
			"category":"account.created",
			"created_at":"2026-03-29T12:00:00Z",
			"type":"event"
		}`))
	})

	_, _, err := services.RetrieveTransfer(context.Background(), api, RetrieveTransferInput{
		EventID: "event_123",
	})
	if err == nil {
		t.Fatal("RetrieveTransfer() error = nil, want validation error")
	}
	var detail *util.ErrorDetail
	if !errors.As(err, &detail) {
		t.Fatalf("RetrieveTransfer() error = %T, want *util.ErrorDetail", err)
	}
	if detail.Code != util.CodeValidationError {
		t.Fatalf("RetrieveTransfer() code = %q, want %q", detail.Code, util.CodeValidationError)
	}
	if detail.Message != "event_id must reference a transfer event" {
		t.Fatalf("RetrieveTransfer() message = %q, want transfer-event guidance", detail.Message)
	}
}

func TestListTransferQueueNormalizesAccountRailAliases(t *testing.T) {
	services := NewServices()
	paths := make([]string, 0, 2)
	api := newTestIncreaseClient(t, func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		if r.URL.Path != "/account_transfers" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Request-Id", "req_queue")
		_, _ = w.Write([]byte(`{
			"data":[{
				"id":"account_transfer_123",
				"account_id":"account_1",
				"amount":500,
				"approval":null,
				"cancellation":null,
				"created_at":"2026-03-29T12:00:00Z",
				"created_by":null,
				"currency":"USD",
				"description":"fund MCP account",
				"destination_account_id":"account_2",
				"destination_transaction_id":null,
				"idempotency_key":null,
				"pending_transaction_id":"pending_transaction_123",
				"status":"pending_approval",
				"transaction_id":null,
				"type":"account_transfer"
			}],
			"has_more":false
		}`))
	})

	internalItems, requestID, err := services.ListTransferQueue(context.Background(), api, "internal", 20)
	if err != nil {
		t.Fatalf("ListTransferQueue(internal) error = %v", err)
	}
	accountItems, accountRequestID, err := services.ListTransferQueue(context.Background(), api, "account", 20)
	if err != nil {
		t.Fatalf("ListTransferQueue(account) error = %v", err)
	}

	if requestID != "req_queue" || accountRequestID != "req_queue" {
		t.Fatalf("requestIDs = %q, %q, want req_queue", requestID, accountRequestID)
	}
	if !reflect.DeepEqual(internalItems, accountItems) {
		t.Fatalf("alias mismatch: internal=%#v account=%#v", internalItems, accountItems)
	}
	if got := strings.Join(paths, ","); got != "/account_transfers,/account_transfers" {
		t.Fatalf("paths = %q, want repeated account transfer list path", got)
	}
	if len(internalItems) != 1 || internalItems[0].Rail != "account" || internalItems[0].Status != "pending_approval" {
		t.Fatalf("internalItems = %#v, want normalized pending approval account transfer", internalItems)
	}
}

func TestResolveTransferRetrievalNormalizesRailAliases(t *testing.T) {
	services := NewServices()
	cases := []struct {
		name       string
		rail       string
		transferID string
		wantRail   string
	}{
		{name: "internal", rail: "internal", transferID: "account_transfer_123", wantRail: "account"},
		{name: "account_transfer", rail: "account_transfer", transferID: "account_transfer_123", wantRail: "account"},
		{name: "rtp", rail: "rtp", transferID: "real_time_payments_transfer_123", wantRail: "real_time_payments"},
		{name: "real-time-payments", rail: "real-time-payments", transferID: "real_time_payments_transfer_123", wantRail: "real_time_payments"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rail, transferID, requestID, err := services.resolveTransferRetrieval(context.Background(), nil, RetrieveTransferInput{
				Rail:       tc.rail,
				TransferID: tc.transferID,
			})
			if err != nil {
				t.Fatalf("resolveTransferRetrieval() error = %v", err)
			}
			if rail != tc.wantRail || transferID != tc.transferID || requestID != "" {
				t.Fatalf("resolveTransferRetrieval() = (%q, %q, %q), want (%q, %q, %q)", rail, transferID, requestID, tc.wantRail, tc.transferID, "")
			}
		})
	}
}

func TestTransferActionPreviewNormalizesRailAliases(t *testing.T) {
	services := NewServices()
	session := Session{ProfileName: "default", Environment: "sandbox"}

	cases := []struct {
		name       string
		preview    func() (*PreviewResult, error)
		wantRail   string
		wantPrefix string
	}{
		{
			name: "approve internal",
			preview: func() (*PreviewResult, error) {
				return services.PreviewApproveTransfer(session, TransferActionInput{
					Rail:       "internal",
					TransferID: "account_transfer_123",
				})
			},
			wantRail:   "account",
			wantPrefix: "Approve account transfer",
		},
		{
			name: "cancel rtp",
			preview: func() (*PreviewResult, error) {
				return services.PreviewCancelTransfer(session, TransferActionInput{
					Rail:       "rtp",
					TransferID: "real_time_payments_transfer_123",
				})
			},
			wantRail:   "real_time_payments",
			wantPrefix: "Cancel real_time_payments transfer",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			preview, err := tc.preview()
			if err != nil {
				t.Fatalf("preview error = %v", err)
			}
			if got := preview.Details["rail"]; got != tc.wantRail {
				t.Fatalf("preview rail = %v, want %q", got, tc.wantRail)
			}
			if !strings.HasPrefix(preview.Summary, tc.wantPrefix) {
				t.Fatalf("preview summary = %q, want prefix %q", preview.Summary, tc.wantPrefix)
			}
		})
	}
}

func TestApproveAndCancelTransferNormalizeAccountRailAliasOnExecute(t *testing.T) {
	services := NewServices()
	session := Session{ProfileName: "default", Environment: "sandbox"}
	paths := make([]string, 0, 2)
	api := newTestIncreaseClient(t, func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Request-Id", "req_transfer")
		switch r.URL.Path {
		case "/account_transfers/account_transfer_123/approve", "/account_transfers/account_transfer_123/cancel":
			_, _ = w.Write([]byte(`{
				"id":"account_transfer_123",
				"account_id":"account_1",
				"amount":500,
				"approval":null,
				"cancellation":null,
				"created_at":"2026-03-29T12:00:00Z",
				"created_by":null,
				"currency":"USD",
				"description":"fund MCP account",
				"destination_account_id":"account_2",
				"destination_transaction_id":null,
				"idempotency_key":null,
				"pending_transaction_id":"pending_transaction_123",
				"status":"pending_approval",
				"transaction_id":null,
				"type":"account_transfer"
			}`))
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	})

	approvePreview, err := services.PreviewApproveTransfer(session, TransferActionInput{
		Rail:       "internal",
		TransferID: "account_transfer_123",
	})
	if err != nil {
		t.Fatalf("PreviewApproveTransfer() error = %v", err)
	}
	approved, requestID, err := services.ExecuteApproveTransfer(context.Background(), api, session, TransferActionInput{
		Rail:              "internal",
		TransferID:        "account_transfer_123",
		ConfirmationToken: approvePreview.ConfirmationToken,
	})
	if err != nil {
		t.Fatalf("ExecuteApproveTransfer() error = %v", err)
	}
	approvedPayload, ok := approved.(map[string]any)
	if !ok {
		t.Fatalf("ExecuteApproveTransfer() type = %T, want map[string]any", approved)
	}
	approvedTransfer, ok := approvedPayload["transfer"].(TransferSummary)
	if !ok {
		t.Fatalf("approved transfer type = %T, want TransferSummary", approvedPayload["transfer"])
	}
	if requestID != "req_transfer" || approvedTransfer.Rail != "account" {
		t.Fatalf("approve result = (%q, %#v), want req_transfer + account rail", requestID, approvedTransfer)
	}

	cancelPreview, err := services.PreviewCancelTransfer(session, TransferActionInput{
		Rail:       "internal",
		TransferID: "account_transfer_123",
	})
	if err != nil {
		t.Fatalf("PreviewCancelTransfer() error = %v", err)
	}
	canceled, requestID, err := services.ExecuteCancelTransfer(context.Background(), api, session, TransferActionInput{
		Rail:              "internal",
		TransferID:        "account_transfer_123",
		ConfirmationToken: cancelPreview.ConfirmationToken,
	})
	if err != nil {
		t.Fatalf("ExecuteCancelTransfer() error = %v", err)
	}
	canceledPayload, ok := canceled.(map[string]any)
	if !ok {
		t.Fatalf("ExecuteCancelTransfer() type = %T, want map[string]any", canceled)
	}
	canceledTransfer, ok := canceledPayload["transfer"].(TransferSummary)
	if !ok {
		t.Fatalf("canceled transfer type = %T, want TransferSummary", canceledPayload["transfer"])
	}
	if requestID != "req_transfer" || canceledTransfer.Rail != "account" {
		t.Fatalf("cancel result = (%q, %#v), want req_transfer + account rail", requestID, canceledTransfer)
	}

	if got := strings.Join(paths, ","); got != "/account_transfers/account_transfer_123/approve,/account_transfers/account_transfer_123/cancel" {
		t.Fatalf("paths = %q, want approve then cancel internal transfer paths", got)
	}
}

func TestTransferActionValidationRejectsUnknownRailAfterNormalization(t *testing.T) {
	services := NewServices()
	session := Session{ProfileName: "default", Environment: "sandbox"}

	preview, err := services.PreviewApproveTransfer(session, TransferActionInput{
		Rail:       "bogus",
		TransferID: "account_transfer_123",
	})
	assertTransferValidationError(t, err, "rail")
	if preview != nil {
		t.Fatalf("preview = %#v, want nil on validation failure", preview)
	}
}

func TestApproveTransferPreviewIncludesExecuteReviewMetadata(t *testing.T) {
	services := NewServices()
	session := Session{ProfileName: "default", Environment: "sandbox"}

	preview, err := services.PreviewApproveTransfer(session, TransferActionInput{
		Rail:       "internal",
		TransferID: "account_transfer_123",
	})
	if err != nil {
		t.Fatalf("PreviewApproveTransfer() error = %v", err)
	}
	if preview.ExecuteAction != "approve_transfer" {
		t.Fatalf("PreviewApproveTransfer() execute_action = %q, want approve_transfer", preview.ExecuteAction)
	}
	if preview.ExecuteSummary != preview.Summary {
		t.Fatalf("PreviewApproveTransfer() execute_summary = %q, want %q", preview.ExecuteSummary, preview.Summary)
	}
	if !preview.ExecuteRequiresConfirmation {
		t.Fatal("PreviewApproveTransfer() execute_requires_confirmation = false, want true")
	}
	if got := preview.ExecuteDetails["rail"]; got != "account" {
		t.Fatalf("PreviewApproveTransfer() execute_details.rail = %v, want account", got)
	}
}
