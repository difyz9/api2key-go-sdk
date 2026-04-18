package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/difyz9/api2key-go-sdk/api2key"
)

type config struct {
	BaseAPIURL string
	APIKey     string
	Model      string
	System     string
	Prompt     string
	Stream     bool
	Timeout    time.Duration
}

func main() {
	cfg := loadConfig()

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	client := api2key.NewClient(
		api2key.WithBaseAPIURL(cfg.BaseAPIURL),
	)

	if cfg.Stream {
		runStreamChat(ctx, client, cfg)
		return
	}

	runStandardChat(ctx, client, cfg)
}

func runStandardChat(ctx context.Context, client *api2key.Client, cfg config) {

	response, err := client.ChatCompletions(ctx, api2key.AIRequest{
		APIKey: cfg.APIKey,
		Body: map[string]any{
			"model": cfg.Model,
			"messages": []map[string]string{
				{
					"role":    "system",
					"content": cfg.System,
				},
				{
					"role":    "user",
					"content": cfg.Prompt,
				},
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	var result struct {
		ID      string `json:"id"`
		Model   string `json:"model"`
		Choices []struct {
			Index   int `json:"index"`
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage map[string]any `json:"usage"`
	}
	if err := response.Decode(&result); err != nil {
		log.Fatal(err)
	}

	if len(result.Choices) == 0 {
		log.Fatal("chat completion returned no choices")
	}

	message := strings.TrimSpace(result.Choices[0].Message.Content)
	if message == "" {
		log.Fatal("chat completion returned empty content")
	}

	fmt.Println("assistant:")
	fmt.Println(message)
	if result.Model != "" {
		fmt.Println()
		fmt.Println("model:", result.Model)
	}
	if len(result.Usage) > 0 {
		fmt.Println("usage:", result.Usage)
	}
}

func runStreamChat(ctx context.Context, client *api2key.Client, cfg config) {
	stream, err := client.ChatCompletionsStream(ctx, api2key.AIRequest{
		APIKey: cfg.APIKey,
		Body: map[string]any{
			"model":  cfg.Model,
			"stream": true,
			"messages": []map[string]string{
				{
					"role":    "system",
					"content": cfg.System,
				},
				{
					"role":    "user",
					"content": cfg.Prompt,
				},
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	defer stream.Body.Close()

	fmt.Println("assistant:")
	if err := printSSEContent(stream.Body); err != nil {
		log.Fatal(err)
	}
	fmt.Println()
}

func printSSEContent(body io.Reader) error {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	type streamPayload struct {
		Choices []struct {
			Delta struct {
				Content string `json:"content"`
			} `json:"delta"`
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "data:") {
			continue
		}

		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "[DONE]" {
			return nil
		}

		var chunk streamPayload
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			continue
		}
		for _, choice := range chunk.Choices {
			if content := strings.TrimSpace(choice.Delta.Content); content != "" {
				fmt.Print(content)
				continue
			}
			if content := strings.TrimSpace(choice.Message.Content); content != "" {
				fmt.Print(content)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read stream: %w", err)
	}
	return nil
}

func loadConfig() config {
	var cfg config
	flag.StringVar(&cfg.BaseAPIURL, "base-url", getenv("API2KEY_BASE_URL", api2key.DefaultBaseAPIURL), "base api url")
	flag.StringVar(&cfg.APIKey, "api-key", getenv("API2KEY_API_KEY", ""), "api key used for ai chat")
	flag.StringVar(&cfg.Model, "model", getenv("API2KEY_AI_MODEL", "openai/gpt-4o-mini"), "chat model id")
	flag.StringVar(&cfg.System, "system", getenv("API2KEY_AI_SYSTEM", "You are a concise helpful assistant."), "system prompt")
	flag.StringVar(&cfg.Prompt, "prompt", getenv("API2KEY_AI_PROMPT", "请用一句话介绍 api2key-go-sdk 的用途。"), "user prompt")
	flag.BoolVar(&cfg.Stream, "stream", getenvBool("API2KEY_AI_STREAM", false), "stream chat output")
	flag.DurationVar(&cfg.Timeout, "timeout", getenvDuration("API2KEY_TIMEOUT", 60*time.Second), "request timeout")
	flag.Parse()

	if strings.TrimSpace(cfg.APIKey) == "" {
		log.Fatal("API2KEY_API_KEY is required")
	}
	if strings.TrimSpace(cfg.Prompt) == "" {
		log.Fatal("prompt is required")
	}

	return cfg
}

func getenv(key string, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
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

func getenvBool(key string, fallback bool) bool {
	value := strings.TrimSpace(strings.ToLower(getenv(key, "")))
	if value == "" {
		return fallback
	}
	switch value {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}
