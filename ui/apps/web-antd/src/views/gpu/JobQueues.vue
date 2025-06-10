<template>
    <div class="job-queue-page">
      <!-- 页面标题区域 -->
      <div class="page-header">
        <h2 class="page-title">作业队列</h2>
        <div class="page-description">管理作业队列</div>
      </div>
  
      <!-- 查询和操作工具栏 -->
      <div class="dashboard-card custom-toolbar">
        <div class="search-filters">
          <a-input 
            v-model:value="searchText" 
            placeholder="请输入作业名称" 
            class="search-input"
          >
            <template #prefix>
              <SearchOutlined class="search-icon" />
            </template>
          </a-input>
          <a-select 
            v-model:value="statusFilter" 
            placeholder="作业状态" 
            class="status-filter"
            allowClear
          >
            <a-select-option value="">全部状态</a-select-option>
            <a-select-option value="Pending">等待中</a-select-option>
            <a-select-option value="Running">运行中</a-select-option>
            <a-select-option value="Completed">已完成</a-select-option>
            <a-select-option value="Failed">失败</a-select-option>
            <a-select-option value="Terminated">已终止</a-select-option>
          </a-select>
          <a-select 
            v-model:value="queueFilter" 
            placeholder="队列名称" 
            class="queue-filter"
            allowClear
          >
            <a-select-option value="">全部队列</a-select-option>
            <a-select-option value="default">default</a-select-option>
            <a-select-option value="high-priority">high-priority</a-select-option>
            <a-select-option value="low-priority">low-priority</a-select-option>
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
            创建作业
          </a-button>
        </div>
      </div>
  
      <!-- 作业列表表格 -->
      <div class="dashboard-card table-container">
        <a-table 
          :columns="columns" 
          :data-source="data" 
          row-key="id" 
          :pagination="false"
          class="custom-table"
          :scroll="{ x: 1400 }"
        >
          <!-- 作业状态列 -->
          <template #status="{ record }">
            <a-tag :color="getStatusColor(record.status)" class="status-tag">
              {{ getStatusText(record.status) }}
            </a-tag>
          </template>
          
          <!-- 资源需求列 -->
          <template #resources="{ record }">
            <div class="resource-container">
              <div class="resource-item">
                <span class="resource-label">CPU:</span>
                <span class="resource-value">{{ record.cpu_request }}</span>
              </div>
              <div class="resource-item">
                <span class="resource-label">内存:</span>
                <span class="resource-value">{{ record.memory_request }}</span>
              </div>
              <div class="resource-item">
                <span class="resource-label">GPU:</span>
                <span class="resource-value">{{ record.gpu_request }}</span>
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
          
          <!-- 优先级列 -->
          <template #priority="{ record }">
            <a-tag :color="getPriorityColor(record.priority)" class="priority-tag">
              {{ getPriorityText(record.priority) }}
            </a-tag>
          </template>
          
          <!-- 运行时间列 -->
          <template #duration="{ record }">
            <div class="duration-container">
              {{ formatDuration(record.start_time, record.completion_time) }}
            </div>
          </template>
          
          <!-- 操作列 -->
          <template #action="{ record }">
            <div class="action-column">
              <a-button type="primary" size="small" @click="handleView(record)">
                查看
              </a-button>
              <a-button type="default" size="small" @click="handleEdit(record)" v-if="record.status === 'Pending'">
                编辑
              </a-button>
              <a-button type="default" size="small" @click="handleStop(record)" v-if="['Pending', 'Running'].includes(record.status)">
                停止
              </a-button>
              <a-button type="default" size="small" @click="handleDelete(record)" v-if="['Completed', 'Failed', 'Terminated'].includes(record.status)">
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
  
      <!-- 创建作业模态框 -->
      <a-modal 
        title="创建训练作业" 
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
                <a-form-item label="作业名称" name="name" :rules="[{ required: true, message: '请输入作业名称' }]">
                  <a-input v-model:value="addForm.name" placeholder="请输入作业名称" />
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item label="队列名称" name="queue" :rules="[{ required: true, message: '请选择队列' }]">
                  <a-select v-model:value="addForm.queue" placeholder="请选择队列">
                    <a-select-option value="default">default</a-select-option>
                    <a-select-option value="high-priority">high-priority</a-select-option>
                    <a-select-option value="low-priority">low-priority</a-select-option>
                  </a-select>
                </a-form-item>
              </a-col>
            </a-row>
            <a-row :gutter="16">
              <a-col :span="12">
                <a-form-item label="优先级" name="priority">
                  <a-select v-model:value="addForm.priority" placeholder="请选择优先级">
                    <a-select-option :value="1">低</a-select-option>
                    <a-select-option :value="5">中</a-select-option>
                    <a-select-option :value="10">高</a-select-option>
                  </a-select>
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item label="任务数量" name="task_count">
                  <a-input-number v-model:value="addForm.task_count" :min="1" :max="100" placeholder="任务数量" class="full-width" />
                </a-form-item>
              </a-col>
            </a-row>
          </div>
  
          <div class="form-section">
            <div class="section-title">容器配置</div>
            <a-row :gutter="16">
              <a-col :span="24">
                <a-form-item label="容器镜像" name="image" :rules="[{ required: true, message: '请输入容器镜像' }]">
                  <a-input v-model:value="addForm.image" placeholder="例如: pytorch/pytorch:1.12.0-cuda11.3-cudnn8-devel" />
                </a-form-item>
              </a-col>
            </a-row>
            <a-row :gutter="16">
              <a-col :span="24">
                <a-form-item label="启动命令" name="command">
                  <a-textarea v-model:value="addForm.command" placeholder="请输入启动命令，多行命令用换行分隔" :rows="3" />
                </a-form-item>
              </a-col>
            </a-row>
          </div>
  
          <div class="form-section">
            <div class="section-title">资源配置</div>
            <a-row :gutter="16">
              <a-col :span="8">
                <a-form-item label="CPU需求" name="cpu_request">
                  <a-input v-model:value="addForm.cpu_request" placeholder="例如: 2" />
                </a-form-item>
              </a-col>
              <a-col :span="8">
                <a-form-item label="内存需求" name="memory_request">
                  <a-input v-model:value="addForm.memory_request" placeholder="例如: 4Gi" />
                </a-form-item>
              </a-col>
              <a-col :span="8">
                <a-form-item label="GPU需求" name="gpu_request">
                  <a-input-number v-model:value="addForm.gpu_request" :min="0" :max="8" placeholder="GPU数量" class="full-width" />
                </a-form-item>
              </a-col>
            </a-row>
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
        </a-form>
      </a-modal>
  
      <!-- 编辑作业模态框 -->
      <a-modal 
        title="编辑训练作业" 
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
                <a-form-item label="作业名称" name="name" :rules="[{ required: true, message: '请输入作业名称' }]">
                  <a-input v-model:value="editForm.name" placeholder="请输入作业名称" />
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item label="队列名称" name="queue" :rules="[{ required: true, message: '请选择队列' }]">
                  <a-select v-model:value="editForm.queue" placeholder="请选择队列">
                    <a-select-option value="default">default</a-select-option>
                    <a-select-option value="high-priority">high-priority</a-select-option>
                    <a-select-option value="low-priority">low-priority</a-select-option>
                  </a-select>
                </a-form-item>
              </a-col>
            </a-row>
            <a-row :gutter="16">
              <a-col :span="12">
                <a-form-item label="优先级" name="priority">
                  <a-select v-model:value="editForm.priority" placeholder="请选择优先级">
                    <a-select-option :value="1">低</a-select-option>
                    <a-select-option :value="5">中</a-select-option>
                    <a-select-option :value="10">高</a-select-option>
                  </a-select>
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item label="任务数量" name="task_count">
                  <a-input-number v-model:value="editForm.task_count" :min="1" :max="100" placeholder="任务数量" class="full-width" />
                </a-form-item>
              </a-col>
            </a-row>
          </div>
  
          <div class="form-section">
            <div class="section-title">容器配置</div>
            <a-row :gutter="16">
              <a-col :span="24">
                <a-form-item label="容器镜像" name="image" :rules="[{ required: true, message: '请输入容器镜像' }]">
                  <a-input v-model:value="editForm.image" placeholder="例如: pytorch/pytorch:1.12.0-cuda11.3-cudnn8-devel" />
                </a-form-item>
              </a-col>
            </a-row>
            <a-row :gutter="16">
              <a-col :span="24">
                <a-form-item label="启动命令" name="command">
                  <a-textarea v-model:value="editForm.command" placeholder="请输入启动命令，多行命令用换行分隔" :rows="3" />
                </a-form-item>
              </a-col>
            </a-row>
          </div>
  
          <div class="form-section">
            <div class="section-title">资源配置</div>
            <a-row :gutter="16">
              <a-col :span="8">
                <a-form-item label="CPU需求" name="cpu_request">
                  <a-input v-model:value="editForm.cpu_request" placeholder="例如: 2" />
                </a-form-item>
              </a-col>
              <a-col :span="8">
                <a-form-item label="内存需求" name="memory_request">
                  <a-input v-model:value="editForm.memory_request" placeholder="例如: 4Gi" />
                </a-form-item>
              </a-col>
              <a-col :span="8">
                <a-form-item label="GPU需求" name="gpu_request">
                  <a-input-number v-model:value="editForm.gpu_request" :min="0" :max="8" placeholder="GPU数量" class="full-width" />
                </a-form-item>
              </a-col>
            </a-row>
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
        </a-form>
      </a-modal>
  
      <!-- 作业详情模态框 -->
      <a-modal 
        title="作业详情" 
        v-model:visible="isViewModalVisible" 
        @cancel="closeViewModal"
        :width="900"
        class="custom-modal"
        :footer="null"
      >
        <div class="job-detail-container" v-if="viewJob">
          <div class="detail-section">
            <div class="section-title">基本信息</div>
            <a-descriptions :column="2" size="small">
              <a-descriptions-item label="作业名称">{{ viewJob.name }}</a-descriptions-item>
              <a-descriptions-item label="命名空间">{{ viewJob.namespace }}</a-descriptions-item>
              <a-descriptions-item label="队列名称">{{ viewJob.queue }}</a-descriptions-item>
              <a-descriptions-item label="状态">
                <a-tag :color="getStatusColor(viewJob.status)">{{ getStatusText(viewJob.status) }}</a-tag>
              </a-descriptions-item>
              <a-descriptions-item label="优先级">
                <a-tag :color="getPriorityColor(viewJob.priority)">{{ getPriorityText(viewJob.priority) }}</a-tag>
              </a-descriptions-item>
              <a-descriptions-item label="任务数量">{{ viewJob.task_count }}</a-descriptions-item>
              <a-descriptions-item label="创建时间">{{ viewJob.created_at }}</a-descriptions-item>
              <a-descriptions-item label="开始时间">{{ viewJob.start_time || '未开始' }}</a-descriptions-item>
            </a-descriptions>
          </div>
  
          <div class="detail-section">
            <div class="section-title">资源信息</div>
            <a-descriptions :column="3" size="small">
              <a-descriptions-item label="CPU需求">{{ viewJob.cpu_request }}</a-descriptions-item>
              <a-descriptions-item label="内存需求">{{ viewJob.memory_request }}</a-descriptions-item>
              <a-descriptions-item label="GPU需求">{{ viewJob.gpu_request }}</a-descriptions-item>
            </a-descriptions>
          </div>
  
          <div class="detail-section">
            <div class="section-title">容器配置</div>
            <a-descriptions :column="1" size="small">
              <a-descriptions-item label="镜像">{{ viewJob.image }}</a-descriptions-item>
              <a-descriptions-item label="启动命令">
                <pre class="command-pre">{{ viewJob.command }}</pre>
              </a-descriptions-item>
            </a-descriptions>
          </div>
  
          <div class="detail-section" v-if="viewJob.env_vars && viewJob.env_vars.length > 0">
            <div class="section-title">环境变量</div>
            <div class="env-list">
              <div class="env-item" v-for="env in viewJob.env_vars" :key="env">
                <span class="env-key">{{ env.split('=')[0] }}</span>
                <span class="env-separator">=</span>
                <span class="env-value">{{ env.split('=')[1] }}</span>
              </div>
            </div>
          </div>
  
          <div class="detail-section" v-if="viewJob.volumes && viewJob.volumes.length > 0">
            <div class="section-title">存储卷</div>
            <div class="volume-list">
              <div class="volume-item" v-for="volume in viewJob.volumes" :key="volume">
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
  
  <script lang="ts" setup>
  import { ref, reactive, onMounted } from 'vue';
  import { message, Modal } from 'ant-design-vue';
  import {
    SearchOutlined,
    ReloadOutlined,
    PlusOutlined,
    MinusCircleOutlined
  } from '@ant-design/icons-vue';
  import type { FormInstance } from 'ant-design-vue';
  
  interface JobItem {
    id: number;
    name: string;
    namespace: string;
    queue: string;
    status: string;
    priority: number;
    task_count: number;
    image: string;
    command: string;
    cpu_request: string;
    memory_request: string;
    gpu_request: number;
    env_vars: string[];
    volumes: string[];
    created_at: string;
    start_time?: string;
    completion_time?: string;
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
  const queueFilter = ref('');
  
  // 表格数据
  const data = ref<JobItem[]>([]);
  
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
  
  // 查看详情的作业
  const viewJob = ref<JobItem | null>(null);
  
  // 新增表单
  const addForm = reactive({
    name: '',
    queue: 'default',
    priority: 5,
    task_count: 1,
    image: '',
    command: '',
    cpu_request: '2',
    memory_request: '4Gi',
    gpu_request: 1,
    env_vars: [] as EnvVar[],
    volumes: [] as Volume[]
  });
  
  // 编辑表单
  const editForm = reactive({
    id: 0,
    name: '',
    queue: 'default',
    priority: 5,
    task_count: 1,
    image: '',
    command: '',
    cpu_request: '2',
    memory_request: '4Gi',
    gpu_request: 1,
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
      title: '作业名称',
      dataIndex: 'name',
      key: 'name',
      width: 150,
    },
    {
      title: '命名空间',
      dataIndex: 'namespace',
      key: 'namespace',
      width: 120,
    },
    {
      title: '队列',
      dataIndex: 'queue',
      key: 'queue',
      width: 120,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      slots: { customRender: 'status' },
      width: 100,
    },
    {
      title: '优先级',
      dataIndex: 'priority',
      key: 'priority',
      slots: { customRender: 'priority' },
      width: 80,
    },
    {
      title: '任务数',
      dataIndex: 'task_count',
      key: 'task_count',
      width: 80,
    },
    {
      title: '资源需求',
      key: 'resources',
      slots: { customRender: 'resources' },
      width: 200,
    },
    {
      title: '容器镜像',
      dataIndex: 'image',
      key: 'image',
      slots: { customRender: 'image' },
      width: 180,
    },
    {
      title: '运行时间',
      key: 'duration',
      slots: { customRender: 'duration' },
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
      width: 200,
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
      'Pending': 'orange',
      'Running': 'blue',
      'Completed': 'green',
      'Failed': 'red',
      'Terminated': 'red'
    };
    return colorMap[status] || 'default';
  };

  // 获取状态文本
  const getStatusText = (status: string) => {
    const textMap: Record<string, string> = {
      'Pending': '等待中',
      'Running': '运行中',
      'Completed': '已完成',
      'Failed': '失败',
      'Terminated': '已终止'
    };
    return textMap[status] || status;
  };

  // 获取优先级颜色
  const getPriorityColor = (priority: number) => {
    if (priority >= 10) return 'red';
    if (priority >= 5) return 'orange';
    return 'green';
  };

  // 获取优先级文本
  const getPriorityText = (priority: number) => {
    if (priority >= 10) return '高';
    if (priority >= 5) return '中';
    return '低';
  };

  // 格式化运行时间
  const formatDuration = (startTime?: string, completionTime?: string) => {
    if (!startTime) return '未开始';
    
    const start = new Date(startTime);
    const end = completionTime ? new Date(completionTime) : new Date();
    const duration = Math.floor((end.getTime() - start.getTime()) / 1000);
    
    const hours = Math.floor(duration / 3600);
    const minutes = Math.floor((duration % 3600) / 60);
    const seconds = duration % 60;
    
    if (hours > 0) {
      return `${hours}h ${minutes}m ${seconds}s`;
    } else if (minutes > 0) {
      return `${minutes}m ${seconds}s`;
    } else {
      return `${seconds}s`;
    }
  };

  // 加载数据
  const loadData = () => {
    // 模拟数据
    const mockData: JobItem[] = [
      {
        id: 1,
        name: 'pytorch-training-job-001',
        namespace: 'default',
        queue: 'default',
        status: 'Running',
        priority: 5,
        task_count: 4,
        image: 'pytorch/pytorch:1.12.0-cuda11.3-cudnn8-devel',
        command: 'python train.py --epochs 100 --batch-size 32',
        cpu_request: '4',
        memory_request: '8Gi',
        gpu_request: 2,
        env_vars: ['CUDA_VISIBLE_DEVICES=0,1', 'PYTHONPATH=/workspace'],
        volumes: ['/data:/workspace/data', '/models:/workspace/models'],
        created_at: '2024-06-09 10:30:00',
        start_time: '2024-06-09 10:32:00',
        creator: 'admin'
      },
      {
        id: 2,
        name: 'tensorflow-train-job-002',
        namespace: 'ml-team',
        queue: 'high-priority',
        status: 'Pending',
        priority: 10,
        task_count: 2,
        image: 'tensorflow/tensorflow:2.8.0-gpu',
        command: 'python main.py --dataset imagenet --model resnet50',
        cpu_request: '8',
        memory_request: '16Gi',
        gpu_request: 4,
        env_vars: ['TF_CPP_MIN_LOG_LEVEL=2'],
        volumes: ['/datasets:/data'],
        created_at: '2024-06-09 11:15:00',
        creator: 'user1'
      },
      {
        id: 3,
        name: 'bert-finetuning-job-003',
        namespace: 'nlp-team',
        queue: 'default',
        status: 'Completed',
        priority: 5,
        task_count: 1,
        image: 'huggingface/transformers-pytorch-gpu:latest',
        command: 'python finetune_bert.py --model bert-base-uncased',
        cpu_request: '2',
        memory_request: '4Gi',
        gpu_request: 1,
        env_vars: ['TRANSFORMERS_CACHE=/workspace/cache'],
        volumes: ['/nlp-data:/workspace/data'],
        created_at: '2024-06-09 09:00:00',
        start_time: '2024-06-09 09:05:00',
        completion_time: '2024-06-09 10:30:00',
        creator: 'user2'
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
    queueFilter.value = '';
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
      queue: 'default',
      priority: 5,
      task_count: 1,
      image: '',
      command: '',
      cpu_request: '2',
      memory_request: '4Gi',
      gpu_request: 1,
      env_vars: [{ envKey: '', envValue: '', key: ++envKeyCounter }],
      volumes: [{ hostPath: '', containerPath: '', key: ++volumeKeyCounter }]
    });
    addFormRef.value?.resetFields();
  };

  // 新增作业
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

      const newJob = {
        ...addForm,
        env_vars: envVars,
        volumes: volumes,
        id: data.value.length + 1,
        namespace: 'default',
        status: 'Pending',
        created_at: new Date().toLocaleString(),
        creator: 'admin'
      };

      // 这里应该调用创建作业的API
      console.log('Creating job:', newJob);
      
      data.value.unshift(newJob as JobItem);
      total.value++;
      
      message.success('作业创建成功');
      closeAddModal();
    } catch (error) {
      console.error('Validation failed:', error);
    }
  };

  // 查看详情
  const handleView = (record: JobItem) => {
    viewJob.value = record;
    isViewModalVisible.value = true;
  };

  // 关闭详情模态框
  const closeViewModal = () => {
    isViewModalVisible.value = false;
    viewJob.value = null;
  };

  // 编辑作业
  const handleEdit = (record: JobItem) => {
    // 填充编辑表单
    Object.assign(editForm, {
      id: record.id,
      name: record.name,
      queue: record.queue,
      priority: record.priority,
      task_count: record.task_count,
      image: record.image,
      command: record.command,
      cpu_request: record.cpu_request,
      memory_request: record.memory_request,
      gpu_request: record.gpu_request,
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

  // 更新作业
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
        Object.assign(data.value[index] as JobItem, {
          ...editForm,
          env_vars: envVars,
          volumes: volumes
        });
      }

      message.success('作业更新成功');
      closeEditModal();
    } catch (error) {
      console.error('Validation failed:', error);
    }
  };

  // 停止作业
  const handleStop = (record: JobItem) => {
    Modal.confirm({
      title: '确认停止作业',
      content: `确定要停止作业 "${record.name}" 吗？`,
      onOk() {
        // 这里应该调用停止作业的API
        record.status = 'Terminated';
        record.completion_time = new Date().toLocaleString();
        message.success('作业已停止');
      },
    });
  };

  // 删除作业
  const handleDelete = (record: JobItem) => {
    Modal.confirm({
      title: '确认删除作业',
      content: `确定要删除作业 "${record.name}" 吗？此操作不可恢复。`,
      onOk() {
        // 这里应该调用删除作业的API
        const index = data.value.findIndex(item => item.id === record.id);
        if (index !== -1) {
          data.value.splice(index, 1);
          total.value--;
        }
        message.success('作业已删除');
      },
    });
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
.job-queue-page {
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

.page-description {
  color: #64748b;
  font-size: 14px;
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
.queue-filter {
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

.priority-tag {
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
  max-width: 150px;
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

.action-column {
  display: flex;
  gap: 8px;
}

.view-button {
  background: #10b981;
  border-color: #10b981;
}

.edit-button {
  background: #f59e0b;
  border-color: #f59e0b;
}

.stop-button,
.delete-button {
  background: #ef4444;
  border-color: #ef4444;
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

.job-detail-container {
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

.command-pre {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 4px;
  padding: 8px;
  font-size: 12px;
  margin: 0;
  white-space: pre-wrap;
  word-break: break-all;
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
  .queue-filter {
    width: 100%;
    min-width: auto;
  }
  
  .action-buttons {
    justify-content: center;
  }
}
</style>