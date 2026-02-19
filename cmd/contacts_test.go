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
		{"update", "update <resource-name>"},
		{"batch-create", "batch-create"},
		{"batch-update", "batch-update"},
		{"batch-delete", "batch-delete"},
		{"directory", "directory"},
		{"directory-search", "directory-search"},
		{"photo", "photo <resource-name>"},
		{"delete-photo", "delete-photo <resource-name>"},
		{"resolve", "resolve"},
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

// TestContactsUpdateCommand_Flags tests update command flags
func TestContactsUpdateCommand_Flags(t *testing.T) {
	cmd := findSubcommand(contactsCmd, "update")
	if cmd == nil {
		t.Fatal("contacts update command not found")
	}

	flags := []string{"name", "email", "phone", "organization", "title", "etag"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected --%s flag", flag)
		}
	}
}

// TestContactsBatchCreateCommand_Flags tests batch-create command flags
func TestContactsBatchCreateCommand_Flags(t *testing.T) {
	cmd := findSubcommand(contactsCmd, "batch-create")
	if cmd == nil {
		t.Fatal("contacts batch-create command not found")
	}

	if cmd.Flags().Lookup("file") == nil {
		t.Error("expected --file flag")
	}
}

// TestContactsBatchUpdateCommand_Flags tests batch-update command flags
func TestContactsBatchUpdateCommand_Flags(t *testing.T) {
	cmd := findSubcommand(contactsCmd, "batch-update")
	if cmd == nil {
		t.Fatal("contacts batch-update command not found")
	}

	if cmd.Flags().Lookup("file") == nil {
		t.Error("expected --file flag")
	}
}

// TestContactsBatchDeleteCommand_Flags tests batch-delete command flags
func TestContactsBatchDeleteCommand_Flags(t *testing.T) {
	cmd := findSubcommand(contactsCmd, "batch-delete")
	if cmd == nil {
		t.Fatal("contacts batch-delete command not found")
	}

	if cmd.Flags().Lookup("resources") == nil {
		t.Error("expected --resources flag")
	}
}

// TestContactsDirectoryCommand_Flags tests directory command flags
func TestContactsDirectoryCommand_Flags(t *testing.T) {
	cmd := findSubcommand(contactsCmd, "directory")
	if cmd == nil {
		t.Fatal("contacts directory command not found")
	}

	if cmd.Flags().Lookup("max") == nil {
		t.Error("expected --max flag")
	}
}

// TestContactsDirectorySearchCommand_Flags tests directory-search command flags
func TestContactsDirectorySearchCommand_Flags(t *testing.T) {
	cmd := findSubcommand(contactsCmd, "directory-search")
	if cmd == nil {
		t.Fatal("contacts directory-search command not found")
	}

	flags := []string{"query", "max"}
	for _, flag := range flags {
		if cmd.Flags().Lookup(flag) == nil {
			t.Errorf("expected --%s flag", flag)
		}
	}
}

// TestContactsPhotoCommand_Flags tests photo command flags
func TestContactsPhotoCommand_Flags(t *testing.T) {
	cmd := findSubcommand(contactsCmd, "photo")
	if cmd == nil {
		t.Fatal("contacts photo command not found")
	}

	if cmd.Flags().Lookup("file") == nil {
		t.Error("expected --file flag")
	}
}

// TestContactsResolveCommand_Flags tests resolve command flags
func TestContactsResolveCommand_Flags(t *testing.T) {
	cmd := findSubcommand(contactsCmd, "resolve")
	if cmd == nil {
		t.Fatal("contacts resolve command not found")
	}

	if cmd.Flags().Lookup("ids") == nil {
		t.Error("expected --ids flag")
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

// TestContactsUpdate_Success tests updating a contact
func TestContactsUpdate_Success(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/people/c123:updateContact": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "PATCH" {
				t.Errorf("expected PATCH, got %s", r.Method)
			}

			updateMask := r.URL.Query().Get("updatePersonFields")
			if updateMask == "" {
				t.Error("expected updatePersonFields parameter")
			}

			json.NewEncoder(w).Encode(&people.Person{
				ResourceName: "people/c123",
				Etag:         "etag2",
				Names:        []*people.Name{{DisplayName: "Jane Doe"}},
				EmailAddresses: []*people.EmailAddress{
					{Value: "jane@example.com"},
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
		Etag:  "etag1",
		Names: []*people.Name{{UnstructuredName: "Jane Doe"}},
		EmailAddresses: []*people.EmailAddress{
			{Value: "jane@example.com"},
		},
	}

	updated, err := svc.People.UpdateContact("people/c123", person).
		UpdatePersonFields("names,emailAddresses").
		PersonFields(personFields).
		Do()
	if err != nil {
		t.Fatalf("failed to update contact: %v", err)
	}

	if updated.ResourceName != "people/c123" {
		t.Errorf("expected resource name 'people/c123', got '%s'", updated.ResourceName)
	}
	if updated.Names[0].DisplayName != "Jane Doe" {
		t.Errorf("expected 'Jane Doe', got '%s'", updated.Names[0].DisplayName)
	}
}

// TestContactsBatchCreate_Success tests batch creating contacts
func TestContactsBatchCreate_Success(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/people:batchCreateContacts": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				t.Errorf("expected POST, got %s", r.Method)
			}

			json.NewEncoder(w).Encode(&people.BatchCreateContactsResponse{
				CreatedPeople: []*people.PersonResponse{
					{
						Person: &people.Person{
							ResourceName: "people/c100",
							Names:        []*people.Name{{DisplayName: "Contact One"}},
						},
					},
					{
						Person: &people.Person{
							ResourceName: "people/c101",
							Names:        []*people.Name{{DisplayName: "Contact Two"}},
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

	req := &people.BatchCreateContactsRequest{
		Contacts: []*people.ContactToCreate{
			{ContactPerson: &people.Person{Names: []*people.Name{{UnstructuredName: "Contact One"}}}},
			{ContactPerson: &people.Person{Names: []*people.Name{{UnstructuredName: "Contact Two"}}}},
		},
		ReadMask: personFields,
	}

	resp, err := svc.People.BatchCreateContacts(req).Do()
	if err != nil {
		t.Fatalf("failed to batch create contacts: %v", err)
	}

	if len(resp.CreatedPeople) != 2 {
		t.Errorf("expected 2 created people, got %d", len(resp.CreatedPeople))
	}
	if resp.CreatedPeople[0].Person.ResourceName != "people/c100" {
		t.Errorf("expected 'people/c100', got '%s'", resp.CreatedPeople[0].Person.ResourceName)
	}
}

// TestContactsBatchUpdate_Success tests batch updating contacts
func TestContactsBatchUpdate_Success(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/people:batchUpdateContacts": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				t.Errorf("expected POST, got %s", r.Method)
			}

			json.NewEncoder(w).Encode(&people.BatchUpdateContactsResponse{
				UpdateResult: map[string]people.PersonResponse{
					"people/c100": {
						Person: &people.Person{
							ResourceName: "people/c100",
							Names:        []*people.Name{{DisplayName: "Updated One"}},
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

	req := &people.BatchUpdateContactsRequest{
		Contacts: map[string]people.Person{
			"people/c100": {
				Etag:  "etag1",
				Names: []*people.Name{{UnstructuredName: "Updated One"}},
			},
		},
		UpdateMask: "names",
		ReadMask:   personFields,
	}

	resp, err := svc.People.BatchUpdateContacts(req).Do()
	if err != nil {
		t.Fatalf("failed to batch update contacts: %v", err)
	}

	if len(resp.UpdateResult) != 1 {
		t.Errorf("expected 1 update result, got %d", len(resp.UpdateResult))
	}
	result, ok := resp.UpdateResult["people/c100"]
	if !ok {
		t.Fatal("expected people/c100 in update results")
	}
	if result.Person.Names[0].DisplayName != "Updated One" {
		t.Errorf("expected 'Updated One', got '%s'", result.Person.Names[0].DisplayName)
	}
}

// TestContactsBatchDelete_Success tests batch deleting contacts
func TestContactsBatchDelete_Success(t *testing.T) {
	deleteCalled := false

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/people:batchDeleteContacts": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				t.Errorf("expected POST, got %s", r.Method)
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

	req := &people.BatchDeleteContactsRequest{
		ResourceNames: []string{"people/c100", "people/c101"},
	}

	_, err = svc.People.BatchDeleteContacts(req).Do()
	if err != nil {
		t.Fatalf("failed to batch delete contacts: %v", err)
	}

	if !deleteCalled {
		t.Error("batch delete endpoint was not called")
	}
}

// TestContactsDirectory_Success tests listing directory people
func TestContactsDirectory_Success(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/people:listDirectoryPeople": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "GET" {
				t.Errorf("expected GET, got %s", r.Method)
			}

			json.NewEncoder(w).Encode(&people.ListDirectoryPeopleResponse{
				People: []*people.Person{
					{
						ResourceName: "people/d1",
						Names:        []*people.Name{{DisplayName: "Directory User"}},
						EmailAddresses: []*people.EmailAddress{
							{Value: "user@company.com"},
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

	resp, err := svc.People.ListDirectoryPeople().
		ReadMask(personFields).
		Sources("DIRECTORY_SOURCE_TYPE_DOMAIN_PROFILE").
		PageSize(50).
		Do()
	if err != nil {
		t.Fatalf("failed to list directory people: %v", err)
	}

	if len(resp.People) != 1 {
		t.Errorf("expected 1 person, got %d", len(resp.People))
	}
	if resp.People[0].Names[0].DisplayName != "Directory User" {
		t.Errorf("expected 'Directory User', got '%s'", resp.People[0].Names[0].DisplayName)
	}
}

// TestContactsDirectorySearch_Success tests searching directory people
func TestContactsDirectorySearch_Success(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/people:searchDirectoryPeople": func(w http.ResponseWriter, r *http.Request) {
			query := r.URL.Query().Get("query")
			if query != "John" {
				t.Errorf("expected query 'John', got '%s'", query)
			}

			json.NewEncoder(w).Encode(&people.SearchDirectoryPeopleResponse{
				People: []*people.Person{
					{
						ResourceName: "people/d2",
						Names:        []*people.Name{{DisplayName: "John Directory"}},
						EmailAddresses: []*people.EmailAddress{
							{Value: "john@company.com"},
						},
					},
				},
				TotalSize: 1,
			})
		},
	}

	server := mockPeopleServer(t, handlers)
	defer server.Close()

	svc, err := people.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create people service: %v", err)
	}

	resp, err := svc.People.SearchDirectoryPeople().
		Query("John").
		ReadMask(personFields).
		Sources("DIRECTORY_SOURCE_TYPE_DOMAIN_PROFILE").
		PageSize(50).
		Do()
	if err != nil {
		t.Fatalf("failed to search directory people: %v", err)
	}

	if len(resp.People) != 1 {
		t.Errorf("expected 1 person, got %d", len(resp.People))
	}
	if resp.People[0].Names[0].DisplayName != "John Directory" {
		t.Errorf("expected 'John Directory', got '%s'", resp.People[0].Names[0].DisplayName)
	}
}

// TestContactsPhoto_Success tests updating a contact photo
func TestContactsPhoto_Success(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/people/c123:updateContactPhoto": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "PATCH" {
				t.Errorf("expected PATCH, got %s", r.Method)
			}

			var req people.UpdateContactPhotoRequest
			json.NewDecoder(r.Body).Decode(&req)

			if req.PhotoBytes == "" {
				t.Error("expected non-empty photo bytes")
			}

			json.NewEncoder(w).Encode(&people.UpdateContactPhotoResponse{
				Person: &people.Person{
					ResourceName: "people/c123",
					Names:        []*people.Name{{DisplayName: "John Doe"}},
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

	req := &people.UpdateContactPhotoRequest{
		PhotoBytes:   "dGVzdHBob3Rv", // base64 of "testphoto"
		PersonFields: personFields,
	}

	resp, err := svc.People.UpdateContactPhoto("people/c123", req).Do()
	if err != nil {
		t.Fatalf("failed to update contact photo: %v", err)
	}

	if resp.Person == nil {
		t.Fatal("expected person in response")
	}
	if resp.Person.ResourceName != "people/c123" {
		t.Errorf("expected 'people/c123', got '%s'", resp.Person.ResourceName)
	}
}

// TestContactsDeletePhoto_Success tests deleting a contact photo
func TestContactsDeletePhoto_Success(t *testing.T) {
	deleteCalled := false

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/people/c123:deleteContactPhoto": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "DELETE" {
				t.Errorf("expected DELETE, got %s", r.Method)
			}
			deleteCalled = true
			json.NewEncoder(w).Encode(&people.DeleteContactPhotoResponse{})
		},
	}

	server := mockPeopleServer(t, handlers)
	defer server.Close()

	svc, err := people.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create people service: %v", err)
	}

	_, err = svc.People.DeleteContactPhoto("people/c123").Do()
	if err != nil {
		t.Fatalf("failed to delete contact photo: %v", err)
	}

	if !deleteCalled {
		t.Error("delete photo endpoint was not called")
	}
}

// TestContactsResolve_Success tests resolving multiple contacts
func TestContactsResolve_Success(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/people:batchGet": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "GET" {
				t.Errorf("expected GET, got %s", r.Method)
			}

			resourceNames := r.URL.Query()["resourceNames"]
			if len(resourceNames) != 2 {
				t.Errorf("expected 2 resource names, got %d", len(resourceNames))
			}

			json.NewEncoder(w).Encode(&people.GetPeopleResponse{
				Responses: []*people.PersonResponse{
					{
						Person: &people.Person{
							ResourceName: "people/c1",
							Names:        []*people.Name{{DisplayName: "John Doe"}},
						},
						RequestedResourceName: "people/c1",
					},
					{
						Person: &people.Person{
							ResourceName: "people/c2",
							Names:        []*people.Name{{DisplayName: "Jane Smith"}},
						},
						RequestedResourceName: "people/c2",
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

	resp, err := svc.People.GetBatchGet().
		ResourceNames("people/c1", "people/c2").
		PersonFields(personFields).
		Do()
	if err != nil {
		t.Fatalf("failed to resolve contacts: %v", err)
	}

	if len(resp.Responses) != 2 {
		t.Errorf("expected 2 responses, got %d", len(resp.Responses))
	}
	if resp.Responses[0].Person.Names[0].DisplayName != "John Doe" {
		t.Errorf("expected 'John Doe', got '%s'", resp.Responses[0].Person.Names[0].DisplayName)
	}
	if resp.Responses[1].Person.Names[0].DisplayName != "Jane Smith" {
		t.Errorf("expected 'Jane Smith', got '%s'", resp.Responses[1].Person.Names[0].DisplayName)
	}
}

// TestFormatPerson tests the formatPerson helper
func TestFormatPerson(t *testing.T) {
	person := &people.Person{
		ResourceName: "people/c1",
		Etag:         "abc123",
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
	if result["etag"] != "abc123" {
		t.Errorf("expected etag 'abc123', got '%v'", result["etag"])
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
	if _, ok := result["etag"]; ok {
		t.Error("expected no etag field for person without etag")
	}
}
