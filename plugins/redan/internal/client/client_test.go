package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/t12e/redan-plugin/internal/models"
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

func TestListFormsSendsTypeAndSearchQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/redan/forms" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.URL.Query().Get("type") != "quote" || r.URL.Query().Get("search") != "fuel" {
			t.Errorf("query = %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[],"meta":{"current_page":1,"per_page":20,"total":0,"last_page":1},"links":{"self":"http://example.test"}}`))
	}))
	defer server.Close()

	result, err := NewWithHTTPClient(server.URL, "secret", server.Client()).ListForms(context.Background(), "fuel", "quote", 0, 20)
	if err != nil {
		t.Fatal(err)
	}
	if result.Meta.Total != 0 {
		t.Fatalf("total = %d", result.Meta.Total)
	}
}

func TestUpdateFormSendsActiveAndLocalizedFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/api/redan/forms/bulk-fuel-quotation" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload["active"] != false {
			t.Fatalf("active = %#v", payload["active"])
		}
		fields, ok := payload["fields"].([]any)
		if !ok || len(fields) != 1 {
			t.Fatalf("fields = %#v", payload["fields"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"key":"bulk-fuel-quotation","type":"quote","name":"Quotation","version":2,"active":false,"fields":[],"field_count":0,"links":{"admin":"http://example.test/admin/quotation-forms/bulk-fuel-quotation/edit"}}}`))
	}))
	defer server.Close()

	result, err := NewWithHTTPClient(server.URL, "secret", server.Client()).UpdateForm(context.Background(), "bulk-fuel-quotation", map[string]any{
		"active": false,
		"fields": []models.FormField{{
			Type:   "text",
			Labels: map[string]string{"en": "Company", "sn": "Kambani", "nd": "Inkampani"},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Data.Version != 2 || result.Data.Active {
		t.Fatalf("result = %+v", result.Data)
	}
}

func TestLoginUsesCredentialsAndParsesTemporarySession(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/redan/auth/login" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		var payload map[string]string
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload["email"] != "admin@example.test" || payload["password"] != "secret" {
			t.Fatalf("credentials = %#v", payload)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"token":"temporary-token","expires_at":"2030-01-01T00:00:00Z","user":{"id":4,"name":"Admin","email":"admin@example.test"}}}`))
	}))
	defer server.Close()

	result, err := Login(context.Background(), server.URL, "admin@example.test", "secret")
	if err != nil {
		t.Fatal(err)
	}
	if result.Data.Token != "temporary-token" || result.Data.User.ID != 4 {
		t.Fatalf("login result = %+v", result.Data)
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
