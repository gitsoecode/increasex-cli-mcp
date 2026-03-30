package app

type ExternalAccountSummary struct {
	ID                  string `json:"id"`
	Description         string `json:"description"`
	AccountHolder       string `json:"account_holder,omitempty"`
	Funding             string `json:"funding,omitempty"`
	RoutingNumber       string `json:"routing_number,omitempty"`
	AccountNumberMasked string `json:"account_number_masked,omitempty"`
	Status              string `json:"status,omitempty"`
	CreatedAt           string `json:"created_at,omitempty"`
}

type CreateExternalAccountInput struct {
	AccountNumber     string `json:"account_number"`
	Description       string `json:"description"`
	RoutingNumber     string `json:"routing_number"`
	AccountHolder     string `json:"account_holder,omitempty"`
	Funding           string `json:"funding,omitempty"`
	IdempotencyKey    string `json:"idempotency_key,omitempty"`
	DryRun            *bool  `json:"dry_run,omitempty"`
	ConfirmationToken string `json:"confirmation_token,omitempty"`
}

type UpdateExternalAccountInput struct {
	ExternalAccountID string `json:"external_account_id"`
	AccountHolder     string `json:"account_holder,omitempty"`
	Description       string `json:"description,omitempty"`
	Funding           string `json:"funding,omitempty"`
	Status            string `json:"status,omitempty"`
	IdempotencyKey    string `json:"idempotency_key,omitempty"`
	DryRun            *bool  `json:"dry_run,omitempty"`
	ConfirmationToken string `json:"confirmation_token,omitempty"`
}

type CardDetailsInput struct {
	CardID string `json:"card_id"`
}

type CardDetailsSummary = CardSummary

type CreateCardDetailsIframeInput struct {
	CardID         string `json:"card_id"`
	PhysicalCardID string `json:"physical_card_id,omitempty"`
}

type CardDetailsIframeResult struct {
	CardID    string `json:"card_id"`
	IframeURL string `json:"iframe_url"`
	ExpiresAt string `json:"expires_at"`
}

type UpdateCardPINInput struct {
	CardID            string `json:"card_id"`
	PIN               string `json:"pin"`
	DryRun            *bool  `json:"dry_run,omitempty"`
	ConfirmationToken string `json:"confirmation_token,omitempty"`
}

type TransferSummary struct {
	Rail                 string `json:"rail"`
	ID                   string `json:"id"`
	AccountID            string `json:"account_id,omitempty"`
	AmountCents          int64  `json:"amount_cents"`
	Status               string `json:"status"`
	CreatedAt            string `json:"created_at,omitempty"`
	ExternalAccountID    string `json:"external_account_id,omitempty"`
	PendingTransactionID string `json:"pending_transaction_id,omitempty"`
	Counterparty         string `json:"counterparty,omitempty"`
}

type ListTransfersInput struct {
	Rail              string `json:"rail"`
	AccountID         string `json:"account_id,omitempty"`
	ExternalAccountID string `json:"external_account_id,omitempty"`
	Status            string `json:"status,omitempty"`
	Since             string `json:"since,omitempty"`
	Cursor            string `json:"cursor,omitempty"`
	Limit             int64  `json:"limit,omitempty"`
}

type TransferActionInput struct {
	Rail              string `json:"rail"`
	TransferID        string `json:"transfer_id"`
	DryRun            *bool  `json:"dry_run,omitempty"`
	ConfirmationToken string `json:"confirmation_token,omitempty"`
}
