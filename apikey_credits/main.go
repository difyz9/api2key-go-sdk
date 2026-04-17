package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/difyz9/api2key-go-sdk/api2key"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()


	// 补一个“纯 API key 查询积分”的最小示例


	baseURL := getenv("API2KEY_BASE_URL", api2key.DefaultBaseAPIURL)
	apiKey := getenv("API2KEY_API_KEY", "")
	projectID := getenv("API2KEY_PROJECT_ID", "")

	if apiKey == "" {
		log.Fatal("API2KEY_API_KEY is required")
	}

	client := api2key.NewClient(
		api2key.WithBaseAPIURL(baseURL),
	)

	balance, err := client.GetCreditsBalanceWithOptions(ctx, api2key.GetCreditsBalanceRequest{
		APIKey: apiKey,
	})
	if err != nil {
		log.Fatal("get credits balance failed:", err)
	}

	ledger, err := client.GetLedger(ctx, api2key.GetLedgerRequest{
		APIKey:    apiKey,
		Page:      1,
		Size:      10,
		ProjectID: projectID,
	})
	if err != nil {
		log.Fatal("get credits ledger failed:", err)
	}

	fmt.Println("=== 积分余额 ===")
	fmt.Printf("余额: %d\n", balance.Balance)
	fmt.Printf("冻结: %d\n", balance.Reserved)
	if balance.Scope.ProjectID != "" {
		fmt.Printf("项目: %s (%s)\n", balance.Scope.ProjectName, balance.Scope.ProjectID)
	}
	fmt.Println()

	fmt.Println("=== 最近积分流水 ===")
	fmt.Printf("总记录数: %d\n", ledger.Pagination.Total)
	if len(ledger.List) == 0 {
		fmt.Println("暂无积分流水")
		return
	}

	for _, item := range ledger.List {
		createdAt := time.UnixMilli(item.CreatedAt).Format("2006-01-02 15:04:05")
		fmt.Printf("[%s] %s | %+d | 余额=%d | 服务=%s | %s\n",
			createdAt,
			item.Type,
			item.Delta,
			item.BalanceAfter,
			item.Service,
			item.Description,
		)
	}
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}