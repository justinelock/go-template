package httptransport

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	paymentapp "go-template/internal/payment/app"
	"go-template/internal/payment/domain"
	"go-template/internal/payment/repo"
	"go-template/internal/platform/errcode"
	"go-template/internal/platform/httpx"
)

// Handler 承担 payment-service 的 HTTP 入站适配：
// 1) 路由注册；2) 参数校验；3) 错误码映射与统一响应。
type Handler struct {
	// 步骤 1：注入 app service。
	svc *paymentapp.Service
}

// createPaymentReq 创建支付单请求体。
type createPaymentReq struct {
	OrderID string  `json:"order_id"`
	Amount  float64 `json:"amount"`
	Channel string  `json:"channel"`
}

// NewHandler 构造 payment HTTP 处理器。
func NewHandler(svc *paymentapp.Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes 注册 payment-service 对外 HTTP 路由。
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// 步骤 1：回调路由须先于前缀路由注册，避免被 /v1/payments/ 吞掉。
	mux.HandleFunc("/v1/payments/callback/", h.callback)
	mux.HandleFunc("/v1/payments", h.payments)
	mux.HandleFunc("/v1/payments/", h.paymentByID)
}

// payments 处理创建支付单 POST /v1/payments。
func (h *Handler) payments(w http.ResponseWriter, r *http.Request) {
	// 步骤 1：校验方法。
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
	// 步骤 3：解析请求体。
	var req createPaymentReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.JSON(w, http.StatusBadRequest, traceID, errcode.BadRequestBody, errcode.MsgInvalidRequestBody, nil)
		return
	}
	// 步骤 4：调用 app 幂等创建。
	id, err := h.svc.EnsurePayment(r.Context(), domain.CreatePaymentReq{
		OrderID: req.OrderID, UserID: userID, Amount: req.Amount, Channel: req.Channel,
	})
	if err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			httpx.JSON(w, http.StatusBadRequest, traceID, errcode.PaymentInvalidInput, errcode.MsgPaymentInvalidInput, nil)
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, traceID, errcode.PaymentCreateFailed, errcode.MsgPaymentCreateFailed, nil)
		return
	}
	// 步骤 5：返回 201 与支付单快照。
	payment, _ := h.svc.GetPayment(r.Context(), id)
	httpx.JSON(w, http.StatusCreated, traceID, errcode.OK, errcode.MsgOK, repo.ToVO(payment))
}

// paymentByID 处理 GET /v1/payments/{id} 与 POST .../mock-pay。
func (h *Handler) paymentByID(w http.ResponseWriter, r *http.Request) {
	traceID := httpx.EnsureTraceID(r)
	// 步骤 1：解析路径段。
	path := strings.TrimPrefix(r.URL.Path, "/v1/payments/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		httpx.JSON(w, http.StatusBadRequest, traceID, errcode.PaymentInvalidInput, errcode.MsgPaymentInvalidInput, nil)
		return
	}
	paymentID := parts[0]
	// 步骤 2：mock-pay 分支。
	if len(parts) == 2 && parts[1] == "mock-pay" && r.Method == http.MethodPost {
		if err := h.svc.MockPay(r.Context(), paymentID); err != nil {
			switch {
			case errors.Is(err, domain.ErrMockPayDisabled):
				httpx.JSON(w, http.StatusForbidden, traceID, errcode.PaymentMockDisabled, errcode.MsgPaymentMockDisabled, nil)
			case errors.Is(err, domain.ErrPaymentNotFound):
				httpx.JSON(w, http.StatusNotFound, traceID, errcode.PaymentNotFound, errcode.MsgPaymentNotFound, nil)
			default:
				httpx.JSON(w, http.StatusInternalServerError, traceID, errcode.PaymentCreateFailed, errcode.MsgPaymentCreateFailed, nil)
			}
			return
		}
		payment, _ := h.svc.GetPayment(r.Context(), paymentID)
		httpx.JSON(w, http.StatusOK, traceID, errcode.OK, errcode.MsgOK, repo.ToVO(payment))
		return
	}
	// 步骤 3：查询分支仅支持 GET。
	if r.Method != http.MethodGet {
		httpx.JSON(w, http.StatusMethodNotAllowed, traceID, errcode.MethodNotAllowed, errcode.MsgMethodNotAllowed, nil)
		return
	}
	payment, err := h.svc.GetPayment(r.Context(), paymentID)
	if err != nil {
		if errors.Is(err, domain.ErrPaymentNotFound) {
			httpx.JSON(w, http.StatusNotFound, traceID, errcode.PaymentNotFound, errcode.MsgPaymentNotFound, nil)
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, traceID, errcode.PaymentCreateFailed, errcode.MsgPaymentCreateFailed, nil)
		return
	}
	httpx.JSON(w, http.StatusOK, traceID, errcode.OK, errcode.MsgOK, repo.ToVO(payment))
}

// callback 处理渠道回调 POST /v1/payments/callback/{channel}/{orderId}。
func (h *Handler) callback(w http.ResponseWriter, r *http.Request) {
	traceID := httpx.EnsureTraceID(r)
	if r.Method != http.MethodPost {
		httpx.JSON(w, http.StatusMethodNotAllowed, traceID, errcode.MethodNotAllowed, errcode.MsgMethodNotAllowed, nil)
		return
	}
	// 步骤 1：解析 channel 与 orderId。
	suffix := strings.TrimPrefix(r.URL.Path, "/v1/payments/callback/")
	parts := strings.Split(strings.Trim(suffix, "/"), "/")
	if len(parts) < 2 {
		httpx.JSON(w, http.StatusBadRequest, traceID, errcode.PaymentInvalidInput, errcode.MsgPaymentInvalidInput, nil)
		return
	}
	channel, orderID := parts[0], parts[1]
	// 步骤 2：调用 app 回调处理（验签占位）。
	if err := h.svc.Callback(r.Context(), channel, orderID); err != nil {
		if errors.Is(err, domain.ErrPaymentNotFound) {
			httpx.JSON(w, http.StatusNotFound, traceID, errcode.PaymentNotFound, errcode.MsgPaymentNotFound, nil)
			return
		}
		httpx.JSON(w, http.StatusInternalServerError, traceID, errcode.PaymentCreateFailed, errcode.MsgPaymentCreateFailed, nil)
		return
	}
	httpx.JSON(w, http.StatusOK, traceID, errcode.OK, errcode.MsgOK, map[string]bool{"ok": true})
}
