package mock

import (
	"log"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"gorm.io/gorm"
)

type ApiMock struct {
	db *gorm.DB
}

func NewApiMock(db *gorm.DB) *ApiMock {
	return &ApiMock{
		db: db,
	}
}
func (m *ApiMock) InitApi() {
	log.Println("[API Mock开始]")
	
	apis := []model.Api{
		{ID: 1, Path: "/*", Method: 1, Name: "所有接口GET权限", Description: "所有接口GET权限", Version: "v1", Category: 1, IsPublic: 1},
		{ID: 2, Path: "/*", Method: 2, Name: "所有接口POST权限", Description: "所有接口POST权限", Version: "v1", Category: 1, IsPublic: 1}, 
		{ID: 3, Path: "/*", Method: 3, Name: "所有接口PUT权限", Description: "所有接口PUT权限", Version: "v1", Category: 1, IsPublic: 1},
		{ID: 4, Path: "/*", Method: 4, Name: "所有接口DELETE权限", Description: "所有接口DELETE权限", Version: "v1", Category: 1, IsPublic: 1},
	}

	for _, api := range apis {
		// 使用FirstOrCreate方法,如果记录存在则跳过,不存在则创建
		result := m.db.Where("id = ?", api.ID).FirstOrCreate(&api)
		if result.Error != nil {
			log.Printf("创建API记录失败: %v", result.Error)
			continue
		}

		if result.RowsAffected == 1 {
			log.Printf("创建API [%s] 成功", api.Name)
		} else {
			log.Printf("API [%s] 已存在,跳过创建", api.Name)
		}
	}

	log.Println("[API Mock结束]")
}
