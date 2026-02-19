export interface Post {
  id: number
  platform: string
  platform_post_id: string
  content: string
  likes: number
  reposts: number
  comments: number
  impressions: number
  posted_at: string
  fetched_at: string
}

export interface TrendingPost {
  id: number
  platform: string
  platform_post_id: string
  author_username: string
  author_name: string
  content: string
  likes: number
  reposts: number
  comments: number
  impressions: number
  niche_tags: string[]
  media: MediaAttachment[]
  posted_at: string
  fetched_at: string
}

export interface MediaAttachment {
  type: string
  url: string
  preview_url: string
  alt_text: string
}

export interface PersonaProfile {
  writing_tone: string
  typical_length: string
  common_themes: string[]
  vocabulary_level: string
  engagement_patterns: string
  structural_patterns: string[]
  emoji_usage: string
  hashtag_usage: string
  call_to_action_style: string
  unique_quirks: string[]
  voice_summary: string
}

export interface Persona {
  id: number
  platform: string
  profile: PersonaProfile
  created_at: string
  updated_at: string
}

export interface GeneratedContent {
  id: number
  source_trending_id: number
  target_platform: string
  original_content: string
  generated_content: string
  persona_id: number
  prompt_used: string
  created_at: string
  status: 'draft' | 'approved' | 'posted'
  platform_post_ids: string
  posted_at: string | null
  image_prompt: string
  image_path: string
  is_repost: boolean
  quote_tweet_id: string
}

export interface ScheduledPost {
  id: number
  generated_content_id: number
  scheduled_at: string
  status: 'pending' | 'scheduled' | 'posted' | 'failed'
  error_message: string
  created_at: string
  content_preview: string
  target_platform: string
  platform_schedule_id: string
}

export interface ProgressEvent {
  type: 'progress' | 'complete' | 'error'
  message: string
  percentage: number
  data?: unknown
}

export interface DaemonBatch {
  id: number
  platform: string
  status: 'pending' | 'notified' | 'awaiting_reply' | 'approved' | 'rejected' | 'posted' | 'scheduled' | 'archived' | 'failed'
  content_ids: number[]
  trending_ids: number[]
  telegram_message_id: number
  approval_source: string
  reply_text: string
  error_message: string
  created_at: string
  updated_at: string
  notified_at: string | null
  resolved_at: string | null
  contents?: GeneratedContent[]
}

export interface DaemonStatus {
  running: boolean
  platforms: Record<string, PlatformDaemonStatus>
}

export interface PlatformDaemonStatus {
  schedule: string
  next_run: string | null
  last_run: string | null
  last_batch_id: number | null
}

export interface DaemonConfig {
  daemon: {
    enabled: boolean
    schedules: Record<string, string>
    max_per_batch: number
    auto_skip_after: string
    trending_limit: number
    min_likes: number
    period: string
  }
  telegram: {
    bot_token: string
    chat_id: number
    webhook_url: string
    connected: boolean
  }
}
