package main

import (
	"fmt"

	"github.com/openmeshguard/openmeshguard/internal/engine"
	"github.com/spf13/cobra"
)

func newControlsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "controls",
		Short: "Inspect and validate control packs",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "validate <path>",
		Short: "Validate a control pack against the frozen v1alpha1 contract",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := engine.ValidateFile(args[0]); err != nil {
				return err
			}
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "valid control pack: %s\n", args[0])
			return err
		},
	})
	return cmd
}
