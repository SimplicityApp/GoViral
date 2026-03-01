import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiClient } from '@/lib/api'
import type { GeneratedContent } from '@/lib/types'
import { Video, Upload, Youtube, Music } from 'lucide-react'
import { toast } from 'sonner'

export function VideoPublish() {
  const [selectedPlatform, setSelectedPlatform] = useState<'youtube' | 'tiktok'>('youtube')
  const queryClient = useQueryClient()

  // Fetch content that has video paths
  const { data: contents = [] } = useQuery<GeneratedContent[]>({
    queryKey: ['history', '', ''],
    queryFn: () => apiClient.get('/history'),
  })

  const videoContents = contents.filter(c => c.video_path && c.status !== 'posted')

  const publishMutation = useMutation({
    mutationFn: (contentId: number) =>
      apiClient.post(`/${selectedPlatform}/upload`, { content_id: contentId }),
    onSuccess: () => {
      toast.success(`Video published to ${selectedPlatform === 'youtube' ? 'YouTube Shorts' : 'TikTok'}!`)
      queryClient.invalidateQueries({ queryKey: ['history'] })
    },
    onError: (err: Error) => {
      toast.error(`Publish failed: ${err.message}`)
    },
  })

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-[var(--color-text)]">Video Publish</h1>
        <p className="text-sm text-[var(--color-text-secondary)]">
          Upload and publish videos to YouTube Shorts and TikTok
        </p>
      </div>

      {/* Platform selector */}
      <div className="flex gap-3">
        <button
          onClick={() => setSelectedPlatform('youtube')}
          className={`flex items-center gap-2 rounded-lg px-4 py-2.5 text-sm font-medium transition-colors ${
            selectedPlatform === 'youtube'
              ? 'bg-red-500/10 text-red-500 ring-1 ring-red-500/30'
              : 'bg-[var(--color-card)] text-[var(--color-text-secondary)] hover:text-[var(--color-text)]'
          }`}
        >
          <Youtube size={18} />
          YouTube Shorts
        </button>
        <button
          onClick={() => setSelectedPlatform('tiktok')}
          className={`flex items-center gap-2 rounded-lg px-4 py-2.5 text-sm font-medium transition-colors ${
            selectedPlatform === 'tiktok'
              ? 'bg-[#00f2ea]/10 text-[#00f2ea] ring-1 ring-[#00f2ea]/30'
              : 'bg-[var(--color-card)] text-[var(--color-text-secondary)] hover:text-[var(--color-text)]'
          }`}
        >
          <Music size={18} />
          TikTok
        </button>
      </div>

      {/* Video content list */}
      {videoContents.length === 0 ? (
        <div className="rounded-xl border border-[var(--color-border)] bg-[var(--color-card)] p-12 text-center">
          <Video size={48} className="mx-auto mb-4 text-[var(--color-text-secondary)]" />
          <h3 className="text-lg font-medium text-[var(--color-text)]">No video content available</h3>
          <p className="mt-1 text-sm text-[var(--color-text-secondary)]">
            Generate content with video paths to publish here.
          </p>
        </div>
      ) : (
        <div className="space-y-4">
          {videoContents.map((content) => (
            <div
              key={content.id}
              className="rounded-xl border border-[var(--color-border)] bg-[var(--color-card)] p-4"
            >
              <div className="flex items-start justify-between gap-4">
                <div className="flex-1 space-y-2">
                  <div className="flex items-center gap-2">
                    <span className="rounded bg-[var(--color-accent)]/10 px-2 py-0.5 text-xs font-medium text-[var(--color-accent)]">
                      {content.status}
                    </span>
                    {content.video_title && (
                      <span className="text-sm font-medium text-[var(--color-text)]">
                        {content.video_title}
                      </span>
                    )}
                  </div>
                  <p className="text-sm text-[var(--color-text-secondary)] line-clamp-3">
                    {content.generated_content}
                  </p>
                  <div className="flex items-center gap-3 text-xs text-[var(--color-text-secondary)]">
                    <span>Video: {content.video_path}</span>
                    {content.video_duration > 0 && (
                      <span>{content.video_duration}s</span>
                    )}
                  </div>
                </div>
                <button
                  onClick={() => publishMutation.mutate(content.id)}
                  disabled={publishMutation.isPending}
                  className="flex items-center gap-2 rounded-lg bg-[var(--color-accent)] px-4 py-2 text-sm font-medium text-white hover:opacity-90 disabled:opacity-50"
                >
                  <Upload size={16} />
                  {publishMutation.isPending ? 'Publishing...' : 'Publish'}
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
