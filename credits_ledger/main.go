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
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()



	baseURL := getenv("API2KEY_BASE_URL", api2key.DefaultBaseAPIURL)
	projectID := getenv("API2KEY_PROJECT_ID", "")
	email := getenv("API2KEY_EMAIL", "")
	password := getenv("API2KEY_PASSWORD", "")

	if email == "" || password == "" {
		log.Fatal("API2KEY_EMAIL and API2KEY_PASSWORD are required")
	}
	if projectID == "" {
		log.Fatal("API2KEY_PROJECT_ID is required")
	}

	client := api2key.NewClient(
		api2key.WithBaseAPIURL(baseURL),
	)

	// 1. 登录获取 access token
	loginResult, err := client.Login(ctx, api2key.LoginRequest{
		Email:     email,
		Password:  password,
		ProjectID: projectID,
	})
	if err != nil {
		log.Fatal("login failed:", err)
	}
	fmt.Println("✓ 登录成功")
	fmt.Println()

	accessToken := loginResult.AccessToken

	// 2. 查询积分余额
	balance, err := client.GetCreditsBalance(ctx, accessToken)
	if err != nil {
		log.Fatal("get credits balance failed:", err)
	}
	fmt.Println("=== 积分余额 ===")
	fmt.Printf("当前余额: %d\n", balance.Balance)
	fmt.Printf("冻结金额: %d\n", balance.Reserved)
	if balance.TotalEarned != nil {
		fmt.Printf("累计获得: %d\n", *balance.TotalEarned)
	}
	if balance.TotalSpent != nil {
		fmt.Printf("累计消费: %d\n", *balance.TotalSpent)
	}
	fmt.Println()

	// 3. 查询积分记录（分页）
	page := 1
	size := 10
	ledger, err := client.GetLedger(ctx, api2key.GetLedgerRequest{
		AccessToken: accessToken,
		Page:        page,
		Size:        size,
	})
	if err != nil {
		log.Fatal("get ledger failed:", err)
	}

	fmt.Println("=== 积分记录 ===")
	fmt.Printf("总记录数: %d\n", ledger.Pagination.Total)
	fmt.Printf("总页数: %d\n", ledger.Pagination.TotalPages)
	fmt.Printf("当前页: %d / 每页: %d\n", ledger.Pagination.Page, ledger.Pagination.Size)
	fmt.Println()

	if len(ledger.List) == 0 {
		fmt.Println("暂无积分记录")
		return
	}

	fmt.Println("最近的积分变动：")
	fmt.Println("--------------------------------------------------")
	for _, item := range ledger.List {
		createdAt := time.UnixMilli(item.CreatedAt).Format("2006-01-02 15:04:05")
		deltaStr := fmt.Sprintf("%+d", item.Delta)
		fmt.Printf("[%s] %s | 变动: %s | 余额: %d | 服务: %s | 说明: %s\n",
			createdAt,
			item.Type,
			deltaStr,
			item.BalanceAfter,
			item.Service,
			item.Description,
		)
	}
	fmt.Println("--------------------------------------------------")

	// 4. 可选：按类型筛选查询（例如只查消费记录）
	// 取消注释以下代码可以筛选特定类型的记录
	/*
		fmt.Println()
		fmt.Println("=== 筛选消费记录 ===")
		spendLedger, err := client.GetLedger(ctx, api2key.GetLedgerRequest{
			AccessToken: accessToken,
			Page:        1,
			Size:        10,
			Type:        "spend", // 可选值: spend, grant, refund 等
		})
		if err != nil {
			log.Fatal("get spend ledger failed:", err)
		}
		fmt.Printf("消费记录数: %d\n", spendLedger.Pagination.Total)
		for _, item := range spendLedger.List {
			fmt.Printf("  - %s: %+d (%s)\n", item.Service, item.Delta, item.Description)
		}
	*/
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
