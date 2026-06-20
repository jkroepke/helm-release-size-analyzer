package cli

import (
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/jkroepke/helm-release-size-analyser/internal/analyse"
	"github.com/jkroepke/helm-release-size-analyser/internal/config"
	"github.com/jkroepke/helm-release-size-analyser/internal/helminstall"
	"github.com/jkroepke/helm-release-size-analyser/internal/releasesecret"
	"github.com/jkroepke/helm-release-size-analyser/internal/report"
	"github.com/jkroepke/helm-release-size-analyser/internal/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func ExitCode(_ error) int {
	return 1
}

func NewRootCommand(stdout, stderr io.Writer) *cobra.Command {
	configLoader := viper.New()

	var configFile string

	root := &cobra.Command{
		Use:           "helm-release-size-analyser",
		Short:         "Analyse the JSON stored in a Helm release Secret",
		Version:       version.String(),
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.PersistentFlags().StringVar(&configFile, "config", "", "configuration file")
	root.PersistentFlags().String("log-level", "info", "log level: debug, info, warn, error")
	root.PersistentFlags().String("log-format", "text", "log format: text or json")

	analyseCmd := &cobra.Command{
		Use:   "analyse CHART",
		Short: "Install a chart in memory and analyse its Helm release Secret",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig(configLoader, cmd, configFile)
			if err != nil {
				return err
			}

			logger := newLogger(stderr, cfg.LogLevel, cfg.LogFormat)
			logger.Debug("starting analysis", slog.String("chart", args[0]))

			installed, err := helminstall.Install(cmd.Context(), args[0], cfg, logger)
			if err != nil {
				return fmt.Errorf("install chart: %w", err)
			}

			releaseJSON, err := releasesecret.DecodeJSON(installed.Secret)
			if err != nil {
				return fmt.Errorf("decode release JSON: %w", err)
			}

			result, err := analyse.Build(releaseJSON)
			if err != nil {
				return fmt.Errorf("analyse release: %w", err)
			}

			err = report.Write(stdout, cfg.Output, result)
			if err != nil {
				return fmt.Errorf("write report: %w", err)
			}

			return nil
		},
	}
	addInstallFlags(analyseCmd)
	flags := analyseCmd.Flags()
	flags.StringP("output", "o", "table", "output format: table or json")

	releaseJSONCmd := &cobra.Command{
		Use:   "release-json CHART",
		Short: "Install a chart in memory and print its uncompressed Helm release JSON",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig(configLoader, cmd, configFile)
			if err != nil {
				return err
			}

			logger := newLogger(stderr, cfg.LogLevel, cfg.LogFormat)
			logger.Debug("generating release JSON", slog.String("chart", args[0]))

			installed, err := helminstall.Install(cmd.Context(), args[0], cfg, logger)
			if err != nil {
				return fmt.Errorf("install chart: %w", err)
			}

			releaseJSON, err := releasesecret.DecodeJSON(installed.Secret)
			if err != nil {
				return fmt.Errorf("decode release JSON: %w", err)
			}

			_, err = stdout.Write(releaseJSON)
			if err != nil {
				return fmt.Errorf("write release JSON: %w", err)
			}

			return nil
		},
	}
	addInstallFlags(releaseJSONCmd)

	root.AddCommand(analyseCmd, releaseJSONCmd)

	return root
}

func addInstallFlags(cmd *cobra.Command) {
	flags := cmd.Flags()
	flags.String("release-name", "", "release name (defaults to chart name)")
	flags.String("namespace", "default", "simulated release namespace")
	flags.StringSliceP("values", "f", nil, "values file (repeatable)")
	flags.StringArray("set", nil, "set a value")
	flags.StringArray("set-string", nil, "set a string value")
	flags.StringArray("set-file", nil, "set a value from a file")
	flags.Bool("include-crds", false, "include CRDs in the stored manifest")
}

func loadConfig(configLoader *viper.Viper, cmd *cobra.Command, configFile string) (config.Config, error) {
	configLoader.SetDefault("namespace", "default")
	configLoader.SetDefault("output", "table")
	configLoader.SetDefault("log-level", "info")
	configLoader.SetDefault("log-format", "text")
	configLoader.SetEnvPrefix("HELM_RELEASE_SIZE_ANALYSER")
	configLoader.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	configLoader.AutomaticEnv()

	err := configLoader.BindPFlags(cmd.Flags())
	if err != nil {
		return config.Config{}, fmt.Errorf("bind command flags: %w", err)
	}

	err = configLoader.BindPFlags(cmd.Root().PersistentFlags())
	if err != nil {
		return config.Config{}, fmt.Errorf("bind global flags: %w", err)
	}

	if configFile != "" {
		configLoader.SetConfigFile(configFile)

		err = configLoader.ReadInConfig()
		if err != nil {
			return config.Config{}, fmt.Errorf("read config file: %w", err)
		}
	}

	var cfg config.Config

	err = configLoader.Unmarshal(&cfg)
	if err != nil {
		return config.Config{}, fmt.Errorf("decode configuration: %w", err)
	}

	err = cfg.Validate()
	if err != nil {
		return config.Config{}, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

func newLogger(out io.Writer, level, format string) *slog.Logger {
	var slogLevel slog.Level

	switch level {
	case "debug":
		slogLevel = slog.LevelDebug
	case "warn":
		slogLevel = slog.LevelWarn
	case "error":
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}

	options := &slog.HandlerOptions{Level: slogLevel}
	if format == "json" {
		return slog.New(slog.NewJSONHandler(out, options))
	}

	return slog.New(slog.NewTextHandler(out, options))
}
