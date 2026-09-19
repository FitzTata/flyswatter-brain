import { useEffect, useRef, type CSSProperties, type PointerEvent } from 'react'
import { UI_CONFIG } from './config'
import type { GameInput, Snapshot, Vec2 } from './types'

interface GameCanvasProps {
  snapshot: Snapshot | null
  input: GameInput
  onInputChange: (input: GameInput) => void
}

export function GameCanvas({ snapshot, input, onInputChange }: GameCanvasProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const positionRef = useRef(input.swatter_position)
  const aspectRatioRef = useRef(input.arena_aspect_ratio ?? UI_CONFIG.defaultArenaAspectRatio)
  const strikeActiveRef = useRef(false)
  const strikeLockedRef = useRef(false)
  const strikeTimerRef = useRef<number | undefined>(undefined)

  useEffect(() => {
    const canvas = canvasRef.current
    const context = canvas?.getContext('2d')
    if (!canvas || !context) {
      return
    }

    const { width, height } = resizeCanvas(canvas, context)
    drawGrid(context, width, height)
    if (snapshot) {
      drawFly(context, snapshot, width, height)
    }
    drawSwatter(context, input, snapshot?.swatter.radius ?? 0.09, width, height)

    if (snapshot && !snapshot.alive) {
      context.fillStyle = 'rgba(255, 81, 47, 0.14)'
      context.fillRect(0, 0, width, height)
    }
  }, [snapshot, input])

  useEffect(
    () => () => {
      window.clearTimeout(strikeTimerRef.current)
    },
    [],
  )

  const updatePosition = (event: PointerEvent<HTMLCanvasElement>) => {
    const { position, aspectRatio } = pointerState(event)
    positionRef.current = position
    aspectRatioRef.current = aspectRatio
    onInputChange({
      swatter_position: position,
      attacking: strikeActiveRef.current,
      arena_aspect_ratio: aspectRatio,
    })
  }

  const strike = (event: PointerEvent<HTMLCanvasElement>) => {
    if (strikeLockedRef.current || strikeActiveRef.current) {
      return
    }

    const { position, aspectRatio } = pointerState(event)
    positionRef.current = position
    aspectRatioRef.current = aspectRatio
    strikeLockedRef.current = true
    strikeActiveRef.current = true
    event.currentTarget.setPointerCapture(event.pointerId)
    onInputChange({ swatter_position: position, attacking: true, arena_aspect_ratio: aspectRatio })

    strikeTimerRef.current = window.setTimeout(() => {
      strikeActiveRef.current = false
      onInputChange({
        swatter_position: positionRef.current,
        attacking: false,
        arena_aspect_ratio: aspectRatioRef.current,
      })
    }, UI_CONFIG.strikeDurationMS)
  }

  const release = (event: PointerEvent<HTMLCanvasElement>) => {
    strikeLockedRef.current = false
    if (event.currentTarget.hasPointerCapture(event.pointerId)) {
      event.currentTarget.releasePointerCapture(event.pointerId)
    }
  }

  const cancel = () => {
    window.clearTimeout(strikeTimerRef.current)
    strikeActiveRef.current = false
    strikeLockedRef.current = false
    onInputChange({
      swatter_position: positionRef.current,
      attacking: false,
      arena_aspect_ratio: aspectRatioRef.current,
    })
  }

  return (
    <canvas
      ref={canvasRef}
      className={`game-canvas ${input.attacking ? 'game-canvas--striking' : ''}`}
      style={{ '--strike-duration': `${UI_CONFIG.strikeDurationMS}ms` } as CSSProperties}
      aria-label="Fly arena. Move the pointer to aim and click to strike."
      onPointerMove={updatePosition}
      onPointerDown={strike}
      onPointerUp={release}
      onPointerCancel={cancel}
    />
  )
}

function resizeCanvas(canvas: HTMLCanvasElement, context: CanvasRenderingContext2D) {
  const bounds = canvas.getBoundingClientRect()
  const width = Math.max(bounds.width, 1)
  const height = Math.max(bounds.height, 1)
  const pixelRatio = window.devicePixelRatio || 1
  const scaledWidth = Math.round(width * pixelRatio)
  const scaledHeight = Math.round(height * pixelRatio)

  if (canvas.width !== scaledWidth || canvas.height !== scaledHeight) {
    canvas.width = scaledWidth
    canvas.height = scaledHeight
  }
  context.setTransform(pixelRatio, 0, 0, pixelRatio, 0, 0)
  context.clearRect(0, 0, width, height)
  return { width, height }
}

function drawGrid(context: CanvasRenderingContext2D, width: number, height: number) {
  context.strokeStyle = 'rgba(151, 255, 84, 0.08)'
  context.lineWidth = 1
  const size = 36

  for (let x = size; x < width; x += size) {
    context.beginPath()
    context.moveTo(x, 0)
    context.lineTo(x, height)
    context.stroke()
  }
  for (let y = size; y < height; y += size) {
    context.beginPath()
    context.moveTo(0, y)
    context.lineTo(width, y)
    context.stroke()
  }
}

function drawFly(context: CanvasRenderingContext2D, snapshot: Snapshot, width: number, height: number) {
  const x = snapshot.fly.position.x * width
  const y = snapshot.fly.position.y * height
  const size = Math.max(snapshot.fly.radius * Math.min(width, height), 9)

  context.save()
  context.translate(x, y)
  context.rotate(snapshot.fly.heading)

  context.fillStyle = 'rgba(213, 247, 255, 0.68)'
  context.beginPath()
  context.ellipse(-size * 0.25, -size * 0.72, size * 0.85, size * 0.38, -0.35, 0, Math.PI * 2)
  context.ellipse(-size * 0.25, size * 0.72, size * 0.85, size * 0.38, 0.35, 0, Math.PI * 2)
  context.fill()

  context.fillStyle = '#c7ff54'
  context.beginPath()
  context.ellipse(0, 0, size, size * 0.52, 0, 0, Math.PI * 2)
  context.fill()

  context.fillStyle = '#101510'
  context.beginPath()
  context.arc(size * 0.56, -size * 0.25, size * 0.18, 0, Math.PI * 2)
  context.arc(size * 0.56, size * 0.25, size * 0.18, 0, Math.PI * 2)
  context.fill()

  context.restore()
}

function drawSwatter(
  context: CanvasRenderingContext2D,
  input: GameInput,
  radius: number,
  width: number,
  height: number,
) {
  const x = input.swatter_position.x * width
  const y = input.swatter_position.y * height
  const size = radius * Math.min(width, height)

  if (input.attacking) {
    drawImpact(context, x, y, size)
  }

  context.save()
  context.translate(x, y)
  context.rotate(-Math.PI / 4)
  context.strokeStyle = input.attacking ? '#ff512f' : '#ff8a4c'
  context.fillStyle = 'rgba(255, 112, 67, 0.08)'
  context.lineWidth = input.attacking ? 4.5 : 3.5
  context.globalAlpha = input.attacking ? 1 : 0.78
  context.shadowColor = input.attacking ? '#ff512f' : 'rgba(255, 112, 67, 0.45)'
  context.shadowBlur = input.attacking ? 18 : 7
  context.lineCap = 'round'

  context.beginPath()
  context.moveTo(0, size * 0.82)
  context.lineTo(0, size * 2.5)
  context.stroke()

  context.shadowBlur = 0
  context.strokeStyle = '#3e2118'
  context.lineWidth = size * 0.24
  context.beginPath()
  context.moveTo(0, size * 1.62)
  context.lineTo(0, size * 2.52)
  context.stroke()

  context.strokeStyle = input.attacking ? '#ff512f' : '#ff8a4c'
  context.lineWidth = size * 0.11
  context.beginPath()
  context.moveTo(0, size * 1.6)
  context.lineTo(0, size * 2.48)
  context.stroke()

  context.strokeStyle = input.attacking ? '#ff512f' : '#ff8a4c'
  context.fillStyle = input.attacking ? 'rgba(255, 81, 47, 0.18)' : 'rgba(255, 112, 67, 0.07)'
  context.lineWidth = input.attacking ? 4.5 : 3.5
  context.shadowColor = input.attacking ? '#ff512f' : 'rgba(255, 112, 67, 0.45)'
  context.shadowBlur = input.attacking ? 18 : 7
  context.beginPath()
  context.ellipse(0, 0, size * UI_CONFIG.swatterWidthRatio, size, 0, 0, Math.PI * 2)
  context.fill()
  context.stroke()

  context.shadowBlur = 0
  context.lineWidth = input.attacking ? 1.6 : 1
  for (let offset = -0.54; offset <= 0.54; offset += 0.27) {
    context.beginPath()
    context.moveTo(-size * 0.62, size * offset)
    context.lineTo(size * 0.62, size * offset)
    context.stroke()
    context.beginPath()
    context.moveTo(size * offset, -size * 0.82)
    context.lineTo(size * offset, size * 0.82)
    context.stroke()
  }
  context.restore()
}

function drawImpact(context: CanvasRenderingContext2D, x: number, y: number, size: number) {
  context.save()
  context.strokeStyle = '#c7ff54'
  context.fillStyle = '#c7ff54'
  context.lineWidth = 2
  context.globalAlpha = 0.82
  context.shadowColor = '#c7ff54'
  context.shadowBlur = 12

  for (const scale of [1.12, 1.38]) {
    context.beginPath()
    context.arc(x, y, size * scale, 0, Math.PI * 2)
    context.stroke()
  }

  for (let index = 0; index < 10; index++) {
    const angle = (Math.PI * 2 * index) / 10
    context.beginPath()
    context.moveTo(x + Math.cos(angle) * size * 1.48, y + Math.sin(angle) * size * 1.48)
    context.lineTo(x + Math.cos(angle) * size * 1.82, y + Math.sin(angle) * size * 1.82)
    context.stroke()

    context.beginPath()
    context.arc(
      x + Math.cos(angle + 0.2) * size * 1.64,
      y + Math.sin(angle + 0.2) * size * 1.64,
      1.8,
      0,
      Math.PI * 2,
    )
    context.fill()
  }
  context.restore()
}

function pointerState(event: PointerEvent<HTMLCanvasElement>): { position: Vec2; aspectRatio: number } {
  const bounds = event.currentTarget.getBoundingClientRect()
  return {
    position: {
      x: clamp((event.clientX - bounds.left) / bounds.width),
      y: clamp((event.clientY - bounds.top) / bounds.height),
    },
    aspectRatio: bounds.width / bounds.height,
  }
}

function clamp(value: number) {
  return Math.max(0, Math.min(1, value))
}
