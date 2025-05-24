package service

import (
	"context"
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

// Helper function to create a new TemplateService with mocks
func newTestTemplateService(t *testing.T) (TemplateService, *mocks.MockTemplateDAO, context.Context) {
	ctrl := gomock.NewController(t)
	mockTemplateDAO := mocks.NewMockTemplateDAO(ctrl)
	logger := zap.NewNop()
	// Assuming NewTemplateService does not require UserDAO, if it does, this needs adjustment.
	// Based on previous template_service.go, UserDAO is not a direct dependency of templateService struct.
	service := NewTemplateService(mockTemplateDAO, logger)
	ctx := context.Background()
	return service, mockTemplateDAO, ctx
}

func TestTemplateService_CreateTemplate(t *testing.T) {
	service, mockTemplateDAO, ctx := newTestTemplateService(t)

	req := &model.CreateTemplateReq{
		Name:        "Test Template",
		Description: "Test Description",
		ProcessID:   1,
		DefaultValues: model.TemplateDefaultValues{
			Fields: map[string]interface{}{"field1": "value1"},
		},
	}
	creatorID := 1
	creatorName := "testuser"

	t.Run("Success", func(t *testing.T) {
		mockTemplateDAO.EXPECT().CreateTemplate(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, tmpl *model.Template) error {
				assert.Equal(t, req.Name, tmpl.Name)
				assert.Equal(t, req.Description, tmpl.Description)
				assert.Equal(t, req.ProcessID, tmpl.ProcessID)
				assert.Equal(t, creatorID, tmpl.CreatorID)
				assert.Equal(t, creatorName, tmpl.CreatorName) // Converter sets this
				expectedDV, _ := utils.Json.Marshal(req.DefaultValues)
				assert.Equal(t, string(expectedDV), tmpl.DefaultValues)
				assert.Equal(t, int8(1), tmpl.Status) // Default status from converter
				return nil
			}).Times(1)

		err := service.CreateTemplate(ctx, req, creatorID, creatorName)
		assert.NoError(t, err)
	})

	t.Run("DAOError", func(t *testing.T) {
		mockTemplateDAO.EXPECT().CreateTemplate(gomock.Any(), gomock.Any()).
			Return(errors.New("DAO create error")).Times(1)

		err := service.CreateTemplate(ctx, req, creatorID, creatorName)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "DAO create error")
	})
}

func TestTemplateService_UpdateTemplate(t *testing.T) {
	service, mockTemplateDAO, ctx := newTestTemplateService(t)

	req := &model.UpdateTemplateReq{
		ID:          1,
		Name:        "Updated Template",
		Description: "Updated Description",
		Status:      1, // Explicitly setting status
	}

	t.Run("Success", func(t *testing.T) {
		mockTemplateDAO.EXPECT().UpdateTemplate(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, tmpl *model.Template) error {
				assert.Equal(t, req.ID, tmpl.ID)
				assert.Equal(t, req.Name, tmpl.Name)
				assert.Equal(t, req.Status, tmpl.Status)
				return nil
			}).Times(1)

		err := service.UpdateTemplate(ctx, req)
		assert.NoError(t, err)
	})

	t.Run("DAOError", func(t *testing.T) {
		mockTemplateDAO.EXPECT().UpdateTemplate(gomock.Any(), gomock.Any()).
			Return(errors.New("DAO update error")).Times(1)

		err := service.UpdateTemplate(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "DAO update error")
	})
}

func TestTemplateService_DeleteTemplate(t *testing.T) {
	service, mockTemplateDAO, ctx := newTestTemplateService(t)
	templateID := 1

	t.Run("Success", func(t *testing.T) {
		mockTemplateDAO.EXPECT().DeleteTemplate(ctx, templateID).Return(nil).Times(1)
		err := service.DeleteTemplate(ctx, templateID)
		assert.NoError(t, err)
	})

	t.Run("DAOError", func(t *testing.T) {
		mockTemplateDAO.EXPECT().DeleteTemplate(ctx, templateID).Return(errors.New("DAO delete error")).Times(1)
		err := service.DeleteTemplate(ctx, templateID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "DAO delete error")
	})
	
	t.Run("DAONotFoundError", func(t *testing.T) {
        mockTemplateDAO.EXPECT().DeleteTemplate(ctx, templateID).Return(gorm.ErrRecordNotFound).Times(1)
        err := service.DeleteTemplate(ctx, templateID)
        assert.Error(t, err)
        assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
    })
}

func TestTemplateService_ListTemplate(t *testing.T) {
	service, mockTemplateDAO, ctx := newTestTemplateService(t)
	req := &model.ListTemplateReq{ /* Populate as needed */ }
	expectedTemplates := []model.Template{
		{ID: 1, Name: "Template 1"},
		{ID: 2, Name: "Template 2"},
	}

	t.Run("Success", func(t *testing.T) {
		mockTemplateDAO.EXPECT().ListTemplate(ctx, req).Return(expectedTemplates, nil).Times(1)
		result, err := service.ListTemplate(ctx, req)
		assert.NoError(t, err)
		assert.Equal(t, expectedTemplates, result)
	})

	t.Run("DAOError", func(t *testing.T) {
		mockTemplateDAO.EXPECT().ListTemplate(ctx, req).Return(nil, errors.New("DAO list error")).Times(1)
		result, err := service.ListTemplate(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "DAO list error")
	})
}

func TestTemplateService_DetailTemplate(t *testing.T) {
	service, mockTemplateDAO, ctx := newTestTemplateService(t)
	templateID := 1

	t.Run("Success", func(t *testing.T) {
		expectedTemplate := model.Template{ID: templateID, Name: "Test Template"}
		mockTemplateDAO.EXPECT().GetTemplate(ctx, templateID).Return(expectedTemplate, nil).Times(1)

		result, err := service.DetailTemplate(ctx, templateID)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, &expectedTemplate, result) // Service returns a pointer
	})

	t.Run("DAOError", func(t *testing.T) {
		mockTemplateDAO.EXPECT().GetTemplate(ctx, templateID).Return(model.Template{}, errors.New("DAO get error")).Times(1)
		result, err := service.DetailTemplate(ctx, templateID)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "DAO get error")
	})

	t.Run("DAONotFound", func(t *testing.T) {
		mockTemplateDAO.EXPECT().GetTemplate(ctx, templateID).Return(model.Template{}, gorm.ErrRecordNotFound).Times(1)
		result, err := service.DetailTemplate(ctx, templateID)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
	})
}

func TestTemplateService_EnableTemplate(t *testing.T) {
	service, mockTemplateDAO, ctx := newTestTemplateService(t)
	templateID := 1
	userID := 1 // userID for permission check placeholder

	t.Run("SuccessAlreadyEnabled", func(t *testing.T) {
		existingTemplate := model.Template{ID: templateID, Status: 1}
		mockTemplateDAO.EXPECT().GetTemplate(ctx, templateID).Return(existingTemplate, nil).Times(1)
		// UpdateTemplateStatus should not be called if already enabled
		err := service.EnableTemplate(ctx, templateID, userID)
		assert.NoError(t, err)
	})

	t.Run("SuccessEnable", func(t *testing.T) {
		existingTemplate := model.Template{ID: templateID, Status: 0} // Currently disabled
		mockTemplateDAO.EXPECT().GetTemplate(ctx, templateID).Return(existingTemplate, nil).Times(1)
		mockTemplateDAO.EXPECT().UpdateTemplateStatus(ctx, templateID, int8(1)).Return(nil).Times(1)
		err := service.EnableTemplate(ctx, templateID, userID)
		assert.NoError(t, err)
	})

	t.Run("GetTemplateError", func(t *testing.T) {
		mockTemplateDAO.EXPECT().GetTemplate(ctx, templateID).Return(model.Template{}, errors.New("DAO get error")).Times(1)
		err := service.EnableTemplate(ctx, templateID, userID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "DAO get error")
	})
	
	t.Run("GetTemplateNotFound", func(t *testing.T) {
		mockTemplateDAO.EXPECT().GetTemplate(ctx, templateID).Return(model.Template{}, gorm.ErrRecordNotFound).Times(1)
		err := service.EnableTemplate(ctx, templateID, userID)
		assert.Error(t, err)
        assert.True(t, errors.Is(err, gorm.ErrRecordNotFound)) // Service returns wrapped error
		assert.Contains(t, err.Error(), fmt.Sprintf("获取模板 (ID: %d) 失败", templateID))
	})


	t.Run("UpdateStatusDAOError", func(t *testing.T) {
		existingTemplate := model.Template{ID: templateID, Status: 0}
		mockTemplateDAO.EXPECT().GetTemplate(ctx, templateID).Return(existingTemplate, nil).Times(1)
		mockTemplateDAO.EXPECT().UpdateTemplateStatus(ctx, templateID, int8(1)).Return(errors.New("DAO update status error")).Times(1)
		err := service.EnableTemplate(ctx, templateID, userID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "DAO update status error")
	})
}

func TestTemplateService_DisableTemplate(t *testing.T) {
	service, mockTemplateDAO, ctx := newTestTemplateService(t)
	templateID := 1
	userID := 1

	t.Run("SuccessAlreadyDisabled", func(t *testing.T) {
		existingTemplate := model.Template{ID: templateID, Status: 0}
		mockTemplateDAO.EXPECT().GetTemplate(ctx, templateID).Return(existingTemplate, nil).Times(1)
		err := service.DisableTemplate(ctx, templateID, userID)
		assert.NoError(t, err)
	})

	t.Run("SuccessDisable", func(t *testing.T) {
		existingTemplate := model.Template{ID: templateID, Status: 1} // Currently enabled
		mockTemplateDAO.EXPECT().GetTemplate(ctx, templateID).Return(existingTemplate, nil).Times(1)
		mockTemplateDAO.EXPECT().UpdateTemplateStatus(ctx, templateID, int8(0)).Return(nil).Times(1)
		err := service.DisableTemplate(ctx, templateID, userID)
		assert.NoError(t, err)
	})

	t.Run("GetTemplateError", func(t *testing.T) {
		mockTemplateDAO.EXPECT().GetTemplate(ctx, templateID).Return(model.Template{}, errors.New("DAO get error")).Times(1)
		err := service.DisableTemplate(ctx, templateID, userID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "DAO get error")
	})
	
	t.Run("GetTemplateNotFound", func(t *testing.T) {
        mockTemplateDAO.EXPECT().GetTemplate(ctx, templateID).Return(model.Template{}, gorm.ErrRecordNotFound).Times(1)
        err := service.DisableTemplate(ctx, templateID, userID)
        assert.Error(t, err)
        assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
		assert.Contains(t, err.Error(), fmt.Sprintf("获取模板 (ID: %d) 失败", templateID))
    })


	t.Run("UpdateStatusDAOError", func(t *testing.T) {
		existingTemplate := model.Template{ID: templateID, Status: 1}
		mockTemplateDAO.EXPECT().GetTemplate(ctx, templateID).Return(existingTemplate, nil).Times(1)
		mockTemplateDAO.EXPECT().UpdateTemplateStatus(ctx, templateID, int8(0)).Return(errors.New("DAO update status error")).Times(1)
		err := service.DisableTemplate(ctx, templateID, userID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "DAO update status error")
	})
}
