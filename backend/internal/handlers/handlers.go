// Package handlers provides HTTP request handlers
package handlers

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"time"

	"indico-backend/internal/errors"
	"indico-backend/internal/logger"
	"indico-backend/internal/metrics"
	"indico-backend/internal/models"
	"indico-backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Handlers contains all HTTP handlers
type Handlers struct {
	services *service.Services
}

// New creates a new handlers instance
func New(services *service.Services) *Handlers {
	return &Handlers{
		services: services,
	}
}

// Order handlers

// CreateOrder handles POST /orders
func (h *Handlers) CreateOrder(c *gin.Context) {
	start := time.Now()
	ctx := c.Request.Context()

	var req models.CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.WithContext(ctx).WithError(err).Error("Invalid request body")
		metrics.HTTPRequestsTotal.WithLabelValues("POST", "/orders", "400").Inc()
		h.respondWithError(c, errors.NewValidationError("Invalid request body: "+err.Error()))
		return
	}

	order, err := h.services.Order.CreateOrder(ctx, &req)
	if err != nil {
		metrics.HTTPRequestsTotal.WithLabelValues("POST", "/orders", "500").Inc()
		h.respondWithError(c, err)
		return
	}

	metrics.HTTPRequestsTotal.WithLabelValues("POST", "/orders", "201").Inc()
	metrics.HTTPRequestDuration.WithLabelValues("POST", "/orders").Observe(time.Since(start).Seconds())
	metrics.OrdersCreated.Inc()

	c.JSON(http.StatusCreated, order)
}

// GetOrder handles GET /orders/:id
func (h *Handlers) GetOrder(c *gin.Context) {
	ctx := c.Request.Context()

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		logger.WithContext(ctx).WithError(err).Error("Invalid order ID")
		h.respondWithError(c, errors.NewValidationError("Invalid order ID"))
		return
	}

	order, err := h.services.Order.GetOrder(ctx, id)
	if err != nil {
		h.respondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, order)
}

// ListOrders handles GET /orders
func (h *Handlers) ListOrders(c *gin.Context) {
	ctx := c.Request.Context()

	// Parse query parameters
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	orders, err := h.services.Order.ListOrders(ctx, limit, offset)
	if err != nil {
		h.respondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"orders": orders,
		"limit":  limit,
		"offset": offset,
	})
}

// Job handlers

// CreateSettlementJob handles POST /jobs/settlement
func (h *Handlers) CreateSettlementJob(c *gin.Context) {
	ctx := c.Request.Context()

	var req models.CreateSettlementJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.WithContext(ctx).WithError(err).Error("Invalid request body")
		h.respondWithError(c, errors.NewValidationError("Invalid request body: "+err.Error()))
		return
	}

	job, err := h.services.Job.CreateSettlementJob(ctx, &req)
	if err != nil {
		h.respondWithError(c, err)
		return
	}

	response := gin.H{
		"job_id": job.ID,
		"status": job.Status,
	}

	c.JSON(http.StatusAccepted, response)
}

// GetJob handles GET /jobs/:id
func (h *Handlers) GetJob(c *gin.Context) {
	ctx := c.Request.Context()

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		logger.WithContext(ctx).WithError(err).Error("Invalid job ID")
		h.respondWithError(c, errors.NewValidationError("Invalid job ID"))
		return
	}

	job, err := h.services.Job.GetJob(ctx, id)
	if err != nil {
		h.respondWithError(c, err)
		return
	}

	response := gin.H{
		"job_id":    job.ID,
		"status":    job.Status,
		"progress":  job.Progress,
		"processed": job.Processed,
		"total":     job.Total,
	}

	// Add download URL if job is completed
	if job.Status == models.JobStatusCompleted && job.DownloadURL != nil {
		response["download_url"] = *job.DownloadURL
	}

	// Add error if job failed
	if job.Status == models.JobStatusFailed && job.Error != nil {
		response["error"] = *job.Error
	}

	c.JSON(http.StatusOK, response)
}

// CancelJob handles POST /jobs/:id/cancel
func (h *Handlers) CancelJob(c *gin.Context) {
	ctx := c.Request.Context()

	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		logger.WithContext(ctx).WithError(err).Error("Invalid job ID")
		h.respondWithError(c, errors.NewValidationError("Invalid job ID"))
		return
	}

	err = h.services.Job.CancelJob(ctx, id)
	if err != nil {
		h.respondWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Job cancellation requested",
	})
}

// DownloadSettlement handles GET /downloads/:filename
func (h *Handlers) DownloadSettlement(c *gin.Context) {
	filename := c.Param("filename")

	// Validate filename format (should be UUID.csv)
	if len(filename) < 40 || filename[len(filename)-4:] != ".csv" {
		h.respondWithError(c, errors.NewValidationError("Invalid filename"))
		return
	}

	// Extract job ID from filename
	jobIDStr := filename[:len(filename)-4]
	if _, err := uuid.Parse(jobIDStr); err != nil {
		h.respondWithError(c, errors.NewValidationError("Invalid job ID in filename"))
		return
	}

	filePath := "/tmp/settlements/" + filename

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		h.respondWithError(c, errors.NewAppError("FILE_NOT_FOUND", "Settlement file not found", http.StatusNotFound))
		return
	}

	// Set headers for file download
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Type", "application/octet-stream")

	c.File(filePath)
}

// Health handlers

// Health handles GET /health
func (h *Handlers) Health(c *gin.Context) {
	ctx := c.Request.Context()

	health, err := h.services.Health.Check(ctx)
	if err != nil {
		h.respondWithError(c, err)
		return
	}

	statusCode := http.StatusOK
	if health.Status != "healthy" {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, health)
}

// Middleware

// RequestID middleware adds a request ID to the context
func (h *Handlers) RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := uuid.New().String()

		// Add to context
		ctx := context.WithValue(c.Request.Context(), logger.RequestIDKey, requestID)
		c.Request = c.Request.WithContext(ctx)

		// Add to response header
		c.Header("X-Request-ID", requestID)

		c.Next()
	}
}

// Logger middleware logs HTTP requests
func (h *Handlers) Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Log request
		latency := time.Since(start)

		if raw != "" {
			path = path + "?" + raw
		}

		logger.WithRequest(
			c.GetString("request_id"),
			c.Request.Method,
			path,
		).WithFields(map[string]interface{}{
			"status":     c.Writer.Status(),
			"latency_ms": latency.Milliseconds(),
			"ip":         c.ClientIP(),
			"user_agent": c.Request.UserAgent(),
		}).Info("HTTP request processed")
	}
}

// ErrorHandler middleware handles panics and converts them to errors
func (h *Handlers) ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				logger.WithContext(c.Request.Context()).
					WithField("error", err).
					Error("Panic recovered")

				h.respondWithError(c, errors.ErrInternalError)
				c.Abort()
			}
		}()

		c.Next()
	}
}

// CORS middleware handles Cross-Origin Resource Sharing
func (h *Handlers) CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Request-ID")
		c.Header("Access-Control-Expose-Headers", "X-Request-ID")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// MetricsHandler returns Prometheus metrics
func (h *Handlers) MetricsHandler() gin.HandlerFunc {
	return gin.WrapH(promhttp.Handler())
}

// Helper methods

// respondWithError responds with an error in a consistent format
func (h *Handlers) respondWithError(c *gin.Context, err error) {
	statusCode := errors.GetStatusCode(err)
	response := errors.ToErrorResponse(err)

	c.JSON(statusCode, response)
}
