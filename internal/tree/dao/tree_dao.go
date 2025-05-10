/*
 * MIT License
 *
 * Copyright (c) 2024 Bamboo
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 *
 */

package dao

import (
	"context"
	"errors"
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type contextKey string

const txKey contextKey = "transaction"

type TreeDAO interface {
	GetTreeList(ctx context.Context, req *model.TreeNodeListReq) ([]*model.TreeNode, error)
	GetNode(ctx context.Context, id int) (*model.TreeNode, error)
	GetChildNodes(ctx context.Context, parentId int) ([]*model.TreeNode, error)
	GetTreeStatistics(ctx context.Context) (*model.TreeStatisticsResp, error)
	GetNodeLevel(ctx context.Context, nodeId int) (int, error)

	CreateNode(ctx context.Context, node *model.TreeNode) error
	UpdateNode(ctx context.Context, node *model.TreeNode) error
	DeleteNode(ctx context.Context, id int) error

	GetNodeResources(ctx context.Context, nodeId int) ([]*model.TreeNodeResourceResp, error)
	BindResource(ctx context.Context, nodeId int, resourceType string, resourceIds []string) error
	UnbindResource(ctx context.Context, nodeId int, resourceType string, resourceId string) error
	GetResourceTypes(ctx context.Context) ([]string, error)

	GetNodeMembers(ctx context.Context, nodeId int, userId int, memberType string) ([]*model.User, error)
	AddNodeMember(ctx context.Context, nodeId int, userId int, memberType string) error
	RemoveNodeMember(ctx context.Context, nodeId int, userId int, memberType string) error

	Transaction(ctx context.Context, fn func(ctx context.Context) error) error
}

type treeDAO struct {
	logger *zap.Logger
	db     *gorm.DB
}

func NewTreeDAO(logger *zap.Logger, db *gorm.DB) TreeDAO {
	return &treeDAO{
		logger: logger,
		db:     db,
	}
}

// checkNodeExists 检查节点是否存在的辅助函数
func (t *treeDAO) checkNodeExists(ctx context.Context, id int) error {
	var count int64
	if err := t.db.WithContext(ctx).Model(&model.TreeNode{}).Where("id = ?", id).Count(&count).Error; err != nil {
		t.logger.Error("检查节点存在性失败", zap.Int("nodeId", id), zap.Error(err))
		return fmt.Errorf("检查节点失败: %w", err)
	}

	if count == 0 {
		return errors.New("节点不存在")
	}
	return nil
}

// AddNodeMember 添加节点成员
func (t *treeDAO) AddNodeMember(ctx context.Context, nodeId int, userId int, memberType string) error {
	// 检查节点是否存在
	if err := t.checkNodeExists(ctx, nodeId); err != nil {
		return err
	}

	// 检查用户是否存在
	var userCount int64
	if err := t.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", userId).Count(&userCount).Error; err != nil {
		t.logger.Error("检查用户存在性失败", zap.Int("userId", userId), zap.Error(err))
		return fmt.Errorf("检查用户失败: %w", err)
	}

	if userCount == 0 {
		return errors.New("用户不存在")
	}

	// 根据成员类型添加关系
	switch memberType {
	case "admin":
		// 创建管理员关系
		admin := model.TreeNodeAdmin{
			TreeNodeID: nodeId,
			UserID:     userId,
		}
		result := t.db.WithContext(ctx).Where("tree_node_id = ? AND user_id = ?", nodeId, userId).First(&model.TreeNodeAdmin{})
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				// 记录不存在，创建新记录
				if err := t.db.WithContext(ctx).Create(&admin).Error; err != nil {
					t.logger.Error("添加节点管理员失败", zap.Int("nodeId", nodeId), zap.Int("userId", userId), zap.Error(err))
					return fmt.Errorf("添加节点管理员失败: %w", err)
				}
			} else {
				t.logger.Error("查询节点管理员失败", zap.Int("nodeId", nodeId), zap.Int("userId", userId), zap.Error(result.Error))
				return fmt.Errorf("查询节点管理员失败: %w", result.Error)
			}
		}
	case "member":
		// 创建成员关系
		member := model.TreeNodeMember{
			TreeNodeID: nodeId,
			UserID:     userId,
		}
		result := t.db.WithContext(ctx).Where("tree_node_id = ? AND user_id = ?", nodeId, userId).First(&model.TreeNodeMember{})
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				// 记录不存在，创建新记录
				if err := t.db.WithContext(ctx).Create(&member).Error; err != nil {
					t.logger.Error("添加节点成员失败", zap.Int("nodeId", nodeId), zap.Int("userId", userId), zap.Error(err))
					return fmt.Errorf("添加节点成员失败: %w", err)
				}
			} else {
				t.logger.Error("查询节点成员失败", zap.Int("nodeId", nodeId), zap.Int("userId", userId), zap.Error(result.Error))
				return fmt.Errorf("查询节点成员失败: %w", result.Error)
			}
		}
	default:
		return errors.New("无效的成员类型")
	}

	return nil
}

// BindResource 绑定资源到节点
func (t *treeDAO) BindResource(ctx context.Context, nodeId int, resourceType string, resourceIds []string) error {
	// 暂时留空
	return nil
}

// CreateNode 创建节点
func (t *treeDAO) CreateNode(ctx context.Context, node *model.TreeNode) error {
	// 检查父节点是否存在
	if node.ParentID != 0 {
		if err := t.checkNodeExists(ctx, node.ParentID); err != nil {
			return errors.New("父节点不存在")
		}
	}

	// 检查节点名称是否重复
	var count int64
	if err := t.db.WithContext(ctx).Model(&model.TreeNode{}).Where("name = ? AND parent_id = ?", node.Name, node.ParentID).Count(&count).Error; err != nil {
		t.logger.Error("检查节点名称失败", zap.String("name", node.Name), zap.Error(err))
		return fmt.Errorf("检查节点名称失败: %w", err)
	}

	if count > 0 {
		return errors.New("同级节点下已存在相同名称的节点")
	}

	// 创建节点
	if err := t.db.WithContext(ctx).Create(node).Error; err != nil {
		t.logger.Error("创建节点失败", zap.String("name", node.Name), zap.Error(err))
		return fmt.Errorf("创建节点失败: %w", err)
	}

	return nil
}

// DeleteNode 删除节点
func (t *treeDAO) DeleteNode(ctx context.Context, id int) error {
	// 检查节点是否存在
	if err := t.checkNodeExists(ctx, id); err != nil {
		return err
	}

	// 检查是否有子节点
	var childCount int64
	if err := t.db.WithContext(ctx).Model(&model.TreeNode{}).Where("parent_id = ?", id).Count(&childCount).Error; err != nil {
		t.logger.Error("检查子节点失败", zap.Int("id", id), zap.Error(err))
		return fmt.Errorf("检查子节点失败: %w", err)
	}

	if childCount > 0 {
		return errors.New("该节点下存在子节点，无法删除")
	}

	// 使用事务确保操作的原子性
	return t.Transaction(ctx, func(txCtx context.Context) error {
		tx := t.db.WithContext(txCtx)

		// 删除节点成员关系
		if err := tx.Where("tree_node_id = ?", id).Delete(&model.TreeNodeAdmin{}).Error; err != nil {
			t.logger.Error("删除管理员关系失败", zap.Int("id", id), zap.Error(err))
			return fmt.Errorf("删除管理员关系失败: %w", err)
		}

		if err := tx.Where("tree_node_id = ?", id).Delete(&model.TreeNodeMember{}).Error; err != nil {
			t.logger.Error("删除成员关系失败", zap.Int("id", id), zap.Error(err))
			return fmt.Errorf("删除成员关系失败: %w", err)
		}

		// 删除节点
		if err := tx.Delete(&model.TreeNode{}, id).Error; err != nil {
			t.logger.Error("删除节点失败", zap.Int("id", id), zap.Error(err))
			return fmt.Errorf("删除节点失败: %w", err)
		}

		return nil
	})
}

// GetChildNodes 获取子节点列表
func (t *treeDAO) GetChildNodes(ctx context.Context, parentId int) ([]*model.TreeNode, error) {
	var nodes []*model.TreeNode
	if err := t.db.WithContext(ctx).Where("parent_id = ?", parentId).Find(&nodes).Error; err != nil {
		t.logger.Error("获取子节点失败", zap.Int("parentId", parentId), zap.Error(err))
		return nil, fmt.Errorf("获取子节点失败: %w", err)
	}

	// 批量获取子节点的子节点数
	if len(nodes) > 0 {
		nodeIds := make([]int, 0, len(nodes))
		for _, node := range nodes {
			nodeIds = append(nodeIds, node.ID)
		}

		// 使用一次查询获取所有子节点的子节点数
		type ChildCount struct {
			ParentID int
			Count    int
		}
		var childCounts []ChildCount

		if err := t.db.WithContext(ctx).Model(&model.TreeNode{}).
			Select("parent_id as parent_id, count(*) as count").
			Where("parent_id IN ?", nodeIds).
			Group("parent_id").
			Scan(&childCounts).Error; err != nil {
			t.logger.Error("获取子节点数量失败", zap.Error(err))
			return nil, fmt.Errorf("获取子节点数量失败: %w", err)
		}

		// 构建映射以便快速查找
		countMap := make(map[int]int)
		for _, cc := range childCounts {
			countMap[cc.ParentID] = cc.Count
		}

		// 更新节点的子节点数
		for _, node := range nodes {
			node.ChildCount = countMap[node.ID]
		}
	}

	return nodes, nil
}

// GetNode 获取节点详情
func (t *treeDAO) GetNode(ctx context.Context, id int) (*model.TreeNode, error) {
	var node model.TreeNode
	if err := t.db.WithContext(ctx).First(&node, id).Error; err != nil {
		t.logger.Error("节点不存在", zap.Int("id", id), zap.Error(err))
		return nil, errors.New("节点不存在")
	}

	// 获取父节点名称
	if node.ParentID != 0 {
		var parentNode model.TreeNode
		if err := t.db.WithContext(ctx).Select("name").First(&parentNode, node.ParentID).Error; err == nil {
			node.ParentName = parentNode.Name
		} else {
			t.logger.Warn("获取父节点名称失败", zap.Int("parentId", node.ParentID), zap.Error(err))
		}
	}

	// 获取子节点数量
	var childCount int64
	if err := t.db.WithContext(ctx).Model(&model.TreeNode{}).Where("parent_id = ?", id).Count(&childCount).Error; err != nil {
		t.logger.Error("获取子节点数量失败", zap.Int("id", id), zap.Error(err))
		return nil, fmt.Errorf("获取子节点数量失败: %w", err)
	}
	node.ChildCount = int(childCount)

	return &node, nil
}

// GetNodeMembers 获取节点成员列表
func (t *treeDAO) GetNodeMembers(ctx context.Context, nodeId int, userId int, memberType string) ([]*model.User, error) {
	// 根据memberType不同查不同的关联表
	var users []*model.User
	query := t.db.WithContext(ctx).Model(&model.User{})

	if memberType == "admin" {
		// 查询管理员
		query = query.Joins("JOIN tree_node_admins ON tree_node_admins.user_id = users.id").
			Where("tree_node_admins.tree_node_id = ?", nodeId)
	} else if memberType == "member" {
		// 查询普通成员
		query = query.Joins("JOIN tree_node_members ON tree_node_members.user_id = users.id").
			Where("tree_node_members.tree_node_id = ?", nodeId)
	} else {
		// 查询所有成员（管理员和普通成员）
		subQuery1 := t.db.Model(&model.TreeNodeAdmin{}).Select("user_id").Where("tree_node_id = ?", nodeId)
		subQuery2 := t.db.Model(&model.TreeNodeMember{}).Select("user_id").Where("tree_node_id = ?", nodeId)
		query = query.Where("users.id IN (?) OR users.id IN (?)", subQuery1, subQuery2)
	}

	// 如果指定了用户ID，则进一步过滤
	if userId > 0 {
		query = query.Where("users.id = ?", userId)
	}

	if err := query.Find(&users).Error; err != nil {
		t.logger.Error("获取节点成员失败",
			zap.Int("nodeId", nodeId),
			zap.String("memberType", memberType),
			zap.Error(err))
		return nil, fmt.Errorf("获取节点成员失败: %w", err)
	}

	return users, nil
}

// GetNodeResources 获取节点绑定的资源列表
func (t *treeDAO) GetNodeResources(ctx context.Context, nodeId int) ([]*model.TreeNodeResourceResp, error) {
	// 暂时留空
	return nil, nil
}

// GetResourceTypes 获取所有资源类型
func (t *treeDAO) GetResourceTypes(ctx context.Context) ([]string, error) {
	// 暂时留空
	return nil, nil
}

// GetTreeList 获取树节点列表
func (t *treeDAO) GetTreeList(ctx context.Context, req *model.TreeNodeListReq) ([]*model.TreeNode, error) {
	var nodes []*model.TreeNode
	query := t.db.WithContext(ctx)

	// 根据请求参数过滤
	if req.Level > 0 {
		query = query.Where("level = ?", req.Level)
	}

	// 执行查询
	if err := query.Find(&nodes).Error; err != nil {
		t.logger.Error("获取树节点列表失败", zap.Error(err))
		return nil, fmt.Errorf("获取树节点列表失败: %w", err)
	}

	if len(nodes) == 0 {
		return nodes, nil
	}

	// 获取所有节点ID和父节点ID
	nodeIds := make([]int, 0, len(nodes))
	parentIdSet := make(map[int]struct{})
	for _, node := range nodes {
		nodeIds = append(nodeIds, node.ID)
		if node.ParentID != 0 {
			parentIdSet[node.ParentID] = struct{}{}
		}
	}

	// 批量获取父节点名称
	parentMap := make(map[int]string)
	if len(parentIdSet) > 0 {
		parentIds := make([]int, 0, len(parentIdSet))
		for id := range parentIdSet {
			parentIds = append(parentIds, id)
		}

		var parents []struct {
			ID   int
			Name string
		}

		if err := t.db.WithContext(ctx).Table("tree_nodes").
			Select("id, name").
			Where("id IN ?", parentIds).
			Find(&parents).Error; err != nil {
			t.logger.Error("获取父节点名称失败", zap.Error(err))
			return nil, fmt.Errorf("获取父节点名称失败: %w", err)
		}

		for _, p := range parents {
			parentMap[p.ID] = p.Name
		}
	}

	// 批量获取子节点数量
	childCounts := make(map[int]int)
	type ChildCount struct {
		ParentID int `gorm:"column:parent_id"`
		Count    int `gorm:"column:count"`
	}
	var counts []ChildCount

	if err := t.db.WithContext(ctx).Model(&model.TreeNode{}).
		Select("parent_id, COUNT(*) as count").
		Where("parent_id IN ?", nodeIds).
		Group("parent_id").
		Scan(&counts).Error; err != nil {
		t.logger.Error("获取子节点数量失败", zap.Error(err))
		return nil, fmt.Errorf("获取子节点数量失败: %w", err)
	}

	for _, c := range counts {
		childCounts[c.ParentID] = c.Count
	}

	// 设置每个节点的额外信息
	for _, node := range nodes {
		if node.ParentID != 0 {
			node.ParentName = parentMap[node.ParentID]
		}
		node.ChildCount = childCounts[node.ID]
	}

	// 构建树形结构
	nodeMap := make(map[int]*model.TreeNode)
	var rootNodes []*model.TreeNode

	for _, node := range nodes {
		nodeClone := *node
		nodeMap[node.ID] = &nodeClone
		nodeMap[node.ID].Children = make([]*model.TreeNode, 0)
	}

	for _, node := range nodes {
		if node.ParentID == 0 || nodeMap[node.ParentID] == nil {
			// 这是根节点或父节点不在当前结果集中
			rootNodes = append(rootNodes, nodeMap[node.ID])
		} else {
			// 将当前节点添加到其父节点的子节点列表中
			parent := nodeMap[node.ParentID]
			parent.Children = append(parent.Children, nodeMap[node.ID])
		}
	}

	// 如果请求特定层级，则返回该层级的所有节点
	if req.Level > 0 {
		return nodes, nil
	}

	return rootNodes, nil
}

// GetTreeStatistics 获取树统计信息
func (t *treeDAO) GetTreeStatistics(ctx context.Context) (*model.TreeStatisticsResp, error) {
	var stats = &model.TreeStatisticsResp{}

	err := t.Transaction(ctx, func(txCtx context.Context) error {
		tx := t.db.WithContext(txCtx)

		// 获取节点总数
		var totalNodes int64
		if err := tx.Model(&model.TreeNode{}).Count(&totalNodes).Error; err != nil {
			t.logger.Error("获取节点总数失败", zap.Error(err))
			return fmt.Errorf("获取节点总数失败: %w", err)
		}
		stats.TotalNodes = int(totalNodes)

		// 获取活跃和非活跃节点数
		type StatusCount struct {
			Status string
			Count  int64
		}
		var statusCounts []StatusCount

		if err := tx.Model(&model.TreeNode{}).
			Select("status, COUNT(*) as count").
			Group("status").
			Scan(&statusCounts).Error; err != nil {
			t.logger.Error("获取节点状态统计失败", zap.Error(err))
			return fmt.Errorf("获取节点状态统计失败: %w", err)
		}

		for _, sc := range statusCounts {
			switch sc.Status {
			case "active":
				stats.ActiveNodes = int(sc.Count)
			case "inactive":
				stats.InactiveNodes = int(sc.Count)
			}
		}

		// 获取管理员总数（去重）
		var totalAdmins int64
		if err := tx.Model(&model.TreeNodeAdmin{}).
			Select("COUNT(DISTINCT user_id)").
			Scan(&totalAdmins).Error; err != nil {
			t.logger.Error("获取管理员总数失败", zap.Error(err))
			return fmt.Errorf("获取管理员总数失败: %w", err)
		}
		stats.TotalAdmins = int(totalAdmins)

		// 获取成员总数（去重）
		var totalMembers int64
		if err := tx.Model(&model.TreeNodeMember{}).
			Select("COUNT(DISTINCT user_id)").
			Scan(&totalMembers).Error; err != nil {
			t.logger.Error("获取成员总数失败", zap.Error(err))
			return fmt.Errorf("获取成员总数失败: %w", err)
		}
		stats.TotalMembers = int(totalMembers)

		// 资源总数暂时设置为0
		stats.TotalResources = 0

		return nil
	})

	if err != nil {
		return nil, err
	}

	return stats, nil
}

// RemoveNodeMember 移除节点成员
func (t *treeDAO) RemoveNodeMember(ctx context.Context, nodeId int, userId int, memberType string) error {
	// 检查节点是否存在
	if err := t.checkNodeExists(ctx, nodeId); err != nil {
		return err
	}

	var result *gorm.DB

	switch memberType {
	case "admin":
		// 删除管理员关系
		result = t.db.WithContext(ctx).Where("tree_node_id = ? AND user_id = ?", nodeId, userId).Delete(&model.TreeNodeAdmin{})
	case "member":
		// 删除成员关系
		result = t.db.WithContext(ctx).Where("tree_node_id = ? AND user_id = ?", nodeId, userId).Delete(&model.TreeNodeMember{})
	default:
		return errors.New("无效的成员类型")
	}

	if result.Error != nil {
		t.logger.Error("移除成员失败", zap.Int("nodeId", nodeId), zap.Int("userId", userId), zap.Error(result.Error))
		return fmt.Errorf("移除成员失败: %w", result.Error)
	}

	// 检查是否实际删除了记录
	if result.RowsAffected == 0 {
		return fmt.Errorf("未找到要移除的%s关系", memberType)
	}

	return nil
}

// UnbindResource 解绑资源
func (t *treeDAO) UnbindResource(ctx context.Context, nodeId int, resourceType string, resourceId string) error {
	// 暂时留空
	return nil
}

// UpdateNode 更新节点
func (t *treeDAO) UpdateNode(ctx context.Context, node *model.TreeNode) error {
	// 检查节点是否存在
	var existingNode model.TreeNode
	if err := t.db.WithContext(ctx).First(&existingNode, node.ID).Error; err != nil {
		t.logger.Error("节点不存在", zap.Int("id", node.ID), zap.Error(err))
		return errors.New("节点不存在")
	}

	// 检查节点名称是否与同级节点重复
	var count int64
	if err := t.db.WithContext(ctx).Model(&model.TreeNode{}).
		Where("name = ? AND parent_id = ? AND id != ?", node.Name, existingNode.ParentID, node.ID).
		Count(&count).Error; err != nil {
		t.logger.Error("检查节点名称失败", zap.String("name", node.Name), zap.Error(err))
		return fmt.Errorf("检查节点名称失败: %w", err)
	}

	if count > 0 {
		return errors.New("同级节点下已存在相同名称的节点")
	}

	// 更新节点信息，只更新允许修改的字段
	updateMap := map[string]interface{}{
		"name":        node.Name,
		"description": node.Description,
		"status":      node.Status,
		"updated_at":  node.UpdatedAt,
	}

	if err := t.db.WithContext(ctx).Model(&model.TreeNode{}).Where("id = ?", node.ID).Updates(updateMap).Error; err != nil {
		t.logger.Error("更新节点失败", zap.Int("id", node.ID), zap.Error(err))
		return fmt.Errorf("更新节点失败: %w", err)
	}

	return nil
}

// GetNodeLevel 获取节点层级
func (t *treeDAO) GetNodeLevel(ctx context.Context, nodeId int) (int, error) {
	var level int
	err := t.db.WithContext(ctx).Model(&model.TreeNode{}).
		Select("level").
		Where("id = ?", nodeId).
		Scan(&level).Error

	if err != nil {
		t.logger.Error("获取节点层级失败", zap.Int("id", nodeId), zap.Error(err))
		return 0, fmt.Errorf("获取节点层级失败: %w", err)
	}

	return level, nil
}

// Transaction 提供事务支持
func (t *treeDAO) Transaction(ctx context.Context, fn func(ctx context.Context) error) error {
	var resultErr error
	err := t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := context.WithValue(ctx, txKey, tx)

		// 执行事务函数
		if err := fn(txCtx); err != nil {
			t.logger.Error("事务执行失败", zap.Error(err))
			return err
		}

		// 获取事务上下文中可能存储的结果
		if v := txCtx.Value("result"); v != nil {
			if e, ok := v.(error); ok {
				resultErr = e
			}
		}

		return nil
	})

	if resultErr != nil {
		return resultErr
	}
	return err
}
