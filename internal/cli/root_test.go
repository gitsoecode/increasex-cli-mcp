package cli

import (
	"strings"
	"testing"
)

func TestNewRootCmdCommandOrder(t *testing.T) {
	cmd := NewRootCmd()
	commands := cmd.Commands()
	names := make([]string, 0, len(commands))
	for _, child := range commands {
		names = append(names, child.Name())
	}

	got := strings.Join(names, ",")
	want := "accounts,balance,transactions,transfer,external-accounts,cards,auth,mcp"
	if got != want {
		t.Fatalf("command order = %q, want %q", got, want)
	}
}

func TestNormalizeTransferRail(t *testing.T) {
	cases := map[string]string{
		"internal":           "account",
		"account_transfer":   "account",
		"rtp":                "real_time_payments",
		"real-time-payments": "real_time_payments",
		"real_time_payments": "real_time_payments",
		"fednow":             "fednow",
	}
	for input, want := range cases {
		if got := normalizeTransferRail(input); got != want {
			t.Fatalf("normalizeTransferRail(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestRenderTableCompactFallback(t *testing.T) {
	t.Setenv("COLUMNS", "60")

	output := renderTable("Transfers", []string{"ID", "DESCRIPTION"}, [][]string{
		{"transfer_1234567890", "A very long description that should not wrap awkwardly in narrow terminals."},
	})

	if !strings.Contains(output, "description:") {
		t.Fatalf("renderTable() should include compact key/value output, got %q", output)
	}
	if !strings.Contains(output, "Transfers") {
		t.Fatalf("renderTable() should include the title, got %q", output)
	}
}
