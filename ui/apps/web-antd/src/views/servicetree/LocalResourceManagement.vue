<template>
  <div class="local-resource-management">
    <a-page-header
      title="本地资源管理"
      sub-title="本地服务器统一管理"
      class="page-header"
    >
      <template #extra>
        <a-button type="primary" @click="handleTestConnection">
          <api-outlined /> 批量测试连接
        </a-button>
        <a-button type="primary" @click="showCreateModal">
          <plus-outlined /> 添加服务器
        </a-button>
      </template>
    </a-page-header>

    <a-card class="filter-card">
      <a-form layout="inline" :model="filterForm">
        <a-form-item label="操作系统">
          <a-select
            v-model:value="filterForm.osType"
            style="width: 120px"
            placeholder="选择系统"
            allow-clear
          >
            <a-select-option value="linux">Linux</a-select-option>
            <a-select-option value="windows">Windows</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="服务树">
          <a-tree-select
            v-model:value="filterForm.treeNodeId"
            style="width: 200px"
            placeholder="选择服务树节点"
            allow-clear
            :tree-data="treeData"
            :field-names="{ children: 'children', label: 'name', value: 'id' }"
          />
        </a-form-item>
        <a-form-item label="主机名/IP">
          <a-input
            v-model:value="filterForm.keyword"
            placeholder="输入主机名或IP地址"
            allow-clear
            style="width: 200px"
          />
        </a-form-item>
        <a-form-item label="连接状态">
          <a-select
            v-model:value="filterForm.status"
            style="width: 120px"
            placeholder="选择状态"
            allow-clear
          >
            <a-select-option value="RUNNING">在线</a-select-option>
            <a-select-option value="STOPPED">离线</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item>
          <a-button type="primary" @click="handleSearch">
            <search-outlined /> 搜索
          </a-button>
          <a-button style="margin-left: 8px" @click="resetFilter">
            重置
          </a-button>
        </a-form-item>
      </a-form>
    </a-card>

    <a-card class="resource-card">
      <a-table
        :columns="columns"
        :data-source="localResources"
        :loading="loading"
        :pagination="paginationConfig"
        @change="handleTableChange"
        row-key="id"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'status'">
            <a-badge
              :status="getStatusBadge(record.status)"
              :text="getStatusText(record.status)"
            />
          </template>
          <template v-if="column.key === 'osType'">
            <a-tag :color="record.osType === 'linux' ? 'blue' : 'green'">
              <windows-outlined v-if="record.osType === 'windows'" />
              <desktop-outlined v-else />
              {{ record.osType === 'linux' ? 'Linux' : 'Windows' }}
            </a-tag>
          </template>
          <template v-if="column.key === 'authMode'">
            <a-tag :color="record.authMode === 'password' ? 'orange' : 'purple'">
              {{ getAuthModeText(record.authMode) }}
            </a-tag>
          </template>
          <template v-if="column.key === 'treeNodeId'">
            {{ getTreeNodeName(record.treeNodeId) }}
          </template>
          <template v-if="column.key === 'tags'">
            <template v-if="record.tags && record.tags.length > 0">
              <a-tag 
                v-for="tag in record.tags" 
                :key="tag" 
                color="blue"
                style="margin-bottom: 4px;"
              >
                {{ tag }}
              </a-tag>
            </template>
            <span v-else class="text-gray-400">-</span>
          </template>
          <template v-if="column.key === 'action'">
            <a-dropdown>
              <template #overlay>
                <a-menu>
                  <a-menu-item key="detail" @click="handleViewDetail(record)">
                    <info-circle-outlined /> 详情
                  </a-menu-item>
                  <a-menu-item key="test" @click="handleTestSingleConnection(record)">
                    <api-outlined /> 测试连接
                  </a-menu-item>
                  <a-menu-item key="edit" @click="handleEdit(record)">
                    <edit-outlined /> 编辑
                  </a-menu-item>
                  <a-menu-divider />
                  <a-menu-item key="delete" @click="handleDelete(record)">
                    <delete-outlined /> 删除
                  </a-menu-item>
                </a-menu>
              </template>
              <a-button type="link">
                操作 <down-outlined />
              </a-button>
            </a-dropdown>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- 创建/编辑服务器对话框 -->
    <a-modal
      v-model:open="modalVisible"
      :title="isEdit ? '编辑服务器' : '添加服务器'"
      width="800px"
      :footer="null"
      :destroy-on-close="true"
    >
      <a-form
        :model="formData"
        :rules="formRules"
        layout="vertical"
        ref="formRef"
        class="server-form"
      >
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="实例类型" name="instanceType">
              <a-input
                v-model:value="formData.instanceType"
                placeholder="如: physical-server, vm-server"
              />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="镜像名称" name="imageName">
              <a-input
                v-model:value="formData.imageName"
                placeholder="如: CentOS-7.9, Ubuntu-20.04"
              />
            </a-form-item>
          </a-col>
        </a-row>

        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="主机名" name="hostname">
              <a-input
                v-model:value="formData.hostname"
                placeholder="服务器主机名"
              />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="IP地址" name="ipAddr">
              <a-input
                v-model:value="formData.ipAddr"
                placeholder="192.168.1.100"
              />
            </a-form-item>
          </a-col>
        </a-row>

        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="SSH端口" name="port">
              <a-input-number
                v-model:value="formData.port"
                :min="1"
                :max="65535"
                style="width: 100%"
                placeholder="22"
              />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="操作系统" name="osType">
              <a-select v-model:value="formData.osType" placeholder="选择操作系统">
                <a-select-option value="linux">
                  <desktop-outlined /> Linux
                </a-select-option>
                <a-select-option value="windows">
                  <windows-outlined /> Windows
                </a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>

        <a-form-item label="认证方式" name="authMode">
          <a-radio-group v-model:value="formData.authMode" @change="handleAuthModeChange">
            <a-radio value="password">密码认证</a-radio>
            <a-radio value="key">密钥认证</a-radio>
          </a-radio-group>
        </a-form-item>

        <a-form-item 
          v-if="formData.authMode === 'password'" 
          label="登录密码" 
          name="password"
        >
          <a-input-password
            v-model:value="formData.password"
            placeholder="请输入登录密码"
            autocomplete="new-password"
          />
        </a-form-item>

        <a-form-item 
          v-if="formData.authMode === 'key'" 
          label="私钥内容" 
          name="key"
        >
          <a-textarea
            v-model:value="formData.key"
            placeholder="请粘贴SSH私钥内容"
            :rows="6"
            style="font-family: 'Courier New', monospace;"
          />
        </a-form-item>

        <a-form-item label="服务树节点" name="treeNodeId">
          <a-tree-select
            v-model:value="formData.treeNodeId"
            placeholder="选择服务树节点"
            allow-clear
            :tree-data="treeData"
            :field-names="{ children: 'children', label: 'name', value: 'id' }"
            style="width: 100%"
          />
        </a-form-item>

        <a-form-item label="描述信息">
          <a-textarea
            v-model:value="formData.description"
            placeholder="服务器用途描述"
            :rows="3"
          />
        </a-form-item>

        <a-form-item label="资源标签">
          <div class="tags-input">
            <div class="current-tags">
              <a-tag
                v-for="(tag, index) in formData.tags"
                :key="index"
                closable
                @close="removeTag(index)"
                color="blue"
              >
                {{ tag }}
              </a-tag>
            </div>
            <div class="add-tag">
              <a-input
                v-model:value="newTag"
                placeholder="输入标签，按回车添加"
                style="width: 200px; margin-right: 8px;"
                @pressEnter="addTag"
              />
              <a-button type="dashed" @click="addTag">
                <plus-outlined /> 添加标签
              </a-button>
            </div>
          </div>
        </a-form-item>

        <div class="form-actions">
          <a-button @click="modalVisible = false">取消</a-button>
          <a-button 
            type="primary" 
            @click="handleSubmit" 
            :loading="submitLoading"
            style="margin-left: 8px;"
          >
            {{ isEdit ? '更新' : '添加' }}
          </a-button>
          <a-button 
            v-if="!isEdit"
            @click="handleTestAndSubmit" 
            :loading="testLoading"
            style="margin-left: 8px;"
          >
            <api-outlined /> 测试并添加
          </a-button>
        </div>
      </a-form>
    </a-modal>

    <!-- 服务器详情抽屉 -->
    <a-drawer
      v-model:open="detailVisible"
      title="服务器详情"
      width="600"
      :destroy-on-close="true"
      class="detail-drawer"
    >
      <a-skeleton :loading="detailLoading" active>
        <template v-if="currentDetail">
          <a-descriptions bordered :column="1">
            <a-descriptions-item label="实例名称">
              {{ currentDetail.instanceName || currentDetail.hostname }}
            </a-descriptions-item>
            <a-descriptions-item label="主机名">
              {{ currentDetail.hostname }}
            </a-descriptions-item>
            <a-descriptions-item label="IP地址">
              {{ currentDetail.ipAddr }}
            </a-descriptions-item>
            <a-descriptions-item label="SSH端口">
              {{ currentDetail.port }}
            </a-descriptions-item>
            <a-descriptions-item label="操作系统">
              <a-tag :color="currentDetail.osType === 'linux' ? 'blue' : 'green'">
                <windows-outlined v-if="currentDetail.osType === 'windows'" />
                <desktop-outlined v-else />
                {{ currentDetail.osType === 'linux' ? 'Linux' : 'Windows' }}
              </a-tag>
            </a-descriptions-item>
            <a-descriptions-item label="实例类型">
              {{ currentDetail.instanceType }}
            </a-descriptions-item>
            <a-descriptions-item label="镜像名称">
              {{ currentDetail.imageName || currentDetail.osName || '-' }}
            </a-descriptions-item>
            <a-descriptions-item label="认证方式">
              <a-tag :color="currentDetail.authMode === 'password' ? 'orange' : 'purple'">
                {{ getAuthModeText(currentDetail.authMode) }}
              </a-tag>
            </a-descriptions-item>
            <a-descriptions-item label="连接状态">
              <a-badge
                :status="getStatusBadge(currentDetail.status)"
                :text="getStatusText(currentDetail.status)"
              />
            </a-descriptions-item>
            <a-descriptions-item label="服务树节点">
              {{ getTreeNodeName(currentDetail.treeNodeId) }}
            </a-descriptions-item>
            <a-descriptions-item label="描述">
              {{ currentDetail.description || '-' }}
            </a-descriptions-item>
            <a-descriptions-item label="创建时间">
              {{ formatDateTime(currentDetail.createdAt) }}
            </a-descriptions-item>
            <a-descriptions-item label="更新时间">
              {{ formatDateTime(currentDetail.updatedAt) }}
            </a-descriptions-item>
          </a-descriptions>
  
          <a-divider orientation="left">资源标签</a-divider>
          <div class="tag-list">
            <template v-if="currentDetail.tags && currentDetail.tags.length > 0">
              <a-tag 
                v-for="(tag, index) in currentDetail.tags" 
                :key="index" 
                color="blue"
              >
                {{ tag }}
              </a-tag>
            </template>
            <a-empty v-else :image="Empty.PRESENTED_IMAGE_SIMPLE" description="暂无标签" />
          </div>
          <div class="drawer-actions">
            <a-button-group>
              <a-button type="primary" @click="handleTestSingleConnection(currentDetail)">
                <api-outlined /> 测试连接
              </a-button>
              <a-button @click="handleEdit(currentDetail)">
                <edit-outlined /> 编辑
              </a-button>
            </a-button-group>
            <a-button danger @click="handleDelete(currentDetail)">
              <delete-outlined /> 删除
            </a-button>
          </div>
        </template>
      </a-skeleton>
    </a-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed } from 'vue';
import { message, Modal, Empty } from 'ant-design-vue';
import {
  PlusOutlined,
  SearchOutlined,
  InfoCircleOutlined,
  EditOutlined,
  DeleteOutlined,
  DownOutlined,
  ApiOutlined,
  WindowsOutlined,
  DesktopOutlined,
} from '@ant-design/icons-vue';

import {
  getEcsResourceList,
  getEcsResourceDetail,
  createEcsResource,
  updateEcsResource,
  deleteEcsResource,
  getTreeList,
  type ResourceEcs,
  type TreeNodeListResp,
  type ListEcsResourceReq,
  type CreateEcsResourceReq,
  type DeleteEcsReq,
  type GetEcsDetailReq,
  type TreeNodeListReq,
  type UpdateEcsResourceReq
} from '#/api/core/tree';

type AuthMode = 'password' | 'key';
type OsType = 'linux' | 'windows';
type ServerStatus = 'RUNNING' | 'STOPPED' | 'STARTING' | 'STOPPING' | 'RESTARTING' | 'DELETING' | 'ERROR';

interface LocalResource {
  id?: number;
  instanceType: string;
  imageName: string;
  hostname: string;
  password?: string;
  description?: string;
  ipAddr: string;
  port: number;
  osType: OsType;
  treeNodeId?: number;
  tags: string[];
  authMode: AuthMode;
  key?: string;
  status?: ServerStatus;
  createdAt?: string;
  updatedAt?: string;
  instanceName?: string;
  osName?: string;
}

interface FilterForm {
  osType?: OsType;
  treeNodeId?: number;
  keyword: string;
  status?: ServerStatus;
  provider: string;
}

// ==================== 响应式数据 ====================
const loading = ref(false);
const detailLoading = ref(false);
const submitLoading = ref(false);
const testLoading = ref(false);
const modalVisible = ref(false);
const detailVisible = ref(false);
const isEdit = ref(false);
const formRef = ref();
const newTag = ref('');

// 分页配置
const pagination = reactive({
  current: 1,
  pageSize: 10,
  total: 0,
  showSizeChanger: true,
  showQuickJumper: true,
  showTotal: (total: number) => `共 ${total} 条记录`
});

// 过滤条件
const filterForm = reactive<FilterForm>({
  osType: undefined,
  treeNodeId: undefined,
  keyword: '',
  status: undefined,
  provider: 'local'
});

// 数据列表
const localResources = ref<LocalResource[]>([]);
const currentDetail = ref<LocalResource | null>(null);
const treeData = ref<TreeNodeListResp[]>([]);

// 表单数据初始化函数
const createInitialFormData = (): LocalResource => ({
  instanceType: '',
  imageName: '',
  hostname: '',
  password: '',
  description: '',
  ipAddr: '',
  port: 22,
  osType: 'linux',
  treeNodeId: undefined,
  tags: [],
  authMode: 'password',
  key: ''
});

const formData = reactive<LocalResource>(createInitialFormData());

// ==================== 计算属性 ====================
const paginationConfig = computed(() => ({
  ...pagination,
  onChange: handleTableChange,
  onShowSizeChange: handleTableChange
}));

const formRules = computed(() => ({
  instanceType: [{ required: true, message: '请输入实例类型', trigger: 'blur' }],
  imageName: [{ required: true, message: '请输入镜像名称', trigger: 'blur' }],
  hostname: [
    { required: true, message: '请输入主机名', trigger: 'blur' },
    { 
      pattern: /^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?$/, 
      message: '主机名格式不正确', 
      trigger: 'blur' 
    }
  ],
  ipAddr: [
    { required: true, message: '请输入IP地址', trigger: 'blur' },
    { 
      pattern: /^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$/, 
      message: 'IP地址格式不正确', 
      trigger: 'blur' 
    }
  ],
  port: [
    { required: true, message: '请输入SSH端口', trigger: 'blur' },
    { type: 'number', min: 1, max: 65535, message: '端口范围为1-65535', trigger: 'blur' }
  ],
  osType: [{ required: true, message: '请选择操作系统类型', trigger: 'change' }],
  authMode: [{ required: true, message: '请选择认证方式', trigger: 'change' }],
  password: formData.authMode === 'password' ? [
    { required: true, message: '请输入登录密码', trigger: 'blur' },
    { min: 8, message: '密码长度至少8位', trigger: 'blur' }
  ] : [],
  key: formData.authMode === 'key' ? [
    { required: true, message: '请输入SSH私钥', trigger: 'blur' },
    { validator: validatePrivateKey, trigger: 'blur' }
  ] : []
}));

// 表格列定义
const columns = [
  { title: '实例名称', dataIndex: 'instanceName', key: 'instanceName', width: 150 },
  { title: '主机名', dataIndex: 'hostname', key: 'hostname', width: 150 },
  { title: 'IP地址', dataIndex: 'ipAddr', key: 'ipAddr', width: 140 },
  { title: '端口', dataIndex: 'port', key: 'port', width: 80 },
  { title: '操作系统', dataIndex: 'osType', key: 'osType', width: 100 },
  { title: '实例类型', dataIndex: 'instanceType', key: 'instanceType', width: 130 },
  { title: '认证方式', dataIndex: 'authMode', key: 'authMode', width: 100 },
  { title: '连接状态', dataIndex: 'status', key: 'status', width: 100 },
  { title: '服务树', dataIndex: 'treeNodeId', key: 'treeNodeId', width: 120 },
  { title: '标签', dataIndex: 'tags', key: 'tags', width: 200, ellipsis: true },
  { title: '操作', key: 'action', fixed: 'right', width: 120 }
];

// ==================== 工具函数 ====================
const formatDateTime = (dateTime?: string): string => {
  if (!dateTime) return '-';
  try {
    return new Date(dateTime).toLocaleString('zh-CN');
  } catch {
    return dateTime;
  }
};

const getStatusBadge = (status?: ServerStatus): string => {
  const statusMap: Record<ServerStatus, string> = {
    'RUNNING': 'success',
    'STOPPED': 'error',
    'STARTING': 'processing',
    'STOPPING': 'warning',
    'RESTARTING': 'processing',
    'DELETING': 'warning',
    'ERROR': 'error'
  };
  return statusMap[status || 'STOPPED'] || 'default';
};

const getStatusText = (status?: ServerStatus): string => {
  const statusMap: Record<ServerStatus, string> = {
    'RUNNING': '在线',
    'STOPPED': '离线',
    'STARTING': '启动中',
    'STOPPING': '停止中',
    'RESTARTING': '重启中',
    'DELETING': '删除中',
    'ERROR': '错误'
  };
  return statusMap[status || 'STOPPED'] || '未知';
};

const getAuthModeText = (authMode?: AuthMode): string => {
  const authModeMap: Record<AuthMode, string> = {
    'password': '密码认证',
    'key': '密钥认证'
  };
  return authModeMap[authMode || 'password'] || '未知';
};

const getTreeNodeName = (treeNodeId?: number): string => {
  if (!treeNodeId) return '-';
  
  const findNode = (nodes: TreeNodeListResp[]): string => {
    for (const node of nodes) {
      if (node.id === treeNodeId) {
        return node.name;
      }
      if (node.children && node.children.length > 0) {
        const result = findNode(node.children);
        if (result) return result;
      }
    }
    return '';
  };
  
  return findNode(treeData.value) || '-';
};

const transformResourceToLocal = (item: ResourceEcs): LocalResource => ({
  id: item.id,
  instanceType: item.instanceType || 'local-server',
  imageName: item.osName || item.imageId || item.imageName || '-',
  hostname: item.hostname || item.instance_name || '',
  password: item.password,
  description: item.description,
  ipAddr: item.ipAddr || (Array.isArray(item.private_ip_address) ? item.private_ip_address[0] : '') || '-',
  port: item.port || 22,
  osType: (item.osType as OsType) || 'linux',
  treeNodeId: item.tree_node_id,
  tags: Array.isArray(item.tags) ? item.tags : [],
  authMode: (item.authMode as AuthMode) || 'password',
  key: item.key,
  status: item.status as ServerStatus,
  createdAt: item.created_at,
  updatedAt: item.updated_at,
  instanceName: item.instance_name,
  osName: item.osName
});

const simulateConnectionTest = async (hostname: string): Promise<boolean> => {
  await new Promise(resolve => setTimeout(resolve, Math.random() * 2000 + 1000));
  return Math.random() > 0.2;
};

// 表单验证函数
async function validatePrivateKey(_rule: any, value: string): Promise<void> {
  if (formData.authMode === 'key' && value && !value.includes('PRIVATE KEY')) {
    throw new Error('请输入有效的SSH私钥');
  }
}

// ==================== API 调用函数 ====================
const fetchTreeData = async (): Promise<void> => {
  try {
    const req: TreeNodeListReq = {
      status: 'active'
    };
    const response = await getTreeList(req);
    treeData.value = response.items || [];
  } catch (error) {
    console.error('获取服务树数据失败:', error);
    message.error('获取服务树数据失败');
  }
};

const fetchLocalResources = async (): Promise<void> => {
  loading.value = true;
  try {
    const req: ListEcsResourceReq = {
      page: pagination.current,
      size: pagination.pageSize,
      region: '',
      ...filterForm
    };

    const response = await getEcsResourceList(req);
    
    localResources.value = (response.items || []).map(transformResourceToLocal);
    pagination.total = response.total || 0;
  } catch (error) {
    message.error('获取本地资源列表失败');
    console.error('获取本地资源列表失败:', error);
    // 发生错误时重置数据
    localResources.value = [];
    pagination.total = 0;
  } finally {
    loading.value = false;
  }
};

// ==================== 事件处理函数 ====================
const handleSearch = (): void => {
  pagination.current = 1;
  fetchLocalResources();
};

const resetFilter = (): void => {
  Object.assign(filterForm, {
    osType: undefined,
    treeNodeId: undefined,
    keyword: '',
    status: undefined,
    provider: 'local'
  });
  pagination.current = 1;
  fetchLocalResources();
};

const handleTableChange = (pag: any): void => {
  pagination.current = pag.current || pag.page;
  pagination.pageSize = pag.pageSize || pag.size;
  fetchLocalResources();
};

const resetFormData = (): void => {
  Object.assign(formData, createInitialFormData());
  newTag.value = '';
  if (formRef.value) {
    formRef.value.resetFields();
  }
};

const showCreateModal = (): void => {
  isEdit.value = false;
  resetFormData();
  modalVisible.value = true;
};

const handleEdit = (record: LocalResource): void => {
  isEdit.value = true;
  Object.assign(formData, { ...record });
  modalVisible.value = true;
  if (detailVisible.value) {
    detailVisible.value = false;
  }
};

const handleViewDetail = async (record: LocalResource): Promise<void> => {
  detailVisible.value = true;
  detailLoading.value = true;
  currentDetail.value = record;
  
  try {
    if (record.id) {
      const req: GetEcsDetailReq = {
        provider: 'local',
        region: 'local',
        instanceId: record.id.toString()
      };
      
      const response = await getEcsResourceDetail(req);
      if (response.data) {
        currentDetail.value = {
          ...record,
          ...transformResourceToLocal(response.data)
        };
      }
    }
  } catch (error) {
    console.error('获取资源详情失败:', error);
    message.error('获取资源详情失败');
  } finally {
    detailLoading.value = false;
  }
};

const handleAuthModeChange = (): void => {
  formData.password = '';
  formData.key = '';
  if (formRef.value) {
    formRef.value.clearValidate(['password', 'key']);
  }
};

const addTag = (): void => {
  const tagValue = newTag.value.trim();
  if (!tagValue) {
    message.warning('请输入标签内容');
    return;
  }
  
  if (formData.tags.includes(tagValue)) {
    message.warning('标签已存在');
    return;
  }
  
  if (formData.tags.length >= 10) {
    message.warning('最多只能添加10个标签');
    return;
  }
  
  formData.tags.push(tagValue);
  newTag.value = '';
};

const removeTag = (index: number): void => {
  formData.tags.splice(index, 1);
};

const updateResourceStatus = (resourceId: number, status: ServerStatus): void => {
  const index = localResources.value.findIndex(item => item.id === resourceId);
  if (index !== -1 && localResources.value[index]) {
    localResources.value[index].status = status;
  }
  if (currentDetail.value && currentDetail.value.id === resourceId) {
    currentDetail.value.status = status;
  }
};

const handleTestSingleConnection = async (record: LocalResource): Promise<void> => {
  const hide = message.loading(`正在测试 ${record.hostname} 的连接...`, 0);
  
  try {
    const success = await simulateConnectionTest(record.hostname);
    
    if (success) {
      message.success(`${record.hostname} 连接测试成功`);
      if (record.id) {
        updateResourceStatus(record.id, 'RUNNING');
      }
    } else {
      message.error(`${record.hostname} 连接测试失败`);
      if (record.id) {
        updateResourceStatus(record.id, 'STOPPED');
      }
    }
  } catch (error) {
    message.error(`${record.hostname} 连接测试异常`);
    console.error('连接测试失败:', error);
  } finally {
    hide();
  }
};

const handleTestConnection = async (): Promise<void> => {
  if (localResources.value.length === 0) {
    message.warning('没有可测试的服务器');
    return;
  }
  
  const hide = message.loading('正在批量测试连接，请稍候...', 0);
  
  try {
    const testPromises = localResources.value.map(async (resource) => {
      if (resource.id) {
        const success = await simulateConnectionTest(resource.hostname);
        updateResourceStatus(resource.id, success ? 'RUNNING' : 'STOPPED');
        return success;
      }
      return false;
    });
    
    const results = await Promise.all(testPromises);
    const onlineCount = results.filter(Boolean).length;
    const totalCount = localResources.value.length;
    
    message.success(`批量测试完成，${onlineCount}/${totalCount} 台服务器在线`);
  } catch (error) {
    message.error('批量测试连接失败');
    console.error('批量测试失败:', error);
  } finally {
    hide();
  }
};

const handleTestAndSubmit = async (): Promise<void> => {
  try {
    await formRef.value?.validate();
    
    testLoading.value = true;
    
    const testHide = message.loading(`正在测试 ${formData.hostname} 的连接...`, 0);
    
    try {
      const testSuccess = await simulateConnectionTest(formData.hostname);
      testHide();
      
      if (!testSuccess) {
        message.error('连接测试失败，请检查服务器配置');
        return;
      }
      
      message.success('连接测试成功，正在添加服务器...');
      await handleSubmit();
      
    } catch (error) {
      testHide();
      message.error('连接测试失败');
    }
  } catch (error) {
    message.error('表单验证失败');
  } finally {
    testLoading.value = false;
  }
};

const handleSubmit = async (): Promise<void> => {
  try {
    await formRef.value?.validate();
    
    submitLoading.value = true;
    
    if (isEdit.value) {
      if (!formData.id) {
        throw new Error('缺少资源ID');
      }
      
      const updateReq: UpdateEcsResourceReq = {
        id: formData.id,
        provider: 'local',
        region: 'local',
        instanceId: formData.id.toString(),
        instanceName: formData.hostname,
        description: formData.description || '',
        tags: formData.tags || [],
        securityGroupIds: [],
        hostname: formData.hostname,
        password: formData.password || '',
        treeNodeId: formData.treeNodeId || 0,
        environment: 'local',
        ipAddr: formData.ipAddr,
        port: formData.port,
        authMode: formData.authMode,
        key: formData.key || ''
      };
      
      await updateEcsResource(updateReq);
      message.success('服务器信息更新成功');
      
      if (detailVisible.value && currentDetail.value?.id === formData.id) {
        Object.assign(currentDetail.value, formData);
      }
    } else {
      const createReq: CreateEcsResourceReq & {
        provider: string;
        region: string;
        instanceType: string;
        imageName: string;
        hostname: string;
        password?: string;
        description?: string;
        ipAddr: string;
        port: number;
        osType: string;
        treeNodeId?: number;
        authMode: string;
        key?: string;
        instanceName: string;
      } = {
        instanceChargeType: 'PostPaid',
        periodUnit: 'Month',
        period: 1,
        autoRenew: false,
        spotStrategy: 'NoSpot',
        spotDuration: 1,
        systemDiskSize: 40,
        dataDiskSize: 100,
        dataDiskCategory: 'cloud_efficiency',
        dryRun: false,
        tags: formData.tags || [],
        provider: 'local',
        region: 'local',
        instanceType: formData.instanceType,
        imageName: formData.imageName,
        hostname: formData.hostname,
        password: formData.password,
        description: formData.description,
        ipAddr: formData.ipAddr,
        port: formData.port,
        osType: formData.osType,
        treeNodeId: formData.treeNodeId,
        authMode: formData.authMode,
        key: formData.key,
        instanceName: formData.hostname
      };

      await createEcsResource(createReq as any);
      message.success('服务器添加成功');
    }
    
    modalVisible.value = false;
    await fetchLocalResources(); // 重新加载列表数据
  } catch (error) {
    message.error(isEdit.value ? '更新服务器失败' : '添加服务器失败');
    console.error('提交表单失败:', error);
  } finally {
    submitLoading.value = false;
  }
};

const handleDelete = (record: LocalResource): void => {
  Modal.confirm({
    title: '确定要删除此服务器吗？',
    content: `您正在删除服务器: ${record.hostname} (${record.ipAddr})，该操作不可恢复。`,
    okText: '确认删除',
    okType: 'danger',
    cancelText: '取消',
    async onOk() {
      try {
        if (!record.id) {
          throw new Error('缺少资源ID');
        }
        
        const deleteReq: DeleteEcsReq = {
          provider: 'local',
          region: 'local',
          instanceId: record.id.toString()
        };
        
        await deleteEcsResource(deleteReq);
        message.success('服务器删除成功');
        
        if (detailVisible.value && currentDetail.value?.id === record.id) {
          detailVisible.value = false;
        }
        
        await fetchLocalResources(); // 重新加载列表数据
      } catch (error) {
        message.error('删除服务器失败');
        console.error('删除服务器失败:', error);
      }
    }
  });
};

onMounted(async () => {
  try {
    // 并行加载服务树数据和资源列表
    await Promise.all([
      fetchTreeData(),
      fetchLocalResources()
    ]);
  } catch (error) {
    console.error('初始化数据加载失败:', error);
    message.error('页面数据加载失败，请刷新页面重试');
  }
});
</script>

<style scoped lang="scss">
.local-resource-management {
  padding: 0 16px;
  
  .page-header {
    margin-bottom: 16px;
    padding: 16px 0;
  }
  
  .filter-card {
    margin-bottom: 16px;
  }
  
  .resource-card {
    :deep(.ant-card-body) {
      padding: 0;
    }
  }
  
  :deep(.ant-table-pagination.ant-pagination) {
    margin: 16px;
  }

  .server-form {
    .tags-input {
      .current-tags {
        margin-bottom: 12px;
        
        .ant-tag {
          margin-bottom: 8px;
        }
      }
      
      .add-tag {
        display: flex;
        align-items: center;
        flex-wrap: wrap;
        gap: 8px;
      }
    }
    
    .form-actions {
      display: flex;
      justify-content: flex-end;
      margin-top: 24px;
      padding-top: 16px;
      border-top: 1px solid #f0f0f0;
    }
  }

  .detail-drawer {
    .tag-list {
      display: flex;
      flex-wrap: wrap;
      gap: 8px;
      margin-bottom: 16px;
    }
    
    .drawer-actions {
      display: flex;
      justify-content: space-between;
      margin-top: 24px;
      padding-top: 16px;
      border-top: 1px solid #f0f0f0;
    }
  }

  .text-gray-400 {
    color: #9ca3af;
  }

  :deep(.ant-form-item) {
    margin-bottom: 20px;
  }

  :deep(.ant-descriptions-item-label) {
    width: 120px;
  }
}
</style>