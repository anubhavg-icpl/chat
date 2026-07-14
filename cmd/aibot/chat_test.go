package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestChatClient_RequestBody verifies the request body is built correctly:
// model and messages are marshaled, and the system prompt + user message are
// sent in order. It also confirms the URL, method, and headers.
func TestChatClient_RequestBody(t *testing.T) {
	var got chatRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/v1/chat/completions", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "Bearer secret-key", r.Header.Get("Authorization"))

		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.NoError(t, json.Unmarshal(body, &got))

		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"ok"}}]}`))
	}))
	defer srv.Close()

	c := NewChatClient(srv.URL+"/v1", "secret-key", "gpt-test")
	reply, err := c.Complete(context.Background(), []Message{
		{Role: "system", Content: "sys"},
		{Role: "user", Content: "hi"},
	})

	assert.NoError(t, err)
	assert.Equal(t, "ok", reply)
	assert.Equal(t, "gpt-test", got.Model)
	assert.Equal(t, []Message{
		{Role: "system", Content: "sys"},
		{Role: "user", Content: "hi"},
	}, got.Messages)
}

// TestChatClient_ResponseParsing drives a canned Chat Completions JSON through
// the parser and asserts the assistant content is extracted.
func TestChatClient_ResponseParsing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{
			"id": "chatcmpl-1",
			"choices": [
				{"index": 0, "message": {"role": "assistant", "content": "Hello, friend!"}, "finish_reason": "stop"}
			]
		}`))
	}))
	defer srv.Close()

	c := NewChatClient(srv.URL, "k", "m")
	reply, err := c.Complete(context.Background(), []Message{{Role: "user", Content: "hi"}})

	assert.NoError(t, err)
	assert.Equal(t, "Hello, friend!", reply)
}

// TestChatClient_NoChoices asserts an empty choices array surfaces as an error.
func TestChatClient_NoChoices(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[]}`))
	}))
	defer srv.Close()

	c := NewChatClient(srv.URL, "k", "m")
	_, err := c.Complete(context.Background(), []Message{{Role: "user", Content: "x"}})
	assert.Error(t, err)
}

// TestChatClient_HTTPError asserts a non-2xx response surfaces as an error.
func TestChatClient_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"message":"boom"}}`))
	}))
	defer srv.Close()

	c := NewChatClient(srv.URL, "k", "m")
	_, err := c.Complete(context.Background(), []Message{{Role: "user", Content: "x"}})
	assert.Error(t, err)
}

// TestChatClient_TrailingSlashBaseURL asserts the endpoint URL tolerates a
// trailing slash on the configured base URL.
func TestChatClient_TrailingSlashBaseURL(t *testing.T) {
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"x"}}]}`))
	}))
	defer srv.Close()

	c := &ChatClient{BaseURL: srv.URL + "/", APIKey: "k", Model: "m", HTTP: http.DefaultClient}
	_, err := c.Complete(context.Background(), []Message{{Role: "user", Content: "x"}})
	assert.NoError(t, err)
	assert.Equal(t, "/chat/completions", path)
}
