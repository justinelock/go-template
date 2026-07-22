package routes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// routesDoc 是路由配置文件的顶层结构（configs/routes.json）。
type routesDoc struct {
	Routes []routeEntry `json:"routes"`
}

// routeEntry 是配置文件中的单条路由声明。
type routeEntry struct {
	PublicPath      string   `json:"public_path"`
	UpstreamPath    string   `json:"upstream_path"`
	ServiceName     string   `json:"service_name"`
	RequiresAuth    bool     `json:"requires_auth"`
	RequiredRoles   []string `json:"required_roles"`
	Match           string   `json:"match"` // exact（默认）| prefix
	UpstreamBaseURL string   `json:"upstream_base_url"`
}

// LoadFile 从 JSON 文件解析并校验路由表；任一条目非法都会整体拒绝（保证原子性）。
func LoadFile(path string) (*table, error) {
	// 步骤 1：读取文件。
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	// 步骤 2：反序列化。
	var doc routesDoc
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parse routes file: %w", err)
	}
	// 步骤 3：校验并转换为运行时快照。
	return doc.toTable()
}

// toTable 将配置文档校验后转换为不可变路由快照。
func (d routesDoc) toTable() (*table, error) {
	if len(d.Routes) == 0 {
		return nil, errors.New("routes file has no routes")
	}
	t := &table{}
	seen := make(map[string]bool, len(d.Routes))
	for i, e := range d.Routes {
		// 步骤 1：路径基本校验。
		if e.PublicPath == "" || !strings.HasPrefix(e.PublicPath, "/") {
			return nil, fmt.Errorf("route[%d]: public_path must start with '/'", i)
		}
		if e.UpstreamPath == "" || !strings.HasPrefix(e.UpstreamPath, "/") {
			return nil, fmt.Errorf("route[%d] %s: upstream_path must start with '/'", i, e.PublicPath)
		}
		// 步骤 2：至少要能定位下游（服务名或直连基址其一）。
		if e.ServiceName == "" && e.UpstreamBaseURL == "" {
			return nil, fmt.Errorf("route[%d] %s: service_name or upstream_base_url is required", i, e.PublicPath)
		}
		route := ProxyRoute{
			PublicPath:      e.PublicPath,
			UpstreamPath:    e.UpstreamPath,
			ServiceName:     e.ServiceName,
			RequiresAuth:    e.RequiresAuth,
			RequiredRoles:   e.RequiredRoles,
			UpstreamBaseURL: e.UpstreamBaseURL,
		}
		// 步骤 3：按匹配类型归类并查重。
		match := strings.ToLower(strings.TrimSpace(e.Match))
		if match == "" {
			match = "exact"
		}
		switch match {
		case "exact":
			key := "e:" + e.PublicPath
			if seen[key] {
				return nil, fmt.Errorf("duplicate exact route %s", e.PublicPath)
			}
			seen[key] = true
			t.exact = append(t.exact, route)
		case "prefix":
			if !strings.HasSuffix(e.PublicPath, "/") {
				return nil, fmt.Errorf("route[%d] %s: prefix public_path must end with '/'", i, e.PublicPath)
			}
			key := "p:" + e.PublicPath
			if seen[key] {
				return nil, fmt.Errorf("duplicate prefix route %s", e.PublicPath)
			}
			seen[key] = true
			route.Prefix = true
			t.prefix = append(t.prefix, route)
		default:
			return nil, fmt.Errorf("route[%d] %s: invalid match %q (want exact|prefix)", i, e.PublicPath, e.Match)
		}
	}
	// 步骤 4：前缀按长度降序，保证更具体者先命中。
	sortPrefix(t.prefix)
	return t, nil
}

// Reload 从文件加载并原子替换当前路由表；校验失败则返回错误且不改动现有表。
func Reload(path string) error {
	t, err := LoadFile(path)
	if err != nil {
		return err
	}
	current.Store(t)
	return nil
}

// StartReloader 启动路由热加载：
//  1. 首次尝试从文件加载（缺失/损坏时保留内置默认路由，保证开箱即用）；
//  2. 定时轮询文件 mtime，变更时原子热更新；
//  3. 监听 SIGHUP 主动触发重载（运维 kill -HUP 即可，无需重启）。
//
// path 为空时完全跳过（纯内置路由）。校验失败始终保留上一版路由，坏配置不会打挂网关。
func StartReloader(ctx context.Context, path string, interval time.Duration, logger *slog.Logger) {
	if logger == nil {
		logger = slog.Default()
	}
	if path == "" {
		logger.Info("gateway routes: file disabled, using builtin defaults", "count", Count())
		return
	}
	// 步骤 1：首次加载（失败不致命，回退内置默认）。
	if err := Reload(path); err != nil {
		logger.Warn("gateway routes: initial load failed, using builtin defaults", "path", path, "err", err)
	} else {
		logger.Info("gateway routes: loaded", "path", path, "count", Count())
	}
	// 步骤 2：后台轮询 + 信号监听。
	go watchLoop(ctx, path, interval, logger)
}

// watchLoop 轮询文件 mtime 并监听 SIGHUP，触发热重载。
func watchLoop(ctx context.Context, path string, interval time.Duration, logger *slog.Logger) {
	// 步骤 1：注册 SIGHUP。
	sighup := make(chan os.Signal, 1)
	signal.Notify(sighup, syscall.SIGHUP)
	defer signal.Stop(sighup)

	// 步骤 2：可选轮询 ticker（interval<=0 时仅靠 SIGHUP）。
	var tickC <-chan time.Time
	if interval > 0 {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		tickC = ticker.C
	}

	// 步骤 3：记录初始 mtime，避免启动后立即重复加载。
	var lastMod time.Time
	if fi, err := os.Stat(path); err == nil {
		lastMod = fi.ModTime()
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-sighup:
			reloadAndLog(path, logger)
		case <-tickC:
			fi, err := os.Stat(path)
			if err != nil {
				continue
			}
			if fi.ModTime().After(lastMod) {
				lastMod = fi.ModTime()
				reloadAndLog(path, logger)
			}
		}
	}
}

// reloadAndLog 执行一次重载并记录结果；失败保留旧表。
func reloadAndLog(path string, logger *slog.Logger) {
	if err := Reload(path); err != nil {
		logger.Error("gateway routes: reload failed, keeping previous table", "path", path, "err", err)
		return
	}
	logger.Info("gateway routes: reloaded", "path", path, "count", Count())
}
