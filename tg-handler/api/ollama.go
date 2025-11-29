package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// Request to Ollama API
type Request struct {
	Model        string  `json:"model"`
	Prompt       string  `json:"prompt"`
	Stream       bool    `json:"stream"`
	Options      Options `json:"options"`
	SystemPrompt string  `json:"system,omitempty"`
	Context      []int   `json:"context,omitempty"`
}

func newRequest(prompt string, settings *Settings) *Request {
	return &Request{
		Model:        model,
		Prompt:       prompt,
		Stream:       false,
		SystemPrompt: settings.BotConf.SystemPrompt,
		Options:      settings.Options,
	}
}

// Response from Ollama API
type Response struct {
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

// API errors
var (
	ErrMarshalFailed     = errors.New("[api] marshal request failed")
	ErrRequestFailed     = errors.New("[api] create request failed")
	ErrSendFailed        = errors.New("[api] send request failed")
	ErrInvalidStatus     = errors.New("[api] invalid status code")
	ErrDecodeFailed      = errors.New("[api] decode response failed")
	ErrRequestIncomplete = errors.New("[api] request not completed")
)

// Eternally sends request to API and logs error
func sendRequestEternal(
	ctx context.Context,
	request *Request,
) (text string) {
	var err error
	for {
		text, err = sendRequest(ctx, request)
		if err == nil {
			break
		}
		log.Printf("Failed send: %v\n", err)
		time.Sleep(retryTime)
	}

	return text
}

// Sends Ollama request
func sendRequest(ctx context.Context, request *Request) (string, error) {
	// Encode request body to JSON data
	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrMarshalFailed, err)
	}

	// Make POST request with JSON data
	req, err := http.NewRequestWithContext(ctx, "POST", apiUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrRequestFailed, err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Set HTTP client
	client := &http.Client{Timeout: waitTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrSendFailed, err)
	}
	defer resp.Body.Close()

	// Validate status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("%w %d: %s", ErrInvalidStatus, resp.StatusCode, string(body))
	}

	// Decode response body
	var response Response
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrDecodeFailed, err)
	}

	// Validate request completeness
	if !response.Done {
		return "", ErrRequestIncomplete
	}

	return response.Response, nil
}
