package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/jkroepke/helm-release-size-analyser/internal/analyse"
	"github.com/jkroepke/helm-release-size-analyser/internal/cli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnalyseCommandJSON(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer

	cmd := cli.NewRootCommand(&stdout, &stderr)
	chartPath := filepath.Join("..", "helminstall", "testdata", "basic")
	cmd.SetArgs([]string{
		"analyse", chartPath,
		"--release-name", "cli-test",
		"--set", "message=from-cli",
		"--output", "json",
	})

	err := cmd.ExecuteContext(context.Background())
	require.NoError(t, err, stderr.String())

	var got analyse.Report

	err = json.Unmarshal(stdout.Bytes(), &got)
	require.NoError(t, err, stdout.String())
	assert.NotZero(t, got.TotalBytes)
	require.NotEmpty(t, got.Properties)
	assert.Equal(t, "name", got.Properties[0].Name)
}

func TestAnalyseCommandRejectsYAML(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer

	cmd := cli.NewRootCommand(&stdout, &stderr)
	chartPath := filepath.Join("..", "helminstall", "testdata", "basic")
	cmd.SetArgs([]string{"analyse", chartPath, "--output", "yaml"})

	err := cmd.ExecuteContext(context.Background())
	require.EqualError(t, err, "invalid configuration: output must be one of table or json")
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

func TestVersionCommand(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer

	cmd := cli.NewRootCommand(&stdout, &stderr)
	cmd.SetArgs([]string{"--version"})

	err := cmd.ExecuteContext(context.Background())
	require.NoError(t, err, stderr.String())

	want := "helm-release-size-analyser version dev (revision: unknown, branch: unknown, built: unknown)\n"
	assert.Equal(t, want, stdout.String())
}
