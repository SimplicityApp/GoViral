package service

import (
	"context"
	"fmt"

	"github.com/shuhao/goviral/apps/server/dto"
	"github.com/shuhao/goviral/internal/ai/claude"
	"github.com/shuhao/goviral/internal/ai/generator"
	"github.com/shuhao/goviral/internal/config"
	"github.com/shuhao/goviral/internal/db"
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
func (s *GenerateService) Generate(ctx context.Context, req dto.GenerateRequest, progress chan<- dto.ProgressEvent) ([]models.GeneratedContent, error) {
	defer close(progress)

	if s.cfg.Claude.APIKey == "" {
		return nil, fmt.Errorf("Claude API key not configured")
	}

	client := claude.NewClient(s.cfg.Claude.APIKey, s.cfg.Claude.Model)
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
	persona, err := s.db.GetPersona(targetPlatform)
	if err != nil {
		return nil, fmt.Errorf("getting persona: %w", err)
	}
	if persona == nil {
		return nil, fmt.Errorf("no persona found for platform %s; build persona first", targetPlatform)
	}

	total := len(req.TrendingPostIDs)
	var allContent []models.GeneratedContent

	for i, tpID := range req.TrendingPostIDs {
		tp, err := s.db.GetTrendingPostByID(tpID)
		if err != nil {
			return nil, fmt.Errorf("getting trending post %d: %w", tpID, err)
		}
		if tp == nil {
			return nil, fmt.Errorf("trending post %d not found", tpID)
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
			Niches:         s.cfg.Niches,
			Count:          count,
			MaxChars:       maxChars,
			ForceImage:     req.ForceImage,
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
				ImagePrompt:      r.ImagePrompt,
				IsRepost:         req.IsRepost,
			}
			if req.IsRepost {
				gc.PromptUsed = "repost-commentary"
				gc.QuoteTweetID = tp.PlatformPostID
			}
			id, err := s.db.InsertGeneratedContent(&gc)
			if err != nil {
				return nil, fmt.Errorf("saving generated content: %w", err)
			}
			gc.ID = id
			allContent = append(allContent, gc)
		}
	}

	progress <- dto.ProgressEvent{
		Type:       "complete",
		Message:    fmt.Sprintf("Generated %d content items", len(allContent)),
		Percentage: 100,
	}

	return allContent, nil
}
