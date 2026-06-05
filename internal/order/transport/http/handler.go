package httptransport

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	orderapp "go-template/internal/order/app"
	"go-template/internal/order/domain"
	"go-template/internal/platform/errcode"
	"go-template/internal/platform/httpx"
)

// Handler 承担 order-service 的 HTTP 入站适配职责：
// 1) 路由注册；
// 2) 参数解析与幂等键校验；
// 3) 错误码映射与统一响应输出。
type Handler struct {
	// 步骤 1：注入 app service，transport 层不直接处理业务规则。
	svc *orderapp.Service
}

// createOrderReq 创建订单请求体。
type createOrderReq struct {
	ProductID string  `json:"product_id"`
	Amount    float64 `json:"amount"`
}

// NewHandler 构造 order HTTP 处理器。
func NewHandler(svc *orderapp.Service) *Handler {
	// 步骤 1：注入依赖并返回 handler 实例。
	return &Handler{svc: svc}
}

// RegisterRoutes 注册 order-service 对外 HTTP 路由。
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// 步骤 1：注册健康检查。
	// 步骤 2：注册创建订单与按 ID 查询路由。
	mux.HandleFunc("/v1/orders", h.orders)
	mux.HandleFunc("/v1/orders/", h.orderByID)
}

// orders 处理创建订单（POST /v1/orders）。
func (h *Handler) orders(w http.ResponseWriter, r *http.Request) {
	// 步骤 1：校验请求方法。
	traceID := httpx.EnsureTraceID(r)
	if r.Method != http.MethodPost {
		httpx.JSON(w, http.StatusMethodNotAllowed, traceID, errcode.MethodNotAllowed, errcode.MsgMethodNotAllowed, nil)
		return
	}

	// 步骤 2：校验网关注入的用户上下文。
	userID := strings.TrimSpace(r.Header.Get("X-User-Id"))
	if userID == "" {
		httpx.JSON(w, http.StatusUnauthorized, traceID, errcode.TokenRequired, errcode.MsgTokenRequired, nil)
		return
	}

	// 步骤 3：校验幂等键（网关已透传 X-Idempotency-Key）。
	idempotencyKey := strings.TrimSpace(r.Header.Get("X-Idempotency-Key"))
	if idempotencyKey == "" {
		httpx.JSON(w, http.StatusBadRequest, traceID, errcode.OrderIdempotencyRequired, errcode.MsgOrderIdempotencyRequired, nil)
		return
	}

	// 步骤 4：解析并校验请求体。
	var req createOrderReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, traceID, errcode.BadRequestBody, errcode.MsgInvalidRequestBody, nil)
		return
	}

	// 步骤 5：调用 app 层创建订单（幂等 + 锁 + MQ）。
	data, err := h.svc.CreateOrder(r.Context(), domain.CreateOrderReq{
		UserID:         userID,
		ProductID:      req.ProductID,
		Amount:         req.Amount,
		IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		// 步骤 6：按领域错误映射 HTTP 错误码。
		switch {
		case errors.Is(err, domain.ErrInvalidOrderInput):
			httpx.JSON(w, http.StatusBadRequest, traceID, errcode.OrderInvalidInput, errcode.MsgOrderInvalidInput, nil)
		case errors.Is(err, domain.ErrLockNotAcquired):
			httpx.JSON(w, http.StatusConflict, traceID, errcode.OrderLockNotAcquired, errcode.MsgOrderLockNotAcquired, nil)
		default:
			httpx.JSON(w, http.StatusInternalServerError, traceID, errcode.OrderCreateFailed, errcode.MsgOrderCreateFailed, nil)
		}
		return
	}

	// 步骤 7：返回 201 与订单快照。
	httpx.JSON(w, http.StatusCreated, traceID, errcode.OK, errcode.MsgOK, data)
}

// orderByID 处理订单详情查询（GET /v1/orders/{id}）。
func (h *Handler) orderByID(w http.ResponseWriter, r *http.Request) {
	// 步骤 1：校验请求方法。
	traceID := httpx.EnsureTraceID(r)
	if r.Method != http.MethodGet {
		httpx.JSON(w, http.StatusMethodNotAllowed, traceID, errcode.MethodNotAllowed, errcode.MsgMethodNotAllowed, nil)
		return
	}

	// 步骤 2：校验用户上下文。
	userID := strings.TrimSpace(r.Header.Get("X-User-Id"))
	if userID == "" {
		httpx.JSON(w, http.StatusUnauthorized, traceID, errcode.TokenRequired, errcode.MsgTokenRequired, nil)
		return
	}

	// 步骤 3：从路径解析 orderID。
	orderID := strings.TrimPrefix(r.URL.Path, "/v1/orders/")
	orderID = strings.Trim(orderID, "/")
	if orderID == "" {
		httpx.JSON(w, http.StatusBadRequest, traceID, errcode.OrderInvalidInput, errcode.MsgOrderInvalidInput, nil)
		return
	}

	// 步骤 4：调用 app 层查询订单。
	data, err := h.svc.GetOrder(r.Context(), userID, orderID)
	if err != nil {
		// 步骤 5：映射不存在与内部错误。
		if errors.Is(err, domain.ErrOrderNotFound) {
			httpx.JSON(w, http.StatusNotFound, traceID, errcode.OrderNotFound, errcode.MsgOrderNotFound, nil)
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, traceID, errcode.OrderQueryFailed, errcode.MsgOrderQueryFailed, nil)
		return
	}

	// 步骤 6：返回订单详情。
	httpx.JSON(w, http.StatusOK, traceID, errcode.OK, errcode.MsgOK, data)
}
