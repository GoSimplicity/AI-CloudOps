<template>
    <div class="job-monitor-page">
      <!-- 页面标题区域 -->
      <div class="page-header">
        <h2 class="page-title">作业监控</h2>
      </div>
  
      <!-- 监控概览卡片 -->
      <div class="overview-cards">
        <div class="dashboard-card stats-card">
          <div class="stats-content">
            <div class="stats-icon running">
              <PlayCircleOutlined />
            </div>
            <div class="stats-info">
              <div class="stats-value">{{ runningJobs }}</div>
              <div class="stats-label">运行中作业</div>
            </div>
          </div>
        </div>
        
        <div class="dashboard-card stats-card">
          <div class="stats-content">
            <div class="stats-icon pending">
              <ClockCircleOutlined />
            </div>
            <div class="stats-info">
              <div class="stats-value">{{ pendingJobs }}</div>
              <div class="stats-label">等待中作业</div>
            </div>
          </div>
        </div>
        
        <div class="dashboard-card stats-card">
          <div class="stats-content">
            <div class="stats-icon completed">
              <CheckCircleOutlined />
            </div>
            <div class="stats-info">
              <div class="stats-value">{{ completedJobs }}</div>
              <div class="stats-label">已完成作业</div>
            </div>
          </div>
        </div>
        
        <div class="dashboard-card stats-card">
          <div class="stats-content">
            <div class="stats-icon failed">
              <ExclamationCircleOutlined />
            </div>
            <div class="stats-info">
              <div class="stats-value">{{ failedJobs }}</div>
              <div class="stats-label">失败作业</div>
            </div>
          </div>
        </div>
      </div>
  
      <!-- 筛选和刷新工具栏 -->
      <div class="dashboard-card custom-toolbar">
        <div class="search-filters">
          <a-select 
            v-model:value="selectedQueue" 
            placeholder="选择队列" 
            class="queue-filter"
            allowClear
          >
            <a-select-option value="">全部队列</a-select-option>
            <a-select-option value="default">default</a-select-option>
            <a-select-option value="high-priority">high-priority</a-select-option>
            <a-select-option value="low-priority">low-priority</a-select-option>
          </a-select>
          
          <a-select 
            v-model:value="selectedStatus" 
            placeholder="作业状态" 
            class="status-filter"
            allowClear
          >
            <a-select-option value="">全部状态</a-select-option>
            <a-select-option value="Running">运行中</a-select-option>
            <a-select-option value="Pending">等待中</a-select-option>
            <a-select-option value="Completed">已完成</a-select-option>
            <a-select-option value="Failed">失败</a-select-option>
          </a-select>
  
          <a-range-picker 
            v-model:value="timeRange"
            show-time
            format="YYYY-MM-DD HH:mm:ss"
            placeholder="['开始时间', '结束时间']"
            class="time-filter"
          />
        </div>
        
        <div class="action-buttons">
          <a-button type="primary" class="action-button" @click="refreshData">
            <template #icon>
              <ReloadOutlined />
            </template>
            刷新数据
          </a-button>
          <a-button class="action-button export-button" @click="exportData">
            <template #icon>
              <DownloadOutlined />
            </template>
            导出数据
          </a-button>
        </div>
      </div>
  
      <!-- 实时监控表格 -->
      <div class="dashboard-card table-container">
        <div class="table-header">
          <h3 class="table-title">实时作业状态</h3>
          <div class="auto-refresh">
            <a-switch v-model:checked="autoRefresh" size="small" />
            <span class="refresh-label">自动刷新</span>
          </div>
        </div>
        
        <a-table 
          :columns="columns" 
          :data-source="monitorData" 
          row-key="id" 
          :pagination="false"
          class="custom-table monitor-table"
          :scroll="{ x: 1500 }"
          :loading="loading"
        >
          <!-- 作业状态列 -->
          <template #status="{ record }">
            <div class="status-cell">
              <a-tag :color="getStatusColor(record.status)" class="status-tag">
                <template #icon>
                  <LoadingOutlined v-if="record.status === 'Running'" spin />
                  <ClockCircleOutlined v-else-if="record.status === 'Pending'" />
                  <CheckCircleOutlined v-else-if="record.status === 'Completed'" />
                  <ExclamationCircleOutlined v-else-if="record.status === 'Failed'" />
                </template>
                {{ getStatusText(record.status) }}
              </a-tag>
            </div>
          </template>
          
          <!-- 进度列 -->
          <template #progress="{ record }">
            <div class="progress-cell">
              <a-progress 
                :percent="record.progress" 
                size="small" 
                :status="getProgressStatus(record.status)"
                :show-info="true"
              />
              <div class="progress-text">{{ record.current_task }}/{{ record.total_tasks }}</div>
            </div>
          </template>
          
          <!-- 资源使用率列 -->
          <template #resources="{ record }">
            <div class="resource-usage">
              <div class="resource-item">
                <span class="resource-label">CPU:</span>
                <a-progress 
                  :percent="record.cpu_usage" 
                  size="small" 
                  :show-info="false"
                  :stroke-color="getResourceColor(record.cpu_usage)"
                />
                <span class="resource-value">{{ record.cpu_usage }}%</span>
              </div>
              <div class="resource-item">
                <span class="resource-label">内存:</span>
                <a-progress 
                  :percent="record.memory_usage" 
                  size="small" 
                  :show-info="false"
                  :stroke-color="getResourceColor(record.memory_usage)"
                />
                <span class="resource-value">{{ record.memory_usage }}%</span>
              </div>
              <div class="resource-item" v-if="record.gpu_usage !== null">
                <span class="resource-label">GPU:</span>
                <a-progress 
                  :percent="record.gpu_usage" 
                  size="small" 
                  :show-info="false"
                  :stroke-color="getResourceColor(record.gpu_usage)"
                />
                <span class="resource-value">{{ record.gpu_usage }}%</span>
              </div>
            </div>
          </template>
          
          <!-- 运行时间列 -->
          <template #duration="{ record }">
            <div class="duration-cell">
              <div class="duration-value">{{ formatDuration(record.start_time) }}</div>
              <div class="eta-value" v-if="record.estimated_completion">
                预计完成: {{ record.estimated_completion }}
              </div>
            </div>
          </template>
          
          <!-- 日志列 -->
          <template #logs="{ record }">
            <div class="log-cell">
              <a-button size="small" type="link" @click="viewLogs(record)">
                <template #icon>
                  <FileTextOutlined />
                </template>
                查看日志
              </a-button>
              <a-button size="small" type="link" @click="viewMetrics(record)">
                <template #icon>
                  <BarChartOutlined />
                </template>
                性能指标
              </a-button>
            </div>
          </template>
          
          <!-- 操作列 -->
          <template #action="{ record }">
            <div class="action-column">
              <a-button type="primary" size="small" @click="handleView(record)">
                详情
              </a-button>
              <a-button 
                type="default" 
                size="small" 
                @click="handlePause(record)" 
                v-if="record.status === 'Running'"
                :disabled="!record.can_pause"
              >
                暂停
              </a-button>
              <a-button 
                type="default" 
                size="small" 
                @click="handleResume(record)" 
                v-if="record.status === 'Paused'"
              >
                恢复
              </a-button>
              <a-button 
                type="default" 
                size="small" 
                danger
                @click="handleStop(record)" 
                v-if="['Running', 'Pending'].includes(record.status)"
              >
                停止
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
  
      <!-- 作业详情模态框 -->
      <a-modal 
        title="作业监控详情" 
        v-model:visible="isDetailModalVisible" 
        @cancel="closeDetailModal"
        :width="1000"
        class="custom-modal"
        :footer="null"
      >
        <div class="job-detail-monitor" v-if="detailJob">
          <a-tabs default-active-key="1">
            <a-tab-pane key="1" tab="基本信息">
              <div class="detail-section">
                <a-descriptions :column="2" size="small">
                  <a-descriptions-item label="作业名称">{{ detailJob.name }}</a-descriptions-item>
                  <a-descriptions-item label="状态">
                    <a-tag :color="getStatusColor(detailJob.status)">{{ getStatusText(detailJob.status) }}</a-tag>
                  </a-descriptions-item>
                  <a-descriptions-item label="队列">{{ detailJob.queue }}</a-descriptions-item>
                  <a-descriptions-item label="优先级">{{ detailJob.priority }}</a-descriptions-item>
                  <a-descriptions-item label="进度">{{ detailJob.current_task }}/{{ detailJob.total_tasks }}</a-descriptions-item>
                  <a-descriptions-item label="完成率">{{ detailJob.progress }}%</a-descriptions-item>
                  <a-descriptions-item label="开始时间">{{ detailJob.start_time }}</a-descriptions-item>
                  <a-descriptions-item label="预计完成">{{ detailJob.estimated_completion || '计算中...' }}</a-descriptions-item>
                </a-descriptions>
              </div>
            </a-tab-pane>
            
            <a-tab-pane key="2" tab="资源监控">
              <div class="resource-monitor">
                <div class="resource-chart">
                  <h4>CPU使用率</h4>
                  <a-progress :percent="detailJob.cpu_usage" :stroke-color="getResourceColor(detailJob.cpu_usage)" />
                </div>
                <div class="resource-chart">
                  <h4>内存使用率</h4>
                  <a-progress :percent="detailJob.memory_usage" :stroke-color="getResourceColor(detailJob.memory_usage)" />
                </div>
                <div class="resource-chart" v-if="detailJob.gpu_usage !== null">
                  <h4>GPU使用率</h4>
                  <a-progress :percent="detailJob.gpu_usage" :stroke-color="getResourceColor(detailJob.gpu_usage)" />
                </div>
              </div>
            </a-tab-pane>
            
            <a-tab-pane key="3" tab="执行日志">
              <div class="log-container">
                <div class="log-header">
                  <a-button size="small" @click="refreshLogs">
                    <template #icon>
                      <ReloadOutlined />
                    </template>
                    刷新日志
                  </a-button>
                  <a-button size="small" @click="downloadLogs">
                    <template #icon>
                      <DownloadOutlined />
                    </template>
                    下载日志
                  </a-button>
                </div>
                <div class="log-content">
                  <pre class="log-text">{{ jobLogs }}</pre>
                </div>
              </div>
            </a-tab-pane>
          </a-tabs>
        </div>
      </a-modal>
  
      <!-- 日志查看模态框 -->
      <a-modal 
        title="作业日志" 
        v-model:visible="isLogModalVisible" 
        @cancel="closeLogModal"
        :width="800"
        class="custom-modal"
        :footer="null"
      >
        <div class="log-viewer">
          <div class="log-toolbar">
            <a-button size="small" type="primary" @click="refreshLogs">
              <template #icon>
                <ReloadOutlined />
              </template>
              刷新
            </a-button>
            <a-button size="small" @click="downloadLogs">
              <template #icon>
                <DownloadOutlined />
              </template>
              下载
            </a-button>
            <a-switch v-model:checked="autoScrollLog" size="small" />
            <span class="log-label">自动滚动</span>
          </div>
          <div class="log-content" ref="logContentRef">
            <pre class="log-text">{{ currentJobLogs }}</pre>
          </div>
        </div>
      </a-modal>
    </div>
  </template>
  
  <script setup lang="ts">
  import { ref, reactive, onMounted, onUnmounted, watch, nextTick } from 'vue';
  import { message, Modal } from 'ant-design-vue';
  import {
    PlayCircleOutlined,
    ClockCircleOutlined,
    CheckCircleOutlined,
    ExclamationCircleOutlined,
    ReloadOutlined,
    DownloadOutlined,
    LoadingOutlined,
    FileTextOutlined,
    BarChartOutlined
  } from '@ant-design/icons-vue';
  import type { Dayjs } from 'dayjs';
  
  interface MonitorJob {
    id: number;
    name: string;
    queue: string;
    status: string;
    progress: number;
    current_task: number;
    total_tasks: number;
    cpu_usage: number;
    memory_usage: number;
    gpu_usage: number | null;
    start_time: string;
    estimated_completion?: string;
    priority: number;
    can_pause: boolean;
    last_update: string;
  }
  
  // 统计数据
  const runningJobs = ref(0);
  const pendingJobs = ref(0);
  const completedJobs = ref(0);
  const failedJobs = ref(0);
  
  // 筛选条件
  const selectedQueue = ref('');
  const selectedStatus = ref('');
  const timeRange = ref<[Dayjs, Dayjs] | null>(null);
  
  // 自动刷新
  const autoRefresh = ref(true);
  const refreshInterval = ref<NodeJS.Timeout | null>(null);
  
  // 表格数据
  const monitorData = ref<MonitorJob[]>([]);
  const loading = ref(false);
  
  // 分页
  const pageSizeOptions = ref<string[]>(['10', '20', '30', '40', '50']);
  const current = ref(1);
  const pageSizeRef = ref(20);
  const total = ref(0);
  
  // 模态框状态
  const isDetailModalVisible = ref(false);
  const isLogModalVisible = ref(false);
  const detailJob = ref<MonitorJob | null>(null);
  const currentLogJob = ref<MonitorJob | null>(null);
  
  // 日志相关
  const jobLogs = ref('');
  const currentJobLogs = ref('');
  const autoScrollLog = ref(true);
  const logContentRef = ref<HTMLElement>();
  
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
      width: 180,
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
      width: 120,
    },
    {
      title: '进度',
      key: 'progress',
      slots: { customRender: 'progress' },
      width: 180,
    },
    {
      title: '资源使用率',
      key: 'resources',
      slots: { customRender: 'resources' },
      width: 250,
    },
    {
      title: '运行时间',
      key: 'duration',
      slots: { customRender: 'duration' },
      width: 150,
    },
    {
      title: '优先级',
      dataIndex: 'priority',
      key: 'priority',
      width: 80,
    },
    {
      title: '操作',
      key: 'logs',
      slots: { customRender: 'logs' },
      width: 160,
    },
    {
      title: '管理',
      key: 'action',
      slots: { customRender: 'action' },
      width: 180,
      fixed: 'right',
    },
  ];
  
  // 初始化
  onMounted(() => {
    loadData();
    startAutoRefresh();
  });
  
  onUnmounted(() => {
    stopAutoRefresh();
  });
  
  // 监听自动刷新开关
  watch(autoRefresh, (newVal) => {
    if (newVal) {
      startAutoRefresh();
    } else {
      stopAutoRefresh();
    }
  });
  
  // 开始自动刷新
  const startAutoRefresh = () => {
    if (refreshInterval.value) {
      clearInterval(refreshInterval.value);
    }
    refreshInterval.value = setInterval(() => {
      if (autoRefresh.value) {
        loadData();
      }
    }, 5000); // 每5秒刷新一次
  };
  
  // 停止自动刷新
  const stopAutoRefresh = () => {
    if (refreshInterval.value) {
      clearInterval(refreshInterval.value);
      refreshInterval.value = null;
    }
  };
  
  // 获取状态颜色
  const getStatusColor = (status: string) => {
    const colorMap: Record<string, string> = {
      'Pending': 'orange',
      'Running': 'blue',
      'Completed': 'green',
      'Failed': 'red',
      'Paused': 'purple',
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
      'Paused': '已暂停',
      'Terminated': '已终止'
    };
    return textMap[status] || status;
  };
  
  // 获取进度状态
  const getProgressStatus = (status: string) => {
    if (status === 'Failed') return 'exception';
    if (status === 'Completed') return 'success';
    return 'active';
  };
  
  // 获取资源使用率颜色
  const getResourceColor = (usage: number) => {
    if (usage >= 90) return '#ff4d4f';
    if (usage >= 70) return '#faad14';
    return '#52c41a';
  };
  
  // 格式化运行时间
  const formatDuration = (startTime: string) => {
    const start = new Date(startTime);
    const now = new Date();
    const duration = Math.floor((now.getTime() - start.getTime()) / 1000);
    
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
  const loadData = async () => {
    loading.value = true;
    try {
      // 模拟API调用
      await new Promise(resolve => setTimeout(resolve, 500));
      
      // 模拟监控数据
      const mockData: MonitorJob[] = [
        {
          id: 1,
          name: 'pytorch-training-001',
          queue: 'default',
          status: 'Running',
          progress: 65,
          current_task: 13,
          total_tasks: 20,
          cpu_usage: 85,
          memory_usage: 72,
          gpu_usage: 94,
          start_time: '2024-06-09 10:30:00',
          estimated_completion: '2024-06-09 14:25:00',
          priority: 5,
          can_pause: true,
          last_update: new Date().toLocaleString()
        },
        {
          id: 2,
          name: 'bert-finetuning-002',
          queue: 'high-priority',
          status: 'Running',
          progress: 45,
          current_task: 9,
          total_tasks: 20,
          cpu_usage: 78,
          memory_usage: 89,
          gpu_usage: 91,
          start_time: '2024-06-09 11:15:00',
          estimated_completion: '2024-06-09 15:45:00',
          priority: 10,
          can_pause: true,
          last_update: new Date().toLocaleString()
        },
        {
          id: 3,
          name: 'data-preprocessing-003',
          queue: 'low-priority',
          status: 'Pending',
          progress: 0,
          current_task: 0,
          total_tasks: 5,
          cpu_usage: 0,
          memory_usage: 0,
          gpu_usage: null,
          start_time: '2024-06-09 12:00:00',
          priority: 1,
          can_pause: false,
          last_update: new Date().toLocaleString()
        }
      ];
      
      monitorData.value = mockData;
      total.value = mockData.length;
      
      // 更新统计数据
      updateStats(mockData);
      
    } catch (error) {
      message.error('加载数据失败');
    } finally {
      loading.value = false;
    }
  };
  
  // 更新统计数据
  const updateStats = (data: MonitorJob[]) => {
    runningJobs.value = data.filter(job => job.status === 'Running').length;
    pendingJobs.value = data.filter(job => job.status === 'Pending').length;
    completedJobs.value = data.filter(job => job.status === 'Completed').length;
    failedJobs.value = data.filter(job => job.status === 'Failed').length;
  };
  
  // 刷新数据
  const refreshData = () => {
    loadData();
    message.success('数据已刷新');
  };
  
  // 导出数据
  const exportData = () => {
    message.success('导出功能开发中...');
  };
  
  // 分页处理
  const handlePageChange = (page: number) => {
    current.value = page;
    loadData();
  };
  
  const handleSizeChange = (current: number, size: number) => {
    pageSizeRef.value = size;
    loadData();
  };
  
  // 查看详情
  const handleView = (record: MonitorJob) => {
    detailJob.value = record;
    jobLogs.value = generateMockLogs(record);
    isDetailModalVisible.value = true;
  };
  
  // 关闭详情模态框
  const closeDetailModal = () => {
    isDetailModalVisible.value = false;
    detailJob.value = null;
  };
  
  // 查看日志
  const viewLogs = (record: MonitorJob) => {
    currentLogJob.value = record;
    currentJobLogs.value = generateMockLogs(record);
    isLogModalVisible.value = true;
    
    if (autoScrollLog.value) {
      nextTick(() => {
        scrollToBottom();
      });
    }
  };
  
  // 关闭日志模态框
  const closeLogModal = () => {
    isLogModalVisible.value = false;
    currentLogJob.value = null;
  };
  
  // 查看性能指标
  const viewMetrics = (record: MonitorJob) => {
    message.info('性能指标功能开发中...');
  };
  
  // 暂停作业
  const handlePause = (record: MonitorJob) => {
    Modal.confirm({
      title: '确认暂停作业',
      content: `确定要暂停作业 "${record.name}" 吗？`,
      onOk() {
        record.status = 'Paused';
        message.success('作业已暂停');
      },
    });
  };
  
  // 恢复作业
  const handleResume = (record: MonitorJob) => {
    record.status = 'Running';
    message.success('作业已恢复');
  };
  
  // 停止作业
  const handleStop = (record: MonitorJob) => {
    Modal.confirm({
      title: '确认停止作业',
      content: `确定要停止作业 "${record.name}" 吗？`,
      onOk() {
        record.status = 'Terminated';
        message.success('作业已停止');
      },
    });
  };
  
  // 刷新日志
  const refreshLogs = () => {
    if (currentLogJob.value) {
      currentJobLogs.value = generateMockLogs(currentLogJob.value);
    }
    if (detailJob.value) {
      jobLogs.value = generateMockLogs(detailJob.value);
    }
    message.success('日志已刷新');
  };
  
  // 下载日志
  const downloadLogs = () => {
    message.success('下载功能开发中...');
  };
  
  // 滚动到底部
  const scrollToBottom = () => {
    if (logContentRef.value) {
      logContentRef.value.scrollTop = logContentRef.value.scrollHeight;
    }
  };
  
  // 生成模拟日志
  const generateMockLogs = (job: MonitorJob) => {
    const logs = [
      `[${new Date().toLocaleString()}] INFO: 开始执行作业 ${job.name}`,
      `[${new Date().toLocaleString()}] INFO: 加载模型配置...`,
      `[${new Date().toLocaleString()}] INFO: 初始化训练参数...`,
      `[${new Date().toLocaleString()}] INFO: 开始训练第1轮...`,
      `[${new Date().toLocaleString()}] INFO: Epoch 1/100 - Loss: 0.8245, Accuracy: 0.7234`,
      `[${new Date().toLocaleString()}] INFO: 开始训练第2轮...`,
      `[${new Date().toLocaleString()}] INFO: Epoch 2/100 - Loss: 0.7123, Accuracy: 0.7891`,
      `[${new Date().toLocaleString()}] INFO: 保存检查点...`,
      `[${new Date().toLocaleString()}] INFO: 当前进度: ${job.progress}%`,
      `[${new Date().toLocaleString()}] INFO: CPU使用率: ${job.cpu_usage}%, 内存使用率: ${job.memory_usage}%`,
    ];
    
    if (job.gpu_usage !== null) {
      logs.push(`[${new Date().toLocaleString()}] INFO: GPU使用率: ${job.gpu_usage}%`);
    }
    
    return logs.join('\n');
  };
  </script>
  
  <style scoped>
  .job-monitor-page {
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
  
  .overview-cards {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
    gap: 20px;
    margin-bottom: 24px;
  }
  
  .stats-card {
    padding: 20px;
    background: white;
    border-radius: 8px;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  }
  
  .stats-content {
    display: flex;
    align-items: center;
    gap: 16px;
  }
  
  .stats-icon {
    width: 48px;
    height: 48px;
    border-radius: 8px;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 24px;
    color: white;
  }
  
  .stats-icon.running {
    background: linear-gradient(135deg, #3b82f6, #1d4ed8);
  }
  
  .stats-icon.pending {
    background: linear-gradient(135deg, #f59e0b, #d97706);
  }
  
  .stats-icon.completed {
    background: linear-gradient(135deg, #10b981, #059669);
  }
  
  .stats-icon.failed {
    background: linear-gradient(135deg, #ef4444, #dc2626);
  }
  
  .stats-info {
    flex: 1;
  }
  
  .stats-value {
    font-size: 28px;
    font-weight: 700;
    color: #1a202c;
    line-height: 1;
  }
  
  .stats-label {
    font-size: 14px;
    color: #64748b;
    margin-top: 4px;
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
  
  .queue-filter,
  .status-filter {
    width: 150px;
  }
  
  .time-filter {
    width: 350px;
  }
  
  .action-button {
    height: 32px;
  }
  
  .export-button {
    background: #f1f5f9;
    border-color: #e2e8f0;
    color: #475569;
  }
  
  .action-buttons {
    display: flex;
    gap: 12px;
  }
  
  .table-container {
    padding: 0;
  }
  
  .table-header {
    padding: 20px 20px 0 20px;
    display: flex;
    justify-content: space-between;
    align-items: center;
  }
  
  .table-title {
    font-size: 16px;
    font-weight: 600;
    color: #1a202c;
    margin: 0;
  }
  
  .auto-refresh {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  
  .refresh-label {
    font-size: 14px;
    color: #64748b;
  }
  
  .monitor-table {
    margin-top: 20px;
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
  
  .status-cell {
    display: flex;
    align-items: center;
  }
  
  .status-tag {
    border-radius: 4px;
    font-size: 12px;
    font-weight: 500;
  }
  
  .progress-cell {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  
  .progress-text {
    font-size: 12px;
    color: #64748b;
    text-align: center;
  }
  
  .resource-usage {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
  
  .resource-item {
    display: flex;
    align-items: center;
    gap: 8px;
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
    min-width: 35px;
  }
  
  .duration-cell {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  
  .duration-value {
    font-size: 12px;
    color: #1a202c;
    font-family: monospace;
    font-weight: 500;
  }
  
  .eta-value {
    font-size: 11px;
    color: #64748b;
  }
  
  .log-cell {
    display: flex;
    flex-direction: column;
    gap: 4px;
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
  
  .job-detail-monitor {
    margin-top: 20px;
  }
  
  .detail-section {
    margin-bottom: 24px;
  }
  
  .resource-monitor {
    display: flex;
    flex-direction: column;
    gap: 20px;
  }
  
  .resource-chart h4 {
    margin: 0 0 12px 0;
    font-size: 14px;
    font-weight: 600;
    color: #1a202c;
  }
  
  .log-container,
  .log-viewer {
    height: 400px;
    display: flex;
    flex-direction: column;
  }
  
  .log-header,
  .log-toolbar {
    padding: 12px;
    border-bottom: 1px solid #e2e8f0;
    display: flex;
    align-items: center;
    gap: 12px;
  }
  
  .log-label {
    font-size: 14px;
    color: #64748b;
  }
  
  .log-content {
    flex: 1;
    overflow-y: auto;
    padding: 12px;
    background: #f8fafc;
  }
  
  .log-text {
    font-size: 12px;
    font-family: 'Courier New', monospace;
    color: #1a202c;
    margin: 0;
    white-space: pre-wrap;
    word-break: break-all;
  }
  
  @media (max-width: 768px) {
    .overview-cards {
      grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
      gap: 16px;
    }
    
    .custom-toolbar {
      flex-direction: column;
      align-items: stretch;
    }
    
    .search-filters {
      justify-content: stretch;
    }
    
    .queue-filter,
    .status-filter,
    .time-filter {
      width: 100%;
      min-width: auto;
    }
    
    .action-buttons {
      justify-content: center;
    }
    
    .table-header {
      flex-direction: column;
      align-items: flex-start;
      gap: 12px;
    }
  }
  </style>