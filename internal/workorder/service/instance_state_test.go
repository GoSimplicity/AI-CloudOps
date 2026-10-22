package service

import (
	"context"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/dao"
	"go.uber.org/zap"
)

type stateGuardDAO struct {
	dao.WorkorderInstanceDAO
	current *model.WorkorderInstance
	updated *model.WorkorderInstance
}

func (d *stateGuardDAO) GetInstanceByID(context.Context, int) (*model.WorkorderInstance, error) {
	return d.current, nil
}
func (d *stateGuardDAO) GetInstanceByTitle(context.Context, string) (*model.WorkorderInstance, error) {
	return nil, dao.ErrInstanceNotFound
}
func (d *stateGuardDAO) UpdateInstance(_ context.Context, p *model.WorkorderInstance) error {
	d.updated = p
	return nil
}

func TestCreateCannotSkipWorkflow(t *testing.T) {
	s := &instanceService{}
	for _, status := range []int8{0, 2, 3, 4, 5, 6} {
		if err := s.CreateInstance(context.Background(), &model.CreateWorkorderInstanceReq{Status: status}); err == nil {
			t.Fatalf("accepted status %d", status)
		}
	}
}

func TestEditCannotChangeWorkflowState(t *testing.T) {
	assignee, other := 7, 8
	now := time.Now()
	for _, tt := range []struct {
		name   string
		req    model.UpdateWorkorderInstanceReq
		reject bool
	}{
		{"status jump", model.UpdateWorkorderInstanceReq{Status: 4}, true},
		{"completion timestamp", model.UpdateWorkorderInstanceReq{CompletedAt: &now}, true},
		{"assignee escalation", model.UpdateWorkorderInstanceReq{AssigneeID: &other}, true},
		{"same state remains compatible", model.UpdateWorkorderInstanceReq{Status: 2, AssigneeID: &assignee, Title: "updated"}, false},
		{"ordinary edit", model.UpdateWorkorderInstanceReq{Title: "updated"}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			d := &stateGuardDAO{current: &model.WorkorderInstance{Status: 2, AssigneeID: &assignee}}
			s := &instanceService{dao: d, logger: zap.NewNop()}
			err := s.UpdateInstance(context.Background(), &tt.req)
			if (err != nil) != tt.reject {
				t.Fatalf("unexpected error %v", err)
			}
			if tt.reject && d.updated != nil {
				t.Fatal("rejected edit reached persistence")
			}
			if !tt.reject && (d.updated.Status != 0 || d.updated.AssigneeID != nil || d.updated.CompletedAt != nil) {
				t.Fatal("ordinary edit attempted to persist state fields")
			}
		})
	}
}
