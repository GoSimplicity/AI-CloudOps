package service

import (
	"context"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/system/dao"
	"go.uber.org/zap"
)

type accountUpdateDAO struct {
	dao.UserDAO
	user *model.User
}

func (d *accountUpdateDAO) GetByID(context.Context, int) (*model.User, error) { return d.user, nil }
func (d *accountUpdateDAO) Update(_ context.Context, u *model.User) error     { d.user = u; return nil }
func TestAccountChangesRequireAdministratorAuthority(t *testing.T) {
	for _, allow := range []bool{false, true} {
		d := &accountUpdateDAO{user: &model.User{AccountType: 1, Enable: 1}}
		s := &userService{dao: d, logger: zap.NewNop()}
		if err := s.UpdateProfile(context.Background(), &model.UpdateProfileReq{AccountType: 2, Enable: 2, RealName: "updated"}, allow); err != nil {
			t.Fatal(err)
		}
		expected := int8(1)
		if allow {
			expected = 2
		}
		if d.user.AccountType != expected || d.user.Enable != expected || d.user.RealName != "updated" {
			t.Fatalf("allow=%v account=%d enable=%d", allow, d.user.AccountType, d.user.Enable)
		}
	}
}
