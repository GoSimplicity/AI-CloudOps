package service

import (
	"context"
	"github.com/GoSimplicity/CloudOps/internal/auth/dao"
	"github.com/GoSimplicity/CloudOps/internal/model"
	userDao "github.com/GoSimplicity/CloudOps/internal/user/dao"
	"go.uber.org/zap"
	"sort"
)

type AuthService interface {
	GetMenuList(ctx context.Context, uid int) ([]*model.Menu, error)
	GetAllMenuList(ctx context.Context) ([]*model.Menu, error)
	UpdateMenu(ctx context.Context, menu model.Menu) error
	CreateMenu(ctx context.Context, menu model.Menu) error
	DeleteMenu(ctx context.Context, id int) error

	GetAllRoleList(ctx context.Context) ([]*model.Role, error)
	CreateRole(ctx context.Context, roles model.Role) error
	UpdateRole(ctx context.Context, roles model.Role) error
	SetRoleStatus(ctx context.Context, id int, status string) error
	DeleteRole(ctx context.Context, id int) error

	GetApiList(ctx context.Context, uid int) ([]*model.Api, error)
	GetApiListAll(ctx context.Context) ([]*model.Api, error)
	DeleteApi(ctx context.Context, apiID int) error
	CreateApi(ctx context.Context, api *model.Api) error
	UpdateApi(ctx context.Context, api *model.Api) error
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

// GetMenuList 根据用户ID获取菜单列表
func (a *authService) GetMenuList(ctx context.Context, uid int) ([]*model.Menu, error) {
	// 获取用户信息
	user, err := a.userDao.GetUserByID(ctx, uint(uid))
	if err != nil {
		a.l.Error("GetUserByID failed", zap.Error(err))
		return nil, err
	}

	// 用于存储父菜单和唯一子菜单
	fatherMenuMap := make(map[uint]*model.Menu)
	uniqueChildMap := make(map[uint]struct{})

	// 遍历用户的角色
	for _, role := range user.Roles {
		// 跳过禁用的角色
		if role.Status == "0" {
			continue
		}

		// 遍历角色的菜单
		for _, menu := range role.Menus {
			// 非超级管理员跳过禁用的菜单
			if menu.Status == "0" && role.RoleValue != "super" {
				continue
			}

			// 设置菜单元数据
			menu.Meta = &model.MenuMeta{
				Icon:            menu.Icon,
				Title:           menu.Title,
				ShowMenu:        menu.Show,
				HideMenu:        !menu.Show,
				IgnoreKeepAlive: true,
			}
			menu.Key = menu.ID
			menu.Value = menu.ID

			// 父菜单处理
			if menu.Pid == 0 {
				fatherMenuMap[menu.ID] = menu
				continue
			}

			// 获取父菜单信息
			fatherMenu, err := a.dao.GetMenuByID(ctx, uint(menu.Pid))
			if err != nil {
				a.l.Error("GetMenuByID failed", zap.Error(err))
				continue
			}

			// 设置父菜单的元数据
			fatherMenu.Meta = &model.MenuMeta{
				Icon:     fatherMenu.Icon,
				Title:    fatherMenu.Title,
				ShowMenu: fatherMenu.Show,
				HideMenu: !fatherMenu.Show,
			}
			fatherMenu.Key = fatherMenu.ID
			fatherMenu.Value = fatherMenu.ID

			// 如果子菜单已处理过，则跳过
			if _, ok := uniqueChildMap[menu.ID]; ok {
				continue
			}
			uniqueChildMap[menu.ID] = struct{}{}

			// 父菜单添加子菜单
			if existingFather, ok := fatherMenuMap[fatherMenu.ID]; !ok {
				fatherMenu.Children = append(fatherMenu.Children, menu)
				fatherMenuMap[fatherMenu.ID] = fatherMenu
			} else {
				existingFather.Children = append(existingFather.Children, menu)
			}
		}
	}

	// 构建最终的菜单列表并进行排序
	finalMenus := make([]*model.Menu, 0, len(fatherMenuMap))
	finalMenuIds := make([]int, 0, len(fatherMenuMap))

	for id := range fatherMenuMap {
		finalMenuIds = append(finalMenuIds, int(id))
	}
	sort.Ints(finalMenuIds)

	for _, id := range finalMenuIds {
		finalMenus = append(finalMenus, fatherMenuMap[uint(id)])
	}

	return finalMenus, nil
}

// GetAllMenuList 获取所有菜单列表
func (a *authService) GetAllMenuList(ctx context.Context) ([]*model.Menu, error) {
	menus, err := a.dao.GetAllMenus(ctx)
	if err != nil {
		a.l.Error("GetAllMenus failed", zap.Error(err))
		return nil, err
	}

	for _, menu := range menus {
		menu.Meta = &model.MenuMeta{
			Icon:     menu.Icon,
			Title:    menu.Title,
			ShowMenu: menu.Show,
		}
		menu.Key = menu.ID
		menu.Value = menu.ID
	}

	return menus, nil
}

// UpdateMenu 更新菜单信息
func (a *authService) UpdateMenu(ctx context.Context, menu model.Menu) error {
	existingMenu, err := a.dao.GetMenuByID(ctx, menu.ID)
	if err != nil {
		a.l.Error("GetMenuByID failed", zap.Error(err))
		return err
	}

	existingMenu.Name = menu.Name
	existingMenu.Title = menu.Title
	existingMenu.Show = menu.Show
	existingMenu.Component = menu.Component
	existingMenu.Path = menu.Path

	return a.dao.UpdateMenu(ctx, existingMenu)
}

// CreateMenu 创建新菜单
func (a *authService) CreateMenu(ctx context.Context, menu model.Menu) error {
	return a.dao.CreateMenu(ctx, &menu)
}

// DeleteMenu 删除菜单
func (a *authService) DeleteMenu(ctx context.Context, id int) error {
	return a.dao.DeleteMenu(ctx, uint(id))
}

// GetAllRoleList 获取所有角色列表
func (a *authService) GetAllRoleList(ctx context.Context) ([]*model.Role, error) {
	roles, err := a.dao.GetAllRoles(ctx)
	if err != nil {
		a.l.Error("GetAllRoles failed", zap.Error(err))
		return nil, err
	}

	return roles, nil
}

// CreateRole 创建新角色
func (a *authService) CreateRole(ctx context.Context, role model.Role) error {
	menus := make([]*model.Menu, 0)
	for _, menuId := range role.MenuIds {
		menu, err := a.dao.GetMenuByID(ctx, uint(menuId))
		if err != nil {
			a.l.Error("GetMenuByID failed", zap.Error(err))
			return err
		}
		menus = append(menus, menu)
	}

	role.Menus = menus

	return a.dao.CreateRole(ctx, &role)
}

// UpdateRole 更新角色信息
func (a *authService) UpdateRole(ctx context.Context, role model.Role) error {
	_, err := a.dao.GetRoleByRoleID(ctx, role.ID)
	if err != nil {
		a.l.Error("GetRoleByRoleID failed", zap.Error(err))
		return err
	}

	menus := make([]*model.Menu, 0)
	for _, menuId := range role.MenuIds {
		menu, err := a.dao.GetMenuByID(ctx, uint(menuId))
		if err != nil {
			a.l.Error("GetMenuByID failed", zap.Error(err))
			return err
		}
		menus = append(menus, menu)
	}

	role.Menus = menus

	return a.dao.UpdateRole(ctx, &role)
}
func (a *authService) SetRoleStatus(ctx context.Context, roleID int, status string) error {
	role, err := a.dao.GetRoleByRoleID(ctx, uint(roleID))
	if err != nil {
		a.l.Error("GetRoleByRoleID failed", zap.Error(err))
		return err
	}

	// 更新角色状态
	role.Status = status

	err = a.dao.UpdateRole(ctx, role)
	if err != nil {
		a.l.Error("UpdateRole failed", zap.Error(err))
		return err
	}

	return nil
}

func (a *authService) DeleteRole(ctx context.Context, id int) error {
	err := a.dao.DeleteRole(ctx, uint(id))
	if err != nil {
		a.l.Error("DeleteRole failed", zap.Error(err))
		return err
	}

	return nil
}

func (a *authService) GetApiList(ctx context.Context, uid int) ([]*model.Api, error) {
	user, err := a.userDao.GetUserByID(ctx, uint(uid))
	if err != nil {
		a.l.Error("GetUserByID failed", zap.Error(err))
		return nil, err
	}

	apis := make([]*model.Api, 0)
	for _, role := range user.Roles {
		roleApis, err := a.dao.GetApisByRoleID(ctx, role.ID)
		if err != nil {
			a.l.Error("GetApisByRoleID failed", zap.Error(err))
			return nil, err
		}
		apis = append(apis, roleApis...)
	}

	return apis, nil
}

func (a *authService) GetApiListAll(ctx context.Context) ([]*model.Api, error) {
	apis, err := a.dao.GetAllApis(ctx)
	if err != nil {
		a.l.Error("GetAllApis failed", zap.Error(err))
		return nil, err
	}

	return apis, nil
}

func (a *authService) DeleteApi(ctx context.Context, apiID int) error {
	err := a.dao.DeleteApi(ctx, uint(apiID))
	if err != nil {
		a.l.Error("DeleteApi failed", zap.Error(err))
		return err
	}

	return nil
}

func (a *authService) CreateApi(ctx context.Context, api *model.Api) error {
	err := a.dao.CreateApi(ctx, api)
	if err != nil {
		a.l.Error("CreateApi failed", zap.Error(err))
		return err
	}

	return nil
}

func (a *authService) UpdateApi(ctx context.Context, api *model.Api) error {
	_, err := a.dao.GetApiByID(ctx, api.ID)
	if err != nil {
		a.l.Error("GetApiByID failed", zap.Error(err))
		return err
	}

	err = a.dao.UpdateApi(ctx, api)
	if err != nil {
		a.l.Error("UpdateApi failed", zap.Error(err))
		return err
	}

	return nil
}
