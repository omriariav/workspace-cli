package markdown

import (
	"encoding/json"
	"fmt"

	"google.golang.org/api/docs/v1"
)

// ParseRichFormat parses a JSON string into Google Docs API requests.
// The input should be a JSON array of request objects matching the
// Google Docs BatchUpdate request format.
func ParseRichFormat(jsonInput string) ([]*docs.Request, error) {
	var requests []*docs.Request
	if err := json.Unmarshal([]byte(jsonInput), &requests); err != nil {
		return nil, fmt.Errorf("invalid richformat JSON: %w", err)
	}
	if len(requests) == 0 {
		return nil, fmt.Errorf("richformat JSON must contain at least one request")
	}
	return requests, nil
}
