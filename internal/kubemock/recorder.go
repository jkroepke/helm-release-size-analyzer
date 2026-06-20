package kubemock

import (
	"bytes"
	"fmt"
	"io"
	"sync"

	"helm.sh/helm/v4/pkg/kube"
	kubefake "helm.sh/helm/v4/pkg/kube/fake"
	"k8s.io/cli-runtime/pkg/resource"
)

// Recorder is a network-free Helm Kubernetes client. Helm still renders and
// stores the complete release, while resource application is recorded only.
type Recorder struct {
	*kubefake.PrintingKubeClient

	mu        sync.Mutex
	manifests [][]byte
}

func NewRecorder() *Recorder {
	return &Recorder{PrintingKubeClient: &kubefake.PrintingKubeClient{
		Out:       io.Discard,
		LogOutput: io.Discard,
	}}
}

func (r *Recorder) Build(reader io.Reader, _ bool) (kube.ResourceList, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read rendered resources: %w", err)
	}
	r.mu.Lock()
	r.manifests = append(r.manifests, bytes.Clone(data))
	r.mu.Unlock()

	// An empty list avoids Helm's live-cluster ownership checks. The complete
	// rendered manifest remains in the release and is what Helm stores.
	return []*resource.Info{}, nil
}

func (r *Recorder) BuildTable(reader io.Reader, validate bool) (kube.ResourceList, error) {
	return r.Build(reader, validate)
}

func (r *Recorder) RecordedBytes() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	total := 0
	for _, manifest := range r.manifests {
		total += len(manifest)
	}
	return total
}
