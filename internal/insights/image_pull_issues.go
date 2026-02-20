package insights

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type imagePullIssuesAnalyzer struct{}

func NewImagePullIssuesAnalyzer() Analyzer { return &imagePullIssuesAnalyzer{} }

func (a *imagePullIssuesAnalyzer) Name() string { return "image_pull_issues" }

func (a *imagePullIssuesAnalyzer) Analyze(ctx context.Context, clientset kubernetes.Interface, namespace string) ([]ClusterInsight, error) {
	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var insights []ClusterInsight
	for _, pod := range pods.Items {
		var issues []string
		var reason string
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.State.Waiting != nil {
				r := cs.State.Waiting.Reason
				if r == "ImagePullBackOff" || r == "ErrImagePull" {
					issues = append(issues, fmt.Sprintf("%s (%s)", cs.Name, cs.Image))
					reason = r
				}
			}
		}
		if len(issues) == 0 {
			continue
		}

		insights = append(insights, ClusterInsight{
			Analyzer:    "image_pull_issues",
			Category:    "reliability",
			Severity:    "action",
			Title:       fmt.Sprintf("Pod %q cannot pull image", pod.Name),
			Description: fmt.Sprintf("Container(s) %s are stuck in %s. Check the image name, tag, and registry credentials.", strings.Join(issues, ", "), reason),
			TargetKind:  "Pod",
			TargetNS:    namespace,
			TargetName:  pod.Name,
			Fingerprint: MakeFingerprint("image_pull_issues", "Pod", namespace, pod.Name),
		})
	}
	return insights, nil
}
