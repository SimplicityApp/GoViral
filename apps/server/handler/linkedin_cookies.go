package handler

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/shuhao/goviral/apps/server/middleware"
	"github.com/shuhao/goviral/internal/config"
	"github.com/shuhao/goviral/internal/db"
	"github.com/shuhao/goviral/internal/platform/linkedin"
)

// LinkedInCookiesHandler manages LinkedIn linkitin cookie authentication.
type LinkedInCookiesHandler struct {
	cfg *config.Config
	db  *db.DB
}

// NewLinkedInCookiesHandler creates a new LinkedInCookiesHandler.
func NewLinkedInCookiesHandler(cfg *config.Config, database *db.DB) *LinkedInCookiesHandler {
	return &LinkedInCookiesHandler{cfg: cfg, db: database}
}

// ExtractCookies extracts LinkedIn session cookies from Chrome.
func (h *LinkedInCookiesHandler) ExtractCookies(w http.ResponseWriter, r *http.Request) {
	lc, err := linkedin.NewLinkitinClient()
	if err != nil {
		middleware.WriteJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "linkitin unavailable: " + err.Error(),
		})
		return
	}

	if err := lc.ExtractCookies(r.Context()); err != nil {
		middleware.WriteJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to extract cookies: " + err.Error(),
		})
		return
	}

	// Read extracted cookies from the global file, save to per-user DB, then clean up.
	userID := middleware.UserIDFromContext(r.Context())
	cookiePath := filepath.Join(config.DefaultConfigDir(), "linkitin_cookies.json")
	if data, err := os.ReadFile(cookiePath); err == nil {
		uc, _ := h.db.GetUserConfig(userID)
		uc.LinkitinCookiesJSON = string(data)
		h.db.SaveUserConfig(userID, uc)
		os.Remove(cookiePath)
	}

	middleware.WriteJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

type loginCookiesRequest struct {
	LiAt       string `json:"li_at"`
	JSessionID string `json:"jsessionid"`
}

// LoginCookies authenticates with manually provided LinkedIn cookies.
// Saves directly to per-user DB without writing to the global cookie file.
func (h *LinkedInCookiesHandler) LoginCookies(w http.ResponseWriter, r *http.Request) {
	var req loginCookiesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}
	if req.LiAt == "" || req.JSessionID == "" {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{
			"error": "li_at and jsessionid are required",
		})
		return
	}

	userID := middleware.UserIDFromContext(r.Context())

	// Build cookie JSON directly and save to DB — no global file needed.
	cookies := map[string]string{
		"li_at":      req.LiAt,
		"JSESSIONID": req.JSessionID,
	}
	data, err := json.MarshalIndent(cookies, "", "  ")
	if err != nil {
		middleware.WriteJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to marshal cookies: " + err.Error(),
		})
		return
	}

	uc, _ := h.db.GetUserConfig(userID)
	uc.LinkitinCookiesJSON = string(data)
	if err := h.db.SaveUserConfig(userID, uc); err != nil {
		middleware.WriteJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to save cookies: " + err.Error(),
		})
		return
	}

	middleware.WriteJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

// Status checks whether linkitin cookies are available for the current user.
func (h *LinkedInCookiesHandler) Status(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	uc, _ := h.db.GetUserConfig(userID)
	middleware.WriteJSON(w, http.StatusOK, map[string]bool{
		"available": uc != nil && uc.LinkitinCookiesJSON != "",
	})
}
