package rewards

import (
	"context"
	"time"

	"donetick.com/core/config"
	rRepo "donetick.com/core/internal/rewards/repo"
	"donetick.com/core/logging"
)

type Scheduler struct {
	rewardsRepo *rRepo.RewardsRepository
	stopChan    chan bool
}

func NewScheduler(cfg *config.Config, rr *rRepo.RewardsRepository) *Scheduler {
	return &Scheduler{
		rewardsRepo: rr,
		stopChan:    make(chan bool),
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	log := logging.FromContext(ctx)
	log.Info("Starting rewards scheduler")
	
	// Update goal progress every hour
	go s.runScheduler(ctx, "GOAL_PROGRESS_UPDATE", s.updateAllGoalProgress, 1*time.Hour)
}

func (s *Scheduler) Stop() {
	s.stopChan <- true
}

func (s *Scheduler) runScheduler(ctx context.Context, jobName string, job func(context.Context) error, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	log := logging.FromContext(ctx)
	
	for {
		select {
		case <-s.stopChan:
			log.Infow("Rewards scheduler stopped", "job", jobName)
			return
		case <-ticker.C:
			log.Debugw("Running rewards scheduler job", "job", jobName)
			
			if err := job(ctx); err != nil {
				log.Errorw("Rewards scheduler job failed", "job", jobName, "error", err)
			}
		}
	}
}

func (s *Scheduler) updateAllGoalProgress(ctx context.Context) error {
	log := logging.FromContext(ctx)
	
	// Get all active circles
	var circleIDs []int
	// This would need access to circle repository, but for now we'll use a simple query
	// In a real implementation, you'd inject the circle repository
	
	log.Debug("Updating goal progress for all circles")
	
	// For now, we'll update progress for circles 1-100 (adjust as needed)
	for circleID := 1; circleID <= 100; circleID++ {
		if err := s.rewardsRepo.UpdateGoalProgress(ctx, circleID); err != nil {
			log.Debugw("Failed to update goal progress for circle", "circleID", circleID, "error", err)
			// Continue with other circles
		}
	}
	
	return nil
}