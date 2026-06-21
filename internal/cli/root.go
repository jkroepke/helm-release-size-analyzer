package cli

import (
	"bytes"
	"context"
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
func NewRootCommand(args []string, stdout, stderr io.Writer) *cobra.Command {
	var logLevel, logFormat string

	root := &cobra.Command{
		Use:           "helm-release-size-analyzer",
		Short:         "Analyze the JSON stored in a Helm release Secret",
		Version:       version.String(),
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	root.SetArgs(args)
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.PersistentFlags().StringVar(&logLevel, "log-level", "info", "log level: debug, info, warn, error")
	root.PersistentFlags().StringVar(&logFormat, "log-format", "text", "log format: text or json")

	analyzeConfig := config.Config{Namespace: "default", Output: "web"}

	analyzeCmd := &cobra.Command{
		Use:   "analyze CHART",
		Short: "Install a chart in memory and analyze its Helm release Secret",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := validatedConfig(analyzeConfig, logLevel, logFormat)
			if err != nil {
				return err
			}

			logger := newLogger(stderr, cfg.LogLevel, cfg.LogFormat)
			logger.Debug("starting analysis", slog.String("chart", args[0]))

			releaseJSON, compressedBytes, err := installReleaseJSON(cmd.Context(), args[0], cfg, logger)
			if err != nil {
				return err
			}

			return writeAnalysis(cmd.Context(), stdout, logger, cfg.Output, releaseJSON, compressedBytes)
		},
	}
	addInstallFlags(analyzeCmd, &analyzeConfig)
	flags := analyzeCmd.Flags()
	flags.StringVarP(&analyzeConfig.Output, "output", "o", analyzeConfig.Output, "output format: table, json, or web")

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

			releaseJSON, _, err := installReleaseJSON(cmd.Context(), args[0], cfg, logger)
			if err != nil {
				return err
			}

			err = writeReleaseJSON(stdout, releaseJSON)
			if err != nil {
				return err
			}

			return nil
		},
	}
	addInstallFlags(releaseJSONCmd, &releaseJSONConfig)

	root.AddCommand(analyzeCmd, releaseJSONCmd)

	return root
}

// installReleaseJSON installs a chart and returns its decoded release JSON and
// the stored Secret payload size.
func installReleaseJSON(
	ctx context.Context,
	chartPath string,
	cfg config.Config,
	logger *slog.Logger,
) ([]byte, int, error) {
	installed, err := helminstall.Install(ctx, chartPath, cfg, logger)
	if err != nil {
		return nil, 0, fmt.Errorf("install chart: %w", err)
	}

	releaseJSON, err := releasesecret.DecodeJSON(installed.Secret)
	if err != nil {
		return nil, 0, fmt.Errorf("decode release JSON: %w", err)
	}

	return releaseJSON, len(installed.Secret.Data["release"]), nil
}

func writeAnalysis(
	ctx context.Context,
	out io.Writer,
	logger *slog.Logger,
	format string,
	releaseJSON []byte,
	compressedBytes int,
) error {
	if format == "web" {
		return serveWebReport(ctx, logger, releaseJSON, compressedBytes)
	}

	// DecodeJSON validates the payload before returning it.
	result, err := analyze.BuildValidated(releaseJSON)
	if err != nil {
		return fmt.Errorf("analyze release: %w", err)
	}

	result.CompressedBytes = compressedBytes

	err = report.Write(out, format, result)
	if err != nil {
		return fmt.Errorf("write report: %w", err)
	}

	return nil
}

func serveWebReport(ctx context.Context, logger *slog.Logger, releaseJSON []byte, compressedBytes int) error {
	tree, err := analyze.BuildTreeValidated(releaseJSON)
	if err != nil {
		return fmt.Errorf("analyze release tree: %w", err)
	}

	tree.CompressedBytes = compressedBytes

	err = report.ServeWeb(ctx, tree, version.Version, func(url string) {
		logger.InfoContext(ctx, "web report ready", slog.String("url", url))

		go func() {
			browserErr := report.OpenBrowser(url)
			if browserErr != nil {
				logger.WarnContext(ctx, "could not open browser", slog.Any("error", browserErr))
			}
		}()
	})
	if err != nil {
		return fmt.Errorf("serve web report: %w", err)
	}

	return nil
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
