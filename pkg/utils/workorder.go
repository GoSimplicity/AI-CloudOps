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

func ConvertFormDesignReq(formDesign *model.FormDesignReq) (*model.FormDesign, error) {
	formDesignMarshal, err := json.Marshal(formDesign.Schema)
	if err != nil {
		return nil, fmt.Errorf("序列化表单 Schema 失败: %v", err)
	}
	return &model.FormDesign{
		ID:          formDesign.ID,
		Name:        formDesign.Name,
		Description: formDesign.Description,
		Schema:      string(formDesignMarshal),
		Version:     formDesign.Version,
		Status:      formDesign.Status,
		CategoryID:  formDesign.CategoryID,
		CreatorID:   formDesign.CreatorID,
		CreatorName: formDesign.CreatorName,
	}, nil
}
func ConvertFormDesign(formDesign *model.FormDesign) (*model.FormDesignReq, error) {
	var p model.Schema
	err := json.Unmarshal([]byte(formDesign.Schema), &p)

	if err != nil {
		return nil, fmt.Errorf("序列化表单 Schema 失败: %v", err)
	}
	return &model.FormDesignReq{
		ID:          formDesign.ID,
		Name:        formDesign.Name,
		Description: formDesign.Description,
		Schema:      p,
		Version:     formDesign.Version,
		Status:      formDesign.Status,
		CategoryID:  formDesign.CategoryID,
		CreatorID:   formDesign.CreatorID,
		CreatorName: formDesign.CreatorName,
	}, nil
}

func ConvertProcessReq(process *model.ProcessReq) (*model.Process, error) {
	processMarshal, err := json.Marshal(process.Definition)
	if err != nil {
		return nil, fmt.Errorf("序列化流程 Schema 失败: %v", err)
	}
	return &model.Process{
		ID:           process.ID,
		Name:         process.Name,
		Description:  process.Description,
		FormDesignID: process.FormDesignID,
		Definition:   string(processMarshal),
		Version:      process.Version,
		Status:       process.Status,
		CategoryID:   process.CategoryID,
		CreatorID:    process.CreatorID,
		CreatorName:  process.CreatorName,
	}, nil
}
func ConvertTemplateReq(template *model.TemplateReq) (*model.Template, error) {
	templateMarshal, err := json.Marshal(template.DefaultValues)
	if err != nil {
		return nil, fmt.Errorf("序列化模板 Schema 失败: %v", err)
	}
	return &model.Template{
		ID:            template.ID,
		Name:          template.Name,
		Description:   template.Description,
		ProcessID:     template.ProcessID,
		DefaultValues: string(templateMarshal),
		Status:        template.Status,
		CategoryID:    template.CategoryID,
		CreatorID:     template.CreatorID,
		CreatorName:   template.CreatorName,
	}, nil
}

func ConvertInstanceReq(instance *model.InstanceReq) (*model.Instance, error) {
	instanceMarshal, err := json.Marshal(instance.FormData)
	if err != nil {
		return nil, fmt.Errorf("序列化实例 Schema 失败: %v", err)
	}
	return &model.Instance{
		ID:             instance.ID,
		Title:          instance.Title,
		ProcessID:      instance.ProcessID,
		ProcessVersion: instance.ProcessVersion,
		FormData:       string(instanceMarshal),
		Status:         instance.Status,
		CategoryID:     instance.CategoryID,
		DueDate:        instance.DueDate,
	}, nil
}
func ConvertInstance(instance *model.Instance) (*model.InstanceReq, error) {
	var p model.FormData
	err := json.Unmarshal([]byte(instance.FormData), &p)
	if err != nil {
		return nil, fmt.Errorf("序列化实例 Schema 失败: %v", err)
	}
	return &model.InstanceReq{
		ID:             instance.ID,
		Title:          instance.Title,
		ProcessID:      instance.ProcessID,
		ProcessVersion: instance.ProcessVersion,
		FormData:       p,
		Status:         instance.Status,
		CategoryID:     instance.CategoryID,
	}, nil
}
func ConvertInstanceFlowReq(instance *model.InstanceFlowReq) (*model.InstanceFlow, error) {
	instanceMarshal, err := json.Marshal(instance.FormData)
	if err != nil {
		return nil, fmt.Errorf("序列化实例 Schema 失败: %v", err)

	}
	return &model.InstanceFlow{
		ID:           instance.ID,
		InstanceID:   instance.InstanceID,
		NodeID:       instance.NodeID,
		NodeName:     instance.NodeName,
		Action:       instance.Action,
		TargetUserID: instance.TargetUserID,
		OperatorID:   instance.OperatorID,
		OperatorName: instance.OperatorName,
		Comment:      instance.Comment,
		FormData:     string(instanceMarshal),
		Attachments:  instance.Attachments,
		CreatedAt:    instance.CreatedAt,
	}, nil
}

func ConvertInstanceCommentReq(instanceComment *model.InstanceCommentReq) (*model.InstanceComment, error) {
	return &model.InstanceComment{
		ID:          instanceComment.ID,
		InstanceID:  instanceComment.InstanceID,
		Attachments: instanceComment.Attachments,
		CreatorID:   instanceComment.CreatorID,
		CreatorName: instanceComment.CreatorName,
		CreatedAt:   instanceComment.CreatedAt,
	}, nil
}
