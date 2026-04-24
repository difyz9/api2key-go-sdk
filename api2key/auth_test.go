package api2key

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLoginAllowsOmittedProjectID(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if got := r.Header.Get("X-Project-Id"); got != "" {
			t.Fatalf("unexpected project header: %s", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":200,"message":"ok","data":{"accessToken":"token-123","expiresIn":3600}}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseAPIURL(server.URL))
	result, err := client.Login(context.Background(), LoginRequest{
		Email:    "user@example.com",
		Password: "password",
	})
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}
	if result.AccessToken != "token-123" {
		t.Fatalf("unexpected access token: %s", result.AccessToken)
	}
}

func TestLoginSendsProjectHeader(t *testing.T) {
	t.Parallel()

	type requestBody struct {
		Email     string `json:"email"`
		Password  string `json:"password"`
		ProjectID string `json:"projectId,omitempty"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if got := r.Header.Get("X-Project-Id"); got != "ytb2bili" {
			t.Fatalf("unexpected project header: %s", got)
		}

		var payload requestBody
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if payload.ProjectID != "ytb2bili" {
			t.Fatalf("unexpected body project id: %s", payload.ProjectID)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":200,"message":"ok","data":{"accessToken":"token-123","expiresIn":3600}}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseAPIURL(server.URL))
	result, err := client.Login(context.Background(), LoginRequest{
		Email:     "user@example.com",
		Password:  "password",
		ProjectID: "ytb2bili",
	})
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}
	if result.AccessToken != "token-123" {
		t.Fatalf("unexpected access token: %s", result.AccessToken)
	}
}
