package insights

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type missingProbesAnalyzer struct{}

func NewMissingProbesAnalyzer() Analyzer { return &missingProbesAnalyzer{} }

func (a *missingProbesAnalyzer) Name() string { return "missing_probes" }

func (a *missingProbesAnalyzer) Analyze(ctx context.Context, clientset kubernetes.Interface, namespace string) ([]ClusterInsight, error) {
	var insights []ClusterInsight

	// Check Deployments
	deploys, err := clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, d := range deploys.Items {
		var missing []string
		for _, c := range d.Spec.Template.Spec.Containers {
			if c.ReadinessProbe == nil || c.LivenessProbe == nil {
				missing = append(missing, c.Name)
			}
		}
		if len(missing) > 0 {
			insights = append(insights, ClusterInsight{
				Analyzer:    "missing_probes",
				Category:    "reliability",
				Severity:    "warning",
				Title:       fmt.Sprintf("Deployment %q has containers without probes", d.Name),
				Description: fmt.Sprintf("Container(s) %s lack readiness/liveness probes. Without probes, Kubernetes cannot detect container health issues.", strings.Join(missing, ", ")),
				TargetKind:  "Deployment",
				TargetNS:    namespace,
				TargetName:  d.Name,
				Fingerprint: MakeFingerprint("missing_probes", "Deployment", namespace, d.Name),
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
			if c.ReadinessProbe == nil || c.LivenessProbe == nil {
				missing = append(missing, c.Name)
			}
		}
		if len(missing) > 0 {
			insights = append(insights, ClusterInsight{
				Analyzer:    "missing_probes",
				Category:    "reliability",
				Severity:    "warning",
				Title:       fmt.Sprintf("StatefulSet %q has containers without probes", s.Name),
				Description: fmt.Sprintf("Container(s) %s lack readiness/liveness probes. Without probes, Kubernetes cannot detect container health issues.", strings.Join(missing, ", ")),
				TargetKind:  "StatefulSet",
				TargetNS:    namespace,
				TargetName:  s.Name,
				Fingerprint: MakeFingerprint("missing_probes", "StatefulSet", namespace, s.Name),
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
			if c.ReadinessProbe == nil || c.LivenessProbe == nil {
				missing = append(missing, c.Name)
			}
		}
		if len(missing) > 0 {
			insights = append(insights, ClusterInsight{
				Analyzer:    "missing_probes",
				Category:    "reliability",
				Severity:    "warning",
				Title:       fmt.Sprintf("DaemonSet %q has containers without probes", d.Name),
				Description: fmt.Sprintf("Container(s) %s lack readiness/liveness probes. Without probes, Kubernetes cannot detect container health issues.", strings.Join(missing, ", ")),
				TargetKind:  "DaemonSet",
				TargetNS:    namespace,
				TargetName:  d.Name,
				Fingerprint: MakeFingerprint("missing_probes", "DaemonSet", namespace, d.Name),
			})
		}
	}

	return insights, nil
}
