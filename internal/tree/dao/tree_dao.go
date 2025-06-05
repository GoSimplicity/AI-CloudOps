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

type contextKey string

const txKey contextKey = "transaction"

type TreeDAO interface {
	GetTreeList(ctx context.Context, req *model.GetTreeListReq) ([]*model.TreeNode, error)
	GetNode(ctx context.Context, id int) (*model.TreeNode, error)
	GetChildNodes(ctx context.Context, parentId int) ([]*model.TreeNode, error)
	GetTreeStatistics(ctx context.Context) (*model.TreeStatisticsResp, error)
	GetNodeLevel(ctx context.Context, nodeId int) (int, error)
	GetLeafNodesByIDs(ctx context.Context, ids []int) ([]*model.TreeNode, error)

	CreateNode(ctx context.Context, node *model.TreeNode) error
	UpdateNode(ctx context.Context, node *model.TreeNode) error
	UpdateNodeStatus(ctx context.Context, id int, status string) error
	DeleteNode(ctx context.Context, id int) error

	GetNodeResources(ctx context.Context, nodeId int) ([]*model.ResourceBase, error)
	BindResource(ctx context.Context, nodeId int, resourceType string, resourceIds []string) error
	UnbindResource(ctx context.Context, nodeId int, resourceType string, resourceId string) error

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

// getDB 获取数据库连接，优先获取事务连接
func (t *treeDAO) getDB(ctx context.Context) *gorm.DB {
	if tx, ok := ctx.Value(txKey).(*gorm.DB); ok {
		return tx.WithContext(ctx)
	}
	return t.db.WithContext(ctx)
}

// checkNodeExists 检查节点是否存在的辅助函数
func (t *treeDAO) checkNodeExists(ctx context.Context, id int) error {
	var count int64
	if err := t.getDB(ctx).Model(&model.TreeNode{}).Where("id = ?", id).Count(&count).Error; err != nil {
		t.logger.Error("检查节点存在性失败", zap.Int("nodeId", id), zap.Error(err))
		return fmt.Errorf("检查节点失败: %w", err)
	}

	if count == 0 {
		return errors.New("节点不存在")
	}
	return nil
}

// checkUserExists 检查用户是否存在的辅助函数
func (t *treeDAO) checkUserExists(ctx context.Context, userId int) error {
	var count int64
	if err := t.getDB(ctx).Model(&model.User{}).Where("id = ?", userId).Count(&count).Error; err != nil {
		t.logger.Error("检查用户存在性失败", zap.Int("userId", userId), zap.Error(err))
		return fmt.Errorf("检查用户失败: %w", err)
	}

	if count == 0 {
		return errors.New("用户不存在")
	}
	return nil
}

// calculateLevel 计算节点层级
func (t *treeDAO) calculateLevel(ctx context.Context, parentId int) (int, error) {
	if parentId == 0 {
		return 1, nil
	}

	var parentLevel int
	if err := t.getDB(ctx).Model(&model.TreeNode{}).
		Select("level").
		Where("id = ?", parentId).
		Scan(&parentLevel).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, errors.New("父节点不存在")
		}
		t.logger.Error("获取父节点层级失败", zap.Int("parentId", parentId), zap.Error(err))
		return 0, fmt.Errorf("获取父节点层级失败: %w", err)
	}

	return parentLevel + 1, nil
}

// updateChildrenIsLeaf 更新子节点的叶子节点状态
func (t *treeDAO) updateChildrenIsLeaf(ctx context.Context, parentId int) error {
	// 检查父节点是否有子节点
	var childCount int64
	if err := t.getDB(ctx).Model(&model.TreeNode{}).Where("parent_id = ?", parentId).Count(&childCount).Error; err != nil {
		return fmt.Errorf("检查子节点数量失败: %w", err)
	}

	// 更新父节点的叶子节点状态
	isLeaf := childCount == 0
	if err := t.getDB(ctx).Model(&model.TreeNode{}).Where("id = ?", parentId).Update("is_leaf", isLeaf).Error; err != nil {
		return fmt.Errorf("更新父节点叶子状态失败: %w", err)
	}

	return nil
}

// GetTreeList 获取树节点列表
func (t *treeDAO) GetTreeList(ctx context.Context, req *model.GetTreeListReq) ([]*model.TreeNode, error) {
	var nodes []*model.TreeNode
	query := t.getDB(ctx).Model(&model.TreeNode{})

	// 根据请求参数过滤
	if req.Level > 0 {
		query = query.Where("level = ?", req.Level)
	}

	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}

	// 排序
	query = query.Order("level ASC, parent_id ASC, name ASC")

	// 执行查询
	if err := query.Find(&nodes).Error; err != nil {
		t.logger.Error("获取树节点列表失败", zap.Error(err))
		return nil, fmt.Errorf("获取树节点列表失败: %w", err)
	}

	if len(nodes) == 0 {
		return nodes, nil
	}

	// 批量获取额外信息
	if err := t.batchLoadNodeExtraInfo(ctx, nodes); err != nil {
		return nil, err
	}

	// 如果请求特定层级，返回平铺结构
	if req.Level > 0 {
		return nodes, nil
	}

	// 构建树形结构
	return t.buildTreeStructure(nodes), nil
}

// batchLoadNodeExtraInfo 批量加载节点的额外信息
func (t *treeDAO) batchLoadNodeExtraInfo(ctx context.Context, nodes []*model.TreeNode) error {
	if len(nodes) == 0 {
		return nil
	}

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
			ID   int    `gorm:"column:id"`
			Name string `gorm:"column:name"`
		}

		if err := t.getDB(ctx).Table("tree_nodes").
			Select("id, name").
			Where("id IN ?", parentIds).
			Scan(&parents).Error; err != nil {
			t.logger.Error("获取父节点名称失败", zap.Error(err))
			return fmt.Errorf("获取父节点名称失败: %w", err)
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

	if err := t.getDB(ctx).Model(&model.TreeNode{}).
		Select("parent_id, COUNT(*) as count").
		Where("parent_id IN ?", nodeIds).
		Group("parent_id").
		Scan(&counts).Error; err != nil {
		t.logger.Error("获取子节点数量失败", zap.Error(err))
		return fmt.Errorf("获取子节点数量失败: %w", err)
	}

	for _, c := range counts {
		childCounts[c.ParentID] = c.Count
	}

	// 设置每个节点的额外信息
	for _, node := range nodes {
		if node.ParentID != 0 {
			node.ParentName = parentMap[node.ParentID]
		}
		childCount := childCounts[node.ID]
		node.ChildCount = childCount
		node.IsLeaf = childCount == 0
	}

	return nil
}

// buildTreeStructure 构建树形结构
func (t *treeDAO) buildTreeStructure(nodes []*model.TreeNode) []*model.TreeNode {
	nodeMap := make(map[int]*model.TreeNode)
	var rootNodes []*model.TreeNode

	// 初始化节点映射和子节点列表
	for _, node := range nodes {
		nodeClone := *node
		nodeClone.Children = make([]*model.TreeNode, 0)
		nodeMap[node.ID] = &nodeClone
	}

	// 构建父子关系
	for _, node := range nodes {
		currentNode := nodeMap[node.ID]
		if node.ParentID == 0 || nodeMap[node.ParentID] == nil {
			// 这是根节点或父节点不在当前结果集中
			rootNodes = append(rootNodes, currentNode)
		} else {
			// 将当前节点添加到其父节点的子节点列表中
			parent := nodeMap[node.ParentID]
			parent.Children = append(parent.Children, currentNode)
		}
	}

	return rootNodes
}

// GetNode 获取节点详情
func (t *treeDAO) GetNode(ctx context.Context, id int) (*model.TreeNode, error) {
	var node model.TreeNode
	if err := t.getDB(ctx).Where("id = ?", id).First(&node).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("节点不存在")
		}
		t.logger.Error("获取节点失败", zap.Int("id", id), zap.Error(err))
		return nil, fmt.Errorf("获取节点失败: %w", err)
	}

	// 获取父节点名称
	if node.ParentID != 0 {
		var parentNode model.TreeNode
		if err := t.getDB(ctx).Select("name").First(&parentNode, node.ParentID).Error; err == nil {
			node.ParentName = parentNode.Name
		} else {
			t.logger.Warn("获取父节点名称失败", zap.Int("parentId", node.ParentID), zap.Error(err))
		}
	}

	// 获取子节点数量
	var childCount int64
	if err := t.getDB(ctx).Model(&model.TreeNode{}).Where("parent_id = ?", id).Count(&childCount).Error; err != nil {
		t.logger.Error("获取子节点数量失败", zap.Int("id", id), zap.Error(err))
		return nil, fmt.Errorf("获取子节点数量失败: %w", err)
	}
	node.ChildCount = int(childCount)
	node.IsLeaf = childCount == 0

	return &node, nil
}

// GetChildNodes 获取子节点列表
func (t *treeDAO) GetChildNodes(ctx context.Context, parentId int) ([]*model.TreeNode, error) {
	var nodes []*model.TreeNode
	if err := t.getDB(ctx).Where("parent_id = ?", parentId).Order("name ASC").Find(&nodes).Error; err != nil {
		t.logger.Error("获取子节点失败", zap.Int("parentId", parentId), zap.Error(err))
		return nil, fmt.Errorf("获取子节点失败: %w", err)
	}

	if len(nodes) == 0 {
		return nodes, nil
	}

	// 批量获取子节点的子节点数
	nodeIds := make([]int, 0, len(nodes))
	for _, node := range nodes {
		nodeIds = append(nodeIds, node.ID)
	}

	// 使用一次查询获取所有子节点的子节点数
	type ChildCount struct {
		ParentID int `gorm:"column:parent_id"`
		Count    int `gorm:"column:count"`
	}
	var childCounts []ChildCount

	if err := t.getDB(ctx).Model(&model.TreeNode{}).
		Select("parent_id, count(*) as count").
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

	// 更新节点的子节点数和叶子节点状态
	for _, node := range nodes {
		childCount := countMap[node.ID]
		node.ChildCount = childCount
		node.IsLeaf = childCount == 0
	}

	return nodes, nil
}

// GetTreeStatistics 获取树统计信息
func (t *treeDAO) GetTreeStatistics(ctx context.Context) (*model.TreeStatisticsResp, error) {
	stats := &model.TreeStatisticsResp{}

	err := t.Transaction(ctx, func(txCtx context.Context) error {
		db := t.getDB(txCtx)

		// 获取节点总数
		var totalNodes int64
		if err := db.Model(&model.TreeNode{}).Count(&totalNodes).Error; err != nil {
			t.logger.Error("获取节点总数失败", zap.Error(err))
			return fmt.Errorf("获取节点总数失败: %w", err)
		}
		stats.TotalNodes = int(totalNodes)

		// 获取活跃和非活跃节点数
		type StatusCount struct {
			Status string `gorm:"column:status"`
			Count  int64  `gorm:"column:count"`
		}
		var statusCounts []StatusCount

		if err := db.Model(&model.TreeNode{}).
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
		if err := db.Model(&model.TreeNodeAdmin{}).
			Select("COUNT(DISTINCT user_id)").
			Scan(&totalAdmins).Error; err != nil {
			t.logger.Error("获取管理员总数失败", zap.Error(err))
			return fmt.Errorf("获取管理员总数失败: %w", err)
		}
		stats.TotalAdmins = int(totalAdmins)

		// 获取成员总数（去重）
		var totalMembers int64
		if err := db.Model(&model.TreeNodeMember{}).
			Select("COUNT(DISTINCT user_id)").
			Scan(&totalMembers).Error; err != nil {
			t.logger.Error("获取成员总数失败", zap.Error(err))
			return fmt.Errorf("获取成员总数失败: %w", err)
		}
		stats.TotalMembers = int(totalMembers)

		// 获取资源总数
		var totalResources int64
		if err := db.Model(&model.TreeNodeResource{}).Count(&totalResources).Error; err != nil {
			t.logger.Error("获取资源总数失败", zap.Error(err))
			return fmt.Errorf("获取资源总数失败: %w", err)
		}
		stats.TotalResources = int(totalResources)

		return nil
	})

	if err != nil {
		return nil, err
	}

	return stats, nil
}

// GetNodeLevel 获取节点层级
func (t *treeDAO) GetNodeLevel(ctx context.Context, nodeId int) (int, error) {
	var level int
	err := t.getDB(ctx).Model(&model.TreeNode{}).
		Select("level").
		Where("id = ?", nodeId).
		Scan(&level).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, errors.New("节点不存在")
		}
		t.logger.Error("获取节点层级失败", zap.Int("id", nodeId), zap.Error(err))
		return 0, fmt.Errorf("获取节点层级失败: %w", err)
	}

	return level, nil
}

// GetLeafNodesByIDs 根据ID列表获取叶子节点
func (t *treeDAO) GetLeafNodesByIDs(ctx context.Context, ids []int) ([]*model.TreeNode, error) {
	if len(ids) == 0 {
		return []*model.TreeNode{}, nil
	}

	var nodes []*model.TreeNode
	if err := t.getDB(ctx).Where("id IN ? AND is_leaf = ?", ids, true).Find(&nodes).Error; err != nil {
		t.logger.Error("获取叶子节点失败", zap.Ints("ids", ids), zap.Error(err))
		return nil, fmt.Errorf("获取叶子节点失败: %w", err)
	}
	return nodes, nil
}

// CreateNode 创建节点
func (t *treeDAO) CreateNode(ctx context.Context, node *model.TreeNode) error {
	// 参数验证
	if node == nil {
		return errors.New("节点信息不能为空")
	}
	if strings.TrimSpace(node.Name) == "" {
		return errors.New("节点名称不能为空")
	}

	// 检查父节点是否存在（如果有父节点）
	if node.ParentID != 0 {
		if err := t.checkNodeExists(ctx, node.ParentID); err != nil {
			return errors.New("父节点不存在")
		}
	}

	// 计算节点层级
	level, err := t.calculateLevel(ctx, node.ParentID)
	if err != nil {
		return err
	}
	node.Level = level

	// 检查节点名称是否在同级重复
	var count int64
	if err := t.getDB(ctx).Model(&model.TreeNode{}).
		Where("name = ? AND parent_id = ?", node.Name, node.ParentID).
		Count(&count).Error; err != nil {
		t.logger.Error("检查节点名称失败", zap.String("name", node.Name), zap.Error(err))
		return fmt.Errorf("检查节点名称失败: %w", err)
	}

	if count > 0 {
		return errors.New("同级节点下已存在相同名称的节点")
	}

	// 设置默认值
	if node.Status == "" {
		node.Status = "active"
	}

	// 使用事务创建节点
	return t.Transaction(ctx, func(txCtx context.Context) error {
		db := t.getDB(txCtx)

		// 创建节点
		if err := db.Create(node).Error; err != nil {
			t.logger.Error("创建节点失败", zap.String("name", node.Name), zap.Error(err))
			return fmt.Errorf("创建节点失败: %w", err)
		}

		// 更新父节点的叶子节点状态
		if node.ParentID != 0 {
			if err := t.updateChildrenIsLeaf(txCtx, node.ParentID); err != nil {
				t.logger.Error("更新父节点叶子状态失败", zap.Int("parentId", node.ParentID), zap.Error(err))
				return err
			}
		}

		return nil
	})
}

// UpdateNode 更新节点
func (t *treeDAO) UpdateNode(ctx context.Context, node *model.TreeNode) error {
	// 参数验证
	if node == nil {
		return errors.New("节点信息不能为空")
	}
	if strings.TrimSpace(node.Name) == "" {
		return errors.New("节点名称不能为空")
	}

	// 检查节点是否存在
	existingNode, err := t.GetNode(ctx, node.ID)
	if err != nil {
		return err
	}

	// 如果父节点ID发生变化，需要验证新的父节点
	if node.ParentID != existingNode.ParentID {
		if node.ParentID != 0 {
			if err := t.checkNodeExists(ctx, node.ParentID); err != nil {
				return errors.New("父节点不存在")
			}
			// 不能将节点移动到自己的子节点下
			if err := t.checkNotDescendant(ctx, node.ID, node.ParentID); err != nil {
				return err
			}
		}

		// 重新计算层级
		level, err := t.calculateLevel(ctx, node.ParentID)
		if err != nil {
			return err
		}
		node.Level = level
	}

	// 检查节点名称是否与同级节点重复
	var count int64
	if err := t.getDB(ctx).Model(&model.TreeNode{}).
		Where("name = ? AND parent_id = ? AND id != ?", node.Name, node.ParentID, node.ID).
		Count(&count).Error; err != nil {
		t.logger.Error("检查节点名称失败", zap.String("name", node.Name), zap.Error(err))
		return fmt.Errorf("检查节点名称失败: %w", err)
	}

	if count > 0 {
		return errors.New("同级节点下已存在相同名称的节点")
	}

	// 使用事务更新
	return t.Transaction(ctx, func(txCtx context.Context) error {
		db := t.getDB(txCtx)

		// 更新节点信息，只更新允许修改的字段
		updateMap := map[string]interface{}{
			"name":        node.Name,
			"description": node.Description,
			"status":      node.Status,
			"parent_id":   node.ParentID,
			"level":       node.Level,
		}

		if err := db.Model(&model.TreeNode{}).Where("id = ?", node.ID).Updates(updateMap).Error; err != nil {
			t.logger.Error("更新节点失败", zap.Int("id", node.ID), zap.Error(err))
			return fmt.Errorf("更新节点失败: %w", err)
		}

		// 如果父节点发生变化，需要更新相关节点的叶子状态
		if node.ParentID != existingNode.ParentID {
			// 更新旧父节点的叶子状态
			if existingNode.ParentID != 0 {
				if err := t.updateChildrenIsLeaf(txCtx, existingNode.ParentID); err != nil {
					return err
				}
			}

			// 更新新父节点的叶子状态
			if node.ParentID != 0 {
				if err := t.updateChildrenIsLeaf(txCtx, node.ParentID); err != nil {
					return err
				}
			}

			// 如果层级发生变化，需要递归更新所有子节点的层级
			if node.Level != existingNode.Level {
				if err := t.updateDescendantLevels(txCtx, node.ID, node.Level); err != nil {
					return err
				}
			}
		}

		return nil
	})
}

// checkNotDescendant 检查targetId不是nodeId的后代节点
func (t *treeDAO) checkNotDescendant(ctx context.Context, nodeId, targetId int) error {
	// 递归查找所有子节点
	var isDescendant bool
	err := t.checkDescendantRecursive(ctx, nodeId, targetId, &isDescendant)
	if err != nil {
		return err
	}

	if isDescendant {
		return errors.New("不能将节点移动到自己的子节点下")
	}

	return nil
}

// checkDescendantRecursive 递归检查是否为后代节点
func (t *treeDAO) checkDescendantRecursive(ctx context.Context, nodeId, targetId int, isDescendant *bool) error {
	if *isDescendant {
		return nil
	}

	var childIds []int
	if err := t.getDB(ctx).Model(&model.TreeNode{}).
		Where("parent_id = ?", nodeId).
		Pluck("id", &childIds).Error; err != nil {
		return fmt.Errorf("查询子节点失败: %w", err)
	}

	for _, childId := range childIds {
		if childId == targetId {
			*isDescendant = true
			return nil
		}
		if err := t.checkDescendantRecursive(ctx, childId, targetId, isDescendant); err != nil {
			return err
		}
	}

	return nil
}

// updateDescendantLevels 递归更新所有后代节点的层级
func (t *treeDAO) updateDescendantLevels(ctx context.Context, parentId, parentLevel int) error {
	var children []*model.TreeNode
	if err := t.getDB(ctx).Where("parent_id = ?", parentId).Find(&children).Error; err != nil {
		return fmt.Errorf("查询子节点失败: %w", err)
	}

	for _, child := range children {
		newLevel := parentLevel + 1
		if err := t.getDB(ctx).Model(&model.TreeNode{}).
			Where("id = ?", child.ID).
			Update("level", newLevel).Error; err != nil {
			return fmt.Errorf("更新子节点层级失败: %w", err)
		}

		// 递归更新子节点的子节点
		if err := t.updateDescendantLevels(ctx, child.ID, newLevel); err != nil {
			return err
		}
	}

	return nil
}

// DeleteNode 删除节点
func (t *treeDAO) DeleteNode(ctx context.Context, id int) error {
	// 检查节点是否存在
	node, err := t.GetNode(ctx, id)
	if err != nil {
		return err
	}

	// 检查是否有子节点
	var childCount int64
	if err := t.getDB(ctx).Model(&model.TreeNode{}).Where("parent_id = ?", id).Count(&childCount).Error; err != nil {
		t.logger.Error("检查子节点失败", zap.Int("id", id), zap.Error(err))
		return fmt.Errorf("检查子节点失败: %w", err)
	}

	if childCount > 0 {
		return errors.New("该节点下存在子节点，无法删除")
	}

	// 使用事务确保操作的原子性
	return t.Transaction(ctx, func(txCtx context.Context) error {
		db := t.getDB(txCtx)

		// 删除节点资源关系
		if err := db.Where("tree_node_id = ?", id).Delete(&model.TreeNodeResource{}).Error; err != nil {
			t.logger.Error("删除资源关系失败", zap.Int("id", id), zap.Error(err))
			return fmt.Errorf("删除资源关系失败: %w", err)
		}

		// 删除节点管理员关系
		if err := db.Where("tree_node_id = ?", id).Delete(&model.TreeNodeAdmin{}).Error; err != nil {
			t.logger.Error("删除管理员关系失败", zap.Int("id", id), zap.Error(err))
			return fmt.Errorf("删除管理员关系失败: %w", err)
		}

		// 删除节点成员关系
		if err := db.Where("tree_node_id = ?", id).Delete(&model.TreeNodeMember{}).Error; err != nil {
			t.logger.Error("删除成员关系失败", zap.Int("id", id), zap.Error(err))
			return fmt.Errorf("删除成员关系失败: %w", err)
		}

		// 删除节点
		if err := db.Delete(&model.TreeNode{}, id).Error; err != nil {
			t.logger.Error("删除节点失败", zap.Int("id", id), zap.Error(err))
			return fmt.Errorf("删除节点失败: %w", err)
		}

		// 更新父节点的叶子节点状态
		if node.ParentID != 0 {
			if err := t.updateChildrenIsLeaf(txCtx, node.ParentID); err != nil {
				t.logger.Error("更新父节点叶子状态失败", zap.Int("parentId", node.ParentID), zap.Error(err))
				return err
			}
		}

		return nil
	})
}

// GetNodeResources 获取节点绑定的资源列表
func (t *treeDAO) GetNodeResources(ctx context.Context, nodeId int) ([]*model.ResourceBase, error) {
	if err := t.checkNodeExists(ctx, nodeId); err != nil {
		return nil, err
	}

	// 获取节点资源关系
	var nodeResources []model.TreeNodeResource
	if err := t.getDB(ctx).Where("tree_node_id = ?", nodeId).Find(&nodeResources).Error; err != nil {
		t.logger.Error("获取节点资源关系失败", zap.Int("nodeId", nodeId), zap.Error(err))
		return nil, fmt.Errorf("获取节点资源关系失败: %w", err)
	}

	if len(nodeResources) == 0 {
		return []*model.ResourceBase{}, nil
	}

	// 收集所有资源ID
	resourceIDs := make([]string, len(nodeResources))
	for i, nr := range nodeResources {
		resourceIDs[i] = nr.ResourceID
	}

	// 查询资源基本信息
	var resources []*model.ResourceBase
	if err := t.getDB(ctx).Where("id IN ?", resourceIDs).Find(&resources).Error; err != nil {
		t.logger.Error("获取资源基本信息失败", zap.Strings("resourceIDs", resourceIDs), zap.Error(err))
		return nil, fmt.Errorf("获取资源基本信息失败: %w", err)
	}

	return resources, nil
}

// BindResource 绑定资源到节点
func (t *treeDAO) BindResource(ctx context.Context, nodeId int, resourceType string, resourceIds []string) error {
	// 检查节点是否存在
	if err := t.checkNodeExists(ctx, nodeId); err != nil {
		return err
	}

	return t.Transaction(ctx, func(txCtx context.Context) error {
		db := t.getDB(txCtx)

		// 批量检查已存在的绑定关系
		var existingBindings []model.TreeNodeResource
		if err := db.Where("tree_node_id = ? AND resource_type = ? AND resource_id IN ?",
			nodeId, resourceType, resourceIds).Find(&existingBindings).Error; err != nil {
			t.logger.Error("查询现有资源绑定失败",
				zap.Int("nodeId", nodeId),
				zap.String("resourceType", resourceType),
				zap.Error(err))
			return fmt.Errorf("查询现有资源绑定失败: %w", err)
		}

		// 创建已存在绑定的映射，用于快速查找
		existingMap := make(map[string]bool)
		for _, binding := range existingBindings {
			existingMap[binding.ResourceID] = true
		}

		// 批量插入新的绑定关系
		var newBindings []model.TreeNodeResource
		for _, resourceId := range resourceIds {
			if strings.TrimSpace(resourceId) == "" {
				t.logger.Warn("跳过空资源ID", zap.Int("nodeId", nodeId))
				continue
			}

			if existingMap[resourceId] {
				t.logger.Debug("资源已存在绑定关系",
					zap.Int("nodeId", nodeId),
					zap.String("resourceId", resourceId),
					zap.String("resourceType", resourceType))
				continue
			}

			newBindings = append(newBindings, model.TreeNodeResource{
				TreeNodeID:   nodeId,
				ResourceID:   resourceId,
				ResourceType: resourceType,
			})
		}

		// 如果有新的绑定关系，批量创建
		if len(newBindings) > 0 {
			if err := db.Create(&newBindings).Error; err != nil {
				t.logger.Error("批量创建资源绑定关系失败",
					zap.Int("nodeId", nodeId),
					zap.String("resourceType", resourceType),
					zap.Int("count", len(newBindings)),
					zap.Error(err))
				return fmt.Errorf("批量创建资源绑定关系失败: %w", err)
			}
		}
		return nil
	})
}

// UnbindResource 解绑资源
func (t *treeDAO) UnbindResource(ctx context.Context, nodeId int, resourceType string, resourceId string) error {
	// 检查节点是否存在
	if err := t.checkNodeExists(ctx, nodeId); err != nil {
		return err
	}

	return t.Transaction(ctx, func(txCtx context.Context) error {
		db := t.getDB(txCtx)
		var count int64
		if err := db.Model(&model.TreeNodeResource{}).
			Where("tree_node_id = ? AND resource_type = ? AND resource_id = ?",
				nodeId, resourceType, resourceId).
			Count(&count).Error; err != nil {
			t.logger.Error("检查资源绑定关系失败",
				zap.Int("nodeId", nodeId),
				zap.String("resourceType", resourceType),
				zap.String("resourceId", resourceId),
				zap.Error(err))
			return fmt.Errorf("检查资源绑定关系失败: %w", err)
		}

		if count == 0 {
			t.logger.Warn("未找到要解绑的资源关系",
				zap.Int("nodeId", nodeId),
				zap.String("resourceType", resourceType),
				zap.String("resourceId", resourceId))
			return errors.New("未找到要解绑的资源关系")
		}

		// 执行删除操作
		result := db.Where("tree_node_id = ? AND resource_type = ? AND resource_id = ?",
			nodeId, resourceType, resourceId).Delete(&model.TreeNodeResource{})

		if result.Error != nil {
			t.logger.Error("解绑资源失败",
				zap.Int("nodeId", nodeId),
				zap.String("resourceType", resourceType),
				zap.String("resourceId", resourceId),
				zap.Error(result.Error))
			return fmt.Errorf("解绑资源失败: %w", result.Error)
		}
		return nil
	})
}

// UpdateNodeStatus 更新节点状态
func (t *treeDAO) UpdateNodeStatus(ctx context.Context, id int, status string) error {
	if status != "active" && status != "inactive" {
		return errors.New("状态只能是active或inactive")
	}

	return t.getDB(ctx).Model(&model.TreeNode{}).Where("id = ?", id).Update("status", status).Error
}

// GetNodeMembers 获取节点成员列表
func (t *treeDAO) GetNodeMembers(ctx context.Context, nodeId int, userId int, memberType string) ([]*model.User, error) {
	var users []*model.User
	db := t.getDB(ctx)

	query := db.Model(&model.User{})

	switch memberType {
	case "admin":
		// 查询管理员
		query = query.Joins("JOIN tree_node_admins ON tree_node_admins.user_id = users.id").
			Where("tree_node_admins.tree_node_id = ?", nodeId)
	case "member":
		// 查询普通成员
		query = query.Joins("JOIN tree_node_members ON tree_node_members.user_id = users.id").
			Where("tree_node_members.tree_node_id = ?", nodeId)
	case "all", "":
		// 查询所有成员（管理员和普通成员）
		subQuery1 := db.Model(&model.TreeNodeAdmin{}).Select("user_id").Where("tree_node_id = ?", nodeId)
		subQuery2 := db.Model(&model.TreeNodeMember{}).Select("user_id").Where("tree_node_id = ?", nodeId)
		query = query.Where("users.id IN (?) OR users.id IN (?)", subQuery1, subQuery2)
	default:
		return nil, errors.New("无效的成员类型，必须是 admin、member 或 all")
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

// AddNodeMember 添加节点成员
func (t *treeDAO) AddNodeMember(ctx context.Context, nodeId int, userId int, memberType string) error {
	// 检查节点是否存在
	if err := t.checkNodeExists(ctx, nodeId); err != nil {
		return err
	}

	// 检查用户是否存在
	if err := t.checkUserExists(ctx, userId); err != nil {
		return err
	}

	db := t.getDB(ctx)

	// 根据成员类型添加关系
	switch memberType {
	case "admin":
		// 检查管理员关系是否已存在
		var count int64
		if err := db.Model(&model.TreeNodeAdmin{}).
			Where("tree_node_id = ? AND user_id = ?", nodeId, userId).
			Count(&count).Error; err != nil {
			t.logger.Error("检查管理员关系失败", zap.Int("nodeId", nodeId), zap.Int("userId", userId), zap.Error(err))
			return fmt.Errorf("检查管理员关系失败: %w", err)
		}

		if count > 0 {
			return errors.New("用户已经是该节点的管理员")
		}

		// 创建管理员关系
		admin := &model.TreeNodeAdmin{
			TreeNodeID: nodeId,
			UserID:     userId,
		}
		if err := db.Create(admin).Error; err != nil {
			t.logger.Error("添加节点管理员失败", zap.Int("nodeId", nodeId), zap.Int("userId", userId), zap.Error(err))
			return fmt.Errorf("添加节点管理员失败: %w", err)
		}

	case "member":
		// 检查成员关系是否已存在
		var count int64
		if err := db.Model(&model.TreeNodeMember{}).
			Where("tree_node_id = ? AND user_id = ?", nodeId, userId).
			Count(&count).Error; err != nil {
			t.logger.Error("检查成员关系失败", zap.Int("nodeId", nodeId), zap.Int("userId", userId), zap.Error(err))
			return fmt.Errorf("检查成员关系失败: %w", err)
		}

		if count > 0 {
			return errors.New("用户已经是该节点的成员")
		}

		// 创建成员关系
		member := &model.TreeNodeMember{
			TreeNodeID: nodeId,
			UserID:     userId,
		}
		if err := db.Create(member).Error; err != nil {
			t.logger.Error("添加节点成员失败", zap.Int("nodeId", nodeId), zap.Int("userId", userId), zap.Error(err))
			return fmt.Errorf("添加节点成员失败: %w", err)
		}

	default:
		return errors.New("无效的成员类型，必须是 admin 或 member")
	}

	return nil
}

// RemoveNodeMember 移除节点成员
func (t *treeDAO) RemoveNodeMember(ctx context.Context, nodeId int, userId int, memberType string) error {
	// 检查节点是否存在
	if err := t.checkNodeExists(ctx, nodeId); err != nil {
		return err
	}

	db := t.getDB(ctx)
	var result *gorm.DB

	switch memberType {
	case "admin":
		// 删除管理员关系
		result = db.Where("tree_node_id = ? AND user_id = ?", nodeId, userId).Delete(&model.TreeNodeAdmin{})
	case "member":
		// 删除成员关系
		result = db.Where("tree_node_id = ? AND user_id = ?", nodeId, userId).Delete(&model.TreeNodeMember{})
	default:
		return errors.New("无效的成员类型，必须是 admin 或 member")
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

// GetUserNodes 获取用户相关的节点
func (t *treeDAO) GetUserNodes(ctx context.Context, userId int, role string) ([]*model.TreeNode, error) {
	var nodes []*model.TreeNode
	db := t.getDB(ctx)

	query := db.Model(&model.TreeNode{})

	switch role {
	case "admin":
		// 查询用户作为管理员的节点
		query = query.Joins("JOIN tree_node_admins ON tree_node_admins.tree_node_id = tree_nodes.id").
			Where("tree_node_admins.user_id = ?", userId)
	case "member":
		// 查询用户作为成员的节点
		query = query.Joins("JOIN tree_node_members ON tree_node_members.tree_node_id = tree_nodes.id").
			Where("tree_node_members.user_id = ?", userId)
	case "all", "":
		// 查询用户相关的所有节点
		subQuery1 := db.Model(&model.TreeNodeAdmin{}).Select("tree_node_id").Where("user_id = ?", userId)
		subQuery2 := db.Model(&model.TreeNodeMember{}).Select("tree_node_id").Where("user_id = ?", userId)
		query = query.Where("tree_nodes.id IN (?) OR tree_nodes.id IN (?)", subQuery1, subQuery2)
	default:
		return nil, errors.New("无效的角色类型，必须是 admin、member 或 all")
	}

	if err := query.Order("tree_nodes.level ASC, tree_nodes.name ASC").Find(&nodes).Error; err != nil {
		t.logger.Error("获取用户节点失败", zap.Int("userId", userId), zap.String("role", role), zap.Error(err))
		return nil, fmt.Errorf("获取用户节点失败: %w", err)
	}

	return nodes, nil
}

// Transaction 提供事务支持
func (t *treeDAO) Transaction(ctx context.Context, fn func(ctx context.Context) error) error {
	// 如果已经在事务中，直接执行函数
	if _, ok := ctx.Value(txKey).(*gorm.DB); ok {
		return fn(ctx)
	}

	// 开始新事务
	return t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txCtx := context.WithValue(ctx, txKey, tx)
		return fn(txCtx)
	})
}
