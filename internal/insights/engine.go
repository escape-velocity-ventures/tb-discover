package insights

import (
	"context"
	"log/slog"
	"sort"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Engine runs all analyzers across non-excluded namespaces.
type Engine struct {
	analyzers         []Analyzer
	excludeNamespaces map[string]bool
	log               *slog.Logger
}

// NewEngine creates an insight engine with all built-in analyzers.
func NewEngine(excludeNamespaces []string) *Engine {
	excl := make(map[string]bool, len(excludeNamespaces))
	for _, ns := range excludeNamespaces {
		excl[ns] = true
	}
	return &Engine{
		analyzers: []Analyzer{
			NewStalePodAnalyzer(),
			NewStuckTerminatingAnalyzer(),
			NewEvictedPodAnalyzer(),
			NewStalePvAffinityAnalyzer(),
			NewMissingProbesAnalyzer(),
			NewUnreadyWorkloadsAnalyzer(),
			NewCrashloopingAnalyzer(),
			NewResourcePressureAnalyzer(),
			NewImagePullIssuesAnalyzer(),
			NewMissingLimitsAnalyzer(),
		},
		excludeNamespaces: excl,
		log:               slog.Default().With("component", "insights"),
	}
}

// Analyze runs all analyzers across all non-excluded namespaces.
func (e *Engine) Analyze(ctx context.Context, clientset kubernetes.Interface) []ClusterInsight {
	// Get namespaces
	nsList, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		e.log.Error("failed to list namespaces for analysis", "error", err)
		return nil
	}

	var allInsights []ClusterInsight
	for _, ns := range nsList.Items {
		if e.excludeNamespaces[ns.Name] {
			continue
		}
		for _, analyzer := range e.analyzers {
			insights, err := analyzer.Analyze(ctx, clientset, ns.Name)
			if err != nil {
				e.log.Warn("analyzer failed", "analyzer", analyzer.Name(), "namespace", ns.Name, "error", err)
				continue
			}
			allInsights = append(allInsights, insights...)
		}
	}

	// Sort by severity: action > warning > suggestion > info
	sortInsights(allInsights)
	return allInsights
}

// ActiveFingerprints returns a sorted list of fingerprints from insights.
func ActiveFingerprints(insights []ClusterInsight) []string {
	fps := make([]string, len(insights))
	for i, ins := range insights {
		fps[i] = ins.Fingerprint
	}
	sort.Strings(fps)
	return fps
}

var severityOrder = map[string]int{
	"action":     0,
	"warning":    1,
	"suggestion": 2,
	"info":       3,
}

func sortInsights(insights []ClusterInsight) {
	sort.Slice(insights, func(i, j int) bool {
		oi, ok := severityOrder[insights[i].Severity]
		if !ok {
			oi = 9
		}
		oj, ok := severityOrder[insights[j].Severity]
		if !ok {
			oj = 9
		}
		return oi < oj
	})
}
