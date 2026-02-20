package insights

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type evictedPodAnalyzer struct{}

func NewEvictedPodAnalyzer() Analyzer { return &evictedPodAnalyzer{} }

func (a *evictedPodAnalyzer) Name() string { return "evicted_pods" }

func (a *evictedPodAnalyzer) Analyze(ctx context.Context, clientset kubernetes.Interface, namespace string) ([]ClusterInsight, error) {
	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var insights []ClusterInsight
	for _, pod := range pods.Items {
		if pod.Status.Reason != "Evicted" {
			continue
		}

		insights = append(insights, ClusterInsight{
			Analyzer:       "evicted_pods",
			Category:       "hygiene",
			Severity:       "suggestion",
			Title:          fmt.Sprintf("Evicted pod %q can be cleaned up", pod.Name),
			Description:    "Pod was evicted by the kubelet (usually due to node resource pressure). It is defunct and can be safely deleted.",
			TargetKind:     "Pod",
			TargetNS:       namespace,
			TargetName:     pod.Name,
			Fingerprint:    MakeFingerprint("evicted_pods", "Pod", namespace, pod.Name),
			ProposedAction: "delete_pod",
			ProposedParams: map[string]any{"pod_status": "Evicted"},
			AutoRemediable: true,
		})
	}
	return insights, nil
}
