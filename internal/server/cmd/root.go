package cmd

import (
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tacokumo/portal-api/pkg/platform"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "portal-api server",
		RunE: func(cmd *cobra.Command, args []string) error {
			var logLevel slog.Level
			switch strings.ToLower(os.Getenv("LOG_LEVEL")) {
			case "debug":
				logLevel = slog.LevelDebug
			case "info":
				logLevel = slog.LevelInfo
			case "warn":
				logLevel = slog.LevelWarn
			case "error":
				logLevel = slog.LevelError
			default:
				logLevel = slog.LevelInfo
			}
			logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
			srv := platform.NewServer(logger)
			if err := srv.Start(cmd.Context()); err != nil {
				return err
			}

			return nil
		},
		SilenceUsage: true,
	}

	// サブコマンドを追加
	cmd.AddCommand(newConfigCommand())

	return cmd
}
