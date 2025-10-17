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

package utils

import (
	"errors"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

// ValidateParentID 验证父节点ID是否有效
func ValidateParentID(parentID int) error {
	if parentID < 0 {
		return errors.New("父节点ID不能为负数")
	}
	return nil
}

// ValidateNodeMove 验证节点移动操作
func ValidateNodeMove(nodeID, newParentID int) error {
	if nodeID == newParentID {
		return errors.New("节点不能移动到自己")
	}
	return nil
}

// ValidateMemberType 验证成员类型
func ValidateMemberType(memberType string) error {
	if memberType != "" && memberType != "admin" && memberType != "member" && memberType != "all" {
		return errors.New("成员类型只能是admin、member或all")
	}
	return nil
}

// ValidateResourceIDs 验证资源ID列表
func ValidateResourceIDs(resourceIDs []int) error {
	if len(resourceIDs) == 0 {
		return errors.New("资源ID列表不能为空")
	}
	return nil
}

// BuildTreeStructure 构建树形结构
func BuildTreeStructure(nodes []*model.TreeNode) []*model.TreeNode {
	// 创建节点映射表，用于快速查找节点
	nodeMap := make(map[int]*model.TreeNode)
	var rootNodes []*model.TreeNode

	// 第一遍遍历：复制节点并初始化Children字段
	for _, node := range nodes {
		nodeClone := *node
		nodeClone.Children = make([]*model.TreeNode, 0)
		nodeMap[node.ID] = &nodeClone
	}

	// 第二遍遍历：建立父子关系
	for _, node := range nodes {
		currentNode := nodeMap[node.ID]
		// 如果节点没有父节点或父节点不存在，则为根节点
		if node.ParentID == 0 || nodeMap[node.ParentID] == nil {
			rootNodes = append(rootNodes, currentNode)
		} else {
			// 将当前节点添加到其父节点的Children列表中
			parent := nodeMap[node.ParentID]
			parent.Children = append(parent.Children, currentNode)
		}
	}

	return rootNodes
}
