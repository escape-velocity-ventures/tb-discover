package insights

import (
	"context"
	"fmt"
	"math"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type stuckTerminatingAnalyzer struct{}

func NewStuckTerminatingAnalyzer() Analyzer { return &stuckTerminatingAnalyzer{} }

func (a *stuckTerminatingAnalyzer) Name() string { return "stuck_terminating" }

func (a *stuckTerminatingAnalyzer) Analyze(ctx context.Context, clientset kubernetes.Interface, namespace string) ([]ClusterInsight, error) {
	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	tenMinAgo := time.Now().Add(-10 * time.Minute)
	var insights []ClusterInsight

	for _, pod := range pods.Items {
		if pod.DeletionTimestamp == nil || pod.DeletionTimestamp.Time.After(tenMinAgo) {
			continue
		}
		stuckMinutes := int(math.Round(time.Since(pod.DeletionTimestamp.Time).Minutes()))

		insights = append(insights, ClusterInsight{
			Analyzer:       "stuck_terminating",
			Category:       "reliability",
			Severity:       "action",
			Title:          fmt.Sprintf("Pod %q stuck terminating for %dmin", pod.Name, stuckMinutes),
			Description:    fmt.Sprintf("Pod has had deletionTimestamp set since %s but has not terminated. This usually indicates a stuck finalizer or unresponsive kubelet.", pod.DeletionTimestamp.Format(time.RFC3339)),
			TargetKind:     "Pod",
			TargetNS:       namespace,
			TargetName:     pod.Name,
			Fingerprint:    MakeFingerprint("stuck_terminating", "Pod", namespace, pod.Name),
			ProposedAction: "force_delete_pod",
			ProposedParams: map[string]any{"reason": "stuck_terminating", "stuck_since": pod.DeletionTimestamp.Format(time.RFC3339)},
			AutoRemediable: true,
		})
	}
	return insights, nil
}
