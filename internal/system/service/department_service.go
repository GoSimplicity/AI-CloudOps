package service

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/system/dao"
)

type DepartmentService interface {
	List(context.Context, *model.ListReq) (*model.ListResp[*model.Department], error)
	Get(context.Context, int) (*model.Department, error)
	Write(context.Context, int, *model.DepartmentWriteReq) (*model.Department, error)
	Delete(context.Context, int) error
	AssignUser(context.Context, int, int) error
}
type departmentService struct{ dao dao.DepartmentDAO }

func NewDepartmentService(d dao.DepartmentDAO) DepartmentService { return &departmentService{d} }
func (s *departmentService) List(ctx context.Context, r *model.ListReq) (*model.ListResp[*model.Department], error) {
	return s.dao.List(ctx, r)
}
func (s *departmentService) Get(ctx context.Context, id int) (*model.Department, error) {
	return s.dao.Get(ctx, id)
}
func (s *departmentService) Write(ctx context.Context, id int, r *model.DepartmentWriteReq) (*model.Department, error) {
	r.Name = strings.TrimSpace(r.Name)
	r.Description = strings.TrimSpace(r.Description)
	if r.Name == "" || utf8.RuneCountInString(r.Name) > 100 || utf8.RuneCountInString(r.Description) > 500 || r.ParentID < 0 {
		return nil, errors.New("部门名称/描述/上级不合法")
	}
	return s.dao.Write(ctx, id, r)
}
func (s *departmentService) Delete(ctx context.Context, id int) error { return s.dao.Delete(ctx, id) }
func (s *departmentService) AssignUser(ctx context.Context, userID, departmentID int) error {
	if userID <= 0 || departmentID < 0 {
		return errors.New("用户或部门ID不合法")
	}
	return s.dao.AssignUser(ctx, userID, departmentID)
}
