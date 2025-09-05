// Package routes provides HTTP route configuration
package routes

import (
	"indico-backend/internal/handlers"

	"github.com/gin-gonic/gin"
)

// SetupRoutes configures all HTTP routes
func SetupRoutes(h *handlers.Handlers) *gin.Engine {
	// Create Gin router
	router := gin.New()

	// Add middleware
	router.Use(h.RequestID())
	router.Use(h.Logger())
	router.Use(h.ErrorHandler())
	router.Use(h.CORS())

	// Health check
	router.GET("/health", h.Health)

	// Metrics endpoint
	router.GET("/metrics", h.MetricsHandler())

	// Order routes
	orderGroup := router.Group("/orders")
	{
		orderGroup.POST("", h.CreateOrder)
		orderGroup.GET("/:id", h.GetOrder)
		orderGroup.GET("", h.ListOrders)
	}

	// Job routes
	jobGroup := router.Group("/jobs")
	{
		jobGroup.POST("/settlement", h.CreateSettlementJob)
		jobGroup.GET("/:id", h.GetJob)
		jobGroup.POST("/:id/cancel", h.CancelJob)
	}

	// Download routes
	router.GET("/downloads/:filename", h.DownloadSettlement)

	return router
}
