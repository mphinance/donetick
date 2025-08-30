package rewards

import (
	"strconv"
	"time"

	auth "donetick.com/core/internal/authorization"
	cRepo "donetick.com/core/internal/circle/repo"
	rModel "donetick.com/core/internal/rewards/model"
	rRepo "donetick.com/core/internal/rewards/repo"
	"donetick.com/core/logging"
	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	rewardsRepo *rRepo.RewardsRepository
	circleRepo  *cRepo.CircleRepository
}

func NewHandler(rr *rRepo.RewardsRepository, cr *cRepo.CircleRepository) *Handler {
	return &Handler{
		rewardsRepo: rr,
		circleRepo:  cr,
	}
}

// Rewards endpoints
func (h *Handler) CreateReward(c *gin.Context) {
	log := logging.FromContext(c)
	currentUser, ok := auth.CurrentUser(c)
	if !ok {
		c.JSON(500, gin.H{"error": "Error getting current user"})
		return
	}

	// Check if user is admin
	if !h.isCircleAdmin(c, currentUser.ID, currentUser.CircleID) {
		c.JSON(403, gin.H{"error": "Only admins can create rewards"})
		return
	}

	type CreateRewardReq struct {
		Name        string  `json:"name" binding:"required"`
		Description *string `json:"description"`
		PointsCost  int     `json:"pointsCost" binding:"required,min=1"`
		Category    string  `json:"category"`
		Icon        *string `json:"icon"`
		Color       *string `json:"color"`
		MaxRedeems  *int    `json:"maxRedeems"`
	}

	var req CreateRewardReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	reward := &rModel.Reward{
		Name:        req.Name,
		Description: req.Description,
		PointsCost:  req.PointsCost,
		CircleID:    currentUser.CircleID,
		CreatedBy:   currentUser.ID,
		Category:    req.Category,
		Icon:        req.Icon,
		Color:       req.Color,
		MaxRedeems:  req.MaxRedeems,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if reward.Category == "" {
		reward.Category = "general"
	}

	if err := h.rewardsRepo.CreateReward(c, reward); err != nil {
		log.Errorw("Failed to create reward", "error", err)
		c.JSON(500, gin.H{"error": "Failed to create reward"})
		return
	}

	c.JSON(201, gin.H{"res": reward})
}

func (h *Handler) GetRewards(c *gin.Context) {
	currentUser, ok := auth.CurrentUser(c)
	if !ok {
		c.JSON(500, gin.H{"error": "Error getting current user"})
		return
	}

	rewards, err := h.rewardsRepo.GetRewardsByCircle(c, currentUser.CircleID)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to get rewards"})
		return
	}

	c.JSON(200, gin.H{"res": rewards})
}

func (h *Handler) RedeemReward(c *gin.Context) {
	log := logging.FromContext(c)
	currentUser, ok := auth.CurrentUser(c)
	if !ok {
		c.JSON(500, gin.H{"error": "Error getting current user"})
		return
	}

	rewardIDStr := c.Param("id")
	rewardID, err := strconv.Atoi(rewardIDStr)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid reward ID"})
		return
	}

	// Get reward details
	reward, err := h.rewardsRepo.GetRewardByID(c, rewardID)
	if err != nil {
		c.JSON(404, gin.H{"error": "Reward not found"})
		return
	}

	if reward.CircleID != currentUser.CircleID {
		c.JSON(403, gin.H{"error": "Reward not available in your circle"})
		return
	}

	// Check if reward is still available
	if reward.MaxRedeems != nil && reward.TimesRedeemed >= *reward.MaxRedeems {
		c.JSON(400, gin.H{"error": "Reward is no longer available"})
		return
	}

	// Get user's current points
	circleUsers, err := h.circleRepo.GetCircleUsers(c, currentUser.CircleID)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to get user points"})
		return
	}

	var userPoints int
	for _, user := range circleUsers {
		if user.UserID == currentUser.ID {
			userPoints = user.Points
			break
		}
	}

	if userPoints < reward.PointsCost {
		c.JSON(400, gin.H{"error": "Insufficient points"})
		return
	}

	// Create redemption
	redemption := &rModel.RewardRedemption{
		RewardID:  rewardID,
		UserID:    currentUser.ID,
		CircleID:  currentUser.CircleID,
		Points:    reward.PointsCost,
		Status:    rModel.RedemptionStatusPending,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	if err := h.rewardsRepo.CreateRedemption(c, redemption); err != nil {
		log.Errorw("Failed to create redemption", "error", err)
		c.JSON(500, gin.H{"error": "Failed to redeem reward"})
		return
	}

	// Deduct points from user
	if err := h.circleRepo.RedeemPoints(c, currentUser.CircleID, currentUser.ID, reward.PointsCost, currentUser.ID); err != nil {
		log.Errorw("Failed to deduct points", "error", err)
		c.JSON(500, gin.H{"error": "Failed to process redemption"})
		return
	}

	c.JSON(200, gin.H{"res": redemption})
}

func (h *Handler) GetRedemptions(c *gin.Context) {
	currentUser, ok := auth.CurrentUser(c)
	if !ok {
		c.JSON(500, gin.H{"error": "Error getting current user"})
		return
	}

	redemptions, err := h.rewardsRepo.GetRedemptionsByUser(c, currentUser.ID, currentUser.CircleID)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to get redemptions"})
		return
	}

	c.JSON(200, gin.H{"res": redemptions})
}

// Goals endpoints
func (h *Handler) CreateGoal(c *gin.Context) {
	log := logging.FromContext(c)
	currentUser, ok := auth.CurrentUser(c)
	if !ok {
		c.JSON(500, gin.H{"error": "Error getting current user"})
		return
	}

	// Check if user is admin
	if !h.isCircleAdmin(c, currentUser.ID, currentUser.CircleID) {
		c.JSON(403, gin.H{"error": "Only admins can create goals"})
		return
	}

	type CreateGoalReq struct {
		Name         string     `json:"name" binding:"required"`
		Description  *string    `json:"description"`
		TargetPoints int        `json:"targetPoints" binding:"required,min=1"`
		UserID       *int       `json:"userId"` // null = circle-wide goal
		Category     string     `json:"category"`
		Icon         *string    `json:"icon"`
		Color        *string    `json:"color"`
		StartDate    *time.Time `json:"startDate"`
		EndDate      *time.Time `json:"endDate"`
		RewardPoints *int       `json:"rewardPoints"`
	}

	var req CreateGoalReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	goal := &rModel.Goal{
		Name:         req.Name,
		Description:  req.Description,
		TargetPoints: req.TargetPoints,
		CircleID:     currentUser.CircleID,
		UserID:       req.UserID,
		CreatedBy:    currentUser.ID,
		Category:     req.Category,
		Icon:         req.Icon,
		Color:        req.Color,
		StartDate:    req.StartDate,
		EndDate:      req.EndDate,
		RewardPoints: req.RewardPoints,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	if goal.Category == "" {
		goal.Category = "general"
	}

	if err := h.rewardsRepo.CreateGoal(c, goal); err != nil {
		log.Errorw("Failed to create goal", "error", err)
		c.JSON(500, gin.H{"error": "Failed to create goal"})
		return
	}

	c.JSON(201, gin.H{"res": goal})
}

func (h *Handler) GetGoals(c *gin.Context) {
	currentUser, ok := auth.CurrentUser(c)
	if !ok {
		c.JSON(500, gin.H{"error": "Error getting current user"})
		return
	}

	goals, err := h.rewardsRepo.GetGoalsByCircle(c, currentUser.CircleID, &currentUser.ID)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to get goals"})
		return
	}

	c.JSON(200, gin.H{"res": goals})
}

func (h *Handler) GetGoalProgress(c *gin.Context) {
	currentUser, ok := auth.CurrentUser(c)
	if !ok {
		c.JSON(500, gin.H{"error": "Error getting current user"})
		return
	}

	progress, err := h.rewardsRepo.GetUserGoalProgress(c, currentUser.ID, currentUser.CircleID)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to get goal progress"})
		return
	}

	c.JSON(200, gin.H{"res": progress})
}

// Leaderboard and stats
func (h *Handler) GetLeaderboard(c *gin.Context) {
	currentUser, ok := auth.CurrentUser(c)
	if !ok {
		c.JSON(500, gin.H{"error": "Error getting current user"})
		return
	}

	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 50 {
		limit = 10
	}

	leaderboard, err := h.rewardsRepo.GetLeaderboard(c, currentUser.CircleID, limit)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to get leaderboard"})
		return
	}

	c.JSON(200, gin.H{"res": leaderboard})
}

func (h *Handler) GetPointsStats(c *gin.Context) {
	currentUser, ok := auth.CurrentUser(c)
	if !ok {
		c.JSON(500, gin.H{"error": "Error getting current user"})
		return
	}

	stats, err := h.rewardsRepo.GetUserPointsStats(c, currentUser.ID, currentUser.CircleID)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to get points stats"})
		return
	}

	c.JSON(200, gin.H{"res": stats})
}

// Admin endpoints
func (h *Handler) GetAllRedemptions(c *gin.Context) {
	currentUser, ok := auth.CurrentUser(c)
	if !ok {
		c.JSON(500, gin.H{"error": "Error getting current user"})
		return
	}

	// Check if user is admin
	if !h.isCircleAdmin(c, currentUser.ID, currentUser.CircleID) {
		c.JSON(403, gin.H{"error": "Only admins can view all redemptions"})
		return
	}

	redemptions, err := h.rewardsRepo.GetRedemptionsByCircle(c, currentUser.CircleID)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to get redemptions"})
		return
	}

	c.JSON(200, gin.H{"res": redemptions})
}

func (h *Handler) UpdateRedemptionStatus(c *gin.Context) {
	log := logging.FromContext(c)
	currentUser, ok := auth.CurrentUser(c)
	if !ok {
		c.JSON(500, gin.H{"error": "Error getting current user"})
		return
	}

	// Check if user is admin
	if !h.isCircleAdmin(c, currentUser.ID, currentUser.CircleID) {
		c.JSON(403, gin.H{"error": "Only admins can update redemption status"})
		return
	}

	redemptionIDStr := c.Param("id")
	redemptionID, err := strconv.Atoi(redemptionIDStr)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid redemption ID"})
		return
	}

	type UpdateStatusReq struct {
		Status rModel.RedemptionStatus `json:"status" binding:"required"`
		Notes  *string                 `json:"notes"`
	}

	var req UpdateStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	if err := h.rewardsRepo.UpdateRedemptionStatus(c, redemptionID, req.Status, req.Notes); err != nil {
		log.Errorw("Failed to update redemption status", "error", err)
		c.JSON(500, gin.H{"error": "Failed to update redemption status"})
		return
	}

	c.JSON(200, gin.H{"message": "Redemption status updated successfully"})
}

func (h *Handler) UpdateGoalProgress(c *gin.Context) {
	log := logging.FromContext(c)
	currentUser, ok := auth.CurrentUser(c)
	if !ok {
		c.JSON(500, gin.H{"error": "Error getting current user"})
		return
	}

	if err := h.rewardsRepo.UpdateGoalProgress(c, currentUser.CircleID); err != nil {
		log.Errorw("Failed to update goal progress", "error", err)
		c.JSON(500, gin.H{"error": "Failed to update goal progress"})
		return
	}

	c.JSON(200, gin.H{"message": "Goal progress updated successfully"})
}

// Helper methods
func (h *Handler) isCircleAdmin(c *gin.Context, userID int, circleID int) bool {
	admins, err := h.circleRepo.GetCircleAdmins(c, circleID)
	if err != nil {
		return false
	}

	for _, admin := range admins {
		if admin.UserID == userID {
			return true
		}
	}
	return false
}

func Routes(r *gin.Engine, h *Handler, auth *jwt.GinJWTMiddleware) {
	rewardsRoutes := r.Group("api/v1/rewards")
	rewardsRoutes.Use(auth.MiddlewareFunc())
	{
		// Rewards
		rewardsRoutes.POST("", h.CreateReward)
		rewardsRoutes.GET("", h.GetRewards)
		rewardsRoutes.POST("/:id/redeem", h.RedeemReward)
		
		// Goals
		rewardsRoutes.POST("/goals", h.CreateGoal)
		rewardsRoutes.GET("/goals", h.GetGoals)
		rewardsRoutes.GET("/goals/progress", h.GetGoalProgress)
		rewardsRoutes.POST("/goals/update-progress", h.UpdateGoalProgress)
		
		// Stats and leaderboard
		rewardsRoutes.GET("/leaderboard", h.GetLeaderboard)
		rewardsRoutes.GET("/stats", h.GetPointsStats)
		
		// Admin endpoints
		rewardsRoutes.GET("/redemptions", h.GetRedemptions)
		rewardsRoutes.GET("/admin/redemptions", h.GetAllRedemptions)
		rewardsRoutes.PUT("/admin/redemptions/:id", h.UpdateRedemptionStatus)
	}
}