package monitor

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

const (
	defaultReadTimeout           = 5 * time.Second
	defaultReadHeaderTimeout     = 2 * time.Second
	defautWriteTimeout           = 5 * time.Second
	defaultIdleTimeout           = 5 * time.Second
	defaultMaxHeaderBytes        = 2 << 10
	defaultServerShutdownTimeout = 60 * time.Second
)

type HealthChecker interface {
	IsHealthy() bool
}

func ListenAndServe(ctx context.Context, healthChecks ...HealthChecker) error {
	serverctx, cancel := context.WithCancel(ctx)
	defer cancel()

	logger := slog.NewLogLogger(slog.Default().Handler(), slog.LevelError)

	mux := http.NewServeMux()
	server := &http.Server{
		Addr:              ":3000",
		Handler:           mux,
		ReadTimeout:       defaultReadTimeout,
		ReadHeaderTimeout: defaultReadHeaderTimeout,
		WriteTimeout:      defautWriteTimeout,
		IdleTimeout:       defaultIdleTimeout,
		MaxHeaderBytes:    defaultMaxHeaderBytes,
		ErrorLog:          logger,
	}

	wg := sync.WaitGroup{}
	wg.Add(1)

	serverErr := new(atomic.Value)
	go func() {
		defer wg.Done()
		defer cancel()

		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr.Store(err)
		}
	}()

	<-serverctx.Done()

	if err, ok := serverErr.Load().(error); ok && err != nil {
		return fmt.Errorf("server closed unexpectedly: %w", err)
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), defaultServerShutdownTimeout)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server shutdown error: %w", err)
	}

	wg.Wait()

	return nil
}
