package scanner

// Registry maps profiles to their scanners.
type Registry struct {
	scanners map[Profile][]Scanner
}

// NewRegistry creates a registry with all known scanners assigned to profiles.
func NewRegistry() *Registry {
	r := &Registry{
		scanners: make(map[Profile][]Scanner),
	}

	// Minimal: just host info
	minimal := []Scanner{
		NewHostScanner(),
	}

	// Standard: host + network + storage + topology
	standard := append(minimal,
		NewNetworkScanner(),
		NewStorageScanner(),
	)

	// Full: standard + containers + k8s + power + iot
	full := append(standard,
		NewContainerScanner(),
		NewK8sScanner(),
		NewPowerScanner(),
		NewIoTScanner(),
	)

	r.scanners[ProfileMinimal] = minimal
	r.scanners[ProfileStandard] = standard
	r.scanners[ProfileFull] = full

	return r
}

// ForProfile returns the scanners for the given profile, filtered to the current platform.
func (r *Registry) ForProfile(p Profile) []Scanner {
	all := r.scanners[p]
	var result []Scanner
	for _, s := range all {
		if SupportsCurrentPlatform(s) {
			result = append(result, s)
		}
	}
	return result
}
