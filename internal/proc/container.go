package proc

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"gomini/internal/fs"
	"gomini/internal/ns"
	"gomini/internal/spec"
	"gomini/internal/util"
)

// ContainerProcess represents a container process configuration
type ContainerProcess struct {
	Config     *spec.Config
	BundleDir  string
	Hostname   string
	Args       []string
	Env        []string
	WorkingDir string
}

// NewContainerProcess creates a new container process configuration
func NewContainerProcess(config *spec.Config, bundleDir string) *ContainerProcess {
	return &ContainerProcess{
		Config:     config,
		BundleDir:  bundleDir,
		Args:       config.Process.Args,
		Env:        config.Process.Env,
		WorkingDir: config.Process.Cwd,
		Hostname:   config.Hostname,
	}
}

// OverrideArgs overrides the process arguments
func (cp *ContainerProcess) OverrideArgs(args []string) {
	if len(args) > 0 {
		cp.Args = args
	}
}

// OverrideHostname overrides the hostname
func (cp *ContainerProcess) OverrideHostname(hostname string) {
	if hostname != "" {
		cp.Hostname = hostname
	}
}

// Run executes the container process
func (cp *ContainerProcess) Run() error {
	// Create namespace configuration from spec
	var nsTypes []string
	for _, ns := range cp.Config.Linux.Namespaces {
		nsTypes = append(nsTypes, ns.Type)
	}
	nsConfig := ns.ConfigFromSpec(nsTypes)

	fmt.Printf("Creating namespaces: %s\n", nsConfig.String())

	// Fork process for namespace isolation
	if nsConfig.PID {
		// When using PID namespace, we need to fork and let the child become PID 1
		return cp.runWithPIDNamespace(nsConfig)
	} else {
		// No PID namespace, run directly with other namespaces
		return cp.runWithNamespaces(nsConfig)
	}
}

// runWithPIDNamespace handles execution with PID namespace
func (cp *ContainerProcess) runWithPIDNamespace(nsConfig *ns.NamespaceConfig) error {
	// Create a pipe for communication with child
	r, w, err := os.Pipe()
	if err != nil {
		return util.NewError("create pipe", err)
	}
	defer r.Close()

	// Fork process
	cmd := exec.Command("/proc/self/exe", "container-init")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.ExtraFiles = []*os.File{w}

	// Set namespace flags
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: nsConfig.CloneFlags(),
	}

	// Set environment variables for child
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("GOMINI_BUNDLE_DIR=%s", cp.BundleDir),
		fmt.Sprintf("GOMINI_HOSTNAME=%s", cp.Hostname),
		fmt.Sprintf("GOMINI_ARGS=%s", joinArgs(cp.Args)),
		fmt.Sprintf("GOMINI_WORKING_DIR=%s", cp.WorkingDir),
	)

	if err := cmd.Start(); err != nil {
		w.Close()
		return util.NewError("start container process", err)
	}

	w.Close()

	// Wait for child process
	if err := cmd.Wait(); err != nil {
		return util.NewError("wait for container process", err)
	}

	return nil
}

// runWithNamespaces handles execution with namespaces but no PID namespace
func (cp *ContainerProcess) runWithNamespaces(nsConfig *ns.NamespaceConfig) error {
	// Create namespaces
	if err := ns.CreateNamespaces(nsConfig); err != nil {
		return util.WrapError("create namespaces", err)
	}

	// Continue with container initialization
	return cp.initContainer()
}

// initContainer initializes the container environment and executes the process
func (cp *ContainerProcess) initContainer() error {
	// Set hostname if UTS namespace is enabled
	if cp.Hostname != "" {
		if err := ns.SetHostname(cp.Hostname); err != nil {
			return util.WrapError("set hostname", err)
		}
	}

	// Switch root filesystem
	rootfsPath := cp.Config.GetRootfsPath(cp.BundleDir)
	rootfsManager := fs.NewRootfsManager(rootfsPath, cp.Config.Root.Readonly)

	if err := rootfsManager.SwitchRoot(); err != nil {
		return util.WrapError("switch root", err)
	}

	// Create basic mounts
	if err := fs.CreateBasicMounts(); err != nil {
		return util.WrapError("create basic mounts", err)
	}

	// Change working directory
	if cp.WorkingDir != "" {
		if err := os.Chdir(cp.WorkingDir); err != nil {
			return util.NewPathError("change working directory", cp.WorkingDir, err)
		}
	}

	// Execute the target process
	return cp.execProcess()
}

// execProcess executes the final container process
func (cp *ContainerProcess) execProcess() error {
	if len(cp.Args) == 0 {
		return util.NewSimpleError("exec process", "no command specified")
	}

	// Prepare environment
	env := cp.Env
	if env == nil {
		env = []string{"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"}
	}

	// Execute the process using syscall.Exec to replace current process
	binary := cp.Args[0]
	args := cp.Args

	if err := syscall.Exec(binary, args, env); err != nil {
		return util.NewError("exec process", err)
	}

	// This should never be reached
	return nil
}

// HandleContainerInit handles the container initialization when called as "container-init"
func HandleContainerInit() error {
	// This function is called when the process is executed with "container-init" argument
	// It runs as PID 1 in the new PID namespace

	// Get configuration from environment variables
	bundleDir := os.Getenv("GOMINI_BUNDLE_DIR")
	hostname := os.Getenv("GOMINI_HOSTNAME")
	argsStr := os.Getenv("GOMINI_ARGS")
	workingDir := os.Getenv("GOMINI_WORKING_DIR")

	if bundleDir == "" {
		return util.NewSimpleError("container init", "GOMINI_BUNDLE_DIR not set")
	}

	// Load config
	config, err := spec.LoadConfig(bundleDir)
	if err != nil {
		return util.WrapError("load config in init", err)
	}

	// Create container process
	cp := NewContainerProcess(config, bundleDir)
	if hostname != "" {
		cp.OverrideHostname(hostname)
	}
	if argsStr != "" {
		cp.OverrideArgs(splitArgs(argsStr))
	}
	if workingDir != "" {
		cp.WorkingDir = workingDir
	}

	// Initialize container environment
	return cp.initContainer()
}

// joinArgs joins arguments into a single string
func joinArgs(args []string) string {
	if len(args) == 0 {
		return ""
	}

	result := args[0]
	for i := 1; i < len(args); i++ {
		result += " " + args[i]
	}
	return result
}

// splitArgs splits a string into arguments (simple space-based splitting)
func splitArgs(argsStr string) []string {
	if argsStr == "" {
		return nil
	}

	// Simple space-based splitting - in production, would need proper shell parsing
	var args []string
	current := ""
	for _, char := range argsStr {
		if char == ' ' {
			if current != "" {
				args = append(args, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}
	if current != "" {
		args = append(args, current)
	}

	return args
}