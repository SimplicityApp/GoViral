import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { toast } from 'sonner'
import {
  useReposQuery,
  useAddRepoMutation,
  useDeleteRepoMutation,
  useUpdateRepoSettingsMutation,
  useRepoCommitsQuery,
} from '@/hooks/useRepos'
import { useFetchCommits } from '@/hooks/useRepoGenerate'
import { usePlatformParam } from '@/hooks/usePlatformParam'
import { RepoList } from '@/components/repo/RepoList'
import { CommitList } from '@/components/repo/CommitList'
import { Sparkles, Code } from 'lucide-react'

export function CodeToPost() {
  const navigate = useNavigate()
  const activePlatform = usePlatformParam()
  const { data: repos, isLoading: reposLoading } = useReposQuery()
  const addRepo = useAddRepoMutation()
  const deleteRepo = useDeleteRepoMutation()
  const updateSettings = useUpdateRepoSettingsMutation()

  const [selectedRepoId, setSelectedRepoId] = useState<number | null>(null)
  const [selectedCommitIds, setSelectedCommitIds] = useState<Set<number>>(new Set())
  const [autoFetch, setAutoFetch] = useState(false)

  const { data: commits, isLoading: commitsLoading } = useRepoCommitsQuery(selectedRepoId)
  const fetchCommits = useFetchCommits(selectedRepoId)

  useEffect(() => {
    if (fetchCommits.error) {
      toast.error(`Failed to fetch commits: ${fetchCommits.error}`)
    }
  }, [fetchCommits.error])

  // Auto-fetch commits for newly imported repos
  useEffect(() => {
    if (autoFetch && selectedRepoId && !fetchCommits.isLoading) {
      setAutoFetch(false)
      fetchCommits.mutate({})
    }
  }, [autoFetch, selectedRepoId, fetchCommits.isLoading])

  const handleAddRepo = (owner: string, name: string) => {
    addRepo.mutate(
      { owner, name },
      {
        onSuccess: (repo) => {
          toast.success(`Added ${owner}/${name}`)
          setSelectedRepoId(repo.id)
          setAutoFetch(true)
        },
        onError: (err) => toast.error(`Failed to add repo: ${err.message}`),
      },
    )
  }

  const handleDeleteRepo = (id: number) => {
    const repo = repos?.find((r) => r.id === id)
    deleteRepo.mutate(id, {
      onSuccess: () => toast.success(`Removed ${repo?.full_name ?? 'repo'}`),
      onError: (err) => toast.error(`Failed to remove repo: ${err.message}`),
    })
    if (selectedRepoId === id) {
      setSelectedRepoId(null)
      setSelectedCommitIds(new Set())
    }
  }

  const handleSelectRepo = (id: number) => {
    setSelectedRepoId(id)
    setSelectedCommitIds(new Set())
  }

  const handleToggleCommit = (id: number) => {
    setSelectedCommitIds((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  const handleFetchCommits = () => {
    if (!selectedRepoId) return
    fetchCommits.mutate({})
  }

  const handleGenerate = () => {
    if (selectedCommitIds.size === 0) return
    const commitIdsStr = Array.from(selectedCommitIds).join(',')
    navigate(`/${activePlatform}/generate?source=code&commitIds=${commitIdsStr}`)
  }

  return (
    <div className="mx-auto max-w-4xl p-6">
      <div className="mb-6 flex items-center gap-2">
        <Code size={20} className="text-[var(--color-accent)]" />
        <h2 className="text-lg font-semibold text-[var(--color-text)]">Code to Post</h2>
      </div>

      {/* Section 1: Connected Repos */}
      <section className="mb-8">
        <h3 className="mb-4 text-sm font-semibold uppercase tracking-wide text-[var(--color-text-secondary)]">
          Connected Repositories
        </h3>
        <RepoList
          repos={repos}
          isLoading={reposLoading}
          onAdd={handleAddRepo}
          isAdding={addRepo.isPending}
          onDelete={handleDeleteRepo}
          onUpdateSettings={(id, targetAudience, links) => {
            updateSettings.mutate(
              { id, target_audience: targetAudience, links },
              {
                onSuccess: () => toast.success('Settings saved'),
                onError: (err) => toast.error(`Failed to save settings: ${err.message}`),
              },
            )
          }}
          isSavingSettings={updateSettings.isPending}
        />
      </section>

      {/* Section 2: Commits */}
      <section className="mb-8">
        <h3 className="mb-4 text-sm font-semibold uppercase tracking-wide text-[var(--color-text-secondary)]">
          Commits
        </h3>
        {(!repos || repos.length === 0) && !reposLoading ? (
          <p className="text-sm text-[var(--color-text-secondary)]">
            Add a repository above to browse its commits.
          </p>
        ) : (
          <CommitList
            repos={repos ?? []}
            selectedRepoId={selectedRepoId}
            onSelectRepo={handleSelectRepo}
            commits={commits}
            isLoadingCommits={commitsLoading}
            selectedCommitIds={selectedCommitIds}
            onToggleCommit={handleToggleCommit}
            onFetchCommits={handleFetchCommits}
            isFetching={fetchCommits.isLoading}
            fetchProgress={fetchCommits.progress}
          />
        )}
      </section>

      {/* Generate button */}
      {selectedCommitIds.size > 0 && (
        <section>
          <div className="flex items-center gap-3">
            <span className="text-sm text-[var(--color-text-secondary)]">
              {selectedCommitIds.size} commit{selectedCommitIds.size !== 1 ? 's' : ''} selected
            </span>
            <button
              onClick={handleGenerate}
              className="flex items-center gap-1.5 rounded-[var(--radius-button)] bg-[var(--color-accent)] px-5 py-2 text-sm font-medium text-white transition-colors hover:bg-[var(--color-accent-hover)]"
            >
              <Sparkles size={16} />
              Generate Posts
            </button>
          </div>
        </section>
      )}
    </div>
  )
}
