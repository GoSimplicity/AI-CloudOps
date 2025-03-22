package service

import (
	"context"

	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
)

type StatisticsService interface {
	GetOverview(ctx context.Context)
	GetTrend(ctx context.Context)
	GetCategoryStats(ctx context.Context)
	GetPerformanceStats(ctx context.Context)
	GetUserStats(ctx context.Context)
}

type statisticsService struct {
	dao dao.StatisticsDAO
}

func NewStatisticsService(dao dao.StatisticsDAO) StatisticsService {
	return &statisticsService{dao: dao}
}

// GetCategoryStats implements StatisticsService.
func (s *statisticsService) GetCategoryStats(ctx context.Context) {
	panic("unimplemented")
}

// GetOverview implements StatisticsService.
func (s *statisticsService) GetOverview(ctx context.Context) {
	panic("unimplemented")
}

// GetPerformanceStats implements StatisticsService.
func (s *statisticsService) GetPerformanceStats(ctx context.Context) {
	panic("unimplemented")
}

// GetTrend implements StatisticsService.
func (s *statisticsService) GetTrend(ctx context.Context) {
	panic("unimplemented")
}

// GetUserStats implements StatisticsService.
func (s *statisticsService) GetUserStats(ctx context.Context) {
	panic("unimplemented")
}
