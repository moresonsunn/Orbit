//go:build linux

package host

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/phili/orbit/internal/model"
	"golang.org/x/sys/unix"
)

var physicalFSTypes = map[string]bool{
	"ext4":    true,
	"ext3":    true,
	"btrfs":   true,
	"xfs":     true,
	"zfs":     true,
	"f2fs":    true,
	"vfat":    true,
	"ntfs":    true,
	"exfat":   true,
	"overlay": true,
}

func (c *Collector) detectMounts() []DiskMount {
	mountFilePath := c.procPath("mounts")
	if _, err := os.Stat(mountFilePath); err != nil {
		mountFilePath = c.procPath("1/mounts")
	}

	content, err := os.ReadFile(mountFilePath)
	if err != nil {
		return nil
	}

	seenMounts := make(map[string]bool)
	var mounts []DiskMount
	scanner := bufio.NewScanner(strings.NewReader(string(content)))

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 3 {
			continue
		}
		device := fields[0]
		mountPoint := fields[1]
		fsType := fields[2]

		if !physicalFSTypes[fsType] {
			continue
		}
		// Skip docker internal container mounts unless root
		if strings.Contains(mountPoint, "/docker/") || strings.Contains(mountPoint, "/containerd/") {
			continue
		}
		if seenMounts[mountPoint] {
			continue
		}
		seenMounts[mountPoint] = true

		targetPath := mountPoint
		if c.cfg.HostRootFS != "" && c.cfg.HostRootFS != "/" {
			targetPath = filepath.Join(c.cfg.HostRootFS, mountPoint)
		}

		var stat unix.Statfs_t
		if err := unix.Statfs(targetPath, &stat); err != nil {
			// Fallback to checking mountPoint directly
			if err := unix.Statfs(mountPoint, &stat); err != nil {
				continue
			}
		}

		total := stat.Blocks * uint64(stat.Bsize)
		free := stat.Bavail * uint64(stat.Bsize)
		if total == 0 {
			continue
		}
		used := total - free

		usedPct := model.Val(round2(float64(used) / float64(total) * 100.0))

		mounts = append(mounts, DiskMount{
			MountPoint:  mountPoint,
			Device:      device,
			FSType:      fsType,
			TotalBytes:  total,
			UsedBytes:   used,
			FreeBytes:   free,
			UsedPercent: usedPct,
			ReadBps:     model.NA(model.Unavailable, "per-mount I/O not separated"),
			WriteBps:    model.NA(model.Unavailable, "per-mount I/O not separated"),
		})
	}

	return mounts
}

