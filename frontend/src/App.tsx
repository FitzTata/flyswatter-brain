import { useState, type FormEvent } from 'react'
import './App.css'
import { UI_CONFIG } from './config'
import { GameCanvas } from './GameCanvas'
import { NeuralGraph } from './NeuralGraph'
import {
  loadMode,
  loadPlayer,
  saveMode,
  savePlayer,
  type ControllerMode,
  type PlayerIdentity,
} from './player'
import type { GameInput } from './types'
import { useGameSocket } from './useGameSocket'

const initialInput: GameInput = {
  swatter_position: { x: 0.75, y: 0.25 },
  attacking: false,
  arena_aspect_ratio: UI_CONFIG.defaultArenaAspectRatio,
}

function App() {
  const [input, setInput] = useState(initialInput)
  const [player, setPlayer] = useState<PlayerIdentity | null>(() => loadPlayer())
  const [mode, setMode] = useState<ControllerMode>(() => loadMode())
  const [nameDraft, setNameDraft] = useState('')
  const { status, snapshot, error, replaced } = useGameSocket(input, player, mode)

  const onSubmitName = (event: FormEvent) => {
    event.preventDefault()
    if (!nameDraft.trim()) {
      return
    }
    setPlayer(savePlayer(nameDraft))
  }

  const onModeChange = (next: ControllerMode) => {
    saveMode(next)
    setMode(next)
  }

  return (
    <main className="app-shell">
      {!player && (
        <div className="player-gate">
          <form className="player-gate__card" onSubmit={onSubmitName}>
            <p className="eyebrow">IDENTIFY</p>
            <h2>Player name</h2>
            <p>One active tab per name. Opening another tab replaces this session.</p>
            <input
              autoFocus
              maxLength={32}
              placeholder="alex"
              value={nameDraft}
              onChange={(event) => setNameDraft(event.target.value)}
            />
            <button type="submit" disabled={!nameDraft.trim()}>
              Enter arena
            </button>
          </form>
        </div>
      )}

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
              {player ? `${player.name.toUpperCase()} · ` : ''}X {input.swatter_position.x.toFixed(3)} / Y{' '}
              {input.swatter_position.y.toFixed(3)}
            </span>
          </div>
          <div className="canvas-frame">
            <GameCanvas snapshot={snapshot} input={input} onInputChange={setInput} />
            {!snapshot && player && (
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
            value={controllerLabel(mode, Boolean(snapshot?.neural_activity))}
            warning={mode !== 'random' && !snapshot?.neural_activity}
            hint="Who steers the fly: MaleCNS or random."
          />

          <div className="mode-picker" role="group" aria-label="Controller mode">
            <span className="mode-picker__label">
              WEIGHTS
              <Hint text="Shared learns together. Static is frozen. Random skips the brain." />
            </span>
            {(
              [
                ['shared', 'Shared'],
                ['static', 'Static'],
                ['random', 'Random'],
              ] as const
            ).map(([value, label]) => (
              <button
                key={value}
                type="button"
                className={mode === value ? 'mode-picker__option mode-picker__option--active' : 'mode-picker__option'}
                onClick={() => onModeChange(value)}
              >
                {label}
              </button>
            ))}
          </div>

          <div className="action-readout">
            <span>
              LAST COMMAND
              <Hint text="Latest decoded motor action from the controller." />
            </span>
            <strong>{formatAction(snapshot?.last_action)}</strong>
          </div>

          <NeuralGraph activity={snapshot?.neural_activity} action={snapshot?.last_action} />

          <div className={`life-state ${snapshot?.alive === false ? 'life-state--hit' : ''}`}>
            <span className="life-state__pulse" />
            {snapshot?.alive === false ? 'CONTACT DETECTED' : 'FLY ACTIVE'}
          </div>

          {(error || replaced) && (
            <p className="error-message">{replaced ? 'Session opened in another tab' : error}</p>
          )}
        </aside>
      </section>

      <section className="disclosure">
        <span>01</span>
        <p>{disclosureText(mode, Boolean(snapshot?.neural_activity))}</p>
        <span>{mode === 'random' ? 'CONTROL BASELINE' : snapshot?.neural_activity ? 'CONNECTOME ONLINE' : 'WAITING'}</span>
      </section>
    </main>
  )
}

interface MetricProps {
  label: string
  value: string | number
  warning?: boolean
  hint?: string
}

function Metric({ label, value, warning = false, hint }: MetricProps) {
  return (
    <div className="metric">
      <span>
        {label}
        {hint ? <Hint text={hint} /> : null}
      </span>
      <strong className={warning ? 'metric__warning' : ''}>{value}</strong>
    </div>
  )
}

function Hint({ text }: { text: string }) {
  return (
    <span className="hint">
      <button type="button" className="hint__mark" aria-label={text}>
        ?
      </button>
      <span className="hint__bubble" role="tooltip">
        {text}
      </span>
    </span>
  )
}

function controllerLabel(mode: ControllerMode, hasNeural: boolean) {
  if (mode === 'random') {
    return 'RANDOM'
  }
  if (!hasNeural) {
    return mode === 'shared' ? 'SHARED?' : 'STATIC?'
  }
  return mode === 'shared' ? 'SHARED LEARN' : 'STATIC'
}

function disclosureText(mode: ControllerMode, hasNeural: boolean) {
  if (mode === 'random') {
    return 'Seeded random controller. No connectome and no learning.'
  }
  if (mode === 'shared') {
    return hasNeural
      ? 'Shared MaleCNS weights with engineered dopamine plasticity. Weight changes are not validated learning.'
      : 'Shared learning mode selected. Waiting for MaleCNS worker.'
  }
  return hasNeural
    ? 'MaleCNS v1.0 with frozen baseline weights. Learning is disabled.'
    : 'Static MaleCNS mode selected. Waiting for neural worker.'
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
