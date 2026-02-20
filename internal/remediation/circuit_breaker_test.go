package remediation

import (
	"testing"
	"time"
)

func TestCircuitBreakerSlidingWindow(t *testing.T) {
	cb := NewCircuitBreaker(3, 5*time.Minute)

	// Record 3 remediations
	cb.Record("Pod", "default", "pod-1")
	cb.Record("Pod", "default", "pod-2")
	cb.Record("Pod", "default", "pod-3")

	if !cb.IsOpen() {
		t.Error("circuit breaker should be open after 3 remediations (max=3)")
	}
}

func TestCircuitBreakerNotOpenBelowMax(t *testing.T) {
	cb := NewCircuitBreaker(10, 5*time.Minute)

	cb.Record("Pod", "default", "pod-1")
	cb.Record("Pod", "default", "pod-2")

	if cb.IsOpen() {
		t.Error("circuit breaker should not be open with only 2 of 10 remediations")
	}
}

func TestCircuitBreakerBoundary(t *testing.T) {
	cb := NewCircuitBreaker(2, 5*time.Minute)

	cb.Record("Pod", "default", "pod-1")
	if cb.IsOpen() {
		t.Error("should not be open at 1/2")
	}

	cb.Record("Pod", "default", "pod-2")
	if !cb.IsOpen() {
		t.Error("should be open at 2/2")
	}
}

func TestPerResourceCooldown(t *testing.T) {
	cb := NewCircuitBreaker(100, 30*time.Minute)

	cb.Record("Pod", "default", "stuck-pod")

	if !cb.IsOnCooldown("Pod", "default", "stuck-pod") {
		t.Error("resource should be on cooldown immediately after recording")
	}

	if cb.IsOnCooldown("Pod", "default", "other-pod") {
		t.Error("different resource should not be on cooldown")
	}

	if cb.IsOnCooldown("Pod", "other-ns", "stuck-pod") {
		t.Error("same name in different namespace should not be on cooldown")
	}
}

func TestCircuitBreakerSlidingWindowExpiry(t *testing.T) {
	cb := NewCircuitBreaker(2, 5*time.Minute)

	// Manually inject old timestamps
	cb.mu.Lock()
	twoHoursAgo := time.Now().Add(-2 * time.Hour)
	cb.recentTimes = []time.Time{twoHoursAgo, twoHoursAgo}
	cb.mu.Unlock()

	// Old entries should be pruned, so breaker should be closed
	if cb.IsOpen() {
		t.Error("circuit breaker should be closed after old entries expire")
	}
}
