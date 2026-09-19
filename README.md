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
{"type":"input","input":{"swatter_position":{"x":0.4,"y":0.6},"attacking":true}}
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

## Status

The interactive random-controller baseline is implemented. The MaleCNS worker is not connected yet. See the [`MVP plan`](docs/MVP_PLAN.md) and [`post-MVP backlog`](docs/BACKLOG.md).
