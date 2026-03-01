import { useState, useRef, useEffect, useMemo } from 'react'
import type { AvailableRepo, Repo, RepoLink } from '@/lib/types'
import { GitBranch, Plus, Trash2, Code, Settings, Lock, Globe, Search, ChevronDown } from 'lucide-react'
import { LoadingSpinner } from '@/components/shared/LoadingSpinner'
import { RepoSettingsPanel } from './RepoSettingsPanel'
import { useAvailableReposQuery } from '@/hooks/useRepos'

interface RepoListProps {
  repos: Repo[] | undefined
  isLoading: boolean
  onAdd: (owner: string, name: string) => void
  isAdding: boolean
  onDelete: (id: number) => void
  onUpdateSettings?: (id: number, targetAudience: string, links: RepoLink[]) => void
  isSavingSettings?: boolean
}

export function RepoList({ repos, isLoading, onAdd, isAdding, onDelete, onUpdateSettings, isSavingSettings }: RepoListProps) {
  const [input, setInput] = useState('')
  const [isOpen, setIsOpen] = useState(false)
  const [expandedSettingsId, setExpandedSettingsId] = useState<number | null>(null)
  const dropdownRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  const { data: availableRepos, isLoading: isLoadingAvailable } = useAvailableReposQuery()

  // Close dropdown on outside click
  useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
        setIsOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  // Already-added repo full names for filtering
  const addedSet = useMemo(
    () => new Set((repos ?? []).map((r) => r.full_name)),
    [repos],
  )

  // Filter and group available repos
  const grouped = useMemo(() => {
    if (!availableRepos) return new Map<string, AvailableRepo[]>()
    const q = input.toLowerCase()
    const filtered = availableRepos.filter(
      (r) =>
        !addedSet.has(r.full_name) &&
        (q === '' ||
          r.full_name.toLowerCase().includes(q) ||
          (r.description && r.description.toLowerCase().includes(q))),
    )
    const map = new Map<string, AvailableRepo[]>()
    for (const r of filtered) {
      const existing = map.get(r.owner)
      if (existing) {
        existing.push(r)
      } else {
        map.set(r.owner, [r])
      }
    }
    return map
  }, [availableRepos, input, addedSet])

  const totalResults = useMemo(() => {
    let count = 0
    for (const repos of grouped.values()) count += repos.length
    return count
  }, [grouped])

  const handleSelect = (repo: AvailableRepo) => {
    onAdd(repo.owner, repo.name)
    setInput('')
    setIsOpen(false)
  }

  const handleManualAdd = () => {
    const trimmed = input.trim()
    if (!trimmed) return
    const parts = trimmed.split('/')
    if (parts.length !== 2 || !parts[0] || !parts[1]) return
    onAdd(parts[0], parts[1])
    setInput('')
    setIsOpen(false)
  }

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      handleManualAdd()
    } else if (e.key === 'Escape') {
      setIsOpen(false)
    }
  }

  // Check if the current input looks like a valid owner/repo that's not in the list
  const isCustomRepo = input.includes('/') && input.split('/').length === 2 && input.split('/').every(Boolean)
  const customNotInList = isCustomRepo && !availableRepos?.some((r) => r.full_name.toLowerCase() === input.toLowerCase())

  return (
    <div>
      {/* Searchable repo picker */}
      <div className="relative mb-4" ref={dropdownRef}>
        <div className="flex gap-2">
          <div className="relative flex-1">
            <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-[var(--color-text-secondary)]" />
            <input
              ref={inputRef}
              type="text"
              value={input}
              onChange={(e) => {
                setInput(e.target.value)
                setIsOpen(true)
              }}
              onFocus={() => setIsOpen(true)}
              onKeyDown={handleKeyDown}
              placeholder="Search repos or type owner/repo..."
              className="w-full rounded-[var(--radius-input)] border border-[var(--color-border)] bg-[var(--color-card)] py-2 pl-9 pr-8 text-sm text-[var(--color-text)] placeholder:text-[var(--color-text-secondary)] focus:outline-none focus:ring-1 focus:ring-[var(--color-accent)]"
            />
            <ChevronDown
              size={14}
              className={`absolute right-3 top-1/2 -translate-y-1/2 text-[var(--color-text-secondary)] transition-transform ${isOpen ? 'rotate-180' : ''}`}
            />
          </div>
          {customNotInList && (
            <button
              onClick={handleManualAdd}
              disabled={isAdding}
              className="flex items-center gap-1.5 rounded-[var(--radius-button)] bg-[var(--color-accent)] px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-[var(--color-accent-hover)] disabled:opacity-50"
            >
              <Plus size={14} />
              Add
            </button>
          )}
        </div>

        {/* Dropdown */}
        {isOpen && (
          <div className="absolute z-50 mt-1 max-h-72 w-full overflow-y-auto rounded-[var(--radius-card)] border border-[var(--color-border)] bg-[var(--color-card)] shadow-lg">
            {isLoadingAvailable && (
              <div className="flex items-center justify-center py-4">
                <LoadingSpinner />
              </div>
            )}

            {!isLoadingAvailable && totalResults === 0 && (
              <div className="px-3 py-3 text-sm text-[var(--color-text-secondary)]">
                {input ? 'No matching repos found. Type owner/repo to add manually.' : 'No repos available.'}
              </div>
            )}

            {!isLoadingAvailable &&
              Array.from(grouped.entries()).map(([owner, ownerRepos]) => (
                <div key={owner}>
                  <div className="sticky top-0 bg-[var(--color-bg)] px-3 py-1.5 text-xs font-semibold uppercase tracking-wider text-[var(--color-text-secondary)]">
                    {owner}
                  </div>
                  {ownerRepos.map((repo) => (
                    <button
                      key={repo.full_name}
                      onClick={() => handleSelect(repo)}
                      disabled={isAdding}
                      className="flex w-full items-center gap-2 px-3 py-2 text-left text-sm transition-colors hover:bg-[var(--color-border)]/30 disabled:opacity-50"
                    >
                      <Code size={14} className="shrink-0 text-[var(--color-accent)]" />
                      <div className="min-w-0 flex-1">
                        <div className="flex items-center gap-1.5">
                          <span className="truncate font-medium text-[var(--color-text)]">{repo.name}</span>
                          {repo.private ? (
                            <Lock size={11} className="shrink-0 text-amber-400" />
                          ) : (
                            <Globe size={11} className="shrink-0 text-[var(--color-text-secondary)]" />
                          )}
                          {repo.language && (
                            <span className="shrink-0 rounded-full border border-[var(--color-border)] px-1.5 py-0 text-[10px] text-[var(--color-text-secondary)]">
                              {repo.language}
                            </span>
                          )}
                        </div>
                        {repo.description && (
                          <p className="truncate text-xs text-[var(--color-text-secondary)]">{repo.description}</p>
                        )}
                      </div>
                    </button>
                  ))}
                </div>
              ))}
          </div>
        )}
      </div>

      {isLoading && (
        <div className="flex justify-center py-8">
          <LoadingSpinner />
        </div>
      )}

      {!isLoading && repos && repos.length === 0 && (
        <p className="text-sm text-[var(--color-text-secondary)]">
          No repositories connected. Search above to add one.
        </p>
      )}

      {repos && repos.length > 0 && (
        <div className="flex flex-col gap-2">
          {repos.map((repo) => (
            <div
              key={repo.id}
              className="overflow-hidden rounded-[var(--radius-card)] border border-[var(--color-border)] bg-[var(--color-card)]"
            >
              <div className="flex items-start justify-between gap-4 p-4">
                <div className="min-w-0 flex-1">
                  <div className="mb-1 flex items-center gap-2">
                    <Code size={14} className="shrink-0 text-[var(--color-accent)]" />
                    <span className="font-medium text-sm text-[var(--color-text)]">
                      {repo.full_name}
                    </span>
                    {repo.language && (
                      <span className="rounded-full border border-[var(--color-border)] px-2 py-0.5 text-xs text-[var(--color-text-secondary)]">
                        {repo.language}
                      </span>
                    )}
                  </div>
                  {repo.description && (
                    <p className="mb-1.5 text-xs text-[var(--color-text-secondary)] line-clamp-2">
                      {repo.description}
                    </p>
                  )}
                  <div className="flex items-center gap-1 text-xs text-[var(--color-text-secondary)]">
                    <GitBranch size={12} />
                    <span>{repo.default_branch}</span>
                    {repo.target_audience && (
                      <>
                        <span className="mx-1">·</span>
                        <span>Audience: {repo.target_audience}</span>
                      </>
                    )}
                  </div>
                </div>
                <div className="flex shrink-0 items-center gap-2">
                  {onUpdateSettings && (
                    <button
                      onClick={() =>
                        setExpandedSettingsId((prev) => (prev === repo.id ? null : repo.id))
                      }
                      className={`text-[var(--color-text-secondary)] transition-colors hover:text-[var(--color-accent)] ${expandedSettingsId === repo.id ? 'text-[var(--color-accent)]' : ''}`}
                      aria-label={`Settings for ${repo.full_name}`}
                    >
                      <Settings size={16} />
                    </button>
                  )}
                  <button
                    onClick={() => onDelete(repo.id)}
                    className="text-[var(--color-text-secondary)] transition-colors hover:text-red-400"
                    aria-label={`Remove ${repo.full_name}`}
                  >
                    <Trash2 size={16} />
                  </button>
                </div>
              </div>
              {expandedSettingsId === repo.id && onUpdateSettings && (
                <RepoSettingsPanel
                  repo={repo}
                  onSave={(audience, links) => onUpdateSettings(repo.id, audience, links)}
                  isSaving={isSavingSettings ?? false}
                />
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
