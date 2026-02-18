import { Menu } from 'lucide-react'
import { useLocation } from 'react-router-dom'
import { useUIStore } from '@/stores/ui-store'

const pageTitles: Record<string, string> = {
  '/': 'Dashboard',
  '/posts': 'My Posts',
  '/trending': 'Trending',
  '/generate': 'Generate',
  '/history': 'History',
  '/publish': 'Publish',
  '/settings': 'Settings',
}

export function TopBar({ actions }: { actions?: React.ReactNode }) {
  const { setSidebarOpen } = useUIStore()
  const location = useLocation()
  const title = pageTitles[location.pathname] ?? 'GoViral'

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
