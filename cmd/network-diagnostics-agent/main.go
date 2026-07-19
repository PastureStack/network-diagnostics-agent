package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/PastureStack/network-diagnostics-agent/internal/collector"
	"github.com/PastureStack/network-diagnostics-agent/internal/diagnostics"
)

var version = "dev"

func main() { os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)) }

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) > 0 && args[0] == "run" {
		return runCollector(args[1:], stdout, stderr)
	}

	flags := flag.NewFlagSet("network-diagnostics-agent", flag.ContinueOnError)
	flags.SetOutput(stderr)
	filePath := flags.String("file", "", "read a collection-policy JSON file instead of standard input")
	showVersion := flags.Bool("version", false, "print the build version")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "unexpected positional arguments")
		return 2
	}
	if *showVersion {
		fmt.Fprintln(stdout, version)
		return 0
	}

	input := stdin
	var file *os.File
	if *filePath != "" {
		var err error
		file, err = os.Open(*filePath)
		if err != nil {
			fmt.Fprintln(stderr, "unable to open input file")
			return 1
		}
		defer file.Close()
		input = file
	}
	config, err := diagnostics.Decode(input)
	if err != nil {
		fmt.Fprintf(stderr, "invalid collection policy: %v\n", err)
		return 1
	}
	plan, err := diagnostics.BuildPlan(config)
	if err != nil {
		fmt.Fprintf(stderr, "invalid diagnostics plan: %v\n", err)
		return 1
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(plan); err != nil {
		fmt.Fprintln(stderr, "unable to encode plan")
		return 1
	}
	return 0
}

func runCollector(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("network-diagnostics-agent run", flag.ContinueOnError)
	flags.SetOutput(stderr)
	serviceURL := flags.String("service-url", envOrDefault("PASTURESTACK_DIAGNOSTICS_URL", "http://network-diagnostics-service:8080"), "diagnostics service URL")
	metadataURL := flags.String("metadata-url", envOrDefault("PASTURESTACK_METADATA_URL", "http://169.254.169.250/2016-07-29"), "metadata API base URL")
	intervalSeconds := flags.Int("interval-seconds", envIntOrDefault("PASTURESTACK_DIAGNOSTICS_INTERVAL_SECONDS", 600), "seconds between snapshots")
	requestTimeoutSeconds := flags.Int("request-timeout-seconds", envIntOrDefault("PASTURESTACK_DIAGNOSTICS_REQUEST_TIMEOUT_SECONDS", 15), "HTTP request timeout in seconds")
	healthAddress := flags.String("health-address", envOrDefault("PASTURESTACK_DIAGNOSTICS_HEALTH_ADDRESS", ":8090"), "health listener address")
	procRoot := flags.String("proc-root", envOrDefault("PASTURESTACK_DIAGNOSTICS_PROC_ROOT", "/host/proc"), "read-only host proc path")
	resolverPath := flags.String("resolver-path", envOrDefault("PASTURESTACK_DIAGNOSTICS_RESOLVER_PATH", "/host/etc/resolv.conf"), "read-only host resolver path")
	machineIDPath := flags.String("machine-id-path", envOrDefault("PASTURESTACK_DIAGNOSTICS_MACHINE_ID_PATH", "/host/etc/machine-id"), "read-only host machine identifier path")
	showVersion := flags.Bool("version", false, "print the build version")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "unexpected positional arguments")
		return 2
	}
	if *showVersion {
		fmt.Fprintln(stdout, version)
		return 0
	}

	config := collector.Config{
		ServiceURL:     *serviceURL,
		Token:          os.Getenv("PASTURESTACK_DIAGNOSTICS_TOKEN"),
		MetadataURL:    *metadataURL,
		HostID:         os.Getenv("PASTURESTACK_HOST_ID"),
		Interval:       time.Duration(*intervalSeconds) * time.Second,
		RequestTimeout: time.Duration(*requestTimeoutSeconds) * time.Second,
		HealthAddress:  *healthAddress,
		ProcRoot:       *procRoot,
		ResolverPath:   *resolverPath,
		MachineIDPath:  *machineIDPath,
		Version:        version,
	}
	if err := config.Validate(); err != nil {
		fmt.Fprintln(stderr, "invalid runtime configuration")
		return 1
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := collector.Run(ctx, config, stdout); err != nil {
		fmt.Fprintln(stderr, "diagnostics agent stopped unexpectedly")
		return 1
	}
	return 0
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func envIntOrDefault(name string, fallback int) int {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
