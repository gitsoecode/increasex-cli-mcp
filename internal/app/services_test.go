package app

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	increase "github.com/Increase/increase-go"
	increasex "github.com/gitsoecode/increasex-cli-mcp/internal/increase"
	"github.com/gitsoecode/increasex-cli-mcp/internal/util"
)

func TestNormalizeTransactionDirection(t *testing.T) {
	credit := normalizeTransaction(increase.Transaction{
		ID:          "txn_credit",
		AccountID:   "account_1",
		Amount:      250,
		Description: "credit",
		CreatedAt:   time.Unix(0, 0),
		Type:        increase.TransactionTypeTransaction,
	})
	if credit.Direction != "credit" {
		t.Fatalf("credit direction = %q, want credit", credit.Direction)
	}

	debit := normalizeTransaction(increase.Transaction{
		ID:          "txn_debit",
		AccountID:   "account_1",
		Amount:      -250,
		Description: "debit",
		CreatedAt:   time.Unix(0, 0),
		Type:        increase.TransactionTypeTransaction,
	})
	if debit.Direction != "debit" {
		t.Fatalf("debit direction = %q, want debit", debit.Direction)
	}
}

func TestAccountScorePrefersExactMatches(t *testing.T) {
	account := AccountSummary{
		ID:       "account_123",
		Name:     "Payroll",
		EntityID: "entity_1",
	}

	if score := accountScore(account, "account_123"); score != 100 {
		t.Fatalf("accountScore exact id = %d, want 100", score)
	}
	if score := accountScore(account, "pay"); score <= 0 {
		t.Fatalf("accountScore partial name = %d, want positive", score)
	}
}

func TestGeneratedInternalDescription(t *testing.T) {
	description := generatedInternalDescription(MoveMoneyInternalInput{
		FromAccountID: "account_a",
		ToAccountID:   "account_b",
		AmountCents:   12345,
	})
	if description == "" {
		t.Fatal("generatedInternalDescription() returned empty string")
	}
}

func TestConfirmationPayloadExcludesControlFields(t *testing.T) {
	input := MoveMoneyInternalInput{
		FromAccountID:     "account_a",
		ToAccountID:       "account_b",
		AmountCents:       100,
		Description:       "CLI test",
		ConfirmationToken: "token",
	}
	dryRun := false
	input.DryRun = &dryRun

	payload := effectiveConfirmationPayload(input)
	if _, ok := payload["confirmation_token"]; ok {
		t.Fatal("confirmation_payload should exclude confirmation_token")
	}
	if _, ok := payload["dry_run"]; ok {
		t.Fatal("confirmation_payload should exclude dry_run")
	}
	if payload["description"] != "CLI test" {
		t.Fatalf("confirmation_payload description = %v, want CLI test", payload["description"])
	}
}

func TestPreviewUpdateCardPINMasksSensitiveValue(t *testing.T) {
	services := NewServices()
	preview, err := services.PreviewUpdateCardPIN(Session{ProfileName: "default", Environment: "sandbox"}, UpdateCardPINInput{
		CardID: "card_123",
		PIN:    "1234",
	})
	if err != nil {
		t.Fatalf("PreviewUpdateCardPIN() error = %v", err)
	}
	if got := preview.Details["pin"]; got != "****" {
		t.Fatalf("PreviewUpdateCardPIN() pin detail = %v, want masked value", got)
	}
	if preview.ConfirmationToken == "" {
		t.Fatal("PreviewUpdateCardPIN() confirmation token = empty, want token")
	}
}

func TestPreviewCreateCardIncludesNestedDetails(t *testing.T) {
	services := NewServices()
	preview, err := services.PreviewCreateCard(Session{ProfileName: "default", Environment: "sandbox"}, CreateCardInput{
		AccountID:   "account_123",
		Description: "team card",
		BillingAddress: &BillingAddressInput{
			Line1:      "123 Test St",
			City:       "San Francisco",
			State:      "CA",
			PostalCode: "94107",
		},
		DigitalWallet: &DigitalWalletInput{
			Email: "wallet@example.com",
		},
	})
	if err != nil {
		t.Fatalf("PreviewCreateCard() error = %v", err)
	}
	billing, ok := preview.Details["billing_address"].(map[string]any)
	if !ok {
		t.Fatalf("billing_address detail = %#v, want nested map", preview.Details["billing_address"])
	}
	if got := billing["line1"]; got != "123 Test St" {
		t.Fatalf("billing_address.line1 = %v, want %q", got, "123 Test St")
	}
	wallet, ok := preview.Details["digital_wallet"].(map[string]any)
	if !ok {
		t.Fatalf("digital_wallet detail = %#v, want nested map", preview.Details["digital_wallet"])
	}
	if got := wallet["email"]; got != "wallet@example.com" {
		t.Fatalf("digital_wallet.email = %v, want %q", got, "wallet@example.com")
	}
	if preview.ConfirmationToken == "" {
		t.Fatal("PreviewCreateCard() confirmation token = empty, want token")
	}
}

func TestPreviewCreateCardRejectsInvalidFields(t *testing.T) {
	services := NewServices()
	_, err := services.PreviewCreateCard(Session{ProfileName: "default", Environment: "sandbox"}, CreateCardInput{
		Description:   strings.Repeat("a", 201),
		DigitalWallet: &DigitalWalletInput{},
	})
	if err == nil {
		t.Fatal("PreviewCreateCard() error = nil, want validation error")
	}
	var detail *util.ErrorDetail
	if !errors.As(err, &detail) {
		t.Fatalf("PreviewCreateCard() error = %T, want *util.ErrorDetail", err)
	}
	if detail.Code != util.CodeValidationError {
		t.Fatalf("validation code = %q, want %q", detail.Code, util.CodeValidationError)
	}
	if len(detail.Fields) < 3 {
		t.Fatalf("validation fields = %#v, want account_id, description, and digital_wallet errors", detail.Fields)
	}
}

func TestTransferPreviewSummariesReflectApprovalState(t *testing.T) {
	services := NewServices()
	session := Session{ProfileName: "default", Environment: "sandbox"}
	requireApproval := true

	cases := []struct {
		name string
		got  func() string
		want string
	}{
		{
			name: "account queued",
			got: func() string {
				preview, err := services.PreviewInternalTransfer(session, MoveMoneyInternalInput{
					FromAccountID:   "account_a",
					ToAccountID:     "account_b",
					AmountCents:     100,
					RequireApproval: &requireApproval,
				})
				return mustPreviewSummary(t, preview, err)
			},
			want: "Queue account transfer $1.00 for approval",
		},
		{
			name: "ach immediate",
			got: func() string {
				preview, err := services.PreviewExternalACH(session, ACHTransferInput{
					AccountID:       "account_a",
					AmountCents:     100,
					RequireApproval: nil,
				})
				return mustPreviewSummary(t, preview, err)
			},
			want: "Create ACH transfer $1.00",
		},
		{
			name: "rtp queued",
			got: func() string {
				preview, err := services.PreviewExternalRTP(session, RTPTransferInput{
					AmountCents:     100,
					RequireApproval: &requireApproval,
				})
				return mustPreviewSummary(t, preview, err)
			},
			want: "Queue Real-Time Payments transfer $1.00 for approval",
		},
		{
			name: "fednow immediate",
			got: func() string {
				preview, err := services.PreviewExternalFedNow(session, FedNowTransferInput{
					AmountCents: 100,
				})
				return mustPreviewSummary(t, preview, err)
			},
			want: "Create FedNow transfer $1.00",
		},
		{
			name: "wire queued",
			got: func() string {
				preview, err := services.PreviewExternalWire(session, WireTransferInput{
					AmountCents:     100,
					RequireApproval: &requireApproval,
				})
				return mustPreviewSummary(t, preview, err)
			},
			want: "Queue wire transfer $1.00 for approval",
		},
	}

	for _, tc := range cases {
		got := tc.got()
		if got != tc.want {
			t.Fatalf("%s summary = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestBuildTransactionListParamsDefaultsToLast30Days(t *testing.T) {
	services := Services{
		now: func() time.Time {
			return time.Date(2026, time.March, 29, 15, 4, 5, 0, time.UTC)
		},
	}

	params, err := services.buildTransactionListParams(ListTransactionsInput{})
	if err != nil {
		t.Fatalf("buildTransactionListParams() error = %v", err)
	}

	if !params.CreatedAt.Present {
		t.Fatal("CreatedAt.Present = false, want true")
	}
	if got, want := params.CreatedAt.Value.OnOrAfter.Value, time.Date(2026, time.February, 27, 15, 4, 5, 0, time.UTC); !got.Equal(want) {
		t.Fatalf("OnOrAfter = %v, want %v", got, want)
	}
	if params.CreatedAt.Value.OnOrBefore.Present {
		t.Fatalf("OnOrBefore.Present = true, want false for default range")
	}
}

func TestBuildTransactionListParamsAcceptsExplicitBounds(t *testing.T) {
	services := Services{}

	params, err := services.buildTransactionListParams(ListTransactionsInput{
		AccountID: "account_123",
		TimeRange: TransactionTimeRangeInput{
			Since: "2026-03-01T00:00:00Z",
			Until: "2026-03-15T12:00:00Z",
		},
		Limit:      10,
		Cursor:     "cursor_123",
		Categories: []string{"account_transfer_intention"},
	})
	if err != nil {
		t.Fatalf("buildTransactionListParams() error = %v", err)
	}

	if got := params.AccountID.Value; got != "account_123" {
		t.Fatalf("AccountID = %q, want %q", got, "account_123")
	}
	if got := params.CreatedAt.Value.OnOrAfter.Value; !got.Equal(time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("OnOrAfter = %v, want exact lower bound", got)
	}
	if got := params.CreatedAt.Value.OnOrBefore.Value; !got.Equal(time.Date(2026, time.March, 15, 12, 0, 0, 0, time.UTC)) {
		t.Fatalf("OnOrBefore = %v, want exact upper bound", got)
	}
	if got := len(params.Category.Value.In.Value); got != 1 {
		t.Fatalf("category count = %d, want 1", got)
	}
}

func TestResolveTransactionTimeRangeSupportsPresets(t *testing.T) {
	services := Services{
		now: func() time.Time {
			return time.Date(2026, time.March, 29, 15, 4, 5, 0, time.UTC)
		},
	}

	since, until, err := services.resolveTransactionTimeRange(TransactionTimeRangeInput{Period: "current-month"})
	if err != nil {
		t.Fatalf("resolveTransactionTimeRange() error = %v", err)
	}
	if since == nil || until == nil {
		t.Fatal("resolveTransactionTimeRange() returned nil bounds for current-month preset")
	}
	if got, want := *since, time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Fatalf("current-month since = %v, want %v", got, want)
	}
	if got, want := *until, time.Date(2026, time.March, 29, 15, 4, 5, 0, time.UTC); !got.Equal(want) {
		t.Fatalf("current-month until = %v, want %v", got, want)
	}

	since, until, err = services.resolveTransactionTimeRange(TransactionTimeRangeInput{Period: "previous-month"})
	if err != nil {
		t.Fatalf("resolveTransactionTimeRange(previous-month) error = %v", err)
	}
	if got, want := *since, time.Date(2026, time.February, 1, 0, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Fatalf("previous-month since = %v, want %v", got, want)
	}
	if got, want := *until, time.Date(2026, time.February, 28, 23, 59, 59, 0, time.UTC); !got.Equal(want) {
		t.Fatalf("previous-month until = %v, want %v", got, want)
	}
}

func TestResolveTransactionTimeRangeRejectsInvalidValues(t *testing.T) {
	services := Services{}

	cases := []struct {
		name  string
		input TransactionTimeRangeInput
		want  string
	}{
		{
			name:  "bad since",
			input: TransactionTimeRangeInput{Since: "yesterday"},
			want:  "since must be RFC3339",
		},
		{
			name:  "bad until",
			input: TransactionTimeRangeInput{Until: "tomorrow"},
			want:  "until must be RFC3339",
		},
		{
			name:  "bad preset",
			input: TransactionTimeRangeInput{Period: "quarter-to-date"},
			want:  "period must be one of",
		},
		{
			name:  "reversed bounds",
			input: TransactionTimeRangeInput{Since: "2026-03-15T00:00:00Z", Until: "2026-03-01T00:00:00Z"},
			want:  "since must be before or equal to until",
		},
	}

	for _, tc := range cases {
		_, _, err := services.resolveTransactionTimeRange(tc.input)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("%s: resolveTransactionTimeRange() error = %v, want substring %q", tc.name, err, tc.want)
		}
	}
}

func mustPreviewSummary(t *testing.T, preview *PreviewResult, err error) string {
	t.Helper()
	if err != nil {
		t.Fatalf("preview error = %v", err)
	}
	return preview.Summary
}

func TestPreviewCreateExternalAccountRejectsInvalidAccountHolder(t *testing.T) {
	services := NewServices()
	_, err := services.PreviewCreateExternalAccount(Session{ProfileName: "default", Environment: "sandbox"}, CreateExternalAccountInput{
		Description:   "Vendor account",
		RoutingNumber: "021000021",
		AccountNumber: "123456789",
		AccountHolder: "TEST USER BNK",
	})
	if err == nil {
		t.Fatal("PreviewCreateExternalAccount() error = nil, want validation error")
	}
	var detail *util.ErrorDetail
	if !errors.As(err, &detail) {
		t.Fatalf("PreviewCreateExternalAccount() error = %T, want *util.ErrorDetail", err)
	}
	if detail.Code != util.CodeValidationError {
		t.Fatalf("validation code = %q, want %q", detail.Code, util.CodeValidationError)
	}
	if len(detail.Fields) != 1 || detail.Fields[0].Field != "account_holder" {
		t.Fatalf("validation fields = %#v, want account_holder field error", detail.Fields)
	}
}

func TestNormalizeAccountNumberMasksSensitiveFields(t *testing.T) {
	number := normalizeAccountNumber(increase.AccountNumber{
		ID:            "account_number_123",
		AccountID:     "account_123",
		Name:          "Primary",
		AccountNumber: "987654321",
		RoutingNumber: "101050001",
		Status:        increase.AccountNumberStatusActive,
		CreatedAt:     time.Unix(0, 0),
		InboundACH: increase.AccountNumberInboundACH{
			DebitStatus: increase.AccountNumberInboundACHDebitStatusAllowed,
		},
		InboundChecks: increase.AccountNumberInboundChecks{
			Status: increase.AccountNumberInboundChecksStatusCheckTransfersOnly,
		},
	})

	if number.AccountNumberMasked == "987654321" || number.AccountNumberMasked == "" {
		t.Fatalf("AccountNumberMasked = %q, want masked value", number.AccountNumberMasked)
	}
	if number.InboundACH == nil || number.InboundACH.DebitStatus != "allowed" {
		t.Fatalf("InboundACH = %#v, want allowed", number.InboundACH)
	}
	if number.InboundChecks == nil || number.InboundChecks.Status != "check_transfers_only" {
		t.Fatalf("InboundChecks = %#v, want check_transfers_only", number.InboundChecks)
	}
}

func TestNormalizeSensitiveAccountNumberDetailsIncludesFullNumber(t *testing.T) {
	details := normalizeSensitiveAccountNumberDetails(increase.AccountNumber{
		ID:             "account_number_123",
		AccountID:      "account_123",
		Name:           "Primary",
		AccountNumber:  "987654321",
		RoutingNumber:  "101050001",
		Status:         increase.AccountNumberStatusActive,
		IdempotencyKey: "idem_123",
		CreatedAt:      time.Unix(0, 0),
		InboundACH: increase.AccountNumberInboundACH{
			DebitStatus: increase.AccountNumberInboundACHDebitStatusAllowed,
		},
		InboundChecks: increase.AccountNumberInboundChecks{
			Status: increase.AccountNumberInboundChecksStatusAllowed,
		},
	})

	if details.AccountNumber != "987654321" {
		t.Fatalf("AccountNumber = %q, want full value", details.AccountNumber)
	}
	if details.AccountNumberMasked == details.AccountNumber {
		t.Fatalf("AccountNumberMasked = %q, want masked value distinct from full number", details.AccountNumberMasked)
	}
}

func TestEnrichAccountNumberSummaryAddsAccountName(t *testing.T) {
	summary := enrichAccountNumberSummary(AccountNumberSummary{
		ID:        "account_number_123",
		AccountID: "account_123",
		Name:      "Revenue",
	}, map[string]string{"account_123": "Operating"})

	if summary.AccountName != "Operating" {
		t.Fatalf("AccountName = %q, want Operating", summary.AccountName)
	}
}

func TestEnrichAccountNumberDetailsLeavesMissingAccountNameEmpty(t *testing.T) {
	details := enrichAccountNumberDetails(AccountNumberDetails{
		AccountNumberSummary: AccountNumberSummary{
			ID:        "account_number_123",
			AccountID: "account_123",
			Name:      "Revenue",
		},
	}, map[string]string{})

	if details.AccountName != "" {
		t.Fatalf("AccountName = %q, want empty when parent account lookup is unavailable", details.AccountName)
	}
}

func TestNormalizeProgramIncludesDefaultCardProfile(t *testing.T) {
	program := normalizeProgram(increase.Program{
		ID:                          "program_123",
		Name:                        "Treasury",
		Bank:                        increase.ProgramBankCoreBank,
		DefaultDigitalCardProfileID: "digital_card_profile_123",
		InterestRate:                "0.01",
		CreatedAt:                   time.Unix(0, 0),
		UpdatedAt:                   time.Unix(60, 0),
		Lending: increase.ProgramLending{
			MaximumExtendableCredit: 5000,
		},
	})

	if program.DefaultDigitalCardProfileID != "digital_card_profile_123" {
		t.Fatalf("DefaultDigitalCardProfileID = %q, want digital_card_profile_123", program.DefaultDigitalCardProfileID)
	}
	if program.MaximumExtendableCredit != 5000 {
		t.Fatalf("MaximumExtendableCredit = %d, want 5000", program.MaximumExtendableCredit)
	}
	if program.Bank != "core_bank" {
		t.Fatalf("Bank = %q, want core_bank", program.Bank)
	}
}

func TestNormalizeDocumentPreservesOperationalFields(t *testing.T) {
	document := normalizeDocument(increasex.Document{
		ID:        "document_123",
		Category:  "funding_instructions",
		EntityID:  "entity_123",
		FileID:    "file_123",
		CreatedAt: time.Unix(0, 0),
		FundingInstructions: map[string]any{
			"account_number_id": "account_number_123",
		},
	})

	if document.FileID != "file_123" {
		t.Fatalf("FileID = %q, want file_123", document.FileID)
	}
	if document.Category != "funding_instructions" {
		t.Fatalf("Category = %q, want funding_instructions", document.Category)
	}
	if document.FundingInstructions["account_number_id"] != "account_number_123" {
		t.Fatalf("FundingInstructions = %#v, want account_number_id detail", document.FundingInstructions)
	}
}

func TestParseOptionalRFC3339BoundsRejectsInvalidOrdering(t *testing.T) {
	_, _, err := parseOptionalRFC3339Bounds("2026-03-15T00:00:00Z", "2026-03-01T00:00:00Z", "since", "until")
	if err == nil || !strings.Contains(err.Error(), "since must be before or equal to until") {
		t.Fatalf("parseOptionalRFC3339Bounds() error = %v, want ordering validation error", err)
	}
}

func TestPreviewCreateAccountNumberRejectsMissingFields(t *testing.T) {
	services := NewServices()
	_, err := services.PreviewCreateAccountNumber(Session{ProfileName: "default", Environment: "sandbox"}, CreateAccountNumberInput{})
	if err == nil {
		t.Fatal("PreviewCreateAccountNumber() error = nil, want validation error")
	}
	var detail *util.ErrorDetail
	if !errors.As(err, &detail) {
		t.Fatalf("PreviewCreateAccountNumber() error = %T, want *util.ErrorDetail", err)
	}
	if len(detail.Fields) != 2 {
		t.Fatalf("validation fields = %#v, want account_id and name", detail.Fields)
	}
}

func TestPreviewDisableAccountNumberSummary(t *testing.T) {
	services := NewServices()
	preview, err := services.PreviewDisableAccountNumber(Session{ProfileName: "default", Environment: "sandbox"}, DisableAccountNumberInput{
		AccountNumberID: "account_number_123",
	})
	if err != nil {
		t.Fatalf("PreviewDisableAccountNumber() error = %v", err)
	}
	if preview.Summary != "Disable account number account_number_123" {
		t.Fatalf("PreviewDisableAccountNumber() summary = %q", preview.Summary)
	}
	if preview.ConfirmationToken == "" {
		t.Fatal("PreviewDisableAccountNumber() confirmation token = empty, want token")
	}
}

func TestTransferSourceIdentifierPayloadsMatchRailRequirements(t *testing.T) {
	services := NewServices()
	session := Session{ProfileName: "default", Environment: "sandbox"}

	achPreview, err := services.PreviewExternalACH(session, ACHTransferInput{
		AccountID:   "account_123",
		AmountCents: 100,
	})
	if err != nil {
		t.Fatalf("PreviewExternalACH() error = %v", err)
	}
	if got := achPreview.Details["account_id"]; got != "account_123" {
		t.Fatalf("ACH preview account_id = %v, want account_123", got)
	}
	if _, ok := achPreview.Details["source_account_number_id"]; ok {
		t.Fatalf("ACH preview should not include source_account_number_id: %#v", achPreview.Details)
	}

	rtpPreview, err := services.PreviewExternalRTP(session, RTPTransferInput{
		AmountCents:           100,
		CreditorName:          "Vendor",
		SourceAccountNumberID: "account_number_123",
	})
	if err != nil {
		t.Fatalf("PreviewExternalRTP() error = %v", err)
	}
	if got := rtpPreview.Details["source_account_number_id"]; got != "account_number_123" {
		t.Fatalf("RTP preview source_account_number_id = %v, want account_number_123", got)
	}

	fedNowPreview, err := services.PreviewExternalFedNow(session, FedNowTransferInput{
		AccountID:             "account_456",
		AmountCents:           100,
		CreditorName:          "Vendor",
		DebtorName:            "Debtor",
		SourceAccountNumberID: "account_number_456",
	})
	if err != nil {
		t.Fatalf("PreviewExternalFedNow() error = %v", err)
	}
	if got := fedNowPreview.Details["account_id"]; got != "account_456" {
		t.Fatalf("FedNow preview account_id = %v, want account_456", got)
	}
	if got := fedNowPreview.Details["source_account_number_id"]; got != "account_number_456" {
		t.Fatalf("FedNow preview source_account_number_id = %v, want account_number_456", got)
	}

	wirePreview, err := services.PreviewExternalWire(session, WireTransferInput{
		AccountID:             "account_789",
		AmountCents:           100,
		BeneficiaryName:       "Vendor",
		SourceAccountNumberID: "account_number_789",
	})
	if err != nil {
		t.Fatalf("PreviewExternalWire() error = %v", err)
	}
	if got := wirePreview.Details["source_account_number_id"]; got != "account_number_789" {
		t.Fatalf("Wire preview source_account_number_id = %v, want account_number_789", got)
	}
}

func TestExecuteExternalTransfersRequireSourceAccountNumberWhenRailDemandsIt(t *testing.T) {
	services := NewServices()
	session := Session{ProfileName: "default", Environment: "sandbox"}

	rtpInput := RTPTransferInput{
		AmountCents:  100,
		CreditorName: "Vendor",
	}
	rtpPreview, err := services.PreviewExternalRTP(session, rtpInput)
	if err != nil {
		t.Fatalf("PreviewExternalRTP() error = %v", err)
	}
	rtpInput.ConfirmationToken = rtpPreview.ConfirmationToken
	if _, _, err := services.ExecuteExternalRTP(context.Background(), nil, session, rtpInput); err == nil || !strings.Contains(err.Error(), "source_account_number_id") {
		t.Fatalf("ExecuteExternalRTP() error = %v, want source_account_number_id guidance", err)
	}

	fedNowInput := FedNowTransferInput{
		AccountID:    "account_123",
		AmountCents:  100,
		CreditorName: "Vendor",
		DebtorName:   "Debtor",
	}
	fedNowPreview, err := services.PreviewExternalFedNow(session, fedNowInput)
	if err != nil {
		t.Fatalf("PreviewExternalFedNow() error = %v", err)
	}
	fedNowInput.ConfirmationToken = fedNowPreview.ConfirmationToken
	if _, _, err := services.ExecuteExternalFedNow(context.Background(), nil, session, fedNowInput); err == nil || !strings.Contains(err.Error(), "source_account_number_id") {
		t.Fatalf("ExecuteExternalFedNow() error = %v, want source_account_number_id guidance", err)
	}
}
