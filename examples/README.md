# api2key Go SDK Examples

这个目录把 SDK 示例独立成一个单独的 Go module，便于单独维护、运行和按能力模块管理。

## 目录结构

```text
examples/
├── ai/
│   ├── chat/
│   └── subtitle_translate/
├── auth/
│   └── ensure_apikey/
├── credits/
│   ├── apikey_balance/
│   └── deduct_and_ledger/
├── speech/
│   ├── smoke_test/
│   └── subtitle_tts/
└── workflow/
    └── full_cli/
```

## 使用方式

```bash
cd examples
go test ./...
```

运行某个示例时，从当前目录执行：

```bash
go run ./ai/chat
go run ./auth/ensure_apikey
go run ./workflow/full_cli
```

这个 examples module 通过 `replace ../` 引用上一级 SDK 主模块，因此可以跟随当前工作区里的源码变化一起调试。
