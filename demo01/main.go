package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/difyz9/api2key-go-sdk/api2key"
)

func main() {
	ctx := context.Background()
	//	os.Setenv("API2KEY_BASE_API_URL", "https://api2key-base-api-dev.guan-afred.workers.dev")
	os.Setenv("API2KEY_SERVICE_SECRET", "sk-afa4958c4760eca0e0acd17140beea8ae5ea17874e6948a90da719758961f670")
	serviceSecret := strings.TrimSpace(os.Getenv("API2KEY_SERVICE_SECRET"))
	options := []api2key.Option{
		api2key.WithBaseAPIURL("https://api2key-base-api-dev.guan-afred.workers.dev"),
	}
	if serviceSecret != "" {
		options = append(options, api2key.WithServiceSecret(serviceSecret))
	}
	client := api2key.NewClient(options...)

	loginResult, err := client.Login(ctx, api2key.LoginRequest{
		Email:    "hello@126.com",
		Password: "Ab123456",
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Login successful! Access Token: %s\n", loginResult.AccessToken)

	apiKeyResult, err := ensureDemoAPIKey(ctx, client, loginResult.AccessToken, "sdk-demo")
	if err != nil {
		log.Fatal(err)
	}

	voices, err := client.ListVoices(ctx, api2key.ListVoicesRequest{
		APIKey:   apiKeyResult.Key.Secret,
		Provider: "azure",
		Locale:   "zh-CN",
		Search:   "Xiaoxiao",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("voices:", voices.Total)

	_, err = client.SaveSpeechToFile(ctx, api2key.SynthesizeSpeechRequest{
		APIKey:   apiKeyResult.Key.Secret,
		Provider: "azure",
		Text:     "你好，这是 SDK 调用测试。",
		Voice:    "zh-CN-XiaoxiaoNeural",
		Locale:   "zh-CN",
		Rate:     1,
		Volume:   100,
		Pitch:    0,
		Format:   "audio-24khz-96kbitrate-mono-mp3",
	}, "output.mp3")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("speech saved: output.mp3")

	if _, statErr := os.Stat("output.mp3"); statErr == nil {
		transcribeResult, err := client.AudioToSRT(ctx, api2key.ASRRequest{
			APIKey:          apiKeyResult.Key.Secret,
			AudioFilePath:   "output.mp3",
			Provider:        "tencent",
			EngineModelType: "16k_zh",
			Async:           true,
		})
		if err != nil {
			log.Fatal(err)
		}

		taskResult, err := client.PollASRTask(ctx, apiKeyResult.Key.Secret, fmt.Sprint(transcribeResult.TaskID), 2*time.Second, 30)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("asr status:", taskResult.StatusStr)
	} else if errors.Is(statErr, os.ErrNotExist) {
		log.Println("skip ASR demo: output.mp3 not found")
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