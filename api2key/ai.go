package api2key

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type AIModel struct {
	Key                  string   `json:"key"`
	ID                   string   `json:"id"`
	Name                 string   `json:"name"`
	ModelID              string   `json:"modelId"`
	Provider             string   `json:"provider"`
	ModelType            string   `json:"modelType"`
	MinTier              string   `json:"minTier"`
	CreditCostPerRequest *float64 `json:"creditCostPerRequest,omitempty"`
	IsFeatured           bool     `json:"isFeatured"`
	Description          *string  `json:"description,omitempty"`
	InputPricing         any      `json:"inputPricing,omitempty"`
	OutputPricing        any      `json:"outputPricing,omitempty"`
	Locked               bool     `json:"locked"`
}

type AIProjectSummary struct {
	ID   string         `json:"id,omitempty"`
	Name string         `json:"name,omitempty"`
	Slug string         `json:"slug,omitempty"`
	Tier string         `json:"tier,omitempty"`
	Role string         `json:"role,omitempty"`
	Type string         `json:"type,omitempty"`
	Raw  map[string]any `json:"-"`
}

type ListAIModelsRequest struct {
	ProjectID     string
	APIKey        string
	AccessToken   string
	OnlyAvailable bool
	Type          string
}

type ListAIModelsResponse struct {
	UserTier       string               `json:"userTier"`
	CurrentProject AIProjectSummary     `json:"currentProject"`
	Models         []AIModel            `json:"models"`
	Grouped        map[string][]AIModel `json:"grouped"`
	RawProject     map[string]any       `json:"-"`
}

type GetAIBalanceRequest struct {
	ProjectID   string
	APIKey      string
	AccessToken string
}

type AIBalanceResponse struct {
	Balance     int    `json:"balance"`
	ProjectID   string `json:"projectId"`
	ProjectName string `json:"projectName"`
	ProjectSlug string `json:"projectSlug"`
}

type AIRequest struct {
	ProjectID   string
	APIKey      string
	AccessToken string
	Headers     map[string]string
	Body        any
}

type AIResponse struct {
	Body    json.RawMessage
	Headers http.Header
}

func (r *AIResponse) Decode(output any) error {
	if r == nil {
		return errors.New("ai response is nil")
	}
	if output == nil {
		return errors.New("output is nil")
	}
	if len(r.Body) == 0 {
		return errors.New("ai response body is empty")
	}
	if err := json.Unmarshal(r.Body, output); err != nil {
		return fmt.Errorf("decode ai response: %w", err)
	}
	return nil

}

type AIStreamResponse struct {
	Body       io.ReadCloser
	Headers    http.Header
	StatusCode int
}

type AIHistoryStep struct {
	Tool      string `json:"tool"`
	Arguments string `json:"arguments"`
	Output    string `json:"output,omitempty"`
	Error     string `json:"error,omitempty"`
}

type AIHistoryMessage struct {
	ID             string          `json:"id"`
	Role           string          `json:"role"`
	Content        string          `json:"content"`
	Timestamp      string          `json:"timestamp"`
	Steps          []AIHistoryStep `json:"steps,omitempty"`
	ExecutionMS    *int            `json:"execution_ms,omitempty"`
	Success        *bool           `json:"success,omitempty"`
	IsError        *bool           `json:"isError,omitempty"`
	VideoID        string          `json:"videoId,omitempty"`
	TrackingStatus string          `json:"trackingStatus,omitempty"`
}

type AIConversationSummary struct {
	ConversationID string `json:"conversationId"`
	Title          string `json:"title"`
	UpdatedAt      string `json:"updatedAt"`
	MessageCount   int    `json:"messageCount"`
	Preview        string `json:"preview"`
}

type ListAIHistoriesResponse struct {
	Conversations []AIConversationSummary `json:"conversations"`
	UpdatedAt     *string                 `json:"updatedAt"`
}

type AIHistoryResponse struct {
	ConversationID string             `json:"conversationId"`
	UpdatedAt      *string            `json:"updatedAt"`
	Messages       []AIHistoryMessage `json:"messages"`
}

type PutAIHistoryRequest struct {
	ConversationID string
	APIKey         string
	AccessToken    string
	Messages       []AIHistoryMessage `json:"messages"`
}

type AISession struct {
	client         *Client
	projectID      string
	apiKey         string
	accessToken    string
	conversationID string
	defaultHeaders map[string]string
}

type AISessionOption func(*AISession)

func NewAISession(client *Client, options ...AISessionOption) *AISession {
	session := &AISession{
		client:         client,
		conversationID: "default",
		defaultHeaders: map[string]string{},
	}
	for _, option := range options {
		option(session)
	}
	if session.client == nil {
		session.client = NewClient()
	}
	return session

}

func WithAISessionProjectID(projectID string) AISessionOption {
	return func(session *AISession) {
		session.projectID = strings.TrimSpace(projectID)
	}
}

func WithAISessionAPIKey(apiKey string) AISessionOption {
	return func(session *AISession) {
		session.apiKey = strings.TrimSpace(apiKey)
	}
}

func WithAISessionAccessToken(accessToken string) AISessionOption {
	return func(session *AISession) {
		session.accessToken = strings.TrimSpace(accessToken)
	}
}

func WithAISessionConversationID(conversationID string) AISessionOption {
	return func(session *AISession) {
		trimmed := strings.TrimSpace(conversationID)
		if trimmed != "" {
			session.conversationID = trimmed
		}
	}
}

func WithAISessionHeader(key, value string) AISessionOption {
	return func(session *AISession) {
		if session.defaultHeaders == nil {
			session.defaultHeaders = map[string]string{}
		}
		session.defaultHeaders[key] = value
	}
}

func (s *AISession) Client() *Client {
	if s == nil {
		return nil
	}
	return s.client
}

func (s *AISession) Clone(options ...AISessionOption) *AISession {
	if s == nil {
		return NewAISession(nil, options...)
	}
	clone := &AISession{
		client:         s.client,
		projectID:      s.projectID,
		apiKey:         s.apiKey,
		accessToken:    s.accessToken,
		conversationID: s.conversationID,
		defaultHeaders: cloneHeaders(s.defaultHeaders),
	}
	for _, option := range options {
		option(clone)
	}
	return clone
}

func (s *AISession) ListModels(ctx context.Context, input ListAIModelsRequest) (*ListAIModelsResponse, error) {
	request := input
	request.ProjectID = firstNonEmpty(request.ProjectID, s.projectID)
	request.APIKey = firstNonEmpty(request.APIKey, s.apiKey)
	request.AccessToken = firstNonEmpty(request.AccessToken, s.accessToken)
	return s.client.ListAIModels(ctx, request)
}

func (s *AISession) Balance(ctx context.Context, input GetAIBalanceRequest) (*AIBalanceResponse, error) {
	request := input
	request.ProjectID = firstNonEmpty(request.ProjectID, s.projectID)
	request.APIKey = firstNonEmpty(request.APIKey, s.apiKey)
	request.AccessToken = firstNonEmpty(request.AccessToken, s.accessToken)
	return s.client.GetAIBalance(ctx, request)
}

func (s *AISession) ChatCompletions(ctx context.Context, body any) (*AIResponse, error) {
	return s.client.ChatCompletions(ctx, s.withDefaults(AIRequest{Body: body}))
}

func (s *AISession) ChatCompletionsStream(ctx context.Context, body any) (*AIStreamResponse, error) {
	return s.client.ChatCompletionsStream(ctx, s.withDefaults(AIRequest{Body: body}))
}

func (s *AISession) Completions(ctx context.Context, body any) (*AIResponse, error) {
	return s.client.Completions(ctx, s.withDefaults(AIRequest{Body: body}))
}

func (s *AISession) CompletionsStream(ctx context.Context, body any) (*AIStreamResponse, error) {
	return s.client.CompletionsStream(ctx, s.withDefaults(AIRequest{Body: body}))
}

func (s *AISession) AnthropicMessages(ctx context.Context, body any, headers map[string]string) (*AIResponse, error) {
	request := s.withDefaults(AIRequest{Body: body, Headers: mergeStringMap(s.defaultHeaders, headers)})
	return s.client.AnthropicMessages(ctx, request)
}

func (s *AISession) AnthropicMessagesStream(ctx context.Context, body any, headers map[string]string) (*AIStreamResponse, error) {
	request := s.withDefaults(AIRequest{Body: body, Headers: mergeStringMap(s.defaultHeaders, headers)})
	return s.client.AnthropicMessagesStream(ctx, request)
}

func (s *AISession) GoogleGenerateContent(ctx context.Context, model string, body any) (*AIResponse, error) {
	return s.client.GoogleGenerateContent(ctx, model, s.withDefaults(AIRequest{Body: body}))
}

func (s *AISession) ListHistories(ctx context.Context) (*ListAIHistoriesResponse, error) {
	return s.client.ListAIHistories(ctx, s.withDefaults(AIRequest{}))
}

func (s *AISession) GetHistory(ctx context.Context, conversationID string) (*AIHistoryResponse, error) {
	return s.client.GetAIHistory(ctx, s.resolveConversationID(conversationID), s.aiAuthHeaders(nil))
}

func (s *AISession) PutHistory(ctx context.Context, messages []AIHistoryMessage, conversationID string) (*AIHistoryResponse, error) {
	return s.client.PutAIHistory(ctx, PutAIHistoryRequest{
		ConversationID: s.resolveConversationID(conversationID),
		APIKey:         s.apiKey,
		AccessToken:    s.accessToken,
		Messages:       messages,
	})
}

func (s *AISession) DeleteHistory(ctx context.Context, conversationID string) error {
	return s.client.DeleteAIHistory(ctx, s.resolveConversationID(conversationID), s.aiAuthHeaders(nil))
}

func (s *AISession) withDefaults(input AIRequest) AIRequest {
	input.ProjectID = firstNonEmpty(input.ProjectID, s.projectID)
	input.APIKey = firstNonEmpty(input.APIKey, s.apiKey)
	input.AccessToken = firstNonEmpty(input.AccessToken, s.accessToken)
	input.Headers = s.aiAuthHeaders(input.Headers)
	return input
}

func (s *AISession) aiAuthHeaders(extra map[string]string) map[string]string {
	return mergeStringMap(authHeaders(s.apiKey, s.accessToken), mergeStringMap(s.defaultHeaders, extra))
}

func (s *AISession) resolveConversationID(conversationID string) string {
	trimmed := strings.TrimSpace(conversationID)
	if trimmed != "" {
		return trimmed
	}
	if strings.TrimSpace(s.conversationID) != "" {
		return s.conversationID
	}
	return "default"
}

func (c *Client) ListAIModels(ctx context.Context, input ListAIModelsRequest) (*ListAIModelsResponse, error) {
	query := url.Values{}
	if input.OnlyAvailable {
		query.Set("onlyAvailable", "true")
	}
	if strings.TrimSpace(input.Type) != "" {
		query.Set("type", input.Type)
	}
	if strings.TrimSpace(input.ProjectID) != "" {
		query.Set("projectId", input.ProjectID)
	}
	endpoint := joinURL(c.baseAPIURL, c.apiPrefix, "ai", "models")
	if encoded := query.Encode(); encoded != "" {
		endpoint += "?" + encoded
	}
	var raw struct {
		UserTier       string               `json:"userTier"`
		CurrentProject map[string]any       `json:"currentProject"`
		Models         []AIModel            `json:"models"`
		Grouped        map[string][]AIModel `json:"grouped"`
	}
	if err := c.requestJSON(ctx, http.MethodGet, endpoint, authHeaders(input.APIKey, input.AccessToken), nil, &raw); err != nil {
		return nil, err
	}
	response := &ListAIModelsResponse{
		UserTier:   raw.UserTier,
		Models:     raw.Models,
		Grouped:    raw.Grouped,
		RawProject: raw.CurrentProject,
	}
	decodeProjectSummary(raw.CurrentProject, &response.CurrentProject)
	return response, nil
}

func (c *Client) GetAIBalance(ctx context.Context, input GetAIBalanceRequest) (*AIBalanceResponse, error) {
	if strings.TrimSpace(input.APIKey) == "" && strings.TrimSpace(input.AccessToken) == "" {
		return nil, errors.New("api key or access token is required")
	}
	query := url.Values{}
	if strings.TrimSpace(input.ProjectID) != "" {
		query.Set("projectId", input.ProjectID)
	}
	endpoint := joinURL(c.baseAPIURL, c.apiPrefix, "ai", "balance")
	if encoded := query.Encode(); encoded != "" {
		endpoint += "?" + encoded
	}
	var out AIBalanceResponse
	if err := c.requestJSON(ctx, http.MethodGet, endpoint, authHeaders(input.APIKey, input.AccessToken), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ChatCompletions(ctx context.Context, input AIRequest) (*AIResponse, error) {
	return c.aiJSON(ctx, joinURL(c.baseAPIURL, c.apiPrefix, "ai", "chat", "completions"), input)
}

func (c *Client) ChatCompletionsStream(ctx context.Context, input AIRequest) (*AIStreamResponse, error) {
	return c.aiStream(ctx, joinURL(c.baseAPIURL, c.apiPrefix, "ai", "chat", "completions"), input)
}

func (c *Client) Completions(ctx context.Context, input AIRequest) (*AIResponse, error) {
	return c.aiJSON(ctx, joinURL(c.baseAPIURL, c.apiPrefix, "ai", "completions"), input)
}

func (c *Client) CompletionsStream(ctx context.Context, input AIRequest) (*AIStreamResponse, error) {
	return c.aiStream(ctx, joinURL(c.baseAPIURL, c.apiPrefix, "ai", "completions"), input)
}

func (c *Client) AnthropicMessages(ctx context.Context, input AIRequest) (*AIResponse, error) {
	return c.aiJSON(ctx, joinURL(c.baseAPIURL, c.apiPrefix, "ai", "anthropic", "v1", "messages"), input)
}

func (c *Client) AnthropicMessagesStream(ctx context.Context, input AIRequest) (*AIStreamResponse, error) {
	return c.aiStream(ctx, joinURL(c.baseAPIURL, c.apiPrefix, "ai", "anthropic", "v1", "messages"), input)
}

func (c *Client) GoogleGenerateContent(ctx context.Context, model string, input AIRequest) (*AIResponse, error) {
	trimmedModel := strings.TrimSpace(model)
	if trimmedModel == "" {
		return nil, errors.New("model is required")
	}
	endpoint := joinURL(c.baseAPIURL, c.apiPrefix, "ai", "google", "v1beta", "models", escapePath(trimmedModel)+":generateContent")
	return c.aiJSON(ctx, endpoint, input)
}

func (c *Client) ListAIHistories(ctx context.Context, input AIRequest) (*ListAIHistoriesResponse, error) {
	if strings.TrimSpace(input.APIKey) == "" && strings.TrimSpace(input.AccessToken) == "" {
		return nil, errors.New("api key or access token is required")
	}
	endpoint := joinURL(c.baseAPIURL, c.apiPrefix, "ai", "histories")
	var out ListAIHistoriesResponse
	if err := c.requestJSON(ctx, http.MethodGet, endpoint, mergeStringMap(authHeaders(input.APIKey, input.AccessToken), input.Headers), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetAIHistory(ctx context.Context, conversationID string, headers map[string]string) (*AIHistoryResponse, error) {
	query := url.Values{}
	if trimmed := strings.TrimSpace(conversationID); trimmed != "" {
		query.Set("conversationId", trimmed)
	}
	endpoint := joinURL(c.baseAPIURL, c.apiPrefix, "ai", "history")
	if encoded := query.Encode(); encoded != "" {
		endpoint += "?" + encoded
	}
	var out AIHistoryResponse
	if err := c.requestJSON(ctx, http.MethodGet, endpoint, headers, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) PutAIHistory(ctx context.Context, input PutAIHistoryRequest) (*AIHistoryResponse, error) {
	if strings.TrimSpace(input.APIKey) == "" && strings.TrimSpace(input.AccessToken) == "" {
		return nil, errors.New("api key or access token is required")
	}
	query := url.Values{}
	if trimmed := strings.TrimSpace(input.ConversationID); trimmed != "" {
		query.Set("conversationId", trimmed)
	}
	endpoint := joinURL(c.baseAPIURL, c.apiPrefix, "ai", "history")
	if encoded := query.Encode(); encoded != "" {
		endpoint += "?" + encoded
	}
	var out AIHistoryResponse
	if err := c.requestJSON(ctx, http.MethodPut, endpoint, authHeaders(input.APIKey, input.AccessToken), map[string]any{"messages": input.Messages}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteAIHistory(ctx context.Context, conversationID string, headers map[string]string) error {
	endpoint := joinURL(c.baseAPIURL, c.apiPrefix, "ai", "history")
	query := url.Values{}
	if trimmed := strings.TrimSpace(conversationID); trimmed != "" {
		query.Set("conversationId", trimmed)
	}
	if encoded := query.Encode(); encoded != "" {
		endpoint += "?" + encoded
	}
	return c.requestJSON(ctx, http.MethodDelete, endpoint, headers, nil, nil)
}

func (c *Client) aiJSON(ctx context.Context, endpoint string, input AIRequest) (*AIResponse, error) {
	if strings.TrimSpace(input.APIKey) == "" && strings.TrimSpace(input.AccessToken) == "" {
		return nil, errors.New("api key or access token is required")
	}
	endpoint = withProjectID(endpoint, input.ProjectID)
	headers := mergeStringMap(authHeaders(input.APIKey, input.AccessToken), input.Headers)
	raw, responseHeaders, err := c.requestRaw(ctx, http.MethodPost, endpoint, headers, input.Body)
	if err != nil {
		return nil, err
	}
	return &AIResponse{Body: json.RawMessage(raw), Headers: responseHeaders}, nil
}

func (c *Client) aiStream(ctx context.Context, endpoint string, input AIRequest) (*AIStreamResponse, error) {
	if strings.TrimSpace(input.APIKey) == "" && strings.TrimSpace(input.AccessToken) == "" {
		return nil, errors.New("api key or access token is required")
	}
	endpoint = withProjectID(endpoint, input.ProjectID)
	headers := mergeStringMap(authHeaders(input.APIKey, input.AccessToken), input.Headers)
	resp, err := c.requestStream(ctx, http.MethodPost, endpoint, headers, input.Body)
	if err != nil {
		return nil, err
	}
	return &AIStreamResponse{
		Body:       resp.Body,
		Headers:    resp.Header.Clone(),
		StatusCode: resp.StatusCode,
	}, nil
}

func withProjectID(endpoint string, projectID string) string {
	trimmedProjectID := strings.TrimSpace(projectID)
	if trimmedProjectID == "" {
		return endpoint
	}
	separator := "?"
	if strings.Contains(endpoint, "?") {
		separator = "&"
	}
	return endpoint + separator + url.Values{"projectId": []string{trimmedProjectID}}.Encode()
}

func mergeStringMap(base map[string]string, extra map[string]string) map[string]string {
	if len(base) == 0 && len(extra) == 0 {
		return map[string]string{}
	}
	merged := make(map[string]string, len(base)+len(extra))
	for key, value := range base {
		merged[key] = value
	}
	for key, value := range extra {
		merged[key] = value
	}
	return merged
}

func cloneHeaders(input map[string]string) map[string]string {
	if len(input) == 0 {
		return map[string]string{}
	}
	cloned := make(map[string]string, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func decodeProjectSummary(raw map[string]any, output *AIProjectSummary) {
	if output == nil {
		return
	}
	output.Raw = raw
	encoded, err := json.Marshal(raw)
	if err != nil {
		return
	}
	_ = json.Unmarshal(encoded, output)
}
