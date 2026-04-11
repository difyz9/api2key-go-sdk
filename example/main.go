package main

import (
	"context"
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

type config struct {
	BaseAPIURL     string
	SpeechURL      string
	ServiceSecret  string
	Email          string
	Password       string
	ProjectID      string
	KeyName        string
	Provider       string
	Locale         string
	Search         string
	Voice          string
	Text           string
	Format         string
	Output         string
	StorageKey     string
	DownloadName   string
	VideoID        string
	AudioFile      string
	AudioURL       string
	EngineModel    string
	SpendUserID    string
	SpendAmount    int
	SpendService   string
	SpendTaskID    string
	PaymentSubject string
	PaymentDesc    string
	PaymentAmount  float64
	PaymentType    string
	Timeout        time.Duration
	DoSpeech       bool
	DoSRT          bool
	DoCredits      bool
	DoDirectPay    bool
	PollDirectPay  bool
	PollAsyncTask  bool
}

func main() {
	cfg := loadConfig()
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	options := []api2key.Option{
		api2key.WithBaseAPIURL(cfg.BaseAPIURL),
		api2key.WithServiceSecret(cfg.ServiceSecret),
	}
	if strings.TrimSpace(cfg.SpeechURL) != "" {
		options = append(options, api2key.WithSpeechURL(cfg.SpeechURL))
	}
	client := api2key.NewClient(options...)

	loginResult, err := client.Login(ctx, api2key.LoginRequest{
		Email:     cfg.Email,
		Password:  cfg.Password,
		ProjectID: cfg.ProjectID,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("login ok")

	apiKeyResult, err := client.CreateAPIKey(ctx, loginResult.AccessToken, api2key.CreateAPIKeyRequest{
		Name: cfg.KeyName,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("api key created:", apiKeyResult.Key.KeyPrefix)

	voices, err := client.ListVoices(ctx, api2key.ListVoicesRequest{
		ProjectID: cfg.ProjectID,
		APIKey:    apiKeyResult.Key.Secret,
		Provider:  cfg.Provider,
		Locale:    cfg.Locale,
		Search:    cfg.Search,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("voices matched: %d\n", voices.Total)
	for index, voice := range voices.Voices {
		if index >= 5 {
			break
		}
		fmt.Printf("  - %s | %s | %s\n", voice.ShortName, voice.DisplayName, voice.Locale)
	}

	if cfg.DoSpeech {
		result, err := client.SaveSpeechToFile(ctx, api2key.SynthesizeSpeechRequest{
			ProjectID:        cfg.ProjectID,
			APIKey:           apiKeyResult.Key.Secret,
			Provider:         cfg.Provider,
			Text:             cfg.Text,
			Voice:            cfg.Voice,
			Locale:           cfg.Locale,
			Rate:             1,
			Volume:           100,
			Pitch:            0,
			Format:           cfg.Format,
			StorageKey:       resolveSpeechStorageKey(cfg),
			DownloadFilename: resolveSpeechDownloadName(cfg),
		}, cfg.Output)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("speech saved:", cfg.Output)
		fmt.Println("speech charged:", result.Charged)
		fmt.Println("speech storage key:", result.StorageKey)
		fmt.Println("speech download url:", result.DownloadURL)
	}

	if cfg.DoSRT {
		request := api2key.ASRRequest{
			ProjectID:       cfg.ProjectID,
			APIKey:          apiKeyResult.Key.Secret,
			AudioFilePath:   cfg.AudioFile,
			AudioURL:        cfg.AudioURL,
			Provider:        "tencent",
			EngineModelType: cfg.EngineModel,
			Async:           cfg.PollAsyncTask,
		}
		result, err := client.AudioToSRT(ctx, request)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("srt submit status:", result.StatusStr)
		fmt.Println("srt task id:", result.TaskID)

		if cfg.PollAsyncTask && result.TaskID != 0 {
			polled, err := client.PollASRTaskWithOptions(ctx, api2key.ASRTaskQueryRequest{
				ProjectID: cfg.ProjectID,
				APIKey:    apiKeyResult.Key.Secret,
				TaskID:    fmt.Sprint(result.TaskID),
				Provider:  request.Provider,
			}, 2*time.Second, 30)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println("srt final status:", polled.StatusStr)
			if strings.TrimSpace(polled.SRT) != "" {
				fmt.Println("srt preview:")
				fmt.Println(firstNLines(polled.SRT, 8))
			}
		} else if strings.TrimSpace(result.SRT) != "" {
			fmt.Println("srt preview:")
			fmt.Println(firstNLines(result.SRT, 8))
		}
	}

	if cfg.DoCredits {
		creditsResult, err := client.SpendCredits(ctx, api2key.SpendCreditsRequest{
			UserID:      cfg.SpendUserID,
			Amount:      cfg.SpendAmount,
			Service:     cfg.SpendService,
			TaskID:      cfg.SpendTaskID,
			Description: "sdk example spend",
		})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("credits balance after:", creditsResult.BalanceAfter)
		fmt.Println("credits idempotent:", creditsResult.Idempotent)
	}

	if cfg.DoDirectPay {
		directPayment, err := client.CreateDirectPayment(ctx, loginResult.AccessToken, api2key.DirectPaymentCreateRequest{
			Subject:     cfg.PaymentSubject,
			Amount:      cfg.PaymentAmount,
			Description: cfg.PaymentDesc,
			ProjectID:   cfg.ProjectID,
			PaymentType: cfg.PaymentType,
		})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("direct payment created:", directPayment.ID)
		fmt.Println("direct payment order no:", directPayment.OrderNo)
		fmt.Println("direct payment qr code:", directPayment.Data.QRCode)
		fmt.Println("direct payment pay url:", directPayment.Data.PayURL)

		if cfg.PollDirectPay {
			paymentStatus, err := client.PollDirectPaymentStatus(ctx, loginResult.AccessToken, api2key.DirectPaymentQueryRequest{
				DirectPaymentID: directPayment.ID,
			}, 3*time.Second, 40)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println("direct payment local status:", paymentStatus.LocalStatus)
			fmt.Println("direct payment unified status:", paymentStatus.UnifiedStatus)
			fmt.Println("direct payment paid:", paymentStatus.Paid)
		}
	}
}

func loadConfig() config {
	var cfg config
	flag.StringVar(&cfg.BaseAPIURL, "base-url", getenv("API2KEY_BASE_URL", api2key.DefaultBaseAPIURL), "base api url")
	flag.StringVar(&cfg.SpeechURL, "speech-url", getenv("API2KEY_SPEECH_URL", getenv("API2KEY_TTS_URL", "")), "optional speech root url override")
	flag.StringVar(&cfg.ServiceSecret, "service-secret", getenv("API2KEY_SERVICE_SECRET", ""), "service secret for credits api")
	flag.StringVar(&cfg.Email, "email", getenv("API2KEY_EMAIL", ""), "login email")
	flag.StringVar(&cfg.Password, "password", getenv("API2KEY_PASSWORD", ""), "login password")
	flag.StringVar(&cfg.ProjectID, "project-id", getenv("API2KEY_PROJECT_ID", ""), "optional project id")
	flag.StringVar(&cfg.KeyName, "key-name", getenv("API2KEY_KEY_NAME", "sdk-example-key"), "created api key name")
	flag.StringVar(&cfg.Provider, "provider", getenv("API2KEY_PROVIDER", "azure"), "tts provider")
	flag.StringVar(&cfg.Locale, "locale", getenv("API2KEY_LOCALE", "zh-CN"), "tts locale")
	flag.StringVar(&cfg.Search, "search", getenv("API2KEY_SEARCH", "Xiaoxiao"), "voice search keyword")
	flag.StringVar(&cfg.Voice, "voice", getenv("API2KEY_VOICE", "zh-CN-XiaoxiaoNeural"), "voice short name")
	flag.StringVar(&cfg.Text, "text", getenv("API2KEY_TEXT", "你好，这是 SDK example。"), "speech text")
	flag.StringVar(&cfg.Format, "format", getenv("API2KEY_FORMAT", "audio-24khz-96kbitrate-mono-mp3"), "speech output format")
	flag.StringVar(&cfg.Output, "output", getenv("API2KEY_OUTPUT", filepath.Join("example", "output.mp3")), "speech output path")
	flag.StringVar(&cfg.StorageKey, "storage-key", getenv("API2KEY_STORAGE_KEY", ""), "optional custom remote storage key, e.g. video_123/index_0001.mp3")
	flag.StringVar(&cfg.DownloadName, "download-name", getenv("API2KEY_DOWNLOAD_FILENAME", ""), "optional custom download filename, e.g. index_0001.mp3")
	flag.StringVar(&cfg.VideoID, "video-id", getenv("API2KEY_VIDEO_ID", "sdk-example-video"), "remote video id used to build default storage key when storage-key is empty")
	flag.StringVar(&cfg.AudioFile, "audio-file", getenv("API2KEY_AUDIO_FILE", ""), "audio file path for srt")
	flag.StringVar(&cfg.AudioURL, "audio-url", getenv("API2KEY_AUDIO_URL", ""), "audio url for srt")
	flag.StringVar(&cfg.EngineModel, "engine-model", getenv("API2KEY_ENGINE_MODEL", "16k_zh"), "asr engine model")
	flag.StringVar(&cfg.SpendUserID, "spend-user-id", getenv("API2KEY_CREDITS_USER_ID", ""), "credits target user id")
	flag.IntVar(&cfg.SpendAmount, "spend-amount", getenvInt("API2KEY_CREDITS_AMOUNT", 10), "credits amount")
	flag.StringVar(&cfg.SpendService, "spend-service", getenv("API2KEY_CREDITS_SERVICE", "ai_chat"), "credits service name")
	flag.StringVar(&cfg.SpendTaskID, "spend-task-id", getenv("API2KEY_CREDITS_TASK_ID", fmt.Sprintf("sdk-example-%d", time.Now().Unix())), "credits task id")
	flag.StringVar(&cfg.PaymentSubject, "payment-subject", getenv("API2KEY_PAYMENT_SUBJECT", "SDK 直付测试"), "direct payment subject")
	flag.StringVar(&cfg.PaymentDesc, "payment-desc", getenv("API2KEY_PAYMENT_DESC", "api2key go sdk direct payment example"), "direct payment description")
	flag.Float64Var(&cfg.PaymentAmount, "payment-amount", getenvFloat64("API2KEY_PAYMENT_AMOUNT", 0.01), "direct payment amount")
	flag.StringVar(&cfg.PaymentType, "payment-type", getenv("API2KEY_PAYMENT_TYPE", api2key.DefaultDirectPaymentType), "direct payment type")
	flag.DurationVar(&cfg.Timeout, "timeout", getenvDuration("API2KEY_TIMEOUT", 60*time.Second), "request timeout")
	flag.BoolVar(&cfg.DoSpeech, "speech", getenvBool("API2KEY_EXAMPLE_DO_SPEECH", false), "run speech synthesis")
	flag.BoolVar(&cfg.DoSRT, "srt", getenvBool("API2KEY_EXAMPLE_DO_SRT", false), "run audio to srt")
	flag.BoolVar(&cfg.DoCredits, "credits", getenvBool("API2KEY_EXAMPLE_DO_CREDITS", false), "run spend credits")
	flag.BoolVar(&cfg.DoDirectPay, "direct-pay", getenvBool("API2KEY_EXAMPLE_DO_DIRECT_PAY", false), "run direct payment create")
	flag.BoolVar(&cfg.PollDirectPay, "poll-direct-pay", getenvBool("API2KEY_EXAMPLE_POLL_DIRECT_PAY", false), "poll direct payment status after create")
	flag.BoolVar(&cfg.PollAsyncTask, "poll", getenvBool("API2KEY_EXAMPLE_POLL", true), "poll async asr task when task id is returned")
	flag.Parse()
	return cfg
}

func resolveSpeechStorageKey(cfg config) string {
	if value := strings.TrimSpace(cfg.StorageKey); value != "" {
		return value
	}
	videoID := strings.Trim(strings.TrimSpace(cfg.VideoID), "/")
	if videoID == "" {
		videoID = "sdk-example-video"
	}
	extension := outputFileExtension(cfg.Format)
	return fmt.Sprintf("%s/index_0001%s", videoID, extension)
}

func resolveSpeechDownloadName(cfg config) string {
	if value := strings.TrimSpace(cfg.DownloadName); value != "" {
		return value
	}
	return "index_0001" + outputFileExtension(cfg.Format)
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

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func getenvBool(key string, fallback bool) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if value == "" {
		return fallback
	}
	return value == "1" || value == "true" || value == "yes" || value == "on"
}

func getenvInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getenvFloat64(key string, fallback float64) float64 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func getenvDuration(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func firstNLines(text string, count int) string {
	lines := strings.Split(text, "\n")
	if len(lines) <= count {
		return text
	}
	return strings.Join(lines[:count], "\n")
}
