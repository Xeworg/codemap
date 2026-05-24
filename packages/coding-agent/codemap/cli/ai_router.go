package cli

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// AIProvider is the set of supported AI providers.
type AIProvider string

const (
	AIProviderOllama  AIProvider = "ollama"
	AIProviderMinimax AIProvider = "minimax"
)

// AITestResult is the data payload for an ai-test command result.
type AITestResult struct {
	Provider    string   `json:"provider"`
	Model       string   `json:"model"`
	BaseURL     string   `json:"base_url"`
	Reachable   bool     `json:"reachable"`
	LatencyMS   int64    `json:"latency_ms"`
	Error       string   `json:"error,omitempty"`
	Suggestions []string `json:"suggestions,omitempty"`
}

// TestConnectivityForProvider tests whether a given provider config is reachable
// using a simple deterministic prompt. It returns latency in milliseconds.
func TestConnectivityForProvider(ctx context.Context, w io.Writer, provider AIProvider, baseURL, model, apiKey string, timeoutSec int) AITestResult {
	if timeoutSec <= 0 {
		timeoutSec = 30
	}
	deadline := time.Now().Add(time.Duration(timeoutSec) * time.Second)
	if deadline.Before(time.Now()) {
		deadline = time.Now().Add(30 * time.Second)
	}
	ctx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()

	var reachable bool
	var latency int64
	var errMsg string
	var suggestions []string

	// Build the request.
	var reqBody io.Reader
	var endpoint string
	switch provider {
	case AIProviderOllama:
		reqBody = strings.NewReader(`{"model":"` + model + `","messages":[{"role":"user","content":"reply with exactly one word: ok","stream":false}],"options":{"num_predict":5}}`)
		endpoint = strings.TrimSuffix(baseURL, "/") + "/api/chat"
	case AIProviderMinimax:
		reqBody = strings.NewReader(`{"model":"` + model + `","messages":[{"role":"user","content":"reply with exactly one word: ok"}]}`)
		endpoint = strings.TrimSuffix(baseURL, "/") + "/v1/text/chatcompletion_pro"
	default:
		errMsg = "unknown provider"
		suggestions = []string{"configure ollama or minimax"}
	}

	if errMsg == "" {
		start := time.Now()
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, reqBody)
		if err != nil {
			errMsg = fmt.Sprintf("build request: %v", err)
			suggestions = []string{"check base_url"}
		} else {
			req.Header.Set("Content-Type", "application/json")
			if apiKey != "" && provider == AIProviderMinimax {
				req.Header.Set("Authorization", "Bearer "+apiKey)
			}
			client := &http.Client{Timeout: time.Duration(timeoutSec) * time.Second}
			resp, err := client.Do(req)
			latency = time.Since(start).Milliseconds()
			if err != nil {
				errMsg = fmt.Sprintf("connection failed: %v", err)
				switch provider {
				case AIProviderOllama:
					suggestions = []string{"ensure ollama is running: ollama serve",
						"verify model is pulled: ollama list"}
				case AIProviderMinimax:
					suggestions = []string{"verify API key in config",
						"check network connectivity to minimax API"}
				}
			} else {
				defer resp.Body.Close()
				reachable = resp.StatusCode < 400
				if !reachable {
					errMsg = fmt.Sprintf("HTTP %d", resp.StatusCode)
					suggestions = []string{
						"check API key for provider " + string(provider),
						"verify model name for provider " + string(provider),
					}
				}
			}
		}
	}

	return AITestResult{
		Provider:    string(provider),
		Model:       model,
		BaseURL:     baseURL,
		Reachable:   reachable,
		LatencyMS:   latency,
		Error:       errMsg,
		Suggestions: suggestions,
	}
}
