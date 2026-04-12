package api2key

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSpendCreditsRequiresProjectID(t *testing.T) {
	client := NewClient(WithServiceSecret("secret"))

	_, err := client.SpendCredits(context.Background(), SpendCreditsRequest{
		UserID:  "user-1",
		Amount:  10,
		Service: "translation",
	})
	if err == nil || err.Error() != "project id is required" {
		t.Fatalf("expected missing project id error, got %v", err)
	}
}

func TestReserveCreditsSendsProjectID(t *testing.T) {
	t.Parallel()

	type requestBody struct {
		ProjectID string `json:"projectId"`
		UserID    string `json:"userId"`
		TaskID    string `json:"taskId"`
		Service   string `json:"service"`
		Amount    int    `json:"amount"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if got := r.Header.Get("X-Service-Secret"); got != "secret" {
			t.Fatalf("unexpected service secret: %s", got)
		}

		var payload requestBody
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if payload.ProjectID != "ytb2bili" {
			t.Fatalf("unexpected project id: %s", payload.ProjectID)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":200,"message":"ok","data":{"reservation_id":"res_123","projectId":"ytb2bili","scopeType":"project"}}`))
	}))
	defer server.Close()

	client := NewClient(
		WithBaseAPIURL(server.URL),
		WithServiceSecret("secret"),
	)

	result, err := client.ReserveCredits(context.Background(), ReserveCreditsRequest{
		ProjectID: "ytb2bili",
		UserID:    "user-1",
		TaskID:    "task-1",
		Service:   "translation",
		Amount:    10,
	})
	if err != nil {
		t.Fatalf("ReserveCredits returned error: %v", err)
	}
	if result.ProjectID != "ytb2bili" {
		t.Fatalf("unexpected response project id: %s", result.ProjectID)
	}
	if result.ScopeType != "project" {
		t.Fatalf("unexpected scope type: %s", result.ScopeType)
	}
	if result.ReservationID != "res_123" {
		t.Fatalf("unexpected reservation id: %s", result.ReservationID)
	}
}
