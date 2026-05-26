//go:build !linux

package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewBootCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "boot",
		Short: "Manage boot options (Not supported on this OS)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("bootloader management is only supported on Linux")
		},
	}
}
