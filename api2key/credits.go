package api2key

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type CreditsBalanceScope struct {
	Type        string `json:"type,omitempty"`
	ProjectID   string `json:"projectId,omitempty"`
	ProjectName string `json:"projectName,omitempty"`
	ProjectSlug string `json:"projectSlug,omitempty"`
}

type CreditsAccountSummary struct {
	Balance     int  `json:"balance"`
	Reserved    int  `json:"reserved"`
	TotalEarned *int `json:"totalEarned,omitempty"`
	TotalSpent  *int `json:"totalSpent,omitempty"`
}

type CreditsBalanceResponse struct {
	Balance     int                   `json:"balance"`
	Reserved    int                   `json:"reserved"`
	TotalEarned *int                  `json:"total_earned,omitempty"`
	TotalSpent  *int                  `json:"total_spent,omitempty"`
	Account     CreditsAccountSummary `json:"account"`
	Scope       CreditsBalanceScope   `json:"scope"`
}

type GetCreditsBalanceRequest struct {
	AccessToken string `json:"-"`
	APIKey      string `json:"-"`
}

type GetLedgerRequest struct {
	AccessToken string `json:"-"`
	APIKey      string `json:"-"`
	Page        int    `json:"-"`
	Size        int    `json:"-"`
	Type        string `json:"-"`
	Service     string `json:"-"`
	ProjectID   string `json:"-"`
}

type LedgerItem struct {
	ID           string `json:"id"`
	UserID       string `json:"userId"`
	ProjectID    string `json:"projectId"`
	Type         string `json:"type"`
	Delta        int    `json:"delta"`
	BalanceAfter int    `json:"balanceAfter"`
	Service      string `json:"service"`
	Model        string `json:"model"`
	TaskID       string `json:"taskId"`
	Description  string `json:"description"`
	CreatedAt    int64  `json:"createdAt"`
}

type LedgerPagination struct {
	Page       int `json:"page"`
	Size       int `json:"size"`
	Total      int `json:"total"`
	TotalPages int `json:"totalPages"`
}

type GetLedgerResponse struct {
	List       []LedgerItem     `json:"list"`
	Pagination LedgerPagination `json:"pagination"`
}

type DeductCreditsRequest struct {
	AccessToken string `json:"-"`
	APIKey      string `json:"-"`
	Amount      int    `json:"amount"`
	Service     string `json:"service"`
	TaskID      string `json:"taskId,omitempty"`
	Description string `json:"description,omitempty"`
}

type DeductCreditsResponse = SpendCreditsResponse

func (c *Client) GetCreditsBalance(ctx context.Context, accessToken string) (*CreditsBalanceResponse, error) {
	return c.GetCreditsBalanceWithOptions(ctx, GetCreditsBalanceRequest{AccessToken: accessToken})
}

func (c *Client) GetCreditsBalanceWithOptions(ctx context.Context, input GetCreditsBalanceRequest) (*CreditsBalanceResponse, error) {
	if strings.TrimSpace(input.AccessToken) == "" && strings.TrimSpace(input.APIKey) == "" {
		return nil, errors.New("access token or api key is required")
	}
	var out CreditsBalanceResponse
	endpoint := joinURL(c.baseAPIURL, c.apiPrefix, "credits", "balance")
	if err := c.requestJSON(ctx, http.MethodGet, endpoint, authHeaders(input.APIKey, input.AccessToken), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetLedger(ctx context.Context, input GetLedgerRequest) (*GetLedgerResponse, error) {
	if strings.TrimSpace(input.AccessToken) == "" && strings.TrimSpace(input.APIKey) == "" {
		return nil, errors.New("access token or api key is required")
	}
	query := url.Values{}
	if input.Page > 0 {
		query.Set("page", strconv.Itoa(input.Page))
	}
	if input.Size > 0 {
		query.Set("size", strconv.Itoa(input.Size))
	}
	if strings.TrimSpace(input.Type) != "" {
		query.Set("type", input.Type)
	}
	if strings.TrimSpace(input.Service) != "" {
		query.Set("service", input.Service)
	}
	if strings.TrimSpace(input.ProjectID) != "" {
		query.Set("projectId", input.ProjectID)
	}
	endpoint := joinURL(c.baseAPIURL, c.apiPrefix, "credits", "ledger")
	if encoded := query.Encode(); encoded != "" {
		endpoint += "?" + encoded
	}
	var out GetLedgerResponse
	if err := c.requestJSON(ctx, http.MethodGet, endpoint, authHeaders(input.APIKey, input.AccessToken), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeductCredits(ctx context.Context, input DeductCreditsRequest) (*DeductCreditsResponse, error) {
	if strings.TrimSpace(input.AccessToken) == "" && strings.TrimSpace(input.APIKey) == "" {
		return nil, errors.New("access token or api key is required")
	}
	if input.Amount <= 0 {
		return nil, errors.New("amount must be greater than 0")
	}
	if strings.TrimSpace(input.Service) == "" {
		return nil, errors.New("service is required")
	}
	var out DeductCreditsResponse
	endpoint := joinURL(c.baseAPIURL, c.apiPrefix, "credits", "deduct")
	if err := c.requestJSON(ctx, http.MethodPost, endpoint, authHeaders(input.APIKey, input.AccessToken), input, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

type SpendCreditsRequest struct {
	// ProjectID is required for service-to-service credits mutations.
	ProjectID   string `json:"projectId"`
	UserID      string `json:"userId"`
	Amount      int    `json:"amount"`
	Service     string `json:"service"`
	TaskID      string `json:"taskId,omitempty"`
	Description string `json:"description,omitempty"`
}

type SpendCreditsResponse struct {
	BalanceAfter int    `json:"balanceAfter"`
	Idempotent   bool   `json:"idempotent,omitempty"`
	ScopeType    string `json:"scopeType,omitempty"`
	ProjectID    string `json:"projectId,omitempty"`
}

type GrantCreditsRequest struct {
	ProjectID   string `json:"projectId"`
	UserID      string `json:"userId"`
	Amount      int    `json:"amount"`
	TaskID      string `json:"taskId,omitempty"`
	Description string `json:"description,omitempty"`
}

type GrantCreditsResponse map[string]any

type RefundCreditsRequest struct {
	ProjectID   string `json:"projectId"`
	UserID      string `json:"userId"`
	Amount      int    `json:"amount"`
	Service     string `json:"service,omitempty"`
	TaskID      string `json:"taskId,omitempty"`
	Description string `json:"description,omitempty"`
}

type RefundCreditsResponse struct {
	BalanceAfter int    `json:"balanceAfter"`
	Idempotent   bool   `json:"idempotent,omitempty"`
	ScopeType    string `json:"scopeType,omitempty"`
	ProjectID    string `json:"projectId,omitempty"`
}

type ReserveCreditsRequest struct {
	// ProjectID is required for service-to-service credits mutations.
	ProjectID string `json:"projectId"`
	UserID    string `json:"userId"`
	TaskID    string `json:"taskId"`
	Service   string `json:"service"`
	Amount    int    `json:"amount"`
}

type ReserveCreditsResponse struct {
	ReservationID string `json:"reservation_id"`
	Idempotent    bool   `json:"idempotent,omitempty"`
	ScopeType     string `json:"scopeType,omitempty"`
	ProjectID     string `json:"projectId,omitempty"`
}

type ConfirmCreditsResponse struct {
	OK bool `json:"ok"`
}

type CancelCreditsResponse struct {
	OK         bool `json:"ok"`
	Idempotent bool `json:"idempotent,omitempty"`
}

func (c *Client) SpendCredits(ctx context.Context, input SpendCreditsRequest) (*SpendCreditsResponse, error) {
	// if err := c.requireServiceSecret(); err != nil {
	// 	return nil, err
	// }
	if strings.TrimSpace(input.ProjectID) == "" {
		return nil, errors.New("project id is required")
	}
	if strings.TrimSpace(input.UserID) == "" {
		return nil, errors.New("user id is required")
	}
	if input.Amount <= 0 {
		return nil, errors.New("amount must be greater than 0")
	}
	if strings.TrimSpace(input.Service) == "" {
		return nil, errors.New("service is required")
	}
	var out SpendCreditsResponse
	endpoint := joinURL(c.baseAPIURL, c.apiPrefix, "credits", "spend")
	headers := map[string]string{"X-Service-Secret": c.serviceSecret}
	if err := c.requestJSON(ctx, http.MethodPost, endpoint, headers, input, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GrantCredits(ctx context.Context, input GrantCreditsRequest) (*GrantCreditsResponse, error) {
	// if err := c.requireServiceSecret(); err != nil {
	// 	return nil, err
	// }
	if strings.TrimSpace(input.ProjectID) == "" {
		return nil, errors.New("project id is required")
	}
	if strings.TrimSpace(input.UserID) == "" {
		return nil, errors.New("user id is required")
	}
	if input.Amount <= 0 {
		return nil, errors.New("amount must be greater than 0")
	}
	var out GrantCreditsResponse
	endpoint := joinURL(c.baseAPIURL, c.apiPrefix, "credits", "grant")
	headers := map[string]string{"X-Service-Secret": c.serviceSecret}
	if err := c.requestJSON(ctx, http.MethodPost, endpoint, headers, input, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) RefundCredits(ctx context.Context, input RefundCreditsRequest) (*RefundCreditsResponse, error) {
	if err := c.requireServiceSecret(); err != nil {
	// 	return nil, err
	// }
	if strings.TrimSpace(input.ProjectID) == "" {
		return nil, errors.New("project id is required")
	}
	if strings.TrimSpace(input.UserID) == "" {
		return nil, errors.New("user id is required")
	}
	if input.Amount <= 0 {
		return nil, errors.New("amount must be greater than 0")
	}
	var out RefundCreditsResponse
	endpoint := joinURL(c.baseAPIURL, c.apiPrefix, "credits", "refund")
	headers := map[string]string{"X-Service-Secret": c.serviceSecret}
	if err := c.requestJSON(ctx, http.MethodPost, endpoint, headers, input, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ReserveCredits(ctx context.Context, input ReserveCreditsRequest) (*ReserveCreditsResponse, error) {
	// if err := c.requireServiceSecret(); err != nil {
	// 	return nil, err
	// }
	if strings.TrimSpace(input.ProjectID) == "" {
		return nil, errors.New("project id is required")
	}
	if strings.TrimSpace(input.UserID) == "" {
		return nil, errors.New("user id is required")
	}
	if strings.TrimSpace(input.TaskID) == "" {
		return nil, errors.New("task id is required")
	}
	if input.Amount <= 0 {
		return nil, errors.New("amount must be greater than 0")
	}
	if strings.TrimSpace(input.Service) == "" {
		return nil, errors.New("service is required")
	}
	var out ReserveCreditsResponse
	endpoint := joinURL(c.baseAPIURL, c.apiPrefix, "credits", "reserve")
	headers := map[string]string{"X-Service-Secret": c.serviceSecret}
	if err := c.requestJSON(ctx, http.MethodPost, endpoint, headers, input, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ConfirmCredits(ctx context.Context, reservationID string) (*ConfirmCreditsResponse, error) {
	// if err := c.requireServiceSecret(); err != nil {
	// 	return nil, err
	// }
	if strings.TrimSpace(reservationID) == "" {
		return nil, errors.New("reservation id is required")
	}
	var out ConfirmCreditsResponse
	endpoint := joinURL(c.baseAPIURL, c.apiPrefix, "credits", "confirm", escapePath(reservationID))
	headers := map[string]string{"X-Service-Secret": c.serviceSecret}
	if err := c.requestJSON(ctx, http.MethodPost, endpoint, headers, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) CancelCredits(ctx context.Context, reservationID string) (*CancelCreditsResponse, error) {
	// if err := c.requireServiceSecret(); err != nil {
	// 	return nil, err
	// }
	if strings.TrimSpace(reservationID) == "" {
		return nil, errors.New("reservation id is required")
	}
	var out CancelCreditsResponse
	endpoint := joinURL(c.baseAPIURL, c.apiPrefix, "credits", "cancel", escapePath(reservationID))
	headers := map[string]string{"X-Service-Secret": c.serviceSecret}
	if err := c.requestJSON(ctx, http.MethodPost, endpoint, headers, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) requireServiceSecret() error {
	if strings.TrimSpace(c.serviceSecret) == "" {
		return errors.New("service secret is required; use WithServiceSecret")
	}
	return nil
}
