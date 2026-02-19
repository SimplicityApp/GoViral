interface TrendingFiltersProps {
  period: string
  minLikes: string
  niche: string
  niches: string[]
  onPeriodChange: (v: string) => void
  onMinLikesChange: (v: string) => void
  onNicheChange: (v: string) => void
  onDiscover: () => void
  onManageNiches: () => void
  isDiscovering: boolean
}

export function TrendingFilters({
  period,
  minLikes,
  niche,
  niches,
  onPeriodChange,
  onMinLikesChange,
  onNicheChange,
  onDiscover,
  onManageNiches,
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

      <select
        value={niche}
        onChange={(e) => onNicheChange(e.target.value)}
        className="rounded-[var(--radius-input)] border border-[var(--color-border)] bg-[var(--color-card)] px-3 py-2 text-sm text-[var(--color-text)]"
      >
        <option value="">All niches</option>
        {niches.map((n) => (
          <option key={n} value={n}>
            {n}
          </option>
        ))}
      </select>

      <button
        onClick={onDiscover}
        disabled={isDiscovering}
        className="rounded-[var(--radius-button)] bg-[var(--color-accent)] px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-[var(--color-accent-hover)] disabled:opacity-50"
      >
        {isDiscovering ? 'Discovering...' : 'Discover'}
      </button>

      <button
        onClick={onManageNiches}
        className="rounded-[var(--radius-button)] border border-[var(--color-border)] px-4 py-2 text-sm font-medium text-[var(--color-text-secondary)] transition-colors hover:text-[var(--color-text)]"
      >
        Manage Niches
      </button>
    </div>
  )
}
