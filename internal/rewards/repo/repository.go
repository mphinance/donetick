package repo

import (
	"context"
	"time"

	"donetick.com/core/config"
	pModel "donetick.com/core/internal/points"
	rModel "donetick.com/core/internal/rewards/model"
	"gorm.io/gorm"
)

type RewardsRepository struct {
	db *gorm.DB
}

func NewRewardsRepository(db *gorm.DB, cfg *config.Config) *RewardsRepository {
	return &RewardsRepository{db: db}
}

// Rewards CRUD
func (r *RewardsRepository) CreateReward(ctx context.Context, reward *rModel.Reward) error {
	return r.db.WithContext(ctx).Create(reward).Error
}

func (r *RewardsRepository) GetRewardsByCircle(ctx context.Context, circleID int) ([]*rModel.Reward, error) {
	var rewards []*rModel.Reward
	if err := r.db.WithContext(ctx).Where("circle_id = ? AND is_active = ?", circleID, true).
		Order("points_cost ASC").Find(&rewards).Error; err != nil {
		return nil, err
	}
	return rewards, nil
}

func (r *RewardsRepository) GetRewardByID(ctx context.Context, rewardID int) (*rModel.Reward, error) {
	var reward rModel.Reward
	if err := r.db.WithContext(ctx).First(&reward, rewardID).Error; err != nil {
		return nil, err
	}
	return &reward, nil
}

func (r *RewardsRepository) UpdateReward(ctx context.Context, reward *rModel.Reward) error {
	return r.db.WithContext(ctx).Save(reward).Error
}

func (r *RewardsRepository) DeleteReward(ctx context.Context, rewardID int) error {
	return r.db.WithContext(ctx).Model(&rModel.Reward{}).Where("id = ?", rewardID).
		Update("is_active", false).Error
}

// Reward Redemptions
func (r *RewardsRepository) CreateRedemption(ctx context.Context, redemption *rModel.RewardRedemption) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Create redemption record
		if err := tx.Create(redemption).Error; err != nil {
			return err
		}

		// Increment times_redeemed for the reward
		if err := tx.Model(&rModel.Reward{}).Where("id = ?", redemption.RewardID).
			Update("times_redeemed", gorm.Expr("times_redeemed + 1")).Error; err != nil {
			return err
		}

		// Create points history record
		pointsHistory := &pModel.PointsHistory{
			Action:    pModel.PointsHistoryActionRedeem,
			Points:    redemption.Points,
			CreatedAt: time.Now().UTC(),
			CreatedBy: redemption.UserID,
			UserID:    redemption.UserID,
			CircleID:  redemption.CircleID,
		}
		if err := tx.Create(pointsHistory).Error; err != nil {
			return err
		}

		return nil
	})
}

func (r *RewardsRepository) GetRedemptionsByUser(ctx context.Context, userID int, circleID int) ([]*rModel.RewardRedemption, error) {
	var redemptions []*rModel.RewardRedemption
	if err := r.db.WithContext(ctx).Preload("Reward").
		Where("user_id = ? AND circle_id = ?", userID, circleID).
		Order("created_at DESC").Find(&redemptions).Error; err != nil {
		return nil, err
	}
	return redemptions, nil
}

func (r *RewardsRepository) GetRedemptionsByCircle(ctx context.Context, circleID int) ([]*rModel.RewardRedemption, error) {
	var redemptions []*rModel.RewardRedemption
	if err := r.db.WithContext(ctx).Preload("Reward").
		Where("circle_id = ?", circleID).
		Order("created_at DESC").Find(&redemptions).Error; err != nil {
		return nil, err
	}
	return redemptions, nil
}

func (r *RewardsRepository) UpdateRedemptionStatus(ctx context.Context, redemptionID int, status rModel.RedemptionStatus, notes *string) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now().UTC(),
	}
	if notes != nil {
		updates["notes"] = *notes
	}
	return r.db.WithContext(ctx).Model(&rModel.RewardRedemption{}).
		Where("id = ?", redemptionID).Updates(updates).Error
}

// Goals CRUD
func (r *RewardsRepository) CreateGoal(ctx context.Context, goal *rModel.Goal) error {
	return r.db.WithContext(ctx).Create(goal).Error
}

func (r *RewardsRepository) GetGoalsByCircle(ctx context.Context, circleID int, userID *int) ([]*rModel.Goal, error) {
	var goals []*rModel.Goal
	query := r.db.WithContext(ctx).Where("circle_id = ? AND is_active = ?", circleID, true)
	
	if userID != nil {
		// Get both circle-wide goals and user-specific goals
		query = query.Where("user_id IS NULL OR user_id = ?", *userID)
	} else {
		// Get only circle-wide goals
		query = query.Where("user_id IS NULL")
	}
	
	if err := query.Order("end_date ASC, target_points ASC").Find(&goals).Error; err != nil {
		return nil, err
	}
	return goals, nil
}

func (r *RewardsRepository) GetGoalByID(ctx context.Context, goalID int) (*rModel.Goal, error) {
	var goal rModel.Goal
	if err := r.db.WithContext(ctx).First(&goal, goalID).Error; err != nil {
		return nil, err
	}
	return &goal, nil
}

func (r *RewardsRepository) UpdateGoal(ctx context.Context, goal *rModel.Goal) error {
	return r.db.WithContext(ctx).Save(goal).Error
}

func (r *RewardsRepository) DeleteGoal(ctx context.Context, goalID int) error {
	return r.db.WithContext(ctx).Model(&rModel.Goal{}).Where("id = ?", goalID).
		Update("is_active", false).Error
}

// Goal Progress
func (r *RewardsRepository) UpsertGoalProgress(ctx context.Context, progress *rModel.GoalProgress) error {
	return r.db.WithContext(ctx).Save(progress).Error
}

func (r *RewardsRepository) GetGoalProgress(ctx context.Context, goalID int, userID int) (*rModel.GoalProgress, error) {
	var progress rModel.GoalProgress
	if err := r.db.WithContext(ctx).Where("goal_id = ? AND user_id = ?", goalID, userID).
		First(&progress).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &progress, nil
}

func (r *RewardsRepository) GetGoalProgressByCircle(ctx context.Context, circleID int) ([]*rModel.GoalProgress, error) {
	var progress []*rModel.GoalProgress
	if err := r.db.WithContext(ctx).Preload("Goal").
		Joins("JOIN goals ON goal_progresses.goal_id = goals.id").
		Where("goals.circle_id = ?", circleID).
		Order("goal_progresses.progress DESC").Find(&progress).Error; err != nil {
		return nil, err
	}
	return progress, nil
}

// Leaderboard
func (r *RewardsRepository) GetLeaderboard(ctx context.Context, circleID int, limit int) ([]*rModel.PointsLeaderboard, error) {
	var leaderboard []*rModel.PointsLeaderboard
	
	query := `
		SELECT 
			u.id as user_id,
			u.username,
			u.display_name,
			u.image,
			uc.points,
			ROW_NUMBER() OVER (ORDER BY uc.points DESC) as rank,
			COALESCE(week_points.points, 0) as points_this_week,
			COALESCE(month_points.points, 0) as points_this_month
		FROM users u
		JOIN user_circles uc ON u.id = uc.user_id
		LEFT JOIN (
			SELECT 
				ph.user_id,
				SUM(ph.points) as points
			FROM points_histories ph
			WHERE ph.action = 0 
				AND ph.created_at >= DATE_SUB(NOW(), INTERVAL 7 DAY)
				AND ph.circle_id = ?
			GROUP BY ph.user_id
		) week_points ON u.id = week_points.user_id
		LEFT JOIN (
			SELECT 
				ph.user_id,
				SUM(ph.points) as points
			FROM points_histories ph
			WHERE ph.action = 0 
				AND ph.created_at >= DATE_SUB(NOW(), INTERVAL 30 DAY)
				AND ph.circle_id = ?
			GROUP BY ph.user_id
		) month_points ON u.id = month_points.user_id
		WHERE uc.circle_id = ? AND uc.is_active = true
		ORDER BY uc.points DESC
		LIMIT ?
	`
	
	if err := r.db.WithContext(ctx).Raw(query, circleID, circleID, circleID, limit).
		Scan(&leaderboard).Error; err != nil {
		return nil, err
	}
	
	return leaderboard, nil
}

// Points Statistics
func (r *RewardsRepository) GetUserPointsStats(ctx context.Context, userID int, circleID int) (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	
	// Get current points
	var currentPoints int
	if err := r.db.WithContext(ctx).Model(&struct {
		Points int `gorm:"column:points"`
	}{}).Table("user_circles").
		Where("user_id = ? AND circle_id = ?", userID, circleID).
		Select("points").Scan(&currentPoints).Error; err != nil {
		return nil, err
	}
	stats["currentPoints"] = currentPoints
	
	// Get points this week
	var weekPoints int
	if err := r.db.WithContext(ctx).Model(&pModel.PointsHistory{}).
		Where("user_id = ? AND circle_id = ? AND action = ? AND created_at >= ?", 
			userID, circleID, pModel.PointsHistoryActionAdd, time.Now().AddDate(0, 0, -7)).
		Select("COALESCE(SUM(points), 0)").Scan(&weekPoints).Error; err != nil {
		return nil, err
	}
	stats["pointsThisWeek"] = weekPoints
	
	// Get points this month
	var monthPoints int
	if err := r.db.WithContext(ctx).Model(&pModel.PointsHistory{}).
		Where("user_id = ? AND circle_id = ? AND action = ? AND created_at >= ?", 
			userID, circleID, pModel.PointsHistoryActionAdd, time.Now().AddDate(0, -1, 0)).
		Select("COALESCE(SUM(points), 0)").Scan(&monthPoints).Error; err != nil {
		return nil, err
	}
	stats["pointsThisMonth"] = monthPoints
	
	// Get total points redeemed
	var redeemedPoints int
	if err := r.db.WithContext(ctx).Model(&pModel.PointsHistory{}).
		Where("user_id = ? AND circle_id = ? AND action = ?", 
			userID, circleID, pModel.PointsHistoryActionRedeem).
		Select("COALESCE(SUM(points), 0)").Scan(&redeemedPoints).Error; err != nil {
		return nil, err
	}
	stats["pointsRedeemed"] = redeemedPoints
	
	return stats, nil
}

func (r *RewardsRepository) UpdateGoalProgress(ctx context.Context, circleID int) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Get all active goals for the circle
		var goals []*rModel.Goal
		if err := tx.Where("circle_id = ? AND is_active = ?", circleID, true).Find(&goals).Error; err != nil {
			return err
		}
		
		for _, goal := range goals {
			if goal.UserID != nil {
				// User-specific goal
				if err := r.updateUserGoalProgress(ctx, tx, goal, *goal.UserID); err != nil {
					return err
				}
			} else {
				// Circle-wide goal - update for all users in circle
				var userIDs []int
				if err := tx.Model(&struct {
					UserID int `gorm:"column:user_id"`
				}{}).Table("user_circles").
					Where("circle_id = ? AND is_active = ?", circleID, true).
					Pluck("user_id", &userIDs).Error; err != nil {
					return err
				}
				
				for _, userID := range userIDs {
					if err := r.updateUserGoalProgress(ctx, tx, goal, userID); err != nil {
						return err
					}
				}
			}
		}
		
		return nil
	})
}

func (r *RewardsRepository) updateUserGoalProgress(ctx context.Context, tx *gorm.DB, goal *rModel.Goal, userID int) error {
	// Calculate current points for the goal period
	var currentPoints int
	query := tx.Model(&pModel.PointsHistory{}).
		Where("user_id = ? AND circle_id = ? AND action = ?", userID, goal.CircleID, pModel.PointsHistoryActionAdd)
	
	if goal.StartDate != nil {
		query = query.Where("created_at >= ?", *goal.StartDate)
	}
	if goal.EndDate != nil {
		query = query.Where("created_at <= ?", *goal.EndDate)
	}
	
	if err := query.Select("COALESCE(SUM(points), 0)").Scan(&currentPoints).Error; err != nil {
		return err
	}
	
	// Calculate progress percentage
	progress := float64(currentPoints) / float64(goal.TargetPoints) * 100
	if progress > 100 {
		progress = 100
	}
	
	// Check if goal is completed
	var completedAt *time.Time
	if currentPoints >= goal.TargetPoints && goal.CompletedAt == nil {
		now := time.Now().UTC()
		completedAt = &now
		
		// Award bonus points if specified
		if goal.RewardPoints != nil && *goal.RewardPoints > 0 {
			// Add bonus points to user
			if err := tx.Model(&struct {
				Points int `gorm:"column:points"`
			}{}).Table("user_circles").
				Where("user_id = ? AND circle_id = ?", userID, goal.CircleID).
				Update("points", gorm.Expr("points + ?", *goal.RewardPoints)).Error; err != nil {
				return err
			}
			
			// Record bonus points in history
			bonusHistory := &pModel.PointsHistory{
				Action:    pModel.PointsHistoryActionAdd,
				Points:    *goal.RewardPoints,
				CreatedAt: time.Now().UTC(),
				CreatedBy: userID,
				UserID:    userID,
				CircleID:  goal.CircleID,
			}
			if err := tx.Create(bonusHistory).Error; err != nil {
				return err
			}
		}
		
		// Mark goal as completed
		if err := tx.Model(&rModel.Goal{}).Where("id = ?", goal.ID).
			Update("completed_at", completedAt).Error; err != nil {
			return err
		}
	}
	
	// Upsert goal progress
	goalProgress := &rModel.GoalProgress{
		GoalID:        goal.ID,
		UserID:        userID,
		CurrentPoints: currentPoints,
		Progress:      progress,
		CompletedAt:   completedAt,
		UpdatedAt:     time.Now().UTC(),
	}
	
	return tx.Save(goalProgress).Error
}

func (r *RewardsRepository) GetUserGoalProgress(ctx context.Context, userID int, circleID int) ([]*rModel.GoalProgress, error) {
	var progress []*rModel.GoalProgress
	if err := r.db.WithContext(ctx).Preload("Goal").
		Joins("JOIN goals ON goal_progresses.goal_id = goals.id").
		Where("goal_progresses.user_id = ? AND goals.circle_id = ? AND goals.is_active = ?", 
			userID, circleID, true).
		Order("goals.end_date ASC, goal_progresses.progress DESC").Find(&progress).Error; err != nil {
		return nil, err
	}
	return progress, nil
}