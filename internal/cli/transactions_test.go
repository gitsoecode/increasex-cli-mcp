package cli

import (
	"testing"

	"github.com/gitsoecode/increasex-cli-mcp/internal/ui"
)

func TestNewTransactionsCmdExposesTimePeriodFlags(t *testing.T) {
	cmd := newTransactionsCmd(&Context{Options: &RootOptions{}})

	for _, name := range []string{"account-id", "since", "until", "period", "cursor", "limit", "category"} {
		if cmd.Flags().Lookup(name) == nil {
			t.Fatalf("transactions command missing --%s flag", name)
		}
	}
}

func TestTransactionTimeRangeInputExplicitBoundsOverridePreset(t *testing.T) {
	result := transactionTimeRangeInput("2026-03-01T00:00:00Z", "", "last-7d")
	if result.Period != "" {
		t.Fatalf("Period = %q, want empty when explicit bounds are present", result.Period)
	}

	result = transactionTimeRangeInput("", "2026-03-15T00:00:00Z", "last-30d")
	if result.Period != "" {
		t.Fatalf("Period = %q, want empty when explicit upper bound is present", result.Period)
	}

	result = transactionTimeRangeInput("", "", "current-month")
	if result.Period != "current-month" {
		t.Fatalf("Period = %q, want current-month", result.Period)
	}
}

func TestPromptTransactionTimeRangeDefaultOption(t *testing.T) {
	original := runPromptSelect
	t.Cleanup(func() { runPromptSelect = original })

	runPromptSelect = func(label string, options []ui.Option) (string, error) {
		return "default", nil
	}

	since, until, period, err := promptTransactionTimeRange()
	if err != nil {
		t.Fatalf("promptTransactionTimeRange() error = %v", err)
	}
	if since != "" || until != "" || period != "" {
		t.Fatalf("promptTransactionTimeRange() = (%q, %q, %q), want empty default selection", since, until, period)
	}
}

func TestPromptTransactionTimeRangeCustomOption(t *testing.T) {
	originalSelect := runPromptSelect
	originalString := runPromptString
	t.Cleanup(func() {
		runPromptSelect = originalSelect
		runPromptString = originalString
	})

	selectCalls := 0
	runPromptSelect = func(label string, options []ui.Option) (string, error) {
		selectCalls++
		if selectCalls == 1 {
			return "custom", nil
		}
		return "enter", nil
	}

	stringValues := []string{"2026-03-01T00:00:00Z", "2026-03-15T23:59:59Z"}
	runPromptString = func(label string, required bool) (string, error) {
		value := stringValues[0]
		stringValues = stringValues[1:]
		return value, nil
	}

	since, until, period, err := promptTransactionTimeRange()
	if err != nil {
		t.Fatalf("promptTransactionTimeRange() error = %v", err)
	}
	if since != "2026-03-01T00:00:00Z" || until != "2026-03-15T23:59:59Z" || period != "" {
		t.Fatalf("promptTransactionTimeRange() = (%q, %q, %q), want custom explicit bounds", since, until, period)
	}
}
