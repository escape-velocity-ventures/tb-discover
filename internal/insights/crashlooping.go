package insights

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type crashloopingAnalyzer struct{}

func NewCrashloopingAnalyzer() Analyzer { return &crashloopingAnalyzer{} }

func (a *crashloopingAnalyzer) Name() string { return "crashlooping" }

func (a *crashloopingAnalyzer) Analyze(ctx context.Context, clientset kubernetes.Interface, namespace string) ([]ClusterInsight, error) {
	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var insights []ClusterInsight
	for _, pod := range pods.Items {
		for _, cs := range pod.Status.ContainerStatuses {
			isCrashloop := cs.State.Waiting != nil && cs.State.Waiting.Reason == "CrashLoopBackOff"
			highRestarts := cs.RestartCount >= 5

			if !isCrashloop && !highRestarts {
				continue
			}

			// Try to resolve owning workload
			targetKind := "Pod"
			targetName := pod.Name
			for _, ref := range pod.OwnerReferences {
				if ref.Kind == "ReplicaSet" {
					// Look up the ReplicaSet to find the Deployment owner
					rs, rsErr := clientset.AppsV1().ReplicaSets(namespace).Get(ctx, ref.Name, metav1.GetOptions{})
					if rsErr == nil {
						for _, rsRef := range rs.OwnerReferences {
							if rsRef.Kind == "Deployment" {
								targetKind = "Deployment"
								targetName = rsRef.Name
							}
						}
					}
				} else if ref.Kind == "StatefulSet" || ref.Kind == "DaemonSet" {
					targetKind = ref.Kind
					targetName = ref.Name
				}
			}

			var title, desc string
			if isCrashloop {
				title = fmt.Sprintf("%s %q has crashlooping pods", targetKind, targetName)
				desc = "One or more pods are in CrashLoopBackOff. Check logs for the root cause."
			} else {
				title = fmt.Sprintf("%s %q pods have %d+ restarts", targetKind, targetName, cs.RestartCount)
				desc = fmt.Sprintf("Pods have restarted %d+ times, indicating instability.", cs.RestartCount)
			}

			severity := "action"
			if !isCrashloop && cs.RestartCount < 10 {
				severity = "warning"
			}

			insights = append(insights, ClusterInsight{
				Analyzer:    "crashlooping",
				Category:    "reliability",
				Severity:    severity,
				Title:       title,
				Description: desc,
				TargetKind:  targetKind,
				TargetNS:    namespace,
				TargetName:  targetName,
				Fingerprint: MakeFingerprint("crashlooping", targetKind, namespace, targetName),
			})
			break // one insight per pod is enough
		}
	}

	// Deduplicate by fingerprint (multiple pods may point to same deployment)
	seen := make(map[string]bool)
	var deduped []ClusterInsight
	for _, i := range insights {
		if seen[i.Fingerprint] {
			continue
		}
		seen[i.Fingerprint] = true
		deduped = append(deduped, i)
	}
	return deduped, nil
}

