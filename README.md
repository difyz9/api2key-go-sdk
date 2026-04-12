# api2key Go SDK

这个仓库提供一个可复用的 Go SDK，覆盖当前项目最常用的客户端能力：

1. 用户登录
2. 创建和管理 API Key
3. 获取语音列表
4. 文本转语音
5. 音频转写 / 生成 SRT
6. 异步 ASR 任务查询与轮询
7. 下载已存储的合成音频
8. 用户积分扣减 / 预扣 / 确认 / 取消
9. 直付支付创建 / 查询 / 轮询

## 安装与验证

当前仓库本身就是一个 Go module：

```bash
cd api2key-go-sdk
go test ./...
```

如果要在自己的项目里使用，直接引用 module 路径即可：

```go
import "github.com/difyz9/api2key-go-sdk/api2key"
```

## 快速开始

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/difyz9/api2key-go-sdk/api2key"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client := api2key.NewClient(
		api2key.WithBaseAPIURL("https://open.api2key.com"),
	)

	loginResult, err := client.Login(ctx, api2key.LoginRequest{
		Email:     "user@example.com",
		Password:  "Test123456!",
		ProjectID: "ytb2bili",
	})
	if err != nil {
		log.Fatal(err)
	}

	apiKeyResult, err := client.EnsureAPIKey(ctx, loginResult.AccessToken, api2key.CreateAPIKeyRequest{
		Name: "sdk-demo",
	})
	if err != nil {
		log.Fatal(err)
	}
	if !apiKeyResult.SecretAvailable {
		log.Fatal("API key already exists, but the historical secret cannot be queried again; persist it when creating the key for the first time")
	}

	voices, err := client.ListVoices(ctx, api2key.ListVoicesRequest{
		ProjectID: "ytb2bili",
		APIKey:    apiKeyResult.Secret,
		Provider:  "azure",
		Locale:    "zh-CN",
		Search:    "Xiaoxiao",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("voices:", voices.Total)

	result, err := client.SaveSpeechToFile(ctx, api2key.SynthesizeSpeechRequest{
		ProjectID: "ytb2bili",
		APIKey:    apiKeyResult.Secret,
		Provider:  "azure",
		Text:      "你好，这是 SDK 调用测试。",
		Voice:     "zh-CN-XiaoxiaoNeural",
		Locale:    "zh-CN",
		Rate:      1,
		Volume:    100,
		Pitch:     0,
		Format:    "audio-24khz-96kbitrate-mono-mp3",
		StorageKey: "video_123/index_0001.mp3",
		DownloadFilename: "index_0001.mp3",
	}, "output.mp3")
	if err != nil {
		log.Fatal(err)
	}

	if result.StorageKey != "" {
		downloaded, err := client.DownloadSpeechAudio(ctx, result.StorageKey)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("downloaded bytes:", len(downloaded.Audio))
	}

	transcribeResult, err := client.AudioToSRT(ctx, api2key.ASRRequest{
		ProjectID:       "ytb2bili",
		APIKey:          apiKeyResult.Secret,
		AudioFilePath:   "sample.wav",
		Provider:        "tencent",
		EngineModelType: "16k_zh",
		Async:           true,
	})
	if err != nil {
		log.Fatal(err)
	}

	taskResult, err := client.PollASRTaskWithOptions(ctx, api2key.ASRTaskQueryRequest{
		ProjectID: "ytb2bili",
		APIKey:    apiKeyResult.Secret,
		TaskID:    fmt.Sprint(transcribeResult.TaskID),
		Provider:  "tencent",
	}, 2*time.Second, 30)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("asr status:", taskResult.StatusStr)
	if taskResult.SRT != "" {
		fmt.Println(taskResult.SRT)
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

	directPayment, err := client.CreateDirectPayment(ctx, loginResult.AccessToken, api2key.DirectPaymentCreateRequest{
		Subject:     "SDK 直付测试",
		Amount:      0.01,
		Description: "Go SDK direct payment example",
		ProjectID:   "ytb2bili",
		PaymentType: "wechat",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("direct payment order:", directPayment.OrderNo)
	fmt.Println("direct payment qr code:", directPayment.Data.QRCode)
}
```

说明：

- TTS / ASR 相关接口现在直接调用 `api2key-base-api` 的内置语音路由。
- 生产环境只需要传 `https://open.api2key.com`，不需要再配置独立语音服务。
- `SynthesizeSpeechRequest` 支持 `StorageKey` 和 `DownloadFilename`，可把远端音频固定存成 `video_id/index_0001.mp3` 这种结构。
- `WithSpeechURL(...)` 是当前推荐的语音根路径覆盖项，适合本地调试、灰度环境或特殊部署。
- `WithTTSURL(...)` 仍然保留，作为兼容别名，不影响旧调用代码。
- `WithServiceSecret(...)` 只在调用积分接口时需要，单独调用 TTS / ASR 不需要。
- 积分相关接口现在要求显式传 `ProjectID`，因为服务端已切换到项目维度积分账户。
- 直付支付接口使用登录返回的 `accessToken`，不使用 `service secret` 或 `api key`。

## 自定义存储路径

如果你希望服务端返回稳定的下载地址，例如：

```text
https://open.api2key.com/api/files/download/video_123/index_0168.mp3
```

调用时传入：

```go
result, err := client.SynthesizeSpeech(ctx, api2key.SynthesizeSpeechRequest{
	ProjectID: "ytb2bili",
	APIKey:    apiKey,
	Provider:  "azure",
	Text:      "不要忽略基础知识。",
	Voice:     "zh-CN-XiaoxiaoNeural",
	Locale:    "zh-CN",
	Format:    "audio-24khz-96kbitrate-mono-mp3",
	StorageKey: "video_123/index_0168.mp3",
	DownloadFilename: "index_0168.mp3",
})
if err != nil {
	log.Fatal(err)
}

fmt.Println(result.StorageKey)
fmt.Println(result.DownloadURL)
```

下载时直接传 `StorageKey` 即可，SDK 会优先走新的路径式下载：

```go
downloaded, err := client.DownloadSpeechAudio(ctx, result.StorageKey)
```

## 可运行示例

如果你的目标只是“先获取用户的 apikey 列表，列表为空才创建一个；列表不为空就直接取一个返回，而且不要重复创建”，推荐直接使用 [ensure_apikey/main.go](/Users/apple/opt/difyz_0329/05/api2key-go-sdk/ensure_apikey/main.go) 这个独立案例：

1. 先登录拿 `accessToken`
2. 调用 `ListAPIKeys` 获取当前用户的 apikey 列表
3. 列表不为空时，直接取一个现有 key 返回，不再创建新的 key
4. 列表为空时，调用 `CreateAPIKey` 创建一个新的 key 并返回

注意：现在服务端列表接口会返回 `secret`。这个案例会优先复用“列表里已有明文 secret 的 key”；如果现有 key 都没有明文 `secret`，就会新创建一个 key，并返回新 key 的完整 apikey。

仓库里有四个示例：

- `ensure_apikey/main.go`：独立 apikey 案例，先查用户 apikey 列表，有就直接取一个，没有才创建，避免重复创建。
- `example/main.go`：通用 CLI 风格示例，适合串联登录、建 key、查 voices、做 speech / SRT / credits。
- `demo01/main.go`：更短的烟雾测试示例，默认会跑登录、建 key、语音合成和一次 ASR 轮询。
- `subtitle_tts/main.go`：只依赖 `baseURL + apiKey` 的字幕转音频示例，适合直接把 `.srt` 或 `.txt` 文本合成音频。
- `subtitle_tts/main.go` 默认会把远端存储路径组织成 `<video-id>/index_0001.mp3` 这种格式。

先进入仓库根目录：

```bash
cd api2key-go-sdk
```

只跑独立的 apikey 案例：

```bash
API2KEY_EMAIL=user@example.com \
API2KEY_PASSWORD='Test123456!' \
API2KEY_KEY_NAME='sdk-example-key' \
go run ./ensure_apikey
```

如果你想指定新建时使用的 key 名称和项目：

```bash
API2KEY_EMAIL=user@example.com \
API2KEY_PASSWORD='Test123456!' \
API2KEY_KEY_NAME='sdk-example-key' \
API2KEY_PROJECT_ID='your-project-id' \
go run ./ensure_apikey
```

通用 CLI 示例仍然保留在 `example/main.go`，只跑登录、创建 key、查询语音列表：

```bash
API2KEY_EMAIL=user@example.com \
API2KEY_PASSWORD='Test123456!' \
go run ./example
```

连同语音合成一起跑：

```bash
API2KEY_EMAIL=user@example.com \
API2KEY_PASSWORD='Test123456!' \
API2KEY_PROJECT_ID=ytb2bili \
API2KEY_VIDEO_ID=video_123 \
API2KEY_EXAMPLE_DO_SPEECH=true \
go run ./example -output ./example/output.mp3
```

如果希望显式指定远端路径，也可以直接传：

```bash
API2KEY_EMAIL=user@example.com \
API2KEY_PASSWORD='Test123456!' \
API2KEY_PROJECT_ID=ytb2bili \
API2KEY_EXAMPLE_DO_SPEECH=true \
API2KEY_STORAGE_KEY='video_123/index_0168.mp3' \
API2KEY_DOWNLOAD_FILENAME='index_0168.mp3' \
go run ./example -output ./example/output.mp3
```

连同 SRT 转写一起跑：

```bash
API2KEY_EMAIL=user@example.com \
API2KEY_PASSWORD='Test123456!' \
API2KEY_PROJECT_ID=ytb2bili \
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

连同直付支付一起跑：

```bash
API2KEY_EMAIL=user@example.com \
API2KEY_PASSWORD='Test123456!' \
API2KEY_PROJECT_ID=ytb2bili \
API2KEY_EXAMPLE_DO_DIRECT_PAY=true \
API2KEY_PAYMENT_AMOUNT=0.01 \
go run ./example
```

创建后立即轮询支付状态：

```bash
API2KEY_EMAIL=user@example.com \
API2KEY_PASSWORD='Test123456!' \
API2KEY_PROJECT_ID=ytb2bili \
API2KEY_EXAMPLE_DO_DIRECT_PAY=true \
API2KEY_EXAMPLE_POLL_DIRECT_PAY=true \
API2KEY_PAYMENT_AMOUNT=0.01 \
go run ./example
```

运行更短的 smoke demo：

```bash
API2KEY_EMAIL=user@example.com \
API2KEY_PASSWORD='Test123456!' \
API2KEY_PROJECT_ID=ytb2bili \
API2KEY_VIDEO_ID=demo-video \
API2KEY_SPEECH_URL=https://open.api2key.com \
go run ./demo01
```

如果已经有现成 API Key，也可以直接跳过登录，只传入口地址和 key：

```go
ctx := context.Background()
client := api2key.NewClient(
	api2key.WithBaseAPIURL("https://open.api2key.com"),
)

voices, err := client.ListVoices(ctx, api2key.ListVoicesRequest{
	ProjectID: "ytb2bili",
	APIKey:    "sk-your-api-key",
	Provider:  "azure",
	Locale:    "zh-CN",
})
if err != nil {
	log.Fatal(err)
}
_ = voices
```

## 字幕文本合成音频案例

这个案例只需要两个核心参数：

- `baseURL`
- `apiKey`

不需要再单独配置第二个 TTS URL。SDK 会基于 `WithBaseAPIURL(...)` 自动推导语音路由。

示例源码见 `subtitle_tts/main.go`。

### 适用输入

- `.srt` 字幕文件：会自动跳过序号行和时间轴行
- `.txt` 纯文本：会按正文直接合成

### 最小示例

```go
package main

import (
	"context"
	"log"
	"time"

	"github.com/difyz9/api2key-go-sdk/api2key"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	client := api2key.NewClient(
		api2key.WithBaseAPIURL("https://open.api2key.com"),
	)

	_, err := client.SaveSpeechToFile(ctx, api2key.SynthesizeSpeechRequest{
		APIKey:   "sk-your-api-key",
		Provider: "azure",
		Text:     "第一句字幕。\n第二句字幕。\n第三句字幕。",
		Voice:    "zh-CN-XiaoxiaoNeural",
		Locale:   "zh-CN",
		Format:   "audio-24khz-96kbitrate-mono-mp3",
	}, "output.mp3")
	if err != nil {
		log.Fatal(err)
	}
}
```

### 直接运行仓库示例

先准备一个字幕文件，例如 `subtitle_tts/subtitle.srt`。

PowerShell:

```powershell
$env:API2KEY_BASE_URL = "https://open.api2key.com"
$env:API2KEY_API_KEY = "sk-your-api-key"
$env:API2KEY_INPUT = "./subtitle_tts/subtitle.srt"
$env:API2KEY_OUTPUT = "./subtitle_tts/output.mp3"
go run ./subtitle_tts
```

也可以显式指定发音人、语言和项目：

```powershell
$env:API2KEY_BASE_URL = "https://open.api2key.com"
$env:API2KEY_API_KEY = "sk-your-api-key"
$env:API2KEY_PROJECT_ID = "your-project-id"
$env:API2KEY_PROVIDER = "azure"
$env:API2KEY_VOICE = "zh-CN-XiaoxiaoNeural"
$env:API2KEY_LOCALE = "zh-CN"
$env:API2KEY_INPUT = "./subtitle_tts/subtitle.srt"
$env:API2KEY_OUTPUT = "./subtitle_tts/output.mp3"
go run ./subtitle_tts
```

运行成功后会输出结果文件路径，以及本次实际使用的 provider、voice、format 和扣费信息。

## API 概览

### 认证与 API Key

- `Login`
- `CreateAPIKey`
- `ListAPIKeys`
- `UpdateAPIKey`
- `DeleteAPIKey`
- `LoginAndCreateAPIKey`

### 语音与转写

- `ListVoices`
- `SynthesizeSpeech`
- `SaveSpeechToFile`
- `TranscribeAudio`
- `AudioToSRT`
- `GetASRTask`
- `GetASRTaskWithOptions`
- `PollASRTask`
- `PollASRTaskWithOptions`
- `DownloadSpeechAudio`

### 积分

- `SpendCredits`
- `ReserveCredits`
- `ConfirmCredits`
- `CancelCredits`

## 设计说明

SDK 目前只依赖 Go 标准库，方便直接集成到业务服务中。错误统一返回 `APIError`，可以通过 `errors.As` 读取 `StatusCode`、`Code` 和 `Balance`。