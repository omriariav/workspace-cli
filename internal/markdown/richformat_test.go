package markdown

import (
	"testing"
)

func TestParseRichFormat_ValidRequests(t *testing.T) {
	input := `[{"insertText":{"location":{"index":1},"text":"Hello"}}]`
	requests, err := ParseRichFormat(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(requests))
	}
	if requests[0].InsertText == nil {
		t.Fatal("expected InsertText request")
	}
	if requests[0].InsertText.Text != "Hello" {
		t.Errorf("expected text 'Hello', got '%s'", requests[0].InsertText.Text)
	}
}

func TestParseRichFormat_MultipleRequests(t *testing.T) {
	input := `[
		{"insertText":{"location":{"index":1},"text":"Hello World"}},
		{"updateTextStyle":{"range":{"startIndex":1,"endIndex":6},"textStyle":{"bold":true},"fields":"bold"}}
	]`
	requests, err := ParseRichFormat(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(requests) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(requests))
	}
}

func TestParseRichFormat_InvalidJSON(t *testing.T) {
	_, err := ParseRichFormat("not json")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseRichFormat_EmptyArray(t *testing.T) {
	_, err := ParseRichFormat("[]")
	if err == nil {
		t.Error("expected error for empty array")
	}
}

func TestParseRichFormat_EmptyString(t *testing.T) {
	_, err := ParseRichFormat("")
	if err == nil {
		t.Error("expected error for empty string")
	}
}
