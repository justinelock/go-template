// Package otel 提供 OpenTelemetry OTLP HTTP 导出初始化（可选启用）。
package otel

import (
	"context"
	"log/slog"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

// Init 初始化 OTLP HTTP exporter；endpoint 为空时跳过并返回空 shutdown。
func Init(ctx context.Context, serviceName, endpoint string) (func(context.Context) error, error) {
	// 步骤 1：未配置 endpoint 时不启用 OTel。
	if strings.TrimSpace(endpoint) == "" {
		return func(context.Context) error { return nil }, nil
	}

	// 步骤 2：创建 OTLP HTTP exporter（去 scheme，insecure 供本地 Jaeger）。
	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(strings.TrimPrefix(strings.TrimPrefix(endpoint, "https://"), "http://")),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	// 步骤 3：合并默认 resource 与服务名属性。
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
		),
	)
	if err != nil {
		return nil, err
	}

	// 步骤 4：注册 TracerProvider 与 W3C TraceContext 传播器。
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	slog.Info("otel enabled", "service", serviceName, "endpoint", endpoint)
	// 步骤 5：返回 shutdown 回调供 main defer 调用。
	return tp.Shutdown, nil
}
