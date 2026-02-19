package handler

import (
	"net/http"

	"github.com/shuhao/goviral/apps/server/dto"
	"github.com/shuhao/goviral/apps/server/middleware"
	"github.com/shuhao/goviral/apps/server/service"
)

// PersonaHandler handles requests for persona profiles.
type PersonaHandler struct {
	svc *service.PersonaService
}

// NewPersonaHandler creates a new PersonaHandler.
func NewPersonaHandler(svc *service.PersonaService) *PersonaHandler {
	return &PersonaHandler{svc: svc}
}

// Get returns the persona for the queried platform.
func (h *PersonaHandler) Get(w http.ResponseWriter, r *http.Request) {
	platform := r.URL.Query().Get("platform")
	if platform == "" {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusBadRequest, dto.ErrCodeValidation, "platform query parameter is required", reqID)
		return
	}

	persona, err := h.svc.Get(platform)
	if err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusInternalServerError, dto.ErrCodeInternal, "failed to get persona", reqID)
		return
	}
	if persona == nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusNotFound, dto.ErrCodeNotFound, "persona not found for platform", reqID)
		return
	}

	// Convert PersonaProfile struct to map for JSON response
	profile := map[string]interface{}{
		"writing_tone":        persona.Profile.WritingTone,
		"typical_length":      persona.Profile.TypicalLength,
		"common_themes":       persona.Profile.CommonThemes,
		"vocabulary_level":    persona.Profile.VocabularyLevel,
		"engagement_patterns": persona.Profile.EngagementPatterns,
		"structural_patterns": persona.Profile.StructuralPatterns,
		"emoji_usage":         persona.Profile.EmojiUsage,
		"hashtag_usage":       persona.Profile.HashtagUsage,
		"call_to_action_style": persona.Profile.CallToActionStyle,
		"unique_quirks":       persona.Profile.UniqueQuirks,
		"voice_summary":       persona.Profile.VoiceSummary,
	}

	middleware.WriteJSON(w, http.StatusOK, dto.PersonaResponse{
		ID:        persona.ID,
		Platform:  persona.Platform,
		Profile:   profile,
		CreatedAt: persona.CreatedAt,
		UpdatedAt: persona.UpdatedAt,
	})
}
