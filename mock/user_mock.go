package mock

import (
	"github.com/GoSimplicity/CloudOps/internal/model"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"log"
)

const (
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
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
	if err != nil {
		log.Println(err)
		return
	}

	// 创建用户
	user := &model.User{
		Username:    "admin",
		Password:    string(hashedPassword),
		RealName:    "管理员账号",
		AccountType: AdminAccountType,
	}

	// 写入数据库并处理错误
	if err := u.db.Model(&model.User{}).Create(user).Error; err != nil {
		log.Println(err)
		return
	}

	log.Println("message: Admin user created successfully")
	log.Println("[用户模块Mock结束]")

}
