const statuses = ['all', 'draft', 'approved', 'posted'] as const

interface HistoryFiltersProps {
  activeStatus: string
  onStatusChange: (status: string) => void
}

export function HistoryFilters({ activeStatus, onStatusChange }: HistoryFiltersProps) {
  return (
    <div className="flex gap-1 rounded-lg border border-[var(--color-border)] p-1">
      {statuses.map((status) => (
        <button
          key={status}
          onClick={() => onStatusChange(status)}
          className={`rounded-md px-3 py-1.5 text-sm font-medium capitalize transition-colors ${
            activeStatus === status
              ? 'bg-[var(--color-accent)] text-white'
              : 'text-[var(--color-text-secondary)] hover:text-[var(--color-text)]'
          }`}
        >
          {status}
        </button>
      ))}
    </div>
  )
}
