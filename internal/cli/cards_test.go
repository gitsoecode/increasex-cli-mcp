package cli

import (
	"testing"

	"github.com/gitsoecode/increasex-cli-mcp/internal/app"
)

func TestPromptCreateCardFieldsPromptsForOptionalSections(t *testing.T) {
	originalPromptString := promptCardString
	originalPromptBool := promptCardBool
	originalSelectAccount := selectCardAccount
	defer func() {
		promptCardString = originalPromptString
		promptCardBool = originalPromptBool
		selectCardAccount = originalSelectAccount
	}()

	stringAnswers := map[string]string{
		"Description (optional)":             "team card",
		"Card program (optional)":            "card_program_123",
		"Entity id (optional)":               "entity_123",
		"Billing address line 1":             "123 Test St",
		"Billing address line 2 (optional)":  "Suite 100",
		"Billing city":                       "San Francisco",
		"Billing state":                      "CA",
		"Billing postal code":                "94107",
		"Digital card profile id (optional)": "",
		"Wallet email (optional)":            "wallet@example.com",
		"Wallet phone (optional)":            "",
	}
	boolAnswers := map[string]bool{
		"Add billing address?":        true,
		"Add digital wallet details?": true,
	}
	promptCardString = func(label string, required bool) (string, error) {
		return stringAnswers[label], nil
	}
	promptCardBool = func(label string, yesLabel, noLabel string) (bool, error) {
		return boolAnswers[label], nil
	}
	selectCardAccount = func(accounts []app.AccountSummary, label string) (string, error) {
		return accounts[0].ID, nil
	}

	input := &app.CreateCardInput{}
	var billing *app.BillingAddressInput
	var wallet *app.DigitalWalletInput

	err := promptCreateCardFields([]app.AccountSummary{{ID: "account_123", Name: "Operating"}}, input, &billing, &wallet)
	if err != nil {
		t.Fatalf("promptCreateCardFields() error = %v", err)
	}
	if input.AccountID != "account_123" {
		t.Fatalf("account id = %q, want %q", input.AccountID, "account_123")
	}
	if billing == nil || billing.Line1 != "123 Test St" || billing.PostalCode != "94107" {
		t.Fatalf("billing = %#v, want populated billing address", billing)
	}
	if wallet == nil || wallet.Email != "wallet@example.com" {
		t.Fatalf("wallet = %#v, want populated digital wallet", wallet)
	}
}

func TestPromptCreateCardFieldsLeavesOptionalSectionsEmptyWhenSkipped(t *testing.T) {
	originalPromptString := promptCardString
	originalPromptBool := promptCardBool
	originalSelectAccount := selectCardAccount
	defer func() {
		promptCardString = originalPromptString
		promptCardBool = originalPromptBool
		selectCardAccount = originalSelectAccount
	}()

	promptCardString = func(label string, required bool) (string, error) {
		return "", nil
	}
	promptCardBool = func(label string, yesLabel, noLabel string) (bool, error) {
		return false, nil
	}
	selectCardAccount = func(accounts []app.AccountSummary, label string) (string, error) {
		return accounts[0].ID, nil
	}

	input := &app.CreateCardInput{}
	var billing *app.BillingAddressInput
	var wallet *app.DigitalWalletInput

	err := promptCreateCardFields([]app.AccountSummary{{ID: "account_123", Name: "Operating"}}, input, &billing, &wallet)
	if err != nil {
		t.Fatalf("promptCreateCardFields() error = %v", err)
	}
	if billing != nil {
		t.Fatalf("billing = %#v, want nil when section skipped", billing)
	}
	if wallet != nil {
		t.Fatalf("wallet = %#v, want nil when section skipped", wallet)
	}
}
