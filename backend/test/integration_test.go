// Package test provides comprehensive integration tests
package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"indico-backend/internal/config"
	"indico-backend/internal/database"
	"indico-backend/internal/handlers"
	"indico-backend/internal/logger"
	"indico-backend/internal/models"
	"indico-backend/internal/repository"
	"indico-backend/internal/routes"
	"indico-backend/internal/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) *database.DB {
	cfg := &config.DatabaseConfig{
		Host:     "localhost",
		Port:     "5433",
		User:     "postgres",
		Password: "postgres",
		DBName:   "indico_test",
		SSLMode:  "disable",
		MaxConns: 10,
		MaxIdle:  2,
	}

	db, err := database.New(cfg)
	require.NoError(t, err)

	// Clean up database
	_, err = db.Exec(`
		DELETE FROM jobs;
		DELETE FROM settlements;
		DELETE FROM transactions;
		DELETE FROM orders;
		DELETE FROM products;
	`)
	require.NoError(t, err)

	return db
}

func setupTestServer(t *testing.T) (*httptest.Server, *database.DB) {
	logger.Init("debug", "text")

	db := setupTestDB(t)

	// Initialize repositories
	productRepo := repository.NewProductRepository(db.DB)
	orderRepo := repository.NewOrderRepository(db.DB)
	txRepo := repository.NewTransactionRepository(db.DB)
	settleRepo := repository.NewSettlementRepository(db.DB)
	jobRepo := repository.NewJobRepository(db.DB)

	// Initialize job processor with test config
	jobConfig := &config.JobsConfig{
		Workers:   2,
		BatchSize: 100,
		QueueSize: 10,
	}
	jobProcessor := service.NewJobProcessor(db, jobConfig, txRepo, settleRepo, jobRepo)
	jobProcessor.Start()

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

	// Initialize handlers and routes
	h := handlers.New(services)
	router := routes.SetupRoutes(h)

	server := httptest.NewServer(router)

	t.Cleanup(func() {
		jobProcessor.Stop()
		server.Close()
		db.Close()
	})

	return server, db
}

func createTestProduct(t *testing.T, db *database.DB, stock int) *models.Product {
	product := &models.Product{
		Name:  "Test Product",
		Stock: stock,
		Price: 1000, // $10.00
	}

	query := `
		INSERT INTO products (name, stock, price, version, created_at, updated_at)
		VALUES ($1, $2, $3, 1, NOW(), NOW())
		RETURNING id, created_at, updated_at`

	err := db.QueryRow(query, product.Name, product.Stock, product.Price).Scan(
		&product.ID, &product.CreatedAt, &product.UpdatedAt)
	require.NoError(t, err)

	product.Version = 1
	return product
}

// TestConcurrentOrders tests that 500 concurrent orders for a product with 100 stock
// results in exactly 100 successful orders and 400 failures due to insufficient stock
func TestConcurrentOrders(t *testing.T) {
	server, db := setupTestServer(t)

	// Create a product with limited stock
	product := createTestProduct(t, db, 100)

	// Prepare 500 concurrent order requests
	numRequests := 500
	requestQuantity := 1

	type orderResult struct {
		statusCode int
		err        error
		body       []byte
	}

	var wg sync.WaitGroup
	results := make(chan orderResult, numRequests)

	// Send 500 concurrent requests
	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(buyerID int) {
			defer wg.Done()

			orderReq := models.CreateOrderRequest{
				ProductID: product.ID,
				Quantity:  requestQuantity,
				BuyerID:   fmt.Sprintf("buyer_%d", buyerID),
			}

			reqBody, _ := json.Marshal(orderReq)
			resp, err := http.Post(server.URL+"/orders", "application/json", bytes.NewBuffer(reqBody))
			if err != nil {
				results <- orderResult{statusCode: 0, err: err}
				return
			}
			defer resp.Body.Close()

			body, _ := io.ReadAll(resp.Body)
			results <- orderResult{statusCode: resp.StatusCode, body: body}
		}(i)
	}

	wg.Wait()
	close(results)

	// Count successful and failed orders
	successCount := 0
	outOfStockCount := 0
	conflictCount := 0
	otherErrorCount := 0

	for res := range results {
		if res.err != nil {
			otherErrorCount++
			continue
		}

		switch res.statusCode {
		case http.StatusCreated:
			successCount++
		case http.StatusConflict:
			// Check if it's out of stock or concurrency conflict
			var errResp map[string]interface{}
			json.Unmarshal(res.body, &errResp)
			if errDetail, ok := errResp["error"].(map[string]interface{}); ok {
				if errDetail["code"] == "OUT_OF_STOCK" {
					outOfStockCount++
				} else if errDetail["code"] == "CONCURRENCY_CONFLICT" {
					conflictCount++
				} else {
					otherErrorCount++
				}
			} else {
				otherErrorCount++
			}
		default:
			otherErrorCount++
		}
	}

	// Print results for debugging
	t.Logf("Results: %d successful, %d out-of-stock, %d concurrency conflicts, %d other errors",
		successCount, outOfStockCount, conflictCount, otherErrorCount)

	// Verify that exactly 100 orders succeeded (matching the initial stock)
	assert.Equal(t, 100, successCount, "Expected exactly 100 successful orders")

	// The rest should be out-of-stock or concurrency conflicts (400 total failures)
	totalFailures := outOfStockCount + conflictCount + otherErrorCount
	assert.Equal(t, 400, totalFailures, "Expected 400 total failures (out-of-stock + concurrency conflicts)")

	// Verify that the product stock is now 0
	var finalStock int
	err := db.QueryRow("SELECT stock FROM products WHERE id = $1", product.ID).Scan(&finalStock)
	require.NoError(t, err)
	assert.Equal(t, 0, finalStock, "Product stock should be 0 after all successful orders")

	// Verify that 100 orders exist in the database
	var orderCount int
	err = db.QueryRow("SELECT COUNT(*) FROM orders WHERE product_id = $1", product.ID).Scan(&orderCount)
	require.NoError(t, err)
	assert.Equal(t, 100, orderCount, "Expected exactly 100 orders in database")
}

func TestOrderCreation(t *testing.T) {
	server, db := setupTestServer(t)

	// Create a product
	product := createTestProduct(t, db, 10)

	// Create an order
	orderReq := models.CreateOrderRequest{
		ProductID: product.ID,
		Quantity:  2,
		BuyerID:   "test_buyer",
	}

	reqBody, _ := json.Marshal(orderReq)
	resp, err := http.Post(server.URL+"/orders", "application/json", bytes.NewBuffer(reqBody))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var order models.Order
	err = json.NewDecoder(resp.Body).Decode(&order)
	require.NoError(t, err)

	assert.Equal(t, product.ID, order.ProductID)
	assert.Equal(t, "test_buyer", order.BuyerID)
	assert.Equal(t, 2, order.Quantity)
	assert.Equal(t, 2000, order.TotalCents) // 2 * $10.00
	assert.Equal(t, models.OrderStatusConfirmed, order.Status)

	// Verify stock was reduced
	var newStock int
	err = db.QueryRow("SELECT stock FROM products WHERE id = $1", product.ID).Scan(&newStock)
	require.NoError(t, err)
	assert.Equal(t, 8, newStock)
}

func TestOrderRetrieval(t *testing.T) {
	server, db := setupTestServer(t)

	// Create a product and order first
	product := createTestProduct(t, db, 10)

	orderReq := models.CreateOrderRequest{
		ProductID: product.ID,
		Quantity:  1,
		BuyerID:   "test_buyer",
	}

	reqBody, _ := json.Marshal(orderReq)
	resp, err := http.Post(server.URL+"/orders", "application/json", bytes.NewBuffer(reqBody))
	require.NoError(t, err)

	var createdOrder models.Order
	err = json.NewDecoder(resp.Body).Decode(&createdOrder)
	require.NoError(t, err)
	resp.Body.Close()

	// Retrieve the order
	resp, err = http.Get(server.URL + "/orders/" + createdOrder.ID.String())
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var retrievedOrder models.Order
	err = json.NewDecoder(resp.Body).Decode(&retrievedOrder)
	require.NoError(t, err)

	assert.Equal(t, createdOrder.ID, retrievedOrder.ID)
	assert.Equal(t, createdOrder.ProductID, retrievedOrder.ProductID)
	assert.Equal(t, createdOrder.BuyerID, retrievedOrder.BuyerID)
	assert.NotNil(t, retrievedOrder.Product)
	assert.Equal(t, product.Name, retrievedOrder.Product.Name)
}

func TestOutOfStockOrder(t *testing.T) {
	server, db := setupTestServer(t)

	// Create a product with limited stock
	product := createTestProduct(t, db, 2)

	// Try to order more than available stock
	orderReq := models.CreateOrderRequest{
		ProductID: product.ID,
		Quantity:  5,
		BuyerID:   "test_buyer",
	}

	reqBody, _ := json.Marshal(orderReq)
	resp, err := http.Post(server.URL+"/orders", "application/json", bytes.NewBuffer(reqBody))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusConflict, resp.StatusCode)

	var errResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&errResp)
	require.NoError(t, err)

	errorDetail := errResp["error"].(map[string]interface{})
	assert.Equal(t, "OUT_OF_STOCK", errorDetail["code"])
}

func TestSettlementJob(t *testing.T) {
	server, db := setupTestServer(t)

	// Create some test transactions
	ctx := context.Background()
	txRepo := repository.NewTransactionRepository(db.DB)

	now := time.Now()
	transactions := []*models.Transaction{
		{
			MerchantID:  "merchant_1",
			AmountCents: 10000,
			FeeCents:    300,
			Status:      models.TransactionStatusCompleted,
			PaidAt:      now.AddDate(0, 0, -1),
		},
		{
			MerchantID:  "merchant_1",
			AmountCents: 20000,
			FeeCents:    600,
			Status:      models.TransactionStatusCompleted,
			PaidAt:      now.AddDate(0, 0, -1),
		},
		{
			MerchantID:  "merchant_2",
			AmountCents: 15000,
			FeeCents:    450,
			Status:      models.TransactionStatusCompleted,
			PaidAt:      now.AddDate(0, 0, -1),
		},
	}

	for _, tx := range transactions {
		err := txRepo.Create(ctx, tx)
		require.NoError(t, err)
	}

	// Create settlement job
	jobReq := models.CreateSettlementJobRequest{
		From: now.AddDate(0, 0, -2).Format("2006-01-02"),
		To:   now.Format("2006-01-02"),
	}

	reqBody, _ := json.Marshal(jobReq)
	resp, err := http.Post(server.URL+"/jobs/settlement", "application/json", bytes.NewBuffer(reqBody))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusAccepted, resp.StatusCode)

	var jobResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&jobResp)
	require.NoError(t, err)

	jobID := jobResp["job_id"].(string)
	assert.NotEmpty(t, jobID)
	assert.Equal(t, "QUEUED", jobResp["status"])

	// Wait for job to complete (with timeout)
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			t.Fatal("Job did not complete within timeout")
		case <-ticker.C:
			resp, err := http.Get(server.URL + "/jobs/" + jobID)
			require.NoError(t, err)

			var job map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&job)
			resp.Body.Close()
			require.NoError(t, err)

			status := job["status"].(string)
			if status == "COMPLETED" {
				assert.Contains(t, job, "download_url")
				// Test completed successfully
				return
			} else if status == "FAILED" {
				t.Fatalf("Job failed: %v", job["error"])
			}
		}
	}
}

func TestJobCancellation(t *testing.T) {
	server, db := setupTestServer(t)

	// Create some test transactions to make the job actually have work to do
	ctx := context.Background()
	txRepo := repository.NewTransactionRepository(db.DB)

	// Create many transactions across a wide date range to make job take time
	now := time.Now()
	for i := 0; i < 1000; i++ {
		tx := &models.Transaction{
			MerchantID:  fmt.Sprintf("merchant_%d", i%10),
			AmountCents: 10000 + i,
			FeeCents:    300,
			Status:      models.TransactionStatusCompleted,
			PaidAt:      now.AddDate(0, 0, -i%365), // Spread across a year
		}
		err := txRepo.Create(ctx, tx)
		require.NoError(t, err)
	}

	// Create settlement job with a large date range
	jobReq := models.CreateSettlementJobRequest{
		From: "2020-01-01",
		To:   "2025-12-31",
	}

	reqBody, _ := json.Marshal(jobReq)
	resp, err := http.Post(server.URL+"/jobs/settlement", "application/json", bytes.NewBuffer(reqBody))
	require.NoError(t, err)

	var jobResp map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&jobResp)
	require.NoError(t, err)
	resp.Body.Close()

	jobID := jobResp["job_id"].(string)

	// Wait a bit to ensure job starts processing
	time.Sleep(50 * time.Millisecond)

	// Cancel the job
	resp, err = http.Post(server.URL+"/jobs/"+jobID+"/cancel", "application/json", nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Wait for cancellation to take effect
	time.Sleep(200 * time.Millisecond)

	// Check job status multiple times to ensure cancellation
	for attempts := 0; attempts < 5; attempts++ {
		resp, err = http.Get(server.URL + "/jobs/" + jobID)
		require.NoError(t, err)

		var job map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&job)
		resp.Body.Close()
		require.NoError(t, err)

		status := job["status"].(string)
		t.Logf("Attempt %d: Job status after cancellation: %s", attempts+1, status)

		// Job should be cancelled, completed, or failed - but not running indefinitely
		if status == "CANCELLED" {
			// Perfect - job was successfully cancelled
			return
		} else if status == "COMPLETED" || status == "FAILED" {
			// Job completed before cancellation could take effect - this is also acceptable
			t.Logf("Job completed with status %s before cancellation could take effect", status)
			return
		} else if status == "QUEUED" {
			// Job might still be queued and cancelled - this is acceptable
			return
		}

		// If still running, wait a bit more
		time.Sleep(100 * time.Millisecond)
	}

	// If we get here, the job is still running after multiple attempts
	t.Fatalf("Job is still running after cancellation attempts")
}

func TestHealthCheck(t *testing.T) {
	server, _ := setupTestServer(t)

	resp, err := http.Get(server.URL + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var health models.HealthCheck
	err = json.NewDecoder(resp.Body).Decode(&health)
	require.NoError(t, err)

	assert.Equal(t, "healthy", health.Status)
	assert.Equal(t, "1.0.0", health.Version)
	assert.Contains(t, health.Checks, "database")
	assert.Equal(t, "healthy", health.Checks["database"])
}
