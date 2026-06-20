package helminstall_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/jkroepke/helm-release-size-analyser/internal/config"
	"github.com/jkroepke/helm-release-size-analyser/internal/helminstall"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstallCapturesHelmSecret(t *testing.T) {
	t.Parallel()

	result, err := helminstall.Install(context.Background(), "testdata/basic", config.Config{
		ReleaseName: "example",
		Namespace:   "testing",
		SetValues:   []string{"message=overridden"},
	}, slog.New(slog.DiscardHandler))
	require.NoError(t, err)
	assert.Equal(t, "sh.helm.release.v1.example.v1", result.Secret.Name)
	assert.NotEmpty(t, result.Secret.Data["release"])
	assert.Equal(t, "deployed", result.Secret.Labels["status"])
}

func BenchmarkInstall(b *testing.B) {
	logger := slog.New(slog.DiscardHandler)
	cfg := config.Config{
		ReleaseName: "benchmark",
		Namespace:   "benchmark",
		SetValues:   []string{"message=profiled"},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_, err := helminstall.Install(context.Background(), "testdata/basic", cfg, logger)
		require.NoError(b, err)
	}
}
