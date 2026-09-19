# Fly vs Flyswatter

An experimental browser game in which a simulated Drosophila connectome controls a virtual fly trying to dodge a flyswatter.

## Concept

The player moves a flyswatter with the mouse. The game produces a visual frame, the backend maps it to photoreceptor stimuli for the MaleCNS model, advances the spiking simulation, and decodes selected descending-neuron activity into one of four commands:

- `turn_left`
- `turn_right`
- `straight`
- `escape`

The frontend applies the command and visualizes the fly's movement, readout-neuron activity, and episode statistics.

## Scientific boundaries

- This is an approximate wiring-constrained simulation, not a digital copy of a living fly.
- Pixel-to-photoreceptor mapping and neuron-to-command decoding are engineered interfaces.
- Synaptic weight changes alone do not demonstrate learning.
- Learned avoidance requires improvement on held-out scenarios against random-controller, frozen-weight, and shuffled-reinforcement controls.

## Proposed stack

- Backend: Go, WebSocket, long-lived game sessions
- Neural worker: an existing MaleCNS kernel from the DOOMFLY/StonkFly ecosystem, or a compatible standalone process
- Frontend: React, TypeScript, Vite, Canvas
- Storage: checkpoint files and JSONL/SQLite, with no separate database server

## Architecture

```text
Browser Canvas
  ├─ renders the fly and flyswatter
  ├─ sends player input
  └─ interpolates snapshots
             │ WebSocket
             ▼
Go backend
  ├─ owns authoritative game state
  ├─ builds sensory frames
  ├─ invokes the neural worker
  ├─ decodes actions
  └─ computes collisions and metrics
             │ IPC
             ▼
MaleCNS neural worker
  ├─ retinal adapter
  ├─ stateful LIF simulation
  ├─ descending-neuron readout
  └─ optional dopamine-gated plasticity
```

## MVP backend

The current backend owns deterministic game state and exposes:

- `GET /healthz` for health checks
- `GET /ws` for stateful game sessions

Run it locally:

```sh
go run ./cmd/server
```

The server listens on `127.0.0.1:8080` by default. Set `FLYSWATTER_ADDR` to override it.

Run the backend checks:

```sh
go test ./...
go vet ./...
```

WebSocket sessions start with a snapshot. The client then sends input messages:

```json
{"type":"input","input":{"swatter_position":{"x":0.4,"y":0.6},"attacking":true,"arena_aspect_ratio":2.0}}
```

Each valid input advances the authoritative game by one fixed step and returns the next snapshot.

## MVP frontend

Start the backend, then run the browser client in another terminal:

```sh
cd frontend
npm install
npm run dev
```

Open `http://127.0.0.1:5173`, move the pointer to aim the flyswatter, and click to strike.

Run the frontend checks:

```sh
npm run typecheck
npm run lint
npm test
npm run build
```

Run all Go, TypeScript, and Python linters from the repository root:

```powershell
.\scripts\lint.ps1
```

## MaleCNS controller

The neural worker uses the complete retained MaleCNS v1.0 graph through the pinned StonkFly submodule. On Windows, setup requires Python 3.14 and a C++17 compiler, downloads about 1.1 GB of source data, verifies its checksums, and builds local graph artifacts:

```powershell
powershell -ExecutionPolicy Bypass -File scripts/setup_neural.ps1
```

The downloaded dataset, compiled kernel, and Python environment remain under ignored local directories.

The server uses `auto` mode by default: it starts MaleCNS when the worker is ready and otherwise reports the error and falls back to the seeded random controller. To require MaleCNS and fail instead of falling back:

```powershell
$env:FLYSWATTER_CONTROLLER = "malecns"
go run ./cmd/server
```

Set `FLYSWATTER_CONTROLLER=random` to force the baseline. The neural worker receives only a rendered RGB arena frame, advances 2 ms of model time per 20-ms game step at the original 0.1-ms integration timestep, and decodes DNp20/DNpe017 rates over a rolling 100-ms model-time window. This is an engineered interface, not a validated natural fly motor decoder. See [`THIRD_PARTY.md`](THIRD_PARTY.md) for attribution.

## Status

The interactive baseline, MaleCNS worker, and neural activity readout are implemented. Learning remains post-MVP work. See the [`MVP plan`](docs/MVP_PLAN.md) and [`post-MVP backlog`](docs/BACKLOG.md).

## License

Project-owned code is available under the [MIT License](LICENSE). StonkFly and MaleCNS data retain their respective third-party licenses; see [`THIRD_PARTY.md`](THIRD_PARTY.md).
