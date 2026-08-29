package cli

import (
	"fmt"
	"runtime/debug"

	"github.com/spf13/cobra"
)

// version, commit, and date are injected via -X ldflags at release build time (T9). They stay
// "" for a plain `go build`/`go install`, in which case buildInfoFallback supplies something
// better than "unknown" whenever the binary carries module information at all (T9).
var (
	version = ""
	commit  = ""
	date    = ""
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the hasp version",
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := fmt.Fprintln(cmd.OutOrStdout(), buildVersionString())
			return err
		},
	}
}

// buildVersionString never reports "unknown" for a build that carries module info at all (T9):
// ldflags-injected values win when present; otherwise runtime/debug.ReadBuildInfo() is the
// fallback for a go install-built binary that never went through GoReleaser.
func buildVersionString() string {
	v, c, d := version, commit, date

	if v == "" {
		if info, ok := debug.ReadBuildInfo(); ok {
			if info.Main.Version != "" && info.Main.Version != "(devel)" {
				v = info.Main.Version
			}
			for _, s := range info.Settings {
				switch s.Key {
				case "vcs.revision":
					if c == "" {
						c = s.Value
					}
				case "vcs.time":
					if d == "" {
						d = s.Value
					}
				}
			}
		}
	}

	if v == "" {
		v = "(devel)"
	}
	if c == "" {
		c = "unknown"
	}
	if d == "" {
		d = "unknown"
	}

	return fmt.Sprintf("hasp %s (commit %s, built %s)", v, c, d)
}
