package mock

import (
	"github.com/casbin/casbin/v2"
	"gorm.io/gorm"
)

type RoleMock struct {
	db *gorm.DB
	ce *casbin.Enforcer
}

func NewRoleMock(db *gorm.DB, ce *casbin.Enforcer) *RoleMock {
	return &RoleMock{
		db: db,
		ce: ce,
	}
}
