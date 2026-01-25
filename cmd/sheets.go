package cmd

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/omriariav/workspace-cli/internal/client"
	"github.com/omriariav/workspace-cli/internal/printer"
	"github.com/spf13/cobra"
	"google.golang.org/api/sheets/v4"
)

var sheetsCmd = &cobra.Command{
	Use:   "sheets",
	Short: "Manage Google Sheets",
	Long:  "Commands for interacting with Google Sheets spreadsheets.",
}

var sheetsInfoCmd = &cobra.Command{
	Use:   "info <spreadsheet-id>",
	Short: "Get spreadsheet info",
	Long:  "Gets metadata about a Google Sheets spreadsheet.",
	Args:  cobra.ExactArgs(1),
	RunE:  runSheetsInfo,
}

var sheetsReadCmd = &cobra.Command{
	Use:   "read <spreadsheet-id> <range>",
	Short: "Read cell values",
	Long: `Reads cell values from a spreadsheet range.

Range format examples:
  Sheet1!A1:D10    - Specific range in Sheet1
  Sheet1!A:D       - Columns A through D in Sheet1
  Sheet1           - All data in Sheet1
  A1:D10           - Range in first sheet`,
	Args: cobra.ExactArgs(2),
	RunE: runSheetsRead,
}

var sheetsListCmd = &cobra.Command{
	Use:   "list <spreadsheet-id>",
	Short: "List sheets",
	Long:  "Lists all sheets in a spreadsheet.",
	Args:  cobra.ExactArgs(1),
	RunE:  runSheetsList,
}

var sheetsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new spreadsheet",
	Long:  "Creates a new Google Sheets spreadsheet with optional sheet names.",
	RunE:  runSheetsCreate,
}

var sheetsWriteCmd = &cobra.Command{
	Use:   "write <spreadsheet-id> <range>",
	Short: "Write values to cells",
	Long: `Writes values to a range of cells in a spreadsheet.

Range format examples:
  Sheet1!A1:D10    - Specific range in Sheet1
  Sheet1!A1        - Single cell in Sheet1
  A1:D10           - Range in first sheet

Values format:
  --values "a,b,c"           - Single row
  --values "a,b,c;d,e,f"     - Multiple rows (semicolon-separated)
  --values-json '[["a","b"],["c","d"]]'  - JSON array format`,
	Args: cobra.ExactArgs(2),
	RunE: runSheetsWrite,
}

var sheetsAppendCmd = &cobra.Command{
	Use:   "append <spreadsheet-id> <range>",
	Short: "Append rows to a sheet",
	Long: `Appends rows after the last row with data in a range.

The range is used to find the table to append to. Data will be added
after the last row of the table.

Values format:
  --values "a,b,c"           - Single row
  --values "a,b,c;d,e,f"     - Multiple rows (semicolon-separated)
  --values-json '[["a","b"],["c","d"]]'  - JSON array format`,
	Args: cobra.ExactArgs(2),
	RunE: runSheetsAppend,
}

func init() {
	rootCmd.AddCommand(sheetsCmd)
	sheetsCmd.AddCommand(sheetsInfoCmd)
	sheetsCmd.AddCommand(sheetsReadCmd)
	sheetsCmd.AddCommand(sheetsListCmd)
	sheetsCmd.AddCommand(sheetsCreateCmd)
	sheetsCmd.AddCommand(sheetsWriteCmd)
	sheetsCmd.AddCommand(sheetsAppendCmd)

	// Read flags
	sheetsReadCmd.Flags().String("output-format", "json", "Output format: json or csv")
	sheetsReadCmd.Flags().Bool("headers", true, "Treat first row as headers (for json output)")

	// Create flags
	sheetsCreateCmd.Flags().String("title", "", "Spreadsheet title (required)")
	sheetsCreateCmd.Flags().StringSlice("sheet-names", nil, "Sheet names (comma-separated, default: Sheet1)")
	sheetsCreateCmd.MarkFlagRequired("title")

	// Write flags
	sheetsWriteCmd.Flags().String("values", "", "Values to write (comma-separated, semicolon for rows)")
	sheetsWriteCmd.Flags().String("values-json", "", "Values as JSON array (e.g., '[[\"a\",\"b\"],[\"c\",\"d\"]]')")

	// Append flags
	sheetsAppendCmd.Flags().String("values", "", "Values to append (comma-separated, semicolon for rows)")
	sheetsAppendCmd.Flags().String("values-json", "", "Values as JSON array")
}

func runSheetsInfo(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Sheets()
	if err != nil {
		return p.PrintError(err)
	}

	spreadsheetID := args[0]

	spreadsheet, err := svc.Spreadsheets.Get(spreadsheetID).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get spreadsheet: %w", err))
	}

	sheets := make([]map[string]interface{}, 0, len(spreadsheet.Sheets))
	for _, sheet := range spreadsheet.Sheets {
		sheetInfo := map[string]interface{}{
			"id":    sheet.Properties.SheetId,
			"title": sheet.Properties.Title,
			"index": sheet.Properties.Index,
		}
		if sheet.Properties.GridProperties != nil {
			sheetInfo["rows"] = sheet.Properties.GridProperties.RowCount
			sheetInfo["columns"] = sheet.Properties.GridProperties.ColumnCount
		}
		sheets = append(sheets, sheetInfo)
	}

	return p.Print(map[string]interface{}{
		"id":          spreadsheet.SpreadsheetId,
		"title":       spreadsheet.Properties.Title,
		"locale":      spreadsheet.Properties.Locale,
		"timezone":    spreadsheet.Properties.TimeZone,
		"sheets":      sheets,
		"sheet_count": len(sheets),
		"url":         spreadsheet.SpreadsheetUrl,
	})
}

func runSheetsRead(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Sheets()
	if err != nil {
		return p.PrintError(err)
	}

	spreadsheetID := args[0]
	rangeStr := args[1]
	outputFormat, _ := cmd.Flags().GetString("output-format")
	useHeaders, _ := cmd.Flags().GetBool("headers")

	resp, err := svc.Spreadsheets.Values.Get(spreadsheetID, rangeStr).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to read range: %w", err))
	}

	if len(resp.Values) == 0 {
		return p.Print(map[string]interface{}{
			"range": resp.Range,
			"data":  []interface{}{},
			"rows":  0,
		})
	}

	// CSV output
	if outputFormat == "csv" {
		var builder strings.Builder
		writer := csv.NewWriter(&builder)

		for _, row := range resp.Values {
			record := make([]string, len(row))
			for i, cell := range row {
				record[i] = fmt.Sprintf("%v", cell)
			}
			writer.Write(record)
		}
		writer.Flush()

		return p.Print(map[string]interface{}{
			"range": resp.Range,
			"csv":   builder.String(),
			"rows":  len(resp.Values),
		})
	}

	// JSON output
	if useHeaders && len(resp.Values) > 1 {
		// Use first row as headers
		headers := make([]string, len(resp.Values[0]))
		for i, cell := range resp.Values[0] {
			headers[i] = fmt.Sprintf("%v", cell)
		}

		data := make([]map[string]interface{}, 0, len(resp.Values)-1)
		for _, row := range resp.Values[1:] {
			rowMap := make(map[string]interface{})
			for i, cell := range row {
				if i < len(headers) {
					rowMap[headers[i]] = cell
				}
			}
			data = append(data, rowMap)
		}

		return p.Print(map[string]interface{}{
			"range":   resp.Range,
			"headers": headers,
			"data":    data,
			"rows":    len(data),
		})
	}

	// Raw values
	return p.Print(map[string]interface{}{
		"range": resp.Range,
		"data":  resp.Values,
		"rows":  len(resp.Values),
	})
}

func runSheetsList(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Sheets()
	if err != nil {
		return p.PrintError(err)
	}

	spreadsheetID := args[0]

	spreadsheet, err := svc.Spreadsheets.Get(spreadsheetID).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get spreadsheet: %w", err))
	}

	sheets := make([]map[string]interface{}, 0, len(spreadsheet.Sheets))
	for _, sheet := range spreadsheet.Sheets {
		sheetInfo := map[string]interface{}{
			"id":    sheet.Properties.SheetId,
			"title": sheet.Properties.Title,
			"index": sheet.Properties.Index,
		}
		if sheet.Properties.GridProperties != nil {
			sheetInfo["rows"] = sheet.Properties.GridProperties.RowCount
			sheetInfo["columns"] = sheet.Properties.GridProperties.ColumnCount
		}
		sheets = append(sheets, sheetInfo)
	}

	return p.Print(map[string]interface{}{
		"sheets": sheets,
		"count":  len(sheets),
	})
}

func runSheetsCreate(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Sheets()
	if err != nil {
		return p.PrintError(err)
	}

	title, _ := cmd.Flags().GetString("title")
	sheetNames, _ := cmd.Flags().GetStringSlice("sheet-names")

	// Build spreadsheet with sheets
	spreadsheet := &sheets.Spreadsheet{
		Properties: &sheets.SpreadsheetProperties{
			Title: title,
		},
	}

	// Add custom sheets if specified
	if len(sheetNames) > 0 {
		spreadsheet.Sheets = make([]*sheets.Sheet, len(sheetNames))
		for i, name := range sheetNames {
			spreadsheet.Sheets[i] = &sheets.Sheet{
				Properties: &sheets.SheetProperties{
					Title: name,
					Index: int64(i),
				},
			}
		}
	}

	created, err := svc.Spreadsheets.Create(spreadsheet).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to create spreadsheet: %w", err))
	}

	// Get sheet names from response
	createdSheets := make([]string, len(created.Sheets))
	for i, sheet := range created.Sheets {
		createdSheets[i] = sheet.Properties.Title
	}

	return p.Print(map[string]interface{}{
		"status":      "created",
		"id":          created.SpreadsheetId,
		"title":       created.Properties.Title,
		"sheets":      createdSheets,
		"sheet_count": len(createdSheets),
		"url":         created.SpreadsheetUrl,
	})
}

func runSheetsWrite(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Sheets()
	if err != nil {
		return p.PrintError(err)
	}

	spreadsheetID := args[0]
	rangeStr := args[1]

	values, err := parseValues(cmd)
	if err != nil {
		return p.PrintError(err)
	}

	if len(values) == 0 {
		return p.PrintError(fmt.Errorf("no values provided; use --values or --values-json"))
	}

	valueRange := &sheets.ValueRange{
		Values: values,
	}

	resp, err := svc.Spreadsheets.Values.Update(spreadsheetID, rangeStr, valueRange).
		ValueInputOption("USER_ENTERED").
		Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to write values: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":        "written",
		"spreadsheet":   resp.SpreadsheetId,
		"range":         resp.UpdatedRange,
		"rows_updated":  resp.UpdatedRows,
		"cells_updated": resp.UpdatedCells,
	})
}

func runSheetsAppend(cmd *cobra.Command, args []string) error {
	p := printer.New(os.Stdout, GetFormat())
	ctx := context.Background()

	factory, err := client.NewFactory(ctx)
	if err != nil {
		return p.PrintError(err)
	}

	svc, err := factory.Sheets()
	if err != nil {
		return p.PrintError(err)
	}

	spreadsheetID := args[0]
	rangeStr := args[1]

	values, err := parseValues(cmd)
	if err != nil {
		return p.PrintError(err)
	}

	if len(values) == 0 {
		return p.PrintError(fmt.Errorf("no values provided; use --values or --values-json"))
	}

	valueRange := &sheets.ValueRange{
		Values: values,
	}

	resp, err := svc.Spreadsheets.Values.Append(spreadsheetID, rangeStr, valueRange).
		ValueInputOption("USER_ENTERED").
		InsertDataOption("INSERT_ROWS").
		Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to append values: %w", err))
	}

	// Guard against nil Updates in response
	if resp.Updates == nil {
		return p.PrintError(fmt.Errorf("unexpected empty response from API"))
	}

	return p.Print(map[string]interface{}{
		"status":        "appended",
		"spreadsheet":   resp.SpreadsheetId,
		"range":         resp.Updates.UpdatedRange,
		"rows_appended": resp.Updates.UpdatedRows,
		"cells_updated": resp.Updates.UpdatedCells,
	})
}

// parseValues parses values from either --values or --values-json flags.
func parseValues(cmd *cobra.Command) ([][]interface{}, error) {
	valuesStr, _ := cmd.Flags().GetString("values")
	valuesJSON, _ := cmd.Flags().GetString("values-json")

	// JSON format takes precedence
	if valuesJSON != "" {
		var rawValues [][]interface{}
		if err := json.Unmarshal([]byte(valuesJSON), &rawValues); err != nil {
			return nil, fmt.Errorf("invalid JSON format: %w", err)
		}
		return rawValues, nil
	}

	// Parse simple format: "a,b,c;d,e,f"
	if valuesStr != "" {
		rows := strings.Split(valuesStr, ";")
		values := make([][]interface{}, len(rows))
		for i, row := range rows {
			cells := strings.Split(row, ",")
			values[i] = make([]interface{}, len(cells))
			for j, cell := range cells {
				values[i][j] = strings.TrimSpace(cell)
			}
		}
		return values, nil
	}

	return nil, nil
}
