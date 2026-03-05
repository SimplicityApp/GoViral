package handler

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/shuhao/goviral/apps/server/dto"
	"github.com/shuhao/goviral/apps/server/middleware"
	"github.com/shuhao/goviral/internal/auth"
	"github.com/shuhao/goviral/internal/config"
	"github.com/shuhao/goviral/internal/db"
)

const (
	xAuthURL        = "https://x.com/i/oauth2/authorize"
	xScopes         = "tweet.read tweet.write users.read offline.access"
	linkedinAuthURL = "https://www.linkedin.com/oauth/v2/authorization"
	linkedinScopes  = "openid profile w_member_social"
	githubAuthURL   = "https://github.com/login/oauth/authorize"
	githubScopes    = "repo"
)

// AuthHandler handles OAuth flow requests.
type AuthHandler struct {
	cfg *config.Config
	db  *db.DB

	mu           sync.Mutex
	pendingAuths map[string]*pendingAuth // keyed by unique key
}

type pendingAuth struct {
	platform     string
	authURL      string
	codeVerifier string // X only (PKCE)
	redirectURI  string
	status       string // "pending", "completed", "failed"
	error        string
	startedAt    time.Time
	userID       string
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(cfg *config.Config, database *db.DB) *AuthHandler {
	return &AuthHandler{
		cfg:          cfg,
		db:           database,
		pendingAuths: make(map[string]*pendingAuth),
	}
}

// generateKey returns a random hex string for use as a pending auth key.
func generateKey() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// fallback to timestamp-based key
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// Start initiates an OAuth flow for the specified platform.
func (h *AuthHandler) Start(w http.ResponseWriter, r *http.Request) {
	platform := chi.URLParam(r, "platform")
	if platform != "x" && platform != "linkedin" && platform != "github" {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusBadRequest, dto.ErrCodeValidation, "platform must be 'x', 'linkedin', or 'github'", reqID)
		return
	}

	key := generateKey()
	state := platform + ":" + key
	redirectURI := h.cfg.Server.BaseURL + "/api/v1/auth/callback"

	var authURL string
	var codeVerifier string

	switch platform {
	case "x":
		if h.cfg.X.ClientID == "" {
			reqID := middleware.RequestIDFromContext(r.Context())
			middleware.WriteError(w, http.StatusBadRequest, dto.ErrCodeValidation, "X client_id not configured", reqID)
			return
		}

		var err error
		codeVerifier, err = auth.GenerateCodeVerifier()
		if err != nil {
			reqID := middleware.RequestIDFromContext(r.Context())
			middleware.WriteError(w, http.StatusInternalServerError, dto.ErrCodeInternal, "failed to generate PKCE verifier", reqID)
			return
		}
		codeChallenge := auth.GenerateCodeChallenge(codeVerifier)

		authURL = fmt.Sprintf("%s?response_type=code&client_id=%s&redirect_uri=%s&scope=%s&code_challenge=%s&code_challenge_method=S256&state=%s",
			xAuthURL,
			url.QueryEscape(h.cfg.X.ClientID),
			url.QueryEscape(redirectURI),
			url.QueryEscape(xScopes),
			url.QueryEscape(codeChallenge),
			url.QueryEscape(state),
		)

	case "linkedin":
		if h.cfg.LinkedIn.ClientID == "" {
			reqID := middleware.RequestIDFromContext(r.Context())
			middleware.WriteError(w, http.StatusBadRequest, dto.ErrCodeValidation, "LinkedIn client_id not configured", reqID)
			return
		}

		authURL = fmt.Sprintf("%s?response_type=code&client_id=%s&redirect_uri=%s&scope=%s&state=%s",
			linkedinAuthURL,
			url.QueryEscape(h.cfg.LinkedIn.ClientID),
			url.QueryEscape(redirectURI),
			url.QueryEscape(linkedinScopes),
			url.QueryEscape(state),
		)

	case "github":
		if h.cfg.GitHub.ClientID == "" {
			reqID := middleware.RequestIDFromContext(r.Context())
			middleware.WriteError(w, http.StatusBadRequest, dto.ErrCodeValidation, "GitHub client_id not configured", reqID)
			return
		}

		authURL = fmt.Sprintf("%s?client_id=%s&redirect_uri=%s&scope=%s&state=%s",
			githubAuthURL,
			url.QueryEscape(h.cfg.GitHub.ClientID),
			url.QueryEscape(redirectURI),
			url.QueryEscape(githubScopes),
			url.QueryEscape(state),
		)
	}

	userID := middleware.UserIDFromContext(r.Context())

	h.mu.Lock()
	h.pendingAuths[key] = &pendingAuth{
		platform:     platform,
		authURL:      authURL,
		codeVerifier: codeVerifier,
		redirectURI:  redirectURI,
		status:       "pending",
		startedAt:    time.Now(),
		userID:       userID,
	}
	h.mu.Unlock()

	middleware.WriteJSON(w, http.StatusOK, map[string]string{
		"auth_url": authURL,
		"key":      key,
		"status":   "pending",
	})
}

// Status returns the current status of an OAuth flow.
func (h *AuthHandler) Status(w http.ResponseWriter, r *http.Request) {
	platform := chi.URLParam(r, "platform")
	key := r.URL.Query().Get("key")

	h.mu.Lock()
	if key != "" {
		// Look up by specific key
		pa, ok := h.pendingAuths[key]
		h.mu.Unlock()

		if !ok || pa.platform != platform {
			middleware.WriteJSON(w, http.StatusOK, map[string]string{
				"status": "none",
			})
			return
		}

		resp := map[string]string{
			"status": pa.status,
		}
		if pa.error != "" {
			resp["error"] = pa.error
		}
		middleware.WriteJSON(w, http.StatusOK, resp)
		return
	}

	// Fallback: find the most recent pending auth for this platform
	var found *pendingAuth
	for _, pa := range h.pendingAuths {
		if pa.platform == platform {
			if found == nil || pa.startedAt.After(found.startedAt) {
				found = pa
			}
		}
	}
	h.mu.Unlock()

	if found == nil {
		middleware.WriteJSON(w, http.StatusOK, map[string]string{
			"status": "none",
		})
		return
	}

	resp := map[string]string{
		"status": found.status,
	}
	if found.error != "" {
		resp["error"] = found.error
	}
	middleware.WriteJSON(w, http.StatusOK, resp)
}

// Callback handles the OAuth callback from the provider.
// This route is unauthenticated since it's a browser redirect from the OAuth provider.
func (h *AuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	stateParam := r.URL.Query().Get("state")

	if code == "" || stateParam == "" {
		http.Error(w, "Missing code or state parameter", http.StatusBadRequest)
		return
	}

	// Parse state: "platform:key"
	parts := strings.SplitN(stateParam, ":", 2)
	if len(parts) != 2 {
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}
	platform := parts[0]
	key := parts[1]

	h.mu.Lock()
	pa, ok := h.pendingAuths[key]
	h.mu.Unlock()

	if !ok || pa.platform != platform {
		http.Error(w, "Unknown or expired authorization request", http.StatusBadRequest)
		return
	}

	redirectURI := pa.redirectURI

	switch platform {
	case "x":
		tokenResp, err := auth.ExchangeXCode(code, h.cfg.X.ClientID, h.cfg.X.ClientSecret, pa.codeVerifier, redirectURI)
		if err != nil {
			h.setAuthFailed(key, err.Error())
			h.writeCallbackHTML(w, false, "X authorization failed: "+err.Error())
			return
		}

		uc, _ := h.db.GetUserConfig(pa.userID)
		uc.XAccessToken = tokenResp.AccessToken
		uc.XRefreshToken = tokenResp.RefreshToken
		if tokenResp.ExpiresIn > 0 {
			uc.XTokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second).Format(time.RFC3339)
		}
		if err := h.db.SaveUserConfig(pa.userID, uc); err != nil {
			h.setAuthFailed(key, fmt.Sprintf("saving user config: %v", err))
			h.writeCallbackHTML(w, false, "Failed to save credentials")
			return
		}

	case "linkedin":
		token, err := auth.ExchangeLinkedInCode(code, h.cfg.LinkedIn.ClientID, h.cfg.LinkedIn.ClientSecret, redirectURI)
		if err != nil {
			h.setAuthFailed(key, err.Error())
			h.writeCallbackHTML(w, false, "LinkedIn authorization failed: "+err.Error())
			return
		}

		uc, _ := h.db.GetUserConfig(pa.userID)
		uc.LinkedInAccessToken = token
		if err := h.db.SaveUserConfig(pa.userID, uc); err != nil {
			h.setAuthFailed(key, fmt.Sprintf("saving user config: %v", err))
			h.writeCallbackHTML(w, false, "Failed to save credentials")
			return
		}

	case "github":
		token, err := auth.ExchangeGitHubCode(code, h.cfg.GitHub.ClientID, h.cfg.GitHub.ClientSecret, redirectURI)
		if err != nil {
			h.setAuthFailed(key, err.Error())
			h.writeCallbackHTML(w, false, "GitHub authorization failed: "+err.Error())
			return
		}

		uc, _ := h.db.GetUserConfig(pa.userID)
		uc.GitHubAccessToken = token
		if err := h.db.SaveUserConfig(pa.userID, uc); err != nil {
			h.setAuthFailed(key, fmt.Sprintf("saving user config: %v", err))
			h.writeCallbackHTML(w, false, "Failed to save credentials")
			return
		}
	}

	h.mu.Lock()
	pa.status = "completed"
	h.mu.Unlock()

	h.writeCallbackHTML(w, true, "")
}

func (h *AuthHandler) setAuthFailed(key, errMsg string) {
	h.mu.Lock()
	if pa, ok := h.pendingAuths[key]; ok {
		pa.status = "failed"
		pa.error = errMsg
	}
	h.mu.Unlock()
}

func (h *AuthHandler) writeCallbackHTML(w http.ResponseWriter, success bool, errMsg string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if success {
		fmt.Fprint(w, `<!DOCTYPE html>
<html><head><title>Authorization Complete</title>
<style>body{font-family:system-ui,sans-serif;display:flex;align-items:center;justify-content:center;min-height:100vh;margin:0;background:#f9fafb}
.card{text-align:center;padding:2rem;border-radius:12px;background:#fff;box-shadow:0 1px 3px rgba(0,0,0,.1)}
.check{color:#22c55e;font-size:3rem;margin-bottom:1rem}
h1{margin:0 0 .5rem;font-size:1.25rem;color:#111}
p{margin:0;color:#6b7280;font-size:.875rem}</style></head>
<body><div class="card"><div class="check">&#10003;</div><h1>Authorization Complete!</h1><p>You can close this tab and return to GoViral.</p></div></body></html>`)
	} else {
		fmt.Fprintf(w, `<!DOCTYPE html>
<html><head><title>Authorization Failed</title>
<style>body{font-family:system-ui,sans-serif;display:flex;align-items:center;justify-content:center;min-height:100vh;margin:0;background:#f9fafb}
.card{text-align:center;padding:2rem;border-radius:12px;background:#fff;box-shadow:0 1px 3px rgba(0,0,0,.1)}
.x{color:#ef4444;font-size:3rem;margin-bottom:1rem}
h1{margin:0 0 .5rem;font-size:1.25rem;color:#111}
p{margin:0;color:#6b7280;font-size:.875rem}</style></head>
<body><div class="card"><div class="x">&#10007;</div><h1>Authorization Failed</h1><p>%s</p><p style="margin-top:1rem">You can close this tab and try again.</p></div></body></html>`, errMsg)
	}
}
