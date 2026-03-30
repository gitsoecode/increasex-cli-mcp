package mcp

import (
	"bufio"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/gitsoecode/increasex-cli-mcp/internal/app"
	"github.com/gitsoecode/increasex-cli-mcp/internal/util"
)

func TestReadFrameAcceptsCanonicalContentLengthHeader(t *testing.T) {
	server := &Server{
		reader: bufio.NewReader(strings.NewReader("Content-Length: 17\r\n\r\n{\"jsonrpc\":\"2.0\"}")),
	}

	frame, err := server.readFrame()
	if err != nil {
		t.Fatalf("readFrame() error = %v", err)
	}

	if got, want := string(frame), "{\"jsonrpc\":\"2.0\"}"; got != want {
		t.Fatalf("readFrame() = %q, want %q", got, want)
	}
}

func TestReadFrameAcceptsLowercaseContentLengthHeader(t *testing.T) {
	server := &Server{
		reader: bufio.NewReader(strings.NewReader("content-length: 17\r\n\r\n{\"jsonrpc\":\"2.0\"}")),
	}

	frame, err := server.readFrame()
	if err != nil {
		t.Fatalf("readFrame() error = %v", err)
	}

	if got, want := string(frame), "{\"jsonrpc\":\"2.0\"}"; got != want {
		t.Fatalf("readFrame() = %q, want %q", got, want)
	}
}

func TestInitializeNegotiatesProtocolVersion(t *testing.T) {
	server := &Server{}
	params, err := json.Marshal(initializeParams{ProtocolVersion: "2025-06-18"})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	resp := server.handle(t.Context(), request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params:  params,
	})
	if resp == nil {
		t.Fatal("handle() = nil, want initialize response")
	}

	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatalf("Result type = %T, want map[string]any", resp.Result)
	}
	if got, want := result["protocolVersion"], "2025-06-18"; got != want {
		t.Fatalf("protocolVersion = %v, want %q", got, want)
	}

	capabilities, ok := result["capabilities"].(map[string]any)
	if !ok {
		t.Fatalf("capabilities type = %T, want map[string]any", result["capabilities"])
	}
	tools, ok := capabilities["tools"].(map[string]any)
	if !ok {
		t.Fatalf("tools capability type = %T, want map[string]any", capabilities["tools"])
	}
	if got, want := tools["listChanged"], false; got != want {
		t.Fatalf("tools.listChanged = %v, want %v", got, want)
	}
}

func TestReadFrameSkipsLeadingBlankLines(t *testing.T) {
	server := &Server{
		reader: bufio.NewReader(strings.NewReader("\r\n\r\nContent-Length: 17\r\n\r\n{\"jsonrpc\":\"2.0\"}")),
	}

	frame, err := server.readFrame()
	if err != nil {
		t.Fatalf("readFrame() error = %v", err)
	}

	if got, want := string(frame), "{\"jsonrpc\":\"2.0\"}"; got != want {
		t.Fatalf("readFrame() = %q, want %q", got, want)
	}
}

func TestReadFrameRejectsMissingContentLength(t *testing.T) {
	server := &Server{
		reader: bufio.NewReader(strings.NewReader("Content-Type: application/json\r\n\r\n{}")),
	}

	_, err := server.readFrame()
	if err == nil {
		t.Fatal("readFrame() error = nil, want missing Content-Length error")
	}
	if got, want := err.Error(), "missing Content-Length header"; got != want {
		t.Fatalf("readFrame() error = %q, want %q", got, want)
	}
}

func TestUnknownNotificationDoesNotEmitResponse(t *testing.T) {
	server := &Server{}

	resp := server.handle(t.Context(), request{
		JSONRPC: "2.0",
		Method:  "notifications/cancelled",
	})
	if resp != nil {
		t.Fatalf("handle() = %#v, want nil for notification", resp)
	}
}

func TestReadFrameAcceptsJSONLineTransport(t *testing.T) {
	server := &Server{
		reader: bufio.NewReader(strings.NewReader("{\"jsonrpc\":\"2.0\",\"id\":0}\n")),
	}

	frame, err := server.readFrame()
	if err != nil {
		t.Fatalf("readFrame() error = %v", err)
	}

	if got, want := string(frame), "{\"jsonrpc\":\"2.0\",\"id\":0}"; got != want {
		t.Fatalf("readFrame() = %q, want %q", got, want)
	}
	if got, want := server.mode, transportModeJSONLine; got != want {
		t.Fatalf("server.mode = %q, want %q", got, want)
	}
}

func TestWriteFrameUsesJSONLineWhenDetected(t *testing.T) {
	var out strings.Builder
	server := &Server{
		writer: &out,
		mode:   transportModeJSONLine,
	}

	if err := server.writeFrame(response{JSONRPC: "2.0", ID: 1, Result: map[string]any{"ok": true}}); err != nil {
		t.Fatalf("writeFrame() error = %v", err)
	}

	if got := out.String(); !strings.HasSuffix(got, "\n") {
		t.Fatalf("writeFrame() = %q, want newline-delimited JSON", got)
	}
	if got := strings.TrimSpace(out.String()); !strings.Contains(got, "\"jsonrpc\":\"2.0\"") {
		t.Fatalf("writeFrame() = %q, want JSON response body", got)
	}
}

func TestReadJSONLineFrameHandlesEOFWithoutTrailingNewline(t *testing.T) {
	server := &Server{
		reader: bufio.NewReader(strings.NewReader("{\"jsonrpc\":\"2.0\"}")),
		mode:   transportModeJSONLine,
	}

	frame, err := server.readFrame()
	if err != nil {
		t.Fatalf("readFrame() error = %v", err)
	}

	if got, want := string(frame), "{\"jsonrpc\":\"2.0\"}"; got != want {
		t.Fatalf("readFrame() = %q, want %q", got, want)
	}

	_, err = server.readFrame()
	if err != io.EOF {
		t.Fatalf("second readFrame() error = %v, want EOF", err)
	}
}

func TestHandleWritePreviewsWhenConfirmationTokenIsMissing(t *testing.T) {
	server := &Server{}
	previewCalled := false
	executeCalled := false

	result, isErr := server.handleWrite(
		t.Context(),
		app.Session{},
		nil,
		boolPtr(false),
		"",
		func() (*app.PreviewResult, error) {
			previewCalled = true
			return &app.PreviewResult{Mode: "preview", Summary: "Preview only"}, nil
		},
		func() (any, string, error) {
			executeCalled = true
			return map[string]any{"mode": "executed"}, "req_123", nil
		},
	)

	if isErr {
		t.Fatalf("handleWrite() isErr = true, want false")
	}
	if !previewCalled {
		t.Fatal("handleWrite() did not call preview when confirmation token was missing")
	}
	if executeCalled {
		t.Fatal("handleWrite() executed despite missing confirmation token")
	}
	text := toJSONString(result)
	if !strings.Contains(text, "\"mode\":\"preview\"") {
		t.Fatalf("handleWrite() = %s, want preview payload", text)
	}
}

func TestHandleWriteExecutesWithConfirmationToken(t *testing.T) {
	server := &Server{}
	previewCalled := false
	executeCalled := false

	result, isErr := server.handleWrite(
		t.Context(),
		app.Session{},
		nil,
		boolPtr(false),
		"token_123",
		func() (*app.PreviewResult, error) {
			previewCalled = true
			return &app.PreviewResult{Mode: "preview"}, nil
		},
		func() (any, string, error) {
			executeCalled = true
			return map[string]any{"mode": "executed"}, "req_123", nil
		},
	)

	if isErr {
		t.Fatalf("handleWrite() isErr = true, want false")
	}
	if previewCalled {
		t.Fatal("handleWrite() previewed despite having dry_run=false and confirmation token")
	}
	if !executeCalled {
		t.Fatal("handleWrite() did not execute when confirmation token was present")
	}
	text := toJSONString(result)
	if !strings.Contains(text, "\"mode\":\"executed\"") {
		t.Fatalf("handleWrite() = %s, want executed payload", text)
	}
}

func TestHandleWriteReturnsValidationErrorWhenTokenOmitsDryRun(t *testing.T) {
	server := &Server{}
	previewCalled := false
	executeCalled := false

	result, isErr := server.handleWrite(
		t.Context(),
		app.Session{},
		nil,
		nil,
		"token_123",
		func() (*app.PreviewResult, error) {
			previewCalled = true
			return &app.PreviewResult{Mode: "preview"}, nil
		},
		func() (any, string, error) {
			executeCalled = true
			return map[string]any{"mode": "executed"}, "req_123", nil
		},
	)

	if !isErr {
		t.Fatal("handleWrite() isErr = false, want true")
	}
	if previewCalled || executeCalled {
		t.Fatalf("handleWrite() called preview=%v execute=%v, want neither", previewCalled, executeCalled)
	}
	detail := unwrapErrorDetail(t, result)
	if detail.Code != util.CodeValidationError {
		t.Fatalf("error code = %q, want %q", detail.Code, util.CodeValidationError)
	}
	if detail.Message == "" || !strings.Contains(detail.Message, "dry_run=false") {
		t.Fatalf("error message = %q, want dry_run=false guidance", detail.Message)
	}
	if len(detail.Fields) != 1 || detail.Fields[0].Field != "dry_run" {
		t.Fatalf("error fields = %#v, want dry_run field error", detail.Fields)
	}
}

func TestHandleWriteReturnsValidationErrorWhenTokenUsesDryRunTrue(t *testing.T) {
	server := &Server{}
	previewCalled := false
	executeCalled := false

	result, isErr := server.handleWrite(
		t.Context(),
		app.Session{},
		nil,
		boolPtr(true),
		"token_123",
		func() (*app.PreviewResult, error) {
			previewCalled = true
			return &app.PreviewResult{Mode: "preview"}, nil
		},
		func() (any, string, error) {
			executeCalled = true
			return map[string]any{"mode": "executed"}, "req_123", nil
		},
	)

	if !isErr {
		t.Fatal("handleWrite() isErr = false, want true")
	}
	if previewCalled || executeCalled {
		t.Fatalf("handleWrite() called preview=%v execute=%v, want neither", previewCalled, executeCalled)
	}
	detail := unwrapErrorDetail(t, result)
	if detail.Code != util.CodeValidationError {
		t.Fatalf("error code = %q, want %q", detail.Code, util.CodeValidationError)
	}
	if len(detail.Fields) != 1 || detail.Fields[0].Field != "dry_run" {
		t.Fatalf("error fields = %#v, want dry_run field error", detail.Fields)
	}
}

func TestToolsExposeNewParitySurface(t *testing.T) {
	server := &Server{}
	tools := server.tools()
	names := make([]string, 0, len(tools))
	for _, tool := range tools {
		names = append(names, tool.Name)
	}
	got := strings.Join(names, ",")

	expected := []string{
		"describe_capabilities",
		"list_account_numbers",
		"retrieve_account_number",
		"retrieve_account_number_sensitive_details",
		"disable_account_number",
		"list_programs",
		"retrieve_program",
		"list_digital_card_profiles",
		"retrieve_digital_card_profile",
		"list_events",
		"retrieve_event",
		"list_documents",
		"retrieve_document",
		"list_external_accounts",
		"retrieve_external_account",
		"create_external_account",
		"update_external_account",
		"list_transfers",
		"retrieve_transfer",
		"list_transfer_queue",
		"approve_transfer",
		"cancel_transfer",
		"create_account_transfer",
		"create_real_time_payments_transfer",
		"create_card_details_iframe",
		"update_card_pin",
	}
	for _, name := range expected {
		if !strings.Contains(got, name) {
			t.Fatalf("tools() missing %q in %q", name, got)
		}
	}
	if !strings.HasPrefix(got, "describe_capabilities,") {
		t.Fatalf("tools() should start with describe_capabilities, got %q", got)
	}
}

func TestTransferCreateToolsDescribeQueueMode(t *testing.T) {
	server := &Server{}
	tools := server.tools()

	descriptions := map[string]string{}
	for _, tool := range tools {
		descriptions[tool.Name] = tool.Description
	}

	for _, name := range []string{
		"create_account",
		"close_account",
		"create_account_number",
		"disable_account_number",
		"create_account_transfer",
		"create_ach_transfer",
		"create_real_time_payments_transfer",
		"create_fednow_transfer",
		"create_wire_transfer",
		"approve_transfer",
		"cancel_transfer",
		"create_external_account",
		"update_external_account",
		"create_card",
		"update_card_pin",
	} {
		description := descriptions[name]
		if strings.HasPrefix(name, "create_") && strings.Contains(name, "transfer") && !strings.Contains(description, "require_approval=true") {
			t.Fatalf("%s description = %q, want require_approval guidance", name, description)
		}
		if strings.HasPrefix(name, "create_") && strings.Contains(name, "transfer") && !strings.Contains(description, "queue for approval") {
			t.Fatalf("%s description = %q, want queue-for-approval wording", name, description)
		}
		if !strings.Contains(description, "dry_run=false") {
			t.Fatalf("%s description = %q, want dry_run=false execution guidance", name, description)
		}
	}
}

func TestDescribeCapabilitiesIncludesWritePatternAndTransferRailAliases(t *testing.T) {
	result := describeCapabilities()

	writePattern, ok := result["write_pattern"].(map[string]any)
	if !ok {
		t.Fatalf("write_pattern type = %T, want map[string]any", result["write_pattern"])
	}
	execute, ok := writePattern["execute"].(string)
	if !ok || !strings.Contains(execute, "dry_run=false") {
		t.Fatalf("write_pattern.execute = %#v, want dry_run=false guidance", writePattern["execute"])
	}

	transferRails, ok := result["transfer_rails"].(map[string]any)
	if !ok {
		t.Fatalf("transfer_rails type = %T, want map[string]any", result["transfer_rails"])
	}
	aliases, ok := transferRails["aliases"].(map[string]string)
	if !ok {
		t.Fatalf("transfer_rails.aliases type = %T, want map[string]string", transferRails["aliases"])
	}
	if aliases["internal"] != "account" || aliases["rtp"] != "real_time_payments" {
		t.Fatalf("transfer rail aliases = %#v, want internal/account and rtp/real_time_payments", aliases)
	}
}

func TestListRecentTransactionsToolSupportsUntil(t *testing.T) {
	server := &Server{}

	for _, tool := range server.tools() {
		if tool.Name != "list_recent_transactions" {
			continue
		}
		properties, ok := tool.InputSchema["properties"].(map[string]any)
		if !ok {
			t.Fatalf("InputSchema.properties type = %T, want map[string]any", tool.InputSchema["properties"])
		}
		if _, ok := properties["until"]; !ok {
			t.Fatal("list_recent_transactions input schema missing until")
		}
		return
	}

	t.Fatal("list_recent_transactions tool definition not found")
}

func TestNewReadToolsExposeExpectedFilters(t *testing.T) {
	server := &Server{}

	for _, tool := range server.tools() {
		switch tool.Name {
		case "list_digital_card_profiles":
			properties, ok := tool.InputSchema["properties"].(map[string]any)
			if !ok {
				t.Fatalf("list_digital_card_profiles properties type = %T, want map[string]any", tool.InputSchema["properties"])
			}
			if _, ok := properties["idempotency_key"]; !ok {
				t.Fatal("list_digital_card_profiles input schema missing idempotency_key")
			}
		case "list_events":
			properties, ok := tool.InputSchema["properties"].(map[string]any)
			if !ok {
				t.Fatalf("list_events properties type = %T, want map[string]any", tool.InputSchema["properties"])
			}
			if _, ok := properties["categories"]; !ok {
				t.Fatal("list_events input schema missing categories")
			}
			if _, ok := properties["until"]; !ok {
				t.Fatal("list_events input schema missing until")
			}
		case "list_documents":
			properties, ok := tool.InputSchema["properties"].(map[string]any)
			if !ok {
				t.Fatalf("list_documents properties type = %T, want map[string]any", tool.InputSchema["properties"])
			}
			if _, ok := properties["categories"]; !ok {
				t.Fatal("list_documents input schema missing categories")
			}
			if _, ok := properties["idempotency_key"]; !ok {
				t.Fatal("list_documents input schema missing idempotency_key")
			}
		}
	}
}

func boolPtr(value bool) *bool {
	return &value
}

func unwrapErrorDetail(t *testing.T, result any) *util.ErrorDetail {
	t.Helper()
	envelope, ok := result.(util.Envelope)
	if !ok {
		t.Fatalf("result type = %T, want util.Envelope", result)
	}
	if envelope.Error == nil {
		t.Fatal("envelope.Error = nil, want populated error detail")
	}
	return envelope.Error
}

func TestRetrieveTransferToolSchemaSupportsEventDrivenLookup(t *testing.T) {
	server := &Server{}

	for _, tool := range server.tools() {
		if tool.Name != "retrieve_transfer" {
			continue
		}

		properties, ok := tool.InputSchema["properties"].(map[string]any)
		if !ok {
			t.Fatalf("retrieve_transfer properties type = %T, want map[string]any", tool.InputSchema["properties"])
		}
		if _, ok := properties["event_id"]; !ok {
			t.Fatal("retrieve_transfer input schema missing event_id")
		}
		if _, ok := tool.InputSchema["required"]; ok {
			t.Fatal("retrieve_transfer should not require rail and transfer_id when event-driven lookup is supported")
		}
		if !strings.Contains(tool.Description, "event_id") {
			t.Fatalf("retrieve_transfer description = %q, want event-driven lookup guidance", tool.Description)
		}
		return
	}

	t.Fatal("retrieve_transfer tool definition not found")
}

func TestExternalTransferToolSchemasUseCorrectSourceIdentifiers(t *testing.T) {
	server := &Server{}
	toolsByName := map[string]toolDefinition{}
	for _, tool := range server.tools() {
		toolsByName[tool.Name] = tool
	}

	assertHasProperty := func(t *testing.T, toolName, property string) {
		t.Helper()
		properties, ok := toolsByName[toolName].InputSchema["properties"].(map[string]any)
		if !ok {
			t.Fatalf("%s properties type = %T, want map[string]any", toolName, toolsByName[toolName].InputSchema["properties"])
		}
		if _, ok := properties[property]; !ok {
			t.Fatalf("%s missing property %q", toolName, property)
		}
	}

	assertMissingProperty := func(t *testing.T, toolName, property string) {
		t.Helper()
		properties, ok := toolsByName[toolName].InputSchema["properties"].(map[string]any)
		if !ok {
			t.Fatalf("%s properties type = %T, want map[string]any", toolName, toolsByName[toolName].InputSchema["properties"])
		}
		if _, ok := properties[property]; ok {
			t.Fatalf("%s should not expose property %q", toolName, property)
		}
	}

	assertHasProperty(t, "create_ach_transfer", "account_id")
	assertMissingProperty(t, "create_ach_transfer", "source_account_number_id")

	for _, toolName := range []string{
		"create_real_time_payments_transfer",
		"create_fednow_transfer",
		"create_wire_transfer",
	} {
		assertHasProperty(t, toolName, "source_account_number_id")
	}
	assertHasProperty(t, "create_fednow_transfer", "account_id")
	assertHasProperty(t, "create_wire_transfer", "account_id")
}

func TestEnvelopePreservesStructuredFieldErrors(t *testing.T) {
	result, isErr := envelope(nil, "req_123", &util.ErrorDetail{
		Code:    util.CodeValidationError,
		Message: "Please correct the highlighted external account fields.",
		Fields: []util.FieldError{
			{Field: "account_holder", Message: "expected one of business, individual, or unknown"},
		},
	})

	if !isErr {
		t.Fatal("envelope() isErr = false, want true")
	}
	text := toJSONString(result)
	if !strings.Contains(text, "\"fields\"") {
		t.Fatalf("envelope() = %s, want structured field errors", text)
	}
	if !strings.Contains(text, "\"account_holder\"") {
		t.Fatalf("envelope() = %s, want account_holder field detail", text)
	}
}
