package cli

import (
	"context"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/gitsoecode/increasex-cli-mcp/internal/app"
	"github.com/gitsoecode/increasex-cli-mcp/internal/util"
	"github.com/spf13/cobra"
)

func TestNewRootCmdCommandOrder(t *testing.T) {
	cmd := NewRootCmd()
	commands := cmd.Commands()
	names := make([]string, 0, len(commands))
	for _, child := range commands {
		names = append(names, child.Name())
	}

	got := strings.Join(names, ",")
	want := "accounts,account-numbers,balance,transactions,transfer,external-accounts,cards,auth,mcp"
	if got != want {
		t.Fatalf("command order = %q, want %q", got, want)
	}
	for _, child := range commands {
		if !child.SilenceUsage || !child.SilenceErrors {
			t.Fatalf("child command %q should silence usage and errors", child.Name())
		}
	}
	if flag := cmd.PersistentFlags().Lookup("advanced"); flag == nil {
		t.Fatal("root command should expose --advanced")
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

func TestRenderRecordListIncludesTitleMetaAndFields(t *testing.T) {
	t.Setenv("COLUMNS", "72")

	output := renderRecordList("Transfers", []recordItem{
		{
			Title: "Payroll vendor transfer for weekly settlement",
			Meta:  "$125.00 • ach • pending_approval",
			Fields: []recordField{
				{Label: "id", Value: "transfer_1234567890abcdef"},
				{Label: "external_account", Value: "external_account_1234567890abcdef"},
			},
		},
	})

	if !strings.Contains(output, "Transfers") {
		t.Fatalf("renderRecordList() should include the list title, got %q", output)
	}
	if !strings.Contains(output, "id:") || !strings.Contains(output, "external_account:") {
		t.Fatalf("renderRecordList() should include labeled fields, got %q", output)
	}
	if !strings.Contains(output, "$125.00") || !strings.Contains(output, "pending_approval") {
		t.Fatalf("renderRecordList() should include the meta line, got %q", output)
	}
}

func TestTransferConfirmationPromptReflectsApprovalState(t *testing.T) {
	requireApproval := true

	cases := []struct {
		rail string
		flag *bool
		want string
	}{
		{rail: "account", flag: &requireApproval, want: "Queue this account transfer for approval?"},
		{rail: "account", flag: nil, want: "Execute this account transfer?"},
		{rail: "ach", flag: &requireApproval, want: "Queue this ACH transfer for approval?"},
		{rail: "real_time_payments", flag: nil, want: "Execute this Real-Time Payments transfer?"},
		{rail: "fednow", flag: &requireApproval, want: "Queue this FedNow transfer for approval?"},
		{rail: "wire", flag: nil, want: "Execute this wire transfer?"},
	}

	for _, tc := range cases {
		if got := transferConfirmationPrompt(tc.rail, tc.flag); got != tc.want {
			t.Fatalf("transferConfirmationPrompt(%q) = %q, want %q", tc.rail, got, tc.want)
		}
	}
}

func TestRenderRecordListKeepsLinesWithinWidthBudget(t *testing.T) {
	widths := []int{48, 72, 96}
	for _, width := range widths {
		t.Run(strconv.Itoa(width), func(t *testing.T) {
			t.Setenv("COLUMNS", strconv.Itoa(width))

			output := renderRecordList("Transactions", []recordItem{
				{
					Title: "Merchant settlement for a deliberately long transaction description",
					Meta:  "$1,234.56 • credit • card_settlement",
					Fields: []recordField{
						{Label: "id", Value: "transaction_1234567890abcdef"},
						{Label: "account", Value: "account_1234567890abcdef"},
						{Label: "counterparty", Value: "A deliberately verbose counterparty summary for wrapping"},
						{Label: "route_id", Value: "route_1234567890abcdef"},
					},
				},
			})

			assertRenderedLinesFitWidth(t, output, width)
			if strings.Contains(output, "ACCOUNT") || strings.Contains(output, "DESCRIPTION") {
				t.Fatalf("renderRecordList() should not emit old table headers, got %q", output)
			}
		})
	}
}

func TestPrintCardsUsesRecordLayout(t *testing.T) {
	t.Setenv("COLUMNS", "96")

	output := captureStdout(t, func() {
		printCards([]app.CardSummary{
			{
				ID:          "sandbox_card_3tpnskm9qi5tzp73xz37",
				AccountID:   "sandbox_account_yb31hvngnl6t16os3wma",
				Last4:       "3925",
				Status:      "active",
				Description: "First card created",
				CreatedAt:   "2026-03-20T04:53:10Z",
			},
			{
				ID:          "sandbox_card_zypnskm9qi5tzp73xz37",
				AccountID:   "sandbox_account_yb31hvngnl6t16os3wmb",
				Last4:       "9286",
				Status:      "active",
				Description: "Company Card",
				CreatedAt:   "2026-02-18T04:10:18Z",
			},
		})
	})

	assertRenderedLinesFitWidth(t, output, 96)
	if !strings.Contains(output, "First card created") || !strings.Contains(output, "last4:") {
		t.Fatalf("printCards() should render record fields, got %q", output)
	}
	if strings.Contains(output, "LAST4") || strings.Contains(output, "ACCOUNT") {
		t.Fatalf("printCards() should not render old table headers, got %q", output)
	}
}

func TestPrintCardsWrapCleanlyAtNarrowWidths(t *testing.T) {
	t.Setenv("COLUMNS", "84")

	output := captureStdout(t, func() {
		printCards([]app.CardSummary{
			{
				ID:          "sandbox_card_3tpnskm9qi5tzp73xz37",
				AccountID:   "sandbox_account_yb31hvngnl6t16os3wma",
				Last4:       "3925",
				Status:      "active",
				Description: "First card created for the engineering team with a long label",
				CreatedAt:   "2026-03-20T04:53:10Z",
			},
		})
	})

	assertRenderedLinesFitWidth(t, output, 84)
	if !strings.Contains(output, "last4:") || !strings.Contains(output, "created_at:") {
		t.Fatalf("printCards() should keep labeled record fields in narrow terminals, got %q", output)
	}
	if strings.Contains(output, "LAST4") {
		t.Fatalf("printCards() should not render old table headers, got %q", output)
	}
}

func TestPrintAccountsUsesRecordLayout(t *testing.T) {
	t.Setenv("COLUMNS", "72")

	output := captureStdout(t, func() {
		printAccounts([]app.AccountSummary{
			{
				ID:        "sandbox_account_1234567890abcdef",
				Name:      "Corporate Checking",
				Status:    "open",
				EntityID:  "sandbox_entity_1234567890abcdef",
				ProgramID: "sandbox_program_1234567890abcdef",
				CreatedAt: "2026-03-29T20:54:10Z",
			},
		})
	})

	assertRenderedLinesFitWidth(t, output, 72)
	if !strings.Contains(output, "Corporate Checking") || !strings.Contains(output, "entity:") || !strings.Contains(output, "program:") {
		t.Fatalf("printAccounts() should render record fields, got %q", output)
	}
	if strings.Contains(output, "NAME") || strings.Contains(output, "PROGRAM") {
		t.Fatalf("printAccounts() should not render old table headers, got %q", output)
	}
}

func TestPrintTransactionsUsesRecordLayoutAndOptionalFields(t *testing.T) {
	t.Setenv("COLUMNS", "96")

	output := captureStdout(t, func() {
		printTransactions([]app.TransactionSummary{
			{
				ID:                  "transaction_1234567890abcdef",
				AccountID:           "account_1234567890abcdef",
				AmountCents:         123456,
				Direction:           "credit",
				Description:         "Merchant settlement",
				Type:                "card_settlement",
				CreatedAt:           "2026-03-29T12:34:56Z",
				RouteID:             "route_1234567890abcdef",
				RouteType:           "card_payment",
				CounterpartySummary: "Merchant Processor",
			},
		})
	})

	assertRenderedLinesFitWidth(t, output, 96)
	if !strings.Contains(output, "Merchant settlement") || !strings.Contains(output, "$1234.56") {
		t.Fatalf("printTransactions() should render title and meta, got %q", output)
	}
	if !strings.Contains(output, "counterparty:") || !strings.Contains(output, "route_type:") || !strings.Contains(output, "route_id:") {
		t.Fatalf("printTransactions() should include optional fields when present, got %q", output)
	}
	if strings.Contains(output, "ACCOUNT") || strings.Contains(output, "DESCRIPTION") {
		t.Fatalf("printTransactions() should not render old table headers, got %q", output)
	}
}

func TestPrintExternalAccountsUsesRecordLayout(t *testing.T) {
	t.Setenv("COLUMNS", "72")

	output := captureStdout(t, func() {
		printExternalAccounts([]app.ExternalAccountSummary{
			{
				ID:                  "external_account_1234567890abcdef",
				Description:         "Primary vendor",
				AccountHolder:       "business",
				Funding:             "checking",
				RoutingNumber:       "021000021",
				AccountNumberMasked: "****6789",
				Status:              "active",
				CreatedAt:           "2026-03-29T20:54:10Z",
			},
		})
	})

	assertRenderedLinesFitWidth(t, output, 72)
	if !strings.Contains(output, "Primary vendor") || !strings.Contains(output, "account:") || !strings.Contains(output, "****6789") {
		t.Fatalf("printExternalAccounts() should render labeled masked fields, got %q", output)
	}
	if strings.Contains(output, "ROUTING") || strings.Contains(output, "ACCOUNT") {
		t.Fatalf("printExternalAccounts() should not render old table headers, got %q", output)
	}
}

func TestPrintTransfersUsesRecordLayout(t *testing.T) {
	t.Setenv("COLUMNS", "72")

	output := captureStdout(t, func() {
		printTransfers("Transfers", []app.TransferSummary{
			{
				Rail:              "ach",
				ID:                "ach_transfer_1234567890abcdef",
				AmountCents:       2550,
				Status:            "pending_approval",
				CreatedAt:         "2026-03-29T12:34:56Z",
				ExternalAccountID: "external_account_1234567890abcdef",
				Counterparty:      "Payroll vendor",
			},
		})
	})

	assertRenderedLinesFitWidth(t, output, 72)
	if !strings.Contains(output, "Payroll vendor") || !strings.Contains(output, "$25.50") || !strings.Contains(output, "external_account:") {
		t.Fatalf("printTransfers() should render record-style transfers, got %q", output)
	}
	if strings.Contains(output, "RAIL") || strings.Contains(output, "STATUS") {
		t.Fatalf("printTransfers() should not render old table headers, got %q", output)
	}
}

func TestPrintAccountNumbersShowsExplicitFields(t *testing.T) {
	t.Setenv("COLUMNS", "96")

	output := captureStdout(t, func() {
		printAccountNumbers([]app.AccountNumberSummary{
			{
				ID:                  "account_number_1234567890abcdef",
				AccountID:           "account_1234567890abcdef",
				AccountName:         "Operating",
				Name:                "Primary operating account",
				RoutingNumber:       "021000021",
				AccountNumberMasked: "******6789",
				Status:              "active",
				InboundACH:          &app.InboundACHInput{DebitStatus: "allowed"},
				InboundChecks:       &app.InboundChecksInput{Status: "check_transfers_only"},
			},
		})
	})

	if !strings.Contains(output, "account:") {
		t.Fatalf("printAccountNumbers() should show account field, got %q", output)
	}
	if !strings.Contains(output, "Operating (account_1234567890abcdef)") {
		t.Fatalf("printAccountNumbers() should show parent account name and id, got %q", output)
	}
	if !strings.Contains(output, "routing:") {
		t.Fatalf("printAccountNumbers() should show routing field, got %q", output)
	}
	if !strings.Contains(output, "number:") {
		t.Fatalf("printAccountNumbers() should show masked number field, got %q", output)
	}
	if strings.Contains(output, "NAME") {
		t.Fatalf("printAccountNumbers() should not fall back to the old wide header row, got %q", output)
	}
}

func TestPrintAccountNumberDetailsShowsSensitiveValueWhenPresent(t *testing.T) {
	t.Setenv("COLUMNS", "96")

	output := captureStdout(t, func() {
		printAccountNumberDetails(&app.AccountNumberDetails{
			AccountNumberSummary: app.AccountNumberSummary{
				ID:                  "account_number_1234567890abcdef",
				AccountID:           "account_1234567890abcdef",
				AccountName:         "Operating",
				Name:                "Expenses",
				RoutingNumber:       "074920909",
				AccountNumberMasked: "****8330",
				Status:              "active",
				InboundACH:          &app.InboundACHInput{DebitStatus: "allowed"},
				InboundChecks:       &app.InboundChecksInput{Status: "allowed"},
				CreatedAt:           "2026-03-20T04:10:18Z",
			},
			AccountNumber: "12345678330",
		})
	})

	if !strings.Contains(output, "account_number:") || !strings.Contains(output, "12345678330") {
		t.Fatalf("printAccountNumberDetails() should show the full account number when present, got %q", output)
	}
	if strings.Contains(output, "account_number_masked:") {
		t.Fatalf("printAccountNumberDetails() should omit the masked account-number line during sensitive inspect, got %q", output)
	}
	if !strings.Contains(output, "account:") || !strings.Contains(output, "Operating (account_1234567890abcdef)") {
		t.Fatalf("printAccountNumberDetails() should show the parent account name and id, got %q", output)
	}
	if strings.Contains(output, "&{allowed}") {
		t.Fatalf("printAccountNumberDetails() should flatten inbound settings instead of printing pointers, got %q", output)
	}
}

func TestPrintAccountNumberDetailsKeepsMaskedValueInNonSensitiveView(t *testing.T) {
	t.Setenv("COLUMNS", "96")

	output := captureStdout(t, func() {
		printAccountNumberDetails(&app.AccountNumberDetails{
			AccountNumberSummary: app.AccountNumberSummary{
				ID:                  "account_number_1234567890abcdef",
				AccountID:           "account_1234567890abcdef",
				AccountName:         "Operating",
				Name:                "Expenses",
				RoutingNumber:       "074920909",
				AccountNumberMasked: "****8330",
				Status:              "active",
				InboundACH:          &app.InboundACHInput{DebitStatus: "allowed"},
				InboundChecks:       &app.InboundChecksInput{Status: "allowed"},
				CreatedAt:           "2026-03-20T04:10:18Z",
			},
		})
	})

	if !strings.Contains(output, "account_number_masked:") || !strings.Contains(output, "****8330") {
		t.Fatalf("printAccountNumberDetails() should show only the masked value outside sensitive inspect, got %q", output)
	}
	if strings.Contains(output, "account_number:") {
		t.Fatalf("printAccountNumberDetails() should not show the full account number outside sensitive inspect, got %q", output)
	}
}

func TestPrintPreviewWrapsLongSummaryAndDetails(t *testing.T) {
	t.Setenv("COLUMNS", "48")

	output := captureStdout(t, func() {
		printPreview(&app.PreviewResult{
			Summary: "This is a very long preview summary that should wrap rather than blow through the frame",
			Details: map[string]any{
				"confirmation_note": "This is a long confirmation note that should wrap inside the panel instead of stretching it off screen.",
			},
			ConfirmationToken: "token_123456789012345678901234567890",
		})
	})

	for _, line := range strings.Split(output, "\n") {
		if len([]rune(line)) > 70 {
			t.Fatalf("printPreview() produced overly wide line %q", line)
		}
	}
}

func TestFormatCLIErrorUsesHumanReadableFieldLines(t *testing.T) {
	err := &util.ErrorDetail{
		Code:    util.CodeValidationError,
		Message: "Please correct the highlighted external account fields.",
		Fields: []util.FieldError{
			{Field: "account_holder", Message: "expected one of business, individual, or unknown"},
		},
	}

	formatted := FormatCLIError(err)
	if !strings.Contains(formatted, "Validation error: Please correct the highlighted external account fields.") {
		t.Fatalf("FormatCLIError() = %q, want human-readable validation summary", formatted)
	}
	if !strings.Contains(formatted, "account_holder: expected one of business, individual, or unknown") {
		t.Fatalf("FormatCLIError() = %q, want field guidance", formatted)
	}
	if strings.Contains(formatted, "POST \"https://") {
		t.Fatalf("FormatCLIError() should not include raw API transport text, got %q", formatted)
	}
}

func TestInvokeCommandParsesForwardedLeafFlags(t *testing.T) {
	parent := &cobra.Command{Use: "parent"}
	parent.SetContext(context.Background())

	var received string
	child := &cobra.Command{
		Use: "child",
		RunE: func(cmd *cobra.Command, args []string) error {
			received = cmd.Flag("card-id").Value.String()
			return nil
		},
	}
	child.Flags().String("card-id", "", "card id")

	if err := invokeCommand(parent, child, "--card-id", "card_123"); err != nil {
		t.Fatalf("invokeCommand() error = %v", err)
	}
	if received != "card_123" {
		t.Fatalf("invokeCommand() forwarded flag = %q, want %q", received, "card_123")
	}
}

func TestInvokeCommandParsesForwardedBoolFlags(t *testing.T) {
	parent := &cobra.Command{Use: "parent"}
	parent.SetContext(context.Background())

	var showSensitive bool
	child := &cobra.Command{
		Use: "child",
		RunE: func(cmd *cobra.Command, args []string) error {
			showSensitive, _ = cmd.Flags().GetBool("show-sensitive")
			return nil
		},
	}
	child.Flags().Bool("show-sensitive", false, "show sensitive values")

	if err := invokeCommand(parent, child, "--show-sensitive"); err != nil {
		t.Fatalf("invokeCommand() error = %v", err)
	}
	if !showSensitive {
		t.Fatal("invokeCommand() should forward bool flags")
	}
}

func TestBuildAccountNumberOptionsIncludesParentAccountNames(t *testing.T) {
	options := buildAccountNumberOptions([]app.AccountNumberSummary{
		{
			ID:                  "account_number_123",
			AccountID:           "account_123",
			AccountName:         "Operating",
			Name:                "Revenue",
			AccountNumberMasked: "****1913",
			RoutingNumber:       "074920909",
			Status:              "active",
		},
	})

	if len(options) != 1 {
		t.Fatalf("len(options) = %d, want 1", len(options))
	}
	if !strings.Contains(options[0].Description, "Operating (account_123)") {
		t.Fatalf("option description = %q, want parent account name and id", options[0].Description)
	}
	if !strings.Contains(options[0].Search, "Operating") {
		t.Fatalf("option search = %q, want parent account name", options[0].Search)
	}
}

func TestNewTransferExternalCmdExposesRailSpecificSourceFlags(t *testing.T) {
	ctx := &Context{Options: &RootOptions{}}
	cmd := newTransferExternalCmd(ctx)

	for _, name := range []string{
		"account-id",
		"rtp-source-account-number-id",
		"fednow-account-id",
		"fednow-source-account-number-id",
		"wire-account-id",
		"wire-source-account-number-id",
	} {
		if cmd.Flags().Lookup(name) == nil {
			t.Fatalf("newTransferExternalCmd() missing flag %q", name)
		}
	}
}

func TestNewTransferExternalCmdUsesRequiredRemittanceHelpText(t *testing.T) {
	ctx := &Context{Options: &RootOptions{}}
	cmd := newTransferExternalCmd(ctx)

	if got := cmd.Flags().Lookup("rtp-remittance-information").Usage; got != "required remittance information" {
		t.Fatalf("rtp-remittance-information usage = %q, want required wording", got)
	}
	if got := cmd.Flags().Lookup("fednow-remittance").Usage; got != "required unstructured remittance info" {
		t.Fatalf("fednow-remittance usage = %q, want required wording", got)
	}
}

func TestTerminalMenuAllowedHonorsAdvancedFlag(t *testing.T) {
	if !terminalMenuAllowed(&RootOptions{}, true) {
		t.Fatal("terminalMenuAllowed() should allow menus for tty by default")
	}
	if terminalMenuAllowed(&RootOptions{Advanced: true}, true) {
		t.Fatal("terminalMenuAllowed() should disable menus when advanced mode is enabled")
	}
	if terminalMenuAllowed(&RootOptions{JSON: true}, true) {
		t.Fatal("terminalMenuAllowed() should disable menus when json mode is enabled")
	}
	if terminalMenuAllowed(&RootOptions{}, false) {
		t.Fatal("terminalMenuAllowed() should disable menus when stdin is not a tty")
	}
}

func TestParseAdvancedCommand(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		want    []string
		wantErr string
	}{
		{name: "plain", input: "auth login --env sandbox", want: []string{"auth", "login", "--env", "sandbox"}},
		{name: "quoted", input: `transfer list --status "pending approval"`, want: []string{"transfer", "list", "--status", "pending approval"}},
		{name: "leading executable", input: "increasex auth status", want: []string{"auth", "status"}},
		{name: "leading path", input: "./increasex auth status", want: []string{"auth", "status"}},
		{name: "blank", input: "   ", want: nil},
		{name: "back", input: "back", want: nil},
		{name: "malformed quote", input: `auth login --name "default`, wantErr: "unmatched quote"},
	}

	for _, tc := range cases {
		got, err := parseAdvancedCommand(tc.input)
		if tc.wantErr != "" {
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("%s: parseAdvancedCommand(%q) error = %v, want %q", tc.name, tc.input, err, tc.wantErr)
			}
			continue
		}
		if err != nil {
			t.Fatalf("%s: parseAdvancedCommand(%q) error = %v", tc.name, tc.input, err)
		}
		if strings.Join(got, "|") != strings.Join(tc.want, "|") {
			t.Fatalf("%s: parseAdvancedCommand(%q) = %v, want %v", tc.name, tc.input, got, tc.want)
		}
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	os.Stdout = writer
	defer func() {
		os.Stdout = old
	}()

	fn()

	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close() error = %v", err)
	}
	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("io.ReadAll() error = %v", err)
	}
	return string(output)
}

func assertRenderedLinesFitWidth(t *testing.T, output string, width int) {
	t.Helper()
	for _, line := range strings.Split(strings.TrimRight(output, "\n"), "\n") {
		if lipgloss.Width(line) > width {
			t.Fatalf("rendered line width = %d, want <= %d: %q", lipgloss.Width(line), width, line)
		}
	}
}

func TestNewAuthCmdProvidesInteractiveMenuEntryPoint(t *testing.T) {
	ctx := &Context{Options: &RootOptions{}}
	cmd := newAuthCmd(ctx)
	if cmd.RunE == nil {
		t.Fatal("auth command should provide a menu entrypoint on tty")
	}
}

func TestBrowserOpenCommandUsesPlatformDefaults(t *testing.T) {
	command, args, err := browserOpenCommand("https://example.com")
	if err != nil {
		t.Fatalf("browserOpenCommand() error = %v", err)
	}

	switch runtime.GOOS {
	case "darwin":
		if command != "open" {
			t.Fatalf("browserOpenCommand() command = %q, want open", command)
		}
	case "linux":
		if command != "xdg-open" {
			t.Fatalf("browserOpenCommand() command = %q, want xdg-open", command)
		}
	case "windows":
		if command != "rundll32" {
			t.Fatalf("browserOpenCommand() command = %q, want rundll32", command)
		}
	default:
		t.Fatalf("unexpected runtime.GOOS %q in test", runtime.GOOS)
	}

	if len(args) == 0 || args[len(args)-1] != "https://example.com" {
		t.Fatalf("browserOpenCommand() args = %v, want url at end", args)
	}
}
