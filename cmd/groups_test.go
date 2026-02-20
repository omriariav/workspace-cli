package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	admin "google.golang.org/api/admin/directory/v1"
	"google.golang.org/api/option"
)

// TestGroupsCommands tests groups command structure
func TestGroupsCommands(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"list"},
		{"members"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := findSubcommand(groupsCmd, tt.name)
			if cmd == nil {
				t.Fatalf("command '%s' not found", tt.name)
			}
		})
	}
}

func TestGroupsListCommand_Flags(t *testing.T) {
	cmd := findSubcommand(groupsCmd, "list")
	if cmd == nil {
		t.Fatal("groups list command not found")
	}
	if cmd.Use != "list" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}

	maxFlag := cmd.Flags().Lookup("max")
	if maxFlag == nil {
		t.Error("expected --max flag")
	}
	if maxFlag.DefValue != "50" {
		t.Errorf("expected --max default 50, got %s", maxFlag.DefValue)
	}

	domainFlag := cmd.Flags().Lookup("domain")
	if domainFlag == nil {
		t.Error("expected --domain flag")
	}

	userEmailFlag := cmd.Flags().Lookup("user-email")
	if userEmailFlag == nil {
		t.Error("expected --user-email flag")
	}
}

func TestGroupsListCommand_Help(t *testing.T) {
	cmd := groupsListCmd
	if cmd.Short == "" {
		t.Error("expected Short description to be set")
	}
	if cmd.Long == "" {
		t.Error("expected Long description to be set")
	}
}

func TestGroupsMembersCommand_Flags(t *testing.T) {
	cmd := findSubcommand(groupsCmd, "members")
	if cmd == nil {
		t.Fatal("groups members command not found")
	}
	if cmd.Use != "members <group-email>" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}
	if cmd.Args == nil {
		t.Error("expected Args validator to be set")
	}

	maxFlag := cmd.Flags().Lookup("max")
	if maxFlag == nil {
		t.Error("expected --max flag")
	}
	if maxFlag.DefValue != "50" {
		t.Errorf("expected --max default 50, got %s", maxFlag.DefValue)
	}

	roleFlag := cmd.Flags().Lookup("role")
	if roleFlag == nil {
		t.Error("expected --role flag")
	}
}

func TestGroupsMembersCommand_Help(t *testing.T) {
	cmd := groupsMembersCmd
	if cmd.Short == "" {
		t.Error("expected Short description to be set")
	}
	if cmd.Long == "" {
		t.Error("expected Long description to be set")
	}
}

// mockAdminServer creates a test server that mocks Admin Directory API responses
func mockAdminServer(t *testing.T, handlers map[string]func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
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

func TestGroupsList_MockServer(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/admin/directory/v1/groups": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "GET" {
				t.Errorf("expected GET, got %s", r.Method)
			}

			// Verify query params
			if r.URL.Query().Get("domain") != "example.com" {
				t.Errorf("expected domain=example.com, got %s", r.URL.Query().Get("domain"))
			}

			resp := &admin.Groups{
				Groups: []*admin.Group{
					{
						Id:                "group-1",
						Email:             "eng@example.com",
						Name:              "Engineering",
						Description:       "Engineering team",
						DirectMembersCount: 25,
					},
					{
						Id:                "group-2",
						Email:             "product@example.com",
						Name:              "Product",
						DirectMembersCount: 10,
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		},
	}

	server := mockAdminServer(t, handlers)
	defer server.Close()

	svc, err := admin.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create admin service: %v", err)
	}

	resp, err := svc.Groups.List().Domain("example.com").MaxResults(50).Do()
	if err != nil {
		t.Fatalf("failed to list groups: %v", err)
	}

	if len(resp.Groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(resp.Groups))
	}
	if resp.Groups[0].Email != "eng@example.com" {
		t.Errorf("expected email 'eng@example.com', got '%s'", resp.Groups[0].Email)
	}
	if resp.Groups[0].Name != "Engineering" {
		t.Errorf("expected name 'Engineering', got '%s'", resp.Groups[0].Name)
	}
	if resp.Groups[0].DirectMembersCount != 25 {
		t.Errorf("expected 25 members, got %d", resp.Groups[0].DirectMembersCount)
	}
	if resp.Groups[1].Email != "product@example.com" {
		t.Errorf("expected email 'product@example.com', got '%s'", resp.Groups[1].Email)
	}
}

func TestGroupsMembers_MockServer(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/admin/directory/v1/groups/eng@example.com/members": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "GET" {
				t.Errorf("expected GET, got %s", r.Method)
			}

			resp := &admin.Members{
				Members: []*admin.Member{
					{
						Id:     "user-1",
						Email:  "alice@example.com",
						Role:   "OWNER",
						Type:   "USER",
						Status: "ACTIVE",
					},
					{
						Id:     "user-2",
						Email:  "bob@example.com",
						Role:   "MEMBER",
						Type:   "USER",
						Status: "ACTIVE",
					},
					{
						Id:    "group-3",
						Email: "subteam@example.com",
						Role:  "MEMBER",
						Type:  "GROUP",
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		},
	}

	server := mockAdminServer(t, handlers)
	defer server.Close()

	svc, err := admin.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create admin service: %v", err)
	}

	resp, err := svc.Members.List("eng@example.com").MaxResults(50).Do()
	if err != nil {
		t.Fatalf("failed to list members: %v", err)
	}

	if len(resp.Members) != 3 {
		t.Fatalf("expected 3 members, got %d", len(resp.Members))
	}
	if resp.Members[0].Email != "alice@example.com" {
		t.Errorf("expected email 'alice@example.com', got '%s'", resp.Members[0].Email)
	}
	if resp.Members[0].Role != "OWNER" {
		t.Errorf("expected role 'OWNER', got '%s'", resp.Members[0].Role)
	}
	if resp.Members[1].Email != "bob@example.com" {
		t.Errorf("expected email 'bob@example.com', got '%s'", resp.Members[1].Email)
	}
	if resp.Members[2].Type != "GROUP" {
		t.Errorf("expected type 'GROUP', got '%s'", resp.Members[2].Type)
	}
}

func TestGroupsMembersWithRole_MockServer(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/admin/directory/v1/groups/eng@example.com/members": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "GET" {
				t.Errorf("expected GET, got %s", r.Method)
			}

			// Verify role filter
			if r.URL.Query().Get("roles") != "OWNER" {
				t.Errorf("expected roles=OWNER, got %s", r.URL.Query().Get("roles"))
			}

			resp := &admin.Members{
				Members: []*admin.Member{
					{
						Id:     "user-1",
						Email:  "alice@example.com",
						Role:   "OWNER",
						Type:   "USER",
						Status: "ACTIVE",
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		},
	}

	server := mockAdminServer(t, handlers)
	defer server.Close()

	svc, err := admin.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create admin service: %v", err)
	}

	resp, err := svc.Members.List("eng@example.com").Roles("OWNER").MaxResults(50).Do()
	if err != nil {
		t.Fatalf("failed to list members with role filter: %v", err)
	}

	if len(resp.Members) != 1 {
		t.Fatalf("expected 1 member, got %d", len(resp.Members))
	}
	if resp.Members[0].Role != "OWNER" {
		t.Errorf("expected role 'OWNER', got '%s'", resp.Members[0].Role)
	}
}
