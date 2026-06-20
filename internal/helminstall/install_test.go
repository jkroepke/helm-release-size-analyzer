package helminstall_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/jkroepke/helm-release-size-analyzer/internal/config"
	"github.com/jkroepke/helm-release-size-analyzer/internal/helminstall"
	"github.com/jkroepke/helm-release-size-analyzer/internal/releasesecret"
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

func TestInstallOptionallyIncludesCRDs(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name        string
		includeCRDs bool
		wantCRD     bool
	}{
		{name: "excluded by default"},
		{name: "included", includeCRDs: true, wantCRD: true},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			result, err := helminstall.Install(context.Background(), "testdata/basic", config.Config{
				ReleaseName: "crd-test",
				Namespace:   "testing",
				IncludeCRDs: testCase.includeCRDs,
			}, slog.New(slog.DiscardHandler))
			require.NoError(t, err)

			releaseJSON, err := releasesecret.DecodeJSON(result.Secret)
			require.NoError(t, err)

			var release struct {
				Manifest string `json:"manifest"`
			}

			require.NoError(t, json.Unmarshal(releaseJSON, &release))
			assert.Equal(t, testCase.wantCRD, strings.Contains(release.Manifest, "kind: CustomResourceDefinition"))
		})
	}
}

func TestInstallDoesNotUseKubeconfigOrContactCluster(t *testing.T) {
	kubeconfig := []byte(`apiVersion: v1
kind: Config
current-context: test
clusters:
- name: test
  cluster:
    server: http://127.0.0.1:1
contexts:
- name: test
  context:
    cluster: test
    user: test
users:
- name: test
  user: {}
`)
	kubeconfigPath := filepath.Join(t.TempDir(), "config")
	require.NoError(t, os.WriteFile(kubeconfigPath, kubeconfig, 0o600))
	t.Setenv("KUBECONFIG", kubeconfigPath)

	_, err := helminstall.Install(context.Background(), "testdata/basic", config.Config{
		ReleaseName: "isolated",
		Namespace:   "testing",
	}, slog.New(slog.DiscardHandler))
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(kubeconfigPath, []byte("not a kubeconfig"), 0o600))

	_, err = helminstall.Install(context.Background(), "testdata/basic", config.Config{
		ReleaseName: "invalid-kubeconfig",
		Namespace:   "testing",
	}, slog.New(slog.DiscardHandler))
	require.NoError(t, err, "installation loaded KUBECONFIG")
}

func TestConcurrentInstallsAreIsolated(t *testing.T) {
	t.Parallel()

	const workers = 8

	errs := make(chan error, workers)

	var waitGroup sync.WaitGroup

	for worker := range workers {
		waitGroup.Go(func() {
			message := fmt.Sprintf("worker-%d", worker)

			result, err := helminstall.Install(context.Background(), "testdata/basic", config.Config{
				ReleaseName: "concurrent",
				Namespace:   "testing",
				SetValues:   []string{"message=" + message},
			}, slog.New(slog.DiscardHandler))
			if err != nil {
				errs <- err

				return
			}

			releaseJSON, err := releasesecret.DecodeJSON(result.Secret)
			if err != nil {
				errs <- err

				return
			}

			var release struct {
				Config map[string]any `json:"config"`
			}

			if err = json.Unmarshal(releaseJSON, &release); err != nil {
				errs <- err

				return
			}

			if release.Config["message"] != message {
				errs <- fmt.Errorf("message = %v, want %q", release.Config["message"], message)
			}
		})
	}

	waitGroup.Wait()
	close(errs)

	for err := range errs {
		assert.NoError(t, err)
	}
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
