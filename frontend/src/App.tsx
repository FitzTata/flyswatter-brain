import { useState } from 'react'
import './App.css'
import { GameCanvas } from './GameCanvas'
import { NeuralGraph } from './NeuralGraph'
import type { GameInput } from './types'
import { useGameSocket } from './useGameSocket'

const initialInput: GameInput = {
  swatter_position: { x: 0.75, y: 0.25 },
  attacking: false,
  arena_aspect_ratio: 2,
}

function App() {
  const [input, setInput] = useState(initialInput)
  const { status, snapshot, error } = useGameSocket(input)

  return (
    <main className="app-shell">
      <header className="masthead">
        <div>
          <p className="eyebrow">MALECNS CONTROL EXPERIMENT / MVP 01</p>
          <h1>
            FLY <span>VS</span> FLYSWATTER
          </h1>
        </div>
        <div className={`connection connection--${status}`}>
          <span aria-hidden="true" />
          {status.toUpperCase()}
        </div>
      </header>

      <section className="experiment">
        <div className="arena-panel">
          <div className="panel-heading">
            <span>LIVE ARENA</span>
            <span className="coordinates">
              X {input.swatter_position.x.toFixed(3)} / Y {input.swatter_position.y.toFixed(3)}
            </span>
          </div>
          <div className="canvas-frame">
            <GameCanvas snapshot={snapshot} input={input} onInputChange={setInput} />
            {!snapshot && (
              <div className="canvas-placeholder">
                <span>WAITING FOR BACKEND</span>
                <small>go run ./cmd/server</small>
              </div>
            )}
            <div className="canvas-instruction">MOVE TO AIM · CLICK TO STRIKE</div>
          </div>
        </div>

        <aside className="telemetry">
          <div className="panel-heading">
            <span>TELEMETRY</span>
            <span>20MS / 2MS</span>
          </div>

          <Metric label="Episode" value={snapshot?.episode ?? '—'} />
          <Metric label="Brain tick" value={snapshot?.tick ?? '—'} />
          <Metric label="Survival" value={formatDuration(snapshot?.survival_ms)} />
          <Metric
            label="Controller"
            value={snapshot?.neural_activity ? 'MALECNS' : 'RANDOM'}
            warning={!snapshot?.neural_activity}
          />

          <div className="action-readout">
            <span>LAST COMMAND</span>
            <strong>{formatAction(snapshot?.last_action)}</strong>
          </div>

          <NeuralGraph activity={snapshot?.neural_activity} action={snapshot?.last_action} />

          <div className={`life-state ${snapshot?.alive === false ? 'life-state--hit' : ''}`}>
            <span className="life-state__pulse" />
            {snapshot?.alive === false ? 'CONTACT DETECTED' : 'FLY ACTIVE'}
          </div>

          {error && <p className="error-message">{error}</p>}
        </aside>
      </section>

      <section className="disclosure">
        <span>01</span>
        <p>
          {snapshot?.neural_activity
            ? 'MaleCNS v1.0 produces the commands through an engineered DNp20 / DNpe017 readout. Learning is disabled.'
            : 'The backend currently uses a seeded random controller. No connectome is attached and no learning is claimed in this build.'}
        </p>
        <span>{snapshot?.neural_activity ? 'CONNECTOME ONLINE' : 'CONTROL BASELINE'}</span>
      </section>
    </main>
  )
}

interface MetricProps {
  label: string
  value: string | number
  warning?: boolean
}

function Metric({ label, value, warning = false }: MetricProps) {
  return (
    <div className="metric">
      <span>{label}</span>
      <strong className={warning ? 'metric__warning' : ''}>{value}</strong>
    </div>
  )
}

function formatDuration(value?: number) {
  if (value === undefined) {
    return '—'
  }
  return `${(value / 1000).toFixed(2)} S`
}

function formatAction(value?: string) {
  return value?.replace('_', ' ').toUpperCase() ?? '—'
}

export default App
