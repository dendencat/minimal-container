# Alpine Bundle Example

This is a minimal Alpine Linux bundle for testing the gomini container runtime.

## Structure

- `config.json` - OCI runtime specification configuration
- `rootfs/` - Root filesystem directory (currently empty)

## Usage

To use this bundle, you need to populate the `rootfs/` directory with a minimal Alpine Linux filesystem. You can do this by:

1. Using Docker to extract an Alpine image:
   ```bash
   docker create --name temp alpine:latest
   docker export temp | tar -C rootfs/ -xf -
   docker rm temp
   ```

2. Or manually creating a minimal filesystem structure:
   ```bash
   mkdir -p rootfs/{bin,etc,lib,tmp,var,proc,sys,dev}
   # Add minimal binaries like busybox
   ```

## Configuration

The `config.json` file includes:
- Basic process configuration (sh shell)
- Standard mounts (proc, dev, sys)
- Resource limits (128MB memory, 10ms CPU quota, 64 processes)
- Default capabilities
- UTS, PID, IPC, and MOUNT namespaces