package routes

import (
	"sort"
	"strings"
	"sync/atomic"
)

// ProxyRoute 描述网关对外路径到下游服务的代理映射。
type ProxyRoute struct {
	// PublicPath 网关对外路径。
	PublicPath string
	// UpstreamPath 下游服务内部路径。
	UpstreamPath string
	// ServiceName 服务名，用于服务发现（Consul）。
	ServiceName string
	// RequiresAuth 是否需在网关层校验 token。
	RequiresAuth bool
	// RequiredRoles 非空时要求用户角色命中其一（如 admin）。
	RequiredRoles []string
	// Prefix 为 true 时按前缀匹配（如 /v1/order/orders/{id}），否则精确匹配。
	Prefix bool
	// UpstreamBaseURL 可选：直连下游基址（形如 http://host:port）。
	// 设置后无需在代码里预置该服务的回退 URL，即可纯配置接入新服务。
	UpstreamBaseURL string
}

// UpstreamTargetPath 返回下游路径部分：前缀路由会把 PublicPath 之后的尾段
// （如 /{id}）追加到 UpstreamPath 后，精确路由直接返回 UpstreamPath。
func (r ProxyRoute) UpstreamTargetPath(reqPath string) string {
	if r.Prefix {
		return r.UpstreamPath + strings.TrimPrefix(reqPath, r.PublicPath)
	}
	return r.UpstreamPath
}

// table 是一份不可变路由快照：exact 精确匹配优先，prefix 前缀匹配兜底。
type table struct {
	exact  []ProxyRoute
	prefix []ProxyRoute
}

// current 保存当前生效的路由快照，热加载时原子替换（读多写少，无锁读取）。
var current atomic.Pointer[table]

func init() {
	// 步骤 1：进程启动即装载内置默认路由，保证无配置文件时开箱即用。
	current.Store(builtinTable())
}

// builtinTable 返回代码内置的默认路由表（配置文件缺失/损坏时的兜底）。
func builtinTable() *table {
	t := &table{
		exact: []ProxyRoute{
			{PublicPath: "/v1/auth/login", UpstreamPath: "/v1/auth/login", ServiceName: "member-service", RequiresAuth: false},
			{PublicPath: "/v1/auth/register", UpstreamPath: "/v1/auth/register", ServiceName: "member-service", RequiresAuth: false},
			{PublicPath: "/v1/auth/logout", UpstreamPath: "/v1/auth/logout", ServiceName: "member-service", RequiresAuth: true},
			{PublicPath: "/v1/auth/refresh", UpstreamPath: "/v1/auth/refresh", ServiceName: "member-service", RequiresAuth: false},
			{PublicPath: "/v1/member/users/profile", UpstreamPath: "/v1/users/profile", ServiceName: "member-service", RequiresAuth: true},
			{PublicPath: "/v1/order/orders", UpstreamPath: "/v1/orders", ServiceName: "order-service", RequiresAuth: true},
			{PublicPath: "/v1/payment/payments", UpstreamPath: "/v1/payments", ServiceName: "payment-service", RequiresAuth: true},
		},
		prefix: []ProxyRoute{
			{PublicPath: "/v1/order/orders/", UpstreamPath: "/v1/orders/", ServiceName: "order-service", RequiresAuth: true, Prefix: true},
			{PublicPath: "/v1/payment/payments/", UpstreamPath: "/v1/payments/", ServiceName: "payment-service", RequiresAuth: true, Prefix: true},
		},
	}
	sortPrefix(t.prefix)
	return t
}

// sortPrefix 按前缀长度降序排序，保证更具体的前缀先匹配（避免覆盖顺序歧义）。
func sortPrefix(routes []ProxyRoute) {
	sort.SliceStable(routes, func(i, j int) bool {
		return len(routes[i].PublicPath) > len(routes[j].PublicPath)
	})
}

// Match 按当前路由快照解析路径：先精确、后前缀；ok=false 表示未命中任何路由。
func Match(path string) (ProxyRoute, bool) {
	// 步骤 1：读取原子快照（无锁）。
	t := current.Load()
	// 步骤 2：精确匹配优先。
	for _, r := range t.exact {
		if r.PublicPath == path {
			return r, true
		}
	}
	// 步骤 3：前缀匹配兜底（已按长度降序，更具体者先命中）。
	for _, r := range t.prefix {
		if strings.HasPrefix(path, r.PublicPath) {
			return r, true
		}
	}
	return ProxyRoute{}, false
}

// Count 返回当前生效路由条数（用于热加载日志）。
func Count() int {
	t := current.Load()
	return len(t.exact) + len(t.prefix)
}

// RequiresAuth 根据当前路由表判断路径是否需在网关鉴权。
func RequiresAuth(path string) bool {
	route, ok := Match(path)
	return ok && route.RequiresAuth
}

// RequiredRolesForPath 返回路径所需角色；nil 表示任意登录用户或未命中路由。
func RequiredRolesForPath(path string) []string {
	route, ok := Match(path)
	if !ok {
		return nil
	}
	return route.RequiredRoles
}
