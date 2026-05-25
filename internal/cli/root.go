package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/jjack/grubstation/internal/config"
	"github.com/jjack/grubstation/internal/grub"
	"github.com/jjack/grubstation/internal/homeassistant"
	"github.com/jjack/grubstation/internal/host"
	"github.com/jjack/grubstation/internal/servicemanager"
	"github.com/jjack/grubstation/internal/version"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type CLI struct {
	Config  *config.Config
	RootCmd *cobra.Command
}

type CommandDeps struct {
	Config     *config.Config
	ConfigFile string
	Grub       *grub.Grub
	Registry   *servicemanager.Registry
	Host       *host.Host
	SaveConfig func(*config.Config, string) error
	DiscoverHA func(context.Context) ([]homeassistant.ServiceInstance, error)
}

func (cd *CommandDeps) Manager(ctx context.Context) (servicemanager.Manager, error) {
	mgr, err := cd.Registry.Detect(ctx)
	if err != nil {
		return nil, fmt.Errorf("manager detection failed: %w", err)
	}
	return mgr, nil
}

func (cli *CLI) LoadConfig(cmd *cobra.Command, cfgFile string) (string, error) {
	v := config.NewViper(cfgFile)
	// Bind flags to viper keys
	_ = v.BindPFlag("grub.config_path", cmd.Flags().Lookup(config.FlagGrubConfig))
	_ = v.BindPFlag("host.mac", cmd.Flags().Lookup(config.FlagMac))
	_ = v.BindPFlag("host.address", cmd.Flags().Lookup(config.FlagAddress))
	_ = v.BindPFlag("wake_on_lan.address", cmd.Flags().Lookup(config.FlagWolBroadcastAddress))
	_ = v.BindPFlag("wake_on_lan.port", cmd.Flags().Lookup(config.FlagWolBroadcastPort))
	_ = v.BindPFlag("homeassistant.url", cmd.Flags().Lookup(config.FlagHassURL))
	_ = v.BindPFlag("homeassistant.webhook_id", cmd.Flags().Lookup(config.FlagHassWebhook))
	_ = v.BindPFlag("daemon.port", cmd.Flags().Lookup(config.FlagAgentPort))
	_ = v.BindPFlag("daemon.api_key", cmd.Flags().Lookup(config.FlagDaemonKey))

	if err := v.ReadInConfig(); err != nil {
		if cfgFile != "" {
			return "", fmt.Errorf("failed to read config file %s: %w", cfgFile, err)
		}
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok && !os.IsNotExist(err) {
			return "", fmt.Errorf("failed to read config file: %w", err)
		}
	}

	cfg, err := config.Unmarshal(v)
	if err != nil {
		return "", err
	}

	if err := cfg.Validate(); err != nil {
		return "", err
	}

	cli.Config = cfg
	resolved := v.ConfigFileUsed()
	if resolved == "" {
		resolved = cfgFile
	}
	if resolved == "" {
		resolved = config.DefaultConfigPath()
	}
	return resolved, nil
}

func NewCLI() *CLI {
	cli := &CLI{}

	deps := &CommandDeps{
		Config:     &config.Config{},
		Grub:       grub.NewGrub(),
		Registry:   servicemanager.NewRegistry(),
		Host:       host.New(),
		SaveConfig: config.Save,
		DiscoverHA: homeassistant.Discover,
	}

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

			deps.Config = cli.Config
			deps.ConfigFile = resolved
			return nil
		},
	}

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is /etc/grubstation/config.yaml)")
	rootCmd.PersistentFlags().String(config.FlagGrubConfig, "", "GRUB config path override")
	rootCmd.PersistentFlags().String(config.FlagMac, "", "MAC Address override")
	rootCmd.PersistentFlags().String(config.FlagAddress, "", "Address override")
	rootCmd.PersistentFlags().String(config.FlagWolBroadcastAddress, "", "WOL target address override (defaults to 255.255.255.255)")
	rootCmd.PersistentFlags().Int(config.FlagWolBroadcastPort, 9, "WOL target port override (defaults to 9)")
	rootCmd.PersistentFlags().String(config.FlagDaemonKey, "", "API key for the daemon")
	rootCmd.PersistentFlags().String(config.FlagHassURL, "", "Home Assistant URL override")
	rootCmd.PersistentFlags().String(config.FlagHassWebhook, "", "Home Assistant Webhook ID override")
	rootCmd.PersistentFlags().BoolVar(&debugMode, "debug", false, "enable debug logging")

	rootCmd.AddCommand(NewBootCmd(deps))
	rootCmd.AddCommand(NewConfigCmd(deps))
	rootCmd.AddCommand(NewServeCmd(deps))
	rootCmd.AddCommand(NewServiceCmd(deps))
	rootCmd.AddCommand(NewSetupCmd(deps))
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
