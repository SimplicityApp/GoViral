import { platforms } from '@/lib/platforms'
import { useCodeImageOptionsQuery } from '@/hooks/useRepos'
import { TemplatePreviewGrid } from './TemplatePreviewGrid'

export interface GenerateConfig {
  target_platform: string
  count: number
  max_chars: number
  force_image: boolean
  include_code_images?: boolean
  style_direction?: string
  code_image_template?: string
  code_image_theme?: string
}

interface GenerateSettingsProps {
  config: GenerateConfig
  onChange: (config: GenerateConfig) => void
  isRepost?: boolean
  sourceType?: 'trending' | 'code'
}

export function GenerateSettings({ config, onChange, isRepost, sourceType }: GenerateSettingsProps) {
  const { data: imageOptions } = useCodeImageOptionsQuery()

  return (
    <div className="flex flex-col gap-4">
      {isRepost && (
        <div className="rounded-lg border border-cyan-500/20 bg-cyan-500/5 p-3 text-sm text-cyan-400">
          {config.target_platform === 'linkedin'
            ? 'Generating short repost commentary'
            : 'Generating short quote tweet commentary'}
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

      {sourceType !== 'code' && (
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
          <p className="mt-1 text-xs text-[var(--color-text-secondary)]">
            {config.target_platform === 'linkedin'
              ? 'LinkedIn default: 2,000 characters'
              : 'X default: 280 characters'}
          </p>
        </div>
      )}

      {sourceType !== 'code' && (
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
              {config.target_platform === 'linkedin'
                ? 'Disabled in repost mode — original post provides the visual'
                : 'Disabled in quote tweet mode — original post provides the visual'}
            </p>
          )}
        </div>
      )}

      {sourceType === 'code' && (
        <>
          <div>
            <label className="mb-1 block text-sm font-medium text-[var(--color-text)]">
              Style Direction
            </label>
            <input
              type="text"
              value={config.style_direction ?? ''}
              onChange={(e) => onChange({ ...config, style_direction: e.target.value })}
              placeholder="e.g. casual and concise, developer-focused, storytelling..."
              className="w-full rounded-[var(--radius-input)] border border-[var(--color-border)] bg-[var(--color-card)] px-3 py-2 text-sm text-[var(--color-text)] placeholder:text-[var(--color-text-secondary)]"
            />
          </div>
          <div>
            <label className="flex items-center gap-2 text-sm text-[var(--color-text)]">
              <input
                type="checkbox"
                checked={config.include_code_images ?? false}
                onChange={(e) => onChange({ ...config, include_code_images: e.target.checked })}
                className="accent-[var(--color-accent)]"
              />
              Include code images
            </label>
            <p className="mt-1 text-xs text-[var(--color-text-secondary)]">
              Attach syntax-highlighted code snippets from the commit diff
            </p>
          </div>

          {config.include_code_images && imageOptions && (
            <TemplatePreviewGrid
              selectedTemplate={config.code_image_template ?? 'github'}
              selectedTheme={config.code_image_theme ?? 'github-dark'}
              onSelectTemplate={(name) => onChange({ ...config, code_image_template: name })}
              onSelectTheme={(name) => onChange({ ...config, code_image_theme: name })}
              themes={imageOptions.themes}
            />
          )}
        </>
      )}
    </div>
  )
}
