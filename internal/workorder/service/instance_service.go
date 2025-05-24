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

package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"

	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
)

type InstanceService interface {
	CreateInstance(ctx context.Context, req model.CreateInstanceReq, creatorID int, creatorName string) error
	UpdateInstance(ctx context.Context, req model.UpdateInstanceReq) error
	DeleteInstance(ctx context.Context, id int) error
	ListInstance(ctx context.Context, req model.ListInstanceReq) ([]model.Instance, error) // This might need to return *model.ListResponse
	DetailInstance(ctx context.Context, id int) (model.Instance, error)
	ProcessInstanceFlow(ctx context.Context, req model.InstanceActionReq, operatorID int, operatorName string) error
	CommentInstance(ctx context.Context, req model.InstanceCommentReq, creatorID int, creatorName string) error
	GetProcessDefinition(ctx context.Context, processID int) (*model.ProcessDefinition, error)
	GetInstanceStatistics(ctx context.Context) (interface{}, error)

	// New methods for attachments, flows, comments, my instances
	UploadAttachment(ctx context.Context, instanceID int, fileName string, fileSize int64, filePath string, fileType string, uploaderID int, uploaderName string) (*model.InstanceAttachmentResp, error)
	DeleteAttachment(ctx context.Context, instanceID int, attachmentID int, userID int) error
	GetInstanceFlows(ctx context.Context, instanceID int) ([]model.InstanceFlowResp, error)
	GetInstanceComments(ctx context.Context, instanceID int) ([]model.InstanceCommentResp, error)
	GetInstanceAttachments(ctx context.Context, instanceID int) ([]model.InstanceAttachmentResp, error)
	GetMyInstances(ctx context.Context, req model.MyInstanceReq, userID int) (*model.ListResponse, error)
}

type instanceService struct {
	dao     dao.InstanceDAO
	userDAO userdao.UserDAO // Added userDAO for fetching user details
	l       *zap.Logger
}

func NewInstanceService(dao dao.InstanceDAO, userDAO userdao.UserDAO, l *zap.Logger) InstanceService { // Updated constructor
	return &instanceService{
		dao:     dao,
		userDAO: userDAO,
		l:       l,
	}
}

// convertToInstanceResp converts model.Instance to model.InstanceResp
// TODO: This function might need to populate more fields like CreatorName, AssigneeName, Category, Process, Template
// by fetching related data if not already preloaded.
func convertToInstanceResp(instance *model.Instance) *model.InstanceResp {
	if instance == nil {
		return nil
	}
	var tags []string
	if instance.Tags != "" {
		tags = []string{instance.Tags} // Assuming tags are comma-separated or simple string for now
	}

	var formData map[string]interface{}
	if instance.FormData != "" {
		err := json.Unmarshal([]byte(instance.FormData), &formData)
		if err != nil {
			// Log error, but continue with empty form data for response
		}
	}

	return &model.InstanceResp{
		ID:           instance.ID,
		Title:        instance.Title,
		TemplateID:   instance.TemplateID,
		// Template:    Populate if needed,
		ProcessID:    instance.ProcessID,
		// Process:     Populate if needed,
		FormData:     formData,
		CurrentStep:  instance.CurrentStep,
		Status:       instance.Status,
		Priority:     instance.Priority,
		CategoryID:   instance.CategoryID,
		// Category:    Populate if needed,
		CreatorID:    instance.CreatorID,
		CreatorName:  instance.CreatorName, // Assuming this is pre-populated or fetched
		Description:  instance.Description,
		AssigneeID:   instance.AssigneeID,
		AssigneeName: instance.AssigneeName, // Assuming this is pre-populated or fetched
		CompletedAt:  instance.CompletedAt,
		DueDate:      instance.DueDate,
		Tags:         tags,
		CreatedAt:    instance.CreatedAt,
		UpdatedAt:    instance.UpdatedAt,
		// Extended info (Flows, Comments, Attachments, NextSteps, IsOverdue) would be populated by DetailInstance or similar
	}
}

// convertToInstanceRespList converts a slice of model.Instance to a slice of model.InstanceResp
func convertToInstanceRespList(instances []model.Instance) []model.InstanceResp {
	respList := make([]model.InstanceResp, 0, len(instances))
	for i := range instances {
		respList = append(respList, *convertToInstanceResp(&instances[i]))
	}
	return respList
}


// CreateInstance 创建工单实例
func (i *instanceService) CreateInstance(ctx context.Context, req model.CreateInstanceReq, creatorID int, creatorName string) error {
	// 将请求转换为实例对象
	formDataBytes, err := json.Marshal(req.FormData)
	if err != nil {
		i.l.Error("序列化表单数据失败", zap.Error(err))
		return fmt.Errorf("序列化表单数据失败: %w", err)
	}

	instance := model.Instance{
		Title:       req.Title,
		TemplateID:  req.TemplateID, // Added TemplateID
		ProcessID:   req.ProcessID,
		Description: req.Description,
		FormData:    string(formDataBytes),
		Status:      model.InstanceStatusDraft, // Initial status
		Priority:    req.Priority,
		CategoryID:  req.CategoryID,
		CreatorID:   creatorID,
		CreatorName: creatorName, // Set CreatorName from parameter
		DueDate:     req.DueDate,   // Added DueDate
		// Tags will be handled if CreateInstanceReq includes it and DAO supports it
	}
	if len(req.Tags) > 0 {
		// Assuming tags are stored as a comma-separated string or similar
		instance.Tags = req.Tags[0] // Simplified: just taking the first tag if any
	}


	// 获取流程定义
	process, err := i.dao.GetProcess(ctx, req.ProcessID) // Changed from GetWorkflow to GetProcess
	if err != nil {
		i.l.Error("获取流程定义失败", zap.Error(err), zap.Int("processID", req.ProcessID))
		return fmt.Errorf("获取流程定义失败: %w", err)
	}

	// 设置初始状态为流程的第一步
	var processDef model.ProcessDefinition
	if err := json.Unmarshal([]byte(process.Definition), &processDef); err != nil {
		i.l.Error("解析流程定义失败", zap.Error(err))
		return fmt.Errorf("解析流程定义失败: %w", err)
	}

	if len(processDef.Steps) > 0 {
		// Assuming the first step is the start step
		var startStep *model.ProcessStep
		for _, step := range processDef.Steps {
			if step.Type == "start" {
				startStep = &step
				break
			}
		}

		if startStep != nil {
			instance.CurrentStep = startStep.ID
			instance.Status = model.InstanceStatusProcessing // Set to processing if start step found

			// Basic assignee lookup
			if len(startStep.Users) > 0 {
				instance.AssigneeID = &startStep.Users[0] // Assign to the first user in the list
				// Fetch AssigneeName
				if assigneeUser, err := i.userDAO.GetUserByID(ctx, *instance.AssigneeID); err == nil {
					instance.AssigneeName = assigneeUser.Username
				} else {
					i.l.Warn("创建实例时无法获取指派人姓名", zap.Int("assigneeID", *instance.AssigneeID), zap.Error(err))
				}
			} else if req.AssigneeID != nil { // Fallback to request-provided assignee
				instance.AssigneeID = req.AssigneeID
				if assigneeUser, err := i.userDAO.GetUserByID(ctx, *instance.AssigneeID); err == nil {
					instance.AssigneeName = assigneeUser.Username
				} else {
					i.l.Warn("创建实例时无法获取请求中指派人姓名", zap.Int("assigneeID", *instance.AssigneeID), zap.Error(err))
				}
			} else {
				i.l.Info("创建实例：启动步骤未指定处理人，且请求中也未指定处理人", zap.String("startStepID", startStep.ID))
			}
		} else if len(processDef.Steps) > 0 { // Fallback if no specific start step type found
			instance.CurrentStep = processDef.Steps[0].ID
			instance.Status = model.InstanceStatusProcessing
			if req.AssigneeID != nil {
				instance.AssigneeID = req.AssigneeID
				if assigneeUser, err := i.userDAO.GetUserByID(ctx, *instance.AssigneeID); err == nil {
					instance.AssigneeName = assigneeUser.Username
				}
			}
			i.l.Warn("创建实例：流程定义中未找到明确的 'start' 类型步骤，使用第一个步骤作为开始", zap.String("stepID", instance.CurrentStep))
		} else {
			i.l.Error("创建实例：流程定义中没有步骤", zap.Int("processID", req.ProcessID))
			return fmt.Errorf("流程定义 (ID: %d) 没有步骤", req.ProcessID)
		}
	} else {
		i.l.Error("创建实例：流程定义中没有步骤", zap.Int("processID", req.ProcessID))
		return fmt.Errorf("流程定义 (ID: %d) 没有步骤", req.ProcessID)
	}


	return i.dao.CreateInstance(ctx, instance)
}

// DeleteInstance 删除工单实例
func (i *instanceService) DeleteInstance(ctx context.Context, id int) error {
	// 先检查工单状态，只有草稿状态可以删除
	instance, err := i.dao.GetInstance(ctx, id)
	if err != nil {
		return err
	}

	if instance.Status != model.InstanceStatusDraft {
		return fmt.Errorf("只有草稿状态的工单可以删除")
	}

	return i.dao.DeleteInstance(ctx, id)
}

// DetailInstance 获取工单实例详情
func (i *instanceService) DetailInstance(ctx context.Context, id int) (model.Instance, error) {
	instance, err := i.dao.GetInstance(ctx, id)
	if err != nil {
		return model.Instance{}, err
	}

	// 获取实例相关的流程记录
	flows, err := i.dao.GetInstanceFlows(ctx, id)
	if err != nil {
		i.l.Warn("获取实例流程记录失败", zap.Error(err), zap.Int("instanceID", id))
	} else {
		instance.Flows = flows
	}

	// 获取实例相关的评论
	comments, err := i.dao.GetInstanceComments(ctx, id)
	if err != nil {
		i.l.Warn("获取实例评论失败", zap.Error(err), zap.Int("instanceID", id))
	} else {
		instance.Comments = comments
	}

	return instance, nil
}

// ListInstance 获取工单实例列表
func (i *instanceService) ListInstance(ctx context.Context, req model.ListInstanceReq) ([]model.Instance, error) {
	instances, _, err := i.dao.ListInstance(ctx, req)
	if err != nil {
		return nil, err
	}

	return instances, nil
}

// UpdateInstance 更新工单实例
func (i *instanceService) UpdateInstance(ctx context.Context, req model.UpdateInstanceReq) error {
	// 获取当前实例
	instance, err := i.dao.GetInstance(ctx, req.ID)
	if err != nil {
		return err
	}

	// 只有草稿状态可以更新
	if instance.Status != model.InstanceStatusDraft {
		return fmt.Errorf("只有草稿状态的工单可以更新")
	}

	// 更新表单数据
	formDataBytes, err := json.Marshal(req.FormData)
	if err != nil {
		i.l.Error("序列化表单数据失败", zap.Error(err))
		return fmt.Errorf("序列化表单数据失败: %w", err)
	}

	instance.Title = req.Title
	instance.FormData = string(formDataBytes)
	instance.Priority = req.Priority
	instance.CategoryID = req.CategoryID

	return i.dao.UpdateInstance(ctx, &instance)
}

// ProcessInstanceFlow 处理工单流程
func (i *instanceService) ProcessInstanceFlow(ctx context.Context, req model.InstanceActionReq, operatorID int, operatorName string) error {
	// 1. 获取当前实例信息
	instance, err := i.dao.GetInstance(ctx, req.InstanceID)
	if err != nil {
		i.l.Error("获取实例失败", zap.Error(err), zap.Int("instanceID", req.InstanceID))
		return err
	}

	// 检查工单状态
	if instance.Status != model.InstanceStatusProcessing {
		return fmt.Errorf("当前工单状态不允许此操作")
	}

	// 检查操作人是否为当前处理人或是否有权限操作（例如，管理员）
	// For now, simple check: if AssigneeID is set, operator must be the assignee.
	if instance.AssigneeID != nil && *instance.AssigneeID != 0 && *instance.AssigneeID != operatorID {
		// TODO: Add role-based permission check if needed (e.g. admin override)
		i.l.Warn("处理工单流程权限不足", zap.Int("instanceID", req.InstanceID), zap.Int("operatorID", operatorID), zap.Intp("assigneeID", instance.AssigneeID))
		return fmt.Errorf("您不是当前工单的处理人，无权操作")
	}

	// 2. 创建流程记录
	var formDataStr string
	if req.FormData != nil && len(req.FormData) > 0 {
		// Safely extract known fields if necessary, or just marshal the whole map
		// Example of safe extraction (if a specific field 'rejectionReason' was expected for 'reject' action):
		if req.Action == "reject" {
			if reason, ok := req.FormData["rejectionReason"].(string); ok {
				i.l.Info("提取到拒绝原因", zap.String("reason", reason), zap.Int("instanceID", req.InstanceID))
				// You might store this specific reason in a dedicated field in InstanceFlow if it had one,
				// or ensure it's part of the marshalled formDataStr.
			} else {
				i.l.Info("拒绝操作未提供明确的 'rejectionReason' 字符串", zap.Int("instanceID", req.InstanceID))
			}
		}

		formDataBytes, err := json.Marshal(req.FormData)
		if err != nil {
			i.l.Error("序列化流程表单数据失败", zap.Error(err), zap.Int("instanceID", req.InstanceID))
			return fmt.Errorf("序列化流程表单数据失败: %w", err)
		}
		formDataStr = string(formDataBytes)
	}


	flow := model.InstanceFlow{
		InstanceID:   req.InstanceID,
		StepID:       instance.CurrentStep,
		StepName:     instance.CurrentStep, // Placeholder, will be updated below
		Action:       req.Action,
		OperatorID:   operatorID,
		OperatorName: operatorName,
		Comment:      req.Comment,
		FormData:     formDataStr,
	}

	if err := i.dao.CreateInstanceFlow(ctx, flow); err != nil {
		i.l.Error("创建实例流程记录失败", zap.Error(err))
		return err
	}

	// 3. 根据操作类型更新实例状态
	switch req.Action {
	case "approve":
		// 获取流程定义
		process, err := i.dao.GetProcess(ctx, instance.ProcessID) // Changed from GetWorkflow to GetProcess
		if err != nil {
			i.l.Error("获取流程定义失败", zap.Error(err))
			return err
		}

		var processDef model.ProcessDefinition
		if err := json.Unmarshal([]byte(process.Definition), &processDef); err != nil {
			i.l.Error("解析流程定义失败", zap.Error(err))
			return err
		}

		// 查找当前步骤索引
		currentStepIndex := -1
		for idx, step := range processDef.Steps {
			if step.ID == instance.CurrentStep { // Compare with step.ID
				currentStepIndex = idx
				// TODO: Populate flow.StepName with step.Name
				flow.StepName = step.Name
				break
			}
		}

		// 更新到下一步骤
		if currentStepIndex != -1 {
			flow.StepName = processDef.Steps[currentStepIndex].Name // Set actual step name

			// Determine next step (simplified: assumes first connection from current step's "approve" or default output)
			var nextStepID string
			for _, conn := range processDef.Connections {
				if conn.From == instance.CurrentStep {
					// TODO: Add condition evaluation for conn.Condition if present
					nextStepID = conn.To
					break // Take the first valid connection
				}
			}

			if nextStepID != "" {
				var nextStepDetails *model.ProcessStep
				for _, s := range processDef.Steps {
					if s.ID == nextStepID {
						nextStepDetails = &s
						break
					}
				}

				if nextStepDetails != nil {
					instance.CurrentStep = nextStepDetails.ID
					instance.AssigneeID = nil // Reset assignee, to be determined for the new step
					instance.AssigneeName = ""

					if nextStepDetails.Type == "end" {
						instance.Status = model.InstanceStatusCompleted
						now := time.Now()
						instance.CompletedAt = &now
						i.l.Info("工单已完成", zap.Int("instanceID", instance.ID))
					} else {
						// Basic next assignee lookup
						if len(nextStepDetails.Users) > 0 {
							newAssigneeID := nextStepDetails.Users[0]
							instance.AssigneeID = &newAssigneeID
							if assigneeUser, err := i.userDAO.GetUserByID(ctx, newAssigneeID); err == nil {
								instance.AssigneeName = assigneeUser.Username
							} else {
								i.l.Warn("处理工单流：无法获取下一步指派人姓名", zap.Int("newAssigneeID", newAssigneeID), zap.Error(err))
							}
							i.l.Info("工单流转到下一步，已指派处理人", zap.String("nextStepID", instance.CurrentStep), zap.Intp("assigneeID", instance.AssigneeID))
						} else {
							i.l.Info("工单流转到下一步，未指定处理人", zap.String("nextStepID", instance.CurrentStep))
						}
					}
				} else {
					i.l.Error("处理工单流：未找到下一步骤的详细定义", zap.String("nextStepID", nextStepID), zap.Int("instanceID", instance.ID))
					// Keep instance in current step or handle error state
				}
			} else { // No outgoing connection from this step, assume it's an end point if not already "end" type
				instance.Status = model.InstanceStatusCompleted
				now := time.Now()
				instance.CompletedAt = &now
				i.l.Info("工单已完成（无明确的下一步连接）", zap.Int("instanceID", instance.ID))
			}
		} else {
			i.l.Error("处理工单流：当前步骤未在流程定义中找到", zap.String("currentStep", instance.CurrentStep), zap.Int("instanceID", instance.ID))
			// Potentially keep the instance in the current step or mark as error
		}

	case "reject":
		instance.Status = model.InstanceStatusRejected
		i.l.Info("工单已拒绝", zap.Int("instanceID", instance.ID))
	case "cancel": // Assuming "cancel" might be an action from user
		instance.Status = model.InstanceStatusCancelled
		i.l.Info("工单已取消", zap.Int("instanceID", instance.ID))
	case "transfer":
		if req.AssigneeID == nil || *req.AssigneeID == 0 {
			return fmt.Errorf("转交操作需要指定有效的 AssigneeID")
		}
		instance.AssigneeID = req.AssigneeID
		if assigneeUser, err := i.userDAO.GetUserByID(ctx, *req.AssigneeID); err == nil {
			instance.AssigneeName = assigneeUser.Username
		} else {
			i.l.Error("处理工单流（转交）：获取指派人姓名失败", zap.Int("assigneeID", *req.AssigneeID), zap.Error(err))
			return fmt.Errorf("获取指派人信息失败: %w", err)
		}
		flow.Action = "transfer"             // Ensure action is set
		flow.ToStepID = instance.CurrentStep // Transferring on the same step
		i.l.Info("工单已转交", zap.Int("instanceID", instance.ID), zap.Intp("newAssigneeID", instance.AssigneeID))
	case "revoke": // Placeholder for revoke
		// Revoke logic can be complex: revert to previous step? change status?
		instance.Status = model.InstanceStatusDraft // Example: revert to draft
		i.l.Info("工单已撤销 (示例：状态改回草稿)", zap.Int("instanceID", instance.ID))
	default:
		i.l.Warn("处理工单流：未知的操作类型", zap.String("action", req.Action), zap.Int("instanceID", req.InstanceID))
		return fmt.Errorf("未知的操作类型: %s", req.Action)
	}

	// 4. 保存实例更新 (make sure flow record is saved before this if flow ID is needed in instance)
	if err := i.dao.UpdateInstance(ctx, &instance); err != nil {
			now := time.Now()
			instance.CompletedAt = &now
		}

	case "reject":
		instance.Status = model.InstanceStatusRejected
	case "cancel":
		instance.Status = model.InstanceStatusCancelled
	case "transfer":
		// TODO: 实现转交逻辑，需要更新AssigneeID和AssigneeName
	}

	// 4. 保存实例更新
	if err := i.dao.UpdateInstance(ctx, &instance); err != nil {
		i.l.Error("更新实例失败", zap.Error(err))
		return err
	}

	return nil
}

// CommentInstance 添加工单实例评论
func (i *instanceService) CommentInstance(ctx context.Context, req model.InstanceCommentReq, creatorID int, creatorName string) error {
	comment := model.InstanceComment{
		InstanceID:  req.InstanceID,
		Content:     req.Content,
		CreatorID:   creatorID,
		CreatorName: creatorName,
		ParentID:    req.ParentID,
	}

	if err := i.dao.CreateInstanceComment(ctx, comment); err != nil {
		i.l.Error("创建实例评论失败", zap.Error(err))
		return err
	}

	return nil
}

// GetProcessDefinition 获取流程定义
func (i *instanceService) GetProcessDefinition(ctx context.Context, processID int) (*model.ProcessDefinition, error) {
	process, err := i.dao.GetProcess(ctx, processID) // Changed from GetWorkflow to GetProcess
	if err != nil {
		i.l.Error("获取流程定义失败", zap.Error(err), zap.Int("processID", processID))
		return nil, fmt.Errorf("获取流程定义失败: %w", err)
	}

	var processDef model.ProcessDefinition
	if err := json.Unmarshal([]byte(process.Definition), &processDef); err != nil {
		i.l.Error("解析流程定义失败", zap.Error(err))
		return nil, fmt.Errorf("解析流程定义失败: %w", err)
	}

	return &processDef, nil
}

// UploadAttachment 上传工单附件
func (i *instanceService) UploadAttachment(ctx context.Context, instanceID int, fileName string, fileSize int64, filePath string, fileType string, uploaderID int, uploaderName string) (*model.InstanceAttachmentResp, error) {
	i.l.Info("开始上传工单附件", zap.Int("instanceID", instanceID), zap.String("fileName", fileName), zap.Int("uploaderID", uploaderID))

	attachment := &model.InstanceAttachment{
		InstanceID:   instanceID,
		FileName:     fileName,
		FileSize:     fileSize,
		FilePath:     filePath,
		FileType:     fileType,
		UploaderID:   uploaderID,
		UploaderName: uploaderName, // This is gorm:"-", will not be saved by default unless DAO handles it
	}

	err := i.dao.CreateInstanceAttachment(ctx, attachment) // Assumes this DAO method exists
	if err != nil {
		i.l.Error("上传工单附件失败", zap.Error(err), zap.Int("instanceID", instanceID), zap.String("fileName", fileName))
		return nil, fmt.Errorf("创建附件记录失败: %w", err)
	}

	resp := &model.InstanceAttachmentResp{
		ID:           attachment.ID,
		InstanceID:   attachment.InstanceID,
		FileName:     attachment.FileName,
		FileSize:     attachment.FileSize,
		FilePath:     attachment.FilePath,
		FileType:     attachment.FileType,
		UploaderID:   attachment.UploaderID,
		UploaderName: uploaderName, // Populate from param as it's not in DB model by default
		CreatedAt:    attachment.CreatedAt,
	}
	i.l.Info("工单附件上传成功", zap.Int("attachmentID", resp.ID))
	return resp, nil
}

// DeleteAttachment 删除工单附件
func (i *instanceService) DeleteAttachment(ctx context.Context, instanceID int, attachmentID int, userID int) error {
	i.l.Info("开始删除工单附件", zap.Int("instanceID", instanceID), zap.Int("attachmentID", attachmentID), zap.Int("userID", userID))

	// Optional: Check if attachment exists and if userID has permission (e.g., is uploader or instance admin)
	// att, err := i.dao.GetInstanceAttachment(ctx, attachmentID)
	// if err != nil { ... }
	// if att.UploaderID != userID { return fmt.Errorf("无权删除此附件") }

	err := i.dao.DeleteInstanceAttachment(ctx, attachmentID) // Assumes this DAO method exists
	if err != nil {
		i.l.Error("删除工单附件失败", zap.Error(err), zap.Int("attachmentID", attachmentID))
		return fmt.Errorf("删除附件记录失败: %w", err)
	}
	i.l.Info("工单附件删除成功", zap.Int("attachmentID", attachmentID))
	return nil
}

// GetInstanceFlows 获取工单流程记录
func (i *instanceService) GetInstanceFlows(ctx context.Context, instanceID int) ([]model.InstanceFlowResp, error) {
	i.l.Info("开始获取工单流程记录", zap.Int("instanceID", instanceID))
	flows, err := i.dao.GetInstanceFlows(ctx, instanceID)
	if err != nil {
		i.l.Error("获取工单流程记录失败", zap.Error(err), zap.Int("instanceID", instanceID))
		return nil, fmt.Errorf("获取流程记录失败: %w", err)
	}

	respFlows := make([]model.InstanceFlowResp, 0, len(flows))
	for _, flow := range flows {
		var formData map[string]interface{}
		if flow.FormData != "" {
			if err := json.Unmarshal([]byte(flow.FormData), &formData); err != nil {
				i.l.Warn("解析流程表单数据失败", zap.Error(err), zap.Int("flowID", flow.ID))
				// Continue with formData as nil or empty map
			}
		}
		respFlows = append(respFlows, model.InstanceFlowResp{
			ID:           flow.ID,
			InstanceID:   flow.InstanceID,
			StepID:       flow.StepID,
			StepName:     flow.StepName,
			Action:       flow.Action,
			OperatorID:   flow.OperatorID,
			OperatorName: flow.OperatorName, // Assuming DAO populates this or needs fetching
			Comment:      flow.Comment,
			FormData:     formData,
			Duration:     flow.Duration,
			FromStepID:   flow.FromStepID,
			ToStepID:     flow.ToStepID,
			CreatedAt:    flow.CreatedAt,
		})
	}
	i.l.Info("工单流程记录获取成功", zap.Int("count", len(respFlows)))
	return respFlows, nil
}

// buildCommentTree 构建评论树
func buildCommentTree(comments []model.InstanceComment, parentID *int) []model.InstanceCommentResp {
	tree := make([]model.InstanceCommentResp, 0)
	for _, comment := range comments {
		// Check if comment.ParentID matches the current parentID for tree building
		// This direct comparison works if parentID is *int and comment.ParentID is *int
		var currentCommentParentID *int
		if comment.ParentID != nil {
			currentCommentParentID = comment.ParentID
		}

		if (parentID == nil && currentCommentParentID == nil) || (parentID != nil && currentCommentParentID != nil && *parentID == *currentCommentParentID) {
			children := buildCommentTree(comments, &comment.ID) // Pass address of comment.ID
			respComment := model.InstanceCommentResp{
				ID:          comment.ID,
				InstanceID:  comment.InstanceID,
				Content:     comment.Content,
				CreatorID:   comment.CreatorID,
				CreatorName: comment.CreatorName, // Assuming DAO populates this
				ParentID:    comment.ParentID,
				IsSystem:    comment.IsSystem,
				CreatedAt:   comment.CreatedAt,
				Children:    children,
			}
			tree = append(tree, respComment)
		}
	}
	return tree
}


// GetInstanceComments 获取工单评论 (树形结构)
func (i *instanceService) GetInstanceComments(ctx context.Context, instanceID int) ([]model.InstanceCommentResp, error) {
	i.l.Info("开始获取工单评论", zap.Int("instanceID", instanceID))
	comments, err := i.dao.GetInstanceComments(ctx, instanceID) // This should fetch all comments for the instance
	if err != nil {
		i.l.Error("获取工单评论失败", zap.Error(err), zap.Int("instanceID", instanceID))
		return nil, fmt.Errorf("获取评论失败: %w", err)
	}

	// Build tree structure from flat list
	commentTree := buildCommentTree(comments, nil) // Start with root comments (ParentID is nil)

	i.l.Info("工单评论获取成功", zap.Int("rootCommentCount", len(commentTree)))
	return commentTree, nil
}


// GetInstanceAttachments 获取工单附件列表
func (i *instanceService) GetInstanceAttachments(ctx context.Context, instanceID int) ([]model.InstanceAttachmentResp, error) {
	i.l.Info("开始获取工单附件列表", zap.Int("instanceID", instanceID))
	attachments, err := i.dao.GetInstanceAttachments(ctx, instanceID) // Assumes this DAO method exists
	if err != nil {
		i.l.Error("获取工单附件列表失败", zap.Error(err), zap.Int("instanceID", instanceID))
		return nil, fmt.Errorf("获取附件列表失败: %w", err)
	}

	respAttachments := make([]model.InstanceAttachmentResp, 0, len(attachments))
	for _, att := range attachments {
		respAttachments = append(respAttachments, model.InstanceAttachmentResp{
			ID:           att.ID,
			InstanceID:   att.InstanceID,
			FileName:     att.FileName,
			FileSize:     att.FileSize,
			FilePath:     att.FilePath,
			FileType:     att.FileType,
			UploaderID:   att.UploaderID,
			UploaderName: att.UploaderName, // Assuming DAO populates this
			CreatedAt:    att.CreatedAt,
		})
	}
	i.l.Info("工单附件列表获取成功", zap.Int("count", len(respAttachments)))
	return respAttachments, nil
}

// GetMyInstances 获取与用户相关的工单列表
func (i *instanceService) GetMyInstances(ctx context.Context, req model.MyInstanceReq, userID int) (*model.ListResponse, error) {
	i.l.Info("开始获取我的工单列表", zap.Int("userID", userID), zap.String("type", req.Type))

	listReq := model.ListInstanceReq{
		ListReq:    req.ListReq, // Embed common list parameters (Page, Size, Search, Status from MyInstanceReq)
		Status:     req.Status,  // Pass through status from MyInstanceReq
		Priority:   req.Priority,
		CategoryID: req.CategoryID,
		ProcessID:  req.ProcessID,
		StartDate:  req.StartDate,
		EndDate:    req.EndDate,
	}

	switch req.Type {
	case "created":
		uid := userID
		listReq.CreatorID = &uid
	case "assigned":
		uid := userID
		listReq.AssigneeID = &uid
	default:
		// If type is empty or invalid, could return error or list all related (created + assigned)
		// For now, let's assume if type is not specified, it implies no specific filter on creator/assignee from this logic block
		i.l.Info("GetMyInstances: 'type' 未指定或无效，不按创建者或处理人筛选", zap.String("type", req.Type))
	}

	instances, total, err := i.dao.ListInstance(ctx, listReq)
	if err != nil {
		i.l.Error("获取我的工单列表失败", zap.Error(err), zap.Int("userID", userID), zap.String("type", req.Type))
		return nil, fmt.Errorf("获取我的工单列表失败: %w", err)
	}

	instanceResps := convertToInstanceRespList(instances)

	i.l.Info("我的工单列表获取成功", zap.Int("count", len(instanceResps)), zap.Int64("total", total))
	return &model.ListResponse{
		Total: int(total),
		Items: instanceResps,
	}, nil
}


// GetInstanceStatistics 获取工单统计信息
func (i *instanceService) GetInstanceStatistics(ctx context.Context) (interface{}, error) {
	// 获取各状态工单数量
	stats, err := i.dao.GetInstanceStatistics(ctx)
	if err != nil {
		return nil, err
	}

	// 获取最近工单趋势
	trend, err := i.dao.GetInstanceTrend(ctx)
	if err != nil {
		i.l.Warn("获取工单趋势失败", zap.Error(err))
		trend = []interface{}{}
	}

	return map[string]interface{}{
		"status_count": stats,
		"trend":        trend,
	}, nil
}
