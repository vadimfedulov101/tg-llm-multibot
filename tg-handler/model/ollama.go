package model

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"tg-handler/conf"
	"tg-handler/logging"
)

// Ollama errors
var (
	ErrCtxDone           = errors.New("context done")
	errMarshalFailed     = errors.New("marshal request failed")
	errRequestFailed     = errors.New("create request failed")
	errSendFailed        = errors.New("send request failed")
	errInvalidStatus     = errors.New("invalid status code")
	errDecodeFailed      = errors.New("decode response failed")
	errRequestIncomplete = errors.New("request not completed")
)

// Request to Ollama
type Request struct {
	Model        string                `json:"model"`
	Prompt       string                `json:"prompt"`
	Stream       bool                  `json:"stream"`
	SystemPrompt string                `json:"system,omitempty"`
	Options      conf.OptionalSettings `json:"options"`
	Context      []int                 `json:"context,omitempty"`
	cleaner      func(string) string
}

func newRequest(
	prompt string,
	model string,
	botConf *conf.BotConf,
	cleaner func(string) string,
) *Request {
	return &Request{
		Model:        model,
		Prompt:       prompt,
		Stream:       false,
		SystemPrompt: botConf.Main.Role,
		Options:      botConf.Optional,
		cleaner:      cleaner,
	}
}

// Response from Ollama
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

// Eternally sends request to API and logs error
func sendRequestEternal(
	ctx context.Context,
	request *Request,
	logger *logging.Logger,
) (string, error) {
	var (
		text string
		err  error
	)

	// Get text
	for {
		// Check if parent context (shutdown is done before trying)
		if ctx.Err() != nil {
			return "", ErrCtxDone
		}

		text, err = sendRequest(ctx, request, logger)
		if err == nil {
			break
		}

		logger.Error("request failed retrying", logging.Err(err))

		select {
		case <-time.After(retryTime):
			continue
		case <-ctx.Done():
			return "", ErrCtxDone
		}
	}

	return text, nil
}

// Sends Ollama request
func sendRequest(
	ctx context.Context,
	request *Request,
	logger *logging.Logger,
) (string, error) {
	// Encode request body to JSON data
	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("%w: %v", errMarshalFailed, err)
	}

	// Create context with timeout for this request
	// to drop connection if response takes too long
	reqCtx, cancel := context.WithTimeout(ctx, waitTimeout)
	defer cancel()

	// Make POST request with JSON data
	req, err := http.NewRequestWithContext(
		reqCtx, "POST", apiUrl, bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return "", fmt.Errorf("%w: %v", errRequestFailed, err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Set HTTP client
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: %v", errSendFailed, err)
	}
	defer resp.Body.Close()

	// Validate status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf(
			"%w %d: %s",
			errInvalidStatus, resp.StatusCode, string(body),
		)
	}

	// Decode response body
	var response Response
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return "", fmt.Errorf("%w: %v", errDecodeFailed, err)
	}

	// Validate request completeness
	if !response.Done {
		return "", errRequestIncomplete
	}

	// Log raw response
	logger.Debug(
		"raw response", logging.RawResponse(response.Response),
	)

	// Clean response
	response.Response = request.cleaner(response.Response)

	return response.Response, nil
}
