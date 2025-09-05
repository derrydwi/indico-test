// Package main is the entry point for the application
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"indico-backend/internal/config"
	"indico-backend/internal/database"
	"indico-backend/internal/handlers"
	"indico-backend/internal/logger"
	"indico-backend/internal/metrics"
	"indico-backend/internal/repository"
	"indico-backend/internal/routes"
	"indico-backend/internal/service"

	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		panic(fmt.Sprintf("Failed to load configuration: %v", err))
	}

	// Initialize logger
	logger.Init(cfg.Log.Level, cfg.Log.Format)

	logger.Info("Starting Indico Backend Service")

	// Connect to database
	db, err := database.New(&cfg.Database)
	if err != nil {
		logger.WithError(err).Fatal("Failed to connect to database")
	}
	defer db.Close()

	// Initialize metrics
	metrics.Init()

	// Initialize repositories
	productRepo := repository.NewProductRepository(db.DB)
	orderRepo := repository.NewOrderRepository(db.DB)
	txRepo := repository.NewTransactionRepository(db.DB)
	settleRepo := repository.NewSettlementRepository(db.DB)
	jobRepo := repository.NewJobRepository(db.DB)

	// Initialize job processor
	jobProcessor := service.NewJobProcessor(db, &cfg.Jobs, txRepo, settleRepo, jobRepo)
	jobProcessor.Start()
	defer jobProcessor.Stop()

	// Initialize services
	deps := &service.Dependencies{
		DB:           db,
		ProductRepo:  productRepo,
		OrderRepo:    orderRepo,
		TxRepo:       txRepo,
		SettleRepo:   settleRepo,
		JobRepo:      jobRepo,
		JobProcessor: jobProcessor,
	}
	services := service.NewServices(deps)

	// Initialize handlers
	h := handlers.New(services)

	// Set Gin mode
	if cfg.Log.Level == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Setup routes
	router := routes.SetupRoutes(h)

	// Create HTTP server
	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start server in a goroutine
	go func() {
		logger.Infof("Starting HTTP server on port %s", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithError(err).Fatal("Failed to start HTTP server")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Give outstanding requests 30 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.WithError(err).Error("Server forced to shutdown")
	} else {
		logger.Info("Server shutdown complete")
	}
}
