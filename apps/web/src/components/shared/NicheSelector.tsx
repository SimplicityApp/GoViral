import { useState } from 'react'
import { X, Plus } from 'lucide-react'

export function NicheSelector({
  selected,
  allNiches,
  onChange,
  onAddNiche,
}: {
  selected: string[]
  allNiches: string[]
  onChange: (niches: string[]) => void
  onAddNiche: (niche: string) => void
}) {
  const [open, setOpen] = useState(false)
  const [inputVal, setInputVal] = useState('')

  const toggle = (niche: string) => {
    onChange(
      selected.includes(niche)
        ? selected.filter((n) => n !== niche)
        : [...selected, niche],
    )
  }

  const handleAdd = () => {
    const tag = inputVal.trim()
    if (tag) {
      onAddNiche(tag)
      setInputVal('')
    }
  }

  return (
    <div className="relative">
      {/* Selected chips */}
      <div className="mb-3 flex flex-wrap gap-2">
        {selected.map((tag) => (
          <span
            key={tag}
            className="flex items-center gap-1 rounded-full border border-[var(--color-border)] bg-[var(--color-card)] px-3 py-1 text-sm text-[var(--color-text)]"
          >
            {tag}
            <button
              onClick={() => toggle(tag)}
              className="text-[var(--color-text-secondary)] hover:text-[var(--color-text)]"
            >
              <X size={14} />
            </button>
          </span>
        ))}
      </div>

      {/* Dropdown trigger */}
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        className="flex items-center gap-2 rounded-[var(--radius-button)] border border-[var(--color-border)] px-3 py-2 text-sm text-[var(--color-text-secondary)] hover:text-[var(--color-text)] transition-colors"
      >
        <Plus size={16} /> Add niches
      </button>

      {/* Dropdown panel */}
      {open && (
        <div className="absolute z-10 mt-1 w-64 rounded-[var(--radius-card)] border border-[var(--color-border)] bg-[var(--color-card)] shadow-lg">
          <div className="max-h-48 overflow-y-auto p-2">
            {allNiches.length === 0 && (
              <p className="px-2 py-1.5 text-sm text-[var(--color-text-secondary)]">
                No niches yet — add one below
              </p>
            )}
            {allNiches.map((niche) => (
              <label
                key={niche}
                className="flex cursor-pointer items-center gap-2 rounded px-2 py-1.5 hover:bg-[var(--color-border)]"
              >
                <input
                  type="checkbox"
                  checked={selected.includes(niche)}
                  onChange={() => toggle(niche)}
                />
                <span className="text-sm text-[var(--color-text)]">{niche}</span>
              </label>
            ))}
          </div>
          {/* Add custom niche */}
          <div className="flex gap-2 border-t border-[var(--color-border)] p-2">
            <input
              type="text"
              placeholder="New niche..."
              value={inputVal}
              onChange={(e) => setInputVal(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && handleAdd()}
              className="flex-1 rounded-[var(--radius-input)] border border-[var(--color-border)] bg-[var(--color-card)] px-2 py-1 text-sm text-[var(--color-text)] placeholder:text-[var(--color-text-secondary)]"
            />
            <button
              type="button"
              onClick={handleAdd}
              className="rounded-[var(--radius-button)] border border-[var(--color-border)] px-2 py-1 text-[var(--color-text-secondary)] hover:text-[var(--color-text)]"
            >
              <Plus size={16} />
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
