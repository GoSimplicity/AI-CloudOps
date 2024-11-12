package ali_resource

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

import (
	"context"
	"encoding/json"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/redis/go-redis/v9"
	"time"
)

type AliResourceDAO interface {
	CreateTask(ctx context.Context, task *model.Task) error
	GetTaskByID(ctx context.Context, taskID string) (*model.Task, error)
	UpdateTaskStatus(ctx context.Context, taskID, status, errorMessage string, retryCount *int) error
}

type aliResourceDAO struct {
	client redis.Cmdable
}

func NewAliResourceDAO(client redis.Cmdable) AliResourceDAO {
	return &aliResourceDAO{
		client: client,
	}
}

// CreateTask 创建任务
func (d *aliResourceDAO) CreateTask(ctx context.Context, task *model.Task) error {
	taskKey := "task:" + task.TaskID
	taskData, err := json.Marshal(task)
	if err != nil {
		return err
	}
	return d.client.Set(ctx, taskKey, taskData, 0).Err()
}

// GetTaskByID 根据任务 ID 获取任务
func (d *aliResourceDAO) GetTaskByID(ctx context.Context, taskID string) (*model.Task, error) {
	taskKey := "task:" + taskID
	taskData, err := d.client.Get(ctx, taskKey).Result()
	if err != nil {
		return nil, err
	}

	var task model.Task
	if err := json.Unmarshal([]byte(taskData), &task); err != nil {
		return nil, err
	}

	return &task, nil
}

// UpdateTaskStatus 更新任务状态
func (d *aliResourceDAO) UpdateTaskStatus(ctx context.Context, taskID, status, errorMessage string, retryCount *int) error {
	task, err := d.GetTaskByID(ctx, taskID)
	if err != nil {
		return err
	}

	task.Status = status
	task.ErrorMessage = errorMessage
	if retryCount != nil {
		task.RetryCount = *retryCount
	}

	task.UpdatedAt = time.Now()

	taskData, err := json.Marshal(task)
	if err != nil {
		return err
	}

	return d.client.Set(ctx, "task:"+taskID, taskData, 0).Err()
}
