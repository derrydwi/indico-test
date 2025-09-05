// Package service provides the job processing functionality
package service

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"time"

	"indico-backend/internal/config"
	"indico-backend/internal/database"
	"indico-backend/internal/logger"
	"indico-backend/internal/metrics"
	"indico-backend/internal/models"
	"indico-backend/internal/repository"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

// JobProcessor handles background job processing
type JobProcessor struct {
	db         *database.DB
	config     *config.JobsConfig
	txRepo     repository.TransactionRepository
	settleRepo repository.SettlementRepository
	jobRepo    repository.JobRepository

	jobQueue  chan *models.Job
	cancelMap sync.Map // map[uuid.UUID]context.CancelFunc
	workers   int
	batchSize int

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewJobProcessor creates a new job processor
func NewJobProcessor(
	db *database.DB,
	cfg *config.JobsConfig,
	txRepo repository.TransactionRepository,
	settleRepo repository.SettlementRepository,
	jobRepo repository.JobRepository,
) *JobProcessor {
	ctx, cancel := context.WithCancel(context.Background())

	return &JobProcessor{
		db:         db,
		config:     cfg,
		txRepo:     txRepo,
		settleRepo: settleRepo,
		jobRepo:    jobRepo,
		jobQueue:   make(chan *models.Job, cfg.QueueSize),
		workers:    cfg.Workers,
		batchSize:  cfg.BatchSize,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Start starts the job processor workers
func (jp *JobProcessor) Start() {
	logger.WithComponent("job_processor").
		WithField("workers", jp.workers).
		WithField("batch_size", jp.batchSize).
		Info("Starting job processor")

	for i := 0; i < jp.workers; i++ {
		jp.wg.Add(1)
		go jp.worker(i)
	}
}

// Stop stops the job processor
func (jp *JobProcessor) Stop() {
	logger.WithComponent("job_processor").Info("Stopping job processor")

	jp.cancel()
	close(jp.jobQueue)
	jp.wg.Wait()

	logger.WithComponent("job_processor").Info("Job processor stopped")
}

// QueueJob queues a job for processing
func (jp *JobProcessor) QueueJob(ctx context.Context, job *models.Job) error {
	select {
	case jp.jobQueue <- job:
		logger.WithJobID(job.ID.String()).Info("Job queued for processing")
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return fmt.Errorf("job queue is full")
	}
}

// CancelJob cancels a running job by cancelling its context
func (jp *JobProcessor) CancelJob(jobID uuid.UUID) {
	if cancelFunc, ok := jp.cancelMap.Load(jobID); ok {
		if cancel, ok := cancelFunc.(context.CancelFunc); ok {
			cancel()
			logger.WithJobID(jobID.String()).Info("Job context cancelled")
		}
	}
}

// worker processes jobs from the queue
func (jp *JobProcessor) worker(workerID int) {
	defer jp.wg.Done()

	log := logger.WithComponent("job_processor").WithField("worker_id", workerID)
	log.Info("Worker started")

	for {
		select {
		case job, ok := <-jp.jobQueue:
			if !ok {
				log.Info("Worker stopped - job queue closed")
				return
			}

			jp.processJob(job, workerID)

		case <-jp.ctx.Done():
			log.Info("Worker stopped - context cancelled")
			return
		}
	}
}

// processJob processes a single job
func (jp *JobProcessor) processJob(job *models.Job, workerID int) {
	start := time.Now()
	log := logger.WithJobID(job.ID.String()).WithField("worker_id", workerID)
	log.Info("Processing job")

	// Create cancellable context for this job
	jobCtx, jobCancel := context.WithCancel(jp.ctx)
	jp.cancelMap.Store(job.ID, jobCancel)
	defer func() {
		jp.cancelMap.Delete(job.ID)
		jobCancel()
	}()

	// Mark job as started
	if err := jp.jobRepo.MarkStarted(jobCtx, job.ID); err != nil {
		log.WithError(err).Error("Failed to mark job as started")
		return
	}

	// Process based on job type
	var err error
	status := "success"
	switch job.Type {
	case models.JobTypeSettlement:
		err = jp.processSettlementJob(jobCtx, job)
	default:
		err = fmt.Errorf("unknown job type: %s", job.Type)
	}

	// Update job status based on result
	if err != nil {
		status = "failed"
		log.WithError(err).Error("Job processing failed")

		if err := jp.jobRepo.UpdateStatus(jobCtx, job.ID, models.JobStatusFailed); err != nil {
			log.WithError(err).Error("Failed to update job status to failed")
		}

		if err := jp.jobRepo.UpdateError(jobCtx, job.ID, err.Error()); err != nil {
			log.WithError(err).Error("Failed to update job error")
		}
	} else {
		log.Info("Job processing completed successfully")

		if err := jp.jobRepo.MarkCompleted(jobCtx, job.ID); err != nil {
			log.WithError(err).Error("Failed to mark job as completed")
		}
	}

	// Record metrics
	metrics.JobsCompleted.WithLabelValues(string(job.Type), status).Inc()
	metrics.JobDuration.WithLabelValues(string(job.Type)).Observe(time.Since(start).Seconds())
}

// processSettlementJob processes a settlement job
func (jp *JobProcessor) processSettlementJob(ctx context.Context, job *models.Job) error {
	log := logger.WithJobID(job.ID.String())

	// Parse job parameters
	var params models.SettlementJobParams
	if err := json.Unmarshal([]byte(job.Parameters), &params); err != nil {
		return fmt.Errorf("failed to parse job parameters: %w", err)
	}

	// Parse dates
	from, err := time.Parse("2006-01-02", params.From)
	if err != nil {
		return fmt.Errorf("invalid from date: %w", err)
	}

	to, err := time.Parse("2006-01-02", params.To)
	if err != nil {
		return fmt.Errorf("invalid to date: %w", err)
	}

	// Add one day to 'to' date to make it inclusive
	to = to.AddDate(0, 0, 1)

	log.WithField("from", from).WithField("to", to).Info("Processing settlement job")

	// Get total transaction count for progress tracking
	totalCount, err := jp.txRepo.GetTotalCount(ctx, from, to)
	if err != nil {
		return fmt.Errorf("failed to get total transaction count: %w", err)
	}

	log.WithField("total_transactions", totalCount).Info("Total transactions to process")

	// Update job total
	if err := jp.jobRepo.UpdateProgress(ctx, job.ID, 0, 0); err != nil {
		log.WithError(err).Error("Failed to update job total")
	}

	// Process transactions in batches
	settlements := make(map[string]*models.Settlement) // key: merchantID_date
	var processed int
	var offset int

	for {
		// Check for cancellation
		select {
		case <-ctx.Done():
			log.Info("Job processing cancelled")
			return ctx.Err()
		default:
		}

		// Check if job was cancelled via API
		cancelled, err := jp.jobRepo.IsCancelled(ctx, job.ID)
		if err != nil {
			log.WithError(err).Error("Failed to check job cancellation status")
		} else if cancelled {
			log.Info("Job was cancelled via API")
			return fmt.Errorf("job was cancelled")
		}

		// Get batch of transactions
		transactions, err := jp.txRepo.GetBatch(ctx, offset, jp.batchSize, from, to)
		if err != nil {
			return fmt.Errorf("failed to get transaction batch: %w", err)
		}

		if len(transactions) == 0 {
			break // No more transactions
		}

		log.WithField("batch_size", len(transactions)).
			WithField("offset", offset).
			Debug("Processing transaction batch")

		// Process batch using worker pool
		if err := jp.processBatch(ctx, transactions, settlements); err != nil {
			return fmt.Errorf("failed to process batch: %w", err)
		}

		processed += len(transactions)
		offset += jp.batchSize

		// Update progress
		progress := float64(processed) / float64(totalCount) * 100
		if err := jp.jobRepo.UpdateProgress(ctx, job.ID, progress, processed); err != nil {
			log.WithError(err).Error("Failed to update job progress")
		}

		log.WithField("processed", processed).
			WithField("progress", fmt.Sprintf("%.2f%%", progress)).
			Debug("Progress updated")
	}

	// Save settlements to database
	if err := jp.saveSettlements(ctx, settlements, job.ID); err != nil {
		return fmt.Errorf("failed to save settlements: %w", err)
	}

	// Ensure /tmp/settlements directory exists as per assignment requirements
	dirPath := "/tmp/settlements"
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return fmt.Errorf("failed to create settlements directory: %w", err)
	}

	// Create CSV file
	csvPath := filepath.Join(dirPath, fmt.Sprintf("%s.csv", job.ID.String()))
	if err := jp.createSettlementCSV(settlements, csvPath); err != nil {
		return fmt.Errorf("failed to create CSV: %w", err)
	}

	// Update job with result path and download URL
	downloadURL := fmt.Sprintf("/downloads/%s.csv", job.ID.String())
	if err := jp.jobRepo.UpdateResult(ctx, job.ID, csvPath, downloadURL); err != nil {
		return fmt.Errorf("failed to update job result: %w", err)
	}

	log.WithField("settlements_count", len(settlements)).
		WithField("csv_path", csvPath).
		Info("Settlement job completed")

	return nil
}

// processBatch processes a batch of transactions using worker pool
func (jp *JobProcessor) processBatch(ctx context.Context, transactions []*models.Transaction, settlements map[string]*models.Settlement) error {
	// Use errgroup for concurrent processing with limited concurrency
	g, _ := errgroup.WithContext(ctx)
	g.SetLimit(4) // Limit concurrent goroutines

	// Mutex to protect settlements map
	var mu sync.Mutex

	for _, tx := range transactions {
		tx := tx // capture loop variable

		g.Go(func() error {
			// Aggregate transaction
			date := tx.PaidAt.Truncate(24 * time.Hour) // Start of day
			key := fmt.Sprintf("%s_%s", tx.MerchantID, date.Format("2006-01-02"))

			mu.Lock()
			settlement, exists := settlements[key]
			if !exists {
				settlement = &models.Settlement{
					MerchantID:  tx.MerchantID,
					Date:        date,
					GrossCents:  0,
					FeeCents:    0,
					NetCents:    0,
					TxnCount:    0,
					GeneratedAt: time.Now(),
					UniqueRunID: uuid.New(),
				}
				settlements[key] = settlement
			}

			settlement.GrossCents += tx.AmountCents
			settlement.FeeCents += tx.FeeCents
			settlement.NetCents += (tx.AmountCents - tx.FeeCents)
			settlement.TxnCount++
			mu.Unlock()

			return nil
		})
	}

	return g.Wait()
}

// saveSettlements saves settlements to database
func (jp *JobProcessor) saveSettlements(ctx context.Context, settlements map[string]*models.Settlement, _ uuid.UUID) error {
	// Save settlements in transaction
	return jp.db.WithTx(ctx, func(tx *sql.Tx) error {
		for _, settlement := range settlements {
			if err := jp.settleRepo.Upsert(ctx, tx, settlement); err != nil {
				return fmt.Errorf("failed to upsert settlement: %w", err)
			}
		}
		return nil
	})
}

// createSettlementCSV creates a CSV file from settlements
func (jp *JobProcessor) createSettlementCSV(settlements map[string]*models.Settlement, filePath string) error {
	// Convert map to slice for consistent ordering
	settlementSlice := make([]*models.Settlement, 0, len(settlements))
	for _, settlement := range settlements {
		settlementSlice = append(settlementSlice, settlement)
	}

	// Sort by merchant ID and date for consistent output
	sort.Slice(settlementSlice, func(i, j int) bool {
		if settlementSlice[i].MerchantID != settlementSlice[j].MerchantID {
			return settlementSlice[i].MerchantID < settlementSlice[j].MerchantID
		}
		return settlementSlice[i].Date.Before(settlementSlice[j].Date)
	})

	// Create the CSV file
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write CSV header
	header := []string{
		"merchant_id",
		"date",
		"gross_cents",
		"fee_cents",
		"net_cents",
		"transaction_count",
		"generated_at",
		"unique_run_id",
	}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write settlement data
	for _, settlement := range settlementSlice {
		record := []string{
			settlement.MerchantID,
			settlement.Date.Format("2006-01-02"),
			strconv.Itoa(settlement.GrossCents),
			strconv.Itoa(settlement.FeeCents),
			strconv.Itoa(settlement.NetCents),
			strconv.Itoa(settlement.TxnCount),
			settlement.GeneratedAt.Format(time.RFC3339),
			settlement.UniqueRunID.String(),
		}
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("failed to write CSV record: %w", err)
		}
	}

	return nil
}
