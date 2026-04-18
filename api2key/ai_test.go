package api2key

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListAIModelsUsesProjectQueryAndAuthHeader(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/api/v1/ai/models" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("projectId"); got != "proj-1" {
			t.Fatalf("unexpected project id: %s", got)
		}
		if got := r.URL.Query().Get("onlyAvailable"); got != "true" {
			t.Fatalf("unexpected onlyAvailable: %s", got)
		}
		if got := r.Header.Get("x-api-key"); got != "ak-test" {
			t.Fatalf("unexpected api key header: %s", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":200,"message":"success","data":{"userTier":"pro","currentProject":{"id":"proj-1","name":"Demo"},"models":[{"key":"openai/gpt-4o-mini","id":"openai/gpt-4o-mini","name":"GPT-4o mini","modelId":"openai/gpt-4o-mini","provider":"OpenAI","modelType":"chat","minTier":"free","isFeatured":true,"locked":false}],"grouped":{"OpenAI":[{"key":"openai/gpt-4o-mini","id":"openai/gpt-4o-mini","name":"GPT-4o mini","modelId":"openai/gpt-4o-mini","provider":"OpenAI","modelType":"chat","minTier":"free","isFeatured":true,"locked":false}]}}}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseAPIURL(server.URL))
	result, err := client.ListAIModels(context.Background(), ListAIModelsRequest{
		ProjectID:     "proj-1",
		APIKey:        "ak-test",
		OnlyAvailable: true,
	})
	if err != nil {
		t.Fatalf("ListAIModels returned error: %v", err)
	}
	if result.UserTier != "pro" {
		t.Fatalf("unexpected user tier: %s", result.UserTier)
	}
	if result.CurrentProject.ID != "proj-1" {
		t.Fatalf("unexpected current project id: %s", result.CurrentProject.ID)
	}
	if len(result.Models) != 1 {
		t.Fatalf("unexpected models count: %d", len(result.Models))
	}
	if result.Models[0].Name != "GPT-4o mini" {
		t.Fatalf("unexpected model name: %s", result.Models[0].Name)
	}
	if len(result.Grouped["OpenAI"]) != 1 {
		t.Fatalf("unexpected grouped models count: %d", len(result.Grouped["OpenAI"]))
	}
}

func TestChatCompletionsUsesRawJSONResponse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/api/v1/ai/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("projectId"); got != "proj-2" {
			t.Fatalf("unexpected project id: %s", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer token-123" {
			t.Fatalf("unexpected auth header: %s", got)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		if !strings.Contains(string(body), `"model":"openai/gpt-4o-mini"`) {
			t.Fatalf("unexpected request body: %s", string(body))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Model", "GPT-4o mini")
		_, _ = w.Write([]byte(`{"id":"chatcmpl_123","choices":[{"message":{"role":"assistant","content":"hello"}}]}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseAPIURL(server.URL))
	response, err := client.ChatCompletions(context.Background(), AIRequest{
		ProjectID:   "proj-2",
		AccessToken: "token-123",
		Body: map[string]any{
			"model":    "openai/gpt-4o-mini",
			"messages": []map[string]string{{"role": "user", "content": "hi"}},
		},
	})
	if err != nil {
		t.Fatalf("ChatCompletions returned error: %v", err)
	}
	if got := response.Headers.Get("X-Model"); got != "GPT-4o mini" {
		t.Fatalf("unexpected response header: %s", got)
	}
	var decoded struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := response.Decode(&decoded); err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	if len(decoded.Choices) != 1 || decoded.Choices[0].Message.Content != "hello" {
		t.Fatalf("unexpected decoded response: %+v", decoded)
	}
}

func TestChatCompletionsStreamReturnsReadableBody(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"he\"}}]}\n\n"))
	}))
	defer server.Close()

	client := NewClient(WithBaseAPIURL(server.URL))
	stream, err := client.ChatCompletionsStream(context.Background(), AIRequest{
		APIKey: "ak-stream",
		Body:   map[string]any{"model": "openai/gpt-4o-mini", "stream": true},
	})
	if err != nil {
		t.Fatalf("ChatCompletionsStream returned error: %v", err)
	}
	defer stream.Body.Close()

	raw, err := io.ReadAll(stream.Body)
	if err != nil {
		t.Fatalf("read stream body: %v", err)
	}
	if stream.Headers.Get("Content-Type") != "text/event-stream" {
		t.Fatalf("unexpected content type: %s", stream.Headers.Get("Content-Type"))
	}
	if !strings.Contains(string(raw), `"content":"he"`) {
		t.Fatalf("unexpected stream body: %s", string(raw))
	}
}

func TestGoogleGenerateContentEscapesModelPath(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.RequestURI, "/api/v1/ai/google/v1beta/models/google-ai-studio%2Fgemini-2.5-pro:generateContent") {
			t.Fatalf("unexpected request uri: %s", r.RequestURI)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"ok"}]}}]}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseAPIURL(server.URL))
	response, err := client.GoogleGenerateContent(context.Background(), "google-ai-studio/gemini-2.5-pro", AIRequest{
		APIKey: "ak-google",
		Body: map[string]any{
			"contents": []map[string]any{{
				"parts": []map[string]string{{"text": "hello"}},
			}},
		},
	})
	if err != nil {
		t.Fatalf("GoogleGenerateContent returned error: %v", err)
	}
	if len(response.Body) == 0 {
		t.Fatal("expected non-empty response body")
	}
}

func TestAISessionAppliesDefaultCredentials(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("projectId"); got != "proj-session" {
			t.Fatalf("unexpected project id: %s", got)
		}
		if got := r.Header.Get("x-api-key"); got != "ak-session" {
			t.Fatalf("unexpected api key header: %s", got)
		}
		if got := r.Header.Get("x-trace-id"); got != "trace-1" {
			t.Fatalf("unexpected trace header: %s", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"chatcmpl_session"}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseAPIURL(server.URL))
	session := NewAISession(
		client,
		WithAISessionProjectID("proj-session"),
		WithAISessionAPIKey("ak-session"),
		WithAISessionHeader("x-trace-id", "trace-1"),
	)

	response, err := session.ChatCompletions(context.Background(), map[string]any{
		"model":    "openai/gpt-4o-mini",
		"messages": []map[string]string{{"role": "user", "content": "hello"}},
	})
	if err != nil {
		t.Fatalf("session.ChatCompletions returned error: %v", err)
	}
	if string(response.Body) != `{"id":"chatcmpl_session"}` {
		t.Fatalf("unexpected response body: %s", string(response.Body))
	}
}
