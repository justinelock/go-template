package errcode

// 业务错误码常量，与 docs/api/error-codes.md 保持同步。

const (
	OK = 0

	MethodNotAllowed = 405

	BadRequestBody          = 40001
	LoginFieldsRequired     = 40002
	RegisterBadRequestBody  = 40011
	RegisterFieldsRequired  = 40012
	UsernameExists          = 40013
	ProfileBadRequestBody   = 40014
	ProfileNoField          = 40015
	MobileExists            = 40016
	TokenRequired           = 40101
	TokenInvalid            = 40102
	InvalidCredentials      = 40103
	UserNotFound            = 40402
	LoginFailed             = 50001
	ProxyBuildFailed        = 50001 // gateway 与 member 复用码值，语义不同见文档
	DownstreamUnavailable   = 50002
	RouteNotFound           = 40404 // gateway 未匹配到任何路由
	LogoutFailed            = 50006
	TokenVerifyFailed       = 50007
	RegisterFailed          = 50008
	QueryUserFailed         = 50009
	UpdateProfileFailed     = 50010
	WebSocketNotImplemented = 50101

	OrderInvalidInput        = 40021
	OrderIdempotencyRequired = 40022
	OrderNotFound            = 40421
	OrderLockNotAcquired     = 40921
	OrderCreateFailed        = 50021
	OrderQueryFailed         = 50022

	ServiceUnavailable  = 50301
	RateLimitExceeded   = 42901
	Forbidden           = 40301
	RefreshTokenInvalid = 40104

	PaymentInvalidInput = 40031
	PaymentNotFound     = 40431
	PaymentCreateFailed = 50031
	PaymentMockDisabled = 40331
)

const (
	MsgOK                       = "ok"
	MsgMethodNotAllowed         = "method not allowed"
	MsgInvalidRequestBody       = "invalid request body"
	MsgUsernamePasswordRequired = "username/password is required"
	MsgRegisterFieldsRequired   = "username/password/mobile is required"
	MsgUsernameExists           = "username already exists"
	MsgMobileExists             = "mobile already exists"
	MsgProfileFieldRequired     = "at least one field is required"
	MsgTokenRequired            = "token is required"
	MsgTokenInvalid             = "token is invalid or expired"
	MsgInvalidCredentials       = "username or password is invalid"
	MsgUserNotFound             = "user not found"
	MsgLoginFailed              = "login failed"
	MsgProxyBuildFailed         = "proxy build request failed"
	MsgDownstreamUnavailable    = "downstream service unavailable"
	MsgRouteNotFound            = "route not found"
	MsgLogoutFailed             = "logout failed"
	MsgTokenVerifyFailed        = "token verify failed"
	MsgRegisterFailed           = "register failed"
	MsgQueryUserFailed          = "query user failed"
	MsgUpdateProfileFailed      = "update profile failed"
	MsgWebSocketNotImplemented  = "websocket not implemented"
	MsgOrderInvalidInput        = "invalid order input"
	MsgOrderIdempotencyRequired = "X-Idempotency-Key is required"
	MsgOrderNotFound            = "order not found"
	MsgOrderLockNotAcquired     = "order creation in progress, retry later"
	MsgOrderCreateFailed        = "create order failed"
	MsgOrderQueryFailed         = "query order failed"

	MsgServiceUnavailable  = "service not ready"
	MsgRateLimitExceeded   = "rate limit exceeded"
	MsgForbidden           = "forbidden"
	MsgRefreshTokenInvalid = "refresh token is invalid or expired"

	MsgPaymentInvalidInput = "invalid payment input"
	MsgPaymentNotFound     = "payment not found"
	MsgPaymentCreateFailed = "create payment failed"
	MsgPaymentMockDisabled = "mock pay is disabled"
)
