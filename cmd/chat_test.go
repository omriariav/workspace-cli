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
	if messagesCmd.Flags().Lookup("filter") == nil {
		t.Error("expected --filter flag on messages")
	}
	if messagesCmd.Flags().Lookup("order-by") == nil {
		t.Error("expected --order-by flag on messages")
	}
	showDeleted := messagesCmd.Flags().Lookup("show-deleted")
	if showDeleted == nil {
		t.Error("expected --show-deleted flag on messages")
	} else if showDeleted.DefValue != "false" {
		t.Errorf("expected --show-deleted default 'false', got '%s'", showDeleted.DefValue)
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

	// Test list command flags
	listCmd := findSubcommand(chatCmd, "list")
	if listCmd == nil {
		t.Fatal("chat list command not found")
	}
	if listCmd.Flags().Lookup("filter") == nil {
		t.Error("expected --filter flag on list")
	}
	pageSizeFlag := listCmd.Flags().Lookup("page-size")
	if pageSizeFlag == nil {
		t.Error("expected --page-size flag on list")
	} else if pageSizeFlag.DefValue != "100" {
		t.Errorf("expected --page-size default '100', got '%s'", pageSizeFlag.DefValue)
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

func TestChatList_WithFilter(t *testing.T) {
	var capturedFilter string
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/spaces": func(w http.ResponseWriter, r *http.Request) {
			capturedFilter = r.URL.Query().Get("filter")
			resp := map[string]interface{}{
				"spaces": []map[string]interface{}{
					{"name": "spaces/AAAA", "displayName": "General", "type": "SPACE"},
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

	filterStr := `spaceType = "SPACE"`
	_, err = svc.Spaces.List().Filter(filterStr).Do()
	if err != nil {
		t.Fatalf("failed to list spaces with filter: %v", err)
	}

	if capturedFilter != filterStr {
		t.Errorf("expected filter '%s', got '%s'", filterStr, capturedFilter)
	}
}

func TestChatList_Pagination(t *testing.T) {
	pagesFetched := 0
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/spaces": func(w http.ResponseWriter, r *http.Request) {
			pagesFetched++
			pageToken := r.URL.Query().Get("pageToken")

			if pageToken == "" {
				resp := map[string]interface{}{
					"spaces": []map[string]interface{}{
						{"name": "spaces/AAAA", "displayName": "Space 1", "type": "ROOM"},
					},
					"nextPageToken": "page2",
				}
				json.NewEncoder(w).Encode(resp)
				return
			}

			resp := map[string]interface{}{
				"spaces": []map[string]interface{}{
					{"name": "spaces/BBBB", "displayName": "Space 2", "type": "ROOM"},
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

	// Simulate pagination loop from runChatList
	var allSpaces []*chat.Space
	var pageToken string
	for {
		call := svc.Spaces.List().PageSize(100)
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}
		resp, err := call.Do()
		if err != nil {
			t.Fatalf("failed to list spaces: %v", err)
		}
		allSpaces = append(allSpaces, resp.Spaces...)
		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}

	if pagesFetched != 2 {
		t.Errorf("expected 2 pages fetched, got %d", pagesFetched)
	}
	if len(allSpaces) != 2 {
		t.Errorf("expected 2 total spaces, got %d", len(allSpaces))
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

func TestChatMessages_WithOrderBy(t *testing.T) {
	var capturedOrderBy string
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/spaces/AAAA/messages": func(w http.ResponseWriter, r *http.Request) {
			capturedOrderBy = r.URL.Query().Get("orderBy")
			resp := map[string]interface{}{
				"messages": []map[string]interface{}{
					{"name": "spaces/AAAA/messages/msg2", "text": "Newer", "createTime": "2026-02-16T10:01:00Z"},
					{"name": "spaces/AAAA/messages/msg1", "text": "Older", "createTime": "2026-02-16T10:00:00Z"},
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

	resp, err := svc.Spaces.Messages.List("spaces/AAAA").OrderBy("createTime DESC").PageSize(25).Do()
	if err != nil {
		t.Fatalf("failed to list messages: %v", err)
	}

	if capturedOrderBy != "createTime DESC" {
		t.Errorf("expected orderBy 'createTime DESC', got '%s'", capturedOrderBy)
	}
	if resp.Messages[0].Text != "Newer" {
		t.Errorf("expected first message 'Newer', got '%s'", resp.Messages[0].Text)
	}
}

func TestChatMessages_WithFilter(t *testing.T) {
	var capturedFilter string
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/spaces/AAAA/messages": func(w http.ResponseWriter, r *http.Request) {
			capturedFilter = r.URL.Query().Get("filter")
			resp := map[string]interface{}{"messages": []map[string]interface{}{}}
			json.NewEncoder(w).Encode(resp)
		},
	}

	server := mockChatServer(t, handlers)
	defer server.Close()

	svc, err := chat.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create chat service: %v", err)
	}

	filterStr := `createTime > "2024-01-01T00:00:00Z"`
	_, err = svc.Spaces.Messages.List("spaces/AAAA").Filter(filterStr).PageSize(25).Do()
	if err != nil {
		t.Fatalf("failed to list messages: %v", err)
	}

	if capturedFilter != filterStr {
		t.Errorf("expected filter '%s', got '%s'", filterStr, capturedFilter)
	}
}

func TestChatMessages_ShowDeleted(t *testing.T) {
	var capturedShowDeleted string
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/spaces/AAAA/messages": func(w http.ResponseWriter, r *http.Request) {
			capturedShowDeleted = r.URL.Query().Get("showDeleted")
			resp := map[string]interface{}{
				"messages": []map[string]interface{}{
					{
						"name":       "spaces/AAAA/messages/msg1",
						"text":       "",
						"createTime": "2026-02-16T10:00:00Z",
						"deleteTime": "2026-02-16T11:00:00Z",
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

	resp, err := svc.Spaces.Messages.List("spaces/AAAA").ShowDeleted(true).PageSize(25).Do()
	if err != nil {
		t.Fatalf("failed to list messages: %v", err)
	}

	if capturedShowDeleted != "true" {
		t.Errorf("expected showDeleted 'true', got '%s'", capturedShowDeleted)
	}
	if resp.Messages[0].DeleteTime != "2026-02-16T11:00:00Z" {
		t.Errorf("expected deleteTime, got '%s'", resp.Messages[0].DeleteTime)
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
	if membersCmd.Flags().Lookup("filter") == nil {
		t.Error("expected --filter flag on members")
	}
	showGroups := membersCmd.Flags().Lookup("show-groups")
	if showGroups == nil {
		t.Error("expected --show-groups flag on members")
	} else if showGroups.DefValue != "false" {
		t.Errorf("expected --show-groups default 'false', got '%s'", showGroups.DefValue)
	}
	showInvited := membersCmd.Flags().Lookup("show-invited")
	if showInvited == nil {
		t.Error("expected --show-invited flag on members")
	} else if showInvited.DefValue != "false" {
		t.Errorf("expected --show-invited default 'false', got '%s'", showInvited.DefValue)
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

func TestChatMembers_WithFilter(t *testing.T) {
	var capturedFilter string
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/spaces/AAAA/members": func(w http.ResponseWriter, r *http.Request) {
			capturedFilter = r.URL.Query().Get("filter")
			resp := map[string]interface{}{
				"memberships": []map[string]interface{}{
					{"name": "spaces/AAAA/members/111", "role": "ROLE_MEMBER", "member": map[string]interface{}{"displayName": "Alice", "name": "users/111", "type": "HUMAN"}},
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

	filterStr := `member.type = "HUMAN"`
	_, err = svc.Spaces.Members.List("spaces/AAAA").Filter(filterStr).PageSize(100).Do()
	if err != nil {
		t.Fatalf("failed to list members: %v", err)
	}

	if capturedFilter != filterStr {
		t.Errorf("expected filter '%s', got '%s'", filterStr, capturedFilter)
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

// --- New command tests ---

func TestChatGetCommand_Flags(t *testing.T) {
	getCmd := findSubcommand(chatCmd, "get")
	if getCmd == nil {
		t.Fatal("chat get command not found")
	}
	if getCmd.Args == nil {
		t.Error("expected Args validator to be set")
	}
	if getCmd.Use != "get <message-name>" {
		t.Errorf("unexpected Use: %s", getCmd.Use)
	}
}

func TestChatGet_MockServer(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/spaces/AAAA/messages/msg1": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "GET" {
				t.Errorf("expected GET, got %s", r.Method)
			}
			resp := map[string]interface{}{
				"name":       "spaces/AAAA/messages/msg1",
				"text":       "Hello world",
				"createTime": "2026-02-16T10:00:00Z",
				"sender":     map[string]interface{}{"displayName": "Alice", "type": "HUMAN", "name": "users/111"},
				"thread":     map[string]interface{}{"name": "spaces/AAAA/threads/thread1"},
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

	msg, err := svc.Spaces.Messages.Get("spaces/AAAA/messages/msg1").Do()
	if err != nil {
		t.Fatalf("failed to get message: %v", err)
	}

	if msg.Name != "spaces/AAAA/messages/msg1" {
		t.Errorf("expected name 'spaces/AAAA/messages/msg1', got '%s'", msg.Name)
	}
	if msg.Text != "Hello world" {
		t.Errorf("expected text 'Hello world', got '%s'", msg.Text)
	}
	if msg.Sender.DisplayName != "Alice" {
		t.Errorf("expected sender 'Alice', got '%s'", msg.Sender.DisplayName)
	}
	if msg.Thread.Name != "spaces/AAAA/threads/thread1" {
		t.Errorf("expected thread name, got '%s'", msg.Thread.Name)
	}
}

func TestChatUpdateCommand_Flags(t *testing.T) {
	updateCmd := findSubcommand(chatCmd, "update")
	if updateCmd == nil {
		t.Fatal("chat update command not found")
	}
	if updateCmd.Args == nil {
		t.Error("expected Args validator to be set")
	}
	textFlag := updateCmd.Flags().Lookup("text")
	if textFlag == nil {
		t.Fatal("expected --text flag on update")
	}
}

func TestChatUpdate_MockServer(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/spaces/AAAA/messages/msg1": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "PATCH" {
				t.Errorf("expected PATCH, got %s", r.Method)
			}

			updateMask := r.URL.Query().Get("updateMask")
			if updateMask != "text" {
				t.Errorf("expected updateMask 'text', got '%s'", updateMask)
			}

			var msg chat.Message
			json.NewDecoder(r.Body).Decode(&msg)

			resp := &chat.Message{
				Name:       "spaces/AAAA/messages/msg1",
				Text:       msg.Text,
				CreateTime: "2026-02-16T10:00:00Z",
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

	msg := &chat.Message{Text: "Updated text"}
	updated, err := svc.Spaces.Messages.Patch("spaces/AAAA/messages/msg1", msg).UpdateMask("text").Do()
	if err != nil {
		t.Fatalf("failed to update message: %v", err)
	}

	if updated.Text != "Updated text" {
		t.Errorf("expected text 'Updated text', got '%s'", updated.Text)
	}
}

func TestChatDeleteCommand_Flags(t *testing.T) {
	deleteCmd := findSubcommand(chatCmd, "delete")
	if deleteCmd == nil {
		t.Fatal("chat delete command not found")
	}
	if deleteCmd.Args == nil {
		t.Error("expected Args validator to be set")
	}
	forceFlag := deleteCmd.Flags().Lookup("force")
	if forceFlag == nil {
		t.Fatal("expected --force flag on delete")
	}
	if forceFlag.DefValue != "false" {
		t.Errorf("expected --force default 'false', got '%s'", forceFlag.DefValue)
	}
}

func TestChatDelete_MockServer(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/spaces/AAAA/messages/msg1": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "DELETE" {
				t.Errorf("expected DELETE, got %s", r.Method)
			}
			json.NewEncoder(w).Encode(map[string]interface{}{})
		},
	}

	server := mockChatServer(t, handlers)
	defer server.Close()

	svc, err := chat.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create chat service: %v", err)
	}

	_, err = svc.Spaces.Messages.Delete("spaces/AAAA/messages/msg1").Do()
	if err != nil {
		t.Fatalf("failed to delete message: %v", err)
	}
}

func TestChatDelete_Force(t *testing.T) {
	var capturedForce string
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/spaces/AAAA/messages/msg1": func(w http.ResponseWriter, r *http.Request) {
			capturedForce = r.URL.Query().Get("force")
			json.NewEncoder(w).Encode(map[string]interface{}{})
		},
	}

	server := mockChatServer(t, handlers)
	defer server.Close()

	svc, err := chat.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create chat service: %v", err)
	}

	_, err = svc.Spaces.Messages.Delete("spaces/AAAA/messages/msg1").Force(true).Do()
	if err != nil {
		t.Fatalf("failed to delete message: %v", err)
	}

	if capturedForce != "true" {
		t.Errorf("expected force 'true', got '%s'", capturedForce)
	}
}

func TestChatReactionsCommand_Flags(t *testing.T) {
	reactionsCmd := findSubcommand(chatCmd, "reactions")
	if reactionsCmd == nil {
		t.Fatal("chat reactions command not found")
	}
	if reactionsCmd.Args == nil {
		t.Error("expected Args validator to be set")
	}
	if reactionsCmd.Flags().Lookup("filter") == nil {
		t.Error("expected --filter flag on reactions")
	}
	pageSizeFlag := reactionsCmd.Flags().Lookup("page-size")
	if pageSizeFlag == nil {
		t.Fatal("expected --page-size flag on reactions")
	}
	if pageSizeFlag.DefValue != "25" {
		t.Errorf("expected --page-size default '25', got '%s'", pageSizeFlag.DefValue)
	}
}

func TestChatReactions_MockServer(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/spaces/AAAA/messages/msg1/reactions": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "GET" {
				t.Errorf("expected GET, got %s", r.Method)
			}
			resp := map[string]interface{}{
				"reactions": []map[string]interface{}{
					{
						"name":  "spaces/AAAA/messages/msg1/reactions/rxn1",
						"emoji": map[string]interface{}{"unicode": "\U0001f44d"},
						"user":  map[string]interface{}{"displayName": "Alice", "name": "users/111"},
					},
					{
						"name":  "spaces/AAAA/messages/msg1/reactions/rxn2",
						"emoji": map[string]interface{}{"unicode": "\u2764\ufe0f"},
						"user":  map[string]interface{}{"displayName": "Bob", "name": "users/222"},
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

	resp, err := svc.Spaces.Messages.Reactions.List("spaces/AAAA/messages/msg1").PageSize(25).Do()
	if err != nil {
		t.Fatalf("failed to list reactions: %v", err)
	}

	if len(resp.Reactions) != 2 {
		t.Fatalf("expected 2 reactions, got %d", len(resp.Reactions))
	}
	if resp.Reactions[0].Emoji.Unicode != "\U0001f44d" {
		t.Errorf("expected thumbs up emoji, got '%s'", resp.Reactions[0].Emoji.Unicode)
	}
	if resp.Reactions[1].User.DisplayName != "Bob" {
		t.Errorf("expected user 'Bob', got '%s'", resp.Reactions[1].User.DisplayName)
	}
}

func TestChatReactCommand_Flags(t *testing.T) {
	reactCmd := findSubcommand(chatCmd, "react")
	if reactCmd == nil {
		t.Fatal("chat react command not found")
	}
	if reactCmd.Args == nil {
		t.Error("expected Args validator to be set")
	}
	emojiFlag := reactCmd.Flags().Lookup("emoji")
	if emojiFlag == nil {
		t.Fatal("expected --emoji flag on react")
	}
}

func TestChatReact_MockServer(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/spaces/AAAA/messages/msg1/reactions": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				t.Errorf("expected POST, got %s", r.Method)
			}

			var reaction chat.Reaction
			json.NewDecoder(r.Body).Decode(&reaction)

			if reaction.Emoji == nil || reaction.Emoji.Unicode != "\U0001f600" {
				t.Errorf("expected emoji unicode '😀', got %+v", reaction.Emoji)
			}

			resp := &chat.Reaction{
				Name:  "spaces/AAAA/messages/msg1/reactions/rxn-new",
				Emoji: reaction.Emoji,
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

	reaction := &chat.Reaction{
		Emoji: &chat.Emoji{Unicode: "\U0001f600"},
	}
	created, err := svc.Spaces.Messages.Reactions.Create("spaces/AAAA/messages/msg1", reaction).Do()
	if err != nil {
		t.Fatalf("failed to create reaction: %v", err)
	}

	if created.Name != "spaces/AAAA/messages/msg1/reactions/rxn-new" {
		t.Errorf("expected reaction name, got '%s'", created.Name)
	}
}

func TestChatUnreactCommand_Flags(t *testing.T) {
	unreactCmd := findSubcommand(chatCmd, "unreact")
	if unreactCmd == nil {
		t.Fatal("chat unreact command not found")
	}
	if unreactCmd.Args == nil {
		t.Error("expected Args validator to be set")
	}
	if unreactCmd.Use != "unreact <reaction-name>" {
		t.Errorf("unexpected Use: %s", unreactCmd.Use)
	}
}

func TestChatUnreact_MockServer(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/spaces/AAAA/messages/msg1/reactions/rxn1": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "DELETE" {
				t.Errorf("expected DELETE, got %s", r.Method)
			}
			json.NewEncoder(w).Encode(map[string]interface{}{})
		},
	}

	server := mockChatServer(t, handlers)
	defer server.Close()

	svc, err := chat.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create chat service: %v", err)
	}

	_, err = svc.Spaces.Messages.Reactions.Delete("spaces/AAAA/messages/msg1/reactions/rxn1").Do()
	if err != nil {
		t.Fatalf("failed to delete reaction: %v", err)
	}
}

func TestEnsureSpaceName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"AAAA", "spaces/AAAA"},
		{"spaces/AAAA", "spaces/AAAA"},
		{"spaces/BBBB", "spaces/BBBB"},
		{"CCCC", "spaces/CCCC"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ensureSpaceName(tt.input)
			if result != tt.expected {
				t.Errorf("ensureSpaceName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
