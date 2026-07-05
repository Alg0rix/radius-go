package radius

import (
	"github.com/labstack/echo/v4"
	"github.com/your-org/radius-go/internal/internalsecret"
)

// RegisterHTTPHandlers registers all HTTP management routes on the Echo router.
// Health endpoints are registered separately in app.Run().
func RegisterHTTPHandlers(e *echo.Echo, svc *Service, internalSecret string) {
	secretMW := internalsecret.Require(internalSecret)

	// Status
	e.GET("/api/v1/radius/status", svc.HandleStatus, secretMW)

	// NAS management
	nases := e.Group("/api/v1/radius/nases", secretMW)
	nases.GET("", svc.HandleListNAS)
	nases.POST("", svc.HandleCreateNAS)
	nases.PUT("/:id", svc.HandleUpdateNAS)
	nases.DELETE("/:id", svc.HandleDeleteNAS)

	// Subscriber management
	subs := e.Group("/api/v1/radius/subscribers", secretMW)
	subs.GET("", svc.HandleListSubscribers)
	subs.POST("", svc.HandleCreateSubscriber)
	subs.PUT("/:id", svc.HandleUpdateSubscriber)
	subs.DELETE("/:id", svc.HandleDeleteSubscriber)

	// Session management
	e.GET("/api/v1/radius/sessions", svc.HandleListSessions, secretMW)
	e.POST("/api/v1/radius/sessions/disconnect", svc.HandleDisconnectUser, secretMW)
	e.POST("/api/v1/radius/subscribers/coa-change", svc.HandleCoAChange, secretMW)
	e.POST("/api/v1/radius/sessions/cleanup", svc.HandleSessionCleanup, secretMW)
	e.POST("/api/v1/radius/sessions/reconcile", svc.HandleSessionReconcile, secretMW)

	// Voucher packages
	pkgs := e.Group("/api/v1/voucher-packages", secretMW)
	pkgs.GET("", svc.HandleListVoucherPackages)
	pkgs.POST("", svc.HandleCreateVoucherPackage)
	pkgs.PUT("/:id", svc.HandleUpdateVoucherPackage)
	pkgs.DELETE("/:id", svc.HandleDeleteVoucherPackage)

	// Vouchers
	e.GET("/api/v1/vouchers", svc.HandleListVouchers, secretMW)
	e.POST("/api/v1/vouchers/generate", svc.HandleGenerateVouchers, secretMW)
	e.GET("/api/v1/vouchers/:code/balance", svc.HandleVoucherBalance, secretMW)
}