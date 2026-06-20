package kubemock

import (
	"fmt"
	"io"

	"helm.sh/helm/v4/pkg/kube"
	kubefake "helm.sh/helm/v4/pkg/kube/fake"
	"k8s.io/cli-runtime/pkg/resource"
)

// Recorder is a network-free Helm Kubernetes client. Helm still renders and
// stores the complete release, while resource application is recorded only.
type Recorder struct {
	*kubefake.PrintingKubeClient
}

// NewRecorder returns a network-free Kubernetes client that discards rendered resources.
func NewRecorder() *Recorder {
	return &Recorder{PrintingKubeClient: &kubefake.PrintingKubeClient{
		Out:       io.Discard,
		LogOutput: io.Discard,
	}}
}

// Build consumes rendered resources without contacting a Kubernetes API.
func (r *Recorder) Build(reader io.Reader, _ bool) (kube.ResourceList, error) {
	_, err := io.Copy(io.Discard, reader)
	if err != nil {
		return nil, fmt.Errorf("read rendered resources: %w", err)
	}

	// An empty list avoids Helm's live-cluster ownership checks. The complete
	// rendered manifest remains in the release and is what Helm stores.
	return make([]*resource.Info, 0), nil
}

// BuildTable delegates table-oriented resource building to Build.
func (r *Recorder) BuildTable(reader io.Reader, validate bool) (kube.ResourceList, error) {
	return r.Build(reader, validate)
}
