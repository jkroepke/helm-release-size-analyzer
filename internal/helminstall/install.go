package helminstall

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"helm-release-size-analyser/internal/config"
	"helm-release-size-analyser/internal/kubemock"

	"helm.sh/helm/v4/pkg/action"
	"helm.sh/helm/v4/pkg/chart"
	"helm.sh/helm/v4/pkg/chart/common"
	"helm.sh/helm/v4/pkg/chart/loader"
	"helm.sh/helm/v4/pkg/cli/values"
	"helm.sh/helm/v4/pkg/getter"
	"helm.sh/helm/v4/pkg/kube"
	"helm.sh/helm/v4/pkg/release"
	"helm.sh/helm/v4/pkg/storage"
	"helm.sh/helm/v4/pkg/storage/driver"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

type Result struct {
	Release       release.Accessor
	Secret        *corev1.Secret
	RecordedBytes int
}

func Install(ctx context.Context, chartPath string, cfg config.Config, logger *slog.Logger) (Result, error) {
	chrt, err := loader.Load(chartPath)
	if err != nil {
		return Result{}, fmt.Errorf("load chart %q: %w", chartPath, err)
	}
	chartAccessor, err := chart.NewAccessor(chrt)
	if err != nil {
		return Result{}, fmt.Errorf("inspect chart %q: %w", chartPath, err)
	}

	valueOptions := values.Options{
		ValueFiles:   cfg.ValueFiles,
		Values:       cfg.SetValues,
		StringValues: cfg.SetStrings,
		FileValues:   cfg.SetFiles,
	}
	mergedValues, err := valueOptions.MergeValues(getter.Providers{})
	if err != nil {
		return Result{}, fmt.Errorf("load values: %w", err)
	}

	if cfg.ReleaseName == "" {
		cfg.ReleaseName = chartAccessor.Name()
	}

	kubeClient := kubemock.NewRecorder()
	clientset := fake.NewClientset()
	secretDriver := driver.NewSecrets(clientset.CoreV1().Secrets(cfg.Namespace))
	secretDriver.SetLogger(logger.Handler())

	actionConfig := action.NewConfiguration(action.ConfigurationSetLogger(logger.Handler()))
	actionConfig.KubeClient = kubeClient
	actionConfig.Releases = storage.Init(secretDriver)
	actionConfig.Capabilities = common.DefaultCapabilities.Copy()
	actionConfig.HookOutputFunc = func(_, _, _ string) io.Writer { return io.Discard }

	install := action.NewInstall(actionConfig)
	install.ReleaseName = cfg.ReleaseName
	install.Namespace = cfg.Namespace
	install.DisableHooks = true
	install.DisableOpenAPIValidation = true
	install.IncludeCRDs = cfg.IncludeCRDs
	install.WaitStrategy = kube.HookOnlyStrategy

	installed, err := install.RunWithContext(ctx, chrt, mergedValues)
	if err != nil {
		return Result{}, fmt.Errorf("install release: %w", err)
	}
	accessor, err := release.NewAccessor(installed)
	if err != nil {
		return Result{}, fmt.Errorf("inspect installed release: %w", err)
	}

	secrets, err := clientset.CoreV1().Secrets(cfg.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("owner=helm,name=%s,version=%d", accessor.Name(), accessor.Version()),
	})
	if err != nil {
		return Result{}, fmt.Errorf("list release secrets: %w", err)
	}
	if len(secrets.Items) != 1 {
		return Result{}, fmt.Errorf("expected one release secret, found %d", len(secrets.Items))
	}

	return Result{
		Release:       accessor,
		Secret:        secrets.Items[0].DeepCopy(),
		RecordedBytes: kubeClient.RecordedBytes(),
	}, nil
}
