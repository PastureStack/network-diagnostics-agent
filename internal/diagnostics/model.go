package diagnostics

const (
	InputSchema = "pasturestack.diagnostics-collection/v1"
	PlanSchema  = "pasturestack.diagnostics-plan/v1"

	MaxInputBytes    = 2 << 20
	MaxProbes        = 32
	MaxRetainedBytes = 512 << 20
)

type Config struct {
	Schema          string  `json:"schema"`
	IntervalSeconds int     `json:"interval_seconds"`
	HistoryLimit    int     `json:"history_limit"`
	Limits          Limits  `json:"limits"`
	Probes          []Probe `json:"probes"`
}

type Limits struct {
	MaxBundleBytes      int64 `json:"max_bundle_bytes"`
	MaxProbeBytes       int64 `json:"max_probe_bytes"`
	ProbeTimeoutSeconds int   `json:"probe_timeout_seconds"`
}

type Probe struct {
	Scope string `json:"scope"`
	Kind  string `json:"kind"`
}

type Plan struct {
	Schema           string      `json:"schema"`
	NormalizedSHA256 string      `json:"normalized_sha256"`
	Schedule         Schedule    `json:"schedule"`
	Limits           Limits      `json:"limits"`
	Probes           ProbeCounts `json:"probes"`
	Estimates        Estimates   `json:"estimates"`
	Safeguards       Safeguards  `json:"safeguards"`
}

type Schedule struct {
	IntervalSeconds int `json:"interval_seconds"`
	HistoryLimit    int `json:"history_limit"`
}

type ProbeCounts struct {
	Total           int `json:"total"`
	Host            int `json:"host"`
	DataPlane       int `json:"data_plane"`
	NameResolution  int `json:"name_resolution"`
	NetworkManager  int `json:"network_manager"`
	HighSensitivity int `json:"high_sensitivity"`
}

type Estimates struct {
	MaximumSnapshotBytes        int64 `json:"maximum_snapshot_bytes"`
	MaximumRetainedBytes        int64 `json:"maximum_retained_bytes"`
	MaximumSerialRuntimeSeconds int   `json:"maximum_serial_runtime_seconds"`
}

type Safeguards struct {
	Mode                     string `json:"mode"`
	RunsCommands             bool   `json:"runs_commands"`
	EntersNamespaces         bool   `json:"enters_namespaces"`
	ReadsHostFiles           bool   `json:"reads_host_files"`
	ReadsMetadata            bool   `json:"reads_metadata"`
	WritesBundles            bool   `json:"writes_bundles"`
	UploadsBundles           bool   `json:"uploads_bundles"`
	AcceptsTargetIdentifiers bool   `json:"accepts_target_identifiers"`
	AcceptsCommandArguments  bool   `json:"accepts_command_arguments"`
	ProductionReady          bool   `json:"production_ready"`
}
