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

## Status

The project is currently at the MVP design stage. The implementation plan is available in [`docs/MVP_PLAN.md`](docs/MVP_PLAN.md).
