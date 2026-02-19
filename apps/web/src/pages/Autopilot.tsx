import { useState } from 'react'
import { Bot } from 'lucide-react'
import { toast } from 'sonner'
import {
  useDaemonStatusQuery,
  useDaemonBatchesQuery,
  useBatchActionMutation,
  useDaemonRunNowMutation,
  useDaemonStartMutation,
  useDaemonStopMutation,
} from '@/hooks/useDaemon'
import { DaemonStatusCard } from '@/components/autopilot/DaemonStatusCard'
import { BatchCard } from '@/components/autopilot/BatchCard'
import { DaemonConfigPanel } from '@/components/autopilot/DaemonConfigPanel'
import { LoadingSpinner } from '@/components/shared/LoadingSpinner'
import { EmptyState } from '@/components/shared/EmptyState'
import type { DaemonBatch } from '@/lib/types'

type TabFilter = 'all' | 'pending' | 'approved' | 'posted' | 'rejected'

const TABS: { value: TabFilter; label: string }[] = [
  { value: 'all', label: 'All' },
  { value: 'pending', label: 'Pending' },
  { value: 'approved', label: 'Approved' },
  { value: 'posted', label: 'Posted' },
  { value: 'rejected', label: 'Rejected' },
]

export function Autopilot() {
  const [activeTab, setActiveTab] = useState<TabFilter>('all')
  const [activeSection, setActiveSection] = useState<'batches' | 'config'>('batches')

  const { data: status, isLoading: statusLoading } = useDaemonStatusQuery()
  const { data: batches, isLoading: batchesLoading } = useDaemonBatchesQuery(
    undefined,
    activeTab === 'all' ? undefined : activeTab,
  )

  const batchAction = useBatchActionMutation()
  const runNow = useDaemonRunNowMutation()
  const start = useDaemonStartMutation()
  const stop = useDaemonStopMutation()

  const handleStart = () => {
    start.mutate(undefined, {
      onSuccess: () => toast.success('Daemon started'),
      onError: () => toast.error('Failed to start daemon'),
    })
  }

  const handleStop = () => {
    stop.mutate(undefined, {
      onSuccess: () => toast.success('Daemon stopped'),
      onError: () => toast.error('Failed to stop daemon'),
    })
  }

  const handleRunNow = () => {
    runNow.mutate(undefined, {
      onSuccess: () => toast.success('Daemon run triggered'),
      onError: () => toast.error('Failed to trigger daemon run'),
    })
  }

  const handleApprove = (id: number) => {
    batchAction.mutate(
      { id, action: 'approve' },
      {
        onSuccess: () => toast.success('Batch approved'),
        onError: () => toast.error('Failed to approve batch'),
      },
    )
  }

  const handleReject = (id: number) => {
    batchAction.mutate(
      { id, action: 'reject' },
      {
        onSuccess: () => toast.success('Batch rejected'),
        onError: () => toast.error('Failed to reject batch'),
      },
    )
  }

  const filteredBatches: DaemonBatch[] = batches ?? []

  return (
    <div className="mx-auto max-w-3xl p-6">
      <h2 className="mb-6 text-lg font-semibold text-[var(--color-text)]">Autopilot</h2>

      {/* Status panel */}
      <section className="mb-6">
        {statusLoading ? (
          <LoadingSpinner />
        ) : status ? (
          <DaemonStatusCard
            status={status}
            onStart={handleStart}
            onStop={handleStop}
            onRunNow={handleRunNow}
            isStarting={start.isPending}
            isStopping={stop.isPending}
            isRunning={runNow.isPending}
          />
        ) : (
          <div className="rounded-[var(--radius-card)] border border-[var(--color-border)] bg-[var(--color-card)] p-5 text-sm text-[var(--color-text-secondary)]">
            Daemon status unavailable
          </div>
        )}
      </section>

      {/* Section tabs */}
      <div className="mb-5 flex gap-1 border-b border-[var(--color-border)]">
        <button
          onClick={() => setActiveSection('batches')}
          className={`px-4 pb-2.5 text-sm font-medium transition-colors ${
            activeSection === 'batches'
              ? 'border-b-2 border-[var(--color-accent)] text-[var(--color-accent)]'
              : 'text-[var(--color-text-secondary)] hover:text-[var(--color-text)]'
          }`}
        >
          Batches
        </button>
        <button
          onClick={() => setActiveSection('config')}
          className={`px-4 pb-2.5 text-sm font-medium transition-colors ${
            activeSection === 'config'
              ? 'border-b-2 border-[var(--color-accent)] text-[var(--color-accent)]'
              : 'text-[var(--color-text-secondary)] hover:text-[var(--color-text)]'
          }`}
        >
          Config
        </button>
      </div>

      {activeSection === 'batches' && (
        <>
          {/* Status filter tabs */}
          <div className="mb-4 flex flex-wrap gap-1">
            {TABS.map((tab) => (
              <button
                key={tab.value}
                onClick={() => setActiveTab(tab.value)}
                className={`rounded-[var(--radius-button)] px-3 py-1 text-sm font-medium transition-colors ${
                  activeTab === tab.value
                    ? 'bg-[var(--color-accent)]/10 text-[var(--color-accent)]'
                    : 'text-[var(--color-text-secondary)] hover:bg-[var(--color-card)] hover:text-[var(--color-text)]'
                }`}
              >
                {tab.label}
              </button>
            ))}
          </div>

          {batchesLoading ? (
            <LoadingSpinner />
          ) : filteredBatches.length > 0 ? (
            <div className="flex flex-col gap-3">
              {filteredBatches.map((batch) => (
                <BatchCard
                  key={batch.id}
                  batch={batch}
                  onApprove={handleApprove}
                  onReject={handleReject}
                  isActing={batchAction.isPending}
                />
              ))}
            </div>
          ) : (
            <EmptyState
              icon={Bot}
              title="No batches yet"
              description="The daemon will generate batches according to the configured schedule."
            />
          )}
        </>
      )}

      {activeSection === 'config' && (
        <DaemonConfigPanel />
      )}
    </div>
  )
}
