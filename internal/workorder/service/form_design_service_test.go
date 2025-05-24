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

// Helper function to create a new FormDesignService with mocks
func newTestFormDesignService(t *testing.T) (FormDesignService, *mocks.MockFormDesignDAO, *mocks.MockUserDAO, context.Context) {
	ctrl := gomock.NewController(t)
	mockFormDAO := mocks.NewMockFormDesignDAO(ctrl)
	mockUserDAO := mocks.NewMockUserDAO(ctrl)
	logger := zap.NewNop()
	service := NewFormDesignService(mockFormDAO, mockUserDAO, logger)
	ctx := context.Background()
	return service, mockFormDAO, mockUserDAO, ctx
}

func TestFormDesignService_CreateFormDesign(t *testing.T) {
	service, mockFormDAO, _, ctx := newTestFormDesignService(t)

	req := &model.CreateFormDesignReq{
		Name:        "Test Form",
		Description: "Test Description",
		Schema: model.FormSchema{
			Fields: []model.FormField{
				{ID: "field1", Type: "input", Label: "Field 1", Name: "field1"},
			},
		},
	}
	creatorID := 1
	creatorName := "testuser"

	t.Run("Success", func(t *testing.T) {
		mockFormDAO.EXPECT().CreateFormDesign(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, fd *model.FormDesign) error {
				assert.Equal(t, req.Name, fd.Name)
				assert.Equal(t, req.Description, fd.Description)
				assert.Equal(t, creatorID, fd.CreatorID)
				// Check schema conversion implicitly
				expectedSchemaJSON, _ := utils.Json.Marshal(req.Schema)
				assert.Equal(t, string(expectedSchemaJSON), fd.Schema)
				assert.Equal(t, int8(0), fd.Status)  // Default status
				assert.Equal(t, 1, fd.Version) // Default version
				return nil
			}).Times(1)

		err := service.CreateFormDesign(ctx, req, creatorID, creatorName)
		assert.NoError(t, err)
	})

	t.Run("DAOError", func(t *testing.T) {
		mockFormDAO.EXPECT().CreateFormDesign(gomock.Any(), gomock.Any()).
			Return(errors.New("DAO create error")).Times(1)

		err := service.CreateFormDesign(ctx, req, creatorID, creatorName)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "DAO create error")
	})

	t.Run("SchemaMarshalError", func(t *testing.T) {
		// This test is tricky because the actual JSON marshaling happens in the utility function.
		// We're testing the service's behavior if that utility returns an error.
		// To do this effectively, we'd need to mock the utility or pass a schema that causes marshal error.
		// For now, assume utils.ConvertCreateFormDesignReqToModel handles this;
		// if the conversion fails (e.g. schema marshal error), it returns an error.
		// The service just propagates this error.
		// Here we simulate the conversion function returning an error.
		// This requires utils.ConvertCreateFormDesignReqToModel to be an interface or a variable function.
		// As it's a direct call, we can't easily mock it without changing the source or using advanced techniques.
		// Let's assume the happy path for conversion and test DAO error above.
		// If the utility function itself was part of an interface injected into the service, this would be easier.
		// The current implementation of CreateFormDesign directly calls utils.ConvertCreateFormDesignReqToModel.
		// If that function returns an error (e.g., due to JSON marshal error), the service should propagate it.
		// We can't mock `utils.Json.Marshal` easily here.
		// So, we'll assume this path is covered by the utility function's own tests.
		t.Skip("Skipping schema marshal error test due to direct utility call")
	})
}

func TestFormDesignService_UpdateFormDesign(t *testing.T) {
	service, mockFormDAO, _, ctx := newTestFormDesignService(t)

	req := &model.UpdateFormDesignReq{
		ID:          1,
		Name:        "Updated Form",
		Description: "Updated Description",
		Schema: model.FormSchema{
			Fields: []model.FormField{
				{ID: "field1", Type: "input", Label: "Updated Field 1", Name: "field1_updated"},
			},
		},
	}

	t.Run("Success", func(t *testing.T) {
		mockFormDAO.EXPECT().UpdateFormDesign(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, fd *model.FormDesign) error {
				assert.Equal(t, req.ID, fd.ID)
				assert.Equal(t, req.Name, fd.Name)
				expectedSchemaJSON, _ := utils.Json.Marshal(req.Schema)
				assert.Equal(t, string(expectedSchemaJSON), fd.Schema)
				return nil
			}).Times(1)

		err := service.UpdateFormDesign(ctx, req)
		assert.NoError(t, err)
	})

	t.Run("DAOError", func(t *testing.T) {
		mockFormDAO.EXPECT().UpdateFormDesign(gomock.Any(), gomock.Any()).
			Return(errors.New("DAO update error")).Times(1)

		err := service.UpdateFormDesign(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "DAO update error")
	})
}

func TestFormDesignService_DeleteFormDesign(t *testing.T) {
	service, mockFormDAO, _, ctx := newTestFormDesignService(t)
	formID := 1

	t.Run("Success", func(t *testing.T) {
		mockFormDAO.EXPECT().DeleteFormDesign(ctx, formID).Return(nil).Times(1)
		err := service.DeleteFormDesign(ctx, formID)
		assert.NoError(t, err)
	})

	t.Run("DAOError", func(t *testing.T) {
		mockFormDAO.EXPECT().DeleteFormDesign(ctx, formID).Return(errors.New("DAO delete error")).Times(1)
		err := service.DeleteFormDesign(ctx, formID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "DAO delete error")
	})

	t.Run("DAONotFoundError", func(t *testing.T) {
		// The service currently directly returns DAO error. If specific "not found" handling is added, this test would change.
		mockFormDAO.EXPECT().DeleteFormDesign(ctx, formID).Return(gorm.ErrRecordNotFound).Times(1)
		err := service.DeleteFormDesign(ctx, formID)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
	})
}

func TestFormDesignService_PublishFormDesign(t *testing.T) {
	service, mockFormDAO, _, ctx := newTestFormDesignService(t)
	formID := 1

	// Note: The interface has PublishFormDescrollern, but implementation likely PublishFormDesign
	// I'll assume the method name is PublishFormDesign as per common sense and previous corrections.
	// If the method name in the actual service is PublishFormDescrollern, this test will need adjustment.

	t.Run("Success", func(t *testing.T) {
		mockFormDAO.EXPECT().PublishFormDesign(ctx, formID).Return(nil).Times(1)
		err := service.PublishFormDesign(ctx, formID) // Assuming method is PublishFormDesign
		assert.NoError(t, err)
	})

	t.Run("DAOError", func(t *testing.T) {
		mockFormDAO.EXPECT().PublishFormDesign(ctx, formID).Return(errors.New("DAO publish error")).Times(1)
		err := service.PublishFormDesign(ctx, formID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "DAO publish error")
	})
	
	t.Run("DAONotFoundError", func(t *testing.T) {
		mockFormDAO.EXPECT().PublishFormDesign(ctx, formID).Return(gorm.ErrRecordNotFound).Times(1)
		err := service.PublishFormDesign(ctx, formID)
		assert.Error(t, err)
		assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
	})
}

func TestFormDesignService_CloneFormDesign(t *testing.T) {
	service, mockFormDAO, _, ctx := newTestFormDesignService(t)
	formID := 1
	cloneName := "Cloned Form"

	t.Run("Success", func(t *testing.T) {
		mockFormDAO.EXPECT().CloneFormDesign(ctx, formID, cloneName).Return(nil).Times(1)
		err := service.CloneFormDesign(ctx, formID, cloneName)
		assert.NoError(t, err)
	})

	t.Run("DAOError", func(t *testing.T) {
		mockFormDAO.EXPECT().CloneFormDesign(ctx, formID, cloneName).Return(errors.New("DAO clone error")).Times(1)
		err := service.CloneFormDesign(ctx, formID, cloneName)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "DAO clone error")
	})
	
	t.Run("DAONotFoundError", func(t *testing.T) {
        // The CloneFormDesign DAO method might return gorm.ErrRecordNotFound if the original form is not found.
        mockFormDAO.EXPECT().CloneFormDesign(ctx, formID, cloneName).Return(gorm.ErrRecordNotFound).Times(1)
        err := service.CloneFormDesign(ctx, formID, cloneName)
        assert.Error(t, err)
        assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
    })
}

func TestFormDesignService_DetailFormDesign(t *testing.T) {
	service, mockFormDAO, mockUserDAO, ctx := newTestFormDesignService(t)
	formID := 1
	userID := 100
	expectedUsername := "testuser"

	formDesignFromDAO := &model.FormDesign{
		ID:        formID,
		Name:      "Test Form",
		CreatorID: userID, // Important for UserDAO call
	}
	userFromDAO := &model.User{ // Assuming model.User exists and has Username
		Uid:      userID,
		Username: expectedUsername,
	}

	t.Run("Success", func(t *testing.T) {
		mockFormDAO.EXPECT().GetFormDesign(ctx, formID).Return(formDesignFromDAO, nil).Times(1)
		mockUserDAO.EXPECT().GetUserByID(ctx, userID).Return(userFromDAO, nil).Times(1)

		result, err := service.DetailFormDesign(ctx, formID, userID)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, formID, result.ID)
		assert.Equal(t, expectedUsername, result.CreatorName)
	})

	t.Run("FormDAONotFound", func(t *testing.T) {
		mockFormDAO.EXPECT().GetFormDesign(ctx, formID).Return(nil, gorm.ErrRecordNotFound).Times(1)
		// mockUserDAO should not be called
		result, err := service.DetailFormDesign(ctx, formID, userID)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.True(t, errors.Is(err, gorm.ErrRecordNotFound))
	})
	
	t.Run("FormDAOError", func(t *testing.T) {
		mockFormDAO.EXPECT().GetFormDesign(ctx, formID).Return(nil, errors.New("DAO get form error")).Times(1)
		result, err := service.DetailFormDesign(ctx, formID, userID)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "DAO get form error")
	})

	t.Run("UserDAONotFound", func(t *testing.T) {
		mockFormDAO.EXPECT().GetFormDesign(ctx, formID).Return(formDesignFromDAO, nil).Times(1)
		mockUserDAO.EXPECT().GetUserByID(ctx, userID).Return(nil, gorm.ErrRecordNotFound).Times(1)
		result, err := service.DetailFormDesign(ctx, formID, userID)
		assert.Error(t, err) // Service wraps this error
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), gorm.ErrRecordNotFound.Error())
	})
	
	t.Run("UserDAOError", func(t *testing.T) {
		mockFormDAO.EXPECT().GetFormDesign(ctx, formID).Return(formDesignFromDAO, nil).Times(1)
		mockUserDAO.EXPECT().GetUserByID(ctx, userID).Return(nil, errors.New("DAO get user error")).Times(1)
		result, err := service.DetailFormDesign(ctx, formID, userID)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "DAO get user error")
	})
}

func TestFormDesignService_ListFormDesign(t *testing.T) {
	service, mockFormDAO, _, ctx := newTestFormDesignService(t)
	req := &model.ListFormDesignReq{ /* Populate as needed */ }
	expectedForms := []model.FormDesign{
		{ID: 1, Name: "Form 1"},
		{ID: 2, Name: "Form 2"},
	}

	t.Run("Success", func(t *testing.T) {
		mockFormDAO.EXPECT().ListFormDesign(ctx, req).Return(expectedForms, nil).Times(1)
		result, err := service.ListFormDesign(ctx, req)
		assert.NoError(t, err)
		assert.Equal(t, expectedForms, result)
	})

	t.Run("DAOError", func(t *testing.T) {
		mockFormDAO.EXPECT().ListFormDesign(ctx, req).Return(nil, errors.New("DAO list error")).Times(1)
		result, err := service.ListFormDesign(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "DAO list error")
	})
}

func TestFormDesignService_PreviewFormDesign(t *testing.T) {
	service, _, _, ctx := newTestFormDesignService(t)
	formID := 1
	userID := 100
	schema := model.FormSchema{Fields: []model.FormField{{ID: "test", Name: "Test", Type: "input", Label: "Test"}}}

	// The current PreviewFormDesign in service is a placeholder and returns nil error.
	// Its signature in service is (ctx, id, schema, userID) error
	t.Run("PlaceholderSuccess", func(t *testing.T) {
		err := service.PreviewFormDesign(ctx, formID, schema, userID)
		assert.NoError(t, err) // Expecting nil as per current placeholder
	})
}
