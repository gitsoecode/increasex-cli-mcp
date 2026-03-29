package cli

import "testing"

func TestAuthExportSafetyMessaging(t *testing.T) {
	if got := authExportWarningText(); got != "Warning: auth export prints your raw API key to stdout." {
		t.Fatalf("authExportWarningText() = %q", got)
	}
	if got := authExportConfirmationPrompt(); got != "Export raw API credentials to stdout?" {
		t.Fatalf("authExportConfirmationPrompt() = %q", got)
	}
}

func TestAuthExportCommandHasConfirmFlag(t *testing.T) {
	cmd := newAuthExportCmd(&Context{Options: &RootOptions{}})
	flag := cmd.Flags().Lookup("confirm")
	if flag == nil {
		t.Fatal("auth export command should expose --confirm")
	}
	if flag.DefValue != "false" {
		t.Fatalf("--confirm default = %q, want false", flag.DefValue)
	}
}
