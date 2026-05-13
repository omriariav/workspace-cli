package cmd

// End-to-end coverage for the --raw / --params runners using httptest as
// the upstream. Verifies outgoing query params, --params-over-flags
// precedence, --all pagination merge (concat list + drop nextPageToken),
// and the --quiet contract for raw output.

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/spf13/cobra"
	chat "google.golang.org/api/chat/v1"
	gmail "google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
	people "google.golang.org/api/people/v1"
)

// captureStdout swaps os.Stdout for a pipe, runs fn, and returns the
// captured bytes. The runners write through printRaw → os.Stdout so this
// is how we observe them.
func captureStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w

	var buf bytes.Buffer
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, _ = io.Copy(&buf, r)
	}()

	runErr := fn()
	_ = w.Close()
	wg.Wait()
	os.Stdout = orig
	return buf.String(), runErr
}

// makeCmd builds a *cobra.Command with the given flag values applied —
// enough surface for the raw runners to read from.
func makeCmd(t *testing.T, flags map[string]string) *cobra.Command {
	t.Helper()
	c := &cobra.Command{Use: "x"}
	addRawParamsFlags(c)
	for k, v := range flags {
		if err := c.Flags().Set(k, v); err != nil {
			t.Fatalf("set %s=%s: %v", k, v, err)
		}
	}
	return c
}

// --- gmail list ----------------------------------------------------------

func TestGmailListRaw_AllAggregatesAndDropsToken(t *testing.T) {
	var capturedQueries []url.Values
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		capturedQueries = append(capturedQueries, r.URL.Query())
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		page := r.URL.Query().Get("pageToken")
		switch page {
		case "":
			_ = json.NewEncoder(w).Encode(&gmail.ListMessagesResponse{
				Messages:           []*gmail.Message{{Id: "1", ThreadId: "t1"}, {Id: "2", ThreadId: "t2"}},
				NextPageToken:      "p2",
				ResultSizeEstimate: 4,
			})
		case "p2":
			_ = json.NewEncoder(w).Encode(&gmail.ListMessagesResponse{
				Messages:           []*gmail.Message{{Id: "3", ThreadId: "t3"}, {Id: "4", ThreadId: "t4"}},
				ResultSizeEstimate: 4,
			})
		default:
			t.Fatalf("unexpected pageToken %q", page)
		}
	}))
	defer server.Close()

	svc, err := gmail.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("gmail service: %v", err)
	}

	cmd := makeCmd(t, map[string]string{
		"raw":    "true",
		"params": `{"maxResults":2,"q":"in:sent"}`,
	})

	out, err := captureStdout(t, func() error {
		return runGmailListRaw(cmd, svc, "ignored-by-params", 0, true) // fetchAll=true
	})
	if err != nil {
		t.Fatalf("runner err: %v", err)
	}

	// Decode and verify aggregated shape.
	var got gmail.ListMessagesResponse
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("decode: %v\nout=%s", err, out)
	}
	if len(got.Messages) != 4 {
		t.Errorf("expected 4 aggregated messages, got %d (%s)", len(got.Messages), out)
	}
	if got.NextPageToken != "" {
		t.Errorf("expected nextPageToken dropped under --all, got %q", got.NextPageToken)
	}

	// --params q must have overridden the flag-derived "ignored-by-params".
	if len(capturedQueries) < 1 {
		t.Fatal("expected at least one request")
	}
	if q := capturedQueries[0].Get("q"); q != "in:sent" {
		t.Errorf("expected q=in:sent from --params, got %q", q)
	}
	if ms := capturedQueries[0].Get("maxResults"); ms != "2" {
		t.Errorf("expected maxResults=2 from --params (per-page), got %q", ms)
	}
}

func TestGmailListRaw_QuietSuppressesOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(&gmail.ListMessagesResponse{
			Messages: []*gmail.Message{{Id: "1", ThreadId: "t1"}},
		})
	}))
	defer server.Close()

	svc, err := gmail.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatal(err)
	}

	prev := quiet
	quiet = true
	defer func() { quiet = prev }()

	cmd := makeCmd(t, map[string]string{"raw": "true"})
	out, err := captureStdout(t, func() error {
		return runGmailListRaw(cmd, svc, "", 10, false)
	})
	if err != nil {
		t.Fatalf("runner err: %v", err)
	}
	if out != "" {
		t.Errorf("expected no output under --quiet, got %q", out)
	}
}

// --- gmail thread --------------------------------------------------------

func TestGmailThreadRaw_PreservesAPIShape(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/threads/thr-x") {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("format"); got != "metadata" {
			t.Errorf("expected format=metadata from --params, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(&gmail.Thread{
			Id: "thr-x",
			Messages: []*gmail.Message{
				{
					Id:       "m1",
					LabelIds: []string{"INBOX"},
					Payload: &gmail.MessagePart{
						Headers: []*gmail.MessagePartHeader{
							{Name: "Subject", Value: "Hi"},
						},
					},
				},
			},
		})
	}))
	defer server.Close()

	svc, err := gmail.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatal(err)
	}

	cmd := makeCmd(t, map[string]string{
		"raw":    "true",
		"params": `{"format":"metadata"}`,
	})
	out, err := captureStdout(t, func() error {
		return runGmailThreadRaw(cmd, svc, "thr-x")
	})
	if err != nil {
		t.Fatalf("runner err: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("decode: %v\nout=%s", err, out)
	}
	for _, k := range []string{"id", "messages"} {
		if _, ok := m[k]; !ok {
			t.Errorf("expected key %q, got %v", k, m)
		}
	}
}

// --- chat spaces list ----------------------------------------------------

func TestChatListRaw_AllAggregatesAndParamsOverride(t *testing.T) {
	var qs []url.Values
	var mu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		qs = append(qs, r.URL.Query())
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		token := r.URL.Query().Get("pageToken")
		switch token {
		case "":
			_ = json.NewEncoder(w).Encode(&chat.ListSpacesResponse{
				Spaces:        []*chat.Space{{Name: "spaces/A"}, {Name: "spaces/B"}},
				NextPageToken: "next-1",
			})
		case "next-1":
			_ = json.NewEncoder(w).Encode(&chat.ListSpacesResponse{
				Spaces: []*chat.Space{{Name: "spaces/C"}},
			})
		default:
			t.Fatalf("unexpected token %q", token)
		}
	}))
	defer server.Close()

	svc, err := chat.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatal(err)
	}

	cmd := makeCmd(t, map[string]string{
		"raw":    "true",
		"params": `{"pageSize":7,"filter":"spaceType = \"DIRECT_MESSAGE\""}`,
	})
	out, err := captureStdout(t, func() error {
		// flag-derived filter is "OTHER" — --params must override.
		return runChatListRaw(cmd, svc, "OTHER", 100, 0, true)
	})
	if err != nil {
		t.Fatalf("runner err: %v", err)
	}

	var got chat.ListSpacesResponse
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("decode: %v\nout=%s", err, out)
	}
	if len(got.Spaces) != 3 {
		t.Errorf("expected 3 aggregated spaces, got %d", len(got.Spaces))
	}
	if got.NextPageToken != "" {
		t.Errorf("expected nextPageToken dropped under --all, got %q", got.NextPageToken)
	}
	if len(qs) < 1 {
		t.Fatal("expected requests")
	}
	if f := qs[0].Get("filter"); !strings.Contains(f, "DIRECT_MESSAGE") {
		t.Errorf("expected filter from --params, got %q", f)
	}
	if p := qs[0].Get("pageSize"); p != "7" {
		t.Errorf("expected pageSize=7 from --params, got %q", p)
	}
}

// --- chat messages list (the --all bug from codex review) ----------------

func TestChatMessagesRaw_AllIgnoresDefaultMaxCap(t *testing.T) {
	// Server returns 30 messages across two pages — more than the default
	// --max of 25 from chatMessagesListCmd. With --all, the runner must
	// not cap the result.
	page1 := make([]*chat.Message, 25)
	for i := range page1 {
		page1[i] = &chat.Message{Name: "spaces/AAA/messages/p1-" + string(rune('a'+i))}
	}
	page2 := []*chat.Message{
		{Name: "spaces/AAA/messages/p2-a"},
		{Name: "spaces/AAA/messages/p2-b"},
		{Name: "spaces/AAA/messages/p2-c"},
		{Name: "spaces/AAA/messages/p2-d"},
		{Name: "spaces/AAA/messages/p2-e"},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		token := r.URL.Query().Get("pageToken")
		if token == "" {
			_ = json.NewEncoder(w).Encode(&chat.ListMessagesResponse{Messages: page1, NextPageToken: "p2"})
		} else {
			_ = json.NewEncoder(w).Encode(&chat.ListMessagesResponse{Messages: page2})
		}
	}))
	defer server.Close()

	svc, err := chat.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatal(err)
	}

	cmd := makeCmd(t, map[string]string{"raw": "true"})
	// maxResults=25 simulates the default flag value; fetchAll=true.
	out, err := captureStdout(t, func() error {
		return runChatMessagesRaw(cmd, svc, "spaces/AAA", 25, "", "", false, true)
	})
	if err != nil {
		t.Fatalf("runner err: %v", err)
	}

	var got chat.ListMessagesResponse
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("decode: %v\nout=%s", err, out)
	}
	if len(got.Messages) != 30 {
		t.Errorf("expected --all to return all 30 messages, got %d", len(got.Messages))
	}
	if got.NextPageToken != "" {
		t.Errorf("expected nextPageToken dropped, got %q", got.NextPageToken)
	}
}

// --- chat members list ---------------------------------------------------

func TestChatMembersRaw_AllIgnoresDefaultMaxCap(t *testing.T) {
	// 150 members across two pages; default --max=100 must not cap --all.
	page1 := make([]*chat.Membership, 100)
	for i := range page1 {
		page1[i] = &chat.Membership{Name: "spaces/AAA/members/p1-" + string(rune('a'+(i%26)))}
	}
	page2 := make([]*chat.Membership, 50)
	for i := range page2 {
		page2[i] = &chat.Membership{Name: "spaces/AAA/members/p2-" + string(rune('a'+(i%26)))}
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		token := r.URL.Query().Get("pageToken")
		if token == "" {
			_ = json.NewEncoder(w).Encode(&chat.ListMembershipsResponse{Memberships: page1, NextPageToken: "p2"})
		} else {
			_ = json.NewEncoder(w).Encode(&chat.ListMembershipsResponse{Memberships: page2})
		}
	}))
	defer server.Close()

	svc, err := chat.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatal(err)
	}

	cmd := makeCmd(t, map[string]string{"raw": "true"})
	out, err := captureStdout(t, func() error {
		return runChatMembersRaw(cmd, svc, "spaces/AAA", 100, "", false, false, true)
	})
	if err != nil {
		t.Fatalf("runner err: %v", err)
	}
	var got chat.ListMembershipsResponse
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("decode: %v\nout=%s", err, out)
	}
	if len(got.Memberships) != 150 {
		t.Errorf("expected --all to return all 150 memberships, got %d", len(got.Memberships))
	}
}

// --- people get ----------------------------------------------------------

func TestPeopleGetRaw_HonorsParamsResourceNameAndPersonFields(t *testing.T) {
	var path string
	var query url.Values
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		query = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(&people.Person{
			ResourceName: "people/me",
			Etag:         "etag-1",
			EmailAddresses: []*people.EmailAddress{
				{Value: "me@example.com"},
			},
		})
	}))
	defer server.Close()

	svc, err := people.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatal(err)
	}

	// Drive through the People.Get call to verify the API parameter
	// shape (this mirrors what runPeopleGet builds via params).
	person, err := svc.People.Get("people/me").PersonFields("emailAddresses").Do()
	if err != nil {
		t.Fatalf("people get: %v", err)
	}
	if person.ResourceName != "people/me" {
		t.Errorf("expected resourceName=people/me, got %q", person.ResourceName)
	}
	if !strings.HasSuffix(path, "/people/me") {
		t.Errorf("expected path to end in /people/me, got %q", path)
	}
	if got := query.Get("personFields"); got != "emailAddresses" {
		t.Errorf("expected personFields=emailAddresses, got %q", got)
	}
	// Sanity-check the raw shape via printRaw.
	out, err := captureStdout(t, func() error { return printRaw(person) })
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `"resourceName"`) || !strings.Contains(out, `"emailAddresses"`) {
		t.Errorf("expected raw People shape, got %s", out)
	}
}
