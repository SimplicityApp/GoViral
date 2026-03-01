import type { Repo, RepoCommit, ProgressEvent } from '@/lib/types'
import { CommitCard } from './CommitCard'
import { LoadingSpinner } from '@/components/shared/LoadingSpinner'
import { Pagination } from '@/components/shared/Pagination'
import { usePagination } from '@/hooks/usePagination'
import { RefreshCw } from 'lucide-react'

interface CommitListProps {
  repos: Repo[]
  selectedRepoId: number | null
  onSelectRepo: (id: number) => void
  commits: RepoCommit[] | undefined
  isLoadingCommits: boolean
  selectedCommitIds: Set<number>
  onToggleCommit: (id: number) => void
  onFetchCommits: () => void
  isFetching: boolean
  fetchProgress: ProgressEvent | null
}

export function CommitList({
  repos,
  selectedRepoId,
  onSelectRepo,
  commits,
  isLoadingCommits,
  selectedCommitIds,
  onToggleCommit,
  onFetchCommits,
  isFetching,
  fetchProgress,
}: CommitListProps) {
  const { page, totalPages, pageItems, nextPage, prevPage, hasNext, hasPrev } = usePagination(
    commits ?? [],
  )

  return (
    <div>
      <div className="mb-4 flex flex-wrap items-center gap-3">
        <select
          value={selectedRepoId ?? ''}
          onChange={(e) => onSelectRepo(Number(e.target.value))}
          className="rounded-[var(--radius-input)] border border-[var(--color-border)] bg-[var(--color-card)] px-3 py-2 text-sm text-[var(--color-text)] focus:outline-none focus:ring-1 focus:ring-[var(--color-accent)]"
        >
          <option value="" disabled>
            Select a repository
          </option>
          {repos.map((repo) => (
            <option key={repo.id} value={repo.id}>
              {repo.full_name}
            </option>
          ))}
        </select>

        {selectedRepoId && (
          <button
            onClick={onFetchCommits}
            disabled={isFetching}
            className="flex items-center gap-1.5 rounded-[var(--radius-button)] border border-[var(--color-border)] px-3 py-2 text-sm font-medium text-[var(--color-text)] transition-colors hover:bg-[var(--color-card)] disabled:opacity-50"
          >
            <RefreshCw size={14} className={isFetching ? 'animate-spin' : ''} />
            {isFetching ? 'Fetching...' : 'Fetch Commits'}
          </button>
        )}
      </div>

      {fetchProgress && isFetching && (
        <div className="mb-4 rounded-[var(--radius-card)] border border-[var(--color-border)] bg-[var(--color-card)] p-3">
          <div className="mb-1 text-sm text-[var(--color-text)]">{fetchProgress.message}</div>
          <div className="h-1.5 w-full rounded-full bg-[var(--color-border)]">
            <div
              className="h-1.5 rounded-full bg-[var(--color-accent)] transition-all"
              style={{ width: `${fetchProgress.percentage}%` }}
            />
          </div>
        </div>
      )}

      {!selectedRepoId && (
        <p className="text-sm text-[var(--color-text-secondary)]">
          Select a repository to view its commits.
        </p>
      )}

      {selectedRepoId && isLoadingCommits && (
        <div className="py-8 flex justify-center">
          <LoadingSpinner />
        </div>
      )}

      {selectedRepoId && !isLoadingCommits && commits && commits.length === 0 && (
        <p className="text-sm text-[var(--color-text-secondary)]">
          No commits found. Click "Fetch Commits" to load them.
        </p>
      )}

      {commits && commits.length > 0 && (
        <div className="flex flex-col gap-2">
          {pageItems.map((commit) => (
            <CommitCard
              key={commit.id}
              commit={commit}
              selected={selectedCommitIds.has(commit.id)}
              onToggle={() => onToggleCommit(commit.id)}
            />
          ))}
          <Pagination
            page={page}
            totalPages={totalPages}
            hasNext={hasNext}
            hasPrev={hasPrev}
            onNext={nextPage}
            onPrev={prevPage}
          />
        </div>
      )}
    </div>
  )
}
