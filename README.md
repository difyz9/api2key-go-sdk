# api2key Go SDK

这个目录提供一个可复用的 Golang SDK，覆盖当前项目最常用的客户端能力：

1. 用户登录
2. 创建 API Key
3. 获取语音列表
4. 音频合成
5. 音频转文字 / 转 SRT
6. 异步 ASR 任务查询与轮询
7. 用户积分扣减 / 预扣 / 确认 / 取消

## 安装方式

当前仓库内使用本地 module：

```bash
cd examples/go/sdk
go test ./...
```

如果你要放到自己的 Go 项目中，推荐直接复制 `api2key` 目录，或者后续再独立发布为单独仓库。

## 包路径

```go
import "api2key-go-sdk/api2key"
```

## 快速开始

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"api2key-go-sdk/api2key"
)

func main() {
	ctx := context.Background()
	client := api2key.NewClient(
		api2key.WithBaseAPIURL("https://open.api2key.com"),
		api2key.WithTTSURL("https://tts.api2key.com"),
		api2key.WithServiceSecret("your-service-secret"),
	)

	loginResult, err := client.Login(ctx, api2key.LoginRequest{
		Email:    "user@example.com",
		Password: "Test123456!",
	})
	if err != nil {
		log.Fatal(err)
	}

	apiKeyResult, err := client.CreateAPIKey(ctx, loginResult.AccessToken, api2key.CreateAPIKeyRequest{
		Name: "sdk-demo",
	})
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

	transcribeResult, err := client.AudioToSRT(ctx, api2key.ASRRequest{
		APIKey:          apiKeyResult.Key.Secret,
		AudioFilePath:   "sample.wav",
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
```

	## 可运行示例

	仓库里带了一个可直接运行的示例程序：

	- `example/main.go`

	先进入目录：

	```bash
	cd examples/go/sdk
	```

	只跑登录、创建 key、查询语音列表：

	```bash
	API2KEY_EMAIL=user@example.com \
	API2KEY_PASSWORD='Test123456!' \
	go run ./example
	```

	连同语音合成一起跑：

	```bash
	API2KEY_EMAIL=user@example.com \
	API2KEY_PASSWORD='Test123456!' \
	API2KEY_EXAMPLE_DO_SPEECH=true \
	go run ./example -output ./example/output.mp3
	```

	连同 SRT 转写一起跑：

	```bash
	API2KEY_EMAIL=user@example.com \
	API2KEY_PASSWORD='Test123456!' \
	API2KEY_EXAMPLE_DO_SRT=true \
	API2KEY_AUDIO_FILE=/path/to/audio.wav \
	go run ./example
	```

	连同积分扣减一起跑：

	```bash
	API2KEY_EMAIL=user@example.com \
	API2KEY_PASSWORD='Test123456!' \
	API2KEY_SERVICE_SECRET=your-service-secret \
	API2KEY_EXAMPLE_DO_CREDITS=true \
	API2KEY_CREDITS_USER_ID=user_123 \
	go run ./example
	```

	这个示例默认总会执行：登录、创建 API Key、查询语音列表。其余动作通过开关控制：

	- `API2KEY_EXAMPLE_DO_SPEECH=true`
	- `API2KEY_EXAMPLE_DO_SRT=true`
	- `API2KEY_EXAMPLE_DO_CREDITS=true`

## API 概览

### 认证与 API Key

- `Login`
- `CreateAPIKey`
- `LoginAndCreateAPIKey`

### 语音与转写

- `ListVoices`
- `SynthesizeSpeech`
- `SaveSpeechToFile`
- `TranscribeAudio`
- `AudioToSRT`
- `GetASRTask`
- `PollASRTask`

### 积分

- `SpendCredits`
- `ReserveCredits`
- `ConfirmCredits`
- `CancelCredits`

## 设计说明

SDK 目前只依赖 Go 标准库，方便你直接拷到业务服务中使用。错误统一返回 `APIError`，可以通过 `errors.As` 读取 `StatusCode`、`Code` 和 `Balance`。