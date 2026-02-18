// tb-discover — TinkerBelle Discovery Agent
//
// A single binary that runs on any machine (bare metal, VM, K8s node)
// and reports system, disk, storage, and network info to TinkerBelle SaaS.
//
// Usage:
//
//	tb-discover                            # daemon mode (default)
//	tb-discover --once                     # one-shot scan and exit
//	tb-discover --mode k8s                 # K8s DaemonSet mode (nsenter)
//	SCAN_INTERVAL_SECONDS=0 tb-discover    # one-shot via env var
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/escape-velocity-ventures/tb-discover/internal/config"
	"github.com/escape-velocity-ventures/tb-discover/internal/scanners"
	"github.com/escape-velocity-ventures/tb-discover/internal/upload"
)

var version = "dev"

func main() {
	var (
		once    bool
		k8sMode bool
		dryRun  bool
		ver     bool
	)

	flag.BoolVar(&once, "once", false, "Run a single scan and exit")
	flag.BoolVar(&k8sMode, "mode-k8s", false, "K8s DaemonSet mode (use nsenter for host access)")
	flag.BoolVar(&dryRun, "dry-run", false, "Scan and print payload as JSON (no upload)")
	flag.BoolVar(&ver, "version", false, "Print version and exit")
	flag.Parse()

	if ver {
		fmt.Printf("tb-discover %s\n", version)
		os.Exit(0)
	}

	// Dry-run mode doesn't need credentials
	if dryRun {
		scanners.SetNsenter(k8sMode)
		nodeName := os.Getenv("NODE_NAME")
		if nodeName == "" || nodeName == "auto" {
			nodeName = scanners.GetHostname()
		}
		hostType := os.Getenv("HOST_TYPE")
		if hostType == "" {
			hostType = "baremetal"
		}
		doDryRun(nodeName, hostType)
		return
	}

	cfg, err := config.Load()
	if err != nil {
		logf("Fatal: %v", err)
		os.Exit(1)
	}

	// Override mode from flags
	if once || cfg.ScanIntervalSeconds == 0 {
		cfg.Mode = "oneshot"
	}
	if k8sMode || cfg.Mode == "k8s" {
		scanners.SetNsenter(true)
		if cfg.Profile == "full" {
			cfg.Profile = "standard" // lighter profile in K8s mode
		}
		if cfg.HostType == "baremetal" {
			cfg.HostType = "vm"
		}
	}

	logf("tb-discover %s starting (mode=%s, profile=%s, interval=%ds)",
		version, cfg.Mode, cfg.Profile, cfg.ScanIntervalSeconds)

	// Resolve hostname
	if cfg.NodeName == "auto" {
		cfg.NodeName = scanners.GetHostname()
	}
	logf("Node: %s", cfg.NodeName)

	// First scan
	runScan(cfg)

	// One-shot: exit
	if cfg.Mode == "oneshot" {
		logf("One-shot mode — exiting")
		return
	}

	// Daemon loop
	for {
		sleepMs := jitter(int64(cfg.ScanIntervalSeconds) * 1000)
		logf("Next scan in %ds", sleepMs/1000)
		time.Sleep(time.Duration(sleepMs) * time.Millisecond)

		runScan(cfg)
	}
}

func runScan(cfg *config.Config) {
	start := time.Now()
	logf("Starting scan for node: %s", cfg.NodeName)

	// Run scanners
	system := scanners.ScanSystem()
	disk := scanners.ScanDisk()
	storage := scanners.ScanStorage()
	network := scanners.ScanNetwork()
	resources := scanners.ScanResources()

	hostname := cfg.NodeName

	// Determine reporting source
	source := "standalone"
	if cfg.Mode == "k8s" {
		source = "daemonset"
	}

	host := scanners.HostScanResult{
		Name:        hostname,
		Type:        cfg.HostType,
		Location:    "",
		Description: fmt.Sprintf("tb-discover scan of %s", hostname),
		Source:      source,
		HardwareID:  scanners.ComputeHardwareID(network, storage),
		System:      system,
		Network: scanners.NetworkInfo{
			Hostname:   hostname,
			Interfaces: network,
		},
		Access: scanners.AccessInfo{
			Primary:              "tb-discover",
			Methods:              []interface{}{},
			SudoRequiresPassword: false,
		},
		Capabilities: map[string]map[string]interface{}{},
		Services:     []scanners.HostService{},
		Disk:         disk,
		Storage:      storage,
		Resources:    resources,
	}

	if host.Disk == nil {
		host.Disk = []scanners.DiskInfo{}
	}
	if host.Storage == nil {
		host.Storage = []scanners.StorageDevice{}
	}

	durationMs := time.Since(start).Milliseconds()
	logf("Scan completed in %dms (hwid=%s) — uploading...", durationMs, host.HardwareID)

	result := upload.Send(
		cfg.IngestURL,
		cfg.AgentToken,
		cfg.AnonKey,
		host,
		"tb-discover/"+version,
		durationMs,
	)

	if result.OK {
		logf("Upload successful (%d)", result.Status)
	} else {
		logf("Upload failed: %d — %s", result.Status, result.Body)
	}
}

func jitter(baseMs int64) int64 {
	// ±60s random jitter to stagger uploads across nodes
	j := int64((rand.Float64() - 0.5) * 2 * 60000)
	result := baseMs + j
	if result < 0 {
		return 0
	}
	return result
}

func doDryRun(nodeName, hostType string) {
	start := time.Now()
	logf("Dry-run scan for node: %s", nodeName)

	system := scanners.ScanSystem()
	disk := scanners.ScanDisk()
	storage := scanners.ScanStorage()
	network := scanners.ScanNetwork()
	resources := scanners.ScanResources()

	host := scanners.HostScanResult{
		Name:        nodeName,
		Type:        hostType,
		Description: fmt.Sprintf("tb-discover scan of %s", nodeName),
		Source:      "standalone",
		HardwareID:  scanners.ComputeHardwareID(network, storage),
		System:      system,
		Network: scanners.NetworkInfo{
			Hostname:   nodeName,
			Interfaces: network,
		},
		Access: scanners.AccessInfo{
			Primary:              "tb-discover",
			Methods:              []interface{}{},
			SudoRequiresPassword: false,
		},
		Capabilities: map[string]map[string]interface{}{},
		Services:     []scanners.HostService{},
		Disk:         disk,
		Storage:      storage,
		Resources:    resources,
	}

	if host.Disk == nil {
		host.Disk = []scanners.DiskInfo{}
	}
	if host.Storage == nil {
		host.Storage = []scanners.StorageDevice{}
	}

	durationMs := time.Since(start).Milliseconds()
	logf("Scan completed in %dms (hwid=%s)", durationMs, host.HardwareID)

	out, _ := json.MarshalIndent(host, "", "  ")
	fmt.Println(string(out))
}

func logf(format string, args ...interface{}) {
	ts := time.Now().UTC().Format(time.RFC3339)
	fmt.Printf("[tb-discover] %s %s\n", ts, fmt.Sprintf(format, args...))
}
