export type Action = 'turn_left' | 'turn_right' | 'straight' | 'escape'

export interface Vec2 {
  x: number
  y: number
}

export interface Fly {
  position: Vec2
  heading: number
  speed: number
  radius: number
}

export interface Swatter {
  position: Vec2
  radius: number
  attacking: boolean
  phase?: 'windup' | 'strike'
}

export interface Snapshot {
  tick: number
  episode: number
  alive: boolean
  survival_ms: number
  last_action: Action
  fly: Fly
  swatter: Swatter
  neural_activity?: NeuralActivity
}

export interface NeuralActivity {
  model: string
  model_time_ms: number
  left_hz: number
  right_hz: number
  difference_hz: number
  gate_spikes: number
  total_spikes: number
  step_seconds: number
  nodes: NeuralNode[]
}

export interface NeuralNode {
  id: string
  label: string
  spikes: number
  rate_hz: number
}

export interface GameInput {
  swatter_position: Vec2
  attacking: boolean
  arena_aspect_ratio?: number
}

export type ConnectionStatus = 'connecting' | 'connected' | 'disconnected'

export type ServerMessage =
  | { type: 'snapshot'; snapshot: Snapshot }
  | { type: 'error'; error: string }

export function decodeServerMessage(raw: string): ServerMessage | null {
  let value: unknown
  try {
    value = JSON.parse(raw)
  } catch {
    return null
  }

  if (!isRecord(value) || typeof value.type !== 'string') {
    return null
  }
  if (value.type === 'error' && typeof value.error === 'string') {
    return { type: 'error', error: value.error }
  }
  if (value.type === 'snapshot' && isSnapshot(value.snapshot)) {
    return { type: 'snapshot', snapshot: value.snapshot }
  }
  return null
}

function isSnapshot(value: unknown): value is Snapshot {
  return (
    isRecord(value) &&
    typeof value.tick === 'number' &&
    typeof value.episode === 'number' &&
    typeof value.alive === 'boolean' &&
    typeof value.survival_ms === 'number' &&
    isAction(value.last_action) &&
    isFly(value.fly) &&
    isSwatter(value.swatter) &&
    (value.neural_activity === undefined || isNeuralActivity(value.neural_activity))
  )
}

function isNeuralActivity(value: unknown): value is NeuralActivity {
  return (
    isRecord(value) &&
    typeof value.model === 'string' &&
    typeof value.model_time_ms === 'number' &&
    typeof value.left_hz === 'number' &&
    typeof value.right_hz === 'number' &&
    typeof value.difference_hz === 'number' &&
    typeof value.gate_spikes === 'number' &&
    typeof value.total_spikes === 'number' &&
    typeof value.step_seconds === 'number' &&
    Array.isArray(value.nodes)
  )
}

function isFly(value: unknown): value is Fly {
  return (
    isRecord(value) &&
    isVec2(value.position) &&
    typeof value.heading === 'number' &&
    typeof value.speed === 'number' &&
    typeof value.radius === 'number'
  )
}

function isSwatter(value: unknown): value is Swatter {
  return (
    isRecord(value) &&
    isVec2(value.position) &&
    typeof value.radius === 'number' &&
    typeof value.attacking === 'boolean' &&
    (value.phase === undefined || value.phase === 'windup' || value.phase === 'strike')
  )
}

function isVec2(value: unknown): value is Vec2 {
  return isRecord(value) && typeof value.x === 'number' && typeof value.y === 'number'
}

function isAction(value: unknown): value is Action {
  return value === 'turn_left' || value === 'turn_right' || value === 'straight' || value === 'escape'
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}
