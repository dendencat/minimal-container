package cg

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gomini/internal/util"
)

// CgroupManager manages cgroup v2 resources for a container
type CgroupManager struct {
	CgroupPath string
	Controllers []string
}

// ResourceLimits defines resource limits for the container
type ResourceLimits struct {
	CPUQuota  int64 // CPU quota in microseconds
	CPUPeriod int64 // CPU period in microseconds
	Memory    int64 // Memory limit in bytes
	Pids      int   // Maximum number of processes
}

const (
	defaultCPUPeriod = 100000 // 100ms default period
)

// DetectCgroupV2MountPoint finds the cgroup v2 mount point
func DetectCgroupV2MountPoint() (string, error) {
	// Check common cgroup v2 mount points
	commonPaths := []string{
		"/sys/fs/cgroup",
		"/sys/fs/cgroup/unified",
	}

	for _, path := range commonPaths {
		if isCgroupV2(path) {
			return path, nil
		}
	}

	// Parse /proc/mounts to find cgroup2 filesystem
	file, err := os.Open("/proc/mounts")
	if err != nil {
		return "", util.NewError("open /proc/mounts", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 3 && fields[2] == "cgroup2" {
			return fields[1], nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", util.NewError("read /proc/mounts", err)
	}

	return "", util.NewSimpleError("detect cgroup v2", "cgroup v2 not found on this system")
}

// isCgroupV2 checks if the given path is a cgroup v2 mount point
func isCgroupV2(path string) bool {
	// Check if cgroup.controllers file exists (cgroup v2 specific)
	controllersPath := filepath.Join(path, "cgroup.controllers")
	if _, err := os.Stat(controllersPath); err == nil {
		return true
	}
	return false
}

// NewCgroupManager creates a new cgroup manager for the container
func NewCgroupManager(containerID string) (*CgroupManager, error) {
	mountPoint, err := DetectCgroupV2MountPoint()
	if err != nil {
		return nil, util.WrapError("detect cgroup v2 mount point", err)
	}

	// Create cgroup path for this container
	cgroupPath := filepath.Join(mountPoint, "gomini", containerID)

	// Get available controllers
	controllers, err := getAvailableControllers(mountPoint)
	if err != nil {
		return nil, util.WrapError("get available controllers", err)
	}

	return &CgroupManager{
		CgroupPath:  cgroupPath,
		Controllers: controllers,
	}, nil
}

// getAvailableControllers reads available controllers from cgroup.controllers
func getAvailableControllers(mountPoint string) ([]string, error) {
	controllersPath := filepath.Join(mountPoint, "cgroup.controllers")
	data, err := os.ReadFile(controllersPath)
	if err != nil {
		return nil, util.NewPathError("read cgroup controllers", controllersPath, err)
	}

	controllers := strings.Fields(strings.TrimSpace(string(data)))
	return controllers, nil
}

// Setup creates the cgroup and enables required controllers
func (cm *CgroupManager) Setup() error {
	// Create cgroup directory
	if err := os.MkdirAll(cm.CgroupPath, 0755); err != nil {
		return util.NewPathError("create cgroup directory", cm.CgroupPath, err)
	}

	// Enable required controllers in parent cgroup
	parentPath := filepath.Dir(cm.CgroupPath)
	subtreeControlPath := filepath.Join(parentPath, "cgroup.subtree_control")

	// Try to enable cpu, memory, and pids controllers
	requiredControllers := []string{"cpu", "memory", "pids"}
	var enabledControllers []string

	for _, controller := range requiredControllers {
		if contains(cm.Controllers, controller) {
			enabledControllers = append(enabledControllers, "+"+controller)
		}
	}

	if len(enabledControllers) > 0 {
		controlString := strings.Join(enabledControllers, " ")
		if err := os.WriteFile(subtreeControlPath, []byte(controlString), 0644); err != nil {
			// Log warning but don't fail - might not have permission or already enabled
			fmt.Fprintf(os.Stderr, "Warning: failed to enable controllers: %v\n", err)
		}
	}

	return nil
}

// ApplyLimits applies resource limits to the cgroup
func (cm *CgroupManager) ApplyLimits(limits *ResourceLimits) error {
	if limits.CPUQuota > 0 {
		if err := cm.setCPULimit(limits.CPUQuota, limits.CPUPeriod); err != nil {
			return util.WrapError("set CPU limit", err)
		}
	}

	if limits.Memory > 0 {
		if err := cm.setMemoryLimit(limits.Memory); err != nil {
			return util.WrapError("set memory limit", err)
		}
	}

	if limits.Pids > 0 {
		if err := cm.setPidsLimit(limits.Pids); err != nil {
			return util.WrapError("set pids limit", err)
		}
	}

	return nil
}

// setCPULimit sets CPU quota and period
func (cm *CgroupManager) setCPULimit(quota int64, period int64) error {
	if period == 0 {
		period = defaultCPUPeriod
	}

	cpuMaxPath := filepath.Join(cm.CgroupPath, "cpu.max")
	cpuMaxValue := fmt.Sprintf("%d %d", quota, period)

	if err := os.WriteFile(cpuMaxPath, []byte(cpuMaxValue), 0644); err != nil {
		return util.NewPathError("write cpu.max", cpuMaxPath, err)
	}

	return nil
}

// setMemoryLimit sets memory limit
func (cm *CgroupManager) setMemoryLimit(limit int64) error {
	memoryMaxPath := filepath.Join(cm.CgroupPath, "memory.max")
	limitValue := strconv.FormatInt(limit, 10)

	if err := os.WriteFile(memoryMaxPath, []byte(limitValue), 0644); err != nil {
		return util.NewPathError("write memory.max", memoryMaxPath, err)
	}

	return nil
}

// setPidsLimit sets maximum number of processes
func (cm *CgroupManager) setPidsLimit(limit int) error {
	pidsMaxPath := filepath.Join(cm.CgroupPath, "pids.max")
	limitValue := strconv.Itoa(limit)

	if err := os.WriteFile(pidsMaxPath, []byte(limitValue), 0644); err != nil {
		return util.NewPathError("write pids.max", pidsMaxPath, err)
	}

	return nil
}

// AddProcess adds a process to the cgroup
func (cm *CgroupManager) AddProcess(pid int) error {
	procsPath := filepath.Join(cm.CgroupPath, "cgroup.procs")
	pidValue := strconv.Itoa(pid)

	if err := os.WriteFile(procsPath, []byte(pidValue), 0644); err != nil {
		return util.NewPathError("write cgroup.procs", procsPath, err)
	}

	return nil
}

// GetStats retrieves resource usage statistics
func (cm *CgroupManager) GetStats() (*ResourceStats, error) {
	stats := &ResourceStats{}

	// Read CPU stats
	cpuStat, err := cm.readCPUStat()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to read CPU stats: %v\n", err)
	} else {
		stats.CPUUsage = cpuStat
	}

	// Read memory stats
	memStat, err := cm.readMemoryStat()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to read memory stats: %v\n", err)
	} else {
		stats.MemoryUsage = memStat
	}

	// Read pids stats
	pidsStat, err := cm.readPidsStat()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to read pids stats: %v\n", err)
	} else {
		stats.PidsCount = pidsStat
	}

	return stats, nil
}

// readCPUStat reads CPU usage statistics
func (cm *CgroupManager) readCPUStat() (int64, error) {
	cpuStatPath := filepath.Join(cm.CgroupPath, "cpu.stat")
	data, err := os.ReadFile(cpuStatPath)
	if err != nil {
		return 0, err
	}

	// Parse usage_usec from cpu.stat
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "usage_usec") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				usage, err := strconv.ParseInt(fields[1], 10, 64)
				if err == nil {
					return usage, nil
				}
			}
		}
	}

	return 0, nil
}

// readMemoryStat reads memory usage statistics
func (cm *CgroupManager) readMemoryStat() (int64, error) {
	memoryCurrentPath := filepath.Join(cm.CgroupPath, "memory.current")
	data, err := os.ReadFile(memoryCurrentPath)
	if err != nil {
		return 0, err
	}

	usage, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return 0, err
	}

	return usage, nil
}

// readPidsStat reads current number of processes
func (cm *CgroupManager) readPidsStat() (int, error) {
	pidsCurrentPath := filepath.Join(cm.CgroupPath, "pids.current")
	data, err := os.ReadFile(pidsCurrentPath)
	if err != nil {
		return 0, err
	}

	count, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, err
	}

	return count, nil
}

// Cleanup removes the cgroup
func (cm *CgroupManager) Cleanup() error {
	if err := os.RemoveAll(cm.CgroupPath); err != nil {
		return util.NewPathError("remove cgroup", cm.CgroupPath, err)
	}

	return nil
}

// ResourceStats holds resource usage statistics
type ResourceStats struct {
	CPUUsage    int64 // CPU usage in microseconds
	MemoryUsage int64 // Memory usage in bytes
	PidsCount   int   // Current number of processes
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}