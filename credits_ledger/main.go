package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/difyz9/api2key-go-sdk/api2key"
)

func main() {
	baseURL := getenv("API2KEY_BASE_URL", api2key.DefaultBaseAPIURL)
	apiKey := getenv("API2KEY_API_KEY", "sk-EA8ADC--")

	if apiKey == "" {
		fmt.Println("API2KEY_API_KEY is required for direct credit deduction")
		os.Exit(1)
	}

	client := api2key.NewClient(
		api2key.WithBaseAPIURL(baseURL),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	fmt.Println("正在获取初始积分余额...")
	initialBalance, err := client.GetCreditsBalanceWithOptions(ctx, api2key.GetCreditsBalanceRequest{APIKey: apiKey})
	if err != nil {
		fmt.Printf("错误: 获取初始积分余额失败: %v\n", err)
		return
	}
	fmt.Printf("✅ 初始积分余额: %d\n\n", initialBalance.Balance)

	// 直接扣减积分
	deductionAmount := 10
	fmt.Printf("正在直接扣减 %d 积分...\n", deductionAmount)
	deductReq := api2key.DeductCreditsRequest{
		APIKey:      apiKey,
		Amount:      deductionAmount,
		Service:     "manual-test",
		TaskID:      fmt.Sprintf("manual-deduct-%d", time.Now().Unix()),
		Description: "手动测试扣减",
	}
	deductResp, err := client.DeductCredits(ctx, deductReq)
	if err != nil {
		fmt.Printf("错误: 扣减积分失败: %v\n", err)
		return
	}
	fmt.Printf("✅ 积分扣减成功。扣减后余额: %d，幂等命中: %t\n\n", deductResp.BalanceAfter, deductResp.Idempotent)

	// 等待几秒钟，确保账单更新
	time.Sleep(3 * time.Second)

	fmt.Println("正在获取最终积分余额...")
	finalBalance, err := client.GetCreditsBalanceWithOptions(ctx, api2key.GetCreditsBalanceRequest{APIKey: apiKey})
	if err != nil {
		fmt.Printf("警告: 获取最终积分余额失败: %v\n", err)
		return
	}
	fmt.Printf("✅ 最终积分余额: %d\n\n", finalBalance.Balance)

	fmt.Println("正在验证积分扣减情况...")
	expectedBalance := initialBalance.Balance - deductionAmount
	if finalBalance.Balance == expectedBalance {
		fmt.Printf("✅ 验证成功! 积分已成功扣减，共消耗了 %d 积分。\n\n", deductionAmount)
	} else {
		fmt.Printf("⚠️ 余额校验未命中。预期余额: %d, 实际余额: %d\n\n", expectedBalance, finalBalance.Balance)
	}

	fmt.Println("正在获取最新的积分账单记录以供核对...")
	ledger, err := client.GetLedger(ctx, api2key.GetLedgerRequest{APIKey: apiKey, Size: 5})
	if err != nil {
		fmt.Printf("警告: 获取积分账单失败: %v\n", err)
		return
	}

	if len(ledger.List) == 0 {
		fmt.Println("未找到积分账单记录。")
		return
	}

	fmt.Println("✅ 最近的积分账单记录:")
	fmt.Println("--------------------------------------------------")
	for _, item := range ledger.List {
		fmt.Printf("- 类型: %-8s | 变动: %-5d | 余额: %-7d | 描述: %s\n", item.Type, item.Delta, item.BalanceAfter, item.Description)
	}
	fmt.Println("--------------------------------------------------")
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
