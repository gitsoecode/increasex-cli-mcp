package app

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/gitsoecode/increasex-cli-mcp/internal/util"
)

func TestTransferWriteValidationRejectsInvalidInputBeforePreviewAndExecute(t *testing.T) {
	services := NewServices()
	session := Session{ProfileName: "default", Environment: "sandbox"}

	type validationCase struct {
		name       string
		preview    func() (*PreviewResult, error)
		execute    func() (any, string, error)
		fieldNames []string
	}

	cases := []validationCase{
		{
			name: "internal transfer",
			preview: func() (*PreviewResult, error) {
				return services.PreviewInternalTransfer(session, MoveMoneyInternalInput{})
			},
			execute: func() (any, string, error) {
				return services.ExecuteInternalTransfer(context.Background(), nil, session, MoveMoneyInternalInput{})
			},
			fieldNames: []string{"from_account_id", "to_account_id", "amount_cents"},
		},
		{
			name: "ach transfer",
			preview: func() (*PreviewResult, error) {
				return services.PreviewExternalACH(session, ACHTransferInput{
					AccountID: "account_123",
				})
			},
			execute: func() (any, string, error) {
				return services.ExecuteExternalACH(context.Background(), nil, session, ACHTransferInput{
					AccountID: "account_123",
				})
			},
			fieldNames: []string{"amount_cents", "statement_descriptor", "account_number", "routing_number"},
		},
		{
			name: "rtp transfer",
			preview: func() (*PreviewResult, error) {
				return services.PreviewExternalRTP(session, RTPTransferInput{
					AmountCents:           100,
					CreditorName:          "Vendor",
					SourceAccountNumberID: "account_number_123",
				})
			},
			execute: func() (any, string, error) {
				return services.ExecuteExternalRTP(context.Background(), nil, session, RTPTransferInput{
					AmountCents:           100,
					CreditorName:          "Vendor",
					SourceAccountNumberID: "account_number_123",
				})
			},
			fieldNames: []string{"remittance_information", "destination_account_number", "destination_routing_number"},
		},
		{
			name: "fednow transfer",
			preview: func() (*PreviewResult, error) {
				return services.PreviewExternalFedNow(session, FedNowTransferInput{
					AccountID:             "account_123",
					AmountCents:           100,
					CreditorName:          "Vendor",
					DebtorName:            "Debtor",
					SourceAccountNumberID: "account_number_123",
				})
			},
			execute: func() (any, string, error) {
				return services.ExecuteExternalFedNow(context.Background(), nil, session, FedNowTransferInput{
					AccountID:             "account_123",
					AmountCents:           100,
					CreditorName:          "Vendor",
					DebtorName:            "Debtor",
					SourceAccountNumberID: "account_number_123",
				})
			},
			fieldNames: []string{"unstructured_remittance_information", "account_number", "routing_number"},
		},
		{
			name: "wire transfer",
			preview: func() (*PreviewResult, error) {
				return services.PreviewExternalWire(session, WireTransferInput{
					AccountID:       "account_123",
					AmountCents:     100,
					BeneficiaryName: "Vendor",
				})
			},
			execute: func() (any, string, error) {
				return services.ExecuteExternalWire(context.Background(), nil, session, WireTransferInput{
					AccountID:       "account_123",
					AmountCents:     100,
					BeneficiaryName: "Vendor",
				})
			},
			fieldNames: []string{"account_number", "routing_number"},
		},
		{
			name: "approve action",
			preview: func() (*PreviewResult, error) {
				return services.PreviewApproveTransfer(session, TransferActionInput{
					Rail: "rtp",
				})
			},
			execute: func() (any, string, error) {
				return services.ExecuteApproveTransfer(context.Background(), nil, session, TransferActionInput{
					Rail: "rtp",
				})
			},
			fieldNames: []string{"transfer_id"},
		},
		{
			name: "cancel action",
			preview: func() (*PreviewResult, error) {
				return services.PreviewCancelTransfer(session, TransferActionInput{
					Rail: "ach",
				})
			},
			execute: func() (any, string, error) {
				return services.ExecuteCancelTransfer(context.Background(), nil, session, TransferActionInput{
					Rail: "ach",
				})
			},
			fieldNames: []string{"transfer_id"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			preview, err := tc.preview()
			assertTransferValidationError(t, err, tc.fieldNames...)
			if preview != nil {
				t.Fatalf("preview = %#v, want nil on validation failure", preview)
			}

			_, _, err = tc.execute()
			assertTransferValidationError(t, err, tc.fieldNames...)
		})
	}
}

func TestTransferWriteValidationRejectsLengthAndConflictErrors(t *testing.T) {
	services := NewServices()
	session := Session{ProfileName: "default", Environment: "sandbox"}

	type previewCase struct {
		name       string
		preview    func() (*PreviewResult, error)
		fieldNames []string
	}

	cases := []previewCase{
		{
			name: "ach length and conflict",
			preview: func() (*PreviewResult, error) {
				return services.PreviewExternalACH(session, ACHTransferInput{
					AccountID:           "account_123",
					AmountCents:         100,
					StatementDescriptor: strings.Repeat("s", 201),
					ExternalAccountID:   "external_account_123",
					AccountNumber:       "123456789",
					RoutingNumber:       "021000021",
					Funding:             "checking",
					IndividualName:      strings.Repeat("i", 23),
				})
			},
			fieldNames: []string{"statement_descriptor", "account_number", "routing_number", "funding", "individual_name"},
		},
		{
			name: "rtp length and conflict",
			preview: func() (*PreviewResult, error) {
				return services.PreviewExternalRTP(session, RTPTransferInput{
					AmountCents:              100,
					CreditorName:             strings.Repeat("c", 141),
					RemittanceInformation:    strings.Repeat("r", 141),
					SourceAccountNumberID:    "account_number_123",
					ExternalAccountID:        "external_account_123",
					DestinationAccountNumber: "123456789",
					DestinationRoutingNumber: "021000021",
				})
			},
			fieldNames: []string{"creditor_name", "remittance_information", "destination_account_number", "destination_routing_number"},
		},
		{
			name: "fednow length conflict and unsupported line2",
			preview: func() (*PreviewResult, error) {
				return services.PreviewExternalFedNow(session, FedNowTransferInput{
					AccountID:                         "account_123",
					AmountCents:                       100,
					CreditorName:                      strings.Repeat("c", 201),
					DebtorName:                        "Debtor",
					SourceAccountNumberID:             "account_number_123",
					UnstructuredRemittanceInformation: strings.Repeat("r", 201),
					ExternalAccountID:                 "external_account_123",
					AccountNumber:                     "123456789",
					RoutingNumber:                     "021000021",
					CreditorAddress: &FedNowAddressInput{
						City:       "New York",
						Line2:      "Floor 2",
						PostalCode: "10045",
						State:      "NY",
					},
				})
			},
			fieldNames: []string{"creditor_name", "unstructured_remittance_information", "account_number", "routing_number", "creditor_address.line2"},
		},
		{
			name: "wire length and dependent originator fields",
			preview: func() (*PreviewResult, error) {
				return services.PreviewExternalWire(session, WireTransferInput{
					AccountID:               "account_123",
					AmountCents:             100,
					BeneficiaryName:         strings.Repeat("b", 36),
					ExternalAccountID:       "external_account_123",
					AccountNumber:           "123456789",
					RoutingNumber:           "021000021",
					OriginatorAddressLine2:  "Suite 100",
					BeneficiaryAddressLine2: "Floor 2",
				})
			},
			fieldNames: []string{"beneficiary_name", "account_number", "routing_number", "originator_name", "originator_address_line1", "beneficiary_address_line1"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			preview, err := tc.preview()
			assertTransferValidationError(t, err, tc.fieldNames...)
			if preview != nil {
				t.Fatalf("preview = %#v, want nil on validation failure", preview)
			}
		})
	}
}

func assertTransferValidationError(t *testing.T, err error, fieldNames ...string) {
	t.Helper()
	if err == nil {
		t.Fatal("err = nil, want validation error")
	}
	var detail *util.ErrorDetail
	if !errors.As(err, &detail) {
		t.Fatalf("err = %T, want *util.ErrorDetail", err)
	}
	if detail.Code != util.CodeValidationError {
		t.Fatalf("validation code = %q, want %q", detail.Code, util.CodeValidationError)
	}
	for _, fieldName := range fieldNames {
		if !hasFieldError(detail.Fields, fieldName) {
			t.Fatalf("validation fields = %#v, want field %q", detail.Fields, fieldName)
		}
	}
}

func hasFieldError(fields []util.FieldError, want string) bool {
	for _, field := range fields {
		if field.Field == want {
			return true
		}
	}
	return false
}
