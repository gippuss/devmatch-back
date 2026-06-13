package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/gippuss/devmatch-back/internal/auth"
	"github.com/gippuss/devmatch-back/internal/config"
	"github.com/gippuss/devmatch-back/internal/platform/logger"
	"github.com/gippuss/devmatch-back/internal/platform/postgres"
	"github.com/gippuss/devmatch-back/internal/repository"
	"github.com/gippuss/devmatch-back/internal/service"
	transport "github.com/gippuss/devmatch-back/internal/transport/http"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	log, err := logger.New(cfg.LogLevel)
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = log.Sync()
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	dbPool, err := postgres.NewPool(ctx, cfg.DatabaseURL, cfg.DatabaseConnectAttempts, cfg.DatabaseConnectDelay)
	if err != nil {
		log.Fatal("failed to connect to postgres", zap.Error(err))
	}
	defer dbPool.Close()

	if cfg.AutoMigrate {
		if err := postgres.RunMigrations(cfg.DatabaseURL, "migrations"); err != nil {
			log.Fatal("failed to run migrations", zap.Error(err))
		}
	}

	userRepo, err := repository.NewUserRepository(dbPool)
	if err != nil {
		log.Fatal("failed to init user repository", zap.Error(err))
	}

	projectRepo, err := repository.NewProjectRepository(dbPool)
	if err != nil {
		log.Fatal("failed to init project repository", zap.Error(err))
	}

	appRepo, err := repository.NewApplicationRepository(dbPool)
	if err != nil {
		log.Fatal("failed to init application repository", zap.Error(err))
	}

	dictRepo, err := repository.NewDictionaryRepository(dbPool)
	if err != nil {
		log.Fatal("failed to init dictionary repository", zap.Error(err))
	}

	refreshTokenRepo, err := repository.NewRefreshTokenRepository(dbPool)
	if err != nil {
		log.Fatal("failed to init refresh token repository", zap.Error(err))
	}

	tokenManager, err := auth.NewTokenManager(auth.Config{
		AccessSecret:  cfg.AuthAccessSecret,
		RefreshSecret: cfg.AuthRefreshSecret,
		AccessTTL:     cfg.AuthAccessTTL,
		RefreshTTL:    cfg.AuthRefreshTTL,
	})
	if err != nil {
		log.Fatal("failed to init token manager", zap.Error(err))
	}

	userService := service.NewUserService(userRepo)
	projectService := service.NewProjectService(projectRepo)
	applicationService := service.NewApplicationService(appRepo)
	dictionaryService := service.NewDictionaryService(dictRepo)
	authService := service.NewAuthService(userRepo, refreshTokenRepo, tokenManager)

	router := transport.NewRouter(
		log,
		tokenManager,
		authService,
		userService,
		projectService,
		applicationService,
		dictionaryService,
		dbPool,
		cfg.CORSOrigins,
	)

	srv := transport.NewServer(cfg, router)

	go func() {
		log.Info("starting HTTP server", zap.String("addr", cfg.HTTPAddr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("http server failed", zap.Error(err))
		}
	}()

	<-ctx.Done()
	log.Info("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("graceful shutdown failed", zap.Error(err))
		os.Exit(1)
	}

	log.Info("server stopped")
}
