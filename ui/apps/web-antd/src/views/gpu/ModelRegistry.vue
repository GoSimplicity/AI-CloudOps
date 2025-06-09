<template>
    <div class="model-registry-page">
      <!-- 页面标题区域 -->
      <div class="page-header">
        <h2 class="page-title">模型注册</h2>
      </div>
  
      <!-- 查询和操作工具栏 -->
      <div class="dashboard-card custom-toolbar">
        <div class="search-filters">
          <a-input 
            v-model:value="searchText" 
            placeholder="请输入模型名称" 
            class="search-input"
          >
            <template #prefix>
              <SearchOutlined class="search-icon" />
            </template>
          </a-input>
          <a-select 
            v-model:value="statusFilter" 
            placeholder="模型状态" 
            class="status-filter"
            allowClear
          >
            <a-select-option value="">全部状态</a-select-option>
            <a-select-option value="draft">草稿</a-select-option>
            <a-select-option value="registered">已注册</a-select-option>
            <a-select-option value="published">已发布</a-select-option>
            <a-select-option value="deprecated">已废弃</a-select-option>
          </a-select>
          <a-select 
            v-model:value="categoryFilter" 
            placeholder="模型分类" 
            class="category-filter"
            allowClear
          >
            <a-select-option value="">全部分类</a-select-option>
            <a-select-option value="cv">计算机视觉</a-select-option>
            <a-select-option value="nlp">自然语言处理</a-select-option>
            <a-select-option value="speech">语音处理</a-select-option>
            <a-select-option value="recommendation">推荐系统</a-select-option>
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
            注册模型
          </a-button>
        </div>
      </div>
  
      <!-- 模型列表表格 -->
      <div class="dashboard-card table-container">
        <a-table 
          :columns="columns" 
          :data-source="data" 
          row-key="id" 
          :pagination="false"
          class="custom-table"
          :scroll="{ x: 1400 }"
        >
          <!-- 模型状态列 -->
          <template #status="{ record }">
            <a-tag :color="getStatusColor(record.status)" class="status-tag">
              {{ getStatusText(record.status) }}
            </a-tag>
          </template>
          
          <!-- 模型信息列 -->
          <template #modelInfo="{ record }">
            <div class="model-info-container">
              <div class="model-info-item">
                <span class="info-label">框架:</span>
                <span class="info-value">{{ record.framework }}</span>
              </div>
              <div class="model-info-item">
                <span class="info-label">版本:</span>
                <span class="info-value">{{ record.version }}</span>
              </div>
              <div class="model-info-item">
                <span class="info-label">大小:</span>
                <span class="info-value">{{ record.model_size }}</span>
              </div>
            </div>
          </template>
          
          <!-- 分类列 -->
          <template #category="{ record }">
            <a-tag :color="getCategoryColor(record.category)" class="category-tag">
              {{ getCategoryText(record.category) }}
            </a-tag>
          </template>
          
          <!-- 准确率列 -->
          <template #accuracy="{ record }">
            <div class="accuracy-container">
              <a-progress 
                :percent="record.accuracy" 
                :stroke-color="getAccuracyColor(record.accuracy)"
                size="small"
              />
            </div>
          </template>
          
          <!-- 操作列 -->
          <template #action="{ record }">
            <div class="action-column">
              <a-button type="primary" size="small" @click="handleView(record)">
                查看
              </a-button>
              <a-button type="default" size="small" @click="handleEdit(record)" v-if="record.status === 'draft'">
                编辑
              </a-button>
              <a-button type="default" size="small" @click="handlePublish(record)" v-if="record.status === 'registered'">
                发布
              </a-button>
              <a-button type="default" size="small" @click="handleDownload(record)">
                下载
              </a-button>
              <a-button type="default" size="small" @click="handleDelete(record)" v-if="['draft', 'deprecated'].includes(record.status)">
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
  
      <!-- 注册模型模态框 -->
      <a-modal 
        title="注册模型" 
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
                <a-form-item label="模型名称" name="name" :rules="[{ required: true, message: '请输入模型名称' }]">
                  <a-input v-model:value="addForm.name" placeholder="请输入模型名称" />
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item label="模型分类" name="category" :rules="[{ required: true, message: '请选择模型分类' }]">
                  <a-select v-model:value="addForm.category" placeholder="请选择模型分类">
                    <a-select-option value="cv">计算机视觉</a-select-option>
                    <a-select-option value="nlp">自然语言处理</a-select-option>
                    <a-select-option value="speech">语音处理</a-select-option>
                    <a-select-option value="recommendation">推荐系统</a-select-option>
                  </a-select>
                </a-form-item>
              </a-col>
            </a-row>
            <a-row :gutter="16">
              <a-col :span="12">
                <a-form-item label="框架类型" name="framework" :rules="[{ required: true, message: '请选择框架类型' }]">
                  <a-select v-model:value="addForm.framework" placeholder="请选择框架类型">
                    <a-select-option value="pytorch">PyTorch</a-select-option>
                    <a-select-option value="tensorflow">TensorFlow</a-select-option>
                    <a-select-option value="onnx">ONNX</a-select-option>
                    <a-select-option value="sklearn">Scikit-learn</a-select-option>
                  </a-select>
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item label="模型版本" name="version" :rules="[{ required: true, message: '请输入模型版本' }]">
                  <a-input v-model:value="addForm.version" placeholder="例如: v1.0.0" />
                </a-form-item>
              </a-col>
            </a-row>
            <a-row :gutter="16">
              <a-col :span="24">
                <a-form-item label="模型描述" name="description">
                  <a-textarea v-model:value="addForm.description" placeholder="请输入模型描述" :rows="3" />
                </a-form-item>
              </a-col>
            </a-row>
          </div>
  
          <div class="form-section">
            <div class="section-title">模型文件</div>
            <a-row :gutter="16">
              <a-col :span="24">
                <a-form-item label="模型文件路径" name="model_path" :rules="[{ required: true, message: '请输入模型文件路径' }]">
                  <a-input v-model:value="addForm.model_path" placeholder="例如: /models/resnet50.pth" />
                </a-form-item>
              </a-col>
            </a-row>
            <a-row :gutter="16">
              <a-col :span="12">
                <a-form-item label="输入形状" name="input_shape">
                  <a-input v-model:value="addForm.input_shape" placeholder="例如: [1, 3, 224, 224]" />
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item label="输出形状" name="output_shape">
                  <a-input v-model:value="addForm.output_shape" placeholder="例如: [1, 1000]" />
                </a-form-item>
              </a-col>
            </a-row>
          </div>
  
          <div class="form-section">
            <div class="section-title">性能指标</div>
            <a-row :gutter="16">
              <a-col :span="8">
                <a-form-item label="准确率(%)" name="accuracy">
                  <a-input-number v-model:value="addForm.accuracy" :min="0" :max="100" :precision="2" placeholder="准确率" class="full-width" />
                </a-form-item>
              </a-col>
              <a-col :span="8">
                <a-form-item label="推理时间(ms)" name="inference_time">
                  <a-input-number v-model:value="addForm.inference_time" :min="0" :precision="2" placeholder="推理时间" class="full-width" />
                </a-form-item>
              </a-col>
              <a-col :span="8">
                <a-form-item label="模型大小(MB)" name="model_size_mb">
                  <a-input-number v-model:value="addForm.model_size_mb" :min="0" :precision="2" placeholder="模型大小" class="full-width" />
                </a-form-item>
              </a-col>
            </a-row>
          </div>
  
          <div class="form-section">
            <div class="section-title">标签</div>
            <a-form-item v-for="(tag, index) in addForm.tags" :key="tag.key"
              :label="index === 0 ? '标签' : ''" :name="['tags', index, 'value']">
              <div class="tag-input-group">
                <a-input v-model:value="tag.value" placeholder="输入标签" class="tag-input" />
                <MinusCircleOutlined v-if="addForm.tags.length > 1" class="dynamic-delete-button"
                  @click="removeTag(tag)" />
              </div>
            </a-form-item>
            <a-form-item>
              <a-button type="dashed" class="add-dynamic-button" @click="addTag">
                <PlusOutlined />
                添加标签
              </a-button>
            </a-form-item>
          </div>
        </a-form>
      </a-modal>
  
      <!-- 编辑模型模态框 -->
      <a-modal 
        title="编辑模型" 
        v-model:visible="isEditModalVisible" 
        @ok="handleUpdate" 
        @cancel="closeEditModal"
        :width="800"
        class="custom-modal"
      >
        <a-form ref="editFormRef" :model="editForm" layout="vertical" class="custom-form">
          <!-- 编辑表单内容与添加表单相同，这里简化显示 -->
          <div class="form-section">
            <div class="section-title">基本信息</div>
            <a-row :gutter="16">
              <a-col :span="12">
                <a-form-item label="模型名称" name="name" :rules="[{ required: true, message: '请输入模型名称' }]">
                  <a-input v-model:value="editForm.name" placeholder="请输入模型名称" />
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item label="模型分类" name="category" :rules="[{ required: true, message: '请选择模型分类' }]">
                  <a-select v-model:value="editForm.category" placeholder="请选择模型分类">
                    <a-select-option value="cv">计算机视觉</a-select-option>
                    <a-select-option value="nlp">自然语言处理</a-select-option>
                    <a-select-option value="speech">语音处理</a-select-option>
                    <a-select-option value="recommendation">推荐系统</a-select-option>
                  </a-select>
                </a-form-item>
              </a-col>
            </a-row>
          </div>
        </a-form>
      </a-modal>
  
      <!-- 模型详情模态框 -->
      <a-modal 
        title="模型详情" 
        v-model:visible="isViewModalVisible" 
        @cancel="closeViewModal"
        :width="900"
        class="custom-modal"
        :footer="null"
      >
        <div class="model-detail-container" v-if="viewModel">
          <div class="detail-section">
            <div class="section-title">基本信息</div>
            <a-descriptions :column="2" size="small">
              <a-descriptions-item label="模型名称">{{ viewModel.name }}</a-descriptions-item>
              <a-descriptions-item label="分类">
                <a-tag :color="getCategoryColor(viewModel.category)">{{ getCategoryText(viewModel.category) }}</a-tag>
              </a-descriptions-item>
              <a-descriptions-item label="框架">{{ viewModel.framework }}</a-descriptions-item>
              <a-descriptions-item label="版本">{{ viewModel.version }}</a-descriptions-item>
              <a-descriptions-item label="状态">
                <a-tag :color="getStatusColor(viewModel.status)">{{ getStatusText(viewModel.status) }}</a-tag>
              </a-descriptions-item>
              <a-descriptions-item label="创建者">{{ viewModel.creator }}</a-descriptions-item>
              <a-descriptions-item label="创建时间">{{ viewModel.created_at }}</a-descriptions-item>
              <a-descriptions-item label="更新时间">{{ viewModel.updated_at }}</a-descriptions-item>
            </a-descriptions>
          </div>
  
          <div class="detail-section">
            <div class="section-title">性能指标</div>
            <a-descriptions :column="3" size="small">
              <a-descriptions-item label="准确率">
                <a-progress :percent="viewModel.accuracy" size="small" />
              </a-descriptions-item>
              <a-descriptions-item label="推理时间">{{ viewModel.inference_time }}ms</a-descriptions-item>
              <a-descriptions-item label="模型大小">{{ viewModel.model_size }}</a-descriptions-item>
            </a-descriptions>
          </div>
  
          <div class="detail-section">
            <div class="section-title">模型信息</div>
            <a-descriptions :column="1" size="small">
              <a-descriptions-item label="描述">{{ viewModel.description }}</a-descriptions-item>
              <a-descriptions-item label="文件路径">{{ viewModel.model_path }}</a-descriptions-item>
              <a-descriptions-item label="输入形状">{{ viewModel.input_shape }}</a-descriptions-item>
              <a-descriptions-item label="输出形状">{{ viewModel.output_shape }}</a-descriptions-item>
            </a-descriptions>
          </div>
  
          <div class="detail-section" v-if="viewModel.tags && viewModel.tags.length > 0">
            <div class="section-title">标签</div>
            <div class="tags-container">
              <a-tag v-for="tag in viewModel.tags" :key="tag" color="blue" class="model-tag">
                {{ tag }}
              </a-tag>
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
  
  interface ModelItem {
    id: number;
    name: string;
    category: string;
    framework: string;
    version: string;
    status: string;
    description: string;
    model_path: string;
    input_shape: string;
    output_shape: string;
    accuracy: number;
    inference_time: number;
    model_size: string;
    model_size_mb: number;
    tags: string[];
    created_at: string;
    updated_at: string;
    creator: string;
  }
  
  interface Tag {
    value: string;
    key: number;
  }
  
  // 搜索和筛选
  const searchText = ref('');
  const statusFilter = ref('');
  const categoryFilter = ref('');
  
  // 表格数据
  const data = ref<ModelItem[]>([]);
  
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
  
  // 查看详情的模型
  const viewModel = ref<ModelItem | null>(null);
  
  // 新增表单
  const addForm = reactive({
    name: '',
    category: '',
    framework: '',
    version: '',
    description: '',
    model_path: '',
    input_shape: '',
    output_shape: '',
    accuracy: 0,
    inference_time: 0,
    model_size_mb: 0,
    tags: [] as Tag[]
  });
  
  // 编辑表单
  const editForm = reactive({
    id: 0,
    name: '',
    category: '',
    framework: '',
    version: '',
    description: '',
    model_path: '',
    input_shape: '',
    output_shape: '',
    accuracy: 0,
    inference_time: 0,
    model_size_mb: 0,
    tags: [] as Tag[]
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
      title: '模型名称',
      dataIndex: 'name',
      key: 'name',
      width: 180,
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
      title: '模型信息',
      key: 'modelInfo',
      slots: { customRender: 'modelInfo' },
      width: 200,
    },
    {
      title: '准确率',
      dataIndex: 'accuracy',
      key: 'accuracy',
      slots: { customRender: 'accuracy' },
      width: 120,
    },
    {
      title: '推理时间',
      dataIndex: 'inference_time',
      key: 'inference_time',
      width: 100,
      customRender: ({ text }: { text: number }) => `${text}ms`
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      width: 200,
      ellipsis: true,
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
  
  // 标签计数器
  let tagKeyCounter = 0;
  
  // 初始化数据
  onMounted(() => {
    initForms();
    loadData();
  });
  
  // 初始化表单
  const initForms = () => {
    addForm.tags = [{ value: '', key: ++tagKeyCounter }];
  };
  
  // 获取状态颜色
  const getStatusColor = (status: string) => {
    const colorMap: Record<string, string> = {
      'draft': 'orange',
      'registered': 'blue',
      'published': 'green',
      'deprecated': 'red'
    };
    return colorMap[status] || 'default';
  };
  
  // 获取状态文本
  const getStatusText = (status: string) => {
    const textMap: Record<string, string> = {
      'draft': '草稿',
      'registered': '已注册',
      'published': '已发布',
      'deprecated': '已废弃'
    };
    return textMap[status] || status;
  };
  
  // 获取分类颜色
  const getCategoryColor = (category: string) => {
    const colorMap: Record<string, string> = {
      'cv': 'blue',
      'nlp': 'green',
      'speech': 'purple',
      'recommendation': 'orange'
    };
    return colorMap[category] || 'default';
  };
  
  // 获取分类文本
  const getCategoryText = (category: string) => {
    const textMap: Record<string, string> = {
      'cv': '计算机视觉',
      'nlp': '自然语言处理',
      'speech': '语音处理',
      'recommendation': '推荐系统'
    };
    return textMap[category] || category;
  };
  
  // 获取准确率颜色
  const getAccuracyColor = (accuracy: number) => {
    if (accuracy >= 90) return '#52c41a';
    if (accuracy >= 80) return '#faad14';
    return '#ff4d4f';
  };
  
  // 加载数据
  const loadData = () => {
    // 模拟数据
    const mockData: ModelItem[] = [
      {
        id: 1,
        name: 'ResNet-50-ImageNet',
        category: 'cv',
        framework: 'pytorch',
        version: 'v1.2.0',
        status: 'published',
        description: '基于ImageNet数据集训练的ResNet-50图像分类模型，支持1000类物体识别',
        model_path: '/models/resnet50/model.pth',
        input_shape: '[1, 3, 224, 224]',
        output_shape: '[1, 1000]',
        accuracy: 94.5,
        inference_time: 15.2,
        model_size: '97.8MB',
        model_size_mb: 97.8,
        tags: ['图像分类', 'ResNet', 'ImageNet'],
        created_at: '2024-06-01 10:30:00',
        updated_at: '2024-06-05 14:20:00',
        creator: 'admin'
      },
      {
        id: 2,
        name: 'BERT-Base-Chinese',
        category: 'nlp',
        framework: 'tensorflow',
        version: 'v2.1.0',
        status: 'registered',
        description: '中文BERT预训练模型，适用于各种中文自然语言处理任务',
        model_path: '/models/bert/chinese-bert-base',
        input_shape: '[1, 512]',
        output_shape: '[1, 768]',
        accuracy: 88.7,
        inference_time: 45.6,
        model_size: '412MB',
        model_size_mb: 412,
        tags: ['文本分类', 'BERT', '中文'],
        created_at: '2024-06-02 09:15:00',
        updated_at: '2024-06-06 11:30:00',
        creator: 'user1'
      },
      {
        id: 3,
        name: 'YOLOv8-Object-Detection',
        category: 'cv',
        framework: 'onnx',
        version: 'v1.0.5',
        status: 'draft',
        description: 'YOLOv8目标检测模型，支持80类目标检测',
        model_path: '/models/yolo/yolov8n.onnx',
        input_shape: '[1, 3, 640, 640]',
        output_shape: '[1, 25200, 85]',
        accuracy: 92.1,
        inference_time: 28.3,
        model_size: '6.2MB',
        model_size_mb: 6.2,
        tags: ['目标检测', 'YOLO', '实时'],
        created_at: '2024-06-03 16:45:00',
        updated_at: '2024-06-07 09:10:00',
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
      category: '',
      framework: '',
      version: '',
      description: '',
      model_path: '',
      input_shape: '',
      output_shape: '',
      accuracy: 0,
      inference_time: 0,
      model_size_mb: 0,
      tags: [{ value: '', key: ++tagKeyCounter }]
    });
    addFormRef.value?.resetFields();
  };
  
  // 新增模型
  const handleAdd = async () => {
    try {
      await addFormRef.value?.validateFields();
      
      const tags = addForm.tags
        .filter(tag => tag.value.trim())
        .map(tag => tag.value.trim());
  
      const newModel = {
        ...addForm,
        tags: tags,
        id: data.value.length + 1,
        status: 'draft',
        model_size: `${addForm.model_size_mb}MB`,
        created_at: new Date().toLocaleString(),
        updated_at: new Date().toLocaleString(),
        creator: 'admin'
      };
  
      console.log('Creating model:', newModel);
      
      data.value.unshift(newModel as ModelItem);
      total.value++;
      
      message.success('模型注册成功');
      closeAddModal();
    } catch (error) {
      console.error('Validation failed:', error);
    }
  };
  
  // 查看详情
  const handleView = (record: ModelItem) => {
    viewModel.value = record;
    isViewModalVisible.value = true;
  };
  
  // 关闭详情模态框
  const closeViewModal = () => {
    isViewModalVisible.value = false;
    viewModel.value = null;
  };
  
  // 编辑模型
  const handleEdit = (record: ModelItem) => {
    Object.assign(editForm, {
      ...record,
      tags: record.tags.map(tag => ({ value: tag, key: ++tagKeyCounter }))
    });
    
    if (editForm.tags.length === 0) {
      editForm.tags.push({ value: '', key: ++tagKeyCounter });
    }
    
    isEditModalVisible.value = true;
  };
  
  // 关闭编辑模态框
  const closeEditModal = () => {
    isEditModalVisible.value = false;
  };
  
  // 更新模型
  const handleUpdate = async () => {
    try {
      await editFormRef.value?.validateFields();
      
      const tags = editForm.tags
        .filter(tag => tag.value.trim())
        .map(tag => tag.value.trim());
  
      const index = data.value.findIndex(item => item.id === editForm.id);
      if (index !== -1) {
        Object.assign(data.value[index] as ModelItem, {
          ...editForm,
          tags: tags,
          model_size: `${editForm.model_size_mb}MB`,
          updated_at: new Date().toLocaleString()
        });
      }
  
      message.success('模型更新成功');
      closeEditModal();
    } catch (error) {
      console.error('Validation failed:', error);
    }
  };
  
  // 发布模型
  const handlePublish = (record: ModelItem) => {
    Modal.confirm({
      title: '确认发布模型',
      content: `确定要发布模型 "${record.name}" 吗？`,
      onOk() {
        record.status = 'published';
        record.updated_at = new Date().toLocaleString();
        message.success('模型已发布');
      },
    });
  };
  
  // 下载模型
  const handleDownload = (record: ModelItem) => {
    message.info(`开始下载模型: ${record.name}`);
    // 这里应该实现真实的下载逻辑
  };
  
  // 删除模型
  const handleDelete = (record: ModelItem) => {
    Modal.confirm({
      title: '确认删除模型',
      content: `确定要删除模型 "${record.name}" 吗？此操作不可恢复。`,
      onOk() {
        const index = data.value.findIndex(item => item.id === record.id);
        if (index !== -1) {
          data.value.splice(index, 1);
          total.value--;
        }
        message.success('模型已删除');
      },
    });
  };
  
  // 添加标签
  const addTag = () => {
    addForm.tags.push({
      value: '',
      key: ++tagKeyCounter
    });
  };
  
  // 删除标签
  const removeTag = (item: Tag) => {
    const index = addForm.tags.indexOf(item);
    if (index !== -1) {
      addForm.tags.splice(index, 1);
    }
  };
  </script>
  
  <style scoped>
  .model-registry-page {
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
  
  .model-info-container {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  
  .model-info-item {
    display: flex;
    align-items: center;
    gap: 4px;
    font-size: 12px;
  }
  
  .info-label {
    color: #64748b;
    font-weight: 500;
    min-width: 35px;
  }
  
  .info-value {
    color: #1a202c;
    font-weight: 500;
  }
  
  .accuracy-container {
    width: 100px;
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
  
  .tag-input-group {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  
  .tag-input {
    flex: 1;
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
  
  .model-detail-container {
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
  
  .tags-container {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
  }
  
  .model-tag {
    margin: 0;
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
  }
  </style>