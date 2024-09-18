package mock

import (
	"github.com/GoSimplicity/CloudOps/internal/model"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"log"
)

const (
	AdminUsername    = "admin"
	AdminPassword    = "admin"
	AdminAccountType = 2
)

type UserMock struct {
	db *gorm.DB
}

func NewUserMock(db *gorm.DB) *UserMock {
	return &UserMock{
		db: db,
	}
}

func (u *UserMock) CreateUserAdmin() {
	log.Println("[用户模块Mock开始]")

	// 生成加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(AdminPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("生成密码失败: %v\n", err)
		log.Println("[用户模块Mock结束]")
		return
	}

	// 创建管理员用户实例
	adminUser := model.User{
		Username:    AdminUsername,
		Password:    string(hashedPassword),
		RealName:    "管理员账号",
		AccountType: AdminAccountType, // 确保 AdminAccountType 已定义
	}

	// 使用 FirstOrCreate 方法查找或创建管理员用户
	result := u.db.Where("username = ?", adminUser.Username).FirstOrCreate(&adminUser)

	// 检查操作是否成功
	if result.Error != nil {
		log.Printf("创建或获取管理员用户失败: %v\n", result.Error)
		log.Println("[用户模块Mock结束]")
		return
	}

	// 根据 RowsAffected 判断用户是否已存在或新创建
	if result.RowsAffected == 1 {
		log.Println("管理员用户创建成功")
	} else {
		log.Println("管理员用户已存在，跳过创建")
	}

	log.Println("[用户模块Mock结束]")
}
