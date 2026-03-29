package cli

import (
	"context"
	"errors"
	"strings"

	"github.com/gitsoecode/increasex-cli-mcp/internal/ui"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

var (
	errNavigateBack = errors.New("interactive navigation back")
	errNavigateExit = errors.New("interactive navigation exit")
	runPromptString = ui.PromptString
	runPromptSelect = ui.PromptSelect
)

type navigationOption int

const (
	navBack navigationOption = iota
	navExit
)

type menuContextKey struct{}

func isNavigateBack(err error) bool {
	return errors.Is(err, errNavigateBack)
}

func isNavigateExit(err error) bool {
	return errors.Is(err, errNavigateExit)
}

func isPromptAbort(err error) bool {
	return errors.Is(err, promptui.ErrAbort)
}

func bubbleNavigation(cmd *cobra.Command, err error) error {
	if err == nil {
		return nil
	}
	if !isNavigateBack(err) && !isNavigateExit(err) {
		return err
	}
	if menuDepthFromContext(cmd.Context()) == 0 {
		return nil
	}
	return err
}

func menuContext(parent context.Context) context.Context {
	depth := menuDepthFromContext(parent)
	return context.WithValue(parent, menuContextKey{}, depth+1)
}

func menuDepthFromContext(ctx context.Context) int {
	if ctx == nil {
		return 0
	}
	depth, _ := ctx.Value(menuContextKey{}).(int)
	return depth
}

func promptStringNavigation(label string, required bool) (string, error) {
	value, err := runPromptString(label, required)
	if err != nil {
		if isPromptAbort(err) {
			return "", errNavigateBack
		}
		return "", err
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "back":
		return "", errNavigateBack
	case "exit":
		return "", errNavigateExit
	default:
		return value, nil
	}
}

func promptSelectNavigation(label string, options []ui.Option, navigation ...navigationOption) (string, error) {
	items := append([]ui.Option{}, options...)
	for _, item := range navigation {
		switch item {
		case navBack:
			items = append(items, ui.Option{Label: "Back", Value: "__nav_back", Description: "Return to the previous menu"})
		case navExit:
			items = append(items, ui.Option{Label: "Exit", Value: "__nav_exit", Description: "Return to the shell"})
		}
	}

	value, err := runPromptSelect(label, items)
	if err != nil {
		if isPromptAbort(err) {
			return "", errNavigateBack
		}
		return "", err
	}
	switch value {
	case "__nav_back":
		return "", errNavigateBack
	case "__nav_exit":
		return "", errNavigateExit
	default:
		return value, nil
	}
}

func promptConfirmationNavigation(label string) (bool, error) {
	value, err := promptSelectNavigation(label, []ui.Option{
		{Label: "Yes", Value: "yes", Description: "Continue with this action"},
		{Label: "No", Value: "no", Description: "Cancel and return to the previous menu"},
	}, navBack, navExit)
	if err != nil {
		return false, err
	}
	return value == "yes", nil
}
