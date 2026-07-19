package diagnostics

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildPlanCountsEstimatesAndPrivacy(t *testing.T) {
	t.Parallel()
	plan, err := BuildPlan(validConfig())
	if err != nil {
		t.Fatal(err)
	}
	if plan.Schema != PlanSchema || len(plan.NormalizedSHA256) != 64 {
		t.Fatalf("unexpected header: %#v", plan)
	}
	wantCounts := ProbeCounts{Total: 6, Host: 3, DataPlane: 1, NameResolution: 1, NetworkManager: 1, HighSensitivity: 2}
	if plan.Probes != wantCounts {
		t.Fatalf("counts=%#v want=%#v", plan.Probes, wantCounts)
	}
	wantEstimates := Estimates{MaximumSnapshotBytes: 6 << 20, MaximumRetainedBytes: 72 << 20, MaximumSerialRuntimeSeconds: 60}
	if plan.Estimates != wantEstimates {
		t.Fatalf("estimates=%#v want=%#v", plan.Estimates, wantEstimates)
	}
	if plan.Safeguards.Mode != "audit-only" || plan.Safeguards.RunsCommands || plan.Safeguards.EntersNamespaces || plan.Safeguards.ReadsHostFiles || plan.Safeguards.ReadsMetadata || plan.Safeguards.WritesBundles || plan.Safeguards.UploadsBundles || plan.Safeguards.AcceptsTargetIdentifiers || plan.Safeguards.AcceptsCommandArguments || plan.Safeguards.ProductionReady {
		t.Fatalf("unexpected safeguards: %#v", plan.Safeguards)
	}
	encoded, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"interface-summary", "overlay-summary", "resolver-summary"} {
		if strings.Contains(string(encoded), value) {
			t.Fatalf("output leaked %q", value)
		}
	}
}

func TestBuildPlanDeterministicAndBundleCapped(t *testing.T) {
	t.Parallel()
	first := validConfig()
	second := validConfig()
	for left, right := 0, len(second.Probes)-1; left < right; left, right = left+1, right-1 {
		second.Probes[left], second.Probes[right] = second.Probes[right], second.Probes[left]
	}
	left, err := BuildPlan(first)
	if err != nil {
		t.Fatal(err)
	}
	right, err := BuildPlan(second)
	if err != nil {
		t.Fatal(err)
	}
	if left.NormalizedSHA256 != right.NormalizedSHA256 {
		t.Fatalf("digests differ: %s != %s", left.NormalizedSHA256, right.NormalizedSHA256)
	}
	first.Limits.MaxBundleBytes = 2 << 20
	first.Limits.MaxProbeBytes = 1 << 20
	plan, err := BuildPlan(first)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Estimates.MaximumSnapshotBytes != 2<<20 {
		t.Fatalf("snapshot cap=%d", plan.Estimates.MaximumSnapshotBytes)
	}
}
