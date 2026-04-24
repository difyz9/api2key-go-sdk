
# api2key Go SDK

这个仓库提供一个可复用的 Go SDK，覆盖当前项目最常用的客户端能力：

1. 用户登录
2. 创建和管理 API Key
3. AI 模型列表 / AI 余额查询
4. OpenAI-compatible 在线会话
5. Anthropic Messages 在线会话
6. Gemini GenerateContent 在线会话
7. AI 会话历史读写
8. 获取语音列表
9. 文本转语音
10. 音频转写 / 生成 SRT
11. 异步 ASR 任务查询与轮询
12. 下载已存储的合成音频
13. 用户积分扣减 / 预扣 / 确认 / 取消
14. 直付支付创建 / 查询 / 轮询

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

建议先按三类场景理解 Go SDK：

1. 用户态：`LoginRequest.ProjectID` 现在是可选的，后端会优先使用当前登录项目，再回落到用户绑定项目或默认项目。
2. API Key 态：适合语音、AI、以及现在已经支持的“基于 API key 直接扣减积分”。
3. API Key 态：语音、AI、以及积分扣减都可以直接使用 API key，不需要 `service secret`。

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
		ProjectID: "ytb2bili", // 可选；直付充值场景可以不传
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

	balanceBefore, err := client.GetCreditsBalanceWithOptions(ctx, api2key.GetCreditsBalanceRequest{
		AccessToken: loginResult.AccessToken,
	})
	if err != nil {
		log.Fatal(err)
	}

	creditsResult, err := client.DeductCredits(ctx, api2key.DeductCreditsRequest{
		APIKey:      apiKeyResult.Key.Secret,
		Amount:      10,
		Service:     "ai_chat",
		TaskID:      "order_20260401_001",
		Description: "SDK 扣费测试",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("balance after:", creditsResult.BalanceAfter)
	fmt.Println("user balance before:", balanceBefore.Balance)

	directPayment, err := client.CreateDirectPayment(ctx, loginResult.AccessToken, api2key.DirectPaymentCreateRequest{
		Subject:     "SDK 直付测试",
		Amount:      0.01,
		Description: "Go SDK direct payment example",
		ProjectID:   "ytb2bili", // 可选；不传时由服务端自动回落默认项目
		PaymentType: "wechat",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("direct payment order:", directPayment.OrderNo)
	fmt.Println("direct payment qr code:", directPayment.Data.QRCode)
}
```

## 直付充值：不依赖项目必填

`/payment/unified/direct/create` 已经对齐到 SDK，`projectId` 现在是可选参数。
如果后端已经为用户配置了当前项目、绑定项目或默认项目，可以直接不传 `ProjectID`：

```go
payment, err := client.CreateDirectPayment(ctx, loginResult.AccessToken, api2key.DirectPaymentCreateRequest{
	Subject:     "账户直充",
	Amount:      0.01,
	Description: "不依赖显式项目 ID 的直付充值",
	PaymentType: api2key.DefaultDirectPaymentType,
})
if err != nil {
	log.Fatal(err)
}

fmt.Println(payment.OrderNo)
fmt.Println(payment.Data.QRCode)
```

命令行示例：

```bash
API2KEY_BASE_URL=https://stage.api2key.com \
API2KEY_EMAIL=your-email@example.com \
API2KEY_PASSWORD='your-password' \
API2KEY_EXAMPLE_DO_DIRECT_PAY=1 \
go run example/main.go
```

如果你明确要锁定某个项目，再额外传 `API2KEY_PROJECT_ID` 即可。

## API Key 直接扣减积分

`/credits/deduct` 已经补进 SDK，对应方法是 `DeductCredits`。它不再要求 `SERVICE_SECRET`，只需要 API key：

```go
resp, err := client.DeductCredits(ctx, api2key.DeductCreditsRequest{
	APIKey:      apiKey,
	Amount:      10,
	Service:     "manual-test",
	TaskID:      "manual-deduct-001",
	Description: "SDK deduct example",
})
if err != nil {
	log.Fatal(err)
}

fmt.Println(resp.BalanceAfter)
fmt.Println(resp.Idempotent)
```

完整可运行案例见 [credits_ledger/main.go](/Users/apple/opt/difyz_0329/0424/api2key-go-sdk/credits_ledger/main.go)。

命令行示例：

```bash
API2KEY_BASE_URL=https://stage.api2key.com \
API2KEY_API_KEY=your-api-key \
go run credits_ledger/main.go
```

## 获取当前用户与项目上下文

SDK 已补齐 `GET /auth/me`，方法是 `GetMe`。这个接口支持 JWT 或 API key：

```go
me, err := client.GetMe(ctx, api2key.GetMeRequest{
	AccessToken: loginResult.AccessToken,
})
if err != nil {
	log.Fatal(err)
}

fmt.Println(me.User.Email)
if me.Scope.ProjectSlug != nil {
	fmt.Println(*me.Scope.ProjectSlug)
}
```

如果用 API key 调用：

```go
me, err := client.GetMe(ctx, api2key.GetMeRequest{
	APIKey: apiKey,
})
```

## AI 在线会话

AI 相关能力已经对齐到 Go SDK，直接对应服务端这些路由：

- `/api/v1/ai/models`
- `/api/v1/ai/balance`
- `/api/v1/ai/chat/completions`
- `/api/v1/ai/completions`
- `/api/v1/ai/anthropic/v1/messages`
- `/api/v1/ai/google/v1beta/models/{model}:generateContent`
- `/api/v1/ai/histories`
- `/api/v1/ai/history`

如果你想把 SDK 作为自己的业务基类来“继承”，Go 里的推荐方式是嵌入 `*api2key.AISession`：

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/difyz9/api2key-go-sdk/api2key"
)

type ChatService struct {
	*api2key.AISession
}

func NewChatService(projectID, apiKey string) *ChatService {
	client := api2key.NewClient(
		api2key.WithBaseAPIURL("https://open.api2key.com"),
	)
	return &ChatService{
		AISession: api2key.NewAISession(
			client,
			api2key.WithAISessionProjectID(projectID),
			api2key.WithAISessionAPIKey(apiKey),
		),
	}
}

func (s *ChatService) Reply(ctx context.Context, prompt string) (string, error) {
	resp, err := s.ChatCompletions(ctx, map[string]any{
		"model": "openai/gpt-4o-mini",
		"messages": []map[string]string{
			{"role": "system", "content": "You are a helpful assistant."},
			{"role": "user", "content": prompt},
		},
	})
	if err != nil {
		return "", err
	}

	var data struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := resp.Decode(&data); err != nil {
		return "", err
	}
	if len(data.Choices) == 0 {
		return "", nil
	}
	return data.Choices[0].Message.Content, nil
}

func main() {
	service := NewChatService("ytb2bili", "your-api-key")
	content, err := service.Reply(context.Background(), "给我一句简短的 Go 建议")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(content)
}
```

流式会话同样可直接使用：

```go
stream, err := session.ChatCompletionsStream(ctx, map[string]any{
	"model": "openai/gpt-4o-mini",
	"stream": true,
	"messages": []map[string]string{{
		"role": "user",
		"content": "请流式输出一段自我介绍",
	}},
})
if err != nil {
	log.Fatal(err)
}
defer stream.Body.Close()

raw, err := io.ReadAll(stream.Body)
if err != nil {
	log.Fatal(err)
}
fmt.Println(string(raw))
```

会话历史接口也可以跟 `AISession` 一起使用：

```go
_, err = session.PutHistory(ctx, []api2key.AIHistoryMessage{
	{
		ID:        "msg-1",
		Role:      "user",
		Content:   "你好",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	},
}, "conversation-1")
if err != nil {
	log.Fatal(err)
}

history, err := session.GetHistory(ctx, "conversation-1")
if err != nil {
	log.Fatal(err)
}
fmt.Println(len(history.Messages))
```

说明：

- `AIResponse` 会保留原始 JSON，不强绑定某个模型厂商的响应结构。
- `AIStreamResponse` 会直接返回可读取的 `io.ReadCloser`，适合 SSE / 流式文本透传。
- `AISession` 会自动复用默认的 `projectId`、`apiKey` 或 `accessToken`，适合你在业务侧嵌入后继续扩展自己的方法。
- 如果需要 Anthropic 或 Gemini，只需调用 `AnthropicMessages`、`AnthropicMessagesStream`、`GoogleGenerateContent`，请求体保持各自官方格式即可。

只用 `baseUrl + apiKey` 直接发起 AI 对话的最小案例已经放在 `ai_chat/main.go`：

```bash
API2KEY_BASE_URL=https://stage.api2key.com \
API2KEY_API_KEY=your-api-key \
API2KEY_AI_MODEL=openai/gpt-4o-mini \
API2KEY_AI_PROMPT='请介绍一下 api2key 的能力' \
go run ./ai_chat
```

这个案例不需要登录，不需要 email/password，也不需要额外创建 session；前提是你传入的 API Key 已经可用于 AI 对话。

如果你想直接看流式输出：

```bash
API2KEY_BASE_URL=https://stage.api2key.com \
API2KEY_API_KEY=your-api-key \
API2KEY_AI_MODEL=openai/gpt-4o-mini \
API2KEY_AI_PROMPT='请流式输出一段简短介绍' \
API2KEY_AI_STREAM=true \
go run ./ai_chat
```

说明：

- TTS / ASR 相关接口现在直接调用 `api2key-api-stage` 的内置语音路由，默认与主 API 共用同一个根域名。
- stage 环境默认统一使用 `https://stage.api2key.com`，不需要再配置独立的 `tts.api2key.com`。
- 生产环境只需要传 `https://open.api2key.com`，也不需要再配置独立语音服务。
- `SynthesizeSpeechRequest` 支持 `StorageKey` 和 `DownloadFilename`，可把远端音频固定存成 `video_id/index_0001.mp3` 这种结构。
- `WithSpeechURL(...)` 是当前推荐的语音根路径覆盖项，适合本地调试、灰度环境或特殊部署。
- `WithTTSURL(...)` 仍然保留，作为兼容别名，不影响旧调用代码。
- 用户侧登录后，`CreateAPIKey` / `EnsureAPIKey` 默认跟随当前 JWT 中的项目上下文，不需要再额外显式传 `ProjectID`。
- 用户态积分查询现在同时支持 `AccessToken` 和 `APIKey`；推荐用 `GetCreditsBalanceWithOptions(...)` 与 `GetLedger(...)` 传入二者之一。
- 积分扣减接口推荐使用 `DeductCredits(...)` 配合 `APIKey` 或 `AccessToken`，不再依赖 `service secret`。
- 直付支付接口使用登录返回的 `accessToken`，不使用 `service secret` 或 `api key`。

## 最小调用模型

### 1. 用户态

用户态最小闭环：

1. 登录时显式传 `ProjectID`
2. 服务端返回带 `projectId` 的 token
3. `CreateAPIKey` / `EnsureAPIKey` 默认跟随该 token 项目

最小示例：

```go
loginResult, err := client.Login(ctx, api2key.LoginRequest{
	Email:     "user@example.com",
	Password:  "Test123456!",
	ProjectID: "ytb2bili",
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
```

### 2. API Key 扣减积分

API key 最小闭环：

1. 登录并创建 API key，或者直接使用已有 API key
2. 调用 `DeductCredits`

最小示例：

```go
_, err = client.DeductCredits(ctx, api2key.DeductCreditsRequest{
	APIKey:  apiKey,
	Amount:  10,
	Service: "ai_chat",
})
if err != nil {
	log.Fatal(err)
}
```

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

如果你的目标只是“先获取用户的 apikey 列表，列表为空才创建一个；列表不为空就直接取一个返回，而且不要重复创建”，推荐直接使用 [ensure_apikey/main.go](/Users/apple/opt/difyz_0329/0413/api2key-go-sdk/ensure_apikey/main.go) 这个独立案例：

1. 先登录拿 `accessToken`
2. 调用 `ListAPIKeys` 获取当前用户的 apikey 列表
3. 列表不为空时，直接取一个现有 key 返回，不再创建新的 key
4. 列表为空时，调用 `CreateAPIKey` 创建一个新的 key 并返回

注意：现在服务端列表接口会返回 `secret`。这个案例会优先复用“列表里已有明文 secret 的 key”；如果现有 key 都没有明文 `secret`，就会新创建一个 key，并返回新 key 的完整 apikey。

仓库里有多个可运行示例：

- `ensure_apikey/main.go`：独立 apikey 案例，先查用户 apikey 列表，有就直接取一个，没有才创建，避免重复创建。
- `apikey_credits/main.go`：纯 API key 查询积分案例，不走登录，直接查询余额和最近流水。
- `ai_chat/main.go`：最小 AI 对话示例，只需要 `baseUrl + apiKey`。
- `demo02/main.go`：字幕翻译示例，直接读取 `.srt`，使用 SDK 封装的 AI chat 按批次翻译后输出新的 `.srt`。
- `example/main.go`：通用 CLI 风格示例，适合串联登录、建 key、查 voices、做 speech / SRT / credits。
- `demo01/main.go`：更短的烟雾测试示例，默认会跑登录、建 key、语音合成和一次 ASR 轮询。
- `subtitle_tts/main.go`：登录后自动加载或创建用户 API Key，再把 `.srt` 或 `.txt` 文本逐段合成音频，并打印积分余额前后变化。
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

运行 `ensure_apikey` 时，项目 ID 现在是登录必填项：

```bash
API2KEY_EMAIL=user@example.com \
API2KEY_PASSWORD='Test123456!' \
API2KEY_KEY_NAME='sdk-example-key' \
API2KEY_PROJECT_ID='your-project-id' \
go run ./ensure_apikey
```

如果你的目标只是“我已经有 sk- 开头的用户 API key，现在只想查积分余额和最近流水”，直接运行这个最小案例：

```bash
API2KEY_BASE_URL=https://open.api2key.com \
API2KEY_API_KEY='sk-your-api-key' \
go run ./apikey_credits
```

如果该 API key 绑定了项目，通常不需要再显式传 `API2KEY_PROJECT_ID`。如果你想在 JWT 场景或未锁项目场景下指定流水过滤项目，也可以追加：

```bash
API2KEY_BASE_URL=https://open.api2key.com \
API2KEY_API_KEY='sk-your-api-key' \
API2KEY_PROJECT_ID='ytb2bili' \
go run ./apikey_credits
```

如果你的目标是“直接把现有英文字幕翻译成中文字幕”，可以运行 `demo02`：

```bash
API2KEY_BASE_URL=https://stage.api2key.com \
API2KEY_API_KEY='sk-your-api-key' \
API2KEY_AI_MODEL='openai/gpt-4o-mini' \
DEMO02_INPUT=./demo02/tp7Ojf7dPVI.srt \
DEMO02_OUTPUT=./demo02/tp7Ojf7dPVI.zh-CN.srt \
DEMO02_SOURCE_LANG='en' \
DEMO02_TARGET_LANG='zh-CN' \
go run ./demo02
```

这个案例会先用模型判断是否真的需要翻译，再按批次和前后文做字幕翻译，最后输出新的 `.srt` 文件。

通用 CLI 示例仍然保留在 `example/main.go`，只跑登录、创建 key、查询语音列表：

```bash
API2KEY_EMAIL=user@example.com \
API2KEY_PASSWORD='Test123456!' \
API2KEY_PROJECT_ID='your-project-id' \
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
API2KEY_PROJECT_ID=ytb2bili \
API2KEY_EXAMPLE_DO_CREDITS=true \
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

这个案例默认走用户态闭环：

1. 邮箱登录
2. 自动查询或创建同名 API Key
3. 逐段把字幕文本合成音频
4. 打印用户积分余额前后变化，用于确认 TTS 是否消耗积分

如果你已经有现成 API Key，也可以直接传 `API2KEY_API_KEY` 跳过登录。

示例源码见 `subtitle_tts/main.go`。

仓库里已经放了可直接 `source` 的模板：[subtitle_tts/.env.example](/Users/apple/opt/difyz_0329/0412/api2key-go-sdk/subtitle_tts/.env.example)。

在 macOS / Linux 下，可以直接这样运行：

```bash
cd api2key-go-sdk
set -a
source ./subtitle_tts/.env.example
set +a
go run ./subtitle_tts
```

如果你已经有现成的 `sk-...`，只要把模板里的 `API2KEY_API_KEY` 填上，同时保留 `API2KEY_PROJECT_ID`，示例就会跳过登录，直接执行字幕合成。

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

	loginResult, err := client.Login(ctx, api2key.LoginRequest{
		Email:     "user@example.com",
		Password:  "Test123456!",
		ProjectID: "your-project-id",
	})
	if err != nil {
		log.Fatal(err)
	}

	ensured, err := client.EnsureAPIKey(ctx, loginResult.AccessToken, api2key.CreateAPIKeyRequest{Name: "subtitle-tts"})
	if err != nil {
		log.Fatal(err)
	}

	balanceBefore, err := client.GetCreditsBalanceWithOptions(ctx, api2key.GetCreditsBalanceRequest{
		AccessToken: loginResult.AccessToken,
	})
	if err != nil {
		log.Fatal(err)
	}

	_, err = client.SaveSpeechToFile(ctx, api2key.SynthesizeSpeechRequest{
		ProjectID: "your-project-id",
		APIKey:    ensured.Secret,
		Provider: "azure",
		Text:     "第一句字幕。\n第二句字幕。\n第三句字幕。",
		Voice:    "zh-CN-XiaoxiaoNeural",
		Locale:   "zh-CN",
		Format:   "audio-24khz-96kbitrate-mono-mp3",
	}, "output.mp3")
	if err != nil {
		log.Fatal(err)
	}

	balanceAfter, err := client.GetCreditsBalanceWithOptions(ctx, api2key.GetCreditsBalanceRequest{
		AccessToken: loginResult.AccessToken,
	})
	if err != nil {
		log.Fatal(err)
	}

如果你已经有用户 API Key，也可以直接查询积分，无需先登录：

```go
balance, err := client.GetCreditsBalanceWithOptions(ctx, api2key.GetCreditsBalanceRequest{
	APIKey: "sk-xxx",
})
if err != nil {
	log.Fatal(err)
}

ledger, err := client.GetLedger(ctx, api2key.GetLedgerRequest{
	APIKey:    "sk-xxx",
	Page:      1,
	Size:      20,
	ProjectID: "ytb2bili",
})
if err != nil {
	log.Fatal(err)
}
fmt.Println(balance.Balance, ledger.Pagination.Total)
```
	log.Println("credits delta:", balanceBefore.Balance-balanceAfter.Balance)
}
```

### 直接运行仓库示例

先准备一个字幕文件，例如 `subtitle_tts/001.srt`，或者直接修改模板里的 `API2KEY_INPUT`。

PowerShell:

```powershell
$env:API2KEY_BASE_URL = "https://open.api2key.com"
$env:API2KEY_EMAIL = "user@example.com"
$env:API2KEY_PASSWORD = "Test123456!"
$env:API2KEY_PROJECT_ID = "your-project-id"
$env:API2KEY_KEY_NAME = "subtitle-tts"
$env:API2KEY_INPUT = "./subtitle_tts/subtitle.srt"
$env:API2KEY_OUTPUT = "./subtitle_tts/output"
go run ./subtitle_tts
```

也可以显式指定发音人、语言和远端视频目录：

```powershell
$env:API2KEY_BASE_URL = "https://open.api2key.com"
$env:API2KEY_EMAIL = "user@example.com"
$env:API2KEY_PASSWORD = "Test123456!"
$env:API2KEY_PROJECT_ID = "your-project-id"
$env:API2KEY_KEY_NAME = "subtitle-tts"
$env:API2KEY_PROVIDER = "azure"
$env:API2KEY_VOICE = "zh-CN-XiaoxiaoNeural"
$env:API2KEY_LOCALE = "zh-CN"
$env:API2KEY_VIDEO_ID = "video_123"
$env:API2KEY_INPUT = "./subtitle_tts/subtitle.srt"
$env:API2KEY_OUTPUT = "./subtitle_tts/output"
go run ./subtitle_tts
```

如果想跳过登录，也可以显式传一个现成 API Key：

```powershell
$env:API2KEY_BASE_URL = "https://open.api2key.com"
$env:API2KEY_PROJECT_ID = "your-project-id"
$env:API2KEY_API_KEY = "sk-your-api-key"
$env:API2KEY_INPUT = "./subtitle_tts/subtitle.srt"
$env:API2KEY_OUTPUT = "./subtitle_tts/output"
go run ./subtitle_tts
```

运行成功后会输出每段音频文件路径、本次实际使用的 provider、voice、format、每段 `charged` 信息，以及登录态下的积分余额前后变化。

## API 概览

### 认证与 API Key

- `Login`
- `GetCreditsBalance`
- `GetCreditsBalanceWithOptions`
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

- `GetCreditsBalance`
- `GetCreditsBalanceWithOptions`
- `GetLedger`
- `DeductCredits`

## 设计说明

SDK 目前只依赖 Go 标准库，方便直接集成到业务服务中。错误统一返回 `APIError`，可以通过 `errors.As` 读取 `StatusCode`、`Code` 和 `Balance`。