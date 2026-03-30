package app

import (
	"strconv"
	"strings"

	"github.com/gitsoecode/increasex-cli-mcp/internal/util"
)

const (
	achAccountNumberMaxLength            = 17
	achCompanyDescriptiveDateMaxLength   = 6
	achCompanyDiscretionaryDataMaxLength = 20
	achCompanyEntryDescriptionMaxLength  = 10
	achCompanyNameMaxLength              = 16
	achIndividualIDMaxLength             = 15
	achIndividualNameMaxLength           = 22
	achRoutingNumberMaxLength            = 9
	achStatementDescriptorMaxLength      = 200
	fedNowAccountNumberMaxLength         = 200
	fedNowNameMaxLength                  = 200
	fedNowRemittanceMaxLength            = 200
	fedNowRoutingNumberMaxLength         = 200
	fedNowAddressFieldMaxLength          = 200
	rtpAccountNumberMaxLength            = 34
	rtpNameMaxLength                     = 140
	rtpRemittanceMaxLength               = 140
	rtpRoutingNumberMaxLength            = 9
	wireAccountNumberMaxLength           = 34
	wireAddressFieldMaxLength            = 35
	wireNameMaxLength                    = 35
	wireRoutingNumberMaxLength           = 9
)

func validateMoveMoneyInternalInput(input MoveMoneyInternalInput) error {
	fields := []util.FieldError{}
	addRequiredStringField(&fields, "from_account_id", input.FromAccountID)
	addRequiredStringField(&fields, "to_account_id", input.ToAccountID)
	addPositiveAmountField(&fields, "amount_cents", input.AmountCents)
	return transferValidationError("Please correct the highlighted account transfer fields.", fields)
}

func validateACHTransferInput(input ACHTransferInput) error {
	fields := []util.FieldError{}
	addRequiredStringField(&fields, "account_id", input.AccountID)
	addNonZeroAmountField(&fields, "amount_cents", input.AmountCents)
	addRequiredStringField(&fields, "statement_descriptor", input.StatementDescriptor)

	addMaxLengthField(&fields, "account_number", input.AccountNumber, achAccountNumberMaxLength)
	addMaxLengthField(&fields, "routing_number", input.RoutingNumber, achRoutingNumberMaxLength)
	addMaxLengthField(&fields, "statement_descriptor", input.StatementDescriptor, achStatementDescriptorMaxLength)
	addMaxLengthField(&fields, "individual_id", input.IndividualID, achIndividualIDMaxLength)
	addMaxLengthField(&fields, "individual_name", input.IndividualName, achIndividualNameMaxLength)
	addMaxLengthField(&fields, "company_name", input.CompanyName, achCompanyNameMaxLength)
	addMaxLengthField(&fields, "company_entry_description", input.CompanyEntryDescription, achCompanyEntryDescriptionMaxLength)
	addMaxLengthField(&fields, "company_descriptive_date", input.CompanyDescriptiveDate, achCompanyDescriptiveDateMaxLength)
	addMaxLengthField(&fields, "company_discretionary_data", input.CompanyDiscretionaryData, achCompanyDiscretionaryDataMaxLength)

	addAllowedValuesField(&fields, "destination_account_holder", input.DestinationAccountHolder, []string{"business", "individual", "unknown"})
	addAllowedValuesField(&fields, "funding", input.Funding, []string{"checking", "savings", "general_ledger"})

	validateManualDestination(
		&fields,
		"external_account_id",
		input.ExternalAccountID,
		"account_number",
		input.AccountNumber,
		"routing_number",
		input.RoutingNumber,
		nil,
	)
	if hasTrimmedValue(input.ExternalAccountID) {
		addForbiddenIfPresent(&fields, "funding", input.Funding, "must be omitted when external_account_id is provided")
	}

	return transferValidationError("Please correct the highlighted ACH transfer fields.", fields)
}

func validateRTPTransferInput(input RTPTransferInput) error {
	fields := []util.FieldError{}
	addPositiveAmountField(&fields, "amount_cents", input.AmountCents)
	addRequiredStringField(&fields, "creditor_name", input.CreditorName)
	addRequiredStringField(&fields, "remittance_information", input.RemittanceInformation)
	addRequiredStringField(&fields, "source_account_number_id", input.SourceAccountNumberID)

	addMaxLengthField(&fields, "creditor_name", input.CreditorName, rtpNameMaxLength)
	addMaxLengthField(&fields, "debtor_name", input.DebtorName, rtpNameMaxLength)
	addMaxLengthField(&fields, "remittance_information", input.RemittanceInformation, rtpRemittanceMaxLength)
	addMaxLengthField(&fields, "destination_account_number", input.DestinationAccountNumber, rtpAccountNumberMaxLength)
	addMaxLengthField(&fields, "destination_routing_number", input.DestinationRoutingNumber, rtpRoutingNumberMaxLength)
	addMaxLengthField(&fields, "ultimate_creditor_name", input.UltimateCreditorName, rtpNameMaxLength)
	addMaxLengthField(&fields, "ultimate_debtor_name", input.UltimateDebtorName, rtpNameMaxLength)

	validateManualDestination(
		&fields,
		"external_account_id",
		input.ExternalAccountID,
		"destination_account_number",
		input.DestinationAccountNumber,
		"destination_routing_number",
		input.DestinationRoutingNumber,
		nil,
	)

	return transferValidationError("Please correct the highlighted Real-Time Payments transfer fields.", fields)
}

func validateFedNowTransferInput(input FedNowTransferInput) error {
	fields := []util.FieldError{}
	addRequiredStringField(&fields, "account_id", input.AccountID)
	addPositiveAmountField(&fields, "amount_cents", input.AmountCents)
	addRequiredStringField(&fields, "creditor_name", input.CreditorName)
	addRequiredStringField(&fields, "debtor_name", input.DebtorName)
	addRequiredStringField(&fields, "source_account_number_id", input.SourceAccountNumberID)
	addRequiredStringField(&fields, "unstructured_remittance_information", input.UnstructuredRemittanceInformation)

	addMaxLengthField(&fields, "account_number", input.AccountNumber, fedNowAccountNumberMaxLength)
	addMaxLengthField(&fields, "creditor_name", input.CreditorName, fedNowNameMaxLength)
	addMaxLengthField(&fields, "debtor_name", input.DebtorName, fedNowNameMaxLength)
	addMaxLengthField(&fields, "routing_number", input.RoutingNumber, fedNowRoutingNumberMaxLength)
	addMaxLengthField(&fields, "unstructured_remittance_information", input.UnstructuredRemittanceInformation, fedNowRemittanceMaxLength)

	validateManualDestination(
		&fields,
		"external_account_id",
		input.ExternalAccountID,
		"account_number",
		input.AccountNumber,
		"routing_number",
		input.RoutingNumber,
		nil,
	)
	validateFedNowAddress(&fields, "creditor_address", input.CreditorAddress)
	validateFedNowAddress(&fields, "debtor_address", input.DebtorAddress)

	return transferValidationError("Please correct the highlighted FedNow transfer fields.", fields)
}

func validateWireTransferInput(input WireTransferInput) error {
	fields := []util.FieldError{}
	addRequiredStringField(&fields, "account_id", input.AccountID)
	addPositiveAmountField(&fields, "amount_cents", input.AmountCents)
	addRequiredStringField(&fields, "beneficiary_name", input.BeneficiaryName)

	addMaxLengthField(&fields, "beneficiary_name", input.BeneficiaryName, wireNameMaxLength)
	addMaxLengthField(&fields, "account_number", input.AccountNumber, wireAccountNumberMaxLength)
	addMaxLengthField(&fields, "routing_number", input.RoutingNumber, wireRoutingNumberMaxLength)
	addMaxLengthField(&fields, "beneficiary_address_line1", input.BeneficiaryAddressLine1, wireAddressFieldMaxLength)
	addMaxLengthField(&fields, "beneficiary_address_line2", input.BeneficiaryAddressLine2, wireAddressFieldMaxLength)
	addMaxLengthField(&fields, "beneficiary_address_line3", input.BeneficiaryAddressLine3, wireAddressFieldMaxLength)
	addMaxLengthField(&fields, "originator_name", input.OriginatorName, wireNameMaxLength)
	addMaxLengthField(&fields, "originator_address_line1", input.OriginatorAddressLine1, wireAddressFieldMaxLength)
	addMaxLengthField(&fields, "originator_address_line2", input.OriginatorAddressLine2, wireAddressFieldMaxLength)
	addMaxLengthField(&fields, "originator_address_line3", input.OriginatorAddressLine3, wireAddressFieldMaxLength)

	validateManualDestination(
		&fields,
		"external_account_id",
		input.ExternalAccountID,
		"account_number",
		input.AccountNumber,
		"routing_number",
		input.RoutingNumber,
		nil,
	)
	if hasTrimmedValue(input.BeneficiaryAddressLine2) || hasTrimmedValue(input.BeneficiaryAddressLine3) {
		addRequiredWhenPresentField(&fields, "beneficiary_address_line1", input.BeneficiaryAddressLine1, []string{"beneficiary_address_line2", "beneficiary_address_line3"})
	}
	if hasAnyTrimmedValue(input.OriginatorAddressLine1, input.OriginatorAddressLine2, input.OriginatorAddressLine3) {
		addRequiredWhenPresentField(&fields, "originator_name", input.OriginatorName, []string{"originator_address_line1", "originator_address_line2", "originator_address_line3"})
	}
	if hasTrimmedValue(input.OriginatorAddressLine2) || hasTrimmedValue(input.OriginatorAddressLine3) {
		addRequiredWhenPresentField(&fields, "originator_address_line1", input.OriginatorAddressLine1, []string{"originator_address_line2", "originator_address_line3"})
	}

	return transferValidationError("Please correct the highlighted wire transfer fields.", fields)
}

func validateTransferActionInput(input TransferActionInput) error {
	fields := []util.FieldError{}
	addRequiredStringField(&fields, "rail", input.Rail)
	addRequiredStringField(&fields, "transfer_id", input.TransferID)
	if value := NormalizeTransferRail(input.Rail); value != "" && !isSupportedTransferRail(value) {
		fields = append(fields, util.FieldError{Field: "rail", Message: "must be one of account, ach, real_time_payments, fednow, or wire"})
	}
	return transferValidationError("Please correct the highlighted transfer action fields.", fields)
}

func validateFedNowAddress(fields *[]util.FieldError, prefix string, address *FedNowAddressInput) {
	if address == nil {
		return
	}
	addRequiredStringField(fields, prefix+".city", address.City)
	addRequiredStringField(fields, prefix+".postal_code", address.PostalCode)
	addRequiredStringField(fields, prefix+".state", address.State)
	addMaxLengthField(fields, prefix+".line1", address.Line1, fedNowAddressFieldMaxLength)
	addMaxLengthField(fields, prefix+".city", address.City, fedNowAddressFieldMaxLength)
	addMaxLengthField(fields, prefix+".state", address.State, fedNowAddressFieldMaxLength)
	addMaxLengthField(fields, prefix+".postal_code", address.PostalCode, fedNowAddressFieldMaxLength)
	if hasTrimmedValue(address.Line2) {
		*fields = append(*fields, util.FieldError{Field: prefix + ".line2", Message: "is not supported by the Increase FedNow API"})
	}
}

func validateManualDestination(fields *[]util.FieldError, externalField, externalValue, accountField, accountValue, routingField, routingValue string, extraConflictFields map[string]string) {
	hasExternal := hasTrimmedValue(externalValue)
	hasAccount := hasTrimmedValue(accountValue)
	hasRouting := hasTrimmedValue(routingValue)

	if hasExternal {
		if hasAccount {
			*fields = append(*fields, util.FieldError{Field: accountField, Message: "must be omitted when " + externalField + " is provided"})
		}
		if hasRouting {
			*fields = append(*fields, util.FieldError{Field: routingField, Message: "must be omitted when " + externalField + " is provided"})
		}
		for field, value := range extraConflictFields {
			addForbiddenIfPresent(fields, field, value, "must be omitted when "+externalField+" is provided")
		}
		return
	}

	if !hasAccount && !hasRouting {
		*fields = append(*fields,
			util.FieldError{Field: accountField, Message: "is required when " + externalField + " is not provided"},
			util.FieldError{Field: routingField, Message: "is required when " + externalField + " is not provided"},
		)
		return
	}
	if !hasAccount {
		*fields = append(*fields, util.FieldError{Field: accountField, Message: "is required when " + externalField + " is not provided"})
	}
	if !hasRouting {
		*fields = append(*fields, util.FieldError{Field: routingField, Message: "is required when " + externalField + " is not provided"})
	}
}

func transferValidationError(message string, fields []util.FieldError) error {
	if len(fields) == 0 {
		return nil
	}
	return &util.ErrorDetail{
		Code:    util.CodeValidationError,
		Message: message,
		Fields:  uniqueFieldErrors(fields),
	}
}

func uniqueFieldErrors(fields []util.FieldError) []util.FieldError {
	seen := map[string]struct{}{}
	out := make([]util.FieldError, 0, len(fields))
	for _, field := range fields {
		key := field.Field + "\x00" + field.Message
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, field)
	}
	return out
}

func addRequiredStringField(fields *[]util.FieldError, name, value string) {
	if strings.TrimSpace(value) == "" {
		*fields = append(*fields, util.FieldError{Field: name, Message: "is required"})
	}
}

func addRequiredWhenPresentField(fields *[]util.FieldError, name, value string, dependentFields []string) {
	if strings.TrimSpace(value) != "" {
		return
	}
	*fields = append(*fields, util.FieldError{Field: name, Message: "is required when " + strings.Join(dependentFields, " or ") + " is provided"})
}

func addPositiveAmountField(fields *[]util.FieldError, name string, value int64) {
	if value <= 0 {
		*fields = append(*fields, util.FieldError{Field: name, Message: "must be greater than 0"})
	}
}

func addNonZeroAmountField(fields *[]util.FieldError, name string, value int64) {
	if value == 0 {
		*fields = append(*fields, util.FieldError{Field: name, Message: "must not be 0"})
	}
}

func addMaxLengthField(fields *[]util.FieldError, name, value string, max int) {
	if len(strings.TrimSpace(value)) > max {
		*fields = append(*fields, util.FieldError{Field: name, Message: "must be " + intString(max) + " characters or fewer"})
	}
}

func addAllowedValuesField(fields *[]util.FieldError, name, value string, allowed []string) {
	if trimmed := strings.TrimSpace(value); trimmed != "" && !allowedValue(trimmed, allowed) {
		*fields = append(*fields, util.FieldError{Field: name, Message: "expected one of " + joinAllowedValues(allowed)})
	}
}

func addForbiddenIfPresent(fields *[]util.FieldError, name, value, message string) {
	if hasTrimmedValue(value) {
		*fields = append(*fields, util.FieldError{Field: name, Message: message})
	}
}

func allowedValue(value string, allowed []string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

func joinAllowedValues(values []string) string {
	if len(values) == 0 {
		return ""
	}
	if len(values) == 1 {
		return values[0]
	}
	return strings.Join(values[:len(values)-1], ", ") + ", or " + values[len(values)-1]
}

func hasTrimmedValue(value string) bool {
	return strings.TrimSpace(value) != ""
}

func hasAnyTrimmedValue(values ...string) bool {
	for _, value := range values {
		if hasTrimmedValue(value) {
			return true
		}
	}
	return false
}

func isSupportedTransferRail(rail string) bool {
	switch NormalizeTransferRail(rail) {
	case "account", "ach", "real_time_payments", "fednow", "wire":
		return true
	default:
		return false
	}
}

func intString(value int) string {
	return strconv.Itoa(value)
}
