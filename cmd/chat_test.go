package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/omriariav/workspace-cli/internal/spacecache"
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
	listMaxFlag := listCmd.Flags().Lookup("max")
	if listMaxFlag == nil {
		t.Error("expected --max flag on list")
	} else if listMaxFlag.DefValue != "0" {
		t.Errorf("expected --max default '0', got '%s'", listMaxFlag.DefValue)
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

func TestChatMessagesCommand_AfterBeforeFlags(t *testing.T) {
	messagesCmd := findSubcommand(chatCmd, "messages")
	if messagesCmd == nil {
		t.Fatal("chat messages command not found")
	}
	if messagesCmd.Flags().Lookup("after") == nil {
		t.Error("expected --after flag on messages")
	}
	if messagesCmd.Flags().Lookup("before") == nil {
		t.Error("expected --before flag on messages")
	}
}

func TestChatMessages_AfterBeforeFilter(t *testing.T) {
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

	// Test after + before combined filter
	filterStr := `createTime > "2026-02-17T00:00:00Z" AND createTime < "2026-02-20T00:00:00Z"`
	_, err = svc.Spaces.Messages.List("spaces/AAAA").Filter(filterStr).PageSize(25).Do()
	if err != nil {
		t.Fatalf("failed to list messages: %v", err)
	}

	if capturedFilter != filterStr {
		t.Errorf("expected filter %q, got %q", filterStr, capturedFilter)
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
	reactionsMaxFlag := reactionsCmd.Flags().Lookup("max")
	if reactionsMaxFlag == nil {
		t.Error("expected --max flag on reactions")
	} else if reactionsMaxFlag.DefValue != "0" {
		t.Errorf("expected --max default '0', got '%s'", reactionsMaxFlag.DefValue)
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

// --- Helper tests ---

func TestEnsureReadStateName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"AAAA", "users/me/spaces/AAAA/spaceReadState"},
		{"spaces/AAAA", "users/me/spaces/AAAA/spaceReadState"},
		{"users/me/spaces/AAAA/spaceReadState", "users/me/spaces/AAAA/spaceReadState"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ensureReadStateName(tt.input)
			if result != tt.expected {
				t.Errorf("ensureReadStateName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// --- Spaces CRUD flag tests ---

func TestChatGetSpaceCommand_Flags(t *testing.T) {
	cmd := findSubcommand(chatCmd, "get-space")
	if cmd == nil {
		t.Fatal("chat get-space command not found")
	}
	if cmd.Args == nil {
		t.Error("expected Args validator to be set")
	}
	if cmd.Use != "get-space <space>" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}
}

func TestChatCreateSpaceCommand_Flags(t *testing.T) {
	cmd := findSubcommand(chatCmd, "create-space")
	if cmd == nil {
		t.Fatal("chat create-space command not found")
	}
	if cmd.Flags().Lookup("display-name") == nil {
		t.Error("expected --display-name flag")
	}
	typeFlag := cmd.Flags().Lookup("type")
	if typeFlag == nil {
		t.Error("expected --type flag")
	} else if typeFlag.DefValue != "SPACE" {
		t.Errorf("expected --type default 'SPACE', got '%s'", typeFlag.DefValue)
	}
	if cmd.Flags().Lookup("description") == nil {
		t.Error("expected --description flag")
	}
}

func TestChatDeleteSpaceCommand_Flags(t *testing.T) {
	cmd := findSubcommand(chatCmd, "delete-space")
	if cmd == nil {
		t.Fatal("chat delete-space command not found")
	}
	if cmd.Args == nil {
		t.Error("expected Args validator to be set")
	}
}

func TestChatUpdateSpaceCommand_Flags(t *testing.T) {
	cmd := findSubcommand(chatCmd, "update-space")
	if cmd == nil {
		t.Fatal("chat update-space command not found")
	}
	if cmd.Args == nil {
		t.Error("expected Args validator to be set")
	}
	if cmd.Flags().Lookup("display-name") == nil {
		t.Error("expected --display-name flag")
	}
	if cmd.Flags().Lookup("description") == nil {
		t.Error("expected --description flag")
	}
}

func TestChatSearchSpacesCommand_Flags(t *testing.T) {
	cmd := findSubcommand(chatCmd, "search-spaces")
	if cmd == nil {
		t.Fatal("chat search-spaces command not found")
	}
	if cmd.Flags().Lookup("query") == nil {
		t.Error("expected --query flag")
	}
	pageSizeFlag := cmd.Flags().Lookup("page-size")
	if pageSizeFlag == nil {
		t.Error("expected --page-size flag")
	} else if pageSizeFlag.DefValue != "100" {
		t.Errorf("expected --page-size default '100', got '%s'", pageSizeFlag.DefValue)
	}
	searchMaxFlag := cmd.Flags().Lookup("max")
	if searchMaxFlag == nil {
		t.Error("expected --max flag on search-spaces")
	} else if searchMaxFlag.DefValue != "0" {
		t.Errorf("expected --max default '0', got '%s'", searchMaxFlag.DefValue)
	}
}

func TestChatFindDmCommand_Flags(t *testing.T) {
	cmd := findSubcommand(chatCmd, "find-dm")
	if cmd == nil {
		t.Fatal("chat find-dm command not found")
	}
	if cmd.Flags().Lookup("user") == nil {
		t.Error("expected --user flag")
	}
}

func TestChatSetupSpaceCommand_Flags(t *testing.T) {
	cmd := findSubcommand(chatCmd, "setup-space")
	if cmd == nil {
		t.Fatal("chat setup-space command not found")
	}
	if cmd.Flags().Lookup("display-name") == nil {
		t.Error("expected --display-name flag")
	}
	if cmd.Flags().Lookup("members") == nil {
		t.Error("expected --members flag")
	}
}

// --- Member management flag tests ---

func TestChatGetMemberCommand_Flags(t *testing.T) {
	cmd := findSubcommand(chatCmd, "get-member")
	if cmd == nil {
		t.Fatal("chat get-member command not found")
	}
	if cmd.Args == nil {
		t.Error("expected Args validator to be set")
	}
}

func TestChatAddMemberCommand_Flags(t *testing.T) {
	cmd := findSubcommand(chatCmd, "add-member")
	if cmd == nil {
		t.Fatal("chat add-member command not found")
	}
	if cmd.Args == nil {
		t.Error("expected Args validator to be set")
	}
	if cmd.Flags().Lookup("user") == nil {
		t.Error("expected --user flag")
	}
	roleFlag := cmd.Flags().Lookup("role")
	if roleFlag == nil {
		t.Error("expected --role flag")
	} else if roleFlag.DefValue != "ROLE_MEMBER" {
		t.Errorf("expected --role default 'ROLE_MEMBER', got '%s'", roleFlag.DefValue)
	}
}

func TestChatRemoveMemberCommand_Flags(t *testing.T) {
	cmd := findSubcommand(chatCmd, "remove-member")
	if cmd == nil {
		t.Fatal("chat remove-member command not found")
	}
	if cmd.Args == nil {
		t.Error("expected Args validator to be set")
	}
}

func TestChatUpdateMemberCommand_Flags(t *testing.T) {
	cmd := findSubcommand(chatCmd, "update-member")
	if cmd == nil {
		t.Fatal("chat update-member command not found")
	}
	if cmd.Args == nil {
		t.Error("expected Args validator to be set")
	}
	if cmd.Flags().Lookup("role") == nil {
		t.Error("expected --role flag")
	}
}

// --- Read state flag tests ---

func TestChatReadStateCommand_Flags(t *testing.T) {
	cmd := findSubcommand(chatCmd, "read-state")
	if cmd == nil {
		t.Fatal("chat read-state command not found")
	}
	if cmd.Args == nil {
		t.Error("expected Args validator to be set")
	}
}

func TestChatMarkReadCommand_Flags(t *testing.T) {
	cmd := findSubcommand(chatCmd, "mark-read")
	if cmd == nil {
		t.Fatal("chat mark-read command not found")
	}
	if cmd.Args == nil {
		t.Error("expected Args validator to be set")
	}
	if cmd.Flags().Lookup("time") == nil {
		t.Error("expected --time flag")
	}
}

func TestChatThreadReadStateCommand_Flags(t *testing.T) {
	cmd := findSubcommand(chatCmd, "thread-read-state")
	if cmd == nil {
		t.Fatal("chat thread-read-state command not found")
	}
	if cmd.Args == nil {
		t.Error("expected Args validator to be set")
	}
}

// --- Attachment/Media/Events flag tests ---

func TestChatAttachmentCommand_Flags(t *testing.T) {
	cmd := findSubcommand(chatCmd, "attachment")
	if cmd == nil {
		t.Fatal("chat attachment command not found")
	}
	if cmd.Args == nil {
		t.Error("expected Args validator to be set")
	}
}

func TestChatUploadCommand_Flags(t *testing.T) {
	cmd := findSubcommand(chatCmd, "upload")
	if cmd == nil {
		t.Fatal("chat upload command not found")
	}
	if cmd.Args == nil {
		t.Error("expected Args validator to be set")
	}
	if cmd.Flags().Lookup("file") == nil {
		t.Error("expected --file flag")
	}
}

func TestChatDownloadCommand_Flags(t *testing.T) {
	cmd := findSubcommand(chatCmd, "download")
	if cmd == nil {
		t.Fatal("chat download command not found")
	}
	if cmd.Args == nil {
		t.Error("expected Args validator to be set")
	}
	if cmd.Flags().Lookup("output") == nil {
		t.Error("expected --output flag")
	}
}

func TestChatEventsCommand_Flags(t *testing.T) {
	cmd := findSubcommand(chatCmd, "events")
	if cmd == nil {
		t.Fatal("chat events command not found")
	}
	if cmd.Args == nil {
		t.Error("expected Args validator to be set")
	}
	if cmd.Flags().Lookup("filter") == nil {
		t.Error("expected --filter flag")
	}
	pageSizeFlag := cmd.Flags().Lookup("page-size")
	if pageSizeFlag == nil {
		t.Error("expected --page-size flag")
	} else if pageSizeFlag.DefValue != "100" {
		t.Errorf("expected --page-size default '100', got '%s'", pageSizeFlag.DefValue)
	}
	eventsMaxFlag := cmd.Flags().Lookup("max")
	if eventsMaxFlag == nil {
		t.Error("expected --max flag on events")
	} else if eventsMaxFlag.DefValue != "0" {
		t.Errorf("expected --max default '0', got '%s'", eventsMaxFlag.DefValue)
	}
}

func TestChatEventCommand_Flags(t *testing.T) {
	cmd := findSubcommand(chatCmd, "event")
	if cmd == nil {
		t.Fatal("chat event command not found")
	}
	if cmd.Args == nil {
		t.Error("expected Args validator to be set")
	}
}

// --- Spaces CRUD mock server tests ---

func TestChatGetSpace_MockServer(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/spaces/AAAA": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "GET" {
				t.Errorf("expected GET, got %s", r.Method)
			}
			resp := map[string]interface{}{
				"name":        "spaces/AAAA",
				"displayName": "General",
				"spaceType":   "SPACE",
				"createTime":  "2025-01-01T00:00:00Z",
				"spaceDetails": map[string]interface{}{
					"description": "General discussion",
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

	space, err := svc.Spaces.Get("spaces/AAAA").Do()
	if err != nil {
		t.Fatalf("failed to get space: %v", err)
	}

	if space.Name != "spaces/AAAA" {
		t.Errorf("expected name 'spaces/AAAA', got '%s'", space.Name)
	}
	if space.DisplayName != "General" {
		t.Errorf("expected displayName 'General', got '%s'", space.DisplayName)
	}
	if space.SpaceType != "SPACE" {
		t.Errorf("expected spaceType 'SPACE', got '%s'", space.SpaceType)
	}
	if space.SpaceDetails == nil || space.SpaceDetails.Description != "General discussion" {
		t.Errorf("expected description 'General discussion'")
	}
}

func TestChatCreateSpace_MockServer(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/spaces": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				t.Errorf("expected POST, got %s", r.Method)
			}

			var space chat.Space
			json.NewDecoder(r.Body).Decode(&space)

			if space.DisplayName != "New Space" {
				t.Errorf("expected displayName 'New Space', got '%s'", space.DisplayName)
			}
			if space.SpaceType != "SPACE" {
				t.Errorf("expected spaceType 'SPACE', got '%s'", space.SpaceType)
			}

			resp := &chat.Space{
				Name:        "spaces/NEWSPACE",
				DisplayName: space.DisplayName,
				SpaceType:   space.SpaceType,
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

	space := &chat.Space{DisplayName: "New Space", SpaceType: "SPACE"}
	created, err := svc.Spaces.Create(space).Do()
	if err != nil {
		t.Fatalf("failed to create space: %v", err)
	}

	if created.Name != "spaces/NEWSPACE" {
		t.Errorf("expected name 'spaces/NEWSPACE', got '%s'", created.Name)
	}
}

func TestChatDeleteSpace_MockServer(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/spaces/AAAA": func(w http.ResponseWriter, r *http.Request) {
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

	_, err = svc.Spaces.Delete("spaces/AAAA").Do()
	if err != nil {
		t.Fatalf("failed to delete space: %v", err)
	}
}

func TestChatUpdateSpace_MockServer(t *testing.T) {
	var capturedUpdateMask string
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/spaces/AAAA": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "PATCH" {
				t.Errorf("expected PATCH, got %s", r.Method)
			}
			capturedUpdateMask = r.URL.Query().Get("updateMask")

			var space chat.Space
			json.NewDecoder(r.Body).Decode(&space)

			resp := &chat.Space{
				Name:        "spaces/AAAA",
				DisplayName: space.DisplayName,
				SpaceType:   "SPACE",
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

	space := &chat.Space{DisplayName: "Renamed"}
	updated, err := svc.Spaces.Patch("spaces/AAAA", space).UpdateMask("display_name").Do()
	if err != nil {
		t.Fatalf("failed to update space: %v", err)
	}

	if capturedUpdateMask != "display_name" {
		t.Errorf("expected updateMask 'display_name', got '%s'", capturedUpdateMask)
	}
	if updated.DisplayName != "Renamed" {
		t.Errorf("expected displayName 'Renamed', got '%s'", updated.DisplayName)
	}
}

func TestChatSearchSpaces_MockServer(t *testing.T) {
	var capturedQuery string
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/spaces:search": func(w http.ResponseWriter, r *http.Request) {
			capturedQuery = r.URL.Query().Get("query")
			resp := map[string]interface{}{
				"spaces": []map[string]interface{}{
					{"name": "spaces/AAAA", "displayName": "Test Space", "spaceType": "SPACE"},
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

	resp, err := svc.Spaces.Search().Query("Test").PageSize(10).Do()
	if err != nil {
		t.Fatalf("failed to search spaces: %v", err)
	}

	if capturedQuery != "Test" {
		t.Errorf("expected query 'Test', got '%s'", capturedQuery)
	}
	if len(resp.Spaces) != 1 {
		t.Fatalf("expected 1 space, got %d", len(resp.Spaces))
	}
	if resp.Spaces[0].DisplayName != "Test Space" {
		t.Errorf("expected 'Test Space', got '%s'", resp.Spaces[0].DisplayName)
	}
}

func TestChatSearchSpaces_Pagination(t *testing.T) {
	pagesFetched := 0
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/spaces:search": func(w http.ResponseWriter, r *http.Request) {
			pagesFetched++
			pageToken := r.URL.Query().Get("pageToken")

			if pageToken == "" {
				resp := map[string]interface{}{
					"spaces":        []map[string]interface{}{{"name": "spaces/AAAA", "displayName": "Space 1", "spaceType": "SPACE"}},
					"nextPageToken": "page2",
				}
				json.NewEncoder(w).Encode(resp)
				return
			}

			resp := map[string]interface{}{
				"spaces": []map[string]interface{}{{"name": "spaces/BBBB", "displayName": "Space 2", "spaceType": "SPACE"}},
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

	var allSpaces []*chat.Space
	var pageToken string
	for {
		call := svc.Spaces.Search().Query("test").PageSize(10)
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}
		resp, err := call.Do()
		if err != nil {
			t.Fatalf("failed to search spaces: %v", err)
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

func TestChatFindDm_MockServer(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/spaces:findDirectMessage": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "GET" {
				t.Errorf("expected GET, got %s", r.Method)
			}
			resp := map[string]interface{}{
				"name":      "spaces/DM123",
				"spaceType": "DIRECT_MESSAGE",
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

	space, err := svc.Spaces.FindDirectMessage().Name("users/123").Do()
	if err != nil {
		t.Fatalf("failed to find DM: %v", err)
	}

	if space.Name != "spaces/DM123" {
		t.Errorf("expected name 'spaces/DM123', got '%s'", space.Name)
	}
}

func TestChatSetupSpace_MockServer(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/spaces:setup": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				t.Errorf("expected POST, got %s", r.Method)
			}

			var req chat.SetUpSpaceRequest
			json.NewDecoder(r.Body).Decode(&req)

			if req.Space == nil || req.Space.DisplayName != "Team Space" {
				t.Errorf("expected displayName 'Team Space'")
			}
			if len(req.Memberships) != 2 {
				t.Errorf("expected 2 memberships, got %d", len(req.Memberships))
			}

			resp := &chat.Space{
				Name:        "spaces/SETUP123",
				DisplayName: req.Space.DisplayName,
				SpaceType:   "SPACE",
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

	req := &chat.SetUpSpaceRequest{
		Space: &chat.Space{DisplayName: "Team Space", SpaceType: "SPACE"},
		Memberships: []*chat.Membership{
			{Member: &chat.User{Name: "users/111", Type: "HUMAN"}},
			{Member: &chat.User{Name: "users/222", Type: "HUMAN"}},
		},
	}

	space, err := svc.Spaces.Setup(req).Do()
	if err != nil {
		t.Fatalf("failed to setup space: %v", err)
	}

	if space.Name != "spaces/SETUP123" {
		t.Errorf("expected name 'spaces/SETUP123', got '%s'", space.Name)
	}
}

func TestChatSetupSpace_DM_NoDisplayName(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/spaces:setup": func(w http.ResponseWriter, r *http.Request) {
			var req chat.SetUpSpaceRequest
			json.NewDecoder(r.Body).Decode(&req)

			// DM should not have displayName
			if req.Space.DisplayName != "" {
				t.Errorf("expected empty displayName for DM, got %q", req.Space.DisplayName)
			}
			if req.Space.SpaceType != "DIRECT_MESSAGE" {
				t.Errorf("expected DIRECT_MESSAGE type, got %q", req.Space.SpaceType)
			}
			if len(req.Memberships) != 1 {
				t.Errorf("expected 1 membership for DM, got %d", len(req.Memberships))
			}

			resp := &chat.Space{
				Name:      "spaces/DM123",
				SpaceType: "DIRECT_MESSAGE",
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

	req := &chat.SetUpSpaceRequest{
		Space: &chat.Space{SpaceType: "DIRECT_MESSAGE"},
		Memberships: []*chat.Membership{
			{Member: &chat.User{Name: "users/111", Type: "HUMAN"}},
		},
	}

	space, err := svc.Spaces.Setup(req).Do()
	if err != nil {
		t.Fatalf("failed to setup DM space: %v", err)
	}

	if space.Name != "spaces/DM123" {
		t.Errorf("expected name 'spaces/DM123', got '%s'", space.Name)
	}
}

func TestChatSetupSpace_GroupChat_NoDisplayName(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/spaces:setup": func(w http.ResponseWriter, r *http.Request) {
			var req chat.SetUpSpaceRequest
			json.NewDecoder(r.Body).Decode(&req)

			if req.Space.DisplayName != "" {
				t.Errorf("expected empty displayName for GROUP_CHAT, got %q", req.Space.DisplayName)
			}
			if req.Space.SpaceType != "GROUP_CHAT" {
				t.Errorf("expected GROUP_CHAT type, got %q", req.Space.SpaceType)
			}
			if len(req.Memberships) < 2 {
				t.Errorf("expected at least 2 memberships for GROUP_CHAT, got %d", len(req.Memberships))
			}

			resp := &chat.Space{
				Name:      "spaces/GC123",
				SpaceType: "GROUP_CHAT",
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

	req := &chat.SetUpSpaceRequest{
		Space: &chat.Space{SpaceType: "GROUP_CHAT"},
		Memberships: []*chat.Membership{
			{Member: &chat.User{Name: "users/111", Type: "HUMAN"}},
			{Member: &chat.User{Name: "users/222", Type: "HUMAN"}},
		},
	}

	space, err := svc.Spaces.Setup(req).Do()
	if err != nil {
		t.Fatalf("failed to setup group chat: %v", err)
	}

	if space.Name != "spaces/GC123" {
		t.Errorf("expected name 'spaces/GC123', got '%s'", space.Name)
	}
}

// --- Member management mock server tests ---

func TestChatGetMember_MockServer(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/spaces/AAAA/members/111": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "GET" {
				t.Errorf("expected GET, got %s", r.Method)
			}
			resp := map[string]interface{}{
				"name":       "spaces/AAAA/members/111",
				"role":       "ROLE_MANAGER",
				"createTime": "2025-01-01T00:00:00Z",
				"member":     map[string]interface{}{"displayName": "Alice", "name": "users/111", "type": "HUMAN"},
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

	member, err := svc.Spaces.Members.Get("spaces/AAAA/members/111").Do()
	if err != nil {
		t.Fatalf("failed to get member: %v", err)
	}

	if member.Name != "spaces/AAAA/members/111" {
		t.Errorf("expected name, got '%s'", member.Name)
	}
	if member.Role != "ROLE_MANAGER" {
		t.Errorf("expected role 'ROLE_MANAGER', got '%s'", member.Role)
	}
	if member.Member.DisplayName != "Alice" {
		t.Errorf("expected displayName 'Alice', got '%s'", member.Member.DisplayName)
	}
}

func TestChatAddMember_MockServer(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/spaces/AAAA/members": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				t.Errorf("expected POST, got %s", r.Method)
			}

			var membership chat.Membership
			json.NewDecoder(r.Body).Decode(&membership)

			if membership.Member == nil || membership.Member.Name != "users/333" {
				t.Errorf("expected user 'users/333'")
			}
			if membership.Role != "ROLE_MEMBER" {
				t.Errorf("expected role 'ROLE_MEMBER', got '%s'", membership.Role)
			}

			resp := &chat.Membership{
				Name:   "spaces/AAAA/members/333",
				Role:   membership.Role,
				Member: membership.Member,
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

	membership := &chat.Membership{
		Member: &chat.User{Name: "users/333", Type: "HUMAN"},
		Role:   "ROLE_MEMBER",
	}

	created, err := svc.Spaces.Members.Create("spaces/AAAA", membership).Do()
	if err != nil {
		t.Fatalf("failed to add member: %v", err)
	}

	if created.Name != "spaces/AAAA/members/333" {
		t.Errorf("expected name 'spaces/AAAA/members/333', got '%s'", created.Name)
	}
}

func TestChatRemoveMember_MockServer(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/spaces/AAAA/members/111": func(w http.ResponseWriter, r *http.Request) {
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

	_, err = svc.Spaces.Members.Delete("spaces/AAAA/members/111").Do()
	if err != nil {
		t.Fatalf("failed to remove member: %v", err)
	}
}

func TestChatUpdateMember_MockServer(t *testing.T) {
	var capturedUpdateMask string
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/spaces/AAAA/members/111": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "PATCH" {
				t.Errorf("expected PATCH, got %s", r.Method)
			}
			capturedUpdateMask = r.URL.Query().Get("updateMask")

			var membership chat.Membership
			json.NewDecoder(r.Body).Decode(&membership)

			resp := &chat.Membership{
				Name: "spaces/AAAA/members/111",
				Role: membership.Role,
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

	membership := &chat.Membership{Role: "ROLE_MANAGER"}
	updated, err := svc.Spaces.Members.Patch("spaces/AAAA/members/111", membership).UpdateMask("role").Do()
	if err != nil {
		t.Fatalf("failed to update member: %v", err)
	}

	if capturedUpdateMask != "role" {
		t.Errorf("expected updateMask 'role', got '%s'", capturedUpdateMask)
	}
	if updated.Role != "ROLE_MANAGER" {
		t.Errorf("expected role 'ROLE_MANAGER', got '%s'", updated.Role)
	}
}

// --- Read state mock server tests ---

func TestChatReadState_MockServer(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/users/me/spaces/AAAA/spaceReadState": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "GET" {
				t.Errorf("expected GET, got %s", r.Method)
			}
			resp := map[string]interface{}{
				"name":         "users/me/spaces/AAAA/spaceReadState",
				"lastReadTime": "2026-02-18T10:00:00Z",
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

	state, err := svc.Users.Spaces.GetSpaceReadState("users/me/spaces/AAAA/spaceReadState").Do()
	if err != nil {
		t.Fatalf("failed to get read state: %v", err)
	}

	if state.LastReadTime != "2026-02-18T10:00:00Z" {
		t.Errorf("expected lastReadTime '2026-02-18T10:00:00Z', got '%s'", state.LastReadTime)
	}
}

func TestChatMarkRead_MockServer(t *testing.T) {
	var capturedUpdateMask string
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/users/me/spaces/AAAA/spaceReadState": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "PATCH" {
				t.Errorf("expected PATCH, got %s", r.Method)
			}
			capturedUpdateMask = r.URL.Query().Get("updateMask")

			var state chat.SpaceReadState
			json.NewDecoder(r.Body).Decode(&state)

			resp := &chat.SpaceReadState{
				Name:         "users/me/spaces/AAAA/spaceReadState",
				LastReadTime: state.LastReadTime,
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

	state := &chat.SpaceReadState{LastReadTime: "2026-02-18T12:00:00Z"}
	updated, err := svc.Users.Spaces.UpdateSpaceReadState("users/me/spaces/AAAA/spaceReadState", state).UpdateMask("last_read_time").Do()
	if err != nil {
		t.Fatalf("failed to mark read: %v", err)
	}

	if capturedUpdateMask != "last_read_time" {
		t.Errorf("expected updateMask 'last_read_time', got '%s'", capturedUpdateMask)
	}
	if updated.LastReadTime != "2026-02-18T12:00:00Z" {
		t.Errorf("expected lastReadTime '2026-02-18T12:00:00Z', got '%s'", updated.LastReadTime)
	}
}

func TestChatThreadReadState_MockServer(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/users/me/spaces/AAAA/threads/thread1/threadReadState": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "GET" {
				t.Errorf("expected GET, got %s", r.Method)
			}
			resp := map[string]interface{}{
				"name":         "users/me/spaces/AAAA/threads/thread1/threadReadState",
				"lastReadTime": "2026-02-18T09:00:00Z",
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

	state, err := svc.Users.Spaces.Threads.GetThreadReadState("users/me/spaces/AAAA/threads/thread1/threadReadState").Do()
	if err != nil {
		t.Fatalf("failed to get thread read state: %v", err)
	}

	if state.LastReadTime != "2026-02-18T09:00:00Z" {
		t.Errorf("expected lastReadTime '2026-02-18T09:00:00Z', got '%s'", state.LastReadTime)
	}
}

// --- Attachment mock server test ---

func TestChatAttachment_MockServer(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/spaces/AAAA/messages/msg1/attachments/att1": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "GET" {
				t.Errorf("expected GET, got %s", r.Method)
			}
			resp := map[string]interface{}{
				"name":         "spaces/AAAA/messages/msg1/attachments/att1",
				"contentName":  "document.pdf",
				"contentType":  "application/pdf",
				"source":       "UPLOADED_CONTENT",
				"downloadUri":  "https://example.com/download",
				"thumbnailUri": "https://example.com/thumb",
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

	att, err := svc.Spaces.Messages.Attachments.Get("spaces/AAAA/messages/msg1/attachments/att1").Do()
	if err != nil {
		t.Fatalf("failed to get attachment: %v", err)
	}

	if att.Name != "spaces/AAAA/messages/msg1/attachments/att1" {
		t.Errorf("expected name, got '%s'", att.Name)
	}
	if att.ContentName != "document.pdf" {
		t.Errorf("expected contentName 'document.pdf', got '%s'", att.ContentName)
	}
	if att.ContentType != "application/pdf" {
		t.Errorf("expected contentType 'application/pdf', got '%s'", att.ContentType)
	}
	if att.Source != "UPLOADED_CONTENT" {
		t.Errorf("expected source 'UPLOADED_CONTENT', got '%s'", att.Source)
	}
	if att.DownloadUri != "https://example.com/download" {
		t.Errorf("expected downloadUri, got '%s'", att.DownloadUri)
	}
	if att.ThumbnailUri != "https://example.com/thumb" {
		t.Errorf("expected thumbnailUri, got '%s'", att.ThumbnailUri)
	}
}

// --- Media mock server tests ---

func TestChatUpload_MockServer(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/upload/v1/spaces/AAAA/attachments:upload": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				t.Errorf("expected POST, got %s", r.Method)
			}
			resp := map[string]interface{}{
				"attachmentDataRef": map[string]interface{}{
					"resourceName": "spaces/AAAA/attachments/data123",
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

	// Create a temp file for upload
	tmpFile, err := os.CreateTemp("", "upload-test-*.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString("test content")
	tmpFile.Close()

	file, err := os.Open(tmpFile.Name())
	if err != nil {
		t.Fatalf("failed to open temp file: %v", err)
	}
	defer file.Close()

	req := &chat.UploadAttachmentRequest{Filename: "test.txt"}
	resp, err := svc.Media.Upload("spaces/AAAA", req).Media(file).Do()
	if err != nil {
		t.Fatalf("failed to upload: %v", err)
	}

	if resp.AttachmentDataRef == nil || resp.AttachmentDataRef.ResourceName != "spaces/AAAA/attachments/data123" {
		t.Errorf("expected attachment data ref resource name")
	}
}

func TestChatDownload_MockServer(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/media/spaces/AAAA/attachments/data123": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "GET" {
				t.Errorf("expected GET, got %s", r.Method)
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write([]byte("downloaded file content"))
		},
	}

	server := mockChatServer(t, handlers)
	defer server.Close()

	svc, err := chat.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create chat service: %v", err)
	}

	resp, err := svc.Media.Download("spaces/AAAA/attachments/data123").Download()
	if err != nil {
		t.Fatalf("failed to download: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}

	if string(body) != "downloaded file content" {
		t.Errorf("expected 'downloaded file content', got '%s'", string(body))
	}
}

// --- Space events mock server tests ---

func TestChatEvents_MockServer(t *testing.T) {
	var capturedFilter string
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/spaces/AAAA/spaceEvents": func(w http.ResponseWriter, r *http.Request) {
			capturedFilter = r.URL.Query().Get("filter")
			resp := map[string]interface{}{
				"spaceEvents": []map[string]interface{}{
					{
						"name":      "spaces/AAAA/spaceEvents/evt1",
						"eventType": "google.workspace.chat.message.v1.created",
						"eventTime": "2026-02-18T10:00:00Z",
					},
					{
						"name":      "spaces/AAAA/spaceEvents/evt2",
						"eventType": "google.workspace.chat.message.v1.created",
						"eventTime": "2026-02-18T10:05:00Z",
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

	filterStr := `event_types:"google.workspace.chat.message.v1.created"`
	resp, err := svc.Spaces.SpaceEvents.List("spaces/AAAA").Filter(filterStr).PageSize(100).Do()
	if err != nil {
		t.Fatalf("failed to list events: %v", err)
	}

	if capturedFilter != filterStr {
		t.Errorf("expected filter '%s', got '%s'", filterStr, capturedFilter)
	}
	if len(resp.SpaceEvents) != 2 {
		t.Fatalf("expected 2 events, got %d", len(resp.SpaceEvents))
	}
	if resp.SpaceEvents[0].EventType != "google.workspace.chat.message.v1.created" {
		t.Errorf("expected event type, got '%s'", resp.SpaceEvents[0].EventType)
	}
}

func TestChatEvents_Pagination(t *testing.T) {
	pagesFetched := 0
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/spaces/AAAA/spaceEvents": func(w http.ResponseWriter, r *http.Request) {
			pagesFetched++
			pageToken := r.URL.Query().Get("pageToken")

			if pageToken == "" {
				resp := map[string]interface{}{
					"spaceEvents": []map[string]interface{}{
						{"name": "spaces/AAAA/spaceEvents/evt1", "eventType": "google.workspace.chat.message.v1.created", "eventTime": "2026-02-18T10:00:00Z"},
					},
					"nextPageToken": "page2",
				}
				json.NewEncoder(w).Encode(resp)
				return
			}

			resp := map[string]interface{}{
				"spaceEvents": []map[string]interface{}{
					{"name": "spaces/AAAA/spaceEvents/evt2", "eventType": "google.workspace.chat.message.v1.created", "eventTime": "2026-02-18T10:05:00Z"},
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

	var allEvents []*chat.SpaceEvent
	var pageToken string
	for {
		call := svc.Spaces.SpaceEvents.List("spaces/AAAA").Filter("event_types:\"test\"").PageSize(10)
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}
		resp, err := call.Do()
		if err != nil {
			t.Fatalf("failed to list events: %v", err)
		}
		allEvents = append(allEvents, resp.SpaceEvents...)
		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}

	if pagesFetched != 2 {
		t.Errorf("expected 2 pages fetched, got %d", pagesFetched)
	}
	if len(allEvents) != 2 {
		t.Errorf("expected 2 total events, got %d", len(allEvents))
	}
}

func TestChatEvent_MockServer(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/spaces/AAAA/spaceEvents/evt1": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "GET" {
				t.Errorf("expected GET, got %s", r.Method)
			}
			resp := map[string]interface{}{
				"name":      "spaces/AAAA/spaceEvents/evt1",
				"eventType": "google.workspace.chat.message.v1.created",
				"eventTime": "2026-02-18T10:00:00Z",
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

	event, err := svc.Spaces.SpaceEvents.Get("spaces/AAAA/spaceEvents/evt1").Do()
	if err != nil {
		t.Fatalf("failed to get event: %v", err)
	}

	if event.Name != "spaces/AAAA/spaceEvents/evt1" {
		t.Errorf("expected name, got '%s'", event.Name)
	}
	if event.EventType != "google.workspace.chat.message.v1.created" {
		t.Errorf("expected event type, got '%s'", event.EventType)
	}
	if event.EventTime != "2026-02-18T10:00:00Z" {
		t.Errorf("expected event time, got '%s'", event.EventTime)
	}
}

// --- Issue #137 tests: --max pagination, lastActiveTime, message consistency ---

func TestChatList_PaginationWithMax(t *testing.T) {
	pagesFetched := 0
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/spaces": func(w http.ResponseWriter, r *http.Request) {
			pagesFetched++
			pageToken := r.URL.Query().Get("pageToken")

			if pageToken == "" {
				resp := map[string]interface{}{
					"spaces": []map[string]interface{}{
						{"name": "spaces/AAAA", "displayName": "Space 1", "spaceType": "SPACE"},
						{"name": "spaces/BBBB", "displayName": "Space 2", "spaceType": "SPACE"},
					},
					"nextPageToken": "page2",
				}
				json.NewEncoder(w).Encode(resp)
				return
			}

			// Page 2 should NOT be fetched when max=2
			resp := map[string]interface{}{
				"spaces": []map[string]interface{}{
					{"name": "spaces/CCCC", "displayName": "Space 3", "spaceType": "SPACE"},
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

	// Simulate runChatList logic with max=2
	maxResults := int64(2)
	var results []map[string]interface{}
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
		for _, space := range resp.Spaces {
			results = append(results, mapSpaceToOutput(space))
			if maxResults > 0 && int64(len(results)) >= maxResults {
				break
			}
		}
		if resp.NextPageToken == "" || (maxResults > 0 && int64(len(results)) >= maxResults) {
			break
		}
		pageToken = resp.NextPageToken
	}
	if maxResults > 0 && int64(len(results)) > maxResults {
		results = results[:maxResults]
	}

	if len(results) != 2 {
		t.Errorf("expected 2 results (capped by max), got %d", len(results))
	}
	if pagesFetched != 1 {
		t.Errorf("expected only 1 page fetched (early stop), got %d", pagesFetched)
	}
}

func TestMapSpaceToOutput_LastActiveTime(t *testing.T) {
	space := &chat.Space{
		Name:                "spaces/AAAA",
		DisplayName:         "Test Space",
		SpaceType:           "SPACE",
		CreateTime:          "2025-01-01T00:00:00Z",
		LastActiveTime:      "2026-02-20T15:30:00Z",
		SpaceThreadingState: "THREADED_MESSAGES",
	}

	result := mapSpaceToOutput(space)

	if result["last_active_time"] != "2026-02-20T15:30:00Z" {
		t.Errorf("expected last_active_time '2026-02-20T15:30:00Z', got %v", result["last_active_time"])
	}
	if result["threading_state"] != "THREADED_MESSAGES" {
		t.Errorf("expected threading_state 'THREADED_MESSAGES', got %v", result["threading_state"])
	}
}

func TestMapSpaceToOutput_NoLastActiveTime(t *testing.T) {
	space := &chat.Space{
		Name:        "spaces/BBBB",
		DisplayName: "DM Space",
		SpaceType:   "DIRECT_MESSAGE",
	}

	result := mapSpaceToOutput(space)

	if _, exists := result["last_active_time"]; exists {
		t.Error("last_active_time should be omitted when empty")
	}
	if _, exists := result["threading_state"]; exists {
		t.Error("threading_state should be omitted when empty")
	}
}

func TestMapMemberToOutput_DeleteTime(t *testing.T) {
	m := &chat.Membership{
		Name:       "spaces/AAAA/members/111",
		Role:       "ROLE_MEMBER",
		CreateTime: "2025-01-01T00:00:00Z",
		DeleteTime: "2026-02-20T10:00:00Z",
		Member: &chat.User{
			DisplayName: "Alice",
			Name:        "users/111",
			Type:        "HUMAN",
		},
	}

	result := mapMemberToOutput(m)

	if result["delete_time"] != "2026-02-20T10:00:00Z" {
		t.Errorf("expected delete_time '2026-02-20T10:00:00Z', got %v", result["delete_time"])
	}
}

func TestMapMemberToOutput_NoDeleteTime(t *testing.T) {
	m := &chat.Membership{
		Name: "spaces/AAAA/members/222",
		Role: "ROLE_MEMBER",
	}

	result := mapMemberToOutput(m)

	if _, exists := result["delete_time"]; exists {
		t.Error("delete_time should be omitted when empty")
	}
}

func TestChatMessages_ThreadAndLastUpdateTime(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/spaces/AAAA/messages": func(w http.ResponseWriter, r *http.Request) {
			resp := map[string]interface{}{
				"messages": []map[string]interface{}{
					{
						"name":           "spaces/AAAA/messages/msg1",
						"text":           "Hello",
						"createTime":     "2026-02-16T10:00:00Z",
						"lastUpdateTime": "2026-02-16T10:05:00Z",
						"thread":         map[string]interface{}{"name": "spaces/AAAA/threads/thread1"},
						"sender":         map[string]interface{}{"displayName": "Alice", "type": "HUMAN", "name": "users/111"},
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
	msg := resp.Messages[0]
	if msg.LastUpdateTime != "2026-02-16T10:05:00Z" {
		t.Errorf("expected lastUpdateTime, got '%s'", msg.LastUpdateTime)
	}
	if msg.Thread == nil || msg.Thread.Name != "spaces/AAAA/threads/thread1" {
		t.Errorf("expected thread name, got %+v", msg.Thread)
	}
}

func TestMapSpaceEventToOutput(t *testing.T) {
	event := &chat.SpaceEvent{
		Name:      "spaces/AAAA/spaceEvents/evt1",
		EventType: "google.workspace.chat.message.v1.created",
		EventTime: "2026-02-18T10:00:00Z",
	}

	result := mapSpaceEventToOutput(event)

	if result["name"] != "spaces/AAAA/spaceEvents/evt1" {
		t.Errorf("expected name, got %v", result["name"])
	}
	if result["event_type"] != "google.workspace.chat.message.v1.created" {
		t.Errorf("expected event_type, got %v", result["event_type"])
	}
	if result["event_time"] != "2026-02-18T10:00:00Z" {
		t.Errorf("expected event_time, got %v", result["event_time"])
	}
}

// --- Build cache / Find group tests ---

func TestChatBuildCacheCommand_Flags(t *testing.T) {
	cmd := findSubcommand(chatCmd, "build-cache")
	if cmd == nil {
		t.Fatal("chat build-cache command not found")
	}
	typeFlag := cmd.Flags().Lookup("type")
	if typeFlag == nil {
		t.Fatal("expected --type flag on build-cache")
	}
	if typeFlag.DefValue != "GROUP_CHAT" {
		t.Errorf("expected --type default 'GROUP_CHAT', got %q", typeFlag.DefValue)
	}
}

func TestChatFindGroupCommand_Flags(t *testing.T) {
	cmd := findSubcommand(chatCmd, "find-group")
	if cmd == nil {
		t.Fatal("chat find-group command not found")
	}
	if cmd.Flags().Lookup("members") == nil {
		t.Error("expected --members flag on find-group")
	}
	refreshFlag := cmd.Flags().Lookup("refresh")
	if refreshFlag == nil {
		t.Fatal("expected --refresh flag on find-group")
	}
	if refreshFlag.DefValue != "false" {
		t.Errorf("expected --refresh default 'false', got %q", refreshFlag.DefValue)
	}
}

func TestChatBuildCache_MockServer(t *testing.T) {
	membersCalled := 0
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/spaces": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "GET" {
				t.Errorf("expected GET, got %s", r.Method)
			}
			resp := map[string]interface{}{
				"spaces": []map[string]interface{}{
					{"name": "spaces/AAAA", "displayName": "Team Chat", "spaceType": "GROUP_CHAT"},
					{"name": "spaces/BBBB", "displayName": "Engineering", "spaceType": "GROUP_CHAT"},
				},
			}
			json.NewEncoder(w).Encode(resp)
		},
		"/v1/spaces/AAAA/members": func(w http.ResponseWriter, r *http.Request) {
			membersCalled++
			resp := map[string]interface{}{
				"memberships": []map[string]interface{}{
					{"name": "spaces/AAAA/members/1", "member": map[string]interface{}{"name": "users/111", "type": "HUMAN"}},
					{"name": "spaces/AAAA/members/2", "member": map[string]interface{}{"name": "users/222", "type": "HUMAN"}},
				},
			}
			json.NewEncoder(w).Encode(resp)
		},
		"/v1/spaces/BBBB/members": func(w http.ResponseWriter, r *http.Request) {
			membersCalled++
			resp := map[string]interface{}{
				"memberships": []map[string]interface{}{
					{"name": "spaces/BBBB/members/1", "member": map[string]interface{}{"name": "users/111", "type": "HUMAN"}},
					{"name": "spaces/BBBB/members/3", "member": map[string]interface{}{"name": "users/333", "type": "HUMAN"}},
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

	// Verify we can list spaces and then fetch members for each
	spacesResp, err := svc.Spaces.List().Do()
	if err != nil {
		t.Fatalf("failed to list spaces: %v", err)
	}
	if len(spacesResp.Spaces) != 2 {
		t.Fatalf("expected 2 spaces, got %d", len(spacesResp.Spaces))
	}

	for _, space := range spacesResp.Spaces {
		membersResp, err := svc.Spaces.Members.List(space.Name).Do()
		if err != nil {
			t.Fatalf("failed to list members for %s: %v", space.Name, err)
		}
		if len(membersResp.Memberships) != 2 {
			t.Errorf("expected 2 members for %s, got %d", space.Name, len(membersResp.Memberships))
		}
	}

	if membersCalled != 2 {
		t.Errorf("expected members endpoint called 2 times, got %d", membersCalled)
	}
}

func TestChatFindGroup_MockServerFlow(t *testing.T) {
	// This test validates the full flow: list spaces → fetch members → search
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/spaces": func(w http.ResponseWriter, r *http.Request) {
			resp := map[string]interface{}{
				"spaces": []map[string]interface{}{
					{"name": "spaces/GC1", "spaceType": "GROUP_CHAT"},
					{"name": "spaces/GC2", "spaceType": "GROUP_CHAT"},
				},
			}
			json.NewEncoder(w).Encode(resp)
		},
		"/v1/spaces/GC1/members": func(w http.ResponseWriter, r *http.Request) {
			resp := map[string]interface{}{
				"memberships": []map[string]interface{}{
					{"name": "spaces/GC1/members/1", "member": map[string]interface{}{"name": "users/111", "type": "HUMAN"}},
					{"name": "spaces/GC1/members/2", "member": map[string]interface{}{"name": "users/222", "type": "HUMAN"}},
				},
			}
			json.NewEncoder(w).Encode(resp)
		},
		"/v1/spaces/GC2/members": func(w http.ResponseWriter, r *http.Request) {
			resp := map[string]interface{}{
				"memberships": []map[string]interface{}{
					{"name": "spaces/GC2/members/1", "member": map[string]interface{}{"name": "users/333", "type": "HUMAN"}},
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

	// Simulate build: list spaces + fetch members
	spacesResp, err := svc.Spaces.List().Do()
	if err != nil {
		t.Fatalf("failed to list spaces: %v", err)
	}

	type spaceInfo struct {
		name    string
		members []string
	}
	var spaces []spaceInfo

	for _, space := range spacesResp.Spaces {
		membersResp, err := svc.Spaces.Members.List(space.Name).Do()
		if err != nil {
			t.Fatalf("failed to list members: %v", err)
		}
		var memberIDs []string
		for _, m := range membersResp.Memberships {
			if m.Member != nil {
				memberIDs = append(memberIDs, m.Member.Name)
			}
		}
		spaces = append(spaces, spaceInfo{name: space.Name, members: memberIDs})
	}

	// Search for users/111 — should match only GC1
	found := 0
	for _, sp := range spaces {
		for _, m := range sp.members {
			if m == "users/111" {
				found++
			}
		}
	}
	if found != 1 {
		t.Errorf("expected users/111 in 1 space, found in %d", found)
	}

	// Verify GC2 has only 1 member
	if len(spaces[1].members) != 1 {
		t.Errorf("expected 1 member in GC2, got %d", len(spaces[1].members))
	}
}

// TestChatBuildCache_E2E exercises the full build-cache pipeline with mock server,
// then verifies the saved cache file contents.
func TestChatBuildCache_E2E(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/spaces", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"spaces": []map[string]interface{}{
				{"name": "spaces/GRP1", "displayName": "Dev Team", "spaceType": "GROUP_CHAT"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})
	mux.HandleFunc("/v1/spaces/GRP1/members", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"memberships": []map[string]interface{}{
				{"name": "spaces/GRP1/members/1", "member": map[string]interface{}{"name": "users/100", "type": "HUMAN"}},
				{"name": "spaces/GRP1/members/2", "member": map[string]interface{}{"name": "users/200", "type": "HUMAN"}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	chatSvc, err := chat.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create chat service: %v", err)
	}

	// Build → Save → Load (mirrors runChatBuildCache internals)
	cache, err := spacecache.Build(context.Background(), chatSvc, nil, "GROUP_CHAT", nil)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	tmpPath := filepath.Join(t.TempDir(), "test-cache.json")
	if err := spacecache.Save(tmpPath, cache); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Reload and verify (mirrors what find-group would read)
	loaded, err := spacecache.Load(tmpPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(loaded.Spaces) != 1 {
		t.Fatalf("expected 1 space, got %d", len(loaded.Spaces))
	}
	entry := loaded.Spaces["spaces/GRP1"]
	if entry.DisplayName != "Dev Team" {
		t.Errorf("expected display name 'Dev Team', got %q", entry.DisplayName)
	}
	if entry.MemberCount != 2 {
		t.Errorf("expected 2 members, got %d", entry.MemberCount)
	}
	if entry.Type != "GROUP_CHAT" {
		t.Errorf("expected type GROUP_CHAT, got %q", entry.Type)
	}
}

// TestChatFindGroup_CommandE2E exercises the find-group Cobra command end-to-end:
// pre-populate cache at DefaultPath (via temp HOME) → execute cmd.RunE → verify JSON output.
func TestChatFindGroup_CommandE2E(t *testing.T) {
	// Use temp HOME so DefaultPath() resolves to our temp dir
	tmpHome := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	// Pre-populate cache at the default path
	cache := &spacecache.CacheData{
		Spaces: map[string]spacecache.SpaceEntry{
			"spaces/GRP1": {
				Type:        "GROUP_CHAT",
				DisplayName: "Project Alpha",
				Members:     []string{"alice@example.com", "bob@example.com"},
				MemberCount: 2,
			},
			"spaces/GRP2": {
				Type:        "GROUP_CHAT",
				DisplayName: "",
				Members:     []string{"alice@example.com", "charlie@example.com"},
				MemberCount: 2,
			},
		},
	}
	if err := spacecache.Save(spacecache.DefaultPath(), cache); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Execute the Cobra command's RunE directly
	cmd := chatFindGroupCmd
	cmd.Flags().Set("members", "alice@example.com")
	cmd.Flags().Set("refresh", "false")

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := cmd.RunE(cmd, []string{})

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("find-group command failed: %v", err)
	}

	output, _ := io.ReadAll(r)

	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("failed to parse output JSON: %v\noutput: %s", err, output)
	}

	count := int(result["count"].(float64))
	if count != 2 {
		t.Errorf("expected 2 matches for alice, got %d", count)
	}

	matches := result["matches"].([]interface{})
	if len(matches) != 2 {
		t.Errorf("expected 2 match entries, got %d", len(matches))
	}
}

// TestChatFindGroup_ErrorOnEmptyMembers verifies --members with only blanks/commas outputs error JSON.
func TestChatFindGroup_ErrorOnEmptyMembers(t *testing.T) {
	tmpHome := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	// Create a non-empty cache so we get past the "no cache" check
	cache := &spacecache.CacheData{
		Spaces: map[string]spacecache.SpaceEntry{
			"spaces/X": {Type: "GROUP_CHAT", Members: []string{"a@b.com"}, MemberCount: 1},
		},
	}
	if err := spacecache.Save(spacecache.DefaultPath(), cache); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	cmd := chatFindGroupCmd
	cmd.Flags().Set("members", "  ,  , ")
	cmd.Flags().Set("refresh", "false")

	// Capture stdout (PrintError writes error JSON to stdout)
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd.RunE(cmd, []string{})

	w.Close()
	os.Stdout = oldStdout

	output, _ := io.ReadAll(r)
	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("failed to parse output: %v\nraw: %s", err, output)
	}
	errMsg, ok := result["error"].(string)
	if !ok || errMsg == "" {
		t.Errorf("expected error message in output, got %v", result)
	}
}

// TestChatFindGroup_NoCache verifies error output when no cache file exists.
func TestChatFindGroup_NoCache(t *testing.T) {
	tmpHome := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	cmd := chatFindGroupCmd
	cmd.Flags().Set("members", "alice@example.com")
	cmd.Flags().Set("refresh", "false")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd.RunE(cmd, []string{})

	w.Close()
	os.Stdout = oldStdout

	output, _ := io.ReadAll(r)
	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		t.Fatalf("failed to parse output: %v\nraw: %s", err, output)
	}
	errMsg, ok := result["error"].(string)
	if !ok || errMsg == "" {
		t.Errorf("expected error message in output, got %v", result)
	}
}
