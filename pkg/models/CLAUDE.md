# Shared Models — `pkg/models/`

## Purpose
This is the **contract layer** — all packages depend on these types. Defines interfaces that platform clients and AI services must implement, plus shared data models.

## Rules
- No business logic
- No external dependencies beyond stdlib + `github.com/google/uuid`
- Changes here affect all consumers — modify with care

## Platform Interfaces (`platform.go`)
| Interface | Methods | Implementors |
|-----------|---------|-------------|
| `PlatformClient` | `FetchMyPosts()`, `FetchTrendingPosts()` | x, linkedin (primary + fallback) |
| `PlatformPoster` | `PostTweet()`, `PostReply()` | x |
| `MediaPoster` | extends PlatformPoster + `UploadMedia()`, `PostTweetWithMedia()`, `PostReplyWithMedia()` | x |
| `PlatformScheduler` | `ScheduleTweet()` | x |
| `QuotePoster` | `PostQuoteTweet()` | x |
| `QuoteScheduler` | `ScheduleQuoteTweet()` | x |
| `LinkedInPoster` | `CreatePost()`, `UploadImage()`, `CreatePostWithImage()`, `CreateScheduledPost()`, ... | linkedin |
| `LinkedInReposter` | `Repost()` | linkedin |
| `LinkedInCommenter` | `CreateComment()` (with threadURN) | linkedin |
| `YouTubePoster` | `UploadVideo()`, `UploadVideoWithThumbnail()` | youtube |
| `TikTokPoster` | `UploadVideo()`, `ScheduleVideo()` | tiktok |
| `VideoFetcher` | `FetchTrendingVideos()` | youtube, tiktok |
| `GitHubClient` | `GetRepo()`, `ListCommits()`, `GetCommit()`, `ListUserRepos()` | github |
| `CodeImageRenderer` | `RenderDiff()`, `Close()` | codeimg |

## AI Interfaces (`ai.go`)
| Interface | Methods |
|-----------|---------|
| `PersonaAnalyzer` | `BuildProfile(ctx, posts, platform)` |
| `ContentGenerator` | `Generate()`, `GenerateComment()`, `GenerateRepoPost()`, `ClassifyPost()`, `ClassifyPosts()`, `SelectActions()`, `DecideImage()`, `GenerateImagePrompt()` |

Key request/response types:
- `GenerateRequest` — trending post + persona + target platform + count + style direction
- `GenerateResult` — content + viral mechanic + confidence score + code snippet
- `ClassifyResult` — decision (rewrite/repost) + reasoning + confidence
- `ActionSelectResult` — action (post/repost/comment) + reasoning
- `ImageDecision` — suggest image (bool) + reasoning

## Data Models
| File | Types | Purpose |
|------|-------|---------|
| `post.go` | `Post` | User's own posts (metrics: likes, reposts, comments, impressions) |
| `trending.go` | `TrendingPost`, `MediaAttachment` | Discovered trending posts with media, video fields, thread URN |
| `persona.go` | `Persona`, `PersonaProfile` | Writing style profile (tone, themes, quirks, emoji usage, etc.) |
| `generated.go` | `GeneratedContent`, `ScheduledPost` | AI-generated variations with status tracking, image/video paths |
| `daemon.go` | `DaemonBatch`, `DaemonIntent`, `AutoPublishResult` | Autopilot batch state machine and approval intents |
| `github.go` | `GitHubRepo`, `GitHubCommit`, `GitHubFileChange`, `RepoLink`, `RenderOptions` | Repository and commit data |
| `fileclass.go` | `IsLockfile()`, `IsSourceFile()`, `IsConfigFile()` | File classification utilities |
| `period.go` | `PeriodCutoff()` | Time period conversion (day/week/month -> duration) |
