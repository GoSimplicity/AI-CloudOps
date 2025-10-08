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

import "github.com/GoSimplicity/AI-CloudOps/internal/model"

// BuildTreeStructure 构建树形结构
// 将扁平化的节点列表转换为具有父子关系的树形结构
// 返回根节点列表，每个节点的Children字段包含其子节点
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
