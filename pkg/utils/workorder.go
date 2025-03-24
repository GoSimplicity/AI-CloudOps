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
