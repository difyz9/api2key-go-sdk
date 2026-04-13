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
	// ProjectID is only required during login to establish the initial project-scoped session.
	ProjectID string `json:"projectId,omitempty"`
}

type LoginResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken,omitempty"`
	ExpiresIn    int64  `json:"expiresIn,omitempty"`
	User         any    `json:"user,omitempty"`
}

type CreateAPIKeyRequest struct {
	// Name is user-facing only; the backend derives project scope from the current JWT.
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

type UserAPIKey struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	KeyPrefix  string `json:"keyPrefix"`
	Secret     string `json:"secret,omitempty"`
	Active     bool   `json:"active"`
	LastUsedAt *int64 `json:"lastUsedAt,omitempty"`
	CreatedAt  int64  `json:"createdAt"`
	UpdatedAt  int64  `json:"updatedAt"`
}

type ListAPIKeysResponse struct {
	Keys []UserAPIKey `json:"keys"`
}

type UpdateAPIKeyRequest struct {
	Name   *string `json:"name,omitempty"`
	Active *bool   `json:"active,omitempty"`
}

var ErrAPIKeySecretUnavailable = errors.New("api key secret is only returned when the key is created")

type EnsureAPIKeyResponse struct {
	Key             UserAPIKey `json:"key"`
	Secret          string     `json:"secret,omitempty"`
	Created         bool       `json:"created"`
	SecretAvailable bool       `json:"secretAvailable"`
}

func (c *Client) Login(ctx context.Context, input LoginRequest) (*LoginResponse, error) {
	if strings.TrimSpace(input.Email) == "" {
		return nil, errors.New("email is required")
	}
	if strings.TrimSpace(input.Password) == "" {
		return nil, errors.New("password is required")
	}
	if strings.TrimSpace(input.ProjectID) == "" {
		return nil, errors.New("project id is required")
	}
	var out LoginResponse
	endpoint := joinURL(c.baseAPIURL, c.apiPrefix, "auth", "login")
	headers := map[string]string{"X-Project-Id": input.ProjectID}
	if err := c.requestJSON(ctx, http.MethodPost, endpoint, headers, input, &out); err != nil {
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
	headers := bearerHeaders(accessToken)
	if err := c.requestJSON(ctx, http.MethodPost, endpoint, headers, input, &out); err != nil {
		return nil, err
	}
	if strings.TrimSpace(out.Key.Secret) == "" {
		return nil, fmt.Errorf("create api key succeeded but returned empty secret")
	}
	return &out, nil
}

func (c *Client) ListAPIKeys(ctx context.Context, accessToken string) (*ListAPIKeysResponse, error) {
	if strings.TrimSpace(accessToken) == "" {
		return nil, errors.New("access token is required")
	}
	var out ListAPIKeysResponse
	endpoint := joinURL(c.baseAPIURL, c.apiPrefix, "user", "api-keys")
	if err := c.requestJSON(ctx, http.MethodGet, endpoint, bearerHeaders(accessToken), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateAPIKey(ctx context.Context, accessToken, keyID string, input UpdateAPIKeyRequest) error {
	if strings.TrimSpace(accessToken) == "" {
		return errors.New("access token is required")
	}
	if strings.TrimSpace(keyID) == "" {
		return errors.New("key id is required")
	}
	if input.Name == nil && input.Active == nil {
		return errors.New("name or active must be provided")
	}
	endpoint := joinURL(c.baseAPIURL, c.apiPrefix, "user", "api-keys", escapePath(keyID))
	return c.requestJSON(ctx, http.MethodPatch, endpoint, bearerHeaders(accessToken), input, nil)
}

func (c *Client) DeleteAPIKey(ctx context.Context, accessToken, keyID string) error {
	if strings.TrimSpace(accessToken) == "" {
		return errors.New("access token is required")
	}
	if strings.TrimSpace(keyID) == "" {
		return errors.New("key id is required")
	}
	endpoint := joinURL(c.baseAPIURL, c.apiPrefix, "user", "api-keys", escapePath(keyID))
	return c.requestJSON(ctx, http.MethodDelete, endpoint, bearerHeaders(accessToken), nil, nil)
}

func (c *Client) FindAPIKeyByName(ctx context.Context, accessToken, keyName string) (*UserAPIKey, error) {
	if strings.TrimSpace(accessToken) == "" {
		return nil, errors.New("access token is required")
	}
	keyName = strings.TrimSpace(keyName)
	if keyName == "" {
		return nil, errors.New("key name is required")
	}

	keys, err := c.ListAPIKeys(ctx, accessToken)
	if err != nil {
		return nil, err
	}

	var candidate *UserAPIKey
	for _, key := range keys.Keys {
		if key.Name != keyName {
			continue
		}
		if candidate == nil || preferAPIKey(key, *candidate) {
			selected := key
			candidate = &selected
		}
	}

	return candidate, nil
}

func (c *Client) EnsureAPIKey(ctx context.Context, accessToken string, input CreateAPIKeyRequest) (*EnsureAPIKeyResponse, error) {
	keyName := strings.TrimSpace(input.Name)
	if keyName == "" {
		return nil, errors.New("api key name is required")
	}

	existing, err := c.FindAPIKeyByName(ctx, accessToken, keyName)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return &EnsureAPIKeyResponse{
			Key:             *existing,
			Secret:          existing.Secret,
			Created:         false,
			SecretAvailable: strings.TrimSpace(existing.Secret) != "",
		}, nil
	}

	created, err := c.CreateAPIKey(ctx, accessToken, CreateAPIKeyRequest{Name: keyName})
	if err != nil {
		return nil, err
	}

	return &EnsureAPIKeyResponse{
		Key: UserAPIKey{
			ID:        created.Key.ID,
			Name:      created.Key.Name,
			KeyPrefix: created.Key.KeyPrefix,
			Secret:    created.Key.Secret,
			Active:    true,
			CreatedAt: created.Key.CreatedAt,
			UpdatedAt: created.Key.CreatedAt,
		},
		Secret:          created.Key.Secret,
		Created:         true,
		SecretAvailable: true,
	}, nil
}

func (c *Client) LoginAndCreateAPIKey(ctx context.Context, loginRequest LoginRequest, createRequest CreateAPIKeyRequest) (*CreateAPIKeyResponse, error) {
	loginResult, err := c.Login(ctx, loginRequest)
	if err != nil {
		return nil, err
	}
	return c.CreateAPIKey(ctx, loginResult.AccessToken, createRequest)
}

func bearerHeaders(accessToken string) map[string]string {
	return map[string]string{"Authorization": "Bearer " + strings.TrimSpace(accessToken)}
}

func preferAPIKey(current, candidate UserAPIKey) bool {
	if current.Active != candidate.Active {
		return current.Active
	}
	if current.UpdatedAt != candidate.UpdatedAt {
		return current.UpdatedAt > candidate.UpdatedAt
	}
	return current.CreatedAt > candidate.CreatedAt
}
