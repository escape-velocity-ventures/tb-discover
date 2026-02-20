package remediation

import (
	"fmt"
	"sync"
	"time"
)

// CircuitBreaker limits remediation rate with a sliding window and per-resource cooldown.
type CircuitBreaker struct {
	mu              sync.Mutex
	maxPerHour      int
	cooldown        time.Duration
	recentTimes     []time.Time
	resourceCooldowns map[string]time.Time
}

// NewCircuitBreaker creates a circuit breaker with the given limits.
func NewCircuitBreaker(maxPerHour int, cooldown time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		maxPerHour:        maxPerHour,
		cooldown:          cooldown,
		resourceCooldowns: make(map[string]time.Time),
	}
}

func resourceKey(kind, namespace, name string) string {
	return fmt.Sprintf("%s/%s/%s", kind, namespace, name)
}

// IsOpen returns true if the circuit breaker has tripped (too many remediations).
func (cb *CircuitBreaker) IsOpen() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.pruneOld()
	return len(cb.recentTimes) >= cb.maxPerHour
}

// IsOnCooldown returns true if the specific resource was recently remediated.
func (cb *CircuitBreaker) IsOnCooldown(kind, namespace, name string) bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	key := resourceKey(kind, namespace, name)
	last, ok := cb.resourceCooldowns[key]
	if !ok {
		return false
	}
	return time.Since(last) < cb.cooldown
}

// Record notes a successful remediation for rate limiting.
func (cb *CircuitBreaker) Record(kind, namespace, name string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	now := time.Now()
	cb.recentTimes = append(cb.recentTimes, now)
	cb.resourceCooldowns[resourceKey(kind, namespace, name)] = now
}

// pruneOld removes entries older than 1 hour from the sliding window.
func (cb *CircuitBreaker) pruneOld() {
	cutoff := time.Now().Add(-1 * time.Hour)
	i := 0
	for i < len(cb.recentTimes) && cb.recentTimes[i].Before(cutoff) {
		i++
	}
	cb.recentTimes = cb.recentTimes[i:]
}
