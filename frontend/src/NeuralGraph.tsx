import type { Action, NeuralActivity, NeuralNode } from './types'

interface NeuralGraphProps {
  activity?: NeuralActivity
  action?: Action
}

const positions = [
  { x: 38, y: 38 },
  { x: 38, y: 112 },
  { x: 120, y: 38 },
  { x: 120, y: 112 },
]

export function NeuralGraph({ activity, action }: NeuralGraphProps) {
  if (!activity) {
    return (
      <section className="neural-graph neural-graph--offline">
        <div className="neural-graph__heading">
          <span>NEURAL READOUT</span>
          <span>NO SIGNAL</span>
        </div>
        <p>NEURAL WORKER NOT CONNECTED</p>
      </section>
    )
  }

  const maxRate = Math.max(1, ...activity.nodes.map((node) => node.rate_hz))
  const nodes = activity.nodes.slice(0, positions.length)

  return (
    <section className="neural-graph" aria-label="MaleCNS neural activity">
      <div className="neural-graph__heading">
        <span>NEURAL READOUT</span>
        <span>{activity.model_time_ms.toFixed(0)} MS</span>
      </div>
      <svg viewBox="0 0 260 170" role="img" aria-label="Active readout neurons and decoded command">
        <g className="neural-graph__edges">
          {nodes.map((node, index) => (
            <line
              key={node.id}
              x1={positions[index].x}
              y1={positions[index].y}
              x2="218"
              y2="76"
              style={{ opacity: edgeOpacity(node, maxRate) }}
            />
          ))}
        </g>

        {nodes.map((node, index) => (
          <Neuron key={node.id} node={node} x={positions[index].x} y={positions[index].y} maxRate={maxRate} />
        ))}

        <g className="neural-graph__command" transform="translate(218 76)">
          <circle r="23" />
          <text y="-3">COMMAND</text>
          <text y="9" className="neural-graph__command-value">
            {shortAction(action)}
          </text>
        </g>
      </svg>
      <div className="neural-graph__footer">
        <span>{activity.total_spikes.toLocaleString()} SPIKES</span>
        <span>Δ {activity.difference_hz.toFixed(1)} HZ</span>
      </div>
    </section>
  )
}

interface NeuronProps {
  node: NeuralNode
  x: number
  y: number
  maxRate: number
}

function Neuron({ node, x, y, maxRate }: NeuronProps) {
  const level = Math.min(1, node.rate_hz / maxRate)
  return (
    <g
      className={`neural-graph__node ${node.spikes > 0 ? 'neural-graph__node--active' : ''}`}
      transform={`translate(${x} ${y})`}
      style={{ opacity: 0.35 + level * 0.65 }}
    >
      <title>
        {node.label} neuron {node.id}: {node.spikes} spikes, {node.rate_hz.toFixed(1)} Hz
      </title>
      <circle r={10 + level * 4} />
      <text y="25">{node.label}</text>
      <text y="36" className="neural-graph__node-id">
        #{node.id}
      </text>
      <text y="47" className="neural-graph__node-rate">
        {node.rate_hz.toFixed(0)} HZ
      </text>
    </g>
  )
}

function edgeOpacity(node: NeuralNode, maxRate: number) {
  return 0.12 + Math.min(1, node.rate_hz / maxRate) * 0.68
}

function shortAction(action?: Action) {
  switch (action) {
    case 'turn_left':
      return 'LEFT'
    case 'turn_right':
      return 'RIGHT'
    case 'escape':
      return 'ESCAPE'
    case 'straight':
      return 'STRAIGHT'
    default:
      return '—'
  }
}
