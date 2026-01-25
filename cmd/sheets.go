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

var sheetsAddSheetCmd = &cobra.Command{
	Use:   "add-sheet <spreadsheet-id>",
	Short: "Add a new sheet",
	Long:  "Adds a new sheet to an existing spreadsheet.",
	Args:  cobra.ExactArgs(1),
	RunE:  runSheetsAddSheet,
}

var sheetsDeleteSheetCmd = &cobra.Command{
	Use:   "delete-sheet <spreadsheet-id>",
	Short: "Delete a sheet",
	Long:  "Deletes a sheet from a spreadsheet by name or ID.",
	Args:  cobra.ExactArgs(1),
	RunE:  runSheetsDeleteSheet,
}

var sheetsClearCmd = &cobra.Command{
	Use:   "clear <spreadsheet-id> <range>",
	Short: "Clear cell values",
	Long: `Clears all values from a range of cells (keeps formatting).

Range format examples:
  Sheet1!A1:D10    - Specific range in Sheet1
  Sheet1           - All data in Sheet1
  A1:D10           - Range in first sheet`,
	Args: cobra.ExactArgs(2),
	RunE: runSheetsClear,
}

var sheetsInsertRowsCmd = &cobra.Command{
	Use:   "insert-rows <spreadsheet-id>",
	Short: "Insert rows into a sheet",
	Long:  "Inserts empty rows at a specified position in a sheet.",
	Args:  cobra.ExactArgs(1),
	RunE:  runSheetsInsertRows,
}

var sheetsDeleteRowsCmd = &cobra.Command{
	Use:   "delete-rows <spreadsheet-id>",
	Short: "Delete rows from a sheet",
	Long:  "Deletes rows from a specified range in a sheet.",
	Args:  cobra.ExactArgs(1),
	RunE:  runSheetsDeleteRows,
}

var sheetsInsertColsCmd = &cobra.Command{
	Use:   "insert-cols <spreadsheet-id>",
	Short: "Insert columns into a sheet",
	Long:  "Inserts empty columns at a specified position in a sheet.",
	Args:  cobra.ExactArgs(1),
	RunE:  runSheetsInsertCols,
}

var sheetsDeleteColsCmd = &cobra.Command{
	Use:   "delete-cols <spreadsheet-id>",
	Short: "Delete columns from a sheet",
	Long:  "Deletes columns from a specified range in a sheet.",
	Args:  cobra.ExactArgs(1),
	RunE:  runSheetsDeleteCols,
}

var sheetsRenameSheetCmd = &cobra.Command{
	Use:   "rename-sheet <spreadsheet-id>",
	Short: "Rename a sheet",
	Long:  "Renames a sheet within a spreadsheet.",
	Args:  cobra.ExactArgs(1),
	RunE:  runSheetsRenameSheet,
}

var sheetsDuplicateSheetCmd = &cobra.Command{
	Use:   "duplicate-sheet <spreadsheet-id>",
	Short: "Duplicate a sheet",
	Long:  "Creates a copy of an existing sheet within the spreadsheet.",
	Args:  cobra.ExactArgs(1),
	RunE:  runSheetsDuplicateSheet,
}

var sheetsMergeCmd = &cobra.Command{
	Use:   "merge <spreadsheet-id> <range>",
	Short: "Merge cells",
	Long: `Merges a range of cells into a single cell.

Range format examples:
  Sheet1!A1:D4     - Merge cells A1 through D4 in Sheet1
  A1:B2            - Merge cells in first sheet`,
	Args: cobra.ExactArgs(2),
	RunE: runSheetsMerge,
}

var sheetsUnmergeCmd = &cobra.Command{
	Use:   "unmerge <spreadsheet-id> <range>",
	Short: "Unmerge cells",
	Long: `Unmerges previously merged cells in a range.

Range format examples:
  Sheet1!A1:D4     - Unmerge cells in range
  A1:B2            - Unmerge cells in first sheet`,
	Args: cobra.ExactArgs(2),
	RunE: runSheetsUnmerge,
}

var sheetsSortCmd = &cobra.Command{
	Use:   "sort <spreadsheet-id> <range>",
	Short: "Sort a range",
	Long: `Sorts data in a range by a specified column.

Range format examples:
  Sheet1!A1:D10    - Sort range in Sheet1
  A1:D10           - Sort range in first sheet`,
	Args: cobra.ExactArgs(2),
	RunE: runSheetsSort,
}

var sheetsFindReplaceCmd = &cobra.Command{
	Use:   "find-replace <spreadsheet-id>",
	Short: "Find and replace in spreadsheet",
	Long:  "Finds and replaces text across the spreadsheet or within a specific sheet.",
	Args:  cobra.ExactArgs(1),
	RunE:  runSheetsFindReplace,
}

func init() {
	rootCmd.AddCommand(sheetsCmd)
	sheetsCmd.AddCommand(sheetsInfoCmd)
	sheetsCmd.AddCommand(sheetsReadCmd)
	sheetsCmd.AddCommand(sheetsListCmd)
	sheetsCmd.AddCommand(sheetsCreateCmd)
	sheetsCmd.AddCommand(sheetsWriteCmd)
	sheetsCmd.AddCommand(sheetsAppendCmd)
	sheetsCmd.AddCommand(sheetsAddSheetCmd)
	sheetsCmd.AddCommand(sheetsDeleteSheetCmd)
	sheetsCmd.AddCommand(sheetsClearCmd)
	sheetsCmd.AddCommand(sheetsInsertRowsCmd)
	sheetsCmd.AddCommand(sheetsDeleteRowsCmd)
	sheetsCmd.AddCommand(sheetsInsertColsCmd)
	sheetsCmd.AddCommand(sheetsDeleteColsCmd)
	sheetsCmd.AddCommand(sheetsRenameSheetCmd)
	sheetsCmd.AddCommand(sheetsDuplicateSheetCmd)
	sheetsCmd.AddCommand(sheetsMergeCmd)
	sheetsCmd.AddCommand(sheetsUnmergeCmd)
	sheetsCmd.AddCommand(sheetsSortCmd)
	sheetsCmd.AddCommand(sheetsFindReplaceCmd)

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

	// Add-sheet flags
	sheetsAddSheetCmd.Flags().String("name", "", "Sheet name (required)")
	sheetsAddSheetCmd.Flags().Int64("rows", 1000, "Number of rows")
	sheetsAddSheetCmd.Flags().Int64("cols", 26, "Number of columns")
	sheetsAddSheetCmd.MarkFlagRequired("name")

	// Delete-sheet flags
	sheetsDeleteSheetCmd.Flags().String("name", "", "Sheet name to delete")
	sheetsDeleteSheetCmd.Flags().Int64("sheet-id", -1, "Sheet ID to delete (alternative to --name)")

	// Insert-rows flags
	sheetsInsertRowsCmd.Flags().String("sheet", "", "Sheet name (required)")
	sheetsInsertRowsCmd.Flags().Int64("at", 0, "Row index to insert at (0-based)")
	sheetsInsertRowsCmd.Flags().Int64("count", 1, "Number of rows to insert")
	sheetsInsertRowsCmd.MarkFlagRequired("sheet")

	// Delete-rows flags
	sheetsDeleteRowsCmd.Flags().String("sheet", "", "Sheet name (required)")
	sheetsDeleteRowsCmd.Flags().Int64("from", 0, "Start row index (0-based, inclusive)")
	sheetsDeleteRowsCmd.Flags().Int64("to", 0, "End row index (0-based, exclusive)")
	sheetsDeleteRowsCmd.MarkFlagRequired("sheet")

	// Insert-cols flags
	sheetsInsertColsCmd.Flags().String("sheet", "", "Sheet name (required)")
	sheetsInsertColsCmd.Flags().Int64("at", 0, "Column index to insert at (0-based)")
	sheetsInsertColsCmd.Flags().Int64("count", 1, "Number of columns to insert")
	sheetsInsertColsCmd.MarkFlagRequired("sheet")

	// Delete-cols flags
	sheetsDeleteColsCmd.Flags().String("sheet", "", "Sheet name (required)")
	sheetsDeleteColsCmd.Flags().Int64("from", 0, "Start column index (0-based, inclusive)")
	sheetsDeleteColsCmd.Flags().Int64("to", 0, "End column index (0-based, exclusive)")
	sheetsDeleteColsCmd.MarkFlagRequired("sheet")

	// Rename-sheet flags
	sheetsRenameSheetCmd.Flags().String("sheet", "", "Current sheet name (required)")
	sheetsRenameSheetCmd.Flags().String("name", "", "New sheet name (required)")
	sheetsRenameSheetCmd.MarkFlagRequired("sheet")
	sheetsRenameSheetCmd.MarkFlagRequired("name")

	// Duplicate-sheet flags
	sheetsDuplicateSheetCmd.Flags().String("sheet", "", "Sheet name to duplicate (required)")
	sheetsDuplicateSheetCmd.Flags().String("new-name", "", "Name for the new sheet")
	sheetsDuplicateSheetCmd.MarkFlagRequired("sheet")

	// Merge flags (no additional flags needed, range is positional)

	// Unmerge flags (no additional flags needed, range is positional)

	// Sort flags
	sheetsSortCmd.Flags().String("by", "A", "Column to sort by (e.g., A, B, C)")
	sheetsSortCmd.Flags().Bool("desc", false, "Sort in descending order")
	sheetsSortCmd.Flags().Bool("has-header", false, "First row is a header (excluded from sort)")

	// Find-replace flags
	sheetsFindReplaceCmd.Flags().String("find", "", "Text to find (required)")
	sheetsFindReplaceCmd.Flags().String("replace", "", "Replacement text (required)")
	sheetsFindReplaceCmd.Flags().String("sheet", "", "Limit to specific sheet (optional)")
	sheetsFindReplaceCmd.Flags().Bool("match-case", false, "Case-sensitive matching")
	sheetsFindReplaceCmd.Flags().Bool("entire-cell", false, "Match entire cell contents only")
	sheetsFindReplaceCmd.MarkFlagRequired("find")
	sheetsFindReplaceCmd.MarkFlagRequired("replace")
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

func runSheetsAddSheet(cmd *cobra.Command, args []string) error {
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
	sheetName, _ := cmd.Flags().GetString("name")
	rows, _ := cmd.Flags().GetInt64("rows")
	cols, _ := cmd.Flags().GetInt64("cols")

	requests := []*sheets.Request{
		{
			AddSheet: &sheets.AddSheetRequest{
				Properties: &sheets.SheetProperties{
					Title: sheetName,
					GridProperties: &sheets.GridProperties{
						RowCount:    rows,
						ColumnCount: cols,
					},
				},
			},
		},
	}

	resp, err := svc.Spreadsheets.BatchUpdate(spreadsheetID, &sheets.BatchUpdateSpreadsheetRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to add sheet: %w", err))
	}

	// Get the new sheet ID from response
	var sheetID int64
	if len(resp.Replies) > 0 && resp.Replies[0].AddSheet != nil && resp.Replies[0].AddSheet.Properties != nil {
		sheetID = resp.Replies[0].AddSheet.Properties.SheetId
	}

	return p.Print(map[string]interface{}{
		"status":       "added",
		"spreadsheet":  spreadsheetID,
		"sheet_name":   sheetName,
		"sheet_id":     sheetID,
		"rows":         rows,
		"cols":         cols,
	})
}

func runSheetsDeleteSheet(cmd *cobra.Command, args []string) error {
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
	sheetName, _ := cmd.Flags().GetString("name")
	sheetID, _ := cmd.Flags().GetInt64("sheet-id")

	// If name provided, look up sheet ID
	if sheetName != "" {
		spreadsheet, err := svc.Spreadsheets.Get(spreadsheetID).Do()
		if err != nil {
			return p.PrintError(fmt.Errorf("failed to get spreadsheet: %w", err))
		}

		found := false
		for _, sheet := range spreadsheet.Sheets {
			if sheet.Properties.Title == sheetName {
				sheetID = sheet.Properties.SheetId
				found = true
				break
			}
		}
		if !found {
			return p.PrintError(fmt.Errorf("sheet '%s' not found", sheetName))
		}
	} else if sheetID < 0 {
		return p.PrintError(fmt.Errorf("must specify --name or --sheet-id"))
	}

	requests := []*sheets.Request{
		{
			DeleteSheet: &sheets.DeleteSheetRequest{
				SheetId: sheetID,
			},
		},
	}

	_, err = svc.Spreadsheets.BatchUpdate(spreadsheetID, &sheets.BatchUpdateSpreadsheetRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to delete sheet: %w", err))
	}

	result := map[string]interface{}{
		"status":      "deleted",
		"spreadsheet": spreadsheetID,
		"sheet_id":    sheetID,
	}
	if sheetName != "" {
		result["sheet_name"] = sheetName
	}

	return p.Print(result)
}

func runSheetsClear(cmd *cobra.Command, args []string) error {
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

	resp, err := svc.Spreadsheets.Values.Clear(spreadsheetID, rangeStr, &sheets.ClearValuesRequest{}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to clear range: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":       "cleared",
		"spreadsheet":  resp.SpreadsheetId,
		"range":        resp.ClearedRange,
	})
}

// getSheetID looks up a sheet ID by name within a spreadsheet.
func getSheetID(svc *sheets.Service, spreadsheetID, sheetName string) (int64, error) {
	spreadsheet, err := svc.Spreadsheets.Get(spreadsheetID).Do()
	if err != nil {
		return 0, fmt.Errorf("failed to get spreadsheet: %w", err)
	}

	for _, sheet := range spreadsheet.Sheets {
		if sheet.Properties.Title == sheetName {
			return sheet.Properties.SheetId, nil
		}
	}
	return 0, fmt.Errorf("sheet '%s' not found", sheetName)
}

func runSheetsInsertRows(cmd *cobra.Command, args []string) error {
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
	sheetName, _ := cmd.Flags().GetString("sheet")
	at, _ := cmd.Flags().GetInt64("at")
	count, _ := cmd.Flags().GetInt64("count")

	sheetID, err := getSheetID(svc, spreadsheetID, sheetName)
	if err != nil {
		return p.PrintError(err)
	}

	requests := []*sheets.Request{
		{
			InsertDimension: &sheets.InsertDimensionRequest{
				Range: &sheets.DimensionRange{
					SheetId:    sheetID,
					Dimension:  "ROWS",
					StartIndex: at,
					EndIndex:   at + count,
				},
				InheritFromBefore: at > 0,
			},
		},
	}

	_, err = svc.Spreadsheets.BatchUpdate(spreadsheetID, &sheets.BatchUpdateSpreadsheetRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to insert rows: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":      "inserted",
		"spreadsheet": spreadsheetID,
		"sheet":       sheetName,
		"at":          at,
		"count":       count,
		"dimension":   "rows",
	})
}

func runSheetsDeleteRows(cmd *cobra.Command, args []string) error {
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
	sheetName, _ := cmd.Flags().GetString("sheet")
	from, _ := cmd.Flags().GetInt64("from")
	to, _ := cmd.Flags().GetInt64("to")

	if to <= from {
		return p.PrintError(fmt.Errorf("--to must be greater than --from"))
	}

	sheetID, err := getSheetID(svc, spreadsheetID, sheetName)
	if err != nil {
		return p.PrintError(err)
	}

	requests := []*sheets.Request{
		{
			DeleteDimension: &sheets.DeleteDimensionRequest{
				Range: &sheets.DimensionRange{
					SheetId:    sheetID,
					Dimension:  "ROWS",
					StartIndex: from,
					EndIndex:   to,
				},
			},
		},
	}

	_, err = svc.Spreadsheets.BatchUpdate(spreadsheetID, &sheets.BatchUpdateSpreadsheetRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to delete rows: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":      "deleted",
		"spreadsheet": spreadsheetID,
		"sheet":       sheetName,
		"from":        from,
		"to":          to,
		"count":       to - from,
		"dimension":   "rows",
	})
}

func runSheetsInsertCols(cmd *cobra.Command, args []string) error {
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
	sheetName, _ := cmd.Flags().GetString("sheet")
	at, _ := cmd.Flags().GetInt64("at")
	count, _ := cmd.Flags().GetInt64("count")

	sheetID, err := getSheetID(svc, spreadsheetID, sheetName)
	if err != nil {
		return p.PrintError(err)
	}

	requests := []*sheets.Request{
		{
			InsertDimension: &sheets.InsertDimensionRequest{
				Range: &sheets.DimensionRange{
					SheetId:    sheetID,
					Dimension:  "COLUMNS",
					StartIndex: at,
					EndIndex:   at + count,
				},
				InheritFromBefore: at > 0,
			},
		},
	}

	_, err = svc.Spreadsheets.BatchUpdate(spreadsheetID, &sheets.BatchUpdateSpreadsheetRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to insert columns: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":      "inserted",
		"spreadsheet": spreadsheetID,
		"sheet":       sheetName,
		"at":          at,
		"count":       count,
		"dimension":   "columns",
	})
}

func runSheetsDeleteCols(cmd *cobra.Command, args []string) error {
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
	sheetName, _ := cmd.Flags().GetString("sheet")
	from, _ := cmd.Flags().GetInt64("from")
	to, _ := cmd.Flags().GetInt64("to")

	if to <= from {
		return p.PrintError(fmt.Errorf("--to must be greater than --from"))
	}

	sheetID, err := getSheetID(svc, spreadsheetID, sheetName)
	if err != nil {
		return p.PrintError(err)
	}

	requests := []*sheets.Request{
		{
			DeleteDimension: &sheets.DeleteDimensionRequest{
				Range: &sheets.DimensionRange{
					SheetId:    sheetID,
					Dimension:  "COLUMNS",
					StartIndex: from,
					EndIndex:   to,
				},
			},
		},
	}

	_, err = svc.Spreadsheets.BatchUpdate(spreadsheetID, &sheets.BatchUpdateSpreadsheetRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to delete columns: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":      "deleted",
		"spreadsheet": spreadsheetID,
		"sheet":       sheetName,
		"from":        from,
		"to":          to,
		"count":       to - from,
		"dimension":   "columns",
	})
}

func runSheetsRenameSheet(cmd *cobra.Command, args []string) error {
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
	oldName, _ := cmd.Flags().GetString("sheet")
	newName, _ := cmd.Flags().GetString("name")

	sheetID, err := getSheetID(svc, spreadsheetID, oldName)
	if err != nil {
		return p.PrintError(err)
	}

	requests := []*sheets.Request{
		{
			UpdateSheetProperties: &sheets.UpdateSheetPropertiesRequest{
				Properties: &sheets.SheetProperties{
					SheetId: sheetID,
					Title:   newName,
				},
				Fields: "title",
			},
		},
	}

	_, err = svc.Spreadsheets.BatchUpdate(spreadsheetID, &sheets.BatchUpdateSpreadsheetRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to rename sheet: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":      "renamed",
		"spreadsheet": spreadsheetID,
		"old_name":    oldName,
		"new_name":    newName,
		"sheet_id":    sheetID,
	})
}

func runSheetsDuplicateSheet(cmd *cobra.Command, args []string) error {
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
	sheetName, _ := cmd.Flags().GetString("sheet")
	newName, _ := cmd.Flags().GetString("new-name")

	sheetID, err := getSheetID(svc, spreadsheetID, sheetName)
	if err != nil {
		return p.PrintError(err)
	}

	duplicateReq := &sheets.DuplicateSheetRequest{
		SourceSheetId: sheetID,
	}
	if newName != "" {
		duplicateReq.NewSheetName = newName
	}

	requests := []*sheets.Request{
		{
			DuplicateSheet: duplicateReq,
		},
	}

	resp, err := svc.Spreadsheets.BatchUpdate(spreadsheetID, &sheets.BatchUpdateSpreadsheetRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to duplicate sheet: %w", err))
	}

	// Get the new sheet info from response
	var newSheetID int64
	var actualNewName string
	if len(resp.Replies) > 0 && resp.Replies[0].DuplicateSheet != nil && resp.Replies[0].DuplicateSheet.Properties != nil {
		newSheetID = resp.Replies[0].DuplicateSheet.Properties.SheetId
		actualNewName = resp.Replies[0].DuplicateSheet.Properties.Title
	}

	return p.Print(map[string]interface{}{
		"status":           "duplicated",
		"spreadsheet":      spreadsheetID,
		"source_sheet":     sheetName,
		"new_sheet_name":   actualNewName,
		"new_sheet_id":     newSheetID,
	})
}

// parseRange parses a Sheets range string (e.g., "Sheet1!A1:D10") and returns sheet name and grid range.
func parseRange(svc *sheets.Service, spreadsheetID, rangeStr string) (int64, *sheets.GridRange, error) {
	// Split sheet name from range
	var sheetName string
	var cellRange string

	if idx := strings.Index(rangeStr, "!"); idx != -1 {
		sheetName = rangeStr[:idx]
		cellRange = rangeStr[idx+1:]
	} else {
		// Assume first sheet if no sheet name
		spreadsheet, err := svc.Spreadsheets.Get(spreadsheetID).Do()
		if err != nil {
			return 0, nil, fmt.Errorf("failed to get spreadsheet: %w", err)
		}
		if len(spreadsheet.Sheets) == 0 {
			return 0, nil, fmt.Errorf("spreadsheet has no sheets")
		}
		sheetName = spreadsheet.Sheets[0].Properties.Title
		cellRange = rangeStr
	}

	sheetID, err := getSheetID(svc, spreadsheetID, sheetName)
	if err != nil {
		return 0, nil, err
	}

	// Parse cell range (e.g., "A1:D10")
	startCol, startRow, endCol, endRow, err := parseCellRange(cellRange)
	if err != nil {
		return 0, nil, err
	}

	return sheetID, &sheets.GridRange{
		SheetId:          sheetID,
		StartColumnIndex: startCol,
		StartRowIndex:    startRow,
		EndColumnIndex:   endCol,
		EndRowIndex:      endRow,
	}, nil
}

// parseCellRange parses a cell range like "A1:D10" into column and row indices.
func parseCellRange(cellRange string) (startCol, startRow, endCol, endRow int64, err error) {
	parts := strings.Split(cellRange, ":")
	if len(parts) != 2 {
		return 0, 0, 0, 0, fmt.Errorf("invalid range format: %s (expected format: A1:D10)", cellRange)
	}

	startCol, startRow, err = parseCellRef(parts[0])
	if err != nil {
		return 0, 0, 0, 0, err
	}

	endCol, endRow, err = parseCellRef(parts[1])
	if err != nil {
		return 0, 0, 0, 0, err
	}

	// End indices are exclusive in Grid API
	endCol++
	endRow++

	return startCol, startRow, endCol, endRow, nil
}

// parseCellRef parses a cell reference like "A1" into column and row indices (0-based).
func parseCellRef(ref string) (col, row int64, err error) {
	ref = strings.ToUpper(strings.TrimSpace(ref))

	// Extract column letters and row number
	colStr := ""
	rowStr := ""
	for _, c := range ref {
		if c >= 'A' && c <= 'Z' {
			colStr += string(c)
		} else if c >= '0' && c <= '9' {
			rowStr += string(c)
		}
	}

	if colStr == "" || rowStr == "" {
		return 0, 0, fmt.Errorf("invalid cell reference: %s", ref)
	}

	// Convert column letters to index (A=0, B=1, ..., Z=25, AA=26, etc.)
	col = 0
	for _, c := range colStr {
		col = col*26 + int64(c-'A'+1)
	}
	col-- // Convert to 0-based

	// Parse row number and convert to 0-based
	var rowNum int
	_, err = fmt.Sscanf(rowStr, "%d", &rowNum)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid row number: %s", rowStr)
	}
	row = int64(rowNum - 1)

	return col, row, nil
}

func runSheetsMerge(cmd *cobra.Command, args []string) error {
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

	_, gridRange, err := parseRange(svc, spreadsheetID, rangeStr)
	if err != nil {
		return p.PrintError(err)
	}

	requests := []*sheets.Request{
		{
			MergeCells: &sheets.MergeCellsRequest{
				Range:     gridRange,
				MergeType: "MERGE_ALL",
			},
		},
	}

	_, err = svc.Spreadsheets.BatchUpdate(spreadsheetID, &sheets.BatchUpdateSpreadsheetRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to merge cells: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":      "merged",
		"spreadsheet": spreadsheetID,
		"range":       rangeStr,
	})
}

func runSheetsUnmerge(cmd *cobra.Command, args []string) error {
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

	_, gridRange, err := parseRange(svc, spreadsheetID, rangeStr)
	if err != nil {
		return p.PrintError(err)
	}

	requests := []*sheets.Request{
		{
			UnmergeCells: &sheets.UnmergeCellsRequest{
				Range: gridRange,
			},
		},
	}

	_, err = svc.Spreadsheets.BatchUpdate(spreadsheetID, &sheets.BatchUpdateSpreadsheetRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to unmerge cells: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":      "unmerged",
		"spreadsheet": spreadsheetID,
		"range":       rangeStr,
	})
}

// columnLetterToIndex converts a column letter (A, B, ..., Z, AA, etc.) to a 0-based index.
func columnLetterToIndex(col string) int64 {
	col = strings.ToUpper(strings.TrimSpace(col))
	var index int64
	for _, c := range col {
		index = index*26 + int64(c-'A'+1)
	}
	return index - 1 // Convert to 0-based
}

func runSheetsSort(cmd *cobra.Command, args []string) error {
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
	sortBy, _ := cmd.Flags().GetString("by")
	desc, _ := cmd.Flags().GetBool("desc")
	hasHeader, _ := cmd.Flags().GetBool("has-header")

	_, gridRange, err := parseRange(svc, spreadsheetID, rangeStr)
	if err != nil {
		return p.PrintError(err)
	}

	// If has header, adjust start row
	if hasHeader {
		gridRange.StartRowIndex++
	}

	sortOrder := "ASCENDING"
	if desc {
		sortOrder = "DESCENDING"
	}

	sortColIndex := columnLetterToIndex(sortBy)

	requests := []*sheets.Request{
		{
			SortRange: &sheets.SortRangeRequest{
				Range: gridRange,
				SortSpecs: []*sheets.SortSpec{
					{
						DimensionIndex: sortColIndex,
						SortOrder:      sortOrder,
					},
				},
			},
		},
	}

	_, err = svc.Spreadsheets.BatchUpdate(spreadsheetID, &sheets.BatchUpdateSpreadsheetRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to sort range: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":      "sorted",
		"spreadsheet": spreadsheetID,
		"range":       rangeStr,
		"sort_column": sortBy,
		"order":       sortOrder,
	})
}

func runSheetsFindReplace(cmd *cobra.Command, args []string) error {
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
	findText, _ := cmd.Flags().GetString("find")
	replaceText, _ := cmd.Flags().GetString("replace")
	sheetName, _ := cmd.Flags().GetString("sheet")
	matchCase, _ := cmd.Flags().GetBool("match-case")
	entireCell, _ := cmd.Flags().GetBool("entire-cell")

	findReplaceReq := &sheets.FindReplaceRequest{
		Find:        findText,
		Replacement: replaceText,
		MatchCase:   matchCase,
		MatchEntireCell: entireCell,
		AllSheets:   sheetName == "",
	}

	// If specific sheet, set sheet ID
	if sheetName != "" {
		sheetID, err := getSheetID(svc, spreadsheetID, sheetName)
		if err != nil {
			return p.PrintError(err)
		}
		findReplaceReq.SheetId = sheetID
	}

	requests := []*sheets.Request{
		{
			FindReplace: findReplaceReq,
		},
	}

	resp, err := svc.Spreadsheets.BatchUpdate(spreadsheetID, &sheets.BatchUpdateSpreadsheetRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to find/replace: %w", err))
	}

	// Get replacement count from response
	var occurrences int64
	var sheetsChanged int64
	if len(resp.Replies) > 0 && resp.Replies[0].FindReplace != nil {
		occurrences = resp.Replies[0].FindReplace.OccurrencesChanged
		sheetsChanged = resp.Replies[0].FindReplace.SheetsChanged
	}

	result := map[string]interface{}{
		"status":            "replaced",
		"spreadsheet":       spreadsheetID,
		"find":              findText,
		"replace":           replaceText,
		"occurrences_changed": occurrences,
		"sheets_changed":    sheetsChanged,
	}
	if sheetName != "" {
		result["sheet"] = sheetName
	}

	return p.Print(result)
}
