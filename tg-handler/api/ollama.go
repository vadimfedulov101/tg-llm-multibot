package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// Ollama types
type OllamaRequest struct {
	Model        string  `json:"model"`
	Prompt       string  `json:"prompt"`
	Stream       bool    `json:"stream"`
	Options      Options `json:"options"`
	SystemPrompt string  `json:"system,omitempty"`
	Context      []int   `json:"context,omitempty"`
}

type OllamaResponse struct {
	Model              string `json:"model"`
	CreatedAt          string `json:"created_at"`
	Response           string `json:"response"`
	Done               bool   `json:"done"`
	Context            []int  `json:"context,omitempty"`
	TotalDuration      int64  `json:"total_duration,omitempty"`
	LoadDuration       int64  `json:"load_duration,omitempty"`
	PromptEvalCount    int    `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration int64  `json:"prompt_eval_duration,omitempty"`
	EvalCount          int    `json:"eval_count,omitempty"`
	EvalDuration       int64  `json:"eval_duration,omitempty"`
}

// Create Ollama request
func newOllamaRequest(prompt string, settings *Settings) *OllamaRequest {
	return &OllamaRequest{
		Model:        OLLAMA_MODEL,
		Prompt:       prompt,
		Stream:       false,
		SystemPrompt: settings.SystemPrompt,
		Options:      settings.Options,
	}
}

// Exhaustively sends request to API
func sendRequestExhaustive(
	ctx context.Context,
	request *OllamaRequest,
) (string, error) {
	var text string
	var err error
	for i := range MAX_SEND_TRY {
		text, err = sendRequest(ctx, request)
		if err == nil {
			break
		}
		log.Printf("Failed send try %d: %v", i, err)
		time.Sleep(RETRY_TIME * time.Duration(1<<(i+1)))
	}

	return text, err
}

// Send Ollama request
func sendRequest(ctx context.Context, request *OllamaRequest) (string, error) {
	// Encode request body to JSON data
	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", err
	}

	// Make new POST request with JSON data
	req, err := http.NewRequestWithContext(ctx, "POST", OLLAMA_API, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("Failed to make request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Set HTTP client
	client := &http.Client{Timeout: API_TIMEOUT}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check status; print status code of error if any
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Status %d: %s", resp.StatusCode, string(body))
	}

	// Decode response body
	var ollamaResp OllamaResponse
	err = json.NewDecoder(resp.Body).Decode(&ollamaResp)
	if err != nil {
		return "", err
	}

	if !ollamaResp.Done {
		return "", fmt.Errorf("Failed to await request completion")
	}

	return ollamaResp.Response, nil
}
