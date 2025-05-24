package utils

import (
	"encoding/json"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

// ConvertCreateFormDesignReqToModel converts a CreateFormDesignReq to a FormDesign model.
func ConvertCreateFormDesignReqToModel(req *model.CreateFormDesignReq) (*model.FormDesign, error) {
	schemaBytes, err := json.Marshal(req.Schema)
	if err != nil {
		return nil, err
	}
	return &model.FormDesign{
		Name:        req.Name,
		Description: req.Description,
		Schema:      string(schemaBytes),
		CategoryID:  req.CategoryID,
		// CreatorID and Version will be set by the service layer or default values in DB
	}, nil
}

// ConvertUpdateFormDesignReqToModel converts an UpdateFormDesignReq to a FormDesign model.
func ConvertUpdateFormDesignReqToModel(req *model.UpdateFormDesignReq) (*model.FormDesign, error) {
	schemaBytes, err := json.Marshal(req.Schema)
	if err != nil {
		return nil, err
	}
	return &model.FormDesign{
		Model:       model.Model{ID: req.ID},
		Name:        req.Name,
		Description: req.Description,
		Schema:      string(schemaBytes),
		CategoryID:  req.CategoryID,
		// Version and Status might be handled by the service layer if they are updatable through this request.
	}, nil
}

// ConvertCreateTemplateReqToModel converts a CreateTemplateReq to a Template model.
// CreatorID and CreatorName should be passed from the authenticated user context.
func ConvertCreateTemplateReqToModel(req *model.CreateTemplateReq, creatorID int, creatorName string) (*model.Template, error) {
	defaultValuesBytes, err := json.Marshal(req.DefaultValues)
	if err != nil {
		return nil, err
	}
	return &model.Template{
		Name:          req.Name,
		Description:   req.Description,
		ProcessID:     req.ProcessID,
		DefaultValues: string(defaultValuesBytes),
		Icon:          req.Icon,
		CategoryID:    req.CategoryID,
		SortOrder:     req.SortOrder,
		CreatorID:     creatorID,
		CreatorName:   creatorName, // CreatorName is often not stored in DB but shown in responses.
		                             // The model.Template has CreatorName as gorm:"-", so it's fine.
		Status:        1, // Default to enabled
	}, nil
}

// ConvertUpdateTemplateReqToModel converts an UpdateTemplateReq to a Template model.
func ConvertUpdateTemplateReqToModel(req *model.UpdateTemplateReq) (*model.Template, error) {
	defaultValuesBytes, err := json.Marshal(req.DefaultValues)
	if err != nil {
		return nil, err
	}
	return &model.Template{
		Model:         model.Model{ID: req.ID},
		Name:          req.Name,
		Description:   req.Description,
		ProcessID:     req.ProcessID,
		DefaultValues: string(defaultValuesBytes),
		Icon:          req.Icon,
		CategoryID:    req.CategoryID,
		SortOrder:     req.SortOrder,
		Status:        req.Status,
	}, nil
}

// ConvertCreateProcessReqToModel converts a CreateProcessReq to a Process model.
// CreatorID and CreatorName should be passed from the authenticated user context.
func ConvertCreateProcessReqToModel(req *model.CreateProcessReq, creatorID int, creatorName string) (*model.Process, error) {
	definitionBytes, err := json.Marshal(req.Definition)
	if err != nil {
		return nil, err
	}
	return &model.Process{
		Name:         req.Name,
		Description:  req.Description,
		FormDesignID: req.FormDesignID,
		Definition:   string(definitionBytes),
		CategoryID:   req.CategoryID,
		CreatorID:    creatorID,
		CreatorName:  creatorName, // gorm:"-"
		Status:       0, // Default to draft
	}, nil
}

// ConvertUpdateProcessReqToModel converts an UpdateProcessReq to a Process model.
func ConvertUpdateProcessReqToModel(req *model.UpdateProcessReq) (*model.Process, error) {
	definitionBytes, err := json.Marshal(req.Definition)
	if err != nil {
		return nil, err
	}
	return &model.Process{
		Model:        model.Model{ID: req.ID},
		Name:         req.Name,
		Description:  req.Description,
		FormDesignID: req.FormDesignID,
		Definition:   string(definitionBytes),
		CategoryID:   req.CategoryID,
		// Status and Version might be handled by the service layer.
	}, nil
}
