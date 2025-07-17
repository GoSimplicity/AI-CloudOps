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

import "time"

// InstanceComment 工单评论实体（DAO层）
type InstanceComment struct {
	Model
	InstanceID  int    `json:"instance_id" gorm:"index;column:instance_id;not null;comment:工单实例ID"`
	UserID      int    `json:"user_id" gorm:"column:user_id;not null;comment:用户ID"`
	Content     string `json:"content" gorm:"column:content;type:text;not null;comment:评论内容"`
	CreatorID   int    `json:"creator_id" gorm:"column:creator_id;not null;comment:创建人ID"`
	CreatorName string `json:"creator_name" gorm:"-"`
	ParentID    *int   `json:"parent_id" gorm:"column:parent_id;default:null;comment:父评论ID"`
	IsSystem    bool   `json:"is_system" gorm:"column:is_system;default:false;comment:是否系统评论"`
}

// TableName 指定工单评论表名
func (InstanceComment) TableName() string {
	return "workorder_instance_comment"
}

// InstanceCommentResp 工单评论响应结构
type InstanceCommentResp struct {
	ID          int                   `json:"id"`
	InstanceID  int                   `json:"instance_id"`
	UserID      int                   `json:"user_id"`
	Content     string                `json:"content"`
	CreatorID   int                   `json:"creator_id"`
	CreatorName string                `json:"creator_name"`
	ParentID    *int                  `json:"parent_id"`
	IsSystem    bool                  `json:"is_system"`
	CreatedAt   time.Time             `json:"created_at"`
	UpdatedAt   time.Time             `json:"updated_at"`
	Children    []InstanceCommentResp `json:"children,omitempty"`
}

type InstanceCommentReq struct {
	InstanceID int    `json:"instance_id" binding:"required"`
	Content    string `json:"content" binding:"required,max=1000"`
	ParentID   *int   `json:"parent_id"`
}

type GetInstanceCommentsReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}
