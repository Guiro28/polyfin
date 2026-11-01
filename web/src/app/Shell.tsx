import { useEffect, useState, type ReactNode } from 'react'
import type { SessionUser } from '@/api'
import { useI18n } from '@/i18n'
import { Brand, LanguageSwitch } from './Brand'
import { CommandPaletteProvider } from './palette/CommandPalette'
import { SessionContext } from './session'
import { Sidebar } from './Sidebar'
import { ThemeToggle } from './ThemeToggle'
import { TopBar, useWebClient } from './TopBar'

function SkipLink() {
  const { t } = useI18n()
  return (
    <a
      href="#main"
      className="fixed top-[-48px] left-4 z-[70] rounded-field bg-accent px-3 py-2 text-control font-medium text-white transition-[top] duration-160 focus:top-3"
    >
      {t.nav.skipToContent}
    </a>
  )
}

/**
 * The signed-in frame, as Stremfin's: the sidebar on the left, and beside it the top bar over the
 * page, in a column of 1320 px at most, and the command palette. Below 1024 px the sidebar
 * slides in from the top bar's menu button; Escape closes it. It provides the session to
 * `useSessionUser`.
 */
export function AppShell({ user, children }: { user: SessionUser; children: ReactNode }) {
  const [menuOpen, setMenuOpen] = useState(false)
  const webClient = useWebClient()
  useEffect(() => {
    if (!menuOpen) return
    const close = (event: KeyboardEvent) => {
      if (event.key === 'Escape') setMenuOpen(false)
    }
    window.addEventListener('keydown', close)
    return () => window.removeEventListener('keydown', close)
  }, [menuOpen])
  return (
    <SessionContext value={user}>
      <CommandPaletteProvider>
        <SkipLink />
        <div className="aurora-shell">
          <Sidebar
            user={user}
            webClient={webClient}
            open={menuOpen}
            onClose={() => setMenuOpen(false)}
          />
          <div className="min-w-0">
            <TopBar user={user} menuOpen={menuOpen} onMenu={() => setMenuOpen((open) => !open)} />
            <main
              id="main"
              tabIndex={-1}
              className="w-full max-w-[1320px] px-9 pt-8 pb-24 outline-none max-xl:px-7 max-md:px-4 max-md:pt-6 max-md:pb-16"
            >
              {children}
            </main>
          </div>
        </div>
      </CommandPaletteProvider>
    </SessionContext>
  )
}

/** The frame without a session (setup, sign-in, loading): the logo, the language, one column. */
export function PublicShell({ children }: { children: ReactNode }) {
  return (
    <div className="flex min-h-dvh flex-col">
      <SkipLink />
      <header className="border-b border-line">
        <div className="mx-auto flex h-topbar w-full max-w-page items-center justify-between gap-4 px-14 max-xl:px-8 max-md:px-5">
          <Brand link={false} />
          <div className="flex items-center gap-2">
            <ThemeToggle />
            <LanguageSwitch />
          </div>
        </div>
      </header>
      <main
        id="main"
        tabIndex={-1}
        className="mx-auto w-full max-w-[880px] flex-1 animate-rise px-5 py-12 outline-none max-md:py-8"
      >
        {children}
      </main>
    </div>
  )
}
