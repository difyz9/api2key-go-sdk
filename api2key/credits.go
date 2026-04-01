package api2key

import (
	"context"
	"errors"
	"net/http"
	"strings"
)

type SpendCreditsRequest struct {
	UserID      string `json:"userId"`
	Amount      int    `json:"amount"`
	Service     string `json:"service"`
	TaskID      string `json:"taskId,omitempty"`
	Description string `json:"description,omitempty"`
}

type SpendCreditsResponse struct {
	BalanceAfter int  `json:"balanceAfter"`
	Idempotent   bool `json:"idempotent,omitempty"`
}

type ReserveCreditsRequest struct {
	UserID  string `json:"userId"`
	TaskID  string `json:"taskId"`
	Service string `json:"service"`
	Amount  int    `json:"amount"`
}

type ReserveCreditsResponse struct {
	ReservationID string `json:"reservation_id"`
	Idempotent    bool   `json:"idempotent,omitempty"`
}

type ConfirmCreditsResponse struct {
	OK bool `json:"ok"`
}

type CancelCreditsResponse struct {
	OK         bool `json:"ok"`
	Idempotent bool `json:"idempotent,omitempty"`
}

func (c *Client) SpendCredits(ctx context.Context, input SpendCreditsRequest) (*SpendCreditsResponse, error) {
	if err := c.requireServiceSecret(); err != nil {
		return nil, err
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

func (c *Client) ReserveCredits(ctx context.Context, input ReserveCreditsRequest) (*ReserveCreditsResponse, error) {
	if err := c.requireServiceSecret(); err != nil {
		return nil, err
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
	if err := c.requireServiceSecret(); err != nil {
		return nil, err
	}
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
	if err := c.requireServiceSecret(); err != nil {
		return nil, err
	}
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
