import { Outlet } from 'react-router'

/**
 * The layout route of a section's pages (Content: Sources, Libraries, Live TV, Versions;
 * System: Health, Statistics, Schedule, Logs, API keys). The sidebar lists the pages under their
 * section, as Stremfin's does, so the page shows no tabs of them.
 */
export function SectionTabs(_props: { section: 'content' | 'system' }) {
  return <Outlet />
}
