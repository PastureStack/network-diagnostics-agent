# Security policy

Report suspected vulnerabilities privately to the organization maintainers. Do not attach production diagnostics, tokens, host identifiers, addresses, routes, resolver configuration, credentials, or archives to a public issue.

The runtime intentionally collects only bounded aggregate counters. It does not execute commands, enter namespaces, access a container-engine socket, read workload content, collect log text, enumerate process names, or transmit direct host or network identifiers. The compatibility host identifier is HMAC-pseudonymized with the deployment token before it leaves the process.

The upload token must be randomly generated, at least 32 characters long, unique to one deployment, and rotated after suspected disclosure. It is accepted only through `PASTURESTACK_DIAGNOSTICS_TOKEN`, never through a command-line option, and is not written to output or health responses.

The supplied Catalog template mounts only `/proc`, `/etc/resolv.conf`, and `/etc/machine-id` read-only. It does not grant root, privileged mode, host PID, host network, Linux capabilities, or a container-engine socket. Operators should preserve these restrictions.

The collector limits source reads, request duration, JSON size, collection frequency, and counter ranges. The service independently authenticates and validates every snapshot; agent-side validation is not a trust boundary.
