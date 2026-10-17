// Safari (notably on iPadOS) draws characters whose default presentation is
// text with a text font even when a color emoji font is listed, so Claude
// Code's ⏺ bullet shows up as a small dot instead of the emoji glyph native
// terminals draw. Appending VARIATION SELECTOR-16 asks for the emoji glyph.
// xterm treats it as a zero-width combining mark, so the layout tmux computed
// is unchanged. Works on raw UTF-8 bytes so a character split across output
// frames is still handled.

// Media-control symbols that render as emoji in native macOS terminals:
// ⏸ U+23F8, ⏹ U+23F9, ⏺ U+23FA (UTF-8: E2 8F B8..BA).
const LEAD = 0xe2
const SECOND = 0x8f
const isTarget = (b: number) => b >= 0xb8 && b <= 0xba
const VS16 = [0xef, 0xb8, 0x8f]

/**
 * Returns a stateful filter for one output stream. Call `reset()` when the
 * stream restarts (session switch or reconnect) to drop any held bytes.
 */
export function createEmojiPresentation() {
  // Up to two trailing bytes that may begin a target character, held back
  // until the next chunk shows whether they do.
  let carry: number[] = []

  function filter(chunk: Uint8Array): Uint8Array {
    const input = carry.length ? Uint8Array.from([...carry, ...chunk]) : chunk
    carry = []
    let out: number[] | null = null
    let copied = 0

    for (let i = 0; i < input.length; i++) {
      if (input[i] !== LEAD) continue
      // An incomplete prefix at the end of the chunk: hold it back.
      if (i + 1 === input.length || (i + 2 === input.length && input[i + 1] === SECOND)) {
        out ??= []
        for (let j = copied; j < i; j++) out.push(input[j]!)
        carry = Array.from(input.subarray(i))
        copied = input.length
        break
      }
      if (input[i + 1] !== SECOND || !isTarget(input[i + 2]!)) continue
      const end = i + 3
      const hasVS16 = input[end] === VS16[0] && input[end + 1] === VS16[1] && input[end + 2] === VS16[2]
      if (hasVS16) continue
      out ??= []
      for (let j = copied; j < end; j++) out.push(input[j]!)
      out.push(...VS16)
      copied = end
      i = end - 1
    }

    if (!out) return input
    for (let j = copied; j < input.length; j++) out.push(input[j]!)
    return Uint8Array.from(out)
  }

  function reset() {
    carry = []
  }

  return { filter, reset }
}
