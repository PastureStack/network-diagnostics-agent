package diagnostics

func validConfig() Config {
	return Config{
		Schema:          InputSchema,
		IntervalSeconds: 600,
		HistoryLimit:    12,
		Limits:          Limits{MaxBundleBytes: 16 << 20, MaxProbeBytes: 1 << 20, ProbeTimeoutSeconds: 10},
		Probes: []Probe{
			{Scope: "host", Kind: "interface-summary"},
			{Scope: "host", Kind: "route-summary"},
			{Scope: "host", Kind: "process-summary"},
			{Scope: "data-plane", Kind: "overlay-summary"},
			{Scope: "name-resolution", Kind: "resolver-summary"},
			{Scope: "network-manager", Kind: "resolver-summary"},
		},
	}
}
