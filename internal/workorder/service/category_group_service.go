package service

import (
	"context"
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	userdao "github.com/GoSimplicity/AI-CloudOps/internal/user/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
	"go.uber.org/zap"
)

// CategoryGroupService 定义了分类管理的服务接口
type CategoryGroupService interface {
	// CreateCategory 创建分类
	CreateCategory(ctx context.Context, req *model.CreateCategoryReq, creatorID int, creatorName string) (*model.CategoryResp, error)
	// UpdateCategory 更新分类
	UpdateCategory(ctx context.Context, req *model.UpdateCategoryReq, userID int) (*model.CategoryResp, error)
	// DeleteCategory 删除分类
	DeleteCategory(ctx context.Context, id int, userID int) error
	// ListCategory 列出分类 (分页)
	ListCategory(ctx context.Context, req model.ListCategoryReq) (*model.ListResponse, error)
	// GetCategory 获取单个分类详情
	GetCategory(ctx context.Context, id int) (*model.CategoryResp, error)
	// GetCategoryTree 获取分类树结构
	GetCategoryTree(ctx context.Context) ([]model.CategoryResp, error)
}

// categoryGroupService实现了CategoryGroupService接口
type categoryGroupService struct {
	categoryDAO dao.CategoryDAO // 数据访问对象，用于分类的CURD
	userDAO     userdao.UserDAO // 用户数据访问对象，可能用于获取创建者/更新者信息
	logger      *zap.Logger     // 日志记录器
}

// NewCategoryGroupService 创建一个新的CategoryGroupService实例
func NewCategoryGroupService(categoryDAO dao.CategoryDAO, userDAO userdao.UserDAO, logger *zap.Logger) CategoryGroupService {
	return &categoryGroupService{
		categoryDAO: categoryDAO,
		userDAO:     userDAO,
		logger:      logger,
	}
}

// convertToCategoryResp 将 model.Category 转换为 model.CategoryResp
func convertToCategoryResp(category *model.Category) *model.CategoryResp {
	if category == nil {
		return nil
	}
	return &model.CategoryResp{
		ID:          category.ID,
		Name:        category.Name,
		ParentID:    category.ParentID,
		Icon:        category.Icon,
		SortOrder:   category.SortOrder,
		Status:      category.Status,
		Description: category.Description,
		CreatedAt:   category.CreatedAt,
		UpdatedAt:   category.UpdatedAt,
		Children:    []model.CategoryResp{}, // 初始化Children，避免nil
	}
}

// convertToCategoryRespList 将 []model.Category 转换为 []model.CategoryResp
func convertToCategoryRespList(categories []model.Category) []model.CategoryResp {
	resps := make([]model.CategoryResp, 0, len(categories))
	for _, category := range categories {
		resps = append(resps, *convertToCategoryResp(&category))
	}
	return resps
}

// CreateCategory 创建分类的实现
func (s *categoryGroupService) CreateCategory(ctx context.Context, req *model.CreateCategoryReq, creatorID int, creatorName string) (*model.CategoryResp, error) {
	s.logger.Info("开始创建分类", zap.String("name", req.Name), zap.Int("creatorID", creatorID))

	category := &model.Category{
		Name:        req.Name,
		ParentID:    req.ParentID,
		Icon:        req.Icon,
		SortOrder:   req.SortOrder,
		Description: req.Description,
		Status:      1, // 默认为启用状态
		// CreatedAt and UpdatedAt will be handled by GORM
		// DeletedAt is for soft delete, not set on create
	}

	// TODO: The model.Category does not have CreatorID or CreatorName fields.
	// If auditing or tracking creator is needed, the model.Category itself should be updated.
	// For now, creatorID and creatorName are logged but not stored directly in the category model.

	err := s.categoryDAO.CreateCategory(ctx, category)
	if err != nil {
		s.logger.Error("创建分类失败", zap.Error(err), zap.String("name", req.Name))
		return nil, fmt.Errorf("创建分类 '%s' 失败: %w", req.Name, err)
	}

	s.logger.Info("分类创建成功", zap.Int("id", category.ID), zap.String("name", category.Name))
	return convertToCategoryResp(category), nil
}

// UpdateCategory 更新分类的实现
func (s *categoryGroupService) UpdateCategory(ctx context.Context, req *model.UpdateCategoryReq, userID int) (*model.CategoryResp, error) {
	s.logger.Info("开始更新分类", zap.Int("id", req.ID), zap.Int("userID", userID))

	// 检查分类是否存在
	existingCategory, err := s.categoryDAO.GetCategory(ctx, req.ID)
	if err != nil {
		s.logger.Error("更新分类失败：获取分类信息失败", zap.Error(err), zap.Int("id", req.ID))
		return nil, fmt.Errorf("获取分类信息失败 (ID: %d): %w", req.ID, err)
	}
	if existingCategory == nil {
		s.logger.Warn("更新分类失败：分类不存在", zap.Int("id", req.ID))
		return nil, fmt.Errorf("分类 (ID: %d) 不存在", req.ID)
	}

	// TODO: Add permission check here if needed: does userID have permission to update this category?

	category := &model.Category{
		ID:          req.ID,
		Name:        req.Name,
		ParentID:    req.ParentID,
		Icon:        req.Icon,
		SortOrder:   req.SortOrder,
		Description: req.Description,
		Status:      req.Status,
	}

	err = s.categoryDAO.UpdateCategory(ctx, category)
	if err != nil {
		s.logger.Error("更新分类失败", zap.Error(err), zap.Int("id", req.ID))
		return nil, fmt.Errorf("更新分类 (ID: %d) 失败: %w", req.ID, err)
	}

	// 获取更新后的完整信息
	updatedCategory, err := s.categoryDAO.GetCategory(ctx, req.ID)
	if err != nil {
		s.logger.Error("更新分类后获取最新数据失败", zap.Error(err), zap.Int("id", req.ID))
		return nil, fmt.Errorf("获取更新后的分类信息失败 (ID: %d): %w", req.ID, err)
	}

	s.logger.Info("分类更新成功", zap.Int("id", updatedCategory.ID), zap.String("name", updatedCategory.Name))
	return convertToCategoryResp(updatedCategory), nil
}

// DeleteCategory 删除分类的实现
func (s *categoryGroupService) DeleteCategory(ctx context.Context, id int, userID int) error {
	s.logger.Info("开始删除分类", zap.Int("id", id), zap.Int("userID", userID))

	// 检查分类是否存在
	existingCategory, err := s.categoryDAO.GetCategory(ctx, id)
	if err != nil {
		s.logger.Error("删除分类失败：获取分类信息失败", zap.Error(err), zap.Int("id", id))
		return fmt.Errorf("获取分类信息失败 (ID: %d): %w", id, err)
	}
	if existingCategory == nil {
		s.logger.Warn("删除分类失败：分类不存在", zap.Int("id", id))
		return fmt.Errorf("分类 (ID: %d) 不存在", id)
	}
	
	// TODO: Add permission check here.
	// TODO: Add check for child categories - should not delete if children exist, or handle recursively.

	err = s.categoryDAO.DeleteCategory(ctx, id)
	if err != nil {
		s.logger.Error("删除分类失败", zap.Error(err), zap.Int("id", id))
		return fmt.Errorf("删除分类 (ID: %d) 失败: %w", id, err)
	}

	s.logger.Info("分类删除成功", zap.Int("id", id))
	return nil
}

// ListCategory 列出分类的实现
func (s *categoryGroupService) ListCategory(ctx context.Context, req model.ListCategoryReq) (*model.ListResponse, error) {
	s.logger.Info("开始列出分类", zap.Any("request", req))

	categories, total, err := s.categoryDAO.ListCategory(ctx, req)
	if err != nil {
		s.logger.Error("列出分类失败", zap.Error(err))
		return nil, fmt.Errorf("列出分类失败: %w", err)
	}

	categoryResps := convertToCategoryRespList(categories)

	s.logger.Info("分类列表获取成功", zap.Int("count", len(categoryResps)), zap.Int64("total", total))
	return &model.ListResponse{
		Total: int(total), // model.ListResponse.Total is int
		Items: categoryResps,
	}, nil
}

// GetCategory 获取单个分类详情的实现
func (s *categoryGroupService) GetCategory(ctx context.Context, id int) (*model.CategoryResp, error) {
	s.logger.Info("开始获取分类详情", zap.Int("id", id))

	category, err := s.categoryDAO.GetCategory(ctx, id)
	if err != nil {
		s.logger.Error("获取分类详情失败", zap.Error(err), zap.Int("id", id))
		return nil, fmt.Errorf("获取分类详情 (ID: %d) 失败: %w", id, err)
	}
	if category == nil {
		s.logger.Warn("获取分类详情失败：分类不存在", zap.Int("id", id))
		return nil, fmt.Errorf("分类 (ID: %d) 不存在", id)
	}

	s.logger.Info("分类详情获取成功", zap.Int("id", category.ID))
	return convertToCategoryResp(category), nil
}

// GetCategoryTree 获取分类树结构的实现
func (s *categoryGroupService) GetCategoryTree(ctx context.Context) ([]model.CategoryResp, error) {
	s.logger.Info("开始获取分类树")

	allCategories, err := s.categoryDAO.GetAllCategories(ctx) // Assuming this DAO method will be created
	if err != nil {
		s.logger.Error("获取所有分类失败（用于构建树）", zap.Error(err))
		return nil, fmt.Errorf("获取所有分类失败: %w", err)
	}

	categoryMap := make(map[int]*model.CategoryResp)
	rootCategories := make([]model.CategoryResp, 0)

	// 将所有分类转换为CategoryResp并存入map
	for _, category := range allCategories {
		// Need to handle the loop variable correctly for pointers
		catCopy := category 
		categoryMap[category.ID] = convertToCategoryResp(&catCopy)
	}

	// 构建树结构
	for _, category := range allCategories {
		respNode, ok := categoryMap[category.ID]
		if !ok { continue } // Should not happen if map is populated correctly

		if category.ParentID == nil || *category.ParentID == 0 {
			rootCategories = append(rootCategories, *respNode)
		} else {
			parentNode, found := categoryMap[*category.ParentID]
			if found {
				parentNode.Children = append(parentNode.Children, *respNode)
			} else {
				// Orphan node, could append to root or log as warning
				s.logger.Warn("发现孤立分类节点（父节点未找到）", zap.Int("categoryID", category.ID), zap.Intp("parentID", category.ParentID))
				// Optionally, add orphans to root level:
				// rootCategories = append(rootCategories, *respNode)
			}
		}
	}

	s.logger.Info("分类树获取成功", zap.Int("rootCategoriesCount", len(rootCategories)))
	return rootCategories, nil
}

// Placeholder for dao.CategoryDAO and its methods.
// These will be implemented in a subsequent step.
// For example, dao.CategoryDAO might look like:
/*
package dao

import (
	"context"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

type CategoryDAO interface {
	CreateCategory(ctx context.Context, category *model.Category) error
	UpdateCategory(ctx context.Context, category *model.Category) error
	DeleteCategory(ctx context.Context, id int) error
	ListCategory(ctx context.Context, req model.ListCategoryReq) ([]model.Category, int64, error)
	GetCategory(ctx context.Context, id int) (*model.Category, error)
	GetAllCategories(ctx context.Context) ([]model.Category, error) // New method for GetCategoryTree
}
*/
// Placeholder for userdao.UserDAO
/*
package dao

import (
	"context"
	"github.com/GoSimplicity/AI-CloudOps/internal/model" // Assuming User model is in model package
)

type UserDAO interface {
    GetUserByID(ctx context.Context, id int) (*model.User, error) // Example method
}
*/

// End of category_group_service.go
