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
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	treeUtils "github.com/GoSimplicity/AI-CloudOps/internal/tree/utils"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type TreeNodeDAO interface {
	// 基础查询方法
	GetTreeList(ctx context.Context, req *model.GetTreeNodeListReq) ([]*model.TreeNode, int64, error)
	GetNode(ctx context.Context, id int) (*model.TreeNode, error)
	GetChildNodes(ctx context.Context, parentID int) ([]*model.TreeNode, error)
	GetTreeStatistics(ctx context.Context) (*model.TreeNodeStatisticsResp, error)

	// 节点管理方法
	CreateNode(ctx context.Context, node *model.TreeNode) error
	UpdateNode(ctx context.Context, node *model.TreeNode) error
	DeleteNode(ctx context.Context, id int) error

	// 资源管理方法
	BindResource(ctx context.Context, nodeId int, resourceIds []int) error
	UnbindResource(ctx context.Context, nodeId int, resourceId int) error

	// 成员管理方法
	GetNodeMembers(ctx context.Context, nodeId int, memberType string) ([]*model.User, error)
	AddNodeMember(ctx context.Context, nodeId int, userId int, memberType model.TreeNodeMemberType) error
	RemoveNodeMember(ctx context.Context, nodeId int, userId int, memberType model.TreeNodeMemberType) error
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
	if req.Status != 0 {
		query = query.Where("status = ?", req.Status)
	}
	if strings.TrimSpace(req.Search) != "" {
		like := "%" + strings.TrimSpace(req.Search) + "%"
		query = query.Where("name LIKE ? OR description LIKE ?", like, like)
	}

	// 获取总数
	if err := query.Count(&count).Error; err != nil {
		t.logger.Error("获取树节点总数失败", zap.Error(err))
		return nil, 0, err
	}

	// 分页查询
	offset := (req.Page - 1) * req.Size
	if err := query.
		Preload("AdminUsers").
		Preload("MemberUsers").
		Preload("TreeLocalResources").
		Order("level ASC, parent_id ASC, name ASC").
		Limit(req.Size).
		Offset(offset).
		Find(&nodes).Error; err != nil {
		t.logger.Error("获取树节点列表失败", zap.Error(err))
		return nil, 0, err
	}

	// 如果指定了层级，直接返回列表（已分页）
	if req.Level > 0 {
		return nodes, count, nil
	}

	// 构建树形结构（基于已分页的数据）
	return treeUtils.BuildTreeStructure(nodes), count, nil
}

// GetNode 获取节点详情
func (t *treeNodeDAO) GetNode(ctx context.Context, id int) (*model.TreeNode, error) {
	var node model.TreeNode
	if err := t.db.WithContext(ctx).
		Preload("AdminUsers").
		Preload("MemberUsers").
		Preload("TreeLocalResources").
		Where("id = ?", id).
		First(&node).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("节点不存在")
		}
		t.logger.Error("获取节点失败", zap.Int("id", id), zap.Error(err))
		return nil, err
	}

	return &node, nil
}

// GetChildNodes 获取直接子节点列表
func (t *treeNodeDAO) GetChildNodes(ctx context.Context, parentID int) ([]*model.TreeNode, error) {
	var nodes []*model.TreeNode
	if err := t.db.WithContext(ctx).
		Where("parent_id = ?", parentID).
		Order("name ASC").
		Find(&nodes).Error; err != nil {
		t.logger.Error("获取子节点失败", zap.Int("parentID", parentID), zap.Error(err))
		return nil, err
	}
	return nodes, nil
}

// GetTreeStatistics 获取服务树统计数据
func (t *treeNodeDAO) GetTreeStatistics(ctx context.Context) (*model.TreeNodeStatisticsResp, error) {
	var stats model.TreeNodeStatisticsResp
	var count int64

	// 节点总数
	if err := t.db.WithContext(ctx).Model(&model.TreeNode{}).Count(&count).Error; err != nil {
		t.logger.Error("统计节点总数失败", zap.Error(err))
	} else {
		stats.TotalNodes = int(count)
	}

	// 活跃节点数
	count = 0
	if err := t.db.WithContext(ctx).Model(&model.TreeNode{}).Where("status = ?", model.ACTIVE).Count(&count).Error; err != nil {
		t.logger.Error("统计活跃节点失败", zap.Error(err))
	} else {
		stats.ActiveNodes = int(count)
	}

	// 非活跃节点数
	count = 0
	if err := t.db.WithContext(ctx).Model(&model.TreeNode{}).Where("status = ?", model.INACTIVE).Count(&count).Error; err != nil {
		t.logger.Error("统计非活跃节点失败", zap.Error(err))
	} else {
		stats.InactiveNodes = int(count)
	}

	// 资源总数
	count = 0
	if err := t.db.WithContext(ctx).Model(&model.TreeLocalResource{}).Count(&count).Error; err != nil {
		t.logger.Error("统计资源总数失败", zap.Error(err))
	} else {
		stats.TotalResources = int(count)
	}

	// 管理员总数（关联关系条目数）
	count = 0
	if err := t.db.WithContext(ctx).Table("cl_tree_node_admin").Count(&count).Error; err != nil {
		t.logger.Error("统计管理员总数失败", zap.Error(err))
	} else {
		stats.TotalAdmins = int(count)
	}

	// 成员总数（关联关系条目数）
	count = 0
	if err := t.db.WithContext(ctx).Table("cl_tree_node_member").Count(&count).Error; err != nil {
		t.logger.Error("统计成员总数失败", zap.Error(err))
	} else {
		stats.TotalMembers = int(count)
	}

	return &stats, nil
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
			return err
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
			return err
		}
		level = parentLevel + 1
	}
	node.Level = level

	// 检查同级节点名称唯一性
	var count int64
	if err := t.db.WithContext(ctx).Model(&model.TreeNode{}).
		Where("name = ? AND parent_id = ?", node.Name, node.ParentID).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New("同级节点下已存在相同名称的节点")
	}

	// 设置默认状态
	if node.Status == 0 {
		node.Status = model.ACTIVE
	}

	// 新创建的节点默认是叶子节点
	if node.IsLeaf == 0 {
		node.IsLeaf = model.IsLeafYes
	}

	// 创建节点
	if err := t.db.WithContext(ctx).Create(node).Error; err != nil {
		t.logger.Error("创建节点失败", zap.String("name", node.Name), zap.Error(err))
		return err
	}

	// 更新父节点的叶子状态
	if node.ParentID != 0 {
		if err := t.db.WithContext(ctx).Model(&model.TreeNode{}).
			Where("id = ?", node.ParentID).
			Update("is_leaf", 2).Error; err != nil {
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
				return err
			}
			if count == 0 {
				return errors.New("父节点不存在")
			}

			// 防止循环依赖
			if node.ParentID == node.ID {
				return errors.New("不能将节点移动到自己下")
			}

			// 防止将节点移动到其子孙节点下：沿新父节点向上回溯，若遇到自身则非法
			cur := node.ParentID
			for cur != 0 {
				if cur == node.ID {
					return errors.New("不能将节点移动到其子孙节点下")
				}
				var pID int
				if err := t.db.WithContext(ctx).Model(&model.TreeNode{}).
					Select("parent_id").Where("id = ?", cur).Scan(&pID).Error; err != nil {
					return err
				}
				cur = pID
			}
		}

		// 重新计算层级
		level := 1
		if node.ParentID != 0 {
			var parentLevel int
			if err := t.db.WithContext(ctx).Model(&model.TreeNode{}).
				Select("level").Where("id = ?", node.ParentID).Scan(&parentLevel).Error; err != nil {
				return err
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
		return err
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
		"is_leaf":     node.IsLeaf,
	}

	if err := t.db.WithContext(ctx).Model(&model.TreeNode{}).Where("id = ?", node.ID).Updates(updateMap).Error; err != nil {
		t.logger.Error("更新节点失败", zap.Int("id", node.ID), zap.Error(err))
		return err
	}

	// 如果父节点发生变化，需要更新相关节点的叶子状态
	if node.ParentID != existingNode.ParentID {
		// 更新原父节点的叶子状态
		if existingNode.ParentID != 0 {
			var remainingChildren int64
			if err := t.db.WithContext(ctx).Model(&model.TreeNode{}).Where("parent_id = ?", existingNode.ParentID).Count(&remainingChildren).Error; err == nil {
				isLeaf := model.IsLeafNo // 默认不是叶子节点
				if remainingChildren == 0 {
					isLeaf = model.IsLeafYes // 是叶子节点
				}
				t.db.WithContext(ctx).Model(&model.TreeNode{}).Where("id = ?", existingNode.ParentID).Update("is_leaf", isLeaf)
			}
		}

		// 更新新父节点的叶子状态
		if node.ParentID != 0 {
			t.db.WithContext(ctx).Model(&model.TreeNode{}).Where("id = ?", node.ParentID).Update("is_leaf", model.IsLeafNo)
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
		return err
	}

	if childCount > 0 {
		return errors.New("该节点下存在子节点，无法删除")
	}

	return t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 清理管理员关联
		if err := tx.Exec("DELETE FROM cl_tree_node_admin WHERE tree_node_id = ?", id).Error; err != nil {
			t.logger.Error("清理管理员关联失败", zap.Int("id", id), zap.Error(err))
			return err
		}

		// 清理成员关联
		if err := tx.Exec("DELETE FROM cl_tree_node_member WHERE tree_node_id = ?", id).Error; err != nil {
			t.logger.Error("清理成员关联失败", zap.Int("id", id), zap.Error(err))
			return err
		}

		// 清理资源关联
		if err := tx.Exec("DELETE FROM cl_tree_node_local WHERE tree_node_id = ?", id).Error; err != nil {
			t.logger.Error("清理资源关联失败", zap.Int("id", id), zap.Error(err))
			return err
		}

		// 删除节点
		if err := tx.Delete(&model.TreeNode{}, id).Error; err != nil {
			t.logger.Error("删除节点失败", zap.Int("id", id), zap.Error(err))
			return err
		}

		// 更新父节点的叶子节点状态
		if node.ParentID != 0 {
			var remainingChildren int64
			if err := tx.Model(&model.TreeNode{}).Where("parent_id = ?", node.ParentID).Count(&remainingChildren).Error; err != nil {
				t.logger.Error("检查剩余子节点失败", zap.Int("parentId", node.ParentID), zap.Error(err))
				return err
			}

			isLeaf := model.IsLeafNo // 默认不是叶子节点
			if remainingChildren == 0 {
				isLeaf = model.IsLeafYes // 是叶子节点
			}
			if err := tx.Model(&model.TreeNode{}).Where("id = ?", node.ParentID).Update("is_leaf", isLeaf).Error; err != nil {
				t.logger.Error("更新父节点叶子状态失败", zap.Int("parentId", node.ParentID), zap.Error(err))
				return err
			}
		}

		return nil
	})
}

// BindResource 绑定资源到节点
func (t *treeNodeDAO) BindResource(ctx context.Context, nodeId int, resourceIds []int) error {
	// 验证资源ID列表
	if err := treeUtils.ValidateResourceIDs(resourceIds); err != nil {
		return err
	}

	// 验证节点存在
	var count int64
	if err := t.db.WithContext(ctx).Model(&model.TreeNode{}).Where("id = ?", nodeId).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return errors.New("节点不存在")
	}

	// 验证资源存在
	var validResourceIds []int
	if err := t.db.WithContext(ctx).Model(&model.TreeLocalResource{}).
		Where("id IN ?", resourceIds).
		Pluck("id", &validResourceIds).Error; err != nil {
		return err
	}

	if len(validResourceIds) == 0 {
		return errors.New("没有找到有效的资源")
	}

	// 获取节点信息并添加关联
	var node model.TreeNode
	if err := t.db.WithContext(ctx).First(&node, nodeId).Error; err != nil {
		return err
	}

	// 获取要绑定的资源
	var resources []*model.TreeLocalResource
	if err := t.db.WithContext(ctx).Where("id IN ?", validResourceIds).Find(&resources).Error; err != nil {
		return err
	}

	// 绑定资源
	if err := t.db.WithContext(ctx).Model(&node).Association("TreeLocalResources").Append(resources); err != nil {
		t.logger.Error("绑定资源失败", zap.Int("nodeId", nodeId), zap.Ints("resourceIds", validResourceIds), zap.Error(err))
		return err
	}

	return nil
}

// UnbindResource 解绑资源
func (t *treeNodeDAO) UnbindResource(ctx context.Context, nodeId int, resourceId int) error {
	// 获取节点信息
	var node model.TreeNode
	if err := t.db.WithContext(ctx).First(&node, nodeId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("节点不存在")
		}
		return err
	}

	// 获取要解绑的资源
	var resource model.TreeLocalResource
	if err := t.db.WithContext(ctx).First(&resource, resourceId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("资源不存在")
		}
		return err
	}

	// 解绑资源
	if err := t.db.WithContext(ctx).Model(&node).Association("TreeLocalResources").Delete(&resource); err != nil {
		t.logger.Error("解绑资源失败", zap.Int("nodeId", nodeId), zap.Int("resourceId", resourceId), zap.Error(err))
		return err
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
		return nil, err
	}

	var users []*model.User
	db := t.db.WithContext(ctx)

	switch memberType {
	case "admin":
		if err := db.Model(&node).Association("AdminUsers").Find(&users); err != nil {
			t.logger.Error("获取管理员失败", zap.Int("nodeId", nodeId), zap.Error(err))
			return nil, err
		}
	case "member":
		if err := db.Model(&node).Association("MemberUsers").Find(&users); err != nil {
			t.logger.Error("获取成员失败", zap.Int("nodeId", nodeId), zap.Error(err))
			return nil, err
		}
	case "all", "":
		// 获取所有用户（管理员+成员）
		var adminUsers []*model.User
		var memberUsers []*model.User

		if err := db.Model(&node).Association("AdminUsers").Find(&adminUsers); err != nil {
			t.logger.Error("获取管理员失败", zap.Int("nodeId", nodeId), zap.Error(err))
			return nil, err
		}

		if err := db.Model(&node).Association("MemberUsers").Find(&memberUsers); err != nil {
			t.logger.Error("获取成员失败", zap.Int("nodeId", nodeId), zap.Error(err))
			return nil, err
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
func (t *treeNodeDAO) AddNodeMember(ctx context.Context, nodeId int, userId int, memberType model.TreeNodeMemberType) error {
	var node model.TreeNode
	if err := t.db.WithContext(ctx).First(&node, nodeId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("节点不存在")
		}
		return err
	}

	var user model.User
	if err := t.db.WithContext(ctx).First(&user, userId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("用户不存在")
		}
		return err
	}

	db := t.db.WithContext(ctx)

	switch memberType {
	case model.AdminRole:
		// 检查是否已存在
		var count int64
		if err := db.Table("cl_tree_node_admin").Where("tree_node_id = ? AND user_id = ?", nodeId, userId).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return errors.New("用户已经是该节点的管理员")
		}

		if err := db.Model(&node).Association("AdminUsers").Append(&user); err != nil {
			t.logger.Error("添加节点管理员失败", zap.Int("nodeId", nodeId), zap.Int("userId", userId), zap.Error(err))
			return err
		}

	case model.MemberRole:
		// 检查是否已存在
		var count int64
		if err := db.Table("cl_tree_node_member").Where("tree_node_id = ? AND user_id = ?", nodeId, userId).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return errors.New("用户已经是该节点的成员")
		}

		if err := db.Model(&node).Association("MemberUsers").Append(&user); err != nil {
			t.logger.Error("添加节点成员失败", zap.Int("nodeId", nodeId), zap.Int("userId", userId), zap.Error(err))
			return err
		}

	default:
		return errors.New("无效的成员类型，必须是 AdminRole 或 MemberRole")
	}

	return nil
}

// RemoveNodeMember 移除节点成员
func (t *treeNodeDAO) RemoveNodeMember(ctx context.Context, nodeId int, userId int, memberType model.TreeNodeMemberType) error {
	var node model.TreeNode
	if err := t.db.WithContext(ctx).First(&node, nodeId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("节点不存在")
		}
		return err
	}

	var user model.User
	if err := t.db.WithContext(ctx).First(&user, userId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("用户不存在")
		}
		return err
	}

	db := t.db.WithContext(ctx)

	switch memberType {
	case model.AdminRole:
		if err := db.Model(&node).Association("AdminUsers").Delete(&user); err != nil {
			t.logger.Error("移除节点管理员失败", zap.Int("nodeId", nodeId), zap.Int("userId", userId), zap.Error(err))
			return err
		}
	case model.MemberRole:
		if err := db.Model(&node).Association("MemberUsers").Delete(&user); err != nil {
			t.logger.Error("移除节点成员失败", zap.Int("nodeId", nodeId), zap.Int("userId", userId), zap.Error(err))
			return err
		}
	default:
		return errors.New("无效的成员类型，必须是 AdminRole 或 MemberRole")
	}

	return nil
}
