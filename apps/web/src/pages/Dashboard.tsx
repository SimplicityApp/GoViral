import { useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'
import { usePlatformParam } from '@/hooks/usePlatformParam'
import { usePostsQuery } from '@/hooks/usePosts'
import { useTrendingQuery } from '@/hooks/useTrending'
import { useHistoryQuery } from '@/hooks/useHistory'
import { usePersonaQuery, useBuildPersonaMutation } from '@/hooks/usePersona'
import { useScheduleQuery, useCancelScheduleMutation, useRunDueMutation, useAcknowledgeScheduleMutation } from '@/hooks/usePublish'
import { LoadingSpinner } from '@/components/shared/LoadingSpinner'
import { formatCount, formatRelativeTime, formatScheduledTime } from '@/lib/format'
import {
  FileText,
  TrendingUp,
  Sparkles,
  Send,
  Calendar,
  User,
  Trash2,
  CheckCircle,
  RefreshCw,
} from 'lucide-react'

function StatCard({
  icon: Icon,
  label,
  value,
}: {
  icon: typeof FileText
  label: string
  value: string
}) {
  return (
    <div className="rounded-[var(--radius-card)] border border-[var(--color-border)] bg-[var(--color-card)] p-4">
      <div className="mb-2 flex items-center gap-2 text-[var(--color-text-secondary)]">
        <Icon size={16} />
        <span className="text-xs font-medium uppercase tracking-wide">{label}</span>
      </div>
      <p className="text-2xl font-bold text-[var(--color-text)]">{value}</p>
    </div>
  )
}

export function Dashboard() {
  const navigate = useNavigate()
  const platform = usePlatformParam()
  const { data: posts, isLoading: postsLoading } = usePostsQuery(platform)
  const { data: trending, isLoading: trendingLoading } = useTrendingQuery({
    platform,
  })
  const { data: historyRaw, isLoading: historyLoading } = useHistoryQuery()
  const { data: postedRaw } = useHistoryQuery('posted')
  const { data: persona } = usePersonaQuery(platform)
  const { data: scheduledRaw } = useScheduleQuery()
  const queryClient = useQueryClient()
  const buildPersona = useBuildPersonaMutation({
    onComplete: () => queryClient.invalidateQueries({ queryKey: ['persona', platform] }),
  })
  const cancelSchedule = useCancelScheduleMutation()
  const ackSchedule = useAcknowledgeScheduleMutation()
  const runDue = useRunDueMutation()

  const historyFiltered = historyRaw?.filter((i) => i.target_platform === platform)
  const history = historyFiltered?.slice(0, 5)
  const posted = postedRaw?.filter((i) => i.target_platform === platform)
  const scheduled = scheduledRaw?.filter((i) => i.target_platform === platform)

  useEffect(() => {
    runDue.mutate()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const isLoading = postsLoading || trendingLoading || historyLoading

  if (isLoading) return <LoadingSpinner />

  return (
    <div className="mx-auto max-w-4xl p-6">
      <h2 className="mb-6 text-lg font-semibold text-[var(--color-text)]">Dashboard</h2>

      {/* Stats */}
      <div className="mb-8 grid grid-cols-2 gap-4 lg:grid-cols-4">
        <StatCard icon={FileText} label="My Posts" value={formatCount(posts?.length ?? 0)} />
        <StatCard icon={TrendingUp} label="Trending" value={formatCount(trending?.length ?? 0)} />
        <StatCard icon={Sparkles} label="Generated" value={formatCount(historyFiltered?.length ?? 0)} />
        <StatCard icon={Send} label="Posted" value={formatCount(posted?.length ?? 0)} />
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        {/* Recent generated */}
        <div>
          <h3 className="mb-3 text-sm font-semibold uppercase tracking-wide text-[var(--color-text-secondary)]">
            Recent Generated
          </h3>
          <div className="flex flex-col gap-2">
            {history?.length ? (
              history.slice(0, 5).map((item) => (
                <div
                  key={item.id}
                  className="rounded-lg border border-[var(--color-border)] bg-[var(--color-card)] p-3"
                >
                  <p className="mb-1 line-clamp-2 text-sm text-[var(--color-text)]">
                    {item.generated_content}
                  </p>
                  <div className="flex items-center justify-between text-xs text-[var(--color-text-secondary)]">
                    <span className="capitalize">{item.status}</span>
                    <span>{formatRelativeTime(item.created_at)}</span>
                  </div>
                </div>
              ))
            ) : (
              <p className="text-sm text-[var(--color-text-secondary)]">
                No generated content yet.
              </p>
            )}
          </div>
        </div>

        {/* Scheduled */}
        <div>
          <h3 className="mb-3 text-sm font-semibold uppercase tracking-wide text-[var(--color-text-secondary)]">
            Scheduled Posts
          </h3>
          <div className="flex flex-col gap-2">
            {scheduled?.length ? (
              scheduled.slice(0, 5).map((item) => (
                <div
                  key={item.id}
                  className="rounded-lg border border-[var(--color-border)] bg-[var(--color-card)] p-3"
                >
                  {item.content_preview && (
                    <p className="mb-2 line-clamp-3 text-sm text-[var(--color-text)]">
                      {item.content_preview}
                    </p>
                  )}
                  <div className="flex items-center gap-2 text-xs text-[var(--color-text-secondary)]">
                    <Calendar size={12} className="text-[var(--color-accent)]" />
                    <span>{formatScheduledTime(item.scheduled_at)}</span>
                    <span className="capitalize">{item.status}</span>
                    {item.target_platform && (
                      <span className="uppercase">{item.target_platform}</span>
                    )}
                    {(item.status === 'pending' || item.status === 'scheduled') && (
                      <div className="ml-auto flex items-center gap-3">
                        <button
                          onClick={() => ackSchedule.mutate(item.id)}
                          className="flex items-center gap-1 text-green-400 transition-colors hover:text-green-300"
                        >
                          <CheckCircle size={12} />
                          Mark Posted
                        </button>
                        <button
                          onClick={() => cancelSchedule.mutate(item.id)}
                          className="flex items-center gap-1 text-red-400 transition-colors hover:text-red-300"
                        >
                          <Trash2 size={12} />
                          Cancel
                        </button>
                      </div>
                    )}
                  </div>
                </div>
              ))
            ) : (
              <p className="text-sm text-[var(--color-text-secondary)]">
                No scheduled posts.
              </p>
            )}
          </div>
        </div>
      </div>

      {/* Persona */}
      <div className="mt-6">
        <div className="mb-3 flex items-center justify-between">
          <h3 className="text-sm font-semibold uppercase tracking-wide text-[var(--color-text-secondary)]">
            Persona ({platform})
          </h3>
          {!buildPersona.isRunning && (
            <button
              onClick={() => buildPersona.mutate({ platform })}
              className="flex items-center gap-1.5 rounded-[var(--radius-button)] border border-[var(--color-border)] px-3 py-1.5 text-xs text-[var(--color-text)] transition-colors hover:bg-[var(--color-card)]"
            >
              <RefreshCw size={12} />
              {persona ? 'Refresh' : 'Generate Persona'}
            </button>
          )}
        </div>

        <div className="rounded-[var(--radius-card)] border border-[var(--color-border)] bg-[var(--color-card)] p-4">
          {buildPersona.isRunning ? (
            <div>
              <p className="mb-2 text-sm text-[var(--color-text-secondary)]">
                {buildPersona.progress?.message ?? 'Building persona…'}
              </p>
              <div className="h-1.5 w-full rounded-full bg-[var(--color-border)]">
                <div
                  className="h-1.5 rounded-full bg-[var(--color-accent)] transition-all"
                  style={{ width: `${buildPersona.progress?.percentage ?? 0}%` }}
                />
              </div>
            </div>
          ) : buildPersona.error ? (
            <p className="text-sm text-red-400">{buildPersona.error}</p>
          ) : persona ? (
            <div>
              <div className="mb-2 flex items-center gap-2">
                <User size={16} className="text-[var(--color-accent)]" />
                <span className="text-sm font-medium text-[var(--color-text)]">Voice Profile</span>
              </div>
              <p className="text-sm text-[var(--color-text-secondary)]">{persona.profile.voice_summary}</p>
            </div>
          ) : (
            <p className="text-sm text-[var(--color-text-secondary)]">
              No persona built yet. Click <strong>Generate Persona</strong> to analyse your posts.
            </p>
          )}
        </div>
      </div>

      {/* Quick actions */}
      <div className="mt-6">
        <h3 className="mb-3 text-sm font-semibold uppercase tracking-wide text-[var(--color-text-secondary)]">
          Quick Actions
        </h3>
        <div className="flex flex-wrap gap-3">
          <button
            onClick={() => navigate(`/${platform}/trending`)}
            className="flex items-center gap-2 rounded-[var(--radius-button)] border border-[var(--color-border)] px-4 py-2 text-sm text-[var(--color-text)] transition-colors hover:bg-[var(--color-card)]"
          >
            <TrendingUp size={16} />
            Discover Trending
          </button>
          <button
            onClick={() => navigate(`/${platform}/generate`)}
            className="flex items-center gap-2 rounded-[var(--radius-button)] bg-[var(--color-accent)] px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-[var(--color-accent-hover)]"
          >
            <Sparkles size={16} />
            Generate Content
          </button>
          <button
            onClick={() => navigate(`/${platform}/publish`)}
            className="flex items-center gap-2 rounded-[var(--radius-button)] border border-[var(--color-border)] px-4 py-2 text-sm text-[var(--color-text)] transition-colors hover:bg-[var(--color-card)]"
          >
            <Send size={16} />
            Publish
          </button>
        </div>
      </div>
    </div>
  )
}
