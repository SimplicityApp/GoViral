import type { TrendingPost } from '@/lib/types'
import { PostCard } from '@/components/shared/PostCard'
import { EmptyState } from '@/components/shared/EmptyState'
import { LoadingSpinner } from '@/components/shared/LoadingSpinner'
import { Pagination } from '@/components/shared/Pagination'
import { usePagination } from '@/hooks/usePagination'
import { TrendingUp } from 'lucide-react'

interface PostSelectorProps {
  posts: TrendingPost[] | undefined
  isLoading: boolean
  selectedIds: Set<number>
  onToggleSelect: (id: number) => void
}

export function PostSelector({ posts, isLoading, selectedIds, onToggleSelect }: PostSelectorProps) {
  const { page, totalPages, pageItems, nextPage, prevPage, hasNext, hasPrev } = usePagination(
    posts ?? []
  )

  if (isLoading) return <LoadingSpinner />

  if (!posts?.length) {
    return (
      <EmptyState
        icon={TrendingUp}
        title="No trending posts available"
        description="Go to the Trending page to discover posts first."
      />
    )
  }

  return (
    <div className="flex flex-col gap-3">
      <p className="text-sm text-[var(--color-text-secondary)]">
        Select posts to use as inspiration ({selectedIds.size} selected)
      </p>
      {pageItems.map((post) => (
        <PostCard
          key={post.id}
          post={post}
          selectable
          selected={selectedIds.has(post.id)}
          onSelect={() => onToggleSelect(post.id)}
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
  )
}
