package model

import (
	"time"
)

type Reward struct {
	ID          int       `json:"id" gorm:"primary_key"`
	Name        string    `json:"name" gorm:"column:name;not null"`
	Description *string   `json:"description" gorm:"column:description;type:text"`
	PointsCost  int       `json:"pointsCost" gorm:"column:points_cost;not null"`
	CircleID    int       `json:"circleId" gorm:"column:circle_id;index;not null"`
	CreatedBy   int       `json:"createdBy" gorm:"column:created_by;not null"`
	IsActive    bool      `json:"isActive" gorm:"column:is_active;default:true;not null"`
	Category    string    `json:"category" gorm:"column:category;default:'general'"`
	Icon        *string   `json:"icon" gorm:"column:icon"`
	Color       *string   `json:"color" gorm:"column:color;default:'#3B82F6'"`
	MaxRedeems  *int      `json:"maxRedeems" gorm:"column:max_redeems"` // null = unlimited
	TimesRedeemed int     `json:"timesRedeemed" gorm:"column:times_redeemed;default:0;not null"`
	CreatedAt   time.Time `json:"createdAt" gorm:"column:created_at"`
	UpdatedAt   time.Time `json:"updatedAt" gorm:"column:updated_at"`
}

type RewardRedemption struct {
	ID        int       `json:"id" gorm:"primary_key"`
	RewardID  int       `json:"rewardId" gorm:"column:reward_id;index;not null"`
	UserID    int       `json:"userId" gorm:"column:user_id;index;not null"`
	CircleID  int       `json:"circleId" gorm:"column:circle_id;index;not null"`
	Points    int       `json:"points" gorm:"column:points;not null"`
	Status    RedemptionStatus `json:"status" gorm:"column:status;default:0;not null"`
	Notes     *string   `json:"notes" gorm:"column:notes;type:text"`
	CreatedAt time.Time `json:"createdAt" gorm:"column:created_at"`
	UpdatedAt time.Time `json:"updatedAt" gorm:"column:updated_at"`
	
	// Relations
	Reward *Reward `json:"reward,omitempty" gorm:"foreignkey:RewardID;references:ID"`
}

type RedemptionStatus int8

const (
	RedemptionStatusPending   RedemptionStatus = 0
	RedemptionStatusApproved  RedemptionStatus = 1
	RedemptionStatusRejected  RedemptionStatus = 2
	RedemptionStatusCompleted RedemptionStatus = 3
)

type Goal struct {
	ID          int       `json:"id" gorm:"primary_key"`
	Name        string    `json:"name" gorm:"column:name;not null"`
	Description *string   `json:"description" gorm:"column:description;type:text"`
	TargetPoints int      `json:"targetPoints" gorm:"column:target_points;not null"`
	CircleID    int       `json:"circleId" gorm:"column:circle_id;index;not null"`
	UserID      *int      `json:"userId" gorm:"column:user_id;index"` // null = circle-wide goal
	CreatedBy   int       `json:"createdBy" gorm:"column:created_by;not null"`
	IsActive    bool      `json:"isActive" gorm:"column:is_active;default:true;not null"`
	Category    string    `json:"category" gorm:"column:category;default:'general'"`
	Icon        *string   `json:"icon" gorm:"column:icon"`
	Color       *string   `json:"color" gorm:"column:color;default:'#10B981'"`
	StartDate   *time.Time `json:"startDate" gorm:"column:start_date"`
	EndDate     *time.Time `json:"endDate" gorm:"column:end_date"`
	RewardPoints *int     `json:"rewardPoints" gorm:"column:reward_points"` // bonus points for completing goal
	CompletedAt *time.Time `json:"completedAt" gorm:"column:completed_at"`
	CreatedAt   time.Time `json:"createdAt" gorm:"column:created_at"`
	UpdatedAt   time.Time `json:"updatedAt" gorm:"column:updated_at"`
}

type GoalProgress struct {
	GoalID        int     `json:"goalId" gorm:"column:goal_id;primaryKey"`
	UserID        int     `json:"userId" gorm:"column:user_id;primaryKey"`
	CurrentPoints int     `json:"currentPoints" gorm:"column:current_points;default:0;not null"`
	Progress      float64 `json:"progress" gorm:"column:progress;default:0.0;not null"` // percentage 0-100
	CompletedAt   *time.Time `json:"completedAt" gorm:"column:completed_at"`
	UpdatedAt     time.Time `json:"updatedAt" gorm:"column:updated_at"`
	
	// Relations
	Goal *Goal `json:"goal,omitempty" gorm:"foreignkey:GoalID;references:ID"`
}

type PointsLeaderboard struct {
	UserID      int    `json:"userId"`
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	Image       string `json:"image"`
	Points      int    `json:"points"`
	Rank        int    `json:"rank"`
	PointsThisWeek int `json:"pointsThisWeek"`
	PointsThisMonth int `json:"pointsThisMonth"`
}