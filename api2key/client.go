package api2key

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	DefaultBaseAPIURL = "https://open.api2key.com"
	DefaultTTSURL     = "https://tts.api2key.com"
	DefaultAPIPrefix  = "/api/v1"
)

type Client struct {
	baseAPIURL    string
	ttsURL        string
	apiPrefix     string
	serviceSecret string
	httpClient    *http.Client
}

type Config struct {
	BaseAPIURL    string
	TTSURL        string
	APIPrefix     string
	ServiceSecret string
	HTTPClient    *http.Client
}

type Option func(*Config)

func NewClient(options ...Option) *Client {
	config := Config{
		BaseAPIURL: DefaultBaseAPIURL,
		APIPrefix:  DefaultAPIPrefix,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
	for _, option := range options {
		option(&config)
	}
	if config.HTTPClient == nil {
		config.HTTPClient = &http.Client{Timeout: 30 * time.Second}
	}
	baseAPIURL := strings.TrimRight(config.BaseAPIURL, "/")
	ttsURL := strings.TrimRight(strings.TrimSpace(config.TTSURL), "/")
	if ttsURL == "" {
		ttsURL = deriveServiceBaseURL(baseAPIURL)
	}
	return &Client{
		baseAPIURL:    baseAPIURL,
		ttsURL:        ttsURL,
		apiPrefix:     normalizeAPIPrefix(config.APIPrefix),
		serviceSecret: strings.TrimSpace(config.ServiceSecret),
		httpClient:    config.HTTPClient,
	}
}

func WithBaseAPIURL(rawURL string) Option {
	return func(config *Config) {
		config.BaseAPIURL = rawURL
	}
}

func WithTTSURL(rawURL string) Option {
	return func(config *Config) {
		config.TTSURL = rawURL
	}
}

func WithAPIPrefix(prefix string) Option {
	return func(config *Config) {
		config.APIPrefix = prefix
	}
}

func WithServiceSecret(secret string) Option {
	return func(config *Config) {
		config.ServiceSecret = secret
	}
}

func WithHTTPClient(httpClient *http.Client) Option {
	return func(config *Config) {
		config.HTTPClient = httpClient
	}
}

func normalizeAPIPrefix(prefix string) string {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return DefaultAPIPrefix
	}
	return "/" + strings.Trim(prefix, "/")
}

func deriveServiceBaseURL(baseAPIURL string) string {
	trimmed := strings.TrimSpace(baseAPIURL)
	if trimmed == "" {
		return DefaultBaseAPIURL
	}

	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Host == "" {
		return DefaultBaseAPIURL
	}

	derived := *parsed
	derived.RawPath = ""
	derived.Path = ""
	derived.RawQuery = ""
	derived.Fragment = ""

	return strings.TrimRight(derived.String(), "/")
}

type envelope[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

type APIError struct {
	StatusCode int
	Code       int
	Message    string
	RawBody    string
	Balance    *int
}

func (e *APIError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Balance != nil {
		return fmt.Sprintf("api2key request failed: status=%d code=%d message=%s balance=%d", e.StatusCode, e.Code, e.Message, *e.Balance)
	}
	if e.RawBody != "" {
		return fmt.Sprintf("api2key request failed: status=%d code=%d message=%s raw=%s", e.StatusCode, e.Code, e.Message, e.RawBody)
	}
	return fmt.Sprintf("api2key request failed: status=%d code=%d message=%s", e.StatusCode, e.Code, e.Message)
}

func (e *APIError) Is(target error) bool {
	t, ok := target.(*APIError)
	if !ok {
		return false
	}
	if t.StatusCode != 0 && t.StatusCode != e.StatusCode {
		return false
	}
	if t.Code != 0 && t.Code != e.Code {
		return false
	}
	return true
}

func IsStatus(err error, statusCode int) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	return apiErr.StatusCode == statusCode
}

type rawPayload struct {
	Balance *int `json:"balance,omitempty"`
}

func (c *Client) requestJSON(ctx context.Context, method, endpoint string, headers map[string]string, input any, output any) error {
	var body io.Reader
	if input != nil {
		raw, err := json.Marshal(input)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
		body = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	if input != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	var result envelope[json.RawMessage]
	if err := json.Unmarshal(raw, &result); err == nil && (result.Code != 0 || resp.Header.Get("Content-Type") == "application/json") {
		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			return buildAPIError(resp.StatusCode, result, raw)
		}
		if output != nil && len(result.Data) > 0 && string(result.Data) != "null" {
			if err := json.Unmarshal(result.Data, output); err != nil {
				return fmt.Errorf("decode response data: %w", err)
			}
		}
		return nil
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return &APIError{StatusCode: resp.StatusCode, Message: strings.TrimSpace(string(raw)), RawBody: string(raw)}
	}
	if output != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, output); err != nil {
			return fmt.Errorf("decode raw response: %w", err)
		}
	}
	return nil
}

func buildAPIError(statusCode int, result envelope[json.RawMessage], raw []byte) error {
	errResp := &APIError{
		StatusCode: statusCode,
		Code:       result.Code,
		Message:    result.Message,
		RawBody:    string(raw),
	}
	if len(result.Data) > 0 && string(result.Data) != "null" {
		var payload rawPayload
		if err := json.Unmarshal(result.Data, &payload); err == nil {
			errResp.Balance = payload.Balance
		}
	}
	if errResp.Message == "" {
		errResp.Message = http.StatusText(statusCode)
	}
	return errResp
}

func (c *Client) requestBinary(ctx context.Context, method, endpoint string, headers map[string]string, input any) ([]byte, http.Header, error) {
	var body io.Reader
	if input != nil {
		raw, err := json.Marshal(input)
		if err != nil {
			return nil, nil, fmt.Errorf("marshal request body: %w", err)
		}
		body = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, nil, fmt.Errorf("build request: %w", err)
	}
	if input != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("read response body: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		var result envelope[json.RawMessage]
		if err := json.Unmarshal(raw, &result); err == nil {
			return nil, nil, buildAPIError(resp.StatusCode, result, raw)
		}
		return nil, nil, &APIError{StatusCode: resp.StatusCode, Message: strings.TrimSpace(string(raw)), RawBody: string(raw)}
	}
	if strings.HasPrefix(strings.ToLower(resp.Header.Get("Content-Type")), "application/json") {
		var result envelope[json.RawMessage]
		if err := json.Unmarshal(raw, &result); err == nil {
			return nil, nil, buildAPIError(resp.StatusCode, result, raw)
		}
		return nil, nil, &APIError{StatusCode: resp.StatusCode, Message: "expected binary response but got json", RawBody: string(raw)}
	}
	return raw, resp.Header.Clone(), nil
}

func (c *Client) requestMultipart(ctx context.Context, endpoint string, headers map[string]string, fields map[string]string, fileField, filePath string, output any) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile(fileField, filepath.Base(filePath))
	if err != nil {
		return fmt.Errorf("create form file: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return fmt.Errorf("copy file data: %w", err)
	}
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			return fmt.Errorf("write form field %s: %w", key, err)
		}
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("close multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, &body)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Accept", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	var result envelope[json.RawMessage]
	if err := json.Unmarshal(raw, &result); err != nil {
		return fmt.Errorf("decode response envelope: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return buildAPIError(resp.StatusCode, result, raw)
	}
	if output != nil && len(result.Data) > 0 && string(result.Data) != "null" {
		if err := json.Unmarshal(result.Data, output); err != nil {
			return fmt.Errorf("decode response data: %w", err)
		}
	}
	return nil
}

func joinURL(baseURL string, elements ...string) string {
	baseURL = strings.TrimRight(baseURL, "/")
	parts := make([]string, 0, len(elements))
	for _, element := range elements {
		trimmed := strings.TrimSpace(element)
		if trimmed == "" {
			continue
		}
		parts = append(parts, strings.Trim(trimmed, "/"))
	}
	if len(parts) == 0 {
		return baseURL
	}
	return baseURL + "/" + strings.Join(parts, "/")
}

func escapePath(value string) string {
	return url.PathEscape(strings.TrimSpace(value))
}
