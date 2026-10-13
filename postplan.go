package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const defaultPostplanURL = "https://postplan.dev"

type uploadDetails struct {
	filename    string
	description string
	repoName    string
	branch      string
}

type uploadRequest struct {
	HTML        string            `json:"html"`
	Filename    string            `json:"filename"`
	Description string            `json:"description,omitempty"`
	Metadata    map[string]string `json:"metadata"`
}

type uploadReceipt struct {
	OK         bool     `json:"ok"`
	DraftID    string   `json:"draftId"`
	PublicURL  string   `json:"publicUrl"`
	VersionURL string   `json:"versionUrl"`
	Error      string   `json:"error"`
	Errors     []string `json:"errors"`
}

type postplanAuth struct {
	apiURL string
	apiKey string
}

func uploadToPostplan(page []byte, details uploadDetails) (*uploadReceipt, error) {
	auth := loadPostplanAuth()
	if auth.apiKey == "" {
		return nil, errors.New("postplan only runs scripts for signed-in uploads, and the diff viewer needs them: run `npx postplan auth login` or set POSTPLAN_API_KEY, or pass -no-upload")
	}

	body, err := json.Marshal(uploadRequest{
		HTML:        string(page),
		Filename:    details.filename,
		Description: details.description,
		Metadata: map[string]string{
			"repoName": details.repoName,
			"branch":   details.branch,
			"client":   "tastic",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("encoding upload: %w", err)
	}

	request, err := http.NewRequest(http.MethodPost, auth.apiURL+"/api/uploads", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+auth.apiKey)
	request.Header.Set("User-Agent", "tastic")

	client := &http.Client{Timeout: 5 * time.Minute}
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("uploading to %s: %w", auth.apiURL, err)
	}
	defer response.Body.Close()

	raw, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("reading postplan response: %w", err)
	}

	var receipt uploadReceipt
	if err := json.Unmarshal(raw, &receipt); err != nil {
		return nil, fmt.Errorf("postplan returned HTTP %d: %s", response.StatusCode, truncate(string(raw), 200))
	}
	if !receipt.OK {
		return nil, fmt.Errorf("postplan rejected the upload (HTTP %d): %s", response.StatusCode, receipt.failureReason())
	}
	return &receipt, nil
}

func (r uploadReceipt) failureReason() string {
	reasons := append([]string{}, r.Errors...)
	if r.Error != "" {
		reasons = append(reasons, r.Error)
	}
	if len(reasons) == 0 {
		return "no reason given"
	}
	return strings.Join(reasons, "; ")
}

func loadPostplanAuth() postplanAuth {
	var config struct {
		APIURL string `json:"apiUrl"`
	}
	var credentials struct {
		APIKey string `json:"apiKey"`
	}
	if home, err := os.UserHomeDir(); err == nil {
		readJSONFile(filepath.Join(home, ".postplan", "config.json"), &config)
		readJSONFile(filepath.Join(home, ".postplan", "credentials.json"), &credentials)
	}

	apiURL := firstNonEmpty(os.Getenv("POSTPLAN_API_URL"), config.APIURL, defaultPostplanURL)
	return postplanAuth{
		apiURL: strings.TrimRight(apiURL, "/"),
		apiKey: firstNonEmpty(os.Getenv("POSTPLAN_API_KEY"), credentials.APIKey),
	}
}

func readJSONFile(path string, target any) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return
	}
	_ = json.Unmarshal(contents, target)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
