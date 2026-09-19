import { useEffect, useRef, useState, type CSSProperties, type PointerEvent } from 'react'
import { UI_CONFIG } from './config'
import type { GameInput, Snapshot, Vec2 } from './types'

type SwingPhase = 'idle' | 'windup' | 'strike'

interface GameCanvasProps {
  snapshot: Snapshot | null
  input: GameInput
  onInputChange: (input: GameInput) => void
}

export function GameCanvas({ snapshot, input, onInputChange }: GameCanvasProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const positionRef = useRef(input.swatter_position)
  const targetRef = useRef(input.swatter_position)
  const velocityRef = useRef<Vec2>({ x: 0, y: 0 })
  const trailRef = useRef<Vec2[]>([input.swatter_position])
  const aspectRatioRef = useRef(input.arena_aspect_ratio ?? UI_CONFIG.defaultArenaAspectRatio)
  const strikeActiveRef = useRef(false)
  const strikeLockedRef = useRef(false)
  const strikeTimerRef = useRef<number | undefined>(undefined)
  const windupTimerRef = useRef<number | undefined>(undefined)
  const onInputChangeRef = useRef(onInputChange)
  const lastSentRef = useRef({ position: input.swatter_position, attacking: input.attacking })
  const [localPhase, setLocalPhase] = useState<SwingPhase>('idle')
  const [renderTick, setRenderTick] = useState(0)

  const swingPhase: SwingPhase = snapshot?.swatter.phase ?? localPhase

  useEffect(() => {
    onInputChangeRef.current = onInputChange
  }, [onInputChange])

  useEffect(() => {
    let frame = 0
    let previous = performance.now()

    const tick = (now: number) => {
      const dt = Math.min(0.05, Math.max(0.001, (now - previous) / 1000))
      previous = now
      stepSwatterPhysics(positionRef.current, targetRef.current, velocityRef.current, dt)
      positionRef.current = {
        x: clamp(positionRef.current.x),
        y: clamp(positionRef.current.y),
      }
      pushTrail(trailRef.current, positionRef.current)
      publishInput(positionRef.current, strikeActiveRef.current, aspectRatioRef.current, lastSentRef, onInputChangeRef)
      setRenderTick((value) => value + 1)
      frame = window.requestAnimationFrame(tick)
    }

    frame = window.requestAnimationFrame(tick)
    return () => window.cancelAnimationFrame(frame)
  }, [])

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
      drawFlyHP(context, snapshot, width, height)
    }
    drawTrail(context, trailRef.current, snapshot?.swatter.radius ?? 0.085, width, height)
    drawSwatter(
      context,
      positionRef.current,
      snapshot?.swatter.radius ?? 0.085,
      width,
      height,
      swingPhase,
    )

    if (snapshot && !snapshot.alive) {
      context.fillStyle = 'rgba(255, 81, 47, 0.14)'
      context.fillRect(0, 0, width, height)
    }
  }, [snapshot, swingPhase, renderTick])

  useEffect(
    () => () => {
      window.clearTimeout(strikeTimerRef.current)
      window.clearTimeout(windupTimerRef.current)
    },
    [],
  )

  const updateTarget = (event: PointerEvent<HTMLCanvasElement>) => {
    const { position, aspectRatio } = pointerState(event)
    targetRef.current = position
    aspectRatioRef.current = aspectRatio
  }

  const strike = (event: PointerEvent<HTMLCanvasElement>) => {
    if (strikeLockedRef.current || strikeActiveRef.current) {
      return
    }

    updateTarget(event)
    strikeLockedRef.current = true
    strikeActiveRef.current = true
    setLocalPhase('windup')
    event.currentTarget.setPointerCapture(event.pointerId)
    publishInput(positionRef.current, true, aspectRatioRef.current, lastSentRef, onInputChangeRef, true)

    windupTimerRef.current = window.setTimeout(() => {
      setLocalPhase('strike')
    }, UI_CONFIG.swingWindupMS)

    strikeTimerRef.current = window.setTimeout(() => {
      strikeActiveRef.current = false
      setLocalPhase('idle')
      publishInput(positionRef.current, false, aspectRatioRef.current, lastSentRef, onInputChangeRef, true)
    }, UI_CONFIG.swingWindupMS + UI_CONFIG.strikeDurationMS)
  }

  const release = (event: PointerEvent<HTMLCanvasElement>) => {
    strikeLockedRef.current = false
    if (event.currentTarget.hasPointerCapture(event.pointerId)) {
      event.currentTarget.releasePointerCapture(event.pointerId)
    }
  }

  const cancel = () => {
    window.clearTimeout(strikeTimerRef.current)
    window.clearTimeout(windupTimerRef.current)
    strikeActiveRef.current = false
    strikeLockedRef.current = false
    setLocalPhase('idle')
    publishInput(positionRef.current, false, aspectRatioRef.current, lastSentRef, onInputChangeRef, true)
  }

  const canvasClass =
    swingPhase === 'windup'
      ? 'game-canvas game-canvas--windup'
      : swingPhase === 'strike' || input.attacking
        ? 'game-canvas game-canvas--striking'
        : 'game-canvas'

  return (
    <canvas
      ref={canvasRef}
      className={canvasClass}
      style={
        {
          '--strike-duration': `${UI_CONFIG.strikeDurationMS}ms`,
          '--swing-windup': `${UI_CONFIG.swingWindupMS}ms`,
        } as CSSProperties
      }
      aria-label="Fly arena. Move the pointer to aim and click to strike."
      onPointerMove={updateTarget}
      onPointerDown={strike}
      onPointerUp={release}
      onPointerCancel={cancel}
    />
  )
}

function stepSwatterPhysics(position: Vec2, target: Vec2, velocity: Vec2, dt: number) {
  const dx = target.x - position.x
  const dy = target.y - position.y
  velocity.x += dx * UI_CONFIG.swatterFollowGain * dt
  velocity.y += dy * UI_CONFIG.swatterFollowGain * dt
  const damp = Math.exp(-UI_CONFIG.swatterDamping * dt)
  velocity.x *= damp
  velocity.y *= damp

  const speed = Math.hypot(velocity.x, velocity.y)
  if (speed > UI_CONFIG.swatterMaxSpeed && speed > 0) {
    const scale = UI_CONFIG.swatterMaxSpeed / speed
    velocity.x *= scale
    velocity.y *= scale
  }

  position.x += velocity.x * dt
  position.y += velocity.y * dt
}

function pushTrail(trail: Vec2[], position: Vec2) {
  const last = trail[trail.length - 1]
  if (last && Math.hypot(last.x - position.x, last.y - position.y) < 0.002) {
    return
  }
  trail.push({ x: position.x, y: position.y })
  while (trail.length > UI_CONFIG.swatterTrailLength) {
    trail.shift()
  }
}

function publishInput(
  position: Vec2,
  attacking: boolean,
  aspectRatio: number,
  lastSentRef: { current: { position: Vec2; attacking: boolean } },
  onInputChangeRef: { current: (input: GameInput) => void },
  force = false,
) {
  const moved =
    Math.hypot(position.x - lastSentRef.current.position.x, position.y - lastSentRef.current.position.y) > 0.0008
  if (!force && !moved && attacking === lastSentRef.current.attacking) {
    return
  }
  lastSentRef.current = { position: { ...position }, attacking }
  onInputChangeRef.current({
    swatter_position: { ...position },
    attacking,
    arena_aspect_ratio: aspectRatio,
  })
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

function drawFlyHP(context: CanvasRenderingContext2D, snapshot: Snapshot, width: number, height: number) {
  const x = snapshot.fly.position.x * width
  const y = snapshot.fly.position.y * height
  const size = Math.max(snapshot.fly.radius * Math.min(width, height), 9)
  const barWidth = Math.max(34, size * 2.4)
  const barHeight = 4
  const left = x - barWidth / 2
  const top = y - size - 12
  const hp = Math.max(0, Math.min(1, snapshot.fly_hp))

  context.fillStyle = 'rgba(8, 11, 9, 0.72)'
  context.fillRect(left - 1, top - 1, barWidth + 2, barHeight + 2)
  context.fillStyle = 'rgba(218, 255, 196, 0.18)'
  context.fillRect(left, top, barWidth, barHeight)
  context.fillStyle = hp > 0.4 ? '#c7ff54' : '#ff7043'
  context.fillRect(left, top, barWidth * hp, barHeight)
}

function drawTrail(context: CanvasRenderingContext2D, trail: Vec2[], radius: number, width: number, height: number) {
  if (trail.length < 2) {
    return
  }
  const size = radius * Math.min(width, height)
  for (let index = 0; index < trail.length - 1; index++) {
    const point = trail[index]
    const t = (index + 1) / trail.length
    const alpha = 0.08 + t * 0.22
    const scale = 0.92 + t * 0.08

    context.save()
    context.translate(point.x * width, point.y * height)
    context.rotate(-Math.PI / 4)
    context.globalAlpha = alpha
    context.strokeStyle = '#ff8a4c'
    context.lineWidth = 1.4 + t
    context.setLineDash([3, 4])
    context.beginPath()
    context.ellipse(0, 0, size * UI_CONFIG.swatterWidthRatio * scale, size * scale, 0, 0, Math.PI * 2)
    context.stroke()
    context.setLineDash([])
    context.beginPath()
    context.moveTo(0, size * 0.82 * scale)
    context.lineTo(0, size * 2.15 * scale)
    context.stroke()
    context.restore()
  }
}

function drawSwatter(
  context: CanvasRenderingContext2D,
  position: Vec2,
  radius: number,
  width: number,
  height: number,
  phase: SwingPhase,
) {
  const x = position.x * width
  const y = position.y * height
  const size = radius * Math.min(width, height)
  const striking = phase === 'strike'
  const winding = phase === 'windup'

  if (striking) {
    drawImpact(context, x, y, size)
  }

  context.save()
  context.translate(x, y)
  context.rotate(-Math.PI / 4 + (winding ? -0.85 : striking ? 0.2 : 0))
  if (winding) {
    context.translate(0, -size * 0.35)
  }
  context.strokeStyle = striking || winding ? '#ff512f' : '#ff8a4c'
  context.fillStyle = 'rgba(255, 112, 67, 0.08)'
  context.lineWidth = striking || winding ? 4.5 : 3.5
  context.globalAlpha = striking || winding ? 1 : 0.78
  context.shadowColor = striking || winding ? '#ff512f' : 'rgba(255, 112, 67, 0.45)'
  context.shadowBlur = striking ? 18 : winding ? 12 : 7
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

  context.strokeStyle = striking || winding ? '#ff512f' : '#ff8a4c'
  context.lineWidth = size * 0.11
  context.beginPath()
  context.moveTo(0, size * 1.6)
  context.lineTo(0, size * 2.48)
  context.stroke()

  context.strokeStyle = striking || winding ? '#ff512f' : '#ff8a4c'
  context.fillStyle = striking
    ? 'rgba(255, 81, 47, 0.18)'
    : winding
      ? 'rgba(255, 112, 67, 0.14)'
      : 'rgba(255, 112, 67, 0.07)'
  context.lineWidth = striking || winding ? 4.5 : 3.5
  context.shadowColor = striking || winding ? '#ff512f' : 'rgba(255, 112, 67, 0.45)'
  context.shadowBlur = striking ? 18 : winding ? 12 : 7
  context.beginPath()
  context.ellipse(0, 0, size * UI_CONFIG.swatterWidthRatio, size, 0, 0, Math.PI * 2)
  context.fill()
  context.stroke()

  context.shadowBlur = 0
  context.lineWidth = striking ? 1.6 : 1
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
