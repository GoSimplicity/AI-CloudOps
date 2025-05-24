package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/service/mocks" // Adjust path if necessary
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// Helper function to create a new StatisticsService with mocks
func newTestStatisticsService(t *testing.T) (
	StatisticsService,
	*mocks.MockStatisticsDAO,
	*mocks.MockUserDAO,
	context.Context,
) {
	ctrl := gomock.NewController(t)
	mockStatsDAO := mocks.NewMockStatisticsDAO(ctrl)
	mockUserDAO := mocks.NewMockUserDAO(ctrl)
	logger := zap.NewNop()

	service := NewStatisticsService(mockStatsDAO, mockUserDAO, logger)
	ctx := context.Background()
	return service, mockStatsDAO, mockUserDAO, ctx
}

func TestStatisticsService_GetOverview(t *testing.T) {
	service, mockStatsDAO, _, ctx := newTestStatisticsService(t)
	req := model.OverviewStatsReq{} // Assuming empty or with date range

	t.Run("Success", func(t *testing.T) {
		mockDAOData := &model.OverviewStatsDAO{
			TotalCount:      100,
			CompletedCount:  70,
			ProcessingCount: 10,
			PendingCount:    5,
			OverdueCount:    2,
			CompletionRate:  70.0,
			AvgProcessTime:  2.5, // hours
			TodayCreated:    8,
			TodayCompleted:  4,
		}
		mockStatsDAO.EXPECT().GetOverviewStats(ctx, req.StartDate, req.EndDate).Return(mockDAOData, nil).Times(1)

		resp, err := service.GetOverview(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, mockDAOData.TotalCount, resp.TotalCount)
		assert.Equal(t, mockDAOData.CompletedCount, resp.CompletedCount)
		assert.Equal(t, mockDAOData.ProcessingCount, resp.ProcessingCount)
		assert.Equal(t, mockDAOData.PendingCount, resp.PendingCount)
		assert.Equal(t, mockDAOData.OverdueCount, resp.OverdueCount)
		assert.Equal(t, mockDAOData.CompletionRate, resp.CompletionRate)
		assert.Equal(t, mockDAOData.AvgProcessTime, resp.AvgProcessTime)
		assert.Equal(t, mockDAOData.TodayCreated, resp.TodayCreated)
		assert.Equal(t, mockDAOData.TodayCompleted, resp.TodayCompleted)
	})

	t.Run("DAOError", func(t *testing.T) {
		mockStatsDAO.EXPECT().GetOverviewStats(ctx, req.StartDate, req.EndDate).Return(nil, errors.New("DAO error")).Times(1)
		resp, err := service.GetOverview(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "DAO error")
	})
}

func TestStatisticsService_GetTrend(t *testing.T) {
	service, mockStatsDAO, _, ctx := newTestStatisticsService(t)
	req := model.TrendStatsReq{
		StartDate: time.Now().Add(-7 * 24 * time.Hour),
		EndDate:   time.Now(),
		Dimension: "day",
	}

	t.Run("Success", func(t *testing.T) {
		mockDAOData := &model.TrendStatsDAO{
			Dates:            []string{"2023-01-01", "2023-01-02"},
			CreatedCounts:    []int{10, 12},
			CompletedCounts:  []int{8, 9},
			ProcessingCounts: []int{2, 3},
		}
		mockStatsDAO.EXPECT().GetInstanceTrendStats(ctx, req.StartDate, req.EndDate, req.Dimension, req.CategoryID).Return(mockDAOData, nil).Times(1)

		resp, err := service.GetTrend(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, mockDAOData.Dates, resp.Dates)
		assert.Equal(t, mockDAOData.CreatedCounts, resp.CreatedCounts)
		assert.Equal(t, mockDAOData.CompletedCounts, resp.CompletedCounts)
		assert.Equal(t, mockDAOData.ProcessingCounts, resp.ProcessingCounts)
	})

	t.Run("DAOError", func(t *testing.T) {
		mockStatsDAO.EXPECT().GetInstanceTrendStats(ctx, req.StartDate, req.EndDate, req.Dimension, req.CategoryID).Return(nil, errors.New("DAO error")).Times(1)
		resp, err := service.GetTrend(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "DAO error")
	})
}

func TestStatisticsService_GetCategoryStats(t *testing.T) {
	service, mockStatsDAO, _, ctx := newTestStatisticsService(t)
	top := 5
	req := model.CategoryStatsReq{Top: top}

	t.Run("Success", func(t *testing.T) {
		mockDAOData := []model.CategoryStatsItemDAO{
			{CategoryID: 1, CategoryName: "Category A", Count: 50, Percentage: 50.0},
			{CategoryID: 2, CategoryName: "Category B", Count: 30, Percentage: 30.0},
		}
		mockStatsDAO.EXPECT().GetWorkloadByCategory(ctx, req.StartDate, req.EndDate, &req.Top).Return(mockDAOData, nil).Times(1)

		resp, err := service.GetCategoryStats(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Items, len(mockDAOData))
		if len(mockDAOData) > 0 {
			assert.Equal(t, mockDAOData[0].CategoryName, resp.Items[0].CategoryName)
			assert.Equal(t, mockDAOData[0].Count, resp.Items[0].Count)
			assert.Equal(t, mockDAOData[0].Percentage, resp.Items[0].Percentage)
		}
	})

	t.Run("DAOError", func(t *testing.T) {
		mockStatsDAO.EXPECT().GetWorkloadByCategory(ctx, req.StartDate, req.EndDate, &req.Top).Return(nil, errors.New("DAO error")).Times(1)
		resp, err := service.GetCategoryStats(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "DAO error")
	})
}

func TestStatisticsService_GetPerformanceStats(t *testing.T) {
	service, mockStatsDAO, mockUserDAO, ctx := newTestStatisticsService(t)
	top := 10
	req := model.PerformanceStatsReq{Top: top}

	userID1, userID2 := 1, 2
	userName1, userName2 := "User One", "User Two"

	t.Run("Success", func(t *testing.T) {
		mockDAOData := []model.PerformanceStatsItemDAO{
			{UserID: userID1, AssignedCount: 20, CompletedCount: 18, OverdueCount: 1, CompletionRate: 90.0, AvgResponseTime: 1.0, AvgProcessingTime: 5.0},
			{UserID: userID2, AssignedCount: 15, CompletedCount: 10, OverdueCount: 0, CompletionRate: 66.67, AvgResponseTime: 1.5, AvgProcessingTime: 7.0},
		}
		mockStatsDAO.EXPECT().GetOperatorPerformance(ctx, req.StartDate, req.EndDate, req.UserID, &req.Top).Return(mockDAOData, nil).Times(1)
		mockUserDAO.EXPECT().GetUserByID(ctx, userID1).Return(&model.User{Uid: userID1, Username: userName1}, nil).Times(1)
		mockUserDAO.EXPECT().GetUserByID(ctx, userID2).Return(&model.User{Uid: userID2, Username: userName2}, nil).Times(1)

		resp, err := service.GetPerformanceStats(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Items, len(mockDAOData))
		if len(resp.Items) > 0 {
			assert.Equal(t, userID1, resp.Items[0].UserID)
			assert.Equal(t, userName1, resp.Items[0].UserName)
			assert.Equal(t, mockDAOData[0].AssignedCount, resp.Items[0].AssignedCount)

			assert.Equal(t, userID2, resp.Items[1].UserID)
			assert.Equal(t, userName2, resp.Items[1].UserName)
		}
	})

	t.Run("DAOError", func(t *testing.T) {
		mockStatsDAO.EXPECT().GetOperatorPerformance(ctx, req.StartDate, req.EndDate, req.UserID, &req.Top).Return(nil, errors.New("DAO error")).Times(1)
		resp, err := service.GetPerformanceStats(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "DAO error")
	})

	t.Run("UserDAOError", func(t *testing.T) {
		mockDAOData := []model.PerformanceStatsItemDAO{
			{UserID: userID1, AssignedCount: 20, CompletedCount: 18},
		}
		mockStatsDAO.EXPECT().GetOperatorPerformance(ctx, req.StartDate, req.EndDate, req.UserID, &req.Top).Return(mockDAOData, nil).Times(1)
		mockUserDAO.EXPECT().GetUserByID(ctx, userID1).Return(nil, errors.New("UserDAO error")).Times(1)

		resp, err := service.GetPerformanceStats(ctx, req)
		assert.NoError(t, err) // Service should log warning and continue
		assert.NotNil(t, resp)
		assert.Len(t, resp.Items, 1)
		if len(resp.Items) > 0 {
			assert.Equal(t, userID1, resp.Items[0].UserID)
			assert.Contains(t, resp.Items[0].UserName, fmt.Sprintf("用户ID %d", userID1)) // Check for placeholder name
		}
	})
}

func TestStatisticsService_GetUserStats(t *testing.T) {
	service, mockStatsDAO, _, ctx := newTestStatisticsService(t)
	userID := 1
	req := model.UserStatsReq{UserID: &userID}

	t.Run("Success", func(t *testing.T) {
		mockDAOData := &model.UserStatsDAO{
			UserID:            userID,
			CreatedCount:      5,
			AssignedCount:     10,
			CompletedCount:    8,
			PendingCount:      1,
			OverdueCount:      1,
			AvgResponseTime:   0.5,
			AvgProcessingTime: 3.0,
			SatisfactionScore: 4.5,
		}
		mockStatsDAO.EXPECT().GetStatsByUser(ctx, req.StartDate, req.EndDate, req.UserID).Return(mockDAOData, nil).Times(1)

		resp, err := service.GetUserStats(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, mockDAOData.CreatedCount, resp.CreatedCount)
		assert.Equal(t, mockDAOData.AssignedCount, resp.AssignedCount)
		assert.Equal(t, mockDAOData.CompletedCount, resp.CompletedCount)
		assert.Equal(t, mockDAOData.PendingCount, resp.PendingCount)
		assert.Equal(t, mockDAOData.OverdueCount, resp.OverdueCount)
		assert.Equal(t, mockDAOData.AvgResponseTime, resp.AvgResponseTime)
		assert.Equal(t, mockDAOData.AvgProcessingTime, resp.AvgProcessingTime)
		assert.Equal(t, mockDAOData.SatisfactionScore, resp.SatisfactionScore)
	})

	t.Run("DAOError", func(t *testing.T) {
		mockStatsDAO.EXPECT().GetStatsByUser(ctx, req.StartDate, req.EndDate, req.UserID).Return(nil, errors.New("DAO error")).Times(1)
		resp, err := service.GetUserStats(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "DAO error")
	})
}
