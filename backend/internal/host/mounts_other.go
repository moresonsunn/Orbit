//go:build !linux

package host

import (
	"github.com/phili/orbit/internal/model"
)

func (c *Collector) detectMounts() []DiskMount {
	// On non-Linux (e.g. Windows during development/testing), provide placeholder mount
	return []DiskMount{
		{
			MountPoint:  "/",
			Device:      "/dev/root",
			FSType:      "ext4",
			TotalBytes:  100 * 1024 * 1024 * 1024,
			UsedBytes:   35 * 1024 * 1024 * 1024,
			FreeBytes:   65 * 1024 * 1024 * 1024,
			UsedPercent: model.Val(35.0),
			ReadBps:     model.NA(model.Unavailable, "dev-mode"),
			WriteBps:    model.NA(model.Unavailable, "dev-mode"),
		},
	}
}

