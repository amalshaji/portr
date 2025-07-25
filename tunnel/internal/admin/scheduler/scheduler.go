package scheduler

import (
	"context"
	"log"
	"time"

	"github.com/amalshaji/portr/internal/admin/models"
	"gorm.io/gorm"
)

// Scheduler handles background tasks for the admin server
type Scheduler struct {
	db     *gorm.DB
	cancel context.CancelFunc
	jobs   []Job
}

// Job represents a background task
type Job struct {
	Name     string
	Interval time.Duration
	Function func(*Scheduler) error
}

// New creates a new scheduler
func New(db *gorm.DB) *Scheduler {
	return &Scheduler{
		db: db,
		jobs: []Job{
			{
				Name:     "Clear expired sessions",
				Interval: 1 * time.Hour,
				Function: (*Scheduler).clearExpiredSessions,
			},
			{
				Name:     "Clear unclaimed connections",
				Interval: 10 * time.Second,
				Function: (*Scheduler).clearUnclaimedConnections,
			},
		},
	}
}

// Start starts all background jobs
func (s *Scheduler) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel

	log.Printf("Starting %d background jobs", len(s.jobs))

	for _, job := range s.jobs {
		go s.runJob(ctx, job)
	}
}

// Stop stops all background jobs
func (s *Scheduler) Stop() {
	if s.cancel != nil {
		log.Println("Stopping background jobs...")
		s.cancel()
	}
}

// runJob runs a single job in a loop
func (s *Scheduler) runJob(ctx context.Context, job Job) {
	ticker := time.NewTicker(job.Interval)
	defer ticker.Stop()

	// Run once immediately
	if err := job.Function(s); err != nil {
		log.Printf("Error running job %s: %v", job.Name, err)
	}

	for {
		select {
		case <-ticker.C:
			if err := job.Function(s); err != nil {
				log.Printf("Error running job %s: %v", job.Name, err)
			}
		case <-ctx.Done():
			log.Printf("Stopping job: %s", job.Name)
			return
		}
	}
}

// clearExpiredSessions removes expired sessions from the database
func (s *Scheduler) clearExpiredSessions() error {
	result := s.db.Where("expires_at < ?", time.Now()).Delete(&models.Session{})
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected > 0 {
		log.Printf("Cleared %d expired sessions", result.RowsAffected)
	}

	return nil
}

// clearUnclaimedConnections removes connections that have been reserved but never activated
func (s *Scheduler) clearUnclaimedConnections() error {
	// Delete connections that have been reserved for more than 5 minutes without being activated
	cutoff := time.Now().Add(-5 * time.Minute)

	result := s.db.Where("status = ? AND created_at < ?",
		models.ConnectionStatusReserved, cutoff).Delete(&models.Connection{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected > 0 {
		log.Printf("Cleared %d unclaimed connections", result.RowsAffected)
	}

	return nil
}
