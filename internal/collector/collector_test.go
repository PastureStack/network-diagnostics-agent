package collector

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCollectAndDeliver(t *testing.T) {
	root := t.TempDir()
	procRoot := filepath.Join(root, "proc")
	if err := os.MkdirAll(filepath.Join(procRoot, "net"), 0o700); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(procRoot, "net", "dev"), "Inter-| Receive | Transmit\n face |bytes packets errs drop fifo frame compressed multicast|bytes packets errs drop fifo colls carrier compressed\n  lo: 10 1 0 0 0 0 0 0 10 1 0 0 0 0 0 0\neth0: 100 2 3 4 0 0 0 0 200 5 6 7 0 0 0 0\n")
	writeTestFile(t, filepath.Join(procRoot, "net", "route"), "Iface Destination Gateway Flags RefCnt Use Metric Mask MTU Window IRTT\neth0 00000000 00000000 0003 0 0 0 00000000 0 0 0\n")
	writeTestFile(t, filepath.Join(procRoot, "uptime"), "123.45 100.00\n")
	resolver := filepath.Join(root, "resolv.conf")
	writeTestFile(t, resolver, "nameserver 192.0.2.1\nsearch internal.example example\n")
	machineID := filepath.Join(root, "machine-id")
	writeTestFile(t, machineID, "fallback-machine-id\n")

	metadata := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/2016-07-29/self/host/uuid" {
			http.NotFound(writer, request)
			return
		}
		_, _ = writer.Write([]byte("host-reference-for-test"))
	}))
	defer metadata.Close()

	const token = "0123456789abcdef0123456789abcdef"
	var delivered Snapshot
	service := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer "+token {
			http.Error(writer, "unauthorized", http.StatusUnauthorized)
			return
		}
		if err := json.NewDecoder(request.Body).Decode(&delivered); err != nil {
			t.Fatal(err)
		}
		writer.WriteHeader(http.StatusAccepted)
	}))
	defer service.Close()

	config := Config{
		ServiceURL:     service.URL,
		Token:          token,
		MetadataURL:    metadata.URL + "/2016-07-29",
		Interval:       time.Minute,
		RequestTimeout: 5 * time.Second,
		HealthAddress:  "127.0.0.1:0",
		ProcRoot:       procRoot,
		ResolverPath:   resolver,
		MachineIDPath:  machineID,
		Version:        "v0.2.0",
	}
	snapshot, err := Collect(context.Background(), config, time.Date(2026, 7, 25, 1, 2, 3, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.AgentID == "" || strings.Contains(snapshot.AgentID, "host-reference") {
		t.Fatalf("identity was not pseudonymized: %q", snapshot.AgentID)
	}
	if snapshot.Metrics.InterfaceCount != 1 || snapshot.Metrics.ReceiveBytes != 100 || snapshot.Metrics.TransmitBytes != 200 {
		t.Fatalf("unexpected interface metrics: %+v", snapshot.Metrics)
	}
	if snapshot.Metrics.DefaultRoutes != 1 || snapshot.Metrics.NameServerCount != 1 || snapshot.Metrics.SearchListCount != 2 {
		t.Fatalf("unexpected network metrics: %+v", snapshot.Metrics)
	}
	if err := Deliver(context.Background(), config, snapshot); err != nil {
		t.Fatal(err)
	}
	if delivered.Schema != SnapshotSchema || delivered.AgentID != snapshot.AgentID {
		t.Fatalf("unexpected delivered snapshot: %+v", delivered)
	}
}

func TestConfigRejectsUnsafeValues(t *testing.T) {
	config := Config{
		ServiceURL:     "http://service:8080",
		Token:          "short",
		MetadataURL:    "http://169.254.169.250/2016-07-29",
		Interval:       time.Minute,
		RequestTimeout: time.Second,
		HealthAddress:  ":8090",
		ProcRoot:       "/host/proc",
		ResolverPath:   "/host/etc/resolv.conf",
		MachineIDPath:  "/host/etc/machine-id",
		Version:        "test",
	}
	if err := config.Validate(); err == nil {
		t.Fatal("expected short token to be rejected")
	}
	config.Token = "0123456789abcdef0123456789abcdef"
	config.ServiceURL = "http://user:password@service"
	if err := config.Validate(); err == nil {
		t.Fatal("expected URL user information to be rejected")
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
