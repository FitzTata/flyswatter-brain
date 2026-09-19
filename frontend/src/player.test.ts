import { describe, expect, it } from 'vitest'
import { playerIdFromName } from './player'
import { websocketURL } from './useGameSocket'

describe('player identity', () => {
  it('builds stable ids from names', () => {
    expect(playerIdFromName('Alex')).toBe('alex')
    expect(playerIdFromName('  Cool Player! ')).toBe('cool-player')
  })
})

describe('websocketURL', () => {
  it('includes session player and mode', () => {
    const url = websocketURL('alex', 'shared')

    expect(url).toContain('session=alex')
    expect(url).toContain('player=alex')
    expect(url).toContain('mode=shared')
  })
})
