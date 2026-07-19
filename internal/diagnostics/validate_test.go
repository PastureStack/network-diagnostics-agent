package diagnostics

import "testing"

func TestValidationRejectsInvalidPolicies(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		edit func(*Config)
	}{
		{"schema", func(c *Config) { c.Schema = "wrong" }},
		{"interval low", func(c *Config) { c.IntervalSeconds = 59 }},
		{"interval high", func(c *Config) { c.IntervalSeconds = 86401 }},
		{"history low", func(c *Config) { c.HistoryLimit = 0 }},
		{"history high", func(c *Config) { c.HistoryLimit = 1001 }},
		{"bundle low", func(c *Config) { c.Limits.MaxBundleBytes = (64 << 10) - 1 }},
		{"bundle high", func(c *Config) { c.Limits.MaxBundleBytes = (64 << 20) + 1 }},
		{"probe low", func(c *Config) { c.Limits.MaxProbeBytes = (4 << 10) - 1 }},
		{"probe high", func(c *Config) { c.Limits.MaxProbeBytes = (8 << 20) + 1 }},
		{"probe above bundle", func(c *Config) { c.Limits.MaxBundleBytes = 64 << 10; c.Limits.MaxProbeBytes = 128 << 10 }},
		{"timeout low", func(c *Config) { c.Limits.ProbeTimeoutSeconds = 0 }},
		{"timeout high", func(c *Config) { c.Limits.ProbeTimeoutSeconds = 31 }},
		{"retention", func(c *Config) { c.HistoryLimit = 9; c.Limits.MaxBundleBytes = 64 << 20 }},
		{"no probes", func(c *Config) { c.Probes = nil }},
		{"too many probes", func(c *Config) { c.Probes = make([]Probe, MaxProbes+1) }},
		{"scope", func(c *Config) { c.Probes[0].Scope = "workload-a" }},
		{"kind", func(c *Config) { c.Probes[0].Kind = "arbitrary-command" }},
		{"combination", func(c *Config) { c.Probes[0] = Probe{Scope: "network-manager", Kind: "interface-summary"} }},
		{"duplicate", func(c *Config) { c.Probes = append(c.Probes, c.Probes[0]) }},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			config := validConfig()
			test.edit(&config)
			if _, err := BuildPlan(config); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

func TestValidationBoundaries(t *testing.T) {
	t.Parallel()
	config := validConfig()
	config.IntervalSeconds = 60
	config.HistoryLimit = 1
	config.Limits = Limits{MaxBundleBytes: 64 << 10, MaxProbeBytes: 4 << 10, ProbeTimeoutSeconds: 1}
	config.Probes = []Probe{{Scope: "data-plane", Kind: "firewall-summary"}}
	if _, err := BuildPlan(config); err != nil {
		t.Fatal(err)
	}
	config.IntervalSeconds = 86400
	config.HistoryLimit = 8
	config.Limits = Limits{MaxBundleBytes: 64 << 20, MaxProbeBytes: 8 << 20, ProbeTimeoutSeconds: 30}
	if _, err := BuildPlan(config); err != nil {
		t.Fatal(err)
	}
}
