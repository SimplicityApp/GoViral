import { Twitter, Linkedin, Github } from 'lucide-react'

const PLATFORMS = [
  { id: 'x', label: 'X (Twitter)', icon: Twitter, desc: 'Post tweets and threads' },
  { id: 'linkedin', label: 'LinkedIn', icon: Linkedin, desc: 'Publish professional content' },
  { id: 'github', label: 'GitHub', icon: Github, desc: 'Turn commits into posts' },
]

export function PlatformsStep({
  selectedPlatforms,
  onToggle,
}: {
  selectedPlatforms: Set<string>
  onToggle: (id: string) => void
}) {
  return (
    <div className="flex flex-col items-center">
      <h2 className="mb-2 text-xl font-bold text-[var(--color-text)]">
        Choose Your Platforms
      </h2>
      <p className="mb-6 text-sm text-[var(--color-text-secondary)]">
        Select the platforms you want to use. You can change this later in Settings.
      </p>
      <div className="grid w-full max-w-md gap-3">
        {PLATFORMS.map(({ id, label, icon: Icon, desc }) => {
          const active = selectedPlatforms.has(id)
          return (
            <button
              key={id}
              type="button"
              onClick={() => onToggle(id)}
              className={`flex items-center gap-4 rounded-[var(--radius-card)] border p-4 text-left transition-colors ${
                active
                  ? 'border-[var(--color-accent)] bg-[var(--color-accent)]/10'
                  : 'border-[var(--color-border)] bg-[var(--color-card)] hover:border-[var(--color-text-secondary)]'
              }`}
            >
              <div
                className={`flex h-10 w-10 items-center justify-center rounded-lg ${
                  active ? 'bg-[var(--color-accent)]/20 text-[var(--color-accent)]' : 'bg-[var(--color-border)] text-[var(--color-text-secondary)]'
                }`}
              >
                <Icon size={20} />
              </div>
              <div className="flex-1">
                <div className="text-sm font-medium text-[var(--color-text)]">{label}</div>
                <div className="text-xs text-[var(--color-text-secondary)]">{desc}</div>
              </div>
              <div
                className={`flex h-5 w-5 items-center justify-center rounded border ${
                  active
                    ? 'border-[var(--color-accent)] bg-[var(--color-accent)]'
                    : 'border-[var(--color-border)]'
                }`}
              >
                {active && (
                  <svg width="12" height="12" viewBox="0 0 12 12" fill="none">
                    <path d="M2.5 6L5 8.5L9.5 3.5" stroke="white" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
                  </svg>
                )}
              </div>
            </button>
          )
        })}
      </div>
    </div>
  )
}
