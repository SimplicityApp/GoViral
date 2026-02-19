package handler

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/shuhao/goviral/apps/server/dto"
	"github.com/shuhao/goviral/apps/server/middleware"
	"github.com/shuhao/goviral/internal/auth"
	"github.com/shuhao/goviral/internal/config"
)

const (
	xAuthURL       = "https://x.com/i/oauth2/authorize"
	xScopes        = "tweet.read tweet.write users.read offline.access"
	linkedinAuthURL = "https://www.linkedin.com/oauth/v2/authorization"
	linkedinScopes  = "openid profile w_member_social"
	callbackPort    = 9876
)

// AuthHandler handles OAuth flow requests.
type AuthHandler struct {
	cfg *config.Config

	mu            sync.Mutex
	pendingAuths  map[string]*pendingAuth
}

type pendingAuth struct {
	platform     string
	authURL      string
	codeVerifier string // X only (PKCE)
	status       string // "pending", "completed", "failed"
	error        string
	startedAt    time.Time
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		cfg:          cfg,
		pendingAuths: make(map[string]*pendingAuth),
	}
}

// Start initiates an OAuth flow for the specified platform.
func (h *AuthHandler) Start(w http.ResponseWriter, r *http.Request) {
	platform := chi.URLParam(r, "platform")
	if platform != "x" && platform != "linkedin" {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusBadRequest, dto.ErrCodeValidation, "platform must be 'x' or 'linkedin'", reqID)
		return
	}

	var authURL string
	var codeVerifier string
	redirectURI := fmt.Sprintf("http://localhost:%d/callback", callbackPort)

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
			url.QueryEscape(platform),
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
			url.QueryEscape(platform),
		)
	}

	h.mu.Lock()
	h.pendingAuths[platform] = &pendingAuth{
		platform:     platform,
		authURL:      authURL,
		codeVerifier: codeVerifier,
		status:       "pending",
		startedAt:    time.Now(),
	}
	h.mu.Unlock()

	// Start the callback server in the background
	go h.waitForCallback(platform, redirectURI)

	middleware.WriteJSON(w, http.StatusOK, map[string]string{
		"auth_url": authURL,
		"status":   "pending",
	})
}

// Status returns the current status of an OAuth flow.
func (h *AuthHandler) Status(w http.ResponseWriter, r *http.Request) {
	platform := chi.URLParam(r, "platform")

	h.mu.Lock()
	pa, ok := h.pendingAuths[platform]
	h.mu.Unlock()

	if !ok {
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
}

func (h *AuthHandler) waitForCallback(platform, redirectURI string) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	code, err := auth.StartCallbackServer(ctx, callbackPort)
	if err != nil {
		h.mu.Lock()
		if pa, ok := h.pendingAuths[platform]; ok {
			pa.status = "failed"
			pa.error = err.Error()
		}
		h.mu.Unlock()
		return
	}

	switch platform {
	case "x":
		h.mu.Lock()
		codeVerifier := h.pendingAuths[platform].codeVerifier
		h.mu.Unlock()

		tokenResp, err := auth.ExchangeXCode(code, h.cfg.X.ClientID, h.cfg.X.ClientSecret, codeVerifier, redirectURI)
		if err != nil {
			h.setAuthFailed(platform, err.Error())
			return
		}

		h.cfg.X.AccessToken = tokenResp.AccessToken
		h.cfg.X.RefreshToken = tokenResp.RefreshToken
		if tokenResp.ExpiresIn > 0 {
			h.cfg.X.TokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second).Format(time.RFC3339)
		}

	case "linkedin":
		token, err := auth.ExchangeLinkedInCode(code, h.cfg.LinkedIn.ClientID, h.cfg.LinkedIn.ClientSecret, redirectURI)
		if err != nil {
			h.setAuthFailed(platform, err.Error())
			return
		}
		h.cfg.LinkedIn.AccessToken = token
	}

	if err := config.Save(h.cfg, config.DefaultConfigPath()); err != nil {
		h.setAuthFailed(platform, fmt.Sprintf("saving config: %v", err))
		return
	}

	h.mu.Lock()
	if pa, ok := h.pendingAuths[platform]; ok {
		pa.status = "completed"
	}
	h.mu.Unlock()
}

func (h *AuthHandler) setAuthFailed(platform, errMsg string) {
	h.mu.Lock()
	if pa, ok := h.pendingAuths[platform]; ok {
		pa.status = "failed"
		pa.error = errMsg
	}
	h.mu.Unlock()
}
