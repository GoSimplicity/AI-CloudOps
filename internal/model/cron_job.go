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

import (
	"time"
)

// CronJobStatus 定时任务状态枚举
type CronJobStatus int8

const (
	CronJobStatusEnabled  CronJobStatus = iota + 1 // 启用
	CronJobStatusDisabled                          // 禁用
	CronJobStatusRunning                           // 运行中
	CronJobStatusError                             // 错误
)

// CronJobType 定时任务类型枚举
type CronJobType int8

const (
	CronJobTypeSystem  CronJobType = iota + 1 // 系统任务
	CronJobTypeCommand                        // 命令行任务
	CronJobTypeHTTP                           // HTTP请求任务
	CronJobTypeScript                         // 脚本任务
	CronJobTypeSSH                            // SSH远程执行任务
)

// CronJob 定时任务模型 - 简洁设计
type CronJob struct {
	Model
	Name          string        `json:"name" gorm:"type:varchar(100);not null;uniqueIndex;comment:任务名称"`
	Description   string        `json:"description" gorm:"type:text;comment:任务描述"`
	JobType       CronJobType   `json:"job_type" gorm:"type:tinyint(1);not null;default:2;comment:任务类型 1系统任务 2命令行 3HTTP 4脚本 5SSH"`
	Status        CronJobStatus `json:"status" gorm:"type:tinyint(1);not null;default:1;comment:任务状态 1启用 2禁用 3运行中 4错误"`
	IsBuiltIn     bool          `json:"is_built_in" gorm:"type:tinyint(1);not null;default:0;comment:是否为内置任务 0否 1是"`
	Schedule      string        `json:"schedule" gorm:"type:varchar(100);not null;comment:调度表达式"`
	Command       string        `json:"command" gorm:"type:text;comment:执行命令"`
	Args          StringList    `json:"args" gorm:"type:text;comment:命令参数"`
	WorkDir       string        `json:"work_dir" gorm:"type:varchar(500);comment:工作目录"`
	Environment   KeyValueList  `json:"environment" gorm:"type:text;comment:环境变量"`
	HTTPMethod    string        `json:"http_method" gorm:"type:varchar(10);comment:HTTP方法"`
	HTTPUrl       string        `json:"http_url" gorm:"type:varchar(500);comment:HTTP URL"`
	HTTPHeaders   KeyValueList  `json:"http_headers" gorm:"type:text;comment:HTTP请求头"`
	HTTPBody      string        `json:"http_body" gorm:"type:text;comment:HTTP请求体"`
	ScriptType    string        `json:"script_type" gorm:"type:varchar(20);comment:脚本类型"`
	ScriptContent string        `json:"script_content" gorm:"type:longtext;comment:脚本内容"`
	// SSH远程执行相关字段
	SSHResourceID   *int               `json:"ssh_resource_id" gorm:"comment:SSH资源ID,关联树资源"`
	SSHResource     *TreeLocalResource `json:"ssh_resource,omitempty" gorm:"foreignKey:SSHResourceID"`
	SSHCommand      string             `json:"ssh_command" gorm:"type:text;comment:SSH执行命令"`
	SSHWorkDir      string             `json:"ssh_work_dir" gorm:"type:varchar(500);comment:SSH工作目录"`
	SSHEnvironment  KeyValueList       `json:"ssh_environment" gorm:"type:text;comment:SSH环境变量"`
	Timeout         int                `json:"timeout" gorm:"default:300;comment:超时时间(秒)"`
	MaxRetry        int                `json:"max_retry" gorm:"default:3;comment:最大重试次数"`
	NextRunTime     *time.Time         `json:"next_run_time" gorm:"comment:下次运行时间"`
	LastRunTime     *time.Time         `json:"last_run_time" gorm:"comment:上次运行时间"`
	LastRunStatus   int8               `json:"last_run_status" gorm:"default:0;comment:上次运行状态 0未执行 1成功 2失败"`
	LastRunDuration int                `json:"last_run_duration" gorm:"default:0;comment:上次运行时长(毫秒)"`
	LastRunError    string             `json:"last_run_error" gorm:"type:text;comment:上次运行错误"`
	LastRunOutput   string             `json:"last_run_output" gorm:"type:text;comment:上次运行输出"`
	RunCount        int                `json:"run_count" gorm:"default:0;comment:运行次数"`
	SuccessCount    int                `json:"success_count" gorm:"default:0;comment:成功次数"`
	FailureCount    int                `json:"failure_count" gorm:"default:0;comment:失败次数"`
	CreatedBy       int                `json:"created_by" gorm:"comment:创建者ID"`
	CreatedByName   string             `json:"created_by_name" gorm:"type:varchar(100);comment:创建者名称"`
}

func (c *CronJob) TableName() string {
	return "cl_cron_jobs"
}

// GetCronJobListReq 获取定时任务列表请求
type GetCronJobListReq struct {
	ListReq
	Status  *CronJobStatus `json:"status" form:"status" binding:"omitempty,oneof=1 2 3 4"`
	JobType *CronJobType   `json:"job_type" form:"job_type" binding:"omitempty,oneof=1 2 3 4 5"`
	Search  string         `json:"search" form:"search"`
}

// CreateCronJobReq 创建定时任务请求
type CreateCronJobReq struct {
	Name          string       `json:"name" binding:"required,min=1,max=100"`
	Description   string       `json:"description" binding:"max=500"`
	JobType       CronJobType  `json:"job_type" binding:"required,oneof=1 2 3 4 5"`
	Schedule      string       `json:"schedule" binding:"required"`
	Command       string       `json:"command"`
	Args          StringList   `json:"args"`
	WorkDir       string       `json:"work_dir"`
	Environment   KeyValueList `json:"environment"`
	HTTPMethod    string       `json:"http_method"`
	HTTPUrl       string       `json:"http_url"`
	HTTPHeaders   KeyValueList `json:"http_headers"`
	HTTPBody      string       `json:"http_body"`
	ScriptType    string       `json:"script_type"`
	ScriptContent string       `json:"script_content"`
	// SSH相关字段
	SSHResourceID  *int         `json:"ssh_resource_id"`
	SSHCommand     string       `json:"ssh_command"`
	SSHWorkDir     string       `json:"ssh_work_dir"`
	SSHEnvironment KeyValueList `json:"ssh_environment"`
	Timeout        int          `json:"timeout" binding:"omitempty,min=1,max=3600"`
	MaxRetry       int          `json:"max_retry" binding:"omitempty,min=0,max=10"`
	CreatedBy      int          `json:"created_by"`
	CreatedByName  string       `json:"created_by_name"`
}

// UpdateCronJobReq 更新定时任务请求
type UpdateCronJobReq struct {
	ID            int          `json:"id" form:"id" binding:"required"`
	Name          string       `json:"name" binding:"required,min=1,max=100"`
	Description   string       `json:"description" binding:"max=500"`
	JobType       CronJobType  `json:"job_type" binding:"required,oneof=1 2 3 4 5"`
	Schedule      string       `json:"schedule" binding:"required"`
	Command       string       `json:"command"`
	Args          StringList   `json:"args"`
	WorkDir       string       `json:"work_dir"`
	Environment   KeyValueList `json:"environment"`
	HTTPMethod    string       `json:"http_method"`
	HTTPUrl       string       `json:"http_url"`
	HTTPHeaders   KeyValueList `json:"http_headers"`
	HTTPBody      string       `json:"http_body"`
	ScriptType    string       `json:"script_type"`
	ScriptContent string       `json:"script_content"`
	// SSH相关字段
	SSHResourceID  *int         `json:"ssh_resource_id"`
	SSHCommand     string       `json:"ssh_command"`
	SSHWorkDir     string       `json:"ssh_work_dir"`
	SSHEnvironment KeyValueList `json:"ssh_environment"`
	Timeout        int          `json:"timeout" binding:"omitempty,min=1,max=3600"`
	MaxRetry       int          `json:"max_retry" binding:"omitempty,min=0,max=10"`
}

// 简化的操作请求
type DeleteCronJobReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}

type GetCronJobReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}

type EnableCronJobReq struct {
	ID     int           `json:"id" form:"id" binding:"required"`
	Status CronJobStatus `json:"status" binding:"required,oneof=1 2"`
}

type TriggerCronJobReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}

type ValidateScheduleReq struct {
	Schedule string `json:"schedule" form:"id" binding:"required"`
}

type DisableCronJobReq struct {
	ID int `json:"id" form:"id" binding:"required"`
}

type ValidateScheduleResp struct {
	Valid        bool     `json:"valid"`                    // 是否有效
	ErrorMessage string   `json:"error_message,omitempty"`  // 错误信息
	NextRunTimes []string `json:"next_run_times,omitempty"` // 下次运行时间预览
}
