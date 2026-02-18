// Package scanners implements host discovery scanners for tb-discover.
package scanners

// HostScanResult is the payload shape expected by edge-ingest.
// Matches the TypeScript HostScanResult in tb-edge/tb-agent.
type HostScanResult struct {
	Name         string                            `json:"name"`
	Type         string                            `json:"type"` // "baremetal", "vm", "cloud"
	Location     string                            `json:"location"`
	Description  string                            `json:"description"`
	System       SystemInfo                        `json:"system"`
	Network      NetworkInfo                       `json:"network"`
	Access       AccessInfo                        `json:"access"`
	Capabilities map[string]map[string]interface{} `json:"capabilities"`
	Services     []HostService                     `json:"services"`
	Disk         []DiskInfo                        `json:"disk"`
	Storage      []StorageDevice                   `json:"storage,omitempty"`
	Resources    *ResourceInfo                     `json:"resources,omitempty"`
}

type SystemInfo struct {
	OS        string `json:"os"`
	OSVersion string `json:"os_version"`
	Arch      string `json:"arch"`
	CPUModel  string `json:"cpu_model"`
	CPUCores  int    `json:"cpu_cores"`
	MemoryGB  int    `json:"memory_gb"`
}

type NetworkInfo struct {
	Hostname   string             `json:"hostname"`
	Interfaces []NetworkInterface `json:"interfaces"`
}

type NetworkInterface struct {
	Name   string `json:"name"`
	Type   string `json:"type"` // "ethernet", "wifi", "bridge", "tunnel", "virtual", "loopback", "other"
	IP     string `json:"ip,omitempty"`
	IPv6   string `json:"ipv6,omitempty"`
	MAC    string `json:"mac,omitempty"`
	Status string `json:"status,omitempty"` // "up", "down"
}

type AccessInfo struct {
	Primary              string        `json:"primary"`
	Methods              []interface{} `json:"methods"`
	SudoRequiresPassword bool          `json:"sudo_requires_password"`
}

type HostService struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Port        int    `json:"port,omitempty"`
	Protocol    string `json:"protocol,omitempty"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status,omitempty"`
}

type DiskInfo struct {
	Filesystem string `json:"filesystem"`
	Size       string `json:"size"`
	Used       string `json:"used"`
	Available  string `json:"available"`
	UsePercent string `json:"use_percent"`
	Mount      string `json:"mount"`
}

type StorageDevice struct {
	Device     string             `json:"device"`
	Model      string             `json:"model,omitempty"`
	Size       string             `json:"size"`
	Bus        string             `json:"bus,omitempty"`
	Removable  bool               `json:"removable"`
	Protocol   string             `json:"protocol,omitempty"`
	Partitions []StoragePartition `json:"partitions"`
}

type StoragePartition struct {
	Device     string `json:"device"`
	Name       string `json:"name,omitempty"`
	FSType     string `json:"fsType,omitempty"`
	Size       string `json:"size"`
	MountPoint string `json:"mountPoint,omitempty"`
}

type ResourceInfo struct {
	LoadAvg [3]float64  `json:"loadAvg,omitempty"`
	Memory  *MemoryInfo `json:"memory,omitempty"`
	Uptime  string      `json:"uptime,omitempty"`
}

type MemoryInfo struct {
	TotalGB     float64 `json:"total_gb"`
	UsedGB      float64 `json:"used_gb"`
	AvailableGB float64 `json:"available_gb"`
	UsePercent  float64 `json:"use_percent"`
}
