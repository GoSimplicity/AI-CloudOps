package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/service/mocks" // Adjust path if necessary
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Helper function to create a new InstanceService with mocks
func newTestInstanceService(t *testing.T) (
	InstanceService,
	*mocks.MockInstanceDAO,
	*mocks.MockUserDAO, 
	context.Context,
) {
	ctrl := gomock.NewController(t)
	mockInstanceDAO := mocks.NewMockInstanceDAO(ctrl)
	mockUserDAO := mocks.NewMockUserDAO(ctrl)
	logger := zap.NewNop()

	service := NewInstanceService(mockInstanceDAO, mockUserDAO, logger)
	ctx := context.Background()
	return service, mockInstanceDAO, mockUserDAO, ctx
}

func TestInstanceService_CreateInstance(t *testing.T) {
	service, mockInstanceDAO, mockUserDAO, ctx := newTestInstanceService(t)

	creatorID := 1
	creatorName := "testuser"
	processID := 10
	assigneeID := 2
	assigneeName := "assignee_user"

	validProcessDef := model.ProcessDefinition{
		Steps: []model.ProcessStep{
			{ID: "start_node", Name: "Start", Type: "start", Users: []int{assigneeID}},
			{ID: "step2", Name: "Step 2", Type: "approve"},
			{ID: "end_node", Name: "End", Type: "end"},
		},
		Connections: []model.ProcessConnection{{From: "start_node", To: "step2"}, {From: "step2", To: "end_node"}},
	}
	validProcessDefJSON, _ := json.Marshal(validProcessDef)
	mockProcess := model.Process{Model: model.Model{ID: processID}, Definition: string(validProcessDefJSON)}

	reqBase := model.CreateInstanceReq{
		Title:     "Test Instance",
		ProcessID: processID,
		FormData:  map[string]interface{}{"field1": "value1"},
		Priority:  model.PriorityNormal,
	}

	t.Run("Success_WithAssigneeInStartStep", func(t *testing.T) {
		req := reqBase
		req.AssigneeID = nil 

		mockInstanceDAO.EXPECT().GetProcess(ctx, processID).Return(mockProcess, nil).Times(1)
		mockUserDAO.EXPECT().GetUserByID(ctx, assigneeID).Return(&model.User{Uid: assigneeID, Username: assigneeName}, nil).Times(1)
		mockInstanceDAO.EXPECT().CreateInstance(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, inst model.Instance) error {
				assert.Equal(t, req.Title, inst.Title)
				assert.Equal(t, processID, inst.ProcessID)
				assert.Equal(t, creatorID, inst.CreatorID)
				assert.Equal(t, creatorName, inst.CreatorName)
				assert.Equal(t, model.InstanceStatusProcessing, inst.Status)
				assert.Equal(t, "start_node", inst.CurrentStep)
				assert.NotNil(t, inst.AssigneeID)
				assert.Equal(t, assigneeID, *inst.AssigneeID)
				assert.Equal(t, assigneeName, inst.AssigneeName)
				expectedFormData, _ := json.Marshal(req.FormData)
				assert.Equal(t, string(expectedFormData), inst.FormData)
				return nil
			}).Times(1)

		err := service.CreateInstance(ctx, req, creatorID, creatorName)
		assert.NoError(t, err)
	})

	t.Run("Success_WithAssigneeInRequest", func(t *testing.T) {
		req := reqBase
		req.AssigneeID = &assigneeID 

		processDefNoUser := model.ProcessDefinition{
			Steps: []model.ProcessStep{{ID: "start_node_nouser", Name: "Start No User", Type: "start"}},
		}
		processDefNoUserJSON, _ := json.Marshal(processDefNoUser)
		mockProcessNoUser := model.Process{Model: model.Model{ID: processID}, Definition: string(processDefNoUserJSON)}

		mockInstanceDAO.EXPECT().GetProcess(ctx, processID).Return(mockProcessNoUser, nil).Times(1)
		mockUserDAO.EXPECT().GetUserByID(ctx, assigneeID).Return(&model.User{Uid: assigneeID, Username: assigneeName}, nil).Times(1)
		mockInstanceDAO.EXPECT().CreateInstance(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, inst model.Instance) error {
				assert.Equal(t, assigneeID, *inst.AssigneeID)
				assert.Equal(t, assigneeName, inst.AssigneeName)
				assert.Equal(t, "start_node_nouser", inst.CurrentStep)
				return nil
			}).Times(1)

		err := service.CreateInstance(ctx, req, creatorID, creatorName)
		assert.NoError(t, err)
	})
	
	t.Run("Success_NoAssignee", func(t *testing.T) {
		req := reqBase
		req.AssigneeID = nil

		processDefNoUser := model.ProcessDefinition{
			Steps: []model.ProcessStep{{ID: "start_node_nouser", Name: "Start No User", Type: "start"}},
		}
		processDefNoUserJSON, _ := json.Marshal(processDefNoUser)
		mockProcessNoUser := model.Process{Model: model.Model{ID: processID}, Definition: string(processDefNoUserJSON)}

		mockInstanceDAO.EXPECT().GetProcess(ctx, processID).Return(mockProcessNoUser, nil).Times(1)
		mockInstanceDAO.EXPECT().CreateInstance(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, inst model.Instance) error {
				assert.Nil(t, inst.AssigneeID)
				assert.Equal(t, "", inst.AssigneeName)
				assert.Equal(t, "start_node_nouser", inst.CurrentStep)
				return nil
			}).Times(1)

		err := service.CreateInstance(ctx, req, creatorID, creatorName)
		assert.NoError(t, err)
	})


	t.Run("Error_GetProcessFails", func(t *testing.T) {
		req := reqBase
		mockInstanceDAO.EXPECT().GetProcess(ctx, processID).Return(model.Process{}, errors.New("DAO get process error")).Times(1)

		err := service.CreateInstance(ctx, req, creatorID, creatorName)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "DAO get process error")
	})

	t.Run("Error_InvalidProcessDefinitionJSON", func(t *testing.T) {
		req := reqBase
		invalidJSONProcess := model.Process{Model: model.Model{ID: processID}, Definition: "{invalid_json"}
		mockInstanceDAO.EXPECT().GetProcess(ctx, processID).Return(invalidJSONProcess, nil).Times(1)

		err := service.CreateInstance(ctx, req, creatorID, creatorName)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "解析流程定义失败")
	})
	
	t.Run("Error_ProcessDefinitionNoSteps", func(t *testing.T) {
        req := reqBase
        processDefNoSteps := model.ProcessDefinition{Steps: []model.ProcessStep{}}
        processDefNoStepsJSON, _ := json.Marshal(processDefNoSteps)
        mockProcessNoSteps := model.Process{Model: model.Model{ID: processID}, Definition: string(processDefNoStepsJSON)}

        mockInstanceDAO.EXPECT().GetProcess(ctx, processID).Return(mockProcessNoSteps, nil).Times(1)
        err := service.CreateInstance(ctx, req, creatorID, creatorName)
        assert.Error(t, err)
        assert.Contains(t, err.Error(), fmt.Sprintf("流程定义 (ID: %d) 没有步骤", processID))
    })


	t.Run("Error_UserDAOFailsForAssignee", func(t *testing.T) {
		req := reqBase
		req.AssigneeID = nil 

		mockInstanceDAO.EXPECT().GetProcess(ctx, processID).Return(mockProcess, nil).Times(1) 
		mockUserDAO.EXPECT().GetUserByID(ctx, assigneeID).Return(nil, errors.New("DAO get user error")).Times(1)
		mockInstanceDAO.EXPECT().CreateInstance(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, inst model.Instance) error {
				assert.Equal(t, assigneeID, *inst.AssigneeID)
				assert.Equal(t, "", inst.AssigneeName) 
				return nil
			}).Times(1)

		err := service.CreateInstance(ctx, req, creatorID, creatorName)
		assert.NoError(t, err) 
	})

	t.Run("Error_CreateInstanceDAOFails", func(t *testing.T) {
		req := reqBase
		mockInstanceDAO.EXPECT().GetProcess(ctx, processID).Return(mockProcess, nil).Times(1)
		mockUserDAO.EXPECT().GetUserByID(ctx, assigneeID).Return(&model.User{Uid: assigneeID, Username: assigneeName}, nil).Times(1)
		mockInstanceDAO.EXPECT().CreateInstance(gomock.Any(), gomock.Any()).Return(errors.New("DAO create instance error")).Times(1)

		err := service.CreateInstance(ctx, req, creatorID, creatorName)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "DAO create instance error")
	})
}

func TestInstanceService_UpdateInstance(t *testing.T) {
	service, mockInstanceDAO, _, ctx := newTestInstanceService(t)
	instanceID := 1
	req := model.UpdateInstanceReq{
		ID:       instanceID,
		Title:    "Updated Title",
		FormData: map[string]interface{}{"field1": "updated_value"},
		Priority: model.PriorityHigh,
	}

	t.Run("Success", func(t *testing.T) {
		existingInstance := model.Instance{Model: model.Model{ID: instanceID}, Status: model.InstanceStatusDraft}
		mockInstanceDAO.EXPECT().GetInstance(ctx, instanceID).Return(existingInstance, nil).Times(1)
		mockInstanceDAO.EXPECT().UpdateInstance(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, inst *model.Instance) error {
				assert.Equal(t, req.Title, inst.Title)
				expectedFormData, _ := json.Marshal(req.FormData)
				assert.Equal(t, string(expectedFormData), inst.FormData)
				assert.Equal(t, req.Priority, inst.Priority)
				return nil
			}).Times(1)

		err := service.UpdateInstance(ctx, req)
		assert.NoError(t, err)
	})

	t.Run("Error_GetInstanceFails", func(t *testing.T) {
		mockInstanceDAO.EXPECT().GetInstance(ctx, instanceID).Return(model.Instance{}, errors.New("DAO get error")).Times(1)
		err := service.UpdateInstance(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "DAO get error")
	})
	
	t.Run("Error_GetInstanceNotFound", func(t *testing.T) {
        mockInstanceDAO.EXPECT().GetInstance(ctx, instanceID).Return(model.Instance{}, gorm.ErrRecordNotFound).Times(1)
        err := service.UpdateInstance(ctx, req)
        assert.Error(t, err)
        assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
    })


	t.Run("Error_NotDraftStatus", func(t *testing.T) {
		existingInstance := model.Instance{Model: model.Model{ID: instanceID}, Status: model.InstanceStatusProcessing}
		mockInstanceDAO.EXPECT().GetInstance(ctx, instanceID).Return(existingInstance, nil).Times(1)
		err := service.UpdateInstance(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "只有草稿状态的工单可以更新")
	})

	t.Run("Error_UpdateInstanceDAOFails", func(t *testing.T) {
		existingInstance := model.Instance{Model: model.Model{ID: instanceID}, Status: model.InstanceStatusDraft}
		mockInstanceDAO.EXPECT().GetInstance(ctx, instanceID).Return(existingInstance, nil).Times(1)
		mockInstanceDAO.EXPECT().UpdateInstance(gomock.Any(), gomock.Any()).Return(errors.New("DAO update error")).Times(1)
		err := service.UpdateInstance(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "DAO update error")
	})
}

func TestInstanceService_DeleteInstance(t *testing.T) {
	service, mockInstanceDAO, _, ctx := newTestInstanceService(t)
	instanceID := 1

	t.Run("Success", func(t *testing.T) {
		existingInstance := model.Instance{Model: model.Model{ID: instanceID}, Status: model.InstanceStatusDraft}
		mockInstanceDAO.EXPECT().GetInstance(ctx, instanceID).Return(existingInstance, nil).Times(1)
		mockInstanceDAO.EXPECT().DeleteInstance(ctx, instanceID).Return(nil).Times(1)
		err := service.DeleteInstance(ctx, instanceID)
		assert.NoError(t, err)
	})

	t.Run("Error_GetInstanceFails", func(t *testing.T) {
		mockInstanceDAO.EXPECT().GetInstance(ctx, instanceID).Return(model.Instance{}, errors.New("DAO get error")).Times(1)
		err := service.DeleteInstance(ctx, instanceID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "DAO get error")
	})

	t.Run("Error_GetInstanceNotFound", func(t *testing.T) {
		mockInstanceDAO.EXPECT().GetInstance(ctx, instanceID).Return(model.Instance{}, gorm.ErrRecordNotFound).Times(1)
		err := service.DeleteInstance(ctx, instanceID)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
	})
	
	t.Run("Error_NotDraftStatus", func(t *testing.T) {
		existingInstance := model.Instance{Model: model.Model{ID: instanceID}, Status: model.InstanceStatusProcessing}
		mockInstanceDAO.EXPECT().GetInstance(ctx, instanceID).Return(existingInstance, nil).Times(1)
		err := service.DeleteInstance(ctx, instanceID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "只有草稿状态的工单可以删除")
	})

	t.Run("Error_DeleteInstanceDAOFails", func(t *testing.T) {
		existingInstance := model.Instance{Model: model.Model{ID: instanceID}, Status: model.InstanceStatusDraft}
		mockInstanceDAO.EXPECT().GetInstance(ctx, instanceID).Return(existingInstance, nil).Times(1)
		mockInstanceDAO.EXPECT().DeleteInstance(ctx, instanceID).Return(errors.New("DAO delete error")).Times(1)
		err := service.DeleteInstance(ctx, instanceID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "DAO delete error")
	})
}

func TestInstanceService_ListInstance(t *testing.T) {
	service, mockInstanceDAO, _, ctx := newTestInstanceService(t)
	req := model.ListInstanceReq{ ListReq: model.ListReq{Page:1, Size:10}}
	expectedInstances := []model.Instance{{Model: model.Model{ID:1}, Title: "Instance 1"}}
	
	t.Run("Success", func(t *testing.T) {
		mockInstanceDAO.EXPECT().ListInstance(ctx, req).Return(expectedInstances, int64(len(expectedInstances)), nil).Times(1)
		instances, err := service.ListInstance(ctx, req)
		assert.NoError(t, err)
		assert.Equal(t, expectedInstances, instances)
	})

	t.Run("DAOError", func(t *testing.T) {
		mockInstanceDAO.EXPECT().ListInstance(ctx, req).Return(nil, int64(0), errors.New("DAO list error")).Times(1)
		instances, err := service.ListInstance(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, instances)
		assert.Contains(t, err.Error(), "DAO list error")
	})
}

func TestInstanceService_DetailInstance(t *testing.T) {
	service, mockInstanceDAO, _, ctx := newTestInstanceService(t)
	instanceID := 1
	expectedInstance := model.Instance{Model: model.Model{ID: instanceID}, Title: "Detail Instance"}
	expectedFlows := []model.InstanceFlow{{Model: model.Model{ID:1}, StepID: "step1"}}
	expectedComments := []model.InstanceComment{{Model: model.Model{ID:1}, Content: "comment1"}}

	t.Run("Success", func(t *testing.T) {
		mockInstanceDAO.EXPECT().GetInstance(ctx, instanceID).Return(expectedInstance, nil).Times(1)
		mockInstanceDAO.EXPECT().GetInstanceFlows(ctx, instanceID).Return(expectedFlows, nil).Times(1)
		mockInstanceDAO.EXPECT().GetInstanceComments(ctx, instanceID).Return(expectedComments, nil).Times(1)

		instance, err := service.DetailInstance(ctx, instanceID)
		assert.NoError(t, err)
		assert.Equal(t, expectedInstance.ID, instance.ID) 
		assert.Equal(t, expectedFlows, instance.Flows)
		assert.Equal(t, expectedComments, instance.Comments)
	})

	t.Run("Error_GetInstanceFails", func(t *testing.T) {
		mockInstanceDAO.EXPECT().GetInstance(ctx, instanceID).Return(model.Instance{}, errors.New("DAO get error")).Times(1)
		_, err := service.DetailInstance(ctx, instanceID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "DAO get error")
	})
	
	t.Run("Error_GetInstanceNotFound", func(t *testing.T) {
        mockInstanceDAO.EXPECT().GetInstance(ctx, instanceID).Return(model.Instance{}, gorm.ErrRecordNotFound).Times(1)
        _, err := service.DetailInstance(ctx, instanceID)
        assert.Error(t, err)
        assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
    })


	t.Run("Success_GetInstanceFlowsFails_ShouldLogWarningAndContinue", func(t *testing.T) {
		mockInstanceDAO.EXPECT().GetInstance(ctx, instanceID).Return(expectedInstance, nil).Times(1)
		mockInstanceDAO.EXPECT().GetInstanceFlows(ctx, instanceID).Return(nil, errors.New("DAO get flows error")).Times(1)
		mockInstanceDAO.EXPECT().GetInstanceComments(ctx, instanceID).Return(expectedComments, nil).Times(1) 

		instance, err := service.DetailInstance(ctx, instanceID)
		assert.NoError(t, err) 
		assert.Equal(t, expectedInstance.ID, instance.ID)
		assert.Nil(t, instance.Flows) 
		assert.Equal(t, expectedComments, instance.Comments)
	})

	t.Run("Success_GetInstanceCommentsFails_ShouldLogWarningAndContinue", func(t *testing.T) {
		mockInstanceDAO.EXPECT().GetInstance(ctx, instanceID).Return(expectedInstance, nil).Times(1)
		mockInstanceDAO.EXPECT().GetInstanceFlows(ctx, instanceID).Return(expectedFlows, nil).Times(1)
		mockInstanceDAO.EXPECT().GetInstanceComments(ctx, instanceID).Return(nil, errors.New("DAO get comments error")).Times(1)

		instance, err := service.DetailInstance(ctx, instanceID)
		assert.NoError(t, err)
		assert.Equal(t, expectedInstance.ID, instance.ID)
		assert.Equal(t, expectedFlows, instance.Flows)
		assert.Nil(t, instance.Comments) 
	})
}

func TestInstanceService_ProcessInstanceFlow_CommonErrors(t *testing.T) {
	service, mockInstanceDAO, _, ctx := newTestInstanceService(t)
	operatorID := 1
	operatorName := "operator"
	instanceID := 100
	actionReq := model.InstanceActionReq{InstanceID: instanceID, Action: "approve"}

	t.Run("Error_GetInstanceFails", func(t *testing.T) {
		mockInstanceDAO.EXPECT().GetInstance(ctx, instanceID).Return(model.Instance{}, errors.New("DAO get error")).Times(1)
		err := service.ProcessInstanceFlow(ctx, actionReq, operatorID, operatorName)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "DAO get error")
	})

	t.Run("Error_InstanceNotProcessing", func(t *testing.T) {
		mockInstanceDAO.EXPECT().GetInstance(ctx, instanceID).Return(model.Instance{Model: model.Model{ID: instanceID}, Status: model.InstanceStatusDraft}, nil).Times(1)
		err := service.ProcessInstanceFlow(ctx, actionReq, operatorID, operatorName)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "当前工单状态不允许此操作")
	})

	t.Run("Error_OperatorNotAssignee", func(t *testing.T) {
		assigneeID := operatorID + 1 
		mockInstanceDAO.EXPECT().GetInstance(ctx, instanceID).Return(model.Instance{Model: model.Model{ID: instanceID}, Status: model.InstanceStatusProcessing, AssigneeID: &assigneeID}, nil).Times(1)
		err := service.ProcessInstanceFlow(ctx, actionReq, operatorID, operatorName)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "您不是当前工单的处理人")
	})
}

func TestInstanceService_ProcessInstanceFlow_Approve(t *testing.T) {
	service, mockInstanceDAO, mockUserDAO, ctx := newTestInstanceService(t)
	
	operatorID := 1
	operatorName := "op_user"
	instanceID := 1
	processID := 10
	currentStepID := "step1"
	nextStepID := "step2"
	finalStepID := "end_node"
	nextAssigneeID := 2
	nextAssigneeName := "next_assignee"

	processDef := model.ProcessDefinition{
		Steps: []model.ProcessStep{
			{ID: currentStepID, Name: "Step 1", Type: "approve"},
			{ID: nextStepID, Name: "Step 2", Type: "approve", Users: []int{nextAssigneeID}},
			{ID: finalStepID, Name: "End", Type: "end"},
		},
		Connections: []model.ProcessConnection{
			{From: currentStepID, To: nextStepID},
			{From: nextStepID, To: finalStepID},
		},
	}
	processDefJSON, _ := json.Marshal(processDef)
	mockProcess := model.Process{Model: model.Model{ID: processID}, Definition: string(processDefJSON)}
	
	approveReq := model.InstanceActionReq{InstanceID: instanceID, Action: "approve", Comment: "Approved"}

	t.Run("Success_ApproveToNextStep", func(t *testing.T) {
		currentInstance := model.Instance{
			Model:       model.Model{ID: instanceID},
			ProcessID:   processID,
			Status:      model.InstanceStatusProcessing,
			AssigneeID:  &operatorID,
			CurrentStep: currentStepID,
		}
		mockInstanceDAO.EXPECT().GetInstance(ctx, instanceID).Return(currentInstance, nil).Times(1)
		mockInstanceDAO.EXPECT().CreateInstanceFlow(gomock.Any(), gomock.Any()).Return(nil).Times(1)
		mockInstanceDAO.EXPECT().GetProcess(ctx, processID).Return(mockProcess, nil).Times(1)
		mockUserDAO.EXPECT().GetUserByID(ctx, nextAssigneeID).Return(&model.User{Uid: nextAssigneeID, Username: nextAssigneeName}, nil).Times(1)
		mockInstanceDAO.EXPECT().UpdateInstance(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, inst *model.Instance) error {
				assert.Equal(t, nextStepID, inst.CurrentStep)
				assert.Equal(t, nextAssigneeID, *inst.AssigneeID)
				assert.Equal(t, nextAssigneeName, inst.AssigneeName)
				assert.Equal(t, model.InstanceStatusProcessing, inst.Status) 
				return nil
			}).Times(1)

		err := service.ProcessInstanceFlow(ctx, approveReq, operatorID, operatorName)
		assert.NoError(t, err)
	})

	t.Run("Success_ApproveToCompleted", func(t *testing.T) {
		currentInstance := model.Instance{
			Model:       model.Model{ID: instanceID},
			ProcessID:   processID,
			Status:      model.InstanceStatusProcessing,
			AssigneeID:  &operatorID,
			CurrentStep: nextStepID, 
		}
		mockInstanceDAO.EXPECT().GetInstance(ctx, instanceID).Return(currentInstance, nil).Times(1)
		mockInstanceDAO.EXPECT().CreateInstanceFlow(gomock.Any(), gomock.Any()).Return(nil).Times(1)
		mockInstanceDAO.EXPECT().GetProcess(ctx, processID).Return(mockProcess, nil).Times(1)
		mockInstanceDAO.EXPECT().UpdateInstance(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, inst *model.Instance) error {
				assert.Equal(t, finalStepID, inst.CurrentStep)
				assert.Equal(t, model.InstanceStatusCompleted, inst.Status)
				assert.NotNil(t, inst.CompletedAt)
				return nil
			}).Times(1)
		
		err := service.ProcessInstanceFlow(ctx, approveReq, operatorID, operatorName)
		assert.NoError(t, err)
	})

	t.Run("Error_Approve_GetProcessFails", func(t *testing.T) {
		currentInstance := model.Instance{Model: model.Model{ID: instanceID}, ProcessID: processID, Status: model.InstanceStatusProcessing, AssigneeID: &operatorID, CurrentStep: currentStepID}
		mockInstanceDAO.EXPECT().GetInstance(ctx, instanceID).Return(currentInstance, nil).Times(1)
		mockInstanceDAO.EXPECT().CreateInstanceFlow(gomock.Any(), gomock.Any()).Return(nil).Times(1)
		mockInstanceDAO.EXPECT().GetProcess(ctx, processID).Return(model.Process{}, errors.New("DAO get process error")).Times(1)
		
		err := service.ProcessInstanceFlow(ctx, approveReq, operatorID, operatorName)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "DAO get process error")
	})
}

func TestInstanceService_ProcessInstanceFlow_Reject(t *testing.T) {
	service, mockInstanceDAO, _, ctx := newTestInstanceService(t)
	operatorID := 1
	operatorName := "op_user"
	instanceID := 1
	rejectReq := model.InstanceActionReq{InstanceID: instanceID, Action: "reject", Comment: "Rejected"}
	currentInstance := model.Instance{Model: model.Model{ID: instanceID}, Status: model.InstanceStatusProcessing, AssigneeID: &operatorID}

	t.Run("Success_Reject", func(t *testing.T) {
		mockInstanceDAO.EXPECT().GetInstance(ctx, instanceID).Return(currentInstance, nil).Times(1)
		mockInstanceDAO.EXPECT().CreateInstanceFlow(gomock.Any(), gomock.Any()).Return(nil).Times(1)
		mockInstanceDAO.EXPECT().UpdateInstance(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, inst *model.Instance) error {
				assert.Equal(t, model.InstanceStatusRejected, inst.Status)
				return nil
			}).Times(1)
		
		err := service.ProcessInstanceFlow(ctx, rejectReq, operatorID, operatorName)
		assert.NoError(t, err)
	})
}

func TestInstanceService_ProcessInstanceFlow_Transfer(t *testing.T) {
	service, mockInstanceDAO, mockUserDAO, ctx := newTestInstanceService(t)
	operatorID := 1
	operatorName := "op_user"
	instanceID := 1
	newAssigneeID := 2
	newAssigneeName := "new_assignee"
	
	transferReq := model.InstanceActionReq{InstanceID: instanceID, Action: "transfer", AssigneeID: &newAssigneeID, Comment: "Transferred"}
	currentInstance := model.Instance{Model: model.Model{ID: instanceID}, Status: model.InstanceStatusProcessing, AssigneeID: &operatorID, CurrentStep: "some_step"}

	t.Run("Success_Transfer", func(t *testing.T) {
		mockInstanceDAO.EXPECT().GetInstance(ctx, instanceID).Return(currentInstance, nil).Times(1)
		mockInstanceDAO.EXPECT().CreateInstanceFlow(gomock.Any(), gomock.Any()).Return(nil).Times(1)
		mockUserDAO.EXPECT().GetUserByID(ctx, newAssigneeID).Return(&model.User{Uid: newAssigneeID, Username: newAssigneeName}, nil).Times(1)
		mockInstanceDAO.EXPECT().UpdateInstance(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, inst *model.Instance) error {
				assert.Equal(t, newAssigneeID, *inst.AssigneeID)
				assert.Equal(t, newAssigneeName, inst.AssigneeName)
				return nil
			}).Times(1)
		
		err := service.ProcessInstanceFlow(ctx, transferReq, operatorID, operatorName)
		assert.NoError(t, err)
	})

	t.Run("Error_Transfer_MissingAssigneeID", func(t *testing.T) {
		reqMissingAssignee := model.InstanceActionReq{InstanceID: instanceID, Action: "transfer"} 
		mockInstanceDAO.EXPECT().GetInstance(ctx, instanceID).Return(currentInstance, nil).Times(1)
		mockInstanceDAO.EXPECT().CreateInstanceFlow(gomock.Any(), gomock.Any()).Return(nil).Times(1) 
		
		err := service.ProcessInstanceFlow(ctx, reqMissingAssignee, operatorID, operatorName)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "转交操作需要指定有效的 AssigneeID")
	})

	t.Run("Error_Transfer_GetUserFails", func(t *testing.T) {
		mockInstanceDAO.EXPECT().GetInstance(ctx, instanceID).Return(currentInstance, nil).Times(1)
		mockInstanceDAO.EXPECT().CreateInstanceFlow(gomock.Any(), gomock.Any()).Return(nil).Times(1)
		mockUserDAO.EXPECT().GetUserByID(ctx, newAssigneeID).Return(nil, errors.New("DAO get user error")).Times(1)
		
		err := service.ProcessInstanceFlow(ctx, transferReq, operatorID, operatorName)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "获取指派人信息失败")
	})
}

func TestInstanceService_CommentInstance(t *testing.T) {
	service, mockInstanceDAO, _, ctx := newTestInstanceService(t)
	creatorID := 1
	creatorName := "test_commenter"
	req := model.InstanceCommentReq{InstanceID: 1, Content: "This is a comment"}

	t.Run("Success", func(t *testing.T) {
		mockInstanceDAO.EXPECT().CreateInstanceComment(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, comment model.InstanceComment) error {
				assert.Equal(t, req.InstanceID, comment.InstanceID)
				assert.Equal(t, req.Content, comment.Content)
				assert.Equal(t, creatorID, comment.CreatorID)
				assert.Equal(t, creatorName, comment.CreatorName)
				return nil
			}).Times(1)
		err := service.CommentInstance(ctx, req, creatorID, creatorName)
		assert.NoError(t, err)
	})

	t.Run("DAOError", func(t *testing.T) {
		mockInstanceDAO.EXPECT().CreateInstanceComment(gomock.Any(), gomock.Any()).Return(errors.New("DAO create comment error")).Times(1)
		err := service.CommentInstance(ctx, req, creatorID, creatorName)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "DAO create comment error")
	})
}

func TestInstanceService_GetProcessDefinition(t *testing.T) {
	service, mockInstanceDAO, _, ctx := newTestInstanceService(t)
	processID := 1
	validDef := model.ProcessDefinition{Steps: []model.ProcessStep{{ID: "s1", Name: "Step1"}}}
	validDefJSON, _ := json.Marshal(validDef)
	mockProcess := model.Process{Model: model.Model{ID: processID}, Definition: string(validDefJSON)}

	t.Run("Success", func(t *testing.T) {
		mockInstanceDAO.EXPECT().GetProcess(ctx, processID).Return(mockProcess, nil).Times(1)
		def, err := service.GetProcessDefinition(ctx, processID)
		assert.NoError(t, err)
		assert.NotNil(t, def)
		assert.Equal(t, validDef.Steps[0].ID, def.Steps[0].ID)
	})

	t.Run("Error_GetProcessFails", func(t *testing.T) {
		mockInstanceDAO.EXPECT().GetProcess(ctx, processID).Return(model.Process{}, errors.New("DAO get process error")).Times(1)
		def, err := service.GetProcessDefinition(ctx, processID)
		assert.Error(t, err)
		assert.Nil(t, def)
		assert.Contains(t, err.Error(), "DAO get process error")
	})

	t.Run("Error_InvalidJSON", func(t *testing.T) {
		invalidJSONProcess := model.Process{Model: model.Model{ID: processID}, Definition: "{invalid"}
		mockInstanceDAO.EXPECT().GetProcess(ctx, processID).Return(invalidJSONProcess, nil).Times(1)
		def, err := service.GetProcessDefinition(ctx, processID)
		assert.Error(t, err)
		assert.Nil(t, def)
		assert.Contains(t, err.Error(), "解析流程定义失败")
	})
}

func TestInstanceService_UploadAttachment(t *testing.T) {
    service, mockInstanceDAO, _, ctx := newTestInstanceService(t)
    instanceID, uploaderID := 1, 10
    fileName, filePath, fileType, uploaderName := "test.txt", "/path/test.txt", "text/plain", "uploader"
    fileSize := int64(1024)
	now := time.Now()

    t.Run("Success", func(t *testing.T) {
        createdAttachment := &model.InstanceAttachment{
            Model: model.Model{ID: 1, CreatedAt: now},
            InstanceID: instanceID, FileName: fileName, FileSize: fileSize, 
            FilePath: filePath, FileType: fileType, UploaderID: uploaderID,
        }
        mockInstanceDAO.EXPECT().CreateInstanceAttachment(ctx, gomock.Any()).
            Return(createdAttachment, nil).Times(1)
        
        resp, err := service.UploadAttachment(ctx, instanceID, fileName, fileSize, filePath, fileType, uploaderID, uploaderName)
        assert.NoError(t, err)
        assert.NotNil(t, resp)
        assert.Equal(t, createdAttachment.ID, resp.ID)
        assert.Equal(t, uploaderName, resp.UploaderName) // Service populates this
    })

    t.Run("DAOError", func(t *testing.T) {
        mockInstanceDAO.EXPECT().CreateInstanceAttachment(ctx, gomock.Any()).
            Return(nil, errors.New("DAO create attachment error")).Times(1)
        
        _, err := service.UploadAttachment(ctx, instanceID, fileName, fileSize, filePath, fileType, uploaderID, uploaderName)
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "DAO create attachment error")
    })
}

func TestInstanceService_DeleteAttachment(t *testing.T) {
    service, mockInstanceDAO, _, ctx := newTestInstanceService(t)
    instanceID, attachmentID, userID := 1, 10, 100

    t.Run("Success", func(t *testing.T) {
        mockInstanceDAO.EXPECT().DeleteInstanceAttachment(ctx, attachmentID).Return(nil).Times(1)
        err := service.DeleteAttachment(ctx, instanceID, attachmentID, userID) // instanceID not used by current DAO mock
        assert.NoError(t, err)
    })

    t.Run("DAOError", func(t *testing.T) {
        mockInstanceDAO.EXPECT().DeleteInstanceAttachment(ctx, attachmentID).
            Return(errors.New("DAO delete attachment error")).Times(1)
        err := service.DeleteAttachment(ctx, instanceID, attachmentID, userID)
        assert.Error(t, err)
        assert.Contains(t, err.Error(), "DAO delete attachment error")
    })
}

func TestInstanceService_GetInstanceFlows(t *testing.T) {
    service, mockInstanceDAO, _, ctx := newTestInstanceService(t)
    instanceID := 1
    daoFlows := []model.InstanceFlow{
        {Model: model.Model{ID: 1}, StepID: "s1", FormData: `{"key":"val"}`},
    }

    t.Run("Success", func(t *testing.T) {
        mockInstanceDAO.EXPECT().GetInstanceFlows(ctx, instanceID).Return(daoFlows, nil).Times(1)
        respFlows, err := service.GetInstanceFlows(ctx, instanceID)
        assert.NoError(t, err)
        assert.Len(t, respFlows, 1)
        assert.Equal(t, daoFlows[0].StepID, respFlows[0].StepID)
        assert.NotNil(t, respFlows[0].FormData)
    })

    t.Run("DAOError", func(t *testing.T) {
        mockInstanceDAO.EXPECT().GetInstanceFlows(ctx, instanceID).Return(nil, errors.New("DAO error")).Times(1)
        _, err := service.GetInstanceFlows(ctx, instanceID)
        assert.Error(t, err)
    })
}

func TestInstanceService_GetInstanceComments(t *testing.T) {
    service, mockInstanceDAO, _, ctx := newTestInstanceService(t)
    instanceID := 1
    parentID := 10
    daoComments := []model.InstanceComment{
        {Model: model.Model{ID: 1}, Content: "Root comment"},
        {Model: model.Model{ID: 2}, Content: "Child comment", ParentID: &parentID}, // This ParentID won't match ID:1 in this flat list for tree building
    }
    
    // Corrected mock data for tree building
    rootCommentID := 1
    childCommentID := 2
    commentsForTree := []model.InstanceComment{
        {Model: model.Model{ID: rootCommentID}, Content: "Root", CreatorName: "UserA"},
        {Model: model.Model{ID: childCommentID}, Content: "Child", ParentID: &rootCommentID, CreatorName: "UserB"},
    }


    t.Run("Success", func(t *testing.T) {
        mockInstanceDAO.EXPECT().GetInstanceComments(ctx, instanceID).Return(commentsForTree, nil).Times(1)
        respComments, err := service.GetInstanceComments(ctx, instanceID)
        assert.NoError(t, err)
        assert.Len(t, respComments, 1) // One root comment
        assert.Equal(t, "Root", respComments[0].Content)
        assert.Len(t, respComments[0].Children, 1)
        if len(respComments[0].Children) == 1 {
            assert.Equal(t, "Child", respComments[0].Children[0].Content)
        }
    })

    t.Run("DAOError", func(t *testing.T) {
        mockInstanceDAO.EXPECT().GetInstanceComments(ctx, instanceID).Return(nil, errors.New("DAO error")).Times(1)
        _, err := service.GetInstanceComments(ctx, instanceID)
        assert.Error(t, err)
    })
}

func TestInstanceService_GetInstanceAttachments(t *testing.T) {
    service, mockInstanceDAO, _, ctx := newTestInstanceService(t)
    instanceID := 1
    daoAttachments := []model.InstanceAttachment{
        {Model: model.Model{ID: 1}, FileName: "file.txt"},
    }

    t.Run("Success", func(t *testing.T) {
        mockInstanceDAO.EXPECT().GetInstanceAttachments(ctx, instanceID).Return(daoAttachments, nil).Times(1)
        respAtts, err := service.GetInstanceAttachments(ctx, instanceID)
        assert.NoError(t, err)
        assert.Len(t, respAtts, 1)
        assert.Equal(t, daoAttachments[0].FileName, respAtts[0].FileName)
    })

    t.Run("DAOError", func(t *testing.T) {
        mockInstanceDAO.EXPECT().GetInstanceAttachments(ctx, instanceID).Return(nil, errors.New("DAO error")).Times(1)
        _, err := service.GetInstanceAttachments(ctx, instanceID)
        assert.Error(t, err)
    })
}

func TestInstanceService_GetMyInstances(t *testing.T) {
    service, mockInstanceDAO, _, ctx := newTestInstanceService(t)
    userID := 1
    
    t.Run("TypeCreated", func(t *testing.T) {
        req := model.MyInstanceReq{Type: "created", ListReq: model.ListReq{Page: 1, Size: 10}}
        expectedInstances := []model.Instance{{Model: model.Model{ID: 1}, CreatorID: userID}}
        
        mockInstanceDAO.EXPECT().ListInstance(ctx, gomock.Any()).
            DoAndReturn(func(_ context.Context, listReq model.ListInstanceReq) ([]model.Instance, int64, error) {
                assert.NotNil(t, listReq.CreatorID)
                assert.Equal(t, userID, *listReq.CreatorID)
                assert.Nil(t, listReq.AssigneeID)
                return expectedInstances, 1, nil
            }).Times(1)

        resp, err := service.GetMyInstances(ctx, req, userID)
        assert.NoError(t, err)
        assert.Equal(t, 1, resp.Total)
        assert.Len(t, resp.Items, 1)
    })

    t.Run("TypeAssigned", func(t *testing.T) {
        req := model.MyInstanceReq{Type: "assigned", ListReq: model.ListReq{Page: 1, Size: 10}}
		assigneeUserID := userID // For clarity
        expectedInstances := []model.Instance{{Model: model.Model{ID: 2}, AssigneeID: &assigneeUserID}}
        
        mockInstanceDAO.EXPECT().ListInstance(ctx, gomock.Any()).
            DoAndReturn(func(_ context.Context, listReq model.ListInstanceReq) ([]model.Instance, int64, error) {
                assert.NotNil(t, listReq.AssigneeID)
                assert.Equal(t, userID, *listReq.AssigneeID)
                assert.Nil(t, listReq.CreatorID)
                return expectedInstances, 1, nil
            }).Times(1)
        
        resp, err := service.GetMyInstances(ctx, req, userID)
        assert.NoError(t, err)
        assert.Equal(t, 1, resp.Total)
    })

    t.Run("DAOError", func(t *testing.T) {
        req := model.MyInstanceReq{Type: "created"}
        mockInstanceDAO.EXPECT().ListInstance(ctx, gomock.Any()).Return(nil, int64(0), errors.New("DAO list error")).Times(1)
        _, err := service.GetMyInstances(ctx, req, userID)
        assert.Error(t, err)
    })
}

func TestInstanceService_GetInstanceStatistics(t *testing.T) {
    service, mockInstanceDAO, _, ctx := newTestInstanceService(t)
    mockStatsData := map[string]interface{}{"total": 10, "completed": 5}
    
    t.Run("Success", func(t *testing.T) {
        mockInstanceDAO.EXPECT().GetInstanceStatistics(ctx).Return(mockStatsData, nil).Times(1)
		// Note: service.GetInstanceStatistics currently returns (interface{}, error)
		// and the DAO returns (interface{}, error) where interface{} is a map[string]interface{}
		// from previous file content. This structure is from the old code.
		// The subtask asks for this method to be tested.
		// The current service layer returns the map directly.
		
        // In the original service implementation, GetInstanceStatistics calls DAO's GetInstanceStatistics
        // and GetInstanceTrend, then combines them into a map.
        // Let's mock both DAO calls.
        mockTrendData := []interface{}{map[string]interface{}{"date": "2023-01-01", "count": 3}}
        mockInstanceDAO.EXPECT().GetInstanceTrend(ctx).Return(mockTrendData, nil).Times(1)


        stats, err := service.GetInstanceStatistics(ctx)
        assert.NoError(t, err)
        assert.NotNil(t, stats)
		
		resultMap, ok := stats.(map[string]interface{})
        assert.True(t, ok)
        assert.Equal(t, mockStatsData, resultMap["status_count"])
        assert.Equal(t, mockTrendData, resultMap["trend"])
    })

    t.Run("DAOError_GetStatistics", func(t *testing.T) {
        mockInstanceDAO.EXPECT().GetInstanceStatistics(ctx).Return(nil, errors.New("DAO stats error")).Times(1)
        // GetInstanceTrend should not be called if GetInstanceStatistics fails
        _, err := service.GetInstanceStatistics(ctx)
        assert.Error(t, err)
		assert.Contains(t, err.Error(), "DAO stats error")
    })
	
	t.Run("DAOError_GetTrend", func(t *testing.T) {
        mockInstanceDAO.EXPECT().GetInstanceStatistics(ctx).Return(mockStatsData, nil).Times(1)
        mockInstanceDAO.EXPECT().GetInstanceTrend(ctx).Return(nil, errors.New("DAO trend error")).Times(1)
        
		// Service logs a warning for trend error but returns the stats part
        stats, err := service.GetInstanceStatistics(ctx)
        assert.NoError(t, err) // Error is logged, not returned for trend part
		resultMap, ok := stats.(map[string]interface{})
        assert.True(t, ok)
		assert.Equal(t, mockStatsData, resultMap["status_count"])
		assert.Len(t, resultMap["trend"].([]interface{}), 0) // Trend should be empty array
    })
}
