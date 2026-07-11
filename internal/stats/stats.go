package stats

import (
	"fmt"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

type SystemStats struct {
	CPUPercent  float64
	RAMPercent  float64
	DiskPercent float64
	Uptime      string
	OS          string
	Kernel      string
}

// GetSystemStats fetches the current device statistics
func GetSystemStats() SystemStats {
	var stats SystemStats

	// CPU
	cpuPercents, err := cpu.Percent(0, false)
	if err == nil && len(cpuPercents) > 0 {
		stats.CPUPercent = cpuPercents[0]
	}

	// RAM
	vMem, err := mem.VirtualMemory()
	if err == nil {
		stats.RAMPercent = vMem.UsedPercent
	}

	// Disk (root directory)
	// On Windows you might use "C:", on Linux "/"
	// Here we use "/" as a common fallback that gopsutil resolves on Unix
	// You might want to detect OS to set path
	path := "/"
	dInfo, err := disk.Usage(path)
	if err == nil {
		stats.DiskPercent = dInfo.UsedPercent
	} else {
        // Fallback for Windows if "/" fails
        dInfo, err = disk.Usage("C:\\")
        if err == nil {
            stats.DiskPercent = dInfo.UsedPercent
        }
    }

	// Host Info
	hInfo, err := host.Info()
	if err == nil {
		stats.OS = fmt.Sprintf("%s %s", hInfo.OS, hInfo.Platform)
		stats.Kernel = hInfo.KernelVersion
		
		// Format Uptime from seconds
		uptimeDuration := time.Duration(hInfo.Uptime) * time.Second
		days := uptimeDuration / (24 * time.Hour)
		uptimeDuration -= days * 24 * time.Hour
		hours := uptimeDuration / time.Hour
		stats.Uptime = fmt.Sprintf("%dd %dh", days, hours)
	}

	return stats
}

// FormatBar creates a textual progress bar. Example: "█████░░░░░ 50%"
func FormatBar(percent float64) string {
	totalBlocks := 10
	filledBlocks := int((percent / 100.0) * float64(totalBlocks))
	if filledBlocks < 0 {
		filledBlocks = 0
	}
	if filledBlocks > totalBlocks {
		filledBlocks = totalBlocks
	}

	bar := ""
	for i := 0; i < totalBlocks; i++ {
		if i < filledBlocks {
			bar += "█"
		} else {
			bar += "░"
		}
	}
	return fmt.Sprintf("%s %d%%", bar, int(percent))
}
