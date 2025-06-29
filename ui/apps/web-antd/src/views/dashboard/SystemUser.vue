<template>
  <div class="user-management">
    <!-- 页面头部 -->
    <div class="page-header">
      <h1>用户管理</h1>
      <div class="header-actions">
        <a-button @click="handleRefresh">
          <Icon icon="material-symbols:refresh" />
          刷新
        </a-button>
        <a-button type="primary" @click="handleAdd">
          <Icon icon="material-symbols:add" />
          新建用户
        </a-button>
      </div>
    </div>

    <!-- 统计卡片 -->
    <div class="stats-grid">
      <div class="stat-card">
        <div class="stat-number">{{ paginationConfig.total }}</div>
        <div class="stat-label">总用户数</div>
      </div>
      <div class="stat-card">
        <div class="stat-number">{{ userStatistics.active_user_count || 0 }}</div>
        <div class="stat-label">活跃用户</div>
      </div>
      <div class="stat-card">
        <div class="stat-number">{{ userStatistics.admin_count || 0 }}</div>
        <div class="stat-label">管理员</div>
      </div>
    </div>

    <!-- 搜索筛选区域 -->
    <div class="search-section">
      <div class="search-left">
        <a-input
          v-model:value="searchParams.search"
          placeholder="搜索用户名"
          allowClear
          @pressEnter="handleSearch"
          class="search-input"
        >
          <template #prefix>
            <Icon icon="material-symbols:search" />
          </template>
        </a-input>
        
        <a-select
          v-model:value="searchParams.enable"
          placeholder="状态筛选"
          allowClear
          class="status-select"
        >
          <a-select-option :value="1">启用</a-select-option>
          <a-select-option :value="2">禁用</a-select-option>
        </a-select>

        <a-select
          v-model:value="searchParams.account_type"
          placeholder="账号类型"
          allowClear
          class="type-select"
        >
          <a-select-option :value="1">普通用户</a-select-option>
          <a-select-option :value="2">服务账号</a-select-option>
        </a-select>
      </div>
      
      <div class="search-right">
        <a-button type="primary" @click="handleSearch">搜索</a-button>
        <a-button @click="handleReset">重置</a-button>
      </div>
    </div>

    <!-- 用户表格 -->
    <div class="table-container">
      <a-table
        :columns="tableColumns"
        :data-source="userList"
        :pagination="paginationConfig"
        :loading="loading"
        row-key="id"
        size="middle"
        @change="handleTableChange"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'user'">
            <div class="user-info">
              <div class="user-avatar">
                <img v-if="record.avatar" :src="getAvatarUrl(record.avatar)" :alt="record.username" @error="handleAvatarError" />
                <div v-else class="default-avatar">{{ getInitials(record.username) }}</div>
              </div>
              <div>
                <div class="user-name">{{ record.real_name || record.username }}</div>
                <div class="user-username">{{ record.username }}</div>
              </div>
            </div>
          </template>
          
          <template v-if="column.key === 'contact'">
            <div class="contact-info">
              <div v-if="record.mobile">{{ record.mobile }}</div>
              <div v-if="record.email">{{ record.email }}</div>
            </div>
          </template>
          
          <template v-if="column.key === 'type'">
            <a-tag v-if="record.account_type === 2" color="orange">服务账号</a-tag>
            <a-tag v-else-if="isAdminUser(record)" color="red">管理员</a-tag>
            <a-tag v-else color="blue">普通用户</a-tag>
          </template>
          
          <template v-if="column.key === 'status'">
            <a-switch 
              :checked="record.enable === 1" 
              @change="(checked: boolean) => handleStatusChange(record, checked ? 1 : 2)"
              size="small"
            />
          </template>
          
          <template v-if="column.key === 'roles'">
            {{ record.roles?.length || 0 }}
          </template>
          
          <template v-if="column.key === 'created_at'">
            {{ formatTime(record.created_at) }}
          </template>
          
          <template v-if="column.key === 'actions'">
            <div class="action-buttons">
              <a-button type="text" size="small" @click="handleView(record)">查看</a-button>
              <a-button type="text" size="small" @click="handleEdit(record)">编辑</a-button>
              <a-button type="text" size="small" @click="handleRoleManagement(record)">角色</a-button>
              <a-popconfirm title="确定要删除吗？" @confirm="handleDelete(record)">
                <a-button type="text" size="small" danger>删除</a-button>
              </a-popconfirm>
            </div>
          </template>
        </template>
      </a-table>
    </div>

    <!-- 查看用户详情 -->
    <a-modal v-model:open="viewModalVisible" title="用户详情" width="700px" :footer="null">
      <div v-if="viewUserData" class="user-detail">
        <div class="detail-section">
          <h3>基本信息</h3>
          <div class="detail-grid">
            <div class="detail-item">
              <label>用户名</label>
              <span>{{ viewUserData.username }}</span>
            </div>
            <div class="detail-item">
              <label>真实姓名</label>
              <span>{{ viewUserData.real_name || '-' }}</span>
            </div>
            <div class="detail-item">
              <label>手机号</label>
              <span>{{ viewUserData.mobile || '-' }}</span>
            </div>
            <div class="detail-item">
              <label>邮箱</label>
              <span>{{ viewUserData.email || '-' }}</span>
            </div>
            <div class="detail-item">
              <label>飞书ID</label>
              <span>{{ viewUserData.fei_shu_user_id || '-' }}</span>
            </div>
            <div class="detail-item avatar-detail">
              <label>头像</label>
              <div class="detail-avatar">
                <img v-if="viewUserData.avatar" :src="getAvatarUrl(viewUserData.avatar)" :alt="viewUserData.username" @error="handleAvatarError" />
                <div v-else class="default-avatar">{{ getInitials(viewUserData.username) }}</div>
              </div>
            </div>
            <div class="detail-item">
              <label>账号类型</label>
              <a-tag :color="viewUserData.account_type === 2 ? 'orange' : 'blue'">
                {{ viewUserData.account_type === 2 ? '服务账号' : '普通用户' }}
              </a-tag>
            </div>
            <div class="detail-item">
              <label>状态</label>
              <a-tag :color="viewUserData.enable === 1 ? 'green' : 'red'">
                {{ viewUserData.enable === 1 ? '启用' : '禁用' }}
              </a-tag>
            </div>
            <div class="detail-item">
              <label>注册时间</label>
              <span>{{ formatTime(viewUserData.created_at) }}</span>
            </div>
            <div class="detail-item" v-if="viewUserData.desc">
              <label>用户描述</label>
              <span>{{ viewUserData.desc }}</span>
            </div>
          </div>
        </div>

        <div class="detail-section" v-if="viewUserData.roles?.length">
          <h3>角色信息</h3>
          <div class="roles-list">
            <a-tag 
              v-for="role in viewUserData.roles" 
              :key="role.id"
              :color="role.status === 1 ? 'blue' : 'default'"
            >
              {{ role.name }}
            </a-tag>
          </div>
        </div>
      </div>
    </a-modal>

    <!-- 编辑用户 -->
    <a-modal
      v-model:open="modalVisible"
      :title="modalTitle"
      width="600px"
      @ok="handleSubmit"
      :confirm-loading="submitLoading"
    >
      <a-form ref="formRef" :model="formData" :rules="formRules" layout="vertical">
        <a-form-item label="头像" name="avatar">
          <div class="avatar-upload">
            <div class="avatar-preview" :class="{ 'avatar-upload-loading': avatarUploading }">
              <img v-if="formData.avatar" :src="getAvatarUrl(formData.avatar)" alt="用户头像" @error="handleAvatarError" />
              <div v-else class="default-avatar">
                <Icon icon="material-symbols:person" style="font-size: 48px;" />
              </div>
              <div v-if="avatarUploading" class="upload-loading">
                <Icon icon="material-symbols:refresh" class="loading-icon" />
              </div>
            </div>
            <div class="avatar-upload-actions">
              <a-upload
                name="avatar"
                :show-upload-list="false"
                action="/api/upload"
                :before-upload="beforeAvatarUpload"
                @change="handleAvatarChange"
                accept="image/*"
              >
                <a-button :loading="avatarUploading">
                  <Icon icon="material-symbols:upload" />
                  上传头像
                </a-button>
              </a-upload>
              <a-button v-if="formData.avatar" @click="handleRemoveAvatar" danger>
                <Icon icon="material-symbols:delete" />
                移除头像
              </a-button>
            </div>
          </div>
        </a-form-item>
        
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="用户名" name="username">
              <a-input v-model:value="formData.username" :disabled="modalTitle === '编辑用户'" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="真实姓名" name="real_name">
              <a-input v-model:value="formData.real_name" />
            </a-form-item>
          </a-col>
        </a-row>
        
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="手机号" name="mobile">
              <a-input v-model:value="formData.mobile" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="邮箱" name="email">
              <a-input v-model:value="formData.email" />
            </a-form-item>
          </a-col>
        </a-row>
        
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="飞书ID" name="fei_shu_user_id">
              <a-input v-model:value="formData.fei_shu_user_id" />
            </a-form-item>
          </a-col>
        </a-row>
        
        <a-row :gutter="16" v-if="modalTitle === '新建用户'">
          <a-col :span="12">
            <a-form-item label="密码" name="password">
              <a-input-password v-model:value="formData.password" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="确认密码" name="confirmPassword">
              <a-input-password v-model:value="formData.confirmPassword" />
            </a-form-item>
          </a-col>
        </a-row>

        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="账号类型" name="account_type">
              <a-select v-model:value="formData.account_type">
                <a-select-option :value="1">普通用户</a-select-option>
                <a-select-option :value="2">服务账号</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="状态" name="enable">
              <a-select v-model:value="formData.enable">
                <a-select-option :value="1">启用</a-select-option>
                <a-select-option :value="2">禁用</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>

        <a-form-item label="用户描述" name="desc">
          <a-textarea v-model:value="formData.desc" :rows="3" placeholder="请输入用户描述信息" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 角色管理 -->
    <a-modal v-model:open="roleModalVisible" title="角色管理" width="700px" :footer="null">
      <div v-if="currentUser">
        <div class="role-header">
          <h4>{{ currentUser.real_name || currentUser.username }} 的角色</h4>
        </div>
        
        <a-tabs>
          <a-tab-pane key="assigned" tab="已分配角色">
            <div class="role-list">
              <div v-if="assignedRoles.length === 0" class="empty-state">
                暂无已分配的角色
              </div>
              <div v-for="role in assignedRoles" :key="role.id" class="role-item">
                <div class="role-info">
                  <div class="role-name">{{ role.name }}</div>
                  <div class="role-desc">{{ role.description || '暂无描述' }}</div>
                </div>
                <a-button type="text" danger size="small" @click="handleRevokeRole(role)">移除</a-button>
              </div>
            </div>
          </a-tab-pane>
          
          <a-tab-pane key="available" tab="可分配角色">
            <div class="role-list">
              <div v-if="availableRoles.length === 0" class="empty-state">
                暂无可分配的角色
              </div>
              <div v-for="role in availableRoles" :key="role.id" class="role-item">
                <div class="role-info">
                  <div class="role-name">{{ role.name }}</div>
                  <div class="role-desc">{{ role.description || '暂无描述' }}</div>
                </div>
                <a-button type="primary" size="small" @click="handleAssignRole(role)">分配</a-button>
              </div>
            </div>
          </a-tab-pane>
        </a-tabs>
      </div>
    </a-modal>
  </div>
</template>

<script lang="ts" setup>
import { ref, reactive, computed, onMounted } from 'vue';
import { message } from 'ant-design-vue';
import { Icon } from '@iconify/vue';
import type { FormInstance } from 'ant-design-vue';

import { 
  getUserList, 
  registerApi, 
  updateUserInfo, 
  deleteUser,
  getUserDetailApi,
  getUserStatistics,
  type UserSignUpReq,
  type UpdateProfileReq,
  type GetUserListReq
} from '#/api/core/user';
import { 
  assignRolesToUserApi,
  revokeRolesFromUserApi,
  getUserRolesApi,
  listRolesApi
} from '#/api/core/system';

// 类型定义
interface UserStatistics {
  admin_count: number;
  active_user_count: number;
}

interface UserRole {
  id: number;
  name: string;
  code?: string;
  description?: string;
  status?: number;
}

interface UserInfo {
  id: number;
  username: string;
  real_name?: string;
  mobile?: string;
  email?: string;
  desc?: string;
  enable: number;
  account_type: number;
  avatar?: string;
  created_at?: any;
  roles?: UserRole[];
  fei_shu_user_id?: string;
  home_path?: string;
}

// 表单引用
const formRef = ref<FormInstance>();

// 表格列配置
const tableColumns = [
  { title: '用户', key: 'user', width: 200, fixed: 'left' },
  { title: '联系方式', key: 'contact', width: 160 },
  { title: '类型', key: 'type', width: 100, align: 'center' },
  { title: '状态', key: 'status', width: 80, align: 'center' },
  { title: '角色数', key: 'roles', width: 80, align: 'center' },
  { title: '注册时间', key: 'created_at', width: 120 },
  { title: '操作', key: 'actions', width: 160, fixed: 'right' }
];

// 状态管理
const loading = ref(false);
const submitLoading = ref(false);
const avatarUploading = ref(false);
const modalVisible = ref(false);
const roleModalVisible = ref(false);
const viewModalVisible = ref(false);
const modalTitle = ref('');

// 数据
const userList = ref<UserInfo[]>([]);
const roleList = ref<UserRole[]>([]);
const currentUser = ref<UserInfo | null>(null);
const assignedRoles = ref<UserRole[]>([]);
const viewUserData = ref<UserInfo | null>(null);
const userStatistics = ref<UserStatistics>({
  admin_count: 0,
  active_user_count: 0
});

// 搜索参数
const searchParams = reactive({
  search: '',
  enable: undefined as number | undefined,
  account_type: undefined as number | undefined
});

// 分页配置
const paginationConfig = reactive({
  current: 1,
  pageSize: 20,
  total: 0,
  showSizeChanger: true,
  showQuickJumper: true,
  pageSizeOptions: ['10', '20', '50', '100'],
  showTotal: (total: number, range: [number, number]) => 
    `第 ${range[0]}-${range[1]} 条，共 ${total} 条`
});

// 表单数据初始化
const initFormData = () => ({
  username: '',
  real_name: '',
  mobile: '',
  email: '',
  desc: '',
  enable: 1,
  account_type: 1,
  password: '',
  confirmPassword: '',
  home_path: '',
  id: 0,
  avatar: '',
  fei_shu_user_id: ''
});

const formData = reactive(initFormData());

// 表单验证规则
const formRules = computed(() => {
  const rules: any = {
    username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
    real_name: [{ required: true, message: '请输入真实姓名', trigger: 'blur' }],
    mobile: [{ required: true, message: '请输入手机号', trigger: 'blur' }]
  };

  if (modalTitle.value === '新建用户') {
    rules.password = [
      { required: true, message: '请输入密码', trigger: 'blur' },
      { min: 6, message: '密码长度至少6位', trigger: 'blur' }
    ];
    rules.confirmPassword = [
      { required: true, message: '请确认密码', trigger: 'blur' },
      {
        validator: (_rule: any, value: string) => {
          if (value && value !== formData.password) {
            return Promise.reject('两次输入的密码不一致');
          }
          return Promise.resolve();
        },
        trigger: 'blur'
      }
    ];
  }

  return rules;
});

// 计算属性
const availableRoles = computed(() => {
  if (!currentUser.value) return [];
  const assignedIds = assignedRoles.value.map(role => role.id);
  return roleList.value.filter(role => !assignedIds.includes(role.id));
});

// 头像相关函数
const getAvatarUrl = (avatar: string) => {
  if (!avatar) return '';
  if (avatar.startsWith('http')) return avatar;
  if (avatar.startsWith('data:')) return avatar;
  return `/api/uploads/${avatar}`;
};

const getInitials = (username: string) => {
  if (!username) return 'U';
  return username.charAt(0).toUpperCase();
};

const handleAvatarError = (event: Event) => {
  const target = event.target as HTMLImageElement;
  if (target) {
    target.style.display = 'none';
  }
};

const beforeAvatarUpload = (file: File) => {
  const isJpgOrPng = file.type === 'image/jpeg' || file.type === 'image/png' || file.type === 'image/gif' || file.type === 'image/webp';
  if (!isJpgOrPng) {
    message.error('只能上传 JPG/PNG/GIF/WebP 格式的图片!');
    return false;
  }
  
  const isLt2M = file.size / 1024 / 1024 < 2;
  if (!isLt2M) {
    message.error('图片大小不能超过 2MB!');
    return false;
  }
  
  handleAvatarUploadLocal(file);
  return false;
};

const handleAvatarUploadLocal = (file: File) => {
  avatarUploading.value = true;
  const reader = new FileReader();
  reader.onload = (e) => {
    setTimeout(() => {
      if (e.target?.result) {
        formData.avatar = e.target.result as string;
        message.success('头像设置成功');
      }
      avatarUploading.value = false;
    }, 1000);
  };
  reader.readAsDataURL(file);
};

const handleAvatarChange = (info: any) => {
  if (info.file.status === 'uploading') {
    avatarUploading.value = true;
    return;
  }
  
  if (info.file.status === 'done') {
    avatarUploading.value = false;
    if (info.file.response && info.file.response.data && info.file.response.data.url) {
      formData.avatar = info.file.response.data.url;
      message.success('头像上传成功');
    } else if (info.file.response && info.file.response.url) {
      formData.avatar = info.file.response.url;
      message.success('头像上传成功');
    } else {
      message.error('头像上传失败，返回数据格式错误');
    }
  }
  
  if (info.file.status === 'error') {
    avatarUploading.value = false;
    message.error('头像上传失败');
  }
};

const handleRemoveAvatar = () => {
  formData.avatar = '';
  message.success('头像已移除');
};

// 工具函数
const formatTime = (timestamp: any) => {
  if (!timestamp) return '-';
  return new Date(typeof timestamp === 'number' ? timestamp * 1000 : timestamp)
    .toLocaleDateString('zh-CN');
};

const isAdminUser = (user: UserInfo) => {
  return user.roles?.some((role: UserRole) => role.code === 'admin') || false;
};

// API 调用
const fetchUserList = async () => {
  loading.value = true;
  try {
    const params: GetUserListReq = {
      page: paginationConfig.current,
      size: paginationConfig.pageSize,
      search: searchParams.search || '',
      enable: 0,
      account_type: 0
    };

    if (searchParams.enable === 1 || searchParams.enable === 2) {
      params.enable = searchParams.enable;
    }

    if (searchParams.account_type === 1 || searchParams.account_type === 2) {
      params.account_type = searchParams.account_type;
    }

    const response = await getUserList(params);
    
    userList.value = response.items || [];
    paginationConfig.total = response.total || 0;
    
  } catch (error: any) {
    message.error(error.message || '获取用户列表失败');
    userList.value = [];
    paginationConfig.total = 0;
  } finally {
    loading.value = false;
  }
};

const fetchRoleList = async () => {
  try {
    const response = await listRolesApi({ page: 1, size: 100 });
    roleList.value = response.items || [];
  } catch (error: any) {
    message.error(error.message || '获取角色列表失败');
  }
};

const fetchUserRoles = async (userId: number) => {
  try {
    const response = await getUserRolesApi(userId);
    assignedRoles.value = response.items || [];
  } catch (error: any) {
    message.error(error.message || '获取用户角色失败');
  }
};

const fetchUserStatistics = async () => {
  try {
    const response = await getUserStatistics();
    userStatistics.value = response;
  } catch (error: any) {
    message.error(error.message || '获取用户统计失败');
    // 设置默认值避免显示undefined
    userStatistics.value = {
      admin_count: 0,
      active_user_count: 0
    };
  }
};

// 事件处理
const handleSearch = () => {
  paginationConfig.current = 1;
  fetchUserList();
};

const handleReset = () => {
  searchParams.search = '';
  searchParams.enable = undefined;
  searchParams.account_type = undefined;
  paginationConfig.current = 1;
  fetchUserList();
};

const handleRefresh = () => {
  fetchUserList();
  fetchUserStatistics();
};

const handleTableChange = (pagination: any) => {
  paginationConfig.current = pagination.current;
  paginationConfig.pageSize = pagination.pageSize;
  fetchUserList();
};

const handleAdd = () => {
  modalTitle.value = '新建用户';
  Object.assign(formData, initFormData());
  modalVisible.value = true;
};

const handleEdit = async (user: UserInfo) => {
  try {
    modalTitle.value = '编辑用户';
    const response = await getUserDetailApi(user.id);
    Object.assign(formData, {
      id: response.id,
      username: response.username,
      real_name: response.real_name,
      mobile: response.mobile,
      email: response.email || '',
      desc: response.desc || '',
      enable: response.enable,
      account_type: response.account_type,
      home_path: response.home_path || '',
      avatar: response.avatar || '',
      fei_shu_user_id: response.fei_shu_user_id || '',
    });
    modalVisible.value = true;
  } catch (error: any) {
    message.error(error.message || '获取用户详情失败');
  }
};

const handleView = async (user: UserInfo) => {
  try {
    const response = await getUserDetailApi(user.id);
    viewUserData.value = response;
    viewModalVisible.value = true;
  } catch (error: any) {
    message.error(error.message || '获取用户详情失败');
  }
};

const handleRoleManagement = async (user: UserInfo) => {
  currentUser.value = user;
  await fetchUserRoles(user.id);
  roleModalVisible.value = true;
};

const handleDelete = async (user: UserInfo) => {
  try {
    await deleteUser(user.id);
    message.success('删除成功');
    if (userList.value.length === 1 && paginationConfig.current > 1) {
      paginationConfig.current--;
    }
    await fetchUserList();
    await fetchUserStatistics(); // 删除后更新统计
  } catch (error: any) {
    message.error(error.message || '删除失败');
  }
};

// 修复状态变更函数
const handleStatusChange = async (user: UserInfo, newStatus: number) => {
  const originalStatus = user.enable;
  
  try {
    // 乐观更新
    user.enable = newStatus;
    
    const updateData: UpdateProfileReq = {
      id: user.id,
      real_name: user.real_name || '',
      mobile: user.mobile || '',
      account_type: user.account_type as 1 | 2,
      enable: newStatus as 1 | 2,
      desc: user.desc || '',
      fei_shu_user_id: user.fei_shu_user_id || '',
      home_path: user.home_path || '',
      email: user.email || '',
      avatar: user.avatar || ''
    };
    
    await updateUserInfo(updateData);
    message.success('状态更新成功');
    
    // 更新统计数据
    await fetchUserStatistics();
  } catch (error: any) {
    // 发生错误时，恢复原来的状态
    user.enable = originalStatus;
    message.error(error.message || '状态更新失败');
  }
};

const handleSubmit = async () => {
  try {
    await formRef.value?.validate();
    submitLoading.value = true;
    
    if (modalTitle.value === '新建用户') {
      const signUpData: UserSignUpReq = {
        username: formData.username,
        password: formData.password,
        mobile: formData.mobile,
        real_name: formData.real_name,
        desc: formData.desc,
        account_type: formData.account_type as 1 | 2,
        enable: formData.enable as 1 | 2,
        home_path: formData.home_path,
        fei_shu_user_id: formData.fei_shu_user_id || ''
      };
      
      // 添加可选字段
      if (formData.email) {
        (signUpData as any).email = formData.email;
      }
      
      if (formData.avatar) {
        (signUpData as any).avatar = formData.avatar;
      }
      
      // 修复：添加飞书ID字段
      if (formData.fei_shu_user_id) {
        (signUpData as any).fei_shu_user_id = formData.fei_shu_user_id;
      }
      
      await registerApi(signUpData);
      message.success('创建成功');
    } else {
      const updateData: UpdateProfileReq = {
        id: formData.id as number,
        real_name: formData.real_name,
        mobile: formData.mobile,
        account_type: formData.account_type as 1 | 2,
        desc: formData.desc,
        enable: formData.enable as 1 | 2,
        home_path: formData.home_path,
        email: formData.email || '',
        fei_shu_user_id: formData.fei_shu_user_id || '',
        avatar: formData.avatar || ''
      };
      await updateUserInfo(updateData);
      message.success('更新成功');
    }
    
    modalVisible.value = false;
    await fetchUserList();
    await fetchUserStatistics(); // 更新统计数据
  } catch (error: any) {
    if (!error.errorFields) {
      message.error(error.message || '操作失败');
    }
  } finally {
    submitLoading.value = false;
  }
};

const handleAssignRole = async (role: UserRole) => {
  try {
    await assignRolesToUserApi({
      user_id: currentUser.value!.id,
      role_ids: [role.id]
    });
    message.success('角色分配成功');
    await fetchUserRoles(currentUser.value!.id);
    await fetchUserList();
    await fetchUserStatistics(); // 角色变更后更新统计
  } catch (error: any) {
    message.error(error.message || '角色分配失败');
  }
};

const handleRevokeRole = async (role: UserRole) => {
  try {
    await revokeRolesFromUserApi({
      user_id: currentUser.value!.id,
      role_ids: [role.id]
    });
    message.success('角色移除成功');
    await fetchUserRoles(currentUser.value!.id);
    await fetchUserList();
    await fetchUserStatistics(); // 角色变更后更新统计
  } catch (error: any) {
    message.error(error.message || '角色移除失败');
  }
};

// 初始化
onMounted(() => {
  fetchUserList();
  fetchRoleList();
  fetchUserStatistics();
});
</script>

<style scoped>
/* 原有样式保持不变 */
.user-management {
  padding: 20px;
  background: #f5f5f5;
  min-height: 100vh;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
  padding: 20px;
  background: white;
  border-radius: 8px;
  border: 1px solid #d9d9d9;
}

.page-header h1 {
  margin: 0;
  font-size: 24px;
  font-weight: 600;
  color: #262626;
}

.header-actions {
  display: flex;
  gap: 12px;
}

.header-actions .ant-btn {
  display: flex;
  align-items: center;
  gap: 4px;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
  margin-bottom: 20px;
}

.stat-card {
  background: white;
  padding: 20px;
  border-radius: 8px;
  text-align: center;
  border: 1px solid #d9d9d9;
}

.stat-number {
  font-size: 28px;
  font-weight: 600;
  color: #1890ff;
  margin-bottom: 8px;
}

.stat-label {
  font-size: 14px;
  color: #8c8c8c;
}

.search-section {
  display: flex;
  justify-content: space-between;
  align-items: flex-end;
  gap: 16px;
  margin-bottom: 20px;
  padding: 20px;
  background: white;
  border-radius: 8px;
  border: 1px solid #d9d9d9;
}

.search-left {
  display: flex;
  gap: 12px;
  flex: 1;
  align-items: flex-end;
}

.search-input {
  flex: 1;
  max-width: 300px;
}

.status-select,
.type-select {
  width: 140px;
}

.search-right {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
}

.table-container {
  background: white;
  border-radius: 8px;
  border: 1px solid #d9d9d9;
  overflow: hidden;
}

.table-container :deep(.ant-table-thead > tr > th) {
  background: #fafafa;
  font-weight: 600;
  color: #262626;
  border-bottom: 1px solid #e8e8e8;
}

.table-container :deep(.ant-table-tbody > tr:hover > td) {
  background: #f5f5f5;
}

.user-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.user-avatar {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  background: #f0f0f0;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-weight: 600;
  overflow: hidden;
  flex-shrink: 0;
  position: relative;
  border: 1px solid rgba(0, 0, 0, 0.1);
}

.user-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.user-avatar:hover {
  transform: scale(1.05);
  transition: transform 0.2s ease;
}

.default-avatar {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  font-weight: 600;
  font-size: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
  border-radius: 50%;
}

.user-name {
  font-weight: 600;
  color: #262626;
  margin-bottom: 2px;
}

.user-username {
  font-size: 12px;
  color: #8c8c8c;
}

.contact-info {
  font-size: 13px;
  color: #595959;
  line-height: 1.4;
}

.action-buttons {
  display: flex;
  gap: 4px;
}

.avatar-upload {
  display: flex;
  align-items: flex-start;
  gap: 16px;
}

.avatar-preview {
  width: 80px;
  height: 80px;
  border-radius: 50%;
  border: 2px solid #d9d9d9;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #fafafa;
  overflow: hidden;
  flex-shrink: 0;
  position: relative;
}

.avatar-preview img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.avatar-preview:hover {
  border-color: #1890ff;
  transition: border-color 0.2s ease;
}

.avatar-upload-loading {
  position: relative;
}

.upload-loading {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(255, 255, 255, 0.8);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
}

.loading-icon {
  animation: rotate 1s linear infinite;
  font-size: 24px;
  color: #1890ff;
}

@keyframes rotate {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

.avatar-upload-actions {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.avatar-upload-actions .ant-btn {
  display: flex;
  align-items: center;
  gap: 6px;
}

.user-detail {
  padding: 8px 0;
}

.detail-section {
  margin-bottom: 24px;
}

.detail-section h3 {
  margin: 0 0 16px 0;
  font-size: 16px;
  font-weight: 600;
  color: #262626;
  padding-bottom: 8px;
  border-bottom: 1px solid #e8e8e8;
}

.detail-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
}

.detail-item {
  padding: 12px;
  background: #fafafa;
  border-radius: 6px;
  border: 1px solid #e8e8e8;
}

.detail-item label {
  display: block;
  font-size: 12px;
  color: #8c8c8c;
  font-weight: 600;
  margin-bottom: 4px;
}

.detail-item span {
  font-size: 14px;
  color: #262626;
}

.avatar-detail {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
}

.detail-avatar {
  width: 60px;
  height: 60px;
  border-radius: 50%;
  border: 2px solid #d9d9d9;
  overflow: hidden;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #fafafa;
  margin-top: 8px;
}

.detail-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.detail-avatar .default-avatar {
  font-size: 24px;
}

.roles-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.role-header {
  margin-bottom: 20px;
  padding-bottom: 12px;
  border-bottom: 1px solid #e8e8e8;
}

.role-header h4 {
  margin: 0;
  font-size: 16px;
  color: #262626;
  font-weight: 600;
}

.role-list {
  max-height: 400px;
  overflow-y: auto;
}

.empty-state {
  text-align: center;
  color: #8c8c8c;
  padding: 40px 0;
  background: #fafafa;
  border: 1px dashed #d9d9d9;
  border-radius: 6px;
}

.role-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  background: #fafafa;
  border: 1px solid #e8e8e8;
  border-radius: 6px;
  margin-bottom: 8px;
}

.role-item:hover {
  background: #f0f0f0;
}

.role-info {
  flex: 1;
}

.role-name {
  font-weight: 600;
  color: #262626;
  margin-bottom: 4px;
}

.role-desc {
  font-size: 12px;
  color: #8c8c8c;
}

.table-container :deep(.ant-btn-text) {
  color: #1890ff;
}

.table-container :deep(.ant-btn-text:hover) {
  color: #40a9ff;
  background: #f0f9ff;
}

.table-container :deep(.ant-btn-text.ant-btn-dangerous) {
  color: #ff4d4f;
}

.table-container :deep(.ant-btn-text.ant-btn-dangerous:hover) {
  color: #ff7875;
  background: #fff2f0;
}

@media (max-width: 1200px) {
  .search-section {
    flex-direction: column;
    align-items: stretch;
    gap: 16px;
  }
  
  .search-left {
    flex-direction: column;
    gap: 12px;
  }
  
  .search-input {
    max-width: none;
  }
  
  .status-select,
  .type-select {
    width: 100%;
  }
  
  .search-right {
    justify-content: flex-end;
  }
}

@media (max-width: 768px) {
  .user-management {
    padding: 12px;
  }
  
  .page-header {
    flex-direction: column;
    gap: 16px;
    text-align: center;
  }
  
  .header-actions {
    width: 100%;
    justify-content: center;
  }
  
  .stats-grid {
    grid-template-columns: 1fr;
  }
  
  .search-right {
    flex-direction: column;
    gap: 8px;
  }
  
  .detail-grid {
    grid-template-columns: 1fr;
  }
  
  .action-buttons {
    flex-direction: column;
    width: 100%;
  }
  
  .user-info {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }
  
  .avatar-upload {
    flex-direction: column;
    gap: 12px;
    text-align: center;
  }
  
  .avatar-preview {
    width: 60px;
    height: 60px;
  }
  
  .user-avatar {
    width: 32px;
    height: 32px;
  }
  
  .detail-avatar {
    width: 50px;
    height: 50px;
  }
}

.role-list::-webkit-scrollbar {
  width: 6px;
}

.role-list::-webkit-scrollbar-track {
  background: #f0f0f0;
  border-radius: 3px;
}

.role-list::-webkit-scrollbar-thumb {
  background: #d9d9d9;
  border-radius: 3px;
}

.role-list::-webkit-scrollbar-thumb:hover {
  background: #bfbfbf;
}
</style>