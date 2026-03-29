package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/jessevaughan/increasex/internal/app"
	"github.com/jessevaughan/increasex/internal/auth"
	increasex "github.com/jessevaughan/increasex/internal/increase"
	"github.com/jessevaughan/increasex/internal/ui"
	"github.com/jessevaughan/increasex/internal/util"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func (c *Context) resolve(ctx context.Context) (*app.Session, *increasex.Client, error) {
	return c.Services.ResolveSession(ctx, auth.ResolveInput{
		ProfileName: c.Options.Profile,
		Environment: c.Options.Environment,
		APIKey:      c.Options.APIKey,
	})
}

func printJSON(value any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(value)
}

func printEnvelopeJSON(data any, requestID string, err error) error {
	if err != nil {
		wrapped := increasex.WrapError(err)
		return printJSON(util.Fail(wrapped, requestID))
	}
	return printJSON(util.Ok(data, requestID))
}

func printKeyValues(values map[string]any) {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	lines := []string{titleStyle.Render("Details")}
	for _, key := range keys {
		lines = append(lines, fmt.Sprintf("%s %s", keyStyle.Render(key+":"), valueStyle.Render(fmt.Sprint(values[key]))))
	}
	fmt.Println(panelStyle.Render(strings.Join(lines, "\n")))
}

func printAccounts(accounts []app.AccountSummary) {
	rows := make([][]string, 0, len(accounts))
	for _, account := range accounts {
		rows = append(rows, []string{account.Name, account.ID, account.Status, account.EntityID, account.ProgramID, account.CreatedAt})
	}
	fmt.Println(renderTable("Accounts", []string{"NAME", "ID", "STATUS", "ENTITY", "PROGRAM", "CREATED"}, rows))
}

func printTransactions(items []app.TransactionSummary) {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{item.ID, item.AccountID, util.FormatUSDMinor(item.AmountCents), item.Direction, item.Description, item.CreatedAt})
	}
	fmt.Println(renderTable("Transactions", []string{"ID", "ACCOUNT", "AMOUNT", "DIRECTION", "DESCRIPTION", "CREATED"}, rows))
}

func printCards(cards []app.CardSummary) {
	rows := make([][]string, 0, len(cards))
	for _, card := range cards {
		rows = append(rows, []string{card.ID, card.AccountID, card.Last4, card.Status, card.Description, card.CreatedAt})
	}
	fmt.Println(renderTable("Cards", []string{"ID", "ACCOUNT", "LAST4", "STATUS", "DESCRIPTION", "CREATED"}, rows))
}

func printExternalAccounts(accounts []app.ExternalAccountSummary) {
	rows := make([][]string, 0, len(accounts))
	for _, account := range accounts {
		rows = append(rows, []string{
			account.Description,
			account.ID,
			account.AccountHolder,
			account.Funding,
			account.RoutingNumber,
			account.AccountNumberMasked,
			account.Status,
			account.CreatedAt,
		})
	}
	fmt.Println(renderTable("External Accounts", []string{"DESCRIPTION", "ID", "HOLDER", "FUNDING", "ROUTING", "ACCOUNT", "STATUS", "CREATED"}, rows))
}

func printTransfers(title string, items []app.TransferSummary) {
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			item.Rail,
			item.ID,
			util.FormatUSDMinor(item.AmountCents),
			item.Status,
			item.Counterparty,
			item.ExternalAccountID,
			item.CreatedAt,
		})
	}
	fmt.Println(renderTable(title, []string{"RAIL", "ID", "AMOUNT", "STATUS", "COUNTERPARTY", "EXTERNAL ACCOUNT", "CREATED"}, rows))
}

func printPreview(preview *app.PreviewResult) {
	lines := []string{titleStyle.Render("Preview"), previewStyle.Render(preview.Summary)}
	if preview.Details != nil {
		keys := make([]string, 0, len(preview.Details))
		for key := range preview.Details {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			lines = append(lines, fmt.Sprintf("%s %s", keyStyle.Render(key+":"), valueStyle.Render(fmt.Sprint(preview.Details[key]))))
		}
	}
	lines = append(lines, fmt.Sprintf("%s %s", keyStyle.Render("confirmation_token:"), tokenStyle.Render(preview.ConfirmationToken)))
	fmt.Println(panelStyle.Render(strings.Join(lines, "\n")))
}

func chooseAccount(accounts []app.AccountSummary, label string) (string, error) {
	options := make([]ui.Option, 0, len(accounts))
	for _, account := range accounts {
		options = append(options, ui.Option{
			Label:       account.Name,
			Value:       account.ID,
			Description: fmt.Sprintf("%s  %s", mutedStyle.Render(account.Status), mutedStyle.Render(account.ID)),
			Search:      strings.Join([]string{account.Name, account.ID, account.Status, account.EntityID, account.ProgramID}, " "),
		})
	}
	return ui.PromptSelect(label, options)
}

func chooseCard(cards []app.CardSummary, label string) (string, error) {
	return ui.PromptSelect(label, buildCardOptions(cards))
}

func chooseExternalAccount(accounts []app.ExternalAccountSummary, label string) (string, error) {
	options := make([]ui.Option, 0, len(accounts))
	for _, account := range accounts {
		descriptionParts := []string{}
		if account.Status != "" {
			descriptionParts = append(descriptionParts, account.Status)
		}
		if account.AccountNumberMasked != "" {
			descriptionParts = append(descriptionParts, account.AccountNumberMasked)
		}
		if account.RoutingNumber != "" {
			descriptionParts = append(descriptionParts, "rt:"+account.RoutingNumber)
		}
		searchTerms := []string{account.Description, account.ID, account.AccountHolder, account.Status, account.AccountNumberMasked, account.RoutingNumber}
		options = append(options, ui.Option{
			Label:       firstNonEmptyLabel(account.Description, account.ID),
			Value:       account.ID,
			Description: strings.Join(descriptionParts, "  "),
			Search:      strings.Join(searchTerms, " "),
		})
	}
	return ui.PromptSelect(label, options)
}

func chooseTransfer(items []app.TransferSummary, label string) (string, error) {
	options := make([]ui.Option, 0, len(items))
	for _, item := range items {
		searchTerms := []string{item.ID, item.Rail, item.Status, item.Counterparty, item.ExternalAccountID}
		description := strings.TrimSpace(strings.Join([]string{
			util.FormatUSDMinor(item.AmountCents),
			item.Status,
			item.Counterparty,
		}, "  "))
		options = append(options, ui.Option{
			Label:       fmt.Sprintf("%s %s", item.Rail, item.ID),
			Value:       item.ID,
			Description: description,
			Search:      strings.Join(searchTerms, " "),
		})
	}
	return ui.PromptSelect(label, options)
}

func isInteractiveRequested(opts *RootOptions) bool {
	return !opts.JSON && (opts.Interactive || ui.IsTTY())
}

func terminalMenuRequested(opts *RootOptions) bool {
	return !opts.JSON && ui.IsTTY()
}

func promptInt64(label string, required bool) (int64, error) {
	value, err := ui.PromptString(label, required)
	if err != nil {
		return 0, err
	}
	if strings.TrimSpace(value) == "" {
		return 0, nil
	}
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer", strings.ToLower(label))
	}
	return parsed, nil
}

func promptBool(label string, yesLabel, noLabel string) (bool, error) {
	if yesLabel == "" {
		yesLabel = "Yes"
	}
	if noLabel == "" {
		noLabel = "No"
	}
	value, err := ui.PromptSelect(label, []ui.Option{
		{Label: yesLabel, Value: "yes"},
		{Label: noLabel, Value: "no"},
	})
	if err != nil {
		return false, err
	}
	return value == "yes", nil
}

func promptRail(label string) (string, error) {
	return ui.PromptSelect(label, []ui.Option{
		{Label: "Account transfer", Value: "account", Description: "Move funds between Increase accounts"},
		{Label: "ACH transfer", Value: "ach", Description: "Send money over ACH"},
		{Label: "Real-Time Payments transfer", Value: "real_time_payments", Description: "Send an RTP transfer"},
		{Label: "FedNow transfer", Value: "fednow", Description: "Send a FedNow transfer"},
		{Label: "Wire transfer", Value: "wire", Description: "Send a wire transfer"},
	})
}

func stringPtr(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}

func boolPtr(value bool) *bool {
	return &value
}

func commandHasSubcommands(cmd *cobra.Command) bool {
	return len(cmd.Commands()) > 0
}

func invokeCommand(parent *cobra.Command, child *cobra.Command, args ...string) error {
	child.SetContext(parent.Context())
	child.SetArgs(args)
	if child.RunE != nil && !commandHasSubcommands(child) {
		return child.RunE(child, args)
	}
	return child.ExecuteContext(parent.Context())
}

var (
	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("69"))
	headerStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230")).Background(lipgloss.Color("24")).Padding(0, 1)
	keyStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("110"))
	valueStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	mutedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	tokenStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("221"))
	previewStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("230"))
	panelStyle   = lipgloss.NewStyle().BorderStyle(asciiBorder()).BorderForeground(lipgloss.Color("63")).Padding(1, 2)
)

func asciiBorder() lipgloss.Border {
	return lipgloss.Border{
		Top:          "-",
		Bottom:       "-",
		Left:         "|",
		Right:        "|",
		TopLeft:      "+",
		TopRight:     "+",
		BottomLeft:   "+",
		BottomRight:  "+",
		MiddleLeft:   "+",
		MiddleRight:  "+",
		Middle:       "-",
		MiddleTop:    "+",
		MiddleBottom: "+",
	}
}

func renderTable(title string, headers []string, rows [][]string) string {
	if len(rows) == 0 {
		return panelStyle.Render(titleStyle.Render(title) + "\n" + mutedStyle.Render("No results"))
	}

	available := terminalWidth() - 8
	if available < 72 {
		return renderCompactTable(title, headers, rows)
	}

	widths := make([]int, len(headers))
	minimums := make([]int, len(headers))
	for i, header := range headers {
		widths[i] = displayWidth(header)
		minimums[i] = minColumnWidth(header)
		if minimums[i] < widths[i] {
			minimums[i] = widths[i]
		}
	}
	for _, row := range rows {
		for i, cell := range row {
			cellWidth := displayWidth(cell)
			if cellWidth > widths[i] {
				widths[i] = cellWidth
			}
		}
	}
	for i, header := range headers {
		maxWidth := suggestedMaxColumnWidth(header)
		if widths[i] > maxWidth {
			widths[i] = maxWidth
		}
		if widths[i] < minimums[i] {
			widths[i] = minimums[i]
		}
	}
	totalWidth := sum(widths) + len(headers) - 1
	for totalWidth > available {
		shrunk := false
		for i, header := range headers {
			if widths[i] > minimums[i] && canShrinkColumn(header) {
				widths[i]--
				totalWidth--
				shrunk = true
				if totalWidth <= available {
					break
				}
			}
		}
		if !shrunk {
			break
		}
	}

	var out strings.Builder
	out.WriteString(titleStyle.Render(title))
	out.WriteString("\n")
	for i, header := range headers {
		out.WriteString(headerStyle.Width(widths[i]).MaxWidth(widths[i]).Render(truncate(header, widths[i])))
		if i < len(headers)-1 {
			out.WriteString(" ")
		}
	}
	out.WriteString("\n")
	for _, row := range rows {
		for i, cell := range row {
			width := widths[i]
			style := valueStyle.Width(width).MaxWidth(width)
			if isMutedColumn(headers[i]) {
				style = mutedStyle.Width(width).MaxWidth(width)
			}
			if isRightAlignedColumn(headers[i]) {
				style = style.Align(lipgloss.Right)
			}
			out.WriteString(style.Render(truncate(cell, width)))
			if i < len(row)-1 {
				out.WriteString(" ")
			}
		}
		out.WriteString("\n")
	}
	return panelStyle.Render(strings.TrimRight(out.String(), "\n"))
}

func truncate(value string, width int) string {
	if width <= 0 || displayWidth(value) <= width {
		return value
	}
	if width <= 3 {
		return string([]rune(value)[:width])
	}
	runes := []rune(value)
	if len(runes) <= width {
		return value
	}
	return string(runes[:width-3]) + "..."
}

func renderCompactTable(title string, headers []string, rows [][]string) string {
	lines := []string{titleStyle.Render(title)}
	for i, row := range rows {
		if i > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, mutedStyle.Render(fmt.Sprintf("[%d]", i+1)))
		for j, header := range headers {
			value := ""
			if j < len(row) {
				value = row[j]
			}
			lines = append(lines, fmt.Sprintf("%s %s", keyStyle.Render(strings.ToLower(header)+":"), valueStyle.Render(value)))
		}
	}
	return panelStyle.Render(strings.Join(lines, "\n"))
}

func terminalWidth() int {
	if width, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && width > 0 {
		return width
	}
	if raw := os.Getenv("COLUMNS"); raw != "" {
		if width, err := strconv.Atoi(raw); err == nil && width > 0 {
			return width
		}
	}
	return 120
}

func displayWidth(value string) int {
	if value == "" {
		return 0
	}
	return utf8.RuneCountInString(value)
}

func sum(values []int) int {
	total := 0
	for _, value := range values {
		total += value
	}
	return total
}

func minColumnWidth(header string) int {
	switch normalizedHeader(header) {
	case "amount", "last4", "status":
		return 8
	case "id", "account", "entity", "program", "routing", "rail":
		return 10
	case "created":
		return 16
	case "counterparty", "description", "external account":
		return 14
	default:
		return max(8, displayWidth(header))
	}
}

func suggestedMaxColumnWidth(header string) int {
	switch normalizedHeader(header) {
	case "description", "counterparty":
		return 24
	case "id", "account", "entity", "program", "external account":
		return 20
	case "created":
		return 20
	default:
		return 16
	}
}

func canShrinkColumn(header string) bool {
	switch normalizedHeader(header) {
	case "description", "counterparty", "id", "account", "entity", "program", "external account", "created":
		return true
	default:
		return false
	}
}

func isMutedColumn(header string) bool {
	switch normalizedHeader(header) {
	case "status", "created", "rail":
		return true
	default:
		return false
	}
}

func isRightAlignedColumn(header string) bool {
	return normalizedHeader(header) == "amount"
}

func normalizedHeader(header string) string {
	return strings.ToLower(strings.TrimSpace(header))
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func firstNonEmptyLabel(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
