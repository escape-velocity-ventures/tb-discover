package insights

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type missingLimitsAnalyzer struct{}

func NewMissingLimitsAnalyzer() Analyzer { return &missingLimitsAnalyzer{} }

func (a *missingLimitsAnalyzer) Name() string { return "missing_limits" }

func (a *missingLimitsAnalyzer) Analyze(ctx context.Context, clientset kubernetes.Interface, namespace string) ([]ClusterInsight, error) {
	var insights []ClusterInsight

	// Check Deployments
	deploys, err := clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, d := range deploys.Items {
		var missing []string
		for _, c := range d.Spec.Template.Spec.Containers {
			mem := c.Resources.Limits.Memory()
			if mem == nil || mem.IsZero() {
				missing = append(missing, c.Name)
			}
		}
		if len(missing) > 0 {
			insights = append(insights, ClusterInsight{
				Analyzer:       "missing_limits",
				Category:       "hygiene",
				Severity:       "suggestion",
				Title:          fmt.Sprintf("Deployment %q has no memory limits", d.Name),
				Description:    fmt.Sprintf("Container(s) %s have no memory limits. Without limits, a container can consume all available memory on the node.", strings.Join(missing, ", ")),
				TargetKind:     "Deployment",
				TargetNS:       namespace,
				TargetName:     d.Name,
				Fingerprint:    MakeFingerprint("missing_limits", "Deployment", namespace, d.Name),
				ProposedAction: "tune_resource_limits",
				ProposedParams: map[string]any{"cpu_limit": "250m", "memory_limit": "256Mi"},
			})
		}
	}

	// Check StatefulSets
	stss, err := clientset.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, s := range stss.Items {
		var missing []string
		for _, c := range s.Spec.Template.Spec.Containers {
			mem := c.Resources.Limits.Memory()
			if mem == nil || mem.IsZero() {
				missing = append(missing, c.Name)
			}
		}
		if len(missing) > 0 {
			insights = append(insights, ClusterInsight{
				Analyzer:       "missing_limits",
				Category:       "hygiene",
				Severity:       "suggestion",
				Title:          fmt.Sprintf("StatefulSet %q has no memory limits", s.Name),
				Description:    fmt.Sprintf("Container(s) %s have no memory limits. Without limits, a container can consume all available memory on the node.", strings.Join(missing, ", ")),
				TargetKind:     "StatefulSet",
				TargetNS:       namespace,
				TargetName:     s.Name,
				Fingerprint:    MakeFingerprint("missing_limits", "StatefulSet", namespace, s.Name),
				ProposedAction: "tune_resource_limits",
				ProposedParams: map[string]any{"cpu_limit": "250m", "memory_limit": "256Mi"},
			})
		}
	}

	// Check DaemonSets
	dss, err := clientset.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, d := range dss.Items {
		var missing []string
		for _, c := range d.Spec.Template.Spec.Containers {
			mem := c.Resources.Limits.Memory()
			if mem == nil || mem.IsZero() {
				missing = append(missing, c.Name)
			}
		}
		if len(missing) > 0 {
			insights = append(insights, ClusterInsight{
				Analyzer:       "missing_limits",
				Category:       "hygiene",
				Severity:       "suggestion",
				Title:          fmt.Sprintf("DaemonSet %q has no memory limits", d.Name),
				Description:    fmt.Sprintf("Container(s) %s have no memory limits. Without limits, a container can consume all available memory on the node.", strings.Join(missing, ", ")),
				TargetKind:     "DaemonSet",
				TargetNS:       namespace,
				TargetName:     d.Name,
				Fingerprint:    MakeFingerprint("missing_limits", "DaemonSet", namespace, d.Name),
				ProposedAction: "tune_resource_limits",
				ProposedParams: map[string]any{"cpu_limit": "250m", "memory_limit": "256Mi"},
			})
		}
	}

	return insights, nil
}
