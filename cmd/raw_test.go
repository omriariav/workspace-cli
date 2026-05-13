package cmd

// Snapshot-style coverage for --raw output shape and --params parsing.
//
// We do not drive the Cobra runners end-to-end (those need real OAuth via
// client.NewFactory). Instead, we marshal the same SDK response structs the
// runners emit and assert that the JSON keys/structure match Google's
// public API reference for each endpoint.

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	chat "google.golang.org/api/chat/v1"
	gmail "google.golang.org/api/gmail/v1"
	people "google.golang.org/api/people/v1"
)

// requireKeys decodes raw JSON and fails if any of want is missing.
func requireKeys(t *testing.T, raw []byte, want ...string) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&m); err != nil {
		t.Fatalf("decode: %v\nraw=%s", err, string(raw))
	}
	for _, k := range want {
		if _, ok := m[k]; !ok {
			t.Errorf("expected top-level key %q in %s", k, string(raw))
		}
	}
	return m
}

// --- parseParams / param accessors ---------------------------------------

func newFlagCmd(rawJSON string, raw bool) *cobra.Command {
	c := &cobra.Command{Use: "x"}
	addRawParamsFlags(c)
	if rawJSON != "" {
		_ = c.Flags().Set("params", rawJSON)
	}
	if raw {
		_ = c.Flags().Set("raw", "true")
	}
	return c
}

func TestParseParams_EmptyAndMissing(t *testing.T) {
	c := newFlagCmd("", false)
	m, err := parseParams(c)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if m != nil {
		t.Fatalf("expected nil map, got %v", m)
	}
}

func TestParseParams_InvalidJSON(t *testing.T) {
	c := newFlagCmd("not json", false)
	if _, err := parseParams(c); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseParams_PreservesNumbersAndStrings(t *testing.T) {
	c := newFlagCmd(`{"pageSize":50,"filter":"createTime > \"2025-01-01T00:00:00Z\"","showDeleted":true,"sources":["READ_SOURCE_TYPE_CONTACT"]}`, true)
	m, err := parseParams(c)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if n, ok := paramInt64(m, "pageSize"); !ok || n != 50 {
		t.Errorf("pageSize: got (%d,%v)", n, ok)
	}
	if s, ok := paramString(m, "filter"); !ok || !strings.Contains(s, "createTime >") {
		t.Errorf("filter: got (%q,%v)", s, ok)
	}
	if b, ok := paramBool(m, "showDeleted"); !ok || !b {
		t.Errorf("showDeleted: got (%v,%v)", b, ok)
	}
	if ss, ok := paramStringSlice(m, "sources"); !ok || len(ss) != 1 || ss[0] != "READ_SOURCE_TYPE_CONTACT" {
		t.Errorf("sources: got (%v,%v)", ss, ok)
	}
	if !isRaw(c) {
		t.Error("expected --raw to be true")
	}
}

// --- API shape: gmail.users.messages.list --------------------------------

func TestRawShape_GmailListMessages(t *testing.T) {
	resp := &gmail.ListMessagesResponse{
		Messages: []*gmail.Message{
			{Id: "18abc", ThreadId: "thr-1"},
			{Id: "18def", ThreadId: "thr-2"},
		},
		NextPageToken:      "ZZZ",
		ResultSizeEstimate: 42,
	}
	raw, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	m := requireKeys(t, raw, "messages", "nextPageToken", "resultSizeEstimate")
	msgs, _ := m["messages"].([]interface{})
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	first, _ := msgs[0].(map[string]interface{})
	if _, ok := first["id"]; !ok {
		t.Errorf("expected message.id key, got %v", first)
	}
	if _, ok := first["threadId"]; !ok {
		t.Errorf("expected camelCase 'threadId', got %v", first)
	}
}

// --- API shape: gmail.users.threads.get ----------------------------------

func TestRawShape_GmailThreadGet(t *testing.T) {
	body := base64.URLEncoding.EncodeToString([]byte("hello world"))
	thread := &gmail.Thread{
		Id:        "thread-1",
		HistoryId: 1234,
		Messages: []*gmail.Message{
			{
				Id:           "msg-1",
				ThreadId:     "thread-1",
				LabelIds:     []string{"INBOX", "UNREAD"},
				Snippet:      "hello",
				InternalDate: 1700000000000,
				Payload: &gmail.MessagePart{
					MimeType: "multipart/alternative",
					Headers: []*gmail.MessagePartHeader{
						{Name: "Subject", Value: "Hi"},
						{Name: "From", Value: "a@b.com"},
					},
					Parts: []*gmail.MessagePart{
						{
							MimeType: "text/plain",
							Body:     &gmail.MessagePartBody{Data: body, Size: 11},
						},
					},
				},
			},
		},
	}
	raw, err := json.Marshal(thread)
	if err != nil {
		t.Fatal(err)
	}

	// Top-level shape: id, messages[], historyId.
	requireKeys(t, raw, "id", "messages", "historyId")

	// Drill into messages[0].payload.headers — must be {name,value} array.
	var threadOut struct {
		Messages []struct {
			Id           string   `json:"id"`
			LabelIds     []string `json:"labelIds"`
			Snippet      string   `json:"snippet"`
			InternalDate string   `json:"internalDate"`
			Payload      struct {
				MimeType string `json:"mimeType"`
				Headers  []struct {
					Name  string `json:"name"`
					Value string `json:"value"`
				} `json:"headers"`
				Parts []struct {
					MimeType string `json:"mimeType"`
					Body     struct {
						Data string `json:"data"`
						Size int64  `json:"size"`
					} `json:"body"`
				} `json:"parts"`
			} `json:"payload"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(raw, &threadOut); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(threadOut.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(threadOut.Messages))
	}
	msg := threadOut.Messages[0]
	if msg.InternalDate == "" {
		t.Error("expected internalDate (Gmail API returns it as a string of millis)")
	}
	if len(msg.LabelIds) != 2 {
		t.Errorf("expected 2 labelIds, got %v", msg.LabelIds)
	}
	if len(msg.Payload.Headers) != 2 || msg.Payload.Headers[0].Name != "Subject" {
		t.Errorf("expected headers as {name,value} array, got %+v", msg.Payload.Headers)
	}
	if len(msg.Payload.Parts) != 1 || msg.Payload.Parts[0].Body.Data != body {
		t.Errorf("expected nested parts[*].body.data preserved (base64), got %+v", msg.Payload.Parts)
	}
}

// --- API shape: chat.spaces.list -----------------------------------------

func TestRawShape_ChatListSpaces(t *testing.T) {
	resp := &chat.ListSpacesResponse{
		Spaces: []*chat.Space{
			{Name: "spaces/AAA", DisplayName: "Team", SpaceType: "SPACE"},
			{Name: "spaces/BBB", DisplayName: "Direct", SpaceType: "DIRECT_MESSAGE"},
		},
		NextPageToken: "tok",
	}
	raw, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	m := requireKeys(t, raw, "spaces", "nextPageToken")
	spaces, _ := m["spaces"].([]interface{})
	if len(spaces) != 2 {
		t.Fatalf("expected 2 spaces, got %d", len(spaces))
	}
	first, _ := spaces[0].(map[string]interface{})
	for _, k := range []string{"name", "displayName", "spaceType"} {
		if _, ok := first[k]; !ok {
			t.Errorf("expected space.%s key, got %v", k, first)
		}
	}
}

// --- API shape: chat.spaces.members.list ---------------------------------

func TestRawShape_ChatListMemberships(t *testing.T) {
	resp := &chat.ListMembershipsResponse{
		Memberships: []*chat.Membership{
			{Name: "spaces/AAA/members/m1", Role: "ROLE_MEMBER", Member: &chat.User{Name: "users/123", Type: "HUMAN", DisplayName: "Alice"}},
		},
		NextPageToken: "tok",
	}
	raw, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	m := requireKeys(t, raw, "memberships", "nextPageToken")
	members, _ := m["memberships"].([]interface{})
	if len(members) != 1 {
		t.Fatalf("expected 1 membership, got %d", len(members))
	}
	memb, _ := members[0].(map[string]interface{})
	for _, k := range []string{"name", "role", "member"} {
		if _, ok := memb[k]; !ok {
			t.Errorf("expected membership.%s key", k)
		}
	}
	user, _ := memb["member"].(map[string]interface{})
	for _, k := range []string{"name", "type", "displayName"} {
		if _, ok := user[k]; !ok {
			t.Errorf("expected member.%s key, got %v", k, user)
		}
	}
}

// --- API shape: chat.spaces.messages.list --------------------------------

func TestRawShape_ChatListMessages(t *testing.T) {
	resp := &chat.ListMessagesResponse{
		Messages: []*chat.Message{
			{
				Name:       "spaces/AAA/messages/msg-1",
				Sender:     &chat.User{Name: "users/123", DisplayName: "Alice"},
				CreateTime: "2026-04-01T10:00:00Z",
				Text:       "hello",
				Thread:     &chat.Thread{Name: "spaces/AAA/threads/t1"},
			},
		},
		NextPageToken: "tok",
	}
	raw, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	m := requireKeys(t, raw, "messages", "nextPageToken")
	msgs, _ := m["messages"].([]interface{})
	first, _ := msgs[0].(map[string]interface{})
	for _, k := range []string{"name", "sender", "createTime", "text", "thread"} {
		if _, ok := first[k]; !ok {
			t.Errorf("expected message.%s key, got %v", k, first)
		}
	}
	sender, _ := first["sender"].(map[string]interface{})
	if _, ok := sender["displayName"]; !ok {
		t.Errorf("expected sender.displayName camelCase, got %v", sender)
	}
}

// --- API shape: people.people.get ----------------------------------------

func TestRawShape_PeopleGet(t *testing.T) {
	person := &people.Person{
		ResourceName: "people/c12345",
		Etag:         "etag-x",
		Names: []*people.Name{
			{DisplayName: "Alice", GivenName: "Alice"},
		},
		EmailAddresses: []*people.EmailAddress{
			{Value: "alice@example.com", Type: "work"},
		},
	}
	raw, err := json.Marshal(person)
	if err != nil {
		t.Fatal(err)
	}
	m := requireKeys(t, raw, "resourceName", "etag", "names", "emailAddresses")
	names, _ := m["names"].([]interface{})
	if len(names) != 1 {
		t.Fatalf("expected 1 name, got %d", len(names))
	}
	first, _ := names[0].(map[string]interface{})
	for _, k := range []string{"displayName", "givenName"} {
		if _, ok := first[k]; !ok {
			t.Errorf("expected name.%s key, got %v", k, first)
		}
	}
	emails, _ := m["emailAddresses"].([]interface{})
	firstEmail, _ := emails[0].(map[string]interface{})
	for _, k := range []string{"value", "type"} {
		if _, ok := firstEmail[k]; !ok {
			t.Errorf("expected emailAddress.%s key, got %v", k, firstEmail)
		}
	}
}

// --- writeRaw produces stable indented JSON ------------------------------

func TestWriteRaw_IndentedJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := writeRaw(&buf, map[string]interface{}{"k": "v"}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "\n  \"k\": \"v\"") {
		t.Errorf("expected 2-space indented JSON, got %q", out)
	}
}
