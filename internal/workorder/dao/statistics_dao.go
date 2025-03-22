package dao

import "context"

type StatisticsDAO interface {
	GetOverview(ctx context.Context)
	GetTrend(ctx context.Context)
	GetCategoryStats(ctx context.Context)
	GetPerformanceStats(ctx context.Context)
	GetUserStats(ctx context.Context)
}

type statisticsDAO struct {
}

func NewStatisticsDAO() StatisticsDAO {
	return &statisticsDAO{}
}

// GetCategoryStats implements StatisticsDAO.
func (s *statisticsDAO) GetCategoryStats(ctx context.Context) {
	panic("unimplemented")
}

// GetOverview implements StatisticsDAO.
func (s *statisticsDAO) GetOverview(ctx context.Context) {
	panic("unimplemented")
}

// GetPerformanceStats implements StatisticsDAO.
func (s *statisticsDAO) GetPerformanceStats(ctx context.Context) {
	panic("unimplemented")
}

// GetTrend implements StatisticsDAO.
func (s *statisticsDAO) GetTrend(ctx context.Context) {
	panic("unimplemented")
}

// GetUserStats implements StatisticsDAO.
func (s *statisticsDAO) GetUserStats(ctx context.Context) {
	panic("unimplemented")
}
