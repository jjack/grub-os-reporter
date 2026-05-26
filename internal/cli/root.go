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
	Config  *config.Config
	RootCmd *cobra.Command
}

var (
	GlobalConfig     *config.Config
	GlobalConfigFile string
)

func GetManager() (servicemanager.Manager, error) {
	registry := servicemanager.NewRegistry()
	servicemanager.RegisterDefaultServices(registry)
	mgr, err := registry.Detect()
	if err != nil {
		return nil, fmt.Errorf("manager detection failed: %w", err)
	}
	return mgr, nil
}

func (cli *CLI) LoadConfig(cmd *cobra.Command, cfgFile string) (string, error) {
	resolved := cfgFile
	if resolved == "" {
		resolved = config.DefaultConfigPath()
	}

	cfg, err := config.LoadConfig(resolved)
	if err != nil {
		if cfgFile != "" || !os.IsNotExist(err) {
			return "", fmt.Errorf("failed to read config file %s: %w", resolved, err)
		}
		cfg = &config.Config{}
	}

	// Manually apply flag overrides
	if cmd.Flags().Changed(config.FlagGrubConfig) {
		cfg.Grub.Path, _ = cmd.Flags().GetString(config.FlagGrubConfig)
	}
	if cmd.Flags().Changed(config.FlagMac) {
		cfg.Host.MAC, _ = cmd.Flags().GetString(config.FlagMac)
	}
	if cmd.Flags().Changed(config.FlagInterface) {
		cfg.Host.Interface, _ = cmd.Flags().GetString(config.FlagInterface)
	}
	if cmd.Flags().Changed(config.FlagWolBroadcastAddress) {
		cfg.WakeOnLan.Address, _ = cmd.Flags().GetString(config.FlagWolBroadcastAddress)
	}
	if cmd.Flags().Changed(config.FlagWolBroadcastPort) {
		cfg.WakeOnLan.Port, _ = cmd.Flags().GetInt(config.FlagWolBroadcastPort)
	}
	if cmd.Flags().Changed(config.FlagAgentPort) {
		cfg.Daemon.Port, _ = cmd.Flags().GetInt(config.FlagAgentPort)
	}
	if cmd.Flags().Changed(config.FlagDaemonKey) {
		cfg.Daemon.APIKey, _ = cmd.Flags().GetString(config.FlagDaemonKey)
	}

	cli.Config = cfg
	return resolved, nil
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

			resolved, err := cli.LoadConfig(cmd, cfgFile)
			if err != nil {
				return err
			}

			GlobalConfig = cli.Config
			GlobalConfigFile = resolved
			return nil
		},
	}

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is /etc/grubstation/config.yaml)")
	rootCmd.PersistentFlags().String(config.FlagGrubConfig, "", "GRUB config path override")
	rootCmd.PersistentFlags().String(config.FlagMac, "", "MAC Address override")
	rootCmd.PersistentFlags().String(config.FlagInterface, "", "network interface to use")
	rootCmd.PersistentFlags().String(config.FlagWolBroadcastAddress, "", "WOL target address override (defaults to 255.255.255.255)")
	rootCmd.PersistentFlags().Int(config.FlagWolBroadcastPort, 9, "WOL target port override (defaults to 9)")
	rootCmd.PersistentFlags().String(config.FlagDaemonKey, "", "API key for the daemon")
	rootCmd.PersistentFlags().String(config.FlagHassURL, "", "Home Assistant URL override")
	rootCmd.PersistentFlags().String(config.FlagHassWebhook, "", "Home Assistant Webhook ID override")
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
