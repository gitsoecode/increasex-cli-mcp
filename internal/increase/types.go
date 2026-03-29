package increasex

import (
	"net/url"
	"strconv"
	"strings"
	"time"
)

type FedNowAddress struct {
	Line1      string `json:"line1,omitempty"`
	Line2      string `json:"line2,omitempty"`
	City       string `json:"city,omitempty"`
	State      string `json:"state,omitempty"`
	PostalCode string `json:"postal_code,omitempty"`
}

type FedNowTransferNewParams struct {
	AccountID                         string         `json:"account_id"`
	AccountNumber                     string         `json:"account_number,omitempty"`
	Amount                            int64          `json:"amount"`
	CreditorAddress                   *FedNowAddress `json:"creditor_address,omitempty"`
	CreditorName                      string         `json:"creditor_name"`
	DebtorAddress                     *FedNowAddress `json:"debtor_address,omitempty"`
	DebtorName                        string         `json:"debtor_name"`
	ExternalAccountID                 string         `json:"external_account_id,omitempty"`
	RequireApproval                   *bool          `json:"require_approval,omitempty"`
	RoutingNumber                     string         `json:"routing_number,omitempty"`
	SourceAccountNumberID             string         `json:"source_account_number_id"`
	UnstructuredRemittanceInformation string         `json:"unstructured_remittance_information"`
}

type FedNowTransfer struct {
	ID                                string    `json:"id"`
	AccountID                         string    `json:"account_id"`
	Amount                            int64     `json:"amount"`
	CreatedAt                         time.Time `json:"created_at"`
	Status                            string    `json:"status"`
	CreditorName                      string    `json:"creditor_name"`
	DebtorName                        string    `json:"debtor_name"`
	RoutingNumber                     string    `json:"routing_number"`
	AccountNumber                     string    `json:"account_number"`
	ExternalAccountID                 string    `json:"external_account_id"`
	SourceAccountNumberID             string    `json:"source_account_number_id"`
	UnstructuredRemittanceInformation string    `json:"unstructured_remittance_information"`
}

type CardCreateRawRequest struct {
	AccountID      string                 `json:"account_id"`
	BillingAddress map[string]any         `json:"billing_address,omitempty"`
	CardProgram    string                 `json:"card_program,omitempty"`
	Description    string                 `json:"description,omitempty"`
	DigitalWallet  map[string]any         `json:"digital_wallet,omitempty"`
	EntityID       string                 `json:"entity_id,omitempty"`
	Extra          map[string]interface{} `json:"-"`
}

type Document struct {
	AccountVerificationLetter map[string]any `json:"account_verification_letter,omitempty"`
	Category                  string         `json:"category"`
	CreatedAt                 time.Time      `json:"created_at"`
	EntityID                  string         `json:"entity_id,omitempty"`
	FileID                    string         `json:"file_id"`
	FundingInstructions       map[string]any `json:"funding_instructions,omitempty"`
	ID                        string         `json:"id"`
	IdempotencyKey            string         `json:"idempotency_key,omitempty"`
	Type                      string         `json:"type,omitempty"`
}

type DocumentListParams struct {
	Cursor            string
	EntityID          string
	Categories        []string
	IdempotencyKey    string
	Limit             int64
	CreatedAfter      *time.Time
	CreatedBefore     *time.Time
	CreatedOnOrAfter  *time.Time
	CreatedOnOrBefore *time.Time
}

func (p DocumentListParams) URLQuery() url.Values {
	values := url.Values{}
	if p.Cursor != "" {
		values.Set("cursor", p.Cursor)
	}
	if p.EntityID != "" {
		values.Set("entity_id", p.EntityID)
	}
	if len(p.Categories) > 0 {
		values.Set("category.in", strings.Join(p.Categories, ","))
	}
	if p.IdempotencyKey != "" {
		values.Set("idempotency_key", p.IdempotencyKey)
	}
	if p.Limit > 0 {
		values.Set("limit", strconv.FormatInt(p.Limit, 10))
	}
	addQueryTime(values, "created_at.after", p.CreatedAfter)
	addQueryTime(values, "created_at.before", p.CreatedBefore)
	addQueryTime(values, "created_at.on_or_after", p.CreatedOnOrAfter)
	addQueryTime(values, "created_at.on_or_before", p.CreatedOnOrBefore)
	return values
}

type documentListResponse struct {
	Data []Document `json:"data"`
}

func addQueryTime(values url.Values, key string, value *time.Time) {
	if value == nil {
		return
	}
	values.Set(key, value.UTC().Format(time.RFC3339))
}
