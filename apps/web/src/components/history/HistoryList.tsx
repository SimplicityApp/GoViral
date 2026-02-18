import type { GeneratedContent } from '@/lib/types'
import { ContentCard } from '@/components/shared/ContentCard'
import { EmptyState } from '@/components/shared/EmptyState'
import { LoadingSpinner } from '@/components/shared/LoadingSpinner'
import { Clock } from 'lucide-react'

interface HistoryListProps {
  items: GeneratedContent[] | undefined
  isLoading: boolean
  onStatusChange: (id: number, status: 'draft' | 'approved' | 'posted') => void
  onDelete: (id: number) => void
}

export function HistoryList({ items, isLoading, onStatusChange, onDelete }: HistoryListProps) {
  if (isLoading) return <LoadingSpinner />

  if (!items?.length) {
    return (
      <EmptyState
        icon={Clock}
        title="No history"
        description="Generated content will appear here."
      />
    )
  }

  return (
    <div className="flex flex-col gap-3">
      {items.map((item) => (
        <ContentCard
          key={item.id}
          content={item}
          onStatusChange={(status) => onStatusChange(item.id, status)}
          onDelete={() => onDelete(item.id)}
        />
      ))}
    </div>
  )
}
