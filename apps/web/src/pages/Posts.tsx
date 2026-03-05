import { useNavigate } from 'react-router-dom'
import { usePlatformParam } from '@/hooks/usePlatformParam'
import { usePostsQuery, useFetchPostsMutation } from '@/hooks/usePosts'
import { PostFilters } from '@/components/posts/PostFilters'
import { PostList } from '@/components/posts/PostList'

export function Posts() {
  const platform = usePlatformParam()
  const navigate = useNavigate()
  const { data: posts, isLoading } = usePostsQuery(platform)
  const fetchMutation = useFetchPostsMutation()

  const handleFetch = () => {
    fetchMutation.mutate({ platform: platform })
  }

  return (
    <div className="mx-auto max-w-3xl p-6">
      <div className="mb-6 flex items-center justify-between">
        <h2 className="text-lg font-semibold text-[var(--color-text)]">My Posts</h2>
        <PostFilters onFetch={handleFetch} isFetching={fetchMutation.isRunning} />
      </div>

      {fetchMutation.error && (
        <div className="mb-4 rounded-lg border border-red-300 bg-red-50 p-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-950 dark:text-red-300">
          <div className="flex items-center justify-between gap-3">
            <span>{fetchMutation.error}</span>
            <button
              onClick={() => navigate('/settings')}
              className="shrink-0 rounded-md bg-red-100 px-3 py-1 text-xs font-medium text-red-800 transition-colors hover:bg-red-200 dark:bg-red-900 dark:text-red-200 dark:hover:bg-red-800"
            >
              Go to Settings
            </button>
          </div>
        </div>
      )}

      {fetchMutation.progress && !fetchMutation.error && (
        <div className="mb-4 rounded-lg border border-[var(--color-border)] bg-[var(--color-card)] p-3">
          <div className="mb-1 text-sm text-[var(--color-text)]">
            {fetchMutation.progress.message}
          </div>
          <div className="h-1.5 w-full rounded-full bg-[var(--color-border)]">
            <div
              className="h-1.5 rounded-full bg-[var(--color-accent)] transition-all"
              style={{ width: `${fetchMutation.progress.percentage}%` }}
            />
          </div>
        </div>
      )}

      <PostList posts={posts} isLoading={isLoading} onFetch={handleFetch} />
    </div>
  )
}
