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
}

export interface Snapshot {
  tick: number
  episode: number
  alive: boolean
  survival_ms: number
  last_action: Action
  fly: Fly
  swatter: Swatter
}

export interface GameInput {
  swatter_position: Vec2
  attacking: boolean
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
    isSwatter(value.swatter)
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
    typeof value.attacking === 'boolean'
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
