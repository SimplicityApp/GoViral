import { useState } from 'react'
import { MaskedInput } from '@/components/shared/MaskedInput'
import { useUpdateConfigMutation } from '@/hooks/useConfig'
import type { AppConfig } from '@/hooks/useConfig'
import { toast } from 'sonner'
import { Check } from 'lucide-react'

export function AIKeysStep({ config }: { config: AppConfig | undefined }) {
  const [claudeKey, setClaudeKey] = useState(config?.claude.user_api_key || '')
  const [geminiKey, setGeminiKey] = useState(config?.gemini.user_api_key || '')
  const [saved, setSaved] = useState(false)
  const updateConfig = useUpdateConfigMutation()

  const handleSave = () => {
    updateConfig.mutate(
      {
        claude: { api_key: claudeKey },
        gemini: { api_key: geminiKey },
      },
      {
        onSuccess: () => {
          toast.success('AI keys saved')
          setSaved(true)
        },
        onError: () => toast.error('Failed to save AI keys'),
      },
    )
  }

  return (
    <div className="flex flex-col items-center">
      <h2 className="mb-2 text-xl font-bold text-[var(--color-text)]">
        AI API Keys
      </h2>
      <p className="mb-6 text-sm text-[var(--color-text-secondary)]">
        GoViral uses Claude for persona analysis and content generation (required),
        and Gemini for image generation (optional).
      </p>
      <div className="flex w-full max-w-md flex-col gap-4">
        <MaskedInput
          label="Claude API Key (required)"
          value={claudeKey}
          onChange={(v) => { setClaudeKey(v); setSaved(false) }}
          placeholder="sk-ant-..."
        />
        <MaskedInput
          label="Gemini API Key (optional — image generation)"
          value={geminiKey}
          onChange={(v) => { setGeminiKey(v); setSaved(false) }}
          placeholder="AI..."
        />
        <button
          onClick={handleSave}
          disabled={updateConfig.isPending}
          className="flex items-center justify-center gap-2 rounded-[var(--radius-button)] bg-[var(--color-accent)] px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-[var(--color-accent-hover)] disabled:opacity-50"
        >
          {saved ? (
            <>
              <Check size={16} /> Saved
            </>
          ) : updateConfig.isPending ? (
            'Saving...'
          ) : (
            'Save Keys'
          )}
        </button>
      </div>
    </div>
  )
}
