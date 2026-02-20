package insights

import "testing"

func TestFingerprintStability(t *testing.T) {
	// Same input must always produce same output
	fp1 := MakeFingerprint("stale_pods", "Pod", "default", "my-pod")
	fp2 := MakeFingerprint("stale_pods", "Pod", "default", "my-pod")
	if fp1 != fp2 {
		t.Errorf("fingerprint not stable: %q != %q", fp1, fp2)
	}
}

func TestFingerprintUniqueness(t *testing.T) {
	fp1 := MakeFingerprint("stale_pods", "Pod", "default", "pod-a")
	fp2 := MakeFingerprint("stale_pods", "Pod", "default", "pod-b")
	fp3 := MakeFingerprint("evicted_pods", "Pod", "default", "pod-a")
	fp4 := MakeFingerprint("stale_pods", "Pod", "other-ns", "pod-a")

	fps := map[string]string{
		"fp1": fp1,
		"fp2": fp2,
		"fp3": fp3,
		"fp4": fp4,
	}

	seen := make(map[string]string)
	for name, fp := range fps {
		if existing, ok := seen[fp]; ok {
			t.Errorf("fingerprint collision between %s and %s: %q", existing, name, fp)
		}
		seen[fp] = name
	}
}

func TestFingerprintLength(t *testing.T) {
	fp := MakeFingerprint("stale_pods", "Pod", "default", "my-pod")
	if len(fp) != 16 {
		t.Errorf("expected 16-char fingerprint, got %d: %q", len(fp), fp)
	}
}
