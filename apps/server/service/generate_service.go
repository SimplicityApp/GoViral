package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/shuhao/goviral/apps/server/dto"
	"github.com/shuhao/goviral/internal/ai/claude"
	"github.com/shuhao/goviral/internal/ai/gemini"
	"github.com/shuhao/goviral/internal/ai/generator"
	"github.com/shuhao/goviral/internal/config"
	"github.com/shuhao/goviral/internal/db"
	"github.com/shuhao/goviral/internal/ratelimit"
	"github.com/shuhao/goviral/pkg/models"
)

// GenerateService handles content generation using AI.
type GenerateService struct {
	db  *db.DB
	cfg *config.Config
}

// NewGenerateService creates a new GenerateService.
func NewGenerateService(database *db.DB, cfg *config.Config) *GenerateService {
	return &GenerateService{db: database, cfg: cfg}
}

// Generate creates content variations for the given trending posts.
func (s *GenerateService) Generate(ctx context.Context, userID string, req dto.GenerateRequest, progress chan<- dto.ProgressEvent) ([]models.GeneratedContent, error) {
	defer close(progress)

	uc, _ := s.db.GetUserConfig(userID)
	claudeCfg := uc.ResolvedClaudeConfig(*s.cfg)
	if claudeCfg.APIKey == "" {
		return nil, fmt.Errorf("Claude API key not configured")
	}
	if !uc.UsingOwnClaudeKey() {
		if err := ratelimit.CheckAIRateLimit(s.db, userID, "claude", s.cfg.Claude.DailyLimit); err != nil {
			return nil, err
		}
	}

	client := claude.NewClient(claudeCfg.APIKey, claudeCfg.Model)
	gen := generator.NewGenerator(client)

	targetPlatform := req.TargetPlatform
	if targetPlatform == "" {
		targetPlatform = "x"
	}
	count := req.Count
	if count <= 0 {
		count = 3
	}

	maxChars := req.MaxChars
	if req.IsRepost && maxChars == 0 {
		maxChars = 200
	}
	if targetPlatform == "linkedin" && maxChars == 0 {
		maxChars = 2000
	}

	// Get persona for target platform
	persona, err := s.db.GetPersona(userID, targetPlatform)
	if err != nil {
		return nil, fmt.Errorf("getting persona: %w", err)
	}
	if persona == nil {
		persona = &models.Persona{
			Platform: targetPlatform,
			Profile:  models.DefaultPersonaProfile(targetPlatform),
		}
	}

	total := len(req.TrendingPostIDs)
	var allContent []models.GeneratedContent

	for i, tpID := range req.TrendingPostIDs {
		tp, err := s.db.GetTrendingPostByID("",tpID)
		if err != nil {
			return nil, fmt.Errorf("getting trending post %d: %w", tpID, err)
		}
		if tp == nil {
			return nil, fmt.Errorf("trending post %d not found", tpID)
		}
		if tp.Platform != targetPlatform {
			return nil, fmt.Errorf("cannot generate %s content from %s post (trending post %d)", targetPlatform, tp.Platform, tp.ID)
		}

		progress <- dto.ProgressEvent{
			Type:       "progress",
			Message:    fmt.Sprintf("Generating variations for post %d/%d", i+1, total),
			Percentage: (i * 100) / total,
		}

		results, err := gen.Generate(ctx, models.GenerateRequest{
			TrendingPost:   *tp,
			Persona:        *persona,
			TargetPlatform: targetPlatform,
			Niches:         uc.MergedNiches(*s.cfg),
			Count:          count,
			MaxChars:       maxChars,
			IsRepost:       req.IsRepost,
		})
		if err != nil {
			return nil, fmt.Errorf("generating content for post %d: %w", tpID, err)
		}

		for _, r := range results {
			gc := models.GeneratedContent{
				SourceTrendingID: tp.ID,
				TargetPlatform:   targetPlatform,
				OriginalContent:  tp.Content,
				GeneratedContent: r.Content,
				PersonaID:        persona.ID,
				Status:           "draft",
				IsRepost:         req.IsRepost,
			}
			if req.IsRepost {
				gc.PromptUsed = fmt.Sprintf("repost-%s", targetPlatform)
				gc.QuoteTweetID = tp.PlatformPostID
			} else {
				gc.PromptUsed = fmt.Sprintf("rewrite-%s", targetPlatform)
			}

			// Image pipeline: decide whether to include an image
			var imagePrompt string
			if req.ForceImage {
				// Skip decision, always generate image prompt
				progress <- dto.ProgressEvent{
					Type:    "progress",
					Message: "Generating image prompt (forced)...",
				}
				imgPrompt, err := gen.GenerateImagePrompt(ctx, r.Content, targetPlatform)
				if err != nil {
					slog.Error("generating image prompt", "error", err)
				} else {
					imagePrompt = imgPrompt
				}
			} else {
				// Ask Claude whether an image is appropriate
				decision, err := gen.DecideImage(ctx, r.Content, targetPlatform)
				if err != nil {
					slog.Error("deciding image", "error", err)
				} else if decision.SuggestImage {
					progress <- dto.ProgressEvent{
						Type:    "progress",
						Message: "Generating image prompt...",
					}
					imgPrompt, err := gen.GenerateImagePrompt(ctx, r.Content, targetPlatform)
					if err != nil {
						slog.Error("generating image prompt", "error", err)
					} else {
						imagePrompt = imgPrompt
					}
				}
			}
			gc.ImagePrompt = imagePrompt

			// Generate actual image via Gemini if we have a prompt
			if imagePrompt != "" {
				geminiCfg := uc.ResolvedGeminiConfig(*s.cfg)
				canGenerate := geminiCfg.APIKey != ""
				if !canGenerate {
					slog.Warn("image prompt generated but no Gemini API key configured, skipping image generation")
					progress <- dto.ProgressEvent{
						Type:    "warning",
						Message: "Gemini API key not configured — skipping image generation",
					}
				} else if !uc.UsingOwnGeminiKey() {
					if rlErr := ratelimit.CheckAIRateLimit(s.db, userID, "gemini", s.cfg.Gemini.DailyLimit); rlErr != nil {
						slog.Warn("gemini rate limit reached, skipping image generation", "error", rlErr)
						progress <- dto.ProgressEvent{
							Type:    "warning",
							Message: fmt.Sprintf("Gemini daily limit reached — skipping image generation: %v", rlErr),
						}
						canGenerate = false
					}
				}
				if canGenerate {
					progress <- dto.ProgressEvent{
						Type:    "progress",
						Message: "Generating image via Gemini...",
					}
					geminiClient := gemini.NewClient(geminiCfg.APIKey, geminiCfg.Model)
					img, err := geminiClient.GenerateImage(ctx, imagePrompt)
					if err != nil {
						slog.Error("generating image via Gemini", "error", err)
						progress <- dto.ProgressEvent{
							Type:    "warning",
							Message: fmt.Sprintf("Gemini image generation failed: %v", err),
						}
					} else {
						name := fmt.Sprintf("gen_%d_%d_%d", tp.ID, i+1, time.Now().Unix())
						path, err := gemini.SaveImage(img, name)
						if err != nil {
							slog.Error("saving generated image", "error", err)
							progress <- dto.ProgressEvent{
								Type:    "warning",
								Message: fmt.Sprintf("Failed to save generated image: %v", err),
							}
						} else {
							gc.ImagePath = path
							progress <- dto.ProgressEvent{
								Type:    "progress",
								Message: fmt.Sprintf("Image generated and saved: %s", path),
							}
							if !uc.UsingOwnGeminiKey() {
								ratelimit.RecordAIUsage(s.db, userID, "gemini")
							}
						}
					}
				}
			}

			id, err := s.db.InsertGeneratedContent(userID, &gc)
			if err != nil {
				return nil, fmt.Errorf("saving generated content: %w", err)
			}
			gc.ID = id
			allContent = append(allContent, gc)
		}
	}

	if !uc.UsingOwnClaudeKey() {
		ratelimit.RecordAIUsage(s.db, userID, "claude")
	}

	progress <- dto.ProgressEvent{
		Type:       "complete",
		Message:    fmt.Sprintf("Generated %d content items", len(allContent)),
		Percentage: 100,
	}

	return allContent, nil
}

// GenerateComment creates comment variations for a trending post.
func (s *GenerateService) GenerateComment(ctx context.Context, userID string, trendingPostID int64, platform string, count int) ([]models.GeneratedContent, error) {
	uc, _ := s.db.GetUserConfig(userID)
	claudeCfg := uc.ResolvedClaudeConfig(*s.cfg)
	if claudeCfg.APIKey == "" {
		return nil, fmt.Errorf("Claude API key not configured")
	}
	if !uc.UsingOwnClaudeKey() {
		if err := ratelimit.CheckAIRateLimit(s.db, userID, "claude", s.cfg.Claude.DailyLimit); err != nil {
			return nil, err
		}
	}

	if platform == "" {
		platform = "linkedin"
	}
	if count <= 0 {
		count = 3
	}

	client := claude.NewClient(claudeCfg.APIKey, claudeCfg.Model)
	gen := generator.NewGenerator(client)

	tp, err := s.db.GetTrendingPostByID("",trendingPostID)
	if err != nil {
		return nil, fmt.Errorf("getting trending post %d: %w", trendingPostID, err)
	}
	if tp == nil {
		return nil, fmt.Errorf("trending post %d not found", trendingPostID)
	}
	if tp.Platform != platform {
		return nil, fmt.Errorf("cannot create %s comment on %s post (trending post %d)", platform, tp.Platform, trendingPostID)
	}

	persona, err := s.db.GetPersona(userID, platform)
	if err != nil {
		return nil, fmt.Errorf("getting persona: %w", err)
	}
	if persona == nil {
		persona = &models.Persona{
			Platform: platform,
			Profile:  models.DefaultPersonaProfile(platform),
		}
	}

	results, err := gen.GenerateComment(ctx, models.GenerateCommentRequest{
		TrendingPost:   *tp,
		Persona:        *persona,
		TargetPlatform: platform,
		Count:          count,
	})
	if err != nil {
		return nil, fmt.Errorf("generating comments for post %d: %w", trendingPostID, err)
	}

	var allContent []models.GeneratedContent
	for _, r := range results {
		gc := models.GeneratedContent{
			SourceTrendingID: tp.ID,
			TargetPlatform:   platform,
			OriginalContent:  tp.Content,
			GeneratedContent: r.Content,
			PersonaID:        persona.ID,
			PromptUsed:       fmt.Sprintf("comment-%s", platform),
			Status:           "draft",
			IsComment:        true,
			QuoteTweetID:     tp.PlatformPostID,
		}

		id, err := s.db.InsertGeneratedContent(userID, &gc)
		if err != nil {
			return nil, fmt.Errorf("saving generated comment: %w", err)
		}
		gc.ID = id
		allContent = append(allContent, gc)
	}

	if !uc.UsingOwnClaudeKey() {
		ratelimit.RecordAIUsage(s.db, userID, "claude")
	}

	return allContent, nil
}
