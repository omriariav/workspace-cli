package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/api/chat/v1"
	"google.golang.org/api/option"
)

func TestChatCommands_Flags(t *testing.T) {
	// Test messages command flags
	messagesCmd := findSubcommand(chatCmd, "messages")
	if messagesCmd == nil {
		t.Fatal("chat messages command not found")
	}
	maxFlag := messagesCmd.Flags().Lookup("max")
	if maxFlag == nil {
		t.Fatal("expected --max flag")
	}
	if maxFlag.DefValue != "25" {
		t.Errorf("expected --max default '25', got '%s'", maxFlag.DefValue)
	}

	// Test send command flags
	sendCmd := findSubcommand(chatCmd, "send")
	if sendCmd == nil {
		t.Fatal("chat send command not found")
	}
	if sendCmd.Flags().Lookup("space") == nil {
		t.Error("expected --space flag")
	}
	if sendCmd.Flags().Lookup("text") == nil {
		t.Error("expected --text flag")
	}
}

func TestChatListCommand_Help(t *testing.T) {
	cmd := chatListCmd
	if cmd.Use != "list" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("expected Short description to be set")
	}
}

func TestChatMessagesCommand_Help(t *testing.T) {
	cmd := chatMessagesCmd
	if cmd.Use != "messages <space-id>" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}
	if cmd.Args == nil {
		t.Error("expected Args validator to be set")
	}
}

func TestChatSendCommand_Help(t *testing.T) {
	cmd := chatSendCmd
	if cmd.Use != "send" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("expected Short description to be set")
	}
}

// mockChatServer creates a test server that mocks Chat API responses
func mockChatServer(t *testing.T, handlers map[string]func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		for pattern, handler := range handlers {
			if r.URL.Path == pattern {
				handler(w, r)
				return
			}
		}

		t.Logf("Unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
}

func TestChatList_MockServer(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/spaces": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "GET" {
				t.Errorf("expected GET, got %s", r.Method)
			}
			resp := map[string]interface{}{
				"spaces": []map[string]interface{}{
					{
						"name":        "spaces/AAAA",
						"displayName": "General",
						"type":        "ROOM",
					},
					{
						"name":        "spaces/BBBB",
						"displayName": "Engineering",
						"type":        "ROOM",
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		},
	}

	server := mockChatServer(t, handlers)
	defer server.Close()

	svc, err := chat.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create chat service: %v", err)
	}

	resp, err := svc.Spaces.List().Do()
	if err != nil {
		t.Fatalf("failed to list spaces: %v", err)
	}

	if len(resp.Spaces) != 2 {
		t.Fatalf("expected 2 spaces, got %d", len(resp.Spaces))
	}
	if resp.Spaces[0].DisplayName != "General" {
		t.Errorf("expected first space 'General', got '%s'", resp.Spaces[0].DisplayName)
	}
	if resp.Spaces[1].Name != "spaces/BBBB" {
		t.Errorf("expected second space name 'spaces/BBBB', got '%s'", resp.Spaces[1].Name)
	}
}

func TestChatMessages_MockServer(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/spaces/AAAA/messages": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "GET" {
				t.Errorf("expected GET, got %s", r.Method)
			}
			resp := map[string]interface{}{
				"messages": []map[string]interface{}{
					{
						"name":       "spaces/AAAA/messages/msg1",
						"text":       "Hello world",
						"createTime": "2026-02-16T10:00:00Z",
						"sender":     map[string]interface{}{"displayName": "Alice"},
					},
					{
						"name":       "spaces/AAAA/messages/msg2",
						"text":       "Hi there",
						"createTime": "2026-02-16T10:01:00Z",
						"sender":     map[string]interface{}{"displayName": "Bob"},
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		},
	}

	server := mockChatServer(t, handlers)
	defer server.Close()

	svc, err := chat.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create chat service: %v", err)
	}

	resp, err := svc.Spaces.Messages.List("spaces/AAAA").PageSize(25).Do()
	if err != nil {
		t.Fatalf("failed to list messages: %v", err)
	}

	if len(resp.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(resp.Messages))
	}
	if resp.Messages[0].Text != "Hello world" {
		t.Errorf("expected first message 'Hello world', got '%s'", resp.Messages[0].Text)
	}
	if resp.Messages[1].Sender.DisplayName != "Bob" {
		t.Errorf("expected second sender 'Bob', got '%s'", resp.Messages[1].Sender.DisplayName)
	}
}

func TestChatSend_MockServer(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/spaces/AAAA/messages": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				t.Errorf("expected POST, got %s", r.Method)
			}

			var msg chat.Message
			json.NewDecoder(r.Body).Decode(&msg)

			if msg.Text != "Test message" {
				t.Errorf("expected text 'Test message', got '%s'", msg.Text)
			}

			resp := &chat.Message{
				Name:       "spaces/AAAA/messages/msg-new",
				Text:       msg.Text,
				CreateTime: "2026-02-16T10:05:00Z",
			}
			json.NewEncoder(w).Encode(resp)
		},
	}

	server := mockChatServer(t, handlers)
	defer server.Close()

	svc, err := chat.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create chat service: %v", err)
	}

	msg := &chat.Message{Text: "Test message"}
	sent, err := svc.Spaces.Messages.Create("spaces/AAAA", msg).Do()
	if err != nil {
		t.Fatalf("failed to send message: %v", err)
	}

	if sent.Name != "spaces/AAAA/messages/msg-new" {
		t.Errorf("expected message name 'spaces/AAAA/messages/msg-new', got '%s'", sent.Name)
	}
	if sent.Text != "Test message" {
		t.Errorf("expected text 'Test message', got '%s'", sent.Text)
	}
}
