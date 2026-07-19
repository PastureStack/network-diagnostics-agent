# Origin and release boundary

This repository retains inherited Git history so prior authorship, copyright, and license evidence remain auditable. The upstream root `LICENSE` is preserved byte-for-byte, and PastureStack does not claim sole authorship of historical work.

The current PastureStack maintenance commit replaces the historical privileged collector with a new bounded runtime and retains the deterministic collection-policy planner. The runtime uses only the Go standard library, emits aggregate counters, pseudonymizes the host reference before transmission, and does not include historical shell collectors, vendored dependencies, container-engine access, namespace entry, or upstream executable artifacts.

PastureStack changes remain subject to the inherited Apache License 2.0. Go toolchain notices used by the reproducible build are listed separately under `LICENSES/`. Product marks are used only in the required independence statement and factual upstream attribution.

Release approval applies only to the reviewed current tree and its reproducible container build. It does not make any representation about unrelated branches, old downstream distributions, or historical binary artifacts.
