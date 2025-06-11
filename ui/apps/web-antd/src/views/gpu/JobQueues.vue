<template>
  <div class="job-queue-page">
    <!-- 页面标题区域 -->
    <transition name="slide-down" appear>
      <div class="page-header">
        <h2 class="page-title">作业管理</h2>
        <div class="page-description">管理作业队列，监控作业状态</div>
        <div class="stats-cards">
          <div class="stat-card" v-for="(stat, index) in stats" :key="stat.label">
            <transition name="count-up" appear :style="{ transitionDelay: `${index * 100}ms` }">
              <div class="stat-content">
                <div class="stat-icon" :style="{ backgroundColor: stat.color }">
                  <component :is="stat.icon" />
                </div>
                <div class="stat-info">
                  <div class="stat-value">{{ stat.value }}</div>
                  <div class="stat-label">{{ stat.label }}</div>
                </div>
              </div>
            </transition>
          </div>
        </div>
      </div>
    </transition>

    <!-- 查询和操作工具栏 -->
    <transition name="slide-up" appear>
      <div class="dashboard-card custom-toolbar">
        <div class="search-filters">
          <a-input 
            v-model:value="searchText" 
            placeholder="请输入作业名称" 
            class="search-input"
            @pressEnter="handleSearch"
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
          <a-button type="primary" class="action-button" @click="handleSearch" :loading="loading">
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
          <a-button class="action-button" @click="handleRefresh" :loading="loading">
            <template #icon>
              <ReloadOutlined />
            </template>
            刷新
          </a-button>
          <a-button class="action-button" @click="handleBatchDelete" :disabled="selectedRowKeys.length === 0">
            <template #icon>
              <DeleteOutlined />
            </template>
            批量删除
          </a-button>
        </div>
      </div>
    </transition>

    <!-- 作业列表表格 -->
    <transition name="fade-in" appear>
      <div class="dashboard-card table-container">
        <a-table 
          :columns="columns" 
          :data-source="filteredData" 
          row-key="id" 
          :pagination="paginationConfig"
          class="custom-table"
          :scroll="{ x: 1400 }"
          :loading="loading"
          :row-selection="rowSelection"
          @change="handleTableChange"
        >
          <!-- 作业状态列 -->
          <template #status="{ record }">
            <a-tag :color="getStatusColor(record.status)" class="status-tag">
              <component :is="getStatusIcon(record.status)" class="status-icon" />
              {{ getStatusText(record.status) }}
            </a-tag>
          </template>
          
          <!-- 资源需求列 -->
          <template #resources="{ record }">
            <div class="resource-container">
              <div class="resource-item">
                <CpuIcon class="resource-icon" />
                <span class="resource-label">CPU:</span>
                <span class="resource-value">{{ record.cpu_request }}</span>
              </div>
              <div class="resource-item">
                <MemoryIcon class="resource-icon" />
                <span class="resource-label">内存:</span>
                <span class="resource-value">{{ record.memory_request }}</span>
              </div>
              <div class="resource-item">
                <GpuIcon class="resource-icon" />
                <span class="resource-label">GPU:</span>
                <span class="resource-value">{{ record.gpu_request }}</span>
              </div>
            </div>
          </template>
          
          <!-- 镜像列 -->
          <template #image="{ record }">
            <a-tooltip :title="record.image">
              <div class="image-container">
                <LockOutlined class="image-icon" />
                {{ record.image.split('/').pop() }}
              </div>
            </a-tooltip>
          </template>
          
          <!-- 优先级列 -->
          <template #priority="{ record }">
            <a-tag :color="getPriorityColor(record.priority)" class="priority-tag">
              <component :is="getPriorityIcon(record.priority)" class="priority-icon" />
              {{ getPriorityText(record.priority) }}
            </a-tag>
          </template>
          
          <!-- 运行时间列 -->
          <template #duration="{ record }">
            <div class="duration-container">
              <ClockCircleOutlined class="duration-icon" />
              {{ formatDuration(record.start_time, record.completion_time) }}
            </div>
          </template>

          <!-- 进度列 -->
          <template #progress="{ record }">
            <div class="progress-container">
              <a-progress 
                :percent="getJobProgress(record)" 
                :status="getProgressStatus(record)"
                size="small"
                :show-info="false"
              />
              <span class="progress-text">{{ getJobProgress(record) }}%</span>
            </div>
          </template>
          
          <!-- 操作列 -->
          <template #action="{ record }">
            <div class="action-column">
              <a-button type="primary" size="small" @click="handleView(record)" class="view-btn">
                <EyeOutlined />
                查看
              </a-button>
              <a-button 
                type="default" 
                size="small" 
                @click="handleEdit(record)" 
                v-if="record.status === 'Pending'"
                class="edit-btn"
              >
                <EditOutlined />
                编辑
              </a-button>
              <a-button 
                type="default" 
                size="small" 
                @click="handleStop(record)" 
                v-if="['Pending', 'Running'].includes(record.status)"
                class="stop-btn"
              >
                <PauseCircleOutlined />
                停止
              </a-button>
              <a-button 
                type="default" 
                size="small" 
                @click="handleRestart(record)" 
                v-if="['Failed', 'Terminated'].includes(record.status)"
                class="restart-btn"
              >
                <PlayCircleOutlined />
                重启
              </a-button>
              <a-popconfirm
                title="确定要删除这个作业吗？"
                @confirm="handleDelete(record)"
                v-if="['Completed', 'Failed', 'Terminated'].includes(record.status)"
              >
                <a-button type="default" size="small" danger class="delete-btn">
                  <DeleteOutlined />
                  删除
                </a-button>
              </a-popconfirm>
            </div>
          </template>
        </a-table>
      </div>
    </transition>

    <!-- 创建作业模态框 -->
    <a-modal 
      title="创建训练作业" 
      v-model:open="isAddModalVisible" 
      @ok="handleAdd" 
      @cancel="closeAddModal"
      :width="900"
      class="custom-modal"
      :confirm-loading="addFormLoading"
    >
      <a-form ref="addFormRef" :model="addForm" layout="vertical" class="custom-form">
        <div class="form-section">
          <div class="section-title">
            <SettingOutlined class="section-icon" />
            基本信息
          </div>
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
          <a-row :gutter="16">
            <a-col :span="24">
              <a-form-item label="作业描述" name="description">
                <a-textarea v-model:value="addForm.description" placeholder="请输入作业描述" :rows="2" />
              </a-form-item>
            </a-col>
          </a-row>
        </div>

        <div class="form-section">
          <div class="section-title">
            <LockOutlined class="section-icon" />
            容器配置
          </div>
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
          <a-row :gutter="16">
            <a-col :span="12">
              <a-form-item label="工作目录" name="workingDir">
                <a-input v-model:value="addForm.workingDir" placeholder="/workspace" />
              </a-form-item>
            </a-col>
            <a-col :span="12">
              <a-form-item label="重启策略" name="restartPolicy">
                <a-select v-model:value="addForm.restartPolicy" placeholder="选择重启策略">
                  <a-select-option value="Never">Never</a-select-option>
                  <a-select-option value="OnFailure">OnFailure</a-select-option>
                  <a-select-option value="Always">Always</a-select-option>
                </a-select>
              </a-form-item>
            </a-col>
          </a-row>
        </div>

        <div class="form-section">
          <div class="section-title">
            <DatabaseOutlined class="section-icon" />
            资源配置
          </div>
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
          <a-row :gutter="16">
            <a-col :span="8">
              <a-form-item label="CPU限制" name="cpu_limit">
                <a-input v-model:value="addForm.cpu_limit" placeholder="例如: 4" />
              </a-form-item>
            </a-col>
            <a-col :span="8">
              <a-form-item label="内存限制" name="memory_limit">
                <a-input v-model:value="addForm.memory_limit" placeholder="例如: 8Gi" />
              </a-form-item>
            </a-col>
            <a-col :span="8">
              <a-form-item label="超时时间(小时)" name="timeout">
                <a-input-number v-model:value="addForm.timeout" :min="1" :max="168" placeholder="超时时间" class="full-width" />
              </a-form-item>
            </a-col>
          </a-row>
        </div>

        <div class="form-section">
          <div class="section-title">
            <EnvironmentOutlined class="section-icon" />
            环境变量
          </div>
          <a-form-item v-for="(env, index) in addForm.env_vars" :key="env.key"
            :label="index === 0 ? '环境变量' : ''" :name="['env_vars', index, 'value']">
            <div class="env-input-group">
              <a-input v-model:value="env.envKey" placeholder="变量名" class="env-key-input" />
              <div class="env-separator">=</div>
              <a-input v-model:value="env.envValue" placeholder="变量值" class="env-value-input" />
              <MinusCircleOutlined 
                v-if="addForm.env_vars.length > 1" 
                class="dynamic-delete-button"
                @click="removeEnvVar(env)" 
              />
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
          <div class="section-title">
            <FolderOutlined class="section-icon" />
            存储配置
          </div>
          <a-form-item v-for="(volume, index) in addForm.volumes" :key="volume.key"
            :label="index === 0 ? '存储卷' : ''" :name="['volumes', index, 'value']">
            <div class="volume-input-group">
              <a-input v-model:value="volume.hostPath" placeholder="主机路径" class="volume-host-input" />
              <div class="volume-separator">:</div>
              <a-input v-model:value="volume.containerPath" placeholder="容器路径" class="volume-container-input" />
              <a-select v-model:value="volume.mode" placeholder="模式" class="volume-mode-input">
                <a-select-option value="rw">读写</a-select-option>
                <a-select-option value="ro">只读</a-select-option>
              </a-select>
              <MinusCircleOutlined 
                v-if="addForm.volumes.length > 1" 
                class="dynamic-delete-button"
                @click="removeVolume(volume)" 
              />
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
          <div class="section-title">
            <NodeIndexOutlined class="section-icon" />
            高级配置
          </div>
          <a-row :gutter="16">
            <a-col :span="12">
              <a-form-item label="节点选择器" name="nodeSelector">
                <a-input v-model:value="addForm.nodeSelector" placeholder="例如: gpu=true" />
              </a-form-item>
            </a-col>
            <a-col :span="12">
              <a-form-item label="容忍度" name="tolerations">
                <a-input v-model:value="addForm.tolerations" placeholder="例如: key=value:NoSchedule" />
              </a-form-item>
            </a-col>
          </a-row>
          <a-row :gutter="16">
            <a-col :span="24">
              <a-form-item label="标签" name="labels">
                <a-input v-model:value="addForm.labels" placeholder="例如: app=training,version=v1.0" />
              </a-form-item>
            </a-col>
          </a-row>
        </div>
      </a-form>
    </a-modal>

    <!-- 编辑作业模态框 -->
    <a-modal 
      title="编辑训练作业" 
      v-model:open="isEditModalVisible" 
      @ok="handleUpdate" 
      @cancel="closeEditModal"
      :width="900"
      class="custom-modal"
      :confirm-loading="editFormLoading"
    >
      <!-- 编辑表单内容与创建表单类似，这里简化处理 -->
      <a-form ref="editFormRef" :model="editForm" layout="vertical" class="custom-form">
        <!-- 基本信息 -->
        <div class="form-section">
          <div class="section-title">
            <SettingOutlined class="section-icon" />
            基本信息
          </div>
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
          <!-- 其他字段... -->
        </div>
        <!-- 其他配置段... -->
      </a-form>
    </a-modal>

    <!-- 作业详情模态框 -->
    <a-modal 
      title="作业详情" 
      v-model:open="isViewModalVisible" 
      @cancel="closeViewModal"
      :width="1000"
      class="custom-modal view-modal"
      :footer="null"
    >
      <div class="job-detail-container" v-if="viewJob">
        <a-tabs v-model:activeKey="activeDetailTab" type="card">
          <a-tab-pane key="basic" tab="基本信息">
            <div class="detail-section">
              <a-descriptions :column="2" size="small" bordered>
                <a-descriptions-item label="作业名称">
                  <a-tag color="blue">{{ viewJob.name }}</a-tag>
                </a-descriptions-item>
                <a-descriptions-item label="命名空间">{{ viewJob.namespace }}</a-descriptions-item>
                <a-descriptions-item label="队列名称">
                  <a-tag color="green">{{ viewJob.queue }}</a-tag>
                </a-descriptions-item>
                <a-descriptions-item label="状态">
                  <a-tag :color="getStatusColor(viewJob.status)">
                    <component :is="getStatusIcon(viewJob.status)" />
                    {{ getStatusText(viewJob.status) }}
                  </a-tag>
                </a-descriptions-item>
                <a-descriptions-item label="优先级">
                  <a-tag :color="getPriorityColor(viewJob.priority)">
                    {{ getPriorityText(viewJob.priority) }}
                  </a-tag>
                </a-descriptions-item>
                <a-descriptions-item label="任务数量">{{ viewJob.task_count }}</a-descriptions-item>
                <a-descriptions-item label="创建时间">{{ viewJob.created_at }}</a-descriptions-item>
                <a-descriptions-item label="开始时间">{{ viewJob.start_time || '未开始' }}</a-descriptions-item>
                <a-descriptions-item label="完成时间">{{ viewJob.completion_time || '未完成' }}</a-descriptions-item>
                <a-descriptions-item label="运行时长">
                  {{ formatDuration(viewJob.start_time, viewJob.completion_time) }}
                </a-descriptions-item>
                <a-descriptions-item label="创建者">
                  <a-avatar size="small">{{ viewJob.creator }}</a-avatar>
                  {{ viewJob.creator }}
                </a-descriptions-item>
                <a-descriptions-item label="进度">
                  <a-progress :percent="getJobProgress(viewJob)" size="small" />
                </a-descriptions-item>
              </a-descriptions>
            </div>
          </a-tab-pane>

          <a-tab-pane key="resource" tab="资源配置">
            <div class="detail-section">
              <div class="resource-detail-grid">
                <div class="resource-detail-card">
                  <div class="resource-detail-header">
                    <CpuIcon class="resource-detail-icon cpu" />
                    <span>CPU</span>
                  </div>
                  <div class="resource-detail-content">
                    <div class="resource-detail-item">
                      <span class="resource-detail-label">请求:</span>
                      <span class="resource-detail-value">{{ viewJob.cpu_request }}</span>
                    </div>
                    <div class="resource-detail-item">
                      <span class="resource-detail-label">限制:</span>
                      <span class="resource-detail-value">{{ viewJob.cpu_limit || '无限制' }}</span>
                    </div>
                  </div>
                </div>
                <div class="resource-detail-card">
                  <div class="resource-detail-header">
                    <MemoryIcon class="resource-detail-icon memory" />
                    <span>内存</span>
                  </div>
                  <div class="resource-detail-content">
                    <div class="resource-detail-item">
                      <span class="resource-detail-label">请求:</span>
                      <span class="resource-detail-value">{{ viewJob.memory_request }}</span>
                    </div>
                    <div class="resource-detail-item">
                      <span class="resource-detail-label">限制:</span>
                      <span class="resource-detail-value">{{ viewJob.memory_limit || '无限制' }}</span>
                    </div>
                  </div>
                </div>
                <div class="resource-detail-card">
                  <div class="resource-detail-header">
                    <GpuIcon class="resource-detail-icon gpu" />
                    <span>GPU</span>
                  </div>
                  <div class="resource-detail-content">
                    <div class="resource-detail-item">
                      <span class="resource-detail-label">数量:</span>
                      <span class="resource-detail-value">{{ viewJob.gpu_request }}</span>
                    </div>
                    <div class="resource-detail-item">
                      <span class="resource-detail-label">类型:</span>
                      <span class="resource-detail-value">NVIDIA GPU</span>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </a-tab-pane>

          <a-tab-pane key="container" tab="容器配置">
            <div class="detail-section">
              <a-descriptions :column="1" size="small" bordered>
                <a-descriptions-item label="镜像">
                  <div class="image-detail">
                    <LockOutlined class="image-detail-icon" />
                    <code>{{ viewJob.image }}</code>
                  </div>
                </a-descriptions-item>
                <a-descriptions-item label="启动命令">
                  <pre class="command-pre">{{ viewJob.command || '默认命令' }}</pre>
                </a-descriptions-item>
                <a-descriptions-item label="工作目录">
                  <code>{{ viewJob.workingDir || '/workspace' }}</code>
                </a-descriptions-item>
                <a-descriptions-item label="重启策略">
                  <a-tag>{{ viewJob.restartPolicy || 'Never' }}</a-tag>
                </a-descriptions-item>
              </a-descriptions>
            </div>
          </a-tab-pane>

          <a-tab-pane key="env" tab="环境配置">
            <div class="detail-section">
              <div class="config-section" v-if="viewJob.env_vars && viewJob.env_vars.length > 0">
                <h4>环境变量</h4>
                <div class="env-list">
                  <div class="env-item" v-for="env in viewJob.env_vars" :key="env">
                    <EnvironmentOutlined class="env-icon" />
                    <span class="env-key">{{ env.split('=')[0] }}</span>
                    <span class="env-separator">=</span>
                    <span class="env-value">{{ env.split('=')[1] }}</span>
                  </div>
                </div>
              </div>

              <div class="config-section" v-if="viewJob.volumes && viewJob.volumes.length > 0">
                <h4>存储卷</h4>
                <div class="volume-list">
                  <div class="volume-item" v-for="volume in viewJob.volumes" :key="volume">
                    <FolderOutlined class="volume-icon" />
                    <span class="volume-host">{{ volume.split(':')[0] }}</span>
                    <ArrowRightOutlined class="volume-arrow" />
                    <span class="volume-container">{{ volume.split(':')[1] }}</span>
                    <a-tag size="small" color="blue">{{ volume.split(':')[2] || 'rw' }}</a-tag>
                  </div>
                </div>
              </div>
            </div>
          </a-tab-pane>

          <a-tab-pane key="logs" tab="日志信息">
            <div class="detail-section">
              <div class="log-container">
                <div class="log-header">
                  <span>作业日志</span>
                  <a-button size="small" @click="refreshLogs">
                    <ReloadOutlined />
                    刷新
                  </a-button>
                </div>
                <div class="log-content">
                  <pre class="log-pre">{{ mockLogs }}</pre>
                </div>
              </div>
            </div>
          </a-tab-pane>
        </a-tabs>
      </div>
    </a-modal>

    <!-- 批量操作确认模态框 -->
    <a-modal
      title="批量删除确认"
      v-model:open="isBatchDeleteModalVisible"
      @ok="confirmBatchDelete"
      @cancel="cancelBatchDelete"
      :confirm-loading="batchDeleteLoading"
    >
      <p>确定要删除选中的 {{ selectedRowKeys.length }} 个作业吗？此操作不可恢复。</p>
      <a-list
        size="small"
        :data-source="selectedJobs"
        class="selected-jobs-list"
      >
        <template #renderItem="{ item }">
          <a-list-item>
            <a-tag :color="getStatusColor(item.status)">{{ item.name }}</a-tag>
          </a-list-item>
        </template>
      </a-list>
    </a-modal>
  </div>
</template>

<script lang="ts" setup>
import { ref, reactive, onMounted, computed, watch, nextTick } from 'vue';
import { message, Modal } from 'ant-design-vue';
import {
  SearchOutlined,
  ReloadOutlined,
  PlusOutlined,
  MinusCircleOutlined,
  EyeOutlined,
  EditOutlined,
  DeleteOutlined,
  PauseCircleOutlined,
  PlayCircleOutlined,
  SettingOutlined,
  LockOutlined,
  DatabaseOutlined,
  EnvironmentOutlined,
  FolderOutlined,
  NodeIndexOutlined,
  ClockCircleOutlined,
  ArrowRightOutlined,
  CheckCircleOutlined,
  ExclamationCircleOutlined,
  SyncOutlined,
  StopOutlined,
  FireOutlined,
  TrophyOutlined,
  AlertOutlined
} from '@ant-design/icons-vue';
import type { FormInstance, TableColumnsType } from 'ant-design-vue';

// 自定义图标组件
const CpuIcon = () => '🖥️';
const MemoryIcon = () => '💾';
const GpuIcon = () => '🎮';

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
  cpu_limit?: string;
  memory_limit?: string;
  env_vars: string[];
  volumes: string[];
  created_at: string;
  start_time?: string;
  completion_time?: string;
  creator: string;
  description?: string;
  workingDir?: string;
  restartPolicy?: string;
  timeout?: number;
  nodeSelector?: string;
  tolerations?: string;
  labels?: string;
  progress?: number;
}

interface EnvVar {
  envKey: string;
  envValue: string;
  key: number;
}

interface Volume {
  hostPath: string;
  containerPath: string;
  mode: string;
  key: number;
}

interface StatItem {
  label: string;
  value: number;
  color: string;
  icon: any;
}

// 响应式数据
const loading = ref(false);
const addFormLoading = ref(false);
const editFormLoading = ref(false);
const batchDeleteLoading = ref(false);

// 搜索和筛选
const searchText = ref('');
const statusFilter = ref('');
const queueFilter = ref('');

// 表格数据
const data = ref<JobItem[]>([]);
const selectedRowKeys = ref<number[]>([]);

// 分页配置
const paginationConfig = reactive({
  current: 1,
  pageSize: 10,
  total: 0,
  showSizeChanger: true,
  showQuickJumper: true,
  showTotal: (total: number, range: [number, number]) => 
    `第 ${range[0]}-${range[1]} 条，共 ${total} 条`,
  pageSizeOptions: ['10', '20', '50', '100']
});

// 模态框状态
const isAddModalVisible = ref(false);
const isEditModalVisible = ref(false);
const isViewModalVisible = ref(false);
const isBatchDeleteModalVisible = ref(false);
const activeDetailTab = ref('basic');

// 表单引用
const addFormRef = ref<FormInstance>();
const editFormRef = ref<FormInstance>();

// 查看详情的作业
const viewJob = ref<JobItem | null>(null);

// 环境变量和存储卷计数器
let envKeyCounter = 0;
let volumeKeyCounter = 0;

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
  cpu_limit: '',
  memory_limit: '',
  description: '',
  workingDir: '/workspace',
  restartPolicy: 'Never',
  timeout: 24,
  nodeSelector: '',
  tolerations: '',
  labels: '',
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
  cpu_limit: '',
  memory_limit: '',
  description: '',
  workingDir: '/workspace',
  restartPolicy: 'Never',
  timeout: 24,
  nodeSelector: '',
  tolerations: '',
  labels: '',
  env_vars: [] as EnvVar[],
  volumes: [] as Volume[]
});

// 统计数据
const stats = ref<StatItem[]>([
  { label: '总作业数', value: 0, color: '#1890ff', icon: DatabaseOutlined },
  { label: '运行中', value: 0, color: '#52c41a', icon: SyncOutlined },
  { label: '等待中', value: 0, color: '#faad14', icon: ClockCircleOutlined },
  { label: '已完成', value: 0, color: '#13c2c2', icon: CheckCircleOutlined },
  { label: '失败', value: 0, color: '#ff4d4f', icon: ExclamationCircleOutlined }
]);

// 模拟日志数据
const mockLogs = ref(`
[2024-06-11 10:30:00] Starting job...
[2024-06-11 10:30:05] Initializing environment...
[2024-06-11 10:30:10] Loading dataset...
[2024-06-11 10:30:15] Starting training process...
[2024-06-11 10:30:20] Epoch 1/100 - Loss: 0.8456, Accuracy: 0.7234
[2024-06-11 10:30:25] Epoch 2/100 - Loss: 0.7891, Accuracy: 0.7456
[2024-06-11 10:30:30] Epoch 3/100 - Loss: 0.7234, Accuracy: 0.7678
[2024-06-11 10:30:35] Saving checkpoint...
[2024-06-11 10:30:40] Training in progress...
`);

// 表格列配置
const columns: TableColumnsType<JobItem> = [
  {
    title: 'ID',
    dataIndex: 'id',
    key: 'id',
    width: 80,
    sorter: (a, b) => a.id - b.id,
  },
  {
    title: '作业名称',
    dataIndex: 'name',
    key: 'name',
    width: 180,
    ellipsis: true,
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
    width: 140,
    filters: [
      { text: 'default', value: 'default' },
      { text: 'high-priority', value: 'high-priority' },
      { text: 'low-priority', value: 'low-priority' },
    ],
  },
  {
    title: '状态',
    dataIndex: 'status',
    key: 'status',
    slots: { customRender: 'status' },
    width: 120,
    filters: [
      { text: '等待中', value: 'Pending' },
      { text: '运行中', value: 'Running' },
      { text: '已完成', value: 'Completed' },
      { text: '失败', value: 'Failed' },
      { text: '已终止', value: 'Terminated' },
    ],
  },
  {
    title: '优先级',
    dataIndex: 'priority',
    key: 'priority',
    slots: { customRender: 'priority' },
    width: 100,
    sorter: (a, b) => a.priority - b.priority,
  },
  {
    title: '任务数',
    dataIndex: 'task_count',
    key: 'task_count',
    width: 80,
    sorter: (a, b) => a.task_count - b.task_count,
  },
  {
    title: '进度',
    key: 'progress',
    slots: { customRender: 'progress' },
    width: 120,
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
    ellipsis: true,
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
    sorter: (a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime(),
  },
  {
    title: '操作',
    key: 'action',
    slots: { customRender: 'action' },
    width: 280,
    fixed: 'right',
  },
];

// 行选择配置
const rowSelection = {
  selectedRowKeys: selectedRowKeys,
  onChange: (keys: number[]) => {
    selectedRowKeys.value = keys;
  },
  getCheckboxProps: (record: JobItem) => ({
    disabled: !['Completed', 'Failed', 'Terminated'].includes(record.status),
  }),
};

// 计算属性
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
  
  if (queueFilter.value) {
    result = result.filter(item => item.queue === queueFilter.value);
  }
  
  return result;
});

const selectedJobs = computed(() => {
  return data.value.filter(job => selectedRowKeys.value.includes(job.id));
});

// 监听数据变化更新统计
watch(data, (newData) => {
  updateStats(newData);
}, { deep: true });

// 初始化
onMounted(() => {
  initForms();
  loadData();
});

// 更新统计数据
const updateStats = (jobData: JobItem[]) => {
  const statusCount = jobData.reduce((acc, job) => {
    acc[job.status] = (acc[job.status] || 0) + 1;
    return acc;
  }, {} as Record<string, number>);

  stats.value = [
    { label: '总作业数', value: jobData.length, color: '#1890ff', icon: DatabaseOutlined },
    { label: '运行中', value: statusCount['Running'] || 0, color: '#52c41a', icon: SyncOutlined },
    { label: '等待中', value: statusCount['Pending'] || 0, color: '#faad14', icon: ClockCircleOutlined },
    { label: '已完成', value: statusCount['Completed'] || 0, color: '#13c2c2', icon: CheckCircleOutlined },
    { label: '失败', value: statusCount['Failed'] || 0, color: '#ff4d4f', icon: ExclamationCircleOutlined }
  ];
};

// 初始化表单
const initForms = () => {
  addForm.env_vars = [{ envKey: '', envValue: '', key: ++envKeyCounter }];
  addForm.volumes = [{ hostPath: '', containerPath: '', mode: 'rw', key: ++volumeKeyCounter }];
};

// 获取状态相关函数
const getStatusColor = (status: string) => {
  const colorMap: Record<string, string> = {
    'Pending': '#faad14',
    'Running': '#52c41a',
    'Completed': '#13c2c2',
    'Failed': '#ff4d4f',
    'Terminated': '#8c8c8c'
  };
  return colorMap[status] || 'default';
};

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

const getStatusIcon = (status: string) => {
  const iconMap: Record<string, any> = {
    'Pending': ClockCircleOutlined,
    'Running': SyncOutlined,
    'Completed': CheckCircleOutlined,
    'Failed': ExclamationCircleOutlined,
    'Terminated': StopOutlined
  };
  return iconMap[status] || ClockCircleOutlined;
};

const getPriorityColor = (priority: number) => {
  if (priority >= 10) return '#ff4d4f';
  if (priority >= 5) return '#faad14';
  return '#52c41a';
};

const getPriorityText = (priority: number) => {
  if (priority >= 10) return '高';
  if (priority >= 5) return '中';
  return '低';
};

const getPriorityIcon = (priority: number) => {
  if (priority >= 10) return FireOutlined;
  if (priority >= 5) return TrophyOutlined;
  return AlertOutlined;
};

const getJobProgress = (job: JobItem): number => {
  if (job.status === 'Completed') return 100;
  if (job.status === 'Failed' || job.status === 'Terminated') return 0;
  if (job.status === 'Running') {
    // 模拟进度计算
    const now = new Date().getTime();
    const start = job.start_time ? new Date(job.start_time).getTime() : now;
    const elapsed = now - start;
    const estimated = elapsed * 2; // 假设总时长是已用时长的2倍
    return Math.min(Math.floor((elapsed / estimated) * 100), 95);
  }
  return 0;
};

const getProgressStatus = (job: JobItem) => {
  if (job.status === 'Completed') return 'success';
  if (job.status === 'Failed') return 'exception';
  if (job.status === 'Running') return 'active';
  return 'normal';
};

// 格式化运行时间
const formatDuration = (startTime?: string, completionTime?: string) => {
  if (!startTime) return '未开始';
  
  const start = new Date(startTime);
  const end = completionTime ? new Date(completionTime) : new Date();
  const duration = Math.floor((end.getTime() - start.getTime()) / 1000);
  
  const days = Math.floor(duration / 86400);
  const hours = Math.floor((duration % 86400) / 3600);
  const minutes = Math.floor((duration % 3600) / 60);
  const seconds = duration % 60;
  
  if (days > 0) {
    return `${days}d ${hours}h ${minutes}m`;
  } else if (hours > 0) {
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
    // 模拟API调用延迟
    await new Promise(resolve => setTimeout(resolve, 800));
    
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
        command: 'python train.py --epochs 100 --batch-size 32 --learning-rate 0.001',
        cpu_request: '4',
        memory_request: '8Gi',
        gpu_request: 2,
        cpu_limit: '8',
        memory_limit: '16Gi',
        env_vars: ['CUDA_VISIBLE_DEVICES=0,1', 'PYTHONPATH=/workspace', 'NCCL_DEBUG=INFO'],
        volumes: ['/data:/workspace/data:rw', '/models:/workspace/models:rw', '/logs:/workspace/logs:rw'],
        created_at: '2024-06-09 10:30:00',
        start_time: '2024-06-09 10:32:00',
        creator: 'admin',
        description: 'PyTorch 模型训练作业，使用 ResNet50 架构',
        workingDir: '/workspace',
        restartPolicy: 'OnFailure',
        timeout: 48,
        nodeSelector: 'gpu=true',
        labels: 'app=training,version=v1.0',
        progress: 65
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
        command: 'python main.py --dataset imagenet --model resnet50 --distributed',
        cpu_request: '8',
        memory_request: '16Gi',
        gpu_request: 4,
        cpu_limit: '16',
        memory_limit: '32Gi',
        env_vars: ['TF_CPP_MIN_LOG_LEVEL=2', 'CUDA_VISIBLE_DEVICES=0,1,2,3'],
        volumes: ['/datasets:/data:ro', '/checkpoints:/workspace/checkpoints:rw'],
        created_at: '2024-06-09 11:15:00',
        creator: 'user1',
        description: 'TensorFlow 分布式训练作业',
        workingDir: '/workspace',
        restartPolicy: 'Never',
        timeout: 72,
        nodeSelector: 'node-type=gpu-node',
        labels: 'team=ml,priority=high',
        progress: 0
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
        command: 'python finetune_bert.py --model bert-base-uncased --task sentiment',
        cpu_request: '2',
        memory_request: '4Gi',
        gpu_request: 1,
        cpu_limit: '4',
        memory_limit: '8Gi',
        env_vars: ['TRANSFORMERS_CACHE=/workspace/cache', 'HF_DATASETS_CACHE=/workspace/datasets'],
        volumes: ['/nlp-data:/workspace/data:ro', '/models:/workspace/models:rw'],
        created_at: '2024-06-09 09:00:00',
        start_time: '2024-06-09 09:05:00',
        completion_time: '2024-06-09 10:30:00',
        creator: 'user2',
        description: 'BERT 模型微调作业',
        workingDir: '/workspace',
        restartPolicy: 'Never',
        timeout: 24,
        labels: 'team=nlp,model=bert',
        progress: 100
      },
      {
        id: 4,
        name: 'yolo-object-detection-004',
        namespace: 'cv-team',
        queue: 'low-priority',
        status: 'Failed',
        priority: 2,
        task_count: 1,
        image: 'ultralytics/yolov8:latest',
        command: 'python train.py --data coco.yaml --epochs 300',
        cpu_request: '4',
        memory_request: '8Gi',
        gpu_request: 2,
        env_vars: ['WANDB_PROJECT=yolo-training'],
        volumes: ['/datasets/coco:/workspace/datasets:ro'],
        created_at: '2024-06-09 08:00:00',
        start_time: '2024-06-09 08:05:00',
        completion_time: '2024-06-09 09:15:00',
        creator: 'user3',
        description: 'YOLO 目标检测模型训练',
        progress: 0
      },
      {
        id: 5,
        name: 'llama-inference-job-005',
        namespace: 'inference',
        queue: 'high-priority',
        status: 'Terminated',
        priority: 8,
        task_count: 1,
        image: 'meta/llama2:latest',
        command: 'python inference.py --model llama2-7b --batch-size 1',
        cpu_request: '8',
        memory_request: '32Gi',
        gpu_request: 1,
        env_vars: ['MODEL_PATH=/models/llama2', 'MAX_LENGTH=2048'],
        volumes: ['/models:/models:ro', '/outputs:/outputs:rw'],
        created_at: '2024-06-09 14:00:00',
        start_time: '2024-06-09 14:05:00',
        completion_time: '2024-06-09 14:30:00',
        creator: 'admin',
        description: 'LLaMA 模型推理服务',
        progress: 0
      }
    ];
    
    data.value = mockData;
    paginationConfig.total = mockData.length;
    updateStats(mockData);
  } catch (error) {
    message.error('加载数据失败');
  } finally {
    loading.value = false;
  }
};

// 搜索处理
const handleSearch = async () => {
  loading.value = true;
  try {
    await new Promise(resolve => setTimeout(resolve, 500));
    message.success('搜索完成');
  } finally {
    loading.value = false;
  }
};

// 重置处理
const handleReset = () => {
  searchText.value = '';
  statusFilter.value = '';
  queueFilter.value = '';
  message.success('重置成功');
};

// 刷新数据
const handleRefresh = () => {
  loadData();
  message.success('数据已刷新');
};

// 表格变化处理
const handleTableChange = (pagination: any, filters: any, sorter: any) => {
  paginationConfig.current = pagination.current;
  paginationConfig.pageSize = pagination.pageSize;
  // 这里可以根据filters和sorter进行数据筛选和排序
  loadData();
};

// 批量删除相关
const handleBatchDelete = () => {
  if (selectedRowKeys.value.length === 0) {
    message.warning('请选择要删除的作业');
    return;
  }
  isBatchDeleteModalVisible.value = true;
};

const confirmBatchDelete = async () => {
  batchDeleteLoading.value = true;
  try {
    await new Promise(resolve => setTimeout(resolve, 1000));
    
    // 删除选中的作业
    data.value = data.value.filter(job => !selectedRowKeys.value.includes(job.id));
    selectedRowKeys.value = [];
    paginationConfig.total = data.value.length;
    updateStats(data.value);
    
    message.success('批量删除成功');
    isBatchDeleteModalVisible.value = false;
  } catch (error) {
    message.error('批量删除失败');
  } finally {
    batchDeleteLoading.value = false;
  }
};

const cancelBatchDelete = () => {
  isBatchDeleteModalVisible.value = false;
};

// 模态框操作
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
    gpu_request: 1,
    cpu_limit: '',
    memory_limit: '',
    description: '',
    workingDir: '/workspace',
    restartPolicy: 'Never',
    timeout: 24,
    nodeSelector: '',
    tolerations: '',
    labels: '',
    env_vars: [{ envKey: '', envValue: '', key: ++envKeyCounter }],
    volumes: [{ hostPath: '', containerPath: '', mode: 'rw', key: ++volumeKeyCounter }]
  });
  addFormRef.value?.resetFields();
};

// 新增作业
const handleAdd = async () => {
  try {
    await addFormRef.value?.validateFields();
    addFormLoading.value = true;
    
    // 模拟API调用
    await new Promise(resolve => setTimeout(resolve, 1000));
    
    // 处理环境变量和存储卷数据
    const envVars = addForm.env_vars
      .filter(env => env.envKey && env.envValue)
      .map(env => `${env.envKey}=${env.envValue}`);
    
    const volumes = addForm.volumes
      .filter(vol => vol.hostPath && vol.containerPath)
      .map(vol => `${vol.hostPath}:${vol.containerPath}:${vol.mode}`);

    const newJob: JobItem = {
      ...addForm,
      env_vars: envVars,
      volumes: volumes,
      id: data.value.length + 1,
      namespace: 'default',
      status: 'Pending',
      created_at: new Date().toLocaleString(),
      creator: 'admin',
      progress: 0
    };

    data.value.unshift(newJob);
    paginationConfig.total++;
    updateStats(data.value);
    
    message.success('作业创建成功');
    closeAddModal();
  } catch (error) {
    console.error('Validation failed:', error);
  } finally {
    addFormLoading.value = false;
  }
};

// 查看详情
const handleView = (record: JobItem) => {
  viewJob.value = record;
  activeDetailTab.value = 'basic';
  isViewModalVisible.value = true;
};

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
    cpu_limit: record.cpu_limit || '',
    memory_limit: record.memory_limit || '',
    description: record.description || '',
    workingDir: record.workingDir || '/workspace',
    restartPolicy: record.restartPolicy || 'Never',
    timeout: record.timeout || 24,
    nodeSelector: record.nodeSelector || '',
    tolerations: record.tolerations || '',
    labels: record.labels || '',
    env_vars: record.env_vars.map(env => {
      const [envKey, envValue] = env.split('=');
      return { envKey, envValue, key: ++envKeyCounter };
    }),
    volumes: record.volumes.map(vol => {
      const parts = vol.split(':');
      return { 
        hostPath: parts[0], 
        containerPath: parts[1], 
        mode: parts[2] || 'rw',
        key: ++volumeKeyCounter 
      };
    })
  });
  
  // 确保至少有一个环境变量和存储卷输入框
  if (editForm.env_vars.length === 0) {
    editForm.env_vars.push({ envKey: '', envValue: '', key: ++envKeyCounter });
  }
  if (editForm.volumes.length === 0) {
    editForm.volumes.push({ hostPath: '', containerPath: '', mode: 'rw', key: ++volumeKeyCounter });
  }
  
  isEditModalVisible.value = true;
};

const closeEditModal = () => {
  isEditModalVisible.value = false;
};

// 更新作业
const handleUpdate = async () => {
  try {
    await editFormRef.value?.validateFields();
    editFormLoading.value = true;
    
    // 模拟API调用
    await new Promise(resolve => setTimeout(resolve, 1000));
    
    // 处理环境变量和存储卷数据
    const envVars = editForm.env_vars
      .filter(env => env.envKey && env.envValue)
      .map(env => `${env.envKey}=${env.envValue}`);
    
    const volumes = editForm.volumes
      .filter(vol => vol.hostPath && vol.containerPath)
      .map(vol => `${vol.hostPath}:${vol.containerPath}:${vol.mode}`);

    // 更新数据
    const index = data.value.findIndex(item => item.id === editForm.id);
    if (index !== -1) {
      Object.assign(data.value[index] as JobItem, {
        ...editForm,
        env_vars: envVars,
        volumes: volumes
      });
    }

    updateStats(data.value);
    message.success('作业更新成功');
    closeEditModal();
  } catch (error) {
    console.error('Validation failed:', error);
  } finally {
    editFormLoading.value = false;
  }
};

// 停止作业
const handleStop = (record: JobItem) => {
  Modal.confirm({
    title: '确认停止作业',
    content: `确定要停止作业 "${record.name}" 吗？`,
    onOk: async () => {
      loading.value = true;
      try {
        await new Promise(resolve => setTimeout(resolve, 500));
        record.status = 'Terminated';
        record.completion_time = new Date().toLocaleString();
        record.progress = 0;
        updateStats(data.value);
        message.success('作业已停止');
      } finally {
        loading.value = false;
      }
    },
  });
};

// 重启作业
const handleRestart = (record: JobItem) => {
  Modal.confirm({
    title: '确认重启作业',
    content: `确定要重启作业 "${record.name}" 吗？`,
    onOk: async () => {
      loading.value = true;
      try {
        await new Promise(resolve => setTimeout(resolve, 500));
        record.status = 'Pending';
        record.start_time = undefined;
        record.completion_time = undefined;
        record.progress = 0;
        updateStats(data.value);
        message.success('作业已重启');
      } finally {
        loading.value = false;
      }
    },
  });
};

// 删除作业
const handleDelete = async (record: JobItem) => {
  loading.value = true;
  try {
    await new Promise(resolve => setTimeout(resolve, 500));
    const index = data.value.findIndex(item => item.id === record.id);
    if (index !== -1) {
      data.value.splice(index, 1);
      paginationConfig.total--;
      updateStats(data.value);
    }
    message.success('作业已删除');
  } finally {
    loading.value = false;
  }
};

// 环境变量操作
const addEnvVar = () => {
  addForm.env_vars.push({
    envKey: '',
    envValue: '',
    key: ++envKeyCounter
  });
};

const removeEnvVar = (item: EnvVar) => {
  const index = addForm.env_vars.indexOf(item);
  if (index !== -1) {
    addForm.env_vars.splice(index, 1);
  }
};

// 存储卷操作
const addVolume = () => {
  addForm.volumes.push({
    hostPath: '',
    containerPath: '',
    mode: 'rw',
    key: ++volumeKeyCounter
  });
};

const removeVolume = (item: Volume) => {
  const index = addForm.volumes.indexOf(item);
  if (index !== -1) {
    addForm.volumes.splice(index, 1);
  }
};

// 刷新日志
const refreshLogs = () => {
  message.success('日志已刷新');
};
</script>

<style scoped>
/* 基础样式 */
.job-queue-page {
  padding: 24px;
  min-height: 100vh;
}

/* 动画效果 */
.slide-down-enter-active,
.slide-down-leave-active {
  transition: all 0.5s ease;
}

.slide-down-enter-from {
  opacity: 0;
  transform: translateY(-30px);
}

.slide-up-enter-active,
.slide-up-leave-active {
  transition: all 0.4s ease;
}

.slide-up-enter-from {
  opacity: 0;
  transform: translateY(30px);
}

.fade-in-enter-active {
  transition: all 0.6s ease;
}

.fade-in-enter-from {
  opacity: 0;
  transform: translateY(20px);
}

.count-up-enter-active {
  transition: all 0.8s ease;
}

.count-up-enter-from {
  opacity: 0;
  transform: scale(0.8);
}

/* 页面头部 */
.page-header {
  margin-bottom: 32px;
}

.page-title {
  font-size: 28px;
  font-weight: 700;
  color: #1a202c;
  margin: 0 0 8px 0;
  background: linear-gradient(45deg, #1890ff, #36cfc9);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.page-description {
  color: #64748b;
  font-size: 16px;
  margin-bottom: 24px;
}

/* 统计卡片 */
.stats-cards {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 20px;
  margin-bottom: 8px;
}

.stat-card {
  background: white;
  border-radius: 12px;
  padding: 20px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.08);
  transition: all 0.3s ease;
  border: 1px solid #f0f0f0;
}

.stat-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 8px 30px rgba(0, 0, 0, 0.12);
}

.stat-content {
  display: flex;
  align-items: center;
  gap: 16px;
}

.stat-icon {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-size: 20px;
}

.stat-info {
  flex: 1;
}

.stat-value {
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

/* 卡片样式 */
.dashboard-card {
  background: white;
  border-radius: 16px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.08);
  margin-bottom: 24px;
  border: 1px solid #f0f0f0;
  overflow: hidden;
}

/* 工具栏 */
.custom-toolbar {
  padding: 24px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 20px;
  background: linear-gradient(135deg, #ffffff 0%, #f8fafc 100%);
}

.search-filters {
  display: flex;
  gap: 16px;
  flex-wrap: wrap;
  align-items: center;
}

.search-input {
  width: 240px;
  border-radius: 8px;
}

.status-filter,
.queue-filter {
  width: 160px;
  border-radius: 8px;
}

.action-button {
  height: 36px;
  border-radius: 8px;
  font-weight: 500;
}

.reset-button {
  background: #f8fafc;
  border-color: #e2e8f0;
  color: #64748b;
}

.reset-button:hover {
  background: #f1f5f9;
  border-color: #cbd5e1;
  color: #475569;
}

.action-buttons {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}

.add-button {
  background: linear-gradient(135deg, #1890ff 0%, #36cfc9 100%);
  border: none;
  height: 36px;
  border-radius: 8px;
  font-weight: 500;
}

.add-button:hover {
  background: linear-gradient(135deg, #40a9ff 0%, #5cdbd3 100%);
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(24, 144, 255, 0.3);
}

/* 表格样式 */
.table-container {
  padding: 0;
  overflow: hidden;
}

.custom-table {
  border-radius: 0 0 16px 16px;
}

.custom-table :deep(.ant-table-thead > tr > th) {
  background: linear-gradient(135deg, #fafbfc 0%, #f1f5f9 100%);
  border-bottom: 2px solid #e2e8f0;
  color: #374151;
  font-weight: 600;
  font-size: 14px;
  padding: 16px 12px;
}

.custom-table :deep(.ant-table-tbody > tr) {
  transition: all 0.2s ease;
}

.custom-table :deep(.ant-table-tbody > tr:hover > td) {
  background: linear-gradient(135deg, #f8fafc 0%, #f1f5f9 100%);
}

.custom-table :deep(.ant-table-tbody > tr > td) {
  padding: 12px;
  border-bottom: 1px solid #f0f0f0;
}

/* 状态标签 */
.status-tag {
  border-radius: 20px;
  font-size: 12px;
  font-weight: 600;
  padding: 4px 12px;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  border: none;
}

.status-icon {
  font-size: 12px;
}

.priority-tag {
  border-radius: 20px;
  font-size: 12px;
  font-weight: 600;
  padding: 4px 12px;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  border: none;
}

.priority-icon {
  font-size: 12px;
}

/* 资源容器 */
.resource-container {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.resource-item {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  padding: 4px 8px;
  background: #f8fafc;
  border-radius: 6px;
}

.resource-icon {
  font-size: 14px;
}

.resource-label {
  color: #64748b;
  font-weight: 600;
  min-width: 30px;
}

.resource-value {
  color: #1a202c;
  font-weight: 600;
  font-family: 'JetBrains Mono', monospace;
}

/* 镜像容器 */
.image-container {
  display: flex;
  align-items: center;
  gap: 6px;
  max-width: 150px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
  color: #4b5563;
  padding: 4px 8px;
  background: #f8fafc;
  border-radius: 6px;
}

.image-icon {
  color: #1890ff;
  font-size: 14px;
}

/* 时长容器 */
.duration-container {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: #4b5563;
  font-family: 'JetBrains Mono', monospace;
  padding: 4px 8px;
  background: #f8fafc;
  border-radius: 6px;
}

.duration-icon {
  color: #52c41a;
  font-size: 14px;
}

/* 进度容器 */
.progress-container {
  display: flex;
  align-items: center;
  gap: 8px;
}

.progress-text {
  font-size: 12px;
  color: #64748b;
  font-weight: 500;
  min-width: 35px;
}

/* 操作列 */
.action-column {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}

.action-column .ant-btn {
  font-size: 12px;
  height: 28px;
  border-radius: 6px;
  font-weight: 500;
}

.view-btn {
  background: #52c41a;
  border-color: #52c41a;
}

.view-btn:hover {
  background: #73d13d;
  border-color: #73d13d;
  transform: translateY(-1px);
}

.edit-btn {
  background: #faad14;
  border-color: #faad14;
  color: white;
}

.edit-btn:hover {
  background: #ffc53d;
  border-color: #ffc53d;
  transform: translateY(-1px);
}

.stop-btn {
  background: #ff7875;
  border-color: #ff7875;
  color: white;
}

.stop-btn:hover {
  background: #ff9c99;
  border-color: #ff9c99;
  transform: translateY(-1px);
}

.restart-btn {
  background: #1890ff;
  border-color: #1890ff;
  color: white;
}

.restart-btn:hover {
  background: #40a9ff;
  border-color: #40a9ff;
  transform: translateY(-1px);
}

.delete-btn:hover {
  transform: translateY(-1px);
}

/* 模态框样式 */
.custom-modal :deep(.ant-modal-header) {
  border-bottom: 2px solid #f0f0f0;
  padding: 20px 24px;
  background: linear-gradient(135deg, #ffffff 0%, #f8fafc 100%);
  border-radius: 16px 16px 0 0;
}

.custom-modal :deep(.ant-modal-title) {
  font-size: 20px;
  font-weight: 700;
  color: #1a202c;
}

.custom-modal :deep(.ant-modal-content) {
  border-radius: 16px;
  overflow: hidden;
}

.custom-modal :deep(.ant-modal-body) {
  padding: 24px;
  max-height: 70vh;
  overflow-y: auto;
}

.view-modal :deep(.ant-modal-body) {
  padding: 0;
}

/* 表单样式 */
.custom-form {
  margin-top: 0;
}

.form-section {
  margin-bottom: 32px;
  border: 1px solid #f0f0f0;
  border-radius: 12px;
  padding: 20px;
  background: #fafbfc;
}

.section-title {
  font-size: 16px;
  font-weight: 600;
  color: #1a202c;
  margin-bottom: 20px;
  padding-bottom: 12px;
  border-bottom: 2px solid #e2e8f0;
  display: flex;
  align-items: center;
  gap: 8px;
}

.section-icon {
  color: #1890ff;
  font-size: 18px;
}

.full-width {
  width: 100%;
}

/* 动态输入组 */
.env-input-group,
.volume-input-group {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.env-key-input,
.volume-host-input {
  flex: 1;
}

.env-separator,
.volume-separator {
  color: #64748b;
  font-weight: 600;
  font-size: 16px;
}

.env-value-input,
.volume-container-input {
  flex: 2;
}

.volume-mode-input {
  width: 80px;
}

.dynamic-delete-button {
  color: #ff4d4f;
  cursor: pointer;
  font-size: 18px;
  transition: all 0.2s ease;
}

.dynamic-delete-button:hover {
  color: #ff7875;
  transform: scale(1.1);
}

.add-dynamic-button {
  border-style: dashed;
  border-color: #d1d5db;
  color: #6b7280;
  border-radius: 8px;
  font-weight: 500;
}

.add-dynamic-button:hover {
  border-color: #1890ff;
  color: #1890ff;
  background: #f0f9ff;
}

/* 作业详情 */
.job-detail-container {
  background: white;
}

.job-detail-container :deep(.ant-tabs-card > .ant-tabs-nav .ant-tabs-tab) {
  border-radius: 8px 8px 0 0;
  border: 1px solid #f0f0f0;
  background: #fafbfc;
}

.job-detail-container :deep(.ant-tabs-card > .ant-tabs-nav .ant-tabs-tab-active) {
  background: white;
  border-bottom-color: white;
}

.detail-section {
  padding: 24px;
}

.detail-section .section-title {
  margin-bottom: 16px;
  font-size: 16px;
  color: #1a202c;
}

/* 资源详情网格 */
.resource-detail-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 20px;
}

.resource-detail-card {
  background: #f8fafc;
  border-radius: 12px;
  padding: 20px;
  border: 1px solid #e2e8f0;
}

.resource-detail-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 16px;
  font-weight: 600;
  color: #1a202c;
}

.resource-detail-icon {
  font-size: 20px;
}

.resource-detail-icon.cpu {
  color: #1890ff;
}

.resource-detail-icon.memory {
  color: #52c41a;
}

.resource-detail-icon.gpu {
  color: #faad14;
}

.resource-detail-content {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.resource-detail-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.resource-detail-label {
  color: #64748b;
  font-size: 14px;
}

.resource-detail-value {
  color: #1a202c;
  font-weight: 600;
  font-family: 'JetBrains Mono', monospace;
}

/* 镜像详情 */
.image-detail {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background: #f0f9ff;
  border-radius: 8px;
  border: 1px solid #bae7ff;
}

.image-detail-icon {
  color: #1890ff;
  font-size: 16px;
}

/* 命令预览 */
.command-pre {
  background: #1f2937;
  color: #f9fafb;
  border: 1px solid #374151;
  border-radius: 8px;
  padding: 16px;
  font-size: 13px;
  margin: 0;
  white-space: pre-wrap;
  word-break: break-all;
  font-family: 'JetBrains Mono', monospace;
  line-height: 1.5;
}

/* 配置段 */
.config-section {
  margin-bottom: 24px;
}

.config-section h4 {
  margin-bottom: 16px;
  color: #1a202c;
  font-weight: 600;
  display: flex;
  align-items: center;
  gap: 8px;
}

/* 环境变量列表 */
.env-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.env-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  background: #f0f9ff;
  border-radius: 8px;
  font-size: 13px;
  font-family: 'JetBrains Mono', monospace;
  border: 1px solid #bae7ff;
}

.env-icon {
  color: #1890ff;
  font-size: 14px;
}

.env-key {
  color: #1890ff;
  font-weight: 600;
}

.env-separator {
  color: #64748b;
  font-weight: 600;
}

.env-value {
  color: #059669;
  font-weight: 500;
}

/* 存储卷列表 */
.volume-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.volume-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  background: #f6ffed;
  border-radius: 8px;
  font-size: 13px;
  font-family: 'JetBrains Mono', monospace;
  border: 1px solid #b7eb8f;
}

.volume-icon {
  color: #52c41a;
  font-size: 14px;
}

.volume-host {
  color: #1890ff;
  font-weight: 600;
}

.volume-arrow {
  color: #64748b;
  font-size: 12px;
}

.volume-container {
  color: #059669;
  font-weight: 500;
}

/* 日志容器 */
.log-container {
  background: #1f2937;
  border-radius: 12px;
  overflow: hidden;
  margin: 24px;
}

.log-header {
  background: #374151;
  padding: 12px 16px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  color: white;
  font-weight: 600;
}

.log-content {
  height: 400px;
  overflow-y: auto;
}

.log-pre {
  background: #1f2937;
  color: #f9fafb;
  padding: 16px;
  margin: 0;
  white-space: pre-wrap;
  word-break: break-all;
  font-family: 'JetBrains Mono', monospace;
  font-size: 12px;
  line-height: 1.5;
  border: none;
}

/* 批量删除列表 */
.selected-jobs-list {
  max-height: 200px;
  overflow-y: auto;
  border: 1px solid #f0f0f0;
  border-radius: 8px;
  padding: 8px;
}

/* 响应式 */
@media (max-width: 768px) {
  .job-queue-page {
    padding: 16px;
  }
  
  .custom-toolbar {
    flex-direction: column;
    align-items: stretch;
    gap: 16px;
  }
  
  .search-filters {
    justify-content: stretch;
    flex-direction: column;
  }
  
  .search-input,
  .status-filter,
  .queue-filter {
    width: 100%;
  }
  
  .action-buttons {
    justify-content: center;
  }
  
  .stats-cards {
    grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
    gap: 12px;
  }
  
  .stat-card {
    padding: 16px;
  }
  
  .stat-content {
    gap: 12px;
  }
  
  .stat-icon {
    width: 40px;
    height: 40px;
    font-size: 18px;
  }
  
  .stat-value {
    font-size: 20px;
  }
  
  .resource-detail-grid {
    grid-template-columns: 1fr;
    gap: 16px;
  }
  
  .env-input-group,
  .volume-input-group {
    flex-direction: column;
    align-items: stretch;
    gap: 8px;
  }
  
  .env-separator,
  .volume-separator {
    align-self: center;
  }
}

/* 滚动条样式 */
::-webkit-scrollbar {
  width: 6px;
  height: 6px;
}

::-webkit-scrollbar-track {
  background: #f1f1f1;
  border-radius: 3px;
}

::-webkit-scrollbar-thumb {
  background: #c1c1c1;
  border-radius: 3px;
}

::-webkit-scrollbar-thumb:hover {
  background: #a8a8a8;
}
</style>