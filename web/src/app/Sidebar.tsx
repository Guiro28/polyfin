import { ArrowSquareOutIcon, CaretDownIcon, GithubLogoIcon, PlayIcon } from '@phosphor-icons/react'
import { useState } from 'react'
import { Link, useLocation } from 'react-router'
import type { SessionUser } from '@/api'
import { useI18n } from '@/i18n'
import { webPlayerLink } from '@/jellyfinApps'
import { cx } from '@/ui'
import { repositoryUrl } from './Brand'
import {
  adminSidebar,
  isCurrent,
  isCurrentPage,
  memberSidebar,
  type NavPage,
  type NavSection,
} from './navigation'

/** Where the sidebar remembers the sections the user folded or unfolded. */
const foldsKey = 'polyfin.sidebar'

/** The sections open until the user folds them; the current one is always open. */
const openByDefault = new Set(['content', 'system'])

function readFolds(): Record<string, boolean> {
  try {
    return JSON.parse(localStorage.getItem(foldsKey) ?? '{}') as Record<string, boolean>
  } catch {
    return {}
  }
}

function saveFolds(folds: Record<string, boolean>) {
  try {
    localStorage.setItem(foldsKey, JSON.stringify(folds))
  } catch {
    // Without storage, the folds last as long as the page.
  }
}

/**
 * The sidebar of the signed-in pages, as in Stremfin and Nuvio Manager: the logo, the pages that
 * stand alone, each section with its pages under it, which folds, the web player and the source
 * code. From 1024 px it stays on the left; below, it slides in over the page from the menu button
 * of the top bar, and closes on a choice.
 */
export function Sidebar({
  user,
  webClient,
  open,
  onClose,
}: {
  user: SessionUser
  webClient: boolean
  open: boolean
  onClose: () => void
}) {
  const { t } = useI18n()
  const { pathname } = useLocation()
  const [folds, setFolds] = useState(readFolds)
  const sections = user.isAdministrator ? adminSidebar : memberSidebar

  const toggle = (section: NavSection, isOpen: boolean) => {
    const next = { ...folds, [section.id]: !isOpen }
    setFolds(next)
    saveFolds(next)
  }

  return (
    <>
      <div
        aria-hidden="true"
        onClick={onClose}
        className={cx(
          'aurora-scrim lg:hidden',
          open ? 'opacity-100' : 'pointer-events-none opacity-0',
        )}
      />
      <aside id="sidebar" className={cx('aurora-sidebar', open && 'is-open')}>
        <Link to="/" className="aurora-brand" onClick={onClose}>
          <img src={`${import.meta.env.BASE_URL}polyfin.svg`} alt="" width={36} height={36} />
          <span className="min-w-0">
            <span className="block text-[18px] leading-tight font-bold tracking-[0.02em] text-ink">
              {t.header.productName}
            </span>
            <span className="block text-[11px] text-ink-3">
              {user.isAdministrator ? t.nav.adminArea : t.nav.memberArea}
            </span>
          </span>
        </Link>
        <nav aria-label={t.nav.label} className="flex flex-col gap-0.5">
          {sections.map((section) =>
            section.pages ? (
              <SidebarGroup
                key={section.id}
                section={section}
                pathname={pathname}
                current={isCurrent(section, pathname)}
                isOpen={
                  isCurrent(section, pathname) ||
                  (folds[section.id] ?? openByDefault.has(section.id))
                }
                onToggle={toggle}
                onNavigate={onClose}
              />
            ) : (
              <SidebarLink
                key={section.id}
                page={section}
                current={isCurrent(section, pathname)}
                onNavigate={onClose}
              />
            ),
          )}
          {webClient && (
            <a {...webPlayerLink(false)} className="aurora-nav-item mt-3" onClick={onClose}>
              <PlayIcon size={18} aria-hidden="true" />
              <span className="flex-1 truncate">{t.nav.webPlayer}</span>
              <ArrowSquareOutIcon size={14} aria-hidden="true" className="text-ink-3" />
              <span className="sr-only">{t.nav.webPlayerHint}</span>
            </a>
          )}
        </nav>
        <div className="aurora-side-foot">
          <a
            href={repositoryUrl}
            target="_blank"
            rel="noopener noreferrer"
            aria-label="GitHub"
            title="GitHub"
            className="aurora-social"
          >
            <GithubLogoIcon size={18} aria-hidden="true" />
          </a>
          <span className="ml-auto text-[11.5px] tracking-[0.02em] text-ink-3">Polyfin Aurora</span>
        </div>
      </aside>
    </>
  )
}

/** A section with its pages under it, folded or not; the current section stays open. */
function SidebarGroup({
  section,
  pathname,
  isOpen,
  current,
  onToggle,
  onNavigate,
}: {
  section: NavSection
  pathname: string
  isOpen: boolean
  current: boolean
  onToggle: (section: NavSection, isOpen: boolean) => void
  onNavigate: () => void
}) {
  const { t } = useI18n()
  const id = `sidebar-${section.id}`
  return (
    <div className="mt-3">
      <button
        type="button"
        aria-expanded={isOpen}
        aria-controls={id}
        disabled={current}
        onClick={() => onToggle(section, isOpen)}
        className="aurora-nav-group"
      >
        <span className="flex-1 text-left">{section.label(t)}</span>
        {!current && (
          <CaretDownIcon
            size={12}
            aria-hidden="true"
            className={cx('transition-transform duration-160', !isOpen && '-rotate-90')}
          />
        )}
      </button>
      <div id={id} hidden={!isOpen} className="flex flex-col gap-0.5">
        {section.pages!.map((page) => (
          <SidebarLink
            key={page.id}
            page={page}
            current={isCurrentPage(page, pathname)}
            onNavigate={onNavigate}
          />
        ))}
      </div>
    </div>
  )
}

/** A page of the sidebar, its icon and name, lit when current. */
function SidebarLink({
  page,
  current,
  onNavigate,
}: {
  page: NavPage
  current: boolean
  onNavigate: () => void
}) {
  const { t } = useI18n()
  return (
    <Link
      to={page.to}
      aria-current={current ? 'page' : undefined}
      className="aurora-nav-item"
      onClick={onNavigate}
    >
      <page.icon size={18} aria-hidden="true" />
      <span className="truncate">{page.label(t)}</span>
    </Link>
  )
}
