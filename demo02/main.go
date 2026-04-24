package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/difyz9/api2key-go-sdk/api2key"
)

const sentenceBreak = "###SENTENCE_BREAK###"

type TranslatorConfig struct {
	BaseAPIURL  string
	APIKey      string
	ModelName   string
	InputPath   string
	OutputPath  string
	SourceLang  string
	TargetLang  string
	BatchSize   int
	ContextSize int
	RetryCount  int
	Timeout     time.Duration
}

type TranslationRunConfig struct {
	SourceLang string
	TargetLang string
	ModelName  string
}

type SubtitleItem struct {
	Index int
	Start string
	End   string
	Text  string
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type LLMBatchTranslator struct {
	client *api2key.Client
	config TranslatorConfig
	logger *log.Logger
}

func main() {
	
	
	os.Setenv("API2KEY_BASE_URL", "https://stage.api2key.com")
	os.Setenv("API2KEY_API_KEY", "sk-EA8ADC--")
	os.Setenv("API2KEY_AI_MODEL", "openai/gpt-4o-mini")
	os.Setenv("DEMO02_INPUT", "demo02/tp7Ojf7dPVI.srt")
	os.Setenv("DEMO02_OUTPUT", "demo02/tp7Ojf7dPVI.zh-CN.srt")
	os.Setenv("DEMO02_SOURCE_LANG", "en")
	os.Setenv("DEMO02_TARGET_LANG", "zh-CN")
	os.Setenv("DEMO02_BATCH_SIZE", "12")
	os.Setenv("DEMO02_CONTEXT_SIZE", "2")
	os.Setenv("DEMO02_RETRY_COUNT", "2")
	os.Setenv("API2KEY_TIMEOUT", "5m")
	cfg := loadConfig()



	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	items, err := parseSRTFile(cfg.InputPath)
	if err != nil {
		log.Fatal(err)
	}
	if len(items) == 0 {
		log.Fatal("no subtitle items found")
	}

	translator := &LLMBatchTranslator{
		client: api2key.NewClient(api2key.WithBaseAPIURL(cfg.BaseAPIURL)),
		config: cfg,
		logger: log.New(os.Stdout, "[demo02] ", log.LstdFlags),
	}

	runConfig := TranslationRunConfig{
		SourceLang: cfg.SourceLang,
		TargetLang: cfg.TargetLang,
		ModelName:  cfg.ModelName,
	}

	texts := extractTexts(items)
	shouldTranslate, detectedLanguage, err := translator.shouldTranslate(ctx, texts, runConfig)
	if err != nil {
		log.Fatal(err)
	}
	translator.logger.Printf("translation decision: should_translate=%t detected_language=%s target=%s", shouldTranslate, detectedLanguage, cfg.TargetLang)
	if !shouldTranslate {
		translator.logger.Println("subtitle content is already in target language, skipping translation")
		return
	}

	translated, err := translator.translateSubtitles(ctx, texts, runConfig)
	if err != nil {
		log.Fatal(err)
	}
	if len(translated) != len(items) {
		log.Fatalf("translated subtitle count mismatch: got %d want %d", len(translated), len(items))
	}

	for index := range items {
		items[index].Text = translated[index]
	}

	if err := writeSRTFile(cfg.OutputPath, items); err != nil {
		log.Fatal(err)
	}

	translator.logger.Printf("translation finished: input=%s output=%s subtitles=%d", cfg.InputPath, cfg.OutputPath, len(items))
}

func loadConfig() TranslatorConfig {
	defaultInputPath := filepath.Join("demo02", "tp7Ojf7dPVI.srt")
	defaultOutputPath := filepath.Join("demo02", "tp7Ojf7dPVI.zh-CN.srt")

	var cfg TranslatorConfig
	flag.StringVar(&cfg.BaseAPIURL, "base-url", getenv("API2KEY_BASE_URL", api2key.DefaultBaseAPIURL), "base api url")
	flag.StringVar(&cfg.APIKey, "api-key", getenv("API2KEY_API_KEY", ""), "api key used for subtitle translation")
	flag.StringVar(&cfg.ModelName, "model", getenv("API2KEY_AI_MODEL", "openai/gpt-4o-mini"), "chat model id")
	flag.StringVar(&cfg.InputPath, "input", getenv("DEMO02_INPUT", defaultInputPath), "input srt file path")
	flag.StringVar(&cfg.OutputPath, "output", getenv("DEMO02_OUTPUT", defaultOutputPath), "output srt file path")
	flag.StringVar(&cfg.SourceLang, "source-lang", getenv("DEMO02_SOURCE_LANG", "en"), "source language code")
	flag.StringVar(&cfg.TargetLang, "target-lang", getenv("DEMO02_TARGET_LANG", "zh-CN"), "target language code")
	flag.IntVar(&cfg.BatchSize, "batch-size", getenvInt("DEMO02_BATCH_SIZE", 12), "subtitle count per translation batch")
	flag.IntVar(&cfg.ContextSize, "context-size", getenvInt("DEMO02_CONTEXT_SIZE", 2), "surrounding subtitle context count")
	flag.IntVar(&cfg.RetryCount, "retries", getenvInt("DEMO02_RETRY_COUNT", 2), "retry count per translation batch")
	flag.DurationVar(&cfg.Timeout, "timeout", getenvDuration("API2KEY_TIMEOUT", 5*time.Minute), "translation timeout")
	flag.Parse()

	if strings.TrimSpace(cfg.APIKey) == "" {
		log.Fatal("API2KEY_API_KEY is required")
	}
	if strings.TrimSpace(cfg.InputPath) == "" {
		log.Fatal("input path is required")
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 12
	}
	if cfg.ContextSize < 0 {
		cfg.ContextSize = 0
	}
	if cfg.RetryCount < 0 {
		cfg.RetryCount = 0
	}
	return cfg
}

func (t *LLMBatchTranslator) translateSubtitles(ctx context.Context, texts []string, runConfig TranslationRunConfig) ([]string, error) {
	translated := make([]string, 0, len(texts))

	for start := 0; start < len(texts); start += t.config.BatchSize {
		end := start + t.config.BatchSize
		if end > len(texts) {
			end = len(texts)
		}

		prevStart := start - t.config.ContextSize
		if prevStart < 0 {
			prevStart = 0
		}
		nextEnd := end + t.config.ContextSize
		if nextEnd > len(texts) {
			nextEnd = len(texts)
		}

		currentTexts := texts[start:end]
		prevContext := compactTexts(texts[prevStart:start])
		nextContext := compactTexts(texts[end:nextEnd])

		result, err := t.translateGroupWithRetry(ctx, currentTexts, prevContext, nextContext, runConfig)
		if err != nil {
			return nil, fmt.Errorf("translate subtitles batch %d-%d: %w", start, end, err)
		}
		translated = append(translated, result...)
		t.logger.Printf("translated batch: start=%d end=%d count=%d", start, end, len(result))
	}

	return translated, nil
}

func (t *LLMBatchTranslator) translateGroupWithRetry(ctx context.Context, texts []string, prevContext, nextContext []string, runConfig TranslationRunConfig) ([]string, error) {
	var lastErr error

	for attempt := 0; attempt <= t.config.RetryCount; attempt++ {
		if attempt > 0 {
			t.logger.Printf("retrying translation: attempt=%d max_retries=%d", attempt, t.config.RetryCount)
			time.Sleep(time.Duration(attempt) * time.Second)
		}

		result, err := t.translateGroupWithContext(ctx, texts, prevContext, nextContext, runConfig)
		if err == nil {
			return result, nil
		}

		lastErr = err
		t.logger.Printf("translation attempt failed: attempt=%d error=%v", attempt+1, err)
	}

	return nil, fmt.Errorf("translation failed after %d retries: %w", t.config.RetryCount, lastErr)
}

func (t *LLMBatchTranslator) translateGroupWithContext(ctx context.Context, texts []string, prevContext, nextContext []string, runConfig TranslationRunConfig) ([]string, error) {
	if len(texts) == 0 {
		return []string{}, nil
	}

	var fullTexts []string
	targetStartIndex := 0

	if len(prevContext) > 0 {
		fullTexts = append(fullTexts, prevContext...)
		targetStartIndex = len(fullTexts)
	}

	fullTexts = append(fullTexts, texts...)
	targetEndIndex := len(fullTexts)

	if len(nextContext) > 0 {
		fullTexts = append(fullTexts, nextContext...)
	}

	contextInfo := ""
	if len(prevContext) > 0 || len(nextContext) > 0 {
		contextInfo = fmt.Sprintf(`

上下文信息：
- 前置上下文：%d 句（仅供参考，不需要翻译）
- 目标翻译：%d 句（位于第 %d-%d 句，需要全部翻译）
- 后置上下文：%d 句（仅供参考，不需要翻译）

请只翻译目标部分（第 %d-%d 句），但要充分考虑前后文的连贯性。`,
			len(prevContext), len(texts), targetStartIndex+1, targetEndIndex,
			len(nextContext), targetStartIndex+1, targetEndIndex)
	}

	systemPrompt := fmt.Sprintf(`你是一个专业的视频字幕翻译专家。我将给你一段连续的%s字幕，其中包含 %d 句需要翻译的内容。%s

翻译要求：
1. 自然流畅：使用口语化表达，符合%s字幕习惯
2. 上下文连贯：理解整体语境，确保翻译前后呼应
3. 准确传神：忠实原文含义，保持语气和情感
4. 简洁明了：字幕需要快速阅读，避免冗长
5. 数量严格：必须输出 %d 句翻译，不多不少
6. 分隔符：每句翻译用"%s"分隔

输入格式：句子用"%s"分隔
输出格式：只返回目标部分的%s翻译，用"%s"分隔

注意：只返回翻译的%s文本，不要添加序号、解释或其他内容。`,
		t.getLanguageName(runConfig.SourceLang),
		len(texts),
		contextInfo,
		t.getLanguageName(runConfig.TargetLang),
		len(texts),
		sentenceBreak,
		sentenceBreak,
		t.getLanguageName(runConfig.TargetLang),
		sentenceBreak,
		t.getLanguageName(runConfig.TargetLang))

	combinedText := strings.Join(fullTexts, "\n"+sentenceBreak+"\n")
	messages := []ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: combinedText},
	}

	translatedText, err := t.chatWithModel(ctx, messages, runConfig.ModelName)
	if err != nil {
		return nil, fmt.Errorf("llm chat failed: %w", err)
	}

	translatedSentences := strings.Split(translatedText, sentenceBreak)
	for index := range translatedSentences {
		translatedSentences[index] = strings.TrimSpace(translatedSentences[index])
	}
	translatedSentences = compactTexts(translatedSentences)

	if len(translatedSentences) != len(texts) {
		t.logger.Printf("translation count mismatch, fixing: expected=%d actual=%d", len(texts), len(translatedSentences))
		for len(translatedSentences) < len(texts) {
			translatedSentences = append(translatedSentences, "[翻译缺失]")
		}
		if len(translatedSentences) > len(texts) {
			translatedSentences = translatedSentences[:len(texts)]
		}
	}

	return translatedSentences, nil
}

func (t *LLMBatchTranslator) shouldTranslate(ctx context.Context, texts []string, runConfig TranslationRunConfig) (bool, string, error) {
	targetLang := strings.TrimSpace(strings.ToLower(runConfig.TargetLang))
	sourceLang := strings.TrimSpace(strings.ToLower(runConfig.SourceLang))
	if targetLang != "" && sourceLang != "" && sourceLang == targetLang {
		return false, sourceLang, nil
	}

	var sampleTexts []string
	for _, text := range texts {
		trimmed := strings.TrimSpace(text)
		if trimmed == "" {
			continue
		}
		sampleTexts = append(sampleTexts, trimmed)
		if len(sampleTexts) >= 8 {
			break
		}
	}
	if len(sampleTexts) == 0 {
		return false, "", nil
	}

	systemPrompt := fmt.Sprintf(`你是字幕翻译前的语言判定器。请判断给定字幕样本是否需要翻译成%s。

规则：
1. 如果字幕主体已经是目标语言，needs_translation=false。
2. 如果字幕主体不是目标语言，needs_translation=true。
3. 混合语言时，以主体语言为准。
4. 只输出 JSON，不要输出解释文字。

输出格式：{"needs_translation":true,"detected_language":"en","reason":"主体为英文"}`,
		t.getLanguageName(runConfig.TargetLang))

	messages := []ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: strings.Join(sampleTexts, "\n"+sentenceBreak+"\n")},
	}

	response, err := t.chatWithModel(ctx, messages, runConfig.ModelName)
	if err != nil {
		return false, "", fmt.Errorf("translation decision chat failed: %w", err)
	}

	var decision struct {
		NeedsTranslation bool   `json:"needs_translation"`
		DetectedLanguage string `json:"detected_language"`
		Reason           string `json:"reason"`
	}
	decoder := json.NewDecoder(strings.NewReader(extractJSON(response)))
	if err := decoder.Decode(&decision); err != nil {
		return false, "", fmt.Errorf("decode translation decision: %w", err)
	}

	return decision.NeedsTranslation, decision.DetectedLanguage, nil
}

func (t *LLMBatchTranslator) chatWithModel(ctx context.Context, messages []ChatMessage, modelName string) (string, error) {
	payloadMessages := make([]map[string]string, 0, len(messages))
	for _, message := range messages {
		payloadMessages = append(payloadMessages, map[string]string{
			"role":    message.Role,
			"content": message.Content,
		})
	}

	response, err := t.client.ChatCompletions(ctx, api2key.AIRequest{
		APIKey: t.config.APIKey,
		Body: map[string]any{
			"model":       modelName,
			"temperature": 0.2,
			"messages":    payloadMessages,
		},
	})
	if err != nil {
		return "", err
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := response.Decode(&result); err != nil {
		return "", err
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("chat completion returned no choices")
	}

	content := strings.TrimSpace(result.Choices[0].Message.Content)
	if content == "" {
		return "", fmt.Errorf("chat completion returned empty content")
	}
	return content, nil
}

func parseSRTFile(path string) ([]SubtitleItem, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read srt file: %w", err)
	}
	return parseSRT(string(raw))
}

func parseSRT(input string) ([]SubtitleItem, error) {
	normalized := strings.ReplaceAll(input, "\r\n", "\n")
	blocks := strings.Split(strings.TrimSpace(normalized), "\n\n")
	items := make([]SubtitleItem, 0, len(blocks))

	for _, block := range blocks {
		lines := strings.Split(strings.TrimSpace(block), "\n")
		if len(lines) < 3 {
			continue
		}
		index, err := strconv.Atoi(strings.TrimSpace(lines[0]))
		if err != nil {
			continue
		}
		timeParts := strings.SplitN(lines[1], " --> ", 2)
		if len(timeParts) != 2 {
			continue
		}
		text := strings.TrimSpace(strings.Join(lines[2:], " "))
		items = append(items, SubtitleItem{
			Index: index,
			Start: strings.TrimSpace(timeParts[0]),
			End:   strings.TrimSpace(timeParts[1]),
			Text:  text,
		})
	}

	return items, nil
}

func writeSRTFile(path string, items []SubtitleItem) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, item := range items {
		if _, err := fmt.Fprintf(writer, "%d\n%s --> %s\n%s\n\n", item.Index, item.Start, item.End, item.Text); err != nil {
			return fmt.Errorf("write output srt: %w", err)
		}
	}
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("flush output srt: %w", err)
	}
	return nil
}

func extractTexts(items []SubtitleItem) []string {
	texts := make([]string, 0, len(items))
	for _, item := range items {
		texts = append(texts, item.Text)
	}
	return texts
}

func compactTexts(texts []string) []string {
	result := make([]string, 0, len(texts))
	for _, text := range texts {
		trimmed := strings.TrimSpace(text)
		if trimmed == "" {
			continue
		}
		result = append(result, trimmed)
	}
	return result
}

func extractJSON(input string) string {
	start := strings.Index(input, "{")
	end := strings.LastIndex(input, "}")
	if start >= 0 && end > start {
		return input[start : end+1]
	}
	return input
}

func (t *LLMBatchTranslator) getLanguageName(code string) string {
	langMap := map[string]string{
		"en":      "英文",
		"zh-Hans": "中文",
		"zh-CN":   "中文",
		"zh":      "中文",
		"ja":      "日文",
		"ko":      "韩文",
		"es":      "西班牙文",
		"fr":      "法文",
		"de":      "德文",
		"ru":      "俄文",
		"ar":      "阿拉伯文",
		"pt":      "葡萄牙文",
		"it":      "意大利文",
		"auto":    "自动检测",
	}

	if name, ok := langMap[code]; ok {
		return name
	}
	return code
}

func getenv(key string, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getenvInt(key string, fallback int) int {
	value := strings.TrimSpace(getenv(key, ""))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getenvDuration(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(getenv(key, ""))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}
