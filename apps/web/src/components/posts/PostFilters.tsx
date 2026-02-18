interface PostFiltersProps {
  onFetch: () => void
  isFetching: boolean
}

export function PostFilters({ onFetch, isFetching }: PostFiltersProps) {
  return (
    <div className="flex items-center gap-3">
      <button
        onClick={onFetch}
        disabled={isFetching}
        className="rounded-[var(--radius-button)] bg-[var(--color-accent)] px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-[var(--color-accent-hover)] disabled:opacity-50"
      >
        {isFetching ? 'Fetching...' : 'Fetch Posts'}
      </button>
    </div>
  )
}
