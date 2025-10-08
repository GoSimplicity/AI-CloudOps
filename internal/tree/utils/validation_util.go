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

import "errors"

// ValidateAndSetPaginationDefaults 验证并设置分页参数的默认值
// 如果page <= 0，设置为1
// 如果size <= 0，设置为10
func ValidateAndSetPaginationDefaults(page, size *int) {
	if *page <= 0 {
		*page = 1
	}
	if *size <= 0 {
		*size = 10
	}
}

// ValidateID 验证ID是否有效
// ID必须大于0
func ValidateID(id int) error {
	if id <= 0 {
		return errors.New("无效的ID")
	}
	return nil
}

// ValidateParentID 验证父节点ID是否有效
// 父节点ID不能为负数
func ValidateParentID(parentID int) error {
	if parentID < 0 {
		return errors.New("父节点ID不能为负数")
	}
	return nil
}

// ValidateNodeMove 验证节点移动操作
// 节点不能移动到自己
func ValidateNodeMove(nodeID, newParentID int) error {
	if nodeID == newParentID {
		return errors.New("节点不能移动到自己")
	}
	return nil
}

// ValidateMemberType 验证成员类型
// 成员类型只能是 admin、member 或 all
func ValidateMemberType(memberType string) error {
	if memberType != "" && memberType != "admin" && memberType != "member" && memberType != "all" {
		return errors.New("成员类型只能是admin、member或all")
	}
	return nil
}

// ValidateResourceIDs 验证资源ID列表
// 资源ID列表不能为空
func ValidateResourceIDs(resourceIDs []int) error {
	if len(resourceIDs) == 0 {
		return errors.New("资源ID列表不能为空")
	}
	return nil
}

// ValidateTreeNodeIDs 验证树节点ID列表
// 如果列表为空，返回false（表示没有需要处理的节点）
func ValidateTreeNodeIDs(treeNodeIDs []int) bool {
	return len(treeNodeIDs) > 0
}
