package service

import (
	"context"
	"errors"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/constants"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/system/dao"
	userutils "github.com/GoSimplicity/AI-CloudOps/internal/system/utils"
	"go.uber.org/zap"
)

type changePasswordDAO struct {
	dao.UserDAO
	users     map[int]*model.User
	changedID int
	changed   string
}

func (d *changePasswordDAO) GetByID(_ context.Context, id int) (*model.User, error) {
	if u, ok := d.users[id]; ok {
		return u, nil
	}
	return nil, errors.New("not found")
}
func (d *changePasswordDAO) ChangePassword(_ context.Context, uid int, h string) error {
	d.changedID, d.changed = uid, h
	return nil
}

func TestChangePasswordVerifiesOldPasswordOfTokenUser(t *testing.T) {
	selfOld, _ := userutils.HashPassword("self-old-x")
	otherOld, _ := userutils.HashPassword("other-old-x")
	d := &changePasswordDAO{users: map[int]*model.User{
		7:  {Model: model.Model{ID: 7}, Password: selfOld},
		99: {Model: model.Model{ID: 99}, Password: otherOld},
	}}
	s := NewUserService(d, nil, zap.NewNop())
	err := s.ChangePassword(context.Background(), &model.ChangePasswordReq{UserID: 7, Password: "wrong-x", NewPassword: "new-pass-x", ConfirmPassword: "new-pass-x"})
	if !errors.Is(err, constants.ErrorPasswordIncorrect) || d.changedID != 0 {
		t.Fatalf("wrong old password accepted: %v", err)
	}
	err = s.ChangePassword(context.Background(), &model.ChangePasswordReq{UserID: 7, Password: "self-old-x", NewPassword: "new-pass-x", ConfirmPassword: "new-pass-x"})
	if err != nil || d.changedID != 7 || userutils.ComparePassword(d.changed, "new-pass-x") != nil {
		t.Fatalf("correct old password failed: %v", err)
	}
	t.Logf("wrong old pwd rejected; correct old pwd -> dao.ChangePassword(uid=%d, bcrypt hash)", d.changedID)
}

func TestChangePasswordRejectsNonPositiveUserID(t *testing.T) {
	d := &changePasswordDAO{users: map[int]*model.User{}}
	s := NewUserService(d, nil, zap.NewNop())
	for _, uid := range []int{0, -1} {
		err := s.ChangePassword(context.Background(), &model.ChangePasswordReq{UserID: uid, Password: "old-x", NewPassword: "new-pass-x", ConfirmPassword: "new-pass-x"})
		if err == nil || d.changedID != 0 {
			t.Fatalf("uid=%d should be rejected, err=%v", uid, err)
		}
	}
}
