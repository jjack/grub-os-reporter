package cli

import (
	"github.com/spf13/cobra"
)

func NewConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage the grubstation configuration",
	}

	cmd.AddCommand(NewConfigValidateCmd())
	cmd.AddCommand(NewConfigInitCmd())

	return cmd
}
