package collector

import (
	"errors"
	"net/url"
	"time"
	"unicode"
)

const SnapshotSchema = "pasturestack.network-diagnostics-snapshot/v1"

type Config struct {
	ServiceURL     string
	Token          string
	MetadataURL    string
	HostID         string
	Interval       time.Duration
	RequestTimeout time.Duration
	HealthAddress  string
	ProcRoot       string
	ResolverPath   string
	MachineIDPath  string
	Version        string
}

type Snapshot struct {
	Schema     string    `json:"schema"`
	AgentID    string    `json:"agent_id"`
	CapturedAt time.Time `json:"captured_at"`
	Version    string    `json:"version"`
	Metrics    Metrics   `json:"metrics"`
}

type Metrics struct {
	InterfaceCount   uint64 `json:"interface_count"`
	ReceiveBytes     uint64 `json:"receive_bytes"`
	ReceiveErrors    uint64 `json:"receive_errors"`
	ReceiveDrops     uint64 `json:"receive_drops"`
	TransmitBytes    uint64 `json:"transmit_bytes"`
	TransmitErrors   uint64 `json:"transmit_errors"`
	TransmitDrops    uint64 `json:"transmit_drops"`
	RouteCount       uint64 `json:"route_count"`
	DefaultRoutes    uint64 `json:"default_route_count"`
	NameServerCount  uint64 `json:"name_server_count"`
	SearchListCount  uint64 `json:"search_list_count"`
	UptimeSeconds    uint64 `json:"uptime_seconds"`
	CollectionErrors uint64 `json:"collection_error_count"`
}

func (c Config) Validate() error {
	if !validToken(c.Token) {
		return errors.New("token must contain 32 through 256 printable non-space ASCII characters")
	}
	if err := validateHTTPURL(c.ServiceURL); err != nil {
		return err
	}
	if err := validateHTTPURL(c.MetadataURL); err != nil {
		return err
	}
	if c.Interval < time.Minute || c.Interval > 24*time.Hour {
		return errors.New("interval is outside the supported range")
	}
	if c.RequestTimeout < time.Second || c.RequestTimeout > time.Minute {
		return errors.New("request timeout is outside the supported range")
	}
	if c.HealthAddress == "" || c.ProcRoot == "" || c.ResolverPath == "" || c.MachineIDPath == "" {
		return errors.New("required runtime path or listener is missing")
	}
	if len(c.Version) == 0 || len(c.Version) > 64 {
		return errors.New("build version is invalid")
	}
	return nil
}

func validToken(token string) bool {
	if len(token) < 32 || len(token) > 256 {
		return false
	}
	for _, character := range token {
		if character > unicode.MaxASCII || character <= ' ' || character == 0x7f {
			return false
		}
	}
	return true
}

func validateHTTPURL(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return errors.New("URL must use HTTP or HTTPS and include a host")
	}
	if parsed.User != nil || parsed.Fragment != "" {
		return errors.New("URL must not contain user information or a fragment")
	}
	return nil
}
