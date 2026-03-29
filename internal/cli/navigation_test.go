package cli

import (
	"context"
	"errors"
	"testing"

	"github.com/gitsoecode/increasex-cli-mcp/internal/ui"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

func TestPromptStringNavigationRecognizesCommands(t *testing.T) {
	original := runPromptString
	t.Cleanup(func() { runPromptString = original })

	runPromptString = func(label string, required bool) (string, error) {
		return "back", nil
	}
	if _, err := promptStringNavigation("Label", true); !isNavigateBack(err) {
		t.Fatalf("promptStringNavigation() error = %v, want back navigation", err)
	}

	runPromptString = func(label string, required bool) (string, error) {
		return "exit", nil
	}
	if _, err := promptStringNavigation("Label", true); !isNavigateExit(err) {
		t.Fatalf("promptStringNavigation() error = %v, want exit navigation", err)
	}
}

func TestPromptStringNavigationMapsAbortToBack(t *testing.T) {
	original := runPromptString
	t.Cleanup(func() { runPromptString = original })

	runPromptString = func(label string, required bool) (string, error) {
		return "", promptui.ErrAbort
	}
	if _, err := promptStringNavigation("Label", true); !isNavigateBack(err) {
		t.Fatalf("promptStringNavigation() error = %v, want back navigation on abort", err)
	}
}

func TestPromptSelectNavigationAppendsExplicitOptions(t *testing.T) {
	original := runPromptSelect
	t.Cleanup(func() { runPromptSelect = original })

	runPromptSelect = func(label string, options []ui.Option) (string, error) {
		if got := options[len(options)-2].Label; got != "Back" {
			t.Fatalf("expected Back option, got %q", got)
		}
		if got := options[len(options)-1].Label; got != "Exit" {
			t.Fatalf("expected Exit option, got %q", got)
		}
		return "__nav_exit", nil
	}

	if _, err := promptSelectNavigation("Label", []ui.Option{{Label: "One", Value: "1"}}, navBack, navExit); !isNavigateExit(err) {
		t.Fatalf("promptSelectNavigation() error = %v, want exit navigation", err)
	}
}

func TestPromptSelectNavigationMapsAbortToBack(t *testing.T) {
	original := runPromptSelect
	t.Cleanup(func() { runPromptSelect = original })

	runPromptSelect = func(label string, options []ui.Option) (string, error) {
		return "", promptui.ErrAbort
	}
	if _, err := promptSelectNavigation("Label", []ui.Option{{Label: "One", Value: "1"}}, navBack); !isNavigateBack(err) {
		t.Fatalf("promptSelectNavigation() error = %v, want back navigation on abort", err)
	}
}

func TestBubbleNavigationSuppressesTopLevelBackAndExit(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	if err := bubbleNavigation(cmd, errNavigateBack); err != nil {
		t.Fatalf("bubbleNavigation(back) = %v, want nil at top level", err)
	}
	if err := bubbleNavigation(cmd, errNavigateExit); err != nil {
		t.Fatalf("bubbleNavigation(exit) = %v, want nil at top level", err)
	}
}

func TestBubbleNavigationPreservesNestedBackAndExit(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetContext(menuContext(context.Background()))

	if err := bubbleNavigation(cmd, errNavigateBack); !isNavigateBack(err) {
		t.Fatalf("bubbleNavigation(back) = %v, want nested back navigation", err)
	}
	if err := bubbleNavigation(cmd, errNavigateExit); !isNavigateExit(err) {
		t.Fatalf("bubbleNavigation(exit) = %v, want nested exit navigation", err)
	}
}

func TestBubbleNavigationLeavesNormalErrorsUntouched(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	boom := errors.New("boom")
	if err := bubbleNavigation(cmd, boom); !errors.Is(err, boom) {
		t.Fatalf("bubbleNavigation(normal) = %v, want original error", err)
	}
}
