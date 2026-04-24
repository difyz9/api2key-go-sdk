package api2key

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetCreditsBalanceWithAPIKey(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if got := r.Header.Get("x-api-key"); got != "sk-test-123" {
			t.Fatalf("unexpected x-api-key: %s", got)
		}
		if got := r.Header.Get("Authorization"); got != "" {
			t.Fatalf("unexpected authorization header: %s", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":200,"message":"ok","data":{"balance":120,"reserved":5,"account":{"balance":120,"reserved":5},"scope":{"type":"project","projectId":"ytb2bili"}}}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseAPIURL(server.URL))
	result, err := client.GetCreditsBalanceWithOptions(context.Background(), GetCreditsBalanceRequest{
		APIKey: "sk-test-123",
	})
	if err != nil {
		t.Fatalf("GetCreditsBalanceWithOptions returned error: %v", err)
	}
	if result.Balance != 120 {
		t.Fatalf("unexpected balance: %d", result.Balance)
	}
}

func TestGetLedgerWithAPIKeyAndProjectID(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if got := r.Header.Get("x-api-key"); got != "sk-test-123" {
			t.Fatalf("unexpected x-api-key: %s", got)
		}
		if !strings.Contains(r.URL.RawQuery, "projectId=ytb2bili") {
			t.Fatalf("unexpected query: %s", r.URL.RawQuery)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":200,"message":"ok","data":{"list":[],"pagination":{"page":1,"size":10,"total":0,"totalPages":0}}}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseAPIURL(server.URL))
	result, err := client.GetLedger(context.Background(), GetLedgerRequest{
		APIKey:    "sk-test-123",
		Page:      1,
		Size:      10,
		ProjectID: "ytb2bili",
	})
	if err != nil {
		t.Fatalf("GetLedger returned error: %v", err)
	}
	if result.Pagination.Page != 1 {
		t.Fatalf("unexpected page: %d", result.Pagination.Page)
	}
}

func TestDeductCreditsRequiresAuth(t *testing.T) {
	client := NewClient()

	_, err := client.DeductCredits(context.Background(), DeductCreditsRequest{
		Amount:  10,
		Service: "translation",
	})
	if err == nil || err.Error() != "access token or api key is required" {
		t.Fatalf("expected missing auth error, got %v", err)
	}
}

func TestDeductCreditsSendsAPIKey(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if got := r.Header.Get("x-api-key"); got != "sk-test-123" {
			t.Fatalf("unexpected x-api-key: %s", got)
		}
		if !strings.Contains(r.URL.Path, "/credits/deduct") {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":200,"message":"ok","data":{"balanceAfter":90,"projectId":"ytb2bili","scopeType":"project"}}`))
	}))
	defer server.Close()

	client := NewClient(WithBaseAPIURL(server.URL))

	result, err := client.DeductCredits(context.Background(), DeductCreditsRequest{
		APIKey:      "sk-test-123",
		Amount:      10,
		Service:     "translation",
		TaskID:      "task-1",
		Description: "sdk test",
	})
	if err != nil {
		t.Fatalf("DeductCredits returned error: %v", err)
	}
	if result.ProjectID != "ytb2bili" {
		t.Fatalf("unexpected response project id: %s", result.ProjectID)
	}
	if result.ScopeType != "project" {
		t.Fatalf("unexpected scope type: %s", result.ScopeType)
	}
	if result.BalanceAfter != 90 {
		t.Fatalf("unexpected balance after: %d", result.BalanceAfter)
	}
}
