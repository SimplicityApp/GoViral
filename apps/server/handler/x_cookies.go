package handler

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/shuhao/goviral/apps/server/middleware"
	"github.com/shuhao/goviral/internal/config"
	"github.com/shuhao/goviral/internal/db"
	x "github.com/shuhao/goviral/internal/platform/x"
)

// XCookiesHandler manages X twikit cookie authentication.
type XCookiesHandler struct {
	cfg *config.Config
	db  *db.DB
}

// NewXCookiesHandler creates a new XCookiesHandler.
func NewXCookiesHandler(cfg *config.Config, database *db.DB) *XCookiesHandler {
	return &XCookiesHandler{cfg: cfg, db: database}
}

// ExtractCookies extracts X session cookies from Chrome.
func (h *XCookiesHandler) ExtractCookies(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	username := h.cfg.X.Username
	if uc, err := h.db.GetUserConfig(userID); err == nil && uc.XUsername != "" {
		username = uc.XUsername
	}

	tc, err := x.NewTwikitClient(username)
	if err != nil {
		middleware.WriteJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "twikit unavailable: " + err.Error(),
		})
		return
	}

	if err := tc.ExtractCookies(r.Context()); err != nil {
		middleware.WriteJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to extract cookies: " + err.Error(),
		})
		return
	}

	cookiePath := filepath.Join(config.DefaultConfigDir(), "twikit_cookies.json")
	if data, err := os.ReadFile(cookiePath); err == nil {
		uc, _ := h.db.GetUserConfig(userID)
		uc.TwikitCookiesJSON = string(data)
		h.db.SaveUserConfig(userID, uc)
	}

	middleware.WriteJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

type xLoginCookiesRequest struct {
	AuthToken string `json:"auth_token"`
	Ct0       string `json:"ct0"`
}

// LoginCookies authenticates with manually provided X cookies.
func (h *XCookiesHandler) LoginCookies(w http.ResponseWriter, r *http.Request) {
	var req xLoginCookiesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid request body",
		})
		return
	}
	if req.AuthToken == "" || req.Ct0 == "" {
		middleware.WriteJSON(w, http.StatusBadRequest, map[string]string{
			"error": "auth_token and ct0 are required",
		})
		return
	}

	userID := middleware.UserIDFromContext(r.Context())
	username := h.cfg.X.Username
	if uc, err := h.db.GetUserConfig(userID); err == nil && uc.XUsername != "" {
		username = uc.XUsername
	}

	tc, err := x.NewTwikitClient(username)
	if err != nil {
		middleware.WriteJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "twikit unavailable: " + err.Error(),
		})
		return
	}

	if err := tc.LoginWithCookies(r.Context(), req.AuthToken, req.Ct0); err != nil {
		middleware.WriteJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "failed to save cookies: " + err.Error(),
		})
		return
	}

	cookiePath := filepath.Join(config.DefaultConfigDir(), "twikit_cookies.json")
	if data, err := os.ReadFile(cookiePath); err == nil {
		uc, _ := h.db.GetUserConfig(userID)
		uc.TwikitCookiesJSON = string(data)
		h.db.SaveUserConfig(userID, uc)
	}

	middleware.WriteJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

// Status checks whether twikit cookies are available.
func (h *XCookiesHandler) Status(w http.ResponseWriter, r *http.Request) {
	cookiePath := filepath.Join(config.DefaultConfigDir(), "twikit_cookies.json")
	_, err := os.Stat(cookiePath)
	middleware.WriteJSON(w, http.StatusOK, map[string]bool{
		"available": err == nil,
	})
}
