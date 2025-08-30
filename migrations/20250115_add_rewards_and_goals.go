package migrations

import (
	"context"

	rModel "donetick.com/core/internal/rewards/model"
	"donetick.com/core/logging"
	"gorm.io/gorm"
)

type AddRewardsAndGoals20250115 struct{}

func (m AddRewardsAndGoals20250115) ID() string {
	return "20250115_add_rewards_and_goals"
}

func (m AddRewardsAndGoals20250115) Description() string {
	return "Add rewards and goals tables for points tracking system"
}

func (m AddRewardsAndGoals20250115) Down(ctx context.Context, db *gorm.DB) error {
	log := logging.FromContext(ctx)
	
	// Drop tables in reverse order due to foreign key constraints
	tables := []string{"goal_progresses", "reward_redemptions", "goals", "rewards"}
	
	for _, table := range tables {
		if err := db.Migrator().DropTable(table); err != nil {
			log.Warnw("Failed to drop table during rollback", "table", table, "error", err)
		}
	}
	
	return nil
}

func (m AddRewardsAndGoals20250115) Up(ctx context.Context, db *gorm.DB) error {
	log := logging.FromContext(ctx)
	
	// Create tables using AutoMigrate to ensure proper schema
	if err := db.AutoMigrate(
		&rModel.Reward{},
		&rModel.RewardRedemption{},
		&rModel.Goal{},
		&rModel.GoalProgress{},
	); err != nil {
		log.Errorw("Failed to create rewards and goals tables", "error", err)
		return err
	}
	
	// Add some default rewards for existing circles
	return db.Transaction(func(tx *gorm.DB) error {
		// Get all existing circles
		var circleIDs []int
		if err := tx.Table("circles").Pluck("id", &circleIDs).Error; err != nil {
			return err
		}
		
		// Add default rewards for each circle
		for _, circleID := range circleIDs {
			// Get circle admin to set as creator
			var adminID int
			if err := tx.Table("user_circles").
				Where("circle_id = ? AND role = 'admin'", circleID).
				Select("user_id").Limit(1).Scan(&adminID).Error; err != nil {
				log.Warnw("No admin found for circle, skipping default rewards", "circleID", circleID)
				continue
			}
			
			defaultRewards := []*rModel.Reward{
				{
					Name:        "Coffee Break",
					Description: stringPtr("Enjoy a nice coffee break!"),
					PointsCost:  50,
					CircleID:    circleID,
					CreatedBy:   adminID,
					Category:    "food",
					Icon:        stringPtr("☕"),
					Color:       stringPtr("#8B4513"),
					CreatedAt:   time.Now().UTC(),
					UpdatedAt:   time.Now().UTC(),
				},
				{
					Name:        "Movie Night",
					Description: stringPtr("Pick the next movie for family movie night"),
					PointsCost:  100,
					CircleID:    circleID,
					CreatedBy:   adminID,
					Category:    "entertainment",
					Icon:        stringPtr("🎬"),
					Color:       stringPtr("#E11D48"),
					CreatedAt:   time.Now().UTC(),
					UpdatedAt:   time.Now().UTC(),
				},
				{
					Name:        "Skip a Chore",
					Description: stringPtr("Skip your next assigned chore"),
					PointsCost:  75,
					CircleID:    circleID,
					CreatedBy:   adminID,
					Category:    "chores",
					Icon:        stringPtr("⏭️"),
					Color:       stringPtr("#F59E0B"),
					MaxRedeems:  intPtr(2), // Limit to 2 per reward cycle
					CreatedAt:   time.Now().UTC(),
					UpdatedAt:   time.Now().UTC(),
				},
			}
			
			if err := tx.Create(&defaultRewards).Error; err != nil {
				log.Warnw("Failed to create default rewards for circle", "circleID", circleID, "error", err)
				// Continue with other circles
			}
			
			// Add a default goal
			defaultGoal := &rModel.Goal{
				Name:         "Weekly Champion",
				Description:  stringPtr("Earn 200 points this week"),
				TargetPoints: 200,
				CircleID:     circleID,
				CreatedBy:    adminID,
				Category:     "weekly",
				Icon:         stringPtr("🏆"),
				Color:        stringPtr("#10B981"),
				StartDate:    timePtr(getStartOfWeek()),
				EndDate:      timePtr(getEndOfWeek()),
				RewardPoints: intPtr(25), // Bonus points for completing
				CreatedAt:    time.Now().UTC(),
				UpdatedAt:    time.Now().UTC(),
			}
			
			if err := tx.Create(defaultGoal).Error; err != nil {
				log.Warnw("Failed to create default goal for circle", "circleID", circleID, "error", err)
			}
		}
		
		return nil
	})
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func getStartOfWeek() time.Time {
	now := time.Now().UTC()
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday = 7
	}
	return now.AddDate(0, 0, -weekday+1).Truncate(24 * time.Hour)
}

func getEndOfWeek() time.Time {
	return getStartOfWeek().AddDate(0, 0, 6).Add(23*time.Hour + 59*time.Minute + 59*time.Second)
}

// Register this migration
func init() {
	Register(AddRewardsAndGoals20250115{})
}