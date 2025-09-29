package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/unix"
	"gomini/internal/util"
)

// RootfsManager handles root filesystem operations
type RootfsManager struct {
	RootfsPath string
	Readonly   bool
}

// NewRootfsManager creates a new rootfs manager
func NewRootfsManager(rootfsPath string, readonly bool) *RootfsManager {
	return &RootfsManager{
		RootfsPath: rootfsPath,
		Readonly:   readonly,
	}
}

// PrepareRootfs prepares the root filesystem for container use
func (rm *RootfsManager) PrepareRootfs() error {
	// Verify rootfs path exists
	if _, err := os.Stat(rm.RootfsPath); os.IsNotExist(err) {
		return util.NewPathError("prepare rootfs", rm.RootfsPath, fmt.Errorf("rootfs path does not exist"))
	}

	// Create old_root directory for pivot_root
	oldRootPath := filepath.Join(rm.RootfsPath, ".old_root")
	if err := os.MkdirAll(oldRootPath, 0755); err != nil {
		return util.NewPathError("create old_root", oldRootPath, err)
	}

	return nil
}

// PivotRoot performs pivot_root to switch to the new root filesystem
func (rm *RootfsManager) PivotRoot() error {
	oldRootPath := filepath.Join(rm.RootfsPath, ".old_root")

	// Attempt pivot_root
	if err := unix.PivotRoot(rm.RootfsPath, oldRootPath); err != nil {
		return util.NewError("pivot_root", err)
	}

	// Change to the new root directory
	if err := unix.Chdir("/"); err != nil {
		return util.NewError("chdir to new root", err)
	}

	// Unmount old root
	if err := unix.Unmount("/.old_root", unix.MNT_DETACH); err != nil {
		// Log warning but don't fail - cleanup can happen later
		fmt.Fprintf(os.Stderr, "Warning: failed to unmount old root: %v\n", err)
	}

	// Remove old root directory
	if err := os.RemoveAll("/.old_root"); err != nil {
		// Log warning but don't fail
		fmt.Fprintf(os.Stderr, "Warning: failed to remove old root directory: %v\n", err)
	}

	return nil
}

// ChrootFallback performs chroot as a fallback when pivot_root fails
func (rm *RootfsManager) ChrootFallback() error {
	// Change to rootfs directory
	if err := unix.Chdir(rm.RootfsPath); err != nil {
		return util.NewPathError("chdir to rootfs", rm.RootfsPath, err)
	}

	// Perform chroot
	if err := unix.Chroot("."); err != nil {
		return util.NewError("chroot", err)
	}

	// Change to root directory in new environment
	if err := unix.Chdir("/"); err != nil {
		return util.NewError("chdir to / after chroot", err)
	}

	return nil
}

// SwitchRoot switches to the new root filesystem using pivot_root with chroot fallback
func (rm *RootfsManager) SwitchRoot() error {
	// Prepare the rootfs first
	if err := rm.PrepareRootfs(); err != nil {
		return util.WrapError("prepare rootfs", err)
	}

	// Try pivot_root first
	if err := rm.PivotRoot(); err != nil {
		fmt.Fprintf(os.Stderr, "pivot_root failed (%v), falling back to chroot\n", err)

		// Fall back to chroot
		if chrootErr := rm.ChrootFallback(); chrootErr != nil {
			return util.WrapError("chroot fallback", chrootErr)
		}
	}

	return nil
}

// MountPoint represents a mount configuration
type MountPoint struct {
	Source      string
	Destination string
	Type        string
	Options     []string
	Flags       uintptr
}

// CreateBasicMounts creates essential mounts for the container
func CreateBasicMounts() error {
	mounts := []MountPoint{
		{
			Source:      "proc",
			Destination: "/proc",
			Type:        "proc",
			Flags:       0,
		},
		{
			Source:      "tmpfs",
			Destination: "/dev",
			Type:        "tmpfs",
			Options:     []string{"nosuid", "strictatime", "mode=755", "size=65536k"},
			Flags:       syscall.MS_NOSUID,
		},
		{
			Source:      "devpts",
			Destination: "/dev/pts",
			Type:        "devpts",
			Options:     []string{"nosuid", "noexec", "newinstance", "ptmxmode=0666", "mode=0620"},
			Flags:       syscall.MS_NOSUID | syscall.MS_NOEXEC,
		},
		{
			Source:      "tmpfs",
			Destination: "/dev/shm",
			Type:        "tmpfs",
			Options:     []string{"nosuid", "noexec", "nodev", "mode=1777", "size=65536k"},
			Flags:       syscall.MS_NOSUID | syscall.MS_NOEXEC | syscall.MS_NODEV,
		},
		{
			Source:      "sysfs",
			Destination: "/sys",
			Type:        "sysfs",
			Options:     []string{"nosuid", "noexec", "nodev", "ro"},
			Flags:       syscall.MS_NOSUID | syscall.MS_NOEXEC | syscall.MS_NODEV | syscall.MS_RDONLY,
		},
	}

	for _, mount := range mounts {
		if err := createMount(mount); err != nil {
			return util.WrapError(fmt.Sprintf("create mount %s", mount.Destination), err)
		}
	}

	return nil
}

// createMount creates a single mount point
func createMount(mount MountPoint) error {
	// Create destination directory if it doesn't exist
	if err := os.MkdirAll(mount.Destination, 0755); err != nil {
		return util.NewPathError("create mount point", mount.Destination, err)
	}

	// Prepare options string
	var data string
	if len(mount.Options) > 0 {
		data = joinOptions(mount.Options)
	}

	// Perform the mount
	if err := unix.Mount(mount.Source, mount.Destination, mount.Type, mount.Flags, data); err != nil {
		return util.NewError("mount", err)
	}

	return nil
}

// joinOptions joins mount options into a comma-separated string
func joinOptions(options []string) string {
	if len(options) == 0 {
		return ""
	}

	result := options[0]
	for i := 1; i < len(options); i++ {
		result += "," + options[i]
	}

	return result
}

// EnsureDirectory ensures a directory exists with the specified permissions
func EnsureDirectory(path string, mode os.FileMode) error {
	if err := os.MkdirAll(path, mode); err != nil {
		return util.NewPathError("ensure directory", path, err)
	}
	return nil
}