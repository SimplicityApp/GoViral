package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/shuhao/goviral/apps/server/dto"
	"github.com/shuhao/goviral/apps/server/middleware"
	"github.com/shuhao/goviral/apps/server/service"
	"github.com/shuhao/goviral/internal/codeimg"
	"github.com/shuhao/goviral/internal/config"
	"github.com/shuhao/goviral/internal/db"
	"github.com/shuhao/goviral/pkg/models"
)

// RepoHandler handles GitHub repo-to-post endpoints.
type RepoHandler struct {
	svc     *service.RepoService
	opStore *service.OperationStore
	cfg     *config.Config
	db      *db.DB
}

// NewRepoHandler creates a new RepoHandler.
func NewRepoHandler(svc *service.RepoService, opStore *service.OperationStore, cfg *config.Config, database *db.DB) *RepoHandler {
	return &RepoHandler{svc: svc, opStore: opStore, cfg: cfg, db: database}
}

// resolveGitHubToken loads the user config and returns the merged GitHub token.
func (h *RepoHandler) resolveGitHubToken(userID string) string {
	uc, _ := h.db.GetUserConfig(userID)
	if uc == nil {
		return h.cfg.GitHub.PersonalAccessToken
	}
	return uc.MergedGitHubToken(*h.cfg)
}

// ListAvailableRepos returns all GitHub repos accessible to the authenticated user.
func (h *RepoHandler) ListAvailableRepos(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	token := h.resolveGitHubToken(userID)
	repos, err := h.svc.ListAvailableRepos(r.Context(), token)
	if err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusInternalServerError, dto.ErrCodeInternal, err.Error(), reqID)
		return
	}

	q := strings.ToLower(r.URL.Query().Get("q"))

	resp := make([]dto.AvailableRepoResponse, 0, len(repos))
	for _, repo := range repos {
		if q != "" && !strings.Contains(strings.ToLower(repo.FullName), q) &&
			!strings.Contains(strings.ToLower(repo.Description), q) {
			continue
		}
		resp = append(resp, dto.AvailableRepoResponse{
			FullName:    repo.FullName,
			Owner:       repo.Owner,
			Name:        repo.Name,
			Description: repo.Description,
			Language:    repo.Language,
			Private:     repo.Private,
		})
	}

	middleware.WriteJSON(w, http.StatusOK, resp)
}

// ListRepos returns all tracked repositories.
func (h *RepoHandler) ListRepos(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	repos, err := h.svc.ListRepos(r.Context(), userID)
	if err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusInternalServerError, dto.ErrCodeInternal, "failed to list repos", reqID)
		return
	}

	resp := make([]dto.RepoResponse, 0, len(repos))
	for _, repo := range repos {
		resp = append(resp, repoToResponse(repo))
	}

	middleware.WriteJSON(w, http.StatusOK, resp)
}

// AddRepo validates and adds a GitHub repo to tracking.
func (h *RepoHandler) AddRepo(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	var req dto.AddRepoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusBadRequest, dto.ErrCodeValidation, "invalid request body", reqID)
		return
	}

	if req.Owner == "" || req.Name == "" {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusBadRequest, dto.ErrCodeValidation, "owner and name are required", reqID)
		return
	}

	token := h.resolveGitHubToken(userID)
	repo, err := h.svc.AddRepo(r.Context(), userID, req.Owner, req.Name, token)
	if err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusInternalServerError, dto.ErrCodeInternal, err.Error(), reqID)
		return
	}

	middleware.WriteJSON(w, http.StatusCreated, repoToResponse(*repo))
}

// DeleteRepo removes a tracked repository.
func (h *RepoHandler) DeleteRepo(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusBadRequest, dto.ErrCodeValidation, "invalid repo ID", reqID)
		return
	}

	if err := h.svc.DeleteRepo(r.Context(), userID, id); err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusInternalServerError, dto.ErrCodeInternal, err.Error(), reqID)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListCommits returns stored commits for a repo.
func (h *RepoHandler) ListCommits(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	idStr := chi.URLParam(r, "id")
	repoID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusBadRequest, dto.ErrCodeValidation, "invalid repo ID", reqID)
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	commits, err := h.svc.ListCommits(r.Context(), userID, repoID, limit)
	if err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusInternalServerError, dto.ErrCodeInternal, "failed to list commits", reqID)
		return
	}

	resp := make([]dto.RepoCommitResponse, 0, len(commits))
	for _, c := range commits {
		resp = append(resp, commitToResponse(c))
	}

	middleware.WriteJSON(w, http.StatusOK, resp)
}

// FetchCommits is an SSE endpoint that fetches commits from GitHub and streams
// progress events back to the client.
func (h *RepoHandler) FetchCommits(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	repoID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusBadRequest, dto.ErrCodeValidation, "invalid repo ID", reqID)
		return
	}

	var req dto.FetchCommitsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Allow empty or missing body (all defaults apply)
		req = dto.FetchCommitsRequest{}
	}

	userID := middleware.UserIDFromContext(r.Context())
	token := h.resolveGitHubToken(userID)

	if WantsSSE(r) {
		svcProgress := make(chan dto.ProgressEvent, 10)
		clientProgress := make(chan dto.ProgressEvent, 10)

		go func() {
			var result []models.RepoCommitRecord
			var fetchErr error

			done := make(chan struct{})
			go func() {
				defer close(done)
				result, fetchErr = h.svc.FetchCommits(r.Context(), userID, repoID, req.Limit, req.Since, token, svcProgress)
			}()

			// Forward service events to the client in real-time
			for evt := range svcProgress {
				if evt.Type == "complete" {
					<-done // ensure result is populated
					if result != nil {
						responses := make([]dto.RepoCommitResponse, 0, len(result))
						for _, c := range result {
							responses = append(responses, commitToResponse(c))
						}
						evt.Data = responses
					}
				}
				clientProgress <- evt
			}

			<-done
			if fetchErr != nil {
				clientProgress <- dto.ProgressEvent{
					Type:    "error",
					Message: fetchErr.Error(),
				}
			}
			close(clientProgress)
		}()

		StreamProgress(w, r, clientProgress)
		return
	}

	// Background mode
	opID := h.opStore.Create()
	go func() {
		progress := make(chan dto.ProgressEvent, 10)
		go func() {
			result, err := h.svc.FetchCommits(context.Background(), userID, repoID, req.Limit, req.Since, token, progress)
			if err != nil {
				h.opStore.Fail(opID, err.Error())
				return
			}
			responses := make([]dto.RepoCommitResponse, 0, len(result))
			for _, c := range result {
				responses = append(responses, commitToResponse(c))
			}
			h.opStore.Complete(opID, responses)
		}()
		for range progress {
		}
	}()

	middleware.WriteJSON(w, http.StatusAccepted, dto.OperationResponse{
		ID:     opID,
		Status: "running",
	})
}

// GeneratePosts is an SSE endpoint that generates posts from commits and streams
// progress events back to the client.
func (h *RepoHandler) GeneratePosts(w http.ResponseWriter, r *http.Request) {
	var req dto.GenerateRepoPostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusBadRequest, dto.ErrCodeValidation, "invalid request body", reqID)
		return
	}

	if len(req.CommitIDs) == 0 {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusBadRequest, dto.ErrCodeValidation, "commit_ids is required", reqID)
		return
	}

	userID := middleware.UserIDFromContext(r.Context())

	if WantsSSE(r) {
		svcProgress := make(chan dto.ProgressEvent, 10)
		clientProgress := make(chan dto.ProgressEvent, 10)

		go func() {
			var result []models.GeneratedContent
			var genErr error

			done := make(chan struct{})
			go func() {
				defer close(done)
				result, genErr = h.svc.GenerateFromCommits(r.Context(), userID, req, svcProgress)
			}()

			for evt := range svcProgress {
				if evt.Type == "complete" {
					<-done
					if result != nil {
						responses := make([]dto.GeneratedContentResponse, len(result))
						for i, gc := range result {
							responses[i] = contentToResponse(gc)
						}
						evt.Data = responses
					}
				}
				clientProgress <- evt
			}

			<-done
			if genErr != nil {
				clientProgress <- dto.ProgressEvent{
					Type:    "error",
					Message: genErr.Error(),
				}
			}
			close(clientProgress)
		}()

		StreamProgress(w, r, clientProgress)
		return
	}

	// Background mode
	opID := h.opStore.Create()
	go func() {
		progress := make(chan dto.ProgressEvent, 10)
		go func() {
			result, err := h.svc.GenerateFromCommits(context.Background(), userID, req, progress)
			if err != nil {
				h.opStore.Fail(opID, err.Error())
				return
			}
			responses := make([]dto.GeneratedContentResponse, len(result))
			for i, gc := range result {
				responses[i] = contentToResponse(gc)
			}
			h.opStore.Complete(opID, responses)
		}()
		for range progress {
		}
	}()

	middleware.WriteJSON(w, http.StatusAccepted, dto.OperationResponse{
		ID:     opID,
		Status: "running",
	})
}

// RenderCodeImage renders a code diff image for a commit and returns PNG bytes.
func (h *RepoHandler) RenderCodeImage(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	var req dto.RenderCodeImageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusBadRequest, dto.ErrCodeValidation, "invalid request body", reqID)
		return
	}

	if req.CommitID == 0 {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusBadRequest, dto.ErrCodeValidation, "commit_id is required", reqID)
		return
	}

	pngBytes, _, err := h.svc.RenderCodeImage(r.Context(), userID, req.CommitID, req.Template, req.Theme)
	if err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusInternalServerError, dto.ErrCodeInternal, err.Error(), reqID)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.WriteHeader(http.StatusOK)
	w.Write(pngBytes) //nolint:errcheck
}

// GetCodeImage serves a pre-rendered PNG code diff image for a commit.
func (h *RepoHandler) GetCodeImage(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	commitIDStr := chi.URLParam(r, "commitId")
	commitID, err := strconv.ParseInt(commitIDStr, 10, 64)
	if err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusBadRequest, dto.ErrCodeValidation, "invalid commit ID", reqID)
		return
	}

	pngBytes, _, err := h.svc.RenderCodeImage(r.Context(), userID, commitID, "", "")
	if err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		if strings.Contains(err.Error(), "no diff") || strings.Contains(err.Error(), "not found") {
			middleware.WriteError(w, http.StatusNotFound, dto.ErrCodeNotFound, "no code image available", reqID)
			return
		}
		middleware.WriteError(w, http.StatusInternalServerError, dto.ErrCodeInternal, err.Error(), reqID)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.WriteHeader(http.StatusOK)
	w.Write(pngBytes) //nolint:errcheck
}

// GetContentCodeImage serves the AI-selected code image PNG for a specific
// generated content item. Falls back to commit-level heuristic rendering
// if no content-specific image exists.
func (h *RepoHandler) GetContentCodeImage(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	contentIDStr := chi.URLParam(r, "contentId")
	contentID, err := strconv.ParseInt(contentIDStr, 10, 64)
	if err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusBadRequest, dto.ErrCodeValidation, "invalid content ID", reqID)
		return
	}

	gc, err := h.svc.GetContentByID(r.Context(), userID, contentID)
	if err != nil || gc == nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusNotFound, dto.ErrCodeNotFound, "content not found", reqID)
		return
	}

	if gc.CodeImagePath == "" {
		// No saved image — try rendering from commit heuristic as fallback
		if gc.SourceCommitID > 0 {
			pngBytes, _, err := h.svc.RenderCodeImage(r.Context(), userID, gc.SourceCommitID, "", "")
			if err != nil {
				reqID := middleware.RequestIDFromContext(r.Context())
				middleware.WriteError(w, http.StatusNotFound, dto.ErrCodeNotFound, "no code image available", reqID)
				return
			}
			w.Header().Set("Content-Type", "image/png")
			w.Header().Set("Cache-Control", "public, max-age=3600")
			w.WriteHeader(http.StatusOK)
			w.Write(pngBytes) //nolint:errcheck
			return
		}
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusNotFound, dto.ErrCodeNotFound, "no code image available", reqID)
		return
	}

	http.ServeFile(w, r, gc.CodeImagePath)
}

// UpdateSettings updates the target audience and links for a repo.
func (h *RepoHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusBadRequest, dto.ErrCodeValidation, "invalid repo ID", reqID)
		return
	}

	var req dto.UpdateRepoSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusBadRequest, dto.ErrCodeValidation, "invalid request body", reqID)
		return
	}

	links := make([]models.RepoLink, len(req.Links))
	for i, l := range req.Links {
		links[i] = models.RepoLink{Label: l.Label, URL: l.URL}
	}

	repo, err := h.svc.UpdateRepoSettings(r.Context(), userID, id, req.TargetAudience, links)
	if err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusInternalServerError, dto.ErrCodeInternal, err.Error(), reqID)
		return
	}
	if repo == nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusNotFound, dto.ErrCodeNotFound, "repo not found", reqID)
		return
	}

	middleware.WriteJSON(w, http.StatusOK, repoToResponse(*repo))
}

// ListCodeImageOptions returns available templates and themes for code image rendering.
func (h *RepoHandler) ListCodeImageOptions(w http.ResponseWriter, r *http.Request) {
	templates := codeimg.TemplateNames()
	themes := codeimg.ThemeNames()

	resp := dto.CodeImageOptionsResponse{
		Templates: make([]dto.CodeImageTemplateResponse, len(templates)),
		Themes:    make([]dto.CodeImageThemeResponse, len(themes)),
	}

	for i, name := range templates {
		spec := codeimg.LookupTemplate(name)
		resp.Templates[i] = dto.CodeImageTemplateResponse{
			Name:                spec.Name,
			Description:         spec.Description,
			SupportsDescription: spec.SupportsDescription,
		}
	}
	for i, name := range themes {
		resp.Themes[i] = dto.CodeImageThemeResponse{Name: name}
	}

	middleware.WriteJSON(w, http.StatusOK, resp)
}

// ListCodeImagePreviews returns rendered HTML previews for all templates with a given theme.
func (h *RepoHandler) ListCodeImagePreviews(w http.ResponseWriter, r *http.Request) {
	theme := r.URL.Query().Get("theme")
	if theme == "" {
		theme = "github-dark"
	}

	samplePatch := `@@ -10,7 +10,9 @@ func main() {
 	fmt.Println("hello")
-	fmt.Println("old line")
+	fmt.Println("new line")
+	fmt.Println("added line")
 	fmt.Println("world")
 }`

	data := codeimg.TemplateDiffData{
		Filename:    "main.go",
		Language:    "Go",
		Description: "Refactored greeting logic",
		RepoName:    "acme/app",
		Theme:       codeimg.LookupTheme(theme),
	}

	// Parse the sample patch into diff lines
	parsed := codeimg.FormatDiffForTemplate(samplePatch, 0)
	data.Lines = parsed.Lines
	data.Additions = parsed.Additions
	data.Deletions = parsed.Deletions

	previews := make(map[string]string)
	for _, name := range codeimg.TemplateNames() {
		html, err := codeimg.RenderHTML(data, name)
		if err != nil {
			continue
		}
		previews[name] = html
	}

	w.Header().Set("Cache-Control", "public, max-age=86400")
	middleware.WriteJSON(w, http.StatusOK, dto.CodeImagePreviewsResponse{
		Theme:    theme,
		Previews: previews,
	})
}

// ReRenderContentCodeImage re-renders the code image for a content item using
// its current code_image_description, then returns the updated content response.
func (h *RepoHandler) ReRenderContentCodeImage(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	contentIDStr := chi.URLParam(r, "contentId")
	contentID, err := strconv.ParseInt(contentIDStr, 10, 64)
	if err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusBadRequest, dto.ErrCodeValidation, "invalid content ID", reqID)
		return
	}

	newPath, err := h.svc.ReRenderCodeImage(r.Context(), userID, contentID)
	if err != nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		if strings.Contains(err.Error(), "not found") {
			middleware.WriteError(w, http.StatusNotFound, dto.ErrCodeNotFound, err.Error(), reqID)
			return
		}
		middleware.WriteError(w, http.StatusInternalServerError, dto.ErrCodeInternal, err.Error(), reqID)
		return
	}

	// Update the code_image_path in the DB if it changed
	gc, err := h.svc.GetContentByID(r.Context(), userID, contentID)
	if err != nil || gc == nil {
		reqID := middleware.RequestIDFromContext(r.Context())
		middleware.WriteError(w, http.StatusInternalServerError, dto.ErrCodeInternal, "failed to reload content", reqID)
		return
	}

	if gc.CodeImagePath != newPath {
		gc.CodeImagePath = newPath
	}

	middleware.WriteJSON(w, http.StatusOK, contentToResponse(*gc))
}

// --- helpers ---

func repoToResponse(repo models.GitHubRepo) dto.RepoResponse {
	links := make([]dto.RepoLinkDTO, len(repo.Links))
	for i, l := range repo.Links {
		links[i] = dto.RepoLinkDTO{Label: l.Label, URL: l.URL}
	}
	if links == nil {
		links = []dto.RepoLinkDTO{}
	}

	return dto.RepoResponse{
		ID:             repo.ID,
		Owner:          repo.Owner,
		Name:           repo.Name,
		FullName:       repo.FullName,
		Description:    repo.Description,
		DefaultBranch:  repo.DefaultBranch,
		Language:       repo.Language,
		AddedAt:        repo.AddedAt.Format("2006-01-02T15:04:05Z07:00"),
		TargetAudience: repo.TargetAudience,
		Links:          links,
	}
}

func commitToResponse(rc models.RepoCommitRecord) dto.RepoCommitResponse {
	var files []dto.RepoFileResponse
	if rc.FilesJSON != "" && rc.FilesJSON != "[]" {
		var rawFiles []models.GitHubFileChange
		if err := json.Unmarshal([]byte(rc.FilesJSON), &rawFiles); err == nil {
			files = make([]dto.RepoFileResponse, 0, len(rawFiles))
			for _, f := range rawFiles {
				files = append(files, dto.RepoFileResponse{
					Filename:  f.Filename,
					Status:    f.Status,
					Additions: f.Additions,
					Deletions: f.Deletions,
				})
			}
		}
	}
	if files == nil {
		files = []dto.RepoFileResponse{}
	}

	return dto.RepoCommitResponse{
		ID:           rc.ID,
		RepoID:       rc.RepoID,
		SHA:          rc.SHA,
		Message:      rc.Message,
		AuthorName:   rc.AuthorName,
		CommittedAt:  rc.CommittedAt.Format("2006-01-02T15:04:05Z07:00"),
		Additions:    rc.Additions,
		Deletions:    rc.Deletions,
		FilesChanged: rc.FilesChanged,
		DiffSummary:  rc.DiffSummary,
		Files:        files,
	}
}
