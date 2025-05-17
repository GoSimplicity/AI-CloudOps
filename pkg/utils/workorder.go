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
	"encoding/json"
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

// ConvertFormDesignReq 转换表单设计请求
func ConvertFormDesignReq(formDesign *model.FormDesignReq) (*model.FormDesign, error) {
	if formDesign == nil {
		return nil, fmt.Errorf("表单设计请求不能为空")
	}

	formDesignMarshal, err := json.Marshal(formDesign.Schema)
	if err != nil {
		return nil, fmt.Errorf("序列化表单 Schema 失败: %v", err)
	}
	return &model.FormDesign{
		Model: model.Model{
			ID: formDesign.ID,
		},
		Name:        formDesign.Name,
		Description: formDesign.Description,
		Schema:      string(formDesignMarshal),
		Version:     formDesign.Version,
		Status:      formDesign.Status,
		CategoryID:  formDesign.CategoryID,
		CreatorID:   formDesign.CreatorID,
	}, nil
}

// ConvertTemplateReq 转换模板请求
func ConvertTemplateReq(template *model.TemplateReq) (*model.Template, error) {
	if template == nil {
		return nil, fmt.Errorf("模板请求不能为空")
	}

	templateMarshal, err := json.Marshal(template.DefaultValues)
	if err != nil {
		return nil, fmt.Errorf("序列化模板 Schema 失败: %v", err)
	}
	return &model.Template{
		Model: model.Model{
			ID: template.ID,
		},
		Name:          template.Name,
		Description:   template.Description,
		ProcessID:     template.ProcessID,
		DefaultValues: string(templateMarshal),
		Status:        template.Status,
		CategoryID:    template.CategoryID,
		CreatorID:     template.CreatorID,
	}, nil
}

// ConvertCreateProcessReq 转换创建流程请求
func ConvertCreateProcessReq(process *model.CreateProcessReq) (*model.Process, error) {
	if process == nil {
		return nil, fmt.Errorf("流程请求不能为空")
	}

	processMarshal, err := json.Marshal(process.Definition)
	if err != nil {
		return nil, fmt.Errorf("序列化流程 Schema 失败: %v", err)
	}
	return &model.Process{
		Name:         process.Name,
		Description:  process.Description,
		FormDesignID: process.FormDesignID,
		Definition:   string(processMarshal),
		Version:      process.Version,
		Status:       process.Status,
		CategoryID:   process.CategoryID,
		CreatorID:    process.CreatorID,
	}, nil
}

// ConvertProcessReq 转换流程请求
func ConvertUpdateProcessReq(process *model.UpdateProcessReq) (*model.Process, error) {
	if process == nil {
		return nil, fmt.Errorf("流程请求不能为空")
	}

	processMarshal, err := json.Marshal(process.Definition)
	if err != nil {
		return nil, fmt.Errorf("序列化流程 Schema 失败: %v", err)
	}
	return &model.Process{
		Model: model.Model{
			ID: process.ID,
		},
		Name:         process.Name,
		Description:  process.Description,
		FormDesignID: process.FormDesignID,
		Definition:   string(processMarshal),
		Version:      process.Version,
		Status:       process.Status,
		CategoryID:   process.CategoryID,
		CreatorID:    process.CreatorID,
	}, nil
}

// ConvertCreateInstanceReq 转换创建实例请求为实例模型
func ConvertCreateInstanceReq(req *model.CreateInstanceReq) (*model.Instance, error) {
	if req == nil {
		return nil, fmt.Errorf("创建实例请求不能为空")
	}

	formDataMarshal, err := json.Marshal(req.FormData)
	if err != nil {
		return nil, fmt.Errorf("序列化实例表单数据失败: %v", err)
	}
	
	instance := &model.Instance{
		Title:       req.Title,
		WorkflowID:  req.WorkflowID,
		FormData:    string(formDataMarshal),
		CurrentStep: "提交申请", // 默认初始步骤
		CurrentRole: "申请人", // 默认初始角色
		Status:      model.InstanceStatusDraft, // 默认为草稿状态
		Priority:    req.Priority,
		CategoryID:  req.CategoryID,
		// CreatorID 和 CreatorName 需要从上下文中获取，这里不能直接从请求中获取
	}
	
	return instance, nil
}

// ConvertUpdateInstanceReq 转换更新实例请求为实例模型
func ConvertUpdateInstanceReq(req *model.UpdateInstanceReq) (*model.Instance, error) {
	if req == nil {
		return nil, fmt.Errorf("更新实例请求不能为空")
	}

	formDataMarshal, err := json.Marshal(req.FormData)
	if err != nil {
		return nil, fmt.Errorf("序列化实例表单数据失败: %v", err)
	}
	
	instance := &model.Instance{
		Model: model.Model{
			ID: req.ID,
		},
		Title:      req.Title,
		FormData:   string(formDataMarshal),
		Priority:   req.Priority,
		CategoryID: req.CategoryID,
	}
	
	return instance, nil
}

// ConvertInstanceFlowReq 转换实例流程请求
func ConvertInstanceFlowReq(instance *model.InstanceFlowReq) (*model.InstanceFlow, error) {
	if instance == nil {
		return nil, fmt.Errorf("实例流程请求不能为空")
	}

	instanceMarshal, err := json.Marshal(instance.FormData)
	if err != nil {
		return nil, fmt.Errorf("序列化实例流程表单数据失败: %v", err)
	}

	return &model.InstanceFlow{
		InstanceID: instance.InstanceID,
		Step:       "", // 需要从上下文中获取当前步骤
		Action:     instance.Action,
		Comment:    instance.Comment,
		FormData:   string(instanceMarshal),
		// OperatorID 和 OperatorName 需要从上下文中获取
	}, nil
}

// ConvertInstanceCommentReq 转换实例评论请求
func ConvertInstanceCommentReq(instanceComment *model.InstanceCommentReq) (*model.InstanceComment, error) {
	if instanceComment == nil {
		return nil, fmt.Errorf("实例评论请求不能为空")
	}

	return &model.InstanceComment{
		InstanceID: instanceComment.InstanceID,
		Content:    instanceComment.Content,
		ParentID:   instanceComment.ParentID,
		// CreatorID 和 CreatorName 需要从上下文中获取
	}, nil
}
