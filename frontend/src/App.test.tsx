import { act, render, screen } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import App from './App'

class MockWebSocket {
  static readonly OPEN = 1
  static instances: MockWebSocket[] = []

  readyState = 0
  onopen: (() => void) | null = null
  onmessage: ((event: MessageEvent) => void) | null = null
  onerror: (() => void) | null = null
  onclose: (() => void) | null = null
  send = vi.fn()
  close = vi.fn()
  readonly url: string

  constructor(url: string) {
    this.url = url
    MockWebSocket.instances.push(this)
  }
}

describe('App', () => {
  beforeEach(() => {
    MockWebSocket.instances = []
    vi.stubGlobal('WebSocket', MockWebSocket)
  })

  it('renders live backend telemetry', () => {
    render(<App />)
    const socket = MockWebSocket.instances[0]

    act(() => {
      socket.readyState = MockWebSocket.OPEN
      socket.onopen?.()
      socket.onmessage?.(
        new MessageEvent('message', {
          data: JSON.stringify({
            type: 'snapshot',
            snapshot: {
              tick: 42,
              episode: 7,
              alive: true,
              survival_ms: 1250,
              last_action: 'turn_left',
              fly: {
                position: { x: 0.4, y: 0.6 },
                heading: 0.5,
                speed: 0.18,
                radius: 0.025,
              },
              swatter: {
                position: { x: 0.7, y: 0.2 },
                radius: 0.09,
                attacking: false,
              },
            },
          }),
        }),
      )
    })

    expect(screen.getByText('CONNECTED')).toBeInTheDocument()
    expect(screen.getByText('42')).toBeInTheDocument()
    expect(screen.getByText('1.25 S')).toBeInTheDocument()
    expect(screen.getByText('TURN LEFT')).toBeInTheDocument()
  })

  it('reconnects after the socket closes', () => {
    vi.useFakeTimers()
    render(<App />)

    act(() => {
      MockWebSocket.instances[0].onclose?.()
    })
    expect(screen.getByText('DISCONNECTED')).toBeInTheDocument()

    act(() => {
      vi.advanceTimersByTime(1000)
    })
    expect(MockWebSocket.instances).toHaveLength(2)
    expect(screen.getByText('CONNECTING')).toBeInTheDocument()
  })

  it('keeps only one game step in flight', () => {
    vi.useFakeTimers()
    render(<App />)
    const socket = MockWebSocket.instances[0]
    socket.readyState = MockWebSocket.OPEN

    act(() => {
      socket.onopen?.()
      vi.advanceTimersByTime(100)
    })
    expect(socket.send).toHaveBeenCalledTimes(1)

    act(() => {
      socket.onmessage?.(new MessageEvent('message', { data: '{"type":"error","error":"done"}' }))
      vi.advanceTimersByTime(20)
    })
    expect(socket.send).toHaveBeenCalledTimes(2)
  })
})
