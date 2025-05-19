<template>
  <div class="service-manager pod-manager">
    <!-- 仪表板标题 -->
    <div class="dashboard-header">
      <h2 class="dashboard-title">
        <CloudServerOutlined class="dashboard-icon" />
        Kubernetes Pod 管理器
      </h2>
      <div class="dashboard-stats">
        <div class="stat-item">
          <div class="stat-value">{{ filteredPods.length }}</div>
          <div class="stat-label">Pods</div>
        </div>
        <div class="stat-item">
          <div class="stat-value">{{ selectedNamespace }}</div>
          <div class="stat-label">命名空间</div>
        </div>
      </div>
    </div>

    <!-- 查询和操作工具栏 -->
    <div class="control-panel">
      <div class="search-filters">
        <a-select
          v-model:value="selectedCluster"
          placeholder="选择集群"
          class="control-item cluster-selector"
          :loading="clustersLoading"
          @change="handleClusterChange"
        >
          <template #suffixIcon><ClusterOutlined /></template>
          <a-select-option v-for="cluster in clusters" :key="cluster.id" :value="cluster.id">
            <span class="cluster-option">
              <CloudServerOutlined />
              {{ cluster.name }}
            </span>
          </a-select-option>
        </a-select>
        
        <a-select
          v-model:value="selectedNamespace"
          placeholder="选择命名空间"
          class="control-item namespace-selector"
          :loading="namespacesLoading"
          @change="handleNamespaceChange"
        >
          <template #suffixIcon><PartitionOutlined /></template>
          <a-select-option v-for="ns in namespaces" :key="ns" :value="ns">
            <span class="namespace-option">
              <AppstoreOutlined />
              {{ ns }}
            </span>
          </a-select-option>
        </a-select>
        
        <a-input-search
          v-model:value="searchText"
          placeholder="搜索 Pod 名称"
          class="control-item search-input"
          @search="onSearch"
          allow-clear
        >
          <template #prefix><SearchOutlined /></template>
        </a-input-search>
      </div>
      
      <div class="action-buttons">
        <a-tooltip title="刷新数据">
          <a-button type="primary" class="refresh-btn" @click="getPods" :loading="loading">
            <template #icon><ReloadOutlined /></template>
          </a-button>
        </a-tooltip>
        
        <a-button 
          type="primary" 
          danger 
          class="delete-btn" 
          @click="handleBatchDelete" 
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
            <div class="metric-value">{{ pods.length }}</div>
          </div>
          <div class="card-title">Pod 总数</div>
        </div>
        <div class="card-footer">
          <div class="footer-text">{{ selectedNamespace }} 命名空间</div>
        </div>
      </div>
      
      <div class="summary-card running-card">
        <div class="card-content">
          <div class="card-metric">
            <CheckCircleOutlined class="metric-icon" />
            <div class="metric-value">{{ runningPodsCount }}</div>
          </div>
          <div class="card-title">运行中 Pods</div>
        </div>
        <div class="card-footer">
          <a-progress 
            :percent="runningPodsPercentage" 
            :stroke-color="{ from: '#1890ff', to: '#52c41a' }" 
            size="small" 
            :show-info="false" 
          />
          <div class="footer-text">{{ runningPodsPercentage }}% 运行正常</div>
        </div>
      </div>
      
      <div class="summary-card problem-card">
        <div class="card-content">
          <div class="card-metric">
            <WarningOutlined class="metric-icon" />
            <div class="metric-value">{{ problemPodsCount }}</div>
          </div>
          <div class="card-title">问题 Pods</div>
        </div>
        <div class="card-footer">
          <a-progress 
            :percent="problemPodsPercentage" 
            status="exception" 
            size="small" 
            :show-info="false"
          />
          <div class="footer-text">{{ problemPodsPercentage }}% 需要关注</div>
        </div>
      </div>
      
      <div class="summary-card cluster-card">
        <div class="card-content">
          <div class="card-metric">
            <ClusterOutlined class="metric-icon" />
            <div class="metric-value cluster-name">{{ selectedClusterName || '未选择' }}</div>
          </div>
          <div class="card-title">当前集群</div>
        </div>
        <div class="card-footer">
          <div class="system-status">
            <span class="status-indicator"></span>
            <span class="status-text">系统在线</span>
          </div>
        </div>
      </div>
    </div>

    <!-- 视图切换 -->
    <div class="view-toggle">
      <a-radio-group v-model:value="viewMode" button-style="solid">
        <a-radio-button value="table">
          <TableOutlined />
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
      :data-source="filteredPods"
      :row-selection="rowSelection"
      :loading="loading"
      row-key="name"
      :pagination="{ 
        pageSize: 10, 
        showSizeChanger: true, 
        showQuickJumper: true,
        showTotal: (total: number) => `共 ${total} 条数据`
      }"
      class="services-table pod-table"
    >
      <!-- Pod名称列 -->
      <template #name="{ text }">
        <div class="pod-name">
          <CodepenOutlined />
          <span>{{ text }}</span>
        </div>
      </template>
      
      <!-- 命名空间列 -->
      <template #namespace="{ text }">
        <a-tag class="namespace-tag">
          <AppstoreOutlined /> {{ text }}
        </a-tag>
      </template>
      
      <!-- 状态列 -->
      <template #status="{ text }">
        <a-tag :color="getPodStatusColor(text)" class="status-tag">
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

      <!-- 创建时间列 -->
      <template #age="{ text }">
        <div class="timestamp">
          <ClockCircleOutlined />
          <span>{{ text }}</span>
        </div>
      </template>

      <!-- 操作列 -->
      <template #action="{ record }">
        <div class="action-column">
          <a-tooltip title="查看 YAML">
            <a-button type="primary" ghost shape="circle" @click="viewPodYaml(record)">
              <template #icon><CodeOutlined /></template>
            </a-button>
          </a-tooltip>
          
          <a-tooltip title="查看日志">
            <a-button type="primary" ghost shape="circle" @click="viewPodLogs(record)">
              <template #icon><FileTextOutlined /></template>
            </a-button>
          </a-tooltip>
          
          <a-tooltip title="删除 Pod">
            <a-popconfirm
              title="确定要删除该 Pod 吗?"
              description="此操作不可撤销"
              @confirm="handleDelete(record)"
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
    </a-table>

    <!-- 卡片视图 -->
    <div v-else class="card-view">
      <a-spin :spinning="loading">
        <a-empty v-if="filteredPods.length === 0" description="暂无 Pod 数据" />
        <div v-else class="service-cards pod-cards">
          <a-checkbox-group v-model:value="selectedCardIds" class="card-checkbox-group">
            <div v-for="pod in filteredPods" :key="pod.name" class="service-card pod-card">
              <div class="card-header">
                <a-checkbox :value="pod.name" class="card-checkbox" />
                <div class="service-title pod-title">
                  <CodepenOutlined class="service-icon" />
                  <h3>{{ pod.name }}</h3>
                </div>
                <a-tag :color="getPodStatusColor(pod.status)" class="card-type-tag status-tag">
                  <span class="status-dot"></span>
                  {{ pod.status }}
                </a-tag>
              </div>
              
              <div class="card-content">
                <div class="card-detail namespace-detail">
                  <span class="detail-label">命名空间:</span>
                  <span class="detail-value">
                    <AppstoreOutlined />
                    {{ pod.namespace }}
                  </span>
                </div>
                <div class="card-detail ip-detail">
                  <span class="detail-label">IP地址:</span>
                  <span class="detail-value">
                    <GlobalOutlined />
                    {{ pod.ip }}
                  </span>
                </div>
                <div class="card-detail age-detail">
                  <span class="detail-label">创建时间:</span>
                  <span class="detail-value">
                    <ClockCircleOutlined />
                    {{ pod.age }}
                  </span>
                </div>
                <div class="card-detail containers-detail">
                  <span class="detail-label">容器数量:</span>
                  <span class="detail-value">{{ pod.containers?.length || 0 }}</span>
                </div>
              </div>
              
              <div class="card-footer card-action-footer">
                <a-button type="primary" ghost size="small" @click="viewPodYaml(pod)">
                  <template #icon><CodeOutlined /></template>
                  YAML
                </a-button>
                <a-button type="primary" ghost size="small" @click="viewPodLogs(pod)">
                  <template #icon><FileTextOutlined /></template>
                  日志
                </a-button>
                <a-popconfirm
                  title="确定要删除该 Pod 吗?"
                  @confirm="handleDelete(pod)"
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

    <!-- Pod YAML 模态框 -->
    <a-modal
      v-model:visible="yamlModalVisible"
      title="Pod YAML 配置"
      width="800px"
      class="yaml-modal"
      :footer="null"
    >
      <a-alert v-if="selectedPod" class="yaml-info" type="info" show-icon>
        <template #message>
          <span>{{ selectedPod.name }} ({{ selectedPod.namespace }})</span>
        </template>
        <template #description>
          <div>状态: {{ selectedPod.status }} | IP: {{ selectedPod.ip }}</div>
        </template>
      </a-alert>
      <div class="yaml-actions">
        <a-button type="primary" size="small" @click="copyYaml">
          <template #icon><CopyOutlined /></template>
          复制
        </a-button>
      </div>
      <pre class="yaml-editor">{{ podYaml }}</pre>
    </a-modal>

    <!-- Pod 日志查看模态框 -->
    <a-modal
      v-model:visible="logModalVisible" 
      title="Pod 日志查看"
      width="800px"
      :footer="null"
      class="yaml-modal logs-modal"
    >
      <a-alert v-if="selectedPod" class="yaml-info" type="info" show-icon>
        <template #message>
          <span>{{ selectedPod.name }} ({{ selectedPod.namespace }})</span>
        </template>
        <template #description>
          <div>状态: {{ selectedPod.status }}</div>
        </template>
      </a-alert>
      
      <div class="logs-toolbar">
        <a-select
          v-model:value="selectedContainer"
          class="container-select"
          placeholder="选择容器"
          @change="handleContainerChange"
        >
          <template #suffixIcon><ContainerOutlined /></template>
          <a-select-option v-for="container in containers" :key="container" :value="container">
            <span class="container-option">
              <ContainerOutlined />
              {{ container }}
            </span>
          </a-select-option>
        </a-select>
        
        <a-button type="primary" @click="fetchPodLogs" :disabled="!selectedContainer" class="logs-refresh-btn">
          <template #icon><SyncOutlined /></template>
          刷新日志
        </a-button>
        
        <a-button type="primary" @click="copyLogs" :disabled="!podLogs">
          <template #icon><CopyOutlined /></template>
          复制
        </a-button>
      </div>
      
      <a-spin :spinning="logsLoading">
        <div class="logs-container">
          <template v-if="podLogs">
            <div class="logs-lines">
              <div v-for="(line, index) in podLogs.split('\n')" :key="index" class="log-line">
                <span class="line-number">{{ index + 1 }}</span>
                <span class="line-content">{{ line }}</span>
              </div>
            </div>
          </template>
          <a-empty v-else description="选择容器并获取日志" />
        </div>
      </a-spin>
    </a-modal>
  </div>
</template>

<script lang="ts" setup>
import { ref, computed, onMounted, watch } from 'vue';
import { message } from 'ant-design-vue';
import {
  getPodsByNamespaceApi,
  getContainersByPodNameApi,
  getContainerLogsApi,
  getPodYamlApi,
  deletePodApi,
  getNamespacesByClusterIdApi,
  getAllClustersApi
} from '#/api';
import { 
  SyncOutlined,
  DeleteOutlined,
  SearchOutlined,
  CloudServerOutlined, 
  TableOutlined, 
  AppstoreOutlined, 
  ReloadOutlined,
  CodepenOutlined,
  EyeOutlined, 
  CodeOutlined,
  WarningOutlined,
  ApiOutlined,
  ClockCircleOutlined,
  CheckCircleOutlined,
  CopyOutlined,
  ClusterOutlined,
  PartitionOutlined,
  DashboardOutlined,
  FileTextOutlined,
  ContainerOutlined,
  GlobalOutlined
} from '@ant-design/icons-vue';

// 类型定义
interface Pod {
  name: string;
  namespace: string;
  status: string;
  containers: string[];
  age: string;
  ip: string;
}

// 状态变量
const loading = ref(false);
const logsLoading = ref(false);
const clustersLoading = ref(false);
const namespacesLoading = ref(false);
const pods = ref<Pod[]>([]);
const searchText = ref('');
const selectedRows = ref<Pod[]>([]);
const namespaces = ref<string[]>(['default']); 
const selectedNamespace = ref('default');
const yamlModalVisible = ref(false);
const logModalVisible = ref(false);
const podYaml = ref('');
const podLogs = ref('');
const selectedPod = ref<Pod | null>(null);
const selectedContainer = ref('');
const containers = ref<string[]>([]);
const clusters = ref<Array<{id: number, name: string}>>([]);
const selectedCluster = ref<number>();
const viewMode = ref<'table' | 'card'>('table');
const selectedCardIds = ref<string[]>([]);

// 表格列配置
const columns = [
  {
    title: 'Pod 名称',
    dataIndex: 'name',
    key: 'name',
    sorter: (a: Pod, b: Pod) => a.name.localeCompare(b.name),
    slots: { customRender: 'name' },
  },
  {
    title: '命名空间',
    dataIndex: 'namespace',
    key: 'namespace',
    slots: { customRender: 'namespace' },
  },
  {
    title: '状态',
    dataIndex: 'status',
    key: 'status',
    slots: { customRender: 'status' },
    filters: [
      { text: 'Running', value: 'Running' },
      { text: 'Pending', value: 'Pending' },
      { text: 'Failed', value: 'Failed' },
      { text: 'Succeeded', value: 'Succeeded' },
      { text: 'Unknown', value: 'Unknown' },
    ],
    onFilter: (value: string, record: Pod) => record.status === value,
  },
  {
    title: 'IP地址',
    dataIndex: 'ip',
    key: 'ip',
    slots: { customRender: 'ip' },
  },
  {
    title: '创建时间',
    dataIndex: 'age',
    key: 'age',
    sorter: (a: Pod, b: Pod) => a.age.localeCompare(b.age),
    slots: { customRender: 'age' },
  },
  {
    title: '操作',
    key: 'action',
    slots: { customRender: 'action' },
    fixed: 'right',
    width: 150,
  },
];

// 计算属性
const filteredPods = computed(() => {
  const searchValue = searchText.value.toLowerCase().trim();
  return pods.value.filter(pod => pod.name.toLowerCase().includes(searchValue));
});

const selectedClusterName = computed(() => {
  const cluster = clusters.value.find(c => c.id === selectedCluster.value);
  return cluster ? cluster.name : '';
});

const runningPodsCount = computed(() => 
  pods.value.filter(pod => pod.status === 'Running').length
);

const runningPodsPercentage = computed(() => 
  pods.value.length > 0 ? Math.round((runningPodsCount.value / pods.value.length) * 100) : 0
);

const problemPodsCount = computed(() => 
  pods.value.filter(pod => ['Failed', 'Unknown', 'Pending'].includes(pod.status)).length
);

const problemPodsPercentage = computed(() => 
  pods.value.length > 0 ? Math.round((problemPodsCount.value / pods.value.length) * 100) : 0
);

// 根据卡片选择更新 selectedRows
watch(selectedCardIds, (newValue) => {
  selectedRows.value = pods.value.filter(pod => 
    newValue.includes(pod.name)
  );
});

// 表格选择配置
const rowSelection = {
  onChange: (selectedRowKeys: string[], selectedRowsData: Pod[]) => {
    selectedRows.value = selectedRowsData;
    selectedCardIds.value = selectedRowsData.map(row => row.name);
  },
  getCheckboxProps: (record: Pod) => ({
    disabled: false, // 可以根据条件禁用某些行的选择
  }),
};

// 获取Pod状态对应的颜色
const getPodStatusColor = (status: string) => {
  const statusColors: Record<string, string> = {
    Running: 'green',
    Pending: 'orange',
    Failed: 'red',
    Succeeded: 'blue',
    Unknown: 'gray',
  };
  return statusColors[status] || 'default';
};

// 获取Pod列表
const getPods = async () => {
  if (!selectedCluster.value) {
    message.warning('请先选择集群');
    return;
  }
  loading.value = true;
  try {
    const res = await getPodsByNamespaceApi(selectedCluster.value, selectedNamespace.value);
    pods.value = res || [];
    selectedRows.value = [];
    selectedCardIds.value = [];
  } catch (error: any) {
    message.error(error.message || '获取Pod列表失败');
  } finally {
    loading.value = false;
  }
};

const getNamespaces = async () => {
  if (!selectedCluster.value) {
    message.warning('请先选择集群');
    return;
  }

  namespacesLoading.value = true;
  try {
    const res = await getNamespacesByClusterIdApi(selectedCluster.value);
    if (!res) {
      throw new Error('获取命名空间数据为空');
    }

    // 只获取name字段组成新数组
    namespaces.value = res.map((ns: { name: string }) => ns.name);

    // 如果没有选中的命名空间,默认选择第一个
    if (namespaces.value.length > 0) {
      selectedNamespace.value = selectedNamespace.value ?? namespaces.value[0];
    }
  } catch (error: any) {
    message.error(error.message || '获取命名空间列表失败');
    namespaces.value = ['default'];
    selectedNamespace.value = 'default';
  } finally {
    namespacesLoading.value = false;
  }
};

const getClusters = async () => {
  clustersLoading.value = true;
  try {
    const res = await getAllClustersApi();
    clusters.value = res || [];
    
    // 如果有集群数据，默认选择第一个
    if (clusters.value.length > 0 && clusters.value[0]?.id) {
      selectedCluster.value = clusters.value[0].id;
      await getNamespaces();
      await getPods();
    }
  } catch (error: any) {
    message.error(error.message || '获取集群列表失败');
    clusters.value = [];
  } finally {
    clustersLoading.value = false;
  }
};

// 复制YAML
const copyYaml = async () => {
  try {
    await navigator.clipboard.writeText(podYaml.value);
    message.success('YAML 已复制到剪贴板');
  } catch (err) {
    message.error('复制失败，请手动选择并复制');
  }
};

// 复制日志
const copyLogs = async () => {
  try {
    await navigator.clipboard.writeText(podLogs.value);
    message.success('日志已复制到剪贴板');
  } catch (err) {
    message.error('复制失败，请手动选择并复制');
  }
};

// 搜索
const onSearch = () => {
  // 搜索逻辑已经在计算属性中实现，这里可以添加其他触发行为
};

// 查看Pod YAML
const viewPodYaml = async (pod: Pod) => {
  if (!selectedCluster.value) return;
  selectedPod.value = pod;
  yamlModalVisible.value = true;
  podYaml.value = '加载中...';
  
  try {
    const res = await getPodYamlApi(selectedCluster.value, pod.name, pod.namespace);
    podYaml.value = res;
  } catch (error: any) {
    message.error(error.message || '获取Pod YAML失败');
    podYaml.value = '加载失败';
  }
};

// 查看Pod日志
const viewPodLogs = async (pod: Pod) => {
  if (!selectedCluster.value) return;
  selectedPod.value = pod;
  logModalVisible.value = true;
  podLogs.value = '';
  selectedContainer.value = '';
  logsLoading.value = true;
  
  try {
    // 获取容器列表
    const res = await getContainersByPodNameApi(selectedCluster.value, pod.name, pod.namespace);
    if (res) {
      containers.value = res.map((container: { name: string }) => container.name);
      
      // 如果有容器，自动选择第一个并获取日志
      if (containers.value.length > 0) {
        selectedContainer.value = containers.value[0] ?? '';
        await fetchPodLogs();
      }
    }
  } catch (error: any) {
    message.error(error.message || '获取容器列表失败');
  } finally {
    logsLoading.value = false;
  }
};

// 获取Pod日志
const fetchPodLogs = async () => {
  if (!selectedPod.value || !selectedContainer.value || !selectedCluster.value) return;
  
  logsLoading.value = true;
  try {
    const logs = await getContainerLogsApi(
      selectedCluster.value,
      selectedPod.value.name, 
      selectedContainer.value,
      selectedPod.value.namespace
    );
    podLogs.value = logs || '暂无日志';
  } catch (error: any) {
    message.error(error.message || '获取容器日志失败');
    podLogs.value = '';
  } finally {
    logsLoading.value = false;
  }
};

// 切换容器时重新获取日志
const handleContainerChange = () => {
  podLogs.value = '';
  fetchPodLogs();
};

// 切换命名空间
const handleNamespaceChange = () => {
  getPods();
};

// 切换集群
const handleClusterChange = () => {
  getNamespaces();
  getPods();
};

// 删除Pod
const handleDelete = async (pod: Pod) => {
  if (!selectedCluster.value) return;
  try {
    await deletePodApi(selectedCluster.value, pod.name, pod.namespace);
    message.success(`Pod ${pod.name} 删除成功`);
    await getPods(); // 删除成功后立即刷新数据
  } catch (error: any) {
    message.error(error.message || '删除Pod失败');
  }
};

// 批量删除Pod
const handleBatchDelete = async () => {
  if (!selectedRows.value.length || !selectedCluster.value) return;
  
  try {
    loading.value = true;
    const promises = selectedRows.value.map(pod => 
      deletePodApi(selectedCluster.value!, pod.name, pod.namespace)
    );
    
    await Promise.all(promises);
    message.success(`成功删除 ${selectedRows.value.length} 个Pod`);
    selectedRows.value = [];
    selectedCardIds.value = [];
    await getPods(); // 删除成功后立即刷新数据
  } catch (error: any) {
    message.error(error.message || '批量删除失败');
  } finally {
    loading.value = false;
  }
};

// 页面加载时获取数据
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

.pod-manager {
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

.delete-btn {
  background: linear-gradient(135deg, #ff4d4f 0%, #cf1322 100%);
  border: none;
  height: 36px;
  padding: 0 16px;
  font-weight: 500;
}

.cluster-option, .namespace-option, .container-option {
  display: flex;
  align-items: center;
  gap: 10px;
}

.cluster-option :deep(svg), .namespace-option :deep(svg), .container-option :deep(svg) {
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

.problem-card .metric-icon {
  color: #f5222d;
}

.cluster-card .metric-icon {
  color: #722ed1;
}

.cluster-name {
  font-size: 22px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 200px;
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

.system-status {
  display: flex;
  align-items: center;
  gap: 10px;
}

.status-indicator {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background-color: #52c41a;
  display: inline-block;
}

.status-text {
  font-size: 13px;
  color: #52c41a;
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

/* Pod表格样式 */
.pod-table {
  background: white;
  border-radius: 8px;
  overflow: hidden;
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.05);
}

.pod-table :deep(.ant-table-thead > tr > th) {
  background-color: #f5f7fa;
  font-weight: 600;
  padding: 14px 16px;
}

.pod-table :deep(.ant-table-tbody > tr > td) {
  padding: 12px 16px;
}

.pod-name {
  display: flex;
  align-items: center;
  gap: 10px;
  font-weight: 500;
}

.namespace-tag {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background: #e6f7ff;
  color: #1890ff;
  border: 1px solid #91d5ff;
  font-size: 13px;
  padding: 2px 8px;
  border-radius: 4px;
}

.status-tag {
  font-weight: 500;
  display: inline-flex;
  align-items: center;
  gap: 6px;
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
.pod-card, .service-card {
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

.pod-card:hover, .service-card:hover {
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

.pod-title, .service-title {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-right: 45px;
}

.pod-title h3, .service-title h3 {
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

/* 状态标签样式 */
.status-tag {
  font-weight: 500;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 3px 10px;
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

/* YAML模态框 */
.yaml-modal {
  font-family: "Consolas", "Monaco", monospace;
}

.yaml-info {
  margin-bottom: 16px;
}

.yaml-actions {
  display: flex;
  justify-content: flex-end;
  margin-bottom: 10px;
}

.yaml-editor {
  background-color: #f5f5f5;
  padding: 16px;
  border-radius: 4px;
  height: 400px;
  overflow: auto;
  font-size: 13px;
  white-space: pre-wrap;
  font-family: "Consolas", "Monaco", monospace;
}

/* 日志模态框 */
.logs-modal .logs-toolbar {
  display: flex;
  gap: 16px;
  margin-bottom: 16px;
}

.logs-modal .container-select {
  flex: 1;
}

.logs-container {
  background-color: #1e1e1e;
  color: #f1f1f1;
  border-radius: 4px;
  height: 400px;
  overflow: auto;
  font-family: "Consolas", "Monaco", monospace;
  font-size: 13px;
}

.logs-lines {
  padding: 8px 0;
}

.log-line {
  display: flex;
  padding: 2px 8px;
}

.log-line:hover {
  background-color: #333;
}

.line-number {
  color: #888;
  min-width: 50px;
  text-align: right;
  padding-right: 16px;
  user-select: none;
}

.line-content {
  white-space: pre-wrap;
  word-break: break-all;
}

/* 响应式调整 */
@media (max-width: 1400px) {
  .card-checkbox-group {
    justify-content: space-around;
  }
  
  .pod-card, .service-card {
    width: 320px;
  }
}

@media (max-width: 768px) {
  .card-checkbox-group {
    flex-direction: column;
    align-items: center;
  }
  
  .pod-card, .service-card {
    width: 100%;
    max-width: 450px;
  }
  
  .card-action-footer {
    flex-wrap: wrap;
  }
}
</style>