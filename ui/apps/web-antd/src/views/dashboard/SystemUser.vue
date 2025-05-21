<template>
  <div class="user-management-container">
    <!-- 顶部卡片 -->
    <div class="dashboard-card">
      <div class="card-title">
        <Icon icon="material-symbols:admin-panel-settings-outline-rounded" class="title-icon" />
        <h2>用户管理</h2>
      </div>
      
      <!-- 查询和操作 -->
      <div class="custom-toolbar">
        <!-- 查询功能 -->
        <div class="search-filters">
          <a-input
            v-model:value="searchText"
            placeholder="请输入用户名或昵称"
            class="search-input"
          >
            <template #prefix>
              <Icon icon="ri:search-line" />
            </template>
          </a-input>
          <a-button type="primary" @click="handleSearch" class="search-button">
            <template #icon><Icon icon="ri:search-line" /></template>
            搜索
          </a-button>
        </div>
        <!-- 操作按钮 -->
        <div class="action-buttons">
          <a-button type="primary" @click="handleAdd" class="add-button">
            <template #icon><Icon icon="material-symbols:add" /></template>
            新增账号
          </a-button>
        </div>
      </div>
    </div>

    <!-- 用户列表表格 -->
    <div class="table-container">
      <a-table 
        :columns="columns" 
        :data-source="userList" 
        :loading="loading" 
        row-key="id"
        :pagination="{ 
          showSizeChanger: true, 
          showQuickJumper: true,
          showTotal: (total: number) => `共 ${total} 条记录`,
          pageSize: 10
        }"
        class="user-table"
      >
        <!-- 用户名列 -->
        <template #bodyCell="{ column, record }">
          <template v-if="column.dataIndex === 'username'">
            <div class="username-cell">
              <div class="avatar-container">
                <div class="user-avatar">{{ record.username.charAt(0).toUpperCase() }}</div>
              </div>
              <span>{{ record.username }}</span>
            </div>
          </template>
        </template>
        
        <!-- 操作列 -->
        <template #action="{ record }">
          <a-space>
            <a-tooltip title="编辑用户信息">
              <a-button type="link" @click="handleEdit(record)" class="action-button edit-button">
                <template #icon><Icon icon="clarity:note-edit-line" /></template>
              </a-button>
            </a-tooltip>
            <a-tooltip title="修改用户密码">
              <a-button type="link" @click="handleChangePassword(record)" class="action-button password-button">
                <template #icon><Icon icon="mdi:key-outline" /></template>
              </a-button>
            </a-tooltip>
            <a-tooltip title="分配用户权限">
              <a-button type="link" @click="handlePermissions(record)" class="action-button permission-button">
                <template #icon><Icon icon="clarity:key-line" /></template>
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
                <a-button type="link" danger class="action-button delete-button">
                  <template #icon><Icon icon="ant-design:delete-outlined" /></template>
                </a-button>
              </a-tooltip>
            </a-popconfirm>
          </a-space>
        </template>
        
        <!-- 角色列 -->
        <template #roles="{ record }">
          <a-space wrap>
            <template v-if="record.roles && record.roles.length > 0">
              <a-tag v-for="role in record.roles" :key="role.id" :color="getTagColor(role.id)" class="role-tag">
                {{ role.name }}
              </a-tag>
            </template>
            <a-tag v-else color="default" class="role-tag">暂无角色</a-tag>
          </a-space>
        </template>
      </a-table>
    </div>

    <!-- 新增/编辑用户对话框 -->
    <a-modal
      v-model:visible="isModalVisible"
      :title="modalTitle"
      @ok="handleModalSubmit"
      @cancel="handleModalCancel"
      :okText="'保存'"
      :cancelText="'取消'"
      class="custom-modal user-form-modal"
      :maskClosable="false"
      :destroyOnClose="true"
      :width="520"
    >
      <div class="modal-content">
        <div class="modal-header-icon" v-if="modalTitle === '新增用户'">
          <div class="icon-wrapper">
            <Icon icon="material-symbols:person-add" />
          </div>
          <div class="header-text">创建新用户账号</div>
        </div>
        <div class="modal-header-icon" v-else>
          <div class="icon-wrapper edit-icon">
            <Icon icon="material-symbols:edit" />
          </div>
          <div class="header-text">编辑用户信息</div>
        </div>
        
        <a-form :model="formData" layout="vertical" class="custom-form">
          <div class="form-grid">
            <a-form-item label="用户名" required v-if="modalTitle === '新增用户'" class="form-item">
              <a-input 
                v-model:value="formData.username" 
                placeholder="请输入用户名"
                class="custom-input" 
              />
            </a-form-item>
            
            <template v-if="modalTitle === '新增用户'">
              <a-form-item label="密码" required class="form-item">
                <a-input-password 
                  v-model:value="formData.password" 
                  placeholder="请输入密码" 
                  class="custom-input"
                />
              </a-form-item>
              
              <a-form-item label="确认密码" required class="form-item">
                <a-input-password 
                  v-model:value="formData.confirmPassword" 
                  placeholder="请再次输入密码" 
                  class="custom-input"
                />
              </a-form-item>
            </template>
            
            <a-form-item label="真实姓名" required class="form-item">
              <a-input 
                v-model:value="formData.real_name" 
                placeholder="请输入真实姓名" 
                class="custom-input"
              />
            </a-form-item>
            
            <a-form-item label="手机号码" class="form-item">
              <a-input 
                v-model:value="formData.mobile" 
                placeholder="请输入手机号码" 
                class="custom-input"
              />
            </a-form-item>
            
            <a-form-item label="飞书用户ID" class="form-item">
              <a-input 
                v-model:value="formData.fei_shu_user_id" 
                placeholder="请输入飞书用户ID" 
                class="custom-input"
              />
            </a-form-item>
            
            <a-form-item label="首页路径" class="form-item">
              <a-input 
                v-model:value="formData.home_path" 
                placeholder="请输入首页路径" 
                class="custom-input"
              />
            </a-form-item>
          </div>
          
          <a-form-item label="描述" class="full-width">
            <a-textarea 
              v-model:value="formData.desc" 
              placeholder="请输入描述" 
              :rows="4"
              class="custom-textarea"
            />
          </a-form-item>
        </a-form>
      </div>
      
      <template #footer>
        <div class="modal-footer">
          <a-button @click="handleModalCancel" class="cancel-button">
            取消
          </a-button>
          <a-button type="primary" @click="handleModalSubmit" class="submit-button">
            <Icon icon="mdi:content-save" class="button-icon" />
            保存
          </a-button>
        </div>
      </template>
    </a-modal>

    <!-- 修改密码对话框 -->
    <a-modal
      v-model:visible="isPasswordModalVisible"
      title="修改密码"
      @ok="handlePasswordSubmit"
      @cancel="handlePasswordCancel"
      :okText="'保存'"
      :cancelText="'取消'"
      class="custom-modal password-modal"
      :maskClosable="false"
      :destroyOnClose="true"
      :width="460"
    >
      <div class="modal-content">
        <div class="modal-header-icon">
          <div class="icon-wrapper password-icon">
            <Icon icon="mdi:lock-reset" />
          </div>
          <div class="header-text">重设用户密码</div>
        </div>
        
        <div class="user-info-display">
          <Icon icon="mdi:account" class="user-icon" />
          <span class="username-display">{{ passwordForm.username }}</span>
        </div>
        
        <a-form :model="passwordForm" layout="vertical" class="custom-form">
          <a-form-item label="原密码" required>
            <a-input-password 
              v-model:value="passwordForm.password" 
              placeholder="请输入原密码" 
              class="custom-input"
            />
          </a-form-item>
          
          <div class="password-divider">
            <div class="divider-line"></div>
            <div class="divider-text">设置新密码</div>
            <div class="divider-line"></div>
          </div>
          
          <a-form-item label="新密码" required>
            <a-input-password 
              v-model:value="passwordForm.newPassword" 
              placeholder="请输入新密码" 
              class="custom-input"
            />
          </a-form-item>
          
          <a-form-item label="确认新密码" required>
            <a-input-password 
              v-model:value="passwordForm.confirmPassword" 
              placeholder="请再次输入新密码" 
              class="custom-input"
            />
          </a-form-item>
        </a-form>
      </div>
      
      <template #footer>
        <div class="modal-footer">
          <a-button @click="handlePasswordCancel" class="cancel-button">
            取消
          </a-button>
          <a-button type="primary" @click="handlePasswordSubmit" class="submit-button">
            <Icon icon="mdi:content-save" class="button-icon" />
            确认修改
          </a-button>
        </div>
      </template>
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
      class="custom-modal permission-modal"
      :maskClosable="false"
      :destroyOnClose="true"
    >
      <div class="modal-content">
        <div class="modal-header-icon">
          <div class="icon-wrapper permission-icon">
            <Icon icon="mdi:shield-key" />
          </div>
          <div class="header-text">设置用户权限</div>
        </div>
        
        <a-tabs v-model:activeKey="activeTabKey" class="permission-tabs" type="card">
          <a-tab-pane key="role" tab="角色分配">
            <div class="tab-content">
              <p class="tab-description">
                <Icon icon="mdi:information-outline" class="info-icon" />
                选择要分配给此用户的角色，用户将继承角色的所有权限
              </p>
              
              <a-form layout="vertical">
                <a-form-item label="选择角色">
                  <div class="role-selection">
                    <a-select
                      v-model:value="selectedRoleIds"
                      mode="multiple"
                      placeholder="请选择角色"
                      style="width: 100%"
                      :options="roleOptions"
                      class="custom-select"
                      :maxTagTextLength="10"
                    />
                  </div>
                  
                  <div class="selected-roles" v-if="selectedRoleIds.length > 0">
                    <div class="selected-title">已选择角色:</div>
                    <div class="role-tags">
                      <a-tag 
                        v-for="roleId in selectedRoleIds" 
                        :key="roleId" 
                        :color="getTagColor(roleId)"
                        class="role-tag selected-role-tag"
                      >
                        {{ getRoleName(roleId) }}
                        <Icon 
                          icon="mdi:close-circle" 
                          class="remove-tag" 
                          @click="removeRole(roleId)" 
                        />
                      </a-tag>
                    </div>
                  </div>
                </a-form-item>
              </a-form>
            </div>
          </a-tab-pane>
          
          <a-tab-pane key="api" tab="接口权限">
            <div class="tab-content">
              <p class="tab-description">
                <Icon icon="mdi:information-outline" class="info-icon" />
                直接设置用户可访问的API接口权限
              </p>
              
              <div class="api-tree-container">
                <div class="tree-search">
                  <a-input 
                    placeholder="搜索API..." 
                    class="tree-search-input"
                  >
                    <template #prefix>
                      <Icon icon="ri:search-line" />
                    </template>
                  </a-input>
                </div>
                
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
                  class="permission-tree"
                />
              </div>
            </div>
          </a-tab-pane>
        </a-tabs>
      </div>
      
      <template #footer>
        <div class="modal-footer">
          <a-button @click="handlePermissionModalCancel" class="cancel-button">
            取消
          </a-button>
          <a-button type="primary" @click="handlePermissionModalSubmit" class="submit-button">
            <Icon icon="mdi:content-save" class="button-icon" />
            保存权限设置
          </a-button>
        </div>
      </template>
    </a-modal>
  </div>
</template>

<script lang="ts" setup>
import { reactive, ref, onMounted, computed } from 'vue';
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

interface Role {
  id: number;
  name: string;
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
    width: 80
  },
  {
    title: '用户名',
    dataIndex: 'username',
    key: 'username',
    width: 150,
    slots: { customRender: 'username' }
  },
  {
    title: '真实姓名',
    dataIndex: 'real_name',
    key: 'real_name',
    width: 120
  },
  {
    title: '手机号码',
    dataIndex: 'mobile',
    key: 'mobile',
    width: 120
  },
  {
    title: '飞书用户ID',
    dataIndex: 'fei_shu_user_id',
    key: 'fei_shu_user_id',
    width: 150
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
    width: 180,
    fixed: 'right'
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
  passwordForm.password = '';
  passwordForm.newPassword = '';
  passwordForm.confirmPassword = '';
  isPasswordModalVisible.value = true;
};

// 处理权限分配
const handlePermissions = (record: any) => {
  currentUserId.value = record.id;
  selectedRoleIds.value = Array.isArray(record.roles) ? record.roles.map((role: any) => role.id) : [];
  selectedApiIds.value = Array.isArray(record.apis) ? record.apis.map((api: any) => api.id) : [];
  isPermissionModalVisible.value = true;
};

// 获取角色名称
const getRoleName = (roleId: number): string => {
  const role = roleOptions.value.find(r => r.value === roleId);
  return role ? role.label : `角色 ${roleId}`;
};

// 移除选定角色
const removeRole = (roleId: number): void => {
  selectedRoleIds.value = selectedRoleIds.value.filter(id => id !== roleId);
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

// 定义标签颜色映射
const tagColors = [
  '#1890ff', // 蓝色
  '#13c2c2', // 青色
  '#52c41a', // 绿色
  '#faad14', // 黄色
  '#722ed1', // 紫色
  '#eb2f96', // 粉色
  '#f5222d', // 红色
  '#fa541c', // 橙色
  '#2f54eb', // 蓝紫色
  '#fadb14', // 金色
  '#a0d911', // 亮绿色
  '#1d39c4'  // 深蓝色
];

// 根据角色ID分配固定颜色
const getTagColor = (roleId: number): string => {
  const index = (roleId % tagColors.length);
  return tagColors[index] || '#1890ff'; // 添加默认颜色作为后备
};
</script>

<style scoped>
/* 整体容器样式 */
.user-management-container {
  padding: 20px;
  background-color: #f0f2f5;
  min-height: 100vh;
  font-family: 'Roboto', 'PingFang SC', 'Microsoft YaHei', sans-serif;
}

/* 顶部卡片样式 */
.dashboard-card {
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
  padding: 20px;
  margin-bottom: 20px;
  transition: all 0.3s;
}

.card-title {
  display: flex;
  align-items: center;
  margin-bottom: 20px;
}

.title-icon {
  font-size: 28px;
  margin-right: 10px;
  color: #1890ff;
}

.card-title h2 {
  margin: 0;
  font-size: 20px;
  font-weight: 500;
  color: #1e293b;
}

/* 工具栏样式 */
.custom-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 16px;
}

.search-filters {
  display: flex;
  align-items: center;
  gap: 12px;
}

.search-input {
  width: 280px;
  border-radius: 6px;
  transition: all 0.3s;
}

.search-input:hover, 
.search-input:focus {
  box-shadow: 0 0 0 2px rgba(24, 144, 255, 0.1);
}

.search-button {
  border-radius: 6px;
  display: flex;
  align-items: center;
  gap: 4px;
}

.add-button {
  border-radius: 6px;
  background: linear-gradient(90deg, #1890ff, #36cfc9);
  border: none;
  box-shadow: 0 2px 6px rgba(24, 144, 255, 0.3);
  display: flex;
  align-items: center;
  gap: 6px;
  transition: all 0.3s;
}

.add-button:hover {
  background: linear-gradient(90deg, #40a9ff, #5cdbd3);
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(24, 144, 255, 0.4);
}

/* 表格容器样式 */
.table-container {
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
  padding: 20px;
  overflow: hidden;
}

.user-table {
  width: 100%;
}

/* 用户名单元格样式 */
.username-cell {
  display: flex;
  align-items: center;
  gap: 12px;
}

.avatar-container {
  width: 36px;
  height: 36px;
}

.user-avatar {
  width: 36px;
  height: 36px;
  background: linear-gradient(135deg, #1890ff, #36cfc9);
  color: white;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 500;
  font-size: 16px;
}

/* 角色标签样式 */
.role-tag {
  border-radius: 4px;
  padding: 2px 8px;
  margin: 2px;
  font-size: 12px;
  border: none;
}

/* 操作按钮样式 */
.action-button {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 6px;
  transition: all 0.2s;
}

.action-button:hover {
  background-color: #f0f0f0;
  transform: translateY(-1px);
}

.edit-button {
  color: #1890ff;
}

.password-button {
  color: #faad14;
}

.permission-button {
  color: #52c41a;
}

.delete-button {
  color: #f5222d;
}

/* 通用模态框样式 */
:deep(.custom-modal .ant-modal-content) {
  border-radius: 12px;
  overflow: hidden;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.15);
}

:deep(.custom-modal .ant-modal-header) {
  background: #fff;
  padding: 20px 24px 0;
  border-bottom: none;
}

:deep(.custom-modal .ant-modal-title) {
  font-size: 18px;
  font-weight: 600;
  color: #1e293b;
}

:deep(.custom-modal .ant-modal-body) {
  padding: 0 24px 24px;
}

:deep(.custom-modal .ant-modal-footer) {
  border-top: 1px solid #f0f0f0;
  padding: 16px 24px;
}

.modal-content {
  padding: 0;
}

.modal-header-icon {
  display: flex;
  flex-direction: column;
  align-items: center;
  margin-bottom: 24px;
  padding-top: 20px;
}

.icon-wrapper {
  width: 64px;
  height: 64px;
  border-radius: 50%;
  background: linear-gradient(135deg, #1890ff, #36cfc9);
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 16px;
  box-shadow: 0 4px 12px rgba(24, 144, 255, 0.25);
}

.icon-wrapper svg {
  font-size: 32px;
  color: white;
}

.edit-icon {
  background: linear-gradient(135deg, #52c41a, #13c2c2);
}

.password-icon {
  background: linear-gradient(135deg, #faad14, #fa541c);
}

.permission-icon {
  background: linear-gradient(135deg, #722ed1, #2f54eb);
}

.header-text {
  font-size: 16px;
  color: #1e293b;
  font-weight: 500;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

.cancel-button {
  border-radius: 6px;
  border: 1px solid #d9d9d9;
  background-color: white;
  color: #595959;
  padding: 0 16px;
  height: 36px;
  transition: all 0.3s;
}

.cancel-button:hover {
  color: #1890ff;
  border-color: #1890ff;
}

.submit-button {
  border-radius: 6px;
  border: none;
  background: linear-gradient(90deg, #1890ff, #36cfc9);
  color: white;
  padding: 0 16px;
  height: 36px;
  display: flex;
  align-items: center;
  gap: 6px;
  box-shadow: 0 2px 6px rgba(24, 144, 255, 0.25);
  transition: all 0.3s;
}

.submit-button:hover {
  background: linear-gradient(90deg, #40a9ff, #5cdbd3);
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(24, 144, 255, 0.35);
}

.button-icon {
  font-size: 16px;
}

/* 用户表单模态框 */
.custom-form {
  width: 100%;
}

.form-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}

.form-item {
  margin-bottom: 16px;
}

.full-width {
  grid-column: span 2;
}

:deep(.custom-input) {
  border-radius: 6px;
  transition: all 0.3s;
  height: 38px;
}

:deep(.custom-input:hover) {
  border-color: #40a9ff;
}

:deep(.custom-input:focus) {
  border-color: #1890ff;
  box-shadow: 0 0 0 2px rgba(24, 144, 255, 0.2);
}

:deep(.custom-textarea) {
  border-radius: 6px;
  transition: all 0.3s;
}

:deep(.custom-textarea:hover) {
  border-color: #40a9ff;
}

:deep(.custom-textarea:focus) {
  border-color: #1890ff;
  box-shadow: 0 0 0 2px rgba(24, 144, 255, 0.2);
}

/* 密码模态框样式 */
.user-info-display {
  background-color: #f9f9f9;
  border-radius: 8px;
  padding: 12px 16px;
  margin-bottom: 24px;
  display: flex;
  align-items: center;
  gap: 10px;
}

.user-icon {
  font-size: 20px;
  color: #1890ff;
}

.username-display {
  font-size: 16px;
  font-weight: 500;
  color: #1e293b;
}

.password-divider {
  display: flex;
  align-items: center;
  margin: 24px 0;
  gap: 12px;
}

.divider-line {
  flex: 1;
  height: 1px;
  background-color: #f0f0f0;
}

.divider-text {
  color: #8c8c8c;
  font-size: 14px;
  white-space: nowrap;
}

/* 权限分配模态框样式 */
:deep(.permission-tabs .ant-tabs-nav) {
  margin-bottom: 24px;
}

:deep(.permission-tabs.ant-tabs-card .ant-tabs-nav .ant-tabs-tab) {
  border-radius: 8px 8px 0 0;
  margin-right: 6px;
  padding: 12px 20px;
  transition: all 0.3s;
}

:deep(.permission-tabs.ant-tabs-card .ant-tabs-nav .ant-tabs-tab-active) {
  background: linear-gradient(to bottom, #f0f7ff, #e6f7ff);
  border-color: #91caff;
  border-bottom: none;
}

:deep(.permission-tabs.ant-tabs-card .ant-tabs-nav .ant-tabs-tab:not(.ant-tabs-tab-active):hover) {
  color: #1890ff;
}

.tab-content {
  padding: 8px 0;
}

.tab-description {
  background-color: #f0f7ff;
  border-radius: 8px;
  padding: 12px 16px;
  margin-bottom: 24px;
  color: #1e293b;
  font-size: 14px;
  display: flex;
  align-items: center;
  gap: 8px;
}

.info-icon {
  color: #1890ff;
  font-size: 18px;
}

.role-selection {
  margin-bottom: 16px;
}

:deep(.custom-select) {
  width: 100%;
  border-radius: 6px;
}

:deep(.custom-select .ant-select-selector) {
  border-radius: 6px !important;
  transition: all 0.3s;
}

:deep(.custom-select:hover .ant-select-selector) {
  border-color: #40a9ff !important;
}

:deep(.custom-select.ant-select-focused .ant-select-selector) {
  border-color: #1890ff !important;
  box-shadow: 0 0 0 2px rgba(24, 144, 255, 0.2) !important;
}

.selected-roles {
  margin-top: 16px;
  border: 1px solid #f0f0f0;
  border-radius: 8px;
  padding: 12px;
  background-color: #fafafa;
}

.selected-title {
  font-weight: 500;
  margin-bottom: 8px;
  color: #1e293b;
}

.role-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.selected-role-tag {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 12px;
  font-size: 13px;
}

.remove-tag {
  cursor: pointer;
  transition: all 0.2s;
}

.remove-tag:hover {
  transform: scale(1.2);
}

.api-tree-container {
  border: 1px solid #f0f0f0;
  border-radius: 8px;
  padding: 0;
  max-height: 400px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.tree-search {
  padding: 12px;
  border-bottom: 1px solid #f0f0f0;
  background-color: #fafafa;
}

.tree-search-input {
  border-radius: 6px;
}

:deep(.permission-tree) {
  flex: 1;
  overflow-y: auto;
  padding: 16px;
}

:deep(.permission-tree .ant-tree-checkbox) {
  margin-right: 8px;
}

:deep(.permission-tree .ant-tree-checkbox-inner) {
  border-radius: 4px;
  transition: all 0.3s;
}

:deep(.permission-tree .ant-tree-checkbox-checked .ant-tree-checkbox-inner) {
  background-color: #1890ff;
  border-color: #1890ff;
}

:deep(.permission-tree .ant-tree-node-content-wrapper) {
  padding: 6px 8px;
  border-radius: 6px;
  transition: all 0.3s;
}

:deep(.permission-tree .ant-tree-node-content-wrapper:hover) {
  background-color: #e6f7ff;
}

:deep(.permission-tree .ant-tree-node-selected) {
  background-color: #e6f7ff !important;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .custom-toolbar {
    flex-direction: column;
    align-items: flex-start;
  }
  
  .search-filters {
    width: 100%;
    margin-bottom: 12px;
  }
  
  .search-input {
    width: 100%;
  }
  
  .form-grid {
    grid-template-columns: 1fr;
  }
  
  .full-width {
    grid-column: span 1;
  }
}
</style>