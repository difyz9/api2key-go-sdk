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
	ProjectID string
	APIKey    string
	Provider  string
	Locale    string
	Search    string
}

type SynthesizeSpeechRequest struct {
	ProjectID        string  `json:"-"`
	APIKey           string  `json:"-"`
	Provider         string  `json:"provider,omitempty"`
	Text             string  `json:"text"`
	Voice            string  `json:"voice,omitempty"`
	Locale           string  `json:"locale,omitempty"`
	Rate             float64 `json:"rate,omitempty"`
	Volume           float64 `json:"volume,omitempty"`
	Pitch            float64 `json:"pitch,omitempty"`
	Format           string  `json:"format,omitempty"`
	Style            string  `json:"style,omitempty"`
	StorageKey       string  `json:"storageKey,omitempty"`
	DownloadFilename string  `json:"downloadFilename,omitempty"`
}

type SynthesizeSpeechResult struct {
	Audio       []byte
	ContentType string
	FileName    string
	Provider    string
	Voice       string
	Locale      string
	Format      string
	Charged     string
	StorageKey  string
	DownloadURL string
	Headers     http.Header
}

type ASRRequest struct {
	ProjectID       string
	APIKey          string
	AudioFilePath   string
	AudioURL        string
	Provider        string
	EngineModelType string
	Async           bool
}

type ASRTaskQueryRequest struct {
	ProjectID string
	APIKey    string
	TaskID    string
	Provider  string
}

type DownloadSpeechAudioResult struct {
	Audio       []byte
	ContentType string
	FileName    string
	Headers     http.Header
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
	if strings.TrimSpace(input.ProjectID) != "" {
		query.Set("projectId", input.ProjectID)
	}
	endpoint := joinURL(c.speechURL, "api", "voices")
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
	endpoint := joinURL(c.speechURL, "api", "speech")
	if strings.TrimSpace(input.ProjectID) != "" {
		endpoint += "?" + url.Values{"projectId": []string{input.ProjectID}}.Encode()
	}
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
	if strings.TrimSpace(input.StorageKey) != "" {
		body["storageKey"] = input.StorageKey
	}
	if strings.TrimSpace(input.DownloadFilename) != "" {
		body["downloadFilename"] = input.DownloadFilename
	}
	raw, headers, err := c.requestBinary(ctx, http.MethodPost, endpoint, map[string]string{"x-api-key": input.APIKey}, body)
	if err != nil {
		return nil, err
	}
	return &SynthesizeSpeechResult{
		Audio:       raw,
		ContentType: headers.Get("Content-Type"),
		FileName:    headerFilename(headers.Get("Content-Disposition")),
		Provider:    headers.Get("X-TTS-Provider"),
		Voice:       headers.Get("X-TTS-Voice"),
		Locale:      headers.Get("X-TTS-Locale"),
		Format:      headers.Get("X-TTS-Format"),
		Charged:     headers.Get("X-TTS-Credits-Charged"),
		StorageKey:  headers.Get("X-TTS-Storage-Key"),
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
	return c.GetASRTaskWithOptions(ctx, ASRTaskQueryRequest{
		APIKey: apiKey,
		TaskID: taskID,
	})
}

func (c *Client) GetASRTaskWithOptions(ctx context.Context, input ASRTaskQueryRequest) (*ASRTaskResponse, error) {
	if strings.TrimSpace(input.APIKey) == "" {
		return nil, errors.New("api key is required")
	}
	if strings.TrimSpace(input.TaskID) == "" {
		return nil, errors.New("task id is required")
	}
	endpoint := joinURL(c.speechURL, "api", "asr", "tasks", escapePath(input.TaskID))
	query := url.Values{}
	if strings.TrimSpace(input.ProjectID) != "" {
		query.Set("projectId", input.ProjectID)
	}
	if strings.TrimSpace(input.Provider) != "" {
		query.Set("provider", input.Provider)
	}
	if encoded := query.Encode(); encoded != "" {
		endpoint += "?" + encoded
	}
	var raw map[string]any
	headers := map[string]string{"x-api-key": input.APIKey}
	if err := c.requestJSON(ctx, http.MethodGet, endpoint, headers, nil, &raw); err != nil {
		return nil, err
	}
	return decodeASRTask(raw), nil
}

func (c *Client) PollASRTask(ctx context.Context, apiKey, taskID string, interval time.Duration, maxAttempts int) (*ASRTaskResponse, error) {
	return c.PollASRTaskWithOptions(ctx, ASRTaskQueryRequest{
		APIKey: apiKey,
		TaskID: taskID,
	}, interval, maxAttempts)
}

func (c *Client) PollASRTaskWithOptions(ctx context.Context, input ASRTaskQueryRequest, interval time.Duration, maxAttempts int) (*ASRTaskResponse, error) {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	if maxAttempts <= 0 {
		maxAttempts = 30
	}
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		result, err := c.GetASRTaskWithOptions(ctx, input)
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
	endpoint := joinURL(c.speechURL, "api", "asr", action)
	query := url.Values{}
	if strings.TrimSpace(input.ProjectID) != "" {
		query.Set("projectId", input.ProjectID)
	}
	if encoded := query.Encode(); encoded != "" {
		endpoint += "?" + encoded
	}
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

func escapeStorageKeyPath(key string) string {
	parts := strings.Split(strings.TrimSpace(key), "/")
	escaped := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		escaped = append(escaped, url.PathEscape(part))
	}
	return strings.Join(escaped, "/")
}

func (c *Client) DownloadSpeechAudio(ctx context.Context, key string) (*DownloadSpeechAudioResult, error) {
	if strings.TrimSpace(key) == "" {
		return nil, errors.New("storage key is required")
	}
	endpoint := joinURL(c.speechURL, "api", "files", "download") + "/" + escapeStorageKeyPath(key)
	raw, headers, err := c.requestBinary(ctx, http.MethodGet, endpoint, nil, nil)
	if err != nil {
		return nil, err
	}
	return &DownloadSpeechAudioResult{
		Audio:       raw,
		ContentType: headers.Get("Content-Type"),
		FileName:    headerFilename(headers.Get("Content-Disposition")),
		Headers:     headers,
	}, nil
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

func headerFilename(contentDisposition string) string {
	trimmed := strings.TrimSpace(contentDisposition)
	if trimmed == "" {
		return ""
	}
	parts := strings.Split(trimmed, ";")
	for _, part := range parts {
		segment := strings.TrimSpace(part)
		if !strings.HasPrefix(strings.ToLower(segment), "filename=") {
			continue
		}
		return strings.Trim(strings.TrimSpace(strings.TrimPrefix(segment, "filename=")), `"`)
	}
	return ""
}
