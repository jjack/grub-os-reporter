//go:build linux

package cli

import (
	"github.com/jjack/grubstation/internal/grub"
	"github.com/spf13/cobra"
)

func NewBootListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available boot options from GRUB",
		RunE: func(cmd *cobra.Command, args []string) error {
			g := grub.NewGrub()
			g.ConfigPath = GlobalConfig.Grub.Path
			options, err := g.GetBootOptions()
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
