package insights

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type resourcePressureAnalyzer struct{}

func NewResourcePressureAnalyzer() Analyzer { return &resourcePressureAnalyzer{} }

func (a *resourcePressureAnalyzer) Name() string { return "resource_pressure" }

func (a *resourcePressureAnalyzer) Analyze(ctx context.Context, clientset kubernetes.Interface, _ string) ([]ClusterInsight, error) {
	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var insights []ClusterInsight
	for _, node := range nodes.Items {
		for _, cond := range node.Status.Conditions {
			if cond.Status != corev1.ConditionTrue {
				continue
			}
			switch cond.Type {
			case corev1.NodeMemoryPressure:
				insights = append(insights, ClusterInsight{
					Analyzer:    "resource_pressure",
					Category:    "performance",
					Severity:    "action",
					Title:       fmt.Sprintf("Node %q under memory pressure", node.Name),
					Description: fmt.Sprintf("Node has MemoryPressure condition. Message: %s", cond.Message),
					TargetKind:  "Node",
					TargetNS:    "",
					TargetName:  node.Name,
					Fingerprint: MakeFingerprint("resource_pressure_mem", "Node", "", node.Name),
				})
			case corev1.NodeDiskPressure:
				insights = append(insights, ClusterInsight{
					Analyzer:    "resource_pressure",
					Category:    "performance",
					Severity:    "action",
					Title:       fmt.Sprintf("Node %q under disk pressure", node.Name),
					Description: fmt.Sprintf("Node has DiskPressure condition. Message: %s", cond.Message),
					TargetKind:  "Node",
					TargetNS:    "",
					TargetName:  node.Name,
					Fingerprint: MakeFingerprint("resource_pressure_disk", "Node", "", node.Name),
				})
			case corev1.NodePIDPressure:
				insights = append(insights, ClusterInsight{
					Analyzer:    "resource_pressure",
					Category:    "performance",
					Severity:    "warning",
					Title:       fmt.Sprintf("Node %q under PID pressure", node.Name),
					Description: fmt.Sprintf("Node has PIDPressure condition. Message: %s", cond.Message),
					TargetKind:  "Node",
					TargetNS:    "",
					TargetName:  node.Name,
					Fingerprint: MakeFingerprint("resource_pressure_pid", "Node", "", node.Name),
				})
			}
		}
	}
	return insights, nil
}
