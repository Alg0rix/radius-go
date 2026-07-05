package runtime

import (
	"context"

	"github.com/labstack/echo/v4"
)

func HealthHandler(deps *Dependencies) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx, cancel := context.WithTimeout(c.Request().Context(), requestTimeout)
		defer cancel()

		if err := deps.DB.Ping(ctx); err != nil {
			return Fail(c, 503, "db_unavailable", "database ping failed", err.Error())
		}
		return OK(c, map[string]string{"status": "ok"})
	}
}

func ReadyHandler(deps *Dependencies) echo.HandlerFunc {
	return func(c echo.Context) error {
		return OK(c, map[string]string{"status": "ready"})
	}
}