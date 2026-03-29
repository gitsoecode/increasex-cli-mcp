package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	increasex "github.com/gitsoecode/increasex-cli-mcp/internal/increase"
	"github.com/gitsoecode/increasex-cli-mcp/internal/util"
)

func FormatCLIError(err error) string {
	if err == nil {
		return ""
	}

	detail := increasex.WrapError(err)
	if detail == nil {
		return err.Error()
	}

	lines := []string{formatCLISummary(detail)}
	for _, field := range detail.Fields {
		label := field.Field
		if strings.TrimSpace(label) == "" {
			label = "field"
		}
		lines = append(lines, fmt.Sprintf("%s: %s", label, field.Message))
	}
	if len(detail.Fields) == 0 {
		if raw, ok := detail.Details["detail"].(string); ok && strings.TrimSpace(raw) != "" {
			for _, line := range strings.Split(strings.TrimSpace(raw), "\n") {
				trimmed := strings.TrimSpace(line)
				if trimmed != "" {
					lines = append(lines, trimmed)
				}
			}
		}
	}
	return strings.Join(lines, "\n")
}

func formatCLISummary(detail *util.ErrorDetail) string {
	switch detail.Code {
	case util.CodeValidationError:
		return "Validation error: " + detail.Message
	case util.CodeAuthError:
		return "Authentication error: " + detail.Message
	case util.CodeNetworkError:
		return "Network error: " + detail.Message
	case util.CodeRateLimited:
		return "Rate limited: " + detail.Message
	case util.CodeNotFound:
		return "Not found: " + detail.Message
	default:
		return detail.Message
	}
}

func IsValidationError(err error) bool {
	if err == nil {
		return false
	}
	var detail *util.ErrorDetail
	if errors.As(err, &detail) {
		return detail.Code == util.CodeValidationError
	}
	return increasex.WrapError(err).Code == util.CodeValidationError
}

func printCLIError(err error) {
	if err == nil {
		return
	}
	fmt.Fprintln(os.Stderr, FormatCLIError(err))
}
