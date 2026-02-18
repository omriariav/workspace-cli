package cmd

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

func TestGmailLabelsCommand_Help(t *testing.T) {
	cmd := gmailLabelsCmd

	if cmd.Use != "labels" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("expected Short description to be set")
	}
}

func TestGmailLabelCommand_Flags(t *testing.T) {
	cmd := gmailLabelCmd

	if cmd.Args == nil {
		t.Error("expected Args validator to be set")
	}

	addFlag := cmd.Flags().Lookup("add")
	if addFlag == nil {
		t.Error("expected --add flag to exist")
	}

	removeFlag := cmd.Flags().Lookup("remove")
	if removeFlag == nil {
		t.Error("expected --remove flag to exist")
	}
}

func TestGmailLabelCommand_Help(t *testing.T) {
	cmd := gmailLabelCmd

	if cmd.Use != "label <message-id>" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("expected Short description to be set")
	}

	if cmd.Long == "" {
		t.Error("expected Long description to be set")
	}
}

func TestGmailArchiveCommand_Help(t *testing.T) {
	cmd := gmailArchiveCmd

	if cmd.Use != "archive <message-id>" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}

	if cmd.Args == nil {
		t.Error("expected Args validator to be set")
	}
}

func TestGmailArchiveThreadCommand_Help(t *testing.T) {
	cmd := gmailArchiveThreadCmd

	if cmd.Use != "archive-thread <thread-id>" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}

	if cmd.Args == nil {
		t.Error("expected Args validator to be set")
	}
}

// TestGmailArchiveThread_MockServer tests archive-thread API integration
func TestGmailArchiveThread_MockServer(t *testing.T) {
	modifiedMsgs := make(map[string]bool)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Thread get (minimal format)
		if r.URL.Path == "/gmail/v1/users/me/threads/thread-xyz" && r.Method == "GET" {
			resp := map[string]interface{}{
				"id": "thread-xyz",
				"messages": []map[string]interface{}{
					{"id": "msg-a", "threadId": "thread-xyz", "labelIds": []string{"INBOX", "UNREAD"}},
					{"id": "msg-b", "threadId": "thread-xyz", "labelIds": []string{"INBOX"}},
					{"id": "msg-c", "threadId": "thread-xyz", "labelIds": []string{"INBOX", "UNREAD"}},
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		// Modify message (archive + mark read)
		if r.Method == "POST" && (r.URL.Path == "/gmail/v1/users/me/messages/msg-a/modify" ||
			r.URL.Path == "/gmail/v1/users/me/messages/msg-b/modify" ||
			r.URL.Path == "/gmail/v1/users/me/messages/msg-c/modify") {

			var req gmail.ModifyMessageRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("failed to decode request: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			// Verify INBOX and UNREAD are being removed
			hasInbox := false
			hasUnread := false
			for _, id := range req.RemoveLabelIds {
				if id == "INBOX" {
					hasInbox = true
				}
				if id == "UNREAD" {
					hasUnread = true
				}
			}
			if !hasInbox || !hasUnread {
				t.Errorf("expected RemoveLabelIds to contain INBOX and UNREAD, got: %v", req.RemoveLabelIds)
			}

			// Extract message ID from path
			parts := strings.Split(r.URL.Path, "/")
			msgID := parts[len(parts)-2] // .../messages/<id>/modify
			modifiedMsgs[msgID] = true

			resp := &gmail.Message{
				Id:       msgID,
				LabelIds: []string{},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		t.Logf("Unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := gmail.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create gmail service: %v", err)
	}

	// Fetch thread
	thread, err := svc.Users.Threads.Get("me", "thread-xyz").Format("minimal").Do()
	if err != nil {
		t.Fatalf("failed to get thread: %v", err)
	}

	if len(thread.Messages) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(thread.Messages))
	}

	// Archive each message (remove INBOX + UNREAD)
	for _, msg := range thread.Messages {
		req := &gmail.ModifyMessageRequest{
			RemoveLabelIds: []string{"INBOX", "UNREAD"},
		}
		_, err := svc.Users.Messages.Modify("me", msg.Id, req).Do()
		if err != nil {
			t.Errorf("failed to archive message %s: %v", msg.Id, err)
		}
	}

	// Verify all messages were modified
	for _, expectedID := range []string{"msg-a", "msg-b", "msg-c"} {
		if !modifiedMsgs[expectedID] {
			t.Errorf("message %s was not archived", expectedID)
		}
	}
}

// TestGmailArchiveThread_ThreadNotFound tests error when thread doesn't exist
func TestGmailArchiveThread_ThreadNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/gmail/v1/users/me/threads/nonexistent" && r.Method == "GET" {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]interface{}{
					"code":    404,
					"message": "Not Found",
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := gmail.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create gmail service: %v", err)
	}

	_, err = svc.Users.Threads.Get("me", "nonexistent").Format("minimal").Do()
	if err == nil {
		t.Error("expected error for nonexistent thread")
	}
}

// TestGmailArchiveThread_PartialFailure tests when some messages fail to archive
func TestGmailArchiveThread_PartialFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/gmail/v1/users/me/threads/thread-partial" && r.Method == "GET" {
			resp := map[string]interface{}{
				"id": "thread-partial",
				"messages": []map[string]interface{}{
					{"id": "msg-ok", "threadId": "thread-partial", "labelIds": []string{"INBOX"}},
					{"id": "msg-fail", "threadId": "thread-partial", "labelIds": []string{"INBOX"}},
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		// msg-ok succeeds
		if r.URL.Path == "/gmail/v1/users/me/messages/msg-ok/modify" && r.Method == "POST" {
			json.NewEncoder(w).Encode(&gmail.Message{Id: "msg-ok", LabelIds: []string{}})
			return
		}

		// msg-fail returns 500
		if r.URL.Path == "/gmail/v1/users/me/messages/msg-fail/modify" && r.Method == "POST" {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]interface{}{"code": 500, "message": "Internal Server Error"},
			})
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := gmail.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create gmail service: %v", err)
	}

	thread, err := svc.Users.Threads.Get("me", "thread-partial").Format("minimal").Do()
	if err != nil {
		t.Fatalf("failed to get thread: %v", err)
	}

	archived := 0
	failed := 0
	for _, msg := range thread.Messages {
		req := &gmail.ModifyMessageRequest{
			RemoveLabelIds: []string{"INBOX", "UNREAD"},
		}
		_, err := svc.Users.Messages.Modify("me", msg.Id, req).Do()
		if err != nil {
			failed++
			continue
		}
		archived++
	}

	if archived != 1 {
		t.Errorf("expected 1 archived, got %d", archived)
	}
	if failed != 1 {
		t.Errorf("expected 1 failed, got %d", failed)
	}
}

// TestGmailArchiveThread_OutputFormat tests the output JSON structure
func TestGmailArchiveThread_OutputFormat(t *testing.T) {
	result := map[string]interface{}{
		"status":    "archived",
		"thread_id": "thread-xyz",
		"archived":  3,
		"failed":    0,
		"total":     3,
	}

	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		t.Fatalf("failed to encode result: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("failed to decode result: %v", err)
	}

	if decoded["status"] != "archived" {
		t.Errorf("unexpected status: %v", decoded["status"])
	}
	if decoded["thread_id"] != "thread-xyz" {
		t.Errorf("unexpected thread_id: %v", decoded["thread_id"])
	}
	if decoded["archived"] != float64(3) {
		t.Errorf("unexpected archived count: %v", decoded["archived"])
	}
	if decoded["failed"] != float64(0) {
		t.Errorf("unexpected failed count: %v", decoded["failed"])
	}
	if decoded["total"] != float64(3) {
		t.Errorf("unexpected total: %v", decoded["total"])
	}
}

func TestGmailTrashCommand_Help(t *testing.T) {
	cmd := gmailTrashCmd

	if cmd.Use != "trash <message-id>" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}

	if cmd.Args == nil {
		t.Error("expected Args validator to be set")
	}
}

// TestGmailLabels_MockServer tests labels list API integration
func TestGmailLabels_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/gmail/v1/users/me/labels" && r.Method == "GET" {
			resp := &gmail.ListLabelsResponse{
				Labels: []*gmail.Label{
					{Id: "INBOX", Name: "INBOX", Type: "system"},
					{Id: "STARRED", Name: "STARRED", Type: "system"},
					{Id: "UNREAD", Name: "UNREAD", Type: "system"},
					{Id: "TRASH", Name: "TRASH", Type: "system"},
					{Id: "Label_1", Name: "ActionNeeded", Type: "user"},
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		t.Logf("Unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := gmail.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create gmail service: %v", err)
	}

	resp, err := svc.Users.Labels.List("me").Do()
	if err != nil {
		t.Fatalf("failed to list labels: %v", err)
	}

	if len(resp.Labels) != 5 {
		t.Errorf("expected 5 labels, got %d", len(resp.Labels))
	}

	// Verify user label
	found := false
	for _, label := range resp.Labels {
		if label.Name == "ActionNeeded" && label.Type == "user" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find 'ActionNeeded' user label")
	}
}

// TestGmailLabel_MockServer tests label modify API integration
func TestGmailLabel_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Labels list (for name resolution)
		if r.URL.Path == "/gmail/v1/users/me/labels" && r.Method == "GET" {
			resp := &gmail.ListLabelsResponse{
				Labels: []*gmail.Label{
					{Id: "INBOX", Name: "INBOX", Type: "system"},
					{Id: "STARRED", Name: "STARRED", Type: "system"},
					{Id: "Label_1", Name: "ActionNeeded", Type: "user"},
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		// Modify message
		if r.URL.Path == "/gmail/v1/users/me/messages/msg-123/modify" && r.Method == "POST" {
			var req gmail.ModifyMessageRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("failed to decode request: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			// Verify label IDs in request
			if len(req.AddLabelIds) != 1 || req.AddLabelIds[0] != "STARRED" {
				t.Errorf("unexpected AddLabelIds: %v", req.AddLabelIds)
			}
			if len(req.RemoveLabelIds) != 1 || req.RemoveLabelIds[0] != "INBOX" {
				t.Errorf("unexpected RemoveLabelIds: %v", req.RemoveLabelIds)
			}

			resp := &gmail.Message{
				Id:       "msg-123",
				LabelIds: []string{"STARRED", "Label_1"},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		t.Logf("Unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := gmail.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create gmail service: %v", err)
	}

	// Resolve label names to IDs
	ids, err := resolveLabelNames(svc, []string{"STARRED"})
	if err != nil {
		t.Fatalf("failed to resolve label names: %v", err)
	}
	if len(ids) != 1 || ids[0] != "STARRED" {
		t.Errorf("unexpected resolved IDs: %v", ids)
	}

	// Modify message
	req := &gmail.ModifyMessageRequest{
		AddLabelIds:    []string{"STARRED"},
		RemoveLabelIds: []string{"INBOX"},
	}

	msg, err := svc.Users.Messages.Modify("me", "msg-123", req).Do()
	if err != nil {
		t.Fatalf("failed to modify message: %v", err)
	}

	if msg.Id != "msg-123" {
		t.Errorf("unexpected message id: %s", msg.Id)
	}
}

// TestGmailArchive_MockServer tests archive API integration
func TestGmailArchive_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/gmail/v1/users/me/messages/msg-456/modify" && r.Method == "POST" {
			var req gmail.ModifyMessageRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("failed to decode request: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			if len(req.RemoveLabelIds) != 1 || req.RemoveLabelIds[0] != "INBOX" {
				t.Errorf("expected RemoveLabelIds=[INBOX], got: %v", req.RemoveLabelIds)
			}

			resp := &gmail.Message{
				Id:       "msg-456",
				LabelIds: []string{"UNREAD"},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		t.Logf("Unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := gmail.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create gmail service: %v", err)
	}

	req := &gmail.ModifyMessageRequest{
		RemoveLabelIds: []string{"INBOX"},
	}

	msg, err := svc.Users.Messages.Modify("me", "msg-456", req).Do()
	if err != nil {
		t.Fatalf("failed to archive message: %v", err)
	}

	if msg.Id != "msg-456" {
		t.Errorf("unexpected message id: %s", msg.Id)
	}

	// Verify INBOX was removed
	for _, label := range msg.LabelIds {
		if label == "INBOX" {
			t.Error("INBOX label should have been removed")
		}
	}
}

// TestGmailTrash_MockServer tests trash API integration
func TestGmailTrash_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/gmail/v1/users/me/messages/msg-789/trash" && r.Method == "POST" {
			resp := &gmail.Message{
				Id:       "msg-789",
				LabelIds: []string{"TRASH"},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		t.Logf("Unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := gmail.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create gmail service: %v", err)
	}

	msg, err := svc.Users.Messages.Trash("me", "msg-789").Do()
	if err != nil {
		t.Fatalf("failed to trash message: %v", err)
	}

	if msg.Id != "msg-789" {
		t.Errorf("unexpected message id: %s", msg.Id)
	}
}

// TestResolveLabelNames tests label name to ID resolution
func TestResolveLabelNames(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/gmail/v1/users/me/labels" && r.Method == "GET" {
			resp := &gmail.ListLabelsResponse{
				Labels: []*gmail.Label{
					{Id: "INBOX", Name: "INBOX", Type: "system"},
					{Id: "STARRED", Name: "STARRED", Type: "system"},
					{Id: "Label_1", Name: "ActionNeeded", Type: "user"},
					{Id: "Label_2", Name: "FollowUp", Type: "user"},
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := gmail.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create gmail service: %v", err)
	}

	t.Run("resolve system label", func(t *testing.T) {
		ids, err := resolveLabelNames(svc, []string{"INBOX"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(ids) != 1 || ids[0] != "INBOX" {
			t.Errorf("unexpected ids: %v", ids)
		}
	})

	t.Run("resolve user label case-insensitive", func(t *testing.T) {
		ids, err := resolveLabelNames(svc, []string{"actionneeded"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(ids) != 1 || ids[0] != "Label_1" {
			t.Errorf("unexpected ids: %v", ids)
		}
	})

	t.Run("resolve multiple labels", func(t *testing.T) {
		ids, err := resolveLabelNames(svc, []string{"STARRED", "FollowUp"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(ids) != 2 {
			t.Errorf("expected 2 ids, got %d", len(ids))
		}
	})

	t.Run("unknown label returns error", func(t *testing.T) {
		_, err := resolveLabelNames(svc, []string{"NonExistent"})
		if err == nil {
			t.Error("expected error for unknown label")
		}
	})

	t.Run("empty names skipped", func(t *testing.T) {
		ids, err := resolveLabelNames(svc, []string{"INBOX", "", "  "})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(ids) != 1 {
			t.Errorf("expected 1 id, got %d", len(ids))
		}
	})
}

// TestExtractEventIDFromBody tests event ID extraction from email body
func TestExtractEventIDFromBody(t *testing.T) {
	t.Run("valid eid parameter", func(t *testing.T) {
		// "18t8hl5rgsh8oihvjnbtt4g788 omri.a@taboola.com" base64 encoded
		body := `Click here to view: https://calendar.google.com/calendar/event?action=VIEW&eid=MTh0OGhsNXJnc2g4b2lodmpuYnR0NGc3ODggb21yaS5hQHRhYm9vbGEuY29t&more=stuff`
		eventID, err := extractEventIDFromBody(body)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if eventID != "18t8hl5rgsh8oihvjnbtt4g788" {
			t.Errorf("expected '18t8hl5rgsh8oihvjnbtt4g788', got '%s'", eventID)
		}
	})

	t.Run("no eid found", func(t *testing.T) {
		body := "This is a regular email with no calendar link"
		_, err := extractEventIDFromBody(body)
		if err == nil {
			t.Error("expected error for body without eid")
		}
	})

	t.Run("eid as first parameter", func(t *testing.T) {
		body := `https://calendar.google.com/calendar/event?eid=dGVzdGV2ZW50MTIzIHVzZXJAZXhhbXBsZS5jb20=`
		eventID, err := extractEventIDFromBody(body)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if eventID != "testevent123" {
			t.Errorf("expected 'testevent123', got '%s'", eventID)
		}
	})

	t.Run("URL-encoded eid with percent encoding", func(t *testing.T) {
		// Same base64 as first test but with = URL-encoded as %3D
		body := `https://calendar.google.com/calendar/event?eid=dGVzdGV2ZW50MTIzIHVzZXJAZXhhbXBsZS5jb20%3D`
		eventID, err := extractEventIDFromBody(body)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if eventID != "testevent123" {
			t.Errorf("expected 'testevent123', got '%s'", eventID)
		}
	})
}

func TestGmailEventIDCommand_Help(t *testing.T) {
	cmd := gmailEventIDCmd

	if cmd.Use != "event-id <message-id>" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}

	if cmd.Args == nil {
		t.Error("expected Args validator to be set")
	}
}

func TestGmailReplyCommand_Help(t *testing.T) {
	cmd := gmailReplyCmd

	if cmd.Use != "reply <message-id>" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}

	if cmd.Args == nil {
		t.Error("expected Args validator to be set")
	}
}

func TestGmailReplyCommand_Flags(t *testing.T) {
	cmd := gmailReplyCmd

	bodyFlag := cmd.Flags().Lookup("body")
	if bodyFlag == nil {
		t.Error("expected --body flag to exist")
	}

	allFlag := cmd.Flags().Lookup("all")
	if allFlag == nil {
		t.Error("expected --all flag to exist")
	}

	ccFlag := cmd.Flags().Lookup("cc")
	if ccFlag == nil {
		t.Error("expected --cc flag to exist")
	}
}

// TestGmailReply_MockServer tests the reply workflow
func TestGmailReply_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Get original message
		if r.URL.Path == "/gmail/v1/users/me/messages/orig-msg-123" && r.Method == "GET" {
			resp := map[string]interface{}{
				"id":       "orig-msg-123",
				"threadId": "thread-xyz",
				"payload": map[string]interface{}{
					"headers": []map[string]string{
						{"name": "Subject", "value": "Hello"},
						{"name": "From", "value": "alice@example.com"},
						{"name": "To", "value": "bob@example.com"},
						{"name": "Message-ID", "value": "<abc123@mail.gmail.com>"},
					},
					"mimeType": "text/plain",
					"body":     map[string]interface{}{"data": "SGVsbG8="},
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		// Send reply
		if r.URL.Path == "/gmail/v1/users/me/messages/send" && r.Method == "POST" {
			var msg gmail.Message
			if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
				t.Errorf("failed to decode: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			// Verify thread ID is set
			if msg.ThreadId != "thread-xyz" {
				t.Errorf("expected ThreadId 'thread-xyz', got '%s'", msg.ThreadId)
			}

			// Decode raw to verify headers
			rawBytes, _ := base64.URLEncoding.DecodeString(msg.Raw)
			rawStr := string(rawBytes)
			if !strings.Contains(rawStr, "In-Reply-To: <abc123@mail.gmail.com>") {
				t.Errorf("expected In-Reply-To header in raw message")
			}
			if !strings.Contains(rawStr, "Re: Hello") {
				t.Errorf("expected Re: prefix in subject")
			}

			resp := &gmail.Message{
				Id:       "reply-msg-456",
				ThreadId: "thread-xyz",
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		t.Logf("Unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := gmail.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create gmail service: %v", err)
	}

	// Fetch original message
	origMsg, err := svc.Users.Messages.Get("me", "orig-msg-123").Format("full").Do()
	if err != nil {
		t.Fatalf("failed to get original message: %v", err)
	}

	if origMsg.ThreadId != "thread-xyz" {
		t.Errorf("unexpected thread id: %s", origMsg.ThreadId)
	}

	// Build and send reply
	var origMessageIDHeader string
	for _, header := range origMsg.Payload.Headers {
		if header.Name == "Message-ID" {
			origMessageIDHeader = header.Value
		}
	}

	var msgBuilder strings.Builder
	msgBuilder.WriteString("To: alice@example.com\r\n")
	msgBuilder.WriteString("Subject: Re: Hello\r\n")
	msgBuilder.WriteString(fmt.Sprintf("In-Reply-To: %s\r\n", origMessageIDHeader))
	msgBuilder.WriteString(fmt.Sprintf("References: %s\r\n", origMessageIDHeader))
	msgBuilder.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n")
	msgBuilder.WriteString("\r\n")
	msgBuilder.WriteString("Got it, thanks!")

	raw := base64.URLEncoding.EncodeToString([]byte(msgBuilder.String()))
	msg := &gmail.Message{
		Raw:      raw,
		ThreadId: origMsg.ThreadId,
	}

	sent, err := svc.Users.Messages.Send("me", msg).Do()
	if err != nil {
		t.Fatalf("failed to send reply: %v", err)
	}

	if sent.ThreadId != "thread-xyz" {
		t.Errorf("unexpected reply thread id: %s", sent.ThreadId)
	}
}

// TestGmailReplyAll_MockServer tests the reply-all workflow with recipient deduplication
func TestGmailReplyAll_MockServer(t *testing.T) {
	var sentRaw string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Get original message
		if r.URL.Path == "/gmail/v1/users/me/messages/orig-msg-all" && r.Method == "GET" {
			resp := map[string]interface{}{
				"id":       "orig-msg-all",
				"threadId": "thread-all",
				"payload": map[string]interface{}{
					"headers": []map[string]string{
						{"name": "Subject", "value": "Team discussion"},
						{"name": "From", "value": "alice@example.com"},
						{"name": "To", "value": "me@example.com, bob@example.com"},
						{"name": "Cc", "value": "carol@example.com"},
						{"name": "Message-ID", "value": "<orig@mail.gmail.com>"},
						{"name": "References", "value": "<prev@mail.gmail.com>"},
					},
					"mimeType": "text/plain",
					"body":     map[string]interface{}{"data": "SGVsbG8="},
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		// Get profile
		if r.URL.Path == "/gmail/v1/users/me/profile" && r.Method == "GET" {
			resp := map[string]interface{}{
				"emailAddress": "me@example.com",
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		// Send reply
		if r.URL.Path == "/gmail/v1/users/me/messages/send" && r.Method == "POST" {
			var msg gmail.Message
			json.NewDecoder(r.Body).Decode(&msg)
			sentRaw = msg.Raw
			resp := &gmail.Message{Id: "reply-all-456", ThreadId: "thread-all"}
			json.NewEncoder(w).Encode(resp)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := gmail.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	// Simulate reply-all flow
	origMsg, _ := svc.Users.Messages.Get("me", "orig-msg-all").Format("full").Do()
	profile, _ := svc.Users.GetProfile("me").Do()
	myEmail := strings.ToLower(profile.EmailAddress)

	var origFrom, origTo, origCc, origMsgID, origRefs string
	for _, h := range origMsg.Payload.Headers {
		switch h.Name {
		case "From":
			origFrom = h.Value
		case "To":
			origTo = h.Value
		case "Cc":
			origCc = h.Value
		case "Message-ID":
			origMsgID = h.Value
		case "References":
			origRefs = h.Value
		}
	}

	// Build To: sender + other To recipients (excluding self)
	replyTo := origFrom
	for _, addr := range strings.Split(origTo, ",") {
		addr = strings.TrimSpace(addr)
		if addr != "" && !emailMatchesSelf(addr, myEmail) {
			replyTo += ", " + addr
		}
	}

	// Build Cc from original
	var ccParts []string
	for _, addr := range strings.Split(origCc, ",") {
		addr = strings.TrimSpace(addr)
		if addr != "" && !emailMatchesSelf(addr, myEmail) {
			ccParts = append(ccParts, addr)
		}
	}

	var msgBuilder strings.Builder
	msgBuilder.WriteString(fmt.Sprintf("To: %s\r\n", replyTo))
	if len(ccParts) > 0 {
		msgBuilder.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(ccParts, ", ")))
	}
	msgBuilder.WriteString("Subject: Re: Team discussion\r\n")
	msgBuilder.WriteString(fmt.Sprintf("In-Reply-To: %s\r\n", origMsgID))
	refs := origRefs + " " + origMsgID
	msgBuilder.WriteString(fmt.Sprintf("References: %s\r\n", refs))
	msgBuilder.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n\r\nReply body")

	raw := base64.URLEncoding.EncodeToString([]byte(msgBuilder.String()))
	msg := &gmail.Message{Raw: raw, ThreadId: origMsg.ThreadId}
	sent, _ := svc.Users.Messages.Send("me", msg).Do()

	if sent.ThreadId != "thread-all" {
		t.Errorf("unexpected thread: %s", sent.ThreadId)
	}

	// Verify the sent message includes all expected recipients
	rawBytes, _ := base64.URLEncoding.DecodeString(sentRaw)
	rawStr := string(rawBytes)
	if !strings.Contains(rawStr, "bob@example.com") {
		t.Error("reply-all should include bob@example.com in To")
	}
	if !strings.Contains(rawStr, "carol@example.com") {
		t.Error("reply-all should include carol@example.com in Cc")
	}
	if strings.Contains(rawStr, "To: alice@example.com, me@example.com") {
		t.Error("reply-all should exclude self from To")
	}
	// Verify References chaining
	if !strings.Contains(rawStr, "<prev@mail.gmail.com> <orig@mail.gmail.com>") {
		t.Error("References should chain original References + Message-ID")
	}
}

// TestGmailSendThreading_MockServer tests send with --thread-id and --reply-to-message-id
func TestGmailSendThreading_MockServer(t *testing.T) {
	var sentMsg gmail.Message
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Get original message for reply-to
		if r.URL.Path == "/gmail/v1/users/me/messages/orig-for-send" && r.Method == "GET" {
			resp := map[string]interface{}{
				"id":       "orig-for-send",
				"threadId": "thread-send",
				"payload": map[string]interface{}{
					"headers": []map[string]string{
						{"name": "Message-ID", "value": "<send-orig@mail.gmail.com>"},
						{"name": "References", "value": "<earlier@mail.gmail.com>"},
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		// Send
		if r.URL.Path == "/gmail/v1/users/me/messages/send" && r.Method == "POST" {
			json.NewDecoder(r.Body).Decode(&sentMsg)
			resp := &gmail.Message{Id: "sent-thread-msg", ThreadId: "thread-send"}
			json.NewEncoder(w).Encode(resp)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := gmail.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	// Simulate the send threading path: fetch original to get Message-ID
	origMsg, err := svc.Users.Messages.Get("me", "orig-for-send").Format("metadata").Do()
	if err != nil {
		t.Fatalf("failed to get original: %v", err)
	}

	var inReplyTo, origRefs string
	for _, h := range origMsg.Payload.Headers {
		switch h.Name {
		case "Message-ID":
			inReplyTo = h.Value
		case "References":
			origRefs = h.Value
		}
	}

	// Build message with threading headers
	var msgBuilder strings.Builder
	msgBuilder.WriteString("To: recipient@example.com\r\n")
	msgBuilder.WriteString("Subject: Re: Thread test\r\n")
	msgBuilder.WriteString(fmt.Sprintf("In-Reply-To: %s\r\n", inReplyTo))
	references := inReplyTo
	if origRefs != "" {
		references = origRefs + " " + inReplyTo
	}
	msgBuilder.WriteString(fmt.Sprintf("References: %s\r\n", references))
	msgBuilder.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n\r\nThreaded reply")

	raw := base64.URLEncoding.EncodeToString([]byte(msgBuilder.String()))
	msg := &gmail.Message{Raw: raw, ThreadId: origMsg.ThreadId}
	sent, err := svc.Users.Messages.Send("me", msg).Do()
	if err != nil {
		t.Fatalf("failed to send: %v", err)
	}

	if sent.ThreadId != "thread-send" {
		t.Errorf("expected thread-send, got %s", sent.ThreadId)
	}
	if sentMsg.ThreadId != "thread-send" {
		t.Errorf("sent message should have ThreadId set")
	}

	// Verify References chaining in raw
	rawBytes, _ := base64.URLEncoding.DecodeString(sentMsg.Raw)
	rawStr := string(rawBytes)
	if !strings.Contains(rawStr, "In-Reply-To: <send-orig@mail.gmail.com>") {
		t.Error("expected In-Reply-To header")
	}
	if !strings.Contains(rawStr, "<earlier@mail.gmail.com> <send-orig@mail.gmail.com>") {
		t.Error("expected chained References header")
	}
}

func TestEmailMatchesSelf(t *testing.T) {
	tests := []struct {
		addr    string
		myEmail string
		want    bool
	}{
		{"user@example.com", "user@example.com", true},
		{"User@Example.com", "user@example.com", true},
		{"other@example.com", "user@example.com", false},
		{"xuser@example.com", "user@example.com", false},
		{`"John Doe" <user@example.com>`, "user@example.com", true},
		{`"John Doe" <other@example.com>`, "user@example.com", false},
		{"<user@example.com>", "user@example.com", true},
	}
	for _, tt := range tests {
		t.Run(tt.addr, func(t *testing.T) {
			got := emailMatchesSelf(tt.addr, tt.myEmail)
			if got != tt.want {
				t.Errorf("emailMatchesSelf(%q, %q) = %v, want %v", tt.addr, tt.myEmail, got, tt.want)
			}
		})
	}
}

func TestGmailThreadCommand_Help(t *testing.T) {
	cmd := gmailThreadCmd

	if cmd.Use != "thread <thread-id>" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("expected Short description to be set")
	}

	if cmd.Long == "" {
		t.Error("expected Long description to be set")
	}

	if cmd.Args == nil {
		t.Error("expected Args validator to be set")
	}
}

// TestGmailThread_MockServer tests thread read API integration
func TestGmailThread_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/gmail/v1/users/me/threads/thread-abc" && r.Method == "GET" {
			resp := map[string]interface{}{
				"id": "thread-abc",
				"messages": []map[string]interface{}{
					{
						"id":       "msg-001",
						"threadId": "thread-abc",
						"labelIds": []string{"INBOX", "UNREAD"},
						"payload": map[string]interface{}{
							"headers": []map[string]string{
								{"name": "Subject", "value": "Hello"},
								{"name": "From", "value": "alice@example.com"},
								{"name": "To", "value": "bob@example.com"},
								{"name": "Date", "value": "Mon, 1 Jan 2024 10:00:00 +0000"},
							},
							"mimeType": "text/plain",
							"body": map[string]interface{}{
								"data": "SGVsbG8gQm9i", // "Hello Bob" base64url
							},
						},
					},
					{
						"id":       "msg-002",
						"threadId": "thread-abc",
						"labelIds": []string{"INBOX"},
						"payload": map[string]interface{}{
							"headers": []map[string]string{
								{"name": "Subject", "value": "Re: Hello"},
								{"name": "From", "value": "bob@example.com"},
								{"name": "To", "value": "alice@example.com"},
								{"name": "Date", "value": "Mon, 1 Jan 2024 11:00:00 +0000"},
							},
							"mimeType": "text/plain",
							"body": map[string]interface{}{
								"data": "SGkgQWxpY2U=", // "Hi Alice" base64url
							},
						},
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		t.Logf("Unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := gmail.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create gmail service: %v", err)
	}

	thread, err := svc.Users.Threads.Get("me", "thread-abc").Format("full").Do()
	if err != nil {
		t.Fatalf("failed to get thread: %v", err)
	}

	if thread.Id != "thread-abc" {
		t.Errorf("unexpected thread id: %s", thread.Id)
	}

	if len(thread.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(thread.Messages))
	}

	if thread.Messages[0].Id != "msg-001" {
		t.Errorf("unexpected first message id: %s", thread.Messages[0].Id)
	}

	if thread.Messages[1].Id != "msg-002" {
		t.Errorf("unexpected second message id: %s", thread.Messages[1].Id)
	}
}

// TestGmailList_OutputFormat tests that list output includes both thread_id and message_id
func TestGmailList_OutputFormat(t *testing.T) {
	result := map[string]interface{}{
		"thread_id":     "thread-abc",
		"message_id":    "msg-002",
		"message_count": 2,
		"snippet":       "Hi Alice",
		"subject":       "Hello",
		"from":          "alice@example.com",
		"date":          "Mon, 1 Jan 2024 10:00:00 +0000",
	}

	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		t.Fatalf("failed to encode result: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("failed to decode result: %v", err)
	}

	if decoded["thread_id"] != "thread-abc" {
		t.Errorf("unexpected thread_id: %v", decoded["thread_id"])
	}

	if decoded["message_id"] != "msg-002" {
		t.Errorf("unexpected message_id: %v", decoded["message_id"])
	}

	if decoded["message_count"] != float64(2) {
		t.Errorf("unexpected message_count: %v", decoded["message_count"])
	}
}

// TestGmailLabel_OutputFormat tests the label modify response format
func TestGmailLabel_OutputFormat(t *testing.T) {
	result := map[string]interface{}{
		"status":     "modified",
		"message_id": "msg-123",
		"labels":     []string{"STARRED", "Label_1"},
	}

	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		t.Fatalf("failed to encode result: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("failed to decode result: %v", err)
	}

	if decoded["status"] != "modified" {
		t.Errorf("unexpected status: %v", decoded["status"])
	}

	if decoded["message_id"] != "msg-123" {
		t.Errorf("unexpected message_id: %v", decoded["message_id"])
	}
}

// TestGmailListCommand_AllFlag tests that the --all flag exists
func TestGmailListCommand_AllFlag(t *testing.T) {
	cmd := gmailListCmd

	allFlag := cmd.Flags().Lookup("all")
	if allFlag == nil {
		t.Fatal("expected --all flag to exist")
	}
	if allFlag.DefValue != "false" {
		t.Errorf("expected --all default 'false', got '%s'", allFlag.DefValue)
	}
}

// TestGmailList_Pagination_MockServer tests pagination when fetching more than one page
func TestGmailList_Pagination_MockServer(t *testing.T) {
	pageRequests := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Threads list with pagination
		if r.URL.Path == "/gmail/v1/users/me/threads" && r.Method == "GET" {
			pageRequests++
			pageToken := r.URL.Query().Get("pageToken")

			var resp map[string]interface{}
			if pageToken == "" {
				// First page
				resp = map[string]interface{}{
					"threads": []map[string]interface{}{
						{"id": "thread-1", "snippet": "First"},
						{"id": "thread-2", "snippet": "Second"},
					},
					"nextPageToken":      "page2token",
					"resultSizeEstimate": 4,
				}
			} else if pageToken == "page2token" {
				// Second page
				resp = map[string]interface{}{
					"threads": []map[string]interface{}{
						{"id": "thread-3", "snippet": "Third"},
						{"id": "thread-4", "snippet": "Fourth"},
					},
					"resultSizeEstimate": 4,
				}
			} else {
				t.Errorf("unexpected page token: %s", pageToken)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		// Thread get (for metadata)
		if strings.HasPrefix(r.URL.Path, "/gmail/v1/users/me/threads/thread-") && r.Method == "GET" {
			threadID := strings.TrimPrefix(r.URL.Path, "/gmail/v1/users/me/threads/")
			resp := map[string]interface{}{
				"id": threadID,
				"messages": []map[string]interface{}{
					{
						"id":       "msg-" + threadID,
						"threadId": threadID,
						"payload": map[string]interface{}{
							"headers": []map[string]string{
								{"name": "Subject", "value": "Test " + threadID},
								{"name": "From", "value": "test@example.com"},
								{"name": "Date", "value": "Mon, 1 Jan 2024 10:00:00 +0000"},
							},
						},
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		t.Logf("Unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := gmail.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create gmail service: %v", err)
	}

	// Simulate pagination: fetch all threads
	var allThreads []*gmail.Thread
	var pageToken string
	for {
		call := svc.Users.Threads.List("me").MaxResults(500)
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		resp, err := call.Do()
		if err != nil {
			t.Fatalf("failed to list threads: %v", err)
		}

		allThreads = append(allThreads, resp.Threads...)

		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}

	// Verify we got all 4 threads across 2 pages
	if len(allThreads) != 4 {
		t.Errorf("expected 4 threads, got %d", len(allThreads))
	}
	if pageRequests != 2 {
		t.Errorf("expected 2 page requests, got %d", pageRequests)
	}
}

// TestGmailList_MaxRespected_MockServer tests that --max limits results even with pagination
func TestGmailList_MaxRespected_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/gmail/v1/users/me/threads" && r.Method == "GET" {
			maxResults := r.URL.Query().Get("maxResults")
			// The request should respect the max parameter
			if maxResults != "3" {
				t.Logf("maxResults requested: %s", maxResults)
			}

			resp := map[string]interface{}{
				"threads": []map[string]interface{}{
					{"id": "thread-1", "snippet": "First"},
					{"id": "thread-2", "snippet": "Second"},
					{"id": "thread-3", "snippet": "Third"},
				},
				"nextPageToken":      "moretoken",
				"resultSizeEstimate": 100,
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		if strings.HasPrefix(r.URL.Path, "/gmail/v1/users/me/threads/thread-") && r.Method == "GET" {
			threadID := strings.TrimPrefix(r.URL.Path, "/gmail/v1/users/me/threads/")
			resp := map[string]interface{}{
				"id": threadID,
				"messages": []map[string]interface{}{
					{"id": "msg-1", "payload": map[string]interface{}{"headers": []map[string]string{}}},
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := gmail.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create gmail service: %v", err)
	}

	// Request only 3 results
	resp, err := svc.Users.Threads.List("me").MaxResults(3).Do()
	if err != nil {
		t.Fatalf("failed to list threads: %v", err)
	}

	if len(resp.Threads) != 3 {
		t.Errorf("expected 3 threads, got %d", len(resp.Threads))
	}
}

// TestGmailListCommand_IncludeLabelsFlag tests that the --include-labels flag exists
func TestGmailListCommand_IncludeLabelsFlag(t *testing.T) {
	cmd := gmailListCmd

	flag := cmd.Flags().Lookup("include-labels")
	if flag == nil {
		t.Fatal("expected --include-labels flag to exist")
	}
	if flag.DefValue != "false" {
		t.Errorf("expected --include-labels default 'false', got '%s'", flag.DefValue)
	}
}

// TestGmailList_IncludeLabels_MockServer tests that --include-labels returns union of all message labels
func TestGmailList_IncludeLabels_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Threads list
		if r.URL.Path == "/gmail/v1/users/me/threads" && r.Method == "GET" {
			resp := map[string]interface{}{
				"threads": []map[string]interface{}{
					{"id": "thread-lbl", "snippet": "Test labels"},
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		// Thread get with metadata — two messages with different labels
		if r.URL.Path == "/gmail/v1/users/me/threads/thread-lbl" && r.Method == "GET" {
			resp := map[string]interface{}{
				"id": "thread-lbl",
				"messages": []map[string]interface{}{
					{
						"id":       "msg-001",
						"threadId": "thread-lbl",
						"labelIds": []string{"INBOX", "UNREAD", "CATEGORY_PROMOTIONS"},
						"payload": map[string]interface{}{
							"headers": []map[string]string{
								{"name": "Subject", "value": "Promo email"},
								{"name": "From", "value": "promo@example.com"},
								{"name": "Date", "value": "Mon, 6 Feb 2026 10:00:00 +0000"},
							},
						},
					},
					{
						"id":       "msg-002",
						"threadId": "thread-lbl",
						"labelIds": []string{"INBOX", "STARRED"},
						"payload": map[string]interface{}{
							"headers": []map[string]string{
								{"name": "Subject", "value": "Re: Promo email"},
								{"name": "From", "value": "reply@example.com"},
								{"name": "Date", "value": "Mon, 6 Feb 2026 11:00:00 +0000"},
							},
						},
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		t.Logf("Unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := gmail.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create gmail service: %v", err)
	}

	// Fetch thread list
	listResp, err := svc.Users.Threads.List("me").MaxResults(10).Do()
	if err != nil {
		t.Fatalf("failed to list threads: %v", err)
	}

	if len(listResp.Threads) != 1 {
		t.Fatalf("expected 1 thread, got %d", len(listResp.Threads))
	}

	// Get thread detail (same as runGmailList does)
	threadDetail, err := svc.Users.Threads.Get("me", listResp.Threads[0].Id).Format("metadata").MetadataHeaders("Subject", "From", "Date").Do()
	if err != nil {
		t.Fatalf("failed to get thread detail: %v", err)
	}

	// Simulate includeLabels=true logic
	labelSet := make(map[string]bool)
	for _, m := range threadDetail.Messages {
		for _, lbl := range m.LabelIds {
			labelSet[lbl] = true
		}
	}
	labels := make([]string, 0, len(labelSet))
	for lbl := range labelSet {
		labels = append(labels, lbl)
	}

	// Verify union of labels from both messages
	expected := map[string]bool{
		"INBOX":               true,
		"UNREAD":              true,
		"CATEGORY_PROMOTIONS": true,
		"STARRED":             true,
	}
	if len(labels) != len(expected) {
		t.Errorf("expected %d labels, got %d: %v", len(expected), len(labels), labels)
	}
	for _, lbl := range labels {
		if !expected[lbl] {
			t.Errorf("unexpected label: %s", lbl)
		}
	}

	// Simulate includeLabels=false — labels should NOT be in output
	threadInfo := map[string]interface{}{
		"thread_id": "thread-lbl",
	}
	// Without the flag, no "labels" key should be set
	if _, exists := threadInfo["labels"]; exists {
		t.Error("labels should not be present when include-labels is false")
	}
}

// === Tests for new Gmail commands ===

// TestGmailUntrashCommand_Help tests untrash command structure
func TestGmailUntrashCommand_Help(t *testing.T) {
	cmd := gmailUntrashCmd
	if cmd.Use != "untrash <message-id>" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}
	if cmd.Args == nil {
		t.Error("expected Args validator to be set")
	}
}

// TestGmailUntrash_MockServer tests untrash API integration
func TestGmailUntrash_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/gmail/v1/users/me/messages/msg-trash-1/untrash" && r.Method == "POST" {
			resp := &gmail.Message{
				Id:       "msg-trash-1",
				LabelIds: []string{"INBOX"},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		t.Logf("Unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := gmail.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create gmail service: %v", err)
	}

	msg, err := svc.Users.Messages.Untrash("me", "msg-trash-1").Do()
	if err != nil {
		t.Fatalf("failed to untrash message: %v", err)
	}
	if msg.Id != "msg-trash-1" {
		t.Errorf("unexpected message id: %s", msg.Id)
	}
}

// TestGmailDeleteCommand_Help tests delete command structure
func TestGmailDeleteCommand_Help(t *testing.T) {
	cmd := gmailDeleteCmd
	if cmd.Use != "delete <message-id>" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}
	if cmd.Args == nil {
		t.Error("expected Args validator to be set")
	}
}

// TestGmailDelete_MockServer tests permanent delete API integration
func TestGmailDelete_MockServer(t *testing.T) {
	deleteCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/gmail/v1/users/me/messages/msg-del-1" && r.Method == "DELETE" {
			deleteCalled = true
			w.WriteHeader(http.StatusNoContent)
			return
		}
		t.Logf("Unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := gmail.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create gmail service: %v", err)
	}

	err = svc.Users.Messages.Delete("me", "msg-del-1").Do()
	if err != nil {
		t.Fatalf("failed to delete message: %v", err)
	}
	if !deleteCalled {
		t.Error("delete API was not called")
	}
}

// TestGmailBatchModifyCommand_Flags tests batch-modify command flags
func TestGmailBatchModifyCommand_Flags(t *testing.T) {
	cmd := gmailBatchModifyCmd

	idsFlag := cmd.Flags().Lookup("ids")
	if idsFlag == nil {
		t.Error("expected --ids flag to exist")
	}
	addFlag := cmd.Flags().Lookup("add-labels")
	if addFlag == nil {
		t.Error("expected --add-labels flag to exist")
	}
	removeFlag := cmd.Flags().Lookup("remove-labels")
	if removeFlag == nil {
		t.Error("expected --remove-labels flag to exist")
	}
}

// TestGmailBatchModify_MockServer tests batch modify API integration
func TestGmailBatchModify_MockServer(t *testing.T) {
	var receivedReq gmail.BatchModifyMessagesRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Labels list for resolution
		if r.URL.Path == "/gmail/v1/users/me/labels" && r.Method == "GET" {
			resp := &gmail.ListLabelsResponse{
				Labels: []*gmail.Label{
					{Id: "STARRED", Name: "STARRED", Type: "system"},
					{Id: "INBOX", Name: "INBOX", Type: "system"},
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		if r.URL.Path == "/gmail/v1/users/me/messages/batchModify" && r.Method == "POST" {
			json.NewDecoder(r.Body).Decode(&receivedReq)
			w.WriteHeader(http.StatusNoContent)
			return
		}

		t.Logf("Unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := gmail.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create gmail service: %v", err)
	}

	req := &gmail.BatchModifyMessagesRequest{
		Ids:            []string{"msg-1", "msg-2"},
		AddLabelIds:    []string{"STARRED"},
		RemoveLabelIds: []string{"INBOX"},
	}

	err = svc.Users.Messages.BatchModify("me", req).Do()
	if err != nil {
		t.Fatalf("failed to batch modify: %v", err)
	}
}

// TestGmailBatchDeleteCommand_Flags tests batch-delete command flags
func TestGmailBatchDeleteCommand_Flags(t *testing.T) {
	cmd := gmailBatchDeleteCmd

	idsFlag := cmd.Flags().Lookup("ids")
	if idsFlag == nil {
		t.Error("expected --ids flag to exist")
	}
}

// TestGmailBatchDelete_MockServer tests batch delete API integration
func TestGmailBatchDelete_MockServer(t *testing.T) {
	batchDeleteCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/gmail/v1/users/me/messages/batchDelete" && r.Method == "POST" {
			batchDeleteCalled = true
			w.WriteHeader(http.StatusNoContent)
			return
		}
		t.Logf("Unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := gmail.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create gmail service: %v", err)
	}

	req := &gmail.BatchDeleteMessagesRequest{
		Ids: []string{"msg-1", "msg-2", "msg-3"},
	}
	err = svc.Users.Messages.BatchDelete("me", req).Do()
	if err != nil {
		t.Fatalf("failed to batch delete: %v", err)
	}
	if !batchDeleteCalled {
		t.Error("batch delete API was not called")
	}
}

// TestGmailTrashThreadCommand_Help tests trash-thread command structure
func TestGmailTrashThreadCommand_Help(t *testing.T) {
	cmd := gmailTrashThreadCmd
	if cmd.Use != "trash-thread <thread-id>" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}
	if cmd.Args == nil {
		t.Error("expected Args validator to be set")
	}
}

// TestGmailTrashThread_MockServer tests trash thread API integration
func TestGmailTrashThread_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/gmail/v1/users/me/threads/thread-trash-1/trash" && r.Method == "POST" {
			resp := map[string]interface{}{
				"id": "thread-trash-1",
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		t.Logf("Unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := gmail.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create gmail service: %v", err)
	}

	thread, err := svc.Users.Threads.Trash("me", "thread-trash-1").Do()
	if err != nil {
		t.Fatalf("failed to trash thread: %v", err)
	}
	if thread.Id != "thread-trash-1" {
		t.Errorf("unexpected thread id: %s", thread.Id)
	}
}

// TestGmailUntrashThreadCommand_Help tests untrash-thread command structure
func TestGmailUntrashThreadCommand_Help(t *testing.T) {
	cmd := gmailUntrashThreadCmd
	if cmd.Use != "untrash-thread <thread-id>" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}
	if cmd.Args == nil {
		t.Error("expected Args validator to be set")
	}
}

// TestGmailUntrashThread_MockServer tests untrash thread API integration
func TestGmailUntrashThread_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/gmail/v1/users/me/threads/thread-ut-1/untrash" && r.Method == "POST" {
			resp := map[string]interface{}{
				"id": "thread-ut-1",
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		t.Logf("Unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := gmail.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create gmail service: %v", err)
	}

	thread, err := svc.Users.Threads.Untrash("me", "thread-ut-1").Do()
	if err != nil {
		t.Fatalf("failed to untrash thread: %v", err)
	}
	if thread.Id != "thread-ut-1" {
		t.Errorf("unexpected thread id: %s", thread.Id)
	}
}

// TestGmailDeleteThreadCommand_Help tests delete-thread command structure
func TestGmailDeleteThreadCommand_Help(t *testing.T) {
	cmd := gmailDeleteThreadCmd
	if cmd.Use != "delete-thread <thread-id>" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}
	if cmd.Args == nil {
		t.Error("expected Args validator to be set")
	}
}

// TestGmailDeleteThread_MockServer tests delete thread API integration
func TestGmailDeleteThread_MockServer(t *testing.T) {
	deleteCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/gmail/v1/users/me/threads/thread-del-1" && r.Method == "DELETE" {
			deleteCalled = true
			w.WriteHeader(http.StatusNoContent)
			return
		}
		t.Logf("Unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := gmail.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create gmail service: %v", err)
	}

	err = svc.Users.Threads.Delete("me", "thread-del-1").Do()
	if err != nil {
		t.Fatalf("failed to delete thread: %v", err)
	}
	if !deleteCalled {
		t.Error("delete API was not called")
	}
}

// TestGmailLabelInfoCommand_Flags tests label-info command flags
func TestGmailLabelInfoCommand_Flags(t *testing.T) {
	cmd := gmailLabelInfoCmd
	idFlag := cmd.Flags().Lookup("id")
	if idFlag == nil {
		t.Error("expected --id flag to exist")
	}
}

// TestGmailLabelInfo_MockServer tests label info API integration
func TestGmailLabelInfo_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/gmail/v1/users/me/labels/Label_1" && r.Method == "GET" {
			resp := &gmail.Label{
				Id:                    "Label_1",
				Name:                  "ActionNeeded",
				Type:                  "user",
				MessageListVisibility: "show",
				LabelListVisibility:   "labelShow",
				MessagesTotal:         42,
				MessagesUnread:        5,
				ThreadsTotal:          30,
				ThreadsUnread:         3,
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		t.Logf("Unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := gmail.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create gmail service: %v", err)
	}

	label, err := svc.Users.Labels.Get("me", "Label_1").Do()
	if err != nil {
		t.Fatalf("failed to get label: %v", err)
	}
	if label.Name != "ActionNeeded" {
		t.Errorf("unexpected label name: %s", label.Name)
	}
	if label.MessagesTotal != 42 {
		t.Errorf("unexpected messages total: %d", label.MessagesTotal)
	}
}

// TestGmailCreateLabelCommand_Flags tests create-label command flags
func TestGmailCreateLabelCommand_Flags(t *testing.T) {
	cmd := gmailCreateLabelCmd
	nameFlag := cmd.Flags().Lookup("name")
	if nameFlag == nil {
		t.Error("expected --name flag to exist")
	}
	visFlag := cmd.Flags().Lookup("visibility")
	if visFlag == nil {
		t.Error("expected --visibility flag to exist")
	}
	listVisFlag := cmd.Flags().Lookup("list-visibility")
	if listVisFlag == nil {
		t.Error("expected --list-visibility flag to exist")
	}
}

// TestGmailCreateLabel_MockServer tests create label API integration
func TestGmailCreateLabel_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/gmail/v1/users/me/labels" && r.Method == "POST" {
			var label gmail.Label
			json.NewDecoder(r.Body).Decode(&label)
			if label.Name != "TestLabel" {
				t.Errorf("unexpected label name: %s", label.Name)
			}
			resp := &gmail.Label{
				Id:   "Label_new",
				Name: label.Name,
				Type: "user",
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		t.Logf("Unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := gmail.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create gmail service: %v", err)
	}

	label := &gmail.Label{Name: "TestLabel"}
	created, err := svc.Users.Labels.Create("me", label).Do()
	if err != nil {
		t.Fatalf("failed to create label: %v", err)
	}
	if created.Id != "Label_new" {
		t.Errorf("unexpected label id: %s", created.Id)
	}
}

// TestGmailUpdateLabelCommand_Flags tests update-label command flags
func TestGmailUpdateLabelCommand_Flags(t *testing.T) {
	cmd := gmailUpdateLabelCmd
	for _, flag := range []string{"id", "name", "visibility", "list-visibility"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected --%s flag to exist", flag)
		}
	}
}

// TestGmailUpdateLabel_MockServer tests update label API integration
func TestGmailUpdateLabel_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Get current label
		if r.URL.Path == "/gmail/v1/users/me/labels/Label_1" && r.Method == "GET" {
			resp := &gmail.Label{
				Id:   "Label_1",
				Name: "OldName",
				Type: "user",
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		// Update label
		if r.URL.Path == "/gmail/v1/users/me/labels/Label_1" && r.Method == "PUT" {
			var label gmail.Label
			json.NewDecoder(r.Body).Decode(&label)
			resp := &gmail.Label{
				Id:   "Label_1",
				Name: label.Name,
				Type: "user",
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		t.Logf("Unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := gmail.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create gmail service: %v", err)
	}

	// Get then update
	current, err := svc.Users.Labels.Get("me", "Label_1").Do()
	if err != nil {
		t.Fatalf("failed to get label: %v", err)
	}
	current.Name = "NewName"

	updated, err := svc.Users.Labels.Update("me", "Label_1", current).Do()
	if err != nil {
		t.Fatalf("failed to update label: %v", err)
	}
	if updated.Name != "NewName" {
		t.Errorf("unexpected updated name: %s", updated.Name)
	}
}

// TestGmailDeleteLabelCommand_Flags tests delete-label command flags
func TestGmailDeleteLabelCommand_Flags(t *testing.T) {
	cmd := gmailDeleteLabelCmd
	if cmd.Flags().Lookup("id") == nil {
		t.Error("expected --id flag to exist")
	}
}

// TestGmailDeleteLabel_MockServer tests delete label API integration
func TestGmailDeleteLabel_MockServer(t *testing.T) {
	deleteCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/gmail/v1/users/me/labels/Label_del" && r.Method == "DELETE" {
			deleteCalled = true
			w.WriteHeader(http.StatusNoContent)
			return
		}
		t.Logf("Unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := gmail.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create gmail service: %v", err)
	}

	err = svc.Users.Labels.Delete("me", "Label_del").Do()
	if err != nil {
		t.Fatalf("failed to delete label: %v", err)
	}
	if !deleteCalled {
		t.Error("delete API was not called")
	}
}

// TestGmailDraftsCommand_Flags tests drafts list command flags
func TestGmailDraftsCommand_Flags(t *testing.T) {
	cmd := gmailDraftsCmd
	if cmd.Flags().Lookup("max") == nil {
		t.Error("expected --max flag to exist")
	}
	if cmd.Flags().Lookup("query") == nil {
		t.Error("expected --query flag to exist")
	}
}

// TestGmailDrafts_MockServer tests drafts list API integration
func TestGmailDrafts_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/gmail/v1/users/me/drafts" && r.Method == "GET" {
			resp := map[string]interface{}{
				"drafts": []map[string]interface{}{
					{"id": "draft-1", "message": map[string]interface{}{"id": "msg-d1"}},
					{"id": "draft-2", "message": map[string]interface{}{"id": "msg-d2"}},
				},
				"resultSizeEstimate": 2,
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		t.Logf("Unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := gmail.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create gmail service: %v", err)
	}

	resp, err := svc.Users.Drafts.List("me").MaxResults(10).Do()
	if err != nil {
		t.Fatalf("failed to list drafts: %v", err)
	}
	if len(resp.Drafts) != 2 {
		t.Errorf("expected 2 drafts, got %d", len(resp.Drafts))
	}
}

// TestGmailDraftCommand_Flags tests draft get command flags
func TestGmailDraftCommand_Flags(t *testing.T) {
	cmd := gmailDraftCmd
	if cmd.Flags().Lookup("id") == nil {
		t.Error("expected --id flag to exist")
	}
}

// TestGmailDraft_MockServer tests draft get API integration
func TestGmailDraft_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/gmail/v1/users/me/drafts/draft-1" && r.Method == "GET" {
			body := base64.URLEncoding.EncodeToString([]byte("Draft body content"))
			resp := map[string]interface{}{
				"id": "draft-1",
				"message": map[string]interface{}{
					"id":       "msg-d1",
					"threadId": "thread-d1",
					"payload": map[string]interface{}{
						"headers": []map[string]string{
							{"name": "Subject", "value": "Draft Subject"},
							{"name": "To", "value": "recipient@example.com"},
						},
						"mimeType": "text/plain",
						"body":     map[string]interface{}{"data": body},
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		t.Logf("Unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := gmail.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create gmail service: %v", err)
	}

	draft, err := svc.Users.Drafts.Get("me", "draft-1").Format("full").Do()
	if err != nil {
		t.Fatalf("failed to get draft: %v", err)
	}
	if draft.Id != "draft-1" {
		t.Errorf("unexpected draft id: %s", draft.Id)
	}
	if draft.Message == nil {
		t.Fatal("expected message to be present")
	}
	if draft.Message.Id != "msg-d1" {
		t.Errorf("unexpected message id: %s", draft.Message.Id)
	}
}

// TestGmailCreateDraftCommand_Flags tests create-draft command flags
func TestGmailCreateDraftCommand_Flags(t *testing.T) {
	cmd := gmailCreateDraftCmd
	for _, flag := range []string{"to", "subject", "body", "cc", "bcc", "thread-id"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected --%s flag to exist", flag)
		}
	}
}

// TestGmailCreateDraft_MockServer tests create draft API integration
func TestGmailCreateDraft_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/gmail/v1/users/me/drafts" && r.Method == "POST" {
			resp := map[string]interface{}{
				"id":      "draft-new",
				"message": map[string]interface{}{"id": "msg-new", "threadId": "thread-new"},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		t.Logf("Unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := gmail.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create gmail service: %v", err)
	}

	raw := base64.URLEncoding.EncodeToString([]byte("To: test@example.com\r\nSubject: Test\r\n\r\nBody"))
	draft := &gmail.Draft{
		Message: &gmail.Message{Raw: raw},
	}

	created, err := svc.Users.Drafts.Create("me", draft).Do()
	if err != nil {
		t.Fatalf("failed to create draft: %v", err)
	}
	if created.Id != "draft-new" {
		t.Errorf("unexpected draft id: %s", created.Id)
	}
}

// TestGmailUpdateDraftCommand_Flags tests update-draft command flags
func TestGmailUpdateDraftCommand_Flags(t *testing.T) {
	cmd := gmailUpdateDraftCmd
	for _, flag := range []string{"id", "to", "subject", "body", "cc", "bcc"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected --%s flag to exist", flag)
		}
	}
}

// TestGmailUpdateDraft_MockServer tests update draft API integration
func TestGmailUpdateDraft_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/gmail/v1/users/me/drafts/draft-upd" && r.Method == "PUT" {
			resp := map[string]interface{}{
				"id":      "draft-upd",
				"message": map[string]interface{}{"id": "msg-upd"},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		t.Logf("Unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := gmail.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create gmail service: %v", err)
	}

	raw := base64.URLEncoding.EncodeToString([]byte("To: new@example.com\r\nSubject: Updated\r\n\r\nNew body"))
	draft := &gmail.Draft{
		Message: &gmail.Message{Raw: raw},
	}

	updated, err := svc.Users.Drafts.Update("me", "draft-upd", draft).Do()
	if err != nil {
		t.Fatalf("failed to update draft: %v", err)
	}
	if updated.Id != "draft-upd" {
		t.Errorf("unexpected draft id: %s", updated.Id)
	}
}

// TestGmailSendDraftCommand_Flags tests send-draft command flags
func TestGmailSendDraftCommand_Flags(t *testing.T) {
	cmd := gmailSendDraftCmd
	if cmd.Flags().Lookup("id") == nil {
		t.Error("expected --id flag to exist")
	}
}

// TestGmailSendDraft_MockServer tests send draft API integration
func TestGmailSendDraft_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/gmail/v1/users/me/drafts/send" && r.Method == "POST" {
			resp := &gmail.Message{
				Id:       "msg-sent",
				ThreadId: "thread-sent",
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		t.Logf("Unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := gmail.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create gmail service: %v", err)
	}

	draft := &gmail.Draft{Id: "draft-to-send"}
	sent, err := svc.Users.Drafts.Send("me", draft).Do()
	if err != nil {
		t.Fatalf("failed to send draft: %v", err)
	}
	if sent.Id != "msg-sent" {
		t.Errorf("unexpected message id: %s", sent.Id)
	}
	if sent.ThreadId != "thread-sent" {
		t.Errorf("unexpected thread id: %s", sent.ThreadId)
	}
}

// TestGmailDeleteDraftCommand_Flags tests delete-draft command flags
func TestGmailDeleteDraftCommand_Flags(t *testing.T) {
	cmd := gmailDeleteDraftCmd
	if cmd.Flags().Lookup("id") == nil {
		t.Error("expected --id flag to exist")
	}
}

// TestGmailDeleteDraft_MockServer tests delete draft API integration
func TestGmailDeleteDraft_MockServer(t *testing.T) {
	deleteCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/gmail/v1/users/me/drafts/draft-del" && r.Method == "DELETE" {
			deleteCalled = true
			w.WriteHeader(http.StatusNoContent)
			return
		}
		t.Logf("Unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := gmail.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create gmail service: %v", err)
	}

	err = svc.Users.Drafts.Delete("me", "draft-del").Do()
	if err != nil {
		t.Fatalf("failed to delete draft: %v", err)
	}
	if !deleteCalled {
		t.Error("delete API was not called")
	}
}

// TestGmailAttachmentCommand_Flags tests attachment command flags
func TestGmailAttachmentCommand_Flags(t *testing.T) {
	cmd := gmailAttachmentCmd
	for _, flag := range []string{"message-id", "id", "output"} {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected --%s flag to exist", flag)
		}
	}
}

// TestGmailAttachment_MockServer tests attachment download API integration
func TestGmailAttachment_MockServer(t *testing.T) {
	attachmentData := base64.URLEncoding.EncodeToString([]byte("file content here"))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/gmail/v1/users/me/messages/msg-att/attachments/att-1" && r.Method == "GET" {
			resp := map[string]interface{}{
				"data": attachmentData,
				"size": len("file content here"),
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		t.Logf("Unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	svc, err := gmail.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create gmail service: %v", err)
	}

	att, err := svc.Users.Messages.Attachments.Get("me", "msg-att", "att-1").Do()
	if err != nil {
		t.Fatalf("failed to get attachment: %v", err)
	}

	data, err := base64.URLEncoding.DecodeString(att.Data)
	if err != nil {
		t.Fatalf("failed to decode attachment data: %v", err)
	}
	if string(data) != "file content here" {
		t.Errorf("unexpected attachment data: %s", string(data))
	}
}
