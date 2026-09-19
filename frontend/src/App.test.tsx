import { act, fireEvent, render, screen } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import App from './App'
import { UI_CONFIG } from './config'

class MockWebSocket {
  static readonly OPEN = 1
  static instances: MockWebSocket[] = []

  readyState = 0
  onopen: (() => void) | null = null
  onmessage: ((event: MessageEvent) => void) | null = null
  onerror: (() => void) | null = null
  onclose: ((event?: CloseEvent) => void) | null = null
  send = vi.fn()
  close = vi.fn()
  readonly url: string

  constructor(url: string) {
    this.url = url
    MockWebSocket.instances.push(this)
  }
}

function seedPlayer() {
  window.localStorage.setItem(
    'flyswatter.player',
    JSON.stringify({ name: 'Alex', id: 'alex' }),
  )
  window.localStorage.setItem('flyswatter.mode', 'static')
}

describe('App', () => {
  beforeEach(() => {
    MockWebSocket.instances = []
    vi.stubGlobal('WebSocket', MockWebSocket)
    window.localStorage.clear()
    document.cookie = 'flyswatter.player=; max-age=0; path=/'
    window.history.replaceState({}, '', '/')
  })

  it('asks for a player name before connecting', () => {
    render(<App />)

    expect(screen.getByText('Player name')).toBeInTheDocument()
    expect(MockWebSocket.instances).toHaveLength(0)
  })

  it('connects with player session and mode query', () => {
    seedPlayer()
    render(<App />)

    expect(MockWebSocket.instances[0].url).toContain('session=alex')
    expect(MockWebSocket.instances[0].url).toContain('mode=static')
  })

  it('reconnects with shared mode when selected', () => {
    seedPlayer()
    render(<App />)

    fireEvent.click(screen.getByRole('button', { name: 'Shared' }))

    expect(MockWebSocket.instances.at(-1)?.url).toContain('mode=shared')
  })

  it('renders live backend telemetry', () => {
    seedPlayer()
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
    seedPlayer()
    render(<App />)

    act(() => {
      MockWebSocket.instances[0].onclose?.({ reason: '' } as CloseEvent)
    })
    expect(screen.getByText('DISCONNECTED')).toBeInTheDocument()

    act(() => {
      vi.advanceTimersByTime(1000)
    })
    expect(MockWebSocket.instances).toHaveLength(2)
    expect(screen.getByText('CONNECTING')).toBeInTheDocument()
  })

  it('does not reconnect after session_replaced', () => {
    vi.useFakeTimers()
    seedPlayer()
    render(<App />)

    act(() => {
      MockWebSocket.instances[0].onclose?.({ reason: 'session_replaced', code: 1008 } as CloseEvent)
    })

    act(() => {
      vi.advanceTimersByTime(2000)
    })

    expect(MockWebSocket.instances).toHaveLength(1)
    expect(screen.getByText('Session opened in another tab')).toBeInTheDocument()
  })

  it('keeps only one game step in flight', () => {
    vi.useFakeTimers()
    seedPlayer()
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
      vi.advanceTimersByTime(UI_CONFIG.gameStepIntervalMS)
    })
    expect(socket.send).toHaveBeenCalledTimes(2)
  })
})
