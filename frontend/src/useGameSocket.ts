import { useEffect, useRef, useState } from 'react'
import { decodeServerMessage, type ConnectionStatus, type GameInput, type Snapshot } from './types'

const stepIntervalMS = 50

export interface GameSocketState {
  status: ConnectionStatus
  snapshot: Snapshot | null
  error: string | null
}

export function useGameSocket(input: GameInput): GameSocketState {
  const inputRef = useRef(input)
  const [state, setState] = useState<GameSocketState>({
    status: 'connecting',
    snapshot: null,
    error: null,
  })

  useEffect(() => {
    inputRef.current = input
  }, [input])

  useEffect(() => {
    const socket = new WebSocket(websocketURL())

    socket.onopen = () => {
      setState((current) => ({ ...current, status: 'connected', error: null }))
    }
    socket.onmessage = (event) => {
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
      setState((current) => ({ ...current, error: 'WebSocket connection failed' }))
    }
    socket.onclose = () => {
      setState((current) => ({ ...current, status: 'disconnected' }))
    }

    const interval = window.setInterval(() => {
      if (socket.readyState !== WebSocket.OPEN) {
        return
      }
      socket.send(JSON.stringify({ type: 'input', input: inputRef.current }))
    }, stepIntervalMS)

    return () => {
      window.clearInterval(interval)
      socket.close()
    }
  }, [])

  return state
}

function websocketURL(): string {
  if (import.meta.env.VITE_WS_URL) {
    return import.meta.env.VITE_WS_URL
  }
  const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws'
  return `${protocol}://${window.location.hostname}:8080/ws`
}
