import {
  ArrowLeftIcon,
  CaretDownIcon,
  GithubLogoIcon,
  ListIcon,
  MagnifyingGlassIcon,
  SignOutIcon,
} from '@phosphor-icons/react'
import { useLocation } from 'react-router'
import { userImageUrl, type SessionUser } from '@/api'
import { languages, useI18n } from '@/i18n'
import { webPlayerInPlace, webPlayerLink } from '@/jellyfinApps'
import { setSkin, setTheme, skinChoices, themeChoices, useSkin, useTheme } from '@/theme'
import {
  Avatar,
  cx,
  Kbd,
  Menu,
  MenuChoice,
  MenuHeader,
  MenuItem,
  MenuSeparator,
  modKey,
} from '@/ui'
import { repositoryUrl } from './Brand'
import { ThemeToggle } from './ThemeToggle'
import { useWebClient } from './TopBar'
import { accountPages, adminSidebar, memberSidebar, whereIs } from './navigation'
import { useOpenPalette } from './palette/CommandPalette'
import { useSignOut } from './session'

/**
 * The bar on top of every signed-in page, beside the sidebar, as in Stremfin: below 1024 px the
 * button that opens the sidebar, the trail of the page (Administration / Content / Sources),
 * then search, the way back to the web player when opened from it, the theme and the account
 * menu. `data-top-bar` marks it for the lists of `Select` and `SuggestInput`, which stay under it.
 */
export function AuroraTopBar({
  user,
  menuOpen,
  onMenu,
}: {
  user: SessionUser
  menuOpen: boolean
  onMenu: () => void
}) {
  const { t } = useI18n()
  const { pathname } = useLocation()
  const openPalette = useOpenPalette()
  const webClient = useWebClient()
  const inPlace = webPlayerInPlace()
  const { section, page } = whereIs(user.isAdministrator ? adminSidebar : memberSidebar, pathname)
  const trail = [
    user.isAdministrator ? t.nav.adminArea : t.nav.memberArea,
    section?.label(t),
    page?.label(t),
  ].filter((part): part is string => part !== undefined)

  return (
    <header data-top-bar className="aurora-header">
      <div className="flex min-w-0 items-center gap-2.5">
        <button
          type="button"
          onClick={onMenu}
          aria-label={t.nav.menu}
          aria-expanded={menuOpen}
          aria-controls="sidebar"
          className="inline-grid size-9 shrink-0 cursor-pointer place-items-center rounded-field text-ink-2 transition-[background-color,color] duration-160 ease-nuit hover:bg-ink/8 hover:text-ink lg:hidden"
        >
          <ListIcon size={20} aria-hidden="true" />
        </button>
        <p className="aurora-trail">
          {trail.map((part, index) => (
            <span key={index} aria-current={index === trail.length - 1 ? 'page' : undefined}>
              {part}
            </span>
          ))}
        </p>
      </div>
      <div className="ml-auto flex items-center gap-2">
        <button
          type="button"
          onClick={openPalette}
          aria-label={t.nav.searchShortcut(`${modKey} K`)}
          aria-keyshortcuts="Meta+K Control+K /"
          className={cx(
            'flex h-9 cursor-pointer items-center gap-2.5 rounded-field border border-line-2 bg-s2 pr-1.5 pl-3 text-control text-ink-3 transition-[background-color,border-color] duration-160 ease-nuit hover:border-line-3 hover:bg-s3',
            'w-[248px] max-xl:w-auto max-md:w-9 max-md:justify-center max-md:border-transparent max-md:bg-transparent max-md:p-0 max-md:text-ink-2',
          )}
        >
          <MagnifyingGlassIcon size={16} aria-hidden="true" className="shrink-0" />
          <span className="flex-1 text-left max-xl:hidden">{t.nav.search}</span>
          <Kbd className="max-md:hidden">{modKey === '⌘' ? '⌘K' : 'Ctrl K'}</Kbd>
        </button>
        {webClient && inPlace && (
          <a
            {...webPlayerLink(true)}
            title={t.nav.backToWebPlayer}
            className="inline-flex h-9 items-center gap-2 rounded-field px-3 text-control font-medium whitespace-nowrap text-ink-2 transition-[background-color,color] duration-160 ease-nuit hover:bg-s2 hover:text-ink"
          >
            <ArrowLeftIcon size={16} aria-hidden="true" />
            <span className="max-lg:sr-only">{t.nav.backToWebPlayer}</span>
          </a>
        )}
        <ThemeToggle />
        <AccountMenu user={user} />
      </div>
    </header>
  )
}

/**
 * The account button and its menu: who is signed in, an administrator's own pages, the
 * language, the theme, the source code and signing out.
 */
function AccountMenu({ user }: { user: SessionUser }) {
  const skin = useSkin()
  const { language, setLanguage, t } = useI18n()
  const theme = useTheme().choice
  const signOut = useSignOut()
  return (
    <Menu
      label={t.nav.accountMenu}
      align="end"
      width={264}
      trigger={(props) => (
        <button
          {...props}
          aria-label={t.nav.accountOf(user.name)}
          className="flex h-9 cursor-pointer items-center gap-1.5 rounded-field pr-1.5 pl-1 text-ink-3 transition-colors duration-160 ease-nuit hover:bg-s2 aria-expanded:bg-s2"
        >
          <Avatar name={user.name} image={userImageUrl(user)} />
          <CaretDownIcon size={14} aria-hidden="true" />
        </button>
      )}
    >
      <MenuHeader>
        <Avatar name={user.name} image={userImageUrl(user)} />
        <div className="min-w-0">
          <p className="truncate text-body font-semibold text-ink">{user.name}</p>
          <p className="text-[12.5px] text-ink-3">
            {user.isAdministrator ? t.nav.administrator : t.nav.member}
          </p>
        </div>
      </MenuHeader>
      <MenuSeparator />
      {user.isAdministrator && (
        <>
          {accountPages.map((page) => (
            <MenuItem key={page.id} icon={page.icon} to={page.to}>
              {page.label(t)}
            </MenuItem>
          ))}
          <MenuSeparator />
        </>
      )}
      <MenuChoice
        label={t.nav.skin}
        value={skin}
        onChange={setSkin}
        options={skinChoices.map((choice) => ({ value: choice, label: t.nav.skins[choice] }))}
      />
      <MenuChoice
        label={t.nav.language}
        value={language}
        onChange={setLanguage}
        options={languages.map((code) => ({
          value: code,
          label: t.language[code].short,
          title: t.language[code].name,
          lang: code,
        }))}
      />
      <MenuChoice
        label={t.nav.theme}
        value={theme}
        onChange={setTheme}
        options={themeChoices.map((choice) => ({
          value: choice,
          label: { auto: t.nav.themeAuto, light: t.nav.themeLight, dark: t.nav.themeDark }[choice],
        }))}
      />
      <MenuSeparator />
      <MenuItem icon={GithubLogoIcon} href={repositoryUrl} external>
        GitHub
      </MenuItem>
      <MenuItem icon={SignOutIcon} disabled={signOut.isPending} onSelect={() => signOut.mutate()}>
        {signOut.isPending ? t.nav.signingOut : t.nav.signOut}
      </MenuItem>
    </Menu>
  )
}
