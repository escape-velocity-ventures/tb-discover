package remediation

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/tinkerbelle-io/tb-discover/internal/insights"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Remediator auto-fixes detected issues with strict safety controls.
type Remediator struct {
	clientset      kubernetes.Interface
	circuitBreaker *CircuitBreaker
	dryRun         bool
	log            *slog.Logger
}

// NewRemediator creates a new remediator.
func NewRemediator(clientset kubernetes.Interface, cb *CircuitBreaker, dryRun bool) *Remediator {
	return &Remediator{
		clientset:      clientset,
		circuitBreaker: cb,
		dryRun:         dryRun,
		log:            slog.Default().With("component", "remediator"),
	}
}

// Remediate processes auto-remediable insights and returns results.
func (r *Remediator) Remediate(ctx context.Context, allInsights []insights.ClusterInsight) []RemediationResult {
	var results []RemediationResult

	for _, insight := range allInsights {
		if !insight.AutoRemediable {
			continue
		}
		action := Action(insight.ProposedAction)
		if !AllowedActions[action] {
			continue
		}

		if r.circuitBreaker.IsOpen() {
			r.log.Warn("circuit breaker open, skipping remaining remediations")
			break
		}

		if r.circuitBreaker.IsOnCooldown(insight.TargetKind, insight.TargetNS, insight.TargetName) {
			r.log.Debug("resource on cooldown, skipping",
				"kind", insight.TargetKind, "ns", insight.TargetNS, "name", insight.TargetName)
			continue
		}

		result := r.execute(ctx, action, insight)
		results = append(results, result)

		if result.Success && !r.dryRun {
			r.circuitBreaker.Record(insight.TargetKind, insight.TargetNS, insight.TargetName)
		}
	}

	if len(results) > 0 {
		succeeded := 0
		for _, res := range results {
			if res.Success {
				succeeded++
			}
		}
		r.log.Info("remediation complete",
			"succeeded", succeeded, "failed", len(results)-succeeded, "dry_run", r.dryRun)
	}

	return results
}

func (r *Remediator) execute(ctx context.Context, action Action, insight insights.ClusterInsight) RemediationResult {
	base := RemediationResult{
		Action:             action,
		TargetKind:         insight.TargetKind,
		TargetNamespace:    insight.TargetNS,
		TargetName:         insight.TargetName,
		InsightFingerprint: insight.Fingerprint,
		Reason:             insight.Title,
		DryRun:             r.dryRun,
	}

	if r.dryRun {
		r.log.Info("[DRY RUN] would execute",
			"action", action, "kind", insight.TargetKind,
			"ns", insight.TargetNS, "name", insight.TargetName)
		base.Success = true
		base.Message = fmt.Sprintf("[DRY RUN] %s skipped", action)
		return base
	}

	var err error
	switch action {
	case ActionDeletePod:
		err = r.clientset.CoreV1().Pods(insight.TargetNS).Delete(ctx, insight.TargetName, metav1.DeleteOptions{})
	case ActionForceDeletePod:
		grace := int64(0)
		err = r.clientset.CoreV1().Pods(insight.TargetNS).Delete(ctx, insight.TargetName, metav1.DeleteOptions{
			GracePeriodSeconds: &grace,
		})
	case ActionDeletePVC:
		err = r.clientset.CoreV1().PersistentVolumeClaims(insight.TargetNS).Delete(ctx, insight.TargetName, metav1.DeleteOptions{})
	default:
		base.Success = false
		base.Message = fmt.Sprintf("unknown action: %s", action)
		return base
	}

	if err != nil {
		base.Success = false
		base.Message = fmt.Sprintf("failed to %s %s/%s: %v", action, insight.TargetNS, insight.TargetName, err)
		r.log.Error("remediation failed", "action", action, "error", err)
	} else {
		base.Success = true
		base.Message = fmt.Sprintf("auto-remediated: %s %s/%s", action, insight.TargetNS, insight.TargetName)
		r.log.Info("remediated", "action", action, "ns", insight.TargetNS, "name", insight.TargetName)
	}

	return base
}
