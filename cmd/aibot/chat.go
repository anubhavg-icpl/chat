package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Message is a single chat message in the OpenAI Chat Completions format.
// Role is "system", "user", or "assistant".
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatClient calls an OpenAI-compatible Chat Completions endpoint (OpenAI,
// OpenRouter, Ollama, LM Studio, etc.) using only the standard library.
type ChatClient struct {
	BaseURL string
	APIKey  string
	Model   string
	HTTP    *http.Client
}

// NewChatClient creates a ChatClient with a default HTTP client.
func NewChatClient(baseURL, apiKey, model string) *ChatClient {
	return &ChatClient{
		BaseURL: baseURL,
		APIKey:  apiKey,
		Model:   model,
		HTTP:    &http.Client{Timeout: 60 * time.Second},
	}
}

// chatRequest is the request body POSTed to {baseURL}/chat/completions.
type chatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

// chatResponse captures only the fields needed from the Chat Completions
// response: choices[0].message.content.
type chatResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
}

// Complete sends the conversation to the completions endpoint and returns the
// assistant's reply text. It returns an error for transport failures, non-2xx
// responses, malformed JSON, or an empty choices array.
func (c *ChatClient) Complete(ctx context.Context, messages []Message) (string, error) {
	payload, err := json.Marshal(chatRequest{Model: c.Model, Messages: messages})
	if err != nil {
		return "", fmt.Errorf("encode request: %w", err)
	}

	url := strings.TrimRight(c.BaseURL, "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("chat completions: HTTP %d: %s", resp.StatusCode, body)
	}

	var cr chatResponse
	if err := json.Unmarshal(body, &cr); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	if len(cr.Choices) == 0 {
		return "", errors.New("chat completions: response contained no choices")
	}
	return cr.Choices[0].Message.Content, nil
}
