package util

import "fmt"

const (
	CodeAuthError            = "auth_error"
	CodeValidationError      = "validation_error"
	CodeNotFound             = "not_found"
	CodeRateLimited          = "rate_limited"
	CodeAPIError             = "api_error"
	CodeConfirmationRequired = "confirmation_required"
	CodeConfirmationInvalid  = "confirmation_invalid"
	CodeIdempotencyConflict  = "idempotency_conflict"
	CodeNetworkError         = "network_error"
	CodeUnknownError         = "unknown_error"
)

type ErrorDetail struct {
	Code      string         `json:"code"`
	Message   string         `json:"message"`
	Details   map[string]any `json:"details,omitempty"`
	Fields    []FieldError   `json:"fields,omitempty"`
	Retryable bool           `json:"retryable,omitempty"`
}

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *ErrorDetail) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func NewError(code, message string, details map[string]any, retryable bool) *ErrorDetail {
	return &ErrorDetail{
		Code:      code,
		Message:   message,
		Details:   details,
		Retryable: retryable,
	}
}
