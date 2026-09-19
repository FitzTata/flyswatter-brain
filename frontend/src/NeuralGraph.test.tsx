import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import { NeuralGraph } from './NeuralGraph'
import type { NeuralActivity } from './types'

const activity: NeuralActivity = {
  model: 'MaleCNS v1.0',
  model_time_ms: 128,
  left_hz: 20,
  right_hz: 0,
  difference_hz: 20,
  gate_spikes: 1,
  total_spikes: 12,
  step_seconds: 0.1,
  nodes: [
    { id: '10162', label: 'DNp20-L', spikes: 2, rate_hz: 20 },
    { id: '10059', label: 'DNp20-R', spikes: 0, rate_hz: 0 },
  ],
}

describe('NeuralGraph', () => {
  it('shows active neurons and decoded command', () => {
    const { container } = render(<NeuralGraph activity={activity} action="turn_left" />)

    expect(screen.getByText('#10162')).toBeInTheDocument()
    expect(screen.getByText('LEFT')).toBeInTheDocument()
    expect(screen.getByText('12 SPIKES')).toBeInTheDocument()
    expect(container.querySelectorAll('.neural-graph__node--active')).toHaveLength(1)
  })

  it('shows no signal without neural telemetry', () => {
    render(<NeuralGraph />)

    expect(screen.getByText('NEURAL WORKER NOT CONNECTED')).toBeInTheDocument()
  })
})
