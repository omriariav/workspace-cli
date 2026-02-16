package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/api/option"
	"google.golang.org/api/people/v1"
)

// TestContactsCommands tests contacts command structure
func TestContactsCommands(t *testing.T) {
	tests := []struct {
		name string
		use  string
	}{
		{"list", "list"},
		{"search", "search <query>"},
		{"get", "get <resource-name>"},
		{"create", "create"},
		{"delete", "delete <resource-name>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := findSubcommand(contactsCmd, tt.name)
			if cmd == nil {
				t.Fatalf("command '%s' not found", tt.name)
			}
			if cmd.Use != tt.use {
				t.Errorf("expected Use '%s', got '%s'", tt.use, cmd.Use)
			}
		})
	}
}

// TestContactsListCommand_Flags tests list command flags
func TestContactsListCommand_Flags(t *testing.T) {
	cmd := findSubcommand(contactsCmd, "list")
	if cmd == nil {
		t.Fatal("contacts list command not found")
	}

	if cmd.Flags().Lookup("max") == nil {
		t.Error("expected --max flag")
	}
}

// TestContactsCreateCommand_Flags tests create command flags
func TestContactsCreateCommand_Flags(t *testing.T) {
	cmd := findSubcommand(contactsCmd, "create")
	if cmd == nil {
		t.Fatal("contacts create command not found")
	}

	flags := []string{"name", "email", "phone"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected --%s flag", flag)
		}
	}
}

// mockPeopleServer creates a test server that mocks People API responses
func mockPeopleServer(t *testing.T, handlers map[string]func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
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

// TestContactsList_Success tests listing contacts
func TestContactsList_Success(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/people/me/connections": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "GET" {
				t.Errorf("expected GET, got %s", r.Method)
			}

			json.NewEncoder(w).Encode(&people.ListConnectionsResponse{
				Connections: []*people.Person{
					{
						ResourceName: "people/c1",
						Names:        []*people.Name{{DisplayName: "John Doe"}},
						EmailAddresses: []*people.EmailAddress{
							{Value: "john@example.com"},
						},
					},
					{
						ResourceName: "people/c2",
						Names:        []*people.Name{{DisplayName: "Jane Smith"}},
						PhoneNumbers: []*people.PhoneNumber{
							{Value: "+1234567890"},
						},
					},
				},
				TotalPeople: 2,
			})
		},
	}

	server := mockPeopleServer(t, handlers)
	defer server.Close()

	svc, err := people.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create people service: %v", err)
	}

	resp, err := svc.People.Connections.List("people/me").PersonFields("names,emailAddresses,phoneNumbers").Do()
	if err != nil {
		t.Fatalf("failed to list contacts: %v", err)
	}

	if len(resp.Connections) != 2 {
		t.Errorf("expected 2 contacts, got %d", len(resp.Connections))
	}

	if resp.Connections[0].Names[0].DisplayName != "John Doe" {
		t.Errorf("expected 'John Doe', got '%s'", resp.Connections[0].Names[0].DisplayName)
	}
}

// TestContactsSearch_Success tests searching contacts
func TestContactsSearch_Success(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/people:searchContacts": func(w http.ResponseWriter, r *http.Request) {
			query := r.URL.Query().Get("query")
			if query != "John" {
				t.Errorf("expected query 'John', got '%s'", query)
			}

			json.NewEncoder(w).Encode(&people.SearchResponse{
				Results: []*people.SearchResult{
					{
						Person: &people.Person{
							ResourceName: "people/c1",
							Names:        []*people.Name{{DisplayName: "John Doe"}},
						},
					},
				},
			})
		},
	}

	server := mockPeopleServer(t, handlers)
	defer server.Close()

	svc, err := people.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create people service: %v", err)
	}

	resp, err := svc.People.SearchContacts().Query("John").ReadMask("names").Do()
	if err != nil {
		t.Fatalf("failed to search contacts: %v", err)
	}

	if len(resp.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(resp.Results))
	}

	if resp.Results[0].Person.Names[0].DisplayName != "John Doe" {
		t.Errorf("expected 'John Doe', got '%s'", resp.Results[0].Person.Names[0].DisplayName)
	}
}

// TestContactsGet_Success tests getting a specific contact
func TestContactsGet_Success(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/people/c1": func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(&people.Person{
				ResourceName: "people/c1",
				Names:        []*people.Name{{DisplayName: "John Doe"}},
				EmailAddresses: []*people.EmailAddress{
					{Value: "john@example.com"},
				},
				PhoneNumbers: []*people.PhoneNumber{
					{Value: "+1234567890"},
				},
				Organizations: []*people.Organization{
					{Name: "Acme Inc", Title: "Engineer"},
				},
			})
		},
	}

	server := mockPeopleServer(t, handlers)
	defer server.Close()

	svc, err := people.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create people service: %v", err)
	}

	person, err := svc.People.Get("people/c1").PersonFields("names,emailAddresses,phoneNumbers,organizations").Do()
	if err != nil {
		t.Fatalf("failed to get contact: %v", err)
	}

	if person.Names[0].DisplayName != "John Doe" {
		t.Errorf("expected 'John Doe', got '%s'", person.Names[0].DisplayName)
	}
	if person.Organizations[0].Name != "Acme Inc" {
		t.Errorf("expected 'Acme Inc', got '%s'", person.Organizations[0].Name)
	}
}

// TestContactsCreate_Success tests creating a contact
func TestContactsCreate_Success(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/people:createContact": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				t.Errorf("expected POST, got %s", r.Method)
			}

			var req people.Person
			json.NewDecoder(r.Body).Decode(&req)

			if len(req.Names) == 0 || req.Names[0].UnstructuredName != "John Doe" {
				t.Errorf("expected name 'John Doe'")
			}

			json.NewEncoder(w).Encode(&people.Person{
				ResourceName: "people/c123",
				Names:        []*people.Name{{DisplayName: "John Doe", UnstructuredName: "John Doe"}},
				EmailAddresses: []*people.EmailAddress{
					{Value: "john@example.com"},
				},
			})
		},
	}

	server := mockPeopleServer(t, handlers)
	defer server.Close()

	svc, err := people.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create people service: %v", err)
	}

	person := &people.Person{
		Names: []*people.Name{{UnstructuredName: "John Doe"}},
		EmailAddresses: []*people.EmailAddress{
			{Value: "john@example.com"},
		},
	}

	created, err := svc.People.CreateContact(person).Do()
	if err != nil {
		t.Fatalf("failed to create contact: %v", err)
	}

	if created.ResourceName != "people/c123" {
		t.Errorf("expected resource name 'people/c123', got '%s'", created.ResourceName)
	}
}

// TestContactsDelete_Success tests deleting a contact
func TestContactsDelete_Success(t *testing.T) {
	deleteCalled := false

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/people/c123:deleteContact": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "DELETE" {
				t.Errorf("expected DELETE, got %s", r.Method)
			}
			deleteCalled = true
			json.NewEncoder(w).Encode(map[string]interface{}{})
		},
	}

	server := mockPeopleServer(t, handlers)
	defer server.Close()

	svc, err := people.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create people service: %v", err)
	}

	_, err = svc.People.DeleteContact("people/c123").Do()
	if err != nil {
		t.Fatalf("failed to delete contact: %v", err)
	}

	if !deleteCalled {
		t.Error("delete endpoint was not called")
	}
}

// TestFormatPerson tests the formatPerson helper
func TestFormatPerson(t *testing.T) {
	person := &people.Person{
		ResourceName: "people/c1",
		Names:        []*people.Name{{DisplayName: "Test User"}},
		EmailAddresses: []*people.EmailAddress{
			{Value: "test@example.com"},
			{Value: "test2@example.com"},
		},
		PhoneNumbers: []*people.PhoneNumber{
			{Value: "+1234567890"},
		},
		Organizations: []*people.Organization{
			{Name: "TestCo", Title: "Developer"},
		},
	}

	result := formatPerson(person)

	if result["resource_name"] != "people/c1" {
		t.Errorf("expected resource_name 'people/c1', got '%v'", result["resource_name"])
	}
	if result["name"] != "Test User" {
		t.Errorf("expected name 'Test User', got '%v'", result["name"])
	}

	emails := result["emails"].([]string)
	if len(emails) != 2 {
		t.Errorf("expected 2 emails, got %d", len(emails))
	}

	phones := result["phones"].([]string)
	if len(phones) != 1 {
		t.Errorf("expected 1 phone, got %d", len(phones))
	}

	org := result["organization"].(map[string]interface{})
	if org["name"] != "TestCo" {
		t.Errorf("expected org name 'TestCo', got '%v'", org["name"])
	}
}

// TestFormatPerson_Minimal tests formatPerson with minimal data
func TestFormatPerson_Minimal(t *testing.T) {
	person := &people.Person{
		ResourceName: "people/c2",
	}

	result := formatPerson(person)

	if result["resource_name"] != "people/c2" {
		t.Errorf("expected resource_name 'people/c2', got '%v'", result["resource_name"])
	}
	if _, ok := result["name"]; ok {
		t.Error("expected no name field for person without names")
	}
	if _, ok := result["emails"]; ok {
		t.Error("expected no emails field for person without emails")
	}
}
