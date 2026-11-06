import { useEffect, useState, type ReactNode } from 'react'
import type { SessionUser } from '@/api'
import { AuroraTopBar } from './AuroraTopBar'
import { Sidebar } from './Sidebar'
import { useWebClient } from './TopBar'

/**
 * The signed-in frame of the Aurora skin, as Stremfin's: the sidebar on the left, and beside it
 * the top bar over the page. Below 1024 px the sidebar slides in from the top bar's menu button;
 * Escape closes it.
 */
export function AuroraFrame({ user, children }: { user: SessionUser; children: ReactNode }) {
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
    <div className="aurora-shell">
      <Sidebar
        user={user}
        webClient={webClient}
        open={menuOpen}
        onClose={() => setMenuOpen(false)}
      />
      <div className="min-w-0">
        <AuroraTopBar user={user} menuOpen={menuOpen} onMenu={() => setMenuOpen((open) => !open)} />
        <main
          id="main"
          tabIndex={-1}
          className="w-full max-w-[1320px] px-9 pt-8 pb-24 outline-none max-xl:px-7 max-md:px-4 max-md:pt-6 max-md:pb-16"
        >
          {children}
        </main>
      </div>
    </div>
  )
}
