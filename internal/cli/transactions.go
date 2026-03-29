package cli

import (
	"strings"

	"github.com/gitsoecode/increasex-cli-mcp/internal/app"
	"github.com/gitsoecode/increasex-cli-mcp/internal/ui"
	"github.com/spf13/cobra"
)

func newTransactionsCmd(ctx *Context) *cobra.Command {
	var accountID, since, until, period, cursor string
	var limit int64
	var categories []string
	cmd := &cobra.Command{
		Use:   "transactions",
		Short: "List recent transactions for an account and time period",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, api, err := ctx.resolve(cmd.Context())
			if err != nil {
				return printEnvelopeJSON(nil, "", err)
			}
			if accountID == "" && isInteractiveRequested(ctx.Options) {
				accounts, _, err := ctx.Services.ListAccounts(cmd.Context(), api, "", 25, "")
				if err != nil {
					return err
				}
				accountID, err = chooseAccount(accounts, "Select an account")
				if err != nil {
					return bubbleNavigation(cmd, err)
				}
			}
			if isInteractiveRequested(ctx.Options) && strings.TrimSpace(since) == "" && strings.TrimSpace(until) == "" && strings.TrimSpace(period) == "" {
				nextSince, nextUntil, nextPeriod, err := promptTransactionTimeRange()
				if err != nil {
					return bubbleNavigation(cmd, err)
				}
				since = nextSince
				until = nextUntil
				period = nextPeriod
			}
			items, requestID, err := ctx.Services.ListRecentTransactions(cmd.Context(), api, app.ListTransactionsInput{
				AccountID:  accountID,
				TimeRange:  transactionTimeRangeInput(since, until, period),
				Cursor:     cursor,
				Limit:      limit,
				Categories: categories,
			})
			if ctx.Options.JSON {
				return printEnvelopeJSON(map[string]any{"transactions": items}, requestID, err)
			}
			if err != nil {
				return err
			}
			printTransactions(items)
			return nil
		},
	}
	cmd.Flags().StringVar(&accountID, "account-id", "", "account id")
	cmd.Flags().StringVar(&since, "since", "", "RFC3339 lower bound")
	cmd.Flags().StringVar(&until, "until", "", "RFC3339 upper bound")
	cmd.Flags().StringVar(&period, "period", "", "preset period: last-7d, last-30d, current-month, previous-month")
	cmd.Flags().StringVar(&cursor, "cursor", "", "page cursor")
	cmd.Flags().Int64Var(&limit, "limit", 20, "maximum transactions to return")
	cmd.Flags().StringSliceVar(&categories, "category", nil, "transaction categories")
	return cmd
}

func transactionTimeRangeInput(since, until, period string) app.TransactionTimeRangeInput {
	if strings.TrimSpace(since) != "" || strings.TrimSpace(until) != "" {
		period = ""
	}
	return app.TransactionTimeRangeInput{
		Since:  since,
		Until:  until,
		Period: period,
	}
}

func promptTransactionTimeRange() (string, string, string, error) {
	choice, err := promptSelectNavigation("Transaction time period", []ui.Option{
		{Label: "Default", Value: "default", Description: "Use the default last 30 days"},
		{Label: "Last 7 days", Value: "last-7d", Description: "Recent activity from the past week"},
		{Label: "Last 30 days", Value: "last-30d", Description: "Recent activity from the past month"},
		{Label: "Current month", Value: "current-month", Description: "From the first day of this month until now"},
		{Label: "Previous month", Value: "previous-month", Description: "The full prior calendar month"},
		{Label: "Custom RFC3339 range", Value: "custom", Description: "Enter exact since and optional until timestamps"},
	}, navBack, navExit)
	if err != nil {
		return "", "", "", err
	}
	switch choice {
	case "default":
		return "", "", "", nil
	case "last-7d", "last-30d", "current-month", "previous-month":
		return "", "", choice, nil
	case "custom":
		return promptCustomTransactionTimeRange()
	default:
		return "", "", "", nil
	}
}

func promptCustomTransactionTimeRange() (string, string, string, error) {
	since, err := promptStringNavigation("Since (RFC3339)", true)
	if err != nil {
		return "", "", "", err
	}
	until, err := promptOptionalSelect("Include an upper bound?", []ui.Option{
		{Label: "No upper bound", Value: "", Description: "List transactions from the lower bound forward"},
		{Label: "Enter until timestamp", Value: "enter", Description: "Provide an RFC3339 upper bound"},
	})
	if err != nil {
		return "", "", "", err
	}
	if until != "enter" {
		return since, "", "", nil
	}
	value, err := promptStringNavigation("Until (RFC3339)", true)
	if err != nil {
		return "", "", "", err
	}
	return since, value, "", nil
}
