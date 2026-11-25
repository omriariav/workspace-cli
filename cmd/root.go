package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/omriariav/workspace-cli/internal/auth"
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

	// executedCmd is the command Cobra dispatched to, recorded in
	// PersistentPreRunE so post-run diagnostics (scope hints) know which
	// service was involved.
	executedCmd *cobra.Command
)

var rootCmd = &cobra.Command{
	Use:   "gws",
	Short: "Google Workspace CLI",
	Long: `gws is a unified command-line interface for Google Workspace services.

It provides structured, token-efficient access to Gmail, Calendar, Drive,
Docs, Sheets, Slides, Tasks, Chat, Forms, Contacts, Groups, Keep,
Workspace Events, and Custom Search.`,
	// SilenceErrors prevents Cobra from re-printing errors we already
	// emitted via PrintError (stderr structured JSON). SilenceUsage
	// prevents the auto-generated help dump on validation failures —
	// we surface a one-line "Error: ..." on stderr instead and direct
	// users to --help. Both must be set on rootCmd (not in PreRun),
	// since arg/flag validation runs before PersistentPreRun.
	SilenceErrors: true,
	SilenceUsage:  true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		executedCmd = cmd
		emitVersionNotice(cmd, os.Stderr, quiet, os.Getenv("GWS_NO_UPDATE_CHECK") != "")
		// Cobra checks required flags after this hook. Check them first,
		// so an incomplete command reports its usage error instead of
		// asking for confirmation of something that cannot run.
		if err := cmd.ValidateRequiredFlags(); err != nil {
			return err
		}
		// A token that was supplied but is empty is refused for every
		// command, here, before any of them can act: not only the ones
		// that go on to build an API client. Some answer from local data
		// (the Chat space cache) and would otherwise serve the stored
		// login's data to a caller that selected a different credential.
		if cmd != versionCmd && (cmd.Parent() == nil || cmd.Parent().Name() != "completion") {
			if err := config.CheckAccessToken(); err != nil {
				return usageErrorf("%s", err.Error())
			}
		}
		yes, _ := cmd.Flags().GetBool("yes")
		if err := confirmPolicyUnknown(cmd, yes); err != nil {
			return err
		}
		return confirmGate(cmd, config.GetConfirmSensitive(), yes, os.Stdin, os.Stderr, stdinIsTerminal())
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
	// --schema / --show-scopes describe a command without running it, so
	// they are answered before Cobra validates args or required flags.
	if handled, err := handleIntrospection(rootCmd, os.Args[1:], os.Stdout); handled {
		return resolveExitError(err, errW)
	}
	return resolveExitError(rootCmd.Execute(), errW)
}

// storedOAuthLoginActive reports whether commands authenticate with the
// token stored by `gws auth login` (not an external access token and not
// a service-account mode).
func storedOAuthLoginActive() bool {
	if config.GetAccessToken() != "" {
		return false
	}
	modeCfg, err := resolveConfiguredAuthMode()
	return err == nil && modeCfg.Mode == auth.AuthModeOAuth
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

	rootCmd.PersistentFlags().String("access-token", "", "use an externally managed OAuth access token (prefer the GWS_ACCESS_TOKEN env var; a flag value is visible in process listings)")
	rootCmd.PersistentFlags().Bool("yes", false, "skip the confirmation prompt for sensitive commands (when confirm_sensitive is on)")
	rootCmd.PersistentFlags().Bool("schema", false, "print this command's flags and arguments as JSON and exit without running it")
	rootCmd.PersistentFlags().Bool("show-scopes", false, "print the OAuth scopes this command's service uses as JSON and exit without running it")

	viper.BindPFlag("format", rootCmd.PersistentFlags().Lookup("format"))
	// Read straight from the flag, never through viper: the config file
	// must not be able to supply an access token.
	config.AccessTokenFlag = func() string {
		v, _ := rootCmd.PersistentFlags().GetString("access-token")
		return v
	}
	config.AccessTokenFlagSet = func() bool {
		return rootCmd.PersistentFlags().Changed("access-token")
	}
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
	configLoadErr = nil
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			configLoadErr = err
			fmt.Fprintf(os.Stderr, "Error reading config: %v\n", err)
		}
	}
}

// configLoadErr is set when a config file exists but could not be read or
// parsed. Most settings then fall back to their defaults, which is fine
// for a format or a timezone but not for a safety policy: see
// confirmPolicyUnknown.
var configLoadErr error

// confirmPolicyUnknown reports an error when a sensitive command is about
// to run but it cannot be established whether the confirmation gate is
// on: the setting has a value that is not a recognisable on/off (a typo
// in GWS_CONFIRM_SENSITIVE or in the config file), or the config file
// that may turn the gate on could not be loaded and the environment does
// not settle it. Treating either as "gate off" would silently remove the
// protection the user asked for. --yes is an explicit approval.
func confirmPolicyUnknown(cmd *cobra.Command, yes bool) error {
	if yes {
		return nil
	}
	effect, sensitive := sensitiveEffect(cmd)
	if !sensitive {
		return nil
	}
	if _, policyErr := config.ConfirmSensitivePolicy(); policyErr != nil {
		return fmt.Errorf("the confirmation setting could not be understood (%v), so it is unknown whether confirm_sensitive is on; `gws %s` %s and was not run. Correct the setting, or re-run with --yes after the user approves", policyErr, commandKey(cmd), effect)
	}
	// The setting parsed. If it came from the environment the policy is
	// known whatever state the config file is in.
	if _, set := os.LookupEnv(config.ConfirmSensitiveEnv); set || configLoadErr == nil {
		return nil
	}
	return fmt.Errorf("the config file could not be read (%v), so it is unknown whether confirm_sensitive is on; `gws %s` %s and was not run. Fix the config file, set %s to true or false, or re-run with --yes after the user approves", configLoadErr, commandKey(cmd), effect, config.ConfirmSensitiveEnv)
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
	return &hintingPrinter{Printer: printer.New(os.Stdout, GetFormat())}
}

// hintingPrinter adds the read-only scope hint to an insufficient-scope
// error before it is printed, so the hint is part of the one structured
// error (JSON, YAML or text) rather than an extra line after it, and is
// suppressed together with it under --quiet.
type hintingPrinter struct {
	printer.Printer
}

func (h *hintingPrinter) PrintError(err error) error {
	return h.Printer.PrintError(withScopeHint(err))
}

// withScopeHint returns err with the scope hint appended when it applies:
// the failure is an auth error, the stored OAuth login is the credential
// in use, and the command's service was not authorized in full.
func withScopeHint(err error) error {
	if err == nil || exitCodeForError(err) != ExitAuth || !storedOAuthLoginActive() {
		return err
	}
	hint := readOnlyScopeHint(executedCmd, err, auth.LoadGrantedAccess())
	if hint == "" {
		return err
	}
	return &hintedError{err: err, hint: hint}
}

// hintedError carries a hint in its message while still unwrapping to the
// original error, so exit-code mapping keeps seeing the API error.
type hintedError struct {
	err  error
	hint string
}

func (e *hintedError) Error() string { return e.err.Error() + "\n" + e.hint }
func (e *hintedError) Unwrap() error { return e.err }
