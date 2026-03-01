package youtube

import (
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/shuhao/goviral/internal/config"
)

//go:embed scripts/youtube_bridge.py
var youtubeBridgeScript []byte

// BridgeClient interacts with YouTube via a Python subprocess using google-api-python-client.
type BridgeClient struct {
	pythonPath string
	scriptPath string
}

type youtubeCommand struct {
	Action        string   `json:"action"`
	VideoPath     string   `json:"video_path,omitempty"`
	ThumbnailPath string   `json:"thumbnail_path,omitempty"`
	Title         string   `json:"title,omitempty"`
	Description   string   `json:"description,omitempty"`
	Tags          []string `json:"tags,omitempty"`
}

// NewBridgeClient creates a Python bridge client for YouTube.
func NewBridgeClient() (*BridgeClient, error) {
	pythonPath, err := ensureYouTubeVenv()
	if err != nil {
		return nil, fmt.Errorf("setting up python venv for youtube: %w", err)
	}

	scriptPath, err := ensureYouTubeScript()
	if err != nil {
		return nil, fmt.Errorf("writing youtube bridge script: %w", err)
	}

	return &BridgeClient{
		pythonPath: pythonPath,
		scriptPath: scriptPath,
	}, nil
}

// UploadVideo uploads a video via the Python bridge.
func (c *BridgeClient) UploadVideo(ctx context.Context, videoPath string, title string, description string, tags []string) (string, error) {
	result, err := c.runCommand(ctx, youtubeCommand{
		Action:      "upload_video",
		VideoPath:   videoPath,
		Title:       title,
		Description: description,
		Tags:        tags,
	})
	if err != nil {
		return "", fmt.Errorf("youtube bridge upload_video: %w", err)
	}
	if errMsg, ok := result["error"].(string); ok && errMsg != "" {
		return "", fmt.Errorf("youtube bridge error: %s", errMsg)
	}
	videoID, _ := result["video_id"].(string)
	if videoID == "" {
		return "", fmt.Errorf("youtube bridge returned empty video_id")
	}
	return videoID, nil
}

// UploadVideoWithThumbnail uploads a video with thumbnail via the Python bridge.
func (c *BridgeClient) UploadVideoWithThumbnail(ctx context.Context, videoPath string, thumbnailPath string, title string, description string, tags []string) (string, error) {
	result, err := c.runCommand(ctx, youtubeCommand{
		Action:        "upload_video",
		VideoPath:     videoPath,
		ThumbnailPath: thumbnailPath,
		Title:         title,
		Description:   description,
		Tags:          tags,
	})
	if err != nil {
		return "", fmt.Errorf("youtube bridge upload_video_with_thumbnail: %w", err)
	}
	if errMsg, ok := result["error"].(string); ok && errMsg != "" {
		return "", fmt.Errorf("youtube bridge error: %s", errMsg)
	}
	videoID, _ := result["video_id"].(string)
	if videoID == "" {
		return "", fmt.Errorf("youtube bridge returned empty video_id")
	}
	return videoID, nil
}

func (c *BridgeClient) runCommand(ctx context.Context, cmd youtubeCommand) (map[string]interface{}, error) {
	inputJSON, err := json.Marshal(cmd)
	if err != nil {
		return nil, fmt.Errorf("marshaling command: %w", err)
	}

	execCmd := exec.CommandContext(ctx, c.pythonPath, c.scriptPath)
	execCmd.Stdin = bytes.NewReader(inputJSON)

	var stdout, stderr bytes.Buffer
	execCmd.Stdout = &stdout
	execCmd.Stderr = &stderr

	if err := execCmd.Run(); err != nil {
		stderrStr := stderr.String()
		if stderrStr != "" {
			slog.Warn("youtube bridge stderr", "output", stderrStr)
		}
		return nil, fmt.Errorf("running youtube bridge: %w (stderr: %s)", err, stderrStr)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("parsing youtube bridge output: %w (raw: %s)", err, stdout.String())
	}

	return result, nil
}

func ensureYouTubeVenv() (string, error) {
	configDir := config.DefaultConfigDir()
	venvDir := filepath.Join(configDir, "venv")

	venvPython := filepath.Join(venvDir, "bin", "python3")
	if _, err := os.Stat(venvPython); err != nil {
		// Find system python
		pythonPath, err := findPython()
		if err != nil {
			return "", err
		}

		// Create venv
		cmd := exec.Command(pythonPath, "-m", "venv", venvDir)
		if out, err := cmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("creating venv: %w (output: %s)", err, string(out))
		}
	}

	// Install dependencies
	pip := filepath.Join(venvDir, "bin", "pip")
	deps := []string{"google-api-python-client", "google-auth-httplib2", "google-auth-oauthlib"}
	cmd := exec.Command(pip, append([]string{"install", "-q"}, deps...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("installing youtube deps: %w (output: %s)", err, string(out))
	}

	return venvPython, nil
}

func findPython() (string, error) {
	for _, name := range []string{"python3", "python"} {
		path, err := exec.LookPath(name)
		if err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("python3 or python not found on PATH")
}

func ensureYouTubeScript() (string, error) {
	configDir := config.DefaultConfigDir()
	scriptDir := filepath.Join(configDir, "scripts")
	if err := os.MkdirAll(scriptDir, 0755); err != nil {
		return "", fmt.Errorf("creating scripts dir: %w", err)
	}

	scriptPath := filepath.Join(scriptDir, "youtube_bridge.py")

	// Check if script already exists and is up to date via content hash comparison.
	existingData, err := os.ReadFile(scriptPath)
	if err == nil {
		existingHash := sha256.Sum256(existingData)
		newHash := sha256.Sum256(youtubeBridgeScript)
		if existingHash == newHash {
			return scriptPath, nil
		}
	}

	if err := os.WriteFile(scriptPath, youtubeBridgeScript, 0644); err != nil {
		return "", fmt.Errorf("writing youtube bridge script: %w", err)
	}

	return scriptPath, nil
}
