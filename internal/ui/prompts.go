package ui

import (
	"errors"
	"os"
	"strconv"
	"strings"

	"github.com/manifoldco/promptui"
	"golang.org/x/term"
)

type Option struct {
	Label       string
	Value       string
	Description string
	Search      string
}

func IsTTY() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

func PromptString(label string, required bool) (string, error) {
	prompt := promptui.Prompt{
		Label: label,
		Validate: func(input string) error {
			if required && strings.TrimSpace(input) == "" {
				return errors.New("value is required")
			}
			return nil
		},
	}
	value, err := prompt.Run()
	if err != nil {
		return "", normalizePromptError(err)
	}
	return strings.TrimSpace(value), nil
}

func PromptSelect(label string, options []Option) (string, error) {
	displayOptions := fitSelectOptionsToWidth(options)
	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}",
		Active:   "-> {{ .Label | cyan }} {{ if .Description }}{{ .Description | faint }}{{ end }}",
		Inactive: "   {{ .Label }} {{ if .Description }}{{ .Description | faint }}{{ end }}",
		Selected: "{{ .Label | green }}",
		Details: `
{{ if .Description }}{{ .Description | faint }}{{ end }}`,
	}

	searcher := func(input string, index int) bool {
		option := options[index]
		haystack := option.Label
		if option.Search != "" {
			haystack += " " + option.Search
		}
		return strings.Contains(strings.ToLower(haystack), strings.ToLower(strings.TrimSpace(input)))
	}

	prompt := promptui.Select{
		Label:             label,
		Items:             displayOptions,
		Templates:         templates,
		Size:              min(12, len(displayOptions)),
		Searcher:          searcher,
		StartInSearchMode: true,
		HideSelected:      false,
	}

	index, _, err := prompt.Run()
	if err != nil {
		return "", normalizePromptError(err)
	}
	return options[index].Value, nil
}

func Confirm(label string) (bool, error) {
	value, err := PromptSelect(label, []Option{
		{Label: "Yes", Value: "yes", Description: "Continue with this action"},
		{Label: "No", Value: "no", Description: "Cancel and return to the shell"},
	})
	if err != nil {
		return false, err
	}
	return value == "yes", nil
}

func fitSelectOptionsToWidth(options []Option) []Option {
	if len(options) == 0 {
		return nil
	}
	maxLineWidth := terminalWidth() - 10
	if maxLineWidth < 24 {
		maxLineWidth = 24
	}
	labelWidth := maxLineWidth
	if labelWidth > 36 {
		labelWidth = 36
	}
	if labelWidth < 12 {
		labelWidth = 12
	}

	fitted := make([]Option, len(options))
	for i, option := range options {
		fitted[i] = option
		if option.Description == "" {
			fitted[i].Label = truncate(option.Label, maxLineWidth)
			continue
		}
		descriptionWidth := maxLineWidth - labelWidth - 1
		if descriptionWidth < 8 {
			descriptionWidth = 8
			labelWidth = maxLineWidth - descriptionWidth - 1
		}
		fitted[i].Label = truncate(option.Label, labelWidth)
		fitted[i].Description = truncate(option.Description, descriptionWidth)
	}
	return fitted
}

func normalizePromptError(err error) error {
	if err == nil {
		return nil
	}
	if err == promptui.ErrInterrupt || err == promptui.ErrEOF {
		return promptui.ErrAbort
	}
	return err
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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

func truncate(value string, width int) string {
	if width <= 0 || len([]rune(value)) <= width {
		return value
	}
	if width <= 3 {
		return string([]rune(value)[:width])
	}
	runes := []rune(value)
	return string(runes[:width-3]) + "..."
}
