package service

import (
	"context"
	"github.com/GoSimplicity/CloudOps/internal/auth/dao"
	"github.com/GoSimplicity/CloudOps/internal/auth/dto"
	"github.com/GoSimplicity/CloudOps/internal/model"
	userDao "github.com/GoSimplicity/CloudOps/internal/user/dao"
	"go.uber.org/zap"
	"sort"
)

type AuthService interface {
	GetMenuList(ctx context.Context, uid int) ([]dto.MenuDTO, error)
	GetAllMenuList(ctx context.Context)
	UpdateMenu(ctx context.Context)
	CreateMenu(ctx context.Context)
	DeleteMenu(ctx context.Context)

	GetAllRoleList(ctx context.Context)
	CreateRole(ctx context.Context)
	UpdateRole(ctx context.Context)
	SetRoleStatus(ctx context.Context)
	DeleteRole(ctx context.Context)

	GetApiList(ctx context.Context)
	GetApiListAll(ctx context.Context)
	DeleteApi(ctx context.Context)
	CreateApi(ctx context.Context)
	UpdateApi(ctx context.Context)
}

type authService struct {
	dao     dao.AuthDAO
	userDao userDao.UserDAO
	l       *zap.Logger
}

func NewAuthService(dao dao.AuthDAO, l *zap.Logger, userDao userDao.UserDAO) AuthService {
	return &authService{
		dao:     dao,
		l:       l,
		userDao: userDao,
	}
}

func (a *authService) GetMenuList(ctx context.Context, uid int) ([]dto.MenuDTO, error) {
	user, err := a.userDao.GetUserByID(ctx, uid)
	if err != nil {
		a.l.Error("GetUserByID failed", zap.Error(err))
		return nil, err
	}

	fatherMenuMap := make(map[uint]*model.Menu)
	uniqueChildMap := make(map[uint]*model.Menu)

	for _, role := range user.Roles {
		if role.Status == "0" {
			continue
		}

		for _, menu := range role.Menus {
			if menu.Status == "0" && role.RoleValue != "super" {
				continue
			}

			menu.Meta = &model.MenuMeta{
				Icon:            menu.Icon,
				Title:           menu.Title,
				ShowMenu:        menu.Show,
				HideMenu:        !menu.Show,
				IgnoreKeepAlive: true,
			}

			menu.Key = menu.ID
			menu.Value = menu.ID

			if menu.Pid == 0 {
				fatherMenuMap[menu.ID] = menu
				continue
			}

			fatherMenu, err := a.dao.GetMenuByID(ctx, menu.Pid)
			if err != nil {
				continue
			}

			fatherMenu.Meta = &model.MenuMeta{
				Icon:     fatherMenu.Icon,
				Title:    fatherMenu.Title,
				ShowMenu: fatherMenu.Show,
				HideMenu: !fatherMenu.Show,
			}

			fatherMenu.Key = fatherMenu.ID
			fatherMenu.Value = fatherMenu.ID

			if _, ok := uniqueChildMap[menu.ID]; ok {
				continue
			}
			uniqueChildMap[menu.ID] = menu

			if load, ok := fatherMenuMap[fatherMenu.ID]; !ok {
				fatherMenu.Children = append(fatherMenu.Children, menu)
				fatherMenuMap[fatherMenu.ID] = fatherMenu
			} else {
				load.Children = append(load.Children, menu)
			}
		}
	}

	// 构建最终的菜单列表
	finalMenus := make([]*model.Menu, 0)
	finalMenuIds := make([]int, 0, len(fatherMenuMap))
	// 将父菜单ID排序
	for id := range fatherMenuMap {
		finalMenuIds = append(finalMenuIds, int(id))
	}
	sort.Ints(finalMenuIds)

	// 根据排序后的ID顺序加入菜单列表
	for _, id := range finalMenuIds {
		finalMenus = append(finalMenus, fatherMenuMap[uint(id)])
	}

	return nil, nil
}

func (a *authService) GetAllMenuList(ctx context.Context) {
	//TODO implement me
	panic("implement me")
}

func (a *authService) UpdateMenu(ctx context.Context) {
	//TODO implement me
	panic("implement me")
}

func (a *authService) CreateMenu(ctx context.Context) {
	//TODO implement me
	panic("implement me")
}

func (a *authService) DeleteMenu(ctx context.Context) {
	//TODO implement me
	panic("implement me")
}

func (a *authService) GetAllRoleList(ctx context.Context) {
	//TODO implement me
	panic("implement me")
}

func (a *authService) CreateRole(ctx context.Context) {
	//TODO implement me
	panic("implement me")
}

func (a *authService) UpdateRole(ctx context.Context) {
	//TODO implement me
	panic("implement me")
}

func (a *authService) SetRoleStatus(ctx context.Context) {
	//TODO implement me
	panic("implement me")
}

func (a *authService) DeleteRole(ctx context.Context) {
	//TODO implement me
	panic("implement me")
}

func (a *authService) GetApiList(ctx context.Context) {
	//TODO implement me
	panic("implement me")
}

func (a *authService) GetApiListAll(ctx context.Context) {
	//TODO implement me
	panic("implement me")
}

func (a *authService) DeleteApi(ctx context.Context) {
	//TODO implement me
	panic("implement me")
}

func (a *authService) CreateApi(ctx context.Context) {
	//TODO implement me
	panic("implement me")
}

func (a *authService) UpdateApi(ctx context.Context) {
	//TODO implement me
	panic("implement me")
}
