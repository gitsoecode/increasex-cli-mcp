package ui

import (
	"errors"
	"os"
	"strings"

	"github.com/manifoldco/promptui"
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
		Items:             options,
		Templates:         templates,
		Size:              min(12, len(options)),
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
