# 步骤级注释约定

本仓库要求新增或修改代码时编写**尽可能详细**的步骤级注释，便于 Code Review、交接与 AI 辅助维护。

## 目标

- 读函数时能按步骤还原控制流，无需逐行猜意图。
- 与 `internal/gateway/transport/http/handler.go`、`internal/member/transport/http/handler.go` 现有风格一致。

## 类型与函数（包注释 / 导出注释）

```go
// Service 封装 member 领域核心用例：
// 1) 用户鉴权（登录/登出/token 校验）；
// 2) 用户资料查询与更新。
type Service struct { ... }

// Login 支持 username/mobile 双入口登录，并返回兼容新老前端字段的 VO。
func (s *Service) Login(ctx context.Context, req domain.LoginReq) (*vo.LoginResp, error) {
```

## 结构体字段

```go
type Handler struct {
	// 步骤 1：基础 HTTP 客户端，用于转发请求到下游服务。
	httpClient *http.Client
	// 步骤 2：服务发现解析器（优先 Consul，失败回退静态地址）。
	resolve gatewayapp.Resolver
}
```

## 函数体内步骤

每个逻辑块一行，编号连续：

```go
func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	// 步骤 1：校验请求方法。
	traceID := httpx.EnsureTraceID(r)
	if r.Method != http.MethodPost {
		httpx.JSON(w, http.StatusMethodNotAllowed, traceID, 405, "method not allowed", nil)
		return
	}

	// 步骤 2：解析并校验请求体。
	var req loginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, traceID, 40001, "invalid request body", nil)
		return
	}

	// 步骤 3：调用 app 层执行登录。
	data, err := h.svc.Login(r.Context(), ...)

	// 步骤 4：按领域错误映射 HTTP 错误码。
	if err != nil { ... }

	// 步骤 5：返回登录成功响应。
	httpx.JSON(w, http.StatusOK, traceID, 0, "ok", data)
}
```

## 分支与错误映射

```go
	if err != nil {
		// 步骤 4.1：用户名已存在，返回 40013。
		if errors.Is(err, domain.ErrUsernameExists) {
			httpx.JSON(w, http.StatusBadRequest, traceID, 40013, "username already exists", nil)
			return
		}
		// 步骤 4.2：其它错误统一 50008。
		httpx.JSON(w, http.StatusInternalServerError, traceID, 50008, "register failed", nil)
		return
	}
```

## 中间件与链式调用

```go
// BuildServer 按顺序装配网关中间层：
// 先鉴权，再 CORS（保证错误响应也带跨域头）。
func (h *Handler) BuildServer(mux *http.ServeMux) http.Handler {
	// 步骤 1：链式包裹中间件并返回最终入口 handler。
	return h.withCORS(h.withAuth(mux))
}
```

## Shell 脚本

```bash
# 步骤 1：编译 gateway 与 member 二进制。
go build -o ./bin/gateway-service ./cmd/gateway-service

# 步骤 2：释放端口，避免启动失败。
stop_port "${GATEWAY_SERVICE_PORT}"
```

## 不必注释的内容

- `api/gen/**` 中 protoc 生成文件。
- 无分支的单行 `return err` 包装（可选一行函数说明即可）。

## 与 Cursor 规则的关系

- 规则：`.cursor/rules/code-comments.mdc`（`alwaysApply: true`）
- 变更日志：`.cursor/rules/changelog.mdc` + 根目录 `CHANGELOG.md`
