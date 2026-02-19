package cmd

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/api/forms/v1"
	"google.golang.org/api/option"
)

func TestFormsCommands_Flags(t *testing.T) {
	// Test info command
	infoCmd := findSubcommand(formsCmd, "info")
	if infoCmd == nil {
		t.Fatal("forms info command not found")
	}
	if infoCmd.Use != "info <form-id>" {
		t.Errorf("unexpected Use: %s", infoCmd.Use)
	}
	if infoCmd.Args == nil {
		t.Error("expected Args validator to be set")
	}

	// Test responses command
	respCmd := findSubcommand(formsCmd, "responses")
	if respCmd == nil {
		t.Fatal("forms responses command not found")
	}
	if respCmd.Use != "responses <form-id>" {
		t.Errorf("unexpected Use: %s", respCmd.Use)
	}
	if respCmd.Args == nil {
		t.Error("expected Args validator to be set")
	}
}

func TestFormsInfoCommand_Help(t *testing.T) {
	cmd := formsInfoCmd
	if cmd.Short == "" {
		t.Error("expected Short description to be set")
	}
	if cmd.Long == "" {
		t.Error("expected Long description to be set")
	}
}

func TestFormsResponsesCommand_Help(t *testing.T) {
	cmd := formsResponsesCmd
	if cmd.Short == "" {
		t.Error("expected Short description to be set")
	}
	if cmd.Long == "" {
		t.Error("expected Long description to be set")
	}
}

func TestFormsGetCommand_Flags(t *testing.T) {
	cmd := findSubcommand(formsCmd, "get")
	if cmd == nil {
		t.Fatal("forms get command not found")
	}
	if cmd.Use != "get <form-id>" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}
	if cmd.Args == nil {
		t.Error("expected Args validator to be set")
	}
}

func TestFormsGetCommand_Help(t *testing.T) {
	cmd := formsGetCmd
	if cmd.Short == "" {
		t.Error("expected Short description to be set")
	}
	if cmd.Long == "" {
		t.Error("expected Long description to be set")
	}
}

func TestFormsResponseCommand_Flags(t *testing.T) {
	cmd := findSubcommand(formsCmd, "response")
	if cmd == nil {
		t.Fatal("forms response command not found")
	}
	if cmd.Use != "response <form-id>" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}
	if cmd.Args == nil {
		t.Error("expected Args validator to be set")
	}

	responseIDFlag := cmd.Flags().Lookup("response-id")
	if responseIDFlag == nil {
		t.Error("expected --response-id flag")
	}
}

func TestFormsResponseCommand_Help(t *testing.T) {
	cmd := formsResponseCmd
	if cmd.Short == "" {
		t.Error("expected Short description to be set")
	}
	if cmd.Long == "" {
		t.Error("expected Long description to be set")
	}
}

func TestFormsCreateCommand_Flags(t *testing.T) {
	cmd := findSubcommand(formsCmd, "create")
	if cmd == nil {
		t.Fatal("forms create command not found")
	}
	if cmd.Use != "create" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}

	titleFlag := cmd.Flags().Lookup("title")
	if titleFlag == nil {
		t.Error("expected --title flag")
	}

	descFlag := cmd.Flags().Lookup("description")
	if descFlag == nil {
		t.Error("expected --description flag")
	}
}

func TestFormsCreateCommand_Help(t *testing.T) {
	cmd := formsCreateCmd
	if cmd.Short == "" {
		t.Error("expected Short description to be set")
	}
	if cmd.Long == "" {
		t.Error("expected Long description to be set")
	}
}

func TestFormsUpdateCommand_Flags(t *testing.T) {
	cmd := findSubcommand(formsCmd, "update")
	if cmd == nil {
		t.Fatal("forms update command not found")
	}
	if cmd.Use != "update <form-id>" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}
	if cmd.Args == nil {
		t.Error("expected Args validator to be set")
	}

	titleFlag := cmd.Flags().Lookup("title")
	if titleFlag == nil {
		t.Error("expected --title flag")
	}

	descFlag := cmd.Flags().Lookup("description")
	if descFlag == nil {
		t.Error("expected --description flag")
	}

	fileFlag := cmd.Flags().Lookup("file")
	if fileFlag == nil {
		t.Error("expected --file flag")
	}
}

func TestFormsUpdateCommand_Help(t *testing.T) {
	cmd := formsUpdateCmd
	if cmd.Short == "" {
		t.Error("expected Short description to be set")
	}
	if cmd.Long == "" {
		t.Error("expected Long description to be set")
	}
}

// mockFormsServer creates a test server that mocks Forms API responses
func mockFormsServer(t *testing.T, handlers map[string]func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
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

func TestFormsInfo_MockServer(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/forms/form-123": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "GET" {
				t.Errorf("expected GET, got %s", r.Method)
			}
			resp := &forms.Form{
				FormId: "form-123",
				Info: &forms.Info{
					Title:         "Feedback Survey",
					DocumentTitle: "Feedback Survey Doc",
					Description:   "Please provide your feedback",
				},
				ResponderUri: "https://docs.google.com/forms/d/e/form-123/viewform",
				Items: []*forms.Item{
					{
						ItemId: "q1",
						Title:  "How was your experience?",
						QuestionItem: &forms.QuestionItem{
							Question: &forms.Question{
								QuestionId: "q1",
								Required:   true,
								ChoiceQuestion: &forms.ChoiceQuestion{
									Type: "RADIO",
									Options: []*forms.Option{
										{Value: "Great"},
										{Value: "Good"},
										{Value: "Poor"},
									},
								},
							},
						},
					},
					{
						ItemId: "q2",
						Title:  "Any comments?",
						QuestionItem: &forms.QuestionItem{
							Question: &forms.Question{
								QuestionId: "q2",
								TextQuestion: &forms.TextQuestion{
									Paragraph: true,
								},
							},
						},
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		},
	}

	server := mockFormsServer(t, handlers)
	defer server.Close()

	svc, err := forms.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create forms service: %v", err)
	}

	form, err := svc.Forms.Get("form-123").Do()
	if err != nil {
		t.Fatalf("failed to get form: %v", err)
	}

	if form.FormId != "form-123" {
		t.Errorf("expected form ID 'form-123', got '%s'", form.FormId)
	}
	if form.Info.Title != "Feedback Survey" {
		t.Errorf("expected title 'Feedback Survey', got '%s'", form.Info.Title)
	}
	if len(form.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(form.Items))
	}
	if form.Items[0].QuestionItem.Question.ChoiceQuestion.Type != "RADIO" {
		t.Errorf("expected first question type 'RADIO', got '%s'", form.Items[0].QuestionItem.Question.ChoiceQuestion.Type)
	}
	if !form.Items[1].QuestionItem.Question.TextQuestion.Paragraph {
		t.Error("expected second question to be paragraph text")
	}
}

func TestFormsResponses_MockServer(t *testing.T) {
	formRequested := false
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/forms/form-123": func(w http.ResponseWriter, r *http.Request) {
			formRequested = true
			resp := &forms.Form{
				FormId: "form-123",
				Info: &forms.Info{
					Title:         "Feedback Survey",
					DocumentTitle: "Feedback Survey Doc",
				},
				Items: []*forms.Item{
					{
						ItemId: "q1",
						Title:  "Rating",
						QuestionItem: &forms.QuestionItem{
							Question: &forms.Question{
								QuestionId: "q1",
							},
						},
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		},
		"/v1/forms/form-123/responses": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "GET" {
				t.Errorf("expected GET, got %s", r.Method)
			}
			resp := &forms.ListFormResponsesResponse{
				Responses: []*forms.FormResponse{
					{
						ResponseId:        "resp-1",
						CreateTime:        "2026-02-15T10:00:00Z",
						LastSubmittedTime: "2026-02-15T10:00:00Z",
						RespondentEmail:   "user@example.com",
						Answers: map[string]forms.Answer{
							"q1": {
								TextAnswers: &forms.TextAnswers{
									Answers: []*forms.TextAnswer{
										{Value: "Great"},
									},
								},
							},
						},
					},
					{
						ResponseId:        "resp-2",
						CreateTime:        "2026-02-16T09:00:00Z",
						LastSubmittedTime: "2026-02-16T09:00:00Z",
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		},
	}

	server := mockFormsServer(t, handlers)
	defer server.Close()

	svc, err := forms.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create forms service: %v", err)
	}

	// Test listing responses
	resp, err := svc.Forms.Responses.List("form-123").Do()
	if err != nil {
		t.Fatalf("failed to list responses: %v", err)
	}

	if len(resp.Responses) != 2 {
		t.Fatalf("expected 2 responses, got %d", len(resp.Responses))
	}
	if resp.Responses[0].ResponseId != "resp-1" {
		t.Errorf("expected first response ID 'resp-1', got '%s'", resp.Responses[0].ResponseId)
	}
	if resp.Responses[0].RespondentEmail != "user@example.com" {
		t.Errorf("expected email 'user@example.com', got '%s'", resp.Responses[0].RespondentEmail)
	}

	// Also fetch the form to verify the flow
	form, err := svc.Forms.Get("form-123").Do()
	if err != nil {
		t.Fatalf("failed to get form: %v", err)
	}
	if !formRequested {
		t.Error("expected form to be requested")
	}
	if form.Info.Title != "Feedback Survey" {
		t.Errorf("expected form title 'Feedback Survey', got '%s'", form.Info.Title)
	}
}

func TestFormsCreate_MockServer(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/forms": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				t.Errorf("expected POST, got %s", r.Method)
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("failed to read request body: %v", err)
			}

			var req forms.Form
			if err := json.Unmarshal(body, &req); err != nil {
				t.Fatalf("failed to parse request body: %v", err)
			}

			if req.Info == nil || req.Info.Title != "Test Form" {
				t.Errorf("expected title 'Test Form', got %v", req.Info)
			}

			if req.Info.DocumentTitle != "Test Form" {
				t.Errorf("expected document title 'Test Form', got '%s'", req.Info.DocumentTitle)
			}

			resp := &forms.Form{
				FormId: "new-form-456",
				Info: &forms.Info{
					Title:         "Test Form",
					DocumentTitle: "Test Form",
					Description:   "A test form",
				},
				ResponderUri: "https://docs.google.com/forms/d/e/new-form-456/viewform",
			}
			json.NewEncoder(w).Encode(resp)
		},
	}

	server := mockFormsServer(t, handlers)
	defer server.Close()

	svc, err := forms.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create forms service: %v", err)
	}

	newForm := &forms.Form{
		Info: &forms.Info{
			Title:         "Test Form",
			DocumentTitle: "Test Form",
			Description:   "A test form",
		},
	}

	form, err := svc.Forms.Create(newForm).Do()
	if err != nil {
		t.Fatalf("failed to create form: %v", err)
	}

	if form.FormId != "new-form-456" {
		t.Errorf("expected form ID 'new-form-456', got '%s'", form.FormId)
	}
	if form.Info.Title != "Test Form" {
		t.Errorf("expected title 'Test Form', got '%s'", form.Info.Title)
	}
	if form.ResponderUri == "" {
		t.Error("expected responder URI to be set")
	}
}

func TestFormsUpdate_MockServer(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/forms/form-123:batchUpdate": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				t.Errorf("expected POST, got %s", r.Method)
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("failed to read request body: %v", err)
			}

			var req forms.BatchUpdateFormRequest
			if err := json.Unmarshal(body, &req); err != nil {
				t.Fatalf("failed to parse request body: %v", err)
			}

			if len(req.Requests) == 0 {
				t.Error("expected at least one request in batch update")
			}

			resp := &forms.BatchUpdateFormResponse{
				Replies: []*forms.Response{
					{},
				},
			}
			json.NewEncoder(w).Encode(resp)
		},
		"/v1/forms/form-123": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "GET" {
				t.Errorf("expected GET for form fetch, got %s", r.Method)
			}
			resp := &forms.Form{
				FormId: "form-123",
				Info: &forms.Info{
					Title:         "Updated Title",
					DocumentTitle: "Updated Title",
					Description:   "Updated description",
				},
				ResponderUri: "https://docs.google.com/forms/d/e/form-123/viewform",
			}
			json.NewEncoder(w).Encode(resp)
		},
	}

	server := mockFormsServer(t, handlers)
	defer server.Close()

	svc, err := forms.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create forms service: %v", err)
	}

	// Test batch update with title change
	batchReq := &forms.BatchUpdateFormRequest{
		Requests: []*forms.Request{
			{
				UpdateFormInfo: &forms.UpdateFormInfoRequest{
					Info: &forms.Info{
						Title: "Updated Title",
					},
					UpdateMask: "title",
				},
			},
		},
	}

	resp, err := svc.Forms.BatchUpdate("form-123", batchReq).Do()
	if err != nil {
		t.Fatalf("failed to batch update form: %v", err)
	}

	if len(resp.Replies) != 1 {
		t.Errorf("expected 1 reply, got %d", len(resp.Replies))
	}

	// Verify updated form
	form, err := svc.Forms.Get("form-123").Do()
	if err != nil {
		t.Fatalf("failed to get updated form: %v", err)
	}

	if form.Info.Title != "Updated Title" {
		t.Errorf("expected title 'Updated Title', got '%s'", form.Info.Title)
	}
}

func TestFormsResponse_MockServer(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/v1/forms/form-123": func(w http.ResponseWriter, r *http.Request) {
			resp := &forms.Form{
				FormId: "form-123",
				Info: &forms.Info{
					Title:         "Feedback Survey",
					DocumentTitle: "Feedback Survey Doc",
				},
				Items: []*forms.Item{
					{
						ItemId: "q1",
						Title:  "Rating",
						QuestionItem: &forms.QuestionItem{
							Question: &forms.Question{
								QuestionId: "q1",
							},
						},
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		},
		"/v1/forms/form-123/responses/resp-1": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "GET" {
				t.Errorf("expected GET, got %s", r.Method)
			}
			resp := &forms.FormResponse{
				ResponseId:        "resp-1",
				CreateTime:        "2026-02-15T10:00:00Z",
				LastSubmittedTime: "2026-02-15T10:00:00Z",
				RespondentEmail:   "user@example.com",
				Answers: map[string]forms.Answer{
					"q1": {
						TextAnswers: &forms.TextAnswers{
							Answers: []*forms.TextAnswer{
								{Value: "Excellent"},
							},
						},
					},
				},
			}
			json.NewEncoder(w).Encode(resp)
		},
	}

	server := mockFormsServer(t, handlers)
	defer server.Close()

	svc, err := forms.NewService(context.Background(), option.WithoutAuthentication(), option.WithEndpoint(server.URL))
	if err != nil {
		t.Fatalf("failed to create forms service: %v", err)
	}

	// Test getting a single response
	resp, err := svc.Forms.Responses.Get("form-123", "resp-1").Do()
	if err != nil {
		t.Fatalf("failed to get response: %v", err)
	}

	if resp.ResponseId != "resp-1" {
		t.Errorf("expected response ID 'resp-1', got '%s'", resp.ResponseId)
	}
	if resp.RespondentEmail != "user@example.com" {
		t.Errorf("expected email 'user@example.com', got '%s'", resp.RespondentEmail)
	}
	if resp.Answers["q1"].TextAnswers.Answers[0].Value != "Excellent" {
		t.Errorf("expected answer 'Excellent', got '%s'", resp.Answers["q1"].TextAnswers.Answers[0].Value)
	}
}
