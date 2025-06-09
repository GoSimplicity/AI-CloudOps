<template>
    <div class="data-management-page">
      <!-- 页面标题区域 -->
      <div class="page-header">
        <h2 class="page-title">数据管理</h2>
        <div class="page-description">管理训练数据集、模型文件和实验结果</div>
      </div>
  
      <!-- 查询和操作工具栏 -->
      <div class="dashboard-card custom-toolbar">
        <div class="search-filters">
          <a-input 
            v-model:value="searchText" 
            placeholder="请输入数据集名称" 
            class="search-input"
          >
            <template #prefix>
              <SearchOutlined class="search-icon" />
            </template>
          </a-input>
          <a-select 
            v-model:value="typeFilter" 
            placeholder="数据类型" 
            class="type-filter"
            allowClear
          >
            <a-select-option value="">全部类型</a-select-option>
            <a-select-option value="dataset">数据集</a-select-option>
            <a-select-option value="model">模型文件</a-select-option>
            <a-select-option value="result">实验结果</a-select-option>
            <a-select-option value="log">日志文件</a-select-option>
          </a-select>
          <a-select 
            v-model:value="statusFilter" 
            placeholder="状态" 
            class="status-filter"
            allowClear
          >
            <a-select-option value="">全部状态</a-select-option>
            <a-select-option value="available">可用</a-select-option>
            <a-select-option value="processing">处理中</a-select-option>
            <a-select-option value="error">错误</a-select-option>
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
          <a-button type="primary" class="add-button" @click="showUploadModal">
            <template #icon>
              <UploadOutlined />
            </template>
            上传数据
          </a-button>
          <a-button class="add-button" @click="showCreateModal">
            <template #icon>
              <PlusOutlined />
            </template>
            创建数据集
          </a-button>
        </div>
      </div>
  
      <!-- 数据列表表格 -->
      <div class="dashboard-card table-container">
        <a-table 
          :columns="columns" 
          :data-source="data" 
          row-key="id" 
          :pagination="false"
          class="custom-table"
          :scroll="{ x: 1200 }"
        >
          <!-- 数据类型列 -->
          <template #type="{ record }">
            <a-tag :color="getTypeColor(record.type)" class="type-tag">
              <template #icon>
                <component :is="getTypeIcon(record.type)" />
              </template>
              {{ getTypeText(record.type) }}
            </a-tag>
          </template>
          
          <!-- 状态列 -->
          <template #status="{ record }">
            <a-tag :color="getStatusColor(record.status)" class="status-tag">
              {{ getStatusText(record.status) }}
            </a-tag>
          </template>
          
          <!-- 大小列 -->
          <template #size="{ record }">
            <div class="size-container">
              {{ formatSize(record.size) }}
            </div>
          </template>
          
          <!-- 路径列 -->
          <template #path="{ record }">
            <a-tooltip :title="record.path">
              <div class="path-container">
                {{ record.path }}
              </div>
            </a-tooltip>
          </template>
          
          <!-- 标签列 -->
          <template #tags="{ record }">
            <div class="tags-container">
              <a-tag 
                v-for="tag in record.tags" 
                :key="tag" 
                class="data-tag"
                :color="getTagColor(tag)"
              >
                {{ tag }}
              </a-tag>
            </div>
          </template>
          
          <!-- 操作列 -->
          <template #action="{ record }">
            <div class="action-column">
              <a-button type="primary" size="small" @click="handleView(record)">
                查看
              </a-button>
              <a-button type="default" size="small" @click="handleDownload(record)">
                下载
              </a-button>
              <a-button type="default" size="small" @click="handleEdit(record)">
                编辑
              </a-button>
              <a-button type="default" size="small" @click="handleDelete(record)" danger>
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
  
      <!-- 上传数据模态框 -->
      <a-modal 
        title="上传数据" 
        v-model:visible="isUploadModalVisible" 
        @ok="handleUpload" 
        @cancel="closeUploadModal"
        :width="600"
        class="custom-modal"
      >
        <a-form ref="uploadFormRef" :model="uploadForm" layout="vertical" class="custom-form">
          <div class="form-section">
            <div class="section-title">基本信息</div>
            <a-row :gutter="16">
              <a-col :span="12">
                <a-form-item label="数据名称" name="name" :rules="[{ required: true, message: '请输入数据名称' }]">
                  <a-input v-model:value="uploadForm.name" placeholder="请输入数据名称" />
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item label="数据类型" name="type" :rules="[{ required: true, message: '请选择数据类型' }]">
                  <a-select v-model:value="uploadForm.type" placeholder="请选择数据类型">
                    <a-select-option value="dataset">数据集</a-select-option>
                    <a-select-option value="model">模型文件</a-select-option>
                    <a-select-option value="result">实验结果</a-select-option>
                    <a-select-option value="log">日志文件</a-select-option>
                  </a-select>
                </a-form-item>
              </a-col>
            </a-row>
            <a-row :gutter="16">
              <a-col :span="24">
                <a-form-item label="描述" name="description">
                  <a-textarea v-model:value="uploadForm.description" placeholder="请输入数据描述" :rows="3" />
                </a-form-item>
              </a-col>
            </a-row>
          </div>
  
          <div class="form-section">
            <div class="section-title">文件上传</div>
            <a-form-item label="选择文件" name="files">
              <a-upload
                v-model:file-list="uploadForm.files"
                name="file"
                multiple
                :before-upload="beforeUpload"
                @remove="handleRemove"
              >
                <a-button>
                  <template #icon>
                    <UploadOutlined />
                  </template>
                  选择文件
                </a-button>
              </a-upload>
            </a-form-item>
          </div>
  
          <div class="form-section">
            <div class="section-title">标签</div>
            <a-form-item label="标签" name="tags">
              <a-select
                v-model:value="uploadForm.tags"
                mode="tags"
                placeholder="添加标签"
                :token-separators="[',']"
              />
            </a-form-item>
          </div>
        </a-form>
      </a-modal>
  
      <!-- 创建数据集模态框 -->
      <a-modal 
        title="创建数据集" 
        v-model:visible="isCreateModalVisible" 
        @ok="handleCreate" 
        @cancel="closeCreateModal"
        :width="700"
        class="custom-modal"
      >
        <a-form ref="createFormRef" :model="createForm" layout="vertical" class="custom-form">
          <div class="form-section">
            <div class="section-title">基本信息</div>
            <a-row :gutter="16">
              <a-col :span="12">
                <a-form-item label="数据集名称" name="name" :rules="[{ required: true, message: '请输入数据集名称' }]">
                  <a-input v-model:value="createForm.name" placeholder="请输入数据集名称" />
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item label="版本" name="version">
                  <a-input v-model:value="createForm.version" placeholder="例如: v1.0" />
                </a-form-item>
              </a-col>
            </a-row>
            <a-row :gutter="16">
              <a-col :span="24">
                <a-form-item label="描述" name="description">
                  <a-textarea v-model:value="createForm.description" placeholder="请输入数据集描述" :rows="3" />
                </a-form-item>
              </a-col>
            </a-row>
          </div>
  
          <div class="form-section">
            <div class="section-title">存储配置</div>
            <a-row :gutter="16">
              <a-col :span="12">
                <a-form-item label="存储路径" name="path" :rules="[{ required: true, message: '请输入存储路径' }]">
                  <a-input v-model:value="createForm.path" placeholder="/data/datasets/my-dataset" />
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item label="存储类型" name="storage_type">
                  <a-select v-model:value="createForm.storage_type" placeholder="请选择存储类型">
                    <a-select-option value="local">本地存储</a-select-option>
                    <a-select-option value="nfs">NFS</a-select-option>
                    <a-select-option value="s3">S3对象存储</a-select-option>
                    <a-select-option value="hdfs">HDFS</a-select-option>
                  </a-select>
                </a-form-item>
              </a-col>
            </a-row>
          </div>
  
          <div class="form-section">
            <div class="section-title">访问控制</div>
            <a-row :gutter="16">
              <a-col :span="12">
                <a-form-item label="访问权限" name="access_level">
                  <a-select v-model:value="createForm.access_level" placeholder="请选择访问权限">
                    <a-select-option value="public">公开</a-select-option>
                    <a-select-option value="private">私有</a-select-option>
                    <a-select-option value="team">团队</a-select-option>
                  </a-select>
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item label="所属团队" name="team">
                  <a-select v-model:value="createForm.team" placeholder="请选择团队">
                    <a-select-option value="ml-team">机器学习团队</a-select-option>
                    <a-select-option value="nlp-team">自然语言处理团队</a-select-option>
                    <a-select-option value="cv-team">计算机视觉团队</a-select-option>
                  </a-select>
                </a-form-item>
              </a-col>
            </a-row>
          </div>
  
          <div class="form-section">
            <div class="section-title">标签和元数据</div>
            <a-row :gutter="16">
              <a-col :span="24">
                <a-form-item label="标签" name="tags">
                  <a-select
                    v-model:value="createForm.tags"
                    mode="tags"
                    placeholder="添加标签"
                    :token-separators="[',']"
                  />
                </a-form-item>
              </a-col>
            </a-row>
          </div>
        </a-form>
      </a-modal>
  
      <!-- 数据详情模态框 -->
      <a-modal 
        title="数据详情" 
        v-model:visible="isViewModalVisible" 
        @cancel="closeViewModal"
        :width="800"
        class="custom-modal"
        :footer="null"
      >
        <div class="data-detail-container" v-if="viewData">
          <div class="detail-section">
            <div class="section-title">基本信息</div>
            <a-descriptions :column="2" size="small">
              <a-descriptions-item label="数据名称">{{ viewData.name }}</a-descriptions-item>
              <a-descriptions-item label="数据类型">
                <a-tag :color="getTypeColor(viewData.type)">{{ getTypeText(viewData.type) }}</a-tag>
              </a-descriptions-item>
              <a-descriptions-item label="状态">
                <a-tag :color="getStatusColor(viewData.status)">{{ getStatusText(viewData.status) }}</a-tag>
              </a-descriptions-item>
              <a-descriptions-item label="大小">{{ formatSize(viewData.size) }}</a-descriptions-item>
              <a-descriptions-item label="创建时间">{{ viewData.created_at }}</a-descriptions-item>
              <a-descriptions-item label="更新时间">{{ viewData.updated_at }}</a-descriptions-item>
              <a-descriptions-item label="创建者">{{ viewData.creator }}</a-descriptions-item>
              <a-descriptions-item label="访问权限">{{ getAccessText(viewData.access_level) }}</a-descriptions-item>
            </a-descriptions>
          </div>
  
          <div class="detail-section">
            <div class="section-title">存储信息</div>
            <a-descriptions :column="1" size="small">
              <a-descriptions-item label="存储路径">
                <code class="path-code">{{ viewData.path }}</code>
              </a-descriptions-item>
              <a-descriptions-item label="存储类型">{{ getStorageTypeText(viewData.storage_type) }}</a-descriptions-item>
            </a-descriptions>
          </div>
  
          <div class="detail-section" v-if="viewData.description">
            <div class="section-title">描述</div>
            <div class="description-content">
              {{ viewData.description }}
            </div>
          </div>
  
          <div class="detail-section" v-if="viewData.tags && viewData.tags.length > 0">
            <div class="section-title">标签</div>
            <div class="tags-container">
              <a-tag 
                v-for="tag in viewData.tags" 
                :key="tag" 
                class="data-tag"
                :color="getTagColor(tag)"
              >
                {{ tag }}
              </a-tag>
            </div>
          </div>
  
          <div class="detail-section" v-if="viewData.metadata">
            <div class="section-title">元数据</div>
            <pre class="metadata-pre">{{ JSON.stringify(viewData.metadata, null, 2) }}</pre>
          </div>
        </div>
      </a-modal>
    </div>
  </template>
  
  <script lang="ts" setup>
  import { ref, reactive, onMounted, h } from 'vue';
  import { message, Modal } from 'ant-design-vue';
  import {
    SearchOutlined,
    ReloadOutlined,
    PlusOutlined,
    UploadOutlined,
    DatabaseOutlined,
    FileOutlined,
    LineChartOutlined,
    FileTextOutlined
  } from '@ant-design/icons-vue';
  import type { FormInstance, UploadProps } from 'ant-design-vue';
  
  interface DataItem {
    id: number;
    name: string;
    type: string;
    status: string;
    size: number;
    path: string;
    description?: string;
    tags: string[];
    created_at: string;
    updated_at: string;
    creator: string;
    access_level: string;
    storage_type: string;
    metadata?: any;
  }
  
  // 搜索和筛选
  const searchText = ref('');
  const typeFilter = ref('');
  const statusFilter = ref('');
  
  // 表格数据
  const data = ref<DataItem[]>([]);
  
  // 分页相关
  const pageSizeOptions = ref<string[]>(['10', '20', '30', '40', '50']);
  const current = ref(1);
  const pageSizeRef = ref(10);
  const total = ref(0);
  
  // 模态框状态
  const isUploadModalVisible = ref(false);
  const isCreateModalVisible = ref(false);
  const isViewModalVisible = ref(false);
  
  // 表单引用
  const uploadFormRef = ref<FormInstance>();
  const createFormRef = ref<FormInstance>();
  
  // 查看详情的数据
  const viewData = ref<DataItem | null>(null);
  
  // 上传表单
  const uploadForm = reactive({
    name: '',
    type: '',
    description: '',
    files: [] as any[],
    tags: [] as string[]
  });
  
  // 创建表单
  const createForm = reactive({
    name: '',
    version: 'v1.0',
    description: '',
    path: '',
    storage_type: 'local',
    access_level: 'private',
    team: '',
    tags: [] as string[]
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
      title: '数据名称',
      dataIndex: 'name',
      key: 'name',
      width: 180,
    },
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      slots: { customRender: 'type' },
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
      title: '大小',
      key: 'size',
      slots: { customRender: 'size' },
      width: 100,
    },
    {
      title: '存储路径',
      dataIndex: 'path',
      key: 'path',
      slots: { customRender: 'path' },
      width: 200,
    },
    {
      title: '标签',
      key: 'tags',
      slots: { customRender: 'tags' },
      width: 150,
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
      width: 220,
      fixed: 'right',
    },
  ];
  
  // 初始化数据
  onMounted(() => {
    loadData();
  });
  
  // 获取类型颜色
  const getTypeColor = (type: string) => {
    const colorMap: Record<string, string> = {
      'dataset': 'blue',
      'model': 'green',
      'result': 'orange',
      'log': 'purple'
    };
    return colorMap[type] || 'default';
  };
  
  // 获取类型文本
  const getTypeText = (type: string) => {
    const textMap: Record<string, string> = {
      'dataset': '数据集',
      'model': '模型文件',
      'result': '实验结果',
      'log': '日志文件'
    };
    return textMap[type] || type;
  };
  
  // 获取类型图标
  const getTypeIcon = (type: string) => {
    const iconMap: Record<string, any> = {
      'dataset': DatabaseOutlined,
      'model': FileOutlined,
      'result': LineChartOutlined,
      'log': FileTextOutlined
    };
    return iconMap[type] || FileOutlined;
  };
  
  // 获取状态颜色
  const getStatusColor = (status: string) => {
    const colorMap: Record<string, string> = {
      'available': 'green',
      'processing': 'blue',
      'error': 'red'
    };
    return colorMap[status] || 'default';
  };
  
  // 获取状态文本
  const getStatusText = (status: string) => {
    const textMap: Record<string, string> = {
      'available': '可用',
      'processing': '处理中',
      'error': '错误'
    };
    return textMap[status] || status;
  };
  
  // 获取访问权限文本
  const getAccessText = (access: string) => {
    const textMap: Record<string, string> = {
      'public': '公开',
      'private': '私有',
      'team': '团队'
    };
    return textMap[access] || access;
  };
  
  // 获取存储类型文本
  const getStorageTypeText = (type: string) => {
    const textMap: Record<string, string> = {
      'local': '本地存储',
      'nfs': 'NFS',
      's3': 'S3对象存储',
      'hdfs': 'HDFS'
    };
    return textMap[type] || type;
  };
  
  // 格式化文件大小
  const formatSize = (bytes: number) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };
  
  // 获取标签颜色
  const getTagColor = (tag: string) => {
    const colors = ['blue', 'green', 'orange', 'red', 'purple', 'cyan'];
    let hash = 0;
    for (let i = 0; i < tag.length; i++) {
      hash = tag.charCodeAt(i) + ((hash << 5) - hash);
    }
    return colors[Math.abs(hash) % colors.length];
  };
  
  // 加载数据
  const loadData = () => {
    // 模拟数据
    const mockData: DataItem[] = [
      {
        id: 1,
        name: 'CIFAR-10数据集',
        type: 'dataset',
        status: 'available',
        size: 162000000,
        path: '/data/datasets/cifar10',
        description: 'CIFAR-10数据集包含60000张32x32像素的彩色图像，分为10个类别',
        tags: ['图像分类', '计算机视觉', '基准数据集'],
        created_at: '2024-06-01 10:00:00',
        updated_at: '2024-06-01 10:00:00',
        creator: 'admin',
        access_level: 'public',
        storage_type: 'local',
        metadata: {
          num_classes: 10,
          num_samples: 60000,
          image_size: [32, 32, 3],
          format: 'numpy'
        }
      },
      {
        id: 2,
        name: 'ResNet-50预训练模型',
        type: 'model',
        status: 'available',
        size: 98000000,
        path: '/data/models/resnet50_pretrained.pth',
        description: 'ImageNet预训练的ResNet-50模型',
        tags: ['预训练模型', 'ResNet', '图像分类'],
        created_at: '2024-06-02 14:30:00',
        updated_at: '2024-06-02 14:30:00',
        creator: 'user1',
        access_level: 'team',
        storage_type: 'local',
        metadata: {
          framework: 'PyTorch',
          accuracy: 0.764,
          parameters: 25000000
        }
      },
      {
        id: 3,
        name: '实验结果-20240609',
        type: 'result',
        status: 'processing',
        size: 15000000,
        path: '/data/results/exp_20240609',
        description: 'BERT模型在文本分类任务上的实验结果',
        tags: ['BERT', '文本分类', '实验结果'],
        created_at: '2024-06-09 09:15:00',
        updated_at: '2024-06-09 11:30:00',
        creator: 'user2',
        access_level: 'private',
        storage_type: 'local',
        metadata: {
          model: 'bert-base-uncased',
          task: 'text_classification',
          f1_score: 0.892
        }
      },
      {
        id: 4,
        name: '训练日志-pytorch-job-001',
        type: 'log',
        status: 'available',
        size: 5200000,
        path: '/data/logs/pytorch_job_001.log',
        description: 'PyTorch训练作业的详细日志',
        tags: ['训练日志', 'PyTorch', '调试'],
        created_at: '2024-06-09 10:30:00',
        updated_at: '2024-06-09 12:00:00',
        creator: 'admin',
        access_level: 'team',
        storage_type: 'local'
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
    typeFilter.value = '';
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
  
  // 显示上传模态框
  const showUploadModal = () => {
    resetUploadForm();
    isUploadModalVisible.value = true;
  };
  
  // 关闭上传模态框
  const closeUploadModal = () => {
    isUploadModalVisible.value = false;
    resetUploadForm();
  };
  
  // 重置上传表单
  const resetUploadForm = () => {
    Object.assign(uploadForm, {
      name: '',
      type: '',
      description: '',
      files: [],
      tags: []
    });
    uploadFormRef.value?.resetFields();
  };
  
  // 上传前处理
  const beforeUpload: UploadProps['beforeUpload'] = (file) => {
    const isLt2G = file.size! / 1024 / 1024 / 1024 < 2;
    if (!isLt2G) {
      message.error('文件大小不能超过2GB!');
    }
    return false; // 阻止自动上传
  };
  
  // 移除文件
  const handleRemove = (file: any) => {
    const index = uploadForm.files.indexOf(file);
    const newFileList = uploadForm.files.slice();
    newFileList.splice(index, 1);
    uploadForm.files = newFileList;
  };
  
  // 上传数据
  const handleUpload = async () => {
    try {
      await uploadFormRef.value?.validateFields();
      
      if (uploadForm.files.length === 0) {
        message.error('请选择要上传的文件');
        return;
      }
  
      // 模拟上传过程
      message.loading('正在上传数据...', 2);
      
      setTimeout(() => {
        message.success('数据上传成功');
        closeUploadModal();
        loadData(); // 重新加载数据
      }, 2000);
    } catch (error) {
      console.error('上传验证失败:', error);
    }
  };
  
  // 显示创建模态框
  const showCreateModal = () => {
    resetCreateForm();
    isCreateModalVisible.value = true;
  };
  
  // 关闭创建模态框
  const closeCreateModal = () => {
    isCreateModalVisible.value = false;
    resetCreateForm();
  };
  
  // 重置创建表单
  const resetCreateForm = () => {
    Object.assign(createForm, {
      name: '',
      version: 'v1.0',
      description: '',
      path: '',
      storage_type: 'local',
      access_level: 'private',
      team: '',
      tags: []
    });
    createFormRef.value?.resetFields();
  };
  
  // 创建数据集
  const handleCreate = async () => {
    try {
      await createFormRef.value?.validateFields();
      
      // 模拟创建过程
      message.loading('正在创建数据集...', 2);
      
      setTimeout(() => {
        message.success('数据集创建成功');
        closeCreateModal();
        loadData(); // 重新加载数据
      }, 2000);
    } catch (error) {
      console.error('创建验证失败:', error);
    }
  };
  
  // 查看详情
  const handleView = (record: DataItem) => {
    viewData.value = record;
    isViewModalVisible.value = true;
  };
  
  // 关闭详情模态框
  const closeViewModal = () => {
    isViewModalVisible.value = false;
    viewData.value = null;
  };
  
  // 下载数据
  const handleDownload = (record: DataItem) => {
    message.loading('正在准备下载...', 1);
    setTimeout(() => {
      message.success(`开始下载 ${record.name}`);
      // 实际项目中这里会处理文件下载逻辑
    }, 1000);
  };
  
  // 编辑数据
  const handleEdit = (record: DataItem) => {
    message.info('编辑功能开发中...');
  };
  
  // 删除数据
  const handleDelete = (record: DataItem) => {
    Modal.confirm({
      title: '确认删除',
      content: `确定要删除数据 "${record.name}" 吗？此操作不可恢复。`,
      okText: '确认',
      cancelText: '取消',
      okType: 'danger',
      onOk() {
        message.loading('正在删除...', 1);
        setTimeout(() => {
          message.success('删除成功');
          loadData(); // 重新加载数据
        }, 1000);
      },
    });
  };
  </script>
  
  <style scoped>
  .data-management-page {
    padding: 20px;
    background-color: #f5f5f5;
    min-height: 100vh;
  }
  
  .page-header {
    margin-bottom: 24px;
  }
  
  .page-title {
    font-size: 24px;
    font-weight: 600;
    color: #262626;
    margin: 0 0 8px 0;
  }
  
  .page-description {
    color: #8c8c8c;
    font-size: 14px;
  }
  
  .dashboard-card {
    background: white;
    border-radius: 8px;
    padding: 20px;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
    margin-bottom: 16px;
  }
  
  .custom-toolbar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 16px 20px;
  }
  
  .search-filters {
    display: flex;
    gap: 12px;
    align-items: center;
  }
  
  .search-input {
    width: 250px;
  }
  
  .type-filter,
  .status-filter {
    width: 120px;
  }
  
  .action-button {
    min-width: 80px;
  }
  
  .reset-button {
    border-color: #d9d9d9;
  }
  
  .action-buttons {
    display: flex;
    gap: 12px;
  }
  
  .add-button {
    min-width: 100px;
  }
  
  .table-container {
    padding: 0;
  }
  
  .custom-table {
    margin: 20px;
  }
  
  .type-tag,
  .status-tag {
    display: flex;
    align-items: center;
    gap: 4px;
  }
  
  .size-container {
    font-family: 'Monaco', 'Consolas', monospace;
    font-size: 13px;
  }
  
  .path-container {
    font-family: 'Monaco', 'Consolas', monospace;
    font-size: 12px;
    color: #666;
    max-width: 180px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  
  .tags-container {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
    max-width: 140px;
  }
  
  .data-tag {
    margin: 2px;
    font-size: 11px;
  }
  
  .action-column {
    display: flex;
    gap: 8px;
    flex-wrap: wrap;
  }
  
  .action-column .ant-btn {
    font-size: 12px;
    height: 28px;
    padding: 0 8px;
  }
  
  .pagination-container {
    padding: 16px 20px;
    border-top: 1px solid #f0f0f0;
    display: flex;
    justify-content: flex-end;
  }
  
  .custom-pagination {
    margin: 0;
  }
  
  .custom-modal .ant-modal-body {
    padding: 20px;
  }
  
  .custom-form {
    margin-top: 16px;
  }
  
  .form-section {
    margin-bottom: 24px;
    padding: 16px;
    border: 1px solid #f0f0f0;
    border-radius: 6px;
    background-color: #fafafa;
  }
  
  .section-title {
    font-size: 16px;
    font-weight: 500;
    color: #262626;
    margin-bottom: 16px;
    padding-bottom: 8px;
    border-bottom: 1px solid #e8e8e8;
  }
  
  .data-detail-container {
    max-height: 600px;
    overflow-y: auto;
  }
  
  .detail-section {
    margin-bottom: 24px;
    padding: 16px;
    border: 1px solid #f0f0f0;
    border-radius: 6px;
    background-color: #fafafa;
  }
  
  .path-code {
    background-color: #f6f8fa;
    padding: 4px 8px;
    border-radius: 4px;
    font-family: 'Monaco', 'Consolas', monospace;
    font-size: 12px;
    color: #e74c3c;
  }
  
  .description-content {
    color: #595959;
    line-height: 1.6;
    padding: 12px;
    background-color: white;
    border-radius: 4px;
    border: 1px solid #e8e8e8;
  }
  
  .metadata-pre {
    background-color: #f6f8fa;
    border: 1px solid #e1e4e8;
    border-radius: 6px;
    padding: 16px;
    font-size: 12px;
    line-height: 1.45;
    overflow-x: auto;
    color: #24292e;
    font-family: 'Monaco', 'Consolas', monospace;
  }
  
  .search-icon {
    color: #bfbfbf;
  }
  
  /* 响应式设计 */
  @media (max-width: 768px) {
    .custom-toolbar {
      flex-direction: column;
      gap: 16px;
    }
    
    .search-filters {
      flex-wrap: wrap;
      width: 100%;
    }
    
    .search-input {
      width: 100%;
    }
    
    .action-buttons {
      width: 100%;
      justify-content: center;
    }
  }
  </style>