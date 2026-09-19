import { useEffect, useRef, useState } from 'react'
import { UI_CONFIG } from './config'
import type { PlayerIdentity } from './player'
import { decodeServerMessage, type ConnectionStatus, type GameInput, type Snapshot } from './types'

export interface GameSocketState {
  status: ConnectionStatus
  snapshot: Snapshot | null
  error: string | null
  replaced: boolean
}

export function useGameSocket(
  input: GameInput,
  player: PlayerIdentity | null,
): GameSocketState {
  const inputRef = useRef(input)
  const inFlightRef = useRef(false)
  const [state, setState] = useState<GameSocketState>({
    status: player ? 'connecting' : 'disconnected',
    snapshot: null,
    error: null,
    replaced: false,
  })

  const effectiveState: GameSocketState = player
    ? state
    : { status: 'disconnected', snapshot: null, error: null, replaced: false }

  useEffect(() => {
    inputRef.current = input
  }, [input])

  useEffect(() => {
    if (!player) {
      return
    }

    let active = true
    let socket: WebSocket | null = null
    let reconnectTimer: number | undefined
    let replaced = false

    const connect = () => {
      setState((current) => ({
        ...current,
        status: 'connecting',
        replaced: false,
        error: null,
      }))
      socket = new WebSocket(websocketURL(player.id))

      socket.onopen = () => {
        if (!active) {
          return
        }
        inFlightRef.current = false
        setState((current) => ({
          ...current,
          status: 'connected',
          error: null,
          replaced: false,
        }))
      }
      socket.onmessage = (event) => {
        if (!active) {
          return
        }
        inFlightRef.current = false
        const message = decodeServerMessage(String(event.data))
        if (!message) {
          setState((current) => ({ ...current, error: 'Invalid server response' }))
          return
        }
        if (message.type === 'error') {
          setState((current) => ({ ...current, error: message.error }))
          return
        }
        setState((current) => ({ ...current, snapshot: message.snapshot, error: null }))
      }
      socket.onerror = () => {
        if (!active) {
          return
        }
        setState((current) => ({ ...current, error: 'WebSocket connection failed' }))
      }
      socket.onclose = (event) => {
        if (!active) {
          return
        }
        inFlightRef.current = false
        if (event.reason === 'session_replaced' || event.code === 1008) {
          replaced = true
          setState((current) => ({
            ...current,
            status: 'disconnected',
            replaced: true,
            error: 'Session opened in another tab',
          }))
          return
        }
        setState((current) => ({ ...current, status: 'disconnected' }))
        if (!replaced) {
          reconnectTimer = window.setTimeout(connect, 1000)
        }
      }
    }
    connect()

    const interval = window.setInterval(() => {
      if (socket?.readyState !== WebSocket.OPEN || inFlightRef.current) {
        return
      }
      inFlightRef.current = true
      try {
        socket.send(JSON.stringify({ type: 'input', input: inputRef.current }))
      } catch {
        inFlightRef.current = false
      }
    }, UI_CONFIG.gameStepIntervalMS)

    return () => {
      active = false
      window.clearInterval(interval)
      window.clearTimeout(reconnectTimer)
      socket?.close()
    }
  }, [player])

  return effectiveState
}

export function websocketURL(playerId: string): string {
  const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws'
  const localHost =
    window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1'
  const defaultURL = localHost
    ? `${protocol}://${window.location.hostname}:8080/ws`
    : `${protocol}://${window.location.host}/ws`
  const baseURL = import.meta.env.VITE_WS_URL || defaultURL
  const url = new URL(baseURL)
  url.searchParams.set('session', playerId)
  url.searchParams.set('player', playerId)
  url.searchParams.set('mode', 'shared')
  return url.toString()
}
