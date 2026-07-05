// Package main is the entry point for the radius-go server.
//
// @title           radius-go API
// @version         1.0
// @description     RADIUS server with HTTP management API.
// @description     Management endpoints require the X-Internal-Secret header.
//
// @host            localhost:8083
// @BasePath        /api/v1
// @schemes         http
//
// @securityDefinitions.apikey InternalSecret
// @in header
// @name X-Internal-Secret
package main

import (
	"context"
	"os/signal"
	"syscall"

	_ "github.com/Alg0rix/radius-go/docs" // swag-generated docs

	"github.com/Alg0rix/radius-go/internal/app"
	"github.com/Alg0rix/radius-go/internal/config"
)

func main() {
	cfg := config.Load("radius")
	if err := cfg.Validate(); err != nil {
		panic(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := app.Run(ctx, cfg); err != nil {
		panic(err)
	}
}