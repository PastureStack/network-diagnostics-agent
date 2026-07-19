package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validDocument = `{
  "schema":"pasturestack.diagnostics-collection/v1",
  "interval_seconds":600,
  "history_limit":12,
  "limits":{"max_bundle_bytes":16777216,"max_probe_bytes":1048576,"probe_timeout_seconds":10},
  "probes":[{"scope":"host","kind":"interface-summary"}]
}`

type errorWriter struct{}

func (errorWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }

func TestRunFromStandardInput(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run(nil, strings.NewReader(validDocument), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"schema": "pasturestack.diagnostics-plan/v1"`) {
		t.Fatalf("unexpected output: %s", stdout.String())
	}
}

func TestRunFromFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "policy.json")
	if err := os.WriteFile(path, []byte(validDocument), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := run([]string{"--file", path}, strings.NewReader(""), &stdout, &stderr); code != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
}

func TestRunVersionAndArguments(t *testing.T) {
	oldVersion := version
	version = "test-version"
	t.Cleanup(func() { version = oldVersion })
	var stdout, stderr bytes.Buffer
	if code := run([]string{"--version"}, strings.NewReader(""), &stdout, &stderr); code != 0 || strings.TrimSpace(stdout.String()) != "test-version" {
		t.Fatalf("code=%d output=%q", code, stdout.String())
	}
	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"extra"}, strings.NewReader(""), &stdout, &stderr); code != 2 {
		t.Fatalf("positional code=%d", code)
	}
	stdout.Reset()
	stderr.Reset()
	if code := run([]string{"--unknown"}, strings.NewReader(""), &stdout, &stderr); code != 2 {
		t.Fatalf("flag code=%d", code)
	}
}

func TestRunErrors(t *testing.T) {
	for _, test := range []struct {
		name  string
		args  []string
		input string
	}{
		{name: "missing file", args: []string{"--file", filepath.Join(t.TempDir(), "missing.json")}},
		{name: "decode", input: "{"},
		{name: "validation", input: `{"schema":"pasturestack.diagnostics-collection/v1","interval_seconds":1,"history_limit":1,"limits":{"max_bundle_bytes":65536,"max_probe_bytes":4096,"probe_timeout_seconds":1},"probes":[]}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := run(test.args, strings.NewReader(test.input), &stdout, &stderr); code != 1 {
				t.Fatalf("code=%d stderr=%s", code, stderr.String())
			}
		})
	}
	var stderr bytes.Buffer
	if code := run(nil, strings.NewReader(validDocument), errorWriter{}, &stderr); code != 1 {
		t.Fatalf("writer code=%d", code)
	}
}
