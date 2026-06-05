package logging

import (
	"log/slog"
	"os"
	"strings"
)

// Init 按配置初始化全局 slog（JSON 或 text），并设为默认 logger。
func Init(level, format string) *slog.Logger {
	// 步骤 1：解析日志级别字符串。
	lvl := parseLevel(level)
	// 步骤 2：按 format 选择 JSON 或 text handler。
	var handler slog.Handler
	opts := &slog.HandlerOptions{Level: lvl}
	if strings.EqualFold(format, "json") {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}
	// 步骤 3：注册为全局默认 logger 并返回实例。
	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}

// parseLevel 将配置字符串映射为 slog.Level。
func parseLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
