---
name: code-change
description: >-
  任意代码/配置/脚本变更的通用流程：步骤级注释 + CHANGELOG。非 API 专项时用本 Skill。
---

# 通用代码变更流程

## 1. 实现

- 按 [`docs/conventions/code-comments.md`](../../../docs/conventions/code-comments.md) 写 **步骤级注释**（函数块注释 + 体内 `// 步骤 N：`）。
- 参照：`internal/gateway/transport/http/handler.go`、`internal/member/transport/http/handler.go`。

## 2. 变更日志

- 更新根目录 [`CHANGELOG.md`](../../../CHANGELOG.md) → **`## [Unreleased]`**。
- 分类：Added / Changed / Fixed / Removed / Docs / Chore。
- 格式：`- **范围**：说明做了什么（必要时写为什么）。`

## 3. 验证

```bash
go build ./...
```

涉及认证流程时另跑：`./scripts/smoke-auth-flow.sh`

## 4. API 相关？

若改动路由、handler、proto、错误码，改用 [api-change](../api-change/SKILL.md) 流程（含 docs/api 与 Postman）。

## 禁止

- 无步骤注释的复杂逻辑
- 不更新 CHANGELOG 就结束任务
- 手改 `api/gen/`
