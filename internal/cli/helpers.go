package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/gitsoecode/increasex-cli-mcp/internal/app"
	"github.com/gitsoecode/increasex-cli-mcp/internal/auth"
	increasex "github.com/gitsoecode/increasex-cli-mcp/internal/increase"
	"github.com/gitsoecode/increasex-cli-mcp/internal/ui"
	"github.com/gitsoecode/increasex-cli-mcp/internal/util"
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
		lines = append(lines, renderDetailLine(key, fmt.Sprint(values[key]))...)
	}
	fmt.Println(renderPanel(strings.Join(lines, "\n")))
}

func printAccounts(accounts []app.AccountSummary) {
	items := make([]recordItem, 0, len(accounts))
	for _, account := range accounts {
		items = append(items, recordItem{
			Title: firstNonEmptyLabel(account.Name, account.ID),
			Meta:  account.Status,
			Fields: []recordField{
				{Label: "id", Value: account.ID},
				{Label: "entity", Value: emptyDash(account.EntityID)},
				{Label: "program", Value: emptyDash(account.ProgramID)},
				{Label: "created_at", Value: emptyDash(account.CreatedAt)},
			},
		})
	}
	fmt.Println(renderRecordList("Accounts", items))
}

func printAccountNumbers(numbers []app.AccountNumberSummary) {
	fmt.Println(renderAccountNumberList(numbers))
}

func printAccountNumberDetails(number *app.AccountNumberDetails) {
	lines := []string{titleStyle.Render("Details")}
	lines = append(lines, renderDetailLine("id", number.ID)...)
	lines = append(lines, renderDetailLine("name", number.Name)...)
	lines = append(lines, renderDetailLine("account", formatAccountNumberParentAccount(number.AccountName, number.AccountID))...)
	if strings.TrimSpace(number.AccountName) == "" {
		lines = append(lines, renderDetailLine("account_id", number.AccountID)...)
	}
	lines = append(lines, renderDetailLine("routing_number", number.RoutingNumber)...)
	if strings.TrimSpace(number.AccountNumber) != "" {
		lines = append(lines, renderDetailLine("account_number", number.AccountNumber)...)
	} else {
		lines = append(lines, renderDetailLine("account_number_masked", number.AccountNumberMasked)...)
	}
	lines = append(lines, renderDetailLine("status", number.Status)...)
	lines = append(lines, renderDetailLine("inbound_ach", formatInboundACHStatus(number.InboundACH))...)
	lines = append(lines, renderDetailLine("inbound_checks", formatInboundChecksStatus(number.InboundChecks))...)
	if strings.TrimSpace(number.IdempotencyKey) != "" {
		lines = append(lines, renderDetailLine("idempotency_key", number.IdempotencyKey)...)
	}
	lines = append(lines, renderDetailLine("created_at", number.CreatedAt)...)
	fmt.Println(renderPanel(strings.Join(lines, "\n")))
}

func printTransactions(items []app.TransactionSummary) {
	records := make([]recordItem, 0, len(items))
	for _, item := range items {
		fields := []recordField{
			{Label: "id", Value: item.ID},
			{Label: "account", Value: emptyDash(item.AccountID)},
			{Label: "type", Value: emptyDash(item.Type)},
			{Label: "created_at", Value: emptyDash(item.CreatedAt)},
		}
		if strings.TrimSpace(item.CounterpartySummary) != "" {
			fields = append(fields, recordField{Label: "counterparty", Value: item.CounterpartySummary})
		}
		if strings.TrimSpace(item.RouteType) != "" {
			fields = append(fields, recordField{Label: "route_type", Value: item.RouteType})
		}
		if strings.TrimSpace(item.RouteID) != "" {
			fields = append(fields, recordField{Label: "route_id", Value: item.RouteID})
		}
		records = append(records, recordItem{
			Title:  firstNonEmptyLabel(item.Description, item.ID),
			Meta:   joinRecordMeta(util.FormatUSDMinor(item.AmountCents), item.Direction),
			Fields: fields,
		})
	}
	fmt.Println(renderRecordList("Transactions", records))
}

func printCards(cards []app.CardSummary) {
	items := make([]recordItem, 0, len(cards))
	for _, card := range cards {
		title := strings.TrimSpace(card.Description)
		if title == "" && strings.TrimSpace(card.Last4) != "" {
			title = "Card • " + card.Last4
		}
		if title == "" {
			title = card.ID
		}
		items = append(items, recordItem{
			Title: title,
			Meta:  card.Status,
			Fields: []recordField{
				{Label: "id", Value: card.ID},
				{Label: "account", Value: emptyDash(card.AccountID)},
				{Label: "last4", Value: emptyDash(card.Last4)},
				{Label: "created_at", Value: emptyDash(card.CreatedAt)},
			},
		})
	}
	fmt.Println(renderRecordList("Cards", items))
}

func printExternalAccounts(accounts []app.ExternalAccountSummary) {
	items := make([]recordItem, 0, len(accounts))
	for _, account := range accounts {
		items = append(items, recordItem{
			Title: firstNonEmptyLabel(account.Description, account.ID),
			Meta:  account.Status,
			Fields: []recordField{
				{Label: "id", Value: account.ID},
				{Label: "holder", Value: emptyDash(account.AccountHolder)},
				{Label: "funding", Value: emptyDash(account.Funding)},
				{Label: "routing", Value: emptyDash(account.RoutingNumber)},
				{Label: "account", Value: emptyDash(account.AccountNumberMasked)},
				{Label: "created_at", Value: emptyDash(account.CreatedAt)},
			},
		})
	}
	fmt.Println(renderRecordList("External Accounts", items))
}

func printTransfers(title string, items []app.TransferSummary) {
	records := make([]recordItem, 0, len(items))
	for _, item := range items {
		records = append(records, recordItem{
			Title: firstNonEmptyLabel(item.Counterparty, item.ID),
			Meta:  joinRecordMeta(util.FormatUSDMinor(item.AmountCents), item.Rail, item.Status),
			Fields: []recordField{
				{Label: "id", Value: item.ID},
				{Label: "external_account", Value: emptyDash(item.ExternalAccountID)},
				{Label: "created_at", Value: emptyDash(item.CreatedAt)},
			},
		})
	}
	fmt.Println(renderRecordList(title, records))
}

func renderAccountNumberList(numbers []app.AccountNumberSummary) string {
	items := make([]recordItem, 0, len(numbers))
	for _, number := range numbers {
		items = append(items, recordItem{
			Title: firstNonEmptyLabel(number.Name, number.ID),
			Meta:  number.Status,
			Fields: []recordField{
				{Label: "account", Value: formatAccountNumberParentAccount(number.AccountName, number.AccountID)},
				{Label: "routing", Value: emptyDash(number.RoutingNumber)},
				{Label: "number", Value: emptyDash(number.AccountNumberMasked)},
				{Label: "ach", Value: formatInboundACHStatus(number.InboundACH)},
				{Label: "checks", Value: formatInboundChecksStatus(number.InboundChecks)},
				{Label: "id", Value: number.ID},
			},
		})
	}
	return renderRecordList("Account Numbers", items)
}

func formatInboundACHStatus(inboundACH *app.InboundACHInput) string {
	if inboundACH == nil || strings.TrimSpace(inboundACH.DebitStatus) == "" {
		return "-"
	}
	return inboundACH.DebitStatus
}

func formatInboundChecksStatus(inboundChecks *app.InboundChecksInput) string {
	if inboundChecks == nil || strings.TrimSpace(inboundChecks.Status) == "" {
		return "-"
	}
	return inboundChecks.Status
}

func formatAccountNumberParentAccount(accountName, accountID string) string {
	name := strings.TrimSpace(accountName)
	id := strings.TrimSpace(accountID)
	switch {
	case name != "" && id != "":
		return fmt.Sprintf("%s (%s)", name, id)
	case name != "":
		return name
	default:
		return id
	}
}

func printPreview(preview *app.PreviewResult) {
	lines := []string{titleStyle.Render("Preview")}
	lines = append(lines, renderWrappedLine(previewStyle, preview.Summary)...)
	if preview.Details != nil {
		keys := make([]string, 0, len(preview.Details))
		for key := range preview.Details {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			lines = append(lines, renderDetailLine(key, fmt.Sprint(preview.Details[key]))...)
		}
	}
	lines = append(lines, renderDetailLine("confirmation_token", preview.ConfirmationToken)...)
	fmt.Println(renderPanel(strings.Join(lines, "\n")))
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
	return promptSelectNavigation(label, options, navBack, navExit)
}

func chooseAccountNumber(numbers []app.AccountNumberSummary, label string) (string, error) {
	return promptSelectNavigation(label, buildAccountNumberOptions(numbers), navBack, navExit)
}

func buildAccountNumberOptions(numbers []app.AccountNumberSummary) []ui.Option {
	options := make([]ui.Option, 0, len(numbers))
	for _, number := range numbers {
		descriptionParts := []string{}
		parentAccount := formatAccountNumberParentAccount(number.AccountName, number.AccountID)
		if parentAccount != "" {
			descriptionParts = append(descriptionParts, parentAccount)
		}
		if number.AccountNumberMasked != "" {
			descriptionParts = append(descriptionParts, number.AccountNumberMasked)
		}
		if number.RoutingNumber != "" {
			descriptionParts = append(descriptionParts, "rt:"+number.RoutingNumber)
		}
		if number.Status != "" {
			descriptionParts = append(descriptionParts, number.Status)
		}
		searchTerms := []string{
			number.Name,
			number.ID,
			number.AccountName,
			number.AccountID,
			number.AccountNumberMasked,
			number.RoutingNumber,
			number.Status,
		}
		options = append(options, ui.Option{
			Label:       firstNonEmptyLabel(number.Name, number.ID),
			Value:       number.ID,
			Description: strings.Join(descriptionParts, "  "),
			Search:      strings.Join(searchTerms, " "),
		})
	}
	return options
}

func chooseCard(cards []app.CardSummary, label string) (string, error) {
	return promptSelectNavigation(label, buildCardOptions(cards), navBack, navExit)
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
	return promptSelectNavigation(label, options, navBack, navExit)
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
	return promptSelectNavigation(label, options, navBack, navExit)
}

func isInteractiveRequested(opts *RootOptions) bool {
	return !opts.JSON && (opts.Interactive || ui.IsTTY())
}

func terminalMenuRequested(opts *RootOptions) bool {
	return terminalMenuAllowed(opts, ui.IsTTY())
}

func terminalMenuAllowed(opts *RootOptions, isTTY bool) bool {
	return isTTY && opts != nil && !opts.JSON && !opts.Advanced
}

func promptInt64(label string, required bool) (int64, error) {
	value, err := promptStringNavigation(label, required)
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
	value, err := promptSelectNavigation(label, []ui.Option{
		{Label: yesLabel, Value: "yes"},
		{Label: noLabel, Value: "no"},
	}, navBack, navExit)
	if err != nil {
		return false, err
	}
	return value == "yes", nil
}

func promptRail(label string) (string, error) {
	return promptSelectNavigation(label, []ui.Option{
		{Label: "Account transfer", Value: "account", Description: "Move funds between Increase accounts"},
		{Label: "ACH transfer", Value: "ach", Description: "Send money over ACH"},
		{Label: "Real-Time Payments transfer", Value: "real_time_payments", Description: "Send an RTP transfer"},
		{Label: "FedNow transfer", Value: "fednow", Description: "Send a FedNow transfer"},
		{Label: "Wire transfer", Value: "wire", Description: "Send a wire transfer"},
	}, navBack, navExit)
}

func transferConfirmationPrompt(rail string, requireApproval *bool) string {
	action := "Execute"
	if requireApproval != nil && *requireApproval {
		action = "Queue"
	}
	switch rail {
	case "account":
		if action == "Queue" {
			return "Queue this account transfer for approval?"
		}
		return "Execute this account transfer?"
	case "ach":
		if action == "Queue" {
			return "Queue this ACH transfer for approval?"
		}
		return "Execute this ACH transfer?"
	case "real_time_payments":
		if action == "Queue" {
			return "Queue this Real-Time Payments transfer for approval?"
		}
		return "Execute this Real-Time Payments transfer?"
	case "fednow":
		if action == "Queue" {
			return "Queue this FedNow transfer for approval?"
		}
		return "Execute this FedNow transfer?"
	case "wire":
		if action == "Queue" {
			return "Queue this wire transfer for approval?"
		}
		return "Execute this wire transfer?"
	default:
		if action == "Queue" {
			return "Queue this transfer for approval?"
		}
		return "Execute this transfer?"
	}
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
	child.SetContext(menuContext(parent.Context()))
	child.SetArgs(args)
	return child.ExecuteContext(parent.Context())
}

func parseAdvancedCommand(input string) ([]string, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return nil, nil
	}

	args := make([]string, 0, 8)
	var current strings.Builder
	var quote rune
	escaped := false

	flush := func() {
		if current.Len() == 0 {
			return
		}
		args = append(args, current.String())
		current.Reset()
	}

	for _, r := range trimmed {
		switch {
		case escaped:
			current.WriteRune(r)
			escaped = false
		case r == '\\':
			escaped = true
		case quote != 0:
			if r == quote {
				quote = 0
				continue
			}
			current.WriteRune(r)
		case r == '\'' || r == '"':
			quote = r
		case unicode.IsSpace(r):
			flush()
		default:
			current.WriteRune(r)
		}
	}

	if escaped {
		return nil, fmt.Errorf("advanced command has an unfinished escape sequence")
	}
	if quote != 0 {
		return nil, fmt.Errorf("advanced command has an unmatched quote")
	}
	flush()

	if len(args) == 0 {
		return nil, nil
	}

	first := filepath.Base(args[0])
	if first == "increasex" {
		args = args[1:]
	}
	if len(args) == 1 && strings.EqualFold(args[0], "back") {
		return nil, nil
	}
	return args, nil
}

func openBrowserURL(rawURL string) error {
	command, args, err := browserOpenCommand(rawURL)
	if err != nil {
		return err
	}
	if err := exec.Command(command, args...).Start(); err != nil {
		return fmt.Errorf("unable to open browser: %w", err)
	}
	return nil
}

func browserOpenCommand(rawURL string) (string, []string, error) {
	switch runtime.GOOS {
	case "darwin":
		return "open", []string{rawURL}, nil
	case "linux":
		return "xdg-open", []string{rawURL}, nil
	case "windows":
		return "rundll32", []string{"url.dll,FileProtocolHandler", rawURL}, nil
	default:
		return "", nil, fmt.Errorf("opening a browser is not supported on %s", runtime.GOOS)
	}
}

var (
	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("69"))
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

type recordField struct {
	Label string
	Value string
}

type recordItem struct {
	Title  string
	Meta   string
	Fields []recordField
}

func renderRecordList(title string, items []recordItem) string {
	if len(items) == 0 {
		return renderPanel(titleStyle.Render(title) + "\n" + mutedStyle.Render("No results"))
	}

	lines := []string{titleStyle.Render(title)}
	for i, item := range items {
		if i > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, valueStyle.Render(item.Title))
		if strings.TrimSpace(item.Meta) != "" {
			lines = append(lines, mutedStyle.Render(item.Meta))
		}
		for _, field := range item.Fields {
			lines = append(lines, renderDetailLine(field.Label, field.Value)...)
		}
	}
	return renderPanel(strings.Join(lines, "\n"))
}

func renderDetailLine(key, value string) []string {
	label := key + ":"
	available := terminalWidth() - 18
	if available < 24 {
		available = 24
	}
	if displayWidth(value) <= available {
		return []string{fmt.Sprintf("%s %s", keyStyle.Render(label), valueStyle.Render(value))}
	}
	wrapped := wrapText(value, available)
	lines := []string{fmt.Sprintf("%s %s", keyStyle.Render(label), valueStyle.Render(wrapped[0]))}
	indent := strings.Repeat(" ", max(displayWidth(label)+1, 18))
	for _, line := range wrapped[1:] {
		lines = append(lines, indent+valueStyle.Render(line))
	}
	return lines
}

func renderWrappedLine(style lipgloss.Style, value string) []string {
	available := terminalWidth() - 10
	if available < 24 {
		available = 24
	}
	if displayWidth(value) <= available {
		return []string{style.Render(value)}
	}
	wrapped := wrapText(value, available)
	lines := make([]string, 0, len(wrapped))
	for _, line := range wrapped {
		lines = append(lines, style.Render(line))
	}
	return lines
}

func renderPanel(content string) string {
	width := terminalWidth() - 2
	if width < 24 {
		width = 24
	}
	return panelStyle.MaxWidth(width).Render(content)
}

func terminalWidth() int {
	if raw := os.Getenv("COLUMNS"); raw != "" {
		if width, err := strconv.Atoi(raw); err == nil && width > 0 {
			return width
		}
	}
	widths := []int{}
	for _, file := range []*os.File{os.Stdout, os.Stderr, os.Stdin} {
		if file == nil {
			continue
		}
		if width, _, err := term.GetSize(int(file.Fd())); err == nil && width > 0 {
			widths = append(widths, width)
		}
	}
	if len(widths) > 0 {
		width := widths[0]
		for _, candidate := range widths[1:] {
			if candidate < width {
				width = candidate
			}
		}
		return width
	}
	return 80
}

func displayWidth(value string) int {
	if value == "" {
		return 0
	}
	return utf8.RuneCountInString(value)
}

func wrapText(value string, width int) []string {
	if width <= 0 || displayWidth(value) <= width {
		return []string{value}
	}
	runes := []rune(value)
	lines := make([]string, 0, (len(runes)/width)+1)
	for len(runes) > 0 {
		end := width
		if end > len(runes) {
			end = len(runes)
		}
		lines = append(lines, string(runes[:end]))
		runes = runes[end:]
	}
	return lines
}

func joinRecordMeta(parts ...string) string {
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			filtered = append(filtered, part)
		}
	}
	return strings.Join(filtered, " • ")
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

func emptyDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}
