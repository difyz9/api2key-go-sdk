package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/difyz9/api2key-go-sdk/api2key"
)

var subtitleIndexPattern = regexp.MustCompile(`^\d+$`)

type config struct {
	BaseURL   string
	APIKey    string
	ProjectID string
	Provider  string
	Search    string
	Voice     string
	Locale    string
	Format    string
	Input     string
	Output    string
	Prefix    string
	VideoID   string
	Timeout   time.Duration
	Rate      float64
	Volume    float64
	Pitch     float64
}

type subtitleEntry struct {
	Index string
	Text  string
}

func defaultTestConfig() config {
	return config{
		BaseURL:   "https://v2.api2key.com",
		APIKey:    "sk-43d392747decb79409d96635e58cd0229efeefe42045e2e52d872e28a9997259",
		ProjectID: "",
		Provider:  "auto",
		Search:    "",
		Voice:     "zh-CN-XiaoxiaoNeural",
		Locale:    "zh-CN",
		Format:    "audio-24khz-96kbitrate-mono-mp3",
		Input:     "001.srt",
		Output:    "output",
		Prefix:    "hell_",
		VideoID:   "video_demo",
		Timeout:   3 * time.Minute,
		Rate:      1,
		Volume:    100,
		Pitch:     0,
	}
}

func main() {
	cfg := loadConfig()

	if strings.TrimSpace(cfg.APIKey) == "" {
		log.Fatal("API key is required")
	}
	if !strings.HasPrefix(strings.TrimSpace(cfg.APIKey), "sk-") {
		log.Fatal("API key must start with sk-")
	}
	if strings.TrimSpace(cfg.Input) == "" {
		log.Fatal("subtitle input file is required")
	}

	entries, err := loadSubtitleEntries(cfg.Input)
	if err != nil {
		log.Fatal(err)
	}
	if len(entries) == 0 {
		log.Fatal("subtitle entries are empty after normalization")
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	client := api2key.NewClient(
		api2key.WithBaseAPIURL(cfg.BaseURL),
	)

	provider, voice, err := resolveTTSProfile(ctx, client, cfg)
	if err != nil {
		log.Fatal(err)
	}

	outputDir := strings.TrimSpace(cfg.Output)
	if outputDir == "" {
		log.Fatal("output directory is required")
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		log.Fatalf("create output directory: %v", err)
	}

	extension := outputFileExtension(cfg.Format)
	videoID := strings.TrimSpace(cfg.VideoID)
	if videoID == "" {
		videoID = strings.TrimSpace(cfg.Prefix)
	}
	if videoID == "" {
		videoID = "video_demo"
	}
	totalCharged := make([]string, 0, len(entries))
	for _, entry := range entries {
		outputPath := filepath.Join(outputDir, cfg.Prefix+entry.Index+extension)
		storageFileName := buildStorageFileName(entry.Index, extension)
		storageKey := strings.Trim(strings.TrimSpace(videoID), "/") + "/" + storageFileName
		result, err := client.SaveSpeechToFile(ctx, api2key.SynthesizeSpeechRequest{
			ProjectID:        cfg.ProjectID,
			APIKey:           cfg.APIKey,
			Provider:         provider,
			Text:             entry.Text,
			Voice:            voice,
			Locale:           cfg.Locale,
			Rate:             cfg.Rate,
			Volume:           cfg.Volume,
			Pitch:            cfg.Pitch,
			Format:           cfg.Format,
			StorageKey:       storageKey,
			DownloadFilename: storageFileName,
		}, outputPath)
		if err != nil {
			log.Fatalf("synthesize subtitle %s failed: %v", entry.Index, err)
		}
		if strings.TrimSpace(result.Charged) != "" {
			totalCharged = append(totalCharged, fmt.Sprintf("%s=%s", entry.Index, result.Charged))
		}
		fmt.Printf("[%s] ok -> %s | remote=%s\n", entry.Index, outputPath, result.StorageKey)
	}

	fmt.Println("subtitle batch synthesis ok")
	fmt.Println("base url:", cfg.BaseURL)
	fmt.Println("input:", cfg.Input)
	fmt.Println("output dir:", outputDir)
	fmt.Println("provider:", provider)
	fmt.Println("voice:", voice)
	fmt.Println("format:", cfg.Format)
	fmt.Println("video id:", videoID)
	fmt.Println("segments:", len(entries))
	if len(totalCharged) > 0 {
		fmt.Println("charged:", strings.Join(totalCharged, ", "))
	}
}

func loadConfig() config {
	cfg := defaultTestConfig()
	flag.StringVar(&cfg.BaseURL, "base-url", cfg.BaseURL, "unified api base url")
	flag.StringVar(&cfg.APIKey, "api-key", cfg.APIKey, "user api key starting with sk-")
	flag.StringVar(&cfg.ProjectID, "project-id", cfg.ProjectID, "optional project id")
	flag.StringVar(&cfg.Provider, "provider", cfg.Provider, "tts provider: auto|azure|tencent")
	flag.StringVar(&cfg.Search, "search", cfg.Search, "optional voice search keyword")
	flag.StringVar(&cfg.Voice, "voice", cfg.Voice, "tts voice short name")
	flag.StringVar(&cfg.Locale, "locale", cfg.Locale, "tts locale")
	flag.StringVar(&cfg.Format, "format", cfg.Format, "output audio format")
	flag.StringVar(&cfg.Input, "input", cfg.Input, "input subtitle file path (.srt or .txt)")
	flag.StringVar(&cfg.Output, "output", cfg.Output, "output directory path")
	flag.StringVar(&cfg.Prefix, "prefix", cfg.Prefix, "optional file name prefix")
	flag.StringVar(&cfg.VideoID, "video-id", cfg.VideoID, "remote storage video id used in storageKey, e.g. video_123")
	flag.DurationVar(&cfg.Timeout, "timeout", cfg.Timeout, "request timeout")
	flag.Float64Var(&cfg.Rate, "rate", cfg.Rate, "tts speaking rate")
	flag.Float64Var(&cfg.Volume, "volume", cfg.Volume, "tts volume")
	flag.Float64Var(&cfg.Pitch, "pitch", cfg.Pitch, "tts pitch")
	flag.Parse()
	return cfg
}

func buildStorageFileName(index string, extension string) string {
	trimmed := strings.TrimSpace(index)
	if trimmed == "" {
		return "index_0001" + extension
	}
	parsed, err := strconv.Atoi(trimmed)
	if err == nil {
		return fmt.Sprintf("index_%04d%s", parsed, extension)
	}
	return "index_" + trimmed + extension
}

func resolveTTSProfile(ctx context.Context, client *api2key.Client, cfg config) (string, string, error) {
	requestedProvider := strings.ToLower(strings.TrimSpace(cfg.Provider))
	if requestedProvider != "" && requestedProvider != "auto" {
		voice, err := resolveVoiceForProvider(ctx, client, cfg, requestedProvider)
		if err != nil {
			return "", "", err
		}
		return requestedProvider, voice, nil
	}

	providers := []string{"azure", "tencent"}
	errorsByProvider := make([]string, 0, len(providers))
	for _, provider := range providers {
		voice, err := resolveVoiceForProvider(ctx, client, cfg, provider)
		if err == nil {
			return provider, voice, nil
		}
		errorsByProvider = append(errorsByProvider, err.Error())
	}

	return "", "", fmt.Errorf("no available TTS provider on %s; tried azure/tencent: %s", cfg.BaseURL, strings.Join(errorsByProvider, " | "))
}

func resolveVoiceForProvider(ctx context.Context, client *api2key.Client, cfg config, provider string) (string, error) {
	search := strings.TrimSpace(cfg.Search)
	if search == "" && strings.TrimSpace(cfg.Voice) != "" {
		search = cfg.Voice
	}

	voices, err := client.ListVoices(ctx, api2key.ListVoicesRequest{
		ProjectID: cfg.ProjectID,
		APIKey:    cfg.APIKey,
		Provider:  provider,
		Locale:    cfg.Locale,
		Search:    search,
	})
	if err != nil {
		var apiErr *api2key.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 401 {
			return "", fmt.Errorf("provider %s auth failed: use a valid user API key starting with sk-", provider)
		}
		if errors.As(err, &apiErr) && apiErr.StatusCode == 502 {
			return "", fmt.Errorf("provider %s is not configured locally", provider)
		}
		return "", fmt.Errorf("list voices for provider %s: %w", provider, err)
	}

	requestedVoice := strings.TrimSpace(cfg.Voice)
	if requestedVoice != "" {
		for _, voice := range voices.Voices {
			if strings.EqualFold(strings.TrimSpace(voice.ShortName), requestedVoice) {
				return voice.ShortName, nil
			}
		}
	}

	if len(voices.Voices) == 0 {
		return "", fmt.Errorf("provider %s returned no voices for locale %s", provider, cfg.Locale)
	}

	return voices.Voices[0].ShortName, nil
}

func loadSubtitleEntries(path string) ([]subtitleEntry, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read subtitle file: %w", err)
	}

	normalized := strings.ReplaceAll(string(raw), "\r\n", "\n")
	if strings.EqualFold(filepath.Ext(path), ".txt") {
		return parseTextEntries(normalized), nil
	}
	return parseSRTEntries(normalized), nil
}

func parseTextEntries(text string) []subtitleEntry {
	scanner := bufio.NewScanner(strings.NewReader(text))
	entries := make([]subtitleEntry, 0, 32)
	index := 1
	for scanner.Scan() {
		line := strings.Join(strings.Fields(strings.TrimSpace(scanner.Text())), " ")
		if line == "" {
			continue
		}
		entries = append(entries, subtitleEntry{
			Index: fmt.Sprintf("%03d", index),
			Text:  line,
		})
		index += 1
	}
	return entries
}

func parseSRTEntries(text string) []subtitleEntry {
	blocks := strings.Split(text, "\n\n")
	entries := make([]subtitleEntry, 0, len(blocks))
	seq := 1
	for _, block := range blocks {
		lines := strings.Split(block, "\n")
		entryLines := make([]string, 0, len(lines))
		entryIndex := fmt.Sprintf("%03d", seq)
		for _, rawLine := range lines {
			line := strings.TrimSpace(strings.TrimPrefix(rawLine, "\ufeff"))
			if line == "" || strings.Contains(line, "-->") {
				continue
			}
			if subtitleIndexPattern.MatchString(line) && len(entryLines) == 0 {
				entryIndex = fmt.Sprintf("%03s", line)
				continue
			}
			entryLines = append(entryLines, strings.Join(strings.Fields(line), " "))
		}
		entryText := normalizeParagraphs(entryLines)
		if entryText == "" {
			continue
		}
		entries = append(entries, subtitleEntry{Index: entryIndex, Text: entryText})
		seq += 1
	}
	return entries
}

func outputFileExtension(format string) string {
	normalized := strings.ToLower(strings.TrimSpace(format))
	switch {
	case strings.Contains(normalized, "mp3"):
		return ".mp3"
	case strings.Contains(normalized, "wav"):
		return ".wav"
	case strings.Contains(normalized, "ogg"):
		return ".ogg"
	default:
		return ".audio"
	}
}

func normalizeParagraphs(lines []string) string {
	cleaned := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.Join(strings.Fields(line), " ")
		if line == "" {
			continue
		}
		cleaned = append(cleaned, line)
	}
	return strings.Join(cleaned, "\n")
}

func getenvFloat(key string, fallback float64) float64 {
	_ = key
	return fallback
}
