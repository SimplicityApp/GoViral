package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/shuhao/goviral/internal/config"
	linkedinplatform "github.com/shuhao/goviral/internal/platform/linkedin"
	xplatform "github.com/shuhao/goviral/internal/platform/x"
)

var syncCookiesCmd = &cobra.Command{
	Use:   "sync-cookies",
	Short: "Extract cookies from Chrome and push them to a remote GoViral server",
	Long: `Extracts X (twikit) and LinkedIn (linkitin) session cookies from your
local Chrome browser, then uploads them to a remote GoViral API server.

This automates the process of keeping your production server authenticated
when using cookie-based fallback clients.

The command will:
  1. Extract fresh cookies from Chrome (like twikit-login + linkitin-login)
  2. Read the cookie files from ~/.goviral/
  3. POST them to the remote server's cookie endpoints

Requires: --server flag with the remote API base URL.`,
	Example: `  goviral sync-cookies --server https://goviral-api.fly.dev/api/v1
  goviral sync-cookies --server https://goviral-api.fly.dev/api/v1 --api-key my-secret
  goviral sync-cookies --server https://goviral-api.fly.dev/api/v1 --skip-extract`,
	RunE: runSyncCookies,
}

var (
	syncServer      string
	syncAPIKey      string
	syncSkipExtract bool
)

func init() {
	syncCookiesCmd.Flags().StringVar(&syncServer, "server", "", "Remote API base URL (e.g. https://goviral-api.fly.dev/api/v1)")
	syncCookiesCmd.Flags().StringVar(&syncAPIKey, "api-key", "", "API key for the remote server (reads from config if omitted)")
	syncCookiesCmd.Flags().BoolVar(&syncSkipExtract, "skip-extract", false, "Skip Chrome extraction, just push existing cookie files")
	_ = syncCookiesCmd.MarkFlagRequired("server")
	rootCmd.AddCommand(syncCookiesCmd)
}

func runSyncCookies(_ *cobra.Command, _ []string) error {
	syncServer = strings.TrimRight(syncServer, "/")

	// Resolve API key.
	apiKey := syncAPIKey
	if apiKey == "" {
		cfg, err := config.Load(config.DefaultConfigPath())
		if err == nil && cfg.Server.APIKey != "" {
			apiKey = cfg.Server.APIKey
		}
	}
	if apiKey == "" {
		return fmt.Errorf("no API key: pass --api-key or set server.api_key in ~/.goviral/config.yaml")
	}

	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	successStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10"))
	warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11"))

	// Step 1: Extract fresh cookies from Chrome (unless skipped).
	if !syncSkipExtract {
		fmt.Println(dimStyle.Render("Extracting cookies from Chrome..."))
		if err := extractLocalCookies(); err != nil {
			fmt.Println(warnStyle.Render("  Cookie extraction had errors: " + err.Error()))
			fmt.Println(warnStyle.Render("  Continuing with any existing cookie files..."))
		} else {
			fmt.Println(successStyle.Render("  Cookies extracted from Chrome"))
		}
	}

	// Step 2: Push X cookies.
	configDir := config.DefaultConfigDir()
	xSynced := false
	twikitPath := filepath.Join(configDir, "twikit_cookies.json")
	if data, err := os.ReadFile(twikitPath); err == nil {
		var cookies map[string]string
		if err := json.Unmarshal(data, &cookies); err == nil {
			authToken := cookies["auth_token"]
			ct0 := cookies["ct0"]
			if authToken != "" && ct0 != "" {
				fmt.Print(dimStyle.Render("Pushing X cookies to remote server... "))
				body := map[string]string{"auth_token": authToken, "ct0": ct0}
				if err := pushCookies(syncServer+"/x/login-cookies", apiKey, body); err != nil {
					fmt.Println(warnStyle.Render("FAILED: " + err.Error()))
				} else {
					fmt.Println(successStyle.Render("OK"))
					xSynced = true
				}
			} else {
				fmt.Println(warnStyle.Render("X cookie file missing auth_token or ct0, skipping"))
			}
		}
	} else {
		fmt.Println(dimStyle.Render("No X cookies found at " + twikitPath + ", skipping"))
	}

	// Step 3: Push LinkedIn cookies.
	liSynced := false
	linkitinPath := filepath.Join(configDir, "linkitin_cookies.json")
	if data, err := os.ReadFile(linkitinPath); err == nil {
		var cookies map[string]interface{}
		if err := json.Unmarshal(data, &cookies); err == nil {
			liAt := cookieString(cookies, "li_at")
			jsessionID := cookieString(cookies, "jsessionid")
			if jsessionID == "" {
				jsessionID = cookieString(cookies, "JSESSIONID")
			}
			if liAt != "" && jsessionID != "" {
				fmt.Print(dimStyle.Render("Pushing LinkedIn cookies to remote server... "))
				body := map[string]string{"li_at": liAt, "jsessionid": jsessionID}
				if err := pushCookies(syncServer+"/linkedin/login-cookies", apiKey, body); err != nil {
					fmt.Println(warnStyle.Render("FAILED: " + err.Error()))
				} else {
					fmt.Println(successStyle.Render("OK"))
					liSynced = true
				}
			} else {
				fmt.Println(warnStyle.Render("LinkedIn cookie file missing li_at or jsessionid, skipping"))
			}
		}
	} else {
		fmt.Println(dimStyle.Render("No LinkedIn cookies found at " + linkitinPath + ", skipping"))
	}

	// Summary.
	fmt.Println()
	if xSynced || liSynced {
		var parts []string
		if xSynced {
			parts = append(parts, "X")
		}
		if liSynced {
			parts = append(parts, "LinkedIn")
		}
		fmt.Println(successStyle.Render("Synced cookies for: " + strings.Join(parts, ", ")))
	} else {
		fmt.Println(warnStyle.Render("No cookies were synced. Make sure you're logged into X/LinkedIn in Chrome."))
	}

	return nil
}

func extractLocalCookies() error {
	ctx := context.Background()
	var errs []string

	// Extract X cookies.
	tc, err := xplatform.NewTwikitClient("")
	if err != nil {
		errs = append(errs, "twikit setup: "+err.Error())
	} else {
		if err := tc.ExtractCookies(ctx); err != nil {
			errs = append(errs, "X: "+err.Error())
		}
	}

	// Extract LinkedIn cookies.
	lc, err := linkedinplatform.NewLinkitinClient()
	if err != nil {
		errs = append(errs, "linkitin setup: "+err.Error())
	} else {
		if err := lc.ExtractCookies(ctx); err != nil {
			errs = append(errs, "LinkedIn: "+err.Error())
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

func pushCookies(url string, apiKey string, body map[string]string) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

func cookieString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
