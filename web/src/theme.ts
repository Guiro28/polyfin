import { useSyncExternalStore } from 'react'

/** The look chosen in the account menu: the system's, or always light or dark. */
export type ThemeChoice = 'auto' | 'light' | 'dark'

export const themeChoices: readonly ThemeChoice[] = ['auto', 'light', 'dark']

const storageKey = 'polyfin.theme'
const systemDark = window.matchMedia('(prefers-color-scheme: dark)')
const listeners = new Set<() => void>()

function readChoice(): ThemeChoice {
  try {
    const saved = localStorage.getItem(storageKey)
    if (saved === 'light' || saved === 'dark' || saved === 'auto') return saved
  } catch {
    // Storage refused, as in a private window: the system's look.
  }
  return 'auto'
}

let choice = readChoice()

/** Whether the page shows the dark look now: the choice, or the system's under `auto`. */
function isDark(): boolean {
  return choice === 'dark' || (choice === 'auto' && systemDark.matches)
}

/**
 * The look of the admin app, kept in this browser: Aurora, the fork's, with its sidebar and its
 * light and dark tokens (see `aurora.css`), or Salon, Polyfin's own, dark alone.
 */
export const skinChoices = ['aurora', 'salon'] as const
export type SkinChoice = (typeof skinChoices)[number]

const skinKey = 'polyfin.skin'

function readSkin(): SkinChoice {
  try {
    if (localStorage.getItem(skinKey) === 'salon') return 'salon'
  } catch {
    // Storage refused, as in a private window: Aurora.
  }
  return 'aurora'
}

let skin = readSkin()

/** Sets `data-skin` and `data-theme` on <html>, which pick the tokens of `aurora.css`. */
function apply() {
  document.documentElement.dataset.skin = skin
  document.documentElement.dataset.theme = skin === 'salon' || isDark() ? 'dark' : 'light'
  listeners.forEach((listener) => listener())
}

apply()
systemDark.addEventListener('change', () => {
  if (choice === 'auto') apply()
})

/** Keeps the choice in this browser and shows it at once. */
export function setTheme(next: ThemeChoice) {
  try {
    localStorage.setItem(storageKey, next)
  } catch {
    // Kept for this page only.
  }
  choice = next
  apply()
}

/** Keeps the skin in this browser and shows it at once. */
export function setSkin(next: SkinChoice) {
  try {
    localStorage.setItem(skinKey, next)
  } catch {
    // Kept for this page only.
  }
  skin = next
  apply()
}

function subscribe(listener: () => void) {
  listeners.add(listener)
  return () => listeners.delete(listener)
}

/** The skin shown; every user of it follows a change. */
export function useSkin(): SkinChoice {
  return useSyncExternalStore(subscribe, () => skin)
}

/** The choice, and whether the dark look shows now; every user of it follows a change. */
export function useTheme(): { choice: ThemeChoice; dark: boolean } {
  const current = useSyncExternalStore(subscribe, () => choice)
  const dark = useSyncExternalStore(subscribe, isDark)
  return { choice: current, dark }
}
