package collector

import (
	"bufio"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const maxIdentityBytes = 1024

func Collect(ctx context.Context, config Config, now time.Time) (Snapshot, error) {
	identity, err := loadIdentity(ctx, config)
	if err != nil {
		return Snapshot{}, err
	}

	metrics := Metrics{}
	if err := collectNetworkDeviceMetrics(filepath.Join(config.ProcRoot, "net", "dev"), &metrics); err != nil {
		metrics.CollectionErrors++
	}
	if err := collectRouteMetrics(filepath.Join(config.ProcRoot, "net", "route"), &metrics); err != nil {
		metrics.CollectionErrors++
	}
	if err := collectResolverMetrics(config.ResolverPath, &metrics); err != nil {
		metrics.CollectionErrors++
	}
	if err := collectUptime(filepath.Join(config.ProcRoot, "uptime"), &metrics); err != nil {
		metrics.CollectionErrors++
	}

	return Snapshot{
		Schema:     SnapshotSchema,
		AgentID:    pseudonymousAgentID(config.Token, identity),
		CapturedAt: now.UTC().Truncate(time.Second),
		Version:    config.Version,
		Metrics:    metrics,
	}, nil
}

func loadIdentity(ctx context.Context, config Config) (string, error) {
	if value := strings.TrimSpace(config.HostID); value != "" {
		return boundedIdentity(value)
	}

	metadataURL := strings.TrimRight(config.MetadataURL, "/") + "/self/host/uuid"
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, metadataURL, nil)
	if err == nil {
		client := &http.Client{Timeout: config.RequestTimeout}
		response, requestErr := client.Do(request)
		if requestErr == nil {
			defer response.Body.Close()
			if response.StatusCode == http.StatusOK {
				content, readErr := io.ReadAll(io.LimitReader(response.Body, maxIdentityBytes+1))
				if readErr == nil && len(content) <= maxIdentityBytes {
					if value := strings.TrimSpace(string(content)); value != "" {
						return boundedIdentity(value)
					}
				}
			}
		}
	}

	content, err := readBoundedFile(config.MachineIDPath, maxIdentityBytes)
	if err != nil {
		return "", errors.New("host identity is unavailable")
	}
	return boundedIdentity(strings.TrimSpace(string(content)))
}

func boundedIdentity(value string) (string, error) {
	if len(value) < 8 || len(value) > maxIdentityBytes || strings.ContainsAny(value, "\x00\r\n") {
		return "", errors.New("host identity is invalid")
	}
	return value, nil
}

func pseudonymousAgentID(token, identity string) string {
	mac := hmac.New(sha256.New, []byte(token))
	_, _ = mac.Write([]byte(identity))
	return hex.EncodeToString(mac.Sum(nil))
}

func collectNetworkDeviceMetrics(path string, metrics *Metrics) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(io.LimitReader(file, 2<<20))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		separator := strings.IndexByte(line, ':')
		if separator < 0 {
			continue
		}
		name := strings.TrimSpace(line[:separator])
		if name == "" || name == "lo" {
			continue
		}
		fields := strings.Fields(line[separator+1:])
		if len(fields) < 16 {
			continue
		}
		values := make([]uint64, 16)
		valid := true
		for index := range values {
			values[index], err = strconv.ParseUint(fields[index], 10, 64)
			if err != nil {
				valid = false
				break
			}
		}
		if !valid {
			continue
		}
		metrics.InterfaceCount++
		metrics.ReceiveBytes += values[0]
		metrics.ReceiveErrors += values[2]
		metrics.ReceiveDrops += values[3]
		metrics.TransmitBytes += values[8]
		metrics.TransmitErrors += values[10]
		metrics.TransmitDrops += values[11]
	}
	return scanner.Err()
}

func collectRouteMetrics(path string, metrics *Metrics) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(io.LimitReader(file, 2<<20))
	first := true
	for scanner.Scan() {
		if first {
			first = false
			continue
		}
		fields := strings.Fields(scanner.Text())
		if len(fields) < 4 {
			continue
		}
		metrics.RouteCount++
		if fields[1] == "00000000" {
			metrics.DefaultRoutes++
		}
	}
	return scanner.Err()
}

func collectResolverMetrics(path string, metrics *Metrics) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(io.LimitReader(file, 1<<20))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		switch fields[0] {
		case "nameserver":
			metrics.NameServerCount++
		case "search":
			metrics.SearchListCount += uint64(len(fields) - 1)
		}
	}
	return scanner.Err()
}

func collectUptime(path string, metrics *Metrics) error {
	content, err := readBoundedFile(path, 4096)
	if err != nil {
		return err
	}
	fields := strings.Fields(string(content))
	if len(fields) == 0 {
		return fmt.Errorf("uptime is empty")
	}
	value, err := strconv.ParseFloat(fields[0], 64)
	if err != nil || value < 0 {
		return fmt.Errorf("uptime is invalid")
	}
	metrics.UptimeSeconds = uint64(value)
	return nil
}

func readBoundedFile(path string, limit int64) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(content)) > limit {
		return nil, errors.New("file exceeds the accepted size")
	}
	return content, nil
}
