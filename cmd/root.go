package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/omriariav/workspace-cli/internal/config"
	"github.com/omriariav/workspace-cli/internal/printer"
	"github.com/omriariav/workspace-cli/internal/updatecheck"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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

func Execute() error {
	return rootCmd.Execute()
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
