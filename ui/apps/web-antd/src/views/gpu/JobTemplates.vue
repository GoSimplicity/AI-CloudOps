<template>
    <div class="job-template-page">
      <!-- 页面标题区域 -->
      <div class="page-header">
        <h2 class="page-title">作业模板</h2>
        <p class="page-description">管理和配置训练作业模板，提高作业创建效率</p>
      </div>
  
      <!-- 查询和操作工具栏 -->
      <div class="dashboard-card custom-toolbar">
        <div class="search-filters">
          <a-input 
            v-model:value="searchText" 
            placeholder="请输入模板名称" 
            class="search-input"
          >
            <template #prefix>
              <SearchOutlined class="search-icon" />
            </template>
          </a-input>
          <a-select 
            v-model:value="categoryFilter" 
            placeholder="模板分类" 
            class="category-filter"
            allowClear
          >
            <a-select-option value="">全部分类</a-select-option>
            <a-select-option value="深度学习">深度学习</a-select-option>
            <a-select-option value="机器学习">机器学习</a-select-option>
            <a-select-option value="数据处理">数据处理</a-select-option>
            <a-select-option value="计算机视觉">计算机视觉</a-select-option>
            <a-select-option value="自然语言处理">自然语言处理</a-select-option>
          </a-select>
          <a-select 
            v-model:value="statusFilter" 
            placeholder="模板状态" 
            class="status-filter"
            allowClear
          >
            <a-select-option value="">全部状态</a-select-option>
            <a-select-option value="active">启用</a-select-option>
            <a-select-option value="inactive">禁用</a-select-option>
            <a-select-option value="draft">草稿</a-select-option>
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
            创建模板
          </a-button>
        </div>
      </div>
  
      <!-- 模板列表表格 -->
      <div class="dashboard-card table-container">
        <a-table 
          :columns="columns" 
          :data-source="data" 
          row-key="id" 
          :pagination="false"
          class="custom-table"
          :scroll="{ x: 1400 }"
        >
          <!-- 模板状态列 -->
          <template #status="{ record }">
            <a-tag :color="getStatusColor(record.status)" class="status-tag">
              {{ getStatusText(record.status) }}
            </a-tag>
          </template>
          
          <!-- 模板分类列 -->
          <template #category="{ record }">
            <a-tag color="blue" class="category-tag">
              {{ record.category }}
            </a-tag>
          </template>
          
          <!-- 资源配置列 -->
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
          
          <!-- 使用次数列 -->
          <template #usage="{ record }">
            <div class="usage-container">
              <span class="usage-count">{{ record.usage_count }}</span>
              <span class="usage-text">次</span>
            </div>
          </template>
          
          <!-- 操作列 -->
          <template #action="{ record }">
            <div class="action-column">
              <a-button type="primary" size="small" @click="handleView(record)">
                查看
              </a-button>
              <a-button type="default" size="small" @click="handleEdit(record)">
                编辑
              </a-button>
              <a-button type="default" size="small" @click="handleCopy(record)">
                复制
              </a-button>
              <a-button type="default" size="small" @click="handleUse(record)">
                使用
              </a-button>
              <a-button type="default" size="small" @click="handleDelete(record)" v-if="record.status !== 'active'">
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
  
      <!-- 创建模板模态框 -->
      <a-modal 
        title="创建作业模板" 
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
                <a-form-item label="模板名称" name="name" :rules="[{ required: true, message: '请输入模板名称' }]">
                  <a-input v-model:value="addForm.name" placeholder="请输入模板名称" />
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item label="模板分类" name="category" :rules="[{ required: true, message: '请选择分类' }]">
                  <a-select v-model:value="addForm.category" placeholder="请选择分类">
                    <a-select-option value="深度学习">深度学习</a-select-option>
                    <a-select-option value="机器学习">机器学习</a-select-option>
                    <a-select-option value="数据处理">数据处理</a-select-option>
                    <a-select-option value="计算机视觉">计算机视觉</a-select-option>
                    <a-select-option value="自然语言处理">自然语言处理</a-select-option>
                  </a-select>
                </a-form-item>
              </a-col>
            </a-row>
            <a-row :gutter="16">
              <a-col :span="24">
                <a-form-item label="模板描述" name="description">
                  <a-textarea v-model:value="addForm.description" placeholder="请输入模板描述" :rows="3" />
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
  
      <!-- 模板详情模态框 -->
      <a-modal 
        title="模板详情" 
        v-model:visible="isViewModalVisible" 
        @cancel="closeViewModal"
        :width="900"
        class="custom-modal"
        :footer="null"
      >
        <div class="template-detail-container" v-if="viewTemplate">
          <div class="detail-section">
            <div class="section-title">基本信息</div>
            <a-descriptions :column="2" size="small">
              <a-descriptions-item label="模板名称">{{ viewTemplate.name }}</a-descriptions-item>
              <a-descriptions-item label="模板分类">
                <a-tag color="blue">{{ viewTemplate.category }}</a-tag>
              </a-descriptions-item>
              <a-descriptions-item label="状态">
                <a-tag :color="getStatusColor(viewTemplate.status)">{{ getStatusText(viewTemplate.status) }}</a-tag>
              </a-descriptions-item>
              <a-descriptions-item label="使用次数">{{ viewTemplate.usage_count }} 次</a-descriptions-item>
              <a-descriptions-item label="创建者">{{ viewTemplate.creator }}</a-descriptions-item>
              <a-descriptions-item label="创建时间">{{ viewTemplate.created_at }}</a-descriptions-item>
              <a-descriptions-item label="描述" :span="2">{{ viewTemplate.description || '暂无描述' }}</a-descriptions-item>
            </a-descriptions>
          </div>
  
          <div class="detail-section">
            <div class="section-title">资源配置</div>
            <a-descriptions :column="3" size="small">
              <a-descriptions-item label="CPU需求">{{ viewTemplate.cpu_request }}</a-descriptions-item>
              <a-descriptions-item label="内存需求">{{ viewTemplate.memory_request }}</a-descriptions-item>
              <a-descriptions-item label="GPU需求">{{ viewTemplate.gpu_request }}</a-descriptions-item>
            </a-descriptions>
          </div>
  
          <div class="detail-section">
            <div class="section-title">容器配置</div>
            <a-descriptions :column="1" size="small">
              <a-descriptions-item label="镜像">{{ viewTemplate.image }}</a-descriptions-item>
              <a-descriptions-item label="启动命令">
                <pre class="command-pre">{{ viewTemplate.command }}</pre>
              </a-descriptions-item>
            </a-descriptions>
          </div>
  
          <div class="detail-section" v-if="viewTemplate.env_vars && viewTemplate.env_vars.length > 0">
            <div class="section-title">环境变量</div>
            <div class="env-list">
              <div class="env-item" v-for="env in viewTemplate.env_vars" :key="env">
                <span class="env-key">{{ env.split('=')[0] }}</span>
                <span class="env-separator">=</span>
                <span class="env-value">{{ env.split('=')[1] }}</span>
              </div>
            </div>
          </div>
  
          <div class="detail-section" v-if="viewTemplate.volumes && viewTemplate.volumes.length > 0">
            <div class="section-title">存储卷</div>
            <div class="volume-list">
              <div class="volume-item" v-for="volume in viewTemplate.volumes" :key="volume">
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
    MinusCircleOutlined
  } from '@ant-design/icons-vue';
  import type { FormInstance } from 'ant-design-vue';
  
  interface TemplateItem {
    id: number;
    name: string;
    category: string;
    description: string;
    status: string;
    image: string;
    command: string;
    cpu_request: string;
    memory_request: string;
    gpu_request: number;
    env_vars: string[];
    volumes: string[];
    usage_count: number;
    created_at: string;
    updated_at?: string;
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
  const categoryFilter = ref('');
  const statusFilter = ref('');
  
  // 表格数据
  const data = ref<TemplateItem[]>([]);
  
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
  
  // 查看详情的模板
  const viewTemplate = ref<TemplateItem | null>(null);
  
  // 新增表单
  const addForm = reactive({
    name: '',
    category: '',
    description: '',
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
      title: '模板名称',
      dataIndex: 'name',
      key: 'name',
      width: 200,
    },
    {
      title: '分类',
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
      width: 180,
    },
    {
      title: '使用次数',
      key: 'usage',
      slots: { customRender: 'usage' },
      width: 100,
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
      width: 280,
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
      'active': 'green',
      'inactive': 'orange',
      'draft': 'blue'
    };
    return colorMap[status] || 'default';
  };

  // 获取状态文本
  const getStatusText = (status: string) => {
    const textMap: Record<string, string> = {
      'active': '启用',
      'inactive': '禁用',
      'draft': '草稿'
    };
    return textMap[status] || status;
  };

  // 加载数据
  const loadData = () => {
    // 模拟数据
    const mockData: TemplateItem[] = [
      {
        id: 1,
        name: 'PyTorch深度学习基础模板',
        category: '深度学习',
        description: '基于PyTorch框架的深度学习训练模板，适用于图像分类、目标检测等任务',
        status: 'active',
        image: 'pytorch/pytorch:1.12.0-cuda11.3-cudnn8-devel',
        command: 'python train.py --epochs 100 --batch-size 32 --lr 0.001',
        cpu_request: '4',
        memory_request: '8Gi',
        gpu_request: 2,
        env_vars: ['CUDA_VISIBLE_DEVICES=0,1', 'PYTHONPATH=/workspace'],
        volumes: ['/data:/workspace/data', '/models:/workspace/models'],
        usage_count: 156,
        created_at: '2024-05-15 10:30:00',
        creator: 'admin'
      },
      {
        id: 2,
        name: 'TensorFlow图像分类模板',
        category: '计算机视觉',
        description: '使用TensorFlow进行图像分类任务的标准模板',
        status: 'active',
        image: 'tensorflow/tensorflow:2.8.0-gpu',
        command: 'python image_classification.py --dataset imagenet --model resnet50',
        cpu_request: '8',
        memory_request: '16Gi',
        gpu_request: 4,
        env_vars: ['TF_CPP_MIN_LOG_LEVEL=2', 'CUDA_VISIBLE_DEVICES=0,1,2,3'],
        volumes: ['/datasets:/data', '/checkpoints:/workspace/checkpoints'],
        usage_count: 89,
        created_at: '2024-05-20 14:20:00',
        creator: 'user1'
      },
      {
        id: 3,
        name: 'BERT文本分类模板',
        category: '自然语言处理',
        description: '基于BERT的文本分类和情感分析模板',
        status: 'active',
        image: 'huggingface/transformers-pytorch-gpu:latest',
        command: 'python finetune_bert.py --model bert-base-chinese --task classification',
        cpu_request: '4',
        memory_request: '8Gi',
        gpu_request: 2,
        env_vars: ['TRANSFORMERS_CACHE=/workspace/cache', 'HF_HOME=/workspace/huggingface'],
        volumes: ['/nlp-data:/workspace/data', '/bert-models:/workspace/models'],
        usage_count: 234,
        created_at: '2024-05-18 09:45:00',
        creator: 'user2'
      },
      {
        id: 4,
        name: 'Scikit-learn机器学习模板',
        category: '机器学习',
        description: '传统机器学习算法训练模板，支持分类、回归、聚类等任务',
        status: 'inactive',
        image: 'python:3.9-slim',
        command: 'python ml_training.py --algorithm random_forest --features auto',
        cpu_request: '2',
        memory_request: '4Gi',
        gpu_request: 0,
        env_vars: ['SKLEARN_ENABLE_RESOURCE_LIMITS=1'],
        volumes: ['/ml-data:/workspace/data'],
        usage_count: 67,
        created_at: '2024-05-10 16:00:00',
        creator: 'user3'
      },
      {
        id: 5,
        name: '数据预处理Pipeline模板',
        category: '数据处理',
        description: '大规模数据预处理和特征工程模板',
        status: 'draft',
        image: 'apache/spark-py:v3.2.0',
        command: 'spark-submit --master local[*] data_preprocessing.py',
        cpu_request: '8',
        memory_request: '32Gi',
        gpu_request: 0,
        env_vars: ['SPARK_DRIVER_MEMORY=16g', 'SPARK_EXECUTOR_MEMORY=8g'],
        volumes: ['/raw-data:/workspace/input', '/processed-data:/workspace/output'],
        usage_count: 23,
        created_at: '2024-06-01 11:30:00',
        creator: 'admin'
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
    categoryFilter.value = '';
    statusFilter.value = '';
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
      category: '',
      description: '',
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

  // 新增模板
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

      const newTemplate = {
        ...addForm,
        env_vars: envVars,
        volumes: volumes,
        id: data.value.length + 1,
        status: 'draft',
        usage_count: 0,
        created_at: new Date().toLocaleString(),
        creator: 'admin'
      };

      console.log('Creating template:', newTemplate);
      
      data.value.unshift(newTemplate as TemplateItem);
      total.value++;
      
      message.success('模板创建成功');
      closeAddModal();
    } catch (error) {
      console.error('Validation failed:', error);
    }
  };

  // 查看详情
  const handleView = (record: TemplateItem) => {
    viewTemplate.value = record;
    isViewModalVisible.value = true;
  };

  // 关闭详情模态框
  const closeViewModal = () => {
    isViewModalVisible.value = false;
    viewTemplate.value = null;
  };

  // 编辑模板
  const handleEdit = (record: TemplateItem) => {
    message.info(`编辑模板: ${record.name}`);
    // 这里可以打开编辑模态框或跳转到编辑页面
  };

  // 复制模板
  const handleCopy = (record: TemplateItem) => {
    const newTemplate = {
      ...record,
      id: data.value.length + 1,
      name: `${record.name} - 副本`,
      status: 'draft',
      usage_count: 0,
      created_at: new Date().toLocaleString(),
      creator: 'admin'
    };
    
    data.value.unshift(newTemplate);
    total.value++;
    
    message.success(`模板 "${record.name}" 复制成功`);
  };

  // 使用模板
  const handleUse = (record: TemplateItem) => {
    // 增加使用次数
    record.usage_count++;
    message.success(`正在使用模板 "${record.name}" 创建作业`);
    // 这里可以跳转到作业创建页面，并预填充模板数据
  };

  // 删除模板
  const handleDelete = (record: TemplateItem) => {
    Modal.confirm({
      title: '确认删除模板',
      content: `确定要删除模板 "${record.name}" 吗？此操作不可恢复。`,
      onOk() {
        const index = data.value.findIndex(item => item.id === record.id);
        if (index !== -1) {
          data.value.splice(index, 1);
          total.value--;
        }
        message.success('模板已删除');
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
</script>

<style scoped>
.job-template-page {
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

.category-filter,
.status-filter {
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

.usage-container {
  display: flex;
  align-items: center;
  gap: 4px;
}

.usage-count {
  font-size: 16px;
  font-weight: 600;
  color: #3b82f6;
}

.usage-text {
  font-size: 12px;
  color: #64748b;
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

.template-detail-container {
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
  .category-filter,
  .status-filter {
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