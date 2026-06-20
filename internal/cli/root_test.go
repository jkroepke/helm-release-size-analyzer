package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"helm-release-size-analyser/internal/analyse"
)

func TestAnalyseCommandJSON(t *testing.T) {
	t.Parallel()

	var stdout, stderr bytes.Buffer
	cmd := NewRootCommand(&stdout, &stderr)
	chartPath := filepath.Join("..", "helminstall", "testdata", "basic")
	cmd.SetArgs([]string{
		"analyse", chartPath,
		"--release-name", "cli-test",
		"--set", "message=from-cli",
		"--output", "json",
	})

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext() error = %v; stderr = %s", err, stderr.String())
	}

	var got analyse.Report
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("decode report: %v\noutput: %s", err, stdout.String())
	}
	if got.ReleaseName != "cli-test" {
		t.Fatalf("release name = %q, want cli-test", got.ReleaseName)
	}
	if got.SecretName != "sh.helm.release.v1.cli-test.v1" {
		t.Fatalf("secret name = %q", got.SecretName)
	}
	if got.Metrics.HelmStoragePayloadBytes == 0 {
		t.Fatal("payload bytes are zero")
	}
}
