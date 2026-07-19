package diagnostics

import (
	"errors"
	"strings"
	"testing"
)

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("read failed") }

func TestDecodeStrictJSON(t *testing.T) {
	t.Parallel()
	valid := `{"schema":"pasturestack.diagnostics-collection/v1","interval_seconds":60,"history_limit":1,"limits":{"max_bundle_bytes":65536,"max_probe_bytes":4096,"probe_timeout_seconds":1},"probes":[]}`
	if _, err := Decode(strings.NewReader(valid)); err != nil {
		t.Fatal(err)
	}
	for _, input := range []string{
		`{"schema":"one","schema":"two"}`,
		`{"unknown":true}`,
		valid + valid,
		`{"schema":`,
		`{"limits":{"max_bundle_bytes":1,"max_bundle_bytes":2}}`,
	} {
		if _, err := Decode(strings.NewReader(input)); err == nil {
			t.Fatalf("expected error for %q", input)
		}
	}
	if _, err := Decode(strings.NewReader(strings.Repeat(" ", MaxInputBytes+1))); err == nil {
		t.Fatal("expected size error")
	}
	if _, err := Decode(failingReader{}); err == nil {
		t.Fatal("expected read error")
	}
}
