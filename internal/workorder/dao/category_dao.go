package dao

import (
	"context"
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// CategoryDAO 定义了分类的数据访问对象接口
type CategoryDAO interface {
	CreateCategory(ctx context.Context, category *model.Category) (*model.Category, error)
	UpdateCategory(ctx context.Context, category *model.Category) (*model.Category, error)
	DeleteCategory(ctx context.Context, id int) error
	ListCategory(ctx context.Context, req model.ListCategoryReq) ([]model.Category, int64, error)
	GetCategory(ctx context.Context, id int) (*model.Category, error)
	GetAllCategories(ctx context.Context) ([]model.Category, error)
}

type categoryDAO struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewCategoryDAO 创建一个新的 CategoryDAO 实例
func NewCategoryDAO(db *gorm.DB, logger *zap.Logger) CategoryDAO {
	return &categoryDAO{
		db:     db,
		logger: logger,
	}
}

// CreateCategory 创建分类
func (dao *categoryDAO) CreateCategory(ctx context.Context, category *model.Category) (*model.Category, error) {
	dao.logger.Debug("开始创建分类 (DAO)", zap.Any("category", category))
	if err := dao.db.WithContext(ctx).Create(category).Error; err != nil {
		dao.logger.Error("创建分类失败 (DAO)", zap.Error(err), zap.Any("category", category))
		return nil, fmt.Errorf("创建分类失败: %w", err)
	}
	dao.logger.Debug("分类创建成功 (DAO)", zap.Int("id", category.ID))
	return category, nil
}

// UpdateCategory 更新分类
func (dao *categoryDAO) UpdateCategory(ctx context.Context, category *model.Category) (*model.Category, error) {
	dao.logger.Debug("开始更新分类 (DAO)", zap.Any("category", category))
	// 使用 map 更新以避免 GORM 的零值问题，仅更新传入的字段
	// 但这里 category 对象通常由 service 构造，包含了要更新的字段
	// GORM 的 Updates 方法会自动忽略零值字段，除非使用 Select 或 Map
	// 如果希望即便传入零值也更新，需要使用 Select(...)
	// result := dao.db.WithContext(ctx).Model(&model.Category{}).Where("id = ?", category.ID).Updates(category)
	// 为了确保所有在 category struct 中设置的字段（包括零值如 status=0）都被更新，
	// 可以使用 Save (如果对象包含主键) 或明确指定要更新的列。
	// Save 会更新所有字段，如果 category 是从数据库读取后修改的，这通常是安全的。
	// 如果 category 是一个新构造的仅包含部分更新字段的对象，则 Updates 更安全。
	// 假设 service 层会先获取 category，然后修改，再调用此 UpdateCategory。
	// 或者，service 层构造一个包含所有需要更新字段的 category 对象。
	
	// 确保只更新传入的字段，而不是整个对象，以避免意外重置
    // 如果 category.ParentID 为 nil 且数据库中 ParentID 有值，直接用 Updates(category) 会将数据库中 ParentID 置为 NULL
    // 因此，最好使用 Map 或者明确指定要更新的字段
    updateData := map[string]interface{}{
        "name":        category.Name,
        "parent_id":   category.ParentID, // GORM 会处理 *int 类型，如果为 nil 则设为 NULL
        "icon":        category.Icon,
        "sort_order":  category.SortOrder,
        "status":      category.Status,
        "description": category.Description,
    }

	result := dao.db.WithContext(ctx).Model(&model.Category{}).Where("id = ?", category.ID).Updates(updateData)
	if result.Error != nil {
		dao.logger.Error("更新分类失败 (DAO)", zap.Error(result.Error), zap.Int("id", category.ID))
		return nil, fmt.Errorf("更新分类 (ID: %d) 失败: %w", category.ID, result.Error)
	}
	if result.RowsAffected == 0 {
		dao.logger.Warn("更新分类：未找到记录或无需更新 (DAO)", zap.Int("id", category.ID))
		// Consider it not an error if RowsAffected is 0, as the data might be the same
		// However, typically we expect an update to affect rows if data is different.
		// For consistency with GetCategory, let's fetch the updated record.
	}
	// 获取更新后的记录
	updatedCategory, err := dao.GetCategory(ctx, category.ID)
	if err != nil {
		dao.logger.Error("更新分类后获取记录失败 (DAO)", zap.Error(err), zap.Int("id", category.ID))
		return nil, err
	}
	dao.logger.Debug("分类更新成功 (DAO)", zap.Int("id", category.ID))
	return updatedCategory, nil
}

// DeleteCategory 删除分类 (软删除)
func (dao *categoryDAO) DeleteCategory(ctx context.Context, id int) error {
	dao.logger.Debug("开始删除分类 (DAO)", zap.Int("id", id))
	if err := dao.db.WithContext(ctx).Delete(&model.Category{}, id).Error; err != nil {
		dao.logger.Error("删除分类失败 (DAO)", zap.Error(err), zap.Int("id", id))
		return fmt.Errorf("删除分类 (ID: %d) 失败: %w", id, err)
	}
	dao.logger.Debug("分类删除成功 (DAO)", zap.Int("id", id))
	return nil
}

// ListCategory 列出分类 (分页)
func (dao *categoryDAO) ListCategory(ctx context.Context, req model.ListCategoryReq) ([]model.Category, int64, error) {
	dao.logger.Debug("开始列出分类 (DAO)", zap.Any("request", req))
	var categories []model.Category
	var total int64

	db := dao.db.WithContext(ctx).Model(&model.Category{})

	// 根据名称搜索
	if req.Name != "" { // Assuming ListCategoryReq has Name for search, not Search from ListReq
		db = db.Where("name LIKE ?", "%"+req.Name+"%")
	}

	// 根据状态筛选
	if req.Status != nil { // Assuming Status is *int8
		db = db.Where("status = ?", *req.Status)
	}
	
	// 计算总数
	if err := db.Count(&total).Error; err != nil {
		dao.logger.Error("列出分类失败 - 计算总数错误 (DAO)", zap.Error(err))
		return nil, 0, fmt.Errorf("计算分类总数失败: %w", err)
	}

	// 分页
	offset := (req.Page - 1) * req.PageSize // Page and PageSize are from model.ListCategoryReq
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10 // Default page size
	}
	
	if err := db.Offset(offset).Limit(req.PageSize).Order("sort_order ASC, id ASC").Find(&categories).Error; err != nil {
		dao.logger.Error("列出分类失败 - 查询错误 (DAO)", zap.Error(err))
		return nil, 0, fmt.Errorf("查询分类列表失败: %w", err)
	}

	dao.logger.Debug("分类列表获取成功 (DAO)", zap.Int("count", len(categories)), zap.Int64("total", total))
	return categories, total, nil
}

// GetCategory 获取单个分类详情
func (dao *categoryDAO) GetCategory(ctx context.Context, id int) (*model.Category, error) {
	dao.logger.Debug("开始获取分类详情 (DAO)", zap.Int("id", id))
	var category model.Category
	if err := dao.db.WithContext(ctx).First(&category, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			dao.logger.Warn("获取分类详情：分类不存在 (DAO)", zap.Int("id", id))
			return nil, nil // Or return specific error like ErrRecordNotFound
		}
		dao.logger.Error("获取分类详情失败 (DAO)", zap.Error(err), zap.Int("id", id))
		return nil, fmt.Errorf("获取分类 (ID: %d) 失败: %w", id, err)
	}
	dao.logger.Debug("分类详情获取成功 (DAO)", zap.Any("category", category))
	return &category, nil
}

// GetAllCategories 获取所有分类 (用于构建树等)
func (dao *categoryDAO) GetAllCategories(ctx context.Context) ([]model.Category, error) {
	dao.logger.Debug("开始获取所有分类 (DAO)")
	var categories []model.Category
	if err := dao.db.WithContext(ctx).Order("sort_order ASC, id ASC").Find(&categories).Error; err != nil {
		dao.logger.Error("获取所有分类失败 (DAO)", zap.Error(err))
		return nil, fmt.Errorf("获取所有分类失败: %w", err)
	}
	dao.logger.Debug("所有分类获取成功 (DAO)", zap.Int("count", len(categories)))
	return categories, nil
}
