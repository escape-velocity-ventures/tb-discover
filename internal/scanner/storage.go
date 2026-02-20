package scanner

import (
	"context"
	"encoding/json"
)

// StorageInfo holds storage scan results.
type StorageInfo struct {
	Filesystems []FilesystemInfo `json:"filesystems"`
	Disks       []DiskInfo       `json:"disks,omitempty"`
}

// FilesystemInfo represents a mounted filesystem.
type FilesystemInfo struct {
	Filesystem string  `json:"filesystem"`
	MountPoint string  `json:"mount_point"`
	Type       string  `json:"type,omitempty"`
	SizeGB     float64 `json:"size_gb"`
	UsedGB     float64 `json:"used_gb"`
	AvailGB    float64 `json:"avail_gb"`
	UsePct     float64 `json:"use_pct"`
}

// DiskInfo represents a physical or virtual disk.
type DiskInfo struct {
	Name     string `json:"name"`
	SizeGB   float64 `json:"size_gb"`
	Type     string `json:"type,omitempty"` // disk, part
	Model    string `json:"model,omitempty"`
	Serial   string `json:"serial,omitempty"`
	ReadOnly bool   `json:"read_only,omitempty"`
}

// StorageScanner collects disk and filesystem information.
type StorageScanner struct{}

// NewStorageScanner creates a new StorageScanner.
func NewStorageScanner() *StorageScanner {
	return &StorageScanner{}
}

func (s *StorageScanner) Name() string       { return "storage" }
func (s *StorageScanner) Platforms() []string { return nil }

func (s *StorageScanner) Scan(ctx context.Context, runner CommandRunner) (json.RawMessage, error) {
	info := StorageInfo{}

	if err := collectStorageInfo(ctx, runner, &info); err != nil {
		return nil, err
	}

	return json.Marshal(info)
}
