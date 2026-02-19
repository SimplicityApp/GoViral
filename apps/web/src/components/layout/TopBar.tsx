import { Menu } from 'lucide-react'
import { useLocation } from 'react-router-dom'
import { useUIStore } from '@/stores/ui-store'

const pageTitles: Record<string, string> = {
  dashboard: 'Dashboard',
  posts: 'My Posts',
  trending: 'Trending',
  generate: 'Generate',
  history: 'History',
  publish: 'Publish',
  settings: 'Settings',
}

export function TopBar({ actions }: { actions?: React.ReactNode }) {
  const { setSidebarOpen } = useUIStore()
  const location = useLocation()

  // Extract the page segment from /:platform/page or /settings
  const segments = location.pathname.split('/').filter(Boolean)
  let pageSegment: string
  if (segments[0] === 'settings') {
    pageSegment = 'settings'
  } else {
    // /:platform/page — page is segments[1]
    pageSegment = segments[1] ?? 'dashboard'
  }
  const title = pageTitles[pageSegment] ?? 'GoViral'

  return (
    <header className="flex h-14 items-center gap-3 border-b border-[var(--color-border)] bg-[var(--color-bg)] px-4">
      <button
        onClick={() => setSidebarOpen(true)}
        className="text-[var(--color-text-secondary)] hover:text-[var(--color-text)] lg:hidden"
      >
        <Menu size={20} />
      </button>
      <h1 className="text-lg font-semibold text-[var(--color-text)]">{title}</h1>
      {actions && <div className="ml-auto">{actions}</div>}
    </header>
  )
}
