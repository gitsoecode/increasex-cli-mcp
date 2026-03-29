package mcp

import (
	"bufio"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/jessevaughan/increasex/internal/app"
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
		false,
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
		false,
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
