package insights

import (
	"context"

	"k8s.io/client-go/kubernetes"
)

// Analyzer detects issues in a Kubernetes namespace.
type Analyzer interface {
	Name() string
	Analyze(ctx context.Context, clientset kubernetes.Interface, namespace string) ([]ClusterInsight, error)
}
