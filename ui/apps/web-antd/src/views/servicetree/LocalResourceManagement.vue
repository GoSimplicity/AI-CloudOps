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
              <a-select-option value="Running">在线</a-select-option>
              <a-select-option value="Stopped">离线</a-select-option>
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
          :pagination="pagination"
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
                {{ record.authMode === 'password' ? '密码认证' : '密钥认证' }}
              </a-tag>
            </template>
            <template v-if="column.key === 'treeNodeId'">
              {{ getTreeNodeName(record.tree_node_id) }}
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
                {{ currentDetail.instance_name || currentDetail.hostname }}
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
                  {{ currentDetail.authMode === 'password' ? '密码认证' : '密钥认证' }}
                </a-tag>
              </a-descriptions-item>
              <a-descriptions-item label="连接状态">
                <a-badge
                  :status="getStatusBadge(currentDetail.status)"
                  :text="getStatusText(currentDetail.status)"
                />
              </a-descriptions-item>
              <a-descriptions-item label="服务树节点">
                {{ getTreeNodeName(currentDetail.tree_node_id) }}
              </a-descriptions-item>
              <a-descriptions-item label="描述">
                {{ currentDetail.description || '-' }}
              </a-descriptions-item>
              <a-descriptions-item label="创建时间">
                {{ currentDetail.created_at }}
              </a-descriptions-item>
              <a-descriptions-item label="更新时间">
                {{ currentDetail.updated_at }}
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
  
  // 导入API方法
  import {
    getEcsResourceList,
    getEcsResourceDetail,
    createEcsResource,
    deleteEcsResource,
    getTreeList,
    type ResourceEcs,
    type TreeNodeListResp,
    type ListEcsResourceReq,
    type CreateEcsResourceReq,
    type DeleteEcsReq,
    type GetEcsDetailReq,
    type TreeNodeListReq
  } from '#/api/core/tree';
  
  // 接口定义
  interface LocalResource {
    id?: number;
    instanceType: string;
    imageName: string;
    hostname: string;
    password?: string;
    description?: string;
    ipAddr: string;
    port: number;
    osType: 'linux' | 'windows';
    treeNodeId?: number;
    tags: string[];
    authMode: 'password' | 'key';
    key?: string;
    status?: string;
    created_at?: string;
    updated_at?: string;
    // 从ResourceEcs继承的字段
    instance_name?: string;
    tree_node_id?: number;
    osName?: string;
  }
  
  // 响应式数据
  const loading = ref(false);
  const detailLoading = ref(false);
  const submitLoading = ref(false);
  const testLoading = ref(false);
  const modalVisible = ref(false);
  const detailVisible = ref(false);
  const isEdit = ref(false);
  const formRef = ref();
  
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
  const filterForm = reactive({
    osType: undefined,
    treeNodeId: undefined,
    keyword: '',
    status: undefined,
    provider: 'local' // 本地资源固定为local
  });
  
  // 本地资源列表
  const localResources = ref<LocalResource[]>([]);
  
  // 当前详情数据
  const currentDetail = ref<LocalResource | null>(null);
  
  // 服务树数据
  const treeData = ref<TreeNodeListResp[]>([]);
  
  // 表单数据
  const formData = reactive<LocalResource>({
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
  
  // 新标签输入
  const newTag = ref('');
  
  // 表格列定义
  const columns = [
    { title: '实例名称', dataIndex: 'instance_name', key: 'instance_name', width: 150 },
    { title: '主机名', dataIndex: 'hostname', key: 'hostname', width: 150 },
    { title: 'IP地址', dataIndex: 'ipAddr', key: 'ipAddr', width: 140 },
    { title: '端口', dataIndex: 'port', key: 'port', width: 80 },
    { title: '操作系统', dataIndex: 'osType', key: 'osType', width: 100 },
    { title: '实例类型', dataIndex: 'instanceType', key: 'instanceType', width: 130 },
    { title: '认证方式', dataIndex: 'authMode', key: 'authMode', width: 100 },
    { title: '连接状态', dataIndex: 'status', key: 'status', width: 100 },
    { title: '服务树', dataIndex: 'tree_node_id', key: 'treeNodeId', width: 120 },
    { title: '标签', dataIndex: 'tags', key: 'tags', width: 200, ellipsis: true },
    { title: '操作', key: 'action', fixed: 'right', width: 120 }
  ];
  
  // 表单验证规则计算属性
  const formRules = computed(() => ({
    instanceType: [{ required: true, message: '请输入实例类型' }],
    imageName: [{ required: true, message: '请输入镜像名称' }],
    hostname: [{ required: true, message: '请输入主机名' }],
    ipAddr: [
      { required: true, message: '请输入IP地址' },
      { pattern: /^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$/, message: 'IP地址格式不正确' }
    ],
    port: [{ required: true, message: '请输入SSH端口' }],
    osType: [{ required: true, message: '请选择操作系统类型' }],
    authMode: [{ required: true, message: '请选择认证方式' }],
    password: [
      { 
        required: formData.authMode === 'password', 
        message: '请输入登录密码' 
      }
    ],
    key: [
      { 
        required: formData.authMode === 'key', 
        message: '请输入SSH私钥' 
      }
    ]
  }));
  
  // 组件挂载时加载数据
  onMounted(() => {
    fetchLocalResources();
    fetchTreeData();
  });
  
  // 获取状态徽章类型
  const getStatusBadge = (status?: string) => {
    const statusMap: Record<string, string> = {
      'Running': 'success',
      'Stopped': 'error',
      'Starting': 'processing',
      'Stopping': 'warning'
    };
    return statusMap[status || 'Stopped'] || 'default';
  };
  
  // 获取状态文本
  const getStatusText = (status?: string) => {
    const statusMap: Record<string, string> = {
      'Running': '在线',
      'Stopped': '离线',
      'Starting': '启动中',
      'Stopping': '停止中'
    };
    return statusMap[status || 'Stopped'] || '未知';
  };
  
  // 获取服务树节点名称
  const getTreeNodeName = (treeNodeId?: number): string => {
    if (!treeNodeId) return '-';
    
    const findNode = (nodes: TreeNodeListResp[]): string => {
      for (const node of nodes) {
        if (node.id === treeNodeId) {
          return node.name;
        }
        if (node.children) {
          const result = findNode(node.children);
          if (result) return result;
        }
      }
      return '';
    };
    
    return findNode(treeData.value) || '-';
  };
  
  // 获取服务树数据
  const fetchTreeData = async () => {
    try {
      const req: TreeNodeListReq = {
        status: 'active'
      };
      const response = await getTreeList(req);
      treeData.value = response.items || [];
    } catch (error) {
      console.error('获取服务树数据失败:', error);
    }
  };
  
  // 获取本地资源列表
  const fetchLocalResources = async () => {
    loading.value = true;
    try {
      const req: ListEcsResourceReq = {
        page: pagination.current,
        size: pagination.pageSize,
        region: '',
        ...filterForm // 应用筛选条件
      };
  
      const response = await getEcsResourceList(req);
      
      // 转换数据格式
      localResources.value = (response.items || []).map((item: ResourceEcs) => ({
        id: item.id,
        instanceType: item.instanceType || 'local-server',
        imageName: item.osName || item.imageId || '-',
        hostname: item.hostname || item.instance_name,
        password: item.password,
        description: item.description,
        ipAddr: item.ipAddr || item.private_ip_address?.[0] || '-',
        port: item.port || 22,
        osType: (item.osType as 'linux' | 'windows') || 'linux',
        treeNodeId: item.tree_node_id,
        tags: item.tags || [],
        authMode: (item.authMode as 'password' | 'key') || 'password',
        key: item.key,
        status: item.status,
        created_at: item.created_at,
        updated_at: item.updated_at,
        instance_name: item.instance_name,
        tree_node_id: item.tree_node_id,
        osName: item.osName
      }));
      
      pagination.total = response.total || 0;
    } catch (error) {
      message.error('获取本地资源列表失败');
      console.error('获取本地资源列表失败:', error);
    } finally {
      loading.value = false;
    }
  };
  
  // 处理搜索
  const handleSearch = () => {
    pagination.current = 1;
    fetchLocalResources();
  };
  
  // 重置过滤条件
  const resetFilter = () => {
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
  
  // 处理表格变化
  const handleTableChange = (pag: any) => {
    pagination.current = pag.current;
    pagination.pageSize = pag.pageSize;
    fetchLocalResources();
  };
  
  // 显示创建模态框
  const showCreateModal = () => {
    isEdit.value = false;
    Object.assign(formData, {
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
    newTag.value = '';
    modalVisible.value = true;
  };
  
  // 处理编辑
  const handleEdit = (record: LocalResource) => {
    isEdit.value = true;
    Object.assign(formData, { ...record });
    modalVisible.value = true;
  };
  
  // 处理查看详情
  const handleViewDetail = async (record: LocalResource) => {
    detailVisible.value = true;
    detailLoading.value = true;
    currentDetail.value = record;
    
    try {
      // 如果有实例ID，获取详细信息
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
            ...response.data,
            hostname: response.data.hostname || record.hostname,
            ipAddr: response.data.ipAddr || record.ipAddr,
            port: response.data.port || record.port,
            osType: (response.data.osType as 'linux' | 'windows') || record.osType,
            authMode: (response.data.authMode as 'password' | 'key') || record.authMode
          };
        }
      }
    } catch (error) {
      console.error('获取资源详情失败:', error);
    } finally {
      detailLoading.value = false;
    }
  };
  
  // 处理认证方式变更
  const handleAuthModeChange = () => {
    formData.password = '';
    formData.key = '';
    // 触发表单验证更新
    formRef.value?.clearValidate(['password', 'key']);
  };
  
  // 添加标签
  const addTag = () => {
    const tagValue = newTag.value.trim();
    if (tagValue && !formData.tags.includes(tagValue)) {
      formData.tags.push(tagValue);
      newTag.value = '';
    } else if (!tagValue) {
      message.warning('请输入标签内容');
    } else {
      message.warning('标签已存在');
    }
  };
  
  // 移除标签
  const removeTag = (index: number) => {
    formData.tags.splice(index, 1);
  };
  
  // 测试单个连接
  const handleTestSingleConnection = async (record: LocalResource) => {
    const hide = message.loading(`正在测试 ${record.hostname} 的连接...`, 0);
    
    try {
      // 模拟连接测试 - 实际项目中应该调用真实的连接测试接口
      await new Promise(resolve => setTimeout(resolve, 2000));
      
      // 这里可以添加真实的连接测试逻辑
      const success = Math.random() > 0.3;
      
      if (success) {
        message.success(`${record.hostname} 连接测试成功`);
        // 更新本地状态
        const index = localResources.value.findIndex(item => item.id === record.id);
        if (index !== -1) {
          if (localResources.value[index]) {
            localResources.value[index].status = 'Running';
          }
        }
        if (currentDetail.value && currentDetail.value.id === record.id) {
          currentDetail.value.status = 'Running';
        }
      } else {
        message.error(`${record.hostname} 连接测试失败`);
        const index = localResources.value.findIndex(item => item.id === record.id);
        if (index !== -1 && localResources.value[index]) {
          localResources.value[index].status = 'Stopped';
        }
        if (currentDetail.value && currentDetail.value.id === record.id) {
          currentDetail.value.status = 'Stopped';
        }
      }
    } catch (error) {
      message.error(`${record.hostname} 连接测试失败`);
    } finally {
      hide();
    }
  };
  
  // 批量测试连接
  const handleTestConnection = async () => {
    if (localResources.value.length === 0) {
      message.warning('没有可测试的服务器');
      return;
    }
    
    const hide = message.loading('正在批量测试连接，请稍候...', 0);
    
    try {
      // 这里应该调用批量连接测试接口
      await new Promise(resolve => setTimeout(resolve, 3000));
      
      // 模拟随机更新状态
      localResources.value.forEach(resource => {
        resource.status = Math.random() > 0.2 ? 'Running' : 'Stopped';
      });
      
      const onlineCount = localResources.value.filter(r => r.status === 'Running').length;
      const totalCount = localResources.value.length;
      
      message.success(`批量测试完成，${onlineCount}/${totalCount} 台服务器在线`);
    } catch (error) {
      message.error('批量测试连接失败');
    } finally {
      hide();
    }
  };
  
  // 测试并提交
  const handleTestAndSubmit = async () => {
    try {
      await formRef.value?.validate();
      
      testLoading.value = true;
      
      // 先测试连接
      const testHide = message.loading(`正在测试 ${formData.hostname} 的连接...`, 0);
      
      try {
        // 这里应该调用真实的连接测试接口
        await new Promise(resolve => setTimeout(resolve, 2000));
        testHide();
        
        const testSuccess = Math.random() > 0.2;
        
        if (!testSuccess) {
          message.error('连接测试失败，请检查服务器配置');
          return;
        }
        
        message.success('连接测试成功，正在添加服务器...');
        
        // 测试成功后提交
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
  
  // 提交表单
  const handleSubmit = async () => {
    try {
      await formRef.value?.validate();
      
      submitLoading.value = true;
      
      if (isEdit.value) {
        // 更新现有资源 - 这里需要调用更新接口
        message.success('服务器信息更新成功');
        modalVisible.value = false;
        fetchLocalResources();
      } else {
        // 创建新资源
        const createReq: CreateEcsResourceReq = {
          // 基础参数
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
          tags: formData.tags || []
        };
  
        // 添加本地服务器特有的参数
        const localCreateReq = {
          ...createReq,
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
          instanceName: formData.hostname // 使用主机名作为实例名称
        };
  
        await createEcsResource(localCreateReq as any);
        message.success('服务器添加成功');
        modalVisible.value = false;
        fetchLocalResources();
      }
    } catch (error) {
      message.error(isEdit.value ? '更新服务器失败' : '添加服务器失败');
      console.error('提交表单失败:', error);
    } finally {
      submitLoading.value = false;
    }
  };
  
  // 删除资源
  const handleDelete = (record: LocalResource) => {
    Modal.confirm({
      title: '确定要删除此服务器吗？',
      content: `您正在删除服务器: ${record.hostname} (${record.ipAddr})，该操作不可恢复。`,
      okText: '确认删除',
      okType: 'danger',
      cancelText: '取消',
      async onOk() {
        try {
          if (record.id) {
            const deleteReq: DeleteEcsReq = {
              provider: 'local',
              region: 'local',
              instanceId: record.id.toString()
            };
            
            await deleteEcsResource(deleteReq);
            message.success('服务器删除成功');
            
            // 如果详情抽屉显示的是被删除的记录，关闭抽屉
            if (detailVisible.value && currentDetail.value?.id === record.id) {
              detailVisible.value = false;
            }
            
            fetchLocalResources();
          }
        } catch (error) {
          message.error('删除服务器失败');
          console.error('删除服务器失败:', error);
        }
      }
    });
  };
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