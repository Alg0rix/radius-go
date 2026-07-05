package app

import (
	"github.com/labstack/echo/v4"
	echoSwagger "github.com/swaggo/echo-swagger"
	"github.com/Alg0rix/radius-go/internal/config"
	"github.com/Alg0rix/radius-go/internal/radius"
	"github.com/Alg0rix/radius-go/internal/runtime"
)

// setupRoutes wires health (public) + management (internal-secret) routes
// onto the Echo router. Pure delegation: keep app.go focused on lifecycle.
func setupRoutes(e *echo.Echo, deps *runtime.Dependencies, svc *radius.Service, cfg config.Config) {
	// Health endpoints (public).
	e.GET("/health", runtime.HealthHandler(deps))
	e.GET("/ready", runtime.ReadyHandler(deps))
	e.GET("/healthz", runtime.HealthHandler(deps))
	e.GET("/readyz", runtime.ReadyHandler(deps))

	// Swagger UI (public).
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	// Management API (internal-secret protected via httpapi).
	radius.RegisterHTTPHandlers(e, svc, cfg.InternalSecret)
}