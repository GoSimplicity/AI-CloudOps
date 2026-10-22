package utils

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"testing"
)

func TestProfileCannotChangePrivileges(t *testing.T) {
	for _, accountType := range []int8{1, 2} {
		user := &model.User{AccountType: 1, Enable: 0}
		ApplyProfileUpdates(user, &model.UpdateProfileReq{AccountType: accountType, Enable: 1, RealName: "Updated"})
		if user.AccountType != 1 || user.Enable != 0 || user.RealName != "Updated" {
			t.Fatal("profile update changed security fields or failed to update profile")
		}
	}
}

func TestCreatePreservesNotificationEmail(t *testing.T) {
	for _, email := range []string{"", "ops@example.com"} {
		user := BuildUserForCreate(&model.UserSignUpReq{Email: email}, "hashed")
		if user.Email != email {
			t.Fatal("registration lost email")
		}
	}
}
