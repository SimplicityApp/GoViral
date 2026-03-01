import type { RepoCommit } from '@/lib/types'
import { GitCommit } from 'lucide-react'

interface CommitCardProps {
  commit: RepoCommit
  selected: boolean
  onToggle: () => void
}

export function CommitCard({ commit, selected, onToggle }: CommitCardProps) {
  const shortSha = commit.sha.slice(0, 7)
  const firstLine = commit.message.split('\n')[0] ?? commit.message

  return (
    <div
      onClick={onToggle}
      className={`cursor-pointer rounded-[var(--radius-card)] border bg-[var(--color-card)] p-4 transition-colors hover:bg-[var(--color-card-hover)] ${
        selected ? 'border-[var(--color-accent)]' : 'border-[var(--color-border)]'
      }`}
    >
      <div className="flex items-start gap-3">
        <input
          type="checkbox"
          checked={selected}
          onChange={onToggle}
          onClick={(e) => e.stopPropagation()}
          className="mt-0.5 h-4 w-4 shrink-0 accent-[var(--color-accent)]"
        />

        <div className="min-w-0 flex-1">
          <div className="mb-1.5 flex flex-wrap items-center gap-2">
            <span className="flex items-center gap-1 rounded-md bg-[var(--color-border)] px-2 py-0.5 font-mono text-xs text-[var(--color-text-secondary)]">
              <GitCommit size={12} />
              {shortSha}
            </span>
            <span className="text-sm font-medium text-[var(--color-text)] truncate">
              {firstLine}
            </span>
          </div>

          <div className="mb-2 flex flex-wrap items-center gap-3 text-xs text-[var(--color-text-secondary)]">
            <span>{commit.author_name}</span>
            <span>
              <span className="text-green-400">+{commit.additions}</span>
              {' / '}
              <span className="text-red-400">-{commit.deletions}</span>
            </span>
            <span>{commit.files_changed} file{commit.files_changed !== 1 ? 's' : ''}</span>
          </div>

          {commit.diff_summary && (
            <p className="text-xs text-[var(--color-text-secondary)] line-clamp-2">
              {commit.diff_summary}
            </p>
          )}
        </div>
      </div>
    </div>
  )
}
