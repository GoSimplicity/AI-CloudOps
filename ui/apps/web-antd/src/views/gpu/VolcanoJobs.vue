<template>
    <div>
      <div class="job-management-page">
        <!-- 页面标题区域 -->
        <div class="page-header">
          <h2 class="page-title">作业管理</h2>
          <p class="page-description">管理和监控您的Volcano作业队列</p>
        </div>
  
        <!-- 统计卡片区域 -->
        <div class="stats-grid">
          <div class="stat-card">
            <div class="stat-icon running">
              <svg viewBox="0 0 24 24" fill="currentColor">
                <path d="M8 5v14l11-7z"/>
              </svg>
            </div>
            <div class="stat-content">
              <div class="stat-number">{{ stats.running }}</div>
              <div class="stat-label">运行中</div>
            </div>
          </div>
          
          <div class="stat-card">
            <div class="stat-icon pending">
              <svg viewBox="0 0 24 24" fill="currentColor">
                <path d="M12,2A10,10 0 0,0 2,12A10,10 0 0,0 12,22A10,10 0 0,0 22,12A10,10 0 0,0 12,2M12,20A8,8 0 0,1 4,12A8,8 0 0,1 12,4A8,8 0 0,1 20,12A8,8 0 0,1 12,20M12,6A6,6 0 0,0 6,12A6,6 0 0,0 12,18A6,6 0 0,0 18,12A6,6 0 0,0 12,6Z"/>
              </svg>
            </div>
            <div class="stat-content">
              <div class="stat-number">{{ stats.pending }}</div>
              <div class="stat-label">等待中</div>
            </div>
          </div>
          
          <div class="stat-card">
            <div class="stat-icon completed">
              <svg viewBox="0 0 24 24" fill="currentColor">
                <path d="M9,20.42L2.79,14.21L5.62,11.38L9,14.77L18.88,4.88L21.71,7.71L9,20.42Z"/>
              </svg>
            </div>
            <div class="stat-content">
              <div class="stat-number">{{ stats.completed }}</div>
              <div class="stat-label">已完成</div>
            </div>
          </div>
          
          <div class="stat-card">
            <div class="stat-icon failed">
              <svg viewBox="0 0 24 24" fill="currentColor">
                <path d="M19,6.41L17.59,5L12,10.59L6.41,5L5,6.41L10.59,12L5,17.59L6.41,19L12,13.41L17.59,19L19,17.59L13.41,12L19,6.41Z"/>
              </svg>
            </div>
            <div class="stat-content">
              <div class="stat-number">{{ stats.failed }}</div>
              <div class="stat-label">失败</div>
            </div>
          </div>
        </div>
  
        <!-- 查询和操作工具栏 -->
        <div class="dashboard-card custom-toolbar">
          <div class="search-filters">
            <div class="search-input-wrapper">
              <svg class="search-icon" viewBox="0 0 24 24" fill="currentColor">
                <path d="M9.5,3A6.5,6.5 0 0,1 16,9.5C16,11.11 15.41,12.59 14.44,13.73L14.71,14H15.5L20.5,19L19,20.5L14,15.5V14.71L13.73,14.44C12.59,15.41 11.11,16 9.5,16A6.5,6.5 0 0,1 3,9.5A6.5,6.5 0 0,1 9.5,3M9.5,5C7,5 5,7 5,9.5C5,12 7,14 9.5,14C12,14 14,12 14,9.5C14,7 12,5 9.5,5Z"/>
              </svg>
              <input 
                v-model="searchText" 
                placeholder="请输入作业名称" 
                class="search-input"
              />
            </div>
            
            <select v-model="statusFilter" class="filter-select">
              <option value="">全部状态</option>
              <option value="Pending">等待中</option>
              <option value="Running">运行中</option>
              <option value="Completed">已完成</option>
              <option value="Failed">失败</option>
              <option value="Terminated">已终止</option>
            </select>
            
            <select v-model="queueFilter" class="filter-select">
              <option value="">全部队列</option>
              <option value="default">default</option>
              <option value="high-priority">high-priority</option>
              <option value="low-priority">low-priority</option>
            </select>
            
            <button class="action-button primary" @click="handleSearch">
              <svg viewBox="0 0 24 24" fill="currentColor">
                <path d="M9.5,3A6.5,6.5 0 0,1 16,9.5C16,11.11 15.41,12.59 14.44,13.73L14.71,14H15.5L20.5,19L19,20.5L14,15.5V14.71L13.73,14.44C12.59,15.41 11.11,16 9.5,16A6.5,6.5 0 0,1 3,9.5A6.5,6.5 0 0,1 9.5,3M9.5,5C7,5 5,7 5,9.5C5,12 7,14 9.5,14C12,14 14,12 14,9.5C14,7 12,5 9.5,5Z"/>
              </svg>
              搜索
            </button>
            
            <button class="action-button reset" @click="handleReset">
              <svg viewBox="0 0 24 24" fill="currentColor">
                <path d="M12,6V9L16,5L12,1V4A8,8 0 0,0 4,12C4,13.57 4.46,15.03 5.24,16.26L6.7,14.8C6.25,13.97 6,13 6,12A6,6 0 0,1 12,6M18.76,7.74L17.3,9.2C17.74,10.04 18,11 18,12A6,6 0 0,1 12,18V15L8,19L12,23V20A8,8 0 0,0 20,12C20,10.43 19.54,8.97 18.76,7.74Z"/>
              </svg>
              重置
            </button>
          </div>
          
          <div class="action-buttons">
            <button class="action-button primary add-button" @click="showAddModal">
              <svg viewBox="0 0 24 24" fill="currentColor">
                <path d="M19,13H13V19H11V13H5V11H11V5H13V11H19V13Z"/>
              </svg>
              创建作业
            </button>
          </div>
        </div>
  
        <!-- 作业列表表格 -->
        <div class="dashboard-card table-container">
          <div class="table-wrapper">
            <table class="custom-table">
              <thead>
                <tr>
                  <th>ID</th>
                  <th>作业名称</th>
                  <th>命名空间</th>
                  <th>队列</th>
                  <th>状态</th>
                  <th>优先级</th>
                  <th>任务数</th>
                  <th>资源需求</th>
                  <th>容器镜像</th>
                  <th>运行时间</th>
                  <th>创建者</th>
                  <th>创建时间</th>
                  <th>操作</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="job in paginatedJobs" :key="job.id">
                  <td>{{ job.id }}</td>
                  <td>{{ job.name }}</td>
                  <td>{{ job.namespace }}</td>
                  <td>{{ job.queue }}</td>
                  <td>
                    <span :class="`status-tag ${job.status.toLowerCase()}`">
                      {{ getStatusText(job.status) }}
                    </span>
                  </td>
                  <td>
                    <span :class="`priority-tag ${getPriorityClass(job.priority)}`">
                      {{ getPriorityText(job.priority) }}
                    </span>
                  </td>
                  <td>{{ job.task_count }}</td>
                  <td>
                    <div class="resource-container">
                      <div class="resource-item">
                        <span class="resource-label">CPU:</span>
                        <span class="resource-value">{{ job.cpu_request }}</span>
                      </div>
                      <div class="resource-item">
                        <span class="resource-label">内存:</span>
                        <span class="resource-value">{{ job.memory_request }}</span>
                      </div>
                      <div class="resource-item">
                        <span class="resource-label">GPU:</span>
                        <span class="resource-value">{{ job.gpu_request }}</span>
                      </div>
                    </div>
                  </td>
                  <td>
                    <div class="image-container" :title="job.image">
                      {{ job.image.split('/').pop() }}
                    </div>
                  </td>
                  <td>
                    <div class="duration-container">
                      {{ formatDuration(job.start_time, job.completion_time) }}
                    </div>
                  </td>
                  <td>{{ job.creator }}</td>
                  <td>{{ job.created_at }}</td>
                  <td>
                    <div class="action-column">
                      <button class="table-action-btn view" @click="handleView(job)">查看</button>
                      <button v-if="job.status === 'Pending'" class="table-action-btn edit" @click="handleEdit(job)">编辑</button>
                      <button v-if="['Pending', 'Running'].includes(job.status)" class="table-action-btn stop" @click="handleStop(job)">停止</button>
                      <button v-if="['Completed', 'Failed', 'Terminated'].includes(job.status)" class="table-action-btn delete" @click="handleDelete(job)">删除</button>
                    </div>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
  
          <!-- 分页器 -->
          <div class="pagination-container">
            <div class="pagination-info">
              显示 {{ (currentPage - 1) * pageSize + 1 }} - {{ Math.min(currentPage * pageSize, filteredJobs.length) }} 条，共 {{ filteredJobs.length }} 条
            </div>
            <div class="pagination">
              <button 
                class="pagination-btn" 
                :disabled="currentPage === 1"
                @click="goToPage(currentPage - 1)"
              >
                上一页
              </button>
              <span class="pagination-current">{{ currentPage }} / {{ totalPages }}</span>
              <button 
                class="pagination-btn" 
                :disabled="currentPage === totalPages"
                @click="goToPage(currentPage + 1)"
              >
                下一页
              </button>
            </div>
          </div>
        </div>
  
        <!-- 创建作业模态框 -->
        <div v-if="isAddModalVisible" class="modal-overlay" @click="closeAddModal">
          <div class="modal-container" @click.stop>
            <div class="modal-header">
              <h3>创建训练作业</h3>
              <button class="modal-close" @click="closeAddModal">
                <svg viewBox="0 0 24 24" fill="currentColor">
                  <path d="M19,6.41L17.59,5L12,10.59L6.41,5L5,6.41L10.59,12L5,17.59L6.41,19L12,13.41L17.59,19L19,17.59L13.41,12L19,6.41Z"/>
                </svg>
              </button>
            </div>
            <div class="modal-body">
              <form class="custom-form" @submit.prevent="handleAdd">
                <div class="form-section">
                  <div class="section-title">基本信息</div>
                  <div class="form-grid">
                    <div class="form-group">
                      <label>作业名称 *</label>
                      <input v-model="addForm.name" placeholder="请输入作业名称" required />
                    </div>
                    <div class="form-group">
                      <label>队列名称 *</label>
                      <select v-model="addForm.queue" required>
                        <option value="default">default</option>
                        <option value="high-priority">high-priority</option>
                        <option value="low-priority">low-priority</option>
                      </select>
                    </div>
                    <div class="form-group">
                      <label>优先级</label>
                      <select v-model="addForm.priority">
                        <option :value="1">低</option>
                        <option :value="5">中</option>
                        <option :value="10">高</option>
                      </select>
                    </div>
                    <div class="form-group">
                      <label>任务数量</label>
                      <input type="number" v-model="addForm.task_count" min="1" max="100" />
                    </div>
                  </div>
                </div>
  
                <div class="form-section">
                  <div class="section-title">容器配置</div>
                  <div class="form-group full-width">
                    <label>容器镜像 *</label>
                    <input v-model="addForm.image" placeholder="例如: pytorch/pytorch:1.12.0-cuda11.3-cudnn8-devel" required />
                  </div>
                  <div class="form-group full-width">
                    <label>启动命令</label>
                    <textarea v-model="addForm.command" placeholder="请输入启动命令，多行命令用换行分隔" rows="3"></textarea>
                  </div>
                </div>
  
                <div class="form-section">
                  <div class="section-title">资源配置</div>
                  <div class="form-grid">
                    <div class="form-group">
                      <label>CPU需求</label>
                      <input v-model="addForm.cpu_request" placeholder="例如: 2" />
                    </div>
                    <div class="form-group">
                      <label>内存需求</label>
                      <input v-model="addForm.memory_request" placeholder="例如: 4Gi" />
                    </div>
                    <div class="form-group">
                      <label>GPU需求</label>
                      <input type="number" v-model="addForm.gpu_request" min="0" max="8" />
                    </div>
                  </div>
                </div>
  
                <div class="modal-footer">
                  <button type="button" class="action-button" @click="closeAddModal">取消</button>
                  <button type="submit" class="action-button primary">创建</button>
                </div>
              </form>
            </div>
          </div>
        </div>
  
        <!-- 作业详情模态框 -->
        <div v-if="isViewModalVisible" class="modal-overlay" @click="closeViewModal">
          <div class="modal-container large" @click.stop>
            <div class="modal-header">
              <h3>作业详情</h3>
              <button class="modal-close" @click="closeViewModal">
                <svg viewBox="0 0 24 24" fill="currentColor">
                  <path d="M19,6.41L17.59,5L12,10.59L6.41,5L5,6.41L10.59,12L5,17.59L6.41,19L12,13.41L17.59,19L19,17.59L13.41,12L19,6.41Z"/>
                </svg>
              </button>
            </div>
            <div class="modal-body">
              <div class="job-detail-container" v-if="viewJob">
                <div class="detail-section">
                  <div class="section-title">基本信息</div>
                  <div class="detail-grid">
                    <div class="detail-item">
                      <span class="detail-label">作业名称:</span>
                      <span class="detail-value">{{ viewJob.name }}</span>
                    </div>
                    <div class="detail-item">
                      <span class="detail-label">命名空间:</span>
                      <span class="detail-value">{{ viewJob.namespace }}</span>
                    </div>
                    <div class="detail-item">
                      <span class="detail-label">队列名称:</span>
                      <span class="detail-value">{{ viewJob.queue }}</span>
                    </div>
                    <div class="detail-item">
                      <span class="detail-label">状态:</span>
                      <span :class="`status-tag ${viewJob.status.toLowerCase()}`">{{ getStatusText(viewJob.status) }}</span>
                    </div>
                    <div class="detail-item">
                      <span class="detail-label">优先级:</span>
                      <span :class="`priority-tag ${getPriorityClass(viewJob.priority)}`">{{ getPriorityText(viewJob.priority) }}</span>
                    </div>
                    <div class="detail-item">
                      <span class="detail-label">任务数量:</span>
                      <span class="detail-value">{{ viewJob.task_count }}</span>
                    </div>
                    <div class="detail-item">
                      <span class="detail-label">创建时间:</span>
                      <span class="detail-value">{{ viewJob.created_at }}</span>
                    </div>
                    <div class="detail-item">
                      <span class="detail-label">开始时间:</span>
                      <span class="detail-value">{{ viewJob.start_time || '未开始' }}</span>
                    </div>
                  </div>
                </div>
  
                <div class="detail-section">
                  <div class="section-title">资源信息</div>
                  <div class="detail-grid">
                    <div class="detail-item">
                      <span class="detail-label">CPU需求:</span>
                      <span class="detail-value">{{ viewJob.cpu_request }}</span>
                    </div>
                    <div class="detail-item">
                      <span class="detail-label">内存需求:</span>
                      <span class="detail-value">{{ viewJob.memory_request }}</span>
                    </div>
                    <div class="detail-item">
                      <span class="detail-label">GPU需求:</span>
                      <span class="detail-value">{{ viewJob.gpu_request }}</span>
                    </div>
                  </div>
                </div>
  
                <div class="detail-section">
                  <div class="section-title">容器配置</div>
                  <div class="detail-item full-width">
                    <span class="detail-label">镜像:</span>
                    <span class="detail-value">{{ viewJob.image }}</span>
                  </div>
                  <div class="detail-item full-width">
                    <span class="detail-label">启动命令:</span>
                    <pre class="command-pre">{{ viewJob.command }}</pre>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </template>
  
  <script setup lang="ts">
  import { ref, reactive, computed, onMounted } from 'vue';
  
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
    created_at: string;
    start_time?: string;
    completion_time?: string;
    creator: string;
  }
  
  // 响应式数据
  const searchText = ref('');
  const statusFilter = ref('');
  const queueFilter = ref('');
  const currentPage = ref(1);
  const pageSize = ref(10);
  
  // 模态框状态
  const isAddModalVisible = ref(false);
  const isViewModalVisible = ref(false);
  const viewJob = ref<JobItem | null>(null);
  
  // 统计数据
  const stats = reactive({
    running: 5,
    pending: 3,
    completed: 12,
    failed: 2
  });
  
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
    gpu_request: 1
  });
  
  // 作业数据
  const jobs = ref<JobItem[]>([
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
      created_at: '2024-06-09 09:00:00',
      start_time: '2024-06-09 09:05:00',
      completion_time: '2024-06-09 10:30:00',
      creator: 'user2'
    }
  ]);
  
  // 计算属性
  const filteredJobs = computed(() => {
    return jobs.value.filter(job => {
      const matchesSearch = !searchText.value || job.name.toLowerCase().includes(searchText.value.toLowerCase());
      const matchesStatus = !statusFilter.value || job.status === statusFilter.value;
      const matchesQueue = !queueFilter.value || job.queue === queueFilter.value;
      return matchesSearch && matchesStatus && matchesQueue;
    });
  });
  
  const totalPages = computed(() => Math.ceil(filteredJobs.value.length / pageSize.value));
  
  const paginatedJobs = computed(() => {
    const start = (currentPage.value - 1) * pageSize.value;
    const end = start + pageSize.value;
    return filteredJobs.value.slice(start, end);
  });
  
  // 工具函数
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
  
  const getPriorityClass = (priority: number) => {
    if (priority >= 10) return 'high';
    if (priority >= 5) return 'medium';
    return 'low';
  };
  
  const getPriorityText = (priority: number) => {
    if (priority >= 10) return '高';
    if (priority >= 5) return '中';
    return '低';
  };
  
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
  
  // 事件处理函数
  const handleSearch = () => {
    currentPage.value = 1;
    console.log('搜索作业');
  };
  
  const handleReset = () => {
    searchText.value = '';
    statusFilter.value = '';
    queueFilter.value = '';
    currentPage.value = 1;
    console.log('重置搜索条件');
  };
  
  const goToPage = (page: number) => {
    if (page >= 1 && page <= totalPages.value) {
      currentPage.value = page;
    }
  };
  
  const showAddModal = () => {
    resetAddForm();
    isAddModalVisible.value = true;
  };
  
  const closeAddModal = () => {
    isAddModalVisible.value = false;
    resetAddForm();
  };
  
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
      gpu_request: 1
    });
  };
  
  const handleAdd = () => {
    const newJob: JobItem = {
      ...addForm,
      id: Math.max(...jobs.value.map(j => j.id)) + 1,
      namespace: 'default',
      status: 'Pending',
      created_at: new Date().toLocaleString(),
      creator: 'admin'
    };
    
    jobs.value.unshift(newJob);
    stats.pending++;
    closeAddModal();
    console.log('创建作业:', newJob);
  };
  
  const handleView = (job: JobItem) => {
    viewJob.value = job;
    isViewModalVisible.value = true;
  };
  
  const closeViewModal = () => {
    isViewModalVisible.value = false;
    viewJob.value = null;
  };
  
  const handleEdit = (job: JobItem) => {
    console.log('编辑作业:', job);
  };
  
  const handleStop = (job: JobItem) => {
    if (confirm(`确定要停止作业 "${job.name}" 吗？`)) {
      job.status = 'Terminated';
      job.completion_time = new Date().toLocaleString();
      if (job.status === 'Running') stats.running--;
      if (job.status === 'Pending') stats.pending--;
      console.log('停止作业:', job);
    }
  };
  
  const handleDelete = (job: JobItem) => {
    if (confirm(`确定要删除作业 "${job.name}" 吗？此操作不可恢复。`)) {
      const index = jobs.value.findIndex(j => j.id === job.id);
      if (index !== -1) {
        jobs.value.splice(index, 1);
        console.log('删除作业:', job);
      }
    }
  };
  
  onMounted(() => {
    console.log('volcano作业管理页面已加载');
  });
  </script>
  
  <style scoped>
  .job-management-page {
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
    margin: 0;
  }
  
  .stats-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
    gap: 20px;
    margin-bottom: 24px;
  }
  
  .stat-card {
    background: white;
    border-radius: 8px;
    padding: 20px;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
    display: flex;
    align-items: center;
    gap: 16px;
  }
  
  .stat-icon {
    width: 48px;
    height: 48px;
    border-radius: 8px;
    display: flex;
    align-items: center;
    justify-content: center;
  }
  
  .stat-icon svg {
    width: 24px;
    height: 24px;
  }
  
  .stat-icon.running {
    background-color: #dbeafe;
    color: #3b82f6;
  }
  
  .stat-icon.pending {
    background-color: #fef3c7;
    color: #f59e0b;
  }
  
  .stat-icon.completed {
    background-color: #d1fae5;
    color: #10b981;
  }
  
  .stat-icon.failed {
    background-color: #fee2e2;
    color: #ef4444;
  }
  
  .stat-content {
    flex: 1;
  }
  
  .stat-number {
    font-size: 24px;
    font-weight: 700;
    color: #1a202c;
    line-height: 1;
  }
  
  .stat-label {
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
  
  .search-input-wrapper {
    position: relative;
    display: flex;
    align-items: center;
  }
  
  .search-icon {
    position: absolute;
    left: 12px;
    width: 16px;
    height: 16px;
    color: #64748b;
    z-index: 1;
  }
  
  .search-input {
    width: 200px;
    padding: 8px 12px 8px 36px;
    border: 1px solid #e2e8f0;
    border-radius: 6px;
    font-size: 14px;
  }
  
  .search-input:focus {
    outline: none;
    border-color: #3b82f6;
    box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
  }
  
  .filter-select {
    width: 150px;
    padding: 8px 12px;
    border: 1px solid #e2e8f0;
    border-radius: 6px;
    font-size: 14px;
    background: white;
  }
  
  .filter-select:focus {
    outline: none;
    border-color: #3b82f6;
    box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
  }
  
  .action-button {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 16px;
    border: 1px solid #e2e8f0;
    border-radius: 6px;
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.2s;
    background: white;
    color: #374151;
  }
  
  .action-button svg {
    width: 16px;
    height: 16px;
  }
  
  .action-button:hover {
    background: #f8fafc;
  }
  
  .action-button.primary {
    background: #3b82f6;
    border-color: #3b82f6;
    color: white;
  }
  
  .action-button.primary:hover {
    background: #2563eb;
  }
  
  .action-button.reset {
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
  
  .table-wrapper {
    overflow-x: auto;
  }
  
  .custom-table {
    width: 100%;
    border-collapse: collapse;
    font-size: 14px;
  }
  
  .custom-table th {
    background-color: #f8fafc;
    border-bottom: 1px solid #e2e8f0;
    color: #374151;
    font-weight: 600;
    padding: 12px;
    text-align: left;
    white-space: nowrap;
  }
  
  .custom-table td {
    padding: 12px;
    border-bottom: 1px solid #f1f5f9;
    vertical-align: top;
  }
  
  .custom-table tbody tr:hover {
    background-color: #f8fafc;
  }
  
  .status-tag, .priority-tag {
    display: inline-block;
    padding: 4px 8px;
    border-radius: 4px;
    font-size: 12px;
    font-weight: 500;
    white-space: nowrap;
  }
  
  .status-tag.pending {
    background-color: #fef3c7;
    color: #92400e;
  }
  
  .status-tag.running {
    background-color: #dbeafe;
    color: #1e40af;
  }
  
  .status-tag.completed {
    background-color: #d1fae5;
    color: #065f46;
  }
  
  .status-tag.failed, .status-tag.terminated {
    background-color: #fee2e2;
    color: #991b1b;
  }
  
  .priority-tag.low {
    background-color: #d1fae5;
    color: #065f46;
  }
  
  .priority-tag.medium {
    background-color: #fef3c7;
    color: #92400e;
  }
  
  .priority-tag.high {
    background-color: #fee2e2;
    color: #991b1b;
  }
  
  .resource-container {
    display: flex;
    flex-direction: column;
    gap: 4px;
    min-width: 160px;
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
    gap: 4px;
    flex-wrap: wrap;
  }
  
  .table-action-btn {
    padding: 4px 8px;
    border: none;
    border-radius: 4px;
    font-size: 12px;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.2s;
  }
  
  .table-action-btn.view {
    background: #10b981;
    color: white;
  }
  
  .table-action-btn.edit {
    background: #f59e0b;
    color: white;
  }
  
  .table-action-btn.stop, .table-action-btn.delete {
    background: #ef4444;
    color: white;
  }
  
  .table-action-btn:hover {
    opacity: 0.8;
  }
  
  .pagination-container {
    padding: 20px;
    display: flex;
    justify-content: space-between;
    align-items: center;
    border-top: 1px solid #e2e8f0;
  }
  
  .pagination-info {
    font-size: 14px;
    color: #64748b;
  }
  
  .pagination {
    display: flex;
    align-items: center;
    gap: 12px;
  }
  
  .pagination-btn {
    padding: 8px 12px;
    border: 1px solid #e2e8f0;
    border-radius: 6px;
    font-size: 14px;
    background: white;
    cursor: pointer;
    transition: all 0.2s;
  }
  
  .pagination-btn:hover:not(:disabled) {
    background: #f8fafc;
  }
  
  .pagination-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
  
  .pagination-current {
    font-size: 14px;
    color: #374151;
    font-weight: 500;
  }
  
  .modal-overlay {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
  }
  
  .modal-container {
    background: white;
    border-radius: 8px;
    width: 90%;
    max-width: 800px;
    max-height: 90vh;
    overflow: hidden;
    box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.1);
  }
  
  .modal-container.large {
    max-width: 900px;
  }
  
  .modal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 16px 24px;
    border-bottom: 1px solid #e2e8f0;
  }
  
  .modal-header h3 {
    font-size: 18px;
    font-weight: 600;
    color: #1a202c;
    margin: 0;
  }
  
  .modal-close {
    background: none;
    border: none;
    padding: 4px;
    cursor: pointer;
    color: #64748b;
  }
  
  .modal-close svg {
    width: 20px;
    height: 20px;
  }
  
  .modal-body {
    padding: 24px;
    max-height: calc(90vh - 120px);
    overflow-y: auto;
  }
  
  .custom-form {
    display: flex;
    flex-direction: column;
    gap: 24px;
  }
  
  .form-section {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }
  
  .section-title {
    font-size: 16px;
    font-weight: 600;
    color: #1a202c;
    padding-bottom: 8px;
    border-bottom: 1px solid #e2e8f0;
  }
  
  .form-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: 16px;
  }
  
  .form-group {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  
  .form-group.full-width {
    grid-column: 1 / -1;
  }
  
  .form-group label {
    font-size: 14px;
    font-weight: 500;
    color: #374151;
  }
  
  .form-group input,
  .form-group select,
  .form-group textarea {
    padding: 8px 12px;
    border: 1px solid #e2e8f0;
    border-radius: 6px;
    font-size: 14px;
  }
  
  .form-group input:focus,
  .form-group select:focus,
  .form-group textarea:focus {
    outline: none;
    border-color: #3b82f6;
    box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
  }
  
  .modal-footer {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
    padding-top: 24px;
    border-top: 1px solid #e2e8f0;
  }
  
  .job-detail-container {
    display: flex;
    flex-direction: column;
    gap: 24px;
  }
  
  .detail-section {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }
  
  .detail-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
    gap: 12px;
  }
  
  .detail-item {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  
  .detail-item.full-width {
    grid-column: 1 / -1;
  }
  
  .detail-label {
    font-size: 12px;
    font-weight: 500;
    color: #64748b;
  }
  
  .detail-value {
    font-size: 14px;
    color: #1a202c;
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
    font-family: monospace;
  }
  
  @media (max-width: 768px) {
    .job-management-page {
      padding: 12px;
    }
    
    .stats-grid {
      grid-template-columns: 1fr;
    }
    
    .custom-toolbar {
      flex-direction: column;
      align-items: stretch;
    }
    
    .search-filters {
      flex-direction: column;
    }
    
    .search-input,
    .filter-select {
      width: 100%;
    }
    
    .action-buttons {
      justify-content: center;
    }
    
    .modal-container {
      width: 95%;
      margin: 20px;
    }
    
    .form-grid {
      grid-template-columns: 1fr;
    }
    
    .detail-grid {
      grid-template-columns: 1fr;
    }
  }
  </style>