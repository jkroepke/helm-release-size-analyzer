package helminstall

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"helm-release-size-analyser/internal/config"
)

func TestInstallCapturesHelmSecret(t *testing.T) {
	t.Parallel()

	result, err := Install(context.Background(), "testdata/basic", config.Config{
		ReleaseName: "example",
		Namespace:   "testing",
		SetValues:   []string{"message=overridden"},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("Install() error = %v", err)
	}

	if got, want := result.Secret.Name, "sh.helm.release.v1.example.v1"; got != want {
		t.Fatalf("secret name = %q, want %q", got, want)
	}
	if got := len(result.Secret.Data["release"]); got == 0 {
		t.Fatal("release Secret payload is empty")
	}
	if got, want := result.Secret.Labels["status"], "deployed"; got != want {
		t.Fatalf("secret status = %q, want %q", got, want)
	}
	if got := result.Release.Manifest(); got == "" {
		t.Fatal("stored release manifest is empty")
	}
	if got := result.RecordedBytes; got == 0 {
		t.Fatal("mock Kubernetes client did not record rendered resources")
	}
}
