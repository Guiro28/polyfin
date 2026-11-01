import { MoonIcon, SunIcon } from '@phosphor-icons/react'
import { useI18n } from '@/i18n'
import { setTheme, useTheme } from '@/theme'
import { IconButton } from '@/ui'

/** The button of the top bar that turns the page light or dark, whatever the system's look. */
export function ThemeToggle() {
  const { t } = useI18n()
  const { dark } = useTheme()
  return (
    <IconButton
      label={dark ? t.nav.toLight : t.nav.toDark}
      icon={dark ? SunIcon : MoonIcon}
      onClick={() => setTheme(dark ? 'light' : 'dark')}
    />
  )
}
