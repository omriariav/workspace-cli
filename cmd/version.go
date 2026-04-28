package cmd

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/omriariav/workspace-cli/internal/config"
	"github.com/omriariav/workspace-cli/internal/updatecheck"
	"github.com/spf13/cobra"
)

// Version information - set via ldflags at build time.
// Falls back to Go module version from debug.ReadBuildInfo()
// when installed via `go install`.
var (
	Version   = ""
	Commit    = ""
	BuildDate = ""
)

func init() {
	if Version == "" || Commit == "" || BuildDate == "" {
		if info, ok := debug.ReadBuildInfo(); ok {
			if Version == "" && info.Main.Version != "" {
				Version = info.Main.Version
			}
			for _, s := range info.Settings {
				if s.Key == "vcs.revision" && Commit == "" && len(s.Value) >= 7 {
					Commit = s.Value[:7]
				}
				if s.Key == "vcs.time" && BuildDate == "" {
					BuildDate = s.Value
				}
			}
		}
	}
	if Version == "" {
		Version = "dev"
	}
	if Commit == "" {
		Commit = "unknown"
	}
	if BuildDate == "" {
		BuildDate = "unknown"
	}

	versionCmd.Flags().Bool("check", false, "Check GitHub for the latest release and report whether the installed version is stale")
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  "Prints the version, commit hash, and build date of gws.",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("gws version %s\n", Version)
		fmt.Printf("  commit:  %s\n", Commit)
		fmt.Printf("  built:   %s\n", BuildDate)
		fmt.Printf("  go:      %s\n", runtime.Version())
		fmt.Printf("  os/arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)

		check, _ := cmd.Flags().GetBool("check")
		if !check {
			return nil
		}

		out := cmd.OutOrStdout()
		fmt.Fprintln(out)

		ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Second)
		defer cancel()

		checker := newVersionChecker()
		res, err := checker.Check(ctx, Version, true)
		if err != nil {
			fmt.Fprintf(out, "update check: failed to query GitHub releases: %v\n", err)
			return nil
		}
		if res.Skipped {
			fmt.Fprintf(out, "update check: skipped (%s); latest release is %s\n", res.SkippedReason, res.Latest)
			return nil
		}
		if res.Stale {
			fmt.Fprintf(out, "update check: a newer version is available\n  installed: %s\n  latest:    %s\n  upgrade:   https://github.com/omriariav/workspace-cli/releases/latest\n", res.Current, res.Latest)
			return nil
		}
		fmt.Fprintf(out, "update check: gws is up to date (latest: %s)\n", res.Latest)
		return nil
	},
}

// newVersionChecker builds an updatecheck.Checker rooted at the gws config dir.
// Tests override the resulting checker via testVersionChecker before invoking
// command code paths.
func newVersionChecker() *updatecheck.Checker {
	if testVersionChecker != nil {
		return testVersionChecker
	}
	cachePath := filepath.Join(config.GetConfigDir(), "version-cache.json")
	return updatecheck.New(cachePath)
}

// testVersionChecker is a test seam that replaces the live GitHub-backed
// checker with one pointed at a httptest server and a temp cache path.
var testVersionChecker *updatecheck.Checker
