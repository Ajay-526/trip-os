package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tripos/backend/internal/config"
	"github.com/tripos/backend/internal/handler"
	"github.com/tripos/backend/internal/platform/db"
	"github.com/tripos/backend/internal/platform/jwt"
	"github.com/tripos/backend/internal/repository/postgres"
	"github.com/tripos/backend/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load failed", "error", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Repositories
	userRepo := postgres.NewUserRepo(pool)
	sessionRepo := postgres.NewSessionRepo(pool)
	tripRepo := postgres.NewTripRepo(pool)
	membershipRepo := postgres.NewMembershipRepo(pool)
	inviteLinkRepo := postgres.NewInviteLinkRepo(pool)
	preferenceRepo := postgres.NewPreferenceRepo(pool)

	// Platform
	issuer := jwt.NewIssuer(cfg.JWTAccessSecret, time.Duration(cfg.JWTAccessTTLMinutes)*time.Minute)

	// Services
	authService := service.NewAuthService(userRepo, sessionRepo, issuer, time.Duration(cfg.RefreshTokenTTLHours)*time.Hour)
	tripService := service.NewTripService(tripRepo, membershipRepo)
	membershipService := service.NewMembershipService(membershipRepo, inviteLinkRepo)
	preferenceService := service.NewPreferenceService(preferenceRepo, membershipRepo)

	// Handlers
	authHandler := handler.NewAuthHandler(authService, userRepo, cfg.CSRFSecret, cfg.CookieDomain, cfg.CookieSecure, cfg.CrossSiteCookies)
	tripHandler := handler.NewTripHandler(tripService)
	membershipHandler := handler.NewMembershipHandler(membershipService, publicWebBase())
	preferenceHandler := handler.NewPreferenceHandler(preferenceService)

	router := handler.NewRouter(handler.Deps{
		Issuer:      issuer,
		CSRFSecret:  cfg.CSRFSecret,
		AllowedOrigin: cfg.AllowedOrigin,
		Auth:        authHandler,
		Trips:       tripHandler,
		Memberships: membershipHandler,
		Preferences: preferenceHandler,
	})

	srv := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		slog.Info("trip-os api listening", "port", cfg.HTTPPort, "env", cfg.AppEnv)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	slog.Info("shutting down")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	_ = srv.Shutdown(shutdownCtx)
}

func publicWebBase() string {
	if v := os.Getenv("PUBLIC_WEB_BASE_URL"); v != "" {
		return v
	}
	return "http://localhost:3000"
}
