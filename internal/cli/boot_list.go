//go:build linux

package cli

import (
	"github.com/spf13/cobra"
)

func NewBootListCmd(deps *CommandDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available boot options from GRUB",
		RunE: func(cmd *cobra.Command, args []string) error {
			options, err := deps.Grub.GetBootOptions()
			if err != nil {
				return err
			}

			if len(options) == 0 {
				cmd.Println("No boot options found in GRUB config. (None found)")
				return nil
			}

			cmd.Println("Available Boot Options:")
			for _, opt := range options {
				cmd.Printf("- %s\n", opt)
			}
			return nil
		},
	}
}
