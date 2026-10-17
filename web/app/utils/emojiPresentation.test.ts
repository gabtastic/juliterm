import { describe, expect, it } from 'vitest'
import { createEmojiPresentation } from './emojiPresentation'

const enc = new TextEncoder()
const dec = (b: Uint8Array) => new TextDecoder().decode(b)

describe('createEmojiPresentation', () => {
  it('adds VS16 after the record, pause and stop symbols', () => {
    const { filter } = createEmojiPresentation()
    expect(dec(filter(enc.encode('⏺ done ⏸ ⏹')))).toBe('⏺️ done ⏸️ ⏹️')
  })

  it('passes other text through unchanged, as the same buffer', () => {
    const { filter } = createEmojiPresentation()
    const chunk = enc.encode('plain ─ │ ● ⚠ text')
    expect(filter(chunk)).toBe(chunk)
  })

  it('does not double an existing VS16', () => {
    const { filter } = createEmojiPresentation()
    expect(dec(filter(enc.encode('⏺️ x')))).toBe('⏺️ x')
  })

  it('handles a symbol split across chunks', () => {
    const { filter } = createEmojiPresentation()
    const bytes = enc.encode('a⏺b')
    for (let split = 1; split < bytes.length; split++) {
      const f = createEmojiPresentation().filter
      const joined = new Uint8Array([...f(bytes.slice(0, split)), ...f(bytes.slice(split))])
      expect(dec(joined)).toBe('a⏺️b')
    }
    expect(dec(filter(bytes))).toBe('a⏺️b')
  })

  it('releases a held lead byte that turns out to be another character', () => {
    const { filter } = createEmojiPresentation()
    const bytes = enc.encode('x─y') // ─ is E2 94 80
    const joined = new Uint8Array([...filter(bytes.slice(0, 2)), ...filter(bytes.slice(2))])
    expect(dec(joined)).toBe('x─y')
  })

  it('drops held bytes on reset', () => {
    const { filter, reset } = createEmojiPresentation()
    filter(Uint8Array.from([0x41, 0xe2]))
    reset()
    expect(dec(filter(enc.encode('ok')))).toBe('ok')
  })
})
