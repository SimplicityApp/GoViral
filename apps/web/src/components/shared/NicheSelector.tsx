import { useState } from 'react'
import { X, Plus } from 'lucide-react'

const SUGGESTED_NICHES = [
  // Tech & Dev
  'AI', 'Machine Learning', 'Web Development', 'Cloud Computing',
  'Cybersecurity', 'DevOps', 'Open Source', 'Blockchain',
  'Data Science', 'Mobile Development',
  // Business
  'Startups', 'SaaS', 'Product Management', 'Venture Capital',
  'Marketing', 'Growth Hacking', 'Remote Work', 'Entrepreneurship',
  'Leadership', 'Personal Branding',
  // Creator Economy
  'Content Creation', 'Social Media Marketing', 'SEO', 'Copywriting',
  'Newsletter Growth', 'YouTube', 'Podcasting',
  // Design
  'UI/UX Design', 'Graphic Design', 'Design Systems',
  // Science & Health
  'Climate Tech', 'Biotech', 'Mental Health', 'Fitness',
  // Finance
  'Fintech', 'Crypto', 'Personal Finance', 'Investing',
  // Career
  'Career Development', 'Freelancing', 'Tech Interviews', 'Developer Relations',
  // Lifestyle
  'Productivity', 'Book Reviews', 'Travel', 'Photography',
]

export function NicheSelector({
  selected,
  onChange,
}: {
  selected: string[]
  onChange: (niches: string[]) => void
}) {
  const [inputVal, setInputVal] = useState('')

  const remove = (niche: string) => {
    onChange(selected.filter((n) => n !== niche))
  }

  const add = (niche: string) => {
    if (!selected.includes(niche)) {
      onChange([...selected, niche])
    }
  }

  const handleAdd = () => {
    const tag = inputVal.trim()
    if (tag) {
      add(tag)
      setInputVal('')
    }
  }

  const selectedSet = new Set(selected)
  const unselectedSuggestions = SUGGESTED_NICHES.filter((n) => !selectedSet.has(n))

  return (
    <div>
      {/* Selected chips */}
      {selected.length > 0 && (
        <div className="mb-3 flex flex-wrap gap-2">
          {selected.map((tag) => (
            <span
              key={tag}
              className="flex items-center gap-1 rounded-full border border-[var(--color-accent)] bg-[var(--color-accent)]/10 px-3 py-1 text-sm text-[var(--color-accent)]"
            >
              {tag}
              <button
                onClick={() => remove(tag)}
                className="text-[var(--color-accent)]/60 hover:text-[var(--color-accent)]"
              >
                <X size={14} />
              </button>
            </span>
          ))}
        </div>
      )}

      {/* Suggested niches grid */}
      {unselectedSuggestions.length > 0 && (
        <div className="mb-3">
          <p className="mb-2 text-xs font-medium text-[var(--color-text-secondary)]">
            Suggestions
          </p>
          <div className="flex max-h-48 flex-wrap gap-1.5 overflow-y-auto">
            {unselectedSuggestions.map((niche) => (
              <button
                key={niche}
                type="button"
                onClick={() => add(niche)}
                className="rounded-full border border-[var(--color-border)] px-3 py-1 text-sm text-[var(--color-text-secondary)] transition-colors hover:border-[var(--color-accent)] hover:text-[var(--color-accent)]"
              >
                {niche}
              </button>
            ))}
          </div>
        </div>
      )}

      {/* Custom input */}
      <div className="flex gap-2">
        <input
          type="text"
          placeholder="Add custom niche..."
          value={inputVal}
          onChange={(e) => setInputVal(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && handleAdd()}
          className="flex-1 rounded-[var(--radius-input)] border border-[var(--color-border)] bg-[var(--color-card)] px-3 py-1.5 text-sm text-[var(--color-text)] placeholder:text-[var(--color-text-secondary)]"
        />
        <button
          type="button"
          onClick={handleAdd}
          className="rounded-[var(--radius-button)] border border-[var(--color-border)] px-2.5 py-1.5 text-[var(--color-text-secondary)] hover:text-[var(--color-text)] transition-colors"
        >
          <Plus size={16} />
        </button>
      </div>
    </div>
  )
}
