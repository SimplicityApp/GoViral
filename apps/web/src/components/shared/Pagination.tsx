import { ChevronLeft, ChevronRight } from 'lucide-react'

interface PaginationProps {
  page: number
  totalPages: number
  hasNext: boolean
  hasPrev: boolean
  onNext: () => void
  onPrev: () => void
}

export function Pagination({ page, totalPages, hasNext, hasPrev, onNext, onPrev }: PaginationProps) {
  if (totalPages <= 1) return null

  return (
    <div className="flex items-center justify-center gap-3 pt-4">
      <button
        onClick={onPrev}
        disabled={!hasPrev}
        className="flex items-center gap-1 rounded-[var(--radius-button)] border border-[var(--color-border)] px-3 py-1.5 text-sm text-[var(--color-text-secondary)] transition-colors hover:text-[var(--color-text)] disabled:opacity-40 disabled:hover:text-[var(--color-text-secondary)]"
      >
        <ChevronLeft size={16} />
        Prev
      </button>
      <span className="text-sm text-[var(--color-text-secondary)]">
        Page {page} of {totalPages}
      </span>
      <button
        onClick={onNext}
        disabled={!hasNext}
        className="flex items-center gap-1 rounded-[var(--radius-button)] border border-[var(--color-border)] px-3 py-1.5 text-sm text-[var(--color-text-secondary)] transition-colors hover:text-[var(--color-text)] disabled:opacity-40 disabled:hover:text-[var(--color-text-secondary)]"
      >
        Next
        <ChevronRight size={16} />
      </button>
    </div>
  )
}
