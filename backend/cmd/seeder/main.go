// Package main provides a data seeder for generating test transactions
package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"indico-backend/internal/config"
	"indico-backend/internal/database"
	"indico-backend/internal/logger"
	"indico-backend/internal/models"
	"indico-backend/internal/repository"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		panic(fmt.Sprintf("Failed to load configuration: %v", err))
	}

	// Initialize logger
	logger.Init("info", "text")

	logger.Info("Starting data seeder")

	// Connect to database
	db, err := database.New(&cfg.Database)
	if err != nil {
		logger.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize repository
	txRepo := repository.NewTransactionRepository(db.DB)

	// Seed transactions
	if err := seedTransactions(txRepo); err != nil {
		logger.Fatalf("Failed to seed transactions: %v", err)
	}

	logger.Info("Data seeding completed successfully")
}

func seedTransactions(txRepo repository.TransactionRepository) error {
	logger.Info("Seeding transactions...")

	merchants := []string{
		"merchant_001", "merchant_002", "merchant_003", "merchant_004", "merchant_005",
		"merchant_006", "merchant_007", "merchant_008", "merchant_009", "merchant_010",
	}

	// Generate transactions for the last 60 days
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -60)

	const batchSize = 1000
	const totalTransactions = 1000000

	// Create a local RNG
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < totalTransactions; i += batchSize {
		var transactions []*models.Transaction

		remaining := totalTransactions - i
		currentBatchSize := batchSize
		if remaining < batchSize {
			currentBatchSize = remaining
		}

		for j := 0; j < currentBatchSize; j++ {
			// Random merchant
			merchantID := merchants[rng.Intn(len(merchants))]

			// Random date between start and end
			daysDiff := int(endDate.Sub(startDate).Hours() / 24)
			randomDays := rng.Intn(daysDiff)
			paidAt := startDate.AddDate(0, 0, randomDays).
				Add(time.Duration(rng.Intn(24)) * time.Hour).
				Add(time.Duration(rng.Intn(60)) * time.Minute)

			// Random amount (100 cents to 50000 cents, i.e., $1 to $500)
			amountCents := rng.Intn(49900) + 100

			// Fee is typically 2.9% + 30 cents
			feeCents := int(float64(amountCents)*0.029) + 30

			transaction := &models.Transaction{
				MerchantID:  merchantID,
				AmountCents: amountCents,
				FeeCents:    feeCents,
				Status:      models.TransactionStatusCompleted,
				PaidAt:      paidAt,
			}

			transactions = append(transactions, transaction)
		}

		// Bulk insert batch
		if err := txRepo.BulkCreate(context.Background(), transactions); err != nil {
			return fmt.Errorf("failed to create transaction batch: %w", err)
		}

		if (i+currentBatchSize)%10000 == 0 {
			logger.Infof("Seeded %d transactions", i+currentBatchSize)
		}
	}

	logger.Infof("Successfully seeded %d transactions", totalTransactions)
	return nil
}
