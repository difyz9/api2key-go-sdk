package api2key

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
)

type ListProductsRequest struct {
	ProjectID string
}

type ListProductsResponse []map[string]any

type GetProductRequest struct {
	ProjectID string
	ProductID string
}

type GetProductResponse map[string]any

func (c *Client) ListProducts(ctx context.Context, input ListProductsRequest) (*ListProductsResponse, error) {
	query := url.Values{}
	if projectID := strings.TrimSpace(input.ProjectID); projectID != "" {
		query.Set("projectId", projectID)
	}

	var out ListProductsResponse
	endpoint := joinURL(c.baseAPIURL, c.apiPrefix, "products")
	if encoded := query.Encode(); encoded != "" {
		endpoint += "?" + encoded
	}
	if err := c.requestJSON(ctx, http.MethodGet, endpoint, nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetProduct(ctx context.Context, input GetProductRequest) (*GetProductResponse, error) {
	if strings.TrimSpace(input.ProductID) == "" {
		return nil, errors.New("product id is required")
	}

	query := url.Values{}
	if projectID := strings.TrimSpace(input.ProjectID); projectID != "" {
		query.Set("projectId", projectID)
	}

	var out GetProductResponse
	endpoint := joinURL(c.baseAPIURL, c.apiPrefix, "products", escapePath(input.ProductID))
	if encoded := query.Encode(); encoded != "" {
		endpoint += "?" + encoded
	}
	if err := c.requestJSON(ctx, http.MethodGet, endpoint, nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
