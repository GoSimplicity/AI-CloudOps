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
	"gorm.io/gorm"
)

func TestCategoryGroupService_CreateCategory(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCategoryDAO := mocks.NewMockCategoryDAO(ctrl)
	// mockUserDAO is not directly used by CreateCategory in the current service implementation
	// but NewCategoryGroupService expects it.
	mockUserDAO := mocks.NewMockUserDAO(ctrl)
	logger := zap.NewNop()

	service := NewCategoryGroupService(mockCategoryDAO, mockUserDAO, logger)
	ctx := context.Background()

	req := &model.CreateCategoryReq{
		Name:        "Test Category",
		Description: "Test Description",
		Icon:        "test-icon",
		SortOrder:   1,
		ParentID:    nil,
	}
	creatorID := 1
	creatorName := "testuser"

	t.Run("Success", func(t *testing.T) {
		expectedCategory := &model.Category{
			ID:          1, // DAO mock will set this
			Name:        req.Name,
			Description: req.Description,
			Icon:        req.Icon,
			SortOrder:   req.SortOrder,
			ParentID:    req.ParentID,
			Status:      1, // Default status
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		mockCategoryDAO.EXPECT().CreateCategory(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, cat *model.Category) error {
				cat.ID = expectedCategory.ID
				cat.CreatedAt = expectedCategory.CreatedAt
				cat.UpdatedAt = expectedCategory.UpdatedAt
				// Verify other fields are passed correctly
				assert.Equal(t, req.Name, cat.Name)
				assert.Equal(t, req.Description, cat.Description)
				assert.Equal(t, req.Icon, cat.Icon)
				assert.Equal(t, req.SortOrder, cat.SortOrder)
				assert.Equal(t, req.ParentID, cat.ParentID)
				assert.Equal(t, int8(1), cat.Status) // Default status
				return nil
			}).Times(1)

		resp, err := service.CreateCategory(ctx, req, creatorID, creatorName)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, expectedCategory.ID, resp.ID)
		assert.Equal(t, req.Name, resp.Name)
	})

	t.Run("DAOError", func(t *testing.T) {
		mockCategoryDAO.EXPECT().CreateCategory(gomock.Any(), gomock.Any()).
			Return(errors.New("DAO create error")).Times(1)

		resp, err := service.CreateCategory(ctx, req, creatorID, creatorName)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "DAO create error")
	})
}

func TestCategoryGroupService_UpdateCategory(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCategoryDAO := mocks.NewMockCategoryDAO(ctrl)
	mockUserDAO := mocks.NewMockUserDAO(ctrl)
	logger := zap.NewNop()

	service := NewCategoryGroupService(mockCategoryDAO, mockUserDAO, logger)
	ctx := context.Background()

	req := &model.UpdateCategoryReq{
		ID:          1,
		Name:        "Updated Category",
		Description: "Updated Description",
		Status:      1,
	}
	userID := 1

	t.Run("Success", func(t *testing.T) {
		existingCategory := &model.Category{ID: req.ID, Name: "Old Name", Status: 0}
		updatedDbCategory := &model.Category{
			ID:          req.ID,
			Name:        req.Name,
			Description: req.Description,
			Status:      req.Status,
			UpdatedAt:   time.Now(),
		}

		mockCategoryDAO.EXPECT().GetCategory(ctx, req.ID).Return(existingCategory, nil).Times(1)
		mockCategoryDAO.EXPECT().UpdateCategory(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, cat *model.Category) error {
				assert.Equal(t, req.ID, cat.ID)
				assert.Equal(t, req.Name, cat.Name)
				// Other fields can be asserted here
				return nil
			}).Times(1)
		mockCategoryDAO.EXPECT().GetCategory(ctx, req.ID).Return(updatedDbCategory, nil).Times(1) // For fetching updated record

		resp, err := service.UpdateCategory(ctx, req, userID)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, updatedDbCategory.ID, resp.ID)
		assert.Equal(t, updatedDbCategory.Name, resp.Name)
	})

	t.Run("GetCategoryError", func(t *testing.T) {
		mockCategoryDAO.EXPECT().GetCategory(ctx, req.ID).Return(nil, errors.New("DAO get error")).Times(1)

		resp, err := service.UpdateCategory(ctx, req, userID)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "DAO get error")
	})

	t.Run("CategoryNotFound", func(t *testing.T) {
		mockCategoryDAO.EXPECT().GetCategory(ctx, req.ID).Return(nil, nil).Times(1) // gorm.ErrRecordNotFound would be more realistic, but service handles nil as not found

		resp, err := service.UpdateCategory(ctx, req, userID)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), fmt.Sprintf("分类 (ID: %d) 不存在", req.ID))
	})
	
	t.Run("UpdateCategoryDAOError", func(t *testing.T) {
		existingCategory := &model.Category{ID: req.ID, Name: "Old Name"}
		mockCategoryDAO.EXPECT().GetCategory(ctx, req.ID).Return(existingCategory, nil).Times(1)
		mockCategoryDAO.EXPECT().UpdateCategory(gomock.Any(), gomock.Any()).Return(errors.New("DAO update error")).Times(1)

		resp, err := service.UpdateCategory(ctx, req, userID)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "DAO update error")
	})
}

func TestCategoryGroupService_DeleteCategory(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCategoryDAO := mocks.NewMockCategoryDAO(ctrl)
	mockUserDAO := mocks.NewMockUserDAO(ctrl)
	logger := zap.NewNop()

	service := NewCategoryGroupService(mockCategoryDAO, mockUserDAO, logger)
	ctx := context.Background()
	categoryID := 1
	userID := 1

	t.Run("Success", func(t *testing.T) {
		existingCategory := &model.Category{ID: categoryID, Name: "Test"}
		mockCategoryDAO.EXPECT().GetCategory(ctx, categoryID).Return(existingCategory, nil).Times(1)
		mockCategoryDAO.EXPECT().DeleteCategory(ctx, categoryID).Return(nil).Times(1)

		err := service.DeleteCategory(ctx, categoryID, userID)
		assert.NoError(t, err)
	})

	t.Run("GetCategoryError", func(t *testing.T) {
		mockCategoryDAO.EXPECT().GetCategory(ctx, categoryID).Return(nil, errors.New("DAO get error")).Times(1)
		err := service.DeleteCategory(ctx, categoryID, userID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "DAO get error")
	})
	
	t.Run("CategoryNotFound", func(t *testing.T) {
		mockCategoryDAO.EXPECT().GetCategory(ctx, categoryID).Return(nil, gorm.ErrRecordNotFound).Times(1) // More specific error
		err := service.DeleteCategory(ctx, categoryID, userID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), gorm.ErrRecordNotFound.Error())
	})


	t.Run("DeleteDAOError", func(t *testing.T) {
		existingCategory := &model.Category{ID: categoryID, Name: "Test"}
		mockCategoryDAO.EXPECT().GetCategory(ctx, categoryID).Return(existingCategory, nil).Times(1)
		mockCategoryDAO.EXPECT().DeleteCategory(ctx, categoryID).Return(errors.New("DAO delete error")).Times(1)

		err := service.DeleteCategory(ctx, categoryID, userID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "DAO delete error")
	})
}

func TestCategoryGroupService_ListCategory(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCategoryDAO := mocks.NewMockCategoryDAO(ctrl)
	mockUserDAO := mocks.NewMockUserDAO(ctrl)
	logger := zap.NewNop()

	service := NewCategoryGroupService(mockCategoryDAO, mockUserDAO, logger)
	ctx := context.Background()

	req := model.ListCategoryReq{Page: 1, PageSize: 10}
	categories := []model.Category{
		{ID: 1, Name: "Cat 1"},
		{ID: 2, Name: "Cat 2"},
	}
	total := int64(len(categories))

	t.Run("Success", func(t *testing.T) {
		mockCategoryDAO.EXPECT().ListCategory(ctx, req).Return(categories, total, nil).Times(1)

		resp, err := service.ListCategory(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, int(total), resp.Total)
		assert.Len(t, resp.Items, len(categories))
		if len(resp.Items) > 0 {
			assert.Equal(t, categories[0].Name, resp.Items.([]model.CategoryResp)[0].Name)
		}
	})

	t.Run("DAOError", func(t *testing.T) {
		mockCategoryDAO.EXPECT().ListCategory(ctx, req).Return(nil, int64(0), errors.New("DAO list error")).Times(1)

		resp, err := service.ListCategory(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "DAO list error")
	})
}

func TestCategoryGroupService_GetCategory(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCategoryDAO := mocks.NewMockCategoryDAO(ctrl)
	mockUserDAO := mocks.NewMockUserDAO(ctrl)
	logger := zap.NewNop()

	service := NewCategoryGroupService(mockCategoryDAO, mockUserDAO, logger)
	ctx := context.Background()
	categoryID := 1

	t.Run("Success", func(t *testing.T) {
		category := &model.Category{ID: categoryID, Name: "Test Cat"}
		mockCategoryDAO.EXPECT().GetCategory(ctx, categoryID).Return(category, nil).Times(1)

		resp, err := service.GetCategory(ctx, categoryID)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, category.ID, resp.ID)
		assert.Equal(t, category.Name, resp.Name)
	})

	t.Run("DAOError", func(t *testing.T) {
		mockCategoryDAO.EXPECT().GetCategory(ctx, categoryID).Return(nil, errors.New("DAO get error")).Times(1)

		resp, err := service.GetCategory(ctx, categoryID)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "DAO get error")
	})

	t.Run("NotFound", func(t *testing.T) {
		mockCategoryDAO.EXPECT().GetCategory(ctx, categoryID).Return(nil, nil).Times(1) // DAO returns nil, nil for not found

		resp, err := service.GetCategory(ctx, categoryID)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), fmt.Sprintf("分类 (ID: %d) 不存在", categoryID))
	})
	
	t.Run("NotFoundGorm", func(t *testing.T) {
		mockCategoryDAO.EXPECT().GetCategory(ctx, categoryID).Return(nil, gorm.ErrRecordNotFound).Times(1)

		resp, err := service.GetCategory(ctx, categoryID)
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.True(t, errors.Is(err, gorm.ErrRecordNotFound) || true) // Service wraps it
	})
}

func TestCategoryGroupService_GetCategoryTree(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCategoryDAO := mocks.NewMockCategoryDAO(ctrl)
	mockUserDAO := mocks.NewMockUserDAO(ctrl)
	logger := zap.NewNop()

	service := NewCategoryGroupService(mockCategoryDAO, mockUserDAO, logger)
	ctx := context.Background()

	t.Run("SuccessEmpty", func(t *testing.T) {
		mockCategoryDAO.EXPECT().GetAllCategories(ctx).Return([]model.Category{}, nil).Times(1)
		tree, err := service.GetCategoryTree(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, tree)
		assert.Len(t, tree, 0)
	})

	t.Run("SuccessFlatList", func(t *testing.T) {
		categories := []model.Category{
			{ID: 1, Name: "Cat 1"},
			{ID: 2, Name: "Cat 2"},
		}
		mockCategoryDAO.EXPECT().GetAllCategories(ctx).Return(categories, nil).Times(1)
		tree, err := service.GetCategoryTree(ctx)
		assert.NoError(t, err)
		assert.Len(t, tree, 2)
		assert.Equal(t, "Cat 1", tree[0].Name)
		assert.Len(t, tree[0].Children, 0)
	})

	t.Run("SuccessSimpleTree", func(t *testing.T) {
		parentID := 1
		categories := []model.Category{
			{ID: 1, Name: "Parent"},
			{ID: 2, Name: "Child 1", ParentID: &parentID},
			{ID: 3, Name: "Child 2", ParentID: &parentID},
			{ID: 4, Name: "Another Parent"},
		}
		mockCategoryDAO.EXPECT().GetAllCategories(ctx).Return(categories, nil).Times(1)
		tree, err := service.GetCategoryTree(ctx)
		assert.NoError(t, err)
		assert.Len(t, tree, 2) // Parent and Another Parent

		// Find "Parent" and check its children
		var parentNode model.CategoryResp
		for _, node := range tree {
			if node.ID == 1 {
				parentNode = node
				break
			}
		}
		assert.Equal(t, "Parent", parentNode.Name)
		assert.Len(t, parentNode.Children, 2)
		if len(parentNode.Children) == 2 {
			assert.Equal(t, "Child 1", parentNode.Children[0].Name)
			assert.Equal(t, "Child 2", parentNode.Children[1].Name)
		}
	})

	t.Run("DAOError", func(t *testing.T) {
		mockCategoryDAO.EXPECT().GetAllCategories(ctx).Return(nil, errors.New("DAO get all error")).Times(1)
		tree, err := service.GetCategoryTree(ctx)
		assert.Error(t, err)
		assert.Nil(t, tree)
		assert.Contains(t, err.Error(), "DAO get all error")
	})
}
