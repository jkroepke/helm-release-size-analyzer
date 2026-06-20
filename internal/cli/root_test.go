package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/jkroepke/helm-release-size-analyzer/internal/analyze"
	"github.com/jkroepke/helm-release-size-analyzer/internal/cli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type shortWriter struct{}

func (shortWriter) Write(data []byte) (int, error) {
	return len(data) - 1, nil
}

func TestAnalyzeCommandJSON(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer

	cmd := cli.NewRootCommand(&stdout, &stderr)
	chartPath := filepath.Join("..", "helminstall", "testdata", "basic")
	cmd.SetArgs([]string{
		"analyze", chartPath,
		"--release-name", "cli-test",
		"--set", "message=from-cli",
		"--output", "json",
	})

	err := cmd.ExecuteContext(context.Background())
	require.NoError(t, err, stderr.String())

	var got analyze.Report

	err = json.Unmarshal(stdout.Bytes(), &got)
	require.NoError(t, err, stdout.String())
	assert.NotZero(t, got.TotalBytes)
	require.NotEmpty(t, got.Properties)
	assert.Equal(t, "name", got.Properties[0].Name)

	var chartValuesBytes int

	for _, property := range got.Properties {
		if property.Name == "chart.values" {
			chartValuesBytes = property.Bytes

			break
		}
	}

	assert.Positive(t, chartValuesBytes)
}

func TestAnalyzeCommandRejectsYAML(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer

	cmd := cli.NewRootCommand(&stdout, &stderr)
	chartPath := filepath.Join("..", "helminstall", "testdata", "basic")
	cmd.SetArgs([]string{"analyze", chartPath, "--output", "yaml"})

	err := cmd.ExecuteContext(context.Background())
	require.EqualError(t, err, "invalid configuration: output must be one of table, json, or web")
}

func TestReleaseJSONCommand(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer

	cmd := cli.NewRootCommand(&stdout, &stderr)
	chartPath := filepath.Join("..", "helminstall", "testdata", "basic")
	cmd.SetArgs([]string{
		"release-json", chartPath,
		"--release-name", "raw-release",
		"--set", "message=from-release-json",
	})

	err := cmd.ExecuteContext(context.Background())
	require.NoError(t, err, stderr.String())

	var got struct {
		Name   string         `json:"name"`
		Config map[string]any `json:"config"`
		Info   struct {
			Status string `json:"status"`
		} `json:"info"`
	}

	err = json.Unmarshal(stdout.Bytes(), &got)
	require.NoError(t, err, stdout.String())
	assert.Equal(t, "raw-release", got.Name)
	assert.Equal(t, "from-release-json", got.Config["message"])
	assert.Equal(t, "deployed", got.Info.Status)
}

func TestReleaseJSONCommandRejectsShortWrite(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer

	cmd := cli.NewRootCommand(shortWriter{}, &stderr)
	chartPath := filepath.Join("..", "helminstall", "testdata", "basic")
	cmd.SetArgs([]string{"release-json", chartPath, "--release-name", "short-write"})

	err := cmd.ExecuteContext(context.Background())
	require.EqualError(t, err, "write release JSON: short write")
}

func TestVersionCommand(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer

	cmd := cli.NewRootCommand(&stdout, &stderr)
	cmd.SetArgs([]string{"--version"})

	err := cmd.ExecuteContext(context.Background())
	require.NoError(t, err, stderr.String())

	want := "helm-release-size-analyzer version dev (revision: unknown, branch: unknown, built: unknown)\n"
	assert.Equal(t, want, stdout.String())
}

func TestConfigurationSourcesAreFlagsOnly(t *testing.T) {
	t.Setenv("HELM_RELEASE_SIZE_analyzer_NAMESPACE", "")

	var stdout, stderr bytes.Buffer

	cmd := cli.NewRootCommand(&stdout, &stderr)
	assert.Nil(t, cmd.PersistentFlags().Lookup("config"))

	chartPath := filepath.Join("..", "helminstall", "testdata", "basic")
	cmd.SetArgs([]string{"release-json", chartPath, "--release-name", "flags-only"})

	err := cmd.ExecuteContext(context.Background())
	require.NoError(t, err, stderr.String())

	var got struct {
		Namespace string `json:"namespace"`
	}

	err = json.Unmarshal(stdout.Bytes(), &got)
	require.NoError(t, err, stdout.String())
	assert.Equal(t, "default", got.Namespace)
}
