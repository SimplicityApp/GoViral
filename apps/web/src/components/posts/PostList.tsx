import type { Post } from '@/lib/types'
import { PostCard } from '@/components/shared/PostCard'
import { EmptyState } from '@/components/shared/EmptyState'
import { LoadingSpinner } from '@/components/shared/LoadingSpinner'
import { FileText } from 'lucide-react'

interface PostListProps {
  posts: Post[] | undefined
  isLoading: boolean
  onFetch: () => void
}

export function PostList({ posts, isLoading, onFetch }: PostListProps) {
  if (isLoading) return <LoadingSpinner />

  if (!posts?.length) {
    return (
      <EmptyState
        icon={FileText}
        title="No posts yet"
        description="Fetch your posts from the connected platform to see them here."
        action={{ label: 'Fetch Posts', onClick: onFetch }}
      />
    )
  }

  return (
    <div className="flex flex-col gap-3">
      {posts.map((post) => (
        <PostCard key={post.id} post={post} />
      ))}
    </div>
  )
}
