<template>
  <div class="cluster-management-dashboard">
    <!-- 顶部统计卡片 -->
    <div class="dashboard-stats">
      <a-card class="stat-card" :bordered="false">
        <template #title>
          <span class="stat-title"><cluster-outlined /> 集群总数</span>
        </template>
        <div class="stat-content">
          <div class="stat-value">{{ clusters.length }}</div>
          <div class="stat-trend">
            <rise-outlined v-if="clusters.length > 0" style="color: #52c41a" /> 在线运行中
          </div>
        </div>
      </a-card>
      
      <a-card class="stat-card" :bordered="false">
        <template #title>
          <span class="stat-title"><environment-outlined /> 环境分布</span>
        </template>
        <div class="stat-chart">
          <div v-for="(count, env) in envDistribution" :key="env" class="env-badge">
            <a-tag :color="getEnvColor(env)">{{ getEnvName(env) }}: {{ count }}</a-tag>
          </div>
        </div>
      </a-card>
      
      <a-card class="stat-card" :bordered="false">
        <template #title>
          <span class="stat-title"><api-outlined /> 资源状态</span>
        </template>
        <div class="stat-content">
          <a-progress :percent="resourceUtilization" status="active" />
          <div class="resource-details">资源使用率</div>
        </div>
      </a-card>
    </div>

    <!-- 工具栏和筛选器 -->
    <div class="action-toolbar">
      <!-- 查询功能 -->
      <div class="search-area">
        <a-input-search
          v-model:value="searchText"
          placeholder="搜索集群名称..."
          allow-clear
          class="search-input"
          @search="onSearch"
        >
          <template #prefix>
            <search-outlined />
          </template>
        </a-input-search>
        
        <a-select
          v-model:value="filterEnv"
          placeholder="环境筛选"
          style="width: 120px"
          allow-clear
          @change="onEnvFilterChange"
        >
          <a-select-option value="dev">开发环境</a-select-option>
          <a-select-option value="prod">生产环境</a-select-option>
          <a-select-option value="stage">阶段环境</a-select-option>
          <a-select-option value="rc">发布候选</a-select-option>
          <a-select-option value="press">压力测试</a-select-option>
        </a-select>
      </div>

      <!-- 操作按钮 -->
      <div class="action-buttons">
        <a-button type="primary" @click="isAddModalVisible = true">
          <template #icon><plus-outlined /></template>
          新增集群
        </a-button>
        <a-button 
          type="primary" 
          danger 
          ghost
          :disabled="selectedRows.length === 0"
          @click="showBatchDeleteConfirm"
        >
          <template #icon><delete-outlined /></template>
          批量删除
        </a-button>
        <a-button type="primary" ghost @click="refreshData">
          <template #icon><reload-outlined /></template>
          刷新
        </a-button>
      </div>
    </div>

    <!-- 卡片视图/表格切换 -->
    <div class="view-toggle">
      <a-radio-group v-model:value="viewMode" button-style="solid">
        <a-radio-button value="table">
          <unordered-list-outlined />
          表格视图
        </a-radio-button>
        <a-radio-button value="card">
          <appstore-outlined />
          卡片视图
        </a-radio-button>
      </a-radio-group>
    </div>

    <!-- 表格视图 -->
    <div v-if="viewMode === 'table'" class="table-view">
      <a-spin :spinning="loading">
        <a-table 
          :dataSource="filteredData" 
          :rowSelection="rowSelection"
          :rowKey="(record: any) => record.id"
          :pagination="{ 
            pageSize: 10,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total: number) => `共 ${total} 条数据`
          }"
          :scroll="{ x: 1200 }"
          bordered
        >
          <!-- 使用 v-slot 代替 column.slots -->
          <a-table-column key="id" title="ID" data-index="id" width="80" />
          
          <a-table-column key="name" title="集群名称" data-index="name" width="180" 
            :sorter="(a: any, b: any) => a.name.localeCompare(b.name)">
            <template #default="{ record }">
              <a-tooltip :title="record.name">
                <a @click="handleViewNodes(record.id)" class="cluster-name">
                  {{ record.name }}
                </a>
              </a-tooltip>
            </template>
          </a-table-column>
          
          <a-table-column key="name_zh" title="集群中文名称" data-index="name_zh" width="150" />
          
          <a-table-column key="env" title="所属环境" data-index="env" width="120"
            :filters="[
              { text: '开发环境', value: 'dev' },
              { text: '生产环境', value: 'prod' },
              { text: '阶段环境', value: 'stage' },
              { text: '发布候选', value: 'rc' },
              { text: '压力测试', value: 'press' },
            ]"
            :onFilter="(value: string, record: any) => record.env === value"
          >
            <template #default="{ record }">
              <a-tag :color="getEnvColor(record.env)">
                {{ getEnvName(record.env) }}
              </a-tag>
            </template>
          </a-table-column>
          
          <a-table-column key="status" title="集群状态" data-index="status" width="120">
            <template #default="{ record }">
              <a-badge :status="getStatusType(record.status)" :text="record.status" />
            </template>
          </a-table-column>
          
          <a-table-column key="user_id" title="创建用户" data-index="user_id" width="120" />
          
          <a-table-column key="version" title="集群版本" data-index="version" width="120" />
          
          <a-table-column key="created_at" title="创建时间" data-index="created_at" width="150"
            :sorter="(a: any, b: any) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime()"
          >
            <template #default="{ record }">
              <a-tooltip :title="formatDateTime(record.created_at)">
                {{ formatDate(record.created_at) }}
              </a-tooltip>
            </template>
          </a-table-column>
          
          <a-table-column key="action" title="操作" fixed="right" width="150">
            <template #default="{ record }">
              <div class="action-column">
                <a-tooltip title="编辑集群">
                  <a-button type="primary" shape="circle" size="small" @click="handleEdit(record.id)">
                    <template #icon><edit-outlined /></template>
                  </a-button>
                </a-tooltip>
                
                <a-tooltip title="查看节点">
                  <a-button type="primary" shape="circle" size="small" @click="handleViewNodes(record.id)">
                    <template #icon><eye-outlined /></template>
                  </a-button>
                </a-tooltip>
                
                <a-popconfirm
                  title="确定删除这个集群吗?"
                  ok-text="删除"
                  cancel-text="取消"
                  @confirm="handleDelete(record.id)"
                >
                  <a-tooltip title="删除集群">
                    <a-button type="primary" danger shape="circle" size="small">
                      <template #icon><delete-outlined /></template>
                    </a-button>
                  </a-tooltip>
                </a-popconfirm>
              </div>
            </template>
          </a-table-column>
        </a-table>
      </a-spin>
    </div>

    <!-- 卡片视图 -->
    <div v-else-if="viewMode === 'card'" class="card-view">
      <a-spin :spinning="loading">
        <a-empty v-if="filteredData.length === 0" description="暂无数据" />
        <div v-else class="cluster-cards">
          <a-card 
            v-for="item in filteredData" 
            :key="item.id" 
            class="cluster-card"
            :hoverable="true"
          >
            <template #title>
              <div class="card-title">
                <span>{{ item.name }}</span>
                <a-tag :color="getEnvColor(item.env)">{{ getEnvName(item.env) }}</a-tag>
              </div>
            </template>
            <template #extra>
              <a-dropdown>
                <template #overlay>
                  <a-menu>
                    <a-menu-item key="1" @click="handleEdit(item.id)">
                      <edit-outlined /> 编辑集群
                    </a-menu-item>
                    <a-menu-item key="2" @click="handleViewNodes(item.id)">
                      <eye-outlined /> 查看节点
                    </a-menu-item>
                    <a-menu-divider />
                    <a-menu-item key="3" danger @click="showDeleteConfirm(item.id)">
                      <delete-outlined /> 删除集群
                    </a-menu-item>
                  </a-menu>
                </template>
                <more-outlined style="font-size: 18px" />
              </a-dropdown>
            </template>
            
            <div class="card-content">
              <p><strong>集群ID:</strong> {{ item.id }}</p>
              <p><strong>中文名称:</strong> {{ item.name_zh || '-' }}</p>
              <p><strong>版本:</strong> {{ item.version || '-' }}</p>
              <p><strong>状态:</strong> <a-badge :status="getStatusType(item.env)" :text="getEnvName(item.env)" /></p>
              <p><strong>创建时间:</strong> {{ formatDate(item.created_at) }}</p>
            </div>
            
            <template #actions>
              <div class="card-actions">
                <a-button type="primary" size="small" @click="handleViewNodes(item.id)">
                  <template #icon><cluster-outlined /></template>
                  查看节点
                </a-button>
                <a-button type="primary" ghost size="small" @click="handleEdit(item.id)">
                  <template #icon><edit-outlined /></template>
                  编辑
                </a-button>
              </div>
            </template>
          </a-card>
        </div>
      </a-spin>
    </div>

    <!-- 新增集群模态框 - 使用 open 替代 visible -->
    <a-modal
      title="新增集群"
      v-model:open="isAddModalVisible"
      :width="700"
      @ok="handleAdd"
      @cancel="closeAddModal"
      :confirmLoading="submitLoading"
    >
      <a-form :model="addForm" layout="vertical">
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="集群名称" name="name" :rules="[{ required: true, message: '请输入集群名称' }]">
              <a-input v-model:value="addForm.name" placeholder="请输入集群名称" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="集群中文名称" name="name_zh">
              <a-input v-model:value="addForm.name_zh" placeholder="请输入集群中文名称" />
            </a-form-item>
          </a-col>
        </a-row>
        
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="环境" name="env" :rules="[{ required: true, message: '请选择环境' }]">
              <a-select v-model:value="addForm.env" placeholder="请选择环境">
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
              <a-input v-model:value="addForm.version" placeholder="请输入集群版本" />
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
          ></a-select>
        </a-form-item>
        
        <a-form-item label="API 服务器地址" name="api_server_addr">
          <a-input v-model:value="addForm.api_server_addr" placeholder="请输入 API 服务器地址" />
        </a-form-item>
        
        <a-form-item label="KubeConfig 内容" name="kube_config_content">
          <a-textarea
            v-model:value="addForm.kube_config_content"
            placeholder="请输入 KubeConfig 内容"
            :auto-size="{ minRows: 4, maxRows: 8 }"
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

    <!-- 编辑集群模态框 - 使用 open 替代 visible -->
    <a-modal
      title="编辑集群"
      v-model:open="isEditModalVisible"
      :width="700"
      @ok="handleUpdate"
      @cancel="closeEditModal"
      :confirmLoading="submitLoading"
    >
      <a-form :model="editForm" layout="vertical">
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="集群名称" name="name" :rules="[{ required: true, message: '请输入集群名称' }]">
              <a-input v-model:value="editForm.name" placeholder="请输入集群名称" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="集群中文名称" name="name_zh">
              <a-input v-model:value="editForm.name_zh" placeholder="请输入集群中文名称" />
            </a-form-item>
          </a-col>
        </a-row>
        
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="环境" name="env" :rules="[{ required: true, message: '请选择环境' }]">
              <a-select v-model:value="editForm.env" placeholder="请选择环境">
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
              <a-input v-model:value="editForm.version" placeholder="请输入集群版本" />
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
          ></a-select>
        </a-form-item>
        
        <a-form-item label="API 服务器地址" name="api_server_addr">
          <a-input v-model:value="editForm.api_server_addr" placeholder="请输入 API 服务器地址" />
        </a-form-item>
        
        <a-form-item label="KubeConfig 内容" name="kube_config_content">
          <a-textarea
            v-model:value="editForm.kube_config_content"
            placeholder="请输入 KubeConfig 内容"
            :auto-size="{ minRows: 4, maxRows: 8 }"
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
import { ref, reactive, computed, onMounted } from 'vue';
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
  MoreOutlined,
  UnorderedListOutlined,
  AppstoreOutlined,
  RiseOutlined // 使用 RiseOutlined 替代不存在的 TrendUpOutlined
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

// 批量选择配置
const rowSelection = {
  selectedRowKeys: computed(() => selectedRows.value.map((row: ClustersItem) => row.id)),
  onChange: (selectedRowKeys: any[], selectedRowData: ClustersItem[]) => {
    selectedRows.value = selectedRowData;
  },
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

// 删除确认对话框
const showDeleteConfirm = (id: number) => {
  Modal.confirm({
    title: '确定要删除此集群吗?',
    content: '删除后将无法恢复，集群相关配置和资源将被清除。',
    okText: '删除',
    okType: 'danger',
    cancelText: '取消',
    async onOk() {
      await handleDelete(id);
    },
  });
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
  return `${date.getFullYear()}-${(date.getMonth() + 1).toString().padStart(2, '0')}-${date.getDate().toString().padStart(2, '0')}`;
};

// 日期时间格式化
const formatDateTime = (dateString: string): string => {
  if (!dateString) return '-';
  const date = new Date(dateString);
  return `${formatDate(dateString)} ${date.getHours().toString().padStart(2, '0')}:${date.getMinutes().toString().padStart(2, '0')}:${date.getSeconds().toString().padStart(2, '0')}`;
};

onMounted(() => {
  console.log('Page mounted, fetching clusters...');
  getClusters();
});
</script>

<style scoped>
.cluster-management-dashboard {
  background-color: #f0f2f5;
  padding: 20px;
  min-height: 100vh;
}

.dashboard-stats {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 16px;
  margin-bottom: 24px;
}

.stat-card {
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.05);
  transition: all 0.3s;
}

.stat-card:hover {
  transform: translateY(-3px);
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.09);
}

.stat-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 15px;
  color: #555;
}

.stat-content {
  padding: 8px 0;
}

.stat-value {
  font-size: 28px;
  font-weight: 600;
  color: #1890ff;
  margin-bottom: 8px;
}

.stat-trend {
  font-size: 13px;
  color: #52c41a;
  display: flex;
  align-items: center;
  gap: 4px;
}

.stat-chart {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 8px;
}

.env-badge {
  margin-bottom: 4px;
}

.resource-details {
  margin-top: 8px;
  font-size: 13px;
  color: #666;
}

.action-toolbar {
  background: white;
  border-radius: 8px;
  padding: 16px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
}

.search-area {
  display: flex;
  align-items: center;
  gap: 12px;
}

.search-input {
  width: 250px;
  border-radius: 4px;
}

.action-buttons {
  display: flex;
  gap: 8px;
}

.view-toggle {
  margin-bottom: 16px;
  display: flex;
  justify-content: flex-end;
}

.table-view {
  background: white;
  border-radius: 8px;
  padding: 16px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
}

.action-column {
  display: flex;
  gap: 8px;
}

.card-view {
  margin-top: 16px;
}

.cluster-cards {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 16px;
}

.cluster-card {
  border-radius: 8px;
  overflow: hidden;
  transition: all 0.3s;
  height: 100%;
}

.cluster-card:hover {
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.1);
  transform: translateY(-4px);
}

.card-title {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-weight: 500;
}

.card-content {
  padding: 8px 0;
  font-size: 14px;
}

.card-content p {
  margin-bottom: 8px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.card-actions {
  display: flex;
  gap: 8px;
  justify-content: center;
}

.cluster-name {
  color: #1890ff;
  font-weight: 500;
  transition: color 0.3s;
}

.cluster-name:hover {
  color: #40a9ff;
  text-decoration: underline;
}

:deep(.ant-modal-content) {
  border-radius: 8px;
  overflow: hidden;
}

:deep(.ant-table-tbody > tr.ant-table-row:hover > td) {
  background-color: rgba(24, 144, 255, 0.05);
}

:deep(.ant-table-column-sorter) {
  color: #bfbfbf;
}

:deep(.ant-form-item-label > label) {
  font-weight: 500;
}

:deep(.ant-divider-inner-text) {
  font-size: 13px;
  color: #888;
}
</style>