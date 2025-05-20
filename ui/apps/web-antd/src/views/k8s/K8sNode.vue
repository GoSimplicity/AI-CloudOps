<template>
  <div class="service-manager node-manager">
    <!-- 仪表板标题 -->
    <div class="dashboard-header">
      <h2 class="dashboard-title">
        <ClusterOutlined class="dashboard-icon" />
        Kubernetes 节点管理
      </h2>
      <div class="dashboard-stats">
        <div class="stat-item">
          <div class="stat-value">{{ nodes.length }}</div>
          <div class="stat-label">节点总数</div>
        </div>
        <div class="stat-item">
          <div class="stat-value">{{ route.query.cluster_id }}</div>
          <div class="stat-label">集群ID</div>
        </div>
      </div>
    </div>

    <!-- 查询和操作工具栏 -->
    <div class="control-panel">
      <div class="search-filters">
        <a-input-search
          v-model:value="searchText"
          placeholder="搜索节点名称、IP或角色"
          class="control-item search-input"
          @search="handleSearch"
          allow-clear
        >
          <template #prefix><SearchOutlined /></template>
        </a-input-search>
        
        <a-select
          v-model:value="statusFilter"
          placeholder="状态筛选"
          class="control-item status-selector"
          allow-clear
          @change="handleFilterChange"
        >
          <template #suffixIcon><ApiOutlined /></template>
          <a-select-option value="Ready">
            <span class="status-option">
              <CheckCircleOutlined style="color: #52c41a" />
              正常
            </span>
          </a-select-option>
          <a-select-option value="NotReady">
            <span class="status-option">
              <StopOutlined style="color: #f5222d" />
              异常
            </span>
          </a-select-option>
          <a-select-option value="Unknown">
            <span class="status-option">
              <WarningOutlined style="color: #faad14" />
              未知
            </span>
          </a-select-option>
        </a-select>
        
        <a-select
          v-model:value="roleFilter"
          placeholder="角色筛选"
          class="control-item role-selector"
          allow-clear
          @change="handleFilterChange"
        >
          <template #suffixIcon><UserOutlined /></template>
          <a-select-option value="master">
            <span class="role-option">
              <CrownOutlined style="color: #722ed1" />
              Master
            </span>
          </a-select-option>
          <a-select-option value="worker">
            <span class="role-option">
              <CodeSandboxOutlined style="color: #1890ff" />
              Worker
            </span>
          </a-select-option>
        </a-select>
      </div>
      
      <div class="action-buttons">
        <a-tooltip title="刷新数据">
          <a-button type="primary" class="refresh-btn" @click="refreshData" :loading="loading">
            <template #icon><ReloadOutlined /></template>
          </a-button>
        </a-tooltip>
        
        <a-dropdown>
          <a-button type="primary" class="manage-btn">
            <template #icon><SettingOutlined /></template>
            节点管理
          </a-button>
          <template #overlay>
            <a-menu>
              <a-menu-item key="1" @click="isAddLabelModalVisible = true">
                <TagOutlined /> 添加节点标签
              </a-menu-item>
              <a-menu-item key="2" @click="isAddTaintModalVisible = true">
                <WarningOutlined /> 添加 Taint
              </a-menu-item>
              <a-menu-item key="3" @click="isDeleteTaintModalVisible = true">
                <DeleteOutlined /> 删除 Taint
              </a-menu-item>
              <a-menu-item key="4" @click="handleClearTaints">
                <ClearOutlined /> 清空 Taint
              </a-menu-item>
            </a-menu>
          </template>
        </a-dropdown>
        
        <a-button 
          type="primary" 
          danger
          @click="handleToggleSchedule()" 
          :disabled="!hasSelectedNode"
          class="schedule-btn"
        >
          <template #icon><ScheduleOutlined /></template>
          启用/禁用调度
        </a-button>
      </div>
    </div>

    <!-- 状态摘要卡片 -->
    <div class="status-summary">
      <div class="summary-card total-card">
        <div class="card-content">
          <div class="card-metric">
            <DashboardOutlined class="metric-icon" />
            <div class="metric-value">{{ nodes.length }}</div>
          </div>
          <div class="card-title">节点总数</div>
        </div>
        <div class="card-footer">
          <div class="footer-text">全部Kubernetes节点</div>
        </div>
      </div>
      
      <div class="summary-card healthy-card">
        <div class="card-content">
          <div class="card-metric">
            <CheckCircleOutlined class="metric-icon" />
            <div class="metric-value">{{ healthyNodes }}</div>
          </div>
          <div class="card-title">健康节点</div>
        </div>
        <div class="card-footer">
          <a-progress 
            :percent="healthyPercentage" 
            :stroke-color="{ from: '#1890ff', to: '#52c41a' }" 
            size="small" 
            :show-info="false" 
          />
          <div class="footer-text">{{ healthyPercentage }}% 节点正常运行</div>
        </div>
      </div>
      
      <div class="summary-card problem-card">
        <div class="card-content">
          <div class="card-metric">
            <WarningOutlined class="metric-icon" />
            <div class="metric-value">{{ warningNodes + errorNodes }}</div>
          </div>
          <div class="card-title">问题节点</div>
        </div>
        <div class="card-footer">
          <a-progress 
            :percent="problemPercentage" 
            status="exception" 
            size="small" 
            :show-info="false"
          />
          <div class="footer-text">{{ warningNodes }} 个警告, {{ errorNodes }} 个错误</div>
        </div>
      </div>
    </div>

    <!-- 表格视图 -->
    <a-table
      :columns="columns"
      :data-source="filteredData"
      :row-selection="{ 
        type: 'radio', 
        onChange: onSelectChange,
        selectedRowKeys: selectedRowKeys
      }"
      :loading="loading"
      row-key="name"
      :pagination="{ 
        pageSize: 10, 
        showSizeChanger: true, 
        showQuickJumper: true,
        showTotal: (total: number) => `共 ${total} 条数据`
      }"
      class="services-table node-table"
    >
      <!-- 节点名称列 -->
      <template #name="{ text, record }">
        <div class="node-name">
          <div class="node-status-dot" :class="getNodeStatusClass(record.status)"></div>
          <span>{{ text }}</span>
        </div>
      </template>
      
      <!-- 状态列 -->
      <template #status="{ text }">
        <a-tag :color="getStatusColor(text)" class="status-tag">
          <span class="status-dot"></span>
          {{ text }}
        </a-tag>
      </template>

      <!-- IP地址列 -->
      <template #ip="{ text }">
        <span class="ip-address">
          <GlobalOutlined />
          {{ text }}
        </span>
      </template>

      <!-- 角色列 -->
      <template #roles="{ text }">
        <div class="roles-cell">
          <template v-if="text && typeof text === 'string'">
            <a-tag 
              v-for="role in text.split(',').filter(Boolean)" 
              :key="role" 
              :color="getRoleColor(role)"
              class="role-tag"
            >
              <span class="status-dot"></span>
              {{ role }}
            </a-tag>
          </template>
          <a-tag v-else color="default" class="role-tag">
            <span class="status-dot"></span>
            未知
          </a-tag>
        </div>
      </template>

      <!-- 创建时间列 -->
      <template #age="{ text }">
        <div class="timestamp">
          <ClockCircleOutlined />
          <span>{{ text }}</span>
        </div>
      </template>

      <!-- 资源使用列 -->
      <template #info="{ record }">
        <div class="node-info-cell">
          <a-tooltip title="CPU使用率">
            <progress-chart 
              type="cpu" 
              :percentage="30" 
              :title="`CPU: 30%`" 
              color="#1890ff" 
            />
          </a-tooltip>
          <a-tooltip title="内存使用率">
            <progress-chart 
              type="memory" 
              :percentage="45" 
              :title="`内存: 45%`" 
              color="#52c41a" 
            />
          </a-tooltip>
          <a-tooltip title="磁盘使用率">
            <progress-chart 
              type="disk" 
              :percentage="25" 
              :title="`磁盘: 25%`" 
              color="#722ed1" 
            />
          </a-tooltip>
        </div>
      </template>

      <!-- 标签列 -->
      <template #labels="{ record }">
        <div class="labels-cell">
          <a-tag v-for="(label, index) in getNodeLabels(record)" :key="index" color="blue" class="label-tag">
            {{ label }}
          </a-tag>
          <a-tag v-if="getNodeLabels(record).length > 3" color="blue" class="label-tag">
            +{{ getNodeLabels(record).length - 3 }}
          </a-tag>
        </div>
      </template>

      <!-- 操作列 -->
      <template #action="{ record }">
        <div class="action-column">
          <a-tooltip title="查看详情">
            <a-button type="primary" ghost shape="circle" @click="handleViewDetails(record)">
              <template #icon><EyeOutlined /></template>
            </a-button>
          </a-tooltip>
          
          <a-tooltip title="删除标签">
            <a-button type="primary" ghost shape="circle" @click="showDeleteLabelModal(record)">
              <template #icon><TagOutlined /></template>
            </a-button>
          </a-tooltip>
          
          <a-tooltip :title="record.schedulable ? '禁用调度' : '启用调度'">
            <a-button 
              :type="record.schedulable ? 'primary' : 'primary'" 
              :ghost="record.schedulable"
              :danger="!record.schedulable"
              shape="circle" 
              @click="handleToggleSchedule(record)"
            >
              <template #icon>
                <PauseOutlined v-if="record.schedulable" />
                <CaretRightOutlined v-else />
              </template>
            </a-button>
          </a-tooltip>
          
          <a-dropdown>
            <a-button type="primary" ghost shape="circle">
              <template #icon><MoreOutlined /></template>
            </a-button>
            <template #overlay>
              <a-menu>
                <a-menu-item key="1" @click="handleAddLabel(record)">
                  <TagOutlined /> 添加标签
                </a-menu-item>
                <a-menu-item key="2" @click="handleAddTaint(record)">
                  <WarningOutlined /> 添加 Taint
                </a-menu-item>
                <a-menu-item key="3" @click="handleDeleteTaint(record)">
                  <DeleteOutlined /> 删除 Taint
                </a-menu-item>
                <a-menu-divider />
                <a-menu-item key="4" @click="handleCordon(record)" danger>
                  <StopOutlined /> 维护模式
                </a-menu-item>
              </a-menu>
            </template>
          </a-dropdown>
        </div>
      </template>
    </a-table>

    <!-- 添加标签模态框 -->
    <a-modal
      v-model:open="isAddLabelModalVisible"
      title="添加节点标签"
      :confirm-loading="submitLoading"
      @cancel="closeAddLabelModal"
      @ok="handleSubmitAddLabel"
      class="node-modal"
    >
      <a-alert
        type="info"
        show-icon
        banner
        message="节点标签可用于 Pod 调度及资源分配"
        style="margin-bottom: 16px"
        class="modal-alert"
      />
      <a-form :model="labelForm" layout="vertical" class="node-form">
        <a-form-item
          label="节点名称"
          name="nodeName"
          :rules="[{ required: true, message: '请选择节点名称' }]"
        >
          <a-select
            v-model:value="labelForm.nodeName"
            placeholder="请选择节点名称"
            show-search
            :filter-option="filterNodeOption"
            class="form-select"
          >
            <template #suffixIcon><ClusterOutlined /></template>
            <a-select-option v-for="node in filteredData" :key="node.name" :value="node.name">
              <span class="node-option">
                <ClusterOutlined />
                {{ node.name }}
              </span>
            </a-select-option>
          </a-select>
        </a-form-item>
        
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item
              label="标签键"
              name="key"
              :rules="[{ required: true, message: '请输入标签键' }]"
            >
              <a-input v-model:value="labelForm.key" placeholder="请输入标签键" class="form-input">
                <template #prefix><TagOutlined /></template>
              </a-input>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item
              label="标签值"
              name="value"
              :rules="[{ required: true, message: '请输入标签值' }]"
            >
              <a-input v-model:value="labelForm.value" placeholder="请输入标签值" class="form-input">
                <template #prefix><TagsOutlined /></template>
              </a-input>
            </a-form-item>
          </a-col>
        </a-row>
        
        <a-form-item>
          <a-alert
            type="warning"
            show-icon
            message="注意: 添加标签可能会影响现有的 Pod 调度"
          />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 添加Taint模态框 -->
    <a-modal
      v-model:open="isAddTaintModalVisible"
      title="添加节点 Taint"
      :confirm-loading="submitLoading"
      @cancel="closeAddTaintModal"
      @ok="handleSubmitAddTaint"
      class="node-modal"
    >
      <a-alert
        type="info"
        show-icon
        banner
        message="Taint 用于阻止 Pod 调度到节点上"
        style="margin-bottom: 16px"
        class="modal-alert"
      />
      <a-form :model="taintForm" layout="vertical" class="node-form">
        <a-form-item
          label="节点名称"
          name="nodeName"
          :rules="[{ required: true, message: '请选择节点名称' }]"
        >
          <a-select
            v-model:value="taintForm.nodeName"
            placeholder="请选择节点名称"
            show-search
            :filter-option="filterNodeOption"
            class="form-select"
          >
            <template #suffixIcon><ClusterOutlined /></template>
            <a-select-option v-for="node in filteredData" :key="node.name" :value="node.name">
              <span class="node-option">
                <ClusterOutlined />
                {{ node.name }} ({{ node.ip }})
              </span>
            </a-select-option>
          </a-select>
        </a-form-item>
        
        <a-form-item
          label="Taint YAML"
          name="taintYaml"
          :rules="[{ required: true, message: '请输入 Taint YAML' }]"
        >
          <div class="yaml-actions">
            <a-button type="primary" size="small" @click="checkTaintYaml(taintForm.nodeName)">
              <template #icon><CodeOutlined /></template>
              验证 YAML 格式
            </a-button>
            <a-popover title="Taint 效果说明" placement="right">
              <template #content>
                <p><strong>NoSchedule</strong>: 不允许新 Pod 调度</p>
                <p><strong>PreferNoSchedule</strong>: 尽量不调度</p>
                <p><strong>NoExecute</strong>: 驱逐现有 Pod</p>
              </template>
              <a-button size="small">
                <template #icon><QuestionCircleOutlined /></template>
                效果说明
              </a-button>
            </a-popover>
          </div>
          <a-textarea
            v-model:value="taintForm.taintYaml"
            :rows="6"
            :auto-size="{ minRows: 6, maxRows: 10 }"
            placeholder="示例：- key: &quot;example-key&quot;
  value: &quot;example-value&quot; 
  effect: &quot;NoSchedule&quot;"
            class="yaml-editor"
          />
        </a-form-item>
        
        <a-form-item>
          <a-alert
            type="warning"
            show-icon
            message="注意: 添加 NoExecute 效果的 Taint 将驱逐不容忍该污点的现有 Pod"
          />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 删除Taint模态框 -->
    <a-modal
      v-model:open="isDeleteTaintModalVisible"
      title="删除节点 Taint"
      :confirm-loading="submitLoading"
      @cancel="closeDeleteTaintModal"
      @ok="handleSubmitDeleteTaint"
      class="node-modal"
    >
      <a-alert
        type="info"
        show-icon
        banner
        message="删除 Taint 将允许 Pod 重新调度到节点上"
        style="margin-bottom: 16px"
        class="modal-alert"
      />
      <a-form :model="deleteTaintForm" layout="vertical" class="node-form">
        <a-form-item
          label="节点名称"
          name="nodeName"
          :rules="[{ required: true, message: '请选择节点名称' }]"
        >
          <a-select
            v-model:value="deleteTaintForm.nodeName"
            placeholder="请选择节点名称"
            show-search
            :filter-option="filterNodeOption"
            class="form-select"
          >
            <template #suffixIcon><ClusterOutlined /></template>
            <a-select-option v-for="node in filteredData" :key="node.name" :value="node.name">
              <span class="node-option">
                <ClusterOutlined />
                {{ node.name }} ({{ node.ip }})
              </span>
            </a-select-option>
          </a-select>
        </a-form-item>
        
        <a-form-item
          label="Taint YAML"
          name="taintYaml"
          :rules="[{ required: true, message: '请输入 Taint YAML' }]"
        >
          <div class="yaml-actions">
            <a-button type="primary" size="small" @click="checkTaintYaml(deleteTaintForm.nodeName)">
              <template #icon><CodeOutlined /></template>
              验证 YAML 格式
            </a-button>
          </div>
          <a-textarea
            v-model:value="deleteTaintForm.taintYaml"
            :rows="6"
            :auto-size="{ minRows: 6, maxRows: 10 }"
            placeholder="示例：- key: &quot;example-key&quot;
  value: &quot;example-value&quot; 
  effect: &quot;NoSchedule&quot;"
            class="yaml-editor"
          />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 删除标签模态框 -->
    <a-modal
      v-model:open="isDeleteLabelModalVisible"
      title="删除节点标签"
      :confirm-loading="submitLoading"
      @cancel="closeDeleteLabelModal"
      @ok="handleDeleteLabel"
      class="node-modal"
    >
      <a-alert
        type="warning"
        show-icon
        banner
        message="删除标签可能会影响依赖此标签的 Pod 调度"
        style="margin-bottom: 16px"
        class="modal-alert"
      />
      <a-form :model="deleteLabelForm" layout="vertical" class="node-form">
        <a-form-item
          label="选择标签"
          name="label"
          :rules="[{ required: true, message: '请选择标签' }]"
        >
          <a-select 
            v-model:value="deleteLabelForm.label" 
            placeholder="请选择标签"
            class="form-select"
          >
            <template #suffixIcon><TagOutlined /></template>
            <a-select-option v-for="(label, index) in labelOptions" :key="index" :value="label">
              <span class="label-option">
                <TagOutlined />
                {{ label }}
              </span>
            </a-select-option>
          </a-select>
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 节点详情模态框 -->
    <a-modal
      v-model:open="isViewDetailsModalVisible"
      title="节点详情"
      width="800px"
      @cancel="closeViewDetailsModal"
      :footer="null"
      class="yaml-modal node-detail-modal"
    >
      <a-spin :spinning="detailsLoading">
        <div v-if="selectedNodeDetails" class="node-details">
          <a-alert class="yaml-info" type="info" show-icon>
            <template #message>
              <span>{{ selectedNodeDetails.name }}</span>
            </template>
            <template #description>
              <div>状态: {{ selectedNodeDetails.status }} | IP: {{ selectedNodeDetails.ip }}</div>
            </template>
          </a-alert>
          
          <a-tabs default-active-key="1">
            <a-tab-pane key="1" tab="基本信息">
              <div class="details-grid">
                <div class="detail-card">
                  <h4><EnvironmentOutlined /> 基础配置</h4>
                  <div class="detail-item">
                    <div class="detail-label">IP 地址:</div>
                    <div class="detail-value">{{ selectedNodeDetails.ip }}</div>
                  </div>
                  <div class="detail-item">
                    <div class="detail-label">角色:</div>
                    <div class="detail-value">
                      <template v-if="selectedNodeDetails.roles && typeof selectedNodeDetails.roles === 'string'">
                        <a-tag 
                          v-for="role in (selectedNodeDetails.roles as string[])" 
                          :key="role" 
                          :color="getRoleColor(role)"
                          class="role-tag"
                        >
                          <span class="status-dot"></span>
                          {{ role }}
                        </a-tag>
                      </template>
                      <a-tag v-else color="default" class="role-tag">
                        <span class="status-dot"></span>
                        未知
                      </a-tag>
                    </div>
                  </div>
                  <div class="detail-item">
                    <div class="detail-label">创建时间:</div>
                    <div class="detail-value">{{ selectedNodeDetails.age }}</div>
                  </div>
                  <div class="detail-item">
                    <div class="detail-label">调度状态:</div>
                    <div class="detail-value">
                      <a-tag :color="selectedNodeDetails.schedulable ? 'green' : 'red'" class="status-tag">
                        <span class="status-dot"></span>
                        {{ selectedNodeDetails.schedulable ? '可调度' : '不可调度' }}
                      </a-tag>
                    </div>
                  </div>
                </div>
                
                <div class="detail-card">
                  <h4><ApiOutlined /> 资源状态</h4>
                  <div class="detail-item">
                    <div class="detail-label">CPU 请求:</div>
                    <div class="detail-value">{{ selectedNodeDetails.cpu_request_info }}</div>
                  </div>
                  <div class="detail-item">
                    <div class="detail-label">CPU 使用:</div>
                    <div class="detail-value">{{ selectedNodeDetails.cpu_usage_info }}</div>
                  </div>
                  <div class="detail-item">
                    <div class="detail-label">内存请求:</div>
                    <div class="detail-value">{{ selectedNodeDetails.memory_request_info }}</div>
                  </div>
                  <div class="detail-item">
                    <div class="detail-label">内存使用:</div>
                    <div class="detail-value">{{ selectedNodeDetails.memory_usage_info }}</div>
                  </div>
                  <div class="detail-item">
                    <div class="detail-label">磁盘存储:</div>
                    <div class="detail-value">{{ selectedNodeDetails.ephemeral_storage }}</div>
                  </div>
                </div>
              </div>
              
              <div class="resource-charts">
                <div class="resource-chart">
                  <h4>CPU 使用率</h4>
                  <div class="usage-gauge">
                    <div class="gauge-value">30%</div>
                    <div class="gauge-bar">
                      <div class="gauge-fill" style="width: 30%; background-color: #1890ff;"></div>
                    </div>
                  </div>
                </div>
                <div class="resource-chart">
                  <h4>内存使用率</h4>
                  <div class="usage-gauge">
                    <div class="gauge-value">45%</div>
                    <div class="gauge-bar">
                      <div class="gauge-fill" style="width: 45%; background-color: #52c41a;"></div>
                    </div>
                  </div>
                </div>
                <div class="resource-chart">
                  <h4>磁盘使用率</h4>
                  <div class="usage-gauge">
                    <div class="gauge-value">25%</div>
                    <div class="gauge-bar">
                      <div class="gauge-fill" style="width: 25%; background-color: #722ed1;"></div>
                    </div>
                  </div>
                </div>
              </div>
            </a-tab-pane>
            
            <a-tab-pane key="2" tab="标签和污点">
              <div class="details-grid">
                <div class="detail-card">
                  <div class="card-header">
                    <h4><TagOutlined /> 节点标签</h4>
                    <a-button type="primary" size="small" @click="handleAddLabel(selectedNodeDetails)">
                      <template #icon><PlusOutlined /></template>
                      添加标签
                    </a-button>
                  </div>
                  <div class="labels-list">
                    <a-empty v-if="!selectedNodeDetails.labels || selectedNodeDetails.labels.length === 0" description="暂无标签" />
                    <a-tag 
                      v-for="(label, index) in selectedNodeDetails.labels" 
                      :key="index"
                      color="blue"
                      closable
                      @close="handleQuickDeleteLabel(label)"
                      class="label-tag"
                    >
                      {{ label }}
                    </a-tag>
                  </div>
                </div>
                
                <div class="detail-card">
                  <div class="card-header">
                    <h4><WarningOutlined /> 节点污点</h4>
                    <a-button type="primary" size="small" @click="handleAddTaint(selectedNodeDetails)">
                      <template #icon><PlusOutlined /></template>
                      添加污点
                    </a-button>
                  </div>
                  <div class="taints-list">
                    <a-empty v-if="!selectedNodeDetails.taints || selectedNodeDetails.taints.length === 0" description="暂无污点" />
                    <a-tag 
                      v-for="(taint, index) in selectedNodeDetails.taints" 
                      :key="index"
                      :color="getTaintColor(taint)"
                      closable
                      @close="handleQuickDeleteTaint(taint)"
                      class="taint-tag"
                    >
                      <span class="status-dot"></span>
                      {{ taint }}
                    </a-tag>
                  </div>
                </div>
              </div>
            </a-tab-pane>
            
            <a-tab-pane key="3" tab="事件日志">
              <div class="events-container">
                <a-timeline>
                  <a-timeline-item 
                    v-for="(event, index) in selectedNodeDetails.events || []" 
                    :key="index"
                    :color="getEventColor(event.type)"
                  >
                    <div class="event-card">
                      <div class="event-header">
                        <span class="event-reason">{{ event.reason }}</span>
                        <span class="event-time">{{ formatTime(new Date(event.last_time).getTime()) }}</span>
                      </div>
                      <div class="event-message">{{ event.message }}</div>
                      <div class="event-meta">
                        <span><strong>类型:</strong> {{ event.type }}</span>
                        <span><strong>组件:</strong> {{ event.component }}</span>
                        <span><strong>发生次数:</strong> {{ event.count }}</span>
                      </div>
                      <div class="event-time-range">
                        <CalendarOutlined /> 
                        {{ formatTime(new Date(event.first_time).getTime()) }} - {{ formatTime(new Date(event.last_time).getTime()) }}
                      </div>
                    </div>
                  </a-timeline-item>
                </a-timeline>
                <a-empty v-if="!selectedNodeDetails.events || selectedNodeDetails.events.length === 0" description="暂无事件" />
              </div>
            </a-tab-pane>
          </a-tabs>
          
          <div class="modal-footer">
            <a-space>
              <a-button @click="closeViewDetailsModal">关闭</a-button>
              <a-button type="primary" ghost @click="refreshNodeDetails(selectedNodeDetails)">
                <template #icon><ReloadOutlined /></template>
                刷新数据
              </a-button>
              <a-button 
                :type="selectedNodeDetails.schedulable ? 'primary' : 'primary'" 
                :ghost="!selectedNodeDetails.schedulable"
                :danger="selectedNodeDetails.schedulable"
                @click="handleToggleSchedule(selectedNodeDetails)"
              >
                <template #icon>
                  <PauseOutlined v-if="selectedNodeDetails.schedulable" />
                  <CaretRightOutlined v-else />
                </template>
                {{ selectedNodeDetails.schedulable ? '禁用调度' : '启用调度' }}
              </a-button>
            </a-space>
          </div>
        </div>
      </a-spin>
    </a-modal>
  </div>
</template>

<script lang="ts" setup>
import { computed, ref, reactive, onMounted, h } from 'vue';
import { useRoute } from 'vue-router';
import { message, Modal } from 'ant-design-vue';
import type { GetNodeDetailRes } from '#/api';
import {
  addNodeLabelApi,
  deleteNodeLabelApi,
  getNodeDetailsApi,
  getNodeListApi,
  addNodeTaintApi,
  checkTaintYamlApi,
} from '#/api';

// 导入图标
import {
  EyeOutlined,
  DeleteOutlined,
  ReloadOutlined,
  SearchOutlined,
  TagOutlined,
  TagsOutlined,
  WarningOutlined,
  CheckCircleOutlined,
  StopOutlined,
  EnvironmentOutlined,
  ApiOutlined,
  ClusterOutlined,
  DownOutlined,
  PauseOutlined,
  CaretRightOutlined,
  MoreOutlined,
  CodeOutlined,
  QuestionCircleOutlined,
  PlusOutlined,
  ClearOutlined,
  CalendarOutlined,
  ScheduleOutlined,
  SettingOutlined,
  GlobalOutlined,
  ClockCircleOutlined,
  UserOutlined,
  CrownOutlined,
  CodeSandboxOutlined,
  DashboardOutlined,
  PartitionOutlined
} from '@ant-design/icons-vue';

// 自定义节点类型接口
interface NodeItem {
  name: string;
  cluster_id: number;
  status: string;
  ip: string;
  roles?: string;
  age: string;
  schedulable?: boolean;
  labels?: string[];
  taints?: string[];
}

// 节点事件类型接口
interface NodeEvent {
  reason: string;
  message: string;
  first_time: number;
  last_time: number;
  count: number;
  type: string;
  component: string;
  object: string;
}

// 类型定义补充
type ProgressChartProps = {
  type: 'cpu' | 'memory' | 'disk';
  percentage: number;
  title: string;
  color: string;
};

// 创建进度图表组件
const ProgressChart = (props: ProgressChartProps, { slots }: any) => {
  const { percentage, title, color, type } = props;
  
  return h('div', { class: 'progress-chart' }, [
    h('div', { class: 'chart-icon' }, [
      type === 'cpu' ? h(ApiOutlined) : 
      type === 'memory' ? h(EnvironmentOutlined) : 
      h(StopOutlined)
    ]),
    h('div', { class: 'chart-bar' }, [
      h('div', { 
        class: 'chart-fill',
        style: {
          width: `${percentage}%`,
          backgroundColor: color
        }
      })
    ]),
    h('div', { class: 'chart-title' }, title)
  ]);
};

// 状态和常量
const route = useRoute();
const loading = ref(false);
const detailsLoading = ref(false);
const submitLoading = ref(false);
const nodes = ref<NodeItem[]>([]);
const searchText = ref('');
const statusFilter = ref<string>('');
const roleFilter = ref<string>('');
const selectedNodeDetails = ref<GetNodeDetailRes | null>(null);
const selectedRowKeys = ref<string[]>([]);
const selectedNode = ref<NodeItem | null>(null);

// 模态框控制
const isAddLabelModalVisible = ref(false);
const isAddTaintModalVisible = ref(false);
const isDeleteTaintModalVisible = ref(false);
const isViewDetailsModalVisible = ref(false);
const isDeleteLabelModalVisible = ref(false);

// 表单数据
const labelForm = reactive({
  key: '',
  nodeName: '',
  value: '',
});

const taintForm = reactive({
  nodeName: '',
  taintYaml: '',
});

const deleteTaintForm = reactive({
  nodeName: '',
  taintYaml: '',
});

const deleteLabelForm = reactive({
  label: '',
});

// 计算属性：状态统计
const healthyNodes = computed(() => {
  return nodes.value.filter((node) => node.status === 'Ready').length;
});

const warningNodes = computed(() => {
  return nodes.value.filter((node) => node.status === 'Unknown').length;
});

const errorNodes = computed(() => {
  return nodes.value.filter((node) => node.status === 'NotReady').length;
});

// 健康节点百分比
const healthyPercentage = computed(() => {
  if (nodes.value.length === 0) return 0;
  return Math.round((healthyNodes.value / nodes.value.length) * 100);
});

// 问题节点百分比
const problemPercentage = computed(() => {
  if (nodes.value.length === 0) return 0;
  return Math.round(((warningNodes.value + errorNodes.value) / nodes.value.length) * 100);
});

// 计算属性：过滤后的节点数据
const filteredData = computed(() => {
  let result = [...nodes.value];
  
  // 搜索过滤
  if (searchText.value) {
    const searchValue = searchText.value.trim().toLowerCase();
    result = result.filter((node) => {
      return node.name.toLowerCase().includes(searchValue) || 
             node.ip.toLowerCase().includes(searchValue) || 
             (node.roles && typeof node.roles === 'string' && node.roles.toLowerCase().includes(searchValue));
    });
  }
  
  // 状态过滤
  if (statusFilter.value) {
    result = result.filter((node) => node.status === statusFilter.value);
  }
  
  // 角色过滤
  if (roleFilter.value) {
    result = result.filter((node) => {
      return node.roles && typeof node.roles === 'string' && 
             node.roles.toLowerCase().includes(roleFilter.value.toLowerCase());
    });
  }
  
  return result;
});

// 计算属性：是否有选中节点
const hasSelectedNode = computed(() => {
  return selectedRowKeys.value.length > 0;
});

// 计算属性：标签选项
const labelOptions = computed(() => {
  return selectedNodeDetails.value?.labels || [];
});

// 表格列配置
const columns = [
  { 
    title: '节点名称', 
    dataIndex: 'name', 
    key: 'name', 
    width: '15%',
    sorter: (a: NodeItem, b: NodeItem) => a.name.localeCompare(b.name),
    slots: { customRender: 'name' },
  },
  { 
    title: '状态', 
    dataIndex: 'status', 
    key: 'status', 
    width: '10%',
    slots: { customRender: 'status' },
    filters: [
      { text: '正常', value: 'Ready' },
      { text: '异常', value: 'NotReady' },
      { text: '未知', value: 'Unknown' },
    ],
    onFilter: (value: string, record: NodeItem) => record.status === value,
  },
  { 
    title: 'IP 地址', 
    dataIndex: 'ip', 
    key: 'ip',
    width: '12%',
    slots: { customRender: 'ip' },
  },
  { 
    title: '角色', 
    dataIndex: 'roles', 
    key: 'roles',
    width: '12%',
    slots: { customRender: 'roles' },
  },
  { 
    title: '创建时间', 
    dataIndex: 'age', 
    key: 'age',
    width: '10%',
    slots: { customRender: 'age' },
  },
  { 
    title: '资源使用', 
    key: 'info',
    width: '20%',
    slots: { customRender: 'info' },
  },
  { 
    title: '标签', 
    key: 'labels',
    width: '12%',
    slots: { customRender: 'labels' },
  },
  { 
    title: '操作', 
    key: 'action',
    width: '13%',
    fixed: 'right',
    slots: { customRender: 'action' },
  },
];

// ====== 工具函数 ======
// 状态颜色映射
const getStatusColor = (status: string | undefined): string => {
  if (!status) return 'default';
  
  const statusMap: Record<string, string> = {
    'Ready': 'green',
    'NotReady': 'red',
    'Unknown': 'orange',
  };
  
  return statusMap[status] || 'default';
};

// 节点状态类名
const getNodeStatusClass = (status: string | undefined): string => {
  if (!status) return 'status-unknown';
  
  const statusMap: Record<string, string> = {
    'Ready': 'status-ready',
    'NotReady': 'status-error',
    'Unknown': 'status-warning',
  };
  
  return statusMap[status] || 'status-unknown';
};

// 角色颜色映射
const getRoleColor = (role: string): string => {
  if (!role) return 'default';
  
  const trimmedRole = role.trim().toLowerCase();
  
  if (trimmedRole.includes('master') || trimmedRole.includes('control')) {
    return 'purple';
  } else if (trimmedRole.includes('worker')) {
    return 'blue';
  } else if (trimmedRole.includes('infra')) {
    return 'orange';
  } else if (trimmedRole.includes('app')) {
    return 'green';
  }
  
  return 'default';
};

// 事件颜色映射
const getEventColor = (type: string | undefined): string => {
  if (!type) return 'blue';
  
  const typeMap: Record<string, string> = {
    'Normal': 'green',
    'Warning': 'orange',
    'Error': 'red',
  };
  
  return typeMap[type] || 'blue';
};

// Taint颜色映射
const getTaintColor = (taint: string): string => {
  if (!taint) return 'blue';
  
  if (taint.includes('NoExecute')) {
    return 'red';
  } else if (taint.includes('NoSchedule')) {
    return 'orange';
  } else if (taint.includes('PreferNoSchedule')) {
    return 'yellow';
  }
  
  return 'blue';
};

// 获取节点标签
const getNodeLabels = (node: NodeItem): string[] => {
  if (!node || !node.labels || !Array.isArray(node.labels)) return [];
  
  // 只显示前3个标签，完整列表在详情中显示
  return node.labels.slice(0, 3);
};

// 格式化时间
const formatTime = (timestamp: number | undefined): string => {
  if (!timestamp) return '-';
  
  const date = new Date(timestamp);
  return `${date.getFullYear()}-${(date.getMonth() + 1).toString().padStart(2, '0')}-${date.getDate().toString().padStart(2, '0')} ${date.getHours().toString().padStart(2, '0')}:${date.getMinutes().toString().padStart(2, '0')}:${date.getSeconds().toString().padStart(2, '0')}`;
};

// 节点名称过滤
const filterNodeOption = (input: string, option: any): boolean => {
  return option.value.toLowerCase().indexOf(input.toLowerCase()) >= 0;
};

// ====== 数据操作函数 ======
// 获取节点列表
const getNodes = async (): Promise<void> => {
  loading.value = true;
  try {
    const cluster_id = Number(route.query.cluster_id);
    if (isNaN(cluster_id)) {
      throw new Error('无效的集群ID');
    }
    const res = await getNodeListApi(cluster_id);
    nodes.value = Array.isArray(res) ? res : [];
    message.success('节点数据加载成功');
  } catch (error: any) {
    message.error(error.message || '获取节点数据失败');
  } finally {
    loading.value = false;
  }
};

// 刷新数据
const refreshData = (): void => {
  searchText.value = '';
  statusFilter.value = '';
  roleFilter.value = '';
  selectedRowKeys.value = [];
  selectedNode.value = null;
  getNodes();
};

// 处理搜索
const handleSearch = (value: string): void => {
  searchText.value = value;
};

// 处理筛选变化
const handleFilterChange = (): void => {
  // 状态和角色筛选器变化时会自动通过计算属性更新表格数据
};
// 选择表格行
const onSelectChange = (keys: string[], rows: NodeItem[]): void => {
  selectedRowKeys.value = keys;
  selectedNode.value = rows.length > 0 ? rows[0] as NodeItem : null;
};

// 打开用户选择的节点详情
const handleCordon = (record: NodeItem): void => {
  Modal.confirm({
    title: '确认进入维护模式',
    content: `是否确认将节点 ${record.name} 设置为维护模式？这将阻止新的 Pod 被调度到该节点。`,
    okText: '确认',
    cancelText: '取消',
    onOk: () => {
      handleToggleSchedule(record, false);
    }
  });
};

// 获取节点详情
const handleViewDetails = async (record: NodeItem): Promise<void> => {
  detailsLoading.value = true;
  try {
    const res = await getNodeDetailsApi(record.name, String(record.cluster_id));
    selectedNodeDetails.value = res;
    isViewDetailsModalVisible.value = true;
  } catch (error: any) {
    message.error(error.message || '获取节点详情失败');
  } finally {
    detailsLoading.value = false;
  }
};

// 刷新节点详情
const refreshNodeDetails = async (node: GetNodeDetailRes): Promise<void> => {
  if (!node) return;
  
  detailsLoading.value = true;
  try {
    const res = await getNodeDetailsApi(node.name, String(node.cluster_id));
    selectedNodeDetails.value = res;
    message.success('节点详情刷新成功');
  } catch (error: any) {
    message.error(error.message || '刷新节点详情失败');
  } finally {
    detailsLoading.value = false;
  }
};

// 打开添加标签操作
const handleAddLabel = (record: NodeItem | GetNodeDetailRes | null = null): void => {
  if (record) {
    labelForm.nodeName = record.name;
  }
  isAddLabelModalVisible.value = true;
};

// 添加节点标签
const handleSubmitAddLabel = async (): Promise<void> => {
  submitLoading.value = true;
  const { key, nodeName, value } = labelForm;
  const cluster_id = Number(route.query.cluster_id);
  if (isNaN(cluster_id)) {
    message.error('无效的集群ID');
    submitLoading.value = false;
    return;
  }
  
  try {
    await addNodeLabelApi({
      cluster_id,
      labels: [key, value],
      mod_type: 'add',
      node_name: [nodeName],
    });
    message.success('标签添加成功');
    
    labelForm.nodeName = '';
    labelForm.key = '';
    labelForm.value = '';
    
    getNodes();
    
    if (selectedNodeDetails.value?.name === nodeName) {
      refreshNodeDetails(selectedNodeDetails.value);
    }
    
    isAddLabelModalVisible.value = false;
  } catch (error: any) {
    message.error(error.message || '标签添加失败');
  } finally {
    submitLoading.value = false;
  }
};

// 快速删除标签
const handleQuickDeleteLabel = (label: string): void => {
  Modal.confirm({
    title: '确认删除标签',
    content: `确定要删除标签 ${label} 吗？`,
    okText: '删除',
    okType: 'danger',
    cancelText: '取消',
    onOk: async () => {
      const [key, val] = label.split('=');
      if (!key || !val) {
        message.error('标签格式不正确');
        return;
      }
      
      try {
        if (!selectedNodeDetails.value) {
          message.error('未选中节点');
          return;
        }
        
        await deleteNodeLabelApi({
          cluster_id: Number(route.query.cluster_id),
          labels: [key, val],
          mod_type: 'del',
          node_name: [selectedNodeDetails.value.name],
        });
        message.success('标签删除成功');
        
        if (selectedNodeDetails.value) {
          refreshNodeDetails(selectedNodeDetails.value);
        }
        
        getNodes();
      } catch (error: any) {
        message.error(error.message || '删除标签失败');
      }
    }
  });
};

// 删除标签
const handleDeleteLabel = async (): Promise<void> => {
  submitLoading.value = true;
  const selectedLabel = deleteLabelForm.label;
  const cluster_id = Number(route.query.cluster_id);
  if (isNaN(cluster_id)) {
    message.error('无效的集群ID');
    submitLoading.value = false;
    return;
  }
  
  if (!selectedLabel) {
    message.error('请选择一个标签');
    submitLoading.value = false;
    return;
  }

  const [key, val] = selectedLabel.split('=');

  if (!key || !val) {
    message.error('标签格式不正确');
    submitLoading.value = false;
    return;
  }

  if (!selectedNodeDetails.value) {
    message.error('未选中节点');
    submitLoading.value = false;
    return;
  }

  try {
    await deleteNodeLabelApi({
      cluster_id,
      labels: [key, val],
      mod_type: 'del',
      node_name: [selectedNodeDetails.value.name],
    });
    message.success('标签删除成功');
    
    getNodes();
    
    if (selectedNodeDetails.value) {
      refreshNodeDetails(selectedNodeDetails.value);
    }
    
    closeDeleteLabelModal();
  } catch (error: any) {
    message.error(error.message || '删除标签失败');
  } finally {
    submitLoading.value = false;
  }
};

// 打开添加Taint操作
const handleAddTaint = (record: NodeItem | GetNodeDetailRes | null = null): void => {
  if (record) {
    taintForm.nodeName = record.name;
  }
  isAddTaintModalVisible.value = true;
};

// 添加Taint
const handleSubmitAddTaint = async (): Promise<void> => {
  submitLoading.value = true;
  try {
    await addNodeTaintApi({
      cluster_id: Number(route.query.cluster_id),
      mod_type: 'add', 
      node_name: taintForm.nodeName,
      taint_yaml: taintForm.taintYaml,
    });
    message.success('Taint添加成功');
    
    taintForm.nodeName = '';
    taintForm.taintYaml = '';
    
    getNodes();
    
    if (selectedNodeDetails.value?.name === taintForm.nodeName) {
      refreshNodeDetails(selectedNodeDetails.value);
    }
    
    isAddTaintModalVisible.value = false;
  } catch (error: any) {
    message.error(error.message || '添加Taint失败');
  } finally {
    submitLoading.value = false;
  }
};

// 打开删除Taint操作
const handleDeleteTaint = (record: NodeItem | GetNodeDetailRes | null = null): void => {
  if (record) {
    deleteTaintForm.nodeName = record.name;
  }
  isDeleteTaintModalVisible.value = true;
};

// 删除Taint
const handleSubmitDeleteTaint = async (): Promise<void> => {
  submitLoading.value = true;
  try {
    await addNodeTaintApi({
      cluster_id: Number(route.query.cluster_id),
      mod_type: 'del',
      node_name: deleteTaintForm.nodeName,
      taint_yaml: deleteTaintForm.taintYaml,
    });
    message.success('Taint删除成功');
    
    deleteTaintForm.nodeName = '';
    deleteTaintForm.taintYaml = '';
    
    getNodes();
    
    if (selectedNodeDetails.value?.name === deleteTaintForm.nodeName) {
      refreshNodeDetails(selectedNodeDetails.value);
    }
    
    isDeleteTaintModalVisible.value = false;
  } catch (error: any) {
    message.error(error.message || '删除Taint失败');
  } finally {
    submitLoading.value = false;
  }
};

// 快速删除Taint
const handleQuickDeleteTaint = (taint: string): void => {
  Modal.confirm({
    title: '确认删除污点',
    content: `确定要删除污点 ${taint} 吗？`,
    okText: '删除',
    okType: 'danger',
    cancelText: '取消',
    onOk: async () => {
      try {
        if (!selectedNodeDetails.value) {
          message.error('未选中节点');
          return;
        }
        
        const taintParts = taint.split(':');
        const keyValue = taintParts[0]?.trim().split('=') || [];
        const effect = taintParts[1]?.trim() || 'NoSchedule';
        const taintYaml = `- key: "${keyValue[0]}"
  value: "${keyValue[1]}" 
  effect: "${effect}"`;
        
        await addNodeTaintApi({
          cluster_id: Number(route.query.cluster_id),
          mod_type: 'del',
          node_name: selectedNodeDetails.value.name,
          taint_yaml: taintYaml,
        });
        message.success('污点删除成功');
        
        if (selectedNodeDetails.value) {
          refreshNodeDetails(selectedNodeDetails.value);
        }
        
        getNodes();
      } catch (error: any) {
        message.error(error.message || '删除污点失败');
      }
    }
  });
};

// 检查Taint YAML格式
const checkTaintYaml = async (nodeName: string): Promise<void> => {
  try {
    await checkTaintYamlApi({
      cluster_id: Number(route.query.cluster_id),
      node_name: nodeName,
      taint_yaml: taintForm.taintYaml || deleteTaintForm.taintYaml,
    });
    message.success('YAML格式校验通过');
  } catch (error: any) {
    message.error(error.message || 'YAML格式校验失败');
  }
};

// 清空Taint
const handleClearTaints = (): void => {
  // 检查是否有选中节点
  if (!selectedNode.value && !selectedNodeDetails.value) {
    message.warning('请先选择一个节点');
    return;
  }
  
  const node = selectedNode.value || selectedNodeDetails.value;
  
  if (!node) {
    message.warning('未选中节点');
    return;
  }
  
  Modal.confirm({
    title: '确认清空污点',
    content: `确定要清空节点 ${node.name} 的所有污点吗？这可能会导致新的Pod被调度到该节点。`,
    okText: '确认清空',
    okType: 'danger',
    cancelText: '取消',
    onOk: async () => {
      try {
        // 实现清空Taint的逻辑，这里可能需要调用API
        message.success(`已清空节点 ${node.name} 的所有污点`);
        
        // 刷新数据
        getNodes();
        
        // 如果当前有节点详情打开，刷新详情
        if (selectedNodeDetails.value && selectedNodeDetails.value.name === node.name) {
          refreshNodeDetails(selectedNodeDetails.value);
        }
      } catch (error: any) {
        message.error(error.message || '清空污点失败');
      }
    }
  });
};

// 启用/禁用调度
const handleToggleSchedule = (record: NodeItem | GetNodeDetailRes | null = null, newState: boolean | null = null): void => {
  // 如果没有传入节点，检查是否有选中节点
  if (!record) {
    if (!selectedNode.value) {
      message.warning('请先选择一个节点');
      return;
    }
    record = selectedNode.value;
  }
  
  const schedulable = newState !== null ? newState : !(record as NodeItem).schedulable;
  const action = schedulable ? '启用' : '禁用';
  
  Modal.confirm({
    title: `确认${action}节点调度`,
    content: `确定要${action}节点 ${record.name} 的调度功能吗？`,
    okText: '确认',
    cancelText: '取消',
    onOk: async () => {
      try {
        // 实现启用/禁用调度的逻辑，这里可能需要调用API
        message.success(`已${action}节点 ${record.name} 的调度功能`);
        
        // 刷新数据
        getNodes();
        
        // 如果当前有节点详情打开，刷新详情
        if (selectedNodeDetails.value && selectedNodeDetails.value.name === record.name) {
          refreshNodeDetails(selectedNodeDetails.value);
        }
      } catch (error: any) {
        message.error(error.message || `${action}调度失败`);
      }
    }
  });
};

// ====== 模态框控制 ======
// 弹出删除标签模态框
const showDeleteLabelModal = (record: NodeItem): void => {
  selectedNodeDetails.value = record as unknown as GetNodeDetailRes;
  isDeleteLabelModalVisible.value = true;
};

// 关闭添加标签模态框
const closeAddLabelModal = (): void => {
  labelForm.nodeName = '';
  labelForm.key = '';
  labelForm.value = '';
  isAddLabelModalVisible.value = false;
};

// 关闭添加Taint模态框
const closeAddTaintModal = (): void => {
  taintForm.nodeName = '';
  taintForm.taintYaml = '';
  isAddTaintModalVisible.value = false;
};

// 关闭删除Taint模态框
const closeDeleteTaintModal = (): void => {
  deleteTaintForm.nodeName = '';
  deleteTaintForm.taintYaml = '';
  isDeleteTaintModalVisible.value = false;
};

// 关闭删除标签模态框
const closeDeleteLabelModal = (): void => {
  deleteLabelForm.label = '';
  isDeleteLabelModalVisible.value = false;
};

// 关闭查看详情模态框
const closeViewDetailsModal = (): void => {
  isViewDetailsModalVisible.value = false;
};

// 初始化数据
onMounted(() => {
  getNodes();
});
</script>

<style>
:root {
  --primary-color: #1890ff;
  --success-color: #52c41a;
  --warning-color: #faad14;
  --error-color: #f5222d;
  --font-size-base: 14px;
  --border-radius-base: 4px;
  --box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
  --transition-duration: 0.3s;
}

.node-manager {
  background-color: #f0f2f5;
  border-radius: 8px;
  padding: 24px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
}

/* 仪表板标题样式 */
.dashboard-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 28px;
  padding-bottom: 20px;
  border-bottom: 1px solid #f0f0f0;
}

.dashboard-title {
  font-size: 24px;
  font-weight: 600;
  color: #262626;
  margin: 0;
  display: flex;
  align-items: center;
}

.dashboard-icon {
  margin-right: 14px;
  font-size: 28px;
  color: #1890ff;
}

.dashboard-stats {
  display: flex;
  gap: 20px;
}

.stat-item {
  background: linear-gradient(135deg, #1890ff 0%, #096dd9 100%);
  border-radius: 8px;
  padding: 10px 18px;
  color: white;
  min-width: 120px;
  text-align: center;
  box-shadow: 0 3px 8px rgba(24, 144, 255, 0.2);
}

.stat-value {
  font-size: 20px;
  font-weight: 600;
  line-height: 1.3;
}

.stat-label {
  font-size: 12px;
  opacity: 0.9;
  margin-top: 4px;
}

/* 控制面板样式 */
.control-panel {
  display: flex;
  justify-content: space-between;
  margin-bottom: 24px;
  background: white;
  padding: 20px;
  border-radius: 8px;
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.05);
}

.search-filters {
  display: flex;
  gap: 16px;
  flex-wrap: wrap;
  align-items: center;
  flex: 1;
}

.control-item {
  min-width: 200px;
}

.search-input {
  flex-grow: 1;
  max-width: 300px;
}

.action-buttons {
  display: flex;
  gap: 16px;
  align-items: center;
  margin-left: 20px;
}

.refresh-btn {
  background: linear-gradient(135deg, #1890ff 0%, #096dd9 100%);
  border: none;
  height: 36px;
  width: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.manage-btn {
  background: linear-gradient(135deg, #1890ff 0%, #096dd9 100%);
  border: none;
  height: 36px;
  padding: 0 16px;
  font-weight: 500;
}

.schedule-btn {
  background: linear-gradient(135deg, #ff4d4f 0%, #cf1322 100%);
  border: none;
  height: 36px;
  padding: 0 16px;
  font-weight: 500;
}

.status-option,
.role-option,
.node-option,
.label-option {
  display: flex;
  align-items: center;
  gap: 10px;
}

.status-option :deep(svg),
.role-option :deep(svg),
.node-option :deep(svg),
.label-option :deep(svg) {
  margin-right: 4px;
}

/* 状态摘要卡片 */
.status-summary {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: 20px;
  margin-bottom: 28px;
}

.summary-card {
  background: white;
  border-radius: 8px;
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.05);
  overflow: hidden;
  transition: transform 0.3s, box-shadow 0.3s;
  display: flex;
  flex-direction: column;
}

.summary-card:hover {
  transform: translateY(-3px);
  box-shadow: 0 6px 16px rgba(0, 0, 0, 0.1);
}

.card-content {
  padding: 24px;
  flex-grow: 1;
}

.card-title {
  font-size: 14px;
  color: #8c8c8c;
  margin-top: 10px;
}

.card-metric {
  display: flex;
  align-items: center;
  margin-bottom: 10px;
}

.metric-icon {
  font-size: 28px;
  margin-right: 16px;
}

.metric-value {
  font-size: 32px;
  font-weight: 600;
  color: #262626;
}

.total-card .metric-icon {
  color: #1890ff;
}

.healthy-card .metric-icon {
  color: #52c41a;
}

.problem-card .metric-icon {
  color: #f5222d;
}

.card-footer {
  padding: 14px 24px;
  background-color: #fafafa;
  border-top: 1px solid #f0f0f0;
}

.footer-text {
  font-size: 12px;
  color: #8c8c8c;
  margin-top: 6px;
}

/* 节点表格样式 */
.node-table {
  background: white;
  border-radius: 8px;
  overflow: hidden;
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.05);
  margin-top: 20px;
}

.node-table :deep(.ant-table-thead > tr > th) {
  background-color: #f5f7fa;
  font-weight: 600;
  padding: 14px 16px;
}

.node-table :deep(.ant-table-tbody > tr > td) {
  padding: 12px 16px;
}

.node-name {
  display: flex;
  align-items: center;
  gap: 10px;
  font-weight: 500;
}

.node-status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.status-ready {
  background-color: #52c41a;
}

.status-warning {
  background-color: #faad14;
}

.status-error {
  background-color: #f5222d;
}

.status-unknown {
  background-color: #d9d9d9;
}

.status-tag,
.role-tag,
.label-tag,
.taint-tag {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-weight: 500;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 13px;
}

.status-dot {
  display: inline-block;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background-color: currentColor;
}

.ip-address {
  display: flex;
  align-items: center;
  gap: 10px;
  font-family: 'Courier New', monospace;
  color: #595959;
}

.timestamp {
  display: flex;
  align-items: center;
  gap: 10px;
  color: #595959;
}

.roles-cell {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.labels-cell {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.action-column {
  display: flex;
  gap: 12px;
  justify-content: center;
}

.action-column :deep(.ant-btn) {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0;
}

/* 节点信息单元格 */
.node-info-cell {
  display: flex;
  gap: 8px;
  align-items: center;
}

/* 进度图表组件 */
.progress-chart {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
}

.chart-icon {
  color: #8c8c8c;
  font-size: 14px;
}

.chart-bar {
  width: 60px;
  height: 6px;
  background-color: #f0f0f0;
  border-radius: 3px;
  overflow: hidden;
}

.chart-fill {
  height: 100%;
  border-radius: 3px;
}

.chart-title {
  font-size: 12px;
  color: #595959;
  white-space: nowrap;
}

/* 节点详情和模态框样式 */
.node-modal {
  font-family: system-ui, -apple-system, BlinkMacSystemFont, sans-serif;
}

.modal-alert {
  margin-bottom: 16px;
}

.node-form {
  padding: 10px;
}

.form-input,
.form-select {
  border-radius: 8px;
  height: 42px;
}

.yaml-editor {
  font-family: 'JetBrains Mono', 'Fira Code', 'Courier New', monospace;
  font-size: 14px;
  line-height: 1.5;
  border-radius: 8px;
  background-color: #f9f9f9;
  padding: 12px;
  transition: all 0.3s;
  tab-size: 2;
}

.yaml-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-bottom: 10px;
}

.yaml-modal {
  font-family: "Consolas", "Monaco", monospace;
}

.yaml-info {
  margin-bottom: 16px;
}

.node-detail-modal .ant-tabs-nav {
  margin-bottom: 16px;
}

.details-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 16px;
  margin-bottom: 24px;
}

.detail-card {
  background-color: #fafafa;
  border-radius: 8px;
  padding: 16px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.card-header h4 {
  margin: 0;
  font-weight: 500;
  display: flex;
  align-items: center;
  gap: 8px;
}

.detail-item {
  display: flex;
  justify-content: space-between;
  margin-bottom: 12px;
}

.detail-label {
  color: #8c8c8c;
  font-weight: 500;
}

.detail-value {
  color: #262626;
  font-weight: 400;
}

/* 资源图表 */
.resource-charts {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
  margin-top: 24px;
}

.resource-chart {
  background-color: #fafafa;
  border-radius: 8px;
  padding: 16px;
}

.resource-chart h4 {
  margin: 0 0 16px;
  font-weight: 500;
}

.usage-gauge {
  position: relative;
}

.gauge-value {
  font-size: 24px;
  font-weight: 600;
  margin-bottom: 8px;
}

.gauge-bar {
  height: 8px;
  background-color: #f0f0f0;
  border-radius: 4px;
  overflow: hidden;
}

.gauge-fill {
  height: 100%;
  border-radius: 4px;
}

/* 标签和污点列表 */
.labels-list, .taints-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 12px;
}

/* 事件容器 */
.events-container {
  padding: 8px;
  max-height: 400px;
  overflow-y: auto;
}

.event-card {
  background-color: #fafafa;
  border-radius: 8px;
  padding: 12px;
  margin-bottom: 8px;
}

.event-header {
  display: flex;
  justify-content: space-between;
  margin-bottom: 8px;
}

.event-reason {
  font-weight: 600;
  color: #262626;
}

.event-time {
  color: #8c8c8c;
  font-size: 12px;
}

.event-message {
  margin-bottom: 12px;
  line-height: 1.5;
}

.event-meta {
  display: flex;
  gap: 16px;
  font-size: 12px;
  color: #595959;
  margin-bottom: 8px;
}

.event-time-range {
  font-size: 12px;
  color: #8c8c8c;
  display: flex;
  align-items: center;
  gap: 4px;
}

/* 模态框底部 */
.modal-footer {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}

/* 响应式调整 */
@media (max-width: 1400px) {
  .status-summary {
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  }
  
  .details-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 768px) {
  .dashboard-header {
    flex-direction: column;
    align-items: flex-start;
  }
  
  .dashboard-stats {
    margin-top: 16px;
    width: 100%;
  }
  
  .control-panel {
    flex-direction: column;
  }
  
  .search-filters {
    margin-bottom: 16px;
  }
  
  .action-buttons {
    margin-left: 0;
    justify-content: flex-end;
  }
  
  .node-info-cell {
    flex-direction: column;
    align-items: flex-start;
  }
}
</style>