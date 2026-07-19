package diagnostics

import (
	"errors"
	"fmt"
	"sort"
)

type probeSpec struct{ highSensitivity bool }

var probeSpecs = map[string]probeSpec{
	"interface-summary":  {},
	"route-summary":      {},
	"neighbor-summary":   {},
	"connection-summary": {highSensitivity: true},
	"firewall-summary":   {highSensitivity: true},
	"overlay-summary":    {highSensitivity: true},
	"resolver-summary":   {},
	"process-summary":    {highSensitivity: true},
	"storage-summary":    {},
	"runtime-summary":    {highSensitivity: true},
}

var allowedProbes = map[string]map[string]struct{}{
	"host":            allow("interface-summary", "route-summary", "neighbor-summary", "connection-summary", "firewall-summary", "overlay-summary", "resolver-summary", "process-summary", "storage-summary", "runtime-summary"),
	"data-plane":      allow("interface-summary", "route-summary", "neighbor-summary", "connection-summary", "firewall-summary", "overlay-summary", "resolver-summary"),
	"name-resolution": allow("interface-summary", "route-summary", "resolver-summary"),
	"network-manager": allow("resolver-summary"),
}

func allow(values ...string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}

type normalizedConfig struct {
	Schema          string  `json:"schema"`
	IntervalSeconds int     `json:"interval_seconds"`
	HistoryLimit    int     `json:"history_limit"`
	Limits          Limits  `json:"limits"`
	Probes          []Probe `json:"probes"`
}

func normalize(config Config) (normalizedConfig, error) {
	if config.Schema != InputSchema {
		return normalizedConfig{}, fmt.Errorf("schema must be %q", InputSchema)
	}
	if config.IntervalSeconds < 60 || config.IntervalSeconds > 86400 {
		return normalizedConfig{}, errors.New("interval_seconds must be between 60 and 86400")
	}
	if config.HistoryLimit < 1 || config.HistoryLimit > 1000 {
		return normalizedConfig{}, errors.New("history_limit must be between 1 and 1000")
	}
	if config.Limits.MaxBundleBytes < 64<<10 || config.Limits.MaxBundleBytes > 64<<20 {
		return normalizedConfig{}, errors.New("limits.max_bundle_bytes must be between 65536 and 67108864")
	}
	if config.Limits.MaxProbeBytes < 4<<10 || config.Limits.MaxProbeBytes > 8<<20 || config.Limits.MaxProbeBytes > config.Limits.MaxBundleBytes {
		return normalizedConfig{}, errors.New("limits.max_probe_bytes is outside the allowed range")
	}
	if config.Limits.ProbeTimeoutSeconds < 1 || config.Limits.ProbeTimeoutSeconds > 30 {
		return normalizedConfig{}, errors.New("limits.probe_timeout_seconds must be between 1 and 30")
	}
	if int64(config.HistoryLimit)*config.Limits.MaxBundleBytes > MaxRetainedBytes {
		return normalizedConfig{}, fmt.Errorf("retention may not exceed %d bytes", MaxRetainedBytes)
	}
	if len(config.Probes) < 1 || len(config.Probes) > MaxProbes {
		return normalizedConfig{}, fmt.Errorf("probes must contain between 1 and %d entries", MaxProbes)
	}

	probes := append([]Probe(nil), config.Probes...)
	seen := make(map[string]struct{}, len(probes))
	for index, probe := range probes {
		allowed, scopeExists := allowedProbes[probe.Scope]
		if !scopeExists {
			return normalizedConfig{}, fmt.Errorf("probes[%d].scope is invalid", index)
		}
		if _, kindExists := probeSpecs[probe.Kind]; !kindExists {
			return normalizedConfig{}, fmt.Errorf("probes[%d].kind is invalid", index)
		}
		if _, compatible := allowed[probe.Kind]; !compatible {
			return normalizedConfig{}, fmt.Errorf("probes[%d] uses an unsupported scope and kind combination", index)
		}
		key := probe.Scope + "\x00" + probe.Kind
		if _, exists := seen[key]; exists {
			return normalizedConfig{}, fmt.Errorf("probes[%d] is duplicated", index)
		}
		seen[key] = struct{}{}
	}
	sort.Slice(probes, func(i, j int) bool {
		if probes[i].Scope == probes[j].Scope {
			return probes[i].Kind < probes[j].Kind
		}
		return probes[i].Scope < probes[j].Scope
	})
	return normalizedConfig{Schema: config.Schema, IntervalSeconds: config.IntervalSeconds, HistoryLimit: config.HistoryLimit, Limits: config.Limits, Probes: probes}, nil
}
