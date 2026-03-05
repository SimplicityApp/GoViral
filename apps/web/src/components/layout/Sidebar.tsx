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
  Bot,
  Code,
  Video,
  Compass,
} from 'lucide-react'
import { PlatformSwitcher } from './PlatformSwitcher'
import { useUIStore } from '@/stores/ui-store'
import { usePlatformStore } from '@/stores/platform-store'

const platformNavItems: { path: string; label: string; icon: typeof FileText; platforms?: string[] }[] = [
  { path: 'dashboard', label: 'Dashboard', icon: LayoutDashboard },
  { path: 'posts', label: 'My Posts', icon: FileText, platforms: ['x', 'linkedin'] },
  { path: 'trending', label: 'Trending', icon: TrendingUp, platforms: ['x', 'linkedin'] },
  { path: 'generate', label: 'Generate', icon: Sparkles, platforms: ['x', 'linkedin'] },
  { path: 'publish', label: 'Publish', icon: Send, platforms: ['x', 'linkedin'] },
  { path: 'history', label: 'History', icon: Clock, platforms: ['x', 'linkedin'] },
  { path: 'code-to-post', label: 'Code to Post', icon: Code, platforms: ['x', 'linkedin'] },
  { path: 'autopilot', label: 'Autopilot', icon: Bot, platforms: ['x', 'linkedin'] },
  { path: 'video', label: 'Video', icon: Video, platforms: ['youtube', 'tiktok'] },
]

export function Sidebar() {
  const { sidebarOpen, setSidebarOpen } = useUIStore()
  const activePlatform = usePlatformStore((s) => s.activePlatform)

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
          <NavLink to="/" className="text-lg font-bold text-[var(--color-text)] hover:opacity-80 transition-opacity">GoViral</NavLink>
          <button
            onClick={() => setSidebarOpen(false)}
            className="text-[var(--color-text-secondary)] hover:text-[var(--color-text)] lg:hidden"
          >
            <X size={20} />
          </button>
        </div>

        <PlatformSwitcher />

        <nav className="flex-1 overflow-y-auto px-2 py-4">
          {platformNavItems
            .filter((item) => !item.platforms || item.platforms.includes(activePlatform))
            .map((item) => (
            <NavLink
              key={item.path}
              to={`/${activePlatform}/${item.path}`}
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
          <NavLink
            to="/settings"
            onClick={() => setSidebarOpen(false)}
            className={({ isActive }) =>
              `flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-colors ${
                isActive
                  ? 'bg-[var(--color-accent)]/10 text-[var(--color-accent)]'
                  : 'text-[var(--color-text-secondary)] hover:bg-[var(--color-card)] hover:text-[var(--color-text)]'
              }`
            }
          >
            <Settings size={18} />
            Settings
          </NavLink>
          <NavLink
            to="/onboarding"
            onClick={() => setSidebarOpen(false)}
            className={({ isActive }) =>
              `flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-colors ${
                isActive
                  ? 'bg-[var(--color-accent)]/10 text-[var(--color-accent)]'
                  : 'text-[var(--color-text-secondary)] hover:bg-[var(--color-card)] hover:text-[var(--color-text)]'
              }`
            }
          >
            <Compass size={18} />
            Setup Guide
          </NavLink>
        </nav>
      </aside>
    </>
  )
}
