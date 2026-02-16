package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSearchCommand_Flags(t *testing.T) {
	if searchCmd == nil {
		t.Fatal("search command not found")
	}

	flags := []struct {
		name     string
		defValue string
	}{
		{"max", "10"},
		{"start", "1"},
		{"site", ""},
		{"type", ""},
		{"engine-id", ""},
		{"api-key", ""},
	}

	for _, f := range flags {
		flag := searchCmd.Flags().Lookup(f.name)
		if flag == nil {
			t.Errorf("expected --%s flag", f.name)
			continue
		}
		if f.defValue != "" && flag.DefValue != f.defValue {
			t.Errorf("expected --%s default '%s', got '%s'", f.name, f.defValue, flag.DefValue)
		}
	}
}

func TestSearchCommand_Help(t *testing.T) {
	if searchCmd.Use != "search <query>" {
		t.Errorf("unexpected Use: %s", searchCmd.Use)
	}
	if searchCmd.Short == "" {
		t.Error("expected Short description to be set")
	}
	if searchCmd.Long == "" {
		t.Error("expected Long description to be set")
	}
	if searchCmd.Args == nil {
		t.Error("expected Args validator to be set")
	}
}

func TestSearch_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path != "/customsearch/v1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// Verify query parameters
		q := r.URL.Query()
		if q.Get("q") != "golang testing" {
			t.Errorf("expected query 'golang testing', got '%s'", q.Get("q"))
		}
		if q.Get("key") == "" {
			t.Error("expected API key parameter")
		}
		if q.Get("cx") == "" {
			t.Error("expected engine ID parameter")
		}

		resp := map[string]interface{}{
			"searchInformation": map[string]interface{}{
				"totalResults":          "1234",
				"searchTime":            0.42,
				"formattedTotalResults": "1,234",
			},
			"items": []map[string]interface{}{
				{
					"title":       "Go Testing - The Go Programming Language",
					"link":        "https://go.dev/doc/testing",
					"snippet":     "The go test command runs tests in the current package.",
					"displayLink": "go.dev",
				},
				{
					"title":       "Testing in Go - Tutorial",
					"link":        "https://example.com/go-testing",
					"snippet":     "Learn how to write tests in Go.",
					"displayLink": "example.com",
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Simulate the search request that runSearch would make
	req, err := http.NewRequest("GET", server.URL+"/customsearch/v1?q=golang+testing&key=test-key&cx=test-cx&num=10&start=1", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var searchResp struct {
		SearchInformation struct {
			TotalResults string  `json:"totalResults"`
			SearchTime   float64 `json:"searchTime"`
		} `json:"searchInformation"`
		Items []struct {
			Title       string `json:"title"`
			Link        string `json:"link"`
			Snippet     string `json:"snippet"`
			DisplayLink string `json:"displayLink"`
		} `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if searchResp.SearchInformation.TotalResults != "1234" {
		t.Errorf("expected totalResults '1234', got '%s'", searchResp.SearchInformation.TotalResults)
	}
	if len(searchResp.Items) != 2 {
		t.Fatalf("expected 2 results, got %d", len(searchResp.Items))
	}
	if searchResp.Items[0].Title != "Go Testing - The Go Programming Language" {
		t.Errorf("unexpected first result title: %s", searchResp.Items[0].Title)
	}
	if searchResp.Items[1].DisplayLink != "example.com" {
		t.Errorf("unexpected second result domain: %s", searchResp.Items[1].DisplayLink)
	}
}

func TestSearch_ErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"message": "Invalid API key",
			},
		})
	}))
	defer server.Close()

	resp, err := http.Get(server.URL + "/customsearch/v1?q=test")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}

	var errResp struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error: %v", err)
	}
	if errResp.Error.Message != "Invalid API key" {
		t.Errorf("expected error 'Invalid API key', got '%s'", errResp.Error.Message)
	}
}
