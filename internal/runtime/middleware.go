package runtime

import (
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

const requestTimeout = 5 * time.Second

func UseBaseMiddleware(e *echo.Echo, deps *Dependencies) {
	e.Use(middleware.RequestIDWithConfig(middleware.RequestIDConfig{
		Generator: func() string { return uuid.New().String() },
	}))
	e.Use(middleware.Recover())
}