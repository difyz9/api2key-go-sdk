package api2key

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	// ProjectID is only required during login to establish the initial project-scoped session.
	ProjectID string `json:"projectId,omitempty"`
}

type LoginResponse struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken,omitempty"`
	ExpiresIn    int64  `json:"expiresIn,omitempty"`
	User         any    `json:"user,omitempty"`
}

type RegisterRequest struct {
	Email     string  `json:"email"`
	Password  string  `json:"password"`
	Name      *string `json:"name,omitempty"`
	ProjectID *string `json:"-"`
}

type VerifyEmailRequest struct {
	Token *string `json:"token,omitempty"`
	Email *string `json:"email,omitempty"`
	Code  *string `json:"code,omitempty"`
}

type ResendVerificationRequest struct {
	Email string `json:"email"`
}

type RefreshTokenRequest struct {
	RefreshToken string  `json:"refreshToken"`
	ProjectID    *string `json:"-"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
	Code            string `json:"code"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

type ResetPasswordRequest struct {
	Email       string `json:"email"`
	Code        string `json:"code"`
	NewPassword string `json:"newPassword"`
}

type AutoLoginRequest struct {
	Token string `json:"token"`
}

type GenerateAutoLoginLinkRequest struct {
	UserID      *string `json:"userId,omitempty"`
	Email       *string `json:"email,omitempty"`
	RedirectURL string  `json:"redirectUrl"`
	ExpiresIn   string  `json:"expiresIn"`
	APIKey      *string `json:"-"`
}

type CreateExtensionGrantRequest struct {
	DeviceID    string  `json:"deviceId"`
	ExtensionID *string `json:"extensionId,omitempty"`
	Source      string  `json:"source"`
	ProjectID   *string `json:"projectId,omitempty"`
	State       string  `json:"state"`
}

type PollExtensionGrantRequest struct {
	GrantID string `json:"grantId"`
	State   string `json:"state"`
}

type ExtensionGrantActionRequest struct {
	State string `json:"state"`
}

type UserProfile struct {
	ID        string  `json:"id"`
	Email     string  `json:"email"`
	Name      string  `json:"name"`
	Avatar    string  `json:"avatar"`
	Role      string  `json:"role"`
	Status    string  `json:"status"`
	ProjectID *string `json:"projectId,omitempty"`
}

type MeScope struct {
	Type        string  `json:"type,omitempty"`
	ProjectID   *string `json:"projectId,omitempty"`
	ProjectName *string `json:"projectName,omitempty"`
	ProjectSlug *string `json:"projectSlug,omitempty"`
}

type MembershipSummary struct {
	Tier    string `json:"tier,omitempty"`
	Status  string `json:"status,omitempty"`
	EndDate *int64 `json:"endDate,omitempty"`
}

type ProjectPermissionSummary struct {
	ID          string  `json:"id,omitempty"`
	ProjectID   string  `json:"projectId,omitempty"`
	ProjectName string  `json:"projectName,omitempty"`
	ProjectSlug string  `json:"projectSlug,omitempty"`
	Role        string  `json:"role,omitempty"`
	Tier        string  `json:"tier,omitempty"`
	Status      string  `json:"status,omitempty"`
	Credits     int     `json:"credits,omitempty"`
	StartDate   *int64  `json:"startDate,omitempty"`
	EndDate     *int64  `json:"endDate,omitempty"`
	AutoRenew   bool    `json:"autoRenew,omitempty"`
}

type CurrentUser struct {
	ID               string                    `json:"id"`
	Email            string                    `json:"email"`
	Name             string                    `json:"name"`
	Avatar           string                    `json:"avatar"`
	Role             string                    `json:"role"`
	EmailVerified    bool                      `json:"emailVerified"`
	EmailVerifiedAt  *int64                    `json:"emailVerifiedAt,omitempty"`
	Credits          int                       `json:"credits"`
	ProjectID        *string                   `json:"projectId,omitempty"`
	ProjectName      *string                   `json:"projectName,omitempty"`
	ProjectSlug      *string                   `json:"projectSlug,omitempty"`
	Membership       *MembershipSummary        `json:"membership,omitempty"`
	ProjectPermission *ProjectPermissionSummary `json:"projectPermission,omitempty"`
}

type GetMeRequest struct {
	AccessToken string
	APIKey      string
	ProjectID   string
}

type GetMeResponse struct {
	Scope MeScope     `json:"scope"`
	User  CurrentUser `json:"user"`
}

type UpdateProfileRequest struct {
	Name     *string `json:"name,omitempty"`
	Username *string `json:"username,omitempty"`
	Avatar   *string `json:"avatar,omitempty"`
}

type CreateAPIKeyRequest struct {
	// Name is user-facing only; the backend derives project scope from the current JWT.
	Name      string  `json:"name,omitempty"`
	ProjectID *string `json:"projectId,omitempty"`
}

type APIKeySecret struct {
	ID               string  `json:"id"`
	Name             string  `json:"name"`
	KeyPrefix        string  `json:"keyPrefix"`
	ProjectID        *string `json:"projectId,omitempty"`
	ProjectName      *string `json:"projectName,omitempty"`
	ProjectSlug      *string `json:"projectSlug,omitempty"`
	ProjectScopeCode *string `json:"projectScopeCode,omitempty"`
	Secret           string  `json:"secret"`
	CreatedAt        int64   `json:"createdAt"`
}

type CreateAPIKeyResponse struct {
	Key APIKeySecret `json:"key"`
}

type UserAPIKey struct {
	ID               string  `json:"id"`
	Name             string  `json:"name"`
	KeyPrefix        string  `json:"keyPrefix"`
	ProjectID        *string `json:"projectId,omitempty"`
	ProjectScopeCode *string `json:"projectScopeCode,omitempty"`
	Secret           string  `json:"secret,omitempty"`
	Active           bool    `json:"active"`
	LastUsedAt       *int64  `json:"lastUsedAt,omitempty"`
	CreatedAt        int64   `json:"createdAt"`
	UpdatedAt        int64   `json:"updatedAt"`
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
	var out LoginResponse
	endpoint := joinURL(c.baseAPIURL, c.apiPrefix, "auth", "login")
	var headers map[string]string
	if projectID := strings.TrimSpace(input.ProjectID); projectID != "" {
		headers = map[string]string{"X-Project-Id": projectID}
	}
	if err := c.requestJSON(ctx, http.MethodPost, endpoint, headers, input, &out); err != nil {
		return nil, err
	}
	if strings.TrimSpace(out.AccessToken) == "" {
		return nil, fmt.Errorf("login succeeded but access token is empty")
	}
	return &out, nil
}

func (c *Client) Register(ctx context.Context, input RegisterRequest) (*map[string]any, error) {
	if strings.TrimSpace(input.Email) == "" {
		return nil, errors.New("email is required")
	}
	if strings.TrimSpace(input.Password) == "" {
		return nil, errors.New("password is required")
	}
	var out map[string]any
	endpoint := joinURL(c.baseAPIURL, c.apiPrefix, "auth", "register")
	var headers map[string]string
	if input.ProjectID != nil {
		projectID := strings.TrimSpace(*input.ProjectID)
		if projectID == "" {
			return nil, errors.New("project id cannot be empty when provided")
		}
		headers = map[string]string{"X-Project-Id": projectID}
	}
	if err := c.requestJSON(ctx, http.MethodPost, endpoint, headers, input, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) VerifyEmail(ctx context.Context, input VerifyEmailRequest) (*map[string]any, error) {
	if input.Token == nil && input.Email == nil && input.Code == nil {
		return nil, errors.New("token, email, or code must be provided")
	}
	var out map[string]any
	endpoint := joinURL(c.baseAPIURL, c.apiPrefix, "auth", "verify-email")
	if err := c.requestJSON(ctx, http.MethodPost, endpoint, nil, input, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ResendVerification(ctx context.Context, input ResendVerificationRequest) (*map[string]any, error) {
	if strings.TrimSpace(input.Email) == "" {
		return nil, errors.New("email is required")
	}
	var out map[string]any
	endpoint := joinURL(c.baseAPIURL, c.apiPrefix, "auth", "resend-verification")
	if err := c.requestJSON(ctx, http.MethodPost, endpoint, nil, input, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) RefreshToken(ctx context.Context, input RefreshTokenRequest) (*LoginResponse, error) {
	if strings.TrimSpace(input.RefreshToken) == "" {
		return nil, errors.New("refresh token is required")
	}
	var out LoginResponse
	endpoint := joinURL(c.baseAPIURL, c.apiPrefix, "auth", "refresh")
	var headers map[string]string
	if input.ProjectID != nil {
		projectID := strings.TrimSpace(*input.ProjectID)
		if projectID == "" {
			return nil, errors.New("project id cannot be empty when provided")
		}
		headers = map[string]string{"X-Project-Id": projectID}
	}
	if err := c.requestJSON(ctx, http.MethodPost, endpoint, headers, input, &out); err != nil {
		return nil, err
	}
	if strings.TrimSpace(out.AccessToken) == "" {
		return nil, fmt.Errorf("refresh token succeeded but access token is empty")
	}
	return &out, nil
}

func (c *Client) Logout(ctx context.Context, accessToken string) (*map[string]any, error) {
	if strings.TrimSpace(accessToken) == "" {
		return nil, errors.New("access token is required")
	}
	var out map[string]any
	endpoint := joinURL(c.baseAPIURL, c.apiPrefix, "auth", "logout")
	if err := c.requestJSON(ctx, http.MethodPost, endpoint, bearerHeaders(accessToken), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ChangePasswordSendCode(ctx context.Context, accessToken string) (*map[string]any, error) {
	if strings.TrimSpace(accessToken) == "" {
		return nil, errors.New("access token is required")
	}
	var out map[string]any
	endpoint := joinURL(c.baseAPIURL, c.apiPrefix, "auth", "change-password", "send-code")
	if err := c.requestJSON(ctx, http.MethodPost, endpoint, bearerHeaders(accessToken), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ChangePassword(ctx context.Context, accessToken string, input ChangePasswordRequest) (*map[string]any, error) {
	if strings.TrimSpace(accessToken) == "" {
		return nil, errors.New("access token is required")
	}
	if strings.TrimSpace(input.CurrentPassword) == "" {
		return nil, errors.New("current password is required")
	}
	if strings.TrimSpace(input.NewPassword) == "" {
		return nil, errors.New("new password is required")
	}
	if strings.TrimSpace(input.Code) == "" {
		return nil, errors.New("code is required")
	}
	var out map[string]any
	endpoint := joinURL(c.baseAPIURL, c.apiPrefix, "auth", "change-password")
	if err := c.requestJSON(ctx, http.MethodPost, endpoint, bearerHeaders(accessToken), input, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ForgotPassword(ctx context.Context, input ForgotPasswordRequest) (*map[string]any, error) {
	if strings.TrimSpace(input.Email) == "" {
		return nil, errors.New("email is required")
	}
	var out map[string]any
	endpoint := joinURL(c.baseAPIURL, c.apiPrefix, "auth", "forgot-password")
	if err := c.requestJSON(ctx, http.MethodPost, endpoint, nil, input, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ResetPassword(ctx context.Context, input ResetPasswordRequest) (*map[string]any, error) {
	if strings.TrimSpace(input.Email) == "" {
		return nil, errors.New("email is required")
	}
	if strings.TrimSpace(input.Code) == "" {
		return nil, errors.New("code is required")
	}
	if strings.TrimSpace(input.NewPassword) == "" {
		return nil, errors.New("new password is required")
	}
	var out map[string]any
	endpoint := joinURL(c.baseAPIURL, c.apiPrefix, "auth", "reset-password")
	if err := c.requestJSON(ctx, http.MethodPost, endpoint, nil, input, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) AutoLogin(ctx context.Context, input AutoLoginRequest) (*LoginResponse, error) {
	if strings.TrimSpace(input.Token) == "" {
		return nil, errors.New("token is required")
	}
	var out LoginResponse
	endpoint := joinURL(c.baseAPIURL, c.apiPrefix, "auth", "auto-login")
	if err := c.requestJSON(ctx, http.MethodPost, endpoint, nil, input, &out); err != nil {
		return nil, err
	}
	if strings.TrimSpace(out.AccessToken) == "" {
		return nil, fmt.Errorf("auto login succeeded but access token is empty")
	}
	return &out, nil
}

func (c *Client) GenerateAutoLoginLink(ctx context.Context, input GenerateAutoLoginLinkRequest) (*map[string]any, error) {
	if strings.TrimSpace(input.RedirectURL) == "" {
		return nil, errors.New("redirect url is required")
	}
	if strings.TrimSpace(input.ExpiresIn) == "" {
		return nil, errors.New("expires in is required")
	}
	var out map[string]any
	endpoint := joinURL(c.baseAPIURL, c.apiPrefix, "auth", "generate-auto-login-link")
	var headers map[string]string
	if input.APIKey != nil {
		apiKey := strings.TrimSpace(*input.APIKey)
		if apiKey == "" {
			return nil, errors.New("api key cannot be empty when provided")
		}
		headers = map[string]string{"x-api-key": apiKey}
	}
	if err := c.requestJSON(ctx, http.MethodPost, endpoint, headers, input, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) CreateExtensionGrant(ctx context.Context, input CreateExtensionGrantRequest) (*map[string]any, error) {
	if strings.TrimSpace(input.DeviceID) == "" {
		return nil, errors.New("device id is required")
	}
	if strings.TrimSpace(input.Source) == "" {
		return nil, errors.New("source is required")
	}
	if strings.TrimSpace(input.State) == "" {
		return nil, errors.New("state is required")
	}
	var out map[string]any
	endpoint := joinURL(c.baseAPIURL, c.apiPrefix, "auth", "extension", "grants")
	if err := c.requestJSON(ctx, http.MethodPost, endpoint, nil, input, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) PollExtensionGrant(ctx context.Context, input PollExtensionGrantRequest) (*map[string]any, error) {
	grantID := strings.TrimSpace(input.GrantID)
	if grantID == "" {
		return nil, errors.New("grant id is required")
	}
	state := strings.TrimSpace(input.State)
	if state == "" {
		return nil, errors.New("state is required")
	}
	var out map[string]any
	endpoint := joinURL(c.baseAPIURL, c.apiPrefix, "auth", "extension", "grants", escapePath(grantID)) + "?state=" + url.QueryEscape(state)
	if err := c.requestJSON(ctx, http.MethodGet, endpoint, nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ApproveExtensionGrant(ctx context.Context, accessToken, grantID string, input ExtensionGrantActionRequest) (*map[string]any, error) {
	if strings.TrimSpace(accessToken) == "" {
		return nil, errors.New("access token is required")
	}
	grantID = strings.TrimSpace(grantID)
	if grantID == "" {
		return nil, errors.New("grant id is required")
	}
	if strings.TrimSpace(input.State) == "" {
		return nil, errors.New("state is required")
	}
	var out map[string]any
	endpoint := joinURL(c.baseAPIURL, c.apiPrefix, "auth", "extension", "grants", escapePath(grantID), "approve")
	if err := c.requestJSON(ctx, http.MethodPost, endpoint, bearerHeaders(accessToken), input, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ExchangeExtensionGrant(ctx context.Context, grantID string, input ExtensionGrantActionRequest) (*map[string]any, error) {
	grantID = strings.TrimSpace(grantID)
	if grantID == "" {
		return nil, errors.New("grant id is required")
	}
	if strings.TrimSpace(input.State) == "" {
		return nil, errors.New("state is required")
	}
	var out map[string]any
	endpoint := joinURL(c.baseAPIURL, c.apiPrefix, "auth", "extension", "grants", escapePath(grantID), "exchange")
	if err := c.requestJSON(ctx, http.MethodPost, endpoint, nil, input, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetProfile(ctx context.Context, accessToken string) (*UserProfile, error) {
	if strings.TrimSpace(accessToken) == "" {
		return nil, errors.New("access token is required")
	}
	var out struct {
		Profile UserProfile `json:"profile"`
	}
	endpoint := joinURL(c.baseAPIURL, c.apiPrefix, "user", "profile")
	if err := c.requestJSON(ctx, http.MethodGet, endpoint, bearerHeaders(accessToken), nil, &out); err != nil {
		return nil, err
	}
	return &out.Profile, nil
}

func (c *Client) UpdateProfile(ctx context.Context, accessToken string, input UpdateProfileRequest) (*UserProfile, error) {
	if strings.TrimSpace(accessToken) == "" {
		return nil, errors.New("access token is required")
	}
	if input.Name == nil && input.Username == nil && input.Avatar == nil {
		return nil, errors.New("name, username, or avatar must be provided")
	}
	var out struct {
		Profile UserProfile `json:"profile"`
	}
	endpoint := joinURL(c.baseAPIURL, c.apiPrefix, "user", "profile")
	if err := c.requestJSON(ctx, http.MethodPut, endpoint, bearerHeaders(accessToken), input, &out); err != nil {
		return nil, err
	}
	return &out.Profile, nil
}

func (c *Client) GetMe(ctx context.Context, input GetMeRequest) (*GetMeResponse, error) {
	if strings.TrimSpace(input.AccessToken) == "" && strings.TrimSpace(input.APIKey) == "" {
		return nil, errors.New("access token or api key is required")
	}
	endpoint := joinURL(c.baseAPIURL, c.apiPrefix, "auth", "me")
	if projectID := strings.TrimSpace(input.ProjectID); projectID != "" {
		endpoint += "?" + url.Values{"projectId": []string{projectID}}.Encode()
	}
	var out GetMeResponse
	if err := c.requestJSON(ctx, http.MethodGet, endpoint, authHeaders(input.APIKey, input.AccessToken), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetSettings(ctx context.Context, accessToken string) (*map[string]any, error) {
	if strings.TrimSpace(accessToken) == "" {
		return nil, errors.New("access token is required")
	}
	var out map[string]any
	endpoint := joinURL(c.baseAPIURL, c.apiPrefix, "user", "settings")
	if err := c.requestJSON(ctx, http.MethodGet, endpoint, bearerHeaders(accessToken), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateSettings(ctx context.Context, accessToken string, input map[string]any) (*map[string]any, error) {
	if strings.TrimSpace(accessToken) == "" {
		return nil, errors.New("access token is required")
	}
	var out map[string]any
	endpoint := joinURL(c.baseAPIURL, c.apiPrefix, "user", "settings")
	if err := c.requestJSON(ctx, http.MethodPut, endpoint, bearerHeaders(accessToken), input, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) CreateAPIKey(ctx context.Context, accessToken string, input CreateAPIKeyRequest) (*CreateAPIKeyResponse, error) {
	if strings.TrimSpace(accessToken) == "" {
		return nil, errors.New("access token is required")
	}
	if input.ProjectID != nil {
		projectID := strings.TrimSpace(*input.ProjectID)
		if projectID == "" {
			return nil, errors.New("project id cannot be empty when provided")
		}
		input.ProjectID = &projectID
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

	created, err := c.CreateAPIKey(ctx, accessToken, input)
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
