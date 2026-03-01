import { useCodeImagePreviewsQuery } from '@/hooks/useRepos'
import type { CodeImageTheme } from '@/lib/types'

const TEMPLATE_LABELS: Record<string, { label: string; description: string }> = {
  github: { label: 'GitHub', description: 'Classic GitHub diff view' },
  macos: { label: 'macOS Window', description: 'Native macOS window chrome' },
  vscode: { label: 'VS Code', description: 'Editor-style diff panel' },
  minimal: { label: 'Minimal', description: 'Clean gradient background' },
  terminal: { label: 'Terminal', description: 'Terminal emulator look' },
  card: { label: 'Social Card', description: 'Card with description overlay' },
}

const THEME_LABELS: Record<string, string> = {
  'github-dark': 'GitHub Dark',
  'one-dark-pro': 'One Dark Pro',
  dracula: 'Dracula',
  nord: 'Nord',
  'solarized-light': 'Solarized Light',
  monokai: 'Monokai',
}

interface TemplatePreviewGridProps {
  selectedTemplate: string
  selectedTheme: string
  onSelectTemplate: (name: string) => void
  onSelectTheme: (name: string) => void
  themes: CodeImageTheme[]
}

export function TemplatePreviewGrid({
  selectedTemplate,
  selectedTheme,
  onSelectTemplate,
  onSelectTheme,
  themes,
}: TemplatePreviewGridProps) {
  const { data, isLoading } = useCodeImagePreviewsQuery(selectedTheme)

  return (
    <div className="flex flex-col gap-4">
      {/* Theme selector pills */}
      <div>
        <label className="mb-2 block text-sm font-medium text-[var(--color-text)]">
          Color Theme
        </label>
        <div className="flex flex-wrap gap-2">
          {themes.map((t) => (
            <button
              key={t.name}
              onClick={() => onSelectTheme(t.name)}
              className={`rounded-full px-3 py-1 text-xs font-medium transition-all ${
                selectedTheme === t.name
                  ? 'bg-[var(--color-accent)] text-white ring-2 ring-[var(--color-accent)] ring-offset-2 ring-offset-[var(--color-bg)]'
                  : 'bg-[var(--color-card)] text-[var(--color-text-secondary)] hover:text-[var(--color-text)]'
              }`}
            >
              {THEME_LABELS[t.name] ?? t.name}
            </button>
          ))}
        </div>
      </div>

      {/* Template grid */}
      <div>
        <label className="mb-2 block text-sm font-medium text-[var(--color-text)]">
          Image Template
        </label>
        <div className="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-3">
          {isLoading || !data
            ? Array.from({ length: 6 }).map((_, i) => (
                <div
                  key={i}
                  className="animate-pulse rounded-lg border border-[var(--color-border)] bg-[var(--color-card)]"
                >
                  <div className="h-[136px] rounded-t-lg bg-[var(--color-border)]" />
                  <div className="p-2">
                    <div className="mb-1 h-4 w-20 rounded bg-[var(--color-border)]" />
                    <div className="h-3 w-32 rounded bg-[var(--color-border)]" />
                  </div>
                </div>
              ))
            : Object.entries(data.previews).map(([name, html]) => {
                const info = TEMPLATE_LABELS[name]
                return (
                  <button
                    key={name}
                    onClick={() => onSelectTemplate(name)}
                    className={`cursor-pointer rounded-lg border text-left transition-all ${
                      selectedTemplate === name
                        ? 'border-[var(--color-accent)] ring-2 ring-[var(--color-accent)]'
                        : 'border-[var(--color-border)] hover:border-[var(--color-text-secondary)]'
                    }`}
                  >
                    <div className="overflow-hidden rounded-t-lg" style={{ width: '100%', height: 136 }}>
                      <iframe
                        srcDoc={html}
                        sandbox=""
                        width={680}
                        height={420}
                        style={{
                          transform: 'scale(0.324)',
                          transformOrigin: 'top left',
                          pointerEvents: 'none',
                          border: 'none',
                        }}
                        title={`${name} template preview`}
                      />
                    </div>
                    <div className="p-2">
                      <div className="text-sm font-medium text-[var(--color-text)]">
                        {info?.label ?? name}
                      </div>
                      <div className="text-xs text-[var(--color-text-secondary)]">
                        {info?.description ?? ''}
                      </div>
                    </div>
                  </button>
                )
              })}
        </div>
      </div>
    </div>
  )
}
