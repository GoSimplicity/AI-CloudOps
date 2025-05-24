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
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Helper function to create a new ProcessService with mocks
func newTestProcessService(t *testing.T) (ProcessService, *mocks.MockProcessDAO, *mocks.MockUserDAO, context.Context) {
	ctrl := gomock.NewController(t)
	mockProcessDAO := mocks.NewMockProcessDAO(ctrl)
	mockUserDAO := mocks.NewMockUserDAO(ctrl) // UserDAO is a dependency of ProcessService
	logger := zap.NewNop()
	service := NewProcessService(mockProcessDAO, logger, mockUserDAO) // Ensure constructor matches
	ctx := context.Background()
	return service, mockProcessDAO, mockUserDAO, ctx
}

func TestProcessService_CreateProcess(t *testing.T) {
	service, mockProcessDAO, _, ctx := newTestProcessService(t)

	req := &model.CreateProcessReq{
		Name:        "Test Process",
		Description: "Test Description",
		FormDesignID: 1,
		Definition: model.ProcessDefinition{
			Steps: []model.ProcessStep{{ID: "start_node", Name: "Start", Type: "start"}},
		},
	}
	creatorID := 1
	creatorName := "testuser"

	t.Run("Success", func(t *testing.T) {
		mockProcessDAO.EXPECT().CreateProcess(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, proc *model.Process) error {
				assert.Equal(t, req.Name, proc.Name)
				assert.Equal(t, req.Description, proc.Description)
				assert.Equal(t, req.FormDesignID, proc.FormDesignID)
				assert.Equal(t, creatorID, proc.CreatorID)
				assert.Equal(t, creatorName, proc.CreatorName)
				expectedDefJSON, _ := utils.Json.Marshal(req.Definition)
				assert.Equal(t, string(expectedDefJSON), proc.Definition)
				assert.Equal(t, int8(0), proc.Status)  // Default status from converter
				assert.Equal(t, 0, proc.Version) // Converter does not set version, defaults to 0
				return nil
			}).Times(1)

		err := service.CreateProcess(ctx, req, creatorID, creatorName)
		assert.NoError(t, err)
	})

	t.Run("DAOError", func(t *testing.T) {
		mockProcessDAO.EXPECT().CreateProcess(gomock.Any(), gomock.Any()).
			Return(errors.New("DAO create error")).Times(1)

		err := service.CreateProcess(ctx, req, creatorID, creatorName)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "DAO create error")
	})
}

func TestProcessService_UpdateProcess(t *testing.T) {
	service, mockProcessDAO, _, ctx := newTestProcessService(t)

	req := &model.UpdateProcessReq{
		ID:          1,
		Name:        "Updated Process",
		Description: "Updated Description",
		Definition: model.ProcessDefinition{
			Steps: []model.ProcessStep{{ID: "node1", Name: "Step 1", Type: "approve"}},
		},
	}

	t.Run("Success", func(t *testing.T) {
		mockProcessDAO.EXPECT().UpdateProcess(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, proc *model.Process) error {
				assert.Equal(t, req.ID, proc.ID)
				assert.Equal(t, req.Name, proc.Name)
				expectedDefJSON, _ := utils.Json.Marshal(req.Definition)
				assert.Equal(t, string(expectedDefJSON), proc.Definition)
				return nil
			}).Times(1)

		err := service.UpdateProcess(ctx, req)
		assert.NoError(t, err)
	})

	t.Run("DAOError", func(t *testing.T) {
		mockProcessDAO.EXPECT().UpdateProcess(gomock.Any(), gomock.Any()).
			Return(errors.New("DAO update error")).Times(1)

		err := service.UpdateProcess(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "DAO update error")
	})
}

func TestProcessService_DeleteProcess(t *testing.T) {
	service, mockProcessDAO, _, ctx := newTestProcessService(t)
	processID := 1

	t.Run("Success", func(t *testing.T) {
		mockProcessDAO.EXPECT().DeleteProcess(ctx, processID).Return(nil).Times(1)
		err := service.DeleteProcess(ctx, processID)
		assert.NoError(t, err)
	})

	t.Run("DAOError", func(t *testing.T) {
		mockProcessDAO.EXPECT().DeleteProcess(ctx, processID).Return(errors.New("DAO delete error")).Times(1)
		err := service.DeleteProcess(ctx, processID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "DAO delete error")
	})

	t.Run("DAONotFoundError", func(t *testing.T) {
        mockProcessDAO.EXPECT().DeleteProcess(ctx, processID).Return(gorm.ErrRecordNotFound).Times(1)
        err := service.DeleteProcess(ctx, processID)
        assert.Error(t, err)
        assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
    })
}

func TestProcessService_ListProcess(t *testing.T) {
	service, mockProcessDAO, _, ctx := newTestProcessService(t)
	req := model.ListProcessReq{ /* Populate as needed */ }
	expectedProcesses := []model.Process{
		{Model: model.Model{ID: 1}, Name: "Process 1"},
		{Model: model.Model{ID: 2}, Name: "Process 2"},
	}

	t.Run("Success", func(t *testing.T) {
		mockProcessDAO.EXPECT().ListProcess(ctx, req).Return(expectedProcesses, nil).Times(1)
		result, err := service.ListProcess(ctx, req)
		assert.NoError(t, err)
		assert.Equal(t, expectedProcesses, result)
	})

	t.Run("DAOError", func(t *testing.T) {
		mockProcessDAO.EXPECT().ListProcess(ctx, req).Return(nil, errors.New("DAO list error")).Times(1)
		result, err := service.ListProcess(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "DAO list error")
	})
}

func TestProcessService_DetailProcess(t *testing.T) {
	service, mockProcessDAO, mockUserDAO, ctx := newTestProcessService(t)
	processID := 1
	userID := 100
	expectedUsername := "testuser"

	processFromDAO := model.Process{
		Model:     model.Model{ID: processID},
		Name:      "Test Process",
		CreatorID: userID, // Important for UserDAO call
	}
	userFromDAO := &model.User{ // Assuming model.User exists and has Username
		Uid:      userID,
		Username: expectedUsername,
	}

	t.Run("Success", func(t *testing.T) {
		mockProcessDAO.EXPECT().GetProcess(ctx, processID).Return(processFromDAO, nil).Times(1)
		mockUserDAO.EXPECT().GetUserByID(ctx, userID).Return(userFromDAO, nil).Times(1)

		result, err := service.DetailProcess(ctx, processID, userID)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, processID, result.ID)
		assert.Equal(t, expectedUsername, result.CreatorName)
	})

	t.Run("ProcessDAONotFound", func(t *testing.T) {
		mockProcessDAO.EXPECT().GetProcess(ctx, processID).Return(model.Process{}, gorm.ErrRecordNotFound).Times(1)
		_, err := service.DetailProcess(ctx, processID, userID)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
	})
	
	t.Run("ProcessDAOError", func(t *testing.T) {
		mockProcessDAO.EXPECT().GetProcess(ctx, processID).Return(model.Process{}, errors.New("DAO get process error")).Times(1)
		_, err := service.DetailProcess(ctx, processID, userID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "DAO get process error")
	})

	t.Run("UserDAONotFound", func(t *testing.T) {
		mockProcessDAO.EXPECT().GetProcess(ctx, processID).Return(processFromDAO, nil).Times(1)
		mockUserDAO.EXPECT().GetUserByID(ctx, userID).Return(nil, gorm.ErrRecordNotFound).Times(1)
		_, err := service.DetailProcess(ctx, processID, userID)
		assert.Error(t, err) 
		assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
	})
	
	t.Run("UserDAOError", func(t *testing.T) {
		mockProcessDAO.EXPECT().GetProcess(ctx, processID).Return(processFromDAO, nil).Times(1)
		mockUserDAO.EXPECT().GetUserByID(ctx, userID).Return(nil, errors.New("DAO get user error")).Times(1)
		_, err := service.DetailProcess(ctx, processID, userID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "DAO get user error")
	})
}

func TestProcessService_PublishProcess(t *testing.T) {
	service, mockProcessDAO, _, ctx := newTestProcessService(t)
	req := model.PublishProcessReq{ID: 1}

	t.Run("Success", func(t *testing.T) {
		mockProcessDAO.EXPECT().PublishProcess(ctx, req.ID).Return(nil).Times(1)
		err := service.PublishProcess(ctx, req)
		assert.NoError(t, err)
	})

	t.Run("DAOError", func(t *testing.T) {
		mockProcessDAO.EXPECT().PublishProcess(ctx, req.ID).Return(errors.New("DAO publish error")).Times(1)
		err := service.PublishProcess(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "DAO publish error")
	})
	
	t.Run("DAONotFoundError", func(t *testing.T) {
        mockProcessDAO.EXPECT().PublishProcess(ctx, req.ID).Return(gorm.ErrRecordNotFound).Times(1)
        err := service.PublishProcess(ctx, req)
        assert.Error(t, err)
        assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
    })
}

func TestProcessService_CloneProcess(t *testing.T) {
	service, mockProcessDAO, _, ctx := newTestProcessService(t)
	cloneReq := model.CloneProcessReq{ID: 1, Name: "Cloned Process"}
	originalProcess := model.Process{
		Model:     model.Model{ID: 1, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		Name:      "Original Process",
		CreatorID: 10, // CreatorID should be preserved or reset based on business logic
	}

	t.Run("Success", func(t *testing.T) {
		mockProcessDAO.EXPECT().GetProcess(ctx, cloneReq.ID).Return(originalProcess, nil).Times(1)
		mockProcessDAO.EXPECT().CreateProcess(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, clonedProc *model.Process) error {
				assert.Equal(t, cloneReq.Name, clonedProc.Name)
				assert.Equal(t, 0, clonedProc.ID) // ID should be reset for new record
				assert.Equal(t, originalProcess.CreatorID, clonedProc.CreatorID) // Check if creator is preserved
				return nil
			}).Times(1)

		err := service.CloneProcess(ctx, cloneReq)
		assert.NoError(t, err)
	})

	t.Run("GetProcessDAOError", func(t *testing.T) {
		mockProcessDAO.EXPECT().GetProcess(ctx, cloneReq.ID).Return(model.Process{}, errors.New("DAO get error")).Times(1)
		err := service.CloneProcess(ctx, cloneReq)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "DAO get error")
	})
	
	t.Run("GetProcessNotFound", func(t *testing.T) {
        mockProcessDAO.EXPECT().GetProcess(ctx, cloneReq.ID).Return(model.Process{}, gorm.ErrRecordNotFound).Times(1)
        err := service.CloneProcess(ctx, cloneReq)
        assert.Error(t, err)
        assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
    })


	t.Run("CreateProcessDAOError", func(t *testing.T) {
		mockProcessDAO.EXPECT().GetProcess(ctx, cloneReq.ID).Return(originalProcess, nil).Times(1)
		mockProcessDAO.EXPECT().CreateProcess(gomock.Any(), gomock.Any()).Return(errors.New("DAO create error")).Times(1)
		err := service.CloneProcess(ctx, cloneReq)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "DAO create error")
	})
}

func TestProcessService_ValidateProcess(t *testing.T) {
	service, mockProcessDAO, _, ctx := newTestProcessService(t)
	processID := 1
	userID := 1 // For permission check placeholder

	t.Run("SuccessValidDefinition", func(t *testing.T) {
		validDef := model.ProcessDefinition{
			Steps: []model.ProcessStep{
				{ID: "start", Name: "Start", Type: "start"},
				{ID: "end", Name: "End", Type: "end"},
			},
			Connections: []model.ProcessConnection{{From: "start", To: "end"}},
		}
		defJSON, _ := json.Marshal(validDef)
		processFromDAO := model.Process{Model: model.Model{ID: processID}, Definition: string(defJSON)}
		mockProcessDAO.EXPECT().GetProcess(ctx, processID).Return(processFromDAO, nil).Times(1)

		resp, err := service.ValidateProcess(ctx, processID, userID)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.True(t, resp.IsValid)
		assert.Empty(t, resp.Errors)
	})

	t.Run("GetProcessDAOError", func(t *testing.T) {
		mockProcessDAO.EXPECT().GetProcess(ctx, processID).Return(model.Process{}, errors.New("DAO get error")).Times(1)
		resp, err := service.ValidateProcess(ctx, processID, userID)
		assert.Error(t, err) // Service returns wrapped error
		assert.NotNil(t, resp)
		assert.False(t, resp.IsValid)
		assert.Contains(t, resp.Errors[0], "DAO get error")
	})
	
	t.Run("GetProcessNotFound", func(t *testing.T) {
        mockProcessDAO.EXPECT().GetProcess(ctx, processID).Return(model.Process{}, gorm.ErrRecordNotFound).Times(1)
        resp, err := service.ValidateProcess(ctx, processID, userID)
        assert.Error(t, err) 
        assert.NotNil(t, resp)
        assert.False(t, resp.IsValid)
        assert.Contains(t, resp.Errors[0], gorm.ErrRecordNotFound.Error())
    })


	t.Run("EmptyDefinition", func(t *testing.T) {
		processFromDAO := model.Process{Model: model.Model{ID: processID}, Definition: ""}
		mockProcessDAO.EXPECT().GetProcess(ctx, processID).Return(processFromDAO, nil).Times(1)
		resp, err := service.ValidateProcess(ctx, processID, userID)
		assert.NoError(t, err) // Service handles this as validation error, not system error
		assert.NotNil(t, resp)
		assert.False(t, resp.IsValid)
		assert.Contains(t, resp.Errors, "流程定义为空。")
	})

	t.Run("InvalidJSONDefinition", func(t *testing.T) {
		processFromDAO := model.Process{Model: model.Model{ID: processID}, Definition: "{invalid_json"}
		mockProcessDAO.EXPECT().GetProcess(ctx, processID).Return(processFromDAO, nil).Times(1)
		resp, err := service.ValidateProcess(ctx, processID, userID)
		assert.NoError(t, err) // Service handles this as validation error
		assert.NotNil(t, resp)
		assert.False(t, resp.IsValid)
		assert.Contains(t, resp.Errors[0], "解析流程定义JSON失败")
	})

	t.Run("NoSteps", func(t *testing.T) {
		def := model.ProcessDefinition{Steps: []model.ProcessStep{}}
		defJSON, _ := json.Marshal(def)
		processFromDAO := model.Process{Model: model.Model{ID: processID}, Definition: string(defJSON)}
		mockProcessDAO.EXPECT().GetProcess(ctx, processID).Return(processFromDAO, nil).Times(1)
		resp, err := service.ValidateProcess(ctx, processID, userID)
		assert.NoError(t, err)
		assert.False(t, resp.IsValid)
		assert.Contains(t, resp.Errors, "流程至少需要一个步骤。")
	})
	
	t.Run("NoStartStep", func(t *testing.T) {
        def := model.ProcessDefinition{
            Steps: []model.ProcessStep{{ID: "s1", Name: "Step 1", Type: "approve"}},
        }
        defJSON, _ := json.Marshal(def)
        processFromDAO := model.Process{Model:model.Model{ID: processID}, Definition: string(defJSON)}
        mockProcessDAO.EXPECT().GetProcess(ctx, processID).Return(processFromDAO, nil).Times(1)
        resp, err := service.ValidateProcess(ctx, processID, userID)
        assert.NoError(t, err)
        assert.False(t, resp.IsValid)
        assert.Contains(t, resp.Errors, "流程必须包含一个开始（start）类型的步骤。")
    })

    t.Run("NoEndStep", func(t *testing.T) {
        def := model.ProcessDefinition{
            Steps: []model.ProcessStep{{ID: "s1", Name: "Step 1", Type: "start"}},
        }
        defJSON, _ := json.Marshal(def)
        processFromDAO := model.Process{Model:model.Model{ID: processID}, Definition: string(defJSON)}
        mockProcessDAO.EXPECT().GetProcess(ctx, processID).Return(processFromDAO, nil).Times(1)
        resp, err := service.ValidateProcess(ctx, processID, userID)
        assert.NoError(t, err)
        assert.False(t, resp.IsValid)
        assert.Contains(t, resp.Errors, "流程必须包含一个结束（end）类型的步骤。")
    })

    t.Run("DuplicateStepID", func(t *testing.T) {
        def := model.ProcessDefinition{
            Steps: []model.ProcessStep{
				{ID: "start", Name: "Start", Type: "start"},
				{ID: "s1", Name: "Step 1", Type: "approve"},
				{ID: "s1", Name: "Step 2", Type: "approve"}, // Duplicate ID
				{ID: "end", Name: "End", Type: "end"},
			},
			Connections: []model.ProcessConnection{
				{From: "start", To: "s1"},
				{From: "s1", To: "end"},
			},
        }
        defJSON, _ := json.Marshal(def)
        processFromDAO := model.Process{Model:model.Model{ID: processID}, Definition: string(defJSON)}
        mockProcessDAO.EXPECT().GetProcess(ctx, processID).Return(processFromDAO, nil).Times(1)
        resp, err := service.ValidateProcess(ctx, processID, userID)
        assert.NoError(t, err)
        assert.False(t, resp.IsValid)
        assert.Contains(t, resp.Errors, "步骤ID 's1' 重复。")
    })
}
