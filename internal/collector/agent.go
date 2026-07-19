package collector

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type healthState struct {
	sync.RWMutex
	startedAt   time.Time
	lastSuccess time.Time
	attempts    uint64
	failures    uint64
}

type healthResponse struct {
	Status               string    `json:"status"`
	StartedAt            time.Time `json:"started_at"`
	LastSuccessfulUpload time.Time `json:"last_successful_upload,omitempty"`
	UploadAttempts       uint64    `json:"upload_attempts"`
	UploadFailures       uint64    `json:"upload_failures"`
}

func Run(ctx context.Context, config Config, output io.Writer) error {
	if err := config.Validate(); err != nil {
		return err
	}

	state := &healthState{startedAt: time.Now().UTC()}
	healthServer := &http.Server{
		Addr:              config.HealthAddress,
		Handler:           healthHandler(state, config.Interval),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
		IdleTimeout:       30 * time.Second,
		MaxHeaderBytes:    8 << 10,
	}
	serverErrors := make(chan error, 1)
	go func() {
		err := healthServer.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			serverErrors <- err
		}
	}()

	defer func() {
		shutdownContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = healthServer.Shutdown(shutdownContext)
	}()

	deliver := func() {
		state.Lock()
		state.attempts++
		state.Unlock()

		snapshot, err := Collect(ctx, config, time.Now())
		if err == nil {
			err = Deliver(ctx, config, snapshot)
		}
		state.Lock()
		if err == nil {
			state.lastSuccess = time.Now().UTC()
		} else {
			state.failures++
		}
		state.Unlock()
		if err == nil {
			fmt.Fprintln(output, "diagnostics snapshot delivered")
		} else {
			fmt.Fprintln(output, "diagnostics snapshot delivery failed; retry scheduled")
		}
	}

	deliver()
	ticker := time.NewTicker(config.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-serverErrors:
			return err
		case <-ticker.C:
			deliver()
		}
	}
}

func Deliver(ctx context.Context, config Config, snapshot Snapshot) error {
	payload, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}
	if len(payload) > 64<<10 {
		return fmt.Errorf("snapshot exceeds the delivery limit")
	}

	base, err := url.Parse(config.ServiceURL)
	if err != nil {
		return err
	}
	base.Path = strings.TrimRight(base.Path, "/") + "/v1/snapshots"
	base.RawQuery = ""
	base.Fragment = ""

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, base.String(), bytes.NewReader(payload))
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+config.Token)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", "PastureStack-Network-Diagnostics-Agent/"+config.Version)

	client := &http.Client{Timeout: config.RequestTimeout}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4<<10))
	if response.StatusCode != http.StatusAccepted {
		return fmt.Errorf("service rejected the snapshot")
	}
	return nil
}

func healthHandler(state *healthState, interval time.Duration) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(writer http.ResponseWriter, _ *http.Request) {
		state.RLock()
		response := healthResponse{
			Status:               "starting",
			StartedAt:            state.startedAt,
			LastSuccessfulUpload: state.lastSuccess,
			UploadAttempts:       state.attempts,
			UploadFailures:       state.failures,
		}
		status := http.StatusServiceUnavailable
		if !state.lastSuccess.IsZero() && time.Since(state.lastSuccess) <= 2*interval+time.Minute {
			response.Status = "healthy"
			status = http.StatusOK
		}
		state.RUnlock()
		writer.Header().Set("Content-Type", "application/json")
		writer.Header().Set("Cache-Control", "no-store")
		writer.WriteHeader(status)
		_ = json.NewEncoder(writer).Encode(response)
	})
	return mux
}
