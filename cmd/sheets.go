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
  A1:B2            - Merge cells in first sheet

Note: Unbounded ranges like "A:A" (whole column) or "1:1" (whole row) are not supported.`,
	Args: cobra.ExactArgs(2),
	RunE: runSheetsMerge,
}

var sheetsUnmergeCmd = &cobra.Command{
	Use:   "unmerge <spreadsheet-id> <range>",
	Short: "Unmerge cells",
	Long: `Unmerges previously merged cells in a range.

Range format examples:
  Sheet1!A1:D4     - Unmerge cells in range
  A1:B2            - Unmerge cells in first sheet

Note: Unbounded ranges like "A:A" (whole column) or "1:1" (whole row) are not supported.`,
	Args: cobra.ExactArgs(2),
	RunE: runSheetsUnmerge,
}

var sheetsSortCmd = &cobra.Command{
	Use:   "sort <spreadsheet-id> <range>",
	Short: "Sort a range",
	Long: `Sorts data in a range by a specified column.

Range format examples:
  Sheet1!A1:D10    - Sort range in Sheet1
  A1:D10           - Sort range in first sheet

Note: Unbounded ranges like "A:A" (whole column) or "1:1" (whole row) are not supported.`,
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

var sheetsFormatCmd = &cobra.Command{
	Use:   "format <spreadsheet-id> <range>",
	Short: "Format cells",
	Long: `Formats cells in a range with text and background styles.

Range format examples:
  Sheet1!A1:D10    - Format range in Sheet1
  A1:D10           - Format range in first sheet

Note: Unbounded ranges like "A:A" (whole column) or "1:1" (whole row) are not supported.`,
	Args: cobra.ExactArgs(2),
	RunE: runSheetsFormat,
}

var sheetsSetColumnWidthCmd = &cobra.Command{
	Use:   "set-column-width <spreadsheet-id>",
	Short: "Set column width",
	Long:  "Sets the width of a column in pixels.",
	Args:  cobra.ExactArgs(1),
	RunE:  runSheetsSetColumnWidth,
}

var sheetsSetRowHeightCmd = &cobra.Command{
	Use:   "set-row-height <spreadsheet-id>",
	Short: "Set row height",
	Long:  "Sets the height of a row in pixels.",
	Args:  cobra.ExactArgs(1),
	RunE:  runSheetsSetRowHeight,
}

var sheetsFreezeCmd = &cobra.Command{
	Use:   "freeze <spreadsheet-id>",
	Short: "Freeze rows and columns",
	Long:  "Freezes rows and/or columns in a sheet so they remain visible when scrolling.",
	Args:  cobra.ExactArgs(1),
	RunE:  runSheetsFreeze,
}

var sheetsCopyToCmd = &cobra.Command{
	Use:   "copy-to <spreadsheet-id>",
	Short: "Copy a sheet to another spreadsheet",
	Long:  "Copies a sheet tab from one spreadsheet to another spreadsheet.",
	Args:  cobra.ExactArgs(1),
	RunE:  runSheetsCopyTo,
}

var sheetsBatchReadCmd = &cobra.Command{
	Use:   "batch-read <spreadsheet-id>",
	Short: "Read multiple ranges",
	Long: `Reads multiple ranges from a spreadsheet in a single API call.

Range format examples:
  --ranges "Sheet1!A1:D10"    - Specific range in Sheet1
  --ranges "A1:B5"            - Range in first sheet
  --ranges "Sheet2!A1:C10"    - Range in Sheet2

Multiple ranges can be specified by repeating the --ranges flag.`,
	Args: cobra.ExactArgs(1),
	RunE: runSheetsBatchRead,
}

var sheetsBatchWriteCmd = &cobra.Command{
	Use:   "batch-write <spreadsheet-id>",
	Short: "Write to multiple ranges",
	Long: `Writes values to multiple ranges in a spreadsheet in a single API call.

Each range-values pair is specified with --ranges and --values flags.
The nth --ranges corresponds to the nth --values.

Example:
  gws sheets batch-write SPREADSHEET_ID \
    --ranges "A1:B2" --values '[[1,2],[3,4]]' \
    --ranges "Sheet2!A1:B1" --values '[["x","y"]]'`,
	Args: cobra.ExactArgs(1),
	RunE: runSheetsBatchWrite,
}

var sheetsAddNamedRangeCmd = &cobra.Command{
	Use:   "add-named-range <spreadsheet-id> <range>",
	Short: "Add a named range",
	Long:  "Adds a named range to a spreadsheet.",
	Args:  cobra.ExactArgs(2),
	RunE:  runSheetsAddNamedRange,
}

var sheetsListNamedRangesCmd = &cobra.Command{
	Use:   "list-named-ranges <spreadsheet-id>",
	Short: "List named ranges",
	Long:  "Lists all named ranges in a spreadsheet.",
	Args:  cobra.ExactArgs(1),
	RunE:  runSheetsListNamedRanges,
}

var sheetsDeleteNamedRangeCmd = &cobra.Command{
	Use:   "delete-named-range <spreadsheet-id>",
	Short: "Delete a named range",
	Long:  "Deletes a named range from a spreadsheet.",
	Args:  cobra.ExactArgs(1),
	RunE:  runSheetsDeleteNamedRange,
}

var sheetsAddFilterCmd = &cobra.Command{
	Use:   "add-filter <spreadsheet-id> <range>",
	Short: "Add a basic filter",
	Long:  "Sets a basic filter on a range in a spreadsheet.",
	Args:  cobra.ExactArgs(2),
	RunE:  runSheetsAddFilter,
}

var sheetsClearFilterCmd = &cobra.Command{
	Use:   "clear-filter <spreadsheet-id>",
	Short: "Clear a basic filter",
	Long:  "Clears the basic filter from a sheet.",
	Args:  cobra.ExactArgs(1),
	RunE:  runSheetsClearFilter,
}

var sheetsAddFilterViewCmd = &cobra.Command{
	Use:   "add-filter-view <spreadsheet-id> <range>",
	Short: "Add a filter view",
	Long:  "Creates a new filter view for a range in a spreadsheet.",
	Args:  cobra.ExactArgs(2),
	RunE:  runSheetsAddFilterView,
}

var sheetsAddChartCmd = &cobra.Command{
	Use:   "add-chart <spreadsheet-id>",
	Short: "Add a chart to a spreadsheet",
	Long:  "Adds an embedded chart (bar, line, area, column, scatter, pie, combo) to a spreadsheet.",
	Args:  cobra.ExactArgs(1),
	RunE:  runSheetsAddChart,
}

var sheetsListChartsCmd = &cobra.Command{
	Use:   "list-charts <spreadsheet-id>",
	Short: "List charts in a spreadsheet",
	Long:  "Lists all embedded charts in a spreadsheet with their IDs, titles, and types.",
	Args:  cobra.ExactArgs(1),
	RunE:  runSheetsListCharts,
}

var sheetsDeleteChartCmd = &cobra.Command{
	Use:   "delete-chart <spreadsheet-id>",
	Short: "Delete a chart from a spreadsheet",
	Long:  "Deletes an embedded chart by its chart ID.",
	Args:  cobra.ExactArgs(1),
	RunE:  runSheetsDeleteChart,
}

var sheetsAddConditionalFormatCmd = &cobra.Command{
	Use:   "add-conditional-format <spreadsheet-id> <range>",
	Short: "Add a conditional formatting rule",
	Long: `Adds a conditional formatting rule to a range of cells.

Rule types:
  >              Number greater than value
  <              Number less than value
  =              Number equal to value
  !=             Number not equal to value
  contains       Text contains value
  not-contains   Text does not contain value
  blank          Cell is blank
  not-blank      Cell is not blank
  formula        Custom formula (value is the formula)`,
	Args: cobra.ExactArgs(2),
	RunE: runSheetsAddConditionalFormat,
}

var sheetsListConditionalFormatsCmd = &cobra.Command{
	Use:   "list-conditional-formats <spreadsheet-id>",
	Short: "List conditional formatting rules",
	Long:  "Lists all conditional formatting rules for a specific sheet.",
	Args:  cobra.ExactArgs(1),
	RunE:  runSheetsListConditionalFormats,
}

var sheetsDeleteConditionalFormatCmd = &cobra.Command{
	Use:   "delete-conditional-format <spreadsheet-id>",
	Short: "Delete a conditional formatting rule",
	Long:  "Deletes a conditional formatting rule by its index within a sheet.",
	Args:  cobra.ExactArgs(1),
	RunE:  runSheetsDeleteConditionalFormat,
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
	sheetsCmd.AddCommand(sheetsFormatCmd)
	sheetsCmd.AddCommand(sheetsSetColumnWidthCmd)
	sheetsCmd.AddCommand(sheetsSetRowHeightCmd)
	sheetsCmd.AddCommand(sheetsFreezeCmd)

	// Format flags
	sheetsFormatCmd.Flags().Bool("bold", false, "Make text bold")
	sheetsFormatCmd.Flags().Bool("italic", false, "Make text italic")
	sheetsFormatCmd.Flags().String("bg-color", "", "Background color (hex, e.g., #FFFF00)")
	sheetsFormatCmd.Flags().String("color", "", "Text color (hex, e.g., #FF0000)")
	sheetsFormatCmd.Flags().Int64("font-size", 0, "Font size in points")

	// Set-column-width flags
	sheetsSetColumnWidthCmd.Flags().String("sheet", "", "Sheet name (required)")
	sheetsSetColumnWidthCmd.Flags().String("col", "", "Column letter (e.g., A, B, AA) (required)")
	sheetsSetColumnWidthCmd.Flags().Int64("width", 100, "Column width in pixels")
	sheetsSetColumnWidthCmd.MarkFlagRequired("sheet")
	sheetsSetColumnWidthCmd.MarkFlagRequired("col")

	// Set-row-height flags
	sheetsSetRowHeightCmd.Flags().String("sheet", "", "Sheet name (required)")
	sheetsSetRowHeightCmd.Flags().Int64("row", 1, "Row number (1-based) (required)")
	sheetsSetRowHeightCmd.Flags().Int64("height", 21, "Row height in pixels")
	sheetsSetRowHeightCmd.MarkFlagRequired("sheet")
	sheetsSetRowHeightCmd.MarkFlagRequired("row")

	// Freeze flags
	sheetsFreezeCmd.Flags().String("sheet", "", "Sheet name (required)")
	sheetsFreezeCmd.Flags().Int64("rows", 0, "Number of rows to freeze")
	sheetsFreezeCmd.Flags().Int64("cols", 0, "Number of columns to freeze")
	sheetsFreezeCmd.MarkFlagRequired("sheet")

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

	// Copy-to command
	sheetsCmd.AddCommand(sheetsCopyToCmd)
	sheetsCopyToCmd.Flags().Int64("sheet-id", 0, "Source sheet ID to copy (required)")
	sheetsCopyToCmd.Flags().String("destination", "", "Destination spreadsheet ID (required)")
	sheetsCopyToCmd.MarkFlagRequired("sheet-id")
	sheetsCopyToCmd.MarkFlagRequired("destination")

	// Batch-read command
	sheetsCmd.AddCommand(sheetsBatchReadCmd)
	sheetsBatchReadCmd.Flags().StringArray("ranges", nil, "Ranges to read (can be repeated)")
	sheetsBatchReadCmd.Flags().String("value-render", "FORMATTED_VALUE", "Value render option: FORMATTED_VALUE, UNFORMATTED_VALUE, FORMULA")
	sheetsBatchReadCmd.MarkFlagRequired("ranges")

	// Batch-write command
	sheetsCmd.AddCommand(sheetsBatchWriteCmd)
	sheetsBatchWriteCmd.Flags().StringArray("ranges", nil, "Target ranges (can be repeated, pairs with --values)")
	sheetsBatchWriteCmd.Flags().StringArray("values", nil, "JSON arrays of values (can be repeated, pairs with --ranges)")
	sheetsBatchWriteCmd.Flags().String("value-input", "USER_ENTERED", "Value input option: RAW, USER_ENTERED")
	sheetsBatchWriteCmd.MarkFlagRequired("ranges")
	sheetsBatchWriteCmd.MarkFlagRequired("values")

	// Named range commands
	sheetsCmd.AddCommand(sheetsAddNamedRangeCmd)
	sheetsAddNamedRangeCmd.Flags().String("name", "", "Name for the named range (required)")
	sheetsAddNamedRangeCmd.MarkFlagRequired("name")

	sheetsCmd.AddCommand(sheetsListNamedRangesCmd)

	sheetsCmd.AddCommand(sheetsDeleteNamedRangeCmd)
	sheetsDeleteNamedRangeCmd.Flags().String("named-range-id", "", "ID of the named range to delete (required)")
	sheetsDeleteNamedRangeCmd.MarkFlagRequired("named-range-id")

	// Filter commands
	sheetsCmd.AddCommand(sheetsAddFilterCmd)

	sheetsCmd.AddCommand(sheetsClearFilterCmd)
	sheetsClearFilterCmd.Flags().String("sheet", "", "Sheet name (required)")
	sheetsClearFilterCmd.MarkFlagRequired("sheet")

	sheetsCmd.AddCommand(sheetsAddFilterViewCmd)
	sheetsAddFilterViewCmd.Flags().String("name", "", "Title for the filter view (required)")
	sheetsAddFilterViewCmd.MarkFlagRequired("name")

	// Add-chart command
	sheetsCmd.AddCommand(sheetsAddChartCmd)
	sheetsAddChartCmd.Flags().String("type", "", "Chart type: BAR, LINE, AREA, COLUMN, SCATTER, PIE, COMBO (required)")
	sheetsAddChartCmd.Flags().String("data", "", "Data range (e.g., Sheet1!A1:B10) (required)")
	sheetsAddChartCmd.Flags().String("title", "", "Chart title")
	sheetsAddChartCmd.Flags().String("sheet", "", "Sheet to place chart on (defaults to new sheet)")
	sheetsAddChartCmd.MarkFlagRequired("type")
	sheetsAddChartCmd.MarkFlagRequired("data")

	// List-charts command
	sheetsCmd.AddCommand(sheetsListChartsCmd)

	// Delete-chart command
	sheetsCmd.AddCommand(sheetsDeleteChartCmd)
	sheetsDeleteChartCmd.Flags().Int64("chart-id", 0, "Chart ID to delete (required)")
	sheetsDeleteChartCmd.MarkFlagRequired("chart-id")

	// Add-conditional-format command
	sheetsCmd.AddCommand(sheetsAddConditionalFormatCmd)
	sheetsAddConditionalFormatCmd.Flags().String("rule", "", "Condition type: >, <, =, !=, contains, not-contains, blank, not-blank, formula (required)")
	sheetsAddConditionalFormatCmd.Flags().String("value", "", "Comparison value (required for >, <, =, !=, contains, not-contains, formula)")
	sheetsAddConditionalFormatCmd.Flags().String("bg-color", "", "Background color (hex, e.g., #FFFF00)")
	sheetsAddConditionalFormatCmd.Flags().String("color", "", "Text color (hex, e.g., #FF0000)")
	sheetsAddConditionalFormatCmd.Flags().Bool("bold", false, "Make matching text bold")
	sheetsAddConditionalFormatCmd.Flags().Bool("italic", false, "Make matching text italic")
	sheetsAddConditionalFormatCmd.MarkFlagRequired("rule")

	// List-conditional-formats command
	sheetsCmd.AddCommand(sheetsListConditionalFormatsCmd)
	sheetsListConditionalFormatsCmd.Flags().String("sheet", "", "Sheet name (required)")
	sheetsListConditionalFormatsCmd.MarkFlagRequired("sheet")

	// Delete-conditional-format command
	sheetsCmd.AddCommand(sheetsDeleteConditionalFormatCmd)
	sheetsDeleteConditionalFormatCmd.Flags().String("sheet", "", "Sheet name (required)")
	sheetsDeleteConditionalFormatCmd.Flags().Int64("index", 0, "0-based index of the rule to delete (required)")
	sheetsDeleteConditionalFormatCmd.MarkFlagRequired("sheet")
	sheetsDeleteConditionalFormatCmd.MarkFlagRequired("index")
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
			_ = writer.Write(record)
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
		"status":      "added",
		"spreadsheet": spreadsheetID,
		"sheet_name":  sheetName,
		"sheet_id":    sheetID,
		"rows":        rows,
		"cols":        cols,
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
		"status":      "cleared",
		"spreadsheet": resp.SpreadsheetId,
		"range":       resp.ClearedRange,
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
		"status":         "duplicated",
		"spreadsheet":    spreadsheetID,
		"source_sheet":   sheetName,
		"new_sheet_name": actualNewName,
		"new_sheet_id":   newSheetID,
	})
}

// parseRange parses a Sheets range string (e.g., "Sheet1!A1:D10") and returns sheet ID and grid range.
// Note: Does not support unbounded ranges like "A:A" (whole column) or "1:1" (whole row).
func parseRange(svc *sheets.Service, spreadsheetID, rangeStr string) (int64, *sheets.GridRange, error) {
	// Split sheet name from range
	var sheetID int64
	var cellRange string

	if idx := strings.Index(rangeStr, "!"); idx != -1 {
		sheetName := rangeStr[:idx]
		cellRange = rangeStr[idx+1:]
		// Look up sheet ID by name
		var err error
		sheetID, err = getSheetID(svc, spreadsheetID, sheetName)
		if err != nil {
			return 0, nil, err
		}
	} else {
		// Assume first sheet if no sheet name - get ID directly to avoid duplicate API call
		spreadsheet, err := svc.Spreadsheets.Get(spreadsheetID).Do()
		if err != nil {
			return 0, nil, fmt.Errorf("failed to get spreadsheet: %w", err)
		}
		if len(spreadsheet.Sheets) == 0 {
			return 0, nil, fmt.Errorf("spreadsheet has no sheets")
		}
		sheetID = spreadsheet.Sheets[0].Properties.SheetId
		cellRange = rangeStr
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
		Find:            findText,
		Replacement:     replaceText,
		MatchCase:       matchCase,
		MatchEntireCell: entireCell,
		AllSheets:       sheetName == "",
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
		"status":              "replaced",
		"spreadsheet":         spreadsheetID,
		"find":                findText,
		"replace":             replaceText,
		"occurrences_changed": occurrences,
		"sheets_changed":      sheetsChanged,
	}
	if sheetName != "" {
		result["sheet"] = sheetName
	}

	return p.Print(result)
}

// parseSheetsHexColor parses a hex color string (#RRGGBB) into a Sheets Color.
func parseSheetsHexColor(hex string) (*sheets.Color, error) {
	if len(hex) != 7 || hex[0] != '#' {
		return nil, fmt.Errorf("invalid hex color format: %s (expected #RRGGBB)", hex)
	}

	var r, g, b int64
	_, err := fmt.Sscanf(hex, "#%02x%02x%02x", &r, &g, &b)
	if err != nil {
		return nil, fmt.Errorf("invalid hex color: %s", hex)
	}

	return &sheets.Color{
		Red:   float64(r) / 255.0,
		Green: float64(g) / 255.0,
		Blue:  float64(b) / 255.0,
	}, nil
}

func runSheetsFormat(cmd *cobra.Command, args []string) error {
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

	bold, _ := cmd.Flags().GetBool("bold")
	italic, _ := cmd.Flags().GetBool("italic")
	bgColor, _ := cmd.Flags().GetString("bg-color")
	textColor, _ := cmd.Flags().GetString("color")
	fontSize, _ := cmd.Flags().GetInt64("font-size")

	_, gridRange, err := parseRange(svc, spreadsheetID, rangeStr)
	if err != nil {
		return p.PrintError(err)
	}

	// Build cell format and field mask
	cellFormat := &sheets.CellFormat{}
	var fields []string

	textFormat := &sheets.TextFormat{}
	hasTextFormat := false

	if cmd.Flags().Changed("bold") {
		textFormat.Bold = bold
		if !bold {
			textFormat.ForceSendFields = append(textFormat.ForceSendFields, "Bold")
		}
		fields = append(fields, "userEnteredFormat.textFormat.bold")
		hasTextFormat = true
	}

	if cmd.Flags().Changed("italic") {
		textFormat.Italic = italic
		if !italic {
			textFormat.ForceSendFields = append(textFormat.ForceSendFields, "Italic")
		}
		fields = append(fields, "userEnteredFormat.textFormat.italic")
		hasTextFormat = true
	}

	if fontSize > 0 {
		textFormat.FontSize = fontSize
		fields = append(fields, "userEnteredFormat.textFormat.fontSize")
		hasTextFormat = true
	}

	if textColor != "" {
		color, err := parseSheetsHexColor(textColor)
		if err != nil {
			return p.PrintError(err)
		}
		textFormat.ForegroundColorStyle = &sheets.ColorStyle{RgbColor: color}
		fields = append(fields, "userEnteredFormat.textFormat.foregroundColorStyle")
		hasTextFormat = true
	}

	if hasTextFormat {
		cellFormat.TextFormat = textFormat
	}

	if bgColor != "" {
		color, err := parseSheetsHexColor(bgColor)
		if err != nil {
			return p.PrintError(err)
		}
		cellFormat.BackgroundColorStyle = &sheets.ColorStyle{RgbColor: color}
		fields = append(fields, "userEnteredFormat.backgroundColorStyle")
	}

	if len(fields) == 0 {
		return p.PrintError(fmt.Errorf("no formatting options specified; use --bold, --italic, --bg-color, --color, or --font-size"))
	}

	requests := []*sheets.Request{
		{
			RepeatCell: &sheets.RepeatCellRequest{
				Range: gridRange,
				Cell: &sheets.CellData{
					UserEnteredFormat: cellFormat,
				},
				Fields: strings.Join(fields, ","),
			},
		},
	}

	_, err = svc.Spreadsheets.BatchUpdate(spreadsheetID, &sheets.BatchUpdateSpreadsheetRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to format cells: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":      "formatted",
		"spreadsheet": spreadsheetID,
		"range":       rangeStr,
	})
}

func runSheetsSetColumnWidth(cmd *cobra.Command, args []string) error {
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
	col, _ := cmd.Flags().GetString("col")
	width, _ := cmd.Flags().GetInt64("width")

	sheetID, err := getSheetID(svc, spreadsheetID, sheetName)
	if err != nil {
		return p.PrintError(err)
	}

	colIndex := columnLetterToIndex(col)

	requests := []*sheets.Request{
		{
			UpdateDimensionProperties: &sheets.UpdateDimensionPropertiesRequest{
				Range: &sheets.DimensionRange{
					SheetId:    sheetID,
					Dimension:  "COLUMNS",
					StartIndex: colIndex,
					EndIndex:   colIndex + 1,
				},
				Properties: &sheets.DimensionProperties{
					PixelSize: width,
				},
				Fields: "pixelSize",
			},
		},
	}

	_, err = svc.Spreadsheets.BatchUpdate(spreadsheetID, &sheets.BatchUpdateSpreadsheetRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to set column width: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":      "updated",
		"spreadsheet": spreadsheetID,
		"sheet":       sheetName,
		"column":      col,
		"width":       width,
	})
}

func runSheetsSetRowHeight(cmd *cobra.Command, args []string) error {
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
	row, _ := cmd.Flags().GetInt64("row")
	height, _ := cmd.Flags().GetInt64("height")

	sheetID, err := getSheetID(svc, spreadsheetID, sheetName)
	if err != nil {
		return p.PrintError(err)
	}

	// Convert 1-based row to 0-based index
	rowIndex := row - 1

	requests := []*sheets.Request{
		{
			UpdateDimensionProperties: &sheets.UpdateDimensionPropertiesRequest{
				Range: &sheets.DimensionRange{
					SheetId:    sheetID,
					Dimension:  "ROWS",
					StartIndex: rowIndex,
					EndIndex:   rowIndex + 1,
				},
				Properties: &sheets.DimensionProperties{
					PixelSize: height,
				},
				Fields: "pixelSize",
			},
		},
	}

	_, err = svc.Spreadsheets.BatchUpdate(spreadsheetID, &sheets.BatchUpdateSpreadsheetRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to set row height: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":      "updated",
		"spreadsheet": spreadsheetID,
		"sheet":       sheetName,
		"row":         row,
		"height":      height,
	})
}

func runSheetsFreeze(cmd *cobra.Command, args []string) error {
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
	freezeRows, _ := cmd.Flags().GetInt64("rows")
	freezeCols, _ := cmd.Flags().GetInt64("cols")

	if !cmd.Flags().Changed("rows") && !cmd.Flags().Changed("cols") {
		return p.PrintError(fmt.Errorf("specify --rows and/or --cols to freeze"))
	}

	sheetID, err := getSheetID(svc, spreadsheetID, sheetName)
	if err != nil {
		return p.PrintError(err)
	}

	gridProps := &sheets.GridProperties{}
	var fieldParts []string

	if cmd.Flags().Changed("rows") {
		gridProps.FrozenRowCount = freezeRows
		if freezeRows == 0 {
			gridProps.ForceSendFields = append(gridProps.ForceSendFields, "FrozenRowCount")
		}
		fieldParts = append(fieldParts, "gridProperties.frozenRowCount")
	}

	if cmd.Flags().Changed("cols") {
		gridProps.FrozenColumnCount = freezeCols
		if freezeCols == 0 {
			gridProps.ForceSendFields = append(gridProps.ForceSendFields, "FrozenColumnCount")
		}
		fieldParts = append(fieldParts, "gridProperties.frozenColumnCount")
	}

	requests := []*sheets.Request{
		{
			UpdateSheetProperties: &sheets.UpdateSheetPropertiesRequest{
				Properties: &sheets.SheetProperties{
					SheetId:        sheetID,
					GridProperties: gridProps,
				},
				Fields: strings.Join(fieldParts, ","),
			},
		},
	}

	_, err = svc.Spreadsheets.BatchUpdate(spreadsheetID, &sheets.BatchUpdateSpreadsheetRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to freeze panes: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":      "frozen",
		"spreadsheet": spreadsheetID,
		"sheet":       sheetName,
		"frozen_rows": freezeRows,
		"frozen_cols": freezeCols,
	})
}

func runSheetsCopyTo(cmd *cobra.Command, args []string) error {
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
	sheetID, _ := cmd.Flags().GetInt64("sheet-id")
	destination, _ := cmd.Flags().GetString("destination")

	req := &sheets.CopySheetToAnotherSpreadsheetRequest{
		DestinationSpreadsheetId: destination,
	}

	resp, err := svc.Spreadsheets.Sheets.CopyTo(spreadsheetID, sheetID, req).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to copy sheet: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":             "copied",
		"source_spreadsheet": spreadsheetID,
		"source_sheet_id":    sheetID,
		"destination":        destination,
		"new_sheet_id":       resp.SheetId,
		"new_sheet_title":    resp.Title,
		"new_sheet_index":    resp.Index,
	})
}

func runSheetsBatchRead(cmd *cobra.Command, args []string) error {
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
	ranges, _ := cmd.Flags().GetStringArray("ranges")
	valueRender, _ := cmd.Flags().GetString("value-render")

	resp, err := svc.Spreadsheets.Values.BatchGet(spreadsheetID).
		Ranges(ranges...).
		ValueRenderOption(valueRender).
		Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to batch read: %w", err))
	}

	results := make([]map[string]interface{}, 0, len(resp.ValueRanges))
	for _, vr := range resp.ValueRanges {
		results = append(results, map[string]interface{}{
			"range": vr.Range,
			"data":  vr.Values,
			"rows":  len(vr.Values),
		})
	}

	return p.Print(map[string]interface{}{
		"spreadsheet": resp.SpreadsheetId,
		"ranges":      results,
		"range_count": len(results),
	})
}

func runSheetsBatchWrite(cmd *cobra.Command, args []string) error {
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
	ranges, _ := cmd.Flags().GetStringArray("ranges")
	valuesStrs, _ := cmd.Flags().GetStringArray("values")
	valueInput, _ := cmd.Flags().GetString("value-input")

	if len(ranges) != len(valuesStrs) {
		return p.PrintError(fmt.Errorf("number of --ranges flags (%d) must match number of --values flags (%d)", len(ranges), len(valuesStrs)))
	}

	data := make([]*sheets.ValueRange, 0, len(ranges))
	for i, rangeStr := range ranges {
		var rawValues [][]interface{}
		if err := json.Unmarshal([]byte(valuesStrs[i]), &rawValues); err != nil {
			return p.PrintError(fmt.Errorf("invalid JSON for values[%d]: %w", i, err))
		}
		data = append(data, &sheets.ValueRange{
			Range:  rangeStr,
			Values: rawValues,
		})
	}

	req := &sheets.BatchUpdateValuesRequest{
		ValueInputOption: valueInput,
		Data:             data,
	}

	resp, err := svc.Spreadsheets.Values.BatchUpdate(spreadsheetID, req).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to batch write: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":         "written",
		"spreadsheet":    resp.SpreadsheetId,
		"sheets_updated": resp.TotalUpdatedSheets,
		"rows_updated":   resp.TotalUpdatedRows,
		"cells_updated":  resp.TotalUpdatedCells,
	})
}

func runSheetsAddNamedRange(cmd *cobra.Command, args []string) error {
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
	name, _ := cmd.Flags().GetString("name")

	_, gridRange, err := parseRange(svc, spreadsheetID, rangeStr)
	if err != nil {
		return p.PrintError(err)
	}

	requests := []*sheets.Request{
		{
			AddNamedRange: &sheets.AddNamedRangeRequest{
				NamedRange: &sheets.NamedRange{
					Name:  name,
					Range: gridRange,
				},
			},
		},
	}

	resp, err := svc.Spreadsheets.BatchUpdate(spreadsheetID, &sheets.BatchUpdateSpreadsheetRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to add named range: %w", err))
	}

	var namedRangeID string
	if len(resp.Replies) > 0 && resp.Replies[0].AddNamedRange != nil && resp.Replies[0].AddNamedRange.NamedRange != nil {
		namedRangeID = resp.Replies[0].AddNamedRange.NamedRange.NamedRangeId
	}

	return p.Print(map[string]interface{}{
		"status":         "added",
		"spreadsheet":    spreadsheetID,
		"name":           name,
		"named_range_id": namedRangeID,
		"range":          rangeStr,
	})
}

func runSheetsListNamedRanges(cmd *cobra.Command, args []string) error {
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

	spreadsheet, err := svc.Spreadsheets.Get(spreadsheetID).Fields("namedRanges").Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get named ranges: %w", err))
	}

	namedRanges := make([]map[string]interface{}, 0, len(spreadsheet.NamedRanges))
	for _, nr := range spreadsheet.NamedRanges {
		entry := map[string]interface{}{
			"name":           nr.Name,
			"named_range_id": nr.NamedRangeId,
		}
		if nr.Range != nil {
			entry["range"] = map[string]interface{}{
				"sheet_id":     nr.Range.SheetId,
				"start_row":    nr.Range.StartRowIndex,
				"end_row":      nr.Range.EndRowIndex,
				"start_column": nr.Range.StartColumnIndex,
				"end_column":   nr.Range.EndColumnIndex,
			}
		}
		namedRanges = append(namedRanges, entry)
	}

	return p.Print(map[string]interface{}{
		"named_ranges": namedRanges,
		"count":        len(namedRanges),
	})
}

func runSheetsDeleteNamedRange(cmd *cobra.Command, args []string) error {
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
	namedRangeID, _ := cmd.Flags().GetString("named-range-id")

	requests := []*sheets.Request{
		{
			DeleteNamedRange: &sheets.DeleteNamedRangeRequest{
				NamedRangeId: namedRangeID,
			},
		},
	}

	_, err = svc.Spreadsheets.BatchUpdate(spreadsheetID, &sheets.BatchUpdateSpreadsheetRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to delete named range: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":         "deleted",
		"spreadsheet":    spreadsheetID,
		"named_range_id": namedRangeID,
	})
}

func runSheetsAddFilter(cmd *cobra.Command, args []string) error {
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
			SetBasicFilter: &sheets.SetBasicFilterRequest{
				Filter: &sheets.BasicFilter{
					Range: gridRange,
				},
			},
		},
	}

	_, err = svc.Spreadsheets.BatchUpdate(spreadsheetID, &sheets.BatchUpdateSpreadsheetRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to add filter: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":      "added",
		"spreadsheet": spreadsheetID,
		"range":       rangeStr,
	})
}

func runSheetsClearFilter(cmd *cobra.Command, args []string) error {
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

	sheetID, err := getSheetID(svc, spreadsheetID, sheetName)
	if err != nil {
		return p.PrintError(err)
	}

	requests := []*sheets.Request{
		{
			ClearBasicFilter: &sheets.ClearBasicFilterRequest{
				SheetId: sheetID,
			},
		},
	}

	_, err = svc.Spreadsheets.BatchUpdate(spreadsheetID, &sheets.BatchUpdateSpreadsheetRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to clear filter: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":      "cleared",
		"spreadsheet": spreadsheetID,
		"sheet":       sheetName,
	})
}

func runSheetsAddFilterView(cmd *cobra.Command, args []string) error {
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
	name, _ := cmd.Flags().GetString("name")

	_, gridRange, err := parseRange(svc, spreadsheetID, rangeStr)
	if err != nil {
		return p.PrintError(err)
	}

	requests := []*sheets.Request{
		{
			AddFilterView: &sheets.AddFilterViewRequest{
				Filter: &sheets.FilterView{
					Title: name,
					Range: gridRange,
				},
			},
		},
	}

	resp, err := svc.Spreadsheets.BatchUpdate(spreadsheetID, &sheets.BatchUpdateSpreadsheetRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to add filter view: %w", err))
	}

	var filterViewID int64
	if len(resp.Replies) > 0 && resp.Replies[0].AddFilterView != nil && resp.Replies[0].AddFilterView.Filter != nil {
		filterViewID = resp.Replies[0].AddFilterView.Filter.FilterViewId
	}

	return p.Print(map[string]interface{}{
		"status":         "added",
		"spreadsheet":    spreadsheetID,
		"name":           name,
		"filter_view_id": filterViewID,
		"range":          rangeStr,
	})
}

func runSheetsAddChart(cmd *cobra.Command, args []string) error {
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
	chartType, _ := cmd.Flags().GetString("type")
	dataRange, _ := cmd.Flags().GetString("data")
	title, _ := cmd.Flags().GetString("title")
	sheetName, _ := cmd.Flags().GetString("sheet")

	chartType = strings.ToUpper(chartType)

	validTypes := map[string]bool{"BAR": true, "LINE": true, "AREA": true, "COLUMN": true, "SCATTER": true, "PIE": true, "COMBO": true}
	if !validTypes[chartType] {
		return p.PrintError(fmt.Errorf("unknown chart type: %s (valid: BAR, LINE, AREA, COLUMN, SCATTER, PIE, COMBO)", chartType))
	}

	_, gridRange, err := parseRange(svc, spreadsheetID, dataRange)
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to parse data range: %w", err))
	}

	sourceRange := &sheets.ChartSourceRange{Sources: []*sheets.GridRange{gridRange}}
	chartData := &sheets.ChartData{SourceRange: sourceRange}

	spec := &sheets.ChartSpec{Title: title}

	if chartType == "PIE" {
		// PIE: first column = labels (domain), remaining columns = data (series)
		if gridRange.EndColumnIndex-gridRange.StartColumnIndex < 2 {
			return p.PrintError(fmt.Errorf("PIE chart requires at least 2 columns (labels + data), got range with %d column(s)", gridRange.EndColumnIndex-gridRange.StartColumnIndex))
		}
		domainRange := &sheets.GridRange{
			SheetId:          gridRange.SheetId,
			StartRowIndex:    gridRange.StartRowIndex,
			EndRowIndex:      gridRange.EndRowIndex,
			StartColumnIndex: gridRange.StartColumnIndex,
			EndColumnIndex:   gridRange.StartColumnIndex + 1,
		}
		seriesRange := &sheets.GridRange{
			SheetId:          gridRange.SheetId,
			StartRowIndex:    gridRange.StartRowIndex,
			EndRowIndex:      gridRange.EndRowIndex,
			StartColumnIndex: gridRange.StartColumnIndex + 1,
			EndColumnIndex:   gridRange.EndColumnIndex,
		}
		spec.PieChart = &sheets.PieChartSpec{
			Domain: &sheets.ChartData{SourceRange: &sheets.ChartSourceRange{Sources: []*sheets.GridRange{domainRange}}},
			Series: &sheets.ChartData{SourceRange: &sheets.ChartSourceRange{Sources: []*sheets.GridRange{seriesRange}}},
		}
	} else {
		spec.BasicChart = &sheets.BasicChartSpec{
			ChartType: chartType,
			Domains: []*sheets.BasicChartDomain{
				{Domain: chartData},
			},
			Series: []*sheets.BasicChartSeries{
				{Series: &sheets.ChartData{SourceRange: sourceRange}},
			},
		}
	}

	position := &sheets.EmbeddedObjectPosition{}
	if sheetName != "" {
		sheetID, err := getSheetID(svc, spreadsheetID, sheetName)
		if err != nil {
			return p.PrintError(err)
		}
		position.OverlayPosition = &sheets.OverlayPosition{
			AnchorCell: &sheets.GridCoordinate{
				SheetId:     sheetID,
				RowIndex:    0,
				ColumnIndex: 0,
			},
		}
	} else {
		position.NewSheet = true
	}

	requests := []*sheets.Request{
		{
			AddChart: &sheets.AddChartRequest{
				Chart: &sheets.EmbeddedChart{
					Spec:     spec,
					Position: position,
				},
			},
		},
	}

	resp, err := svc.Spreadsheets.BatchUpdate(spreadsheetID, &sheets.BatchUpdateSpreadsheetRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to add chart: %w", err))
	}

	var chartID int64
	if len(resp.Replies) > 0 && resp.Replies[0].AddChart != nil && resp.Replies[0].AddChart.Chart != nil {
		chartID = resp.Replies[0].AddChart.Chart.ChartId
	}

	return p.Print(map[string]interface{}{
		"status":      "added",
		"spreadsheet": spreadsheetID,
		"chart_id":    chartID,
		"type":        chartType,
		"title":       title,
		"data_range":  dataRange,
	})
}

func runSheetsListCharts(cmd *cobra.Command, args []string) error {
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

	spreadsheet, err := svc.Spreadsheets.Get(spreadsheetID).Fields("sheets.charts,sheets.properties").Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get spreadsheet: %w", err))
	}

	var charts []map[string]interface{}
	for _, sheet := range spreadsheet.Sheets {
		for _, chart := range sheet.Charts {
			chartInfo := map[string]interface{}{
				"chart_id": chart.ChartId,
				"sheet":    sheet.Properties.Title,
				"sheet_id": sheet.Properties.SheetId,
			}
			if chart.Spec != nil {
				chartInfo["title"] = chart.Spec.Title
				if chart.Spec.BasicChart != nil {
					chartInfo["type"] = chart.Spec.BasicChart.ChartType
				} else if chart.Spec.PieChart != nil {
					chartInfo["type"] = "PIE"
				}
			}
			charts = append(charts, chartInfo)
		}
	}

	if charts == nil {
		charts = []map[string]interface{}{}
	}

	return p.Print(map[string]interface{}{
		"spreadsheet": spreadsheetID,
		"charts":      charts,
		"count":       len(charts),
	})
}

func runSheetsDeleteChart(cmd *cobra.Command, args []string) error {
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
	chartID, _ := cmd.Flags().GetInt64("chart-id")

	requests := []*sheets.Request{
		{
			DeleteEmbeddedObject: &sheets.DeleteEmbeddedObjectRequest{
				ObjectId: chartID,
			},
		},
	}

	_, err = svc.Spreadsheets.BatchUpdate(spreadsheetID, &sheets.BatchUpdateSpreadsheetRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to delete chart: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":      "deleted",
		"spreadsheet": spreadsheetID,
		"chart_id":    chartID,
	})
}

// mapConditionType maps user-friendly rule names to Sheets API condition types.
func mapConditionType(rule string) (string, error) {
	switch rule {
	case ">":
		return "NUMBER_GREATER", nil
	case "<":
		return "NUMBER_LESS", nil
	case "=":
		return "NUMBER_EQ", nil
	case "!=":
		return "NUMBER_NOT_EQ", nil
	case "contains":
		return "TEXT_CONTAINS", nil
	case "not-contains":
		return "TEXT_NOT_CONTAINS", nil
	case "blank":
		return "BLANK", nil
	case "not-blank":
		return "NOT_BLANK", nil
	case "formula":
		return "CUSTOM_FORMULA", nil
	default:
		return "", fmt.Errorf("unknown rule type: %s (valid: >, <, =, !=, contains, not-contains, blank, not-blank, formula)", rule)
	}
}

func runSheetsAddConditionalFormat(cmd *cobra.Command, args []string) error {
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
	rule, _ := cmd.Flags().GetString("rule")
	value, _ := cmd.Flags().GetString("value")
	bgColor, _ := cmd.Flags().GetString("bg-color")
	textColor, _ := cmd.Flags().GetString("color")
	bold, _ := cmd.Flags().GetBool("bold")
	italic, _ := cmd.Flags().GetBool("italic")

	conditionType, err := mapConditionType(rule)
	if err != nil {
		return p.PrintError(err)
	}

	// Validate --value is provided for rules that require it
	needsValue := map[string]bool{">": true, "<": true, "=": true, "!=": true, "contains": true, "not-contains": true, "formula": true}
	if needsValue[rule] && value == "" {
		return p.PrintError(fmt.Errorf("--value is required for rule type %q", rule))
	}

	_, gridRange, err := parseRange(svc, spreadsheetID, rangeStr)
	if err != nil {
		return p.PrintError(err)
	}

	conditionValues := []*sheets.ConditionValue{}
	if value != "" {
		conditionValues = append(conditionValues, &sheets.ConditionValue{UserEnteredValue: value})
	}

	cellFormat := &sheets.CellFormat{}
	hasFormat := false

	if bgColor != "" {
		color, err := parseSheetsHexColor(bgColor)
		if err != nil {
			return p.PrintError(err)
		}
		cellFormat.BackgroundColorStyle = &sheets.ColorStyle{RgbColor: color}
		hasFormat = true
	}

	if textColor != "" {
		color, err := parseSheetsHexColor(textColor)
		if err != nil {
			return p.PrintError(err)
		}
		cellFormat.TextFormat = &sheets.TextFormat{}
		cellFormat.TextFormat.ForegroundColorStyle = &sheets.ColorStyle{RgbColor: color}
		hasFormat = true
	}

	if bold || italic {
		if cellFormat.TextFormat == nil {
			cellFormat.TextFormat = &sheets.TextFormat{}
		}
		cellFormat.TextFormat.Bold = bold
		cellFormat.TextFormat.Italic = italic
		hasFormat = true
	}

	if !hasFormat {
		cellFormat.BackgroundColorStyle = &sheets.ColorStyle{
			RgbColor: &sheets.Color{Red: 1.0, Green: 1.0, Blue: 0.0},
		}
	}

	conditionalRule := &sheets.ConditionalFormatRule{
		Ranges: []*sheets.GridRange{gridRange},
		BooleanRule: &sheets.BooleanRule{
			Condition: &sheets.BooleanCondition{
				Type:   conditionType,
				Values: conditionValues,
			},
			Format: cellFormat,
		},
	}

	requests := []*sheets.Request{
		{
			AddConditionalFormatRule: &sheets.AddConditionalFormatRuleRequest{
				Rule:  conditionalRule,
				Index: 0,
			},
		},
	}

	_, err = svc.Spreadsheets.BatchUpdate(spreadsheetID, &sheets.BatchUpdateSpreadsheetRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to add conditional format: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":         "added",
		"spreadsheet":    spreadsheetID,
		"range":          rangeStr,
		"condition_type": conditionType,
		"value":          value,
	})
}

func runSheetsListConditionalFormats(cmd *cobra.Command, args []string) error {
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

	spreadsheet, err := svc.Spreadsheets.Get(spreadsheetID).Fields("sheets.conditionalFormats,sheets.properties").Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to get spreadsheet: %w", err))
	}

	var rules []map[string]interface{}
	found := false
	for _, sheet := range spreadsheet.Sheets {
		if sheet.Properties.Title == sheetName {
			found = true
			for i, rule := range sheet.ConditionalFormats {
				ruleInfo := map[string]interface{}{
					"index": i,
				}

				if len(rule.Ranges) > 0 {
					ranges := make([]map[string]interface{}, len(rule.Ranges))
					for j, r := range rule.Ranges {
						ranges[j] = map[string]interface{}{
							"sheet_id":  r.SheetId,
							"start_row": r.StartRowIndex,
							"end_row":   r.EndRowIndex,
							"start_col": r.StartColumnIndex,
							"end_col":   r.EndColumnIndex,
						}
					}
					ruleInfo["ranges"] = ranges
				}

				if rule.BooleanRule != nil {
					if rule.BooleanRule.Condition != nil {
						condition := map[string]interface{}{
							"type": rule.BooleanRule.Condition.Type,
						}
						if len(rule.BooleanRule.Condition.Values) > 0 {
							values := make([]string, len(rule.BooleanRule.Condition.Values))
							for k, v := range rule.BooleanRule.Condition.Values {
								values[k] = v.UserEnteredValue
							}
							condition["values"] = values
						}
						ruleInfo["condition"] = condition
					}
					if rule.BooleanRule.Format != nil {
						format := map[string]interface{}{}
						if rule.BooleanRule.Format.TextFormat != nil {
							format["bold"] = rule.BooleanRule.Format.TextFormat.Bold
							format["italic"] = rule.BooleanRule.Format.TextFormat.Italic
						}
						ruleInfo["format"] = format
					}
				}

				rules = append(rules, ruleInfo)
			}
			break
		}
	}

	if !found {
		return p.PrintError(fmt.Errorf("sheet '%s' not found", sheetName))
	}

	if rules == nil {
		rules = []map[string]interface{}{}
	}

	return p.Print(map[string]interface{}{
		"spreadsheet": spreadsheetID,
		"sheet":       sheetName,
		"rules":       rules,
		"count":       len(rules),
	})
}

func runSheetsDeleteConditionalFormat(cmd *cobra.Command, args []string) error {
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
	index, _ := cmd.Flags().GetInt64("index")

	if index < 0 {
		return p.PrintError(fmt.Errorf("--index must be >= 0, got %d", index))
	}

	sheetID, err := getSheetID(svc, spreadsheetID, sheetName)
	if err != nil {
		return p.PrintError(err)
	}

	requests := []*sheets.Request{
		{
			DeleteConditionalFormatRule: &sheets.DeleteConditionalFormatRuleRequest{
				SheetId: sheetID,
				Index:   index,
			},
		},
	}

	_, err = svc.Spreadsheets.BatchUpdate(spreadsheetID, &sheets.BatchUpdateSpreadsheetRequest{
		Requests: requests,
	}).Do()
	if err != nil {
		return p.PrintError(fmt.Errorf("failed to delete conditional format rule: %w", err))
	}

	return p.Print(map[string]interface{}{
		"status":      "deleted",
		"spreadsheet": spreadsheetID,
		"sheet":       sheetName,
		"index":       index,
	})
}
