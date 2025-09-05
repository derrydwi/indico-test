// Package service provides business logic implementation
package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"indico-backend/internal/database"
	"indico-backend/internal/errors"
	"indico-backend/internal/logger"
	"indico-backend/internal/metrics"
	"indico-backend/internal/models"
	"indico-backend/internal/repository"

	"github.com/google/uuid"
)

// OrderService handles order business logic
type OrderService interface {
	CreateOrder(ctx context.Context, req *models.CreateOrderRequest) (*models.Order, error)
	GetOrder(ctx context.Context, id uuid.UUID) (*models.Order, error)
	ListOrders(ctx context.Context, limit, offset int) ([]*models.Order, error)
}

// JobService handles job business logic
type JobService interface {
	CreateSettlementJob(ctx context.Context, req *models.CreateSettlementJobRequest) (*models.Job, error)
	GetJob(ctx context.Context, id uuid.UUID) (*models.Job, error)
	CancelJob(ctx context.Context, id uuid.UUID) error
}

// HealthService handles health check logic
type HealthService interface {
	Check(ctx context.Context) (*models.HealthCheck, error)
}

// Services contains all service implementations
type Services struct {
	Order  OrderService
	Job    JobService
	Health HealthService
}

// Dependencies contains service dependencies
type Dependencies struct {
	DB           *database.DB
	ProductRepo  repository.ProductRepository
	OrderRepo    repository.OrderRepository
	TxRepo       repository.TransactionRepository
	SettleRepo   repository.SettlementRepository
	JobRepo      repository.JobRepository
	JobProcessor *JobProcessor
}

// NewServices creates a new services instance
func NewServices(deps *Dependencies) *Services {
	return &Services{
		Order:  NewOrderService(deps),
		Job:    NewJobService(deps),
		Health: NewHealthService(deps),
	}
}

// orderService implements OrderService
type orderService struct {
	db          *database.DB
	productRepo repository.ProductRepository
	orderRepo   repository.OrderRepository
}

// NewOrderService creates a new order service
func NewOrderService(deps *Dependencies) OrderService {
	return &orderService{
		db:          deps.DB,
		productRepo: deps.ProductRepo,
		orderRepo:   deps.OrderRepo,
	}
}

func (s *orderService) CreateOrder(ctx context.Context, req *models.CreateOrderRequest) (*models.Order, error) {
	// Validate request
	if req.Quantity <= 0 {
		return nil, errors.NewValidationError("quantity must be positive")
	}

	// Create order within transaction to ensure consistency
	var order *models.Order
	err := s.db.WithTx(ctx, func(tx *sql.Tx) error {
		// Get product with lock for update
		product, err := s.productRepo.GetByIDForUpdate(ctx, tx, req.ProductID)
		if err != nil {
			return err
		}

		// Check stock availability
		if product.Stock < req.Quantity {
			return errors.ErrOutOfStock
		}

		// Calculate total
		totalCents := product.Price * req.Quantity

		// Create order
		order = &models.Order{
			ID:         uuid.New(),
			ProductID:  req.ProductID,
			BuyerID:    req.BuyerID,
			Quantity:   req.Quantity,
			Status:     models.OrderStatusPending,
			TotalCents: totalCents,
		}

		if err := s.orderRepo.Create(ctx, tx, order); err != nil {
			return err
		}

		// Update product stock with optimistic locking
		if err := s.productRepo.UpdateStock(ctx, tx, product.ID, req.Quantity, product.Version); err != nil {
			return err
		}

		// Update order status to confirmed
		order.Status = models.OrderStatusConfirmed
		return nil
	})

	if err != nil {
		logger.WithContext(ctx).WithError(err).Error("Failed to create order")
		return nil, err
	}

	logger.WithContext(ctx).
		WithField("order_id", order.ID).
		WithField("buyer_id", req.BuyerID).
		WithField("product_id", req.ProductID).
		WithField("quantity", req.Quantity).
		Info("Order created successfully")

	return order, nil
}

func (s *orderService) GetOrder(ctx context.Context, id uuid.UUID) (*models.Order, error) {
	order, err := s.orderRepo.GetByID(ctx, id)
	if err != nil {
		logger.WithContext(ctx).WithError(err).WithField("order_id", id).Error("Failed to get order")
		return nil, err
	}

	return order, nil
}

func (s *orderService) ListOrders(ctx context.Context, limit, offset int) ([]*models.Order, error) {
	if limit <= 0 || limit > 100 {
		limit = 10 // Default limit
	}
	if offset < 0 {
		offset = 0
	}

	orders, err := s.orderRepo.List(ctx, limit, offset)
	if err != nil {
		logger.WithContext(ctx).WithError(err).Error("Failed to list orders")
		return nil, err
	}

	return orders, nil
}

// jobService implements JobService
type jobService struct {
	db           *database.DB
	jobRepo      repository.JobRepository
	jobProcessor *JobProcessor
}

// NewJobService creates a new job service
func NewJobService(deps *Dependencies) JobService {
	return &jobService{
		db:           deps.DB,
		jobRepo:      deps.JobRepo,
		jobProcessor: deps.JobProcessor,
	}
}

func (s *jobService) CreateSettlementJob(ctx context.Context, req *models.CreateSettlementJobRequest) (*models.Job, error) {
	// Validate date format and range
	from, err := time.Parse("2006-01-02", req.From)
	if err != nil {
		return nil, errors.NewValidationError("invalid from date format, expected YYYY-MM-DD")
	}

	to, err := time.Parse("2006-01-02", req.To)
	if err != nil {
		return nil, errors.NewValidationError("invalid to date format, expected YYYY-MM-DD")
	}

	if to.Before(from) {
		return nil, errors.NewValidationError("to date must be after from date")
	}

	// Create job parameters
	params := models.SettlementJobParams{
		From: req.From,
		To:   req.To,
	}

	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal job parameters: %w", err)
	}

	// Create job
	job := &models.Job{
		ID:         uuid.New(),
		Type:       models.JobTypeSettlement,
		Status:     models.JobStatusQueued,
		Progress:   0,
		Processed:  0,
		Total:      0, // Will be calculated when job starts
		Parameters: string(paramsJSON),
	}

	if err := s.jobRepo.Create(ctx, job); err != nil {
		logger.WithContext(ctx).WithError(err).Error("Failed to create settlement job")
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	// Queue job for processing
	if err := s.jobProcessor.QueueJob(ctx, job); err != nil {
		logger.WithContext(ctx).WithError(err).WithField("job_id", job.ID).Error("Failed to queue job")
		return nil, fmt.Errorf("failed to queue job: %w", err)
	}

	// Record metrics
	metrics.JobsCreated.WithLabelValues(string(job.Type)).Inc()

	logger.WithContext(ctx).
		WithField("job_id", job.ID).
		WithField("from", req.From).
		WithField("to", req.To).
		Info("Settlement job created and queued")

	return job, nil
}

func (s *jobService) GetJob(ctx context.Context, id uuid.UUID) (*models.Job, error) {
	job, err := s.jobRepo.GetByID(ctx, id)
	if err != nil {
		logger.WithContext(ctx).WithError(err).WithField("job_id", id).Error("Failed to get job")
		return nil, err
	}

	return job, nil
}

func (s *jobService) CancelJob(ctx context.Context, id uuid.UUID) error {
	// Mark job as cancelled in database
	err := s.jobRepo.Cancel(ctx, id)
	if err != nil {
		logger.WithContext(ctx).WithError(err).WithField("job_id", id).Error("Failed to cancel job")
		return err
	}

	// Cancel the job context if it's currently running
	s.jobProcessor.CancelJob(id)

	logger.WithContext(ctx).WithField("job_id", id).Info("Job cancelled")
	return nil
}

// healthService implements HealthService
type healthService struct {
	db *database.DB
}

// NewHealthService creates a new health service
func NewHealthService(deps *Dependencies) HealthService {
	return &healthService{
		db: deps.DB,
	}
}

var startTime = time.Now()

func (s *healthService) Check(ctx context.Context) (*models.HealthCheck, error) {
	checks := make(map[string]string)

	// Check database
	if err := s.db.Health(ctx); err != nil {
		checks["database"] = "unhealthy: " + err.Error()
	} else {
		checks["database"] = "healthy"
	}

	// Determine overall status
	status := "healthy"
	for _, check := range checks {
		if check != "healthy" && check[:7] != "healthy" {
			status = "unhealthy"
			break
		}
	}

	uptime := time.Since(startTime).String()

	return &models.HealthCheck{
		Status:    status,
		Version:   "1.0.0",
		Checks:    checks,
		Uptime:    uptime,
		Timestamp: time.Now(),
	}, nil
}

// // Utility function to create settlement CSV
// func createSettlementCSV(settlements []*models.Settlement, filePath string) error {
// 	// Ensure directory exists
// 	dir := filepath.Dir(filePath)
// 	if err := os.MkdirAll(dir, 0755); err != nil {
// 		return fmt.Errorf("failed to create directory: %w", err)
// 	}

// 	file, err := os.Create(filePath)
// 	if err != nil {
// 		return fmt.Errorf("failed to create CSV file: %w", err)
// 	}
// 	defer file.Close()

// 	writer := csv.NewWriter(file)
// 	defer writer.Flush()

// 	// Write header
// 	if err := writer.Write([]string{"merchant_id", "date", "gross", "fee", "net", "txn_count"}); err != nil {
// 		return fmt.Errorf("failed to write CSV header: %w", err)
// 	}

// 	// Write data
// 	for _, settlement := range settlements {
// 		record := []string{
// 			settlement.MerchantID,
// 			settlement.Date.Format("2006-01-02"),
// 			fmt.Sprintf("%.2f", float64(settlement.GrossCents)/100),
// 			fmt.Sprintf("%.2f", float64(settlement.FeeCents)/100),
// 			fmt.Sprintf("%.2f", float64(settlement.NetCents)/100),
// 			fmt.Sprintf("%d", settlement.TxnCount),
// 		}
// 		if err := writer.Write(record); err != nil {
// 			return fmt.Errorf("failed to write CSV record: %w", err)
// 		}
// 	}

// 	return nil
// }
