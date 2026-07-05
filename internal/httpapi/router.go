// Package httpapi owns management REST route registration. The actual
// handler implementations live on *radius.Service; this package wires them
// onto the Echo router with the internal-secret middleware applied.
package httpapi

import (
	"github.com/labstack/echo/v4"
	"github.com/your-org/radius-go/internal/radius"
)

// RegisterRoutes is the single entry point used by app.Run() to set up the
// management API. The internal-secret middleware is applied inside the
// radius package so route + guard live together.
func RegisterRoutes(e *echo.Echo, svc *radius.Service, internalSecret string) {
	radius.RegisterHTTPHandlers(e, svc, internalSecret)
}