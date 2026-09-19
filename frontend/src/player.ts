export type ControllerMode = 'shared' | 'static' | 'random'

const PLAYER_KEY = 'flyswatter.player'
const MODE_KEY = 'flyswatter.mode'

export interface PlayerIdentity {
  name: string
  id: string
}

export function loadPlayer(): PlayerIdentity | null {
  const raw = window.localStorage.getItem(PLAYER_KEY) || readCookie(PLAYER_KEY)
  if (!raw) {
    return null
  }
  try {
    const parsed = JSON.parse(raw) as PlayerIdentity
    if (typeof parsed?.name === 'string' && typeof parsed?.id === 'string' && parsed.id) {
      return parsed
    }
  } catch {
    // ignore
  }
  return null
}

export function savePlayer(name: string): PlayerIdentity {
  const trimmed = name.trim().slice(0, 32)
  const id = playerIdFromName(trimmed)
  const identity = { name: trimmed, id }
  const payload = JSON.stringify(identity)
  window.localStorage.setItem(PLAYER_KEY, payload)
  document.cookie = `${PLAYER_KEY}=${encodeURIComponent(payload)}; path=/; max-age=31536000; SameSite=Lax`
  return identity
}

export function playerIdFromName(name: string): string {
  const slug = name
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
    .slice(0, 48)
  return slug || 'player'
}

export function loadMode(): ControllerMode {
  const raw = window.localStorage.getItem(MODE_KEY)
  if (raw === 'shared' || raw === 'static' || raw === 'random') {
    return raw
  }
  return 'static'
}

export function saveMode(mode: ControllerMode) {
  window.localStorage.setItem(MODE_KEY, mode)
}

function readCookie(name: string): string | null {
  const prefix = `${name}=`
  for (const part of document.cookie.split(';')) {
    const value = part.trim()
    if (value.startsWith(prefix)) {
      return decodeURIComponent(value.slice(prefix.length))
    }
  }
  return null
}
