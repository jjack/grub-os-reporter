//go:build linux

package cli

import (
	"github.com/spf13/cobra"
)

func NewBootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "boot",
		Short: "Manage boot options",
	}

	cmd.AddCommand(NewBootListCmd())
	cmd.AddCommand(NewBootPushCmd())

	return cmd
}
