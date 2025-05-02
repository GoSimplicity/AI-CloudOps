import type {
  ComponentRecordType,
  GenerateMenuAndRoutesOptions,
} from '@vben/types';

import { generateAccessible } from '@vben/access';
import { preferences } from '@vben/preferences';

import { message } from 'ant-design-vue';

// import { listMenusApi } from '#/api/core/system';
import { getUserInfoApi } from '#/api/core/user';
import { BasicLayout, IFrameView } from '#/layouts';
import { $t } from '#/locales';

const forbiddenComponent = () => import('#/views/_core/fallback/forbidden.vue');

async function generateAccess(options: GenerateMenuAndRoutesOptions) {
  const pageMap: ComponentRecordType = import.meta.glob('../views/**/*.vue');

  const layoutMap: ComponentRecordType = {
    BasicLayout,
    IFrameView,
  };

  // 生成权限路由
  return await generateAccessible(preferences.app.accessMode, {
    ...options,
    // 异步获取菜单列表
    fetchMenuListAsync: async () => {
      // 显示加载提示
      message.loading({
        content: `${$t('common.loadingMenu')}...`,
        duration: 1.5,
      });
      // // 调用接口获取菜单数据
      // return await listMenusApi({
      //   page_number: 1,
      //   page_size: 999
      // });

      const res = await getUserInfoApi();

      return res.menus;
    },
    // 可以指定没有权限跳转403页面
    forbiddenComponent,
    // 如果 route.meta.menuVisibleWithForbidden = true
    layoutMap,
    pageMap,
  });
}

export { generateAccess };
