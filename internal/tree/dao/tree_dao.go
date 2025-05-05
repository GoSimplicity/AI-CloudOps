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
	// 树结构相关接口
	GetTreeList(ctx context.Context, req *model.TreeNodeListReq) ([]*model.TreeNode, error)
	GetNodeDetail(ctx context.Context, id int) (*model.TreeNode, error)
	GetChildNodes(ctx context.Context, parentId int) ([]*model.TreeNode, error)
	GetNodePath(ctx context.Context, nodeId int) ([]*model.TreeNode, error)
	GetTreeStatistics(ctx context.Context) (*model.TreeStatisticsResp, error)
	GetNodeLevel(ctx context.Context, nodeId int) (int, error)

	// 节点管理接口
	CreateNode(ctx context.Context, node *model.TreeNode) error
	UpdateNode(ctx context.Context, node *model.TreeNode) error
	DeleteNode(ctx context.Context, id int) error
	UpdateNodePath(ctx context.Context, nodeId int, path string) error

	// 资源绑定接口
	GetNodeResources(ctx context.Context, nodeId int) ([]*model.TreeNodeResourceResp, error)
	BindResource(ctx context.Context, nodeId int, resourceType string, resourceIds []string) error
	UnbindResource(ctx context.Context, nodeId int, resourceType string, resourceId string) error
	GetResourceTypes(ctx context.Context) ([]string, error)

	// 成员管理接口
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
func (t *treeDAO) checkNodeExists(id int) error {
	var node model.TreeNode
	if err := t.db.First(&node, id).Error; err != nil {
		t.logger.Error("节点不存在", zap.Int("nodeId", id), zap.Error(err))
		return errors.New("节点不存在")
	}
	return nil
}

// AddNodeMember 添加节点成员
func (t *treeDAO) AddNodeMember(ctx context.Context, nodeId int, userId int, memberType string) error {
	// 检查节点是否存在
	if err := t.checkNodeExists(nodeId); err != nil {
		return err
	}

	// 检查用户是否存在
	var user model.User
	if err := t.db.First(&user, userId).Error; err != nil {
		t.logger.Error("用户不存在", zap.Int("userId", userId), zap.Error(err))
		return errors.New("用户不存在")
	}

	// 根据成员类型添加关系
	var insertQuery string
	var params []interface{}

	if memberType == "admin" {
		insertQuery = "INSERT INTO tree_node_admins (tree_node_id, user_id) VALUES (?, ?) ON DUPLICATE KEY UPDATE tree_node_id=tree_node_id"
		params = []interface{}{nodeId, userId}
	} else if memberType == "member" {
		insertQuery = "INSERT INTO tree_node_members (tree_node_id, user_id) VALUES (?, ?) ON DUPLICATE KEY UPDATE tree_node_id=tree_node_id"
		params = []interface{}{nodeId, userId}
	} else {
		return errors.New("无效的成员类型")
	}

	if err := t.db.Exec(insertQuery, params...).Error; err != nil {
		t.logger.Error("添加节点成员失败", zap.Int("nodeId", nodeId), zap.Int("userId", userId), zap.String("memberType", memberType), zap.Error(err))
		return err
	}

	return nil
}

// BindResource 绑定资源到节点
func (t *treeDAO) BindResource(ctx context.Context, nodeId int, resourceType string, resourceIds []string) error {
	// 检查节点是否存在
	if err := t.checkNodeExists(nodeId); err != nil {
		return err
	}

	// 开启事务
	tx := t.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 为每个资源ID创建绑定关系
	for _, resourceId := range resourceIds {
		// 检查资源是否已经绑定到任何节点(包括当前节点)
		var existingBindings []struct {
			TreeNodeID int
		}

		if err := tx.Table("tree_node_resources").
			Select("tree_node_id").
			Where("resource_id = ? AND resource_type = ?", resourceId, resourceType).
			Find(&existingBindings).Error; err != nil {
			tx.Rollback()
			t.logger.Error("检查资源绑定状态失败", zap.String("resourceId", resourceId), zap.Error(err))
			return err
		}

		// 如果资源已绑定到别的节点(不包括当前节点)，报错
		for _, binding := range existingBindings {
			if binding.TreeNodeID != nodeId {
				tx.Rollback()
				return fmt.Errorf("资源 %s 已经绑定到节点 %d", resourceId, binding.TreeNodeID)
			}
		}

		// 如果已经绑定到当前节点，则跳过
		alreadyBound := false
		for _, binding := range existingBindings {
			if binding.TreeNodeID == nodeId {
				alreadyBound = true
				break
			}
		}

		if alreadyBound {
			continue
		}

		// 创建绑定关系
		if err := tx.Exec("INSERT INTO tree_node_resources (tree_node_id, resource_id, resource_type) VALUES (?, ?, ?)",
			nodeId, resourceId, resourceType).Error; err != nil {
			tx.Rollback()
			t.logger.Error("绑定资源失败", zap.Int("nodeId", nodeId), zap.String("resourceId", resourceId), zap.Error(err))
			return err
		}
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		t.logger.Error("提交事务失败", zap.Error(err))
		return err
	}

	return nil
}

// CreateNode 创建节点
func (t *treeDAO) CreateNode(ctx context.Context, node *model.TreeNode) error {
	// 检查父节点是否存在
	if node.ParentID != 0 {
		if err := t.checkNodeExists(node.ParentID); err != nil {
			return errors.New("父节点不存在")
		}
	}

	// 检查节点名称是否重复
	var count int64
	if err := t.db.Model(&model.TreeNode{}).Where("name = ? AND parent_id = ?", node.Name, node.ParentID).Count(&count).Error; err != nil {
		t.logger.Error("检查节点名称失败", zap.String("name", node.Name), zap.Error(err))
		return err
	}

	if count > 0 {
		return errors.New("同级节点下已存在相同名称的节点")
	}

	// 创建节点
	if err := t.db.Create(node).Error; err != nil {
		t.logger.Error("创建节点失败", zap.String("name", node.Name), zap.Error(err))
		return err
	}

	return nil
}

// DeleteNode 删除节点
func (t *treeDAO) DeleteNode(ctx context.Context, id int) error {
	// 检查节点是否存在
	if err := t.checkNodeExists(id); err != nil {
		return err
	}

	// 检查是否有子节点
	var childCount int64
	if err := t.db.Model(&model.TreeNode{}).Where("parent_id = ?", id).Count(&childCount).Error; err != nil {
		t.logger.Error("检查子节点失败", zap.Int("id", id), zap.Error(err))
		return err
	}

	if childCount > 0 {
		return errors.New("该节点下存在子节点，无法删除")
	}

	// 检查是否有绑定的资源
	var resourceCount int64
	if err := t.db.Table("tree_node_resources").Where("tree_node_id = ?", id).Count(&resourceCount).Error; err != nil {
		t.logger.Error("检查绑定资源失败", zap.Int("id", id), zap.Error(err))
		return err
	}

	if resourceCount > 0 {
		return errors.New("该节点下存在绑定的资源，无法删除")
	}

	// 开启事务
	tx := t.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 删除节点成员关系
	if err := tx.Exec("DELETE FROM tree_node_admins WHERE tree_node_id = ?", id).Error; err != nil {
		tx.Rollback()
		t.logger.Error("删除管理员关系失败", zap.Int("id", id), zap.Error(err))
		return err
	}

	if err := tx.Exec("DELETE FROM tree_node_members WHERE tree_node_id = ?", id).Error; err != nil {
		tx.Rollback()
		t.logger.Error("删除成员关系失败", zap.Int("id", id), zap.Error(err))
		return err
	}

	// 删除节点
	if err := tx.Delete(&model.TreeNode{}, id).Error; err != nil {
		tx.Rollback()
		t.logger.Error("删除节点失败", zap.Int("id", id), zap.Error(err))
		return err
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		t.logger.Error("提交事务失败", zap.Error(err))
		return err
	}

	return nil
}

// GetChildNodes 获取子节点列表
func (t *treeDAO) GetChildNodes(ctx context.Context, parentId int) ([]*model.TreeNode, error) {
	var nodes []*model.TreeNode
	if err := t.db.WithContext(ctx).Where("parent_id = ?", parentId).Find(&nodes).Error; err != nil {
		t.logger.Error("获取子节点失败", zap.Int("parentId", parentId), zap.Error(err))
		return nil, err
	}

	// 使用一个查询获取所有子节点的子节点数和资源数
	if len(nodes) > 0 {
		var nodeIds []int
		for _, node := range nodes {
			nodeIds = append(nodeIds, int(node.ID))
		}

		// 获取父节点的路径，用于构建子节点路径前缀
		var parentNode model.TreeNode
		parentPath := ""
		if parentId > 0 {
			if err := t.db.WithContext(ctx).Select("path").First(&parentNode, parentId).Error; err == nil {
				parentPath = parentNode.Path
			} else {
				t.logger.Warn("获取父节点路径失败", zap.Int("parentId", parentId), zap.Error(err))
			}
		}

		// 获取子节点数 - 使用路径前缀优化查询
		childCounts := make(map[int]int)
		for _, node := range nodes {
			// 构建子节点路径
			nodePath := fmt.Sprintf("%s/%d", parentPath, node.ID)
			if parentPath == "" {
				nodePath = fmt.Sprintf("/%d", node.ID)
			}

			// 使用LIKE查询获取直接子节点数量
			var count int64
			pathPattern := nodePath + "/%"
			// 不包括更深层级的节点，只计算直接子节点
			if err := t.db.WithContext(ctx).Model(&model.TreeNode{}).Where("path LIKE ? AND path NOT LIKE ?", pathPattern, pathPattern+"/%").Count(&count).Error; err != nil {
				t.logger.Error("获取子节点数量失败", zap.Int("nodeId", int(node.ID)), zap.Error(err))
				continue
			}
			childCounts[int(node.ID)] = int(count)
		}

		// 获取资源数
		resourceCounts := make(map[int]int)
		rows, err := t.db.WithContext(ctx).Raw(`
			 SELECT tree_node_id, COUNT(*) as count 
			 FROM tree_node_resources 
			 WHERE tree_node_id IN (?)
			 GROUP BY tree_node_id
		 `, nodeIds).Rows()

		if err != nil {
			t.logger.Error("获取资源数量失败", zap.Error(err))
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			var nodeID int
			var count int
			if err := rows.Scan(&nodeID, &count); err != nil {
				t.logger.Error("扫描资源数量失败", zap.Error(err))
				return nil, err
			}
			resourceCounts[nodeID] = count
		}

		// 更新节点的子节点数和资源数
		for _, node := range nodes {
			node.ChildCount = childCounts[int(node.ID)]
			node.ResourceCount = resourceCounts[int(node.ID)]
		}
	}

	return nodes, nil
}

// GetNodeDetail 获取节点详情
func (t *treeDAO) GetNodeDetail(ctx context.Context, id int) (*model.TreeNode, error) {
	var node model.TreeNode
	if err := t.db.First(&node, id).Error; err != nil {
		t.logger.Error("节点不存在", zap.Int("id", id), zap.Error(err))
		return nil, errors.New("节点不存在")
	}

	// 获取父节点名称
	if node.ParentID != 0 {
		var parentNode model.TreeNode
		if err := t.db.Select("name").First(&parentNode, node.ParentID).Error; err == nil {
			node.ParentName = parentNode.Name
		}
	}

	// 获取管理员列表
	var admins []*model.User
	if err := t.db.Table("users").
		Select("users.*").
		Joins("JOIN tree_node_admins ON tree_node_admins.user_id = users.id").
		Where("tree_node_admins.tree_node_id = ?", id).
		Find(&admins).Error; err != nil {
		t.logger.Error("获取管理员列表失败", zap.Int("id", id), zap.Error(err))
		return nil, err
	}
	node.Admins = admins

	// 获取成员列表
	var members []*model.User
	if err := t.db.Table("users").
		Select("users.*").
		Joins("JOIN tree_node_members ON tree_node_members.user_id = users.id").
		Where("tree_node_members.tree_node_id = ?", id).
		Find(&members).Error; err != nil {
		t.logger.Error("获取成员列表失败", zap.Int("id", id), zap.Error(err))
		return nil, err
	}
	node.Members = members

	// 获取子节点数量
	var childCount int64
	if err := t.db.Model(&model.TreeNode{}).Where("parent_id = ?", id).Count(&childCount).Error; err != nil {
		t.logger.Error("获取子节点数量失败", zap.Int("id", id), zap.Error(err))
		return nil, err
	}
	node.ChildCount = int(childCount)

	// 获取资源数量
	var resourceCount int64
	if err := t.db.Table("tree_node_resources").Where("tree_node_id = ?", id).Count(&resourceCount).Error; err != nil {
		t.logger.Error("获取资源数量失败", zap.Int("id", id), zap.Error(err))
		return nil, err
	}
	node.ResourceCount = int(resourceCount)

	return &node, nil
}

// GetNodeMembers 获取节点成员列表
func (t *treeDAO) GetNodeMembers(ctx context.Context, nodeId int, userId int, memberType string) ([]*model.User, error) {
	// 检查节点是否存在
	if err := t.checkNodeExists(nodeId); err != nil {
		return nil, err
	}

	// 使用UNION查询获取所有成员（包括管理员和普通成员）
	var users []*model.User
	if err := t.db.Raw(`
		 SELECT DISTINCT u.* FROM users u
		 WHERE u.id IN (
			 SELECT user_id FROM tree_node_admins WHERE tree_node_id = ?
			 UNION
			 SELECT user_id FROM tree_node_members WHERE tree_node_id = ?
		 )
	 `, nodeId, nodeId).Find(&users).Error; err != nil {
		t.logger.Error("获取节点成员失败", zap.Int("nodeId", nodeId), zap.Error(err))
		return nil, err
	}

	return users, nil
}

// GetNodePath 获取节点路径
func (t *treeDAO) GetNodePath(ctx context.Context, nodeId int) ([]*model.TreeNode, error) {
	var path []*model.TreeNode

	// 获取当前节点
	var currentNode model.TreeNode
	if err := t.db.First(&currentNode, nodeId).Error; err != nil {
		t.logger.Error("节点不存在", zap.Int("nodeId", nodeId), zap.Error(err))
		return nil, errors.New("节点不存在")
	}

	// 添加当前节点到路径
	path = append(path, &currentNode)

	// 递归获取父节点
	for currentNode.ParentID != 0 {
		var parentNode model.TreeNode
		if err := t.db.First(&parentNode, currentNode.ParentID).Error; err != nil {
			t.logger.Error("获取父节点失败", zap.Int("parentId", currentNode.ParentID), zap.Error(err))
			break
		}
		path = append([]*model.TreeNode{&parentNode}, path...)
		currentNode = parentNode
	}

	return path, nil
}

// GetNodeResources 获取节点绑定的资源列表
func (t *treeDAO) GetNodeResources(ctx context.Context, nodeId int) ([]*model.TreeNodeResourceResp, error) {
	// 检查节点是否存在
	if err := t.checkNodeExists(nodeId); err != nil {
		return nil, err
	}

	// 查询节点绑定的所有资源
	var resources []*model.TreeNodeResourceResp
	rows, err := t.db.Raw(`
		 SELECT r.id, r.resource_id, r.resource_type, 
		 CASE 
			 WHEN r.resource_type = 'ecs' THEN e.name
			 WHEN r.resource_type = 'rds' THEN d.name
			 WHEN r.resource_type = 'vpc' THEN v.name
			 ELSE 'unknown'
		 END as resource_name,
		 CASE 
			 WHEN r.resource_type = 'ecs' THEN e.status
			 WHEN r.resource_type = 'rds' THEN d.status
			 WHEN r.resource_type = 'vpc' THEN v.status
			 ELSE 'unknown'
		 END as resource_status,
		 CASE 
			 WHEN r.resource_type = 'ecs' THEN e.created_at
			 WHEN r.resource_type = 'rds' THEN d.created_at
			 WHEN r.resource_type = 'vpc' THEN v.created_at
			 ELSE NULL
		 END as resource_create_time,
		 CASE 
			 WHEN r.resource_type = 'ecs' THEN e.updated_at
			 WHEN r.resource_type = 'rds' THEN d.updated_at
			 WHEN r.resource_type = 'vpc' THEN v.updated_at
			 ELSE NULL
		 END as resource_update_time,
		 CASE 
			 WHEN r.resource_type = 'ecs' THEN e.deleted_at
			 WHEN r.resource_type = 'rds' THEN d.deleted_at
			 WHEN r.resource_type = 'vpc' THEN v.deleted_at
			 ELSE NULL
		 END as resource_delete_time
		 FROM tree_node_resources r
		 LEFT JOIN resource_ecs e ON r.resource_id = e.id AND r.resource_type = 'ecs'
		 LEFT JOIN resource_rds d ON r.resource_id = d.id AND r.resource_type = 'rds'
		 LEFT JOIN resource_vpc v ON r.resource_id = v.id AND r.resource_type = 'vpc'
		 WHERE r.tree_node_id = ?
	 `, nodeId).Rows()

	if err != nil {
		t.logger.Error("查询节点资源失败", zap.Int("nodeId", nodeId), zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var resource model.TreeNodeResourceResp
		var createTime, updateTime, deleteTime *string
		if err := rows.Scan(
			&resource.ID,
			&resource.ResourceID,
			&resource.ResourceType,
			&resource.ResourceName,
			&resource.ResourceStatus,
			&createTime,
			&updateTime,
			&deleteTime,
		); err != nil {
			t.logger.Error("扫描资源数据失败", zap.Error(err))
			return nil, err
		}

		if createTime != nil {
			resource.ResourceCreateTime = *createTime
		}
		if updateTime != nil {
			resource.ResourceUpdateTime = *updateTime
		}
		if deleteTime != nil {
			resource.ResourceDeleteTime = *deleteTime
		}

		resources = append(resources, &resource)
	}

	return resources, nil
}

// GetResourceTypes 获取所有资源类型
func (t *treeDAO) GetResourceTypes(ctx context.Context) ([]string, error) {
	// 返回系统支持的资源类型
	return []string{"ecs", "rds", "vpc"}, nil
}

// GetTreeList 获取树节点列表
func (t *treeDAO) GetTreeList(ctx context.Context, req *model.TreeNodeListReq) ([]*model.TreeNode, error) {
	var nodes []*model.TreeNode
	query := t.db

	// 根据请求参数过滤
	if req.ParentID > 0 {
		query = query.Where("parent_id = ?", req.ParentID)
	} else if req.Level > 0 {
		query = query.Where("level = ?", req.Level)
	}

	// 执行查询
	if err := query.Find(&nodes).Error; err != nil {
		t.logger.Error("获取树节点列表失败", zap.Error(err))
		return nil, err
	}

	if len(nodes) > 0 {
		// 获取所有节点ID
		var nodeIds []int
		parentIds := make(map[int]bool)
		for _, node := range nodes {
			nodeIds = append(nodeIds, int(node.ID))
			if node.ParentID != 0 {
				parentIds[node.ParentID] = true
			}
		}

		// 批量获取父节点名称
		parentMap := make(map[int]string)
		if len(parentIds) > 0 {
			var parentIdsList []int
			for id := range parentIds {
				parentIdsList = append(parentIdsList, id)
			}

			var parents []struct {
				ID   int
				Name string
			}

			if err := t.db.Table("tree_nodes").
				Select("id, name").
				Where("id IN ?", parentIdsList).
				Find(&parents).Error; err != nil {
				t.logger.Error("获取父节点名称失败", zap.Error(err))
				return nil, err
			}

			for _, p := range parents {
				parentMap[p.ID] = p.Name
			}
		}

		// 批量获取子节点数量
		childCounts := make(map[int]int)
		rows, err := t.db.Raw(`
			 SELECT parent_id, COUNT(*) as count 
			 FROM tree_nodes 
			 WHERE parent_id IN (?)
			 GROUP BY parent_id
		 `, nodeIds).Rows()

		if err != nil {
			t.logger.Error("获取子节点数量失败", zap.Error(err))
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			var parentID int
			var count int
			if err := rows.Scan(&parentID, &count); err != nil {
				t.logger.Error("扫描子节点数量失败", zap.Error(err))
				return nil, err
			}
			childCounts[parentID] = count
		}

		// 批量获取资源数量
		resourceCounts := make(map[int]int)
		rows, err = t.db.Raw(`
			 SELECT tree_node_id, COUNT(*) as count 
			 FROM tree_node_resources 
			 WHERE tree_node_id IN (?)
			 GROUP BY tree_node_id
		 `, nodeIds).Rows()

		if err != nil {
			t.logger.Error("获取资源数量失败", zap.Error(err))
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			var nodeID int
			var count int
			if err := rows.Scan(&nodeID, &count); err != nil {
				t.logger.Error("扫描资源数量失败", zap.Error(err))
				return nil, err
			}
			resourceCounts[nodeID] = count
		}

		// 设置每个节点的额外信息
		for _, node := range nodes {
			if node.ParentID != 0 {
				node.ParentName = parentMap[node.ParentID]
			}
			node.ChildCount = childCounts[int(node.ID)]
			node.ResourceCount = resourceCounts[int(node.ID)]
		}
	}

	return nodes, nil
}

// GetTreeStatistics 获取树统计信息
func (t *treeDAO) GetTreeStatistics(ctx context.Context) (*model.TreeStatisticsResp, error) {
	// 使用事务保证数据一致性
	tx := t.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var stats model.TreeStatisticsResp

	// 获取节点总数
	var totalNodes int64
	if err := tx.Model(&model.TreeNode{}).Count(&totalNodes).Error; err != nil {
		tx.Rollback()
		t.logger.Error("获取节点总数失败", zap.Error(err))
		return nil, err
	}
	stats.TotalNodes = int(totalNodes)

	// 获取活跃和非活跃节点数
	var statusCounts []struct {
		Status string
		Count  int64
	}

	if err := tx.Model(&model.TreeNode{}).
		Select("status, COUNT(*) as count").
		Group("status").
		Find(&statusCounts).Error; err != nil {
		tx.Rollback()
		t.logger.Error("获取节点状态统计失败", zap.Error(err))
		return nil, err
	}

	for _, sc := range statusCounts {
		if sc.Status == "active" {
			stats.ActiveNodes = int(sc.Count)
		} else if sc.Status == "inactive" {
			stats.InactiveNodes = int(sc.Count)
		}
	}

	// 获取资源总数
	var totalResources int64
	if err := tx.Table("tree_node_resources").Count(&totalResources).Error; err != nil {
		tx.Rollback()
		t.logger.Error("获取资源总数失败", zap.Error(err))
		return nil, err
	}
	stats.TotalResources = int(totalResources)

	// 获取成员总数（去重）
	var totalMembers int64
	if err := tx.Raw(`
		 SELECT COUNT(DISTINCT user_id) FROM (
			 SELECT user_id FROM tree_node_admins
			 UNION
			 SELECT user_id FROM tree_node_members
		 ) as members
	 `).Scan(&totalMembers).Error; err != nil {
		tx.Rollback()
		t.logger.Error("获取成员总数失败", zap.Error(err))
		return nil, err
	}
	stats.TotalMembers = int(totalMembers)

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		t.logger.Error("提交事务失败", zap.Error(err))
		return nil, err
	}

	return &stats, nil
}

// RemoveNodeMember 移除节点成员
func (t *treeDAO) RemoveNodeMember(ctx context.Context, nodeId int, userId int, memberType string) error {
	// 检查节点是否存在
	if err := t.checkNodeExists(nodeId); err != nil {
		return err
	}

	// 根据成员类型删除关系
	var deleteQuery string
	var params []interface{}

	if memberType == "admin" {
		deleteQuery = "DELETE FROM tree_node_admins WHERE tree_node_id = ? AND user_id = ?"
		params = []interface{}{nodeId, userId}
	} else if memberType == "member" {
		deleteQuery = "DELETE FROM tree_node_members WHERE tree_node_id = ? AND user_id = ?"
		params = []interface{}{nodeId, userId}
	} else {
		return errors.New("无效的成员类型")
	}

	result := t.db.Exec(deleteQuery, params...)
	if result.Error != nil {
		t.logger.Error("移除成员失败", zap.Int("nodeId", nodeId), zap.Int("userId", userId), zap.Error(result.Error))
		return result.Error
	}

	// 检查是否实际删除了记录
	if result.RowsAffected == 0 {
		return fmt.Errorf("未找到要移除的%s关系", memberType)
	}

	return nil
}

// UnbindResource 解绑资源
func (t *treeDAO) UnbindResource(ctx context.Context, nodeId int, resourceType string, resourceId string) error {
	// 检查节点是否存在
	if err := t.checkNodeExists(nodeId); err != nil {
		return err
	}

	// 删除绑定关系
	result := t.db.Exec("DELETE FROM tree_node_resources WHERE tree_node_id = ? AND resource_type = ? AND resource_id = ?",
		nodeId, resourceType, resourceId)

	if result.Error != nil {
		t.logger.Error("解绑资源失败", zap.Int("nodeId", nodeId), zap.String("resourceId", resourceId), zap.Error(result.Error))
		return result.Error
	}

	// 检查是否实际删除了记录
	if result.RowsAffected == 0 {
		return errors.New("资源未绑定到该节点")
	}

	return nil
}

// UpdateNode 更新节点
func (t *treeDAO) UpdateNode(ctx context.Context, node *model.TreeNode) error {
	// 检查节点是否存在
	var existingNode model.TreeNode
	if err := t.db.First(&existingNode, node.ID).Error; err != nil {
		t.logger.Error("节点不存在", zap.Int("id", int(node.ID)), zap.Error(err))
		return errors.New("节点不存在")
	}

	// 检查节点名称是否与同级节点重复
	var count int64
	if err := t.db.Model(&model.TreeNode{}).Where("name = ? AND parent_id = ? AND id != ?", node.Name, existingNode.ParentID, node.ID).Count(&count).Error; err != nil {
		t.logger.Error("检查节点名称失败", zap.String("name", node.Name), zap.Error(err))
		return err
	}

	if count > 0 {
		return errors.New("同级节点下已存在相同名称的节点")
	}

	// 更新节点信息
	// 只更新允许修改的字段
	updateMap := map[string]interface{}{
		"name":        node.Name,
		"description": node.Description,
		"status":      node.Status,
		"updated_at":  node.UpdatedAt,
	}

	if err := t.db.Model(&model.TreeNode{}).Where("id = ?", node.ID).Updates(updateMap).Error; err != nil {
		t.logger.Error("更新节点失败", zap.Int("id", int(node.ID)), zap.Error(err))
		return err
	}

	return nil
}

// GetNodeLevel 获取节点层级
func (t *treeDAO) GetNodeLevel(ctx context.Context, nodeId int) (int, error) {
	var node model.TreeNode
	if err := t.db.WithContext(ctx).Where("id = ?", nodeId).First(&node).Error; err != nil {
		t.logger.Error("节点不存在", zap.Int("id", nodeId), zap.Error(err))
		return 0, err
	}

	return node.Level, nil
}

// Transaction 暴露给service层使用
func (t *treeDAO) Transaction(ctx context.Context, fn func(ctx context.Context) error) error {
	err := t.db.Transaction(func(tx *gorm.DB) error {
		txCtx := context.WithValue(ctx, txKey, tx)

		err := fn(txCtx)
		if err != nil {
			t.logger.Error("事务执行失败", zap.Error(err))
			return err
		}
		return nil
	})

	return err
}

// UpdateNodePath 更新节点路径
func (t *treeDAO) UpdateNodePath(ctx context.Context, nodeId int, path string) error {
	if err := t.db.Model(&model.TreeNode{}).Where("id = ?", nodeId).Update("path", path).Error; err != nil {
		t.logger.Error("更新节点路径失败", zap.Int("nodeId", nodeId), zap.String("path", path), zap.Error(err))
		return err
	}

	return nil
}
