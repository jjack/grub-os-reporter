package cli

import (
	"fmt"
	"os"

	"github.com/jjack/grubstation/internal/config"
	"github.com/jjack/grubstation/internal/servicemanager"
	"github.com/jjack/grubstation/internal/version"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

type CLI struct {
	RootCmd *cobra.Command
}

func GetManager() (servicemanager.Manager, error) {
	registry := servicemanager.NewRegistry()
	servicemanager.RegisterDefaultServices(registry)
	mgr, err := registry.Detect()
	if err != nil {
		return nil, fmt.Errorf("manager detection failed: %w", err)
	}
	return mgr, nil
}

// GetConfig loads the configuration from the specified path or the default.
func GetConfig(cmd *cobra.Command) (*config.Config, string, error) {
	cfgFile, _ := cmd.Flags().GetString("config")
	resolved := cfgFile
	if resolved == "" {
		resolved = config.DefaultConfigPath()
	}

	cfg, err := config.LoadConfig(resolved)
	if err != nil {
		if cfgFile != "" || !os.IsNotExist(err) {
			return nil, "", fmt.Errorf("failed to read config file %s: %w", resolved, err)
		}
		cfg = &config.Config{}
	}

	return cfg, resolved, nil
}

func NewCLI() *CLI {
	cli := &CLI{}

	var cfgFile string
	var debugMode bool

	rootCmd := &cobra.Command{
		Use:   "grubstation",
		Short: "Remote Boot Agent for Home Assistant",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Name() == "help" || cmd.Name() == "init" || cmd.Name() == "version" {
				return nil
			}

			if debugMode || os.Getenv("DEBUG") == "true" {
				zerolog.SetGlobalLevel(zerolog.DebugLevel)
			}

			return nil
		},
	}

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is /etc/grubstation/config.yaml)")
	rootCmd.PersistentFlags().BoolVar(&debugMode, "debug", false, "enable debug logging")

	rootCmd.AddCommand(NewBootCmd())
	rootCmd.AddCommand(NewConfigCmd())
	rootCmd.AddCommand(NewServeCmd())
	rootCmd.AddCommand(NewServiceCmd())
	rootCmd.AddCommand(NewSetupCmd())
	rootCmd.AddCommand(NewVersionCmd())

	// get rid of the completion command because it doesn't make sense here
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	cli.RootCmd = rootCmd
	return cli
}

func NewVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version number of GrubStation",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("GrubStation %s\n", version.Version)
		},
	}
}

func (cli *CLI) Execute() error {
	return cli.RootCmd.Execute()
}
