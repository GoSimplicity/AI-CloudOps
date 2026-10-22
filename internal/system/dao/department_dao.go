package dao

import (
	"context"
	"errors"
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type DepartmentDAO interface {
	List(context.Context, *model.ListReq) (*model.ListResp[*model.Department], error)
	Get(context.Context, int) (*model.Department, error)
	Write(context.Context, int, *model.DepartmentWriteReq) (*model.Department, error)
	Delete(context.Context, int) error
	AssignUser(context.Context, int, int) error
}
type departmentDAO struct{ db *gorm.DB }

func NewDepartmentDAO(db *gorm.DB) DepartmentDAO { return &departmentDAO{db} }
func (d *departmentDAO) List(ctx context.Context, req *model.ListReq) (*model.ListResp[*model.Department], error) {
	result := &model.ListResp[*model.Department]{Items: []*model.Department{}}
	query := d.db.WithContext(ctx).Model(&model.Department{})
	if req.Search != "" {
		query = query.Where("name LIKE ?", "%"+req.Search+"%")
	}
	if err := query.Count(&result.Total).Error; err != nil {
		return nil, err
	}
	page, size := req.Page, req.Size
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 100
	}
	err := query.Order("parent_id, id").Offset((page - 1) * size).Limit(size).Find(&result.Items).Error
	return result, err
}
func (d *departmentDAO) Get(ctx context.Context, id int) (*model.Department, error) {
	var department model.Department
	err := d.db.WithContext(ctx).First(&department, id).Error
	return &department, err
}
func (d *departmentDAO) Write(ctx context.Context, id int, req *model.DepartmentWriteReq) (*model.Department, error) {
	var result model.Department
	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if id != 0 {
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&result, id).Error; err != nil {
				return err
			}
		}
		// Lock every ancestor so deleting a parent and adding a child serialize.
		seen := map[int]bool{}
		for parent := req.ParentID; parent != 0; {
			if parent == id || seen[parent] {
				return errors.New("部门不能以自身或子部门作为上级")
			}
			seen[parent] = true
			var ancestor model.Department
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&ancestor, parent).Error; err != nil {
				return fmt.Errorf("上级部门不存在: %w", err)
			}
			parent = ancestor.ParentID
		}
		var count int64
		if err := tx.Model(&model.Department{}).Where("parent_id = ? AND name = ? AND id <> ?", req.ParentID, req.Name, id).Count(&count).Error; err != nil {
			return err
		}
		if count != 0 {
			return errors.New("同一上级下已存在同名部门")
		}
		if id == 0 {
			result = model.Department{Name: req.Name, ParentID: req.ParentID, Description: req.Description}
			return tx.Create(&result).Error
		}
		if err := tx.Model(&result).Updates(map[string]any{"name": req.Name, "parent_id": req.ParentID, "description": req.Description}).Error; err != nil {
			return err
		}
		return tx.First(&result, id).Error
	})
	return &result, err
}
func (d *departmentDAO) Delete(ctx context.Context, id int) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var department model.Department
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&department, id).Error; err != nil {
			return err
		}
		var children, users int64
		if err := tx.Model(&model.Department{}).Where("parent_id = ?", id).Count(&children).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.User{}).Where("department_id = ?", id).Count(&users).Error; err != nil {
			return err
		}
		if children > 0 || users > 0 {
			return errors.New("部门存在子部门或成员，请先迁移后删除")
		}
		return tx.Delete(&department).Error
	})
}
func (d *departmentDAO) AssignUser(ctx context.Context, userID, departmentID int) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if departmentID != 0 {
			var department model.Department
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&department, departmentID).Error; err != nil {
				return fmt.Errorf("部门不存在: %w", err)
			}
		}
		var user model.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, userID).Error; err != nil {
			return err
		}
		return tx.Model(&user).Update("department_id", departmentID).Error
	})
}
