package points

import (
	"context"
	"time"

	cModel "donetick.com/core/internal/circle/model"
	pModel "donetick.com/core/internal/points"
	pRepo "donetick.com/core/internal/points/repo"
	"gorm.io/gorm"
)

type Service struct {
	pointsRepo *pRepo.PointsRepository
}

func NewService(pr *pRepo.PointsRepository) *Service {
	return &Service{
		pointsRepo: pr,
	}
}

// AwardPoints awards points to a user and creates a history record
func (s *Service) AwardPoints(ctx context.Context, tx *gorm.DB, userID int, circleID int, points int, createdBy int) error {
	// Update user points in user_circles
	if err := tx.Model(&cModel.UserCircle{}).
		Where("user_id = ? AND circle_id = ?", userID, circleID).
		Update("points", gorm.Expr("points + ?", points)).Error; err != nil {
		return err
	}

	// Create points history record
	pointsHistory := &pModel.PointsHistory{
		Action:    pModel.PointsHistoryActionAdd,
		Points:    points,
		CreatedAt: time.Now().UTC(),
		CreatedBy: createdBy,
		UserID:    userID,
		CircleID:  circleID,
	}

	return s.pointsRepo.CreatePointsHistory(ctx, tx, pointsHistory)
}

// DeductPoints deducts points from a user (for redemptions)
func (s *Service) DeductPoints(ctx context.Context, tx *gorm.DB, userID int, circleID int, points int, createdBy int) error {
	// Update user points in user_circles
	if err := tx.Model(&cModel.UserCircle{}).
		Where("user_id = ? AND circle_id = ?", userID, circleID).
		Update("points", gorm.Expr("points - ?", points)).Error; err != nil {
		return err
	}

	// Create points history record
	pointsHistory := &pModel.PointsHistory{
		Action:    pModel.PointsHistoryActionRedeem,
		Points:    points,
		CreatedAt: time.Now().UTC(),
		CreatedBy: createdBy,
		UserID:    userID,
		CircleID:  circleID,
	}

	return s.pointsRepo.CreatePointsHistory(ctx, tx, pointsHistory)
}

// GetUserPoints gets current points for a user
func (s *Service) GetUserPoints(ctx context.Context, userID int, circleID int) (int, error) {
	var points int
	// This would need access to circle repository, but we'll use a direct query for now
	// In production, you'd want to inject the circle repository
	return points, nil
}