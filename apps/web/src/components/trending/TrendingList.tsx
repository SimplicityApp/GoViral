import type { TrendingPost } from '@/lib/types'
import { PostCard } from '@/components/shared/PostCard'
import { EmptyState } from '@/components/shared/EmptyState'
import { LoadingSpinner } from '@/components/shared/LoadingSpinner'
import { Pagination } from '@/components/shared/Pagination'
import { usePagination } from '@/hooks/usePagination'
import { TrendingUp } from 'lucide-react'

interface TrendingListProps {
  posts: TrendingPost[] | undefined
  isLoading: boolean
  selectedIds: Set<number>
  onToggleSelect: (id: number) => void
  onRepost?: (post: TrendingPost) => void
}

export function TrendingList({ posts, isLoading, selectedIds, onToggleSelect, onRepost }: TrendingListProps) {
  const { page, totalPages, pageItems, nextPage, prevPage, hasNext, hasPrev } = usePagination(
    posts ?? []
  )

  if (isLoading) return <LoadingSpinner />

  if (!posts?.length) {
    return (
      <EmptyState
        icon={TrendingUp}
        title="No trending posts"
        description="Use the Discover button to find trending content in your niche."
      />
    )
  }

  return (
    <div className="flex flex-col gap-3">
      {pageItems.map((post) => (
        <PostCard
          key={post.id}
          post={post}
          selectable
          selected={selectedIds.has(post.id)}
          onSelect={() => onToggleSelect(post.id)}
          onRepost={onRepost ? () => onRepost(post) : undefined}
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
