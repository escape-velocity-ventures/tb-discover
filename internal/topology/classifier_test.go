package topology

import "testing"

func TestClassifyNIC(t *testing.T) {
	tests := []struct {
		name string
		want NICType
	}{
		// Loopback
		{"lo", NICLoopback},
		{"lo0", NICLoopback},

		// Physical
		{"en0", NICPhysical},
		{"en1", NICPhysical},
		{"eth0", NICPhysical},
		{"eth1", NICPhysical},
		{"eno1", NICPhysical},
		{"em0", NICPhysical},

		// CNI (Kubernetes)
		{"cali123abc", NICCNI},
		{"flannel.1", NICCNI},
		{"cilium_host", NICCNI},
		{"vxlan.calico", NICCNI},
		{"weave", NICCNI},
		{"cni0", NICCNI},
		{"veth1234", NICCNI},

		// Bridge (hypervisor)
		{"vmnet0", NICBridge},
		{"vmnet1", NICBridge},
		{"virbr0", NICBridge},
		{"br-abc123", NICBridge},
		{"bridge0", NICBridge},
		{"docker0", NICBridge},

		// Virtio (VM guest)
		{"enp0s1", NICVirtio},
		{"enp0s3", NICVirtio},
		{"ens3", NICVirtio},
		{"ens160", NICVirtio},

		// Tunnel / VPN
		{"wg0", NICTunnel},
		{"tun0", NICTunnel},
		{"utun0", NICTunnel},
		{"utun3", NICTunnel},
		{"tailscale0", NICTunnel},

		// Wireless
		{"wlan0", NICWireless},
		{"wlp2s0", NICWireless},

		// Unknown
		{"awdl0", NICUnknown},
		{"llw0", NICUnknown},
		{"anpi0", NICUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyNIC(tt.name)
			if got != tt.want {
				t.Errorf("ClassifyNIC(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}
