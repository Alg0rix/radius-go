package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/your-org/radius-go/internal/config"
	"github.com/your-org/radius-go/internal/radius"
	"github.com/your-org/radius-go/internal/runtime"
)

func Run(ctx context.Context, cfg config.Config) error {
	deps, err := runtime.Bootstrap(ctx, cfg)
	if err != nil {
		return err
	}
	defer deps.DB.Close()

	svc := radius.NewService(deps, cfg)
	if err := svc.Start(); err != nil {
		return fmt.Errorf("app: start radius: %w", err)
	}
	defer svc.Shutdown(context.Background())

	e := echo.New()
	e.HideBanner = true
	runtime.UseBaseMiddleware(e, deps)

	// Health + management routes (delegated to router.go).
	setupRoutes(e, deps, svc, cfg)

	deps.Logger.Info().
		Str("addr", cfg.HTTPAddr).
		Msg("http server starting")

	go func() {
		if err := e.Start(cfg.HTTPAddr); err != nil && err != http.ErrServerClosed {
			deps.Logger.Error().Err(err).Msg("http server error")
		}
	}()

	<-ctx.Done()
	deps.Logger.Info().Msg("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := e.Shutdown(shutdownCtx); err != nil {
		deps.Logger.Error().Err(err).Msg("http shutdown error")
	}

	deps.Logger.Info().Msg("server stopped")
	return nil
}