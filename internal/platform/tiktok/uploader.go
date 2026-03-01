package tiktok

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
	"time"

	"github.com/shuhao/goviral/internal/config"
)

//go:embed scripts/tiktok_bridge.py
var tiktokBridgeScript []byte

// UploaderClient interacts with TikTok via the tiktok-uploader Python package (Playwright-based).
type UploaderClient struct {
	pythonPath string
	scriptPath string
}

type tiktokCommand struct {
	Action      string   `json:"action"`
	VideoPath   string   `json:"video_path,omitempty"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	ScheduleAt  int64    `json:"schedule_at,omitempty"` // unix timestamp
	CookiePath  string   `json:"cookie_path,omitempty"`
}

// NewUploaderClient creates a Python-based TikTok uploader client.
func NewUploaderClient() (*UploaderClient, error) {
	pythonPath, err := ensureTikTokVenv()
	if err != nil {
		return nil, fmt.Errorf("setting up python venv for tiktok: %w", err)
	}

	scriptPath, err := ensureTikTokScript()
	if err != nil {
		return nil, fmt.Errorf("writing tiktok bridge script: %w", err)
	}

	return &UploaderClient{
		pythonPath: pythonPath,
		scriptPath: scriptPath,
	}, nil
}

// UploadVideo uploads a video via the tiktok-uploader Python package.
func (c *UploaderClient) UploadVideo(ctx context.Context, videoPath string, description string, tags []string) (string, error) {
	cookiePath := filepath.Join(config.DefaultConfigDir(), "tiktok_cookies.json")
	result, err := c.runCommand(ctx, tiktokCommand{
		Action:      "upload_video",
		VideoPath:   videoPath,
		Description: description,
		Tags:        tags,
		CookiePath:  cookiePath,
	})
	if err != nil {
		return "", fmt.Errorf("tiktok bridge upload_video: %w", err)
	}
	if errMsg, ok := result["error"].(string); ok && errMsg != "" {
		return "", fmt.Errorf("tiktok bridge error: %s", errMsg)
	}
	videoID, _ := result["video_id"].(string)
	if videoID == "" {
		videoID = "uploaded" // tiktok-uploader doesn't always return an ID
	}
	return videoID, nil
}

// ScheduleVideo schedules a video upload via the tiktok-uploader Python package.
func (c *UploaderClient) ScheduleVideo(ctx context.Context, videoPath string, description string, tags []string, scheduledAt time.Time) (string, error) {
	cookiePath := filepath.Join(config.DefaultConfigDir(), "tiktok_cookies.json")
	result, err := c.runCommand(ctx, tiktokCommand{
		Action:      "schedule_video",
		VideoPath:   videoPath,
		Description: description,
		Tags:        tags,
		ScheduleAt:  scheduledAt.Unix(),
		CookiePath:  cookiePath,
	})
	if err != nil {
		return "", fmt.Errorf("tiktok bridge schedule_video: %w", err)
	}
	if errMsg, ok := result["error"].(string); ok && errMsg != "" {
		return "", fmt.Errorf("tiktok bridge error: %s", errMsg)
	}
	videoID, _ := result["video_id"].(string)
	if videoID == "" {
		videoID = "scheduled"
	}
	return videoID, nil
}

func (c *UploaderClient) runCommand(ctx context.Context, cmd tiktokCommand) (map[string]interface{}, error) {
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
			slog.Warn("tiktok bridge stderr", "output", stderrStr)
		}
		return nil, fmt.Errorf("running tiktok bridge: %w (stderr: %s)", err, stderrStr)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("parsing tiktok bridge output: %w (raw: %s)", err, stdout.String())
	}

	return result, nil
}

func ensureTikTokVenv() (string, error) {
	configDir := config.DefaultConfigDir()
	venvDir := filepath.Join(configDir, "venv")

	venvPython := filepath.Join(venvDir, "bin", "python")
	if _, err := os.Stat(venvPython); err != nil {
		pythonPath, err := findPython()
		if err != nil {
			return "", err
		}
		cmd := exec.Command(pythonPath, "-m", "venv", venvDir)
		if out, err := cmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("creating venv: %w (output: %s)", err, string(out))
		}
	}

	pip := filepath.Join(venvDir, "bin", "pip")
	deps := []string{"tiktok-uploader", "playwright"}
	cmd := exec.Command(pip, append([]string{"install", "-q"}, deps...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("installing tiktok deps: %w (output: %s)", err, string(out))
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

func ensureTikTokScript() (string, error) {
	configDir := config.DefaultConfigDir()
	scriptDir := filepath.Join(configDir, "scripts")
	if err := os.MkdirAll(scriptDir, 0755); err != nil {
		return "", fmt.Errorf("creating scripts dir: %w", err)
	}

	scriptPath := filepath.Join(scriptDir, "tiktok_bridge.py")

	existingData, err := os.ReadFile(scriptPath)
	if err == nil {
		existingHash := sha256.Sum256(existingData)
		newHash := sha256.Sum256(tiktokBridgeScript)
		if existingHash == newHash {
			return scriptPath, nil
		}
	}

	if err := os.WriteFile(scriptPath, tiktokBridgeScript, 0644); err != nil {
		return "", fmt.Errorf("writing tiktok bridge script: %w", err)
	}

	return scriptPath, nil
}
