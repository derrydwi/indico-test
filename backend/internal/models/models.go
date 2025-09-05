// Package models defines the domain models for the application
package models

import (
	"time"

	"github.com/google/uuid"
)

// Product represents a product in the system
type Product struct {
	ID        int       `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Stock     int       `json:"stock" db:"stock"`
	Price     int       `json:"price" db:"price"`     // in cents
	Version   int       `json:"version" db:"version"` // for optimistic locking
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Order represents an order in the system
type Order struct {
	ID         uuid.UUID   `json:"id" db:"id"`
	ProductID  int         `json:"product_id" db:"product_id"`
	BuyerID    string      `json:"buyer_id" db:"buyer_id"`
	Quantity   int         `json:"quantity" db:"quantity"`
	Status     OrderStatus `json:"status" db:"status"`
	TotalCents int         `json:"total_cents" db:"total_cents"`
	CreatedAt  time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time   `json:"updated_at" db:"updated_at"`
	Product    *Product    `json:"product,omitempty"` // for joins
}

// OrderStatus represents the status of an order
type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "PENDING"
	OrderStatusConfirmed OrderStatus = "CONFIRMED"
	OrderStatusCancelled OrderStatus = "CANCELLED"
)

// Transaction represents a financial transaction
type Transaction struct {
	ID          int               `json:"id" db:"id"`
	MerchantID  string            `json:"merchant_id" db:"merchant_id"`
	AmountCents int               `json:"amount_cents" db:"amount_cents"`
	FeeCents    int               `json:"fee_cents" db:"fee_cents"`
	Status      TransactionStatus `json:"status" db:"status"`
	PaidAt      time.Time         `json:"paid_at" db:"paid_at"`
	CreatedAt   time.Time         `json:"created_at" db:"created_at"`
}

// TransactionStatus represents the status of a transaction
type TransactionStatus string

const (
	TransactionStatusPending   TransactionStatus = "PENDING"
	TransactionStatusCompleted TransactionStatus = "COMPLETED"
	TransactionStatusFailed    TransactionStatus = "FAILED"
)

// Settlement represents an aggregated settlement
type Settlement struct {
	ID          int       `json:"id" db:"id"`
	MerchantID  string    `json:"merchant_id" db:"merchant_id"`
	Date        time.Time `json:"date" db:"date"`
	GrossCents  int       `json:"gross_cents" db:"gross_cents"`
	FeeCents    int       `json:"fee_cents" db:"fee_cents"`
	NetCents    int       `json:"net_cents" db:"net_cents"`
	TxnCount    int       `json:"txn_count" db:"txn_count"`
	GeneratedAt time.Time `json:"generated_at" db:"generated_at"`
	UniqueRunID uuid.UUID `json:"unique_run_id" db:"unique_run_id"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// Job represents a background job
type Job struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	Type        JobType    `json:"type" db:"type"`
	Status      JobStatus  `json:"status" db:"status"`
	Progress    float64    `json:"progress" db:"progress"`
	Processed   int        `json:"processed" db:"processed"`
	Total       int        `json:"total" db:"total"`
	Parameters  string     `json:"parameters" db:"parameters"` // JSON
	ResultPath  *string    `json:"result_path,omitempty" db:"result_path"`
	DownloadURL *string    `json:"download_url,omitempty" db:"download_url"`
	Error       *string    `json:"error,omitempty" db:"error"`
	StartedAt   *time.Time `json:"started_at,omitempty" db:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}

// JobType represents the type of job
type JobType string

const (
	JobTypeSettlement JobType = "SETTLEMENT"
)

// JobStatus represents the status of a job
type JobStatus string

const (
	JobStatusQueued    JobStatus = "QUEUED"
	JobStatusRunning   JobStatus = "RUNNING"
	JobStatusCompleted JobStatus = "COMPLETED"
	JobStatusFailed    JobStatus = "FAILED"
	JobStatusCancelled JobStatus = "CANCELLED"
)

// SettlementJobParams represents parameters for settlement job
type SettlementJobParams struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// CreateOrderRequest represents a request to create an order
type CreateOrderRequest struct {
	ProductID int    `json:"product_id" binding:"required,min=1"`
	Quantity  int    `json:"quantity" binding:"required,min=1"`
	BuyerID   string `json:"buyer_id" binding:"required"`
}

// CreateSettlementJobRequest represents a request to create a settlement job
type CreateSettlementJobRequest struct {
	From string `json:"from" binding:"required"`
	To   string `json:"to" binding:"required"`
}

// HealthCheck represents the health status of the service
type HealthCheck struct {
	Status    string            `json:"status"`
	Version   string            `json:"version"`
	Checks    map[string]string `json:"checks"`
	Uptime    string            `json:"uptime"`
	Timestamp time.Time         `json:"timestamp"`
}
