package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
	"go.uber.org/zap"
)

type processDAOStub struct {
	createdProcess *model.WorkorderProcess
}

func (s *processDAOStub) CreateProcess(_ context.Context, process *model.WorkorderProcess) error {
	s.createdProcess = process
	return nil
}

func (s *processDAOStub) UpdateProcess(context.Context, *model.WorkorderProcess) error {
	return nil
}

func (s *processDAOStub) DeleteProcess(context.Context, int) error {
	return nil
}

func (s *processDAOStub) ListProcess(context.Context, *model.ListWorkorderProcessReq) ([]*model.WorkorderProcess, int64, error) {
	return nil, 0, nil
}

func (s *processDAOStub) GetProcessByID(context.Context, int) (*model.WorkorderProcess, error) {
	return nil, dao.ErrProcessNotFound
}

func (s *processDAOStub) CheckProcessNameExists(context.Context, string, ...int) (bool, error) {
	return false, nil
}

func (s *processDAOStub) ValidateProcessDefinition(_ context.Context, definition *model.ProcessDefinition) error {
	if definition == nil || len(definition.Steps) == 0 || len(definition.Connections) == 0 {
		return fmt.Errorf("invalid process definition")
	}
	return nil
}

type formDesignDAOStub struct{}

func (s *formDesignDAOStub) CreateFormDesign(context.Context, *model.WorkorderFormDesign) error {
	return nil
}

func (s *formDesignDAOStub) UpdateFormDesign(context.Context, *model.WorkorderFormDesign) error {
	return nil
}

func (s *formDesignDAOStub) DeleteFormDesign(context.Context, int) error {
	return nil
}

func (s *formDesignDAOStub) GetFormDesign(context.Context, int) (*model.WorkorderFormDesign, error) {
	return &model.WorkorderFormDesign{}, nil
}

func (s *formDesignDAOStub) GetFormDesignByName(context.Context, string) (*model.WorkorderFormDesign, error) {
	return nil, dao.ErrFormDesignNotFound
}

func (s *formDesignDAOStub) ListFormDesign(context.Context, *model.ListWorkorderFormDesignReq) ([]*model.WorkorderFormDesign, int64, error) {
	return nil, 0, nil
}

func (s *formDesignDAOStub) CheckFormDesignNameExists(context.Context, string, ...int) (bool, error) {
	return false, nil
}

type categoryDAOStub struct{}

func (s *categoryDAOStub) CreateCategory(context.Context, *model.WorkorderCategory) error {
	return nil
}

func (s *categoryDAOStub) UpdateCategory(context.Context, *model.WorkorderCategory) error {
	return nil
}

func (s *categoryDAOStub) DeleteCategory(context.Context, int) error {
	return nil
}

func (s *categoryDAOStub) ListCategory(context.Context, model.ListWorkorderCategoryReq) ([]*model.WorkorderCategory, int64, error) {
	return nil, 0, nil
}

func (s *categoryDAOStub) ListCategoryByIDs(context.Context, []int) ([]*model.WorkorderCategory, error) {
	return nil, nil
}

func (s *categoryDAOStub) GetCategory(context.Context, int) (*model.WorkorderCategory, error) {
	return &model.WorkorderCategory{}, nil
}

func (s *categoryDAOStub) GetCategoryByName(context.Context, string) (*model.WorkorderCategory, error) {
	return nil, nil
}

type instanceDAOStub struct{}

func (s *instanceDAOStub) CreateInstance(context.Context, *model.WorkorderInstance) error {
	return nil
}

func (s *instanceDAOStub) UpdateInstance(context.Context, *model.WorkorderInstance) error {
	return nil
}

func (s *instanceDAOStub) DeleteInstance(context.Context, int) error {
	return nil
}

func (s *instanceDAOStub) GetInstanceByID(context.Context, int) (*model.WorkorderInstance, error) {
	return nil, dao.ErrInstanceNotFound
}

func (s *instanceDAOStub) GetInstanceByTitle(context.Context, string) (*model.WorkorderInstance, error) {
	return nil, dao.ErrInstanceNotFound
}

func (s *instanceDAOStub) ListInstance(context.Context, *model.ListWorkorderInstanceReq) ([]*model.WorkorderInstance, int64, error) {
	return nil, 0, nil
}

func (s *instanceDAOStub) GenerateSerialNumber(context.Context) (string, error) {
	return "", nil
}

func (s *instanceDAOStub) UpdateInstanceStatus(context.Context, int, int8) error {
	return nil
}

func (s *instanceDAOStub) UpdateInstanceAssignee(context.Context, int, *int) error {
	return nil
}

func TestBuildProcessDefinitionMapUsesDefaultWhenEmpty(t *testing.T) {
	definition, definitionMap, err := buildProcessDefinitionMap(model.ProcessDefinition{})
	if err != nil {
		t.Fatalf("buildProcessDefinitionMap returned error: %v", err)
	}

	if len(definition.Steps) == 0 {
		t.Fatal("expected default process definition to include steps")
	}
	if len(definition.Connections) == 0 {
		t.Fatal("expected default process definition to include connections")
	}
	if definitionMap == nil {
		t.Fatal("expected definition map to be non-nil")
	}
	if _, ok := definitionMap["steps"]; !ok {
		t.Fatal("expected definition map to include steps")
	}
	if _, ok := definitionMap["connections"]; !ok {
		t.Fatal("expected definition map to include connections")
	}
}

func TestBuildProcessDefinitionMapPreservesProvidedDefinition(t *testing.T) {
	input := model.ProcessDefinition{
		Steps: []model.ProcessStep{
			{ID: "start", Type: model.ProcessStepTypeStart, Name: "Start", SortOrder: 1},
			{ID: "end", Type: model.ProcessStepTypeEnd, Name: "End", SortOrder: 2},
		},
		Connections: []model.ProcessConnection{
			{From: "start", To: "end"},
		},
	}

	definition, definitionMap, err := buildProcessDefinitionMap(input)
	if err != nil {
		t.Fatalf("buildProcessDefinitionMap returned error: %v", err)
	}

	if len(definition.Steps) != len(input.Steps) {
		t.Fatalf("expected %d steps, got %d", len(input.Steps), len(definition.Steps))
	}
	if definition.Steps[0].ID != input.Steps[0].ID {
		t.Fatalf("expected first step ID %q, got %q", input.Steps[0].ID, definition.Steps[0].ID)
	}
	if definitionMap == nil {
		t.Fatal("expected definition map to be non-nil")
	}
}

func TestCreateWorkorderProcessDefaultsDefinitionWhenMissing(t *testing.T) {
	processDAO := &processDAOStub{}
	svc := NewWorkorderProcessService(
		processDAO,
		&formDesignDAOStub{},
		&categoryDAOStub{},
		&instanceDAOStub{},
		zap.NewNop(),
	)

	err := svc.CreateWorkorderProcess(context.Background(), &model.CreateWorkorderProcessReq{
		Name:         "test",
		FormDesignID: 1,
		Status:       model.ProcessStatusDraft,
		OperatorID:   1,
		OperatorName: "admin",
		IsDefault:    2,
	})
	if err != nil {
		t.Fatalf("CreateWorkorderProcess returned error: %v", err)
	}

	if processDAO.createdProcess == nil {
		t.Fatal("expected process to be created")
	}
	if processDAO.createdProcess.Definition == nil {
		t.Fatal("expected created process definition to be non-nil")
	}
	if _, ok := processDAO.createdProcess.Definition["steps"]; !ok {
		t.Fatal("expected created process definition to include steps")
	}
	if _, ok := processDAO.createdProcess.Definition["connections"]; !ok {
		t.Fatal("expected created process definition to include connections")
	}
}

type updateProcessDAO struct {
	processDAOStub
	existing, updated *model.WorkorderProcess
}

func (d *updateProcessDAO) GetProcessByID(context.Context, int) (*model.WorkorderProcess, error) {
	return d.existing, nil
}
func (d *updateProcessDAO) UpdateProcess(_ context.Context, p *model.WorkorderProcess) error {
	d.updated = p
	return nil
}
func TestPartialProcessUpdate(t *testing.T) {
	empty := ""
	for _, tt := range []struct {
		name        string
		description *string
		want        string
	}{
		{"omitted description preserved", nil, "keep"}, {"explicit description cleared", &empty, ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			d := &updateProcessDAO{existing: &model.WorkorderProcess{Model: model.Model{ID: 7}, Name: "existing", Description: "keep", FormDesignID: 9, Status: 2, IsDefault: 2, Definition: model.JSONMap{"steps": []any{}}}}
			s := &workorderProcessService{dao: d, logger: zap.NewNop()}
			if err := s.UpdateWorkorderProcess(context.Background(), &model.UpdateWorkorderProcessReq{ID: 7, Status: 3, Description: tt.description}); err != nil {
				t.Fatal(err)
			}
			if d.updated.Name != "existing" || d.updated.FormDesignID != 9 || d.updated.Status != 3 || d.updated.IsDefault != 2 || d.updated.Description != tt.want || d.updated.Definition == nil {
				t.Fatalf("partial update lost persisted fields: %+v", d.updated)
			}
		})
	}
}
