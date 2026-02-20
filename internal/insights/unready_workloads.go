package insights

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type unreadyWorkloadsAnalyzer struct{}

func NewUnreadyWorkloadsAnalyzer() Analyzer { return &unreadyWorkloadsAnalyzer{} }

func (a *unreadyWorkloadsAnalyzer) Name() string { return "unready_workloads" }

func (a *unreadyWorkloadsAnalyzer) Analyze(ctx context.Context, clientset kubernetes.Interface, namespace string) ([]ClusterInsight, error) {
	deploys, err := clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var insights []ClusterInsight
	for _, d := range deploys.Items {
		desired := int32(0)
		if d.Spec.Replicas != nil {
			desired = *d.Spec.Replicas
		}
		if desired == 0 {
			continue
		}
		ready := d.Status.ReadyReplicas
		if ready >= desired {
			continue
		}

		insights = append(insights, ClusterInsight{
			Analyzer:    "unready_workloads",
			Category:    "reliability",
			Severity:    "warning",
			Title:       fmt.Sprintf("Deployment %q has %d/%d ready", d.Name, ready, desired),
			Description: fmt.Sprintf("Only %d of %d desired replicas are ready. This may indicate resource pressure, failed scheduling, or container issues.", ready, desired),
			TargetKind:  "Deployment",
			TargetNS:    namespace,
			TargetName:  d.Name,
			Fingerprint: MakeFingerprint("unready_workloads", "Deployment", namespace, d.Name),
		})
	}

	// Also check StatefulSets
	stss, err := clientset.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, s := range stss.Items {
		desired := int32(0)
		if s.Spec.Replicas != nil {
			desired = *s.Spec.Replicas
		}
		if desired == 0 {
			continue
		}
		ready := s.Status.ReadyReplicas
		if ready >= desired {
			continue
		}

		insights = append(insights, ClusterInsight{
			Analyzer:    "unready_workloads",
			Category:    "reliability",
			Severity:    "warning",
			Title:       fmt.Sprintf("StatefulSet %q has %d/%d ready", s.Name, ready, desired),
			Description: fmt.Sprintf("Only %d of %d desired replicas are ready. This may indicate resource pressure, failed scheduling, or container issues.", ready, desired),
			TargetKind:  "StatefulSet",
			TargetNS:    namespace,
			TargetName:  s.Name,
			Fingerprint: MakeFingerprint("unready_workloads", "StatefulSet", namespace, s.Name),
		})
	}

	return insights, nil
}
