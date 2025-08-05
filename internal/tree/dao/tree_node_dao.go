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
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)


type TreeNodeDAO interface {
	// 基础查询方法
	GetTreeList(ctx context.Context, req *model.GetTreeNodeListReq) ([]*model.TreeNode, int64, error)
	GetNode(ctx context.Context, id int) (*model.TreeNode, error)
	GetChildNodes(ctx context.Context, parentId int) ([]*model.TreeNode, error)
	GetTreeStatistics(ctx context.Context) (*model.TreeNodeStatisticsResp, error)

	// 节点管理方法
	CreateNode(ctx context.Context, node *model.TreeNode) error
	UpdateNode(ctx context.Context, node *model.TreeNode) error
	UpdateNodeStatus(ctx context.Context, id int, status string) error
	DeleteNode(ctx context.Context, id int) error

	// 资源管理方法
	GetNodeResources(ctx context.Context, nodeId int) ([]*model.TreeNodeResource, error)
	BindResource(ctx context.Context, nodeId int, resourceType model.CloudProvider, resourceIds []string) error
	UnbindResource(ctx context.Context, nodeId int, resourceType string, resourceId string) error

	// 成员管理方法
	GetNodeMembers(ctx context.Context, nodeId int, memberType string) ([]*model.User, error)
	AddNodeMember(ctx context.Context, nodeId int, userId int, memberType string) error
	RemoveNodeMember(ctx context.Context, nodeId int, userId int, memberType string) error
}

type treeNodeDAO struct {
	logger *zap.Logger
	db     *gorm.DB
}

func NewTreeNodeDAO(logger *zap.Logger, db *gorm.DB) TreeNodeDAO {
	return &treeNodeDAO{
		logger: logger,
		db:     db,
	}
}

// GetTreeList 获取树节点列表
func (t *treeNodeDAO) GetTreeList(ctx context.Context, req *model.GetTreeNodeListReq) ([]*model.TreeNode, int64, error) {
	var nodes []*model.TreeNode
	var count int64

	query := t.db.WithContext(ctx).Model(&model.TreeNode{})

	// 添加过滤条件
	if req.Level > 0 {
		query = query.Where("level = ?", req.Level)
	}
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}

	// 获取总数
	if err := query.Count(&count).Error; err != nil {
		t.logger.Error("获取树节点总数失败", zap.Error(err))
		return nil, 0, fmt.Errorf("获取树节点总数失败: %w", err)
	}

	// 预加载关联数据并查询
	if err := query.Preload("AdminUsers").
		Preload("MemberUsers").
		Order("level ASC, parent_id ASC, name ASC").
		Find(&nodes).Error; err != nil {
		t.logger.Error("获取树节点列表失败", zap.Error(err))
		return nil, 0, fmt.Errorf("获取树节点列表失败: %w", err)
	}

	// 填充额外信息
	if err := t.fillNodesExtraInfo(ctx, nodes); err != nil {
		return nil, 0, err
	}

	// 如果指定了层级，直接返回列表
	if req.Level > 0 {
		return nodes, count, nil
	}

	// 构建树形结构
	return t.buildTreeStructure(nodes), count, nil
}

// fillNodesExtraInfo 填充节点的额外信息
func (t *treeNodeDAO) fillNodesExtraInfo(ctx context.Context, nodes []*model.TreeNode) error {
	if len(nodes) == 0 {
		return nil
	}

	// 填充父节点名称
	parentMap := make(map[int]string)
	creatorMap := make(map[int]string)
	nodeIds := make([]int, len(nodes))

	for i, node := range nodes {
		nodeIds[i] = node.ID
		if node.ParentID != 0 {
			parentMap[node.ParentID] = ""
		}
		if node.CreatorID != 0 {
			creatorMap[node.CreatorID] = ""
		}
	}

	// 获取父节点名称
	if len(parentMap) > 0 {
		var parentIds []int
		for id := range parentMap {
			parentIds = append(parentIds, id)
		}

		var parents []model.TreeNode
		if err := t.db.WithContext(ctx).Select("id, name").Where("id IN ?", parentIds).Find(&parents).Error; err != nil {
			t.logger.Error("获取父节点名称失败", zap.Error(err))
		} else {
			for _, p := range parents {
				parentMap[p.ID] = p.Name
			}
		}
	}

	// 获取创建者名称
	if len(creatorMap) > 0 {
		var creatorIds []int
		for id := range creatorMap {
			creatorIds = append(creatorIds, id)
		}

		var creators []model.User
		if err := t.db.WithContext(ctx).Select("id, username").Where("id IN ?", creatorIds).Find(&creators).Error; err != nil {
			t.logger.Error("获取创建者名称失败", zap.Error(err))
		} else {
			for _, c := range creators {
				creatorMap[c.ID] = c.Username
			}
		}
	}

	// 获取子节点数量和资源数量
	childCounts := make(map[int]int64)
	resourceCounts := make(map[int]int64)

	// 批量查询子节点数量
	var childCountResults []struct {
		ParentID int   `gorm:"column:parent_id"`
		Count    int64 `gorm:"column:count"`
	}
	if err := t.db.WithContext(ctx).Model(&model.TreeNode{}).
		Select("parent_id, COUNT(*) as count").
		Where("parent_id IN ?", nodeIds).
		Group("parent_id").
		Scan(&childCountResults).Error; err != nil {
		t.logger.Error("获取子节点数量失败", zap.Error(err))
	} else {
		for _, r := range childCountResults {
			childCounts[r.ParentID] = r.Count
		}
	}

	// 批量查询资源数量
	var resourceCountResults []struct {
		TreeNodeID int   `gorm:"column:tree_node_id"`
		Count      int64 `gorm:"column:count"`
	}
	if err := t.db.WithContext(ctx).Model(&model.TreeNodeResource{}).
		Select("tree_node_id, COUNT(*) as count").
		Where("tree_node_id IN ?", nodeIds).
		Group("tree_node_id").
		Scan(&resourceCountResults).Error; err != nil {
		t.logger.Error("获取资源数量失败", zap.Error(err))
	} else {
		for _, r := range resourceCountResults {
			resourceCounts[r.TreeNodeID] = r.Count
		}
	}

	// 设置节点信息
	for _, node := range nodes {
		if node.ParentID != 0 {
			node.ParentName = parentMap[node.ParentID]
		}
		if node.CreatorID != 0 {
			node.CreatorName = creatorMap[node.CreatorID]
		}
		node.ChildCount = int(childCounts[node.ID])
		node.IsLeaf = childCounts[node.ID] == 0
		node.ResourceCount = int(resourceCounts[node.ID])
	}

	return nil
}

// buildTreeStructure 构建树形结构
func (t *treeNodeDAO) buildTreeStructure(nodes []*model.TreeNode) []*model.TreeNode {
	nodeMap := make(map[int]*model.TreeNode)
	var rootNodes []*model.TreeNode

	for _, node := range nodes {
		nodeClone := *node
		nodeClone.Children = make([]*model.TreeNode, 0)
		nodeMap[node.ID] = &nodeClone
	}

	for _, node := range nodes {
		currentNode := nodeMap[node.ID]
		if node.ParentID == 0 || nodeMap[node.ParentID] == nil {
			rootNodes = append(rootNodes, currentNode)
		} else {
			parent := nodeMap[node.ParentID]
			parent.Children = append(parent.Children, currentNode)
		}
	}

	return rootNodes
}

// GetNode 获取节点详情
func (t *treeNodeDAO) GetNode(ctx context.Context, id int) (*model.TreeNode, error) {
	var node model.TreeNode
	if err := t.db.WithContext(ctx).
		Preload("AdminUsers").
		Preload("MemberUsers").
		Where("id = ?", id).
		First(&node).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("节点不存在")
		}
		t.logger.Error("获取节点失败", zap.Int("id", id), zap.Error(err))
		return nil, fmt.Errorf("获取节点失败: %w", err)
	}

	// 填充额外信息
	if err := t.fillNodesExtraInfo(ctx, []*model.TreeNode{&node}); err != nil {
		return nil, err
	}

	return &node, nil
}

// GetChildNodes 获取子节点列表
func (t *treeNodeDAO) GetChildNodes(ctx context.Context, parentId int) ([]*model.TreeNode, error) {
	var nodes []*model.TreeNode
	if err := t.db.WithContext(ctx).
		Preload("AdminUsers").
		Preload("MemberUsers").
		Where("parent_id = ?", parentId).
		Order("name ASC").
		Find(&nodes).Error; err != nil {
		t.logger.Error("获取子节点失败", zap.Int("parentId", parentId), zap.Error(err))
		return nil, fmt.Errorf("获取子节点失败: %w", err)
	}

	// 填充额外信息
	if err := t.fillNodesExtraInfo(ctx, nodes); err != nil {
		return nil, err
	}

	return nodes, nil
}

// GetTreeStatistics 获取树统计信息
func (t *treeNodeDAO) GetTreeStatistics(ctx context.Context) (*model.TreeNodeStatisticsResp, error) {
	stats := &model.TreeNodeStatisticsResp{}
	db := t.db.WithContext(ctx)

	// 获取节点总数
	var totalNodes int64
	if err := db.Model(&model.TreeNode{}).Count(&totalNodes).Error; err != nil {
		t.logger.Error("获取节点总数失败", zap.Error(err))
		return nil, fmt.Errorf("获取节点总数失败: %w", err)
	}
	stats.TotalNodes = int(totalNodes)

	// 获取活跃和非活跃节点数
	var statusCounts []struct {
		Status string `gorm:"column:status"`
		Count  int64  `gorm:"column:count"`
	}
	if err := db.Model(&model.TreeNode{}).
		Select("status, COUNT(*) as count").
		Group("status").
		Scan(&statusCounts).Error; err != nil {
		t.logger.Error("获取节点状态统计失败", zap.Error(err))
		return nil, fmt.Errorf("获取节点状态统计失败: %w", err)
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
	if err := db.Table("cl_tree_node_admin").
		Select("COUNT(DISTINCT user_id)").
		Scan(&totalAdmins).Error; err != nil {
		t.logger.Error("获取管理员总数失败", zap.Error(err))
		return nil, fmt.Errorf("获取管理员总数失败: %w", err)
	}
	stats.TotalAdmins = int(totalAdmins)

	// 获取成员总数（去重）
	var totalMembers int64
	if err := db.Table("cl_tree_node_member").
		Select("COUNT(DISTINCT user_id)").
		Scan(&totalMembers).Error; err != nil {
		t.logger.Error("获取成员总数失败", zap.Error(err))
		return nil, fmt.Errorf("获取成员总数失败: %w", err)
	}
	stats.TotalMembers = int(totalMembers)

	// 获取资源总数
	var totalResources int64
	if err := db.Model(&model.TreeNodeResource{}).Count(&totalResources).Error; err != nil {
		t.logger.Error("获取资源总数失败", zap.Error(err))
		return nil, fmt.Errorf("获取资源总数失败: %w", err)
	}
	stats.TotalResources = int(totalResources)

	return stats, nil
}

// CreateNode 创建节点
func (t *treeNodeDAO) CreateNode(ctx context.Context, node *model.TreeNode) error {
	if node == nil {
		return errors.New("节点信息不能为空")
	}
	if strings.TrimSpace(node.Name) == "" {
		return errors.New("节点名称不能为空")
	}

	// 验证父节点存在性
	if node.ParentID != 0 {
		var count int64
		if err := t.db.WithContext(ctx).Model(&model.TreeNode{}).Where("id = ?", node.ParentID).Count(&count).Error; err != nil {
			return fmt.Errorf("验证父节点失败: %w", err)
		}
		if count == 0 {
			return errors.New("父节点不存在")
		}
	}

	// 计算层级
	level := 1
	if node.ParentID != 0 {
		var parentLevel int
		if err := t.db.WithContext(ctx).Model(&model.TreeNode{}).
			Select("level").Where("id = ?", node.ParentID).Scan(&parentLevel).Error; err != nil {
			return fmt.Errorf("获取父节点层级失败: %w", err)
		}
		level = parentLevel + 1
	}
	node.Level = level

	// 检查同级节点名称唯一性
	var count int64
	if err := t.db.WithContext(ctx).Model(&model.TreeNode{}).
		Where("name = ? AND parent_id = ?", node.Name, node.ParentID).
		Count(&count).Error; err != nil {
		return fmt.Errorf("检查节点名称失败: %w", err)
	}
	if count > 0 {
		return errors.New("同级节点下已存在相同名称的节点")
	}

	// 设置默认状态
	if node.Status == "" {
		node.Status = "active"
	}

	// 创建节点
	if err := t.db.WithContext(ctx).Create(node).Error; err != nil {
		t.logger.Error("创建节点失败", zap.String("name", node.Name), zap.Error(err))
		return fmt.Errorf("创建节点失败: %w", err)
	}

	// 更新父节点的叶子状态
	if node.ParentID != 0 {
		if err := t.db.WithContext(ctx).Model(&model.TreeNode{}).
			Where("id = ?", node.ParentID).
			Update("is_leaf", false).Error; err != nil {
			t.logger.Error("更新父节点叶子状态失败", zap.Int("parentId", node.ParentID), zap.Error(err))
		}
	}

	return nil
}

// UpdateNode 更新节点
func (t *treeNodeDAO) UpdateNode(ctx context.Context, node *model.TreeNode) error {
	if node == nil {
		return errors.New("节点信息不能为空")
	}
	if strings.TrimSpace(node.Name) == "" {
		return errors.New("节点名称不能为空")
	}

	// 获取现有节点信息
	existingNode, err := t.GetNode(ctx, node.ID)
	if err != nil {
		return err
	}

	// 如果父节点发生变化，需要验证和计算层级
	if node.ParentID != existingNode.ParentID {
		// 验证新父节点存在（如果不是根节点）
		if node.ParentID != 0 {
			var count int64
			if err := t.db.WithContext(ctx).Model(&model.TreeNode{}).Where("id = ?", node.ParentID).Count(&count).Error; err != nil {
				return fmt.Errorf("验证父节点失败: %w", err)
			}
			if count == 0 {
				return errors.New("父节点不存在")
			}

			// 简单验证：不能移动到自己的子节点下
			if node.ParentID == node.ID {
				return errors.New("不能将节点移动到自己下")
			}
		}

		// 重新计算层级
		level := 1
		if node.ParentID != 0 {
			var parentLevel int
			if err := t.db.WithContext(ctx).Model(&model.TreeNode{}).
				Select("level").Where("id = ?", node.ParentID).Scan(&parentLevel).Error; err != nil {
				return fmt.Errorf("获取父节点层级失败: %w", err)
			}
			level = parentLevel + 1
		}
		node.Level = level
	} else {
		// 父节点未变化，保持原层级
		node.Level = existingNode.Level
	}

	// 检查同级节点名称唯一性
	var count int64
	if err := t.db.WithContext(ctx).Model(&model.TreeNode{}).
		Where("name = ? AND parent_id = ? AND id != ?", node.Name, node.ParentID, node.ID).
		Count(&count).Error; err != nil {
		t.logger.Error("检查节点名称失败", zap.String("name", node.Name), zap.Error(err))
		return fmt.Errorf("检查节点名称失败: %w", err)
	}

	if count > 0 {
		return errors.New("同级节点下已存在相同名称的节点")
	}

	// 执行更新
	updateMap := map[string]any{
		"name":        node.Name,
		"description": node.Description,
		"status":      node.Status,
		"parent_id":   node.ParentID,
		"level":       node.Level,
	}

	if err := t.db.WithContext(ctx).Model(&model.TreeNode{}).Where("id = ?", node.ID).Updates(updateMap).Error; err != nil {
		t.logger.Error("更新节点失败", zap.Int("id", node.ID), zap.Error(err))
		return fmt.Errorf("更新节点失败: %w", err)
	}

	// 如果父节点发生变化，需要更新相关节点的叶子状态
	if node.ParentID != existingNode.ParentID {
		// 更新原父节点的叶子状态
		if existingNode.ParentID != 0 {
			var remainingChildren int64
			if err := t.db.WithContext(ctx).Model(&model.TreeNode{}).Where("parent_id = ?", existingNode.ParentID).Count(&remainingChildren).Error; err == nil {
				isLeaf := remainingChildren == 0
				t.db.WithContext(ctx).Model(&model.TreeNode{}).Where("id = ?", existingNode.ParentID).Update("is_leaf", isLeaf)
			}
		}

		// 更新新父节点的叶子状态
		if node.ParentID != 0 {
			t.db.WithContext(ctx).Model(&model.TreeNode{}).Where("id = ?", node.ParentID).Update("is_leaf", false)
		}
	}

	return nil
}


// DeleteNode 删除节点
func (t *treeNodeDAO) DeleteNode(ctx context.Context, id int) error {
	node, err := t.GetNode(ctx, id)
	if err != nil {
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

	return t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 获取要删除的节点（用于清理关联）
		var nodeToDelete model.TreeNode
		if err := tx.Preload("AdminUsers").Preload("MemberUsers").First(&nodeToDelete, id).Error; err != nil {
			return fmt.Errorf("获取节点信息失败: %w", err)
		}

		// 清理管理员关联
		if err := tx.Model(&nodeToDelete).Association("AdminUsers").Clear(); err != nil {
			t.logger.Error("清理管理员关联失败", zap.Int("id", id), zap.Error(err))
			return fmt.Errorf("清理管理员关联失败: %w", err)
		}

		// 清理成员关联
		if err := tx.Model(&nodeToDelete).Association("MemberUsers").Clear(); err != nil {
			t.logger.Error("清理成员关联失败", zap.Int("id", id), zap.Error(err))
			return fmt.Errorf("清理成员关联失败: %w", err)
		}

		// 删除节点资源关系
		if err := tx.Where("tree_node_id = ?", id).Delete(&model.TreeNodeResource{}).Error; err != nil {
			t.logger.Error("删除资源关系失败", zap.Int("id", id), zap.Error(err))
			return fmt.Errorf("删除资源关系失败: %w", err)
		}

		// 删除节点
		if err := tx.Delete(&model.TreeNode{}, id).Error; err != nil {
			t.logger.Error("删除节点失败", zap.Int("id", id), zap.Error(err))
			return fmt.Errorf("删除节点失败: %w", err)
		}

		// 更新父节点的叶子节点状态
		if node.ParentID != 0 {
			var remainingChildren int64
			if err := tx.Model(&model.TreeNode{}).Where("parent_id = ?", node.ParentID).Count(&remainingChildren).Error; err != nil {
				t.logger.Error("检查剩余子节点失败", zap.Int("parentId", node.ParentID), zap.Error(err))
				return fmt.Errorf("检查剩余子节点失败: %w", err)
			}

			isLeaf := remainingChildren == 0
			if err := tx.Model(&model.TreeNode{}).Where("id = ?", node.ParentID).Update("is_leaf", isLeaf).Error; err != nil {
				t.logger.Error("更新父节点叶子状态失败", zap.Int("parentId", node.ParentID), zap.Error(err))
				return fmt.Errorf("更新父节点叶子状态失败: %w", err)
			}
		}

		return nil
	})
}

// GetNodeResources 获取节点绑定的资源列表
func (t *treeNodeDAO) GetNodeResources(ctx context.Context, nodeId int) ([]*model.TreeNodeResource, error) {
	var nodeResources []*model.TreeNodeResource
	if err := t.db.WithContext(ctx).Where("tree_node_id = ?", nodeId).Find(&nodeResources).Error; err != nil {
		t.logger.Error("获取节点资源关系失败", zap.Int("nodeId", nodeId), zap.Error(err))
		return nil, fmt.Errorf("获取节点资源关系失败: %w", err)
	}

	return nodeResources, nil
}

// BindResource 绑定资源到节点
func (t *treeNodeDAO) BindResource(ctx context.Context, nodeId int, resourceType model.CloudProvider, resourceIds []string) error {
	// 验证节点存在
	var count int64
	if err := t.db.WithContext(ctx).Model(&model.TreeNode{}).Where("id = ?", nodeId).Count(&count).Error; err != nil {
		return fmt.Errorf("验证节点存在失败: %w", err)
	}
	if count == 0 {
		return errors.New("节点不存在")
	}

	return t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 查询现有绑定
		var existingBindings []model.TreeNodeResource
		if err := tx.Where("tree_node_id = ? AND resource_type = ? AND resource_id IN ?",
			nodeId, resourceType, resourceIds).Find(&existingBindings).Error; err != nil {
			t.logger.Error("查询现有资源绑定失败",
				zap.Int("nodeId", nodeId),
				zap.String("resourceType", string(resourceType)),
				zap.Error(err))
			return fmt.Errorf("查询现有资源绑定失败: %w", err)
		}

		// 构建已存在的资源ID映射
		existingMap := make(map[string]bool)
		for _, binding := range existingBindings {
			existingMap[binding.ResourceID] = true
		}

		// 准备新的绑定记录
		var newBindings []model.TreeNodeResource
		for _, resourceId := range resourceIds {
			if strings.TrimSpace(resourceId) == "" {
				continue
			}

			if existingMap[resourceId] {
				continue
			}

			newBindings = append(newBindings, model.TreeNodeResource{
				TreeNodeID:   nodeId,
				ResourceID:   resourceId,
				ResourceType: resourceType,
			})
		}

		// 批量创建新绑定
		if len(newBindings) > 0 {
			if err := tx.Create(&newBindings).Error; err != nil {
				t.logger.Error("批量创建资源绑定关系失败",
					zap.Int("nodeId", nodeId),
					zap.String("resourceType", string(resourceType)),
					zap.Int("count", len(newBindings)),
					zap.Error(err))
				return fmt.Errorf("批量创建资源绑定关系失败: %w", err)
			}
		}
		return nil
	})
}

// UnbindResource 解绑资源
func (t *treeNodeDAO) UnbindResource(ctx context.Context, nodeId int, resourceType string, resourceId string) error {
	result := t.db.WithContext(ctx).Where("tree_node_id = ? AND resource_type = ? AND resource_id = ?",
		nodeId, resourceType, resourceId).Delete(&model.TreeNodeResource{})

	if result.Error != nil {
		t.logger.Error("解绑资源失败",
			zap.Int("nodeId", nodeId),
			zap.String("resourceType", resourceType),
			zap.String("resourceId", resourceId),
			zap.Error(result.Error))
		return fmt.Errorf("解绑资源失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.New("未找到要解绑的资源关系")
	}

	return nil
}

// UpdateNodeStatus 更新节点状态
func (t *treeNodeDAO) UpdateNodeStatus(ctx context.Context, id int, status string) error {
	if status != "active" && status != "inactive" {
		return errors.New("状态只能是active或inactive")
	}

	result := t.db.WithContext(ctx).Model(&model.TreeNode{}).Where("id = ?", id).Update("status", status)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("节点不存在")
	}

	return nil
}

// GetNodeMembers 获取节点成员列表
func (t *treeNodeDAO) GetNodeMembers(ctx context.Context, nodeId int, memberType string) ([]*model.User, error) {
	var node model.TreeNode
	if err := t.db.WithContext(ctx).First(&node, nodeId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("节点不存在")
		}
		return nil, fmt.Errorf("查询节点失败: %w", err)
	}

	var users []*model.User
	db := t.db.WithContext(ctx)

	switch memberType {
	case "admin":
		if err := db.Model(&node).Association("AdminUsers").Find(&users); err != nil {
			t.logger.Error("获取管理员失败", zap.Int("nodeId", nodeId), zap.Error(err))
			return nil, fmt.Errorf("获取管理员失败: %w", err)
		}
	case "member":
		if err := db.Model(&node).Association("MemberUsers").Find(&users); err != nil {
			t.logger.Error("获取成员失败", zap.Int("nodeId", nodeId), zap.Error(err))
			return nil, fmt.Errorf("获取成员失败: %w", err)
		}
	case "all", "":
		// 获取所有用户（管理员+成员）
		var adminUsers []*model.User
		var memberUsers []*model.User
		
		if err := db.Model(&node).Association("AdminUsers").Find(&adminUsers); err != nil {
			t.logger.Error("获取管理员失败", zap.Int("nodeId", nodeId), zap.Error(err))
			return nil, fmt.Errorf("获取管理员失败: %w", err)
		}
		
		if err := db.Model(&node).Association("MemberUsers").Find(&memberUsers); err != nil {
			t.logger.Error("获取成员失败", zap.Int("nodeId", nodeId), zap.Error(err))
			return nil, fmt.Errorf("获取成员失败: %w", err)
		}
		
		// 合并并去重
		userMap := make(map[int]*model.User)
		for _, user := range adminUsers {
			userMap[user.ID] = user
		}
		for _, user := range memberUsers {
			userMap[user.ID] = user
		}
		
		users = make([]*model.User, 0, len(userMap))
		for _, user := range userMap {
			users = append(users, user)
		}
	default:
		return nil, errors.New("无效的成员类型，必须是 admin、member 或 all")
	}

	return users, nil
}

// AddNodeMember 添加节点成员
func (t *treeNodeDAO) AddNodeMember(ctx context.Context, nodeId int, userId int, memberType string) error {
	var node model.TreeNode
	if err := t.db.WithContext(ctx).First(&node, nodeId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("节点不存在")
		}
		return fmt.Errorf("查询节点失败: %w", err)
	}

	var user model.User
	if err := t.db.WithContext(ctx).First(&user, userId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("用户不存在")
		}
		return fmt.Errorf("查询用户失败: %w", err)
	}

	db := t.db.WithContext(ctx)

	switch memberType {
	case "admin":
		// 检查是否已存在
		var existingUsers []*model.User
		if err := db.Model(&node).Association("AdminUsers").Find(&existingUsers); err != nil {
			return fmt.Errorf("查询现有管理员失败: %w", err)
		}
		
		for _, existing := range existingUsers {
			if existing.ID == userId {
				return errors.New("用户已经是该节点的管理员")
			}
		}
		
		if err := db.Model(&node).Association("AdminUsers").Append(&user); err != nil {
			t.logger.Error("添加节点管理员失败", zap.Int("nodeId", nodeId), zap.Int("userId", userId), zap.Error(err))
			return fmt.Errorf("添加节点管理员失败: %w", err)
		}

	case "member":
		// 检查是否已存在
		var existingUsers []*model.User
		if err := db.Model(&node).Association("MemberUsers").Find(&existingUsers); err != nil {
			return fmt.Errorf("查询现有成员失败: %w", err)
		}
		
		for _, existing := range existingUsers {
			if existing.ID == userId {
				return errors.New("用户已经是该节点的成员")
			}
		}
		
		if err := db.Model(&node).Association("MemberUsers").Append(&user); err != nil {
			t.logger.Error("添加节点成员失败", zap.Int("nodeId", nodeId), zap.Int("userId", userId), zap.Error(err))
			return fmt.Errorf("添加节点成员失败: %w", err)
		}

	default:
		return errors.New("无效的成员类型，必须是 admin 或 member")
	}

	return nil
}

// RemoveNodeMember 移除节点成员
func (t *treeNodeDAO) RemoveNodeMember(ctx context.Context, nodeId int, userId int, memberType string) error {
	var node model.TreeNode
	if err := t.db.WithContext(ctx).First(&node, nodeId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("节点不存在")
		}
		return fmt.Errorf("查询节点失败: %w", err)
	}

	var user model.User
	if err := t.db.WithContext(ctx).First(&user, userId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("用户不存在")
		}
		return fmt.Errorf("查询用户失败: %w", err)
	}

	db := t.db.WithContext(ctx)

	switch memberType {
	case "admin":
		if err := db.Model(&node).Association("AdminUsers").Delete(&user); err != nil {
			t.logger.Error("移除节点管理员失败", zap.Int("nodeId", nodeId), zap.Int("userId", userId), zap.Error(err))
			return fmt.Errorf("移除节点管理员失败: %w", err)
		}
	case "member":
		if err := db.Model(&node).Association("MemberUsers").Delete(&user); err != nil {
			t.logger.Error("移除节点成员失败", zap.Int("nodeId", nodeId), zap.Int("userId", userId), zap.Error(err))
			return fmt.Errorf("移除节点成员失败: %w", err)
		}
	default:
		return errors.New("无效的成员类型，必须是 admin 或 member")
	}

	return nil
}
