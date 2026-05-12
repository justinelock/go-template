package httptransport

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	memberapp "pricing-assistant/internal/member/app"
	"pricing-assistant/internal/member/domain"
	"pricing-assistant/internal/platform/httpx"
)

// Handler 承担 member-service 的 HTTP 入站适配职责：
// 1) 路由注册；
// 2) 参数解析与基础校验；
// 3) 错误码映射与统一响应输出。
type Handler struct {
	// 步骤 1：注入 app service，transport 层不直接处理业务规则。
	svc *memberapp.Service
}

// loginReq 登录请求体。
type loginReq struct {
	Username string `json:"username"`
	Mobile   string `json:"mobile"`
	Password string `json:"password"`
}

// registerReq 注册请求体。
type registerReq struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	Email      string `json:"email"`
	Mobile     string `json:"mobile"`
	Nickname   string `json:"nickname"`
	ParentID   string `json:"parentId"`
	InviteCode string `json:"inviteCode"`
	Avatar     string `json:"avatar"`
	Remark     string `json:"remark"`
}

// updateMeReq 用户资料更新请求体。
type updateMeReq struct {
	Email    string `json:"email"`
	Mobile   string `json:"mobile"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
	Remark   string `json:"remark"`
}

// introspectResp token 反查结果响应体。
type introspectResp struct {
	UserID       string `json:"userId"`
	UserIDLegacy string `json:"user_id,omitempty"`
}

// NewHandler 构造 member HTTP 处理器。
func NewHandler(svc *memberapp.Service) *Handler {
	// 步骤 1：注入依赖并返回 handler 实例。
	return &Handler{svc: svc}
}

// RegisterRoutes 注册 member-service 对外 HTTP 路由。
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// 步骤 1：注册健康检查与认证相关路由。
	mux.HandleFunc("/healthz", h.healthz)
	mux.HandleFunc("/v1/auth/login", h.login)
	mux.HandleFunc("/v1/auth/register", h.register)
	mux.HandleFunc("/v1/auth/logout", h.logout)
	mux.HandleFunc("/v1/auth/introspect", h.introspect)
	// 步骤 2：注册用户资料路由。
	mux.HandleFunc("/v1/users/profile", h.profile)
}

// healthz 返回 member 服务健康状态。
func (h *Handler) healthz(w http.ResponseWriter, r *http.Request) {
	// 步骤 1：生成 traceID 并输出统一响应。
	traceID := httpx.EnsureTraceID(r)
	httpx.JSON(w, http.StatusOK, traceID, 0, "ok", map[string]string{"service": "member-service"})
}

// login 处理登录请求。
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
	if strings.TrimSpace(req.Username) == "" && strings.TrimSpace(req.Mobile) == "" || req.Password == "" {
		httpx.JSON(w, http.StatusBadRequest, traceID, 40002, "username/password is required", nil)
		return
	}

	// 步骤 3：调用 app 层执行登录。
	data, err := h.svc.Login(r.Context(), domain.LoginReq{
		Username: req.Username,
		Mobile:   req.Mobile,
		Password: req.Password,
	})

	if err != nil {
		// 步骤 4：按领域错误映射 HTTP 错误码。
		if errors.Is(err, domain.ErrInvalidCredentials) {
			httpx.JSON(w, http.StatusUnauthorized, traceID, 40103, "username or password is invalid", nil)
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, traceID, 50001, "login failed", nil)
		return
	}

	// 步骤 5：返回登录成功响应。
	httpx.JSON(w, http.StatusOK, traceID, 0, "ok", data)
}

// register 处理用户注册请求。
func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	// 步骤 1：校验请求方法。
	traceID := httpx.EnsureTraceID(r)
	if r.Method != http.MethodPost {
		httpx.JSON(w, http.StatusMethodNotAllowed, traceID, 405, "method not allowed", nil)
		return
	}

	// 步骤 2：解析请求体并校验必填。
	var req registerReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, traceID, 40011, "invalid request body", nil)
		return
	}
	if strings.TrimSpace(req.Username) == "" || strings.TrimSpace(req.Password) == "" || strings.TrimSpace(req.Mobile) == "" {
		httpx.JSON(w, http.StatusBadRequest, traceID, 40012, "username/password/mobile is required", nil)
		return
	}

	// 步骤 3：调用 app 层执行注册。
	data, err := h.svc.Register(r.Context(), domain.RegisterReq{
		Username:   req.Username,
		Password:   req.Password,
		Email:      req.Email,
		Mobile:     req.Mobile,
		Nickname:   req.Nickname,
		ParentID:   req.ParentID,
		InviteCode: req.InviteCode,
		Avatar:     req.Avatar,
		Remark:     req.Remark,
	})
	if err != nil {
		// 步骤 4：按领域错误映射客户端可读错误。
		if errors.Is(err, domain.ErrUsernameExists) {
			httpx.JSON(w, http.StatusBadRequest, traceID, 40013, "username already exists", nil)
			return
		}
		if errors.Is(err, domain.ErrMobileExists) {
			httpx.JSON(w, http.StatusBadRequest, traceID, 40016, "mobile already exists", nil)
			return
		}
		if errors.Is(err, domain.ErrInvalidCredentials) {
			httpx.JSON(w, http.StatusBadRequest, traceID, 40012, "username/password/mobile is required", nil)
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, traceID, 50008, "register failed", nil)
		return
	}

	// 步骤 5：返回新建用户资料。
	httpx.JSON(w, http.StatusOK, traceID, 0, "ok", data)
}

// logout 处理用户登出。
func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	// 步骤 1：校验请求方法与 token 存在性。
	traceID := httpx.EnsureTraceID(r)
	if r.Method != http.MethodPost {
		httpx.JSON(w, http.StatusMethodNotAllowed, traceID, 405, "method not allowed", nil)
		return
	}
	token := extractToken(r)
	if token == "" {
		httpx.JSON(w, http.StatusUnauthorized, traceID, 40101, "token is required", nil)
		return
	}

	// 步骤 2：调用 app 层执行 token 失效。
	if err := h.svc.Logout(r.Context(), token); err != nil {
		httpx.JSON(w, http.StatusInternalServerError, traceID, 50006, "logout failed", nil)
		return
	}

	// 步骤 3：返回登出成功响应（驼峰主字段 + 兼容旧字段）。
	httpx.JSON(w, http.StatusOK, traceID, 0, "ok", map[string]any{
		"loggedOut":  true,
		"logged_out": true,
	})
}

// introspect 校验 token 并返回 userId。
func (h *Handler) introspect(w http.ResponseWriter, r *http.Request) {
	// 步骤 1：校验方法与 token 参数。
	traceID := httpx.EnsureTraceID(r)
	if r.Method != http.MethodGet {
		httpx.JSON(w, http.StatusMethodNotAllowed, traceID, 405, "method not allowed", nil)
		return
	}
	token := extractToken(r)
	if token == "" {
		httpx.JSON(w, http.StatusUnauthorized, traceID, 40101, "token is required", nil)
		return
	}

	// 步骤 2：调用 app 层反查 userID。
	userID, err := h.svc.IntrospectToken(r.Context(), token)
	if err != nil {
		// 步骤 3：映射 token 无效与内部异常。
		if errors.Is(err, domain.ErrTokenInvalid) {
			httpx.JSON(w, http.StatusUnauthorized, traceID, 40102, "token is invalid or expired", nil)
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, traceID, 50007, "token verify failed", nil)
		return
	}

	// 步骤 4：返回反查结果。
	httpx.JSON(w, http.StatusOK, traceID, 0, "ok", introspectResp{
		UserID:       userID,
		UserIDLegacy: userID,
	})
}

// usersMe 处理用户资料查询/更新。
func (h *Handler) profile(w http.ResponseWriter, r *http.Request) {
	// 步骤 1：统一解析当前 userID。
	traceID := httpx.EnsureTraceID(r)
	userID, err := h.svc.ResolveUserID(r.Context(), r.Header.Get("X-User-Id"), extractToken(r))
	if err != nil {
		httpx.JSON(w, http.StatusUnauthorized, traceID, 40102, "token is invalid or expired", nil)
		return
	}

	// 步骤 2：按方法分支处理 GET/PUT。
	switch r.Method {
	case http.MethodGet:
		// 步骤 2.1：查询用户资料。
		data, err := h.svc.GetUserProfile(r.Context(), userID)
		if err != nil {
			if errors.Is(err, domain.ErrUserNotFound) {
				httpx.JSON(w, http.StatusNotFound, traceID, 40402, "user not found", nil)
				return
			}
			httpx.JSON(w, http.StatusInternalServerError, traceID, 50009, "query user failed", nil)
			return
		}
		httpx.JSON(w, http.StatusOK, traceID, 0, "ok", data)
	case http.MethodPut:
		// 步骤 2.2：解析并校验更新请求。
		var req updateMeReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpx.JSON(w, http.StatusBadRequest, traceID, 40014, "invalid request body", nil)
			return
		}
		if strings.TrimSpace(req.Email) == "" && strings.TrimSpace(req.Mobile) == "" && strings.TrimSpace(req.Nickname) == "" && strings.TrimSpace(req.Avatar) == "" && strings.TrimSpace(req.Remark) == "" {
			httpx.JSON(w, http.StatusBadRequest, traceID, 40015, "at least one field is required", nil)
			return
		}

		// 步骤 2.3：执行资料更新并返回最新资料。
		data, err := h.svc.UpdateUserProfile(r.Context(), userID, domain.UpdateMeReq{
			Email:    req.Email,
			Mobile:   req.Mobile,
			Nickname: req.Nickname,
			Avatar:   req.Avatar,
			Remark:   req.Remark,
		})
		if err != nil {
			httpx.JSON(w, http.StatusInternalServerError, traceID, 50010, "update profile failed", nil)
			return
		}
		httpx.JSON(w, http.StatusOK, traceID, 0, "ok", data)
	default:
		// 步骤 2.4：其他方法统一返回 405。
		httpx.JSON(w, http.StatusMethodNotAllowed, traceID, 405, "method not allowed", nil)
	}
}

// extractToken 统一 token 提取优先级：
// Authorization Bearer > token header > token query。
func extractToken(r *http.Request) string {
	// 步骤 1：优先读取 Authorization: Bearer xxx。
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		return strings.TrimSpace(authHeader[7:])
	}

	// 步骤 2：回退 token header。
	if token := strings.TrimSpace(r.Header.Get("token")); token != "" {
		return token
	}

	// 步骤 3：最终回退 query 参数。
	return strings.TrimSpace(r.URL.Query().Get("token"))
}
