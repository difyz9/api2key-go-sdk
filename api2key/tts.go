package api2key

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Voice struct {
	ShortName   string `json:"shortName"`
	DisplayName string `json:"displayName"`
	Locale      string `json:"locale"`
	Provider    string `json:"provider"`
}

type ListVoicesResponse struct {
	Provider string  `json:"provider"`
	Total    int     `json:"total"`
	Voices   []Voice `json:"voices"`
}

type ListVoicesRequest struct {
	APIKey   string
	Provider string
	Locale   string
	Search   string
}

type SynthesizeSpeechRequest struct {
	APIKey   string  `json:"-"`
	Provider string  `json:"provider,omitempty"`
	Text     string  `json:"text"`
	Voice    string  `json:"voice,omitempty"`
	Locale   string  `json:"locale,omitempty"`
	Rate     float64 `json:"rate,omitempty"`
	Volume   float64 `json:"volume,omitempty"`
	Pitch    float64 `json:"pitch,omitempty"`
	Format   string  `json:"format,omitempty"`
	Style    string  `json:"style,omitempty"`
}

type SynthesizeSpeechResult struct {
	Audio       []byte
	Provider    string
	Voice       string
	Charged     string
	DownloadURL string
	Headers     http.Header
}

type ASRRequest struct {
	APIKey          string
	AudioFilePath   string
	AudioURL        string
	Provider        string
	EngineModelType string
	Async           bool
}

type ASRTaskResponse struct {
	TaskID            int64          `json:"taskId"`
	StatusStr         string         `json:"statusStr"`
	RequestID         string         `json:"requestId,omitempty"`
	EngineModelType   string         `json:"engineModelType,omitempty"`
	SourceType        string         `json:"sourceType,omitempty"`
	SourceName        string         `json:"sourceName,omitempty"`
	SourceSize        int64          `json:"sourceSize,omitempty"`
	Provider          string         `json:"provider,omitempty"`
	RetryAfterSeconds int64          `json:"retryAfterSeconds,omitempty"`
	Text              string         `json:"text,omitempty"`
	SRT               string         `json:"srt,omitempty"`
	ResultURL         string         `json:"resultUrl,omitempty"`
	Error             string         `json:"error,omitempty"`
	Segments          any            `json:"segments,omitempty"`
	Raw               map[string]any `json:"-"`
}

func (c *Client) ListVoices(ctx context.Context, input ListVoicesRequest) (*ListVoicesResponse, error) {
	if strings.TrimSpace(input.APIKey) == "" {
		return nil, errors.New("api key is required")
	}
	query := url.Values{}
	if strings.TrimSpace(input.Provider) != "" {
		query.Set("provider", input.Provider)
	}
	if strings.TrimSpace(input.Locale) != "" {
		query.Set("locale", input.Locale)
	}
	if strings.TrimSpace(input.Search) != "" {
		query.Set("search", input.Search)
	}
	endpoint := joinURL(c.ttsURL, "api", "voices")
	if encoded := query.Encode(); encoded != "" {
		endpoint += "?" + encoded
	}
	var out ListVoicesResponse
	headers := map[string]string{"x-api-key": input.APIKey}
	if err := c.requestJSON(ctx, http.MethodGet, endpoint, headers, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) SynthesizeSpeech(ctx context.Context, input SynthesizeSpeechRequest) (*SynthesizeSpeechResult, error) {
	if strings.TrimSpace(input.APIKey) == "" {
		return nil, errors.New("api key is required")
	}
	if strings.TrimSpace(input.Text) == "" {
		return nil, errors.New("text is required")
	}
	endpoint := joinURL(c.ttsURL, "api", "speech")
	body := map[string]any{
		"provider": input.Provider,
		"text":     input.Text,
		"voice":    input.Voice,
		"locale":   input.Locale,
		"rate":     input.Rate,
		"volume":   input.Volume,
		"pitch":    input.Pitch,
		"format":   input.Format,
	}
	if strings.TrimSpace(input.Style) != "" {
		body["style"] = input.Style
	}
	raw, headers, err := c.requestBinary(ctx, http.MethodPost, endpoint, map[string]string{"x-api-key": input.APIKey}, body)
	if err != nil {
		return nil, err
	}
	return &SynthesizeSpeechResult{
		Audio:       raw,
		Provider:    headers.Get("X-TTS-Provider"),
		Voice:       headers.Get("X-TTS-Voice"),
		Charged:     headers.Get("X-TTS-Credits-Charged"),
		DownloadURL: headers.Get("X-TTS-Download-Url"),
		Headers:     headers,
	}, nil
}

func (c *Client) SaveSpeechToFile(ctx context.Context, input SynthesizeSpeechRequest, outputPath string) (*SynthesizeSpeechResult, error) {
	result, err := c.SynthesizeSpeech(ctx, input)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(outputPath, result.Audio, 0o644); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) TranscribeAudio(ctx context.Context, input ASRRequest) (*ASRTaskResponse, error) {
	return c.submitASR(ctx, "transcribe", input)
}

func (c *Client) AudioToSRT(ctx context.Context, input ASRRequest) (*ASRTaskResponse, error) {
	return c.submitASR(ctx, "srt", input)
}

func (c *Client) GetASRTask(ctx context.Context, apiKey, taskID string) (*ASRTaskResponse, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, errors.New("api key is required")
	}
	if strings.TrimSpace(taskID) == "" {
		return nil, errors.New("task id is required")
	}
	endpoint := joinURL(c.ttsURL, "api", "asr", "tasks", escapePath(taskID))
	var raw map[string]any
	headers := map[string]string{"x-api-key": apiKey}
	if err := c.requestJSON(ctx, http.MethodGet, endpoint, headers, nil, &raw); err != nil {
		return nil, err
	}
	return decodeASRTask(raw), nil
}

func (c *Client) PollASRTask(ctx context.Context, apiKey, taskID string, interval time.Duration, maxAttempts int) (*ASRTaskResponse, error) {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	if maxAttempts <= 0 {
		maxAttempts = 30
	}
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		result, err := c.GetASRTask(ctx, apiKey, taskID)
		if err != nil {
			return nil, err
		}
		status := strings.ToLower(strings.TrimSpace(result.StatusStr))
		if status == "success" || status == "failed" {
			return result, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(interval):
		}
	}
	return nil, fmt.Errorf("poll asr task timed out after %d attempts", maxAttempts)
}

func (c *Client) submitASR(ctx context.Context, action string, input ASRRequest) (*ASRTaskResponse, error) {
	if strings.TrimSpace(input.APIKey) == "" {
		return nil, errors.New("api key is required")
	}
	if strings.TrimSpace(input.AudioFilePath) == "" && strings.TrimSpace(input.AudioURL) == "" {
		return nil, errors.New("audio file path or audio url is required")
	}
	provider := strings.TrimSpace(input.Provider)
	if provider == "" {
		provider = "tencent"
	}
	engineModelType := strings.TrimSpace(input.EngineModelType)
	if engineModelType == "" {
		engineModelType = "16k_zh"
	}
	endpoint := joinURL(c.ttsURL, "api", "asr", action)
	headers := map[string]string{"x-api-key": input.APIKey}

	if strings.TrimSpace(input.AudioURL) != "" {
		payload := map[string]any{
			"provider":        provider,
			"audioUrl":        input.AudioURL,
			"engineModelType": engineModelType,
			"async":           input.Async,
		}
		var raw map[string]any
		if err := c.requestJSON(ctx, http.MethodPost, endpoint, headers, payload, &raw); err != nil {
			return nil, err
		}
		return decodeASRTask(raw), nil
	}

	fields := map[string]string{
		"provider":        provider,
		"engineModelType": engineModelType,
	}
	if input.Async {
		fields["async"] = "true"
	}
	var raw map[string]any
	if err := c.requestMultipart(ctx, endpoint, headers, fields, "file", input.AudioFilePath, &raw); err != nil {
		return nil, err
	}
	return decodeASRTask(raw), nil
}

func decodeASRTask(raw map[string]any) *ASRTaskResponse {
	result := &ASRTaskResponse{Raw: raw}
	result.TaskID = anyToInt64(raw["taskId"])
	result.StatusStr = anyToString(raw["statusStr"])
	result.RequestID = anyToString(raw["requestId"])
	result.EngineModelType = anyToString(raw["engineModelType"])
	result.SourceType = anyToString(raw["sourceType"])
	result.SourceName = anyToString(raw["sourceName"])
	result.SourceSize = anyToInt64(raw["sourceSize"])
	result.Provider = anyToString(raw["provider"])
	result.RetryAfterSeconds = anyToInt64(raw["retryAfterSeconds"])
	result.Text = anyToString(raw["text"])
	result.SRT = anyToString(raw["srt"])
	result.ResultURL = anyToString(raw["resultUrl"])
	result.Error = anyToString(raw["error"])
	result.Segments = raw["segments"]
	return result
}

func anyToString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		return ""
	}
}

func anyToInt64(value any) int64 {
	switch v := value.(type) {
	case float64:
		return int64(v)
	case float32:
		return int64(v)
	case int:
		return int64(v)
	case int64:
		return v
	case int32:
		return int64(v)
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		if err == nil {
			return parsed
		}
	}
	return 0
}
