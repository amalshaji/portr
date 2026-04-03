package main

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	clientdb "github.com/amalshaji/portr/internal/client/db"
	requestlogs "github.com/amalshaji/portr/internal/client/logs"
	clientreplay "github.com/amalshaji/portr/internal/client/replay"
)

func TestParseReplayCommandArgsSupportsExactIDAndEdits(t *testing.T) {
	opts, err := parseReplayCommandArgs([]string{
		"req-123",
		"--method", "PATCH",
		"--path", "/edited",
		"--header", "Content-Type: application/json",
		"--drop-header", "Authorization",
		"--body=", "--json",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if opts.RequestID != "req-123" {
		t.Fatalf("expected request id req-123, got %q", opts.RequestID)
	}
	if opts.UseLatest {
		t.Fatal("expected exact id mode")
	}
	if opts.Edit.Method != "PATCH" {
		t.Fatalf("expected PATCH, got %q", opts.Edit.Method)
	}
	if opts.Edit.Path != "/edited" {
		t.Fatalf("expected /edited, got %q", opts.Edit.Path)
	}
	if opts.Edit.Headers["Content-Type"] != "application/json" {
		t.Fatalf("expected content-type override, got %#v", opts.Edit.Headers)
	}
	if len(opts.Edit.DropHeaders) != 1 || opts.Edit.DropHeaders[0] != "Authorization" {
		t.Fatalf("unexpected drop headers %#v", opts.Edit.DropHeaders)
	}
	if opts.Body.Kind != replayBodyInline {
		t.Fatalf("expected inline body source, got %q", opts.Body.Kind)
	}
	if opts.Body.Value != "" {
		t.Fatalf("expected empty inline body, got %q", opts.Body.Value)
	}
	if !opts.JSON {
		t.Fatal("expected json mode")
	}
}

func TestParseReplayCommandArgsSupportsLatest(t *testing.T) {
	originalLocal := time.Local
	time.Local = time.FixedZone("IST", 5*60*60+30*60)
	defer func() {
		time.Local = originalLocal
	}()

	opts, err := parseReplayCommandArgs([]string{
		"--latest",
		"--subdomain", "demo",
		"--filter", "/api",
		"--since", "2026-03-14",
		"--stdin",
		"--body-encoding", "base64",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !opts.UseLatest {
		t.Fatal("expected latest mode")
	}
	if opts.Subdomain != "demo" {
		t.Fatalf("expected subdomain demo, got %q", opts.Subdomain)
	}
	if opts.Query.Filter != "/api" {
		t.Fatalf("expected filter /api, got %q", opts.Query.Filter)
	}
	expectedSince := time.Date(2026, 3, 13, 18, 30, 0, 0, time.UTC)
	if opts.Query.Since == nil || !opts.Query.Since.Equal(expectedSince) {
		t.Fatalf("expected since %v, got %v", expectedSince, opts.Query.Since)
	}
	if opts.Body.Kind != replayBodyStdin {
		t.Fatalf("expected stdin body source, got %q", opts.Body.Kind)
	}
	if opts.Body.Encoding != "base64" {
		t.Fatalf("expected base64 body encoding, got %q", opts.Body.Encoding)
	}
}

func TestParseReplayCommandArgsRejectsConflicts(t *testing.T) {
	testCases := [][]string{
		{"req-1", "--latest"},
		{"req-1", "--subdomain", "demo"},
		{"--latest", "--subdomain", "demo", "--body", "a", "--stdin"},
		{"--latest"},
		{"req-1", "--body-encoding", "base64"},
	}

	for _, args := range testCases {
		if _, err := parseReplayCommandArgs(args); err == nil {
			t.Fatalf("expected error for args %#v, got nil", args)
		}
	}
}

func TestReplayWantsJSONAndHelp(t *testing.T) {
	if !replayWantsJSON([]string{"req-1", "--json"}) {
		t.Fatal("expected --json to be detected")
	}
	if replayWantsJSON([]string{"req-1", "--", "--json"}) {
		t.Fatal("expected --json after -- to be positional")
	}
	if !replayWantsHelp([]string{"--help"}) {
		t.Fatal("expected --help to be detected")
	}
	if replayWantsHelp([]string{"req-1", "--", "--help"}) {
		t.Fatal("expected --help after -- to be positional")
	}
}

func TestShouldSuppressUpdateNoticeForReplay(t *testing.T) {
	if !shouldSuppressUpdateNotice([]string{"portr", "replay", "req-1", "--json"}) {
		t.Fatal("expected replay json to suppress update notice")
	}
	if !shouldSuppressUpdateNotice([]string{"portr", "replay", "--help"}) {
		t.Fatal("expected replay help to suppress update notice")
	}
	if shouldSuppressUpdateNotice([]string{"portr", "replay", "req-1"}) {
		t.Fatal("expected plain replay not to suppress update notice")
	}
}

func TestRenderReplayJSON(t *testing.T) {
	var output bytes.Buffer

	request := &clientdb.Request{
		ID:        "req-1",
		Subdomain: "demo",
		Localport: 3000,
		Host:      "demo.example.com",
		Url:       "/submit",
		Method:    "POST",
	}

	result := &clientreplay.Result{
		OriginalRequest:  request,
		EffectiveMethod:  "PATCH",
		EffectivePath:    "/edited",
		EffectiveURL:     "https://demo.example.com/edited",
		EffectiveHeaders: map[string]string{"Content-Type": "application/json"},
		EffectiveBody:    []byte(`{"ok":true}`),
		ResponseStatus:   201,
		ResponseHeaders:  map[string][]string{"Content-Type": {"text/plain; charset=utf-8"}},
		ResponseBody:     []byte("done"),
	}

	if err := renderReplayJSON(&output, replayCommandOptions{RequestID: "req-1"}, request, result, nil); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var decoded replayJSONOutput
	if err := json.Unmarshal(output.Bytes(), &decoded); err != nil {
		t.Fatalf("expected valid json, got %v", err)
	}

	if !decoded.OK {
		t.Fatalf("expected ok output, got %#v", decoded)
	}
	if decoded.RequestID != "req-1" {
		t.Fatalf("expected request id req-1, got %q", decoded.RequestID)
	}
	if decoded.EffectiveBody != "eyJvayI6dHJ1ZX0=" {
		t.Fatalf("expected effective body to be base64 encoded, got %q", decoded.EffectiveBody)
	}
	if decoded.ResponseBody != "ZG9uZQ==" {
		t.Fatalf("expected response body to be base64 encoded, got %q", decoded.ResponseBody)
	}
	if decoded.EffectiveBodyText == nil || *decoded.EffectiveBodyText != `{"ok":true}` {
		t.Fatalf("expected effective body text, got %#v", decoded.EffectiveBodyText)
	}
	if decoded.ResponseBodyText == nil || *decoded.ResponseBodyText != "done" {
		t.Fatalf("expected response body text, got %#v", decoded.ResponseBodyText)
	}

	output.Reset()

	failure := &clientreplay.Failure{
		StatusCode: 503,
		Reason:     "connection-lost",
		Message:    "The tunnel connection was lost. Please try again in a bit.",
	}
	if err := renderReplayJSON(&output, replayCommandOptions{
		UseLatest: true,
		Subdomain: "demo",
		Query: requestlogs.QueryOptions{
			Filter: "/api",
		},
	}, request, result, failure); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if err := json.Unmarshal(output.Bytes(), &decoded); err != nil {
		t.Fatalf("expected valid json, got %v", err)
	}
	if decoded.OK {
		t.Fatalf("expected failure output, got %#v", decoded)
	}
	if decoded.Error == nil || decoded.Error.Reason != "connection-lost" {
		t.Fatalf("expected replay failure details, got %#v", decoded.Error)
	}
}
