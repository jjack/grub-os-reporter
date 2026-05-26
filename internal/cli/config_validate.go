package cli

import (
	"github.com/spf13/cobra"
)

func NewConfigValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate an existing configuration file",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Configuration is already loaded and validated (structurally) by PersistentPreRunE
			cmd.Println("Configuration is valid.")
			return nil
		},
	}
}
