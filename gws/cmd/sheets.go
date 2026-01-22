package cmd

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"strings"

	"github.com/omriariav/workspace-cli/gws/internal/client"
	"github.com/omriariav/workspace-cli/gws/internal/printer"
	"github.com/spf13/cobra"
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

func init() {
	rootCmd.AddCommand(sheetsCmd)
	sheetsCmd.AddCommand(sheetsInfoCmd)
	sheetsCmd.AddCommand(sheetsReadCmd)
	sheetsCmd.AddCommand(sheetsListCmd)

	// Read flags
	sheetsReadCmd.Flags().String("output-format", "json", "Output format: json or csv")
	sheetsReadCmd.Flags().Bool("headers", true, "Treat first row as headers (for json output)")
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
