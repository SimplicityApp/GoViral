import { Twitter, Linkedin } from 'lucide-react'
import { platforms } from '@/lib/platforms'
import { usePlatformStore } from '@/stores/platform-store'

const iconMap: Record<string, typeof Twitter> = {
  Twitter,
  Linkedin,
}

export function PlatformSwitcher() {
  const { activePlatform, setActivePlatform } = usePlatformStore()

  return (
    <div className="flex border-b border-[var(--color-border)]">
      {platforms.map((platform) => {
        const Icon = iconMap[platform.icon]
        const isActive = activePlatform === platform.id
        return (
          <button
            key={platform.id}
            onClick={() => setActivePlatform(platform.id)}
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
