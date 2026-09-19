# Docker decision

Docker is deferred for now.

## Why not yet

- MaleCNS setup already needs a local C++17 toolchain, a ~1.1 GB dataset download, and a Windows-friendly Python 3.14 environment managed by `uv`.
- A useful image would have to ship or rebuild that toolchain and dataset, which makes the container heavy, slow to iterate on, and awkward for the current single-machine experiment loop.
- The interactive path is already `go run` + Vite + `scripts/setup_neural.ps1`. Adding Docker before learning/eval workloads would duplicate that path without reducing real setup pain.

## When to revisit

Add Docker (or a slim Compose stack) when one of these becomes true:

1. CI needs a reproducible MaleCNS worker without installing the toolchain on every runner.
2. Learning experiments require identical worker images across machines.
3. We want a one-command demo that does not depend on a local `.venv-neural`.

Until then, prefer the documented local scripts and keep container packaging out of the critical path.
