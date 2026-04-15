package api2key

import (
	"context"
	"errors"
	"net/http"
	"strings"
)

type ListOrdersResponse []map[string]any

type CreateOrderRequest struct {
	ProductID string `json:"productId"`
}

type CreateOrderResponse map[string]any

type GetOrderResponse map[string]any

func (c *Client) ListOrders(ctx context.Context, accessToken string) (*ListOrdersResponse, error) {
	if strings.TrimSpace(accessToken) == "" {
		return nil, errors.New("access token is required")
	}

	var out ListOrdersResponse
	endpoint := joinURL(c.baseAPIURL, c.apiPrefix, "orders")
	if err := c.requestJSON(ctx, http.MethodGet, endpoint, bearerHeaders(accessToken), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) CreateOrder(ctx context.Context, accessToken string, input CreateOrderRequest) (*CreateOrderResponse, error) {
	if strings.TrimSpace(accessToken) == "" {
		return nil, errors.New("access token is required")
	}
	if strings.TrimSpace(input.ProductID) == "" {
		return nil, errors.New("product id is required")
	}

	var out CreateOrderResponse
	endpoint := joinURL(c.baseAPIURL, c.apiPrefix, "orders")
	if err := c.requestJSON(ctx, http.MethodPost, endpoint, bearerHeaders(accessToken), input, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetOrder(ctx context.Context, accessToken, orderID string) (*GetOrderResponse, error) {
	if strings.TrimSpace(accessToken) == "" {
		return nil, errors.New("access token is required")
	}
	if strings.TrimSpace(orderID) == "" {
		return nil, errors.New("order id is required")
	}

	var out GetOrderResponse
	endpoint := joinURL(c.baseAPIURL, c.apiPrefix, "orders", escapePath(orderID))
	if err := c.requestJSON(ctx, http.MethodGet, endpoint, bearerHeaders(accessToken), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
