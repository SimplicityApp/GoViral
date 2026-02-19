import { useNavigate, useLocation } from 'react-router-dom'
import { Twitter, Linkedin } from 'lucide-react'
import { platforms } from '@/lib/platforms'
import { usePlatformStore } from '@/stores/platform-store'

const iconMap: Record<string, typeof Twitter> = {
  Twitter,
  Linkedin,
}

export function PlatformSwitcher() {
  const activePlatform = usePlatformStore((s) => s.activePlatform)
  const navigate = useNavigate()
  const location = useLocation()

  const handleSwitch = (platformId: string) => {
    // Parse the current path: /:platform/page or /settings
    const segments = location.pathname.split('/').filter(Boolean)
    if (segments.length >= 2) {
      // Replace the platform segment, keep the rest
      const rest = segments.slice(1).join('/')
      navigate(`/${platformId}/${rest}`)
    } else {
      // On /settings or root, navigate to the platform's dashboard
      navigate(`/${platformId}/dashboard`)
    }
  }

  return (
    <div className="flex border-b border-[var(--color-border)]">
      {platforms.map((platform) => {
        const Icon = iconMap[platform.icon]
        const isActive = activePlatform === platform.id
        return (
          <button
            key={platform.id}
            onClick={() => handleSwitch(platform.id)}
            className={`flex flex-1 items-center justify-center gap-2 px-3 py-3 text-sm font-medium transition-colors ${
              isActive
                ? 'border-b-2 border-[var(--color-accent)] text-[var(--color-accent)]'
                : 'text-[var(--color-text-secondary)] hover:text-[var(--color-text)]'
            }`}
          >
            {Icon && <Icon size={16} />}
            {platform.name}
          </button>
        )
      })}
    </div>
  )
}
