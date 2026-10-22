package service

import (
	"context"
	"errors"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	userDao "github.com/GoSimplicity/AI-CloudOps/internal/system/dao"
	"go.uber.org/zap"
)

type recipientDAO struct {
	userDao.UserDAO
	fail bool
}

func (d recipientDAO) GetByRoleIDs(_ context.Context, _ []int) ([]*model.User, error) {
	if d.fail {
		return nil, errors.New("database unavailable")
	}
	return []*model.User{{Model: model.Model{ID: 7}, RealName: "Operator"}}, nil
}
func TestNotificationRecipients(t *testing.T) {
	for _, tt := range []struct {
		name          string
		types         []string
		roles         []string
		fail, invalid bool
		want          int
	}{
		{"role and creator deduplicated", []string{model.RecipientTypeRole, model.RecipientTypeCreator}, []string{"1"}, false, false, 1},
		{"invalid role", []string{model.RecipientTypeRole}, []string{"bad"}, false, true, 0},
		{"database failure", []string{model.RecipientTypeRole}, []string{"1"}, true, true, 0},
		{"unsupported department", []string{model.RecipientTypeDept}, nil, false, true, 0},
		{"empty", nil, nil, false, true, 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := &workorderNotificationService{userDAO: recipientDAO{fail: tt.fail}, logger: zap.NewNop()}
			recipients, err := s.getRecipients(context.Background(), &model.WorkorderNotification{RecipientTypes: tt.types, RecipientRoles: tt.roles}, &model.WorkorderInstance{OperatorID: 7})
			if (err != nil) != tt.invalid || len(recipients) != tt.want {
				t.Fatalf("recipients=%v err=%v", recipients, err)
			}
		})
	}
}
