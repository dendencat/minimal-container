package main

import (
	"flag"
	"fmt"
	"os"

	"gomini/internal/spec"
)

const version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "run":
		runCommand(os.Args[2:])
	case "version", "--version", "-v":
		fmt.Printf("gomini version %s\n", version)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Printf(`gomini - minimal container runtime

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

Examples:
  gomini run --bundle ./examples/alpine-bundle --hostname mini1 --cpu 10000 --mem 134217728 --pids 64 --cmd /bin/sh
  gomini run --bundle ./examples/alpine-bundle --verbose -- /bin/sh -c 'echo hello'
`)
}

func runCommand(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)

	bundle := fs.String("bundle", ".", "Bundle directory path")
	hostname := fs.String("hostname", "", "Set container hostname")
	cpu := fs.Int64("cpu", 0, "CPU quota in microseconds per 100ms period")
	mem := fs.Int64("mem", 0, "Memory limit in bytes")
	pids := fs.Int("pids", 0, "Maximum number of processes")
	net := fs.String("net", "none", "Network mode (none, host)")
	cmd := fs.String("cmd", "", "Override command to run")
	verbose := fs.Bool("verbose", false, "Enable verbose output")

	fs.Parse(args)

	// Capture positional arguments after flags (for -- command syntax)
	positionalArgs := fs.Args()

	if *verbose {
		fmt.Printf("Run command called with:\n")
		fmt.Printf("  Bundle: %s\n", *bundle)
		fmt.Printf("  Hostname: %s\n", *hostname)
		fmt.Printf("  CPU: %d\n", *cpu)
		fmt.Printf("  Memory: %d\n", *mem)
		fmt.Printf("  PIDs: %d\n", *pids)
		fmt.Printf("  Network: %s\n", *net)
		fmt.Printf("  Command override: %s\n", *cmd)
		fmt.Printf("  Positional args: %v\n", positionalArgs)
	}

	// Load and validate bundle configuration
	config, err := spec.LoadConfig(*bundle)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Determine final command to execute
	var finalArgs []string
	if *cmd != "" {
		// --cmd flag overrides everything
		finalArgs = []string{*cmd}
	} else if len(positionalArgs) > 0 {
		// Positional args (after --) override bundle config
		finalArgs = positionalArgs
	} else {
		// Use bundle config as default
		finalArgs = config.Process.Args
	}

	if *verbose {
		fmt.Printf("\nLoaded config from bundle:\n")
		fmt.Printf("  OCI Version: %s\n", config.OCIVersion)
		fmt.Printf("  Bundle Process Args: %v\n", config.Process.Args)
		fmt.Printf("  Root Path: %s\n", config.Root.Path)
		fmt.Printf("  Hostname: %s\n", config.Hostname)
		fmt.Printf("  Namespaces: %d configured\n", len(config.Linux.Namespaces))
	}

	fmt.Printf("Final command to execute: %v\n", finalArgs)
	fmt.Println("Container runtime not yet implemented")
}