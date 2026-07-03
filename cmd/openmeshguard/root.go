package main

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

const (
	versionPlaceholder         = "dev"
	resolverVersionPlaceholder = "resolver-v0-placeholder"
)

var errNotImplemented = errors.New("not implemented")

type versionInfo struct {
	Version         string
	ResolverVersion string
}

func defaultVersionInfo() versionInfo {
	return versionInfo{
		Version:         versionPlaceholder,
		ResolverVersion: resolverVersionPlaceholder,
	}
}

func newRootCommand(info versionInfo) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "openmeshguard",
		Short:         "Read-only Istio mesh posture scanner",
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	cmd.AddCommand(newVersionCommand(info))
	for _, name := range []string{"scan", "report", "export", "score"} {
		cmd.AddCommand(newStubCommand(name))
	}

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

func newStubCommand(name string) *cobra.Command {
	return &cobra.Command{
		Use:   name,
		Short: fmt.Sprintf("%s is not implemented yet", name),
		RunE: func(_ *cobra.Command, _ []string) error {
			return errNotImplemented
		},
	}
}

func exitCode(err error) int {
	if errors.Is(err, errNotImplemented) {
		return 2
	}

	return 1
}
