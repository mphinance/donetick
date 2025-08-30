package rewards

import (
	"strconv"

	"donetick.com/core/config"
	rRepo "donetick.com/core/internal/rewards/repo"
	uRepo "donetick.com/core/internal/user/repo"
	"donetick.com/core/internal/utils"
	jwt "github.com/appleboy/gin-jwt/v2"
	"github.com/gin-gonic/gin"
	"github.com/ulule/limiter/v3"
)

type API struct {
	rewardsRepo *rRepo.RewardsRepository
	userRepo    *uRepo.UserRepository
}

func NewAPI(rr *rRepo.RewardsRepository, ur *uRepo.UserRepository) *API {
	return &API{
		rewardsRepo: rr,
		userRepo:    ur,
	}
}

func (a *API) GetUserStats(c *gin.Context) {
	apiToken := c.GetHeader("secretkey")
	if apiToken == "" {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}
	
	user, err := a.userRepo.GetUserByToken(c, apiToken)
	if err != nil {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}

	stats, err := a.rewardsRepo.GetUserPointsStats(c, user.ID, user.CircleID)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to get user stats"})
		return
	}

	c.JSON(200, stats)
}

func (a *API) GetLeaderboard(c *gin.Context) {
	apiToken := c.GetHeader("secretkey")
	if apiToken == "" {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}
	
	user, err := a.userRepo.GetUserByToken(c, apiToken)
	if err != nil {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}

	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 50 {
		limit = 10
	}

	leaderboard, err := a.rewardsRepo.GetLeaderboard(c, user.CircleID, limit)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to get leaderboard"})
		return
	}

	c.JSON(200, leaderboard)
}

func (a *API) GetUserGoals(c *gin.Context) {
	apiToken := c.GetHeader("secretkey")
	if apiToken == "" {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}
	
	user, err := a.userRepo.GetUserByToken(c, apiToken)
	if err != nil {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}

	progress, err := a.rewardsRepo.GetUserGoalProgress(c, user.ID, user.CircleID)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to get goal progress"})
		return
	}

	c.JSON(200, progress)
}

func APIs(cfg *config.Config, api *API, r *gin.Engine, auth *jwt.GinJWTMiddleware, limiter *limiter.Limiter) {
	rewardsAPI := r.Group("eapi/v1/rewards")
	rewardsAPI.Use(utils.TimeoutMiddleware(cfg.Server.WriteTimeout), utils.RateLimitMiddleware(limiter))
	{
		rewardsAPI.GET("/stats", api.GetUserStats)
		rewardsAPI.GET("/leaderboard", api.GetLeaderboard)
		rewardsAPI.GET("/goals", api.GetUserGoals)
	}
}