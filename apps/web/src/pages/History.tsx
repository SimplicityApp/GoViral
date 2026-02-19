import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { usePlatformParam } from '@/hooks/usePlatformParam'
import { useHistoryQuery, useUpdateStatusMutation, useDeleteContentMutation } from '@/hooks/useHistory'
import { HistoryFilters } from '@/components/history/HistoryFilters'
import { HistoryList } from '@/components/history/HistoryList'

export function History() {
  const navigate = useNavigate()
  const platform = usePlatformParam()
  const [activeStatus, setActiveStatus] = useState('all')
  const statusFilter = activeStatus === 'all' ? undefined : activeStatus
  const { data: itemsRaw, isLoading } = useHistoryQuery(statusFilter)
  const items = itemsRaw?.filter((i) => i.target_platform === platform)
  const updateStatus = useUpdateStatusMutation()
  const deleteContent = useDeleteContentMutation()

  const handleStatusChange = (id: number, status: 'draft' | 'approved' | 'posted') => {
    if (status === 'posted') {
      navigate(`/${platform}/publish?id=${id}`)
      return
    }
    updateStatus.mutate({ id, status })
  }

  const handleDelete = (id: number) => {
    deleteContent.mutate(id)
  }

  return (
    <div className="mx-auto max-w-3xl p-6">
      <div className="mb-6 flex items-center justify-between">
        <h2 className="text-lg font-semibold text-[var(--color-text)]">History</h2>
        <HistoryFilters activeStatus={activeStatus} onStatusChange={setActiveStatus} />
      </div>
      <HistoryList
        items={items}
        isLoading={isLoading}
        onStatusChange={handleStatusChange}
        onDelete={handleDelete}
      />
    </div>
  )
}
