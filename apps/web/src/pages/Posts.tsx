import { usePlatformParam } from '@/hooks/usePlatformParam'
import { usePostsQuery, useFetchPostsMutation } from '@/hooks/usePosts'
import { PostFilters } from '@/components/posts/PostFilters'
import { PostList } from '@/components/posts/PostList'

export function Posts() {
  const platform = usePlatformParam()
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

      {fetchMutation.progress && (
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
