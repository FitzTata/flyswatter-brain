import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { UI_CONFIG } from './config'
import { GameCanvas } from './GameCanvas'

describe('GameCanvas', () => {
  it('renders the strike impact state', () => {
    render(
      <GameCanvas
        snapshot={{
          tick: 1,
          episode: 1,
          alive: true,
          fly_hp: 1,
          survival_ms: 20,
          last_action: 'straight',
          fly: { position: { x: 0.5, y: 0.5 }, heading: 0, speed: 0.28, radius: 0.025 },
          swatter: {
            position: { x: 0.5, y: 0.5 },
            radius: 0.09,
            attacking: true,
            phase: 'strike',
          },
        }}
        input={{ swatter_position: { x: 0.5, y: 0.5 }, attacking: true }}
        onInputChange={vi.fn()}
      />,
    )

    expect(screen.getByLabelText(/click to strike/i)).toHaveClass('game-canvas--striking')
  })

  it('emits one swing with non-zero windup per pointer press', () => {
    vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout'] })
    const onInputChange = vi.fn()
    render(
      <GameCanvas
        snapshot={null}
        input={{ swatter_position: { x: 0.5, y: 0.5 }, attacking: false }}
        onInputChange={onInputChange}
      />,
    )
    const canvas = screen.getByLabelText(/click to strike/i)

    fireEvent.pointerDown(canvas, { clientX: 400, clientY: 260, pointerId: 1 })
    fireEvent.pointerDown(canvas, { clientX: 400, clientY: 260, pointerId: 1 })
    expect(canvas).toHaveClass('game-canvas--windup')
    expect(onInputChange).toHaveBeenCalledWith(
      expect.objectContaining({
        attacking: true,
        arena_aspect_ratio: 800 / 520,
      }),
    )

    vi.advanceTimersByTime(UI_CONFIG.swingWindupMS + UI_CONFIG.strikeDurationMS)

    expect(onInputChange).toHaveBeenCalledWith(
      expect.objectContaining({
        attacking: false,
        arena_aspect_ratio: 800 / 520,
      }),
    )
  })
})
