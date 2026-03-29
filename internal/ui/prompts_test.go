package ui

import "testing"

func TestFitSelectOptionsToWidthTruncatesLongValues(t *testing.T) {
	t.Setenv("COLUMNS", "40")

	options := fitSelectOptionsToWidth([]Option{
		{
			Label:       "This is an extremely long label that should be truncated for the prompt",
			Value:       "card",
			Description: "This description is also very long and should not overflow the select box in narrow terminals",
		},
	})

	if got := options[0].Label; got == "" || len([]rune(got)) >= len([]rune("This is an extremely long label that should be truncated for the prompt")) || got[len(got)-3:] != "..." {
		t.Fatalf("label = %q, want truncated label with ellipsis", got)
	}
	if got := options[0].Description; got == "" || len([]rune(got)) >= len([]rune("This description is also very long and should not overflow the select box in narrow terminals")) || got[len(got)-3:] != "..." {
		t.Fatalf("description = %q, want truncated description with ellipsis", got)
	}
}
