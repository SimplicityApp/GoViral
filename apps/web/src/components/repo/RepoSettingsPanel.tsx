import { useState } from 'react'
import type { Repo, RepoLink } from '@/lib/types'
import { Plus, Trash2, Save, Loader2 } from 'lucide-react'

interface RepoSettingsPanelProps {
  repo: Repo
  onSave: (targetAudience: string, links: RepoLink[]) => void
  isSaving: boolean
}

export function RepoSettingsPanel({ repo, onSave, isSaving }: RepoSettingsPanelProps) {
  const [targetAudience, setTargetAudience] = useState(repo.target_audience ?? '')
  const [links, setLinks] = useState<RepoLink[]>(
    repo.links?.length ? repo.links : [],
  )

  const handleAddLink = () => {
    setLinks((prev) => [...prev, { label: '', url: '' }])
  }

  const handleRemoveLink = (index: number) => {
    setLinks((prev) => prev.filter((_, i) => i !== index))
  }

  const handleLinkChange = (index: number, field: keyof RepoLink, value: string) => {
    setLinks((prev) =>
      prev.map((link, i) => (i === index ? { ...link, [field]: value } : link)),
    )
  }

  const handleSave = () => {
    const cleanedLinks = links.filter((l) => l.label.trim() || l.url.trim())
    onSave(targetAudience.trim(), cleanedLinks)
  }

  return (
    <div className="border-t border-[var(--color-border)] bg-[var(--color-bg)] p-4">
      <div className="mb-3">
        <label className="mb-1 block text-xs font-medium text-[var(--color-text-secondary)]">
          Target Audience
        </label>
        <input
          type="text"
          value={targetAudience}
          onChange={(e) => setTargetAudience(e.target.value)}
          placeholder="e.g. python developers, backend engineers"
          className="w-full rounded-[var(--radius-input)] border border-[var(--color-border)] bg-[var(--color-card)] px-3 py-1.5 text-sm text-[var(--color-text)] placeholder:text-[var(--color-text-secondary)] focus:outline-none focus:ring-1 focus:ring-[var(--color-accent)]"
        />
      </div>

      <div className="mb-3">
        <label className="mb-1 block text-xs font-medium text-[var(--color-text-secondary)]">
          Links (included in every generated post)
        </label>
        {links.map((link, i) => (
          <div key={i} className="mb-1.5 flex items-center gap-2">
            <input
              type="text"
              value={link.label}
              onChange={(e) => handleLinkChange(i, 'label', e.target.value)}
              placeholder="Label (e.g. GitHub)"
              className="w-28 rounded-[var(--radius-input)] border border-[var(--color-border)] bg-[var(--color-card)] px-2 py-1.5 text-sm text-[var(--color-text)] placeholder:text-[var(--color-text-secondary)] focus:outline-none focus:ring-1 focus:ring-[var(--color-accent)]"
            />
            <input
              type="text"
              value={link.url}
              onChange={(e) => handleLinkChange(i, 'url', e.target.value)}
              placeholder="https://..."
              className="flex-1 rounded-[var(--radius-input)] border border-[var(--color-border)] bg-[var(--color-card)] px-2 py-1.5 text-sm text-[var(--color-text)] placeholder:text-[var(--color-text-secondary)] focus:outline-none focus:ring-1 focus:ring-[var(--color-accent)]"
            />
            <button
              onClick={() => handleRemoveLink(i)}
              className="shrink-0 text-[var(--color-text-secondary)] transition-colors hover:text-red-400"
              aria-label="Remove link"
            >
              <Trash2 size={14} />
            </button>
          </div>
        ))}
        <button
          onClick={handleAddLink}
          className="mt-1 flex items-center gap-1 text-xs text-[var(--color-accent)] transition-colors hover:text-[var(--color-accent-hover)]"
        >
          <Plus size={12} />
          Add Link
        </button>
      </div>

      <button
        onClick={handleSave}
        disabled={isSaving}
        className="flex items-center gap-1.5 rounded-[var(--radius-button)] bg-[var(--color-accent)] px-4 py-1.5 text-sm font-medium text-white transition-colors hover:bg-[var(--color-accent-hover)] disabled:opacity-50"
      >
        {isSaving ? <Loader2 size={14} className="animate-spin" /> : <Save size={14} />}
        Save Settings
      </button>
    </div>
  )
}
