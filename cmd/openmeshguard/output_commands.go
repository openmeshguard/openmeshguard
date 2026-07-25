package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/openmeshguard/openmeshguard/internal/output"
	"github.com/spf13/cobra"
)

const maxProjectionInputBytes = 64 << 20

type projectionOptions struct {
	Input  string
	Output string
	Format string
}

type scoreOptions struct {
	Input         string
	Output        string
	Namespace     string
	FailOn        string
	FailOnUnknown bool
}

func newReportCommand() *cobra.Command {
	opts := projectionOptions{Input: "-", Output: "-", Format: "html"}
	cmd := &cobra.Command{
		Use:   "report",
		Short: "Render a static report from canonical JSON",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if strings.ToLower(strings.TrimSpace(opts.Format)) != "html" {
				return fmt.Errorf("unsupported report format %q; want html", opts.Format)
			}
			return runProjection(cmd, opts, output.WriteHTML)
		},
	}
	addProjectionFlags(cmd, &opts)
	return cmd
}

func newExportCommand() *cobra.Command {
	opts := projectionOptions{Input: "-", Output: "-", Format: "sarif"}
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export canonical JSON to a compatibility format",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if strings.ToLower(strings.TrimSpace(opts.Format)) != "sarif" {
				return fmt.Errorf("unsupported export format %q; want sarif", opts.Format)
			}
			return runProjection(cmd, opts, output.WriteSARIF)
		},
	}
	addProjectionFlags(cmd, &opts)
	return cmd
}

func newScoreCommand() *cobra.Command {
	opts := scoreOptions{Input: "-", Output: "-"}
	cmd := &cobra.Command{
		Use:   "score",
		Short: "Read canonical JSON scores and evaluate CI thresholds",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			failOn, err := normalizeFailOn(opts.FailOn)
			if err != nil {
				return err
			}
			data, err := readProjectionInput(cmd, opts.Input)
			if err != nil {
				return err
			}
			if err := writeProjectionOutput(cmd, opts.Output, func(writer io.Writer) error {
				return output.WriteScore(writer, bytes.NewReader(data), opts.Namespace)
			}); err != nil {
				return err
			}
			failed, err := output.FailsThreshold(bytes.NewReader(data), failOn, opts.FailOnUnknown)
			if err != nil {
				return fmt.Errorf("evaluate score exit status: %w", err)
			}
			if failed {
				return fmt.Errorf("%w: canonical findings meet the configured failure condition", errFindingsThreshold)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&opts.Input, "input", "-", "canonical JSON input path; - reads stdin")
	cmd.Flags().StringVar(&opts.Output, "output", "-", "score output path; - writes stdout")
	cmd.Flags().StringVar(&opts.Namespace, "namespace", "", "show the canonical score for one namespace")
	cmd.Flags().StringVar(&opts.FailOn, "fail-on", "", "exit 1 for open findings at or above this severity")
	cmd.Flags().BoolVar(&opts.FailOnUnknown, "fail-on-unknown", false, "exit 1 when any finding is unknown")
	return cmd
}

func addProjectionFlags(cmd *cobra.Command, opts *projectionOptions) {
	cmd.Flags().StringVar(&opts.Input, "input", "-", "canonical JSON input path; - reads stdin")
	cmd.Flags().StringVar(&opts.Output, "output", "-", "output path; - writes stdout")
	cmd.Flags().StringVar(&opts.Format, "format", opts.Format, "output format")
}

func runProjection(
	cmd *cobra.Command,
	opts projectionOptions,
	render func(io.Writer, io.Reader) error,
) error {
	data, err := readProjectionInput(cmd, opts.Input)
	if err != nil {
		return err
	}
	return writeProjectionOutput(cmd, opts.Output, func(writer io.Writer) error {
		return render(writer, bytes.NewReader(data))
	})
}

func readProjectionInput(cmd *cobra.Command, path string) ([]byte, error) {
	path = strings.TrimSpace(path)
	var reader io.Reader
	var closeInput func() error
	if path == "" || path == "-" {
		reader = cmd.InOrStdin()
		closeInput = func() error { return nil }
	} else {
		file, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("open canonical JSON input %s: %w", path, err)
		}
		reader = file
		closeInput = file.Close
	}
	data, err := io.ReadAll(io.LimitReader(reader, maxProjectionInputBytes+1))
	closeErr := closeInput()
	if err != nil {
		return nil, fmt.Errorf("read canonical JSON input: %w", err)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close canonical JSON input: %w", closeErr)
	}
	if len(data) > maxProjectionInputBytes {
		return nil, fmt.Errorf("read canonical JSON input: input exceeds %d bytes", maxProjectionInputBytes)
	}
	return data, nil
}

func writeProjectionOutput(cmd *cobra.Command, path string, render func(io.Writer) error) error {
	path = strings.TrimSpace(path)
	if path == "" || path == "-" {
		return render(cmd.OutOrStdout())
	}
	directory := filepath.Dir(path)
	temporary, err := os.CreateTemp(directory, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temporary output for %s: %w", path, err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)

	if err := render(temporary); err != nil {
		_ = temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary output for %s: %w", path, err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("replace output %s: %w", path, err)
	}
	return nil
}

func normalizeFailOn(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "", "critical", "high", "medium", "low", "info":
		return value, nil
	default:
		return "", fmt.Errorf(
			"invalid --fail-on severity %q; want critical, high, medium, low, or info",
			value,
		)
	}
}
