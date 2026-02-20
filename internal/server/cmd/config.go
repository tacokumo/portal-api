package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/tacokumo/portal-api/pkg/config"
	"os"
)

func newConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Configuration management commands",
	}

	cmd.AddCommand(
		newConfigInitCommand(),
		newConfigValidateCommand(),
		newConfigShowCommand(),
	)

	return cmd
}

func newConfigInitCommand() *cobra.Command {
	var outputFile string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Generate sample configuration file",
		RunE: func(cmd *cobra.Command, args []string) error {
			if outputFile == "" {
				outputFile = "config.yaml"
			}

			if err := config.GenerateDefaultConfig(outputFile); err != nil {
				return err
			}

			fmt.Printf("Configuration file generated: %s\n", outputFile)
			fmt.Println("Please set required environment variables for sensitive information.")
			return nil
		},
	}

	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "output file path (default: config.yaml)")
	return cmd
}

func newConfigValidateCommand() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			// 一時的にconfigPathを設定してロード
			originalArgs := os.Args
			if configPath != "" {
				os.Args = append(os.Args[:1], "--config", configPath)
			}
			defer func() { os.Args = originalArgs }()

			cfg, err := config.Load()
			if err != nil {
				fmt.Printf("Configuration validation failed: %v\n", err)
				return err
			}

			fmt.Println("Configuration is valid!")
			fmt.Printf("Portal Name: %s\n", cfg.PortalName)
			fmt.Printf("Server Port: %d\n", cfg.Server.Port)
			return nil
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "", "configuration file path")
	return cmd
}

func newConfigShowCommand() *cobra.Command {
	var configPath string
	var showSecrets bool

	cmd := &cobra.Command{
		Use:   "show",
		Short: "Display current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			// 一時的にconfigPathを設定してロード
			originalArgs := os.Args
			if configPath != "" {
				os.Args = append(os.Args[:1], "--config", configPath)
			}
			defer func() { os.Args = originalArgs }()

			cfg, err := config.Load()
			if err != nil {
				return err
			}

			display, err := cfg.Display(!showSecrets)
			if err != nil {
				return err
			}

			fmt.Println(display)
			return nil
		},
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "", "configuration file path")
	cmd.Flags().BoolVarP(&showSecrets, "show-secrets", "s", false, "show secret values (use with caution)")
	return cmd
}
