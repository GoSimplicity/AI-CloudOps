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

// 评论类型常量
const (
	CommentTypeNormal  = "normal"  // 普通评论
	CommentTypeSystem  = "system"  // 系统评论
	CommentTypePrivate = "private" // 私有评论
	CommentTypePublic  = "public"  // 公开评论
	CommentTypeInternal = "internal" // 内部评论
)

// 评论状态常量
const (
	CommentStatusNormal  int8 = 1 // 正常
	CommentStatusDeleted int8 = 0 // 已删除
	CommentStatusHidden  int8 = 2 // 已隐藏
)

// WorkorderInstanceComment 工单实例评论实体
type WorkorderInstanceComment struct {
	Model
	InstanceID     int    `json:"instance_id" gorm:"column:instance_id;not null;index;comment:工单实例ID"`
	UserID         int    `json:"user_id" gorm:"column:user_id;not null;index;comment:评论用户ID"`
	UserName       string `json:"user_name" gorm:"-"`
	UserAvatar     string `json:"user_avatar" gorm:"-"`
	Content        string `json:"content" gorm:"column:content;type:text;not null;comment:评论内容"`
	ContentHTML    string `json:"content_html" gorm:"column:content_html;type:text;comment:评论HTML内容"`
	ParentID       *int   `json:"parent_id" gorm:"column:parent_id;index;comment:父评论ID"`
	RootID         *int   `json:"root_id" gorm:"column:root_id;index;comment:根评论ID"`
	ReplyToUserID  *int   `json:"reply_to_user_id" gorm:"column:reply_to_user_id;index;comment:回复目标用户ID"`
	ReplyToUserName string `json:"reply_to_user_name" gorm:"-"`
	Type           string `json:"type" gorm:"column:type;type:varchar(20);not null;default:'normal';index;comment:评论类型"`
	Status         int8   `json:"status" gorm:"column:status;not null;default:1;index;comment:状态：1-正常，0-已删除，2-已隐藏"`
	IsSystem       bool   `json:"is_system" gorm:"column:is_system;not null;default:false;index;comment:是否系统评论"`
	IsPrivate      bool   `json:"is_private" gorm:"column:is_private;not null;default:false;comment:是否私有评论"`
	IsEdited       bool   `json:"is_edited" gorm:"column:is_edited;not null;default:false;comment:是否已编辑"`
	EditedAt       *time.Time `json:"edited_at" gorm:"column:edited_at;comment:编辑时间"`
	LikeCount      int    `json:"like_count" gorm:"column:like_count;not null;default:0;comment:点赞数"`
	ReplyCount     int    `json:"reply_count" gorm:"column:reply_count;not null;default:0;comment:回复数"`
	AttachmentCount int   `json:"attachment_count" gorm:"column:attachment_count;not null;default:0;comment:附件数量"`
	Mentions       StringList `json:"mentions" gorm:"column:mentions;comment:提及的用户"`
	Tags           StringList `json:"tags" gorm:"column:tags;comment:标签"`
	ClientIP       string `json:"client_ip" gorm:"column:client_ip;type:varchar(50);comment:客户端IP"`
	UserAgent      string `json:"user_agent" gorm:"column:user_agent;type:varchar(500);comment:用户代理"`
	ExtendedData   JSONMap `json:"extended_data" gorm:"column:extended_data;type:json;comment:扩展数据"`

	// 关联信息（不存储到数据库）
	InstanceTitle  string                      `json:"instance_title,omitempty" gorm:"-"`
	Children       []WorkorderInstanceComment  `json:"children,omitempty" gorm:"-"`
	IsLiked        bool                        `json:"is_liked,omitempty" gorm:"-"`        // 当前用户是否点赞
	CanEdit        bool                        `json:"can_edit,omitempty" gorm:"-"`        // 是否可编辑
	CanDelete      bool                        `json:"can_delete,omitempty" gorm:"-"`      // 是否可删除
}

// TableName 指定工单实例评论表名
func (WorkorderInstanceComment) TableName() string {
	return "cl_workorder_instance_comment"
}

// WorkorderInstanceCommentResp 工单实例评论响应结构
type WorkorderInstanceCommentResp struct {
	ID              int                            `json:"id"`
	InstanceID      int                            `json:"instance_id"`
	UserID          int                            `json:"user_id"`
	UserName        string                         `json:"user_name"`
	UserAvatar      string                         `json:"user_avatar"`
	Content         string                         `json:"content"`
	ContentHTML     string                         `json:"content_html"`
	ParentID        *int                           `json:"parent_id"`
	RootID          *int                           `json:"root_id"`
	ReplyToUserID   *int                           `json:"reply_to_user_id"`
	ReplyToUserName string                         `json:"reply_to_user_name"`
	Type            string                         `json:"type"`
	Status          int8                           `json:"status"`
	IsSystem        bool                           `json:"is_system"`
	IsPrivate       bool                           `json:"is_private"`
	IsEdited        bool                           `json:"is_edited"`
	EditedAt        *time.Time                     `json:"edited_at"`
	LikeCount       int                            `json:"like_count"`
	ReplyCount      int                            `json:"reply_count"`
	AttachmentCount int                            `json:"attachment_count"`
	Mentions        []string                       `json:"mentions"`
	Tags            []string                       `json:"tags"`
	InstanceTitle   string                         `json:"instance_title"`
	Children        []WorkorderInstanceCommentResp `json:"children,omitempty"`
	IsLiked         bool                           `json:"is_liked"`
	CanEdit         bool                           `json:"can_edit"`
	CanDelete       bool                           `json:"can_delete"`
	CreatedAt       time.Time                      `json:"created_at"`
	UpdatedAt       time.Time                      `json:"updated_at"`
}

// CreateWorkorderInstanceCommentReq 创建工单实例评论请求
type CreateWorkorderInstanceCommentReq struct {
	InstanceID    int                    `json:"instance_id" binding:"required,min=1"`
	Content       string                 `json:"content" binding:"required,min=1,max=10000"`
	ContentHTML   string                 `json:"content_html" binding:"omitempty,max=20000"`
	ParentID      *int                   `json:"parent_id" binding:"omitempty,min=1"`
	ReplyToUserID *int                   `json:"reply_to_user_id" binding:"omitempty,min=1"`
	Type          string                 `json:"type" binding:"omitempty,oneof=normal system private public internal"`
	IsPrivate     bool                   `json:"is_private"`
	Mentions      []string               `json:"mentions" binding:"omitempty"`
	Tags          []string               `json:"tags" binding:"omitempty"`
	ExtendedData  map[string]interface{} `json:"extended_data"`
}

// UpdateWorkorderInstanceCommentReq 更新工单实例评论请求
type UpdateWorkorderInstanceCommentReq struct {
	ID           int                    `json:"id" binding:"required,min=1"`
	Content      string                 `json:"content" binding:"required,min=1,max=10000"`
	ContentHTML  string                 `json:"content_html" binding:"omitempty,max=20000"`
	IsPrivate    bool                   `json:"is_private"`
	Mentions     []string               `json:"mentions" binding:"omitempty"`
	Tags         []string               `json:"tags" binding:"omitempty"`
	ExtendedData map[string]interface{} `json:"extended_data"`
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
	InstanceID    *int    `json:"instance_id" form:"instance_id" binding:"omitempty,min=1"`
	UserID        *int    `json:"user_id" form:"user_id" binding:"omitempty,min=1"`
	ParentID      *int    `json:"parent_id" form:"parent_id" binding:"omitempty,min=1"`
	RootID        *int    `json:"root_id" form:"root_id" binding:"omitempty,min=1"`
	Type          *string `json:"type" form:"type" binding:"omitempty,oneof=normal system private public internal"`
	Status        *int8   `json:"status" form:"status" binding:"omitempty,oneof=1 0 2"`
	IsSystem      *bool   `json:"is_system" form:"is_system"`
	IsPrivate     *bool   `json:"is_private" form:"is_private"`
	StartDate     *time.Time `json:"start_date" form:"start_date"`
	EndDate       *time.Time `json:"end_date" form:"end_date"`
}

// GetWorkorderInstanceCommentsReq 获取工单实例评论请求
type GetWorkorderInstanceCommentsReq struct {
	InstanceID int    `json:"instance_id" form:"instance_id" binding:"required,min=1"`
	Type       string `json:"type" form:"type" binding:"omitempty,oneof=tree flat"`
	OnlyRoot   bool   `json:"only_root" form:"only_root"`
}

// LikeWorkorderInstanceCommentReq 点赞工单实例评论请求
type LikeWorkorderInstanceCommentReq struct {
	CommentID int  `json:"comment_id" binding:"required,min=1"`
	IsLike    bool `json:"is_like"`
}

// HideWorkorderInstanceCommentReq 隐藏工单实例评论请求
type HideWorkorderInstanceCommentReq struct {
	ID     int    `json:"id" binding:"required,min=1"`
	Reason string `json:"reason" binding:"omitempty,max=500"`
}

// RestoreWorkorderInstanceCommentReq 恢复工单实例评论请求
type RestoreWorkorderInstanceCommentReq struct {
	ID int `json:"id" binding:"required,min=1"`
}

// BatchDeleteWorkorderInstanceCommentReq 批量删除工单实例评论请求
type BatchDeleteWorkorderInstanceCommentReq struct {
	IDs    []int  `json:"ids" binding:"required,min=1,dive,min=1"`
	Reason string `json:"reason" binding:"omitempty,max=500"`
}

// BatchHideWorkorderInstanceCommentReq 批量隐藏工单实例评论请求
type BatchHideWorkorderInstanceCommentReq struct {
	IDs    []int  `json:"ids" binding:"required,min=1,dive,min=1"`
	Reason string `json:"reason" binding:"omitempty,max=500"`
}

// GetCommentLikesReq 获取评论点赞列表请求
type GetCommentLikesReq struct {
	CommentID int `json:"comment_id" form:"comment_id" binding:"required,min=1"`
	ListReq
}

// GetCommentRepliesReq 获取评论回复列表请求
type GetCommentRepliesReq struct {
	CommentID int `json:"comment_id" form:"comment_id" binding:"required,min=1"`
	ListReq
}

// SearchWorkorderInstanceCommentReq 搜索工单实例评论请求
type SearchWorkorderInstanceCommentReq struct {
	ListReq
	Keyword    string     `json:"keyword" form:"keyword" binding:"required,min=1"`
	InstanceID *int       `json:"instance_id" form:"instance_id" binding:"omitempty,min=1"`
	UserID     *int       `json:"user_id" form:"user_id" binding:"omitempty,min=1"`
	Type       *string    `json:"type" form:"type" binding:"omitempty,oneof=normal system private public internal"`
	StartDate  *time.Time `json:"start_date" form:"start_date"`
	EndDate    *time.Time `json:"end_date" form:"end_date"`
}

// WorkorderInstanceCommentStatistics 工单实例评论统计
type WorkorderInstanceCommentStatistics struct {
	TotalCount      int64 `json:"total_count"`      // 总评论数
	NormalCount     int64 `json:"normal_count"`     // 普通评论数
	SystemCount     int64 `json:"system_count"`     // 系统评论数
	PrivateCount    int64 `json:"private_count"`    // 私有评论数
	PublicCount     int64 `json:"public_count"`     // 公开评论数
	DeletedCount    int64 `json:"deleted_count"`    // 已删除评论数
	HiddenCount     int64 `json:"hidden_count"`     // 已隐藏评论数
	TotalLikes      int64 `json:"total_likes"`      // 总点赞数
	TotalReplies    int64 `json:"total_replies"`    // 总回复数
	AvgCommentLength int64 `json:"avg_comment_length"` // 平均评论长度
	ActiveUsers     int64 `json:"active_users"`     // 活跃评论用户数
}

// WorkorderInstanceCommentActivity 工单实例评论活动
type WorkorderInstanceCommentActivity struct {
	Date         string `json:"date"`          // 日期
	CommentCount int64  `json:"comment_count"` // 评论数量
	UserCount    int64  `json:"user_count"`    // 用户数量
	LikeCount    int64  `json:"like_count"`    // 点赞数量
}

// WorkorderInstanceCommentUserStats 工单实例评论用户统计
type WorkorderInstanceCommentUserStats struct {
	UserID       int     `json:"user_id"`       // 用户ID
	UserName     string  `json:"user_name"`     // 用户名称
	CommentCount int64   `json:"comment_count"` // 评论数量
	LikeCount    int64   `json:"like_count"`    // 获得点赞数
	ReplyCount   int64   `json:"reply_count"`   // 回复数量
	AvgLength    float64 `json:"avg_length"`    // 平均评论长度
	LastComment  *time.Time `json:"last_comment"` // 最后评论时间
}

// WorkorderInstanceCommentLike 工单实例评论点赞关联表
type WorkorderInstanceCommentLike struct {
	Model
	CommentID int `json:"comment_id" gorm:"column:comment_id;not null;index;comment:评论ID"`
	UserID    int `json:"user_id" gorm:"column:user_id;not null;index;comment:用户ID"`
}

// TableName 指定工单实例评论点赞表名
func (WorkorderInstanceCommentLike) TableName() string {
	return "cl_workorder_instance_comment_like"
}
