<template>
  <div>
    <!-- 查询和操作 -->
    <div class="custom-toolbar">
      <!-- 查询功能 -->
      <div class="search-filters">
        <!-- 搜索输入框 -->
        <a-input
          v-model:value="searchText"
          placeholder="请输入用户名或昵称"
          style="width: 200px; margin-right: 16px;"
        />
        <!-- 搜索按钮 -->
        <a-button type="primary" @click="handleSearch">搜索</a-button>
      </div>
      <!-- 操作按钮 -->
      <div class="action-buttons">
        <a-button type="primary" @click="handleAdd">新增账号</a-button>
      </div>
    </div>
    <!-- 用户列表表格 -->
    <a-table :columns="columns" :data-source="userList" :loading="loading" row-key="id">
      <!-- 操作列 -->
      <template #action="{ record }">
        <a-space>
          <a-tooltip title="编辑用户信息">
            <a-button type="link" @click="handleEdit(record)">
              <template #icon><Icon icon="clarity:note-edit-line" style="font-size: 22px" /></template>
            </a-button>
          </a-tooltip>
          <a-tooltip title="修改用户密码">
            <a-button type="link" @click="handleChangePassword(record)">
              <template #icon><Icon icon="mdi:key-outline" style="font-size: 22px" /></template>
            </a-button>
          </a-tooltip>
          <a-tooltip title="分配用户权限">
            <a-button type="link" @click="handlePermissions(record)">
              <template #icon><Icon icon="clarity:key-line" style="font-size: 22px" /></template>
            </a-button>
          </a-tooltip>
          <a-popconfirm
            title="确定要注销这个用户吗?"
            ok-text="确定"
            cancel-text="取消"
            placement="left"
            @confirm="handleWriteOff(record)"
          >
            <a-tooltip title="注销用户">
              <a-button type="link" danger>
                <template #icon><Icon icon="ant-design:delete-outlined" style="font-size: 22px" /></template>
              </a-button>
            </a-tooltip>
          </a-popconfirm>
        </a-space>
      </template>
      <!-- 角色列 -->
      <template #roles="{ record }">
        <a-space wrap>
          <template v-if="record.roles && record.roles.length > 0">
            <a-tag v-for="role in record.roles" :key="role.id" :color="getRandomColor()">
              {{ role.name }}
            </a-tag>
          </template>
          <a-tag v-else color="default">暂无角色</a-tag>
        </a-space>
      </template>
    </a-table>

    <!-- 新增/编辑用户对话框 -->
    <a-modal
      v-model:visible="isModalVisible"
      :title="modalTitle"
      @ok="handleModalSubmit"
      @cancel="handleModalCancel"
      :okText="'保存'"
      :cancelText="'取消'"
    >
      <a-form :model="formData" layout="vertical">
        <a-form-item label="用户名" required v-if="modalTitle === '新增用户'">
          <a-input v-model:value="formData.username" placeholder="请输入用户名" />
        </a-form-item>
        <a-form-item label="密码" required v-if="modalTitle === '新增用户'">
          <a-input-password v-model:value="formData.password" placeholder="请输入密码" />
        </a-form-item>
        <a-form-item label="确认密码" required v-if="modalTitle === '新增用户'">
          <a-input-password v-model:value="formData.confirmPassword" placeholder="请再次输入密码" />
        </a-form-item>
        <a-form-item label="真实姓名" required>
          <a-input v-model:value="formData.real_name" placeholder="请输入真实姓名" />
        </a-form-item>
        <a-form-item label="手机号码">
          <a-input v-model:value="formData.mobile" placeholder="请输入手机号码" />
        </a-form-item>
        <a-form-item label="飞书用户ID">
          <a-input v-model:value="formData.fei_shu_user_id" placeholder="请输入飞书用户ID" />
        </a-form-item>
        <a-form-item label="首页路径">
          <a-input v-model:value="formData.home_path" placeholder="请输入首页路径" />
        </a-form-item>
        <a-form-item label="描述">
          <a-textarea v-model:value="formData.desc" placeholder="请输入描述" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 修改密码对话框 -->
    <a-modal
      v-model:visible="isPasswordModalVisible"
      title="修改密码"
      @ok="handlePasswordSubmit"
      @cancel="handlePasswordCancel"
      :okText="'保存'"
      :cancelText="'取消'"
    >
      <a-form :model="passwordForm" layout="vertical">
        <a-form-item label="原密码" required>
          <a-input-password v-model:value="passwordForm.password" placeholder="请输入原密码" />
        </a-form-item>
        <a-form-item label="新密码" required>
          <a-input-password v-model:value="passwordForm.newPassword" placeholder="请输入新密码" />
        </a-form-item>
        <a-form-item label="确认新密码" required>
          <a-input-password v-model:value="passwordForm.confirmPassword" placeholder="请再次输入新密码" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 权限分配对话框 -->
    <a-modal
      v-model:visible="isPermissionModalVisible"
      title="权限分配"
      @ok="handlePermissionModalSubmit"
      @cancel="handlePermissionModalCancel"
      :okText="'保存'"
      :cancelText="'取消'"
      width="800px"
    >
      <a-tabs v-model:activeKey="activeTabKey">
        <a-tab-pane key="role" tab="角色分配">
          <a-form layout="vertical">
            <a-form-item label="角色">
              <a-select
                v-model:value="selectedRoleIds"
                mode="multiple"
                placeholder="请选择角色"
                style="width: 100%"
                :options="roleOptions"
              />
            </a-form-item>
          </a-form>
        </a-tab-pane>
        <a-tab-pane key="api" tab="接口权限">
          <a-tree
            v-model:checkedKeys="selectedApiIds"
            :tree-data="apiTreeData"
            checkable
            :defaultExpandAll="true"
            :fieldNames="{
              title: 'name',
              key: 'id',
              children: 'children'
            }"
          />
        </a-tab-pane>
      </a-tabs>
    </a-modal>
  </div>
</template>

<script lang="ts" setup>
import { reactive, ref, onMounted } from 'vue';
import { message } from 'ant-design-vue';
import { Icon } from '@iconify/vue';
import { getAllUsers, registerApi, changePassword, deleteUser, updateUserInfo } from '#/api';
import { listRolesApi, listApisApi } from '#/api';
interface SystemApi {
  id: number;
  name: string;
  path: string;
  method: number;
}

// 表格加载状态
const loading = ref(false);

// 搜索文本
const searchText = ref('');

// 用户列表数据
const userList = ref<any[]>([]);

// 权限分配相关
const isPermissionModalVisible = ref(false);
const activeTabKey = ref('role');
const selectedApiIds = ref<number[]>([]);
const selectedRoleIds = ref<number[]>([]);
const roleOptions = ref<{label: string, value: number}[]>([]);
const apiTreeData = ref<any[]>([]);
const currentUserId = ref<number>();

// 表格列配置
const columns = [
  {
    title: '用户ID',
    dataIndex: 'id',
    key: 'id',
  },
  {
    title: '用户名',
    dataIndex: 'username',
    key: 'username',
  },
  {
    title: '真实姓名',
    dataIndex: 'real_name',
    key: 'real_name',
  },
  {
    title: '手机号码',
    dataIndex: 'mobile',
    key: 'mobile',
  },
  {
    title: '飞书用户ID',
    dataIndex: 'fei_shu_user_id',
    key: 'fei_shu_user_id',
  },
  {
    title: '角色',
    dataIndex: 'roles',
    key: 'roles',
    slots: { customRender: 'roles' }
  },
  {
    title: '操作',
    key: 'action',
    slots: { customRender: 'action' },
  },
];

// 模态框相关
const isModalVisible = ref(false);
const modalTitle = ref('新增用户');
const formData = reactive({
  username: '',
  password: '',
  confirmPassword: '',
  real_name: '',
  mobile: '',
  fei_shu_user_id: '',
  home_path: '',
  desc: '',
  userId: 0
});

// 密码修改模态框相关
const isPasswordModalVisible = ref(false);
const passwordForm = reactive({
  username: '',
  password: '',
  newPassword: '',
  confirmPassword: ''
});

// 获取用户列表
const fetchUserList = async () => {
  loading.value = true;
  try {
    const res = await getAllUsers();
    userList.value = res;
  } catch (error: any) {
    message.error(error.message || '获取用户列表失败');
  } finally {
    loading.value = false;
  }
};

// 获取角色列表
const fetchRoleList = async () => {
  try {
    const res = await listRolesApi({
      page_number: 1,
      page_size: 100
    });
    if (res && res.items) {
      roleOptions.value = res.items.map((role: any) => ({
        label: role.name,
        value: role.id
      }));
    } else {
      roleOptions.value = [];
      console.error('获取角色列表返回数据格式不正确:', res);
    }
  } catch (error: any) {
    message.error(error.message || '获取角色列表失败');
    roleOptions.value = [];
  }
};

// 获取所有API并构建树状结构
const fetchApis = async () => {
  try {
    const apiRes = await listApisApi({
      page_number: 1,
      page_size: 1000
    });
    
    // 定义API分类
    const apiCategories: {
      [key: string]: {
        title: string;
        path: string;
        children: Array<{
          id: number;
          name: string;
          path: string;
        }>;
      };
    } = {
      all: { title: '所有权限', path: '/*', children: [] },
      user: { title: '用户权限', path: '/api/user', children: [] },
      menu: { title: '菜单权限', path: '/api/menus', children: [] },
      api: { title: 'API权限', path: '/api/apis', children: [] },
      role: { title: '角色权限', path: '/api/roles', children: [] },
      permission: { title: '策略权限', path: '/api/permissions', children: [] },
      tree: { title: '服务树权限', path: '/api/tree', children: [] },
      monitor: { title: '监控权限', path: '/api/monitor', children: [] },
      k8s: { title: 'K8S权限', path: '/api/k8s', children: [] },
    };

    // 将API按路径分类
    if (apiRes && apiRes.list) {
      apiRes.list.forEach((api: SystemApi) => {
        // 如果api.id是数字类型,跳过处理
        if (typeof api.id === 'number') {
          const apiNode = {
            id: api.id,
            name: `${api.name} [${getMethodText(api.method)}]`,
            path: api.path
          };

          // 遍历所有分类,检查API路径是否匹配分类路径前缀
          Object.values(apiCategories).forEach(category => {
            if (api.path && api.path.startsWith(category.path)) {
              category.children.push(apiNode);
            }
          });
        }
      });

      // 构建最终的树状数据,过滤掉空分类
      apiTreeData.value = Object.values(apiCategories)
        .filter(category => category.children.length > 0)
        .map(category => ({
          id: category.path,
          name: category.title,
          children: category.children.sort((a, b) => a.id - b.id) // 按id排序
        }));
    } else {
      apiTreeData.value = [];
      console.error('获取API列表返回数据格式不正确:', apiRes);
    }

  } catch (error: any) {
    message.error(error.message || '获取权限数据失败');
    apiTreeData.value = [];
  }
};

// 获取HTTP方法文本
const getMethodText = (method: number) => {
  switch (method) {
    case 1: return 'GET';
    case 2: return 'POST';
    case 3: return 'PUT';
    case 4: return 'DELETE';
    default: return '未知';
  }
};

// 处理搜索
const handleSearch = () => {
  fetchUserList();
};

// 处理新增账号
const handleAdd = () => {
  modalTitle.value = '新增用户';
  Object.assign(formData, {
    username: '',
    password: '',
    confirmPassword: '',
    real_name: '',
    mobile: '',
    fei_shu_user_id: '',
    home_path: '',
    desc: '',
    userId: 0
  });
  isModalVisible.value = true;
};

// 处理编辑用户
const handleEdit = (record: any) => {
  modalTitle.value = '编辑用户';
  Object.assign(formData, {
    real_name: record.real_name,
    mobile: record.mobile,
    fei_shu_user_id: record.fei_shu_user_id,
    home_path: record.home_path,
    desc: record.desc,
    userId: record.id
  });
  isModalVisible.value = true;
};

// 处理修改密码
const handleChangePassword = (record: any) => {
  passwordForm.username = record.username;
  isPasswordModalVisible.value = true;
};

// 处理权限分配
const handlePermissions = (record: any) => {
  currentUserId.value = record.id;
  selectedRoleIds.value = Array.isArray(record.roles) ? record.roles.map((role: any) => role.id) : [];
  selectedApiIds.value = Array.isArray(record.apis) ? record.apis.map((api: any) => api.id) : [];
  isPermissionModalVisible.value = true;
};

// 处理权限分配提交
const handlePermissionModalSubmit = async () => {
  try {
    if (!currentUserId.value) {
      message.error('用户ID不能为空');
      return;
    }

    // 这里需要实现权限分配的逻辑
    message.success('权限设置成功');
    isPermissionModalVisible.value = false;
    fetchUserList();
  } catch (error: any) {
    message.error(error.message || '权限设置失败');
  }
};

// 处理权限分配取消
const handlePermissionModalCancel = () => {
  isPermissionModalVisible.value = false;
};

// 处理注销用户
const handleWriteOff = async (record: any) => {
  try {
    await deleteUser(record.id);
    message.success('用户删除成功');
    fetchUserList();
  } catch (error: any) {
    message.error(error.message || '用户删除失败');
  }
};

// 处理模态框提交
const handleModalSubmit = async () => {
  try {
    if (modalTitle.value === '新增用户') {
      await registerApi({
        username: formData.username,
        password: formData.password,
        confirmPassword: formData.confirmPassword,
        fei_shu_user_id: formData.fei_shu_user_id,
        desc: formData.desc,
        real_name: formData.real_name,
        mobile: formData.mobile,
        home_path: formData.home_path
      });
      message.success('新增用户成功');
    } else {
      await updateUserInfo({
        user_id: formData.userId,
        real_name: formData.real_name,
        desc: formData.desc,
        mobile: formData.mobile,
        fei_shu_user_id: formData.fei_shu_user_id,
        account_type: 1,
        home_path: formData.home_path,
        enable: 1
      });
      message.success('编辑用户成功');
    }
    isModalVisible.value = false;
    fetchUserList();
  } catch (error: any) {
    message.error(error.message || (modalTitle.value === '新增用户' ? '新增用户失败' : '编辑用户失败'));
  }
};

// 处理密码修改提交
const handlePasswordSubmit = async () => {
  try {
    await changePassword(passwordForm);
    message.success('密码修改成功');
    isPasswordModalVisible.value = false;
  } catch (error: any) {
    message.error(error.message || '密码修改失败');
  }
};

// 处理模态框取消
const handleModalCancel = () => {
  isModalVisible.value = false;
};

// 处理密码修改取消
const handlePasswordCancel = () => {
  isPasswordModalVisible.value = false;
};

// 页面加载时获取数据
onMounted(() => {
  fetchUserList();
  fetchRoleList();
  fetchApis();
});

// 添加随机颜色函数
const tagColors = [
  'pink',
  'red', 
  'orange',
  'green',
  'cyan',
  'blue',
  'purple',
  'geekblue',
  'magenta',
  'volcano',
  'gold',
  'lime'
];

const getRandomColor = () => {
  const index = Math.floor(Math.random() * tagColors.length);
  return tagColors[index];
};
</script>

<style scoped>
.custom-toolbar {
  padding: 8px;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.search-filters {
  display: flex;
  align-items: center;
}
</style>
