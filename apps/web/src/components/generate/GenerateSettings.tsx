import { platforms } from '@/lib/platforms'

export interface GenerateConfig {
  target_platform: string
  count: number
  max_chars: number
  force_image: boolean
}

interface GenerateSettingsProps {
  config: GenerateConfig
  onChange: (config: GenerateConfig) => void
  isRepost?: boolean
}

export function GenerateSettings({ config, onChange, isRepost }: GenerateSettingsProps) {
  return (
    <div className="flex flex-col gap-4">
      {isRepost && (
        <div className="rounded-lg border border-cyan-500/20 bg-cyan-500/5 p-3 text-sm text-cyan-400">
          Generating short quote tweet commentary
        </div>
      )}
      <div>
        <label className="mb-1 block text-sm font-medium text-[var(--color-text)]">
          Target Platform
        </label>
        <select
          value={config.target_platform}
          onChange={(e) => onChange({ ...config, target_platform: e.target.value })}
          className="w-full rounded-[var(--radius-input)] border border-[var(--color-border)] bg-[var(--color-card)] px-3 py-2 text-sm text-[var(--color-text)]"
        >
          {platforms.map((p) => (
            <option key={p.id} value={p.id}>
              {p.name}
            </option>
          ))}
        </select>
      </div>

      <div>
        <label className="mb-1 block text-sm font-medium text-[var(--color-text)]">
          Variations ({config.count})
        </label>
        <input
          type="range"
          min={1}
          max={5}
          value={config.count}
          onChange={(e) => onChange({ ...config, count: Number(e.target.value) })}
          className="w-full accent-[var(--color-accent)]"
        />
        <div className="flex justify-between text-xs text-[var(--color-text-secondary)]">
          <span>1</span>
          <span>5</span>
        </div>
      </div>

      <div>
        <label className="mb-1 block text-sm font-medium text-[var(--color-text)]">
          Max Characters
        </label>
        <input
          type="number"
          value={config.max_chars}
          onChange={(e) => onChange({ ...config, max_chars: Number(e.target.value) })}
          className="w-full rounded-[var(--radius-input)] border border-[var(--color-border)] bg-[var(--color-card)] px-3 py-2 text-sm text-[var(--color-text)]"
        />
      </div>

      <div>
        <label className="flex items-center gap-2 text-sm text-[var(--color-text)]">
          <input
            type="checkbox"
            checked={config.force_image}
            onChange={(e) => onChange({ ...config, force_image: e.target.checked })}
            className="accent-[var(--color-accent)]"
            disabled={isRepost}
          />
          Always include image prompt
        </label>
        <p className="mt-1 text-xs text-[var(--color-text-secondary)]">
          When off, AI decides based on post context
        </p>
        {isRepost && (
          <p className="mt-1 text-xs text-[var(--color-text-secondary)]">
            Disabled in quote tweet mode — original post provides the visual
          </p>
        )}
      </div>
    </div>
  )
}
