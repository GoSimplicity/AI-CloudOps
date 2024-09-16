package service

import (
	"context"
	"github.com/GoSimplicity/CloudOps/internal/auth/dao/auth"
	"sort"

	"github.com/GoSimplicity/CloudOps/internal/model"
	userDao "github.com/GoSimplicity/CloudOps/internal/user/dao"
	"go.uber.org/zap"
)

type AuthService interface {
	GetMenuList(ctx context.Context, uid int) ([]*model.Menu, error)
	GetAllMenuList(ctx context.Context) ([]*model.Menu, error)
	UpdateMenu(ctx context.Context, menu model.Menu) error
	CreateMenu(ctx context.Context, menu model.Menu) error
	DeleteMenu(ctx context.Context, id string) error

	GetAllRoleList(ctx context.Context) ([]*model.Role, error)
	CreateRole(ctx context.Context, roles model.Role) error
	UpdateRole(ctx context.Context, roles model.Role) error
	SetRoleStatus(ctx context.Context, id int, status string) error
	DeleteRole(ctx context.Context, id string) error

	GetApiList(ctx context.Context, uid int) ([]*model.Api, error)
	GetApiListAll(ctx context.Context) ([]*model.Api, error)
	DeleteApi(ctx context.Context, apiID string) error
	CreateApi(ctx context.Context, api *model.Api) error
	UpdateApi(ctx context.Context, api *model.Api) error
}

type authService struct {
	dao     auth.AuthDAO
	userDao userDao.UserDAO
	l       *zap.Logger
}

func NewAuthService(dao auth.AuthDAO, l *zap.Logger, userDao userDao.UserDAO) AuthService {
	return &authService{
		dao:     dao,
		l:       l,
		userDao: userDao,
	}
}

// GetMenuList 根据用户ID获取菜单列表，支持按角色过滤菜单
func (a *authService) GetMenuList(ctx context.Context, uid int) ([]*model.Menu, error) {
	// 获取用户信息
	user, err := a.userDao.GetUserByID(ctx, uid)
	if err != nil {
		a.l.Error("GetUserByID failed", zap.Error(err))
		return nil, err
	}

	// 父菜单映射和子菜单唯一性检查
	fatherMenuMap := make(map[int]*model.Menu)
	uniqueChildMap := make(map[int]struct{})

	// 遍历用户角色，处理菜单
	for _, role := range user.Roles {
		if role.Status == "0" {
			// 跳过禁用的角色
			continue
		}

		// 处理角色的菜单
		for _, menu := range role.Menus {
			if menu.Status == "0" && role.RoleValue != "super" {
				// 非超级管理员跳过禁用菜单
				continue
			}

			// 设置菜单元数据
			a.setMenuMeta(menu)

			// 父菜单处理
			if menu.Pid == 0 {
				fatherMenuMap[menu.ID] = menu
			} else {
				// 处理子菜单并附加到父菜单
				if err := a.attachToFatherMenu(ctx, menu, fatherMenuMap, uniqueChildMap); err != nil {
					a.l.Error("attachToFatherMenu failed", zap.Error(err))
					continue
				}
			}
		}
	}

	// 对菜单进行排序并返回
	return a.sortedMenuList(fatherMenuMap), nil
}

// GetAllMenuList 获取所有菜单列表
func (a *authService) GetAllMenuList(ctx context.Context) ([]*model.Menu, error) {
	// 从数据库获取所有菜单
	menus, err := a.dao.GetAllMenus(ctx)
	if err != nil {
		a.l.Error("GetAllMenus failed", zap.Error(err))
		return nil, err
	}

	// 设置每个菜单的元数据
	for _, menu := range menus {
		a.setMenuMeta(menu)
	}

	return menus, nil
}

// UpdateMenu 更新菜单信息
func (a *authService) UpdateMenu(ctx context.Context, menu model.Menu) error {
	return a.dao.UpdateMenu(ctx, &menu)
}

// CreateMenu 创建新菜单
func (a *authService) CreateMenu(ctx context.Context, menu model.Menu) error {
	return a.dao.CreateMenu(ctx, &menu)
}

// DeleteMenu 删除菜单
func (a *authService) DeleteMenu(ctx context.Context, id string) error {
	return a.dao.DeleteMenu(ctx, id)
}

// GetAllRoleList 获取所有角色列表
func (a *authService) GetAllRoleList(ctx context.Context) ([]*model.Role, error) {
	return a.dao.GetAllRoles(ctx)
}

// CreateRole 创建新角色
func (a *authService) CreateRole(ctx context.Context, role model.Role) error {
	// 通过菜单ID列表获取菜单对象
	menus, err := a.getMenusByIDs(ctx, role.MenuIds)
	if err != nil {
		return err
	}

	// 将菜单分配给角色
	role.Menus = menus

	return a.dao.CreateRole(ctx, &role)
}

// UpdateRole 更新角色信息
func (a *authService) UpdateRole(ctx context.Context, role model.Role) error {
	// 通过菜单ID列表获取菜单对象
	menus, err := a.getMenusByIDs(ctx, role.MenuIds)
	if err != nil {
		return err
	}

	// 更新角色菜单
	role.Menus = menus

	return a.dao.UpdateRole(ctx, &role)
}

func (a *authService) SetRoleStatus(ctx context.Context, roleID int, status string) error {
	return a.dao.UpdateRoleStatus(ctx, roleID, status)
}

func (a *authService) DeleteRole(ctx context.Context, id string) error {
	return a.dao.DeleteRole(ctx, id)
}

func (a *authService) GetApiList(ctx context.Context, uid int) ([]*model.Api, error) {
	user, err := a.userDao.GetUserByID(ctx, uid)
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
	return a.dao.GetAllApis(ctx)
}

func (a *authService) DeleteApi(ctx context.Context, apiID string) error {
	return a.dao.DeleteApi(ctx, apiID)
}

func (a *authService) CreateApi(ctx context.Context, api *model.Api) error {
	return a.dao.CreateApi(ctx, api)
}

func (a *authService) UpdateApi(ctx context.Context, api *model.Api) error {
	return a.dao.UpdateApi(ctx, api)
}

// attachToFatherMenu 将子菜单附加到父菜单
func (a *authService) attachToFatherMenu(ctx context.Context, menu *model.Menu, fatherMenuMap map[int]*model.Menu, uniqueChildMap map[int]struct{}) error {
	// 获取父菜单
	fatherMenu, err := a.dao.GetMenuByID(ctx, int(menu.Pid))
	if err != nil {
		return err
	}

	// 设置父菜单的元数据
	a.setMenuMeta(fatherMenu)

	// 确保子菜单唯一性
	if _, exists := uniqueChildMap[menu.ID]; !exists {
		uniqueChildMap[menu.ID] = struct{}{}

		// 添加子菜单到父菜单
		if existingFather, ok := fatherMenuMap[fatherMenu.ID]; ok {
			existingFather.Children = append(existingFather.Children, menu)
		} else {
			fatherMenu.Children = append(fatherMenu.Children, menu)
			fatherMenuMap[fatherMenu.ID] = fatherMenu
		}
	}

	return nil
}

// sortedMenuList 根据ID对菜单进行排序并返回列表
func (a *authService) sortedMenuList(fatherMenuMap map[int]*model.Menu) []*model.Menu {
	finalMenus := make([]*model.Menu, 0, len(fatherMenuMap))
	finalMenuIds := make([]int, 0, len(fatherMenuMap))

	for id := range fatherMenuMap {
		finalMenuIds = append(finalMenuIds, int(id))
	}

	sort.Ints(finalMenuIds)

	for _, id := range finalMenuIds {
		finalMenus = append(finalMenus, fatherMenuMap[int(id)])
	}

	return finalMenus
}

// setMenuMeta 设置菜单的元数据信息
func (a *authService) setMenuMeta(menu *model.Menu) {
	menu.Meta = &model.MenuMeta{
		Icon:            menu.Icon,
		Title:           menu.Title,
		ShowMenu:        menu.Show,
		HideMenu:        !menu.Show,
		IgnoreKeepAlive: true,
	}

	menu.Key = menu.ID
	menu.Value = menu.ID
}

// getMenusByIDs 根据菜单ID列表获取对应的菜单对象
func (a *authService) getMenusByIDs(ctx context.Context, menuIds []int) ([]*model.Menu, error) {
	menus := make([]*model.Menu, 0)

	for _, menuId := range menuIds {
		// 根据ID获取菜单信息
		menu, err := a.dao.GetMenuByID(ctx, int(menuId))
		if err != nil {
			a.l.Error("GetMenuByID failed", zap.Error(err))
			return nil, err
		}

		menus = append(menus, menu)
	}

	return menus, nil
}
