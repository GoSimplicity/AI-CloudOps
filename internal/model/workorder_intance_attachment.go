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

// InstanceAttachment 工单附件实体（DAO层）
type InstanceAttachment struct {
	Model
	InstanceID   int    `json:"instance_id" gorm:"index;column:instance_id;not null;comment:工单实例ID"`
	FileName     string `json:"file_name" gorm:"column:file_name;not null;comment:文件名"`
	FileSize     int64  `json:"file_size" gorm:"column:file_size;not null;comment:文件大小(字节)"`
	FilePath     string `json:"file_path" gorm:"column:file_path;not null;comment:文件路径"`
	FileType     string `json:"file_type" gorm:"column:file_type;not null;comment:文件类型"`
	UploaderID   int    `json:"uploader_id" gorm:"column:uploader_id;not null;comment:上传人ID"`
	UploaderName string `json:"uploader_name" gorm:"-"`
	Description  string `json:"description" gorm:"column:description;type:text;comment:附件描述"`
}

// TableName 指定工单附件表名
func (InstanceAttachment) TableName() string {
	return "workorder_instance_attachment"
}

// InstanceAttachmentResp 工单附件响应结构
type InstanceAttachmentResp struct {
	ID           int       `json:"id"`
	InstanceID   int       `json:"instance_id"`
	FileName     string    `json:"file_name"`
	FileSize     int64     `json:"file_size"`
	FilePath     string    `json:"file_path"`
	FileType     string    `json:"file_type"`
	UploaderID   int       `json:"uploader_id"`
	UploaderName string    `json:"uploader_name"`
	CreatedAt    time.Time `json:"created_at"`
	Description  string    `json:"description"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// 附件请求结构
type UploadAttachmentReq struct {
	InstanceID  int    `json:"instance_id" form:"instance_id" binding:"required"`
	Description string `json:"description" form:"description"`
}

// 删除附件请求
type DeleteAttachmentReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}
