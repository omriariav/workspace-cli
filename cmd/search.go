package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/omriariav/workspace-cli/internal/printer"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search with Google Custom Search",
	Long: `Searches using Google Programmable Search Engine (Custom Search).

Requires additional setup:
1. Create a Programmable Search Engine at https://programmablesearchengine.google.com/
2. Get an API key from Google Cloud Console
3. Set GWS_SEARCH_ENGINE_ID and GWS_SEARCH_API_KEY environment variables
   or add search_engine_id and search_api_key to your config file`,
	Args: cobra.ExactArgs(1),
	RunE: runSearch,
}

func init() {
	rootCmd.AddCommand(searchCmd)

	// Search flags
	searchCmd.Flags().Int64("max", 10, "Maximum number of results (1-10)")
	searchCmd.Flags().Int64("start", 1, "Start index for results (for pagination)")
	searchCmd.Flags().String("site", "", "Restrict search to a specific site")
	searchCmd.Flags().String("type", "", "Search type: image or empty for web")
	searchCmd.Flags().String("engine-id", "", "Search Engine ID (overrides config)")
	searchCmd.Flags().String("api-key", "", "API Key (overrides config)")

	viper.BindPFlag("search_engine_id", searchCmd.Flags().Lookup("engine-id"))
	viper.BindPFlag("search_api_key", searchCmd.Flags().Lookup("api-key"))
}

func runSearch(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())

	// Get credentials
	engineID := viper.GetString("search_engine_id")
	apiKey := viper.GetString("search_api_key")

	if engineID == "" {
		return p.PrintError(fmt.Errorf("missing Search Engine ID: set GWS_SEARCH_ENGINE_ID environment variable or use --engine-id flag"))
	}
	if apiKey == "" {
		return p.PrintError(fmt.Errorf("missing API Key: set GWS_SEARCH_API_KEY environment variable or use --api-key flag"))
	}

	query := args[0]
	maxResults, _ := cmd.Flags().GetInt64("max")
	start, _ := cmd.Flags().GetInt64("start")
	site, _ := cmd.Flags().GetString("site")
	searchType, _ := cmd.Flags().GetString("type")

	// Limit max results (API limit is 10 per request)
	if maxResults > 10 {
		maxResults = 10
	}

	// Build URL
	baseURL := "https://www.googleapis.com/customsearch/v1"
	params := url.Values{}
	params.Set("key", apiKey)
	params.Set("cx", engineID)
	params.Set("q", query)
	params.Set("num", fmt.Sprintf("%d", maxResults))
	params.Set("start", fmt.Sprintf("%d", start))

	if site != "" {
		params.Set("siteSearch", site)
	}
	if searchType == "image" {
		params.Set("searchType", "image")
	}

	// Make request
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"?"+params.Encode(), nil)
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to create request: %w", err))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return p.PrintError(fmt.Errorf("search request failed: %w", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		json.NewDecoder(resp.Body).Decode(&errResp)
		return p.PrintError(fmt.Errorf("search failed: %s", errResp.Error.Message))
	}

	// Parse response
	var searchResp struct {
		SearchInformation struct {
			TotalResults     string  `json:"totalResults"`
			SearchTime       float64 `json:"searchTime"`
			FormattedResults string  `json:"formattedTotalResults"`
		} `json:"searchInformation"`
		Items []struct {
			Title       string `json:"title"`
			Link        string `json:"link"`
			Snippet     string `json:"snippet"`
			DisplayLink string `json:"displayLink"`
			Image       struct {
				ContextLink   string `json:"contextLink"`
				Height        int    `json:"height"`
				Width         int    `json:"width"`
				ThumbnailLink string `json:"thumbnailLink"`
			} `json:"image,omitempty"`
		} `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return p.PrintError(fmt.Errorf("failed to parse response: %w", err))
	}

	// Format results
	results := make([]map[string]interface{}, 0, len(searchResp.Items))
	for _, item := range searchResp.Items {
		result := map[string]interface{}{
			"title":   item.Title,
			"link":    item.Link,
			"snippet": item.Snippet,
			"domain":  item.DisplayLink,
		}
		if searchType == "image" && item.Image.ThumbnailLink != "" {
			result["thumbnail"] = item.Image.ThumbnailLink
			result["width"] = item.Image.Width
			result["height"] = item.Image.Height
		}
		results = append(results, result)
	}

	return p.Print(map[string]interface{}{
		"query":         query,
		"total_results": searchResp.SearchInformation.TotalResults,
		"search_time":   searchResp.SearchInformation.SearchTime,
		"results":       results,
		"count":         len(results),
	})
}
