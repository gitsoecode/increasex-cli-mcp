package increasex

import "time"

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
