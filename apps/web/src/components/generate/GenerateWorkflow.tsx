import { useState, useEffect } from 'react'
import { useSearchParams } from 'react-router-dom'
import type { GeneratedContent } from '@/lib/types'
import { usePlatformParam } from '@/hooks/usePlatformParam'
import { useTrendingQuery } from '@/hooks/useTrending'
import { useGenerateMutation } from '@/hooks/useGenerate'
import { useUpdateStatusMutation } from '@/hooks/useHistory'
import { useRepoGenerate } from '@/hooks/useRepoGenerate'
import { PostSelector } from './PostSelector'
import { GenerateSettings, type GenerateConfig } from './GenerateSettings'
import { VariationCard } from './VariationCard'
import { ArrowLeft, ArrowRight, Sparkles } from 'lucide-react'

export function GenerateWorkflow() {
  const activePlatform = usePlatformParam()
  const [searchParams, setSearchParams] = useSearchParams()
  const [step, setStep] = useState(0)
  const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set())
  const [results, setResults] = useState<GeneratedContent[]>([])
  const [isRepost, setIsRepost] = useState(false)
  const [sourceType, setSourceType] = useState<'trending' | 'code'>('trending')
  const [commitIds, setCommitIds] = useState<number[]>([])

  const steps =
    sourceType === 'code'
      ? ['Configure', 'Generating', 'Review']
      : ['Select Posts', 'Configure', 'Generating', 'Review']

  const platformDefaultChars = activePlatform === 'linkedin' ? 2000 : 280
  const [config, setConfig] = useState<GenerateConfig>({
    target_platform: activePlatform,
    count: 3,
    max_chars: platformDefaultChars,
    force_image: false,
    include_code_images: true,
  })

  // Read URL search params on mount
  useEffect(() => {
    const idsParam = searchParams.get('ids')
    const stepParam = searchParams.get('step')
    const sourceParam = searchParams.get('source')
    const commitIdsParam = searchParams.get('commitIds')

    if (idsParam) {
      const ids = idsParam
        .split(',')
        .map(Number)
        .filter((n) => !isNaN(n) && n > 0)
      if (ids.length > 0) {
        setSelectedIds(new Set(ids))
      }
    }
    if (stepParam) {
      const s = Number(stepParam)
      if (!isNaN(s) && s >= 0) {
        setStep(s)
      }
    }
    const repostParam = searchParams.get('repost')
    if (repostParam === 'true') {
      setIsRepost(true)
    }
    if (sourceParam === 'code' && commitIdsParam) {
      const cIds = commitIdsParam
        .split(',')
        .map(Number)
        .filter((n) => !isNaN(n) && n > 0)
      if (cIds.length > 0) {
        setSourceType('code')
        setCommitIds(cIds)
        setStep(0) // Configure step for code source
      }
    }
    if (idsParam || stepParam || sourceParam) {
      setSearchParams({}, { replace: true })
    }
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    if (isRepost) {
      setConfig((prev) => ({ ...prev, max_chars: 200, force_image: false }))
    }
  }, [isRepost])

  const { data: trending, isLoading: trendingLoading } = useTrendingQuery({
    platform: activePlatform,
  })
  const generate = useGenerateMutation()
  const repoGenerate = useRepoGenerate()
  const updateStatus = useUpdateStatusMutation()

  const toggleSelect = (id: number) => {
    setSelectedIds((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  const generatingStep = sourceType === 'code' ? 1 : 2
  const reviewStep = sourceType === 'code' ? 2 : 3
  const configStep = sourceType === 'code' ? 0 : 1

  const handleGenerate = () => {
    setStep(generatingStep)
    if (sourceType === 'code') {
      repoGenerate.mutate({
        commit_ids: commitIds,
        platform: config.target_platform,
        count: config.count,
        style_direction: config.style_direction || undefined,
        include_code_images: config.include_code_images,
        code_image_template: config.code_image_template || undefined,
        code_image_theme: config.code_image_theme || undefined,
      })
    } else {
      generate.mutate({
        trending_post_ids: Array.from(selectedIds),
        target_platform: config.target_platform,
        count: config.count,
        max_chars: config.max_chars,
        force_image: config.force_image,
        is_repost: isRepost,
      })
    }
  }

  // Move to review when generation completes
  if (sourceType === 'trending' && step === generatingStep && !generate.isGenerating && generate.result) {
    setResults(generate.result)
    setStep(reviewStep)
  }

  if (sourceType === 'code' && step === generatingStep && !repoGenerate.isLoading && repoGenerate.result) {
    setResults(repoGenerate.result)
    setStep(reviewStep)
  }

  const activeProgress = sourceType === 'code' ? repoGenerate.progress : generate.progress
  const activeError = sourceType === 'code' ? repoGenerate.error : generate.error

  return (
    <div>
      {/* Step indicator */}
      <div className="mb-6 flex items-center gap-2">
        {steps.map((label, i) => (
          <div key={label} className="flex items-center gap-2">
            <div
              className={`flex h-7 w-7 items-center justify-center rounded-full text-xs font-medium ${
                i <= step
                  ? 'bg-[var(--color-accent)] text-white'
                  : 'bg-[var(--color-card)] text-[var(--color-text-secondary)]'
              }`}
            >
              {i + 1}
            </div>
            <span
              className={`text-sm ${
                i <= step ? 'text-[var(--color-text)]' : 'text-[var(--color-text-secondary)]'
              }`}
            >
              {label}
            </span>
            {i < steps.length - 1 && (
              <div className="mx-2 h-px w-8 bg-[var(--color-border)]" />
            )}
          </div>
        ))}
      </div>

      {isRepost && (
        <div className="mb-4 flex items-center gap-2">
          <span className="rounded-full bg-cyan-500/10 px-3 py-1 text-xs font-medium text-cyan-400">
            {activePlatform === 'linkedin' ? 'Repost mode' : 'Quote Tweet mode'}
          </span>
        </div>
      )}

      {/* Navigation — between step indicator and step content */}
      <div className="mb-6 flex items-center justify-between">
        {step > 0 && step < generatingStep && sourceType === 'trending' ? (
          <button
            onClick={() => setStep(step - 1)}
            className="flex items-center gap-1.5 rounded-[var(--radius-button)] border border-[var(--color-border)] px-4 py-2 text-sm text-[var(--color-text-secondary)] transition-colors hover:text-[var(--color-text)]"
          >
            <ArrowLeft size={16} />
            Back
          </button>
        ) : (
          <div />
        )}
        <div className="flex items-center gap-3">
          {step === configStep && sourceType === 'code' && commitIds.length > 0 && (
            <span className="text-sm text-[var(--color-accent)]">
              {commitIds.length} commit{commitIds.length > 1 ? 's' : ''} selected
            </span>
          )}
          {step === 0 && sourceType === 'trending' && selectedIds.size > 0 && (
            <>
              <span className="text-sm text-[var(--color-accent)]">
                {selectedIds.size} post{selectedIds.size > 1 ? 's' : ''} selected
              </span>
              <button
                onClick={() => setStep(1)}
                className="flex items-center gap-1.5 rounded-[var(--radius-button)] bg-[var(--color-accent)] px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-[var(--color-accent-hover)]"
              >
                Next
                <ArrowRight size={16} />
              </button>
            </>
          )}
          {step === configStep && (
            <button
              onClick={handleGenerate}
              className="flex items-center gap-1.5 rounded-[var(--radius-button)] bg-[var(--color-accent)] px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-[var(--color-accent-hover)]"
            >
              <Sparkles size={16} />
              Generate
            </button>
          )}
          {step === reviewStep && (
            <button
              onClick={() => {
                setStep(0)
                setSelectedIds(new Set())
                setResults([])
                if (sourceType === 'code') {
                  setSourceType('trending')
                  setCommitIds([])
                }
              }}
              className="rounded-[var(--radius-button)] border border-[var(--color-border)] px-4 py-2 text-sm text-[var(--color-text-secondary)] transition-colors hover:text-[var(--color-text)]"
            >
              Start Over
            </button>
          )}
        </div>
      </div>

      {/* Step content */}
      {step === 0 && sourceType === 'trending' && (
        <PostSelector
          posts={trending}
          isLoading={trendingLoading}
          selectedIds={selectedIds}
          onToggleSelect={toggleSelect}
        />
      )}

      {step === configStep && (
        <div className={sourceType === 'code' ? 'max-w-2xl' : 'max-w-md'}>
          <GenerateSettings config={config} onChange={setConfig} isRepost={isRepost} sourceType={sourceType} />
        </div>
      )}

      {step === generatingStep && (
        <div className="flex flex-col items-center py-16">
          <Sparkles size={48} className="mb-4 animate-pulse text-[var(--color-accent)]" />
          <h3 className="mb-2 text-lg font-semibold text-[var(--color-text)]">
            Generating content...
          </h3>
          {activeProgress && (
            <>
              <p className="mb-4 text-sm text-[var(--color-text-secondary)]">
                {activeProgress.message}
              </p>
              <div className="h-2 w-64 rounded-full bg-[var(--color-border)]">
                <div
                  className="h-2 rounded-full bg-[var(--color-accent)] transition-all"
                  style={{ width: `${activeProgress.percentage}%` }}
                />
              </div>
            </>
          )}
          {activeError && (
            <p className="mt-4 text-sm text-red-400">{activeError}</p>
          )}
        </div>
      )}

      {step === reviewStep && (
        <div className="flex flex-col gap-4">
          {results.map((content) => (
            <VariationCard
              key={content.id}
              content={content}
              onApprove={() =>
                updateStatus.mutate(
                  { id: content.id, status: 'approved' },
                  {
                    onSuccess: (updated) =>
                      setResults((prev) => prev.map((c) => (c.id === updated.id ? updated : c))),
                  },
                )
              }
              onReject={() =>
                updateStatus.mutate(
                  { id: content.id, status: 'draft' },
                  {
                    onSuccess: (updated) =>
                      setResults((prev) => prev.map((c) => (c.id === updated.id ? updated : c))),
                  },
                )
              }
              onEdit={() => {
                // In a full implementation, this would PATCH the content
              }}
            />
          ))}
          {results.length === 0 && (
            <p className="py-8 text-center text-sm text-[var(--color-text-secondary)]">
              No content was generated. Try again with different settings.
            </p>
          )}
        </div>
      )}
    </div>
  )
}
