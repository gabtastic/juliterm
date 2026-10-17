import { ref } from 'vue'

// Module-scoped so MobileKeyBar and TerminalView share touch state.

/** While on, a one-finger drag selects text instead of scrolling. */
export const selectMode = ref(false)

// TerminalView registers how to paste into the terminal (xterm's paste path,
// which wraps the text in bracketed-paste markers when the app asked for it).
let pasteTarget: ((text: string) => void) | null = null

export function setPasteTarget(fn: ((text: string) => void) | null) {
  pasteTarget = fn
}

export function pasteIntoTerminal(text: string) {
  pasteTarget?.(text)
}
