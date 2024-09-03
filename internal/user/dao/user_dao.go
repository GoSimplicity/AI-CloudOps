package dao

import (
	"context"
	"github.com/GoSimplicity/CloudOps/internal/model"
	"gorm.io/gorm"
	_ "gorm.io/gorm"
	_ "gorm.io/gorm/clause"
)

type UserDAO interface {
	Create(ctx context.Context, user model.User) error
}

type userDAO struct {
	db *gorm.DB
}

func NewUserDAO(db *gorm.DB) UserDAO {
	return &userDAO{
		db: db,
	}
}

func (u userDAO) Create(ctx context.Context, user model.User) error {
	//TODO implement me
	panic("implement me")
}
