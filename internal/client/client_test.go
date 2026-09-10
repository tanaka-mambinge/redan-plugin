package client

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListFAQsSendsBearerTokenAndQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Errorf("authorization header = %q", r.Header.Get("Authorization"))
		}
		if r.URL.Query().Get("search") != "fuel" || r.URL.Query().Get("category") != "cards" {
			t.Errorf("query = %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[],"meta":{"current_page":1,"per_page":20,"total":0,"last_page":1},"links":{"self":"http://example.test"}}`))
	}))
	defer server.Close()

	result, err := NewWithHTTPClient(server.URL, "secret", server.Client()).ListFAQs(context.Background(), "fuel", "cards", 0, 20)
	if err != nil {
		t.Fatal(err)
	}
	if result.Meta.Total != 0 {
		t.Fatalf("total = %d", result.Meta.Total)
	}
}

func TestAPIErrorPreservesRetryAfterWithoutLeakingToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "12")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":{"code":"RATE_LIMITED","message":"slow down super-secret-token"}}`))
	}))
	defer server.Close()

	_, err := NewWithHTTPClient(server.URL, "super-secret-token", server.Client()).ListCategories(context.Background(), "", 0, 20)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %v", err)
	}
	if apiErr.StatusCode != http.StatusTooManyRequests || apiErr.RetryAfter != "12" {
		t.Fatalf("api error = %+v", apiErr)
	}
	if strings.Contains(err.Error(), "super-secret-token") {
		t.Fatal("API token leaked in error")
	}
}
