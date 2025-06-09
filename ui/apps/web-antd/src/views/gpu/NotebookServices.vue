<template>
    <div class="notebook-service-page">
      <!-- 页面标题区域 -->
      <div class="page-header">
        <h2 class="page-title">Notebook服务</h2>
      </div>
  
      <!-- 查询和操作工具栏 -->
      <div class="dashboard-card custom-toolbar">
        <div class="search-filters">
          <a-input 
            v-model:value="searchText" 
            placeholder="请输入Notebook名称" 
            class="search-input"
          >
            <template #prefix>
              <SearchOutlined class="search-icon" />
            </template>
          </a-input>
          <a-select 
            v-model:value="statusFilter" 
            placeholder="运行状态" 
            class="status-filter"
            allowClear
          >
            <a-select-option value="">全部状态</a-select-option>
            <a-select-option value="Creating">创建中</a-select-option>
            <a-select-option value="Running">运行中</a-select-option>
            <a-select-option value="Stopped">已停止</a-select-option>
            <a-select-option value="Failed">失败</a-select-option>
            <a-select-option value="Deleting">删除中</a-select-option>
          </a-select>
          <a-select 
            v-model:value="typeFilter" 
            placeholder="Notebook类型" 
            class="type-filter"
            allowClear
          >
            <a-select-option value="">全部类型</a-select-option>
            <a-select-option value="jupyter">Jupyter</a-select-option>
            <a-select-option value="vscode">VS Code</a-select-option>
            <a-select-option value="rstudio">RStudio</a-select-option>
          </a-select>
          <a-button type="primary" class="action-button" @click="handleSearch">
            <template #icon>
              <SearchOutlined />
            </template>
            搜索
          </a-button>
          <a-button class="action-button reset-button" @click="handleReset">
            <template #icon>
              <ReloadOutlined />
            </template>
            重置
          </a-button>
        </div>
        <div class="action-buttons">
          <a-button type="primary" class="add-button" @click="showAddModal">
            <template #icon>
              <PlusOutlined />
            </template>
            创建Notebook
          </a-button>
        </div>
      </div>
  
      <!-- Notebook列表表格 -->
      <div class="dashboard-card table-container">
        <a-table 
          :columns="columns" 
          :data-source="data" 
          row-key="id" 
          :pagination="false"
          class="custom-table"
          :scroll="{ x: 1400 }"
        >
          <!-- 运行状态列 -->
          <template #status="{ record }">
            <a-tag :color="getStatusColor(record.status)" class="status-tag">
              {{ getStatusText(record.status) }}
            </a-tag>
          </template>
          
          <!-- 资源配置列 -->
          <template #resources="{ record }">
            <div class="resource-container">
              <div class="resource-item">
                <span class="resource-label">CPU:</span>
                <span class="resource-value">{{ record.cpu_limit }}</span>
              </div>
              <div class="resource-item">
                <span class="resource-label">内存:</span>
                <span class="resource-value">{{ record.memory_limit }}</span>
              </div>
              <div class="resource-item">
                <span class="resource-label">GPU:</span>
                <span class="resource-value">{{ record.gpu_limit }}</span>
              </div>
            </div>
          </template>
          
          <!-- 镜像列 -->
          <template #image="{ record }">
            <a-tooltip :title="record.image">
              <div class="image-container">
                {{ record.image.split('/').pop() }}
              </div>
            </a-tooltip>
          </template>
          
          <!-- Notebook类型列 -->
          <template #type="{ record }">
            <a-tag :color="getTypeColor(record.type)" class="type-tag">
              {{ getTypeText(record.type) }}
            </a-tag>
          </template>
          
          <!-- 运行时间列 -->
          <template #duration="{ record }">
            <div class="duration-container">
              {{ formatDuration(record.start_time) }}
            </div>
          </template>
  
          <!-- 访问地址列 -->
          <template #access="{ record }">
            <div class="access-container" v-if="record.status === 'Running'">
              <a-button type="link" size="small" @click="openNotebook(record)" class="access-link">
                <template #icon>
                  <LinkOutlined />
                </template>
                访问
              </a-button>
            </div>
            <span v-else class="access-disabled">未启动</span>
          </template>
          
          <!-- 操作列 -->
          <template #action="{ record }">
            <div class="action-column">
              <a-button type="primary" size="small" @click="handleView(record)">
                查看
              </a-button>
              <a-button type="default" size="small" @click="handleEdit(record)" v-if="['Stopped', 'Failed'].includes(record.status)">
                编辑
              </a-button>
              <a-button type="default" size="small" @click="handleStart(record)" v-if="record.status === 'Stopped'">
                启动
              </a-button>
              <a-button type="default" size="small" @click="handleStop(record)" v-if="record.status === 'Running'">
                停止
              </a-button>
              <a-button type="default" size="small" @click="handleDelete(record)" v-if="['Stopped', 'Failed'].includes(record.status)">
                删除
              </a-button>
            </div>
          </template>
        </a-table>
  
        <!-- 分页器 -->
        <div class="pagination-container">
          <a-pagination 
            v-model:current="current" 
            v-model:pageSize="pageSizeRef" 
            :page-size-options="pageSizeOptions"
            :total="total" 
            show-size-changer 
            @change="handlePageChange" 
            @showSizeChange="handleSizeChange" 
            class="custom-pagination"
          >
            <template #buildOptionText="props">
              <span v-if="props.value !== '50'">{{ props.value }}条/页</span>
              <span v-else>全部</span>
            </template>
          </a-pagination>
        </div>
      </div>
  
      <!-- 创建Notebook模态框 -->
      <a-modal 
        title="创建Notebook服务" 
        v-model:visible="isAddModalVisible" 
        @ok="handleAdd" 
        @cancel="closeAddModal"
        :width="800"
        class="custom-modal"
      >
        <a-form ref="addFormRef" :model="addForm" layout="vertical" class="custom-form">
          <div class="form-section">
            <div class="section-title">基本信息</div>
            <a-row :gutter="16">
              <a-col :span="12">
                <a-form-item label="Notebook名称" name="name" :rules="[{ required: true, message: '请输入Notebook名称' }]">
                  <a-input v-model:value="addForm.name" placeholder="请输入Notebook名称" />
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item label="Notebook类型" name="type" :rules="[{ required: true, message: '请选择Notebook类型' }]">
                  <a-select v-model:value="addForm.type" placeholder="请选择Notebook类型">
                    <a-select-option value="jupyter">Jupyter</a-select-option>
                    <a-select-option value="vscode">VS Code</a-select-option>
                    <a-select-option value="rstudio">RStudio</a-select-option>
                  </a-select>
                </a-form-item>
              </a-col>
            </a-row>
            <a-row :gutter="16">
              <a-col :span="24">
                <a-form-item label="描述信息" name="description">
                  <a-textarea v-model:value="addForm.description" placeholder="请输入描述信息" :rows="2" />
                </a-form-item>
              </a-col>
            </a-row>
          </div>
  
          <div class="form-section">
            <div class="section-title">镜像配置</div>
            <a-row :gutter="16">
              <a-col :span="24">
                <a-form-item label="容器镜像" name="image" :rules="[{ required: true, message: '请输入容器镜像' }]">
                  <a-select v-model:value="addForm.image" placeholder="选择或输入容器镜像" mode="combobox">
                    <a-select-option value="jupyter/tensorflow-notebook:latest">jupyter/tensorflow-notebook:latest</a-select-option>
                    <a-select-option value="jupyter/pytorch-notebook:latest">jupyter/pytorch-notebook:latest</a-select-option>
                    <a-select-option value="jupyter/datascience-notebook:latest">jupyter/datascience-notebook:latest</a-select-option>
                    <a-select-option value="codercom/code-server:latest">codercom/code-server:latest</a-select-option>
                  </a-select>
                </a-form-item>
              </a-col>
            </a-row>
          </div>
  
          <div class="form-section">
            <div class="section-title">资源配置</div>
            <a-row :gutter="16">
              <a-col :span="8">
                <a-form-item label="CPU限制" name="cpu_limit">
                  <a-input v-model:value="addForm.cpu_limit" placeholder="例如: 2" />
                </a-form-item>
              </a-col>
              <a-col :span="8">
                <a-form-item label="内存限制" name="memory_limit">
                  <a-input v-model:value="addForm.memory_limit" placeholder="例如: 4Gi" />
                </a-form-item>
              </a-col>
              <a-col :span="8">
                <a-form-item label="GPU限制" name="gpu_limit">
                  <a-input-number v-model:value="addForm.gpu_limit" :min="0" :max="8" placeholder="GPU数量" class="full-width" />
                </a-form-item>
              </a-col>
            </a-row>
          </div>
  
          <div class="form-section">
            <div class="section-title">存储配置</div>
            <a-form-item v-for="(volume, index) in addForm.volumes" :key="volume.key"
              :label="index === 0 ? '存储卷' : ''" :name="['volumes', index, 'value']">
              <div class="volume-input-group">
                <a-input v-model:value="volume.hostPath" placeholder="主机路径" class="volume-host-input" />
                <div class="volume-separator">:</div>
                <a-input v-model:value="volume.containerPath" placeholder="容器路径" class="volume-container-input" />
                <MinusCircleOutlined v-if="addForm.volumes.length > 1" class="dynamic-delete-button"
                  @click="removeVolume(volume)" />
              </div>
            </a-form-item>
            <a-form-item>
              <a-button type="dashed" class="add-dynamic-button" @click="addVolume">
                <PlusOutlined />
                添加存储卷
              </a-button>
            </a-form-item>
          </div>
  
          <div class="form-section">
            <div class="section-title">环境变量</div>
            <a-form-item v-for="(env, index) in addForm.env_vars" :key="env.key"
              :label="index === 0 ? '环境变量' : ''" :name="['env_vars', index, 'value']">
              <div class="env-input-group">
                <a-input v-model:value="env.envKey" placeholder="变量名" class="env-key-input" />
                <div class="env-separator">=</div>
                <a-input v-model:value="env.envValue" placeholder="变量值" class="env-value-input" />
                <MinusCircleOutlined v-if="addForm.env_vars.length > 1" class="dynamic-delete-button"
                  @click="removeEnvVar(env)" />
              </div>
            </a-form-item>
            <a-form-item>
              <a-button type="dashed" class="add-dynamic-button" @click="addEnvVar">
                <PlusOutlined />
                添加环境变量
              </a-button>
            </a-form-item>
          </div>
        </a-form>
      </a-modal>
  
      <!-- 编辑Notebook模态框 -->
      <a-modal 
        title="编辑Notebook服务" 
        v-model:visible="isEditModalVisible" 
        @ok="handleUpdate" 
        @cancel="closeEditModal"
        :width="800"
        class="custom-modal"
      >
        <a-form ref="editFormRef" :model="editForm" layout="vertical" class="custom-form">
          <div class="form-section">
            <div class="section-title">基本信息</div>
            <a-row :gutter="16">
              <a-col :span="12">
                <a-form-item label="Notebook名称" name="name" :rules="[{ required: true, message: '请输入Notebook名称' }]">
                  <a-input v-model:value="editForm.name" placeholder="请输入Notebook名称" />
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item label="Notebook类型" name="type" :rules="[{ required: true, message: '请选择Notebook类型' }]">
                  <a-select v-model:value="editForm.type" placeholder="请选择Notebook类型">
                    <a-select-option value="jupyter">Jupyter</a-select-option>
                    <a-select-option value="vscode">VS Code</a-select-option>
                    <a-select-option value="rstudio">RStudio</a-select-option>
                  </a-select>
                </a-form-item>
              </a-col>
            </a-row>
            <a-row :gutter="16">
              <a-col :span="24">
                <a-form-item label="描述信息" name="description">
                  <a-textarea v-model:value="editForm.description" placeholder="请输入描述信息" :rows="2" />
                </a-form-item>
              </a-col>
            </a-row>
          </div>
  
          <div class="form-section">
            <div class="section-title">镜像配置</div>
            <a-row :gutter="16">
              <a-col :span="24">
                <a-form-item label="容器镜像" name="image" :rules="[{ required: true, message: '请输入容器镜像' }]">
                  <a-select v-model:value="editForm.image" placeholder="选择或输入容器镜像" mode="combobox">
                    <a-select-option value="jupyter/tensorflow-notebook:latest">jupyter/tensorflow-notebook:latest</a-select-option>
                    <a-select-option value="jupyter/pytorch-notebook:latest">jupyter/pytorch-notebook:latest</a-select-option>
                    <a-select-option value="jupyter/datascience-notebook:latest">jupyter/datascience-notebook:latest</a-select-option>
                    <a-select-option value="codercom/code-server:latest">codercom/code-server:latest</a-select-option>
                  </a-select>
                </a-form-item>
              </a-col>
            </a-row>
          </div>
  
          <div class="form-section">
            <div class="section-title">资源配置</div>
            <a-row :gutter="16">
              <a-col :span="8">
                <a-form-item label="CPU限制" name="cpu_limit">
                  <a-input v-model:value="editForm.cpu_limit" placeholder="例如: 2" />
                </a-form-item>
              </a-col>
              <a-col :span="8">
                <a-form-item label="内存限制" name="memory_limit">
                  <a-input v-model:value="editForm.memory_limit" placeholder="例如: 4Gi" />
                </a-form-item>
              </a-col>
              <a-col :span="8">
                <a-form-item label="GPU限制" name="gpu_limit">
                  <a-input-number v-model:value="editForm.gpu_limit" :min="0" :max="8" placeholder="GPU数量" class="full-width" />
                </a-form-item>
              </a-col>
            </a-row>
          </div>
  
          <div class="form-section">
            <div class="section-title">存储配置</div>
            <a-form-item v-for="(volume, index) in editForm.volumes" :key="volume.key"
              :label="index === 0 ? '存储卷' : ''" :name="['volumes', index, 'value']">
              <div class="volume-input-group">
                <a-input v-model:value="volume.hostPath" placeholder="主机路径" class="volume-host-input" />
                <div class="volume-separator">:</div>
                <a-input v-model:value="volume.containerPath" placeholder="容器路径" class="volume-container-input" />
                <MinusCircleOutlined v-if="editForm.volumes.length > 1" class="dynamic-delete-button"
                  @click="removeVolumeEdit(volume)" />
              </div>
            </a-form-item>
            <a-form-item>
              <a-button type="dashed" class="add-dynamic-button" @click="addVolumeEdit">
                <PlusOutlined />
                添加存储卷
              </a-button>
            </a-form-item>
          </div>
  
          <div class="form-section">
            <div class="section-title">环境变量</div>
            <a-form-item v-for="(env, index) in editForm.env_vars" :key="env.key"
              :label="index === 0 ? '环境变量' : ''" :name="['env_vars', index, 'value']">
              <div class="env-input-group">
                <a-input v-model:value="env.envKey" placeholder="变量名" class="env-key-input" />
                <div class="env-separator">=</div>
                <a-input v-model:value="env.envValue" placeholder="变量值" class="env-value-input" />
                <MinusCircleOutlined v-if="editForm.env_vars.length > 1" class="dynamic-delete-button"
                  @click="removeEnvVarEdit(env)" />
              </div>
            </a-form-item>
            <a-form-item>
              <a-button type="dashed" class="add-dynamic-button" @click="addEnvVarEdit">
                <PlusOutlined />
                添加环境变量
              </a-button>
            </a-form-item>
          </div>
        </a-form>
      </a-modal>
  
      <!-- Notebook详情模态框 -->
      <a-modal 
        title="Notebook详情" 
        v-model:visible="isViewModalVisible" 
        @cancel="closeViewModal"
        :width="900"
        class="custom-modal"
        :footer="null"
      >
        <div class="notebook-detail-container" v-if="viewNotebook">
          <div class="detail-section">
            <div class="section-title">基本信息</div>
            <a-descriptions :column="2" size="small">
              <a-descriptions-item label="Notebook名称">{{ viewNotebook.name }}</a-descriptions-item>
              <a-descriptions-item label="命名空间">{{ viewNotebook.namespace }}</a-descriptions-item>
              <a-descriptions-item label="Notebook类型">
                <a-tag :color="getTypeColor(viewNotebook.type)">{{ getTypeText(viewNotebook.type) }}</a-tag>
              </a-descriptions-item>
              <a-descriptions-item label="运行状态">
                <a-tag :color="getStatusColor(viewNotebook.status)">{{ getStatusText(viewNotebook.status) }}</a-tag>
              </a-descriptions-item>
              <a-descriptions-item label="访问地址" v-if="viewNotebook.status === 'Running'">
                <a :href="viewNotebook.access_url" target="_blank" class="access-url-link">
                  {{ viewNotebook.access_url }}
                </a>
              </a-descriptions-item>
              <a-descriptions-item label="描述信息" :span="2">{{ viewNotebook.description || '无' }}</a-descriptions-item>
              <a-descriptions-item label="创建时间">{{ viewNotebook.created_at }}</a-descriptions-item>
              <a-descriptions-item label="启动时间">{{ viewNotebook.start_time || '未启动' }}</a-descriptions-item>
            </a-descriptions>
          </div>
  
          <div class="detail-section">
            <div class="section-title">资源配置</div>
            <a-descriptions :column="3" size="small">
              <a-descriptions-item label="CPU限制">{{ viewNotebook.cpu_limit }}</a-descriptions-item>
              <a-descriptions-item label="内存限制">{{ viewNotebook.memory_limit }}</a-descriptions-item>
              <a-descriptions-item label="GPU限制">{{ viewNotebook.gpu_limit }}</a-descriptions-item>
            </a-descriptions>
          </div>
  
          <div class="detail-section">
            <div class="section-title">镜像信息</div>
            <a-descriptions :column="1" size="small">
              <a-descriptions-item label="容器镜像">{{ viewNotebook.image }}</a-descriptions-item>
            </a-descriptions>
          </div>
  
          <div class="detail-section" v-if="viewNotebook.env_vars && viewNotebook.env_vars.length > 0">
            <div class="section-title">环境变量</div>
            <div class="env-list">
              <div class="env-item" v-for="env in viewNotebook.env_vars" :key="env">
                <span class="env-key">{{ env.split('=')[0] }}</span>
                <span class="env-separator">=</span>
                <span class="env-value">{{ env.split('=')[1] }}</span>
              </div>
            </div>
          </div>
  
          <div class="detail-section" v-if="viewNotebook.volumes && viewNotebook.volumes.length > 0">
            <div class="section-title">存储卷</div>
            <div class="volume-list">
              <div class="volume-item" v-for="volume in viewNotebook.volumes" :key="volume">
                <span class="volume-host">{{ volume.split(':')[0] }}</span>
                <span class="volume-separator">:</span>
                <span class="volume-container">{{ volume.split(':')[1] }}</span>
              </div>
            </div>
          </div>
        </div>
      </a-modal>
    </div>
  </template>
  
  <script setup lang="ts">
  import { ref, reactive, onMounted } from 'vue';
  import { message, Modal } from 'ant-design-vue';
  import {
    SearchOutlined,
    ReloadOutlined,
    PlusOutlined,
    MinusCircleOutlined,
    LinkOutlined
  } from '@ant-design/icons-vue';
  import type { FormInstance } from 'ant-design-vue';
  
  interface NotebookItem {
    id: number;
    name: string;
    namespace: string;
    type: string;
    status: string;
    image: string;
    description: string;
    cpu_limit: string;
    memory_limit: string;
    gpu_limit: number;
    env_vars: string[];
    volumes: string[];
    access_url?: string;
    created_at: string;
    start_time?: string;
    creator: string;
  }
  
  interface EnvVar {
    envKey: string;
    envValue: string;
    key: number;
  }
  
  interface Volume {
    hostPath: string;
    containerPath: string;
    key: number;
  }
  
  // 搜索和筛选
  const searchText = ref('');
  const statusFilter = ref('');
  const typeFilter = ref('');
  
  // 表格数据
  const data = ref<NotebookItem[]>([]);
  
  // 分页相关
  const pageSizeOptions = ref<string[]>(['10', '20', '30', '40', '50']);
  const current = ref(1);
  const pageSizeRef = ref(10);
  const total = ref(0);
  
  // 模态框状态
  const isAddModalVisible = ref(false);
  const isEditModalVisible = ref(false);
  const isViewModalVisible = ref(false);
  
  // 表单引用
  const addFormRef = ref<FormInstance>();
  const editFormRef = ref<FormInstance>();
  
  // 查看详情的Notebook
  const viewNotebook = ref<NotebookItem | null>(null);
  
  // 新增表单
  const addForm = reactive({
    name: '',
    type: 'jupyter',
    description: '',
    image: 'jupyter/tensorflow-notebook:latest',
    cpu_limit: '2',
    memory_limit: '4Gi',
    gpu_limit: 0,
    env_vars: [] as EnvVar[],
    volumes: [] as Volume[]
  });
  
  // 编辑表单
  const editForm = reactive({
    id: 0,
    name: '',
    type: 'jupyter',
    description: '',
    image: 'jupyter/tensorflow-notebook:latest',
    cpu_limit: '2',
    memory_limit: '4Gi',
    gpu_limit: 0,
    env_vars: [] as EnvVar[],
    volumes: [] as Volume[]
  });
  
  // 表格列配置
  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      key: 'id',
      width: 80,
    },
    {
      title: 'Notebook名称',
      dataIndex: 'name',
      key: 'name',
      width: 180,
    },
    {
      title: '命名空间',
      dataIndex: 'namespace',
      key: 'namespace',
      width: 120,
    },
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      slots: { customRender: 'type' },
      width: 100,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      slots: { customRender: 'status' },
      width: 100,
    },
    {
      title: '资源配置',
      key: 'resources',
      slots: { customRender: 'resources' },
      width: 200,
    },
    {
      title: '容器镜像',
      dataIndex: 'image',
      key: 'image',
      slots: { customRender: 'image' },
      width: 200,
    },
    {
      title: '运行时间',
      key: 'duration',
      slots: { customRender: 'duration' },
      width: 120,
    },
    {
      title: '访问地址',
      key: 'access',
      slots: { customRender: 'access' },
      width: 120,
    },
    {
      title: '创建者',
      dataIndex: 'creator',
      key: 'creator',
      width: 100,
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 150,
    },
    {
      title: '操作',
      key: 'action',
      slots: { customRender: 'action' },
      width: 250,
      fixed: 'right',
    },
  ];
  
  // 环境变量和存储卷计数器
  let envKeyCounter = 0;
  let volumeKeyCounter = 0;
  
  // 初始化数据
  onMounted(() => {
    initForms();
    loadData();
  });
  
  // 初始化表单
  const initForms = () => {
    // 初始化新增表单
    addForm.env_vars = [{ envKey: '', envValue: '', key: ++envKeyCounter }];
    addForm.volumes = [{ hostPath: '', containerPath: '', key: ++volumeKeyCounter }];
  };
  
  // 获取状态颜色
  const getStatusColor = (status: string) => {
    const colorMap: Record<string, string> = {
      'Creating': 'orange',
      'Running': 'green',
      'Stopped': 'default',
      'Failed': 'red',
      'Deleting': 'red'
    };
    return colorMap[status] || 'default';
  };
  
  // 获取状态文本
  const getStatusText = (status: string) => {
    const textMap: Record<string, string> = {
      'Creating': '创建中',
      'Running': '运行中',
      'Stopped': '已停止',
      'Failed': '失败',
      'Deleting': '删除中'
    };
    return textMap[status] || status;
  };
  
  // 获取类型颜色
  const getTypeColor = (type: string) => {
    const colorMap: Record<string, string> = {
      'jupyter': 'blue',
      'vscode': 'purple',
      'rstudio': 'cyan'
    };
    return colorMap[type] || 'default';
  };
  
  // 获取类型文本
  const getTypeText = (type: string) => {
    const textMap: Record<string, string> = {
      'jupyter': 'Jupyter',
      'vscode': 'VS Code',
      'rstudio': 'RStudio'
    };
    return textMap[type] || type;
  };
  
  // 格式化运行时间
  const formatDuration = (startTime?: string) => {
    if (!startTime) return '未启动';
    
    const start = new Date(startTime);
    const now = new Date();
    const duration = Math.floor((now.getTime() - start.getTime()) / 1000);
    
    const days = Math.floor(duration / 86400);
    const hours = Math.floor((duration % 86400) / 3600);
    const minutes = Math.floor((duration % 3600) / 60);
    
    if (days > 0) {
      return `${days}天 ${hours}小时`;
    } else if (hours > 0) {
      return `${hours}小时 ${minutes}分钟`;
    } else {
      return `${minutes}分钟`;
    }
  };
  
  // 加载数据
  const loadData = () => {
    // 模拟数据
    const mockData: NotebookItem[] = [
      {
        id: 1,
        name: 'tensorflow-research-nb',
        namespace: 'default',
        type: 'jupyter',
        status: 'Running',
        image: 'jupyter/tensorflow-notebook:latest',
        description: 'TensorFlow深度学习研究环境',
        cpu_limit: '4',
        memory_limit: '8Gi',
        gpu_limit: 1,
        env_vars: ['JUPYTER_ENABLE_LAB=yes', 'PYTHONPATH=/workspace'],
        volumes: ['/data:/home/jovyan/work', '/models:/home/jovyan/models'],
        access_url: 'https://notebook-001.ml-platform.com',
        created_at: '2024-06-09 09:30:00',
        start_time: '2024-06-09 09:32:00',
        creator: 'admin'
      },
      {
        id: 2,
        name: 'vscode-dev-env',
        namespace: 'dev-team',
        type: 'vscode',
        status: 'Running',
        image: 'codercom/code-server:latest',
        description: 'VS Code开发环境',
        cpu_limit: '2',
        memory_limit: '4Gi',
        gpu_limit: 0,
        env_vars: ['PASSWORD=mypassword'],
        volumes: ['/workspace:/home/coder/workspace'],
        access_url: 'https://vscode-001.ml-platform.com',
        created_at: '2024-06-09 10:15:00',
        start_time: '2024-06-09 10:17:00',
        creator: 'developer1'
      },
      {
        id: 3,
        name: 'r-analysis-notebook',
        namespace: 'data-team',
        type: 'rstudio',
        status: 'Stopped',
        image: 'rocker/rstudio:latest',
        description: 'R语言数据分析环境',
        cpu_limit: '2',
        memory_limit: '4Gi',
        gpu_limit: 0,
        env_vars: ['DISABLE_AUTH=true'],
        volumes: ['/r-data:/home/rstudio/data'],
        created_at: '2024-06-08 14:20:00',
        start_time: '2024-06-08 14:22:00',
        creator: 'analyst1'
      },
      {
        id: 4,
        name: 'pytorch-experiment',
        namespace: 'research',
        type: 'jupyter',
        status: 'Creating',
        image: 'jupyter/pytorch-notebook:latest',
        description: 'PyTorch实验环境',
        cpu_limit: '8',
        memory_limit: '16Gi',
        gpu_limit: 2,
        env_vars: ['CUDA_VISIBLE_DEVICES=0,1'],
        volumes: ['/experiments:/home/jovyan/experiments'],
        created_at: '2024-06-09 11:45:00',
        creator: 'researcher1'
      }
    ];
    
    data.value = mockData;
    total.value = mockData.length;
  };
  
  // 搜索处理
  const handleSearch = () => {
    loadData(); // 这里应该调用真实的搜索API
    message.success('搜索完成');
  };
  
  // 重置处理
  const handleReset = () => {
    searchText.value = '';
    statusFilter.value = '';
    typeFilter.value = '';
    loadData();
    message.success('重置成功');
  };
  
  // 分页处理
  const handlePageChange = (page: number) => {
    current.value = page;
    loadData();
  };
  
  // 页面大小改变处理
  const handleSizeChange = (current: number, size: number) => {
    pageSizeRef.value = size;
    loadData();
  };
  
  // 显示新增模态框
  const showAddModal = () => {
    resetAddForm();
    isAddModalVisible.value = true;
  };
  
  // 关闭新增模态框
  const closeAddModal = () => {
    isAddModalVisible.value = false;
    resetAddForm();
  };
  
  // 重置新增表单
  const resetAddForm = () => {
    Object.assign(addForm, {
      name: '',
      type: 'jupyter',
      description: '',
      image: 'jupyter/tensorflow-notebook:latest',
      cpu_limit: '2',
      memory_limit: '4Gi',
      gpu_limit: 0,
      env_vars: [{ envKey: '', envValue: '', key: ++envKeyCounter }],
      volumes: [{ hostPath: '', containerPath: '', key: ++volumeKeyCounter }]
    });
    addFormRef.value?.resetFields();
  };
  
  // 新增Notebook
  const handleAdd = async () => {
    try {
      await addFormRef.value?.validateFields();
      
      // 处理环境变量和存储卷数据
      const envVars = addForm.env_vars
        .filter(env => env.envKey && env.envValue)
        .map(env => `${env.envKey}=${env.envValue}`);
      
      const volumes = addForm.volumes
        .filter(vol => vol.hostPath && vol.containerPath)
        .map(vol => `${vol.hostPath}:${vol.containerPath}`);
  
      const newNotebook = {
        ...addForm,
        env_vars: envVars,
        volumes: volumes,
        id: data.value.length + 1,
        namespace: 'default',
        status: 'Creating',
        created_at: new Date().toLocaleString(),
        creator: 'admin'
      };
  
      // 这里应该调用创建Notebook的API
      console.log('Creating notebook:', newNotebook);
      
      data.value.unshift(newNotebook as NotebookItem);
      total.value++;
      
      message.success('Notebook创建成功');
      closeAddModal();
    } catch (error) {
      console.error('Validation failed:', error);
    }
  };
  
  // 查看详情
  const handleView = (record: NotebookItem) => {
    viewNotebook.value = record;
    isViewModalVisible.value = true;
  };
  
  // 关闭详情模态框
  const closeViewModal = () => {
    isViewModalVisible.value = false;
    viewNotebook.value = null;
  };
  
  // 编辑Notebook
  const handleEdit = (record: NotebookItem) => {
    // 填充编辑表单
    Object.assign(editForm, {
      id: record.id,
      name: record.name,
      type: record.type,
      description: record.description,
      image: record.image,
      cpu_limit: record.cpu_limit,
      memory_limit: record.memory_limit,
      gpu_limit: record.gpu_limit,
      env_vars: record.env_vars.map(env => {
        const [envKey, envValue] = env.split('=');
        return { envKey, envValue, key: ++envKeyCounter };
      }),
      volumes: record.volumes.map(vol => {
        const [hostPath, containerPath] = vol.split(':');
        return { hostPath, containerPath, key: ++volumeKeyCounter };
      })
    });
    
    // 确保至少有一个环境变量和存储卷输入框
    if (editForm.env_vars.length === 0) {
      editForm.env_vars.push({ envKey: '', envValue: '', key: ++envKeyCounter });
    }
    if (editForm.volumes.length === 0) {
      editForm.volumes.push({ hostPath: '', containerPath: '', key: ++volumeKeyCounter });
    }
    
    isEditModalVisible.value = true;
  };
  
  // 关闭编辑模态框
  const closeEditModal = () => {
    isEditModalVisible.value = false;
  };
  
  // 更新Notebook
  const handleUpdate = async () => {
    try {
      await editFormRef.value?.validateFields();
      
      // 处理环境变量和存储卷数据
      const envVars = editForm.env_vars
        .filter(env => env.envKey && env.envValue)
        .map(env => `${env.envKey}=${env.envValue}`);
      
      const volumes = editForm.volumes
        .filter(vol => vol.hostPath && vol.containerPath)
        .map(vol => `${vol.hostPath}:${vol.containerPath}`);
  
      // 更新数据
      const index = data.value.findIndex(item => item.id === editForm.id);
      if (index !== -1) {
        Object.assign(data.value[index] as NotebookItem, {
          ...editForm,
          env_vars: envVars,
          volumes: volumes
        });
      }
  
      message.success('Notebook更新成功');
      closeEditModal();
    } catch (error) {
      console.error('Validation failed:', error);
    }
  };
  
  // 启动Notebook
  const handleStart = (record: NotebookItem) => {
    Modal.confirm({
      title: '确认启动Notebook',
      content: `确定要启动Notebook "${record.name}" 吗？`,
      onOk() {
        // 这里应该调用启动Notebook的API
        record.status = 'Running';
        record.start_time = new Date().toLocaleString();
        record.access_url = `https://notebook-${record.id}.ml-platform.com`;
        message.success('Notebook启动成功');
      },
    });
  };
  
  // 停止Notebook
  const handleStop = (record: NotebookItem) => {
    Modal.confirm({
      title: '确认停止Notebook',
      content: `确定要停止Notebook "${record.name}" 吗？`,
      onOk() {
        // 这里应该调用停止Notebook的API
        record.status = 'Stopped';
        record.access_url = undefined;
        message.success('Notebook已停止');
      },
    });
  };
  
  // 删除Notebook
  const handleDelete = (record: NotebookItem) => {
    Modal.confirm({
      title: '确认删除Notebook',
      content: `确定要删除Notebook "${record.name}" 吗？此操作不可恢复。`,
      onOk() {
        // 这里应该调用删除Notebook的API
        const index = data.value.findIndex(item => item.id === record.id);
        if (index !== -1) {
          data.value.splice(index, 1);
          total.value--;
        }
        message.success('Notebook已删除');
      },
    });
  };
  
  // 打开Notebook
  const openNotebook = (record: NotebookItem) => {
    if (record.access_url) {
      window.open(record.access_url, '_blank');
    }
  };
  
  // 添加环境变量
  const addEnvVar = () => {
    addForm.env_vars.push({
      envKey: '',
      envValue: '',
      key: ++envKeyCounter
    });
  };
  
  // 删除环境变量
  const removeEnvVar = (item: EnvVar) => {
    const index = addForm.env_vars.indexOf(item);
    if (index !== -1) {
      addForm.env_vars.splice(index, 1);
    }
  };
  
  // 添加存储卷
  const addVolume = () => {
    addForm.volumes.push({
      hostPath: '',
      containerPath: '',
      key: ++volumeKeyCounter
    });
  };
  
  // 删除存储卷
  const removeVolume = (item: Volume) => {
    const index = addForm.volumes.indexOf(item);
    if (index !== -1) {
      addForm.volumes.splice(index, 1);
    }
  };
  
  // 编辑表单的环境变量操作
  const addEnvVarEdit = () => {
    editForm.env_vars.push({
      envKey: '',
      envValue: '',
      key: ++envKeyCounter
    });
  };
  
  const removeEnvVarEdit = (item: EnvVar) => {
    const index = editForm.env_vars.indexOf(item);
    if (index !== -1) {
      editForm.env_vars.splice(index, 1);
    }
  };
  
  // 编辑表单的存储卷操作
  const addVolumeEdit = () => {
    editForm.volumes.push({
      hostPath: '',
      containerPath: '',
      key: ++volumeKeyCounter
    });
  };
  
  const removeVolumeEdit = (item: Volume) => {
    const index = editForm.volumes.indexOf(item);
    if (index !== -1) {
      editForm.volumes.splice(index, 1);
    }
  };
  </script>
  
  <style scoped>
  .notebook-service-page {
    padding: 20px;
    background-color: #f5f7fa;
    min-height: 100vh;
  }
  
  .page-header {
    margin-bottom: 24px;
  }
  
  .page-title {
    font-size: 24px;
    font-weight: 600;
    color: #1a202c;
    margin: 0 0 8px 0;
  }
  
  .dashboard-card {
    background: white;
    border-radius: 8px;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
    margin-bottom: 24px;
  }
  
  .custom-toolbar {
    padding: 20px;
    display: flex;
    justify-content: space-between;
    align-items: center;
    flex-wrap: wrap;
    gap: 16px;
  }
  
  .search-filters {
    display: flex;
    gap: 12px;
    flex-wrap: wrap;
    align-items: center;
  }
  
  .search-input {
    width: 200px;
  }
  
  .status-filter,
  .type-filter {
    width: 150px;
  }
  
  .action-button {
    height: 32px;
  }
  
  .reset-button {
    background: #f1f5f9;
    border-color: #e2e8f0;
    color: #475569;
  }
  
  .action-buttons {
    display: flex;
    gap: 12px;
  }
  
  .add-button {
    background: #3b82f6;
    border-color: #3b82f6;
  }
  
  .table-container {
    padding: 0;
  }
  
  .custom-table {
    border-radius: 8px;
    overflow: hidden;
  }
  
  .custom-table :deep(.ant-table-thead > tr > th) {
    background-color: #f8fafc;
    border-bottom: 1px solid #e2e8f0;
    color: #374151;
    font-weight: 600;
  }
  
  .custom-table :deep(.ant-table-tbody > tr:hover > td) {
    background-color: #f8fafc;
  }
  
  .status-tag {
    border-radius: 4px;
    font-size: 12px;
    font-weight: 500;
  }
  
  .type-tag {
    border-radius: 4px;
    font-size: 12px;
    font-weight: 500;
  }
  
  .resource-container {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  
  .resource-item {
    display: flex;
    align-items: center;
    gap: 4px;
    font-size: 12px;
  }
  
  .resource-label {
    color: #64748b;
    font-weight: 500;
    min-width: 35px;
  }
  
  .resource-value {
    color: #1a202c;
    font-weight: 500;
  }
  
  .image-container {
    max-width: 180px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: 12px;
    color: #4b5563;
  }
  
  .duration-container {
    font-size: 12px;
    color: #4b5563;
    font-family: monospace;
  }
  
  .access-container {
    text-align: center;
  }
  
  .access-link {
    padding: 2px 8px;
    font-size: 12px;
  }
  
  .access-disabled {
    color: #9ca3af;
    font-size: 12px;
  }
  
  .access-url-link {
    color: #3b82f6;
    text-decoration: none;
  }
  
  .access-url-link:hover {
    text-decoration: underline;
  }
  
  .action-column {
    display: flex;
    gap: 8px;
    flex-wrap: wrap;
  }
  
  .pagination-container {
    padding: 20px;
    display: flex;
    justify-content: flex-end;
    border-top: 1px solid #e2e8f0;
  }
  
  .custom-pagination {
    margin: 0;
  }
  
  .custom-modal :deep(.ant-modal-header) {
    border-bottom: 1px solid #e2e8f0;
    padding: 16px 24px;
  }
  
  .custom-modal :deep(.ant-modal-title) {
    font-size: 18px;
    font-weight: 600;
    color: #1a202c;
  }
  
  .custom-form {
    margin-top: 20px;
  }
  
  .form-section {
    margin-bottom: 32px;
  }
  
  .section-title {
    font-size: 16px;
    font-weight: 600;
    color: #1a202c;
    margin-bottom: 16px;
    padding-bottom: 8px;
    border-bottom: 1px solid #e2e8f0;
  }
  
  .full-width {
    width: 100%;
  }
  
  .env-input-group,
  .volume-input-group {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  
  .env-key-input,
  .volume-host-input {
    flex: 1;
  }
  
  .env-separator,
  .volume-separator {
    color: #64748b;
    font-weight: 500;
  }
  
  .env-value-input,
  .volume-container-input {
    flex: 2;
  }
  
  .dynamic-delete-button {
    color: #ef4444;
    cursor: pointer;
    font-size: 16px;
  }
  
  .dynamic-delete-button:hover {
    color: #dc2626;
  }
  
  .add-dynamic-button {
    border-style: dashed;
    border-color: #d1d5db;
    color: #6b7280;
  }
  
  .add-dynamic-button:hover {
    border-color: #3b82f6;
    color: #3b82f6;
  }
  
  .notebook-detail-container {
    max-height: 600px;
    overflow-y: auto;
  }
  
  .detail-section {
    margin-bottom: 24px;
  }
  
  .detail-section .section-title {
    margin-bottom: 12px;
    font-size: 14px;
  }
  
  .env-list,
  .volume-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  
  .env-item,
  .volume-item {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px;
    background: #f8fafc;
    border-radius: 4px;
    font-size: 12px;
    font-family: monospace;
  }
  
  .env-key,
  .volume-host {
    color: #3b82f6;
    font-weight: 600;
  }
  
  .env-separator,
  .volume-separator {
    color: #64748b;
  }
  
  .env-value,
  .volume-container {
    color: #059669;
  }
  
  @media (max-width: 768px) {
    .custom-toolbar {
      flex-direction: column;
      align-items: stretch;
    }
    
    .search-filters {
      justify-content: stretch;
    }
    
    .search-input,
    .status-filter,
    .type-filter {
      width: 100%;
      min-width: auto;
    }
    
    .action-buttons {
      justify-content: center;
    }
  
    .action-column {
      flex-direction: column;
      gap: 4px;
    }
  }
  </style>