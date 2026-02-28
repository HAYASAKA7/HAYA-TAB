//go:build !windows

package app

import (
	"syscall"
)

// GetDiskFreeSpace returns the free space of the drive containing the path in bytes.
func GetDiskFreeSpace(path string) (uint64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, err
	}
	// Available blocks * size per block
	return stat.Bavail * uint64(stat.Bsize), nil
}
