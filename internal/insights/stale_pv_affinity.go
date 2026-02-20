package insights

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type stalePvAffinityAnalyzer struct{}

func NewStalePvAffinityAnalyzer() Analyzer { return &stalePvAffinityAnalyzer{} }

func (a *stalePvAffinityAnalyzer) Name() string { return "stale_pv_affinity" }

func (a *stalePvAffinityAnalyzer) Analyze(ctx context.Context, clientset kubernetes.Interface, namespace string) ([]ClusterInsight, error) {
	// Get cluster node names
	nodeList, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	nodeNames := make(map[string]bool, len(nodeList.Items))
	for _, n := range nodeList.Items {
		nodeNames[n.Name] = true
	}

	// Get PVCs in this namespace
	pvcList, err := clientset.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	// Get all PVs
	pvList, err := clientset.CoreV1().PersistentVolumes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	pvMap := make(map[string]corev1.PersistentVolume, len(pvList.Items))
	for _, pv := range pvList.Items {
		pvMap[pv.Name] = pv
	}

	var insights []ClusterInsight
	for _, pvc := range pvcList.Items {
		if pvc.Spec.VolumeName == "" {
			continue
		}
		pv, ok := pvMap[pvc.Spec.VolumeName]
		if !ok {
			continue
		}

		affinityNodes := extractPVNodeNames(pv)
		if len(affinityNodes) == 0 {
			continue
		}

		allMissing := true
		for _, n := range affinityNodes {
			if nodeNames[n] {
				allMissing = false
				break
			}
		}
		if !allMissing {
			continue
		}

		insights = append(insights, ClusterInsight{
			Analyzer:       "stale_pv_affinity",
			Category:       "reliability",
			Severity:       "action",
			Title:          fmt.Sprintf("PVC %q bound to PV with stale node affinity", pvc.Name),
			Description:    fmt.Sprintf("PV %q has nodeAffinity to [%s] but none of these nodes exist in the cluster. Pods using this PVC cannot schedule. Delete the PVC to allow reprovisioning.", pv.Name, strings.Join(affinityNodes, ", ")),
			TargetKind:     "PersistentVolumeClaim",
			TargetNS:       namespace,
			TargetName:     pvc.Name,
			Fingerprint:    MakeFingerprint("stale_pv_affinity", "PersistentVolumeClaim", namespace, pvc.Name),
			ProposedAction: "delete_pvc",
			ProposedParams: map[string]any{"pv_name": pv.Name, "stale_nodes": affinityNodes},
			AutoRemediable: true,
		})
	}
	return insights, nil
}

// extractPVNodeNames returns node names from a PV's nodeAffinity.
func extractPVNodeNames(pv corev1.PersistentVolume) []string {
	if pv.Spec.NodeAffinity == nil || pv.Spec.NodeAffinity.Required == nil {
		return nil
	}
	var names []string
	for _, term := range pv.Spec.NodeAffinity.Required.NodeSelectorTerms {
		for _, expr := range term.MatchExpressions {
			if expr.Key == "kubernetes.io/hostname" && expr.Operator == corev1.NodeSelectorOpIn {
				names = append(names, expr.Values...)
			}
		}
	}
	return names
}
