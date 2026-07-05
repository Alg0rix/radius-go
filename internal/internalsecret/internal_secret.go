package internalsecret

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

func Require(secret string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			token := extractToken(c.Request())
			if subtle.ConstantTimeCompare([]byte(token), []byte(secret)) != 1 {
				return c.JSON(http.StatusUnauthorized, map[string]any{
					"success": false,
					"error": map[string]any{
						"code":    "unauthorized",
						"message": "invalid or missing Authorization: Bearer token",
					},
				})
			}
			return next(c)
		}
	}
}

func extractToken(r *http.Request) string {
	if auth := r.Header.Get("Authorization"); auth != "" {
		if scheme, token, ok := strings.Cut(auth, " "); ok && strings.EqualFold(scheme, "Bearer") {
			return token
		}
	}
	// Deprecated fallback for existing clients; prefer Authorization: Bearer.
	return r.Header.Get("X-Internal-Secret")
}