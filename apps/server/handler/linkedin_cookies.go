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

	userID := middleware.UserIDFromContext(r.Context())
	cookiePath := filepath.Join(config.DefaultConfigDir(), "linkitin_cookies.json")
	if data, err := os.ReadFile(cookiePath); err == nil {
		uc, _ := h.db.GetUserConfig(userID)
		uc.LinkitinCookiesJSON = string(data)
		h.db.SaveUserConfig(userID, uc)
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

	lc, err := linkedin.NewLinkitinClient()
	if err != nil {
		middleware.WriteJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "linkitin unavailable: " + err.Error(),
		})
		return
	}

	if err := lc.LoginWithCookies(r.Context(), req.LiAt, req.JSessionID); err != nil {
		middleware.WriteJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to login with cookies: " + err.Error(),
		})
		return
	}

	userID := middleware.UserIDFromContext(r.Context())
	cookiePath := filepath.Join(config.DefaultConfigDir(), "linkitin_cookies.json")
	if data, err := os.ReadFile(cookiePath); err == nil {
		uc, _ := h.db.GetUserConfig(userID)
		uc.LinkitinCookiesJSON = string(data)
		h.db.SaveUserConfig(userID, uc)
	}

	middleware.WriteJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

// Status checks whether linkitin cookies are available.
func (h *LinkedInCookiesHandler) Status(w http.ResponseWriter, r *http.Request) {
	cookiePath := filepath.Join(config.DefaultConfigDir(), "linkitin_cookies.json")
	_, err := os.Stat(cookiePath)
	middleware.WriteJSON(w, http.StatusOK, map[string]bool{
		"available": err == nil,
	})
}
