package cmd

import (
	"fmt"
	"runtime"
	"runtime/debug"

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
	if Version == "" {
		if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
			Version = info.Main.Version
		} else {
			Version = "dev"
		}
	}
	if Commit == "" {
		if info, ok := debug.ReadBuildInfo(); ok {
			for _, s := range info.Settings {
				if s.Key == "vcs.revision" && len(s.Value) >= 7 {
					Commit = s.Value[:7]
					break
				}
			}
		}
		if Commit == "" {
			Commit = "unknown"
		}
	}
	if BuildDate == "" {
		if info, ok := debug.ReadBuildInfo(); ok {
			for _, s := range info.Settings {
				if s.Key == "vcs.time" {
					BuildDate = s.Value
					break
				}
			}
		}
		if BuildDate == "" {
			BuildDate = "unknown"
		}
	}

	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  "Prints the version, commit hash, and build date of gws.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("gws version %s\n", Version)
		fmt.Printf("  commit:  %s\n", Commit)
		fmt.Printf("  built:   %s\n", BuildDate)
		fmt.Printf("  go:      %s\n", runtime.Version())
		fmt.Printf("  os/arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	},
}
