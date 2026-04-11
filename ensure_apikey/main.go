package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/difyz9/api2key-go-sdk/api2key"
)

type config struct {
	BaseAPIURL string
	Email      string
	Password   string
	ProjectID  string
	KeyName    string
	Timeout    time.Duration
}

type resolvedAPIKey struct {
	ID        string
	Name      string
	KeyPrefix string
	Secret    string
	Source    string
	Created   bool
	ListCount int
}

func main() {
	cfg := loadConfig()
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	client := api2key.NewClient(
		api2key.WithBaseAPIURL(cfg.BaseAPIURL),
	)

	loginResult, err := client.Login(ctx, api2key.LoginRequest{
		Email:     cfg.Email,
		Password:  cfg.Password,
		ProjectID: cfg.ProjectID,
	})
	if err != nil {
		log.Fatal(err)
	}

	apiKey, err := ensureAPIKey(ctx, client, loginResult.AccessToken, cfg)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("api key list length:", apiKey.ListCount)
	fmt.Println("api key:", displayAPIKey(apiKey))
}

func loadConfig() config {


	var cfg config
	flag.StringVar(&cfg.BaseAPIURL, "base-url", getenv("API2KEY_BASE_URL", api2key.DefaultBaseAPIURL), "base api url")
	flag.StringVar(&cfg.Email, "email", getenv("API2KEY_EMAIL", ""), "login email")
	flag.StringVar(&cfg.Password, "password", getenv("API2KEY_PASSWORD", ""), "login password")
	flag.StringVar(&cfg.ProjectID, "project-id", getenv("API2KEY_PROJECT_ID", ""), "optional project id")
	flag.StringVar(&cfg.KeyName, "key-name", getenv("API2KEY_KEY_NAME", "sdk-example-key"), "api key name used when a new key must be created")
	flag.DurationVar(&cfg.Timeout, "timeout", 30*time.Second, "request timeout")
	flag.Parse()

	if strings.TrimSpace(cfg.Email) == "" || strings.TrimSpace(cfg.Password) == "" {
		log.Fatal("API2KEY_EMAIL and API2KEY_PASSWORD are required")
	}
	if strings.TrimSpace(cfg.KeyName) == "" {
		log.Fatal("API2KEY_KEY_NAME is required")
	}

	return cfg
}

func ensureAPIKey(ctx context.Context, client *api2key.Client, accessToken string, cfg config) (*resolvedAPIKey, error) {
	keys, err := client.ListAPIKeys(ctx, accessToken)
	if err != nil {
		return nil, err
	}
	if picked, ok := pickAPIKeyWithSecret(keys.Keys); ok {
		return &resolvedAPIKey{
			ID:        picked.ID,
			Name:      picked.Name,
			KeyPrefix: picked.KeyPrefix,
			Secret:    picked.Secret,
			Source:    "existing",
			Created:   false,
			ListCount: len(keys.Keys),
		}, nil
	}

	created, err := client.CreateAPIKey(ctx, accessToken, api2key.CreateAPIKeyRequest{Name: cfg.KeyName})
	if err != nil {
		return nil, err
	}
	return &resolvedAPIKey{
		ID:        created.Key.ID,
		Name:      created.Key.Name,
		KeyPrefix: created.Key.KeyPrefix,
		Secret:    created.Key.Secret,
		Source:    "created",
		Created:   true,
		ListCount: len(keys.Keys),
	}, nil
}

func getenv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func pickAPIKey(keys []api2key.UserAPIKey) api2key.UserAPIKey {
	picked := keys[0]
	for _, key := range keys[1:] {
		if key.Active != picked.Active {
			if key.Active {
				picked = key
			}
			continue
		}
		if key.UpdatedAt > picked.UpdatedAt {
			picked = key
			continue
		}
		if key.UpdatedAt == picked.UpdatedAt && key.CreatedAt > picked.CreatedAt {
			picked = key
		}
	}
	return picked
}

func pickAPIKeyWithSecret(keys []api2key.UserAPIKey) (api2key.UserAPIKey, bool) {
	if len(keys) == 0 {
		return api2key.UserAPIKey{}, false
	}

	var candidates []api2key.UserAPIKey
	for _, key := range keys {
		if strings.TrimSpace(key.Secret) == "" {
			continue
		}
		candidates = append(candidates, key)
	}
	if len(candidates) == 0 {
		return api2key.UserAPIKey{}, false
	}

	return pickAPIKey(candidates), true
}

func displayAPIKey(apiKey *resolvedAPIKey) string {
	if apiKey == nil {
		return ""
	}
	if strings.TrimSpace(apiKey.Secret) != "" {
		return apiKey.Secret
	}
	return apiKey.KeyPrefix
}
