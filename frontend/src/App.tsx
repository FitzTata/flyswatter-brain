import { useState, type FormEvent } from 'react'
import './App.css'
import { UI_CONFIG } from './config'
import { GameCanvas } from './GameCanvas'
import { NeuralGraph } from './NeuralGraph'
import { loadPlayer, savePlayer, type PlayerIdentity } from './player'
import type { GameInput } from './types'
import { useGameSocket } from './useGameSocket'

const GITHUB_URL = 'https://github.com/FitzTata/flyswatter-brain'

const initialInput: GameInput = {
  swatter_position: { x: 0.75, y: 0.25 },
  attacking: false,
  arena_aspect_ratio: UI_CONFIG.defaultArenaAspectRatio,
}

function App() {
  const [input, setInput] = useState(initialInput)
  const [player, setPlayer] = useState<PlayerIdentity | null>(() => loadPlayer())
  const [nameDraft, setNameDraft] = useState('')
  const { status, snapshot, error, replaced } = useGameSocket(input, player)

  const onSubmitName = (event: FormEvent) => {
    event.preventDefault()
    if (!nameDraft.trim()) {
      return
    }
    setPlayer(savePlayer(nameDraft))
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
            <GithubLink />
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
          <div className="hp-meter" aria-label="Fly HP">
            <span>FLY HP</span>
            <div className="hp-meter__track">
              <div
                className="hp-meter__fill"
                style={{ width: `${Math.max(0, Math.min(1, snapshot?.fly_hp ?? 1)) * 100}%` }}
              />
            </div>
            <strong>{Math.round(Math.max(0, Math.min(1, snapshot?.fly_hp ?? 1)) * 100)}%</strong>
          </div>
          <Metric
            label="Controller"
            value={snapshot?.neural_activity ? 'SHARED LEARN' : 'SHARED?'}
            warning={!snapshot?.neural_activity}
            hint="MaleCNS shared weights steer the fly."
          />

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
            {snapshot?.alive === false ? 'FLY DOWN' : 'FLY ACTIVE'}
          </div>

          {(error || replaced) && (
            <p className="error-message">{replaced ? 'Session opened in another tab' : error}</p>
          )}
        </aside>
      </section>

      <section className="disclosure">
        <span>01</span>
        <div className="disclosure__body">
          <p>{disclosureText(Boolean(snapshot?.neural_activity))}</p>
          <GithubLink />
        </div>
        <span>{snapshot?.neural_activity ? 'CONNECTOME ONLINE' : 'WAITING'}</span>
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

function GithubLink() {
  return (
    <a className="github-link" href={GITHUB_URL} target="_blank" rel="noreferrer">
      <svg className="github-link__icon" viewBox="0 0 16 16" aria-hidden="true">
        <path
          fill="currentColor"
          d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27s1.36.09 2 .27c1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.01 8.01 0 0 0 16 8c0-4.42-3.58-8-8-8"
        />
      </svg>
      GitHub
    </a>
  )
}

function disclosureText(hasNeural: boolean) {
  return hasNeural
    ? 'Shared MaleCNS weights with engineered dopamine plasticity. Weight changes are not validated learning.'
    : 'Waiting for MaleCNS worker.'
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
