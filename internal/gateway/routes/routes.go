package routes

import "strings"

// ProxyRoute 描述网关对外路径到下游服务的代理映射。
type ProxyRoute struct {
	// PublicPath 网关对外路径。
	PublicPath string
	// UpstreamPath 下游服务内部路径。
	UpstreamPath string
	// ServiceName Consul 服务名，用于服务发现。
	ServiceName string
	// RequiresAuth 是否需在网关层校验 token。
	RequiresAuth bool
}

// ProxyRoutes 当前已注册的反向代理路由表（新增服务在此追加条目）。
var ProxyRoutes = []ProxyRoute{
	{PublicPath: "/v1/auth/login", UpstreamPath: "/v1/auth/login", ServiceName: "member-service", RequiresAuth: false},
	{PublicPath: "/v1/auth/register", UpstreamPath: "/v1/auth/register", ServiceName: "member-service", RequiresAuth: false},
	{PublicPath: "/v1/auth/logout", UpstreamPath: "/v1/auth/logout", ServiceName: "member-service", RequiresAuth: true},
	{PublicPath: "/v1/member/users/profile", UpstreamPath: "/v1/users/profile", ServiceName: "member-service", RequiresAuth: true},
	{PublicPath: "/v1/order/orders", UpstreamPath: "/v1/orders", ServiceName: "order-service", RequiresAuth: true},
}

// ProxyPrefixRoutes 前缀匹配代理（如 GET /v1/order/orders/{id}）。
var ProxyPrefixRoutes = []ProxyRoute{
	{PublicPath: "/v1/order/orders/", UpstreamPath: "/v1/orders/", ServiceName: "order-service", RequiresAuth: true},
}

// RequiresAuth 根据路由表判断路径是否需要在网关鉴权。
func RequiresAuth(path string) bool {
	// 步骤 1：精确匹配 ProxyRoutes。
	for _, route := range ProxyRoutes {
		if route.PublicPath == path {
			return route.RequiresAuth
		}
	}
	// 步骤 2：前缀匹配 ProxyPrefixRoutes（如 /v1/order/orders/{id}）。
	for _, route := range ProxyPrefixRoutes {
		if strings.HasPrefix(path, route.PublicPath) {
			return route.RequiresAuth
		}
	}
	return false
}
