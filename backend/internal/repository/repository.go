// Package repository provides data access layer implementations
package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"indico-backend/internal/errors"
	"indico-backend/internal/models"

	"github.com/google/uuid"
)

// ProductRepository handles product data operations
type ProductRepository interface {
	GetByID(ctx context.Context, id int) (*models.Product, error)
	GetByIDForUpdate(ctx context.Context, tx *sql.Tx, id int) (*models.Product, error)
	UpdateStock(ctx context.Context, tx *sql.Tx, id int, quantity int, version int) error
	Create(ctx context.Context, product *models.Product) error
}

// OrderRepository handles order data operations
type OrderRepository interface {
	Create(ctx context.Context, tx *sql.Tx, order *models.Order) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Order, error)
	List(ctx context.Context, limit, offset int) ([]*models.Order, error)
}

// TransactionRepository handles transaction data operations
type TransactionRepository interface {
	GetBatch(ctx context.Context, offset, limit int, from, to time.Time) ([]*models.Transaction, error)
	GetTotalCount(ctx context.Context, from, to time.Time) (int, error)
	Create(ctx context.Context, tx *models.Transaction) error
	BulkCreate(ctx context.Context, transactions []*models.Transaction) error
}

// SettlementRepository handles settlement data operations
type SettlementRepository interface {
	Upsert(ctx context.Context, tx *sql.Tx, settlement *models.Settlement) error
	GetByMerchantAndDate(ctx context.Context, merchantID string, date time.Time) (*models.Settlement, error)
}

// JobRepository handles job data operations
type JobRepository interface {
	Create(ctx context.Context, job *models.Job) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Job, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status models.JobStatus) error
	UpdateProgress(ctx context.Context, id uuid.UUID, progress float64, processed int) error
	UpdateResult(ctx context.Context, id uuid.UUID, resultPath, downloadURL string) error
	UpdateError(ctx context.Context, id uuid.UUID, errMsg string) error
	MarkStarted(ctx context.Context, id uuid.UUID) error
	MarkCompleted(ctx context.Context, id uuid.UUID) error
	Cancel(ctx context.Context, id uuid.UUID) error
	IsCancelled(ctx context.Context, id uuid.UUID) (bool, error)
}

// productRepository implements ProductRepository
type productRepository struct {
	db *sql.DB
}

// NewProductRepository creates a new product repository
func NewProductRepository(db *sql.DB) ProductRepository {
	return &productRepository{db: db}
}

func (r *productRepository) GetByID(ctx context.Context, id int) (*models.Product, error) {
	query := `
		SELECT id, name, stock, price, version, created_at, updated_at
		FROM products 
		WHERE id = $1`

	var product models.Product
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&product.ID,
		&product.Name,
		&product.Stock,
		&product.Price,
		&product.Version,
		&product.CreatedAt,
		&product.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, errors.ErrProductNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get product: %w", err)
	}

	return &product, nil
}

func (r *productRepository) GetByIDForUpdate(ctx context.Context, tx *sql.Tx, id int) (*models.Product, error) {
	query := `
		SELECT id, name, stock, price, version, created_at, updated_at
		FROM products 
		WHERE id = $1
		FOR UPDATE`

	var product models.Product
	err := tx.QueryRowContext(ctx, query, id).Scan(
		&product.ID,
		&product.Name,
		&product.Stock,
		&product.Price,
		&product.Version,
		&product.CreatedAt,
		&product.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, errors.ErrProductNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get product for update: %w", err)
	}

	return &product, nil
}

func (r *productRepository) UpdateStock(ctx context.Context, tx *sql.Tx, id int, quantity int, version int) error {
	query := `
		UPDATE products 
		SET stock = stock - $1, version = version + 1, updated_at = NOW()
		WHERE id = $2 AND version = $3 AND stock >= $1`

	result, err := tx.ExecContext(ctx, query, quantity, id, version)
	if err != nil {
		return fmt.Errorf("failed to update stock: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		// Check if it's a stock issue or version conflict
		var currentStock int
		checkQuery := "SELECT stock FROM products WHERE id = $1"
		if err := tx.QueryRowContext(ctx, checkQuery, id).Scan(&currentStock); err != nil {
			return fmt.Errorf("failed to check current stock: %w", err)
		}

		if currentStock < quantity {
			return errors.ErrOutOfStock
		}

		return errors.NewConcurrencyError("product was modified by another transaction")
	}

	return nil
}

func (r *productRepository) Create(ctx context.Context, product *models.Product) error {
	query := `
		INSERT INTO products (name, stock, price, version, created_at, updated_at)
		VALUES ($1, $2, $3, 1, NOW(), NOW())
		RETURNING id, created_at, updated_at`

	err := r.db.QueryRowContext(ctx, query, product.Name, product.Stock, product.Price).Scan(
		&product.ID,
		&product.CreatedAt,
		&product.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create product: %w", err)
	}

	product.Version = 1
	return nil
}

// orderRepository implements OrderRepository
type orderRepository struct {
	db *sql.DB
}

// NewOrderRepository creates a new order repository
func NewOrderRepository(db *sql.DB) OrderRepository {
	return &orderRepository{db: db}
}

func (r *orderRepository) Create(ctx context.Context, tx *sql.Tx, order *models.Order) error {
	query := `
		INSERT INTO orders (id, product_id, buyer_id, quantity, status, total_cents, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		RETURNING created_at, updated_at`

	err := tx.QueryRowContext(ctx, query,
		order.ID,
		order.ProductID,
		order.BuyerID,
		order.Quantity,
		order.Status,
		order.TotalCents,
	).Scan(&order.CreatedAt, &order.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create order: %w", err)
	}

	return nil
}

func (r *orderRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Order, error) {
	query := `
		SELECT o.id, o.product_id, o.buyer_id, o.quantity, o.status, o.total_cents, 
			   o.created_at, o.updated_at,
			   p.id, p.name, p.stock, p.price, p.version, p.created_at, p.updated_at
		FROM orders o
		JOIN products p ON o.product_id = p.id
		WHERE o.id = $1`

	var order models.Order
	var product models.Product

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&order.ID,
		&order.ProductID,
		&order.BuyerID,
		&order.Quantity,
		&order.Status,
		&order.TotalCents,
		&order.CreatedAt,
		&order.UpdatedAt,
		&product.ID,
		&product.Name,
		&product.Stock,
		&product.Price,
		&product.Version,
		&product.CreatedAt,
		&product.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, errors.ErrOrderNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	order.Product = &product
	return &order, nil
}

func (r *orderRepository) List(ctx context.Context, limit, offset int) ([]*models.Order, error) {
	query := `
		SELECT id, product_id, buyer_id, quantity, status, total_cents, created_at, updated_at
		FROM orders
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list orders: %w", err)
	}
	defer rows.Close()

	var orders []*models.Order
	for rows.Next() {
		var order models.Order
		err := rows.Scan(
			&order.ID,
			&order.ProductID,
			&order.BuyerID,
			&order.Quantity,
			&order.Status,
			&order.TotalCents,
			&order.CreatedAt,
			&order.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order: %w", err)
		}
		orders = append(orders, &order)
	}

	return orders, nil
}

// transactionRepository implements TransactionRepository
type transactionRepository struct {
	db *sql.DB
}

// NewTransactionRepository creates a new transaction repository
func NewTransactionRepository(db *sql.DB) TransactionRepository {
	return &transactionRepository{db: db}
}

func (r *transactionRepository) GetBatch(ctx context.Context, offset, limit int, from, to time.Time) ([]*models.Transaction, error) {
	query := `
		SELECT id, merchant_id, amount_cents, fee_cents, status, paid_at, created_at
		FROM transactions
		WHERE paid_at >= $1 AND paid_at < $2 AND status = 'COMPLETED'
		ORDER BY id
		LIMIT $3 OFFSET $4`

	rows, err := r.db.QueryContext(ctx, query, from, to, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction batch: %w", err)
	}
	defer rows.Close()

	var transactions []*models.Transaction
	for rows.Next() {
		var tx models.Transaction
		err := rows.Scan(
			&tx.ID,
			&tx.MerchantID,
			&tx.AmountCents,
			&tx.FeeCents,
			&tx.Status,
			&tx.PaidAt,
			&tx.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}
		transactions = append(transactions, &tx)
	}

	return transactions, nil
}

func (r *transactionRepository) GetTotalCount(ctx context.Context, from, to time.Time) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM transactions
		WHERE paid_at >= $1 AND paid_at < $2 AND status = 'COMPLETED'`

	var count int
	err := r.db.QueryRowContext(ctx, query, from, to).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get transaction count: %w", err)
	}

	return count, nil
}

func (r *transactionRepository) Create(ctx context.Context, tx *models.Transaction) error {
	query := `
		INSERT INTO transactions (merchant_id, amount_cents, fee_cents, status, paid_at, created_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		RETURNING id, created_at`

	err := r.db.QueryRowContext(ctx, query,
		tx.MerchantID,
		tx.AmountCents,
		tx.FeeCents,
		tx.Status,
		tx.PaidAt,
	).Scan(&tx.ID, &tx.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}

	return nil
}

func (r *transactionRepository) BulkCreate(ctx context.Context, transactions []*models.Transaction) error {
	if len(transactions) == 0 {
		return nil
	}

	query := `
		INSERT INTO transactions (merchant_id, amount_cents, fee_cents, status, paid_at, created_at)
		VALUES `

	args := make([]interface{}, 0, len(transactions)*5)
	placeholders := make([]string, 0, len(transactions))

	for i, tx := range transactions {
		placeholderGroup := fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, NOW())",
			i*5+1, i*5+2, i*5+3, i*5+4, i*5+5)
		placeholders = append(placeholders, placeholderGroup)

		args = append(args,
			tx.MerchantID,
			tx.AmountCents,
			tx.FeeCents,
			tx.Status,
			tx.PaidAt,
		)
	}

	query += strings.Join(placeholders, ",")

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to bulk create transactions: %w", err)
	}

	return nil
}

// settlementRepository implements SettlementRepository
type settlementRepository struct {
	db *sql.DB
}

// NewSettlementRepository creates a new settlement repository
func NewSettlementRepository(db *sql.DB) SettlementRepository {
	return &settlementRepository{db: db}
}

func (r *settlementRepository) Upsert(ctx context.Context, tx *sql.Tx, settlement *models.Settlement) error {
	query := `
		INSERT INTO settlements (merchant_id, date, gross_cents, fee_cents, net_cents, txn_count, generated_at, unique_run_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
		ON CONFLICT (merchant_id, date)
		DO UPDATE SET 
			gross_cents = settlements.gross_cents + EXCLUDED.gross_cents,
			fee_cents = settlements.fee_cents + EXCLUDED.fee_cents,
			net_cents = settlements.net_cents + EXCLUDED.net_cents,
			txn_count = settlements.txn_count + EXCLUDED.txn_count,
			updated_at = NOW()
		RETURNING id, created_at, updated_at`

	err := tx.QueryRowContext(ctx, query,
		settlement.MerchantID,
		settlement.Date,
		settlement.GrossCents,
		settlement.FeeCents,
		settlement.NetCents,
		settlement.TxnCount,
		settlement.GeneratedAt,
		settlement.UniqueRunID,
	).Scan(&settlement.ID, &settlement.CreatedAt, &settlement.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to upsert settlement: %w", err)
	}

	return nil
}

func (r *settlementRepository) GetByMerchantAndDate(ctx context.Context, merchantID string, date time.Time) (*models.Settlement, error) {
	query := `
		SELECT id, merchant_id, date, gross_cents, fee_cents, net_cents, txn_count, generated_at, unique_run_id, created_at, updated_at
		FROM settlements
		WHERE merchant_id = $1 AND date = $2`

	var settlement models.Settlement
	err := r.db.QueryRowContext(ctx, query, merchantID, date).Scan(
		&settlement.ID,
		&settlement.MerchantID,
		&settlement.Date,
		&settlement.GrossCents,
		&settlement.FeeCents,
		&settlement.NetCents,
		&settlement.TxnCount,
		&settlement.GeneratedAt,
		&settlement.UniqueRunID,
		&settlement.CreatedAt,
		&settlement.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // Settlement not found is not an error in this case
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get settlement: %w", err)
	}

	return &settlement, nil
}

// jobRepository implements JobRepository
type jobRepository struct {
	db *sql.DB
}

// NewJobRepository creates a new job repository
func NewJobRepository(db *sql.DB) JobRepository {
	return &jobRepository{db: db}
}

func (r *jobRepository) Create(ctx context.Context, job *models.Job) error {
	paramsJSON, err := json.Marshal(job.Parameters)
	if err != nil {
		return fmt.Errorf("failed to marshal job parameters: %w", err)
	}

	query := `
		INSERT INTO jobs (id, type, status, progress, processed, total, parameters, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		RETURNING created_at, updated_at`

	err = r.db.QueryRowContext(ctx, query,
		job.ID,
		job.Type,
		job.Status,
		job.Progress,
		job.Processed,
		job.Total,
		string(paramsJSON),
	).Scan(&job.CreatedAt, &job.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create job: %w", err)
	}

	return nil
}

func (r *jobRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Job, error) {
	query := `
		SELECT id, type, status, progress, processed, total, parameters, result_path, download_url, error, started_at, completed_at, created_at, updated_at
		FROM jobs
		WHERE id = $1`

	var job models.Job
	var params string

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&job.ID,
		&job.Type,
		&job.Status,
		&job.Progress,
		&job.Processed,
		&job.Total,
		&params,
		&job.ResultPath,
		&job.DownloadURL,
		&job.Error,
		&job.StartedAt,
		&job.CompletedAt,
		&job.CreatedAt,
		&job.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, errors.ErrJobNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	job.Parameters = params
	return &job, nil
}

func (r *jobRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status models.JobStatus) error {
	query := `UPDATE jobs SET status = $1, updated_at = NOW() WHERE id = $2`

	_, err := r.db.ExecContext(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	return nil
}

func (r *jobRepository) UpdateProgress(ctx context.Context, id uuid.UUID, progress float64, processed int) error {
	query := `UPDATE jobs SET progress = $1, processed = $2, updated_at = NOW() WHERE id = $3`

	_, err := r.db.ExecContext(ctx, query, progress, processed, id)
	if err != nil {
		return fmt.Errorf("failed to update job progress: %w", err)
	}

	return nil
}

func (r *jobRepository) UpdateResult(ctx context.Context, id uuid.UUID, resultPath, downloadURL string) error {
	query := `UPDATE jobs SET result_path = $1, download_url = $2, updated_at = NOW() WHERE id = $3`

	_, err := r.db.ExecContext(ctx, query, resultPath, downloadURL, id)
	if err != nil {
		return fmt.Errorf("failed to update job result: %w", err)
	}

	return nil
}

func (r *jobRepository) UpdateError(ctx context.Context, id uuid.UUID, errMsg string) error {
	query := `UPDATE jobs SET error = $1, updated_at = NOW() WHERE id = $2`

	_, err := r.db.ExecContext(ctx, query, errMsg, id)
	if err != nil {
		return fmt.Errorf("failed to update job error: %w", err)
	}

	return nil
}

func (r *jobRepository) MarkStarted(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE jobs SET status = $1, started_at = NOW(), updated_at = NOW() WHERE id = $2`

	_, err := r.db.ExecContext(ctx, query, models.JobStatusRunning, id)
	if err != nil {
		return fmt.Errorf("failed to mark job as started: %w", err)
	}

	return nil
}

func (r *jobRepository) MarkCompleted(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE jobs SET status = $1, completed_at = NOW(), updated_at = NOW() WHERE id = $2`

	_, err := r.db.ExecContext(ctx, query, models.JobStatusCompleted, id)
	if err != nil {
		return fmt.Errorf("failed to mark job as completed: %w", err)
	}

	return nil
}

func (r *jobRepository) Cancel(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE jobs 
		SET status = $1, updated_at = NOW() 
		WHERE id = $2 AND status IN ($3, $4)`

	result, err := r.db.ExecContext(ctx, query, models.JobStatusCancelled, id, models.JobStatusQueued, models.JobStatusRunning)
	if err != nil {
		return fmt.Errorf("failed to cancel job: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return errors.ErrJobAlreadyCancelled
	}

	return nil
}

func (r *jobRepository) IsCancelled(ctx context.Context, id uuid.UUID) (bool, error) {
	query := `SELECT status FROM jobs WHERE id = $1`

	var status models.JobStatus
	err := r.db.QueryRowContext(ctx, query, id).Scan(&status)
	if err == sql.ErrNoRows {
		return false, errors.ErrJobNotFound
	}
	if err != nil {
		return false, fmt.Errorf("failed to check job status: %w", err)
	}

	return status == models.JobStatusCancelled, nil
}
