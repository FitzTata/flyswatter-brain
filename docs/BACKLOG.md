# Post-MVP backlog

## Neural activity visualization

Add a compact animated circuit view beside the arena so viewers can see how simulated activity becomes a command.

### Display

- Show a fixed, simplified path from sensory input to readout neurons and the selected action.
- Label every visible node with its actual model identifier or cell type, for example `DNp20-L`, `DNp20-R`, or `DNpe017`.
- Encode recent spikes as short pulses and firing rate as node intensity.
- Animate propagation without implying that the simplified layout is the anatomical connectome.
- Keep the current command visible as the final node in the path.

### Protocol

Snapshots may include an optional telemetry payload:

```json
{
  "neural_activity": {
    "model_time_ms": 1200,
    "source": "malecns_v1",
    "nodes": [
      {"id": "DNp20-L", "spikes": 2, "rate_hz": 4.0, "activation": 0.4},
      {"id": "DNp20-R", "spikes": 7, "rate_hz": 14.0, "activation": 1.0},
      {"id": "DNpe017", "spikes": 1, "rate_hz": 2.0, "activation": 0.2}
    ]
  }
}
```

### Scientific boundary

- Never generate fake neural activity for the random-controller baseline.
- Show `NEURAL WORKER NOT CONNECTED` until real worker telemetry is available.
- Describe action nodes as engineered readouts, not discovered “turn” or “escape” neurons.
- Preserve raw spike counts in logs so the animation can be audited.

### Acceptance criteria

- The graph changes only from worker telemetry.
- A recorded input can reproduce the same visual trace.
- Labels match the exact model configuration.
- Missing or stale telemetry is visibly marked.
