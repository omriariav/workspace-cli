package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/omriariav/workspace-cli/internal/config"
	"github.com/omriariav/workspace-cli/internal/printer"
	"github.com/omriariav/workspace-cli/internal/updatecheck"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/api/googleapi"
)

var (
	cfgFile string
	format  string
	quiet   bool
)

var rootCmd = &cobra.Command{
	Use:   "gws",
	Short: "Google Workspace CLI",
	Long: `gws is a unified command-line interface for Google Workspace services.

It provides structured, token-efficient access to Gmail, Calendar, Drive,
Docs, Sheets, Slides, Tasks, Chat, Forms, Contacts, Groups, Keep,
and Custom Search.`,
	// SilenceErrors prevents Cobra from re-printing errors we already
	// emitted via PrintError (stderr structured JSON). SilenceUsage
	// prevents the auto-generated help dump on validation failures —
	// we surface a one-line "Error: ..." on stderr instead and direct
	// users to --help. Both must be set on rootCmd (not in PreRun),
	// since arg/flag validation runs before PersistentPreRun.
	SilenceErrors: true,
	SilenceUsage:  true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		emitVersionNotice(cmd, os.Stderr, quiet, os.Getenv("GWS_NO_UPDATE_CHECK") != "")
	},
}

// emitVersionNotice writes a low-noise line when a newer release is
// available. All errors are swallowed so unrelated commands stay healthy.
// Suppressed by --quiet, by GWS_NO_UPDATE_CHECK (passed in as suppressEnv),
// and on the version command itself (which has its own --check path) and
// shell completion subcommands.
func emitVersionNotice(cmd *cobra.Command, w io.Writer, quietFlag, suppressEnv bool) {
	if quietFlag || suppressEnv {
		return
	}
	if cmd == nil {
		return
	}
	if cmd == versionCmd || (cmd.Parent() != nil && cmd.Parent().Name() == "completion") {
		return
	}

	checker := newVersionChecker()
	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()

	res, err := checker.Check(ctx, Version, false)
	if err != nil || res == nil {
		return
	}
	if notice := updatecheck.FormatPassiveNotice(res); notice != "" {
		fmt.Fprint(w, notice)
	}
}

// Exit codes per the #190 contract:
//
//	0 — success
//	1 — generic API / runtime error
//	2 — CLI usage error (wrong args, unknown flag)
//	3 — auth failure (HTTP 401 / 403)
//	4 — transient / retryable (HTTP 429, 5xx)
const (
	ExitOK        = 0
	ExitError     = 1
	ExitUsage     = 2
	ExitAuth      = 3
	ExitTransient = 4
)

// Execute runs the root command and exits with the appropriate code.
func Execute() error {
	code := executeAndResolve(os.Stderr)
	if code != ExitOK {
		os.Exit(code)
	}
	return nil
}

// executeAndResolve runs the root command and returns the exit code.
// Errors from PrintError are mapped via exitCodeForError; Cobra usage
// errors are printed to errW as plain text and exit ExitUsage.
func executeAndResolve(errW io.Writer) int {
	return resolveExitError(rootCmd.Execute(), errW)
}

// resolveExitError maps a Cobra execution error to an exit code. Pure
// function — no os.Exit, no rootCmd reference — so unit tests can drive
// every branch of the dispatch contract.
func resolveExitError(err error, errW io.Writer) int {
	if err == nil {
		return ExitOK
	}

	// Printed via PrintError → already on stderr; just map the code.
	var printed *printer.AlreadyPrintedError
	if errors.As(err, &printed) {
		// Runtime input-validation paths wrap their message in
		// usageError so the user sees the same exit code (2) and
		// format (plain text) as Cobra's own arg/flag errors.
		var ue *usageError
		if errors.As(err, &ue) {
			return ExitUsage
		}
		return exitCodeForError(err)
	}

	// Unprinted error from rootCmd.Execute. Print it ourselves on
	// stderr, then decide the exit code: Cobra arg/flag validation
	// failures get ExitUsage; any other unwrapped runtime error
	// (e.g. a RunE that forgot to call PrintError) gets ExitError.
	fmt.Fprintf(errW, "Error: %s\n", err.Error())
	if isCobraUsageError(err) {
		return ExitUsage
	}
	return ExitError
}

// isCobraUsageError reports whether an error returned from rootCmd.Execute
// came from Cobra's own arg/flag validation rather than a RunE. Cobra does
// not expose typed sentinels for these, so we match the stable shapes
// Cobra produces — using strict prefix/substring patterns so unrelated
// runtime errors that happen to contain a Cobra-ish word don't get
// misclassified as ExitUsage.
func isCobraUsageError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	// Prefix-only patterns. Cobra emits these at the start of the
	// error message; a runtime error that happens to say e.g.
	// "this argument is invalid" will not match.
	for _, p := range cobraUsageErrorPrefixes {
		if strings.HasPrefix(msg, p) {
			return true
		}
	}
	// Argument-count errors from cobra.ExactArgs/MinimumNArgs/etc.
	// Cobra renders them as "accepts N arg(s), received M".
	if strings.HasPrefix(msg, "accepts ") && strings.Contains(msg, "arg(s), received") {
		return true
	}
	if msg == "subcommand is required" {
		return true
	}
	return false
}

var cobraUsageErrorPrefixes = []string{
	"unknown command \"",       // unknown command "foo" for "gws"
	"unknown flag: ",           // unknown flag: --bogus
	"unknown shorthand flag: ", // unknown shorthand flag: 'x' in -x
	"flag needs an argument: ", // flag needs an argument: --to
	`invalid argument "`,       // invalid argument "abc" for "--max" flag: ...
	"required flag(s) ",        // required flag(s) "to" not set
	"requires at least ",       // cobra.MinimumNArgs
	"requires exactly ",        // pre-Cobra-1.x exact-args message
}

// exitCodeForError maps a Go error to the appropriate CLI exit code by
// inspecting the underlying googleapi.Error HTTP status.
func exitCodeForError(err error) int {
	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) {
		switch {
		case apiErr.Code == 401 || apiErr.Code == 403:
			return ExitAuth
		case apiErr.Code == 429 || apiErr.Code >= 500:
			return ExitTransient
		}
	}
	return ExitError
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ~/.config/gws/config.yaml)")
	rootCmd.PersistentFlags().StringVar(&format, "format", "json", "output format: json, text, or yaml")
	rootCmd.PersistentFlags().BoolVar(&quiet, "quiet", false, "suppress output (useful for scripted actions)")

	viper.BindPFlag("format", rootCmd.PersistentFlags().Lookup("format"))
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		configDir := config.GetConfigDir()
		viper.AddConfigPath(configDir)
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	// Environment variables
	viper.SetEnvPrefix("GWS")
	viper.AutomaticEnv()

	// Read config file if it exists
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			fmt.Fprintf(os.Stderr, "Error reading config: %v\n", err)
		}
	}
}

func GetFormat() string {
	return viper.GetString("format")
}

// GetPrinter returns a Printer based on current flags.
// Returns NullPrinter when --quiet is set, otherwise the format-appropriate printer.
func GetPrinter() printer.Printer {
	if quiet {
		return printer.NewNullPrinter()
	}
	return printer.New(os.Stdout, GetFormat())
}
