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

package model

// 评论类型常量
const (
	CommentTypeNormal = "normal" // 普通评论
	CommentTypeSystem = "system" // 系统评论
)

// 评论状态常量
const (
	CommentStatusNormal  int8 = 1 // 正常
	CommentStatusDeleted int8 = 2 // 已删除
	CommentStatusHidden  int8 = 3 // 已隐藏
)

// WorkorderInstanceComment 工单实例评论实体
type WorkorderInstanceComment struct {
	Model
	InstanceID     int                        `json:"instance_id" gorm:"not null;index;comment:工单实例ID"`
	CreateUserID   int                        `json:"create_user_id" gorm:"not null;index;comment:创建人ID"`
	CreateUserName string                     `json:"create_user_name" gorm:"type:varchar(200);not null;comment:创建人名称"`
	Content        string                     `json:"content" gorm:"type:text;not null;comment:评论内容"`
	ParentID       *int                       `json:"parent_id,omitempty" gorm:"index;comment:父评论ID"`
	Type           string                     `json:"type" gorm:"type:varchar(20);not null;default:'normal';comment:评论类型"`
	Status         int8                       `json:"status" gorm:"not null;default:1;index;comment:状态：1-正常，2-已删除，3-已隐藏"`
	IsSystem       bool                       `json:"is_system" gorm:"not null;default:false;comment:是否系统评论"`
	Children       []WorkorderInstanceComment `json:"children,omitempty" gorm:"-"`
}

// TableName 指定工单实例评论表名
func (WorkorderInstanceComment) TableName() string {
	return "cl_workorder_instance_comment"
}

// CreateWorkorderInstanceCommentReq 创建工单实例评论请求
type CreateWorkorderInstanceCommentReq struct {
	InstanceID     int    `json:"instance_id" binding:"required,min=1"`
	CreateUserID   int    `json:"create_user_id" binding:"required,min=1"`
	CreateUserName string `json:"create_user_name" binding:"required,min=1,max=200"`
	Content        string `json:"content" binding:"required,min=1,max=2000"`
	ParentID       *int   `json:"parent_id" binding:"omitempty,min=1"`
	Type           string `json:"type" binding:"omitempty,oneof=normal system"`
	Status         int8   `json:"status" binding:"omitempty,oneof=1 2 3"`
	IsSystem       bool   `json:"is_system" binding:"omitempty"`
}

// UpdateWorkorderInstanceCommentReq 更新工单实例评论请求
type UpdateWorkorderInstanceCommentReq struct {
	ID       int    `json:"id" binding:"required,min=1"`
	Content  string `json:"content" binding:"required,min=1,max=2000"`
	Status   int8   `json:"status" binding:"omitempty,oneof=1 2 3"`
	IsSystem bool   `json:"is_system" binding:"omitempty"`
}

// DeleteWorkorderInstanceCommentReq 删除工单实例评论请求
type DeleteWorkorderInstanceCommentReq struct {
	ID int `json:"id" form:"id" binding:"required,min=1"`
}

// DetailWorkorderInstanceCommentReq 获取工单实例评论详情请求
type DetailWorkorderInstanceCommentReq struct {
	ID int `json:"id" form:"id" binding:"required,min=1"`
}

// ListWorkorderInstanceCommentReq 工单实例评论列表请求
type ListWorkorderInstanceCommentReq struct {
	ListReq
	InstanceID *int    `json:"instance_id" form:"instance_id" binding:"omitempty,min=1"`
	Type       *string `json:"type" form:"type" binding:"omitempty,oneof=normal system"`
	Status     *int8   `json:"status" form:"status" binding:"omitempty,oneof=1 2 3"`
}
