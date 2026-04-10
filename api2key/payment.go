package api2key

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const DefaultDirectPaymentType = "wechat"

type UnifiedPaymentPayload struct {
	QRCode        string  `json:"qrCode,omitempty"`
	PayURL        string  `json:"payUrl,omitempty"`
	PayPalOrderID *string `json:"paypalOrderId,omitempty"`
}

type DirectPaymentCreateRequest struct {
	Subject     string  `json:"subject"`
	Amount      float64 `json:"amount"`
	Description string  `json:"description,omitempty"`
	ProjectID   string  `json:"projectId,omitempty"`
	PaymentType string  `json:"paymentType,omitempty"`
}

type DirectPaymentCreateResponse struct {
	ID             string                `json:"id"`
	Subject        string                `json:"subject"`
	Amount         float64               `json:"amount"`
	Currency       string                `json:"currency"`
	PaymentType    string                `json:"paymentType"`
	OrderNo        string                `json:"orderNo"`
	UnifiedOrderNo string                `json:"unifiedOrderNo"`
	Data           UnifiedPaymentPayload `json:"data"`
}

type DirectPaymentQueryRequest struct {
	DirectPaymentID string
	OrderNo         string
}

type DirectPaymentRecord struct {
	ID       string  `json:"id"`
	OrderNo  string  `json:"orderNo"`
	Subject  string  `json:"subject"`
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
	Status   string  `json:"status"`
	PaidAt   *int64  `json:"paidAt,omitempty"`
}

type DirectPaymentQueryResponse struct {
	LocalStatus       string              `json:"localStatus"`
	UnifiedStatus     int                 `json:"unifiedStatus"`
	UnifiedStatusDesc string              `json:"unifiedStatusDesc"`
	Paid              bool                `json:"paid"`
	Payment           DirectPaymentRecord `json:"payment"`
}

func (c *Client) CreateDirectPayment(ctx context.Context, accessToken string, input DirectPaymentCreateRequest) (*DirectPaymentCreateResponse, error) {
	if strings.TrimSpace(accessToken) == "" {
		return nil, errors.New("access token is required")
	}
	if strings.TrimSpace(input.Subject) == "" {
		return nil, errors.New("subject is required")
	}
	if input.Amount <= 0 {
		return nil, errors.New("amount must be greater than 0")
	}
	input.PaymentType = normalizeDirectPaymentType(input.PaymentType)

	var out DirectPaymentCreateResponse
	endpoint := joinURL(c.baseAPIURL, c.apiPrefix, "payment", "unified", "direct", "create")
	if err := c.requestJSON(ctx, http.MethodPost, endpoint, bearerHeaders(accessToken), input, &out); err != nil {
		return nil, err
	}
	if strings.TrimSpace(out.ID) == "" {
		return nil, fmt.Errorf("create direct payment succeeded but returned empty id")
	}
	return &out, nil
}

func (c *Client) GetDirectPaymentStatus(ctx context.Context, accessToken string, input DirectPaymentQueryRequest) (*DirectPaymentQueryResponse, error) {
	if strings.TrimSpace(accessToken) == "" {
		return nil, errors.New("access token is required")
	}
	query := url.Values{}
	if strings.TrimSpace(input.DirectPaymentID) != "" {
		query.Set("directPaymentId", input.DirectPaymentID)
	}
	if strings.TrimSpace(input.OrderNo) != "" {
		query.Set("orderNo", input.OrderNo)
	}
	if len(query) == 0 {
		return nil, errors.New("direct payment id or order no is required")
	}

	var out DirectPaymentQueryResponse
	endpoint := joinURL(c.baseAPIURL, c.apiPrefix, "payment", "unified", "direct", "query") + "?" + query.Encode()
	if err := c.requestJSON(ctx, http.MethodGet, endpoint, bearerHeaders(accessToken), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) PollDirectPaymentStatus(ctx context.Context, accessToken string, input DirectPaymentQueryRequest, interval time.Duration, maxAttempts int) (*DirectPaymentQueryResponse, error) {
	if interval <= 0 {
		interval = 3 * time.Second
	}
	if maxAttempts <= 0 {
		maxAttempts = 30
	}
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		result, err := c.GetDirectPaymentStatus(ctx, accessToken, input)
		if err != nil {
			return nil, err
		}
		if result.Paid || isTerminalDirectPaymentStatus(result.LocalStatus) || isTerminalDirectPaymentStatus(result.Payment.Status) {
			return result, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(interval):
		}
	}
	return nil, fmt.Errorf("poll direct payment timed out after %d attempts", maxAttempts)
}

func normalizeDirectPaymentType(paymentType string) string {
	paymentType = strings.TrimSpace(strings.ToLower(paymentType))
	if paymentType == "" {
		return DefaultDirectPaymentType
	}
	return paymentType
}

func isTerminalDirectPaymentStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "paid", "failed", "cancelled", "refunded":
		return true
	default:
		return false
	}
}
