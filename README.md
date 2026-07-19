# Network Diagnostics Agent

This repository provides the host-side collector used by PastureStack Network Diagnostics. The agent runs once per managed host, reads a bounded set of read-only host network counters, removes direct host and network identifiers at the source, and sends a compact snapshot to the authenticated diagnostics service.

PastureStack is an independent community effort to preserve, audit, and modernize the Rancher 1.6 ecosystem. It is not affiliated with or endorsed by Rancher Labs or SUSE.

**Upstream:** [`rancher/network-diagnostics-agent`](https://github.com/rancher/network-diagnostics-agent). This GitHub fork retains the upstream Git history, authorship, dates, and license notices unchanged; PastureStack maintenance is consolidated into one commit after the preserved upstream boundary.

## Runtime

The `run` command collects only aggregate counters:

- interface count and aggregate receive/transmit byte, error, and drop counters;
- route and default-route counts;
- resolver and search-list counts;
- host uptime;
- a count of collection sources that could not be read.

Interface names, IP and MAC addresses, routes, resolver addresses, search domains, host names, container names, process lists, commands, environment variables, files, and log content are never included. A host identifier is read from the compatibility metadata endpoint or a read-only machine identifier, then converted to an HMAC-SHA-256 pseudonym before transmission.

Every upload requires the same 32-to-256-character token configured on the service. The token is read only from `PASTURESTACK_DIAGNOSTICS_TOKEN`; it is not accepted as a command-line argument and is never logged.

```sh
docker run --rm \
  -e PASTURESTACK_DIAGNOSTICS_TOKEN='replace-with-a-random-secret-of-at-least-32-characters' \
  -e PASTURESTACK_DIAGNOSTICS_URL='http://network-diagnostics-service:8080' \
  -v /proc:/host/proc:ro \
  -v /etc/resolv.conf:/host/etc/resolv.conf:ro \
  -v /etc/machine-id:/host/etc/machine-id:ro \
  ghcr.io/pasturestack/network-diagnostics-agent:v0.2.0
```

The container runs without root, host PID, host network, Linux capabilities, or a container-engine socket. Its health endpoint is available on port `8090`; it becomes healthy only after at least one successful upload.

Runtime settings:

| Variable | Default | Constraint |
|---|---|---|
| `PASTURESTACK_DIAGNOSTICS_TOKEN` | none | Required; 32–256 characters |
| `PASTURESTACK_DIAGNOSTICS_URL` | `http://network-diagnostics-service:8080` | HTTP or HTTPS URL without embedded credentials |
| `PASTURESTACK_METADATA_URL` | `http://169.254.169.250/2016-07-29` | Compatibility metadata API base |
| `PASTURESTACK_DIAGNOSTICS_INTERVAL_SECONDS` | `600` | 60–86,400 |
| `PASTURESTACK_DIAGNOSTICS_REQUEST_TIMEOUT_SECONDS` | `15` | 1–60 |

## Audit planner

The inherited PastureStack audit contract remains available as the default command. It validates a proposed collection policy and emits a deterministic, non-executable plan:

```sh
network-diagnostics-agent < collection-policy.json
```

The deployable container starts the `run` command explicitly.

## Build and test

```sh
go test ./...
go vet ./...
go build -trimpath -o bin/network-diagnostics-agent ./cmd/network-diagnostics-agent
docker build --build-arg VERSION=v0.2.0 -t network-diagnostics-agent:v0.2.0 .
```

Run `sh scripts/validate.sh` on Unix-like systems or `pwsh -File scripts/validate.ps1` on Windows for formatting, repeated tests, vetting, module verification, and reproducible-build checks.

## Licensing

The inherited root `LICENSE` is preserved byte-for-byte and contains the Apache License 2.0 text. Go toolchain notices used for reproducible builds are included under `LICENSES/`. See `ORIGIN.md` for the preservation and release boundary.
