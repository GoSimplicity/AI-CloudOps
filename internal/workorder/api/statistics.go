package api

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/service"
	"github.com/gin-gonic/gin"
)

type StatisticsHandler struct {
	service service.StatisticsService
}

func NewStatisticsHandler(service service.StatisticsService) *StatisticsHandler {
	return &StatisticsHandler{
		service: service,
	}
}

func (h *StatisticsHandler) RegisterRouters(server *gin.Engine) {
	statsGroup := server.Group("/api/workorder/statistics")
	{
		statsGroup.POST("/overview", h.GetOverview)
		statsGroup.POST("/trend", h.GetTrend)
		statsGroup.POST("/category", h.GetCategoryStats)
		statsGroup.POST("/performance", h.GetPerformanceStats)
		statsGroup.POST("/user", h.GetUserStats)
	}
}

func (h *StatisticsHandler) GetOverview(ctx *gin.Context) {
}

func (h *StatisticsHandler) GetTrend(ctx *gin.Context) {
}

func (h *StatisticsHandler) GetCategoryStats(ctx *gin.Context) {
}

func (h *StatisticsHandler) GetPerformanceStats(ctx *gin.Context) {
}

func (h *StatisticsHandler) GetUserStats(ctx *gin.Context) {

}
