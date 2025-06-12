<template>
  <div class="notebook-service-page">
    <!-- 页面标题区域 -->
    <div class="page-header">
      <h2 class="page-title">Notebook服务管理</h2>
      <p class="page-description">管理和监控您的Jupyter、VS Code和RStudio开发环境</p>
    </div>

    <!-- 统计卡片 -->
    <div class="stats-container">
      <div class="stat-card" v-for="stat in statistics" :key="stat.title">
        <div class="stat-icon" :style="{ background: stat.color + '20', color: stat.color }">
          <component :is="stat.icon" />
        </div>
        <div class="stat-content">
          <div class="stat-value">{{ stat.value }}</div>
          <div class="stat-title">{{ stat.title }}</div>
        </div>
      </div>
    </div>

    <!-- 查询和操作工具栏 -->
    <div class="dashboard-card custom-toolbar">
      <div class="search-filters">
        <a-input 
          v-model:value="searchText" 
          placeholder="搜索Notebook名称或创建者" 
          class="search-input"
          allow-clear
        >
          <template #prefix>
            <SearchOutlined class="search-icon" />
          </template>
        </a-input>
        <a-select 
          v-model:value="statusFilter" 
          placeholder="运行状态" 
          class="status-filter"
          allow-clear
        >
          <a-select-option value="">全部状态</a-select-option>
          <a-select-option value="Creating">
            <a-tag color="orange" size="small">创建中</a-tag>
          </a-select-option>
          <a-select-option value="Running">
            <a-tag color="green" size="small">运行中</a-tag>
          </a-select-option>
          <a-select-option value="Stopped">
            <a-tag color="default" size="small">已停止</a-tag>
          </a-select-option>
          <a-select-option value="Failed">
            <a-tag color="red" size="small">失败</a-tag>
          </a-select-option>
        </a-select>
        <a-select 
          v-model:value="typeFilter" 
          placeholder="Notebook类型" 
          class="type-filter"
          allow-clear
        >
          <a-select-option value="">全部类型</a-select-option>
          <a-select-option value="jupyter">
            <a-tag color="blue" size="small">Jupyter</a-tag>
          </a-select-option>
          <a-select-option value="vscode">
            <a-tag color="purple" size="small">VS Code</a-tag>
          </a-select-option>
          <a-select-option value="rstudio">
            <a-tag color="cyan" size="small">RStudio</a-tag>
          </a-select-option>
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
          创建Notebook
        </a-button>
        <a-button class="import-button" @click="showImportModal">
          <template #icon>
            <CloudUploadOutlined />
          </template>
          导入配置
        </a-button>
        <a-button class="export-button" @click="handleExport">
          <template #icon>
            <CloudDownloadOutlined />
          </template>
          导出数据
        </a-button>
      </div>
    </div>

    <!-- Notebook列表表格 -->
    <div class="dashboard-card table-container">
      <a-table 
        :columns="columns" 
        :data-source="filteredData" 
        row-key="id" 
        :pagination="paginationConfig"
        class="custom-table"
        :scroll="{ x: 1600 }"
        :loading="loading"
        @change="handleTableChange"
      >
        <!-- 运行状态列 -->
        <template #status="{ record }">
          <a-tag :color="getStatusColor(record.status)" class="status-tag">
            <template #icon>
              <LoadingOutlined v-if="record.status === 'Creating'" spin />
              <CheckCircleOutlined v-else-if="record.status === 'Running'" />
              <StopOutlined v-else-if="record.status === 'Stopped'" />
              <CloseCircleOutlined v-else-if="record.status === 'Failed'" />
            </template>
            {{ getStatusText(record.status) }}
          </a-tag>
        </template>
        
        <!-- 资源配置列 -->
        <template #resources="{ record }">
          <div class="resource-container">
            <div class="resource-item">
              <div class="resource-icon cpu-icon">C</div>
              <span class="resource-label">CPU:</span>
              <span class="resource-value">{{ record.cpu_limit }}</span>
            </div>
            <div class="resource-item">
              <div class="resource-icon memory-icon">M</div>
              <span class="resource-label">内存:</span>
              <span class="resource-value">{{ record.memory_limit }}</span>
            </div>
            <div class="resource-item">
              <div class="resource-icon gpu-icon">G</div>
              <span class="resource-label">GPU:</span>
              <span class="resource-value">{{ record.gpu_limit || 0 }}</span>
            </div>
          </div>
        </template>
        
        <!-- 镜像列 -->
        <template #image="{ record }">
          <a-tooltip :title="record.image">
            <div class="image-container">
              <div class="image-icon">
                <ContainerOutlined />
              </div>
              <span class="image-text">{{ record.image.split('/').pop() }}</span>
            </div>
          </a-tooltip>
        </template>
        
        <!-- Notebook类型列 -->
        <template #type="{ record }">
          <a-tag :color="getTypeColor(record.type)" class="type-tag">
            <template #icon>
              <CodeOutlined v-if="record.type === 'vscode'" />
              <ExperimentOutlined v-else-if="record.type === 'jupyter'" />
              <BarChartOutlined v-else-if="record.type === 'rstudio'" />
            </template>
            {{ getTypeText(record.type) }}
          </a-tag>
        </template>
        
        <!-- 运行时间列 -->
        <template #duration="{ record }">
          <div class="duration-container">
            <ClockCircleOutlined class="duration-icon" />
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
            <a-button type="text" size="small" @click="copyAccessUrl(record)" class="copy-btn">
              <template #icon>
                <CopyOutlined />
              </template>
            </a-button>
          </div>
          <span v-else class="access-disabled">
            <StopOutlined />
            未启动
          </span>
        </template>
        
        <!-- 操作列 -->
        <template #action="{ record }">
          <div class="action-column">
            <a-tooltip title="查看详情">
              <a-button type="primary" size="small" @click="handleView(record)" class="action-btn">
                <EyeOutlined />
              </a-button>
            </a-tooltip>
            <a-tooltip title="编辑配置" v-if="['Stopped', 'Failed'].includes(record.status)">
              <a-button type="default" size="small" @click="handleEdit(record)" class="action-btn">
                <EditOutlined />
              </a-button>
            </a-tooltip>
            <a-tooltip title="启动服务" v-if="record.status === 'Stopped'">
              <a-button type="default" size="small" @click="handleStart(record)" class="action-btn start-btn">
                <PlayCircleOutlined />
              </a-button>
            </a-tooltip>
            <a-tooltip title="停止服务" v-if="record.status === 'Running'">
              <a-button type="default" size="small" @click="handleStop(record)" class="action-btn stop-btn">
                <PauseCircleOutlined />
              </a-button>
            </a-tooltip>
            <a-tooltip title="重启服务" v-if="record.status === 'Running'">
              <a-button type="default" size="small" @click="handleRestart(record)" class="action-btn restart-btn">
                <ReloadOutlined />
              </a-button>
            </a-tooltip>
            <a-tooltip title="克隆配置">
              <a-button type="default" size="small" @click="handleClone(record)" class="action-btn clone-btn">
                <CopyOutlined />
              </a-button>
            </a-tooltip>
            <a-tooltip title="删除" v-if="['Stopped', 'Failed'].includes(record.status)">
              <a-button type="default" size="small" @click="handleDelete(record)" class="action-btn delete-btn">
                <DeleteOutlined />
              </a-button>
            </a-tooltip>
          </div>
        </template>
      </a-table>
    </div>

    <!-- 创建Notebook模态框 -->
    <a-modal 
      title="创建Notebook服务" 
      v-model:open="isAddModalVisible" 
      @ok="handleAdd" 
      @cancel="closeAddModal"
      :width="900"
      class="custom-modal"
      :confirm-loading="submitLoading"
      ok-text="创建"
      cancel-text="取消"
    >
      <a-form ref="addFormRef" :model="addForm" layout="vertical" class="custom-form" :rules="formRules">
        <div class="form-section">
          <div class="section-title">
            <SettingOutlined />
            基本信息
          </div>
          <a-row :gutter="16">
            <a-col :span="12">
              <a-form-item label="Notebook名称" name="name">
                <a-input v-model:value="addForm.name" placeholder="请输入Notebook名称" />
              </a-form-item>
            </a-col>
            <a-col :span="12">
              <a-form-item label="Notebook类型" name="type">
                <a-select v-model:value="addForm.type" placeholder="请选择Notebook类型">
                  <a-select-option value="jupyter">
                    <ExperimentOutlined /> Jupyter Notebook
                  </a-select-option>
                  <a-select-option value="vscode">
                    <CodeOutlined /> VS Code Server
                  </a-select-option>
                  <a-select-option value="rstudio">
                    <BarChartOutlined /> RStudio Server
                  </a-select-option>
                </a-select>
              </a-form-item>
            </a-col>
          </a-row>
          <a-row :gutter="16">
            <a-col :span="12">
              <a-form-item label="命名空间" name="namespace">
                <a-select v-model:value="addForm.namespace" placeholder="选择命名空间">
                  <a-select-option value="default">default</a-select-option>
                  <a-select-option value="dev-team">dev-team</a-select-option>
                  <a-select-option value="data-team">data-team</a-select-option>
                  <a-select-option value="research">research</a-select-option>
                </a-select>
              </a-form-item>
            </a-col>
            <a-col :span="12">
              <a-form-item label="优先级" name="priority">
                <a-select v-model:value="addForm.priority" placeholder="选择优先级">
                  <a-select-option value="low">低</a-select-option>
                  <a-select-option value="normal">普通</a-select-option>
                  <a-select-option value="high">高</a-select-option>
                  <a-select-option value="urgent">紧急</a-select-option>
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
          <div class="section-title">
            <ContainerOutlined />
            镜像配置
          </div>
          <a-row :gutter="16">
            <a-col :span="20">
              <a-form-item label="容器镜像" name="image">
                <a-select v-model:value="addForm.image" placeholder="选择或输入容器镜像" mode="combobox" show-search>
                  <a-select-option value="jupyter/tensorflow-notebook:latest">
                    <div class="image-option">
                      <span class="image-name">jupyter/tensorflow-notebook:latest</span>
                      <a-tag color="blue" size="small">TensorFlow</a-tag>
                    </div>
                  </a-select-option>
                  <a-select-option value="jupyter/pytorch-notebook:latest">
                    <div class="image-option">
                      <span class="image-name">jupyter/pytorch-notebook:latest</span>
                      <a-tag color="orange" size="small">PyTorch</a-tag>
                    </div>
                  </a-select-option>
                  <a-select-option value="jupyter/datascience-notebook:latest">
                    <div class="image-option">
                      <span class="image-name">jupyter/datascience-notebook:latest</span>
                      <a-tag color="green" size="small">DataScience</a-tag>
                    </div>
                  </a-select-option>
                  <a-select-option value="codercom/code-server:latest">
                    <div class="image-option">
                      <span class="image-name">codercom/code-server:latest</span>
                      <a-tag color="purple" size="small">VS Code</a-tag>
                    </div>
                  </a-select-option>
                </a-select>
              </a-form-item>
            </a-col>
            <a-col :span="4">
              <a-form-item label="镜像拉取策略" name="imagePullPolicy">
                <a-select v-model:value="addForm.imagePullPolicy" placeholder="拉取策略">
                  <a-select-option value="Always">Always</a-select-option>
                  <a-select-option value="IfNotPresent">IfNotPresent</a-select-option>
                  <a-select-option value="Never">Never</a-select-option>
                </a-select>
              </a-form-item>
            </a-col>
          </a-row>
        </div>

        <div class="form-section">
          <div class="section-title">
            <ThunderboltOutlined />
            资源配置
          </div>
          <div class="resource-config-container">
            <a-row :gutter="16">
              <a-col :span="8">
                <a-form-item label="CPU限制" name="cpu_limit">
                  <a-input-number 
                    v-model:value="addForm.cpu_limit_num" 
                    :min="0.1" 
                    :max="32" 
                    :step="0.1"
                    placeholder="CPU核心数" 
                    class="full-width"
                    @change="updateCpuLimit"
                  />
                  <div class="resource-hint">推荐: 2-8核</div>
                </a-form-item>
              </a-col>
              <a-col :span="8">
                <a-form-item label="内存限制" name="memory_limit">
                  <a-input-number 
                    v-model:value="addForm.memory_limit_num" 
                    :min="1" 
                    :max="128" 
                    placeholder="内存大小(GB)" 
                    class="full-width"
                    @change="updateMemoryLimit"
                  />
                  <div class="resource-hint">推荐: 4-16GB</div>
                </a-form-item>
              </a-col>
              <a-col :span="8">
                <a-form-item label="GPU限制" name="gpu_limit">
                  <a-input-number 
                    v-model:value="addForm.gpu_limit" 
                    :min="0" 
                    :max="8" 
                    placeholder="GPU数量" 
                    class="full-width" 
                  />
                  <div class="resource-hint">可选: 0-8块</div>
                </a-form-item>
              </a-col>
            </a-row>
            <div class="resource-preview">
              <a-alert 
                :message="`资源预览: CPU ${addForm.cpu_limit || '未设置'}, 内存 ${addForm.memory_limit || '未设置'}, GPU ${addForm.gpu_limit || 0}块`"
                type="info" 
                show-icon 
              />
            </div>
          </div>
        </div>

        <div class="form-section">
          <div class="section-title">
            <DatabaseOutlined />
            存储配置
          </div>
          <div class="dynamic-form-container">
            <a-form-item 
              v-for="(volume, index) in addForm.volumes" 
              :key="volume.key"
              :label="index === 0 ? '存储卷映射' : ''" 
              :name="['volumes', index, 'hostPath']"
              class="volume-form-item"
            >
              <div class="volume-input-group">
                <a-input 
                  v-model:value="volume.hostPath" 
                  placeholder="主机路径 (如: /data/workspace)" 
                  class="volume-host-input" 
                />
                <div class="volume-separator">→</div>
                <a-input 
                  v-model:value="volume.containerPath" 
                  placeholder="容器路径 (如: /home/jovyan/work)" 
                  class="volume-container-input" 
                />
                <a-tooltip title="删除存储卷">
                  <a-button 
                    v-if="addForm.volumes.length > 1" 
                    type="text" 
                    danger 
                    @click="removeVolume(volume)"
                    class="remove-btn"
                  >
                    <DeleteOutlined />
                  </a-button>
                </a-tooltip>
              </div>
            </a-form-item>
            <a-form-item>
              <a-button type="dashed" class="add-dynamic-button" @click="addVolume" block>
                <PlusOutlined />
                添加存储卷
              </a-button>
            </a-form-item>
          </div>
        </div>

        <div class="form-section">
          <div class="section-title">
            <BugOutlined />
            环境变量
          </div>
          <div class="dynamic-form-container">
            <a-form-item 
              v-for="(env, index) in addForm.env_vars" 
              :key="env.key"
              :label="index === 0 ? '环境变量' : ''" 
              :name="['env_vars', index, 'envKey']"
              class="env-form-item"
            >
              <div class="env-input-group">
                <a-input 
                  v-model:value="env.envKey" 
                  placeholder="变量名 (如: CUDA_VISIBLE_DEVICES)" 
                  class="env-key-input" 
                />
                <div class="env-separator">=</div>
                <a-input 
                  v-model:value="env.envValue" 
                  placeholder="变量值 (如: 0,1)" 
                  class="env-value-input" 
                />
                <a-tooltip title="删除环境变量">
                  <a-button 
                    v-if="addForm.env_vars.length > 1" 
                    type="text" 
                    danger 
                    @click="removeEnvVar(env)"
                    class="remove-btn"
                  >
                    <DeleteOutlined />
                  </a-button>
                </a-tooltip>
              </div>
            </a-form-item>
            <a-form-item>
              <a-button type="dashed" class="add-dynamic-button" @click="addEnvVar" block>
                <PlusOutlined />
                添加环境变量
              </a-button>
            </a-form-item>
          </div>
        </div>

        <div class="form-section">
          <div class="section-title">
            <SafetyCertificateOutlined />
            高级配置
          </div>
          <a-row :gutter="16">
            <a-col :span="12">
              <a-form-item label="自动重启" name="autoRestart">
                <a-switch v-model:checked="addForm.autoRestart" />
                <div class="config-hint">启动失败时自动重启</div>
              </a-form-item>
            </a-col>
            <a-col :span="12">
              <a-form-item label="资源监控" name="enableMonitoring">
                <a-switch v-model:checked="addForm.enableMonitoring" />
                <div class="config-hint">启用资源使用监控</div>
              </a-form-item>
            </a-col>
          </a-row>
          <a-row :gutter="16">
            <a-col :span="12">
              <a-form-item label="网络模式" name="networkMode">
                <a-select v-model:value="addForm.networkMode" placeholder="选择网络模式">
                  <a-select-option value="bridge">Bridge</a-select-option>
                  <a-select-option value="host">Host</a-select-option>
                  <a-select-option value="none">None</a-select-option>
                </a-select>
              </a-form-item>
            </a-col>
            <a-col :span="12">
              <a-form-item label="超时时间(分钟)" name="timeout">
                <a-input-number 
                  v-model:value="addForm.timeout" 
                  :min="5" 
                  :max="1440" 
                  placeholder="启动超时时间" 
                  class="full-width" 
                />
              </a-form-item>
            </a-col>
          </a-row>
        </div>
      </a-form>
    </a-modal>

    <!-- 编辑Notebook模态框 -->
    <a-modal 
      title="编辑Notebook服务" 
      v-model:open="isEditModalVisible" 
      @ok="handleUpdate" 
      @cancel="closeEditModal"
      :width="900"
      class="custom-modal"
      :confirm-loading="submitLoading"
      ok-text="更新"
      cancel-text="取消"
    >
      <a-form ref="editFormRef" :model="editForm" layout="vertical" class="custom-form" :rules="formRules">
        <!-- 与创建表单相同的结构，但使用editForm -->
        <div class="form-section">
          <div class="section-title">
            <SettingOutlined />
            基本信息
          </div>
          <a-row :gutter="16">
            <a-col :span="12">
              <a-form-item label="Notebook名称" name="name">
                <a-input v-model:value="editForm.name" placeholder="请输入Notebook名称" />
              </a-form-item>
            </a-col>
            <a-col :span="12">
              <a-form-item label="Notebook类型" name="type">
                <a-select v-model:value="editForm.type" placeholder="请选择Notebook类型">
                  <a-select-option value="jupyter">
                    <ExperimentOutlined /> Jupyter Notebook
                  </a-select-option>
                  <a-select-option value="vscode">
                    <CodeOutlined /> VS Code Server
                  </a-select-option>
                  <a-select-option value="rstudio">
                    <BarChartOutlined /> RStudio Server
                  </a-select-option>
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
          <div class="section-title">
            <ContainerOutlined />
            镜像配置
          </div>
          <a-row :gutter="16">
            <a-col :span="24">
              <a-form-item label="容器镜像" name="image">
                <a-select v-model:value="editForm.image" placeholder="选择或输入容器镜像" mode="combobox" show-search>
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
          <div class="section-title">
            <ThunderboltOutlined />
            资源配置
          </div>
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
          <div class="section-title">
            <DatabaseOutlined />
            存储配置
          </div>
          <a-form-item v-for="(volume, index) in editForm.volumes" :key="volume.key"
            :label="index === 0 ? '存储卷' : ''" :name="['volumes', index, 'hostPath']">
            <div class="volume-input-group">
              <a-input v-model:value="volume.hostPath" placeholder="主机路径" class="volume-host-input" />
              <div class="volume-separator">→</div>
              <a-input v-model:value="volume.containerPath" placeholder="容器路径" class="volume-container-input" />
              <a-button v-if="editForm.volumes.length > 1" type="text" danger @click="removeVolumeEdit(volume)" class="remove-btn">
                <DeleteOutlined />
              </a-button>
            </div>
          </a-form-item>
          <a-form-item>
            <a-button type="dashed" class="add-dynamic-button" @click="addVolumeEdit" block>
              <PlusOutlined />
              添加存储卷
            </a-button>
          </a-form-item>
        </div>

        <div class="form-section">
          <div class="section-title">
            <BugOutlined />
            环境变量
          </div>
          <a-form-item v-for="(env, index) in editForm.env_vars" :key="env.key"
            :label="index === 0 ? '环境变量' : ''" :name="['env_vars', index, 'envKey']">
            <div class="env-input-group">
              <a-input v-model:value="env.envKey" placeholder="变量名" class="env-key-input" />
              <div class="env-separator">=</div>
              <a-input v-model:value="env.envValue" placeholder="变量值" class="env-value-input" />
              <a-button v-if="editForm.env_vars.length > 1" type="text" danger @click="removeEnvVarEdit(env)" class="remove-btn">
                <DeleteOutlined />
              </a-button>
            </div>
          </a-form-item>
          <a-form-item>
            <a-button type="dashed" class="add-dynamic-button" @click="addEnvVarEdit" block>
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
      v-model:open="isViewModalVisible" 
      @cancel="closeViewModal"
      :width="1000"
      class="custom-modal detail-modal"
      :footer="null"
    >
      <div class="notebook-detail-container" v-if="viewNotebook">
        <a-tabs default-active-key="basic" class="detail-tabs">
          <a-tab-pane key="basic" tab="基本信息">
            <div class="detail-section">
              <a-descriptions :column="2" size="middle" bordered>
                <a-descriptions-item label="Notebook名称" :span="2">
                  <span class="detail-value">{{ viewNotebook.name }}</span>
                </a-descriptions-item>
                <a-descriptions-item label="命名空间">{{ viewNotebook.namespace }}</a-descriptions-item>
                <a-descriptions-item label="创建者">{{ viewNotebook.creator }}</a-descriptions-item>
                <a-descriptions-item label="Notebook类型">
                  <a-tag :color="getTypeColor(viewNotebook.type)">{{ getTypeText(viewNotebook.type) }}</a-tag>
                </a-descriptions-item>
                <a-descriptions-item label="运行状态">
                  <a-tag :color="getStatusColor(viewNotebook.status)">{{ getStatusText(viewNotebook.status) }}</a-tag>
                </a-descriptions-item>
                <a-descriptions-item label="访问地址" :span="2" v-if="viewNotebook.status === 'Running'">
                  <a :href="viewNotebook.access_url" target="_blank" class="access-url-link">
                    {{ viewNotebook.access_url }}
                  </a>
                  <a-button type="text" size="small" @click="copyAccessUrl(viewNotebook)" class="copy-btn">
                    <CopyOutlined />
                  </a-button>
                </a-descriptions-item>
                <a-descriptions-item label="描述信息" :span="2">{{ viewNotebook.description || '无' }}</a-descriptions-item>
                <a-descriptions-item label="创建时间">{{ viewNotebook.created_at }}</a-descriptions-item>
                <a-descriptions-item label="启动时间">{{ viewNotebook.start_time || '未启动' }}</a-descriptions-item>
              </a-descriptions>
            </div>
          </a-tab-pane>

          <a-tab-pane key="resources" tab="资源配置">
            <div class="detail-section">
              <div class="resource-cards">
                <div class="resource-card">
                  <div class="resource-card-header">
                    <div class="resource-card-icon cpu-card">
                      <span>CPU</span>
                    </div>
                    <div class="resource-card-title">CPU配置</div>
                  </div>
                  <div class="resource-card-content">
                    <div class="resource-metric">
                      <span class="metric-label">限制:</span>
                      <span class="metric-value">{{ viewNotebook.cpu_limit }}</span>
                    </div>
                  </div>
                </div>

                <div class="resource-card">
                  <div class="resource-card-header">
                    <div class="resource-card-icon memory-card">
                      <span>RAM</span>
                    </div>
                    <div class="resource-card-title">内存配置</div>
                  </div>
                  <div class="resource-card-content">
                    <div class="resource-metric">
                      <span class="metric-label">限制:</span>
                      <span class="metric-value">{{ viewNotebook.memory_limit }}</span>
                    </div>
                  </div>
                </div>

                <div class="resource-card">
                  <div class="resource-card-header">
                    <div class="resource-card-icon gpu-card">
                      <span>GPU</span>
                    </div>
                    <div class="resource-card-title">GPU配置</div>
                  </div>
                  <div class="resource-card-content">
                    <div class="resource-metric">
                      <span class="metric-label">数量:</span>
                      <span class="metric-value">{{ viewNotebook.gpu_limit || 0 }}块</span>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </a-tab-pane>

          <a-tab-pane key="config" tab="配置详情">
            <div class="detail-section">
              <div class="config-section">
                <h4>
                  <ContainerOutlined />
                  镜像信息
                </h4>
                <div class="config-item">
                  <span class="config-label">容器镜像:</span>
                  <span class="config-value">{{ viewNotebook.image }}</span>
                </div>
              </div>

              <div class="config-section" v-if="viewNotebook.env_vars && viewNotebook.env_vars.length > 0">
                <h4>
                  <BugOutlined />
                  环境变量
                </h4>
                <div class="env-list">
                  <div class="env-item" v-for="env in viewNotebook.env_vars" :key="env">
                    <span class="env-key">{{ env.split('=')[0] }}</span>
                    <span class="env-separator">=</span>
                    <span class="env-value">{{ env.split('=')[1] }}</span>
                  </div>
                </div>
              </div>

              <div class="config-section" v-if="viewNotebook.volumes && viewNotebook.volumes.length > 0">
                <h4>
                  <DatabaseOutlined />
                  存储卷
                </h4>
                <div class="volume-list">
                  <div class="volume-item" v-for="volume in viewNotebook.volumes" :key="volume">
                    <span class="volume-host">{{ volume.split(':')[0] }}</span>
                    <span class="volume-separator">→</span>
                    <span class="volume-container">{{ volume.split(':')[1] }}</span>
                  </div>
                </div>
              </div>
            </div>
          </a-tab-pane>

          <a-tab-pane key="logs" tab="运行日志">
            <div class="logs-container">
              <div class="logs-header">
                <a-button type="primary" size="small" @click="refreshLogs">
                  <ReloadOutlined />
                  刷新日志
                </a-button>
                <a-button size="small" @click="downloadLogs">
                  <CloudDownloadOutlined />
                  下载日志
                </a-button>
              </div>
              <div class="logs-content">
                <pre class="logs-text">{{ mockLogs }}</pre>
              </div>
            </div>
          </a-tab-pane>
        </a-tabs>
      </div>
    </a-modal>

    <!-- 导入配置模态框 -->
    <a-modal 
      title="导入Notebook配置" 
      v-model:open="isImportModalVisible" 
      @ok="handleImport" 
      @cancel="closeImportModal"
      :width="600"
      class="custom-modal"
    >
      <div class="import-container">
        <a-upload-dragger
          v-model:file-list="importFileList"
          name="file"
          :multiple="false"
          accept=".json,.yaml,.yml"
          :before-upload="beforeUpload"
          @remove="handleRemove"
        >
          <p class="ant-upload-drag-icon">
            <CloudUploadOutlined />
          </p>
          <p class="ant-upload-text">点击或拖拽文件到此区域上传</p>
          <p class="ant-upload-hint">
            支持 JSON 和 YAML 格式的配置文件
          </p>
        </a-upload-dragger>
      </div>
    </a-modal>

    <!-- 克隆配置模态框 -->
    <a-modal 
      title="克隆Notebook配置" 
      v-model:open="isCloneModalVisible" 
      @ok="handleCloneConfirm" 
      @cancel="closeCloneModal"
      :width="500"
      class="custom-modal"
    >
      <a-form layout="vertical">
        <a-form-item label="新Notebook名称" required>
          <a-input v-model:value="cloneForm.name" placeholder="请输入新的Notebook名称" />
        </a-form-item>
        <a-form-item label="目标命名空间">
          <a-select v-model:value="cloneForm.namespace" placeholder="选择命名空间">
            <a-select-option value="default">default</a-select-option>
            <a-select-option value="dev-team">dev-team</a-select-option>
            <a-select-option value="data-team">data-team</a-select-option>
            <a-select-option value="research">research</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="是否立即启动">
          <a-switch v-model:checked="cloneForm.autoStart" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed } from 'vue';
import { message, Modal } from 'ant-design-vue';
import {
  SearchOutlined,
  ReloadOutlined,
  PlusOutlined,
  DeleteOutlined,
  LinkOutlined,
  EyeOutlined,
  EditOutlined,
  PlayCircleOutlined,
  PauseCircleOutlined,
  StopOutlined,
  LoadingOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  CopyOutlined,
  CodeOutlined,
  ExperimentOutlined,
  BarChartOutlined,
  ContainerOutlined,
  ClockCircleOutlined,
  ThunderboltOutlined,
  DatabaseOutlined,
  BugOutlined,
  SettingOutlined,
  SafetyCertificateOutlined,
  CloudUploadOutlined,
  CloudDownloadOutlined
} from '@ant-design/icons-vue';
import type { FormInstance, UploadFile } from 'ant-design-vue';

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
  priority?: string;
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

interface StatisticItem {
  title: string;
  value: string | number;
  color: string;
  icon: any;
}

// 响应式数据
const searchText = ref('');
const statusFilter = ref('');
const typeFilter = ref('');
const loading = ref(false);
const submitLoading = ref(false);

// 表格数据
const data = ref<NotebookItem[]>([]);

// 分页配置
const paginationConfig = reactive({
  current: 1,
  pageSize: 10,
  total: 0,
  showSizeChanger: true,
  showQuickJumper: true,
  showTotal: (total: number, range: [number, number]) => 
    `共 ${total} 条记录，当前显示 ${range[0]}-${range[1]} 条`,
  pageSizeOptions: ['10', '20', '50', '100'],
});

// 模态框状态
const isAddModalVisible = ref(false);
const isEditModalVisible = ref(false);
const isViewModalVisible = ref(false);
const isImportModalVisible = ref(false);
const isCloneModalVisible = ref(false);

// 表单引用
const addFormRef = ref<FormInstance>();
const editFormRef = ref<FormInstance>();

// 查看详情的Notebook
const viewNotebook = ref<NotebookItem | null>(null);

// 统计数据
const statistics = ref<StatisticItem[]>([
  { title: '总数量', value: 0, color: '#1890ff', icon: 'ContainerOutlined' },
  { title: '运行中', value: 0, color: '#52c41a', icon: 'PlayCircleOutlined' },
  { title: '已停止', value: 0, color: '#faad14', icon: 'PauseCircleOutlined' },
  { title: '失败', value: 0, color: '#ff4d4f', icon: 'CloseCircleOutlined' }
]);

// 表单验证规则
const formRules = {
  name: [
    { required: true, message: '请输入Notebook名称', trigger: 'blur' },
    { min: 3, max: 50, message: '名称长度应在3-50个字符之间', trigger: 'blur' }
  ],
  type: [{ required: true, message: '请选择Notebook类型', trigger: 'change' }],
  image: [{ required: true, message: '请输入容器镜像', trigger: 'blur' }],
  namespace: [{ required: true, message: '请选择命名空间', trigger: 'change' }]
};

// 环境变量和存储卷计数器
let envKeyCounter = 0;
let volumeKeyCounter = 0;

// 新增表单
const addForm = reactive({
  name: '',
  type: 'jupyter',
  namespace: 'default',
  priority: 'normal',
  description: '',
  image: 'jupyter/tensorflow-notebook:latest',
  imagePullPolicy: 'IfNotPresent',
  cpu_limit: '2',
  cpu_limit_num: 2,
  memory_limit: '4Gi',
  memory_limit_num: 4,
  gpu_limit: 0,
  env_vars: [] as EnvVar[],
  volumes: [] as Volume[],
  autoRestart: false,
  enableMonitoring: true,
  networkMode: 'bridge',
  timeout: 30
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

// 克隆表单
const cloneForm = reactive({
  name: '',
  namespace: 'default',
  autoStart: false,
  sourceNotebook: null as NotebookItem | null
});

// 导入文件列表
const importFileList = ref<UploadFile[]>([]);

// 模拟日志数据
const mockLogs = ref(`[2024-06-11 10:30:00] Starting Jupyter notebook server...
[2024-06-11 10:30:01] Loading configuration files...
[2024-06-11 10:30:02] Initializing GPU environment...
[2024-06-11 10:30:03] CUDA devices detected: 2
[2024-06-11 10:30:04] Setting up workspace directories...
[2024-06-11 10:30:05] Installing additional packages...
[2024-06-11 10:30:15] Server started successfully
[2024-06-11 10:30:15] Jupyter server is running at: http://localhost:8888
[2024-06-11 10:30:15] Token: abc123def456ghi789
[2024-06-11 10:30:16] Ready to accept connections`);

// 表格列配置
const columns = [
  {
    title: 'ID',
    dataIndex: 'id',
    key: 'id',
    width: 70,
    fixed: 'left',
  },
  {
    title: 'Notebook名称',
    dataIndex: 'name',
    key: 'name',
    width: 180,
    fixed: 'left',
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
    title: '资源配置',
    key: 'resources',
    slots: { customRender: 'resources' },
    width: 180,
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
    width: 160,
  },
  {
    title: '操作',
    key: 'action',
    slots: { customRender: 'action' },
    width: 280,
    fixed: 'right',
  },
];

// 计算属性：过滤后的数据
const filteredData = computed(() => {
  let result = data.value;
  
  if (searchText.value) {
    result = result.filter(item => 
      item.name.toLowerCase().includes(searchText.value.toLowerCase()) ||
      item.creator.toLowerCase().includes(searchText.value.toLowerCase())
    );
  }
  
  if (statusFilter.value) {
    result = result.filter(item => item.status === statusFilter.value);
  }
  
  if (typeFilter.value) {
    result = result.filter(item => item.type === typeFilter.value);
  }
  
  return result;
});

// 初始化数据
onMounted(() => {
  initForms();
  loadData();
});

// 初始化表单
const initForms = () => {
  addForm.env_vars = [{ envKey: '', envValue: '', key: ++envKeyCounter }];
  addForm.volumes = [{ hostPath: '', containerPath: '', key: ++volumeKeyCounter }];
};
// 更新统计数据
const updateStatistics = () => {
  if (!statistics.value) return;
  
  const total = data.value.length;
  const running = data.value.filter(item => item.status === 'Running').length;
  const stopped = data.value.filter(item => item.status === 'Stopped').length;
  const failed = data.value.filter(item => item.status === 'Failed').length;
  
  if (statistics.value[0]) statistics.value[0].value = total;
  if (statistics.value[1]) statistics.value[1].value = running;
  if (statistics.value[2]) statistics.value[2].value = stopped;
  if (statistics.value[3]) statistics.value[3].value = failed;
};

// 获取状态颜色
const getStatusColor = (status: string) => {
  const colorMap: Record<string, string> = {
    'Creating': 'processing',
    'Running': 'success',
    'Stopped': 'default',
    'Failed': 'error',
    'Deleting': 'error'
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
    return `${days}天${hours}时`;
  } else if (hours > 0) {
    return `${hours}时${minutes}分`;
  } else {
    return `${minutes}分钟`;
  }
};

// 加载数据
const loadData = async () => {
  loading.value = true;
  try {
    // 模拟API调用延迟
    await new Promise(resolve => setTimeout(resolve, 500));
    
    // 模拟数据
    const mockData: NotebookItem[] = [
      {
        id: 1,
        name: 'tensorflow-research-nb',
        namespace: 'default',
        type: 'jupyter',
        status: 'Running',
        image: 'jupyter/tensorflow-notebook:latest',
        description: 'TensorFlow深度学习研究环境，用于计算机视觉和自然语言处理实验',
        cpu_limit: '4',
        memory_limit: '8Gi',
        gpu_limit: 1,
        env_vars: ['JUPYTER_ENABLE_LAB=yes', 'PYTHONPATH=/workspace', 'CUDA_VISIBLE_DEVICES=0'],
        volumes: ['/data:/home/jovyan/work', '/models:/home/jovyan/models'],
        access_url: 'https://notebook-001.ml-platform.com',
        created_at: '2024-06-09 09:30:00',
        start_time: '2024-06-09 09:32:00',
        creator: 'admin',
        priority: 'high'
      },
      {
        id: 2,
        name: 'vscode-dev-env',
        namespace: 'dev-team',
        type: 'vscode',
        status: 'Running',
        image: 'codercom/code-server:latest',
        description: 'VS Code开发环境，支持多种编程语言和扩展',
        cpu_limit: '2',
        memory_limit: '4Gi',
        gpu_limit: 0,
        env_vars: ['PASSWORD=mypassword', 'SUDO_PASSWORD=mypassword'],
        volumes: ['/workspace:/home/coder/workspace', '/home/user/.ssh:/home/coder/.ssh'],
        access_url: 'https://vscode-001.ml-platform.com',
        created_at: '2024-06-09 10:15:00',
        start_time: '2024-06-09 10:17:00',
        creator: 'developer1',
        priority: 'normal'
      },
      {
        id: 3,
        name: 'r-analysis-notebook',
        namespace: 'data-team',
        type: 'rstudio',
        status: 'Stopped',
        image: 'rocker/rstudio:latest',
        description: 'R语言数据分析环境，包含统计学习和数据可视化包',
        cpu_limit: '2',
        memory_limit: '4Gi',
        gpu_limit: 0,
        env_vars: ['DISABLE_AUTH=true', 'ROOT=TRUE'],
        volumes: ['/r-data:/home/rstudio/data', '/r-projects:/home/rstudio/projects'],
        created_at: '2024-06-08 14:20:00',
        start_time: '2024-06-08 14:22:00',
        creator: 'analyst1',
        priority: 'low'
      },
      {
        id: 4,
        name: 'pytorch-experiment',
        namespace: 'research',
        type: 'jupyter',
        status: 'Creating',
        image: 'jupyter/pytorch-notebook:latest',
        description: 'PyTorch实验环境，用于深度学习模型训练和推理',
        cpu_limit: '8',
        memory_limit: '16Gi',
        gpu_limit: 2,
        env_vars: ['CUDA_VISIBLE_DEVICES=0,1', 'PYTHONPATH=/workspace', 'OMP_NUM_THREADS=8'],
        volumes: ['/experiments:/home/jovyan/experiments', '/datasets:/home/jovyan/datasets'],
        created_at: '2024-06-11 11:45:00',
        creator: 'researcher1',
        priority: 'urgent'
      },
      {
        id: 5,
        name: 'data-preprocessing',
        namespace: 'data-team',
        type: 'jupyter',
        status: 'Failed',
        image: 'jupyter/datascience-notebook:latest',
        description: '数据预处理工作环境，支持pandas、numpy等数据科学库',
        cpu_limit: '4',
        memory_limit: '8Gi',
        gpu_limit: 0,
        env_vars: ['JUPYTER_ENABLE_LAB=yes'],
        volumes: ['/raw-data:/home/jovyan/raw-data', '/processed-data:/home/jovyan/processed'],
        created_at: '2024-06-10 16:30:00',
        creator: 'data_engineer1',
        priority: 'normal'
      }
    ];
    
    data.value = mockData;
    paginationConfig.total = mockData.length;
    updateStatistics();
  } catch (error) {
    message.error('加载数据失败');
  } finally {
    loading.value = false;
  }
};

// 搜索处理
const handleSearch = async () => {
  loading.value = true;
  await new Promise(resolve => setTimeout(resolve, 300));
  loading.value = false;
  message.success('搜索完成');
};

// 重置处理
const handleReset = () => {
  searchText.value = '';
  statusFilter.value = '';
  typeFilter.value = '';
  message.success('重置成功');
};

// 表格变化处理
const handleTableChange = (pagination: any) => {
  paginationConfig.current = pagination.current;
  paginationConfig.pageSize = pagination.pageSize;
};

// 更新CPU限制
const updateCpuLimit = (value: number) => {
  addForm.cpu_limit = value ? `${value}` : '';
};

// 更新内存限制
const updateMemoryLimit = (value: number) => {
  addForm.memory_limit = value ? `${value}Gi` : '';
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
    namespace: 'default',
    priority: 'normal',
    description: '',
    image: 'jupyter/tensorflow-notebook:latest',
    imagePullPolicy: 'IfNotPresent',
    cpu_limit: '2',
    cpu_limit_num: 2,
    memory_limit: '4Gi',
    memory_limit_num: 4,
    gpu_limit: 0,
    env_vars: [{ envKey: '', envValue: '', key: ++envKeyCounter }],
    volumes: [{ hostPath: '', containerPath: '', key: ++volumeKeyCounter }],
    autoRestart: false,
    enableMonitoring: true,
    networkMode: 'bridge',
    timeout: 30
  });
  addFormRef.value?.resetFields();
};

// 新增Notebook
const handleAdd = async () => {
  try {
    submitLoading.value = true;
    await addFormRef.value?.validateFields();
    
    // 处理环境变量和存储卷数据
    const envVars = addForm.env_vars
      .filter(env => env.envKey && env.envValue)
      .map(env => `${env.envKey}=${env.envValue}`);
    
    const volumes = addForm.volumes
      .filter(vol => vol.hostPath && vol.containerPath)
      .map(vol => `${vol.hostPath}:${vol.containerPath}`);

    const newNotebook: NotebookItem = {
      id: data.value.length + 1,
      name: addForm.name,
      namespace: addForm.namespace,
      type: addForm.type,
      status: 'Creating',
      image: addForm.image,
      description: addForm.description,
      cpu_limit: addForm.cpu_limit,
      memory_limit: addForm.memory_limit,
      gpu_limit: addForm.gpu_limit,
      env_vars: envVars,
      volumes: volumes,
      created_at: new Date().toLocaleString(),
      creator: 'current_user',
      priority: addForm.priority
    };

    // 模拟API调用
    await new Promise(resolve => setTimeout(resolve, 1000));
    
    data.value.unshift(newNotebook);
    paginationConfig.total++;
    updateStatistics();
    
    message.success('Notebook创建成功');
    closeAddModal();
  } catch (error) {
    console.error('创建失败:', error);
    message.error('创建失败，请检查输入信息');
  } finally {
    submitLoading.value = false;
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
      return { envKey: envKey || '', envValue: envValue || '', key: ++envKeyCounter };
    }),
    volumes: record.volumes.map(vol => {
      const [hostPath, containerPath] = vol.split(':');
      return { hostPath: hostPath || '', containerPath: containerPath || '', key: ++volumeKeyCounter };
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
    submitLoading.value = true;
    await editFormRef.value?.validateFields();
    
    // 处理环境变量和存储卷数据
    const envVars = editForm.env_vars
      .filter(env => env.envKey && env.envValue)
      .map(env => `${env.envKey}=${env.envValue}`);
    
    const volumes = editForm.volumes
      .filter(vol => vol.hostPath && vol.containerPath)
      .map(vol => `${vol.hostPath}:${vol.containerPath}`);

    // 模拟API调用
    await new Promise(resolve => setTimeout(resolve, 800));

    // 更新数据
    const index = data.value.findIndex(item => item.id === editForm.id);
    if (index !== -1) {
      Object.assign(data.value[index] as NotebookItem, {
        name: editForm.name,
        type: editForm.type,
        description: editForm.description,
        image: editForm.image,
        cpu_limit: editForm.cpu_limit,
        memory_limit: editForm.memory_limit,
        gpu_limit: editForm.gpu_limit,
        env_vars: envVars,
        volumes: volumes
      });
    }

    message.success('Notebook更新成功');
    closeEditModal();
  } catch (error) {
    console.error('更新失败:', error);
    message.error('更新失败，请检查输入信息');
  } finally {
    submitLoading.value = false;
  }
};

// 启动Notebook
const handleStart = (record: NotebookItem) => {
  Modal.confirm({
    title: '确认启动Notebook',
    content: `确定要启动Notebook "${record.name}" 吗？`,
    okText: '确认',
    cancelText: '取消',
    onOk() {
      record.status = 'Creating';
      setTimeout(() => {
        record.status = 'Running';
        record.start_time = new Date().toLocaleString();
        record.access_url = `https://notebook-${record.id}.ml-platform.com`;
        updateStatistics();
        message.success('Notebook启动成功');
      }, 2000);
      message.loading('正在启动Notebook...', 2);
    },
  });
};

// 停止Notebook
const handleStop = (record: NotebookItem) => {
  Modal.confirm({
    title: '确认停止Notebook',
    content: `确定要停止Notebook "${record.name}" 吗？`,
    okText: '确认',
    cancelText: '取消',
    onOk() {
      record.status = 'Stopped';
      record.access_url = undefined;
      updateStatistics();
      message.success('Notebook已停止');
    },
  });
};

// 重启Notebook
const handleRestart = (record: NotebookItem) => {
  Modal.confirm({
    title: '确认重启Notebook',
    content: `确定要重启Notebook "${record.name}" 吗？`,
    okText: '确认',
    cancelText: '取消',
    onOk() {
      record.status = 'Creating';
      setTimeout(() => {
        record.status = 'Running';
        record.start_time = new Date().toLocaleString();
        updateStatistics();
        message.success('Notebook重启成功');
      }, 3000);
      message.loading('正在重启Notebook...', 3);
    },
  });
};

// 克隆Notebook
const handleClone = (record: NotebookItem) => {
  cloneForm.sourceNotebook = record;
  cloneForm.name = `${record.name}-copy`;
  cloneForm.namespace = record.namespace;
  cloneForm.autoStart = false;
  isCloneModalVisible.value = true;
};

// 关闭克隆模态框
const closeCloneModal = () => {
  isCloneModalVisible.value = false;
  cloneForm.sourceNotebook = null;
};

// 确认克隆
const handleCloneConfirm = async () => {
  if (!cloneForm.name) {
    message.error('请输入新Notebook名称');
    return;
  }
  
  const sourceNotebook = cloneForm.sourceNotebook!;
  const newNotebook: NotebookItem = {
    ...sourceNotebook,
    id: data.value.length + 1,
    name: cloneForm.name,
    namespace: cloneForm.namespace,
    status: cloneForm.autoStart ? 'Creating' : 'Stopped',
    created_at: new Date().toLocaleString(),
    start_time: cloneForm.autoStart ? new Date().toLocaleString() : undefined,
    access_url: undefined,
    creator: 'current_user'
  };
  
  if (cloneForm.autoStart) {
    setTimeout(() => {
      newNotebook.status = 'Running';
      newNotebook.access_url = `https://notebook-${newNotebook.id}.ml-platform.com`;
      updateStatistics();
    }, 2000);
  }
  
  data.value.unshift(newNotebook);
  paginationConfig.total++;
  updateStatistics();
  
  message.success('Notebook克隆成功');
  closeCloneModal();
};

// 删除Notebook
const handleDelete = (record: NotebookItem) => {
  Modal.confirm({
    title: '确认删除Notebook',
    content: `确定要删除Notebook "${record.name}" 吗？此操作不可恢复。`,
    okText: '确认删除',
    cancelText: '取消',
    okType: 'danger',
    onOk() {
      const index = data.value.findIndex(item => item.id === record.id);
      if (index !== -1) {
        data.value.splice(index, 1);
        paginationConfig.total--;
        updateStatistics();
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

// 复制访问地址
const copyAccessUrl = async (record: NotebookItem) => {
  if (record.access_url) {
    try {
      await navigator.clipboard.writeText(record.access_url);
      message.success('访问地址已复制到剪贴板');
    } catch (error) {
      message.error('复制失败');
    }
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

// 导入相关功能
const showImportModal = () => {
  isImportModalVisible.value = true;
};

const closeImportModal = () => {
  isImportModalVisible.value = false;
  importFileList.value = [];
};

const beforeUpload = (file: UploadFile) => {
  const isValidType = file.type === 'application/json' || 
                      file.name?.endsWith('.yaml') || 
                      file.name?.endsWith('.yml');
  if (!isValidType) {
    message.error('只支持 JSON 和 YAML 格式文件！');
  }
  const isLt2M = file.size! / 1024 / 1024 < 2;
  if (!isLt2M) {
    message.error('文件大小不能超过 2MB！');
  }
  return isValidType && isLt2M;
};

const handleRemove = (file: UploadFile) => {
  const index = importFileList.value.indexOf(file);
  if (index > -1) {
    importFileList.value.splice(index, 1);
  }
};

const handleImport = async () => {
  if (importFileList.value.length === 0) {
    message.error('请选择要导入的配置文件');
    return;
  }
  
  // 模拟导入处理
  message.loading('正在导入配置...', 2);
  await new Promise(resolve => setTimeout(resolve, 2000));
  
  message.success('配置导入成功');
  closeImportModal();
  loadData();
};

// 导出数据
const handleExport = () => {
  const exportData = {
    notebooks: data.value,
    exportTime: new Date().toISOString(),
    version: '1.0'
  };
  
  const blob = new Blob([JSON.stringify(exportData, null, 2)], {
    type: 'application/json'
  });
  
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = `notebooks-export-${new Date().toISOString().split('T')[0]}.json`;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(url);
  
  message.success('数据导出成功');
};

// 刷新日志
const refreshLogs = () => {
  message.loading('正在刷新日志...', 1);
  // 模拟日志刷新
  setTimeout(() => {
    mockLogs.value += `\n[${new Date().toLocaleString()}] Log refreshed by user`;
    message.success('日志刷新成功');
  }, 1000);
};

// 下载日志
const downloadLogs = () => {
  const blob = new Blob([mockLogs.value], { type: 'text/plain' });
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = `notebook-${viewNotebook.value?.id}-logs.txt`;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(url);
  
  message.success('日志下载成功');
};
</script>

<style scoped>
/* 基础样式 */
.notebook-service-page {
  padding: 24px;
  min-height: 100vh;
}

/* 页面标题 */
.page-header {
  margin-bottom: 32px;
  text-align: center;
}

.page-title {
  font-size: 32px;
  font-weight: 700;
  color: #1a202c;
  margin: 0 0 8px 0;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  background-clip: text;
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
}

.page-description {
  font-size: 16px;
  color: #64748b;
  margin: 0;
}

/* 统计卡片 */
.stats-container {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 20px;
  margin-bottom: 32px;
}

.stat-card {
  background: white;
  border-radius: 12px;
  padding: 24px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.08);
  border: 1px solid #e2e8f0;
  display: flex;
  align-items: center;
  gap: 16px;
  transition: transform 0.2s ease, box-shadow 0.2s ease;
}

.stat-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 30px rgba(0, 0, 0, 0.12);
}

.stat-icon {
  width: 56px;
  height: 56px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
  font-weight: bold;
}

.stat-content {
  flex: 1;
}

.stat-value {
  font-size: 28px;
  font-weight: 700;
  color: #1a202c;
  line-height: 1;
}

.stat-title {
  font-size: 14px;
  color: #64748b;
  margin-top: 4px;
}

/* 卡片样式 */
.dashboard-card {
  background: white;
  border-radius: 12px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.08);
  border: 1px solid #e2e8f0;
  margin-bottom: 24px;
  overflow: hidden;
}

/* 工具栏 */
.custom-toolbar {
  padding: 24px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 16px;
  border-bottom: 1px solid #e2e8f0;
}

.search-filters {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
  align-items: center;
}

.search-input {
  width: 280px;
}

.status-filter,
.type-filter {
  width: 180px;
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

.action-buttons {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}

.add-button {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border: none;
  color: white;
}

.import-button {
  background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
  border: none;
  color: white;
}

.export-button {
  background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
  border: none;
  color: white;
}

/* 表格样式 */
.table-container {
  padding: 0;
}

.custom-table {
  border-radius: 0;
}

.custom-table :deep(.ant-table-thead > tr > th) {
  background: linear-gradient(135deg, #f8fafc 0%, #e2e8f0 100%);
  border-bottom: 2px solid #cbd5e0;
  color: #374151;
  font-weight: 600;
  padding: 16px 12px;
}

.custom-table :deep(.ant-table-tbody > tr) {
  transition: all 0.2s ease;
}

.custom-table :deep(.ant-table-tbody > tr:hover > td) {
  background: linear-gradient(135deg, #f8fafc 0%, #e2e8f0 30%);
}

.custom-table :deep(.ant-table-tbody > tr > td) {
  padding: 16px 12px;
  border-bottom: 1px solid #e2e8f0;
}

/* 状态标签 */
.status-tag {
  border-radius: 6px;
  font-size: 12px;
  font-weight: 600;
  padding: 4px 8px;
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.type-tag {
  border-radius: 6px;
  font-size: 12px;
  font-weight: 600;
  padding: 4px 8px;
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

/* 资源配置 */
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
}

.resource-icon {
  width: 20px;
  height: 20px;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 10px;
  font-weight: bold;
  color: white;
}

.cpu-icon {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.memory-icon {
  background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
}

.gpu-icon {
  background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
}

.resource-label {
  color: #64748b;
  font-weight: 500;
  min-width: 32px;
}

.resource-value {
  color: #1a202c;
  font-weight: 600;
}

/* 镜像显示 */
.image-container {
  display: flex;
  align-items: center;
  gap: 8px;
  max-width: 180px;
}

.image-icon {
  color: #3b82f6;
  font-size: 16px;
}

.image-text {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
  color: #4b5563;
}

/* 运行时间 */
.duration-container {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: #4b5563;
  font-family: 'Monaco', 'Menlo', monospace;
}

.duration-icon {
  color: #64748b;
}

/* 访问地址 */
.access-container {
  display: flex;
  align-items: center;
  gap: 4px;
}

.access-link {
  padding: 4px 8px;
  font-size: 12px;
  border-radius: 4px;
}

.copy-btn {
  padding: 4px;
  font-size: 12px;
  color: #64748b;
}

.access-disabled {
  color: #9ca3af;
  font-size: 12px;
  display: flex;
  align-items: center;
  gap: 4px;
}

.access-url-link {
  color: #3b82f6;
  text-decoration: none;
  font-weight: 500;
}

.access-url-link:hover {
  text-decoration: underline;
}

/* 操作列 */
.action-column {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}

.action-btn {
  min-width: 32px;
  height: 32px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  transition: all 0.2s ease;
}

.start-btn:hover {
  color: #52c41a;
  border-color: #52c41a;
}

.stop-btn:hover {
  color: #faad14;
  border-color: #faad14;
}

.restart-btn:hover {
  color: #1890ff;
  border-color: #1890ff;
}

.clone-btn:hover {
  color: #722ed1;
  border-color: #722ed1;
}

.delete-btn:hover {
  color: #ff4d4f;
  border-color: #ff4d4f;
}

/* 模态框样式 */
.custom-modal :deep(.ant-modal-header) {
  background: linear-gradient(135deg, #f8fafc 0%, #e2e8f0 100%);
  border-bottom: 2px solid #cbd5e0;
  padding: 20px 24px;
  border-radius: 12px 12px 0 0;
}

.custom-modal :deep(.ant-modal-title) {
  font-size: 20px;
  font-weight: 700;
  color: #1a202c;
}

.custom-modal :deep(.ant-modal-content) {
  border-radius: 12px;
  overflow: hidden;
}

.custom-modal :deep(.ant-modal-body) {
  padding: 24px;
  max-height: 70vh;
  overflow-y: auto;
}

/* 表单样式 */
.custom-form {
  margin-top: 0;
}

.form-section {
  margin-bottom: 32px;
  padding: 20px;
  background: #f8fafc;
  border-radius: 8px;
  border: 1px solid #e2e8f0;
}

.section-title {
  font-size: 16px;
  font-weight: 700;
  color: #1a202c;
  margin-bottom: 16px;
  padding-bottom: 8px;
  border-bottom: 2px solid #e2e8f0;
  display: flex;
  align-items: center;
  gap: 8px;
}

.resource-config-container {
  background: white;
  padding: 16px;
  border-radius: 6px;
  border: 1px solid #e2e8f0;
}

.resource-hint,
.config-hint {
  font-size: 12px;
  color: #64748b;
  margin-top: 4px;
}

.resource-preview {
  margin-top: 16px;
}

.full-width {
  width: 100%;
}

/* 动态表单 */
.dynamic-form-container {
  background: white;
  padding: 16px;
  border-radius: 6px;
  border: 1px solid #e2e8f0;
}

.volume-input-group,
.env-input-group {
  display: flex;
  align-items: center;
  gap: 8px;
}

.volume-form-item,
.env-form-item {
  margin-bottom: 16px;
}

.env-key-input,
.volume-host-input {
  flex: 1;
}

.env-separator,
.volume-separator {
  color: #64748b;
  font-weight: 600;
  font-size: 14px;
  min-width: 20px;
  text-align: center;
}

.env-value-input,
.volume-container-input {
  flex: 1.5;
}

.remove-btn {
  color: #ef4444;
  border: none;
  background: none;
  min-width: 32px;
  height: 32px;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.remove-btn:hover {
  background: #fef2f2;
  color: #dc2626;
}

.add-dynamic-button {
  border: 2px dashed #d1d5db;
  color: #6b7280;
  background: #f9fafb;
  height: 40px;
  border-radius: 6px;
  transition: all 0.2s ease;
}

.add-dynamic-button:hover {
  border-color: #3b82f6;
  color: #3b82f6;
  background: #eff6ff;
}

/* 镜像选项 */
.image-option {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
}

.image-name {
  flex: 1;
  font-size: 14px;
}

/* 详情模态框 */
.detail-modal :deep(.ant-modal-body) {
  padding: 0;
}

.notebook-detail-container {
  max-height: 70vh;
  overflow-y: auto;
}

.detail-tabs :deep(.ant-tabs-content-holder) {
  padding: 24px;
}

.detail-section {
  margin-bottom: 0;
}

/* 资源卡片 */
.resource-cards {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 20px;
}

.resource-card {
  background: white;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 20px;
  transition: all 0.2s ease;
}

.resource-card:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.resource-card-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}

.resource-card-icon {
  width: 40px;
  height: 40px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: bold;
  color: white;
}

.cpu-card {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.memory-card {
  background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
}

.gpu-card {
  background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
}

.resource-card-title {
  font-size: 16px;
  font-weight: 600;
  color: #1a202c;
}

.resource-metric {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.metric-label {
  color: #64748b;
  font-size: 14px;
}

.metric-value {
  color: #1a202c;
  font-size: 18px;
  font-weight: 600;
}

/* 配置详情 */
.config-section {
  margin-bottom: 24px;
  padding: 16px;
  background: #f8fafc;
  border-radius: 6px;
  border: 1px solid #e2e8f0;
}

.config-section h4 {
  font-size: 16px;
  font-weight: 600;
  color: #1a202c;
  margin: 0 0 12px 0;
  display: flex;
  align-items: center;
  gap: 8px;
}

.config-item {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.config-label {
  color: #64748b;
  font-weight: 500;
  min-width: 80px;
}

.config-value {
  color: #1a202c;
  font-weight: 500;
  font-family: 'Monaco', 'Menlo', monospace;
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
  padding: 12px;
  background: white;
  border-radius: 6px;
  border: 1px solid #e2e8f0;
  font-size: 12px;
  font-family: 'Monaco', 'Menlo', monospace;
}

.env-key,
.volume-host {
  color: #3b82f6;
  font-weight: 600;
}

.env-separator,
.volume-separator {
  color: #64748b;
  font-weight: 500;
}

.env-value,
.volume-container {
  color: #059669;
  font-weight: 500;
}

/* 日志样式 */
.logs-container {
  background: #1a202c;
  border-radius: 8px;
  overflow: hidden;
}

.logs-header {
  background: #2d3748;
  padding: 12px 16px;
  display: flex;
  gap: 8px;
  border-bottom: 1px solid #4a5568;
}

.logs-content {
  height: 400px;
  overflow-y: auto;
  background: #1a202c;
}

.logs-text {
  color: #e2e8f0;
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 12px;
  line-height: 1.5;
  padding: 16px;
  margin: 0;
  white-space: pre-wrap;
  word-wrap: break-word;
}

/* 导入样式 */
.import-container {
  padding: 20px 0;
}

/* 响应式设计 */
@media (max-width: 1200px) {
  .stats-container {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 768px) {
  .notebook-service-page {
    padding: 16px;
  }
  
  .page-title {
    font-size: 24px;
  }
  
  .stats-container {
    grid-template-columns: 1fr;
  }
  
  .custom-toolbar {
    flex-direction: column;
    align-items: stretch;
  }
  
  .search-filters {
    justify-content: stretch;
    flex-direction: column;
  }
  
  .search-input,
  .status-filter,
  .type-filter {
    width: 100%;
  }
  
  .action-buttons {
    justify-content: center;
  }
  
  .action-column {
    flex-direction: column;
    gap: 4px;
  }
  
  .resource-cards {
    grid-template-columns: 1fr;
  }
  
  .custom-modal :deep(.ant-modal-body) {
    max-height: 60vh;
  }
}

/* 动画效果 */
@keyframes fadeInUp {
  from {
    opacity: 0;
    transform: translateY(20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.dashboard-card,
.stat-card {
  animation: fadeInUp 0.3s ease;
}

/* 滚动条样式 */
.notebook-detail-container::-webkit-scrollbar,
.logs-content::-webkit-scrollbar {
  width: 6px;
}

.notebook-detail-container::-webkit-scrollbar-track,
.logs-content::-webkit-scrollbar-track {
  background: #f1f5f9;
}

.notebook-detail-container::-webkit-scrollbar-thumb,
.logs-content::-webkit-scrollbar-thumb {
  background: #cbd5e0;
  border-radius: 3px;
}

.notebook-detail-container::-webkit-scrollbar-thumb:hover,
.logs-content::-webkit-scrollbar-thumb:hover {
  background: #94a3b8;
}
</style>
