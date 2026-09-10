package client

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/t12e/redan-plugin/internal/auth"
	"github.com/t12e/redan-plugin/internal/models"
)

const DefaultAPIBase = "http://127.0.0.1:8205"

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
	store      *auth.Store
}

type APIError struct {
	StatusCode int
	Code       string
	Message    string
	RetryAfter string
}

func (e *APIError) Error() string {
	if e.RetryAfter != "" {
		return fmt.Sprintf("Redan API error (%s, HTTP %d): %s; retry-after: %s", e.Code, e.StatusCode, e.Message, e.RetryAfter)
	}
	return fmt.Sprintf("Redan API error (%s, HTTP %d): %s", e.Code, e.StatusCode, e.Message)
}

func NewAuthenticated() (*Client, error) {
	store := auth.NewStore()
	credential, err := store.Load()
	if err != nil {
		return nil, err
	}
	baseURL := strings.TrimRight(strings.TrimSpace(credential.APIURL), "/")
	if baseURL == "" {
		baseURL = DefaultAPIBase
	}
	if err := validateBaseURL(baseURL); err != nil {
		return nil, err
	}
	return &Client{baseURL: baseURL, token: credential.Token, httpClient: &http.Client{Timeout: 30 * time.Second}, store: store}, nil
}

func NewUnauthenticated(baseURL string) (*Client, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = DefaultAPIBase
	}
	if err := validateBaseURL(baseURL); err != nil {
		return nil, err
	}
	return &Client{baseURL: baseURL, httpClient: &http.Client{Timeout: 30 * time.Second}}, nil
}

func Login(ctx context.Context, baseURL, email, password string) (models.LoginResponse, error) {
	api, err := NewUnauthenticated(baseURL)
	if err != nil {
		return models.LoginResponse{}, err
	}
	var result models.LoginResponse
	err = api.do(ctx, http.MethodPost, "/api/redan/auth/login", nil, map[string]string{"email": email, "password": password}, &result)
	return result, err
}

func (c *Client) Status(ctx context.Context) (models.AuthStatusResponse, error) {
	var result models.AuthStatusResponse
	err := c.do(ctx, http.MethodGet, "/api/redan/auth/status", nil, nil, &result)
	return result, err
}

func (c *Client) Logout(ctx context.Context) error {
	return c.do(ctx, http.MethodPost, "/api/redan/auth/logout", nil, nil, nil)
}

func validateBaseURL(baseURL string) error {
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return errors.New("Redan API URL must be an absolute URL")
	}
	return nil
}

func NewWithHTTPClient(baseURL, token string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), token: token, httpClient: httpClient}
}

func (c *Client) ListCategories(ctx context.Context, search string, page, perPage int) (models.CategoryPage, error) {
	var result models.CategoryPage
	values := url.Values{}
	if search != "" {
		values.Set("search", search)
	}
	if page > 0 {
		values.Set("page", strconv.Itoa(page))
	}
	if perPage > 0 {
		values.Set("per_page", strconv.Itoa(perPage))
	}
	err := c.do(ctx, http.MethodGet, "/api/redan/faq/categories", values, nil, &result)
	return result, err
}
func (c *Client) GetCategory(ctx context.Context, id string) (models.CategoryResponse, error) {
	var result models.CategoryResponse
	err := c.do(ctx, http.MethodGet, "/api/redan/faq/categories/"+url.PathEscape(id), nil, nil, &result)
	return result, err
}
func (c *Client) CreateCategory(ctx context.Context, id, name string) (models.CategoryResponse, error) {
	var result models.CategoryResponse
	payload := map[string]string{"name": name}
	if id != "" {
		payload["id"] = id
	}
	err := c.do(ctx, http.MethodPost, "/api/redan/faq/categories", nil, payload, &result)
	return result, err
}
func (c *Client) UpdateCategory(ctx context.Context, id, name string) (models.CategoryResponse, error) {
	var result models.CategoryResponse
	err := c.do(ctx, http.MethodPatch, "/api/redan/faq/categories/"+url.PathEscape(id), nil, map[string]string{"name": name}, &result)
	return result, err
}
func (c *Client) DeleteCategory(ctx context.Context, id string) (models.DeleteResponse, error) {
	var result models.DeleteResponse
	err := c.do(ctx, http.MethodDelete, "/api/redan/faq/categories/"+url.PathEscape(id), nil, nil, &result)
	return result, err
}
func (c *Client) ListFAQs(ctx context.Context, search, category string, page, perPage int) (models.FAQPage, error) {
	var result models.FAQPage
	values := url.Values{}
	if search != "" {
		values.Set("search", search)
	}
	if category != "" {
		values.Set("category", category)
	}
	if page > 0 {
		values.Set("page", strconv.Itoa(page))
	}
	if perPage > 0 {
		values.Set("per_page", strconv.Itoa(perPage))
	}
	err := c.do(ctx, http.MethodGet, "/api/redan/faq/faqs", values, nil, &result)
	return result, err
}
func (c *Client) GetFAQ(ctx context.Context, id string) (models.FAQResponse, error) {
	var result models.FAQResponse
	err := c.do(ctx, http.MethodGet, "/api/redan/faq/faqs/"+url.PathEscape(id), nil, nil, &result)
	return result, err
}
func (c *Client) CreateFAQ(ctx context.Context, payload map[string]any) (models.FAQResponse, error) {
	var result models.FAQResponse
	err := c.do(ctx, http.MethodPost, "/api/redan/faq/faqs", nil, payload, &result)
	return result, err
}
func (c *Client) UpdateFAQ(ctx context.Context, id string, payload map[string]any) (models.FAQResponse, error) {
	var result models.FAQResponse
	err := c.do(ctx, http.MethodPatch, "/api/redan/faq/faqs/"+url.PathEscape(id), nil, payload, &result)
	return result, err
}
func (c *Client) DeleteFAQ(ctx context.Context, id string) (models.DeleteResponse, error) {
	var result models.DeleteResponse
	err := c.do(ctx, http.MethodDelete, "/api/redan/faq/faqs/"+url.PathEscape(id), nil, nil, &result)
	return result, err
}

func (c *Client) do(ctx context.Context, method, path string, query url.Values, payload any, output any) error {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
		body = bytes.NewReader(encoded)
	}
	u, err := url.Parse(c.baseURL + path)
	if err != nil {
		return fmt.Errorf("build request URL: %w", err)
	}
	if query != nil {
		u.RawQuery = query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	requestID := make([]byte, 16)
	if _, err := rand.Read(requestID); err != nil {
		return fmt.Errorf("create request ID: %w", err)
	}
	req.Header.Set("X-Request-ID", hex.EncodeToString(requestID))
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request Redan API: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return fmt.Errorf("read Redan API response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var envelope struct {
			Error   struct{ Code, Message string } `json:"error"`
			Message string                         `json:"message"`
		}
		_ = json.Unmarshal(data, &envelope)
		message := envelope.Error.Message
		if message == "" {
			message = envelope.Message
		}
		if message == "" {
			message = strings.TrimSpace(string(data))
		}
		if message == "" {
			message = http.StatusText(resp.StatusCode)
		}
		message = strings.ReplaceAll(message, c.token, "[redacted]")
		code := envelope.Error.Code
		if code == "" {
			code = fmt.Sprintf("HTTP_%d", resp.StatusCode)
		}
		if resp.StatusCode == http.StatusUnauthorized && c.store != nil {
			_ = c.store.Delete()
		}
		return &APIError{StatusCode: resp.StatusCode, Code: code, Message: message, RetryAfter: resp.Header.Get("Retry-After")}
	}
	if output == nil || len(bytes.TrimSpace(data)) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, output); err != nil {
		return fmt.Errorf("decode Redan API response: %w", err)
	}
	return nil
}
