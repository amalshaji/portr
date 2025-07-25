package scheduler

import (
	"context"
	"time"

	"github.com/amalshaji/portr/internal/admin/models"
	"github.com/charmbracelet/log"
	"gorm.io/gorm"
)

type Scheduler struct {
	db     *gorm.DB
	cancel context.CancelFunc
	jobs   []Job
}

type Job struct {
	Name     string
	Interval time.Duration
	Function func(*Scheduler) error
}

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

func (s *Scheduler) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel

	log.Info("Starting background jobs", "job_count", len(s.jobs))

	for _, job := range s.jobs {
		go s.runJob(ctx, job)
	}
}

func (s *Scheduler) Stop() {
	if s.cancel != nil {
		log.Info("Stopping background jobs")
		s.cancel()
	}
}

func (s *Scheduler) runJob(ctx context.Context, job Job) {
	ticker := time.NewTicker(job.Interval)
	defer ticker.Stop()

	if err := job.Function(s); err != nil {
		log.Error("Job execution failed", "job", job.Name, "error", err)
	}

	for {
		select {
		case <-ticker.C:
			if err := job.Function(s); err != nil {
				log.Error("Job execution failed", "job", job.Name, "error", err)
			}
		case <-ctx.Done():
			log.Info("Stopping job", "job", job.Name)
			return
		}
	}
}

func (s *Scheduler) clearExpiredSessions() error {
	result := s.db.Where("expires_at < ?", time.Now()).Delete(&models.Session{})
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected > 0 {
		log.Info("Cleared expired sessions", "count", result.RowsAffected)
	}

	return nil
}

func (s *Scheduler) clearUnclaimedConnections() error {
	// Delete connections that have been reserved for more than 5 minutes without being activated
	cutoff := time.Now().Add(-5 * time.Minute)

	result := s.db.Where("status = ? AND created_at < ?",
		models.ConnectionStatusReserved, cutoff).Delete(&models.Connection{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected > 0 {
		log.Info("Cleared unclaimed connections", "count", result.RowsAffected)
	}

	return nil
}
