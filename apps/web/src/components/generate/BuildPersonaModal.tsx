import { useState, useEffect } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { useConfigQuery, useUpdateConfigMutation } from '@/hooks/useConfig'
import { usePostsQuery } from '@/hooks/usePosts'
import { useBuildPersonaMutation } from '@/hooks/usePersona'
import { X } from 'lucide-react'

interface BuildPersonaModalProps {
  open: boolean
  onClose: () => void
  platform: string
}

export function BuildPersonaModal({ open, onClose, platform }: BuildPersonaModalProps) {
  const queryClient = useQueryClient()
  const { data: config } = useConfigQuery()
  const { data: posts } = usePostsQuery(platform)
  const updateConfig = useUpdateConfigMutation()
  const buildPersona = useBuildPersonaMutation({
    onComplete: () => {
      void queryClient.invalidateQueries({ queryKey: ['persona', platform] })
      onClose()
    },
  })

  const [description, setDescription] = useState('')

  useEffect(() => {
    if (config?.self_description) {
      setDescription(config.self_description)
    }
  }, [config?.self_description])

  if (!open) return null

  const postCount = posts?.length ?? 0
  const canBuild = postCount > 0 || description.trim().length > 0

  const handleBuild = () => {
    if (description.trim()) {
      updateConfig.mutate({ self_description: description.trim() })
    }
    buildPersona.mutate({ platform, self_description: description.trim() })
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={onClose}>
      <div
        className="w-full max-w-lg rounded-[var(--radius-card)] border border-[var(--color-border)] bg-[var(--color-bg)] p-6 shadow-xl"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="mb-4 flex items-center justify-between">
          <h3 className="text-lg font-semibold text-[var(--color-text)]">Build Your Persona</h3>
          <button
            onClick={onClose}
            className="rounded p-1 text-[var(--color-text-secondary)] transition-colors hover:text-[var(--color-text)]"
          >
            <X size={18} />
          </button>
        </div>

        <p className="mb-4 text-sm text-[var(--color-text-secondary)]">
          We'll analyze your writing style from your posts and any description you provide to create a personalized voice profile.
        </p>

        <div className="mb-4">
          <label className="mb-1.5 block text-sm font-medium text-[var(--color-text)]">
            Describe yourself, your expertise, and your content style
          </label>
          <textarea
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            rows={4}
            placeholder="e.g. I'm a frontend developer who writes about React, TypeScript, and web performance. My style is practical and no-nonsense..."
            className="w-full rounded-[var(--radius-input)] border border-[var(--color-border)] bg-[var(--color-card)] px-3 py-2 text-sm text-[var(--color-text)] placeholder:text-[var(--color-text-secondary)]"
          />
        </div>

        <p className="mb-4 text-xs text-[var(--color-text-secondary)]">
          {postCount > 0
            ? `We found ${postCount} post${postCount > 1 ? 's' : ''} to analyze alongside your description.`
            : 'No posts found — persona will be built from your description alone.'}
        </p>

        {buildPersona.error && (
          <p className="mb-4 text-sm text-red-400">{buildPersona.error}</p>
        )}

        {buildPersona.isRunning && buildPersona.progress && (
          <div className="mb-4">
            <p className="mb-2 text-sm text-[var(--color-text-secondary)]">{buildPersona.progress.message}</p>
            <div className="h-2 w-full rounded-full bg-[var(--color-border)]">
              <div
                className="h-2 rounded-full bg-[var(--color-accent)] transition-all"
                style={{ width: `${buildPersona.progress.percentage}%` }}
              />
            </div>
          </div>
        )}

        <div className="flex justify-end gap-3">
          <button
            onClick={onClose}
            disabled={buildPersona.isRunning}
            className="rounded-[var(--radius-button)] border border-[var(--color-border)] px-4 py-2 text-sm text-[var(--color-text-secondary)] transition-colors hover:text-[var(--color-text)] disabled:opacity-50"
          >
            Cancel
          </button>
          <button
            onClick={handleBuild}
            disabled={!canBuild || buildPersona.isRunning}
            className="rounded-[var(--radius-button)] bg-[var(--color-accent)] px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-[var(--color-accent-hover)] disabled:opacity-50"
          >
            {buildPersona.isRunning ? 'Building...' : 'Build Persona'}
          </button>
        </div>
      </div>
    </div>
  )
}
