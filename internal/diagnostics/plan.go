package diagnostics

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

func BuildPlan(config Config) (Plan, error) {
	normalized, err := normalize(config)
	if err != nil {
		return Plan{}, err
	}
	encoded, err := json.Marshal(normalized)
	if err != nil {
		return Plan{}, fmt.Errorf("encode normalized collection policy: %w", err)
	}
	digest := sha256.Sum256(encoded)

	counts := ProbeCounts{Total: len(normalized.Probes)}
	for _, probe := range normalized.Probes {
		switch probe.Scope {
		case "host":
			counts.Host++
		case "data-plane":
			counts.DataPlane++
		case "name-resolution":
			counts.NameResolution++
		case "network-manager":
			counts.NetworkManager++
		}
		if probeSpecs[probe.Kind].highSensitivity {
			counts.HighSensitivity++
		}
	}
	maximumSnapshot := int64(len(normalized.Probes)) * normalized.Limits.MaxProbeBytes
	if maximumSnapshot > normalized.Limits.MaxBundleBytes {
		maximumSnapshot = normalized.Limits.MaxBundleBytes
	}
	return Plan{
		Schema:           PlanSchema,
		NormalizedSHA256: hex.EncodeToString(digest[:]),
		Schedule:         Schedule{IntervalSeconds: normalized.IntervalSeconds, HistoryLimit: normalized.HistoryLimit},
		Limits:           normalized.Limits,
		Probes:           counts,
		Estimates: Estimates{
			MaximumSnapshotBytes:        maximumSnapshot,
			MaximumRetainedBytes:        maximumSnapshot * int64(normalized.HistoryLimit),
			MaximumSerialRuntimeSeconds: len(normalized.Probes) * normalized.Limits.ProbeTimeoutSeconds,
		},
		Safeguards: Safeguards{
			Mode: "audit-only", RunsCommands: false, EntersNamespaces: false, ReadsHostFiles: false,
			ReadsMetadata: false, WritesBundles: false, UploadsBundles: false,
			AcceptsTargetIdentifiers: false, AcceptsCommandArguments: false, ProductionReady: false,
		},
	}, nil
}
