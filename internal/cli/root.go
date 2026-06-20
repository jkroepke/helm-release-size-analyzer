package cli

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"

	"github.com/jkroepke/helm-release-size-analyzer/internal/analyze"
	"github.com/jkroepke/helm-release-size-analyzer/internal/config"
	"github.com/jkroepke/helm-release-size-analyzer/internal/helminstall"
	"github.com/jkroepke/helm-release-size-analyzer/internal/releasesecret"
	"github.com/jkroepke/helm-release-size-analyzer/internal/report"
	"github.com/jkroepke/helm-release-size-analyzer/internal/version"
	"github.com/spf13/cobra"
)

// ExitCode maps a command error to the process exit status.
func ExitCode(_ error) int {
	return 1
}

// NewRootCommand constructs the CLI command tree with the provided output streams.
func NewRootCommand(stdout, stderr io.Writer) *cobra.Command {
	var logLevel, logFormat string

	root := &cobra.Command{
		Use:           "helm-release-size-analyzer",
		Short:         "Analyse the JSON stored in a Helm release Secret",
		Version:       version.String(),
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.PersistentFlags().StringVar(&logLevel, "log-level", "info", "log level: debug, info, warn, error")
	root.PersistentFlags().StringVar(&logFormat, "log-format", "text", "log format: text or json")

	analyseConfig := config.Config{Namespace: "default", Output: "table"}

	analyseCmd := &cobra.Command{
		Use:   "analyse CHART",
		Short: "Install a chart in memory and analyse its Helm release Secret",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := validatedConfig(analyseConfig, logLevel, logFormat)
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

			// DecodeJSON validates the payload before returning it.
			result, err := analyze.BuildValidated(releaseJSON)
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
	addInstallFlags(analyseCmd, &analyseConfig)
	flags := analyseCmd.Flags()
	flags.StringVarP(&analyseConfig.Output, "output", "o", analyseConfig.Output, "output format: table or json")

	releaseJSONConfig := config.Config{Namespace: "default", Output: "table"}

	releaseJSONCmd := &cobra.Command{
		Use:   "release-json CHART",
		Short: "Install a chart in memory and print its uncompressed Helm release JSON",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := validatedConfig(releaseJSONConfig, logLevel, logFormat)
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

			err = writeReleaseJSON(stdout, releaseJSON)
			if err != nil {
				return err
			}

			return nil
		},
	}
	addInstallFlags(releaseJSONCmd, &releaseJSONConfig)

	root.AddCommand(analyseCmd, releaseJSONCmd)

	return root
}

// writeReleaseJSON writes the complete release payload or reports a short write.
func writeReleaseJSON(out io.Writer, releaseJSON []byte) error {
	_, err := io.Copy(out, bytes.NewReader(releaseJSON))
	if err != nil {
		return fmt.Errorf("write release JSON: %w", err)
	}

	return nil
}

// addInstallFlags binds Helm installation flags directly to cfg.
func addInstallFlags(cmd *cobra.Command, cfg *config.Config) {
	flags := cmd.Flags()
	flags.StringVar(&cfg.ReleaseName, "release-name", "", "release name (defaults to chart name)")
	flags.StringVar(&cfg.Namespace, "namespace", cfg.Namespace, "simulated release namespace")
	flags.StringSliceVarP(&cfg.ValueFiles, "values", "f", nil, "values file (repeatable)")
	flags.StringArrayVar(&cfg.SetValues, "set", nil, "set a value")
	flags.StringArrayVar(&cfg.SetStrings, "set-string", nil, "set a string value")
	flags.StringArrayVar(&cfg.SetFiles, "set-file", nil, "set a value from a file")
	flags.BoolVar(&cfg.IncludeCRDs, "include-crds", false, "include CRDs in the stored manifest")
}

// validatedConfig applies global logging flags and validates the resulting configuration.
func validatedConfig(cfg config.Config, logLevel, logFormat string) (config.Config, error) {
	cfg.LogLevel = logLevel
	cfg.LogFormat = logFormat

	err := cfg.Validate()
	if err != nil {
		return config.Config{}, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// newLogger creates a structured logger for the selected level and output format.
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
