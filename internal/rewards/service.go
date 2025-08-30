package rewards

import (
	"context"
	"time"

	cModel "donetick.com/core/internal/circle/model"
	pModel "donetick.com/core/internal/points"
	rModel "donetick.com/core/internal/rewards/model"
	rRepo "donetick.com/core/internal/rewards/repo"
	"donetick.com/core/logging"
)

type Service struct {
	rewardsRepo *rRepo.RewardsRepository
}

func NewService(rr *rRepo.RewardsRepository) *Service {
	return &Service{
		rewardsRepo: rr,
	}
}

// UpdateGoalProgressForUser updates goal progress when a user earns points
func (s *Service) UpdateGoalProgressForUser(ctx context.Context, userID int, circleID int, pointsEarned int) error {
	log := logging.FromContext(ctx)
	
	// Get all active goals for the user and circle
	goals, err := s.rewardsRepo.GetGoalsByCircle(ctx, circleID, &userID)
	if err != nil {
		log.Errorw("Failed to get goals for progress update", "error", err)
		return err
	}

	for _, goal := range goals {
		// Skip if goal is user-specific and doesn't match current user
		if goal.UserID != nil && *goal.UserID != userID {
			continue
		}

		// Skip if goal is already completed
		if goal.CompletedAt != nil {
			continue
		}

		// Skip if goal is outside date range
		now := time.Now().UTC()
		if goal.StartDate != nil && now.Before(*goal.StartDate) {
			continue
		}
		if goal.EndDate != nil && now.After(*goal.EndDate) {
			continue
		}

		// Get current progress
		progress, err := s.rewardsRepo.GetGoalProgress(ctx, goal.ID, userID)
		if err != nil {
			log.Errorw("Failed to get goal progress", "goalID", goal.ID, "userID", userID, "error", err)
			continue
		}

		// Calculate new progress
		newCurrentPoints := pointsEarned
		if progress != nil {
			newCurrentPoints += progress.CurrentPoints
		}

		newProgressPercent := float64(newCurrentPoints) / float64(goal.TargetPoints) * 100
		if newProgressPercent > 100 {
			newProgressPercent = 100
		}

		// Check if goal is now completed
		var completedAt *time.Time
		if newCurrentPoints >= goal.TargetPoints {
			completedNow := time.Now().UTC()
			completedAt = &completedNow
		}

		// Update or create progress record
		goalProgress := &rModel.GoalProgress{
			GoalID:        goal.ID,
			UserID:        userID,
			CurrentPoints: newCurrentPoints,
			Progress:      newProgressPercent,
			CompletedAt:   completedAt,
			UpdatedAt:     time.Now().UTC(),
		}

		if err := s.rewardsRepo.UpsertGoalProgress(ctx, goalProgress); err != nil {
			log.Errorw("Failed to update goal progress", "goalID", goal.ID, "userID", userID, "error", err)
			continue
		}

		// If goal completed and has reward points, award them
		if completedAt != nil && goal.RewardPoints != nil && *goal.RewardPoints > 0 {
			log.Infow("Goal completed, awarding bonus points", 
				"goalID", goal.ID, "userID", userID, "bonusPoints", *goal.RewardPoints)
			
			// This would need to be handled by the calling service to avoid circular dependencies
			// For now, we'll log it and let the caller handle the bonus points
		}
	}

	return nil
}

// GetAvailableRewards returns rewards that a user can afford
func (s *Service) GetAvailableRewards(ctx context.Context, circleID int, userPoints int) ([]*rModel.Reward, error) {
	rewards, err := s.rewardsRepo.GetRewardsByCircle(ctx, circleID)
	if err != nil {
		return nil, err
	}

	var available []*rModel.Reward
	for _, reward := range rewards {
		// Check if user can afford it
		if reward.PointsCost <= userPoints {
			// Check if reward is still available (max redeems)
			if reward.MaxRedeems == nil || reward.TimesRedeemed < *reward.MaxRedeems {
				available = append(available, reward)
			}
		}
	}

	return available, nil
}

// CalculateUserRank calculates a user's rank in the circle
func (s *Service) CalculateUserRank(ctx context.Context, userID int, circleID int) (int, error) {
	leaderboard, err := s.rewardsRepo.GetLeaderboard(ctx, circleID, 100) // Get top 100
	if err != nil {
		return 0, err
	}

	for _, entry := range leaderboard {
		if entry.UserID == userID {
			return entry.Rank, nil
		}
	}

	return len(leaderboard) + 1, nil // User not in top 100
}