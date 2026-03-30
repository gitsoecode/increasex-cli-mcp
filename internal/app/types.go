package app

import "encoding/json"

type Session struct {
	ProfileName string
	Environment string
	TokenSource string
}

type AccountSummary struct {
	ID                    string `json:"id"`
	Name                  string `json:"name"`
	Status                string `json:"status"`
	EntityID              string `json:"entity_id,omitempty"`
	InformationalEntityID string `json:"informational_entity_id,omitempty"`
	ProgramID             string `json:"program_id,omitempty"`
	CreatedAt             string `json:"created_at,omitempty"`
}

type AccountNumberSummary struct {
	ID                  string              `json:"id"`
	AccountID           string              `json:"account_id"`
	AccountName         string              `json:"account_name,omitempty"`
	Name                string              `json:"name"`
	RoutingNumber       string              `json:"routing_number,omitempty"`
	AccountNumberMasked string              `json:"account_number_masked,omitempty"`
	Status              string              `json:"status,omitempty"`
	CreatedAt           string              `json:"created_at,omitempty"`
	InboundACH          *InboundACHInput    `json:"inbound_ach,omitempty"`
	InboundChecks       *InboundChecksInput `json:"inbound_checks,omitempty"`
}

type AccountNumberDetails struct {
	AccountNumberSummary
	IdempotencyKey string `json:"idempotency_key,omitempty"`
	AccountNumber  string `json:"account_number,omitempty"`
}

type ProgramSummary struct {
	ID                          string `json:"id"`
	Name                        string `json:"name"`
	Bank                        string `json:"bank"`
	BillingAccountID            string `json:"billing_account_id,omitempty"`
	DefaultDigitalCardProfileID string `json:"default_digital_card_profile_id,omitempty"`
	InterestRate                string `json:"interest_rate,omitempty"`
	MaximumExtendableCredit     int64  `json:"maximum_extendable_credit,omitempty"`
	CreatedAt                   string `json:"created_at,omitempty"`
	UpdatedAt                   string `json:"updated_at,omitempty"`
}

type DigitalCardProfileTextColorSummary struct {
	Red   int64 `json:"red"`
	Green int64 `json:"green"`
	Blue  int64 `json:"blue"`
}

type DigitalCardProfileSummary struct {
	ID                    string                             `json:"id"`
	Description           string                             `json:"description"`
	CardDescription       string                             `json:"card_description,omitempty"`
	IssuerName            string                             `json:"issuer_name,omitempty"`
	Status                string                             `json:"status,omitempty"`
	AppIconFileID         string                             `json:"app_icon_file_id,omitempty"`
	BackgroundImageFileID string                             `json:"background_image_file_id,omitempty"`
	ContactEmail          string                             `json:"contact_email,omitempty"`
	ContactPhone          string                             `json:"contact_phone,omitempty"`
	ContactWebsite        string                             `json:"contact_website,omitempty"`
	TextColor             DigitalCardProfileTextColorSummary `json:"text_color"`
	CreatedAt             string                             `json:"created_at,omitempty"`
}

type BalanceSummary struct {
	AccountID        string `json:"account_id"`
	CurrentBalance   int64  `json:"current_balance"`
	AvailableBalance int64  `json:"available_balance"`
}

type TransactionSummary struct {
	ID                  string `json:"id"`
	AccountID           string `json:"account_id,omitempty"`
	AmountCents         int64  `json:"amount_cents"`
	Direction           string `json:"direction"`
	Description         string `json:"description,omitempty"`
	Type                string `json:"type"`
	CreatedAt           string `json:"created_at"`
	RouteID             string `json:"route_id,omitempty"`
	RouteType           string `json:"route_type,omitempty"`
	CounterpartySummary string `json:"counterparty_summary,omitempty"`
}

type EventSummary struct {
	ID                   string `json:"id"`
	AssociatedObjectID   string `json:"associated_object_id,omitempty"`
	AssociatedObjectType string `json:"associated_object_type,omitempty"`
	Category             string `json:"category"`
	CreatedAt            string `json:"created_at,omitempty"`
}

type TransactionTimeRangeInput struct {
	Since  string `json:"since,omitempty"`
	Until  string `json:"until,omitempty"`
	Period string `json:"period,omitempty"`
}

type ListTransactionsInput struct {
	AccountID  string                    `json:"account_id,omitempty"`
	TimeRange  TransactionTimeRangeInput `json:"time_range,omitempty"`
	Cursor     string                    `json:"cursor,omitempty"`
	Limit      int64                     `json:"limit,omitempty"`
	Categories []string                  `json:"categories,omitempty"`
}

type ListEventsInput struct {
	AssociatedObjectID string                    `json:"associated_object_id,omitempty"`
	TimeRange          TransactionTimeRangeInput `json:"time_range,omitempty"`
	Cursor             string                    `json:"cursor,omitempty"`
	Limit              int64                     `json:"limit,omitempty"`
	Categories         []string                  `json:"categories,omitempty"`
}

type DocumentSummary struct {
	ID                        string         `json:"id"`
	Category                  string         `json:"category"`
	EntityID                  string         `json:"entity_id,omitempty"`
	FileID                    string         `json:"file_id,omitempty"`
	IdempotencyKey            string         `json:"idempotency_key,omitempty"`
	CreatedAt                 string         `json:"created_at,omitempty"`
	AccountVerificationLetter map[string]any `json:"account_verification_letter,omitempty"`
	FundingInstructions       map[string]any `json:"funding_instructions,omitempty"`
}

type ListDocumentsInput struct {
	EntityID       string                    `json:"entity_id,omitempty"`
	TimeRange      TransactionTimeRangeInput `json:"time_range,omitempty"`
	Cursor         string                    `json:"cursor,omitempty"`
	Limit          int64                     `json:"limit,omitempty"`
	Categories     []string                  `json:"categories,omitempty"`
	IdempotencyKey string                    `json:"idempotency_key,omitempty"`
}

type CardSummary struct {
	ID                   string              `json:"id"`
	AccountID            string              `json:"account_id"`
	Last4                string              `json:"last4,omitempty"`
	Status               string              `json:"status"`
	Description          string              `json:"description,omitempty"`
	EntityID             string              `json:"entity_id,omitempty"`
	ExpirationMonth      int64               `json:"expiration_month,omitempty"`
	ExpirationYear       int64               `json:"expiration_year,omitempty"`
	CreatedAt            string              `json:"created_at,omitempty"`
	BillingDetails       *CardBillingDetails `json:"billing_details,omitempty"`
	PrimaryAccountNumber string              `json:"primary_account_number,omitempty"`
	VerificationCode     string              `json:"verification_code,omitempty"`
	PIN                  string              `json:"pin,omitempty"`
}

type CardBillingDetails struct {
	City       string `json:"city,omitempty"`
	Line1      string `json:"line1,omitempty"`
	Line2      string `json:"line2,omitempty"`
	PostalCode string `json:"postal_code,omitempty"`
	State      string `json:"state,omitempty"`
}

type PreviewResult struct {
	Mode              string         `json:"mode"`
	Summary           string         `json:"summary"`
	ConfirmationToken string         `json:"confirmation_token"`
	Details           map[string]any `json:"details,omitempty"`
}

type ExecutedResult struct {
	Mode    string         `json:"mode"`
	Details map[string]any `json:"details"`
}

type CreateAccountInput struct {
	Name                  string `json:"name"`
	EntityID              string `json:"entity_id,omitempty"`
	InformationalEntityID string `json:"informational_entity_id,omitempty"`
	ProgramID             string `json:"program_id,omitempty"`
	IdempotencyKey        string `json:"idempotency_key,omitempty"`
	DryRun                *bool  `json:"dry_run,omitempty"`
	ConfirmationToken     string `json:"confirmation_token,omitempty"`
}

type CloseAccountInput struct {
	AccountID         string `json:"account_id"`
	IdempotencyKey    string `json:"idempotency_key,omitempty"`
	DryRun            *bool  `json:"dry_run,omitempty"`
	ConfirmationToken string `json:"confirmation_token,omitempty"`
}

type CreateAccountNumberInput struct {
	AccountID         string              `json:"account_id"`
	Name              string              `json:"name"`
	InboundACH        *InboundACHInput    `json:"inbound_ach,omitempty"`
	InboundChecks     *InboundChecksInput `json:"inbound_checks,omitempty"`
	IdempotencyKey    string              `json:"idempotency_key,omitempty"`
	DryRun            *bool               `json:"dry_run,omitempty"`
	ConfirmationToken string              `json:"confirmation_token,omitempty"`
}

type InboundACHInput struct {
	DebitStatus string `json:"debit_status"`
}

type InboundChecksInput struct {
	Status string `json:"status"`
}

type DisableAccountNumberInput struct {
	AccountNumberID   string `json:"account_number_id"`
	IdempotencyKey    string `json:"idempotency_key,omitempty"`
	DryRun            *bool  `json:"dry_run,omitempty"`
	ConfirmationToken string `json:"confirmation_token,omitempty"`
}

type MoveMoneyInternalInput struct {
	FromAccountID     string `json:"from_account_id"`
	ToAccountID       string `json:"to_account_id"`
	AmountCents       int64  `json:"amount_cents"`
	Description       string `json:"description,omitempty"`
	RequireApproval   *bool  `json:"require_approval,omitempty"`
	IdempotencyKey    string `json:"idempotency_key,omitempty"`
	DryRun            *bool  `json:"dry_run,omitempty"`
	ConfirmationToken string `json:"confirmation_token,omitempty"`
}

type BillingAddressInput struct {
	City       string `json:"city"`
	Line1      string `json:"line1"`
	Line2      string `json:"line2,omitempty"`
	PostalCode string `json:"postal_code"`
	State      string `json:"state"`
}

type DigitalWalletInput struct {
	DigitalCardProfileID string `json:"digital_card_profile_id,omitempty"`
	Email                string `json:"email,omitempty"`
	Phone                string `json:"phone,omitempty"`
}

type CreateCardInput struct {
	AccountID         string               `json:"account_id"`
	Description       string               `json:"description,omitempty"`
	BillingAddress    *BillingAddressInput `json:"billing_address,omitempty"`
	CardProgram       string               `json:"card_program,omitempty"`
	DigitalWallet     *DigitalWalletInput  `json:"digital_wallet,omitempty"`
	EntityID          string               `json:"entity_id,omitempty"`
	IdempotencyKey    string               `json:"idempotency_key,omitempty"`
	DryRun            *bool                `json:"dry_run,omitempty"`
	ConfirmationToken string               `json:"confirmation_token,omitempty"`
}

type ACHTransferInput struct {
	AccountID                string `json:"account_id"`
	AmountCents              int64  `json:"amount_cents"`
	StatementDescriptor      string `json:"statement_descriptor"`
	AccountNumber            string `json:"account_number,omitempty"`
	RoutingNumber            string `json:"routing_number,omitempty"`
	ExternalAccountID        string `json:"external_account_id,omitempty"`
	Funding                  string `json:"funding,omitempty"`
	DestinationAccountHolder string `json:"destination_account_holder,omitempty"`
	IndividualID             string `json:"individual_id,omitempty"`
	IndividualName           string `json:"individual_name,omitempty"`
	CompanyName              string `json:"company_name,omitempty"`
	CompanyEntryDescription  string `json:"company_entry_description,omitempty"`
	CompanyDescriptiveDate   string `json:"company_descriptive_date,omitempty"`
	CompanyDiscretionaryData string `json:"company_discretionary_data,omitempty"`
	RequireApproval          *bool  `json:"require_approval,omitempty"`
	IdempotencyKey           string `json:"idempotency_key,omitempty"`
	DryRun                   *bool  `json:"dry_run,omitempty"`
	ConfirmationToken        string `json:"confirmation_token,omitempty"`
}

type RTPTransferInput struct {
	AmountCents              int64  `json:"amount_cents"`
	CreditorName             string `json:"creditor_name"`
	RemittanceInformation    string `json:"remittance_information"`
	SourceAccountNumberID    string `json:"source_account_number_id"`
	DebtorName               string `json:"debtor_name,omitempty"`
	DestinationAccountNumber string `json:"destination_account_number,omitempty"`
	DestinationRoutingNumber string `json:"destination_routing_number,omitempty"`
	ExternalAccountID        string `json:"external_account_id,omitempty"`
	UltimateCreditorName     string `json:"ultimate_creditor_name,omitempty"`
	UltimateDebtorName       string `json:"ultimate_debtor_name,omitempty"`
	RequireApproval          *bool  `json:"require_approval,omitempty"`
	IdempotencyKey           string `json:"idempotency_key,omitempty"`
	DryRun                   *bool  `json:"dry_run,omitempty"`
	ConfirmationToken        string `json:"confirmation_token,omitempty"`
}

type FedNowAddressInput struct {
	Line1      string `json:"line1,omitempty"`
	Line2      string `json:"line2,omitempty"`
	City       string `json:"city,omitempty"`
	State      string `json:"state,omitempty"`
	PostalCode string `json:"postal_code,omitempty"`
}

type FedNowTransferInput struct {
	AccountID                         string              `json:"account_id"`
	AmountCents                       int64               `json:"amount_cents"`
	CreditorName                      string              `json:"creditor_name"`
	DebtorName                        string              `json:"debtor_name"`
	SourceAccountNumberID             string              `json:"source_account_number_id"`
	UnstructuredRemittanceInformation string              `json:"unstructured_remittance_information"`
	AccountNumber                     string              `json:"account_number,omitempty"`
	RoutingNumber                     string              `json:"routing_number,omitempty"`
	ExternalAccountID                 string              `json:"external_account_id,omitempty"`
	CreditorAddress                   *FedNowAddressInput `json:"creditor_address,omitempty"`
	DebtorAddress                     *FedNowAddressInput `json:"debtor_address,omitempty"`
	RequireApproval                   *bool               `json:"require_approval,omitempty"`
	IdempotencyKey                    string              `json:"idempotency_key,omitempty"`
	DryRun                            *bool               `json:"dry_run,omitempty"`
	ConfirmationToken                 string              `json:"confirmation_token,omitempty"`
}

type WireTransferInput struct {
	AccountID               string `json:"account_id"`
	AmountCents             int64  `json:"amount_cents"`
	BeneficiaryName         string `json:"beneficiary_name"`
	MessageToRecipient      string `json:"message_to_recipient"`
	SourceAccountNumberID   string `json:"source_account_number_id,omitempty"`
	AccountNumber           string `json:"account_number,omitempty"`
	RoutingNumber           string `json:"routing_number,omitempty"`
	ExternalAccountID       string `json:"external_account_id,omitempty"`
	BeneficiaryAddressLine1 string `json:"beneficiary_address_line1,omitempty"`
	BeneficiaryAddressLine2 string `json:"beneficiary_address_line2,omitempty"`
	BeneficiaryAddressLine3 string `json:"beneficiary_address_line3,omitempty"`
	OriginatorName          string `json:"originator_name,omitempty"`
	OriginatorAddressLine1  string `json:"originator_address_line1,omitempty"`
	OriginatorAddressLine2  string `json:"originator_address_line2,omitempty"`
	OriginatorAddressLine3  string `json:"originator_address_line3,omitempty"`
	RequireApproval         *bool  `json:"require_approval,omitempty"`
	IdempotencyKey          string `json:"idempotency_key,omitempty"`
	DryRun                  *bool  `json:"dry_run,omitempty"`
	ConfirmationToken       string `json:"confirmation_token,omitempty"`
}

func IsDryRun(value *bool) bool {
	return value == nil || *value
}

func CloneMap(value any) map[string]any {
	raw, _ := json.Marshal(value)
	out := map[string]any{}
	_ = json.Unmarshal(raw, &out)
	return out
}
