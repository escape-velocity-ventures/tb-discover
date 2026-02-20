package insights

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type stalePodAnalyzer struct{}

func NewStalePodAnalyzer() Analyzer { return &stalePodAnalyzer{} }

func (a *stalePodAnalyzer) Name() string { return "stale_pods" }

func (a *stalePodAnalyzer) Analyze(ctx context.Context, clientset kubernetes.Interface, namespace string) ([]ClusterInsight, error) {
	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	oneHourAgo := time.Now().Add(-1 * time.Hour)
	var insights []ClusterInsight

	for _, pod := range pods.Items {
		phase := string(pod.Status.Phase)
		if phase != "Succeeded" && phase != "Failed" {
			continue
		}
		if pod.Status.StartTime == nil || pod.Status.StartTime.Time.After(oneHourAgo) {
			continue
		}

		insights = append(insights, ClusterInsight{
			Analyzer:       "stale_pods",
			Category:       "hygiene",
			Severity:       "suggestion",
			Title:          fmt.Sprintf("%s pod %q can be cleaned up", phase, pod.Name),
			Description:    fmt.Sprintf("Pod has been in %s state since %s. It is no longer running and can be safely deleted.", phase, pod.Status.StartTime.Format(time.RFC3339)),
			TargetKind:     "Pod",
			TargetNS:       namespace,
			TargetName:     pod.Name,
			Fingerprint:    MakeFingerprint("stale_pods", "Pod", namespace, pod.Name),
			ProposedAction: "delete_pod",
			ProposedParams: map[string]any{"pod_status": phase},
			AutoRemediable: true,
		})
	}
	return insights, nil
}
