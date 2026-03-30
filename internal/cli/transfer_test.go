package cli

import "testing"

func TestNewTransferRetrieveCmdExposesEventIDFlag(t *testing.T) {
	cmd := newTransferRetrieveCmd(&Context{})
	if flag := cmd.Flags().Lookup("event-id"); flag == nil {
		t.Fatal("newTransferRetrieveCmd() missing event-id flag")
	}
}

func TestShouldPromptTransferRetrieveRail(t *testing.T) {
	cases := []struct {
		name       string
		transferID string
		eventID    string
		want       bool
	}{
		{name: "event id skips prompt", eventID: "event_123", want: false},
		{name: "inferable transfer id skips prompt", transferID: "account_transfer_123", want: false},
		{name: "unknown transfer id prompts", transferID: "transfer_123", want: true},
		{name: "empty input prompts", want: true},
	}

	for _, tc := range cases {
		if got := shouldPromptTransferRetrieveRail(tc.transferID, tc.eventID); got != tc.want {
			t.Fatalf("%s: shouldPromptTransferRetrieveRail(%q, %q) = %v, want %v", tc.name, tc.transferID, tc.eventID, got, tc.want)
		}
	}
}
