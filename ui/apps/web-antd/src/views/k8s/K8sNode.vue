<template>
  <div class="node-management-container">
    <!-- 页面标题和状态概览 -->
    <div class="dashboard-header">
      <div class="header-title">
        <cluster-outlined class="title-icon" />
        <div>
          <h2 class="page-title">Kubernetes 节点管理</h2>
          <p class="subtitle">集群 ID: {{ route.query.cluster_id }}</p>
        </div>
      </div>
      <div class="status-overview">
        <a-card class="status-card">
          <template #title>
            <check-circle-outlined class="status-icon success" />
            <span>健康节点</span>
          </template>
          <div class="status-value">{{ healthyNodes }}</div>
        </a-card>
        <a-card class="status-card">
          <template #title>
            <warning-outlined class="status-icon warning" />
            <span>警告节点</span>
          </template>
          <div class="status-value">{{ warningNodes }}</div>
        </a-card>
        <a-card class="status-card">
          <template #title>
            <stop-outlined class="status-icon error" />
            <span>故障节点</span>
          </template>
          <div class="status-value">{{ errorNodes }}</div>
        </a-card>
      </div>
    </div>

    <!-- 操作栏 -->
    <div class="node-actions">
      <div class="search-area">
        <a-input-search
          v-model:value="searchText"
          allow-clear
          placeholder="搜索节点名称、IP或角色..."
          style="width: 300px"
          @search="handleSearch"
        >
          <template #prefix>
            <search-outlined />
          </template>
        </a-input-search>
        <a-select
          v-model:value="statusFilter"
          allow-clear
          placeholder="状态筛选"
          style="width: 150px; margin-left: 8px"
          @change="handleFilterChange"
        >
          <a-select-option value="Ready">正常</a-select-option>
          <a-select-option value="NotReady">异常</a-select-option>
          <a-select-option value="Unknown">未知</a-select-option>
        </a-select>
        <a-select
          v-model:value="roleFilter"
          allow-clear
          placeholder="角色筛选"
          style="width: 150px; margin-left: 8px"
          @change="handleFilterChange"
        >
          <a-select-option value="master">Master</a-select-option>
          <a-select-option value="worker">Worker</a-select-option>
        </a-select>
      </div>
      <div class="action-buttons">
        <a-dropdown>
          <template #overlay>
            <a-menu>
              <a-menu-item key="1" @click="isAddLabelModalVisible = true">
                <tag-outlined /> 添加节点标签
              </a-menu-item>
              <a-menu-item key="2" @click="isAddTaintModalVisible = true">
                <warning-outlined /> 添加 Taint
              </a-menu-item>
              <a-menu-item key="3" @click="isDeleteTaintModalVisible = true">
                <delete-outlined /> 删除 Taint
              </a-menu-item>
              <a-menu-item key="4" @click="handleClearTaints">
                <clear-outlined /> 清空 Taint
              </a-menu-item>
            </a-menu>
          </template>
          <a-button type="primary">
            节点管理
            <down-outlined />
          </a-button>
        </a-dropdown>
        <a-button 
          type="primary" 
          @click="handleToggleSchedule()" 
          :disabled="!hasSelectedNode"
        >
          <schedule-outlined />
          启用/禁用调度
        </a-button>
        <a-button 
          type="primary" 
          ghost 
          @click="refreshData"
        >
          <reload-outlined />
          刷新
        </a-button>
      </div>
    </div>

    <!-- 节点表格 -->
    <div class="node-table-container">
      <a-spin :spinning="loading" tip="加载节点数据...">
        <a-table
          :row-selection="{ 
            type: 'radio', 
            onChange: onSelectChange,
            selectedRowKeys: selectedRowKeys
          }"
          :columns="columns"
          :data-source="filteredData"
          :pagination="{ 
            pageSize: 10, 
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total) => `共 ${total} 个节点`
          }"
          :scroll="{ x: '100%' }"
          row-key="name"
          bordered
        >
          <!-- 节点名称列 -->
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'name'">
              <div class="node-name-cell">
                <div class="node-status-dot" :class="getNodeStatusClass(record.status)"></div>
                <span>{{ record.name }}</span>
              </div>
            </template>
            
            <!-- 状态列 -->
            <template v-else-if="column.key === 'status'">
              <a-tag :color="getStatusColor(record.status)">
                {{ record.status }}
              </a-tag>
            </template>
            
            <!-- 角色列 -->
            <template v-else-if="column.key === 'roles'">
              <div class="roles-cell">
                <template v-if="record.roles && typeof record.roles === 'string'">
                  <a-tag 
                    v-for="role in record.roles.split(',').filter(Boolean)" 
                    :key="role" 
                    :color="getRoleColor(role)"
                  >
                    {{ role }}
                  </a-tag>
                </template>
                <a-tag v-else color="default">未知</a-tag>
              </div>
            </template>
            
            <!-- 节点信息列 -->
            <template v-else-if="column.key === 'info'">
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
            <template v-else-if="column.key === 'labels'">
              <div class="labels-cell">
                <a-tag v-for="(label, index) in getNodeLabels(record)" :key="index" color="blue">
                  {{ label }}
                </a-tag>
                <a-tag v-if="getNodeLabels(record).length > 3" color="blue">
                  +{{ getNodeLabels(record).length - 3 }}
                </a-tag>
              </div>
            </template>
            
            <!-- 操作列 -->
            <template v-else-if="column.key === 'action'">
              <div class="action-cell">
                <a-tooltip title="查看详情">
                  <a-button type="primary" shape="circle" size="small" @click="handleViewDetails(record)">
                    <template #icon><eye-outlined /></template>
                  </a-button>
                </a-tooltip>
                <a-tooltip title="删除标签">
                  <a-button type="primary" shape="circle" size="small" @click="showDeleteLabelModal(record)">
                    <template #icon><tag-outlined /></template>
                  </a-button>
                </a-tooltip>
                <a-tooltip :title="record.schedulable ? '禁用调度' : '启用调度'">
                  <a-button 
                    :type="record.schedulable ? 'default' : 'primary'" 
                    shape="circle" 
                    size="small" 
                    @click="handleToggleSchedule(record)"
                  >
                    <template #icon>
                      <pause-outlined v-if="record.schedulable" />
                      <caret-right-outlined v-else />
                    </template>
                  </a-button>
                </a-tooltip>
                <a-dropdown>
                  <template #overlay>
                    <a-menu>
                      <a-menu-item key="1" @click="handleAddLabel(record)">
                        <tag-outlined /> 添加标签
                      </a-menu-item>
                      <a-menu-item key="2" @click="handleAddTaint(record)">
                        <warning-outlined /> 添加 Taint
                      </a-menu-item>
                      <a-menu-item key="3" @click="handleDeleteTaint(record)">
                        <delete-outlined /> 删除 Taint
                      </a-menu-item>
                      <a-menu-divider />
                      <a-menu-item key="4" @click="handleCordon(record)" danger>
                        <stop-outlined /> 维护模式
                      </a-menu-item>
                    </a-menu>
                  </template>
                  <a-button type="primary" shape="circle" size="small">
                    <template #icon><more-outlined /></template>
                  </a-button>
                </a-dropdown>
              </div>
            </template>
          </template>
        </a-table>
      </a-spin>
    </div>

    <!-- 添加标签模态框 -->
    <a-modal
      v-model:open="isAddLabelModalVisible"
      title="添加节点标签"
      :confirm-loading="submitLoading"
      @cancel="closeAddLabelModal"
      @ok="handleSubmitAddLabel"
    >
      <a-form :model="labelForm" layout="vertical">
        <a-alert
          type="info"
          show-icon
          banner
          message="节点标签可用于 Pod 调度及资源分配"
          style="margin-bottom: 16px"
        />
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
          >
            <a-select-option v-for="node in filteredData" :key="node.name" :value="node.name">
              {{ node.name }}
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
              <a-input v-model:value="labelForm.key" placeholder="请输入标签键" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item
              label="标签值"
              name="value"
              :rules="[{ required: true, message: '请输入标签值' }]"
            >
              <a-input v-model:value="labelForm.value" placeholder="请输入标签值" />
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
    >
      <a-form :model="taintForm" layout="vertical">
        <a-alert
          type="info"
          show-icon
          banner
          message="Taint 用于阻止 Pod 调度到节点上"
          style="margin-bottom: 16px"
        />
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
          >
            <a-select-option v-for="node in filteredData" :key="node.name" :value="node.name">
              {{ node.name }} ({{ node.ip }})
            </a-select-option>
          </a-select>
        </a-form-item>
        
        <a-form-item
          label="Taint YAML"
          name="taintYaml"
          :rules="[{ required: true, message: '请输入 Taint YAML' }]"
        >
          <a-textarea
            v-model:value="taintForm.taintYaml"
            :rows="6"
            :auto-size="{ minRows: 6, maxRows: 10 }"
            placeholder="示例：- key: &quot;example-key&quot;
  value: &quot;example-value&quot; 
  effect: &quot;NoSchedule&quot;"
          />
        </a-form-item>
        
        <a-form-item>
          <div class="taint-actions">
            <a-button type="primary" @click="checkTaintYaml(taintForm.nodeName)">
              <code-outlined /> 验证 YAML 格式
            </a-button>
            <a-popover title="Taint 效果说明" placement="right">
              <template #content>
                <p><strong>NoSchedule</strong>: 不允许新 Pod 调度</p>
                <p><strong>PreferNoSchedule</strong>: 尽量不调度</p>
                <p><strong>NoExecute</strong>: 驱逐现有 Pod</p>
              </template>
              <a-button>
                <question-circle-outlined /> 效果说明
              </a-button>
            </a-popover>
          </div>
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
    >
      <a-form :model="deleteTaintForm" layout="vertical">
        <a-alert
          type="info"
          show-icon
          banner
          message="删除 Taint 将允许 Pod 重新调度到节点上"
          style="margin-bottom: 16px"
        />
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
          >
            <a-select-option v-for="node in filteredData" :key="node.name" :value="node.name">
              {{ node.name }} ({{ node.ip }})
            </a-select-option>
          </a-select>
        </a-form-item>
        
        <a-form-item
          label="Taint YAML"
          name="taintYaml"
          :rules="[{ required: true, message: '请输入 Taint YAML' }]"
        >
          <a-textarea
            v-model:value="deleteTaintForm.taintYaml"
            :rows="6"
            :auto-size="{ minRows: 6, maxRows: 10 }"
            placeholder="示例：- key: &quot;example-key&quot;
  value: &quot;example-value&quot; 
  effect: &quot;NoSchedule&quot;"
          />
        </a-form-item>
        
        <a-form-item>
          <a-button type="primary" @click="checkTaintYaml(deleteTaintForm.nodeName)">
            <code-outlined /> 验证 YAML 格式
          </a-button>
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
    >
      <a-alert
        type="warning"
        show-icon
        banner
        message="删除标签可能会影响依赖此标签的 Pod 调度"
        style="margin-bottom: 16px"
      />
      <a-form :model="deleteLabelForm" layout="vertical">
        <a-form-item
          label="选择标签"
          name="label"
          :rules="[{ required: true, message: '请选择标签' }]"
        >
          <a-select v-model:value="deleteLabelForm.label" placeholder="请选择标签">
            <a-select-option v-for="(label, index) in labelOptions" :key="index" :value="label">
              {{ label }}
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
    >
      <a-spin :spinning="detailsLoading">
        <div v-if="selectedNodeDetails" class="node-details">
          <div class="node-details-header">
            <div class="node-title-area">
              <div class="node-status-badge" :class="getNodeStatusClass(selectedNodeDetails.status)"></div>
              <h2>{{ selectedNodeDetails.name }}</h2>
            </div>
            <a-tag :color="getStatusColor(selectedNodeDetails.status)">{{ selectedNodeDetails.status }}</a-tag>
          </div>
          
          <a-divider />
          
          <a-tabs default-active-key="1">
            <a-tab-pane key="1" tab="基本信息">
              <div class="details-grid">
                <div class="detail-card">
                  <h4><environment-outlined /> 基础配置</h4>
                  <div class="detail-item">
                    <div class="detail-label">IP 地址:</div>
                    <div class="detail-value">{{ selectedNodeDetails.ip }}</div>
                  </div>
                  <div class="detail-item">
                    <div class="detail-label">角色:</div>
                    <div class="detail-value">
                      <template v-if="selectedNodeDetails.roles && typeof selectedNodeDetails.roles === 'string'">
                        <a-tag 
                          v-for="role in selectedNodeDetails.roles.split(',').filter(Boolean)" 
                          :key="role" 
                          :color="getRoleColor(role)"
                        >
                          {{ role }}
                        </a-tag>
                      </template>
                      <a-tag v-else color="default">未知</a-tag>
                    </div>
                  </div>
                  <div class="detail-item">
                    <div class="detail-label">创建时间:</div>
                    <div class="detail-value">{{ selectedNodeDetails.age }}</div>
                  </div>
                  <div class="detail-item">
                    <div class="detail-label">调度状态:</div>
                    <div class="detail-value">
                      <a-tag :color="selectedNodeDetails.schedulable ? 'green' : 'red'">
                        {{ selectedNodeDetails.schedulable ? '可调度' : '不可调度' }}
                      </a-tag>
                    </div>
                  </div>
                </div>
                
                <div class="detail-card">
                  <h4><api-outlined /> 资源状态</h4>
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
                    <h4><tag-outlined /> 节点标签</h4>
                    <a-button type="primary" size="small" @click="handleAddLabel(selectedNodeDetails)">
                      <plus-outlined /> 添加标签
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
                    >
                      {{ label }}
                    </a-tag>
                  </div>
                </div>
                
                <div class="detail-card">
                  <div class="card-header">
                    <h4><warning-outlined /> 节点污点</h4>
                    <a-button type="primary" size="small" @click="handleAddTaint(selectedNodeDetails)">
                      <plus-outlined /> 添加污点
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
                    >
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
                        <span class="event-time">{{ formatTime(event.last_time) }}</span>
                      </div>
                      <div class="event-message">{{ event.message }}</div>
                      <div class="event-meta">
                        <span><strong>类型:</strong> {{ event.type }}</span>
                        <span><strong>组件:</strong> {{ event.component }}</span>
                        <span><strong>发生次数:</strong> {{ event.count }}</span>
                      </div>
                      <div class="event-time-range">
                        <calendar-outlined /> 
                        {{ formatTime(event.first_time) }} - {{ formatTime(event.last_time) }}
                      </div>
                    </div>
                  </a-timeline-item>
                </a-timeline>
                <a-empty v-if="!selectedNodeDetails.events || selectedNodeDetails.events.length === 0" description="暂无事件" />
              </div>
            </a-tab-pane>
          </a-tabs>
          
          <a-divider />
          
          <div class="modal-footer">
            <a-space>
              <a-button @click="closeViewDetailsModal">关闭</a-button>
              <a-button type="primary" @click="refreshNodeDetails(selectedNodeDetails)">
                <reload-outlined /> 刷新数据
              </a-button>
              <a-button 
                :type="selectedNodeDetails.schedulable ? 'default' : 'primary'" 
                @click="handleToggleSchedule(selectedNodeDetails)"
              >
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
    fixed: 'left',
    width: 220,
    sorter: (a: NodeItem, b: NodeItem) => a.name.localeCompare(b.name),
  },
  { 
    title: '状态', 
    dataIndex: 'status', 
    key: 'status', 
    width: 100,
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
    width: 120,
  },
  { 
    title: '角色', 
    dataIndex: 'roles', 
    key: 'roles',
    width: 120,
  },
  { 
    title: '创建时间', 
    dataIndex: 'age', 
    key: 'age',
    width: 100,
  },
  { 
    title: '资源使用', 
    key: 'info',
    width: 240,
  },
  { 
    title: '标签', 
    key: 'labels',
    width: 200,
  },
  { 
    title: '操作', 
    key: 'action',
    fixed: 'right',
    width: 180,
  },
];

// ====== 工具函数 ======
// 状态颜色映射
const getStatusColor = (status: string | undefined): string => {
  if (!status) return 'default';
  
  const statusMap: Record<string, string> = {
    'Ready': 'success',
    'NotReady': 'error',
    'Unknown': 'warning',
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
  selectedNode.value = rows.length > 0 ? rows[0] : null;
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
    const res = await getNodeDetailsApi(record.name, record.cluster_id);
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
    const res = await getNodeDetailsApi(node.name, node.cluster_id);
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
      labels: [key, value], // 标签是交替格式 key, val
      mod_type: 'add',
      node_name: nodeName,
    });
    message.success('标签添加成功');
    
    // 清除表单数据
    labelForm.nodeName = '';
    labelForm.key = '';
    labelForm.value = '';
    
    // 刷新数据
    getNodes();
    
    // 如果当前有节点详情打开，刷新详情
    if (selectedNodeDetails.value && selectedNodeDetails.value.name === nodeName) {
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
          node_name: selectedNodeDetails.value.name,
        });
        message.success('标签删除成功');
        
        // 刷新节点详情
        if (selectedNodeDetails.value) {
          refreshNodeDetails(selectedNodeDetails.value);
        }
        
        // 刷新节点列表
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
      node_name: selectedNodeDetails.value.name,
    });
    message.success('标签删除成功');
    
    // 刷新数据
    getNodes();
    
    // 刷新节点详情
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
    
    // 清除表单数据
    taintForm.nodeName = '';
    taintForm.taintYaml = '';
    
    // 刷新数据
    getNodes();
    
    // 如果当前有节点详情打开，刷新详情
    if (selectedNodeDetails.value && selectedNodeDetails.value.name === taintForm.nodeName) {
      refreshNodeDetails(selectedNodeDetails.value);
    }
    
    // 关闭模态框
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
    
    // 清除表单数据
    deleteTaintForm.nodeName = '';
    deleteTaintForm.taintYaml = '';
    
    // 刷新数据
    getNodes();
    
    // 如果当前有节点详情打开，刷新详情
    if (selectedNodeDetails.value && selectedNodeDetails.value.name === deleteTaintForm.nodeName) {
      refreshNodeDetails(selectedNodeDetails.value);
    }
    
    // 关闭模态框
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
        
        // 将单行taint转换为yaml格式
        const taintParts = taint.split(':');
        const keyValue = taintParts[0].trim().split('=');
        const effect = taintParts[1] ? taintParts[1].trim() : 'NoSchedule';
        
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
        
        // 刷新节点详情
        if (selectedNodeDetails.value) {
          refreshNodeDetails(selectedNodeDetails.value);
        }
        
        // 刷新节点列表
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
  
  const schedulable = newState !== null ? newState : !record.schedulable;
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

<style scoped>
.node-management-container {
  background-color: #f0f2f5;
  padding: 24px;
  min-height: 100vh;
}

/* 页面标题和状态概览 */
.dashboard-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 24px;
}

.header-title {
  display: flex;
  align-items: center;
  gap: 16px;
}

.title-icon {
  font-size: 32px;
  color: #1890ff;
  background-color: rgba(24, 144, 255, 0.1);
  padding: 12px;
  border-radius: 8px;
}

.page-title {
  margin: 0;
  color: #262626;
  font-weight: 600;
}

.subtitle {
  margin: 4px 0 0;
  color: #8c8c8c;
  font-size: 14px;
}

.status-overview {
  display: flex;
  gap: 16px;
}

.status-card {
  width: 150px;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
  transition: all 0.3s;
}

.status-card:hover {
  transform: translateY(-3px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.status-icon {
  margin-right: 8px;
  font-size: 14px;
}

.status-icon.success {
  color: #52c41a;
}

.status-icon.warning {
  color: #faad14;
}

.status-icon.error {
  color: #f5222d;
}

.status-value {
  font-size: 28px;
  font-weight: 600;
  color: #1f1f1f;
  text-align: center;
}

/* 操作栏 */
.node-actions {
  background-color: white;
  border-radius: 8px;
  padding: 16px;
  margin-bottom: 24px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}

.search-area {
  display: flex;
  align-items: center;
}

.action-buttons {
  display: flex;
  gap: 12px;
}

/* 节点表格 */
.node-table-container {
  background-color: white;
  border-radius: 8px;
  padding: 16px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}

/* 节点名称单元格 */
.node-name-cell {
  display: flex;
  align-items: center;
  gap: 12px;
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

/* 角色单元格 */
.roles-cell {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

/* 节点信息单元格 */
.node-info-cell {
  display: flex;
  gap: 8px;
  align-items: center;
}

/* 标签单元格 */
.labels-cell {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

/* 操作单元格 */
.action-cell {
  display: flex;
  gap: 8px;
}

/* 进度图表组件 */
.progress-chart {
  display: flex;
  align-items: center;
  gap: 8px;
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

/* Taint操作按钮 */
.taint-actions {
  display: flex;
  gap: 8px;
}

/* 节点详情样式 */
.node-details-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.node-title-area {
  display: flex;
  align-items: center;
  gap: 12px;
}

.node-status-badge {
  width: 12px;
  height: 12px;
  border-radius: 50%;
}

.node-details-header h2 {
  margin: 0;
  font-weight: 600;
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
}
</style>