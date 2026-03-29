package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/jessevaughan/increasex/internal/app"
	"github.com/jessevaughan/increasex/internal/auth"
	increasex "github.com/jessevaughan/increasex/internal/increase"
	"github.com/jessevaughan/increasex/internal/ui"
	"github.com/jessevaughan/increasex/internal/util"
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

func isInteractiveRequested(opts *RootOptions) bool {
	return !opts.JSON && (opts.Interactive || ui.IsTTY())
}

func stringPtr(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
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

	widths := make([]int, len(headers))
	for i, header := range headers {
		widths[i] = len(header)
	}
	for _, row := range rows {
		for i, cell := range row {
			if len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	var out strings.Builder
	out.WriteString(titleStyle.Render(title))
	out.WriteString("\n")
	for i, header := range headers {
		out.WriteString(headerStyle.Width(widths[i] + 2).Render(header))
		if i < len(headers)-1 {
			out.WriteString(" ")
		}
	}
	out.WriteString("\n")
	for _, row := range rows {
		for i, cell := range row {
			style := valueStyle.Width(widths[i] + 2)
			if i == 2 || i == 3 {
				style = mutedStyle.Width(widths[i] + 2)
			}
			out.WriteString(style.Render(truncate(cell, widths[i]+2)))
			if i < len(row)-1 {
				out.WriteString(" ")
			}
		}
		out.WriteString("\n")
	}
	return panelStyle.Render(strings.TrimRight(out.String(), "\n"))
}

func truncate(value string, width int) string {
	if width <= 0 || len(value) <= width {
		return value
	}
	if width <= 3 {
		return value[:width]
	}
	return value[:width-3] + "..."
}
