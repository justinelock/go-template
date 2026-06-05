// Package runtime 提供服务进程级运行时能力（优雅关闭等）。
package runtime

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// RunHTTPServer 启动 HTTP 并支持 SIGTERM/SIGINT 优雅关闭。
func RunHTTPServer(addr string, handler http.Handler, onShutdown ...func(context.Context)) error {
	// 步骤 1：构造 Server 并在 goroutine 中监听。
	srv := &http.Server{Addr: addr, Handler: handler}
	errCh := make(chan error, 1)
	go func() {
		slog.Info("http listening", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// 步骤 2：等待退出信号或监听错误。
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return err
	case <-stop:
	}

	// 步骤 3：执行可选清理（MQ、Consul、gRPC 等）。
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for _, fn := range onShutdown {
		if fn != nil {
			fn(ctx)
		}
	}
	// 步骤 4：优雅关闭 HTTP Server。
	return srv.Shutdown(ctx)
}
