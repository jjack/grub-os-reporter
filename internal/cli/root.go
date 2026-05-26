package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jjack/grubstation/internal/config"
	"github.com/jjack/grubstation/internal/grub"
	"github.com/jjack/grubstation/internal/servicemanager"
	"github.com/jjack/grubstation/internal/version"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

type CLI struct {
	RootCmd *cobra.Command
}

type Env struct {
	Config     *config.Config
	State      *config.State
	ConfigPath string
	StatePath  string
	Grub       *grub.Grub
	Manager    servicemanager.Manager
}

// GetEnv loads the full application environment, including configuration, state, and services.
func GetEnv(cmd *cobra.Command) (*Env, error) {
	// 1. Resolve paths
	cfgPath, _ := cmd.Flags().GetString("config")
	if cfgPath == "" {
		cfgPath = config.DefaultConfigPath()
	}

	statePath := config.DefaultStatePath()
	// If config was overridden, we assume state is next to it unless we add a --state flag later
	if cmd.Flags().Changed("config") {
		statePath = filepath.Join(filepath.Dir(cfgPath), "state.json")
	}

	// 2. Load data
	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		if cmd.Flags().Changed("config") || !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to read config file %s: %w", cfgPath, err)
		}
		cfg = &config.Config{}
	}

	state, err := config.LoadState(statePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load state: %w", err)
	}

	// 3. Initialize services
	g := grub.NewGrub()
	g.ConfigPath = cfg.Grub.Path

	registry := servicemanager.NewRegistry()
	servicemanager.RegisterDefaultServices(registry)
	mgr, _ := registry.Detect() // Detect can fail if no supported init system found

	return &Env{
		Config:     cfg,
		State:      state,
		ConfigPath: cfgPath,
		StatePath:  statePath,
		Grub:       g,
		Manager:    mgr,
	}, nil
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
