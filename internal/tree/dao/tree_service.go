package dao

import "gorm.io/gorm"

type TreeDAO interface {
}

type treeDAO struct {
	db     *gorm.DB
}

func NewTreeDAO(db *gorm.DB) TreeDAO {
	return &treeDAO{
		db: db,
	}
}



