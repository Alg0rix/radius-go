package internalsecret

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func Require(secret string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if c.Request().Header.Get("X-Internal-Secret") != secret {
				return c.JSON(http.StatusUnauthorized, map[string]any{
					"success": false,
					"error": map[string]any{
						"code":    "unauthorized",
						"message": "invalid or missing X-Internal-Secret header",
					},
				})
			}
			return next(c)
		}
	}
}