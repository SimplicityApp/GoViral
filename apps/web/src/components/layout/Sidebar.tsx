import { NavLink } from 'react-router-dom'
import {
  LayoutDashboard,
  FileText,
  TrendingUp,
  Sparkles,
  Clock,
  Send,
  Settings,
  X,
} from 'lucide-react'
import { PlatformSwitcher } from './PlatformSwitcher'
import { useUIStore } from '@/stores/ui-store'

const navItems = [
  { to: '/', label: 'Dashboard', icon: LayoutDashboard },
  { to: '/posts', label: 'My Posts', icon: FileText },
  { to: '/trending', label: 'Trending', icon: TrendingUp },
  { to: '/generate', label: 'Generate', icon: Sparkles },
  { to: '/history', label: 'History', icon: Clock },
  { to: '/publish', label: 'Publish', icon: Send },
  { to: '/settings', label: 'Settings', icon: Settings },
]

export function Sidebar() {
  const { sidebarOpen, setSidebarOpen } = useUIStore()

  return (
    <>
      {/* Mobile overlay */}
      {sidebarOpen && (
        <div
          className="fixed inset-0 z-40 bg-black/50 lg:hidden"
          onClick={() => setSidebarOpen(false)}
        />
      )}

      <aside
        className={`fixed top-0 left-0 z-50 flex h-full w-64 flex-col border-r border-[var(--color-border)] bg-[var(--color-bg)] transition-transform lg:static lg:translate-x-0 ${
          sidebarOpen ? 'translate-x-0' : '-translate-x-full'
        }`}
      >
        <div className="flex items-center justify-between px-4 py-4">
          <span className="text-lg font-bold text-[var(--color-text)]">GoViral</span>
          <button
            onClick={() => setSidebarOpen(false)}
            className="text-[var(--color-text-secondary)] hover:text-[var(--color-text)] lg:hidden"
          >
            <X size={20} />
          </button>
        </div>

        <PlatformSwitcher />

        <nav className="flex-1 overflow-y-auto px-2 py-4">
          {navItems.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.to === '/'}
              onClick={() => setSidebarOpen(false)}
              className={({ isActive }) =>
                `flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-colors ${
                  isActive
                    ? 'bg-[var(--color-accent)]/10 text-[var(--color-accent)]'
                    : 'text-[var(--color-text-secondary)] hover:bg-[var(--color-card)] hover:text-[var(--color-text)]'
                }`
              }
            >
              <item.icon size={18} />
              {item.label}
            </NavLink>
          ))}
        </nav>
      </aside>
    </>
  )
}
