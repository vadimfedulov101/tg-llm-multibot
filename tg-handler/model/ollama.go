package model

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"tg-handler/conf"
)

// Ollama errors
var (
	ErrLoadEnvFailed = errors.New(
		"[model] failed to load $LLM_MODEL from environment",
	)
)

// Environment constant
var model = getEnv("LLM_MODEL", ErrLoadEnvFailed)

func getEnv(key string, err error) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	log.Fatal(err)
	return ""
}

// Request to Ollama
type Request struct {
	Model        string                `json:"model"`
	Prompt       string                `json:"prompt"`
	Stream       bool                  `json:"stream"`
	Options      conf.OptionalSettings `json:"options"`
	SystemPrompt string                `json:"system,omitempty"`
	Context      []int                 `json:"context,omitempty"`
}

func newRequest(prompt string, botConf *conf.BotConf) *Request {
	return &Request{
		Model:        model, // Loaded from environment
		Prompt:       prompt,
		Stream:       false,
		SystemPrompt: botConf.Main.Role,
		Options:      botConf.Optional,
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

// Ollama errors
var (
	ErrMarshalFailed     = errors.New("[model] marshal request failed")
	ErrRequestFailed     = errors.New("[model] create request failed")
	ErrSendFailed        = errors.New("[model] send request failed")
	ErrInvalidStatus     = errors.New("[model] invalid status code")
	ErrDecodeFailed      = errors.New("[model] decode response failed")
	ErrRequestIncomplete = errors.New("[model] request not completed")
)

// Eternally sends request to API and logs error
func sendRequestEternal(ctx context.Context, request *Request) string {
	var (
		text string
		err  error
	)

	// Get text
	for {
		text, err = sendRequest(ctx, request)
		if err == nil {
			break
		}
		log.Printf("%v: %v", ErrSendFailed, err)
		time.Sleep(retryTime)
	}

	// Clean text
	log.Println("Raw text:", text)
	text = trimNoise(text)
	log.Println("Cleaned text:", text)

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
	req, err := http.NewRequestWithContext(
		ctx, "POST", apiUrl, bytes.NewBuffer(jsonData),
	)
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
		return "", fmt.Errorf(
			"%w %d: %s", ErrInvalidStatus, resp.StatusCode, string(body),
		)
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
