package remediation

import (
	"context"
	"testing"
	"time"

	"github.com/tinkerbelle-io/tb-discover/internal/insights"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestRemediatorDryRun(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "stale-pod", Namespace: "default"},
			Status:     corev1.PodStatus{Phase: corev1.PodSucceeded},
		},
	)

	cb := NewCircuitBreaker(10, 30*time.Minute)
	r := NewRemediator(clientset, cb, true) // dry-run=true

	ins := []insights.ClusterInsight{
		{
			Analyzer:       "stale_pods",
			TargetKind:     "Pod",
			TargetNS:       "default",
			TargetName:     "stale-pod",
			Fingerprint:    "abc123",
			ProposedAction: "delete_pod",
			AutoRemediable: true,
		},
	}

	results := r.Remediate(context.Background(), ins)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].DryRun {
		t.Error("result should be marked as dry-run")
	}
	if !results[0].Success {
		t.Error("dry-run should succeed")
	}

	// Verify pod still exists (not actually deleted)
	_, err := clientset.CoreV1().Pods("default").Get(context.Background(), "stale-pod", metav1.GetOptions{})
	if err != nil {
		t.Errorf("pod should still exist in dry-run mode: %v", err)
	}
}

func TestRemediatorLiveExecution(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "stale-pod", Namespace: "default"},
			Status:     corev1.PodStatus{Phase: corev1.PodSucceeded},
		},
	)

	cb := NewCircuitBreaker(10, 30*time.Minute)
	r := NewRemediator(clientset, cb, false) // dry-run=false

	ins := []insights.ClusterInsight{
		{
			Analyzer:       "stale_pods",
			TargetKind:     "Pod",
			TargetNS:       "default",
			TargetName:     "stale-pod",
			Fingerprint:    "abc123",
			ProposedAction: "delete_pod",
			AutoRemediable: true,
		},
	}

	results := r.Remediate(context.Background(), ins)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].DryRun {
		t.Error("result should not be dry-run")
	}
	if !results[0].Success {
		t.Errorf("expected success, got failure: %s", results[0].Message)
	}

	// Verify pod was deleted
	_, err := clientset.CoreV1().Pods("default").Get(context.Background(), "stale-pod", metav1.GetOptions{})
	if err == nil {
		t.Error("pod should have been deleted")
	}
}

func TestRemediatorRejectsDisallowedActions(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	cb := NewCircuitBreaker(10, 30*time.Minute)
	r := NewRemediator(clientset, cb, false)

	ins := []insights.ClusterInsight{
		{
			Analyzer:       "test",
			TargetKind:     "Deployment",
			TargetNS:       "default",
			TargetName:     "bad-deploy",
			Fingerprint:    "def456",
			ProposedAction: "restart_deployment", // not in allowlist
			AutoRemediable: true,
		},
	}

	results := r.Remediate(context.Background(), ins)
	if len(results) != 0 {
		t.Errorf("disallowed actions should be filtered out, got %d results", len(results))
	}
}

func TestRemediatorSkipsNonRemediable(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	cb := NewCircuitBreaker(10, 30*time.Minute)
	r := NewRemediator(clientset, cb, false)

	ins := []insights.ClusterInsight{
		{
			Analyzer:       "missing_probes",
			TargetKind:     "Deployment",
			TargetNS:       "default",
			TargetName:     "deploy",
			Fingerprint:    "ghi789",
			AutoRemediable: false, // not auto-remediable
		},
	}

	results := r.Remediate(context.Background(), ins)
	if len(results) != 0 {
		t.Errorf("non-remediable insights should be skipped, got %d results", len(results))
	}
}

func TestRemediatorCircuitBreakerIntegration(t *testing.T) {
	clientset := fake.NewSimpleClientset(
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pod-1", Namespace: "default"}},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pod-2", Namespace: "default"}},
		&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "pod-3", Namespace: "default"}},
	)

	cb := NewCircuitBreaker(2, 30*time.Minute) // max 2 per hour
	r := NewRemediator(clientset, cb, false)

	ins := []insights.ClusterInsight{
		{Analyzer: "stale_pods", TargetKind: "Pod", TargetNS: "default", TargetName: "pod-1", Fingerprint: "fp1", ProposedAction: "delete_pod", AutoRemediable: true},
		{Analyzer: "stale_pods", TargetKind: "Pod", TargetNS: "default", TargetName: "pod-2", Fingerprint: "fp2", ProposedAction: "delete_pod", AutoRemediable: true},
		{Analyzer: "stale_pods", TargetKind: "Pod", TargetNS: "default", TargetName: "pod-3", Fingerprint: "fp3", ProposedAction: "delete_pod", AutoRemediable: true},
	}

	results := r.Remediate(context.Background(), ins)
	// Should only remediate 2 (circuit breaker trips before 3rd)
	if len(results) != 2 {
		t.Errorf("expected 2 results (circuit breaker at max=2), got %d", len(results))
	}
}
