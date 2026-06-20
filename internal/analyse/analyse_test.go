package analyse

import (
	"testing"

	"helm-release-size-analyser/internal/helminstall"
	"helm.sh/helm/v4/pkg/release"
	releasev1 "helm.sh/helm/v4/pkg/release/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildStatuses(t *testing.T) {
	t.Parallel()

	rel := &releasev1.Release{Name: "example", Namespace: "default", Version: 1, Manifest: "abc"}
	accessor, err := release.NewAccessor(rel)
	if err != nil {
		t.Fatal(err)
	}
	result := helminstall.Result{
		Release: accessor,
		Secret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "release-secret"},
			Data:       map[string][]byte{"release": make([]byte, 80)},
		},
	}

	report, err := Build(result, 100)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if got, want := report.Status, StatusWarning; got != want {
		t.Fatalf("status = %q, want %q", got, want)
	}
	if got, want := report.Metrics.HelmStoragePayloadBytes, 80; got != want {
		t.Fatalf("payload bytes = %d, want %d", got, want)
	}
	if got, want := report.Metrics.ManifestBytes, 3; got != want {
		t.Fatalf("manifest bytes = %d, want %d", got, want)
	}
}
