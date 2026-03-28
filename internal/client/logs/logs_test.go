package logs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	clientdb "github.com/amalshaji/portr/internal/client/db"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestStoreListAggregatesAcrossPortsAndOrdersNewestFirst(t *testing.T) {
	store := openTestStore(t, []clientdb.Request{
		{
			ID:                 "req-1",
			Subdomain:          "demo",
			Localport:          3000,
			Url:                "/alpha",
			Method:             "GET",
			ResponseStatusCode: 200,
			LoggedAt:           testTime(2026, 3, 14, 10, 0, 0),
		},
		{
			ID:                 "req-2",
			Subdomain:          "demo",
			Localport:          3001,
			Url:                "/beta",
			Method:             "POST",
			ResponseStatusCode: 201,
			LoggedAt:           testTime(2026, 3, 14, 11, 0, 0),
		},
		{
			ID:                 "req-3",
			Subdomain:          "demo",
			Localport:          3000,
			Url:                "/gamma",
			Method:             "DELETE",
			ResponseStatusCode: 500,
			LoggedAt:           testTime(2026, 3, 14, 12, 0, 0),
		},
		{
			ID:                 "req-4",
			Subdomain:          "other",
			Localport:          4000,
			Url:                "/ignored",
			Method:             "GET",
			ResponseStatusCode: 200,
			LoggedAt:           testTime(2026, 3, 14, 13, 0, 0),
		},
	})
	defer store.Close()

	requests, err := store.List("demo", QueryOptions{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(requests) != 3 {
		t.Fatalf("expected 3 requests, got %d", len(requests))
	}

	if requests[0].ID != "req-3" || requests[1].ID != "req-2" || requests[2].ID != "req-1" {
		t.Fatalf("unexpected order: %#v", []string{requests[0].ID, requests[1].ID, requests[2].ID})
	}

	if requests[0].Localport != 3000 || requests[1].Localport != 3001 || requests[2].Localport != 3000 {
		t.Fatalf("unexpected ports: %#v", []int{requests[0].Localport, requests[1].Localport, requests[2].Localport})
	}
}

func TestStoreListUsesDefaultCount(t *testing.T) {
	var seeded []clientdb.Request
	base := testTime(2026, 3, 14, 0, 0, 0)
	for i := 0; i < 25; i++ {
		seeded = append(seeded, clientdb.Request{
			ID:                 fmt.Sprintf("req-%d", i),
			Subdomain:          "demo",
			Localport:          3000 + i%2,
			Url:                "/entry",
			Method:             "GET",
			ResponseStatusCode: 200,
			LoggedAt:           base.Add(time.Duration(i) * time.Minute),
		})
	}

	store := openTestStore(t, seeded)
	defer store.Close()

	requests, err := store.List("demo", QueryOptions{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(requests) != DefaultCount {
		t.Fatalf("expected %d requests, got %d", DefaultCount, len(requests))
	}

	if requests[0].LoggedAt.Before(requests[len(requests)-1].LoggedAt) {
		t.Fatalf("expected descending order, got %#v then %#v", requests[0].LoggedAt, requests[len(requests)-1].LoggedAt)
	}
}

func TestStoreListAppliesCountSinceAndFilter(t *testing.T) {
	store := openTestStore(t, []clientdb.Request{
		{
			ID:                 "req-1",
			Subdomain:          "demo",
			Localport:          3000,
			Url:                "/alpha",
			Method:             "GET",
			ResponseStatusCode: 200,
			LoggedAt:           testTime(2026, 3, 14, 9, 0, 0),
		},
		{
			ID:                 "req-2",
			Subdomain:          "demo",
			Localport:          3001,
			Url:                "/beta",
			Method:             "POST",
			ResponseStatusCode: 201,
			LoggedAt:           testTime(2026, 3, 14, 10, 0, 0),
		},
		{
			ID:                 "req-3",
			Subdomain:          "demo",
			Localport:          3002,
			Url:                "/beta/details",
			Method:             "PUT",
			ResponseStatusCode: 202,
			LoggedAt:           testTime(2026, 3, 14, 11, 0, 0),
		},
	})
	defer store.Close()

	since := testTime(2026, 3, 14, 9, 30, 0)
	requests, err := store.List("demo", QueryOptions{
		Count:  1,
		Since:  &since,
		Filter: "/BETA",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(requests))
	}

	if requests[0].ID != "req-3" {
		t.Fatalf("expected req-3, got %s", requests[0].ID)
	}
}

func TestStoreListRejectsInvalidCount(t *testing.T) {
	store := openTestStore(t, []clientdb.Request{
		{
			ID:                 "req-1",
			Subdomain:          "demo",
			Localport:          3000,
			Url:                "/alpha",
			Method:             "GET",
			ResponseStatusCode: 200,
			LoggedAt:           testTime(2026, 3, 14, 10, 0, 0),
		},
	})
	defer store.Close()

	_, err := store.List("demo", QueryOptions{Count: -1})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestStoreListReturnsEmptySliceWhenNoMatches(t *testing.T) {
	store := openTestStore(t, []clientdb.Request{
		{
			ID:                 "req-1",
			Subdomain:          "demo",
			Localport:          3000,
			Url:                "/alpha",
			Method:             "GET",
			ResponseStatusCode: 200,
			LoggedAt:           testTime(2026, 3, 14, 10, 0, 0),
		},
	})
	defer store.Close()

	requests, err := store.List("missing", QueryOptions{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if requests == nil {
		t.Fatal("expected an empty slice, got nil")
	}

	if len(requests) != 0 {
		t.Fatalf("expected 0 requests, got %d", len(requests))
	}
}

func TestOpenMissingDB(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.sqlite")

	_, err := Open(path)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestParseSince(t *testing.T) {
	rfc3339, err := ParseSince("2026-03-14T12:30:00+05:30")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expectedRFC3339 := testTime(2026, 3, 14, 7, 0, 0)
	if rfc3339 == nil || !rfc3339.Equal(expectedRFC3339) {
		t.Fatalf("expected %v, got %v", expectedRFC3339, rfc3339)
	}

	originalLocal := time.Local
	time.Local = time.FixedZone("IST", 5*60*60+30*60)
	defer func() {
		time.Local = originalLocal
	}()

	dateOnly, err := ParseSince("2026-03-14")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expectedDateOnly := testTime(2026, 3, 13, 18, 30, 0)
	if dateOnly == nil || !dateOnly.Equal(expectedDateOnly) {
		t.Fatalf("expected %v, got %v", expectedDateOnly, dateOnly)
	}

	_, err = ParseSince("March 14")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestParseCommandArgsSupportsFlagsAfterSubdomain(t *testing.T) {
	originalLocal := time.Local
	time.Local = time.FixedZone("IST", 5*60*60+30*60)
	defer func() {
		time.Local = originalLocal
	}()

	opts, err := ParseCommandArgs([]string{"amal-test", "--json", "--count", "50", "--since", "2026-03-14"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if opts.Subdomain != "amal-test" {
		t.Fatalf("expected subdomain amal-test, got %q", opts.Subdomain)
	}

	if !opts.JSON {
		t.Fatal("expected json mode to be enabled")
	}

	if opts.Query.Count != 50 {
		t.Fatalf("expected count 50, got %d", opts.Query.Count)
	}

	expectedSince := testTime(2026, 3, 13, 18, 30, 0)
	if opts.Query.Since == nil || !opts.Query.Since.Equal(expectedSince) {
		t.Fatalf("expected since %v, got %v", expectedSince, opts.Query.Since)
	}
}

func TestParseCommandArgsSupportsFilterAndLeadingFlags(t *testing.T) {
	opts, err := ParseCommandArgs([]string{"--json", "--count=5", "amal-test", "/cmd/"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !opts.JSON {
		t.Fatal("expected json mode to be enabled")
	}

	if opts.Query.Count != 5 {
		t.Fatalf("expected count 5, got %d", opts.Query.Count)
	}

	if opts.Subdomain != "amal-test" {
		t.Fatalf("expected subdomain amal-test, got %q", opts.Subdomain)
	}

	if opts.Query.Filter != "/cmd/" {
		t.Fatalf("expected filter /cmd/, got %q", opts.Query.Filter)
	}
}

func TestParseCommandArgsRejectsUnknownFlag(t *testing.T) {
	_, err := ParseCommandArgs([]string{"amal-test", "--wat"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestParseCommandArgsRejectsNonPositiveCount(t *testing.T) {
	testCases := [][]string{
		{"amal-test", "--count", "0"},
		{"amal-test", "--count=-1"},
		{"amal-test", "-n=0"},
	}

	for _, args := range testCases {
		_, err := ParseCommandArgs(args)
		if err == nil {
			t.Fatalf("expected error for args %#v, got nil", args)
		}

		if err.Error() != "count must be greater than 0" {
			t.Fatalf("expected count error for args %#v, got %v", args, err)
		}
	}
}

func TestWantsHelp(t *testing.T) {
	if !WantsHelp([]string{"--help"}) {
		t.Fatal("expected --help to request help")
	}

	if !WantsHelp([]string{"amal-test", "-h"}) {
		t.Fatal("expected -h to request help")
	}

	if WantsHelp([]string{"amal-test", "--", "--help"}) {
		t.Fatal("expected --help after -- to be treated as positional")
	}

	if WantsHelp([]string{"amal-test"}) {
		t.Fatal("expected plain args not to request help")
	}
}

func TestWantsJSON(t *testing.T) {
	if !WantsJSON([]string{"amal-test", "--json"}) {
		t.Fatal("expected --json to request json output")
	}

	if WantsJSON([]string{"amal-test", "--", "--json"}) {
		t.Fatal("expected --json after -- to be treated as positional")
	}

	if WantsJSON([]string{"amal-test"}) {
		t.Fatal("expected plain args not to request json output")
	}
}

func TestJSONFlagUsageMentionsPayloads(t *testing.T) {
	if !strings.Contains(JSONFlagUsage, "BodyText") {
		t.Fatalf("expected JSONFlagUsage to mention BodyText, got %q", JSONFlagUsage)
	}
}

func TestOpenAppliesSQLitePragmas(t *testing.T) {
	store := openTestStore(t, nil)
	defer store.Close()

	sqlDB, err := store.conn.DB()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var busyTimeout int
	if err := sqlDB.QueryRow("PRAGMA busy_timeout;").Scan(&busyTimeout); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if busyTimeout != 5000 {
		t.Fatalf("expected busy_timeout 5000, got %d", busyTimeout)
	}
}

func TestRenderText(t *testing.T) {
	var output bytes.Buffer

	err := RenderText(&output, []clientdb.Request{
		{
			ID:                 "req-1",
			Subdomain:          "demo",
			Localport:          3000,
			Url:                "/alpha",
			Method:             "GET",
			ResponseStatusCode: 200,
			LoggedAt:           testTime(2026, 3, 14, 10, 0, 0),
		},
		{
			ID:                 "req-2",
			Subdomain:          "demo",
			Localport:          3001,
			Url:                "/beta",
			Method:             "POST",
			ResponseStatusCode: 201,
			LoggedAt:           testTime(2026, 3, 14, 11, 0, 0),
			IsReplayed:         true,
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := "" +
		"2026-03-14T10:00:00Z 3000 GET 200 /alpha\n" +
		"2026-03-14T11:00:00Z 3001 POST 201 /beta [replayed]\n"
	if output.String() != expected {
		t.Fatalf("unexpected output:\n%s", output.String())
	}

	output.Reset()

	err = RenderText(&output, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if output.String() != "no logs found\n" {
		t.Fatalf("unexpected empty output: %q", output.String())
	}
}

func TestRenderJSON(t *testing.T) {
	var output bytes.Buffer

	expected := []clientdb.Request{
		{
			ID:                 "req-1",
			Subdomain:          "demo",
			Localport:          3000,
			Url:                "/alpha",
			Method:             "GET",
			Body:               []byte("hello"),
			ResponseBody:       []byte("world"),
			ResponseStatusCode: 200,
			LoggedAt:           testTime(2026, 3, 14, 10, 0, 0),
		},
		{
			ID:                 "req-2",
			Subdomain:          "demo",
			Localport:          3001,
			Url:                "/binary",
			Method:             "POST",
			Body:               []byte{0xff, 0xfe},
			ResponseBody:       []byte{0x00, 0xff},
			ResponseStatusCode: 201,
			LoggedAt:           testTime(2026, 3, 14, 11, 0, 0),
		},
	}

	if err := RenderJSON(&output, expected); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	var decoded []struct {
		Subdomain        string
		Body             string
		ResponseBody     string
		BodyText         *string
		ResponseBodyText *string
	}
	if err := json.Unmarshal(output.Bytes(), &decoded); err != nil {
		t.Fatalf("expected valid json, got %v", err)
	}

	if len(decoded) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(decoded))
	}

	if decoded[0].Subdomain != expected[0].Subdomain {
		t.Fatalf("unexpected decoded request: %#v", decoded[0])
	}

	if decoded[0].Body != "aGVsbG8=" {
		t.Fatalf("expected request body to be base64 encoded, got %q", decoded[0].Body)
	}

	if decoded[0].ResponseBody != "d29ybGQ=" {
		t.Fatalf("expected response body to be base64 encoded, got %q", decoded[0].ResponseBody)
	}

	if decoded[0].BodyText == nil || *decoded[0].BodyText != "hello" {
		t.Fatalf("expected request body text to be decoded, got %#v", decoded[0].BodyText)
	}

	if decoded[0].ResponseBodyText == nil || *decoded[0].ResponseBodyText != "world" {
		t.Fatalf("expected response body text to be decoded, got %#v", decoded[0].ResponseBodyText)
	}

	if decoded[1].BodyText != nil {
		t.Fatalf("expected binary request body text to be omitted, got %#v", decoded[1].BodyText)
	}

	if decoded[1].ResponseBodyText != nil {
		t.Fatalf("expected binary response body text to be omitted, got %#v", decoded[1].ResponseBodyText)
	}

	output.Reset()

	if err := RenderJSON(&output, nil); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if output.String() != "[]\n" {
		t.Fatalf("unexpected empty json output: %q", output.String())
	}
}

func openTestStore(t *testing.T, requests []clientdb.Request) *Store {
	t.Helper()

	path := filepath.Join(t.TempDir(), "db.sqlite")
	conn, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if err := conn.AutoMigrate(&clientdb.Request{}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(requests) > 0 {
		if err := conn.Create(&requests).Error; err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	}

	sqlDB, err := conn.DB()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if err := sqlDB.Close(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	store, err := Open(path)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	return store
}

func testTime(year int, month time.Month, day, hour, minute, second int) time.Time {
	return time.Date(year, month, day, hour, minute, second, 0, time.UTC)
}
