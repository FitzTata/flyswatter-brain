import { useEffect, useRef, type PointerEvent } from 'react'
import type { GameInput, Snapshot, Vec2 } from './types'

interface GameCanvasProps {
  snapshot: Snapshot | null
  input: GameInput
  onInputChange: (input: GameInput) => void
}

export function GameCanvas({ snapshot, input, onInputChange }: GameCanvasProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null)

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

  const updatePosition = (event: PointerEvent<HTMLCanvasElement>, attacking: boolean) => {
    const position = pointerPosition(event)
    onInputChange({ swatter_position: position, attacking })
  }

  return (
    <canvas
      ref={canvasRef}
      className="game-canvas"
      aria-label="Fly arena. Move the pointer to aim and hold to strike."
      onPointerMove={(event) => updatePosition(event, input.attacking)}
      onPointerDown={(event) => {
        event.currentTarget.setPointerCapture(event.pointerId)
        updatePosition(event, true)
      }}
      onPointerUp={(event) => {
        event.currentTarget.releasePointerCapture(event.pointerId)
        updatePosition(event, false)
      }}
      onPointerCancel={(event) => updatePosition(event, false)}
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
  const size = radius * Math.min(width, height) * (input.attacking ? 0.88 : 1)

  context.save()
  context.strokeStyle = input.attacking ? '#ff512f' : '#ff8a4c'
  context.lineWidth = input.attacking ? 4 : 3
  context.globalAlpha = input.attacking ? 1 : 0.78

  context.beginPath()
  context.moveTo(x + size * 0.62, y + size * 0.62)
  context.lineTo(x + size * 1.8, y + size * 1.8)
  context.stroke()

  context.beginPath()
  context.arc(x, y, size, 0, Math.PI * 2)
  context.stroke()

  context.lineWidth = 1
  for (let offset = -0.55; offset <= 0.55; offset += 0.275) {
    context.beginPath()
    context.moveTo(x - size * 0.72, y + size * offset)
    context.lineTo(x + size * 0.72, y + size * offset)
    context.stroke()
    context.beginPath()
    context.moveTo(x + size * offset, y - size * 0.72)
    context.lineTo(x + size * offset, y + size * 0.72)
    context.stroke()
  }
  context.restore()
}

function pointerPosition(event: PointerEvent<HTMLCanvasElement>): Vec2 {
  const bounds = event.currentTarget.getBoundingClientRect()
  return {
    x: clamp((event.clientX - bounds.left) / bounds.width),
    y: clamp((event.clientY - bounds.top) / bounds.height),
  }
}

function clamp(value: number) {
  return Math.max(0, Math.min(1, value))
}
