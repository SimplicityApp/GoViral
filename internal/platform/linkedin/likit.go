package linkedin

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/shuhao/goviral/internal/config"
	"github.com/shuhao/goviral/pkg/models"
)

//go:embed scripts/likit_bridge.py
var likitScript []byte

// LikitClient interacts with LinkedIn via a Python likit subprocess using cookie-based auth.
// Requires a one-time login via ExtractCookies() which saves cookies to ~/.goviral/likit_cookies.json.
type LikitClient struct {
	pythonPath string
	scriptPath string
}

// NewLikitClient creates a LikitClient. Returns an error if python3/python
// is not on PATH or the embedded script cannot be written to disk.
// It reuses the shared virtualenv at ~/.goviral/venv/.
func NewLikitClient() (*LikitClient, error) {
	pythonPath, err := ensureLikitVenv()
	if err != nil {
		return nil, fmt.Errorf("setting up python venv for likit: %w", err)
	}

	scriptPath, err := ensureLikitScript()
	if err != nil {
		return nil, fmt.Errorf("writing likit script: %w", err)
	}

	return &LikitClient{
		pythonPath: pythonPath,
		scriptPath: scriptPath,
	}, nil
}

// ExtractCookies extracts LinkedIn session cookies from Chrome and saves them
// to ~/.goviral/likit_cookies.json. The user must be logged into LinkedIn in Chrome.
func (c *LikitClient) ExtractCookies(ctx context.Context) error {
	result, err := c.runCommand(ctx, likitCommand{Action: "login_browser"})
	if err != nil {
		return fmt.Errorf("extracting LinkedIn cookies: %w", err)
	}
	if errMsg := result["error"]; errMsg != nil {
		return fmt.Errorf("extracting LinkedIn cookies: %s", errMsg)
	}
	return nil
}

// LoginWithCookies authenticates with manually provided cookies.
func (c *LikitClient) LoginWithCookies(ctx context.Context, liAt string, jsessionID string) error {
	result, err := c.runCommand(ctx, likitCommand{
		Action:     "login",
		LiAt:       liAt,
		JSessionID: jsessionID,
	})
	if err != nil {
		return fmt.Errorf("logging in with cookies: %w", err)
	}
	if errMsg := result["error"]; errMsg != nil {
		return fmt.Errorf("logging in with cookies: %s", errMsg)
	}
	return nil
}

// FetchMyPosts fetches the user's LinkedIn posts via the likit subprocess.
func (c *LikitClient) FetchMyPosts(ctx context.Context, limit int) ([]models.Post, error) {
	result, err := c.runCommand(ctx, likitCommand{
		Action: "get_my_posts",
		Limit:  limit,
	})
	if err != nil {
		return nil, fmt.Errorf("fetching LinkedIn posts: %w", err)
	}
	if errMsg := result["error"]; errMsg != nil {
		return nil, fmt.Errorf("fetching LinkedIn posts: %s", errMsg)
	}

	return parseLikitPosts(result)
}

// FetchTrendingPosts searches for trending LinkedIn posts matching the given niches.
func (c *LikitClient) FetchTrendingPosts(ctx context.Context, niches []string, period string, minLikes int, limit int) ([]models.TrendingPost, error) {
	// For LinkedIn, we use search_posts with niche keywords.
	var allPosts []models.TrendingPost
	seen := make(map[string]bool)
	now := time.Now()

	for _, niche := range niches {
		result, err := c.runCommand(ctx, likitCommand{
			Action:   "search_posts",
			Keywords: niche,
			Limit:    limit,
		})
		if err != nil {
			continue
		}
		if errMsg := result["error"]; errMsg != nil {
			continue
		}

		posts, err := parseLikitPosts(result)
		if err != nil {
			continue
		}

		for _, p := range posts {
			if seen[p.PlatformPostID] {
				continue
			}
			if p.Likes < minLikes {
				continue
			}
			seen[p.PlatformPostID] = true

			allPosts = append(allPosts, models.TrendingPost{
				Platform:       "linkedin",
				PlatformPostID: p.PlatformPostID,
				Content:        p.Content,
				Likes:          p.Likes,
				Reposts:        p.Reposts,
				Comments:       p.Comments,
				Impressions:    p.Impressions,
				NicheTags:      []string{niche},
				PostedAt:       p.PostedAt,
				FetchedAt:      now,
			})
		}
	}

	if limit > 0 && len(allPosts) > limit {
		allPosts = allPosts[:limit]
	}
	return allPosts, nil
}

// CreatePost creates a new LinkedIn post.
func (c *LikitClient) CreatePost(ctx context.Context, text string) (string, error) {
	result, err := c.runCommand(ctx, likitCommand{
		Action: "create_post",
		Text:   text,
	})
	if err != nil {
		return "", fmt.Errorf("creating LinkedIn post: %w", err)
	}
	if errMsg := result["error"]; errMsg != nil {
		return "", fmt.Errorf("creating LinkedIn post: %s", errMsg)
	}

	urn, ok := result["urn"].(string)
	if !ok || urn == "" {
		return "", fmt.Errorf("likit returned empty URN for created post")
	}
	return urn, nil
}

// SearchPosts searches for LinkedIn posts matching keywords.
func (c *LikitClient) SearchPosts(ctx context.Context, keywords string, limit int) ([]models.Post, error) {
	result, err := c.runCommand(ctx, likitCommand{
		Action:   "search_posts",
		Keywords: keywords,
		Limit:    limit,
	})
	if err != nil {
		return nil, fmt.Errorf("searching LinkedIn posts: %w", err)
	}
	if errMsg := result["error"]; errMsg != nil {
		return nil, fmt.Errorf("searching LinkedIn posts: %s", errMsg)
	}

	return parseLikitPosts(result)
}

// UploadImage uploads an image and returns the media URN.
func (c *LikitClient) UploadImage(ctx context.Context, imageData []byte, filename string) (string, error) {
	encoded := base64.StdEncoding.EncodeToString(imageData)
	result, err := c.runCommand(ctx, likitCommand{
		Action:    "upload_image",
		ImageData: encoded,
		Filename:  filename,
	})
	if err != nil {
		return "", fmt.Errorf("uploading image to LinkedIn: %w", err)
	}
	if errMsg := result["error"]; errMsg != nil {
		return "", fmt.Errorf("uploading image to LinkedIn: %s", errMsg)
	}

	mediaURN, ok := result["media_urn"].(string)
	if !ok || mediaURN == "" {
		return "", fmt.Errorf("likit returned empty media URN")
	}
	return mediaURN, nil
}

// CreatePostWithImage creates a post with an attached image.
func (c *LikitClient) CreatePostWithImage(ctx context.Context, text string, imageData []byte, filename string) (string, error) {
	encoded := base64.StdEncoding.EncodeToString(imageData)
	result, err := c.runCommand(ctx, likitCommand{
		Action:    "create_post_with_image",
		Text:      text,
		ImageData: encoded,
		Filename:  filename,
	})
	if err != nil {
		return "", fmt.Errorf("creating LinkedIn post with image: %w", err)
	}
	if errMsg := result["error"]; errMsg != nil {
		return "", fmt.Errorf("creating LinkedIn post with image: %s", errMsg)
	}

	urn, ok := result["urn"].(string)
	if !ok || urn == "" {
		return "", fmt.Errorf("likit returned empty URN for created post with image")
	}
	return urn, nil
}

// runCommand executes a single command against the likit bridge script.
// Each invocation spawns the script, sends the JSON command via stdin, and reads the response.
func (c *LikitClient) runCommand(ctx context.Context, cmd likitCommand) (map[string]interface{}, error) {
	cmdJSON, err := json.Marshal(cmd)
	if err != nil {
		return nil, fmt.Errorf("marshaling likit command: %w", err)
	}
	// Append newline so the bridge reads the line.
	cmdJSON = append(cmdJSON, '\n')

	var stdout, stderr bytes.Buffer
	proc := exec.CommandContext(ctx, c.pythonPath, c.scriptPath)
	proc.Stdin = bytes.NewReader(cmdJSON)
	proc.Stdout = &stdout
	proc.Stderr = &stderr

	if err := proc.Run(); err != nil {
		if stdout.Len() > 0 {
			var errResp map[string]interface{}
			if jsonErr := json.Unmarshal(stdout.Bytes(), &errResp); jsonErr == nil {
				if errMsg, ok := errResp["error"]; ok {
					return nil, fmt.Errorf("likit: %s", errMsg)
				}
			}
		}
		stderrMsg := stderr.String()
		if stderrMsg != "" {
			return nil, fmt.Errorf("running likit subprocess: %w (stderr: %s)", err, stderrMsg)
		}
		return nil, fmt.Errorf("running likit subprocess: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("parsing likit output: %w (raw: %s)", err, stdout.String())
	}
	return result, nil
}

// likitCommand represents a JSON command sent to the likit bridge script.
type likitCommand struct {
	Action     string `json:"action"`
	LiAt       string `json:"li_at,omitempty"`
	JSessionID string `json:"jsessionid,omitempty"`
	Text       string `json:"text,omitempty"`
	Keywords   string `json:"keywords,omitempty"`
	Visibility string `json:"visibility,omitempty"`
	ImageData  string `json:"image_data,omitempty"`
	Filename   string `json:"filename,omitempty"`
	Limit      int    `json:"limit,omitempty"`
}

// likitPostJSON represents a post in the likit bridge JSON response.
type likitPostJSON struct {
	URN        string `json:"urn"`
	Text       string `json:"text"`
	Likes      int    `json:"likes"`
	Comments   int    `json:"comments"`
	Reposts    int    `json:"reposts"`
	Impressions int   `json:"impressions"`
	CreatedAt  string `json:"created_at"`
	Author     *struct {
		URN       string `json:"urn"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Headline  string `json:"headline"`
	} `json:"author"`
}

// parseLikitPosts parses the posts array from a likit bridge response.
func parseLikitPosts(result map[string]interface{}) ([]models.Post, error) {
	postsRaw, ok := result["posts"]
	if !ok {
		return nil, fmt.Errorf("likit response missing 'posts' field")
	}

	postsJSON, err := json.Marshal(postsRaw)
	if err != nil {
		return nil, fmt.Errorf("re-marshaling likit posts: %w", err)
	}

	var likitPosts []likitPostJSON
	if err := json.Unmarshal(postsJSON, &likitPosts); err != nil {
		return nil, fmt.Errorf("parsing likit posts: %w", err)
	}

	now := time.Now()
	posts := make([]models.Post, 0, len(likitPosts))
	for _, lp := range likitPosts {
		var postedAt time.Time
		if lp.CreatedAt != "" {
			if t, err := time.Parse(time.RFC3339, lp.CreatedAt); err == nil {
				postedAt = t
			}
		}

		posts = append(posts, models.Post{
			Platform:       "linkedin",
			PlatformPostID: lp.URN,
			Content:        lp.Text,
			Likes:          lp.Likes,
			Reposts:        lp.Reposts,
			Comments:       lp.Comments,
			Impressions:    lp.Impressions,
			PostedAt:       postedAt,
			FetchedAt:      now,
		})
	}

	return posts, nil
}

// ensureLikitVenv ensures the shared virtualenv exists and likit dependencies are installed.
func ensureLikitVenv() (string, error) {
	venvDir := filepath.Join(config.DefaultConfigDir(), "venv")
	venvPython := filepath.Join(venvDir, "bin", "python3")

	// If venv python already exists, ensure likit deps are installed.
	if _, err := os.Stat(venvPython); err == nil {
		if err := ensureLikitDeps(venvPython); err != nil {
			return "", fmt.Errorf("installing likit dependencies: %w", err)
		}
		return venvPython, nil
	}

	// Find system python to create the venv.
	systemPython, err := findLikitPython()
	if err != nil {
		return "", err
	}

	// Create the virtualenv.
	cmd := exec.Command(systemPython, "-m", "venv", venvDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("creating venv: %w (output: %s)", err, string(output))
	}

	if err := ensureLikitDeps(venvPython); err != nil {
		return "", fmt.Errorf("installing likit dependencies: %w", err)
	}

	return venvPython, nil
}

// ensureLikitDeps installs the likit Python dependencies into the venv.
func ensureLikitDeps(pythonPath string) error {
	deps := []string{"httpx", "pydantic", "browser-cookie3"}
	for _, dep := range deps {
		cmd := exec.Command(pythonPath, "-c", "import "+depImportName(dep))
		if err := cmd.Run(); err != nil {
			// Dependency not installed, install it.
			install := exec.Command(pythonPath, "-m", "pip", "install", dep, "-q")
			if output, installErr := install.CombinedOutput(); installErr != nil {
				return fmt.Errorf("installing %s: %w (output: %s)", dep, installErr, string(output))
			}
		}
	}
	return nil
}

// depImportName maps pip package names to Python import names.
func depImportName(pipName string) string {
	switch pipName {
	case "browser-cookie3":
		return "browser_cookie3"
	default:
		return pipName
	}
}

// findLikitPython locates python3 or python on PATH.
func findLikitPython() (string, error) {
	for _, name := range []string{"python3", "python"} {
		path, err := exec.LookPath(name)
		if err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("neither python3 nor python found on PATH")
}

// ensureLikitScript writes the embedded likit_bridge.py to ~/.goviral/scripts/
// along with the likit package files so the bridge can import them.
func ensureLikitScript() (string, error) {
	scriptsDir := filepath.Join(config.DefaultConfigDir(), "scripts")
	if err := os.MkdirAll(scriptsDir, 0755); err != nil {
		return "", fmt.Errorf("creating scripts directory: %w", err)
	}

	scriptPath := filepath.Join(scriptsDir, "likit_bridge.py")

	// Always overwrite to keep the script in sync with the embedded version.
	if err := os.WriteFile(scriptPath, likitScript, 0755); err != nil {
		return "", fmt.Errorf("writing script file: %w", err)
	}

	// Also write the likit package files so the bridge can import them.
	if err := ensureLikitPackage(scriptsDir); err != nil {
		return "", fmt.Errorf("writing likit package: %w", err)
	}

	return scriptPath, nil
}

// ensureLikitPackage installs the likit Python package into the venv using pip.
// This installs from ~/Project/likit/ if it exists (dev mode),
// otherwise relies on the pip-installed package.
func ensureLikitPackage(scriptsDir string) error {
	venvPython := filepath.Join(config.DefaultConfigDir(), "venv", "bin", "python3")

	// Check if likit is already importable.
	cmd := exec.Command(venvPython, "-c", "import likit")
	if err := cmd.Run(); err == nil {
		return nil // Already installed.
	}

	// Try to install from the local development copy.
	home, _ := os.UserHomeDir()
	likitDir := filepath.Join(home, "Project", "likit")
	if _, err := os.Stat(filepath.Join(likitDir, "pyproject.toml")); err == nil {
		install := exec.Command(venvPython, "-m", "pip", "install", "-e", likitDir, "-q")
		if output, err := install.CombinedOutput(); err != nil {
			return fmt.Errorf("installing likit from %s: %w (output: %s)", likitDir, err, string(output))
		}
		return nil
	}

	return fmt.Errorf("likit package not found - install it with: pip install -e ~/Project/likit/")
}

