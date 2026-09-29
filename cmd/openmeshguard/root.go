package main

import (
	"errors"
	"fmt"
	"runtime/debug"

	"github.com/openmeshguard/openmeshguard/internal/resolver"
	"github.com/spf13/cobra"
)

const (
	versionPlaceholder = "dev"
)

// version is set via -ldflags "-X main.version=..." on release builds.
var version string

var errFindingsThreshold = errors.New("findings threshold met")

type versionInfo struct {
	Version         string
	ResolverVersion string
}

func defaultVersionInfo() versionInfo {
	resolved := resolver.New()
	return versionInfo{
		Version:         scannerVersion(version, readModuleVersion),
		ResolverVersion: resolved.Version(),
	}
}

// scannerVersion resolves the reported scanner version: an explicit ldflags
// value wins, then the module version stamped by `go install module@version`,
// then the local-build placeholder. Local `go build` yields "(devel)" from
// build info, which must keep reporting the placeholder.
func scannerVersion(ldflagsVersion string, moduleVersion func() string) string {
	if ldflagsVersion != "" {
		return ldflagsVersion
	}
	if fromModule := moduleVersion(); fromModule != "" && fromModule != "(devel)" {
		return fromModule
	}
	return versionPlaceholder
}

func readModuleVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	return info.Main.Version
}

func newRootCommand(info versionInfo) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "openmeshguard",
		Short:         "Read-only Istio mesh posture scanner",
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	cmd.AddCommand(newVersionCommand(info))
	cmd.AddCommand(newScanCommand(info))
	cmd.AddCommand(newControlsCommand())
	cmd.AddCommand(newReportCommand())
	cmd.AddCommand(newExportCommand())
	cmd.AddCommand(newScoreCommand())

	return cmd
}

func newVersionCommand(info versionInfo) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print OpenMeshGuard version information",
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := fmt.Fprintf(
				cmd.OutOrStdout(),
				"version=%s\nresolverVersion=%s\n",
				info.Version,
				info.ResolverVersion,
			)
			return err
		},
	}
}

func exitCode(err error) int {
	if errors.Is(err, errFindingsThreshold) {
		return 1
	}
	return 2
}
