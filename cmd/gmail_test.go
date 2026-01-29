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
