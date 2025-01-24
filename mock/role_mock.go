package mock

import (
	"log"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
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

func (r *RoleMock) InitRole() error {
	// 检查是否已经初始化过角色
	var count int64
	r.db.Model(&model.Role{}).Count(&count)
	if count > 0 {
		log.Println("[角色已经初始化过,跳过Mock]")
		return nil
	}

	log.Println("[角色模块Mock开始]")

	// 创建普通用户角色
	role := model.Role{
		Name:      "user",
		Desc:      "普通用户",
		RoleType:  2,
		IsDefault: 1,
	}

	if err := r.db.Create(&role).Error; err != nil {
		log.Printf("创建普通用户角色失败: %v", err)
		return err
	}

	// 使用casbin添加权限策略
	if ok, err := r.ce.AddPolicy("user", "/*", "GET"); err == nil && ok {
		log.Printf("成功添加权限策略: 角色=user, 路径=/*, 方法=GET")
	} else if err != nil {
		log.Printf("添加权限策略失败: %v", err)
		return err
	} else {
		log.Printf("权限策略已存在: 角色=user, 路径=/*, 方法=GET")
	}

	if err := r.ce.SavePolicy(); err != nil {
		log.Printf("保存策略失败: %v", err)
		return err
	}

	log.Printf("创建普通用户角色 [%s] 成功", role.Name)
	log.Println("[角色模块Mock结束]")

	return nil
}
