import { describe, expect, it } from 'vitest'
import { decodeServerMessage } from './types'

describe('decodeServerMessage', () => {
  const snapshot = {
    tick: 1,
    episode: 2,
    alive: true,
    fly_hp: 1,
    survival_ms: 50,
    last_action: 'straight',
    fly: {
      position: { x: 0.5, y: 0.5 },
      heading: 0,
      speed: 0.18,
      radius: 0.025,
    },
    swatter: {
      position: { x: 0.2, y: 0.3 },
      radius: 0.09,
      attacking: false,
    },
  }

  it('decodes a snapshot', () => {
    const result = decodeServerMessage(JSON.stringify({ type: 'snapshot', snapshot }))

    expect(result).toEqual({ type: 'snapshot', snapshot })
  })

  it.each(['not json', '{}', '{"type":"snapshot","snapshot":{}}'])('rejects invalid payload %s', (raw) => {
    expect(decodeServerMessage(raw)).toBeNull()
  })
})
