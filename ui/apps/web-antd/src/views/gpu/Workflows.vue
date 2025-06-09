<template>
    <div class="workflow-page">
      <!-- 页面标题区域 -->
      <div class="page-header">
        <h2 class="page-title">工作流</h2>
      </div>
  
      <!-- 查询和操作工具栏 -->
      <div class="dashboard-card custom-toolbar">
        <div class="search-filters">
          <a-input 
            v-model:value="searchText" 
            placeholder="请输入工作流名称" 
            class="search-input"
          >
            <template #prefix>
              <SearchOutlined class="search-icon" />
            </template>
          </a-input>
          <a-select 
            v-model:value="statusFilter" 
            placeholder="工作流状态" 
            class="status-filter"
            allowClear
          >
            <a-select-option value="">全部状态</a-select-option>
            <a-select-option value="Draft">草稿</a-select-option>
            <a-select-option value="Running">运行中</a-select-option>
            <a-select-option value="Completed">已完成</a-select-option>
            <a-select-option value="Failed">失败</a-select-option>
            <a-select-option value="Paused">已暂停</a-select-option>
          </a-select>
          <a-select 
            v-model:value="categoryFilter" 
            placeholder="工作流类型" 
            class="category-filter"
            allowClear
          >
            <a-select-option value="">全部类型</a-select-option>
            <a-select-option value="training">模型训练</a-select-option>
            <a-select-option value="inference">模型推理</a-select-option>
            <a-select-option value="preprocessing">数据预处理</a-select-option>
            <a-select-option value="analysis">数据分析</a-select-option>
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
            创建工作流
          </a-button>
        </div>
      </div>
  
      <!-- 工作流列表表格 -->
      <div class="dashboard-card table-container">
        <a-table 
          :columns="columns" 
          :data-source="data" 
          row-key="id" 
          :pagination="false"
          class="custom-table"
          :scroll="{ x: 1200 }"
        >
          <!-- 工作流状态列 -->
          <template #status="{ record }">
            <a-tag :color="getStatusColor(record.status)" class="status-tag">
              {{ getStatusText(record.status) }}
            </a-tag>
          </template>
          
          <!-- 工作流类型列 -->
          <template #category="{ record }">
            <a-tag :color="getCategoryColor(record.category)" class="category-tag">
              {{ getCategoryText(record.category) }}
            </a-tag>
          </template>
          
          <!-- 步骤进度列 -->
          <template #progress="{ record }">
            <div class="progress-container">
              <a-progress 
                :percent="getProgressPercent(record)" 
                :size="'small'" 
                :status="getProgressStatus(record.status)"
              />
              <div class="progress-text">
                {{ record.completed_steps }}/{{ record.total_steps }} 步骤
              </div>
            </div>
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
              <a-button type="default" size="small" @click="handleEdit(record)" v-if="record.status === 'Draft'">
                编辑
              </a-button>
              <a-button type="default" size="small" @click="handleStart(record)" v-if="record.status === 'Draft'">
                启动
              </a-button>
              <a-button type="default" size="small" @click="handlePause(record)" v-if="record.status === 'Running'">
                暂停
              </a-button>
              <a-button type="default" size="small" @click="handleResume(record)" v-if="record.status === 'Paused'">
                继续
              </a-button>
              <a-button type="default" size="small" @click="handleStop(record)" v-if="['Running', 'Paused'].includes(record.status)">
                停止
              </a-button>
              <a-button type="default" size="small" @click="handleDelete(record)" v-if="['Completed', 'Failed', 'Draft'].includes(record.status)">
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
  
      <!-- 创建工作流模态框 -->
      <a-modal 
        title="创建工作流" 
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
                <a-form-item label="工作流名称" name="name" :rules="[{ required: true, message: '请输入工作流名称' }]">
                  <a-input v-model:value="addForm.name" placeholder="请输入工作流名称" />
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item label="工作流类型" name="category" :rules="[{ required: true, message: '请选择工作流类型' }]">
                  <a-select v-model:value="addForm.category" placeholder="请选择工作流类型">
                    <a-select-option value="training">模型训练</a-select-option>
                    <a-select-option value="inference">模型推理</a-select-option>
                    <a-select-option value="preprocessing">数据预处理</a-select-option>
                    <a-select-option value="analysis">数据分析</a-select-option>
                  </a-select>
                </a-form-item>
              </a-col>
            </a-row>
            <a-row :gutter="16">
              <a-col :span="24">
                <a-form-item label="工作流描述" name="description">
                  <a-textarea v-model:value="addForm.description" placeholder="请输入工作流描述" :rows="3" />
                </a-form-item>
              </a-col>
            </a-row>
          </div>
  
          <div class="form-section">
            <div class="section-title">工作流步骤</div>
            <a-form-item v-for="(step, index) in addForm.steps" :key="step.key"
              :label="index === 0 ? '步骤配置' : ''" :name="['steps', index, 'name']"
              :rules="[{ required: true, message: '请输入步骤名称' }]">
              <div class="step-input-group">
                <div class="step-number">{{ index + 1 }}</div>
                <a-input v-model:value="step.name" placeholder="步骤名称" class="step-name-input" />
                <a-select v-model:value="step.type" placeholder="步骤类型" class="step-type-select">
                  <a-select-option value="data-load">数据加载</a-select-option>
                  <a-select-option value="data-process">数据处理</a-select-option>
                  <a-select-option value="model-train">模型训练</a-select-option>
                  <a-select-option value="model-eval">模型评估</a-select-option>
                  <a-select-option value="model-deploy">模型部署</a-select-option>
                </a-select>
                <a-input v-model:value="step.command" placeholder="执行命令" class="step-command-input" />
                <MinusCircleOutlined v-if="addForm.steps.length > 1" class="dynamic-delete-button"
                  @click="removeStep(step)" />
              </div>
            </a-form-item>
            <a-form-item>
              <a-button type="dashed" class="add-dynamic-button" @click="addStep">
                <PlusOutlined />
                添加步骤
              </a-button>
            </a-form-item>
          </div>
  
          <div class="form-section">
            <div class="section-title">执行配置</div>
            <a-row :gutter="16">
              <a-col :span="8">
                <a-form-item label="并行执行" name="parallel">
                  <a-switch v-model:checked="addForm.parallel" />
                </a-form-item>
              </a-col>
              <a-col :span="8">
                <a-form-item label="失败重试次数" name="retry_count">
                  <a-input-number v-model:value="addForm.retry_count" :min="0" :max="5" class="full-width" />
                </a-form-item>
              </a-col>
              <a-col :span="8">
                <a-form-item label="超时时间(分钟)" name="timeout">
                  <a-input-number v-model:value="addForm.timeout" :min="1" :max="1440" class="full-width" />
                </a-form-item>
              </a-col>
            </a-row>
          </div>
        </a-form>
      </a-modal>
  
      <!-- 工作流详情模态框 -->
      <a-modal 
        title="工作流详情" 
        v-model:visible="isViewModalVisible" 
        @cancel="closeViewModal"
        :width="1000"
        class="custom-modal"
        :footer="null"
      >
        <div class="workflow-detail-container" v-if="viewWorkflow">
          <div class="detail-section">
            <div class="section-title">基本信息</div>
            <a-descriptions :column="2" size="small">
              <a-descriptions-item label="工作流名称">{{ viewWorkflow.name }}</a-descriptions-item>
              <a-descriptions-item label="工作流类型">
                <a-tag :color="getCategoryColor(viewWorkflow.category)">{{ getCategoryText(viewWorkflow.category) }}</a-tag>
              </a-descriptions-item>
              <a-descriptions-item label="状态">
                <a-tag :color="getStatusColor(viewWorkflow.status)">{{ getStatusText(viewWorkflow.status) }}</a-tag>
              </a-descriptions-item>
              <a-descriptions-item label="创建者">{{ viewWorkflow.creator }}</a-descriptions-item>
              <a-descriptions-item label="创建时间">{{ viewWorkflow.created_at }}</a-descriptions-item>
              <a-descriptions-item label="开始时间">{{ viewWorkflow.start_time || '未开始' }}</a-descriptions-item>
            </a-descriptions>
          </div>
  
          <div class="detail-section">
            <div class="section-title">执行进度</div>
            <div class="progress-detail">
              <a-progress 
                :percent="getProgressPercent(viewWorkflow)" 
                :status="getProgressStatus(viewWorkflow.status)"
              />
              <div class="progress-info">
                <span>已完成: {{ viewWorkflow.completed_steps }}/{{ viewWorkflow.total_steps }} 步骤</span>
                <span>运行时间: {{ formatDuration(viewWorkflow.start_time, viewWorkflow.completion_time) }}</span>
              </div>
            </div>
          </div>
  
          <div class="detail-section">
            <div class="section-title">工作流描述</div>
            <div class="description-content">
              {{ viewWorkflow.description || '无描述' }}
            </div>
          </div>
  
          <div class="detail-section">
            <div class="section-title">步骤详情</div>
            <div class="steps-timeline">
              <a-timeline>
                <a-timeline-item 
                  v-for="(step, index) in viewWorkflow.steps" 
                  :key="index"
                  :color="getStepColor(step.status)"
                >
                  <div class="step-item">
                    <div class="step-header">
                      <span class="step-name">{{ step.name }}</span>
                      <a-tag :color="getStepStatusColor(step.status)" size="small">
                        {{ getStepStatusText(step.status) }}
                      </a-tag>
                    </div>
                    <div class="step-details">
                      <div class="step-type">类型: {{ getStepTypeText(step.type) }}</div>
                      <div class="step-command">命令: {{ step.command }}</div>
                      <div class="step-duration" v-if="step.start_time">
                        执行时间: {{ formatDuration(step.start_time, step.completion_time) }}
                      </div>
                    </div>
                  </div>
                </a-timeline-item>
              </a-timeline>
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
    MinusCircleOutlined
  } from '@ant-design/icons-vue';
  import type { FormInstance } from 'ant-design-vue';
  
  interface WorkflowStep {
    name: string;
    type: string;
    command: string;
    status: string;
    start_time?: string;
    completion_time?: string;
    key: number;
  }
  
  interface WorkflowItem {
    id: number;
    name: string;
    category: string;
    status: string;
    description: string;
    total_steps: number;
    completed_steps: number;
    parallel: boolean;
    retry_count: number;
    timeout: number;
    steps: WorkflowStep[];
    created_at: string;
    start_time?: string;
    completion_time?: string;
    creator: string;
  }
  
  // 搜索和筛选
  const searchText = ref('');
  const statusFilter = ref('');
  const categoryFilter = ref('');
  
  // 表格数据
  const data = ref<WorkflowItem[]>([]);
  
  // 分页相关
  const pageSizeOptions = ref<string[]>(['10', '20', '30', '40', '50']);
  const current = ref(1);
  const pageSizeRef = ref(10);
  const total = ref(0);
  
  // 模态框状态
  const isAddModalVisible = ref(false);
  const isViewModalVisible = ref(false);
  
  // 表单引用
  const addFormRef = ref<FormInstance>();
  
  // 查看详情的工作流
  const viewWorkflow = ref<WorkflowItem | null>(null);
  
  // 步骤计数器
  let stepKeyCounter = 0;
  
  // 新增表单
  const addForm = reactive({
    name: '',
    category: 'training',
    description: '',
    parallel: false,
    retry_count: 1,
    timeout: 60,
    steps: [] as WorkflowStep[]
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
      title: '工作流名称',
      dataIndex: 'name',
      key: 'name',
      width: 200,
    },
    {
      title: '类型',
      dataIndex: 'category',
      key: 'category',
      slots: { customRender: 'category' },
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
      title: '执行进度',
      key: 'progress',
      slots: { customRender: 'progress' },
      width: 200,
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
      width: 250,
      fixed: 'right',
    },
  ];
  
  // 初始化数据
  onMounted(() => {
    initForms();
    loadData();
  });
  
  // 初始化表单
  const initForms = () => {
    addForm.steps = [{ 
      name: '', 
      type: 'data-load', 
      command: '', 
      status: 'pending', 
      key: ++stepKeyCounter 
    }];
  };
  
  // 获取状态颜色
  const getStatusColor = (status: string) => {
    const colorMap: Record<string, string> = {
      'Draft': 'default',
      'Running': 'blue',
      'Completed': 'green',
      'Failed': 'red',
      'Paused': 'orange'
    };
    return colorMap[status] || 'default';
  };
  
  // 获取状态文本
  const getStatusText = (status: string) => {
    const textMap: Record<string, string> = {
      'Draft': '草稿',
      'Running': '运行中',
      'Completed': '已完成',
      'Failed': '失败',
      'Paused': '已暂停'
    };
    return textMap[status] || status;
  };
  
  // 获取类型颜色
  const getCategoryColor = (category: string) => {
    const colorMap: Record<string, string> = {
      'training': 'blue',
      'inference': 'green',
      'preprocessing': 'orange',
      'analysis': 'purple'
    };
    return colorMap[category] || 'default';
  };
  
  // 获取类型文本
  const getCategoryText = (category: string) => {
    const textMap: Record<string, string> = {
      'training': '模型训练',
      'inference': '模型推理',
      'preprocessing': '数据预处理',
      'analysis': '数据分析'
    };
    return textMap[category] || category;
  };
  
  // 获取进度百分比
  const getProgressPercent = (workflow: WorkflowItem) => {
    if (workflow.total_steps === 0) return 0;
    return Math.round((workflow.completed_steps / workflow.total_steps) * 100);
  };
  
  // 获取进度状态
  const getProgressStatus = (status: string) => {
    if (status === 'Failed') return 'exception';
    if (status === 'Completed') return 'success';
    return 'active';
  };
  
  // 获取步骤颜色
  const getStepColor = (status: string) => {
    const colorMap: Record<string, string> = {
      'pending': 'gray',
      'running': 'blue',
      'completed': 'green',
      'failed': 'red'
    };
    return colorMap[status] || 'gray';
  };
  
  // 获取步骤状态颜色
  const getStepStatusColor = (status: string) => {
    return getStepColor(status);
  };
  
  // 获取步骤状态文本
  const getStepStatusText = (status: string) => {
    const textMap: Record<string, string> = {
      'pending': '等待中',
      'running': '运行中',
      'completed': '已完成',
      'failed': '失败'
    };
    return textMap[status] || status;
  };
  
  // 获取步骤类型文本
  const getStepTypeText = (type: string) => {
    const textMap: Record<string, string> = {
      'data-load': '数据加载',
      'data-process': '数据处理',
      'model-train': '模型训练',
      'model-eval': '模型评估',
      'model-deploy': '模型部署'
    };
    return textMap[type] || type;
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
    const mockData: WorkflowItem[] = [
      {
        id: 1,
        name: '图像分类模型训练流程',
        category: 'training',
        status: 'Running',
        description: '基于ResNet50的图像分类模型训练工作流',
        total_steps: 5,
        completed_steps: 3,
        parallel: false,
        retry_count: 2,
        timeout: 120,
        steps: [
          { name: '数据加载', type: 'data-load', command: 'python load_data.py', status: 'completed', start_time: '2024-06-09 10:00:00', completion_time: '2024-06-09 10:05:00', key: 1 },
          { name: '数据预处理', type: 'data-process', command: 'python preprocess.py', status: 'completed', start_time: '2024-06-09 10:05:00', completion_time: '2024-06-09 10:15:00', key: 2 },
          { name: '模型训练', type: 'model-train', command: 'python train.py --epochs 100', status: 'running', start_time: '2024-06-09 10:15:00', key: 3 },
          { name: '模型评估', type: 'model-eval', command: 'python evaluate.py', status: 'pending', key: 4 },
          { name: '模型部署', type: 'model-deploy', command: 'python deploy.py', status: 'pending', key: 5 }
        ],
        created_at: '2024-06-09 09:30:00',
        start_time: '2024-06-09 10:00:00',
        creator: 'admin'
      },
      {
        id: 2,
        name: '文本情感分析流程',
        category: 'analysis',
        status: 'Completed',
        description: '基于BERT的文本情感分析工作流',
        total_steps: 4,
        completed_steps: 4,
        parallel: true,
        retry_count: 1,
        timeout: 90,
        steps: [
          { name: '文本数据加载', type: 'data-load', command: 'python load_text.py', status: 'completed', start_time: '2024-06-09 08:00:00', completion_time: '2024-06-09 08:10:00', key: 1 },
          { name: '文本预处理', type: 'data-process', command: 'python text_preprocess.py', status: 'completed', start_time: '2024-06-09 08:10:00', completion_time: '2024-06-09 08:25:00', key: 2 },
          { name: 'BERT模型训练', type: 'model-train', command: 'python bert_train.py', status: 'completed', start_time: '2024-06-09 08:25:00', completion_time: '2024-06-09 09:15:00', key: 3 },
          { name: '情感分析评估', type: 'model-eval', command: 'python sentiment_eval.py', status: 'completed', start_time: '2024-06-09 09:15:00', completion_time: '2024-06-09 09:30:00', key: 4 }
        ],
        created_at: '2024-06-09 07:45:00',
        start_time: '2024-06-09 08:00:00',
        completion_time: '2024-06-09 09:30:00',
        creator: 'user1'
      },
      {
        id: 3,
        name: '数据预处理管道',
        category: 'preprocessing',
        status: 'Draft',
        description: '大规模数据清洗和特征工程流程',
        total_steps: 3,
        completed_steps: 0,
        parallel: false,
        retry_count: 1,
        timeout: 60,
        steps: [
          { name: '数据清洗', type: 'data-process', command: 'python clean_data.py', status: 'pending', key: 1 },
          { name: '特征工程', type: 'data-process', command: 'python feature_engineering.py', status: 'pending', key: 2 },
          { name: '数据验证', type: 'data-process', command: 'python validate_data.py', status: 'pending', key: 3 }
        ],
        created_at: '2024-06-09 11:00:00',
        creator: 'user2'
      }
    ];
    
    data.value = mockData;
    total.value = mockData.length;
  };
  
  // 搜索处理
  const handleSearch = () => {
    loadData();
    message.success('搜索完成');
  };
  
  // 重置处理
  const handleReset = () => {
    searchText.value = '';
    statusFilter.value = '';
    categoryFilter.value = '';
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
      category: 'training',
      description: '',
      parallel: false,
      retry_count: 1,
      timeout: 60,
      steps: [{ 
        name: '', 
        type: 'data-load', 
        command: '', 
        status: 'pending', 
        key: ++stepKeyCounter 
      }]
    });
    addFormRef.value?.resetFields();
  };
  
  // 新增工作流
  const handleAdd = async () => {
    try {
      await addFormRef.value?.validateFields();
      
      const validSteps = addForm.steps.filter(step => step.name && step.command);
      
      const newWorkflow = {
        ...addForm,
        id: data.value.length + 1,
        status: 'Draft',
        total_steps: validSteps.length,
        completed_steps: 0,
        steps: validSteps,
        created_at: new Date().toLocaleString(),
        creator: 'admin'
      };
  
      console.log('Creating workflow:', newWorkflow);
      
      data.value.unshift(newWorkflow as WorkflowItem);
      total.value++;
      
      message.success('工作流创建成功');
      closeAddModal();
    } catch (error) {
      console.error('Validation failed:', error);
    }
  };
  
  // 查看详情
  const handleView = (record: WorkflowItem) => {
    viewWorkflow.value = record;
    isViewModalVisible.value = true;
  };
  
  // 关闭详情模态框
  const closeViewModal = () => {
    isViewModalVisible.value = false;
    viewWorkflow.value = null;
  };
  
  // 编辑工作流
  const handleEdit = (record: WorkflowItem) => {
    message.info('编辑功能开发中');
  };
  
  // 启动工作流
  const handleStart = (record: WorkflowItem) => {
    Modal.confirm({
      title: '确认启动工作流',
      content: `确定要启动工作流 "${record.name}" 吗？`,
      onOk() {
        record.status = 'Running';
        record.start_time = new Date().toLocaleString();
        // 开始执行第一个步骤
        if (record.steps.length > 0) {
          record.steps[0]!.status = 'running';
          record.steps[0]!.start_time = record.start_time;
        }
        message.success('工作流已启动');
      },
    });
  };
  
  // 暂停工作流
  const handlePause = (record: WorkflowItem) => {
    Modal.confirm({
      title: '确认暂停工作流',
      content: `确定要暂停工作流 "${record.name}" 吗？`,
      onOk() {
        record.status = 'Paused';
        message.success('工作流已暂停');
      },
    });
  };
  
  // 继续工作流
  const handleResume = (record: WorkflowItem) => {
    Modal.confirm({
      title: '确认继续工作流',
      content: `确定要继续执行工作流 "${record.name}" 吗？`,
      onOk() {
        record.status = 'Running';
        message.success('工作流已继续');
      },
    });
  };
  
  // 停止工作流
  const handleStop = (record: WorkflowItem) => {
    Modal.confirm({
      title: '确认停止工作流',
      content: `确定要停止工作流 "${record.name}" 吗？`,
      onOk() {
        record.status = 'Failed';
        record.completion_time = new Date().toLocaleString();
        message.success('工作流已停止');
      },
    });
  };
  
  // 删除工作流
  const handleDelete = (record: WorkflowItem) => {
    Modal.confirm({
      title: '确认删除工作流',
      content: `确定要删除工作流 "${record.name}" 吗？此操作不可恢复。`,
      onOk() {
        const index = data.value.findIndex(item => item.id === record.id);
        if (index !== -1) {
          data.value.splice(index, 1);
          total.value--;
        }
        message.success('工作流已删除');
      },
    });
  };
  
  // 添加步骤
  const addStep = () => {
    addForm.steps.push({
      name: '',
      type: 'data-load',
      command: '',
      status: 'pending',
      key: ++stepKeyCounter
    });
  };
  
  // 删除步骤
  const removeStep = (item: WorkflowStep) => {
    const index = addForm.steps.indexOf(item);
    if (index !== -1) {
      addForm.steps.splice(index, 1);
    }
  };
  </script>
  
  <style scoped>
  .workflow-page {
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
  .category-filter {
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
  
  .status-tag,
  .category-tag {
    border-radius: 4px;
    font-size: 12px;
    font-weight: 500;
  }
  
  .progress-container {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  
  .progress-text {
    font-size: 12px;
    color: #64748b;
    text-align: center;
  }
  
  .duration-container {
    font-size: 12px;
    color: #4b5563;
    font-family: monospace;
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
  
  .step-input-group {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  
  .step-number {
    width: 24px;
    height: 24px;
    border-radius: 50%;
    background: #3b82f6;
    color: white;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 12px;
    font-weight: 600;
    flex-shrink: 0;
  }
  
  .step-name-input {
    width: 150px;
  }
  
  .step-type-select {
    width: 120px;
  }
  
  .step-command-input {
    flex: 1;
  }
  
  .dynamic-delete-button {
    color: #ef4444;
    cursor: pointer;
    font-size: 16px;
    flex-shrink: 0;
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
  
  .workflow-detail-container {
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
  
  .progress-detail {
    padding: 16px;
    background: #f8fafc;
    border-radius: 6px;
  }
  
  .progress-info {
    display: flex;
    justify-content: space-between;
    margin-top: 8px;
    font-size: 12px;
    color: #64748b;
  }
  
  .description-content {
    padding: 12px;
    background: #f8fafc;
    border-radius: 6px;
    color: #4b5563;
    font-size: 14px;
    line-height: 1.5;
  }
  
  .steps-timeline {
    margin-top: 16px;
  }
  
  .step-item {
    margin-bottom: 8px;
  }
  
  .step-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 8px;
  }
  
  .step-name {
    font-weight: 600;
    color: #1a202c;
  }
  
  .step-details {
    font-size: 12px;
    color: #64748b;
    line-height: 1.4;
  }
  
  .step-type,
  .step-command,
  .step-duration {
    margin-bottom: 4px;
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
    .category-filter {
      width: 100%;
      min-width: auto;
    }
    
    .action-buttons {
      justify-content: center;
    }
    
    .step-input-group {
      flex-direction: column;
      align-items: stretch;
      gap: 8px;
    }
    
    .step-name-input,
    .step-type-select {
      width: 100%;
    }
  }
  </style>