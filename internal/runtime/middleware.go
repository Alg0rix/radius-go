package runtime

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
	"golang.org/x/time/rate"
)

const requestTimeout = 5 * time.Second

func UseBaseMiddleware(e *echo.Echo, deps *Dependencies) {
	e.Use(middleware.RequestIDWithConfig(middleware.RequestIDConfig{
		Generator: func() string { return uuid.New().String() },
	}))
	e.Use(auditMiddleware(deps.Logger))
	e.Use(middleware.Recover())
	e.Use(middleware.Secure())
	e.Use(middleware.BodyLimit("1M"))
	e.Use(rateLimiter())
}

func rateLimiter() echo.MiddlewareFunc {
	return middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		Skipper: func(c echo.Context) bool {
			// Allow health probes to stay cheap.
			p := c.Request().URL.Path
			return p == "/health" || p == "/ready" || p == "/healthz" || p == "/readyz"
		},
		Store: middleware.NewRateLimiterMemoryStore(rate.Limit(1000.0 / 60.0)), // 1000 req/min per IP
		IdentifierExtractor: func(c echo.Context) (string, error) {
			return c.RealIP(), nil
		},
		ErrorHandler: func(c echo.Context, err error) error {
			return c.JSON(http.StatusTooManyRequests, Envelope{
				Success: false,
				Error: map[string]any{
					"code":    "too_many_requests",
					"message": "rate limit exceeded",
				},
			})
		},
		DenyHandler: func(c echo.Context, identifier string, err error) error {
			return c.JSON(http.StatusTooManyRequests, Envelope{
				Success: false,
				Error: map[string]any{
					"code":    "too_many_requests",
					"message": "rate limit exceeded",
				},
			})
		},
	})
}

func auditMiddleware(logger zerolog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			err := next(c)
			status := c.Response().Status
			method := c.Request().Method

			// Log all mutating requests and any 4xx/5xx responses.
			isMutating := method == http.MethodPost || method == http.MethodPut || method == http.MethodDelete || method == http.MethodPatch
			if isMutating || status >= 400 {
				logger.Info().
					Str("method", method).
					Str("path", c.Request().URL.Path).
					Str("ip", c.RealIP()).
					Int("status", status).
					Str("request_id", c.Response().Header().Get(echo.HeaderXRequestID)).
					Err(err).
					Msg("http audit")
			}
			return err
		}
	}
}