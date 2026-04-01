package api2key

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

type LoginRequest struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	ProjectID string `json:"projectId,omitempty"`
}

type LoginResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken,omitempty"`
	ExpiresIn    int64  `json:"expiresIn,omitempty"`
	User         any    `json:"user,omitempty"`
}

type CreateAPIKeyRequest struct {
	Name string `json:"name,omitempty"`
}

type APIKeySecret struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	KeyPrefix string `json:"keyPrefix"`
	Secret    string `json:"secret"`
	CreatedAt int64  `json:"createdAt"`
}

type CreateAPIKeyResponse struct {
	Key APIKeySecret `json:"key"`
}

func (c *Client) Login(ctx context.Context, input LoginRequest) (*LoginResponse, error) {
	if strings.TrimSpace(input.Email) == "" {
		return nil, errors.New("email is required")
	}
	if strings.TrimSpace(input.Password) == "" {
		return nil, errors.New("password is required")
	}
	var out LoginResponse
	endpoint := joinURL(c.baseAPIURL, c.apiPrefix, "auth", "login")
	if err := c.requestJSON(ctx, http.MethodPost, endpoint, nil, input, &out); err != nil {
		return nil, err
	}
	if strings.TrimSpace(out.AccessToken) == "" {
		return nil, fmt.Errorf("login succeeded but access token is empty")
	}
	return &out, nil
}

func (c *Client) CreateAPIKey(ctx context.Context, accessToken string, input CreateAPIKeyRequest) (*CreateAPIKeyResponse, error) {
	if strings.TrimSpace(accessToken) == "" {
		return nil, errors.New("access token is required")
	}
	var out CreateAPIKeyResponse
	endpoint := joinURL(c.baseAPIURL, c.apiPrefix, "user", "api-keys")
	headers := map[string]string{"Authorization": "Bearer " + accessToken}
	if err := c.requestJSON(ctx, http.MethodPost, endpoint, headers, input, &out); err != nil {
		return nil, err
	}
	if strings.TrimSpace(out.Key.Secret) == "" {
		return nil, fmt.Errorf("create api key succeeded but returned empty secret")
	}
	return &out, nil
}

func (c *Client) LoginAndCreateAPIKey(ctx context.Context, loginRequest LoginRequest, createRequest CreateAPIKeyRequest) (*CreateAPIKeyResponse, error) {
	loginResult, err := c.Login(ctx, loginRequest)
	if err != nil {
		return nil, err
	}
	return c.CreateAPIKey(ctx, loginResult.AccessToken, createRequest)
}
