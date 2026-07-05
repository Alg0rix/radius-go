package runtime

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/Alg0rix/radius-go/internal/config"
	"github.com/Alg0rix/radius-go/internal/database"
)

type Dependencies struct {
	DB     *pgxpool.Pool
	Logger zerolog.Logger
	Config config.Config
}

func Bootstrap(ctx context.Context, cfg config.Config) (*Dependencies, error) {
	logger := NewLogger(cfg)

	pool, err := database.NewPool(ctx, cfg.DBDSN)
	if err != nil {
		return nil, fmt.Errorf("runtime: bootstrap: %w", err)
	}
	logger.Info().Str("dsn", cfg.DBDSN[:min(len(cfg.DBDSN), 30)]+"...").Msg("db pool initialized")

	if err := database.RunMigrations(ctx, pool, logger); err != nil {
		pool.Close()
		return nil, fmt.Errorf("runtime: migrations: %w", err)
	}

	return &Dependencies{
		DB:     pool,
		Logger: logger,
		Config: cfg,
	}, nil
}