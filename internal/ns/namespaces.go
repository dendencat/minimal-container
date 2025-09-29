package ns

import (
	"fmt"
	"os"
	"syscall"

	"golang.org/x/sys/unix"
	"gomini/internal/util"
)

// NamespaceType represents the type of namespace
type NamespaceType string

const (
	UTS   NamespaceType = "uts"
	PID   NamespaceType = "pid"
	MOUNT NamespaceType = "mount"
	IPC   NamespaceType = "ipc"
	NET   NamespaceType = "net"
	USER  NamespaceType = "user"
)

// NamespaceConfig holds configuration for namespace creation
type NamespaceConfig struct {
	UTS   bool
	PID   bool
	Mount bool
	IPC   bool
	Net   bool
	User  bool
}

// CloneFlags converts namespace configuration to clone flags
func (nc *NamespaceConfig) CloneFlags() uintptr {
	var flags uintptr

	if nc.UTS {
		flags |= unix.CLONE_NEWUTS
	}
	if nc.PID {
		flags |= unix.CLONE_NEWPID
	}
	if nc.Mount {
		flags |= unix.CLONE_NEWNS
	}
	if nc.IPC {
		flags |= unix.CLONE_NEWIPC
	}
	if nc.Net {
		flags |= unix.CLONE_NEWNET
	}
	if nc.User {
		flags |= unix.CLONE_NEWUSER
	}

	return flags
}

// String returns a human-readable description of enabled namespaces
func (nc *NamespaceConfig) String() string {
	var enabled []string

	if nc.UTS {
		enabled = append(enabled, "UTS")
	}
	if nc.PID {
		enabled = append(enabled, "PID")
	}
	if nc.Mount {
		enabled = append(enabled, "MOUNT")
	}
	if nc.IPC {
		enabled = append(enabled, "IPC")
	}
	if nc.Net {
		enabled = append(enabled, "NET")
	}
	if nc.User {
		enabled = append(enabled, "USER")
	}

	return fmt.Sprintf("Namespaces: %v", enabled)
}

// CreateNamespaces creates the specified namespaces using unshare
func CreateNamespaces(config *NamespaceConfig) error {
	flags := config.CloneFlags()

	if flags == 0 {
		return nil // No namespaces to create
	}

	if err := unix.Unshare(int(flags)); err != nil {
		return util.NewError("create namespaces", err)
	}

	return nil
}

// SetHostname sets the hostname in the UTS namespace
func SetHostname(hostname string) error {
	if hostname == "" {
		return nil
	}

	if err := unix.Sethostname([]byte(hostname)); err != nil {
		return util.NewError("set hostname", err)
	}

	return nil
}

// NamespaceFromSpec converts OCI spec namespace type to our enum
func NamespaceFromSpec(specType string) NamespaceType {
	switch specType {
	case "uts":
		return UTS
	case "pid":
		return PID
	case "mount":
		return MOUNT
	case "ipc":
		return IPC
	case "network":
		return NET
	case "user":
		return USER
	default:
		return ""
	}
}

// ConfigFromSpec creates NamespaceConfig from OCI spec namespaces
func ConfigFromSpec(namespaces []string) *NamespaceConfig {
	config := &NamespaceConfig{}

	for _, nsType := range namespaces {
		switch NamespaceFromSpec(nsType) {
		case UTS:
			config.UTS = true
		case PID:
			config.PID = true
		case MOUNT:
			config.Mount = true
		case IPC:
			config.IPC = true
		case NET:
			config.Net = true
		case USER:
			config.User = true
		}
	}

	return config
}

// IsNamespaced checks if we're running in a namespace
func IsNamespaced() bool {
	// Check if we're PID 1 (indicates we're in a PID namespace)
	return os.Getpid() == 1
}

// WaitForChild waits for a child process and returns its exit status
func WaitForChild(pid int) (int, error) {
	var status syscall.WaitStatus

	_, err := syscall.Wait4(pid, &status, 0, nil)
	if err != nil {
		return -1, util.NewError("wait for child", err)
	}

	if status.Exited() {
		return status.ExitStatus(), nil
	}

	if status.Signaled() {
		return 128 + int(status.Signal()), nil
	}

	return -1, util.NewSimpleError("wait for child", "unexpected child status")
}