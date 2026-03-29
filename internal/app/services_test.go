package app

import (
	"testing"
	"time"

	increase "github.com/increase/increase-go"
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
