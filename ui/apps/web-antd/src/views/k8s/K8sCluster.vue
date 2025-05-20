<template>
  <div class="service-manager cluster-manager">
    <!-- 仪表板标题 -->
    <div class="dashboard-header">
      <h2 class="dashboard-title">
        <ClusterOutlined class="dashboard-icon" />
        Kubernetes 集群管理器
      </h2>
      <div class="dashboard-stats">
        <div class="stat-item">
          <div class="stat-value">{{ clusters.length }}</div>
          <div class="stat-label">集群总数</div>
        </div>
        <div class="stat-item">
          <div class="stat-value">{{ Object.keys(envDistribution).length }}</div>
          <div class="stat-label">环境分类</div>
        </div>
      </div>
    </div>

    <!-- 查询和操作工具栏 -->
    <div class="control-panel">
      <div class="search-filters">
        <a-input-search
          v-model:value="searchText"
          placeholder="搜索集群名称"
          class="control-item search-input"
          @search="onSearch"
          allow-clear
        >
          <template #prefix><SearchOutlined /></template>
        </a-input-search>
        
        <a-select
          v-model:value="filterEnv"
          placeholder="环境筛选"
          class="control-item env-selector"
          allow-clear
          @change="onEnvFilterChange"
        >
          <template #suffixIcon><EnvironmentOutlined /></template>
          <a-select-option value="dev">
            <span class="env-option">
              <ApiOutlined />
              开发环境
            </span>
          </a-select-option>
          <a-select-option value="prod">
            <span class="env-option">
              <ApiOutlined />
              生产环境
            </span>
          </a-select-option>
          <a-select-option value="stage">
            <span class="env-option">
              <ApiOutlined />
              阶段环境
            </span>
          </a-select-option>
          <a-select-option value="rc">
            <span class="env-option">
              <ApiOutlined />
              发布候选
            </span>
          </a-select-option>
          <a-select-option value="press">
            <span class="env-option">
              <ApiOutlined />
              压力测试
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
        
        <a-button 
          type="primary" 
          class="create-btn" 
          @click="isAddModalVisible = true"
        >
          <template #icon><PlusOutlined /></template>
          新增集群
        </a-button>
        
        <a-button 
          type="primary" 
          danger 
          class="delete-btn" 
          @click="showBatchDeleteConfirm" 
          :disabled="!selectedRows.length"
        >
          <template #icon><DeleteOutlined /></template>
          批量删除 {{ selectedRows.length ? `(${selectedRows.length})` : '' }}
        </a-button>
      </div>
    </div>

    <!-- 状态摘要卡片 -->
    <div class="status-summary">
      <div class="summary-card total-card">
        <div class="card-content">
          <div class="card-metric">
            <DashboardOutlined class="metric-icon" />
            <div class="metric-value">{{ clusters.length }}</div>
          </div>
          <div class="card-title">集群总数</div>
        </div>
        <div class="card-footer">
          <div class="footer-text">管理您的全部Kubernetes集群</div>
        </div>
      </div>
      
      <div class="summary-card running-card">
        <div class="card-content">
          <div class="card-metric">
            <CheckCircleOutlined class="metric-icon" />
            <div class="metric-value">{{ resourceUtilization }}%</div>
          </div>
          <div class="card-title">资源使用率</div>
        </div>
        <div class="card-footer">
          <a-progress 
            :percent="resourceUtilization" 
            :stroke-color="{ from: '#1890ff', to: '#52c41a' }" 
            size="small" 
            :show-info="false" 
          />
          <div class="footer-text">集群资源平均使用率</div>
        </div>
      </div>
      
      <div class="summary-card env-card">
        <div class="card-content">
          <div class="card-metric">
            <EnvironmentOutlined class="metric-icon" />
            <div class="metric-value">{{ Object.keys(envDistribution).length }}</div>
          </div>
          <div class="card-title">环境分类</div>
        </div>
        <div class="card-footer">
          <div class="env-distribution">
            <div v-for="(count, env) in envDistribution" :key="env" class="env-badge">
              <a-tag :color="getEnvColor(env)">{{ getEnvName(env) }}: {{ count }}</a-tag>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 视图切换 -->
    <div class="view-toggle">
      <a-radio-group v-model:value="viewMode" button-style="solid">
        <a-radio-button value="table">
          <UnorderedListOutlined />
          表格视图
        </a-radio-button>
        <a-radio-button value="card">
          <AppstoreOutlined />
          卡片视图
        </a-radio-button>
      </a-radio-group>
    </div>

    <!-- 表格视图 -->
    <a-table
      v-if="viewMode === 'table'"
      :columns="columns"
      :data-source="filteredData"
      :row-selection="rowSelection"
      :loading="loading"
      row-key="id"
      :pagination="{ 
        pageSize: 10, 
        showSizeChanger: true, 
        showQuickJumper: true,
        showTotal: (total: number) => `共 ${total} 条数据`
      }"
      class="services-table cluster-table"
    >
      <!-- 集群名称列 -->
      <template #name="{ text, record }">
        <div class="cluster-name">
          <ClusterOutlined />
          <a @click="handleViewNodes(record.id)">{{ text }}</a>
        </div>
      </template>
      
      <!-- 环境列 -->
      <template #env="{ text }">
        <a-tag :color="getEnvColor(text)" class="env-tag">
          <span class="status-dot"></span>
          {{ getEnvName(text) }}
        </a-tag>
      </template>
      
      <!-- 状态列 -->
      <template #status="{ text }">
        <a-tag :color="getStatusColor(text)" class="status-tag">
          <span class="status-dot"></span>
          {{ text }}
        </a-tag>
      </template>

      <!-- 创建时间列 -->
      <template #created_at="{ text }">
        <div class="timestamp">
          <ClockCircleOutlined />
          <a-tooltip :title="formatDateTime(text)">
            {{ formatDate(text) }}
          </a-tooltip>
        </div>
      </template>

      <!-- 操作列 -->
      <template #action="{ record }">
        <div class="action-column">
          <a-tooltip title="编辑集群">
            <a-button type="primary" ghost shape="circle" @click="handleEdit(record.id)">
              <template #icon><EditOutlined /></template>
            </a-button>
          </a-tooltip>
          
          <a-tooltip title="查看节点">
            <a-button type="primary" ghost shape="circle" @click="handleViewNodes(record.id)">
              <template #icon><EyeOutlined /></template>
            </a-button>
          </a-tooltip>
          
          <a-tooltip title="删除集群">
            <a-popconfirm
              title="确定要删除该集群吗?"
              description="此操作不可撤销"
              @confirm="handleDelete(record.id)"
              ok-text="确定"
              cancel-text="取消"
            >
              <a-button type="primary" danger ghost shape="circle">
                <template #icon><DeleteOutlined /></template>
              </a-button>
            </a-popconfirm>
          </a-tooltip>
        </div>
      </template>

      <!-- 空状态 -->
      <template #emptyText>
        <div class="empty-state">
          <ClusterOutlined style="font-size: 48px; color: #d9d9d9; margin-bottom: 16px" />
          <p>暂无集群数据</p>
          <a-button type="primary" @click="isAddModalVisible = true">新增第一个集群</a-button>
        </div>
      </template>
    </a-table>

    <!-- 卡片视图 -->
    <div v-else class="card-view">
      <a-spin :spinning="loading">
        <a-empty v-if="filteredData.length === 0" description="暂无集群数据" />
        <div v-else class="service-cards cluster-cards">
          <a-checkbox-group v-model:value="selectedCardIds" class="card-checkbox-group">
            <div v-for="cluster in filteredData" :key="cluster.id" class="service-card cluster-card">
              <div class="card-header">
                <a-checkbox :value="cluster.id" class="card-checkbox" />
                <div class="service-title cluster-title">
                  <ClusterOutlined class="service-icon" />
                  <h3>{{ cluster.name }}</h3>
                </div>
                <a-tag :color="getEnvColor(cluster.env)" class="card-type-tag env-tag">
                  <span class="status-dot"></span>
                  {{ getEnvName(cluster.env) }}
                </a-tag>
              </div>
              
              <div class="card-content">
                <div class="card-detail name-zh-detail">
                  <span class="detail-label">中文名称:</span>
                  <span class="detail-value">
                    {{ cluster.name_zh || '-' }}
                  </span>
                </div>
                <div class="card-detail version-detail">
                  <span class="detail-label">版本:</span>
                  <span class="detail-value">
                    {{ cluster.version || '-' }}
                  </span>
                </div>
                <div class="card-detail status-detail">
                  <span class="detail-label">状态:</span>
                  <span class="detail-value">
                    <a-badge :status="getStatusType(cluster.status)" :text="cluster.status || '未知'" />
                  </span>
                </div>
                <div class="card-detail created-detail">
                  <span class="detail-label">创建时间:</span>
                  <span class="detail-value">
                    <ClockCircleOutlined />
                    {{ formatDate(cluster.created_at) }}
                  </span>
                </div>
              </div>
              
              <div class="card-footer card-action-footer">
                <a-button type="primary" ghost size="small" @click="handleViewNodes(cluster.id)">
                  <template #icon><EyeOutlined /></template>
                  查看节点
                </a-button>
                <a-button type="primary" ghost size="small" @click="handleEdit(cluster.id)">
                  <template #icon><EditOutlined /></template>
                  编辑
                </a-button>
                <a-popconfirm
                  title="确定要删除该集群吗?"
                  @confirm="handleDelete(cluster.id)"
                  ok-text="确定"
                  cancel-text="取消"
                >
                  <a-button type="primary" danger ghost size="small">
                    <template #icon><DeleteOutlined /></template>
                    删除
                  </a-button>
                </a-popconfirm>
              </div>
            </div>
          </a-checkbox-group>
        </div>
      </a-spin>
    </div>

    <!-- 新增集群模态框 -->
    <a-modal
      v-model:open="isAddModalVisible"
      title="新增集群"
      :width="800"
      @ok="handleAdd"
      @cancel="closeAddModal"
      :confirmLoading="submitLoading"
      class="cluster-modal"
    >
      <a-alert type="info" show-icon class="modal-alert">
        <template #message>新增Kubernetes集群</template>
        <template #description>请填写集群的基本信息和连接配置</template>
      </a-alert>
      
      <a-form :model="addForm" layout="vertical" class="cluster-form">
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="集群名称" name="name" :rules="[{ required: true, message: '请输入集群名称' }]">
              <a-input v-model:value="addForm.name" placeholder="请输入集群名称" class="form-input">
                <template #prefix><ClusterOutlined /></template>
              </a-input>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="集群中文名称" name="name_zh">
              <a-input v-model:value="addForm.name_zh" placeholder="请输入集群中文名称" class="form-input">
                <template #prefix><FontSizeOutlined /></template>
              </a-input>
            </a-form-item>
          </a-col>
        </a-row>
        
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="环境" name="env" :rules="[{ required: true, message: '请选择环境' }]">
              <a-select v-model:value="addForm.env" placeholder="请选择环境" class="form-select">
                <template #suffixIcon><EnvironmentOutlined /></template>
                <a-select-option value="dev">开发环境</a-select-option>
                <a-select-option value="prod">生产环境</a-select-option>
                <a-select-option value="stage">阶段环境</a-select-option>
                <a-select-option value="rc">发布候选</a-select-option>
                <a-select-option value="press">压力测试</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="集群版本" name="version">
              <a-input v-model:value="addForm.version" placeholder="请输入集群版本" class="form-input">
                <template #prefix><TagOutlined /></template>
              </a-input>
            </a-form-item>
          </a-col>
        </a-row>
        
        <a-divider orientation="left">资源配置</a-divider>
        
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="CPU 请求" name="cpu_request">
              <a-input-number v-model:value="addForm.cpu_request" style="width: 100%" placeholder="请输入 CPU 请求" addon-after="cores" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="CPU 限制" name="cpu_limit">
              <a-input-number v-model:value="addForm.cpu_limit" style="width: 100%" placeholder="请输入 CPU 限制" addon-after="cores" />
            </a-form-item>
          </a-col>
        </a-row>
        
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="内存请求" name="memory_request">
              <a-input-number v-model:value="addForm.memory_request" style="width: 100%" placeholder="请输入内存请求" addon-after="Mi" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="内存限制" name="memory_limit">
              <a-input-number v-model:value="addForm.memory_limit" style="width: 100%" placeholder="请输入内存限制" addon-after="Mi" />
            </a-form-item>
          </a-col>
        </a-row>
        
        <a-divider orientation="left">高级配置</a-divider>
        
        <a-form-item label="限制命名空间" name="restricted_name_space">
          <a-select
            v-model:value="addForm.restricted_name_space"
            mode="tags"
            placeholder="请选择限制命名空间"
            style="width: 100%"
            :token-separators="[',']"
          >
            <template #suffixIcon><PartitionOutlined /></template>
          </a-select>
        </a-form-item>
        
        <a-form-item label="API 服务器地址" name="api_server_addr">
          <a-input v-model:value="addForm.api_server_addr" placeholder="请输入 API 服务器地址" class="form-input">
            <template #prefix><ApiOutlined /></template>
          </a-input>
        </a-form-item>
        
        <a-form-item label="KubeConfig 内容" name="kube_config_content">
          <a-textarea
            v-model:value="addForm.kube_config_content"
            placeholder="请输入 KubeConfig 内容"
            :auto-size="{ minRows: 4, maxRows: 8 }"
            class="form-textarea"
          />
        </a-form-item>
        
        <a-form-item label="操作超时（秒）" name="action_timeout_seconds">
          <a-slider
            v-model:value="addForm.action_timeout_seconds"
            :min="10"
            :max="300"
            :step="10"
            :marks="{
              10: '10s',
              60: '60s',
              120: '120s',
              300: '300s',
            }"
          />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 编辑集群模态框 -->
    <a-modal
      v-model:open="isEditModalVisible"
      title="编辑集群"
      :width="800"
      @ok="handleUpdate"
      @cancel="closeEditModal"
      :confirmLoading="submitLoading"
      class="cluster-modal"
    >
      <a-alert v-if="editForm.id" type="info" show-icon class="modal-alert">
        <template #message>编辑集群: {{ editForm.name }}</template>
        <template #description>ID: {{ editForm.id }} | 环境: {{ getEnvName(editForm.env) }}</template>
      </a-alert>
      
      <a-form :model="editForm" layout="vertical" class="cluster-form">
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="集群名称" name="name" :rules="[{ required: true, message: '请输入集群名称' }]">
              <a-input v-model:value="editForm.name" placeholder="请输入集群名称" class="form-input">
                <template #prefix><ClusterOutlined /></template>
              </a-input>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="集群中文名称" name="name_zh">
              <a-input v-model:value="editForm.name_zh" placeholder="请输入集群中文名称" class="form-input">
                <template #prefix><FontSizeOutlined /></template>
              </a-input>
            </a-form-item>
          </a-col>
        </a-row>
        
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="环境" name="env" :rules="[{ required: true, message: '请选择环境' }]">
              <a-select v-model:value="editForm.env" placeholder="请选择环境" class="form-select">
                <template #suffixIcon><EnvironmentOutlined /></template>
                <a-select-option value="dev">开发环境</a-select-option>
                <a-select-option value="prod">生产环境</a-select-option>
                <a-select-option value="stage">阶段环境</a-select-option>
                <a-select-option value="rc">发布候选</a-select-option>
                <a-select-option value="press">压力测试</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="集群版本" name="version">
              <a-input v-model:value="editForm.version" placeholder="请输入集群版本" class="form-input">
                <template #prefix><TagOutlined /></template>
              </a-input>
            </a-form-item>
          </a-col>
        </a-row>
        
        <a-divider orientation="left">资源配置</a-divider>
        
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="CPU 请求" name="cpu_request">
              <a-input-number v-model:value="editForm.cpu_request" style="width: 100%" placeholder="请输入 CPU 请求" addon-after="cores" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="CPU 限制" name="cpu_limit">
              <a-input-number v-model:value="editForm.cpu_limit" style="width: 100%" placeholder="请输入 CPU 限制" addon-after="cores" />
            </a-form-item>
          </a-col>
        </a-row>
        
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="内存请求" name="memory_request">
              <a-input-number v-model:value="editForm.memory_request" style="width: 100%" placeholder="请输入内存请求" addon-after="Mi" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="内存限制" name="memory_limit">
              <a-input-number v-model:value="editForm.memory_limit" style="width: 100%" placeholder="请输入内存限制" addon-after="Mi" />
            </a-form-item>
          </a-col>
        </a-row>
        
        <a-divider orientation="left">高级配置</a-divider>
        
        <a-form-item label="限制命名空间" name="restricted_name_space">
          <a-select
            v-model:value="editForm.restricted_name_space"
            mode="tags"
            placeholder="请选择限制命名空间"
            style="width: 100%"
            :token-separators="[',']"
          >
            <template #suffixIcon><PartitionOutlined /></template>
          </a-select>
        </a-form-item>
        
        <a-form-item label="API 服务器地址" name="api_server_addr">
          <a-input v-model:value="editForm.api_server_addr" placeholder="请输入 API 服务器地址" class="form-input">
            <template #prefix><ApiOutlined /></template>
          </a-input>
        </a-form-item>
        
        <a-form-item label="KubeConfig 内容" name="kube_config_content">
          <a-textarea
            v-model:value="editForm.kube_config_content"
            placeholder="请输入 KubeConfig 内容"
            :auto-size="{ minRows: 4, maxRows: 8 }"
            class="form-textarea"
          />
        </a-form-item>
        
        <a-form-item label="操作超时（秒）" name="action_timeout_seconds">
          <a-slider
            v-model:value="editForm.action_timeout_seconds"
            :min="10"
            :max="300"
            :step="10"
            :marks="{
              10: '10s',
              60: '60s',
              120: '120s',
              300: '300s',
            }"
          />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script lang="ts" setup>
import { ref, reactive, computed, onMounted, watch } from 'vue';
import { message, Modal } from 'ant-design-vue';
import { 
  getAllClustersApi, 
  getClusterApi, 
  createClusterApi, 
  updateClusterApi, 
  deleteClusterApi 
} from '#/api';
import type { ClustersItem } from '#/api';
import { useRouter } from 'vue-router';
import { 
  EditOutlined, 
  DeleteOutlined, 
  EyeOutlined,
  SearchOutlined,
  PlusOutlined,
  ReloadOutlined,
  ClusterOutlined,
  EnvironmentOutlined,
  ApiOutlined,
  UnorderedListOutlined,
  AppstoreOutlined,
  CloudServerOutlined,
  ClockCircleOutlined,
  CheckCircleOutlined,
  DashboardOutlined,
  PartitionOutlined,
  TagOutlined,
  FontSizeOutlined
} from '@ant-design/icons-vue';

// 路由和状态管理
const router = useRouter();
const loading = ref(false);
const submitLoading = ref(false);
const clusters = ref<ClustersItem[]>([]);
const searchText = ref('');
const filterEnv = ref<string | undefined>(undefined);
const selectedRows = ref<ClustersItem[]>([]);
const isAddModalVisible = ref(false);
const isEditModalVisible = ref(false);
const viewMode = ref<'table' | 'card'>('table');
const resourceUtilization = ref(68); // 模拟数据，实际应从API获取
const selectedCardIds = ref<number[]>([]);

// 表格列配置
const columns = [
  {
    title: '集群名称',
    dataIndex: 'name',
    key: 'name',
    width: '20%',
    sorter: (a: ClustersItem, b: ClustersItem) => a.name.localeCompare(b.name),
    slots: { customRender: 'name' },
  },
  {
    title: '中文名称',
    dataIndex: 'name_zh',
    key: 'name_zh',
    width: '15%'
  },
  {
    title: '环境',
    dataIndex: 'env',
    key: 'env',
    width: '10%',
    slots: { customRender: 'env' },
    filters: [
      { text: '开发环境', value: 'dev' },
      { text: '生产环境', value: 'prod' },
      { text: '阶段环境', value: 'stage' },
      { text: '发布候选', value: 'rc' },
      { text: '压力测试', value: 'press' },
    ],
    onFilter: (value: string, record: ClustersItem) => record.env === value,
  },
  {
    title: '状态',
    dataIndex: 'status',
    key: 'status',
    width: '10%',
    slots: { customRender: 'status' },
  },
  {
    title: '版本',
    dataIndex: 'version',
    key: 'version',
    width: '10%',
  },
  {
    title: '创建者',
    dataIndex: 'user_id',
    key: 'user_id',
    width: '10%',
  },
  {
    title: '创建时间',
    dataIndex: 'created_at',
    key: 'created_at',
    width: '15%',
    sorter: (a: ClustersItem, b: ClustersItem) => {
      if (!a.created_at || !b.created_at) return 0;
      return new Date(a.created_at).getTime() - new Date(b.created_at).getTime();
    },
    slots: { customRender: 'created_at' },
  },
  {
    title: '操作',
    key: 'action',
    width: '15%',
    fixed: 'right',
    slots: { customRender: 'action' },
  },
];

// 计算属性：环境分布
const envDistribution = computed(() => {
  const distribution: Record<string, number> = {};
  clusters.value.forEach(cluster => {
    const env = cluster.env || 'unknown';
    if (!distribution[env]) {
      distribution[env] = 0;
    }
    distribution[env]++;
  });
  return distribution;
});

// 根据搜索和筛选条件过滤数据
const filteredData = computed(() => {
  const searchValue = searchText.value.trim().toLowerCase();
  return clusters.value.filter(item => {
    const matchSearch = item.name.toLowerCase().includes(searchValue) || 
                       (item.name_zh && item.name_zh.toLowerCase().includes(searchValue));
    const matchEnv = !filterEnv.value || item.env === filterEnv.value;
    return matchSearch && matchEnv;
  });
});

// 根据卡片选择更新 selectedRows
watch(selectedCardIds, (newValue) => {
  selectedRows.value = clusters.value.filter(cluster => 
    newValue.includes(cluster.id)
  );
});

// 批量选择配置
const rowSelection = {
  onChange: (selectedRowKeys: number[], selectedRowsData: ClustersItem[]) => {
    selectedRows.value = selectedRowsData;
    selectedCardIds.value = selectedRowsData.map(row => row.id);
  },
  getCheckboxProps: (record: ClustersItem) => ({
    disabled: false, // 可以根据条件禁用某些行的选择
  }),
};

// 新增、编辑表单
interface ClusterForm {
  name: string;
  name_zh: string;
  cpu_request: string;
  cpu_limit: string;
  memory_request: string;
  memory_limit: string;
  restricted_name_space: string[];
  env: string;
  version: string;
  api_server_addr: string;
  kube_config_content: string;
  action_timeout_seconds: number;
}

const addForm = reactive<ClusterForm>({
  name: '',
  name_zh: '',
  cpu_request: '',
  cpu_limit: '',
  memory_request: '',
  memory_limit: '',
  restricted_name_space: [],
  env: 'dev',
  version: '',
  api_server_addr: '',
  kube_config_content: '',
  action_timeout_seconds: 60,
});

const editForm = reactive<ClusterForm & { id: number }>({
  id: 0,
  name: '',
  name_zh: '',
  cpu_request: '',
  cpu_limit: '',
  memory_request: '',
  memory_limit: '',
  restricted_name_space: [],
  env: 'dev',
  version: '',
  api_server_addr: '',
  kube_config_content: '',
  action_timeout_seconds: 60,
});

// 获取集群列表
const getClusters = async () => {
  loading.value = true;
  try {
    const res = await getAllClustersApi();
    clusters.value = res;
  } catch (error: any) {
    message.error(error.message || '获取集群数据失败');
  } finally {
    loading.value = false;
  }
};

// 刷新数据
const refreshData = () => {
  searchText.value = '';
  filterEnv.value = undefined;
  getClusters();
};

// 搜索处理
const onSearch = (value: string) => {
  searchText.value = value;
};

// 环境筛选变化
const onEnvFilterChange = (value: string) => {
  filterEnv.value = value;
};

// 新增集群
const handleAdd = async () => {
  submitLoading.value = true;
  try {
    const formToSubmit = {
      ...addForm,
      restricted_name_space: addForm.restricted_name_space
    };
    await createClusterApi(formToSubmit);
    message.success('集群新增成功');
    getClusters();
    isAddModalVisible.value = false;
    
    // 重置表单
    Object.keys(addForm).forEach(key => {
      const k = key as keyof ClusterForm;
      if (k === 'env') {
        addForm[k] = 'dev';
      } else if (k === 'action_timeout_seconds') {
        addForm[k] = 60;
      } else if (k === 'restricted_name_space') {
        addForm[k] = [];
      } else {
        addForm[k] = '';
      }
    });
  } catch (error: any) {
    message.error(error.message || '新增集群失败');
  } finally {
    submitLoading.value = false;
  }
};

// 编辑集群
const handleEdit = async (id: number) => {
  loading.value = true;
  try {
    const res = await getClusterApi(id);
    editForm.id = res.id;
    editForm.name = res.name;
    editForm.name_zh = res.name_zh || '';
    editForm.cpu_request = res.cpu_request || '';
    editForm.cpu_limit = res.cpu_limit || '';
    editForm.memory_request = res.memory_request || '';
    editForm.memory_limit = res.memory_limit || '';
    editForm.restricted_name_space = res.restricted_name_space || [];
    editForm.env = res.env || 'dev';
    editForm.version = res.version || '';
    editForm.api_server_addr = res.api_server_addr || '';
    editForm.kube_config_content = res.kube_config_content || '';
    editForm.action_timeout_seconds = res.action_timeout_seconds || 60;
    isEditModalVisible.value = true;
  } catch (error: any) {
    message.error(error.message || '获取集群数据失败');
  } finally {
    loading.value = false;
  }
};

// 更新集群
const handleUpdate = async () => {
  if (!editForm.id) {
    message.error('集群 ID 无效');
    return;
  }
  
  submitLoading.value = true;
  try {
    const formToSubmit = {
      ...editForm,
      restricted_name_space: editForm.restricted_name_space,
    };
    await updateClusterApi(formToSubmit);
    message.success('集群更新成功');
    getClusters();
    isEditModalVisible.value = false;
  } catch (error: any) {
    message.error(error.message || '更新集群失败');
  } finally {
    submitLoading.value = false;
  }
};

// 批量删除确认
const showBatchDeleteConfirm = () => {
  if (selectedRows.value.length === 0) {
    message.warning('请先选择要删除的集群');
    return;
  }
  
  Modal.confirm({
    title: `确定要删除选中的 ${selectedRows.value.length} 个集群吗?`,
    content: '删除后将无法恢复，集群相关配置和资源将被清除。',
    okText: '批量删除',
    okType: 'danger',
    cancelText: '取消',
    async onOk() {
      loading.value = true;
      try {
        for (const row of selectedRows.value) {
          await deleteClusterApi(row.id);
        }
        message.success(`成功删除 ${selectedRows.value.length} 个集群`);
        selectedRows.value = [];
        selectedCardIds.value = [];
        getClusters();
      } catch (error: any) {
        message.error(error.message || '批量删除集群失败');
      } finally {
        loading.value = false;
      }
    },
  });
};

// 删除集群
const handleDelete = async (id: number) => {
  loading.value = true;
  try {
    await deleteClusterApi(id);
    message.success('集群删除成功');
    getClusters();
  } catch (error: any) {
    message.error(error.message || '删除集群失败');
  } finally {
    loading.value = false;
  }
};

// 查看节点
const handleViewNodes = (id: number) => {
  router.push({ name: 'K8sNode', query: { cluster_id: id } });
};

// 关闭模态框
const closeAddModal = () => {
  isAddModalVisible.value = false;
};

const closeEditModal = () => {
  isEditModalVisible.value = false;
};

// 环境颜色映射
const getEnvColor = (env: string): string => {
  const colorMap: Record<string, string> = {
    dev: 'blue',
    prod: 'red',
    stage: 'green',
    rc: 'orange',
    press: 'purple'
  };
  return colorMap[env] || 'default';
};

// 状态颜色映射
const getStatusColor = (status: string): string => {
  if (!status) return 'default';
  
  const statusMap: Record<string, string> = {
    'Running': 'green',
    'Pending': 'orange',
    'Warning': 'orange',
    'Error': 'red',
    'Failed': 'red',
    'Unknown': 'gray'
  };
  
  return statusMap[status] || 'default';
};

// 环境名称映射
const getEnvName = (env: string): string => {
  const nameMap: Record<string, string> = {
    dev: '开发环境',
    prod: '生产环境',
    stage: '阶段环境',
    rc: '发布候选',
    press: '压力测试'
  };
  return nameMap[env] || env;
};

// 状态类型映射
const getStatusType = (status: string): string => {
  if (!status) return 'default';
  
  const statusMap: Record<string, string> = {
    'Running': 'success',
    'Pending': 'processing',
    'Warning': 'warning',
    'Error': 'error',
    'Failed': 'error',
    'Unknown': 'default'
  };
  
  return statusMap[status] || 'default';
};

// 日期格式化
const formatDate = (dateString: string): string => {
  if (!dateString) return '-';
  const date = new Date(dateString);
  return `${date.getMonth() + 1}月${date.getDate()}日 ${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`;
};

// 日期时间格式化
const formatDateTime = (dateString: string): string => {
  if (!dateString) return '-';
  const date = new Date(dateString);
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')} ${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}:${String(date.getSeconds()).padStart(2, '0')}`;
};

onMounted(() => {
  getClusters();
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

.cluster-manager {
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

.env-selector {
  width: 200px;
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

.create-btn {
  background: linear-gradient(135deg, #1890ff 0%, #096dd9 100%);
  border: none;
  height: 36px;
  padding: 0 16px;
  font-weight: 500;
}

.delete-btn {
  background: linear-gradient(135deg, #ff4d4f 0%, #cf1322 100%);
  border: none;
  height: 36px;
  padding: 0 16px;
  font-weight: 500;
}

.env-option {
  display: flex;
  align-items: center;
  gap: 10px;
}

.env-option :deep(svg) {
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

.running-card .metric-icon {
  color: #52c41a;
}

.env-card .metric-icon {
  color: #722ed1;
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

.env-distribution {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 6px;
}

/* 视图切换按钮 */
.view-toggle {
  margin-bottom: 20px;
  text-align: right;
}

.view-toggle :deep(.ant-radio-button-wrapper) {
  padding: 0 16px;
  height: 36px;
  line-height: 34px;
  display: inline-flex;
  align-items: center;
}

.view-toggle :deep(.ant-radio-button-wrapper svg) {
  margin-right: 6px;
}

/* 集群表格样式 */
.cluster-table {
  background: white;
  border-radius: 8px;
  overflow: hidden;
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.05);
}

.cluster-table :deep(.ant-table-thead > tr > th) {
  background-color: #f5f7fa;
  font-weight: 600;
  padding: 14px 16px;
}

.cluster-table :deep(.ant-table-tbody > tr > td) {
  padding: 12px 16px;
}

.cluster-name {
  display: flex;
  align-items: center;
  gap: 10px;
  font-weight: 500;
}

.cluster-name a {
  color: #1890ff;
}

.env-tag {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-weight: 500;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 13px;
}

.status-tag {
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

.timestamp {
  display: flex;
  align-items: center;
  gap: 10px;
  color: #595959;
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

/* 卡片视图容器 */
.card-view {
  background: white;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.05);
}

/* 卡片容器布局优化 - 横向排列 */
.card-checkbox-group {
  display: flex;
  flex-wrap: wrap;
  gap: 30px;
  padding: 10px;
}

/* 卡片样式优化 */
.cluster-card {
  background: white;
  border-radius: 8px;
  box-shadow: 0 2px 10px rgba(0, 0, 0, 0.08);
  transition: transform 0.3s, box-shadow 0.3s;
  overflow: hidden;
  position: relative;
  display: flex;
  flex-direction: column;
  width: 350px;
  border: 1px solid #eaeaea;
  margin-bottom: 20px;
}

.cluster-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.12);
}

/* 卡片头部样式 */
.card-header {
  padding: 16px 20px;
  border-bottom: 1px solid #f0f0f0;
  background-color: #fafafa;
  position: relative;
}

.cluster-title {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-right: 45px;
}

.cluster-title h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: #333;
  word-break: break-all;
  line-height: 1.4;
}

.service-icon {
  font-size: 20px;
  color: #1890ff;
}

.card-type-tag {
  position: absolute;
  top: 12px;
  right: 50px;
  padding: 2px 10px;
}

.card-checkbox {
  position: absolute;
  top: 12px;
  right: 12px;
}

/* 卡片内容区域 */
.card-content {
  padding: 20px;
  flex-grow: 1;
  display: flex;
  flex-direction: column;
  gap: 16px;
  background: #fff;
}

.card-detail {
  display: flex;
  align-items: center;
  line-height: 1.5;
}

.detail-label {
  color: #666;
  min-width: 100px;
  font-size: 14px;
}

.detail-value {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 14px;
  color: #333;
  flex: 1;
}

/* 卡片底部按钮区域 */
.card-action-footer {
  padding: 16px 20px;
  background-color: #f5f7fa;
  border-top: 1px solid #eeeeee;
  display: flex;
  justify-content: space-between;
  gap: 16px;
}

.card-action-footer .ant-btn {
  flex: 1;
  min-width: 80px;
  border-radius: 4px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.card-action-footer .ant-btn svg {
  margin-right: 8px;
}

/* 集群模态框样式 */
.cluster-modal {
  font-family: system-ui, -apple-system, BlinkMacSystemFont, sans-serif;
}

.modal-alert {
  margin-bottom: 16px;
}

.cluster-form {
  padding: 10px;
}

.form-input {
  border-radius: 8px;
  height: 42px;
}

.form-select {
  border-radius: 8px;
  height: 42px;
}

.form-textarea {
  border-radius: 8px;
  line-height: 1.5;
}

/* 空状态样式 */
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 32px 0;
}

/* 响应式调整 */
@media (max-width: 1400px) {
  .card-checkbox-group {
    justify-content: space-around;
  }
  
  .cluster-card {
    width: 320px;
  }
}

@media (max-width: 768px) {
  .card-checkbox-group {
    flex-direction: column;
    align-items: center;
  }
  
  .cluster-card {
    width: 100%;
    max-width: 450px;
  }
  
  .card-action-footer {
    flex-wrap: wrap;
  }
  
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
}
</style>