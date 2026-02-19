import { useSearchParams } from 'react-router-dom'
import { usePlatformParam } from '@/hooks/usePlatformParam'
import { useHistoryQuery } from '@/hooks/useHistory'
import { PublishPanel } from '@/components/publish/PublishPanel'
import { EmptyState } from '@/components/shared/EmptyState'
import { LoadingSpinner } from '@/components/shared/LoadingSpinner'
import { Send } from 'lucide-react'

export function Publish() {
  const [searchParams] = useSearchParams()
  const platform = usePlatformParam()
  const preselectedId = searchParams.get('id') ? Number(searchParams.get('id')) : undefined
  const { data: approvedRaw, isLoading } = useHistoryQuery('approved')
  const approved = approvedRaw?.filter((i) => i.target_platform === platform)

  if (isLoading) return <LoadingSpinner />

  return (
    <div className="mx-auto max-w-3xl p-6">
      <h2 className="mb-6 text-lg font-semibold text-[var(--color-text)]">Publish</h2>
      {approved?.length ? (
        <PublishPanel items={approved} initialSelectedId={preselectedId} />
      ) : (
        <EmptyState
          icon={Send}
          title="Nothing to publish"
          description="Approve generated content from the History page first."
        />
      )}
    </div>
  )
}
