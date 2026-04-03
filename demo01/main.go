package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/difyz9/api2key-go-sdk/api2key"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	baseURL := getenv("API2KEY_BASE_URL", api2key.DefaultBaseAPIURL)
	projectID := getenv("API2KEY_PROJECT_ID", "")
	serviceSecret := getenv("API2KEY_SERVICE_SECRET", "")
	provider := getenv("API2KEY_PROVIDER", "azure")
	locale := getenv("API2KEY_LOCALE", "zh-CN")
	voice := getenv("API2KEY_VOICE", "zh-CN-XiaoxiaoNeural")
	text := getenv("API2KEY_TEXT", "你好，这是 demo01 的集成语音调用。")
	outputPath := getenv("API2KEY_OUTPUT", filepath.Join("demo01", "output.mp3"))
	audioFile := getenv("API2KEY_AUDIO_FILE", outputPath)
	audioURL := getenv("API2KEY_AUDIO_URL", "")
	email := getenv("API2KEY_EMAIL", "")
	password := getenv("API2KEY_PASSWORD", "")

	if email == "" || password == "" {
		log.Fatal("API2KEY_EMAIL and API2KEY_PASSWORD are required")
	}

	options := []api2key.Option{
		api2key.WithBaseAPIURL(baseURL),
	}
	if serviceSecret != "" {
		options = append(options, api2key.WithServiceSecret(serviceSecret))
	}
	client := api2key.NewClient(options...)

	loginResult, err := client.Login(ctx, api2key.LoginRequest{
		Email:     email,
		Password:  password,
		ProjectID: projectID,
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("login ok")

	apiKeyResult, err := ensureDemoAPIKey(ctx, client, loginResult.AccessToken, "sdk-demo")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("api key ready:", apiKeyResult.Key.KeyPrefix)

	voices, err := client.ListVoices(ctx, api2key.ListVoicesRequest{
		ProjectID: projectID,
		APIKey:    apiKeyResult.Key.Secret,
		Provider:  provider,
		Locale:    locale,
		Search:    "Xiaoxiao",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("voices:", voices.Total)

	result, err := client.SaveSpeechToFile(ctx, api2key.SynthesizeSpeechRequest{
		ProjectID: projectID,
		APIKey:    apiKeyResult.Key.Secret,
		Provider:  provider,
		Text:      text,
		Voice:     voice,
		Locale:    locale,
		Rate:      1,
		Volume:    100,
		Pitch:     0,
		Format:    "audio-24khz-96kbitrate-mono-mp3",
	}, outputPath)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("speech saved:", outputPath)
	if strings.TrimSpace(result.StorageKey) != "" {
		downloaded, downloadErr := client.DownloadSpeechAudio(ctx, result.StorageKey)
		if downloadErr != nil {
			log.Fatal(downloadErr)
		}
		mirrorPath := strings.TrimSuffix(outputPath, filepath.Ext(outputPath)) + ".downloaded.mp3"
		if writeErr := os.WriteFile(mirrorPath, downloaded.Audio, 0o644); writeErr != nil {
			log.Fatal(writeErr)
		}
		fmt.Println("downloaded copy:", mirrorPath)
	}

	if strings.TrimSpace(audioURL) != "" {
		transcribeResult, err := client.AudioToSRT(ctx, api2key.ASRRequest{
			ProjectID:       projectID,
			APIKey:          apiKeyResult.Key.Secret,
			AudioURL:        audioURL,
			Provider:        "tencent",
			EngineModelType: "16k_zh",
			Async:           true,
		})
		if err != nil {
			log.Fatal(err)
		}

		taskResult, err := client.PollASRTaskWithOptions(ctx, api2key.ASRTaskQueryRequest{
			ProjectID: projectID,
			APIKey:    apiKeyResult.Key.Secret,
			TaskID:    fmt.Sprint(transcribeResult.TaskID),
			Provider:  "tencent",
		}, 2*time.Second, 30)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("asr status:", taskResult.StatusStr)
	} else if _, statErr := os.Stat(audioFile); statErr == nil {
		transcribeResult, err := client.AudioToSRT(ctx, api2key.ASRRequest{
			ProjectID:       projectID,
			APIKey:          apiKeyResult.Key.Secret,
			AudioFilePath:   audioFile,
			Provider:        "tencent",
			EngineModelType: "16k_zh",
			Async:           true,
		})
		if err != nil {
			log.Fatal(err)
		}

		taskResult, err := client.PollASRTaskWithOptions(ctx, api2key.ASRTaskQueryRequest{
			ProjectID: projectID,
			APIKey:    apiKeyResult.Key.Secret,
			TaskID:    fmt.Sprint(transcribeResult.TaskID),
			Provider:  "tencent",
		}, 2*time.Second, 30)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("asr status:", taskResult.StatusStr)
	} else if errors.Is(statErr, os.ErrNotExist) {
		log.Println("skip ASR demo: audio file not found")
	} else {
		log.Fatal(statErr)
	}

	if serviceSecret == "" {
		log.Println("skip credits demo: API2KEY_SERVICE_SECRET not set")
		return
	}

	creditsResult, err := client.SpendCredits(ctx, api2key.SpendCreditsRequest{
		UserID:      "user_123",
		Amount:      10,
		Service:     "ai_chat",
		TaskID:      "order_20260401_001",
		Description: "SDK 扣费测试",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("balance after:", creditsResult.BalanceAfter)
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func ensureDemoAPIKey(ctx context.Context, client *api2key.Client, accessToken, keyName string) (*api2key.CreateAPIKeyResponse, error) {
	created, err := client.CreateAPIKey(ctx, accessToken, api2key.CreateAPIKeyRequest{Name: keyName})
	if err == nil {
		return created, nil
	}

	var apiErr *api2key.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != 400 || apiErr.Message != "每个用户最多创建 10 个 API Key" {
		return nil, err
	}

	keys, listErr := client.ListAPIKeys(ctx, accessToken)
	if listErr != nil {
		return nil, fmt.Errorf("list api keys after create limit hit: %w", listErr)
	}
	if len(keys.Keys) == 0 {
		return nil, err
	}

	candidate := keys.Keys[len(keys.Keys)-1]
	for index := len(keys.Keys) - 1; index >= 0; index-- {
		if keys.Keys[index].Name == keyName {
			candidate = keys.Keys[index]
			break
		}
	}

	if deleteErr := client.DeleteAPIKey(ctx, accessToken, candidate.ID); deleteErr != nil {
		return nil, fmt.Errorf("delete stale api key %s: %w", candidate.ID, deleteErr)
	}

	return client.CreateAPIKey(ctx, accessToken, api2key.CreateAPIKeyRequest{Name: keyName})
}
