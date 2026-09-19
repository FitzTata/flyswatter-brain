import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { GameCanvas } from './GameCanvas'

describe('GameCanvas', () => {
  it('renders the strike impact state', () => {
    render(
      <GameCanvas
        snapshot={null}
        input={{ swatter_position: { x: 0.5, y: 0.5 }, attacking: true }}
        onInputChange={vi.fn()}
      />,
    )

    expect(screen.getByLabelText(/click to strike/i)).toHaveClass('game-canvas--striking')
  })

  it('emits one short strike per pointer press', () => {
    vi.useFakeTimers()
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
    vi.advanceTimersByTime(75)

    expect(onInputChange).toHaveBeenCalledTimes(2)
    expect(onInputChange).toHaveBeenNthCalledWith(1, {
      swatter_position: { x: 0.5, y: 0.5 },
      attacking: true,
      arena_aspect_ratio: 800 / 520,
    })
    expect(onInputChange).toHaveBeenNthCalledWith(2, {
      swatter_position: { x: 0.5, y: 0.5 },
      attacking: false,
      arena_aspect_ratio: 800 / 520,
    })
  })
})
