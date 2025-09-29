# GoMini - Minimal Container Runtime

[![Go Version](https://img.shields.io/badge/go-1.22+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-Apache%202.0-green.svg)](LICENSE)
[![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg)](https://github.com/dendencat/minimal-container/actions)

A **minimal container runtime** implementation in Go, designed for **educational purposes** to understand container internals. GoMini provides core container isolation features including namespaces, filesystem separation, and process management.

## Project Overview

GoMini is a simplified container runtime similar to `runc`, built to demonstrate how containers work under the hood. It implements essential container technologies step by step, prioritizing **readability** and **educational value** over production features.

### Key Features

- **Namespace Isolation**: UTS, PID, MOUNT, and IPC namespaces
- **Filesystem Isolation**: Root filesystem switching with pivot_root/chroot fallback
- **Process Management**: Container process forking and execution
- **OCI Compatibility**: Bundle-based configuration (subset of OCI runtime spec)
- **CLI Interface**: Simple command-line interface similar to runc
- **Educational Focus**: Clean, well-documented code for learning

### Learning Objectives

This project helps you understand:
- How container namespaces provide isolation
- Filesystem isolation techniques (pivot_root, chroot)
- Process management in containerized environments
- OCI runtime specification basics
- Linux system calls for container operations

## Quick Start

### Prerequisites

- **Linux** (kernel 4.15+ recommended for full namespace support)
- **Go 1.22+**
- **Root privileges** (required for namespace operations)
- **Git** for cloning the repository

### Installation

1. **Clone the repository**:
   ```bash
   git clone https://github.com/dendencat/minimal-container.git
   cd minimal-container
   ```

2. **Build the binary**:
   ```bash
   make build
   ```

3. **Verify installation**:
   ```bash
   ./bin/gomini version
   ```

### Basic Usage

1. **Create a simple container**:
   ```bash
   sudo ./bin/gomini run --bundle ./examples/simple-test --hostname mycontainer
   ```

2. **Run with custom command**:
   ```bash
   sudo ./bin/gomini run --bundle ./examples/simple-test --cmd /bin/echo --verbose
   ```

3. **Override with command arguments**:
   ```bash
   sudo ./bin/gomini run --bundle ./examples/simple-test -- /bin/sh -c 'echo "Hello from container!"'
   ```

## Usage Guide

### Command Line Interface

```bash
gomini - minimal container runtime

Usage:
  gomini run [options] -- [command]
  gomini version
  gomini help

Commands:
  run     Run a container from a bundle
  version Show version information
  help    Show this help message

Options for 'run':
  --bundle DIR     Bundle directory path (default: current directory)
  --hostname NAME  Set container hostname
  --cpu QUOTA      CPU quota in microseconds per 100ms period
  --mem BYTES      Memory limit in bytes
  --pids COUNT     Maximum number of processes
  --net MODE       Network mode (none, host) [default: none]
  --cmd COMMAND    Override command to run
  --verbose        Enable verbose output
```

### Examples

#### Basic Container Execution
```bash
# Run container with default configuration
sudo ./bin/gomini run --bundle ./examples/simple-test

# Run with custom hostname
sudo ./bin/gomini run --bundle ./examples/simple-test --hostname webserver

# Run with verbose output for debugging
sudo ./bin/gomini run --bundle ./examples/simple-test --verbose
```

#### Command Overrides
```bash
# Override command with --cmd flag
sudo ./bin/gomini run --bundle ./examples/simple-test --cmd /bin/ls

# Override with positional arguments (highest priority)
sudo ./bin/gomini run --bundle ./examples/simple-test -- /bin/sh -c 'ls -la'

# Run interactive shell
sudo ./bin/gomini run --bundle ./examples/simple-test -- /bin/sh
```

#### Resource Limits (Future: M2)
```bash
# Set memory limit to 128MB
sudo ./bin/gomini run --bundle ./examples/simple-test --mem 134217728

# Set CPU quota (10ms per 100ms period = 10% CPU)
sudo ./bin/gomini run --bundle ./examples/simple-test --cpu 10000

# Limit number of processes
sudo ./bin/gomini run --bundle ./examples/simple-test --pids 64
```

## Configuration

### Bundle Structure

GoMini uses OCI-compatible bundles with the following structure:

```
my-bundle/
├── config.json          # Container configuration
└── rootfs/              # Root filesystem
    ├── bin/
    ├── etc/
    ├── lib/
    └── ...
```

### Configuration File (config.json)

The `config.json` file defines container settings. Here's a minimal example:

```json
{
    "ociVersion": "1.0.2",
    "process": {
        "terminal": false,
        "user": {
            "uid": 0,
            "gid": 0
        },
        "args": ["/bin/sh"],
        "env": [
            "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
            "TERM=xterm"
        ],
        "cwd": "/"
    },
    "root": {
        "path": "rootfs",
        "readonly": false
    },
    "hostname": "gomini-container",
    "linux": {
        "namespaces": [
            {"type": "pid"},
            {"type": "uts"},
            {"type": "mount"},
            {"type": "ipc"}
        ]
    }
}
```

### Configuration Options

#### Process Configuration
- **`args`**: Command and arguments to execute
- **`env`**: Environment variables
- **`cwd`**: Working directory
- **`user`**: User ID and group ID

#### Root Filesystem
- **`path`**: Path to rootfs directory (relative to bundle)
- **`readonly`**: Whether filesystem should be read-only

#### Namespaces
Supported namespace types:
- **`pid`**: Process ID isolation
- **`uts`**: Hostname and domain name isolation
- **`mount`**: Filesystem mount isolation
- **`ipc`**: Inter-process communication isolation

#### Mounts (Advanced)
```json
"mounts": [
    {
        "destination": "/proc",
        "type": "proc",
        "source": "proc"
    },
    {
        "destination": "/dev",
        "type": "tmpfs",
        "source": "tmpfs",
        "options": ["nosuid", "strictatime", "mode=755"]
    }
]
```

## Development

### Building from Source

```bash
# Install dependencies
make deps

# Build binary
make build

# Run linting
make lint

# Run tests
make test

# Install locally
make install
```

### Creating Custom Bundles

1. **Create bundle directory**:
   ```bash
   mkdir my-bundle
   cd my-bundle
   ```

2. **Set up rootfs**:
   ```bash
   mkdir rootfs
   # Option 1: Use Docker to extract rootfs
   docker create --name temp alpine:latest
   docker export temp | tar -C rootfs/ -xf -
   docker rm temp

   # Option 2: Use busybox for minimal setup
   mkdir -p rootfs/{bin,etc,lib,tmp,var,proc,sys,dev}
   cp /usr/bin/busybox rootfs/bin/
   cd rootfs/bin && ./busybox --install .
   ```

3. **Create config.json**:
   ```bash
   cat > config.json << 'EOF'
   {
       "ociVersion": "1.0.2",
       "process": {
           "args": ["/bin/sh"],
           "env": ["PATH=/bin:/usr/bin"],
           "cwd": "/"
       },
       "root": {
           "path": "rootfs"
       },
       "linux": {
           "namespaces": [
               {"type": "pid"},
               {"type": "uts"},
               {"type": "mount"}
           ]
       }
   }
   EOF
   ```

4. **Test your bundle**:
   ```bash
   sudo gomini run --bundle . --verbose
   ```

### Project Structure

```
.
├── cmd/gomini/           # CLI entrypoint
├── internal/
│   ├── spec/             # Config & bundle parsing
│   ├── fs/               # Filesystem isolation
│   ├── ns/               # Namespace management
│   ├── proc/             # Process management
│   └── util/             # Utilities
├── examples/             # Example bundles
├── _docs/                # Project documentation
├── Makefile              # Build configuration
└── README.md
```

## Security Considerations

### Root Privileges Required

GoMini requires root privileges for:
- Creating namespaces (`unshare` system call)
- Mounting filesystems (`mount` system call)
- Changing root directory (`pivot_root`/`chroot`)

### Current Security Status

**Implemented**:
- Namespace isolation (PID, UTS, MOUNT, IPC)
- Filesystem isolation
- Process isolation

**Planned** (Future milestones):
- Resource limits (cgroups v2) - M2
- Capability dropping - M3
- Seccomp filtering - M5

### Safe Usage

1. **Test Environment**: Use in isolated test environments only
2. **Rootfs Validation**: Ensure rootfs doesn't contain malicious binaries
3. **Resource Limits**: Plan to implement cgroups for production use
4. **Network Isolation**: Currently defaults to "none" network mode

## Testing

### Manual Testing

```bash
# Test basic functionality
sudo ./bin/gomini run --bundle ./examples/simple-test --verbose

# Test namespace isolation
sudo ./bin/gomini run --bundle ./examples/simple-test --hostname isolated-test

# Test command override
sudo ./bin/gomini run --bundle ./examples/simple-test --cmd /bin/pwd
```

### Expected Behaviors

- **Permission Errors**: Normal without root privileges
- **Mount Failures**: Expected in some test environments
- **Pivot Root Fallback**: Chroot fallback should trigger automatically

## Roadmap

### Completed (M0, M1)
- [x] CLI framework and argument parsing
- [x] OCI bundle configuration parsing
- [x] Namespace creation and management
- [x] Filesystem isolation with pivot_root/chroot
- [x] Process forking and execution

### In Progress (M2)
- [ ] cgroups v2 integration
- [ ] CPU, memory, and PID limits
- [ ] Resource monitoring

### Planned (M3-M5)
- [ ] Capability management and dropping
- [ ] Seccomp filtering
- [ ] Advanced networking
- [ ] Container lifecycle management

## Contributing

This is an educational project. Contributions that improve clarity, add documentation, or fix bugs are welcome!

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## Learning Resources

- [OCI Runtime Specification](https://github.com/opencontainers/runtime-spec)
- [Linux Namespaces](https://man7.org/linux/man-pages/man7/namespaces.7.html)
- [Container Internals](https://container.training/)
- [cgroups Documentation](https://docs.kernel.org/admin-guide/cgroup-v2.html)

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## Disclaimer

GoMini is designed for **educational purposes only**. It is not intended for production use and lacks many security features required for production container runtimes. Use in isolated environments only.

---

**Built for learning container internals**