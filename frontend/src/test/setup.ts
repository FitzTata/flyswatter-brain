import '@testing-library/jest-dom/vitest'
import { cleanup } from '@testing-library/react'
import { afterEach, vi } from 'vitest'

const canvasContext = {
  arc: vi.fn(),
  beginPath: vi.fn(),
  clearRect: vi.fn(),
  ellipse: vi.fn(),
  fill: vi.fn(),
  fillRect: vi.fn(),
  lineTo: vi.fn(),
  moveTo: vi.fn(),
  restore: vi.fn(),
  rotate: vi.fn(),
  save: vi.fn(),
  setTransform: vi.fn(),
  stroke: vi.fn(),
  translate: vi.fn(),
}

Object.defineProperty(HTMLCanvasElement.prototype, 'getContext', {
  value: vi.fn(() => canvasContext),
})

Object.defineProperty(HTMLCanvasElement.prototype, 'getBoundingClientRect', {
  value: vi.fn(() => ({
    width: 800,
    height: 520,
    top: 0,
    left: 0,
    right: 800,
    bottom: 520,
    x: 0,
    y: 0,
    toJSON: () => ({}),
  })),
})

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
  vi.unstubAllGlobals()
})
