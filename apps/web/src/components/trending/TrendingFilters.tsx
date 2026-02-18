interface TrendingFiltersProps {
  period: string
  minLikes: string
  niche: string
  onPeriodChange: (v: string) => void
  onMinLikesChange: (v: string) => void
  onNicheChange: (v: string) => void
  onDiscover: () => void
  isDiscovering: boolean
}

export function TrendingFilters({
  period,
  minLikes,
  niche,
  onPeriodChange,
  onMinLikesChange,
  onNicheChange,
  onDiscover,
  isDiscovering,
}: TrendingFiltersProps) {
  return (
    <div className="flex flex-wrap items-center gap-3">
      <select
        value={period}
        onChange={(e) => onPeriodChange(e.target.value)}
        className="rounded-[var(--radius-input)] border border-[var(--color-border)] bg-[var(--color-card)] px-3 py-2 text-sm text-[var(--color-text)]"
      >
        <option value="24h">Last 24h</option>
        <option value="7d">Last 7 days</option>
        <option value="30d">Last 30 days</option>
      </select>

      <input
        type="number"
        placeholder="Min likes"
        value={minLikes}
        onChange={(e) => onMinLikesChange(e.target.value)}
        className="w-28 rounded-[var(--radius-input)] border border-[var(--color-border)] bg-[var(--color-card)] px-3 py-2 text-sm text-[var(--color-text)] placeholder:text-[var(--color-text-secondary)]"
      />

      <input
        type="text"
        placeholder="Niche"
        value={niche}
        onChange={(e) => onNicheChange(e.target.value)}
        className="w-32 rounded-[var(--radius-input)] border border-[var(--color-border)] bg-[var(--color-card)] px-3 py-2 text-sm text-[var(--color-text)] placeholder:text-[var(--color-text-secondary)]"
      />

      <button
        onClick={onDiscover}
        disabled={isDiscovering}
        className="rounded-[var(--radius-button)] bg-[var(--color-accent)] px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-[var(--color-accent-hover)] disabled:opacity-50"
      >
        {isDiscovering ? 'Discovering...' : 'Discover'}
      </button>
    </div>
  )
}
