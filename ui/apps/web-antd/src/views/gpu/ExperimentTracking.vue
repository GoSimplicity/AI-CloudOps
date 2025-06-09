<template>
    <div class="experiment-tracking-page">
      <!-- 页面标题区域 -->
      <div class="page-header">
        <h2 class="page-title">实验跟踪</h2>
        <p class="page-description">管理和监控机器学习实验</p>
      </div>
  
      <!-- 查询和操作工具栏 -->
      <div class="dashboard-card custom-toolbar">
        <div class="search-filters">
          <a-input 
            v-model:value="searchText" 
            placeholder="请输入实验名称" 
            class="search-input"
          >
            <template #prefix>
              <SearchOutlined class="search-icon" />
            </template>
          </a-input>
          <a-select 
            v-model:value="statusFilter" 
            placeholder="实验状态" 
            class="status-filter"
            allowClear
          >
            <a-select-option value="">全部状态</a-select-option>
            <a-select-option value="Running">运行中</a-select-option>
            <a-select-option value="Completed">已完成</a-select-option>
            <a-select-option value="Failed">失败</a-select-option>
            <a-select-option value="Stopped">已停止</a-select-option>
          </a-select>
          <a-select 
            v-model:value="projectFilter" 
            placeholder="项目名称" 
            class="project-filter"
            allowClear
          >
            <a-select-option value="">全部项目</a-select-option>
            <a-select-option value="image-classification">图像分类</a-select-option>
            <a-select-option value="nlp-sentiment">情感分析</a-select-option>
            <a-select-option value="object-detection">目标检测</a-select-option>
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
            创建实验
          </a-button>
        </div>
      </div>
  
      <!-- 实验列表表格 -->
      <div class="dashboard-card table-container">
        <a-table 
          :columns="columns" 
          :data-source="filteredData" 
          row-key="id" 
          :pagination="false"
          class="custom-table"
          :scroll="{ x: 1600 }"
        >
          <!-- 实验状态列 -->
          <template #status="{ record }">
            <a-tag :color="getStatusColor(record.status)" class="status-tag">
              {{ getStatusText(record.status) }}
            </a-tag>
          </template>
          
          <!-- 指标列 -->
          <template #metrics="{ record }">
            <div class="metrics-container">
              <div class="metric-item" v-for="(value, key) in record.metrics" :key="key">
                <span class="metric-label">{{ key }}:</span>
                <span class="metric-value">{{ formatMetric(value) }}</span>
              </div>
            </div>
          </template>
          
          <!-- 超参数列 -->
          <template #hyperparams="{ record }">
            <a-tooltip :title="JSON.stringify(record.hyperparams, null, 2)">
              <div class="hyperparams-container">
                <div class="hyperparam-item" v-for="(value, key) in record.hyperparams" :key="key">
                  <span class="hyperparam-label">{{ key }}:</span>
                  <span class="hyperparam-value">{{ formatHyperparam(value) }}</span>
                </div>
              </div>
            </a-tooltip>
          </template>
          
          <!-- 运行时间列 -->
          <template #duration="{ record }">
            <div class="duration-container">
              {{ formatDuration(record.start_time, record.end_time) }}
            </div>
          </template>
          
          <!-- 进度列 -->
          <template #progress="{ record }">
            <div class="progress-container">
              <a-progress 
                :percent="record.progress" 
                :status="record.status === 'Failed' ? 'exception' : 'normal'"
                size="small"
              />
              <span class="progress-text">{{ record.current_epoch }}/{{ record.total_epochs }} epochs</span>
            </div>
          </template>
          
          <!-- 操作列 -->
          <template #action="{ record }">
            <div class="action-column">
              <a-button type="primary" size="small" @click="handleView(record)">
                查看
              </a-button>
              <a-button type="default" size="small" @click="handleCompare(record)">
                对比
              </a-button>
              <a-button type="default" size="small" @click="handleStop(record)" v-if="record.status === 'Running'">
                停止
              </a-button>
              <a-button type="default" size="small" @click="handleDelete(record)" v-if="['Completed', 'Failed', 'Stopped'].includes(record.status)">
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
  
      <!-- 创建实验模态框 -->
      <a-modal 
        title="创建实验" 
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
                <a-form-item label="实验名称" name="name" :rules="[{ required: true, message: '请输入实验名称' }]">
                  <a-input v-model:value="addForm.name" placeholder="请输入实验名称" />
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item label="项目名称" name="project" :rules="[{ required: true, message: '请选择项目' }]">
                  <a-select v-model:value="addForm.project" placeholder="请选择项目">
                    <a-select-option value="image-classification">图像分类</a-select-option>
                    <a-select-option value="nlp-sentiment">情感分析</a-select-option>
                    <a-select-option value="object-detection">目标检测</a-select-option>
                  </a-select>
                </a-form-item>
              </a-col>
            </a-row>
            <a-row :gutter="16">
              <a-col :span="24">
                <a-form-item label="实验描述" name="description">
                  <a-textarea v-model:value="addForm.description" placeholder="请输入实验描述" :rows="2" />
                </a-form-item>
              </a-col>
            </a-row>
          </div>
  
          <div class="form-section">
            <div class="section-title">模型配置</div>
            <a-row :gutter="16">
              <a-col :span="12">
                <a-form-item label="模型类型" name="model_type" :rules="[{ required: true, message: '请选择模型类型' }]">
                  <a-select v-model:value="addForm.model_type" placeholder="请选择模型类型">
                    <a-select-option value="resnet">ResNet</a-select-option>
                    <a-select-option value="vgg">VGG</a-select-option>
                    <a-select-option value="bert">BERT</a-select-option>
                    <a-select-option value="yolo">YOLO</a-select-option>
                  </a-select>
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item label="预训练模型" name="pretrained">
                  <a-switch v-model:checked="addForm.pretrained" />
                </a-form-item>
              </a-col>
            </a-row>
          </div>
  
          <div class="form-section">
            <div class="section-title">训练配置</div>
            <a-row :gutter="16">
              <a-col :span="8">
                <a-form-item label="学习率" name="learning_rate">
                  <a-input-number v-model:value="addForm.learning_rate" :min="0" :max="1" :step="0.001" placeholder="学习率" class="full-width" />
                </a-form-item>
              </a-col>
              <a-col :span="8">
                <a-form-item label="批量大小" name="batch_size">
                  <a-input-number v-model:value="addForm.batch_size" :min="1" :max="512" placeholder="批量大小" class="full-width" />
                </a-form-item>
              </a-col>
              <a-col :span="8">
                <a-form-item label="训练轮数" name="epochs">
                  <a-input-number v-model:value="addForm.epochs" :min="1" :max="1000" placeholder="训练轮数" class="full-width" />
                </a-form-item>
              </a-col>
            </a-row>
            <a-row :gutter="16">
              <a-col :span="12">
                <a-form-item label="优化器" name="optimizer">
                  <a-select v-model:value="addForm.optimizer" placeholder="请选择优化器">
                    <a-select-option value="adam">Adam</a-select-option>
                    <a-select-option value="sgd">SGD</a-select-option>
                    <a-select-option value="rmsprop">RMSprop</a-select-option>
                  </a-select>
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item label="损失函数" name="loss_function">
                  <a-select v-model:value="addForm.loss_function" placeholder="请选择损失函数">
                    <a-select-option value="crossentropy">CrossEntropy</a-select-option>
                    <a-select-option value="mse">MSE</a-select-option>
                    <a-select-option value="bce">BCE</a-select-option>
                  </a-select>
                </a-form-item>
              </a-col>
            </a-row>
          </div>
  
          <div class="form-section">
            <div class="section-title">数据集配置</div>
            <a-row :gutter="16">
              <a-col :span="12">
                <a-form-item label="数据集名称" name="dataset_name" :rules="[{ required: true, message: '请输入数据集名称' }]">
                  <a-input v-model:value="addForm.dataset_name" placeholder="请输入数据集名称" />
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item label="数据集路径" name="dataset_path" :rules="[{ required: true, message: '请输入数据集路径' }]">
                  <a-input v-model:value="addForm.dataset_path" placeholder="请输入数据集路径" />
                </a-form-item>
              </a-col>
            </a-row>
          </div>
  
          <div class="form-section">
            <div class="section-title">资源配置</div>
            <a-row :gutter="16">
              <a-col :span="8">
                <a-form-item label="GPU数量" name="gpu_count">
                  <a-input-number v-model:value="addForm.gpu_count" :min="0" :max="8" placeholder="GPU数量" class="full-width" />
                </a-form-item>
              </a-col>
              <a-col :span="8">
                <a-form-item label="CPU核心数" name="cpu_count">
                  <a-input-number v-model:value="addForm.cpu_count" :min="1" :max="32" placeholder="CPU核心数" class="full-width" />
                </a-form-item>
              </a-col>
              <a-col :span="8">
                <a-form-item label="内存(GB)" name="memory_gb">
                  <a-input-number v-model:value="addForm.memory_gb" :min="1" :max="128" placeholder="内存大小" class="full-width" />
                </a-form-item>
              </a-col>
            </a-row>
          </div>
        </a-form>
      </a-modal>
  
      <!-- 实验详情模态框 -->
      <a-modal 
        title="实验详情" 
        v-model:visible="isViewModalVisible" 
        @cancel="closeViewModal"
        :width="1000"
        class="custom-modal"
        :footer="null"
      >
        <div class="experiment-detail-container" v-if="viewExperiment">
          <a-tabs defaultActiveKey="1">
            <a-tab-pane key="1" tab="基本信息">
              <div class="detail-section">
                <div class="section-title">实验信息</div>
                <a-descriptions :column="2" size="small">
                  <a-descriptions-item label="实验名称">{{ viewExperiment.name }}</a-descriptions-item>
                  <a-descriptions-item label="项目名称">{{ viewExperiment.project }}</a-descriptions-item>
                  <a-descriptions-item label="状态">
                    <a-tag :color="getStatusColor(viewExperiment.status)">{{ getStatusText(viewExperiment.status) }}</a-tag>
                  </a-descriptions-item>
                  <a-descriptions-item label="创建者">{{ viewExperiment.creator }}</a-descriptions-item>
                  <a-descriptions-item label="开始时间">{{ viewExperiment.start_time }}</a-descriptions-item>
                  <a-descriptions-item label="结束时间">{{ viewExperiment.end_time || '进行中' }}</a-descriptions-item>
                  <a-descriptions-item label="运行时间">{{ formatDuration(viewExperiment.start_time, viewExperiment.end_time) }}</a-descriptions-item>
                  <a-descriptions-item label="进度">{{ viewExperiment.progress }}%</a-descriptions-item>
                </a-descriptions>
              </div>
              
              <div class="detail-section">
                <div class="section-title">模型配置</div>
                <a-descriptions :column="2" size="small">
                  <a-descriptions-item label="模型类型">{{ viewExperiment.model_type }}</a-descriptions-item>
                  <a-descriptions-item label="预训练模型">{{ viewExperiment.pretrained ? '是' : '否' }}</a-descriptions-item>
                  <a-descriptions-item label="数据集">{{ viewExperiment.dataset_name }}</a-descriptions-item>
                  <a-descriptions-item label="数据集路径">{{ viewExperiment.dataset_path }}</a-descriptions-item>
                </a-descriptions>
              </div>
            </a-tab-pane>
            
            <a-tab-pane key="2" tab="超参数">
              <div class="hyperparams-detail">
                <a-descriptions :column="1" size="small">
                  <a-descriptions-item v-for="(value, key) in viewExperiment.hyperparams" :key="key" :label="key">
                    {{ formatHyperparam(value) }}
                  </a-descriptions-item>
                </a-descriptions>
              </div>
            </a-tab-pane>
            
            <a-tab-pane key="3" tab="指标">
              <div class="metrics-detail">
                <a-row :gutter="16">
                  <a-col :span="12" v-for="(value, key) in viewExperiment.metrics" :key="key">
                    <a-card size="small" :title="key" class="metric-card">
                      <div class="metric-value-large">{{ formatMetric(value) }}</div>
                    </a-card>
                  </a-col>
                </a-row>
              </div>
            </a-tab-pane>
            
            <a-tab-pane key="4" tab="日志">
              <div class="logs-container">
                <a-textarea 
                  :value="viewExperiment.logs" 
                  :rows="15" 
                  readonly
                  class="logs-textarea"
                />
              </div>
            </a-tab-pane>
          </a-tabs>
        </div>
      </a-modal>
  
      <!-- 实验对比模态框 -->
      <a-modal 
        title="实验对比" 
        v-model:visible="isCompareModalVisible" 
        @cancel="closeCompareModal"
        :width="1200"
        class="custom-modal"
        :footer="null"
      >
        <div class="compare-container">
          <div class="compare-select">
            <a-select 
              v-model:value="compareExperiments" 
              mode="multiple" 
              placeholder="选择要对比的实验" 
              class="compare-select-input"
              :max-tag-count="3"
            >
              <a-select-option v-for="exp in data" :key="exp.id" :value="exp.id">
                {{ exp.name }}
              </a-select-option>
            </a-select>
          </div>
          
          <div class="compare-table" v-if="compareExperiments.length > 0">
            <a-table 
              :columns="compareColumns" 
              :data-source="getCompareData()" 
              :pagination="false"
              size="small"
            />
          </div>
        </div>
      </a-modal>
    </div>
  </template>
  
  <script lang="ts" setup>
  import { ref, reactive, onMounted, computed } from 'vue';
  import { message, Modal } from 'ant-design-vue';
  import {
    SearchOutlined,
    ReloadOutlined,
    PlusOutlined
  } from '@ant-design/icons-vue';
  import type { FormInstance } from 'ant-design-vue';
  
  interface ExperimentItem {
    id: number;
    name: string;
    project: string;
    status: string;
    model_type: string;
    pretrained: boolean;
    dataset_name: string;
    dataset_path: string;
    hyperparams: Record<string, any>;
    metrics: Record<string, number>;
    progress: number;
    current_epoch: number;
    total_epochs: number;
    start_time: string;
    end_time?: string;
    creator: string;
    description?: string;
    logs: string;
  }
  
  // 搜索和筛选
  const searchText = ref('');
  const statusFilter = ref('');
  const projectFilter = ref('');
  
  // 表格数据
  const data = ref<ExperimentItem[]>([]);
  
  // 过滤后的数据
  const filteredData = computed(() => {
    let result = data.value;
    
    if (searchText.value) {
      result = result.filter(item => 
        item.name.toLowerCase().includes(searchText.value.toLowerCase())
      );
    }
    
    if (statusFilter.value) {
      result = result.filter(item => item.status === statusFilter.value);
    }
    
    if (projectFilter.value) {
      result = result.filter(item => item.project === projectFilter.value);
    }
    
    return result;
  });
  
  // 分页相关
  const pageSizeOptions = ref<string[]>(['10', '20', '30', '40', '50']);
  const current = ref(1);
  const pageSizeRef = ref(10);
  const total = computed(() => filteredData.value.length);
  
  // 模态框状态
  const isAddModalVisible = ref(false);
  const isViewModalVisible = ref(false);
  const isCompareModalVisible = ref(false);
  
  // 表单引用
  const addFormRef = ref<FormInstance>();
  
  // 查看详情的实验
  const viewExperiment = ref<ExperimentItem | null>(null);
  
  // 对比实验
  const compareExperiments = ref<number[]>([]);
  
  // 新增表单
  const addForm = reactive({
    name: '',
    project: '',
    description: '',
    model_type: '',
    pretrained: false,
    learning_rate: 0.001,
    batch_size: 32,
    epochs: 100,
    optimizer: 'adam',
    loss_function: 'crossentropy',
    dataset_name: '',
    dataset_path: '',
    gpu_count: 1,
    cpu_count: 4,
    memory_gb: 8
  });
  
  // 表格列配置
  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      key: 'id',
      width: 60,
    },
    {
      title: '实验名称',
      dataIndex: 'name',
      key: 'name',
      width: 200,
    },
    {
      title: '项目',
      dataIndex: 'project',
      key: 'project',
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
      title: '模型类型',
      dataIndex: 'model_type',
      key: 'model_type',
      width: 100,
    },
    {
      title: '进度',
      key: 'progress',
      slots: { customRender: 'progress' },
      width: 200,
    },
    {
      title: '指标',
      key: 'metrics',
      slots: { customRender: 'metrics' },
      width: 200,
    },
    {
      title: '超参数',
      key: 'hyperparams',
      slots: { customRender: 'hyperparams' },
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
      title: '开始时间',
      dataIndex: 'start_time',
      key: 'start_time',
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
  
  // 对比表格列配置
  const compareColumns = computed(() => [
    {
      title: '属性',
      dataIndex: 'property',
      key: 'property',
      width: 150,
    },
    ...compareExperiments.value.map((expId, index) => {
      const exp = data.value.find(e => e.id === expId);
      return {
        title: exp ? exp.name : `实验 ${index + 1}`,
        dataIndex: `experiment_${index}`,
        key: `experiment_${index}`,
      };
    })
  ]);
  
  // 初始化数据
  onMounted(() => {
    loadData();
  });
  
  // 获取状态颜色
  const getStatusColor = (status: string) => {
    const colorMap: Record<string, string> = {
      'Running': 'blue',
      'Completed': 'green',
      'Failed': 'red',
      'Stopped': 'orange'
    };
    return colorMap[status] || 'default';
  };
  
  // 获取状态文本
  const getStatusText = (status: string) => {
    const textMap: Record<string, string> = {
      'Running': '运行中',
      'Completed': '已完成',
      'Failed': '失败',
      'Stopped': '已停止'
    };
    return textMap[status] || status;
  };
  
  // 格式化指标
  const formatMetric = (value: number) => {
    return typeof value === 'number' ? value.toFixed(4) : value;
  };
  
  // 格式化超参数
  const formatHyperparam = (value: any) => {
    if (typeof value === 'number') {
      return value.toString();
    }
    return value;
  };
  
  // 格式化运行时间
  const formatDuration = (startTime: string, endTime?: string) => {
    if (!startTime) return '未开始';
    
    const start = new Date(startTime);
    const end = endTime ? new Date(endTime) : new Date();
    const duration = Math.floor((end.getTime() - start.getTime()) / 1000);
    
    const hours = Math.floor(duration / 3600);
    const minutes = Math.floor((duration % 3600) / 60);
    const seconds = duration % 60;
    
    if (hours > 0) {
      return `${hours}h ${minutes}m`;
    } else if (minutes > 0) {
      return `${minutes}m ${seconds}s`;
    } else {
      return `${seconds}s`;
    }
  };
  
  // 加载数据
  const loadData = () => {
    // 模拟数据
    const mockData: ExperimentItem[] = [
      {
        id: 1,
        name: 'resnet50-imagenet-v1',
        project: 'image-classification',
        status: 'Running',
        model_type: 'resnet',
        pretrained: true,
        dataset_name: 'ImageNet',
        dataset_path: '/data/imagenet',
        hyperparams: {
          learning_rate: 0.001,
          batch_size: 64,
          epochs: 200,
          optimizer: 'adam',
          weight_decay: 0.0001
        },
        metrics: {
          accuracy: 0.7534,
          loss: 0.8765,
          val_accuracy: 0.7123,
          val_loss: 0.9234
        },
        progress: 45,
        current_epoch: 90,
        total_epochs: 200,
        start_time: '2024-06-09 10:30:00',
        creator: 'admin',
        description: 'ResNet50在ImageNet上的图像分类实验',
        logs: 'Epoch 90/200\n1563/1563 [==============================] - 45s 29ms/step - loss: 0.8765 - accuracy: 0.7534 - val_loss: 0.9234 - val_accuracy: 0.7123'
      },
      {
        id: 2,
        name: 'bert-sentiment-analysis',
        project: 'nlp-sentiment',
        status: 'Completed',
        model_type: 'bert',
        pretrained: true,
        dataset_name: 'IMDB Reviews',
        dataset_path: '/data/imdb',
        hyperparams: {
          learning_rate: 0.00002,
          batch_size: 16,
          epochs: 5,
          optimizer: 'adamw',
          max_length: 512
        },
        metrics: {
          accuracy: 0.9234,
          f1_score: 0.9156,
          precision: 0.9278,
          recall: 0.9045
        },
        progress: 100,
        current_epoch: 5,
        total_epochs: 5,
        start_time: '2024-06-09 08:00:00',
        end_time: '2024-06-09 10:15:00',
        creator: 'user1',
        description: 'BERT模型在IMDB数据集上的情感分析',
        logs: 'Training completed successfully!\nFinal accuracy: 92.34%\nF1 Score: 91.56%'
      },
      {
        id: 3,
        name: 'yolo-object-detection-v1',
        project: 'object-detection',
        status: 'Failed',
        model_type: 'yolo',
        pretrained: false,
        dataset_name: 'COCO',
        dataset_path: '/data/coco',
        hyperparams: {
          learning_rate: 0.01,
          batch_size: 8,
          epochs: 300,
          optimizer: 'sgd',
          momentum: 0.9
        },
        metrics: {
          mAP: 0.0,
          mAP_50: 0.0,
          precision: 0.0,
          recall: 0.0
        },
        progress: 15,
        current_epoch: 45,
        total_epochs: 300,
        start_time: '2024-06-09 14:00:00',
        end_time: '2024-06-09 15:30:00',
        creator: 'user2',
        description: 'YOLO模型在COCO数据集上的目标检测实验',
        logs: 'Error: CUDA out of memory. Tried to allocate 2.00 GiB (GPU 0; 8.00 GiB total capacity; 6.89 GiB already allocated'
      },
      {
        id: 4,
        name: 'vgg16-transfer-learning',
        project: 'image-classification',
        status: 'Stopped',
        model_type: 'vgg',
        pretrained: true,
        dataset_name: 'CIFAR-10',
        dataset_path: '/data/cifar10',
        hyperparams: {
          learning_rate: 0.0001,
          batch_size: 32,
          epochs: 50,
          optimizer: 'adam',
          dropout: 0.5
        },
        metrics: {
          accuracy: 0.8567,
          loss: 0.4321,
          val_accuracy: 0.8234,
          val_loss: 0.5123
        },
        progress: 80,
        current_epoch: 40,
        total_epochs: 50,
        start_time: '2024-06-09 12:00:00',
        end_time: '2024-06-09 13:45:00',
        creator: 'admin',
        description: 'VGG16在CIFAR-10上的迁移学习实验',
        logs: 'Training stopped by user\nLast epoch: 40/50\nBest accuracy: 85.67%'
      }
    ];
    
    data.value = mockData;
  };
  
  // 搜索处理
  const handleSearch = () => {
    current.value = 1;
    message.success('搜索完成');
  };
  
  // 重置处理
  const handleReset = () => {
    searchText.value = '';
    statusFilter.value = '';
    projectFilter.value = '';
    current.value = 1;
    message.success('已重置搜索条件');
  };
  
  // 分页变更处理
  const handlePageChange = (page: number, pageSize: number) => {
    current.value = page;
    pageSizeRef.value = pageSize;
  };
  
  // 页面大小变更处理
  const handleSizeChange = (currentPage: number, size: number) => {
    current.value = currentPage;
    pageSizeRef.value = size;
  };
  
  // 显示新增模态框
  const showAddModal = () => {
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
      project: '',
      description: '',
      model_type: '',
      pretrained: false,
      learning_rate: 0.001,
      batch_size: 32,
      epochs: 100,
      optimizer: 'adam',
      loss_function: 'crossentropy',
      dataset_name: '',
      dataset_path: '',
      gpu_count: 1,
      cpu_count: 4,
      memory_gb: 8
    });
    addFormRef.value?.resetFields();
  };
  
  // 新增实验处理
  const handleAdd = async () => {
    try {
      await addFormRef.value?.validate();
      
      const newExperiment: ExperimentItem = {
        id: data.value.length + 1,
        name: addForm.name,
        project: addForm.project,
        status: 'Running',
        model_type: addForm.model_type,
        pretrained: addForm.pretrained,
        dataset_name: addForm.dataset_name,
        dataset_path: addForm.dataset_path,
        hyperparams: {
          learning_rate: addForm.learning_rate,
          batch_size: addForm.batch_size,
          epochs: addForm.epochs,
          optimizer: addForm.optimizer,
          loss_function: addForm.loss_function
        },
        metrics: {
          accuracy: 0,
          loss: 0
        },
        progress: 0,
        current_epoch: 0,
        total_epochs: addForm.epochs,
        start_time: new Date().toLocaleString(),
        creator: 'current_user',
        description: addForm.description,
        logs: 'Experiment created and started...'
      };
      
      data.value.unshift(newExperiment);
      message.success('实验创建成功');
      closeAddModal();
    } catch (error) {
      console.error('表单验证失败:', error);
    }
  };
  
  // 查看实验详情
  const handleView = (record: ExperimentItem) => {
    viewExperiment.value = record;
    isViewModalVisible.value = true;
  };
  
  // 关闭查看模态框
  const closeViewModal = () => {
    isViewModalVisible.value = false;
    viewExperiment.value = null;
  };
  
  // 对比实验
  const handleCompare = (record: ExperimentItem) => {
    if (!compareExperiments.value.includes(record.id)) {
      compareExperiments.value.push(record.id);
    }
    isCompareModalVisible.value = true;
  };
  
  // 关闭对比模态框
  const closeCompareModal = () => {
    isCompareModalVisible.value = false;
  };
  
  // 获取对比数据
  const getCompareData = () => {
    const properties = [
      { key: 'name', label: '实验名称' },
      { key: 'status', label: '状态' },
      { key: 'model_type', label: '模型类型' },
      { key: 'progress', label: '进度' },
      { key: 'accuracy', label: '准确率' },
      { key: 'loss', label: '损失' },
      { key: 'learning_rate', label: '学习率' },
      { key: 'batch_size', label: '批量大小' },
      { key: 'optimizer', label: '优化器' }
    ];
    
    return properties.map(prop => {
      const row: any = { property: prop.label };
      
      compareExperiments.value.forEach((expId, index) => {
        const exp = data.value.find(e => e.id === expId);
        if (exp) {
          let value = '';
          switch (prop.key) {
            case 'accuracy':
              value = exp.metrics.accuracy ? formatMetric(exp.metrics.accuracy) : '-';
              break;
            case 'loss':
              value = exp.metrics.loss ? formatMetric(exp.metrics.loss) : '-';
              break;
            case 'learning_rate':
              value = exp.hyperparams.learning_rate?.toString() || '-';
              break;
            case 'batch_size':
              value = exp.hyperparams.batch_size?.toString() || '-';
              break;
            case 'optimizer':
              value = exp.hyperparams.optimizer || '-';
              break;
            case 'progress':
              value = `${exp.progress}%`;
              break;
            default:
              value = exp[prop.key as keyof ExperimentItem]?.toString() || '-';
          }
          row[`experiment_${index}`] = value;
        }
      });
      
      return row;
    });
  };
  
  // 停止实验
  const handleStop = (record: ExperimentItem) => {
    Modal.confirm({
      title: '确认停止实验',
      content: `确定要停止实验 "${record.name}" 吗？`,
      onOk: () => {
        record.status = 'Stopped';
        record.end_time = new Date().toLocaleString();
        message.success('实验已停止');
      }
    });
  };
  
  // 删除实验
  const handleDelete = (record: ExperimentItem) => {
    Modal.confirm({
      title: '确认删除实验',
      content: `确定要删除实验 "${record.name}" 吗？此操作不可恢复。`,
      onOk: () => {
        const index = data.value.findIndex(item => item.id === record.id);
        if (index > -1) {
          data.value.splice(index, 1);
          message.success('实验已删除');
        }
      }
    });
  };
  </script>
  
  <style scoped>
  .experiment-tracking-page {
    padding: 24px;
  }
  
  .page-header {
    margin-bottom: 24px;
  }
  
  .page-title {
    font-size: 24px;
    font-weight: 600;
    margin-bottom: 8px;
    color: #262626;
  }
  
  .page-description {
    color: #8c8c8c;
    margin: 0;
  }
  
  .dashboard-card {
    background: #fff;
    border-radius: 8px;
    padding: 24px;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
    margin-bottom: 24px;
  }
  
  .custom-toolbar {
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
  }
  
  .search-input {
    width: 200px;
  }
  
  .status-filter,
  .project-filter {
    width: 120px;
  }
  
  .action-button {
    border-radius: 6px;
  }
  
  .reset-button {
    border-color: #d9d9d9;
  }
  
  .action-buttons {
    display: flex;
    gap: 12px;
  }
  
  .add-button {
    border-radius: 6px;
  }
  
  .table-container {
    padding: 0;
  }
  
  .custom-table {
    border-radius: 8px;
    overflow: hidden;
  }
  
  .status-tag {
    font-size: 12px;
    padding: 2px 8px;
    border-radius: 4px;
  }
  
  .metrics-container,
  .hyperparams-container {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  
  .metric-item,
  .hyperparam-item {
    display: flex;
    align-items: center;
    font-size: 12px;
  }
  
  .metric-label,
  .hyperparam-label {
    font-weight: 500;
    margin-right: 4px;
    min-width: 60px;
  }
  
  .metric-value,
  .hyperparam-value {
    color: #1890ff;
    font-family: 'Monaco', 'Consolas', monospace;
  }
  
  .progress-container {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  
  .progress-text {
    font-size: 12px;
    color: #8c8c8c;
  }
  
  .action-column {
    display: flex;
    gap: 8px;
    flex-wrap: wrap;
  }
  
  .pagination-container {
    margin-top: 24px;
    display: flex;
    justify-content: center;
  }
  
  .custom-pagination {
    border-top: 1px solid #f0f0f0;
    padding-top: 16px;
  }
  
  .custom-modal :deep(.ant-modal-header) {
    border-radius: 8px 8px 0 0;
  }
  
  .custom-form {
    max-height: 400px;
    overflow-y: auto;
  }
  
  .form-section {
    margin-bottom: 24px;
    padding-bottom: 16px;
    border-bottom: 1px solid #f0f0f0;
  }
  
  .form-section:last-child {
    border-bottom: none;
    margin-bottom: 0;
  }
  
  .section-title {
    font-size: 16px;
    font-weight: 600;
    margin-bottom: 16px;
    color: #262626;
  }
  
  .full-width {
    width: 100%;
  }
  
  .experiment-detail-container {
    max-height: 600px;
    overflow-y: auto;
  }
  
  .detail-section {
    margin-bottom: 24px;
  }
  
  .hyperparams-detail,
  .metrics-detail {
    padding: 16px;
    background: #fafafa;
    border-radius: 6px;
  }
  
  .metric-card {
    margin-bottom: 8px;
  }
  
  .metric-value-large {
    font-size: 24px;
    font-weight: 600;
    color: #1890ff;
    text-align: center;
  }
  
  .logs-container {
    padding: 16px;
    background: #f5f5f5;
    border-radius: 6px;
  }
  
  .logs-textarea {
    font-family: 'Monaco', 'Consolas', monospace;
    font-size: 12px;
    background: #000;
    color: #fff;
    border: none;
  }
  
  .compare-container {
    min-height: 400px;
  }
  
  .compare-select {
    margin-bottom: 24px;
  }
  
  .compare-select-input {
    width: 100%;
  }
  
  .compare-table {
    border: 1px solid #f0f0f0;
    border-radius: 6px;
    overflow: hidden;
  }
  
  @media (max-width: 768px) {
    .custom-toolbar {
      flex-direction: column;
      align-items: stretch;
    }
    
    .search-filters {
      justify-content: center;
    }
    
    .search-input,
    .status-filter,
    .project-filter {
      width: 100%;
      max-width: 200px;
    }
  }
  </style>