// Package aiclient wraps the Gemini API, turning a document's extracted text
// into a summary, classification and tags. It is the only part of the AI
// feature set that leaves the machine or costs quota.
package aiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ErrNotConfigured is returned when no API key is set, so callers can tell
// a missing feature from a broken one.
var ErrNotConfigured = fmt.Errorf("AI features are not configured: set GEMINI_API_KEY")

type Client struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

// New returns a Client even with an empty key - every call then fails fast
// with ErrNotConfigured rather than the caller needing to check separately
// whether AI is "on".
func New(apiKey, model string) *Client {
	if model == "" {
		model = "gemini-flash-lite-latest"
	}
	return &Client{
		apiKey: apiKey,
		model:  model,
		// A long document takes a few seconds of model time, and the default
		// (no timeout) would hang a worker against a stalled connection.
		httpClient: &http.Client{Timeout: 45 * time.Second},
	}
}

func (c *Client) Configured() bool { return c.apiKey != "" }

type Analysis struct {
	Summary  string   `json:"summary"`
	Category string   `json:"category"`
	Tags     []string `json:"tags"`
}

// summarySpecs maps the length a caller asked for to concrete prompt
// wording. Left as a fixed set (not a free-text length request) so the
// output stays predictable regardless of what a user types.
var summarySpecs = map[string]string{
	"short":    "1 sentence, the single most important point only",
	"detailed": "a thorough summary of 2-3 paragraphs covering all major points, not just the headline",
}

const defaultSummarySpec = "2-3 sentences"

// Analyze asks Gemini for a summary, a single category label and a handful
// of tags in one call rather than three, because the free tier's request
// budget (15/min) is the actual constraint - three calls per document would
// cut effective throughput to a third for no accuracy benefit.
//
// length selects how long the summary should be ("short", "detailed", or
// "" for the default) - the one knob exposed on top of the fixed prompt,
// since "always gives a short summary" was the one complaint about the
// fixed version.
func (c *Client) Analyze(ctx context.Context, text, length string) (*Analysis, error) {
	if !c.Configured() {
		return nil, ErrNotConfigured
	}

	// Gemini's context window comfortably fits far more than this, but the
	// summary itself degrades in usefulness past a few thousand words of
	// input, and it keeps requests fast and cheap either way.
	if r := []rune(text); len(r) > 12000 {
		text = string(r[:12000])
	}

	spec, ok := summarySpecs[length]
	if !ok {
		spec = defaultSummarySpec
	}

	prompt := "You are analyzing a document for a file-management application. " +
		"Given the document text below, respond with ONLY a JSON object matching this shape " +
		`(no markdown fences, no commentary): {"summary": string (` + spec + `), ` +
		`"category": string (one of: invoice, contract, letter, report, resume, receipt, form, article, other), ` +
		`"tags": string[] (3-6 short keywords)}.\n\nDocument text:\n` + text

	respText, err := c.generate(ctx, prompt)
	if err != nil {
		return nil, err
	}

	analysis, err := parseAnalysis(respText)
	if err != nil {
		return nil, fmt.Errorf("parse model response: %w", err)
	}
	return analysis, nil
}

type geminiRequest struct {
	Contents         []geminiContent  `json:"contents"`
	GenerationConfig generationConfig `json:"generationConfig"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type generationConfig struct {
	ResponseMimeType string  `json:"responseMimeType"`
	Temperature      float64 `json:"temperature"`
}

type geminiResponse struct {
	Candidates []struct {
		Content geminiContent `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (c *Client) generate(ctx context.Context, prompt string) (string, error) {
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", c.model, c.apiKey)

	reqBody := geminiRequest{
		Contents: []geminiContent{{Parts: []geminiPart{{Text: prompt}}}},
		GenerationConfig: generationConfig{
			ResponseMimeType: "application/json",
			// Low temperature: this is document classification, not creative
			// writing - consistency between runs on the same document matters
			// more than variety.
			Temperature: 0.2,
		},
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("gemini request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read gemini response: %w", err)
	}

	var parsed geminiResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", fmt.Errorf("unmarshal gemini response: %w (status %d)", err, resp.StatusCode)
	}

	if parsed.Error != nil {
		if resp.StatusCode == http.StatusTooManyRequests {
			return "", fmt.Errorf("gemini rate limit exceeded, try again shortly: %s", parsed.Error.Message)
		}
		return "", fmt.Errorf("gemini error: %s", parsed.Error.Message)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("gemini returned status %d: %s", resp.StatusCode, string(respBody))
	}
	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini returned no content (the document may have triggered a safety filter)")
	}

	return parsed.Candidates[0].Content.Parts[0].Text, nil
}

func parseAnalysis(raw string) (*Analysis, error) {
	// responseMimeType=application/json should mean raw is already clean
	// JSON, but stripping a stray ```json fence costs nothing and guards
	// against a model that ignores the instruction.
	cleaned := strings.TrimSpace(raw)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	var a Analysis
	if err := json.Unmarshal([]byte(cleaned), &a); err != nil {
		return nil, fmt.Errorf("%w (raw: %s)", err, cleaned)
	}
	if a.Summary == "" {
		return nil, fmt.Errorf("model response had no summary")
	}
	return &a, nil
}
