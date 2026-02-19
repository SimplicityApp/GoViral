import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { usePlatformParam } from '@/hooks/usePlatformParam'
import { useHistoryQuery, useUpdateStatusMutation, useDeleteContentMutation } from '@/hooks/useHistory'
import { HistoryFilters } from '@/components/history/HistoryFilters'
import { HistoryList } from '@/components/history/HistoryList'
import { Trash2, X } from 'lucide-react'

export function History() {
  const navigate = useNavigate()
  const platform = usePlatformParam()
  const [activeStatus, setActiveStatus] = useState('all')
  const statusFilter = activeStatus === 'all' ? undefined : activeStatus
  const { data: itemsRaw, isLoading } = useHistoryQuery(statusFilter)
  const items = itemsRaw?.filter((i) => i.target_platform === platform)
  const updateStatus = useUpdateStatusMutation()
  const deleteContent = useDeleteContentMutation()

  const [selectionMode, setSelectionMode] = useState(false)
  const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set())

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

  const toggleSelect = (id: number) => {
    setSelectedIds((prev) => {
      const next = new Set(prev)
      next.has(id) ? next.delete(id) : next.add(id)
      return next
    })
  }

  const selectAll = () => {
    if (!items) return
    setSelectedIds(new Set(items.map((i) => i.id)))
  }

  const clearSelection = () => {
    setSelectedIds(new Set())
    setSelectionMode(false)
  }

  const handleBulkDelete = async () => {
    await Promise.all([...selectedIds].map((id) => deleteContent.mutateAsync(id)))
    clearSelection()
  }

  const allSelected = !!items?.length && selectedIds.size === items.length

  return (
    <div className="mx-auto max-w-3xl p-6">
      <div className="mb-6 flex items-center justify-between gap-3">
        {selectionMode ? (
          <>
            <span className="text-sm font-medium text-[var(--color-text)]">
              {selectedIds.size} selected
            </span>
            <button
              onClick={allSelected ? () => setSelectedIds(new Set()) : selectAll}
              className="text-sm text-[var(--color-text-secondary)] hover:text-[var(--color-text)] transition-colors"
            >
              {allSelected ? 'Deselect all' : 'Select all'}
            </button>
            <button
              onClick={handleBulkDelete}
              disabled={selectedIds.size === 0 || deleteContent.isPending}
              className="ml-auto flex items-center gap-1.5 rounded-[var(--radius-button)] bg-red-500 px-3 py-1.5 text-sm font-medium text-white transition-colors hover:bg-red-400 disabled:opacity-50"
            >
              <Trash2 size={14} />
              Delete {selectedIds.size > 0 ? `(${selectedIds.size})` : ''}
            </button>
            <button
              onClick={clearSelection}
              className="flex items-center gap-1 text-sm text-[var(--color-text-secondary)] hover:text-[var(--color-text)] transition-colors"
            >
              <X size={14} />
              Cancel
            </button>
          </>
        ) : (
          <>
            <h2 className="text-lg font-semibold text-[var(--color-text)]">History</h2>
            <div className="flex items-center gap-2">
              <HistoryFilters activeStatus={activeStatus} onStatusChange={setActiveStatus} />
              {!!items?.length && (
                <button
                  onClick={() => setSelectionMode(true)}
                  className="rounded-[var(--radius-button)] border border-[var(--color-border)] px-3 py-1.5 text-sm font-medium text-[var(--color-text-secondary)] transition-colors hover:text-[var(--color-text)]"
                >
                  Select
                </button>
              )}
            </div>
          </>
        )}
      </div>
      <HistoryList
        items={items}
        isLoading={isLoading}
        onStatusChange={handleStatusChange}
        onDelete={handleDelete}
        selectionMode={selectionMode}
        selectedIds={selectedIds}
        onToggleSelect={toggleSelect}
      />
    </div>
  )
}
