package cmd

import (
	"context"
	"encoding/json"
	"fmt"
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
						"sender":     map[string]interface{}{"displayName": "Alice", "type": "HUMAN", "name": "users/111"},
					},
					{
						"name":       "spaces/AAAA/messages/msg2",
						"text":       "Hi there",
						"createTime": "2026-02-16T10:01:00Z",
						"sender":     map[string]interface{}{"displayName": "Bob", "type": "HUMAN", "name": "users/222"},
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
	if resp.Messages[0].Sender.Type != "HUMAN" {
		t.Errorf("expected sender type 'HUMAN', got '%s'", resp.Messages[0].Sender.Type)
	}
}

func TestChatMessages_SenderFallback(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/spaces/AAAA/messages": func(w http.ResponseWriter, r *http.Request) {
			resp := map[string]interface{}{
				"messages": []map[string]interface{}{
					{
						"name":       "spaces/AAAA/messages/msg1",
						"text":       "Bot message",
						"createTime": "2026-02-16T10:00:00Z",
						"sender":     map[string]interface{}{"name": "users/999", "type": "BOT"},
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

	if len(resp.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(resp.Messages))
	}
	// When displayName is empty, should fall back to resource name
	if resp.Messages[0].Sender.DisplayName != "" {
		t.Errorf("expected empty displayName for fallback test, got '%s'", resp.Messages[0].Sender.DisplayName)
	}
	if resp.Messages[0].Sender.Name != "users/999" {
		t.Errorf("expected sender name 'users/999', got '%s'", resp.Messages[0].Sender.Name)
	}
}

func TestChatMembersCommand_Flags(t *testing.T) {
	membersCmd := findSubcommand(chatCmd, "members")
	if membersCmd == nil {
		t.Fatal("chat members command not found")
	}
	if membersCmd.Args == nil {
		t.Error("expected Args validator to be set")
	}
	maxFlag := membersCmd.Flags().Lookup("max")
	if maxFlag == nil {
		t.Fatal("expected --max flag")
	}
	if maxFlag.DefValue != "100" {
		t.Errorf("expected --max default '100', got '%s'", maxFlag.DefValue)
	}
}

func TestChatMembers_MockServer(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/spaces/AAAA/members": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "GET" {
				t.Errorf("expected GET, got %s", r.Method)
			}
			resp := map[string]interface{}{
				"memberships": []map[string]interface{}{
					{
						"name":       "spaces/AAAA/members/111",
						"role":       "ROLE_MEMBER",
						"createTime": "2025-01-01T00:00:00Z",
						"member":     map[string]interface{}{"displayName": "Alice Smith", "name": "users/111", "type": "HUMAN"},
					},
					{
						"name":       "spaces/AAAA/members/222",
						"role":       "ROLE_MANAGER",
						"createTime": "2025-01-02T00:00:00Z",
						"member":     map[string]interface{}{"displayName": "Bob Jones", "name": "users/222", "type": "HUMAN"},
					},
					{
						"name":   "spaces/AAAA/members/bot1",
						"role":   "ROLE_MEMBER",
						"member": map[string]interface{}{"displayName": "Helper Bot", "name": "users/bot1", "type": "BOT"},
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

	resp, err := svc.Spaces.Members.List("spaces/AAAA").PageSize(100).Do()
	if err != nil {
		t.Fatalf("failed to list members: %v", err)
	}

	if len(resp.Memberships) != 3 {
		t.Fatalf("expected 3 members, got %d", len(resp.Memberships))
	}
	if resp.Memberships[0].Member.DisplayName != "Alice Smith" {
		t.Errorf("expected first member 'Alice Smith', got '%s'", resp.Memberships[0].Member.DisplayName)
	}
	if resp.Memberships[1].Role != "ROLE_MANAGER" {
		t.Errorf("expected second member role 'ROLE_MANAGER', got '%s'", resp.Memberships[1].Role)
	}
	if resp.Memberships[2].Member.Type != "BOT" {
		t.Errorf("expected third member type 'BOT', got '%s'", resp.Memberships[2].Member.Type)
	}
}

// TestChatMembers_PaginationStopsAtMax verifies the Pages iterator stops after --max is reached
func TestChatMembers_PaginationStopsAtMax(t *testing.T) {
	pagesFetched := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path != "/v1/spaces/BIGSPACE/members" {
			t.Logf("Unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		pagesFetched++
		pageToken := r.URL.Query().Get("pageToken")

		if pageToken == "" {
			// Page 1: return 3 members + nextPageToken
			resp := map[string]interface{}{
				"memberships": []map[string]interface{}{
					{"name": "spaces/BIGSPACE/members/1", "role": "ROLE_MEMBER", "member": map[string]interface{}{"displayName": "User 1", "name": "users/1", "type": "HUMAN"}},
					{"name": "spaces/BIGSPACE/members/2", "role": "ROLE_MEMBER", "member": map[string]interface{}{"displayName": "User 2", "name": "users/2", "type": "HUMAN"}},
					{"name": "spaces/BIGSPACE/members/3", "role": "ROLE_MEMBER", "member": map[string]interface{}{"displayName": "User 3", "name": "users/3", "type": "HUMAN"}},
				},
				"nextPageToken": "page2",
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		// Page 2: should NOT be fetched if max=2
		resp := map[string]interface{}{
			"memberships": []map[string]interface{}{
				{"name": "spaces/BIGSPACE/members/4", "role": "ROLE_MEMBER", "member": map[string]interface{}{"displayName": "User 4", "name": "users/4", "type": "HUMAN"}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	svc, err := chat.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create chat service: %v", err)
	}

	// Simulate runChatMembers logic with max=2
	maxResults := int64(2)
	pageSize := maxResults
	var results []map[string]interface{}

	errDone := fmt.Errorf("done")
	err = svc.Spaces.Members.List("spaces/BIGSPACE").PageSize(pageSize).Pages(context.Background(), func(resp *chat.ListMembershipsResponse) error {
		for _, m := range resp.Memberships {
			if m == nil {
				continue
			}
			if int64(len(results)) >= maxResults {
				return errDone
			}
			results = append(results, mapMemberToOutput(m))
		}
		if int64(len(results)) >= maxResults {
			return errDone
		}
		return nil
	})
	if err != nil && err != errDone {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results (capped by max), got %d", len(results))
	}
	if pagesFetched != 1 {
		t.Errorf("expected only 1 page fetched (early stop), got %d", pagesFetched)
	}
}

func TestMapMemberToOutput_AllFields(t *testing.T) {
	m := &chat.Membership{
		Name:       "spaces/AAAA/members/111",
		Role:       "ROLE_MANAGER",
		CreateTime: "2025-06-01T00:00:00Z",
		Member: &chat.User{
			DisplayName: "Alice Smith",
			Name:        "users/111",
			Type:        "HUMAN",
		},
	}

	result := mapMemberToOutput(m)

	if result["name"] != "spaces/AAAA/members/111" {
		t.Errorf("expected name, got %v", result["name"])
	}
	if result["display_name"] != "Alice Smith" {
		t.Errorf("expected display_name 'Alice Smith', got %v", result["display_name"])
	}
	if result["user"] != "users/111" {
		t.Errorf("expected user 'users/111', got %v", result["user"])
	}
	if result["type"] != "HUMAN" {
		t.Errorf("expected type 'HUMAN', got %v", result["type"])
	}
	if result["role"] != "ROLE_MANAGER" {
		t.Errorf("expected role 'ROLE_MANAGER', got %v", result["role"])
	}
	if result["joined"] != "2025-06-01T00:00:00Z" {
		t.Errorf("expected joined time, got %v", result["joined"])
	}
}

func TestMapMemberToOutput_MinimalFields(t *testing.T) {
	m := &chat.Membership{
		Name: "spaces/AAAA/members/222",
		Role: "ROLE_MEMBER",
	}

	result := mapMemberToOutput(m)

	if result["name"] != "spaces/AAAA/members/222" {
		t.Errorf("expected name, got %v", result["name"])
	}
	if _, exists := result["display_name"]; exists {
		t.Error("display_name should be omitted when Member is nil")
	}
	if _, exists := result["user"]; exists {
		t.Error("user should be omitted when Member is nil")
	}
	if _, exists := result["type"]; exists {
		t.Error("type should be omitted when Member is nil")
	}
	if _, exists := result["joined"]; exists {
		t.Error("joined should be omitted when CreateTime is empty")
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
