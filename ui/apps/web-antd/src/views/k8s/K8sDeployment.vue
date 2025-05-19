<template>
  <div class="deployment-manager">
    <!-- 仪表板标题 -->
    <div class="dashboard-header">
      <h2 class="dashboard-title">Deployment 资源管理器</h2>
      <div class="dashboard-stats">
        <div class="stat-item">
          <div class="stat-value">{{ filteredDeployments.length }}</div>
          <div class="stat-label">部署</div>
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
          placeholder="搜索 Deployment 名称"
          class="control-item search-input"
          @search="onSearch"
          allow-clear
        >
          <template #prefix><SearchOutlined /></template>
        </a-input-search>
      </div>
      
      <div class="action-buttons">
        <a-tooltip title="刷新数据">
          <a-button type="primary" class="refresh-btn" @click="refreshData" :loading="loading">
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

    <!-- Deployment 卡片/表格切换视图 -->
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
      :data-source="filteredDeployments"
      :row-selection="rowSelection"
      :loading="loading"
      row-key="uid"
      :pagination="{ 
        pageSize: 10, 
        showSizeChanger: true, 
        showQuickJumper: true,
        showTotal: (total: number) => `共 ${total} 条数据`
      }"
      class="deployments-table"
    >
      <!-- Deployment名称列 -->
      <template #name="{ text, record }">
        <div class="deployment-name">
          <RocketOutlined />
          <span>{{ text }}</span>
        </div>
      </template>
      
      <!-- 状态列 -->
      <template #status="{ record }">
        <div class="status-wrapper">
          <a-tag :color="getStatusColor(record)" class="status-tag">
            {{ record.status?.availableReplicas || 0 }}/{{ record.status?.replicas || 0 }} 副本
          </a-tag>
          <a-progress 
            :percent="getStatusPercent(record)" 
            :status="getProgressStatus(record)"
            size="small"
            :stroke-color="getStatusColor(record)"
            :show-info="false"
            class="status-progress"
          />
        </div>
      </template>

      <!-- 镜像列 -->
      <template #image="{ text }">
        <div class="image-tag">
          <ContainerOutlined />
          <span class="image-text">{{ text }}</span>
        </div>
      </template>

      <!-- 创建时间列 -->
      <template #creationTimestamp="{ text }">
        <div class="timestamp">
          <ClockCircleOutlined />
          <span>{{ formatDate(text) }}</span>
          <a-tooltip :title="getRelativeTime(text)">
            <span class="relative-time">{{ getRelativeTime(text) }}</span>
          </a-tooltip>
        </div>
      </template>

      <!-- 操作列 -->
      <template #action="{ record }">
        <div class="action-column">
          <a-tooltip title="查看 YAML">
            <a-button type="primary" ghost shape="circle" @click="viewDeploymentYaml(record)">
              <template #icon><CodeOutlined /></template>
            </a-button>
          </a-tooltip>
          
          <a-tooltip title="重启 Deployment">
            <a-popconfirm
              title="确定要重启该 Deployment 吗?"
              @confirm="handleRestart(record)"
              ok-text="确定"
              cancel-text="取消"
            >
              <a-button type="primary" ghost shape="circle">
                <template #icon><SyncOutlined /></template>
              </a-button>
            </a-popconfirm>
          </a-tooltip>
          
          <a-tooltip title="删除 Deployment">
            <a-popconfirm
              title="确定要删除该 Deployment 吗?"
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
        <a-empty v-if="filteredDeployments.length === 0" description="暂无部署数据" />
        <div v-else class="deployment-cards">
          <a-checkbox-group v-model:value="selectedCardIds" class="card-checkbox-group">
            <div v-for="deployment in filteredDeployments" :key="deployment.metadata.uid" class="deployment-card">
              <div class="card-header">
                <a-checkbox :value="deployment.metadata.uid" class="card-checkbox" />
                <div class="deployment-title">
                  <RocketOutlined class="deployment-icon" />
                  <h3>{{ deployment.metadata.name }}</h3>
                </div>
                <a-tag :color="getStatusColor(deployment)" class="card-status-tag">
                  {{ deployment.status?.availableReplicas || 0 }}/{{ deployment.status?.replicas || 0 }}
                </a-tag>
              </div>
              
              <div class="card-content">
                <div class="card-detail">
                  <span class="detail-label">命名空间:</span>
                  <span class="detail-value">{{ deployment.metadata.namespace }}</span>
                </div>
                <div class="card-detail">
                  <span class="detail-label">镜像:</span>
                  <span class="detail-value">
                    <ContainerOutlined />
                    {{ deployment.spec?.template?.spec?.containers?.[0]?.image }}
                  </span>
                </div>
                <div class="card-detail">
                  <span class="detail-label">创建时间:</span>
                  <span class="detail-value">{{ formatDate(deployment.metadata.creationTimestamp) }}</span>
                </div>
                <div class="card-status">
                  <span class="detail-label">状态:</span>
                  <a-progress 
                    :percent="getStatusPercent(deployment)" 
                    :status="getProgressStatus(deployment)"
                    size="small"
                    :stroke-color="getStatusColor(deployment)"
                    class="card-progress"
                  />
                </div>
              </div>
              
              <div class="card-footer">
                <a-button type="primary" ghost size="small" @click="viewDeploymentYaml(deployment)">
                  <template #icon><CodeOutlined /></template>
                  YAML
                </a-button>
                <a-button type="primary" ghost size="small" @click="handleRestart(deployment)">
                  <template #icon><SyncOutlined /></template>
                  重启
                </a-button>
                <a-popconfirm
                  title="确定要删除该 Deployment 吗?"
                  @confirm="handleDelete(deployment)"
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

    <!-- 查看 Deployment YAML 模态框 -->
    <a-modal
      v-model:visible="viewYamlModalVisible"
      title="Deployment YAML 配置"
      width="800px"
      class="yaml-modal"
      :footer="null"
    >
      <a-alert v-if="currentDeployment" class="yaml-info" type="info" show-icon>
        <template #message>
          <span>{{ currentDeployment.metadata.name }} ({{ currentDeployment.metadata.namespace }})</span>
        </template>
        <template #description>
          <div>状态: {{ currentDeployment.status.availableReplicas || 0 }}/{{ currentDeployment.status.replicas || 0 }} | 创建于: {{ formatDate(currentDeployment.metadata.creationTimestamp) }}</div>
        </template>
      </a-alert>
      <div class="yaml-actions">
        <a-button type="primary" size="small" @click="copyYaml">
          <template #icon><CopyOutlined /></template>
          复制
        </a-button>
      </div>
      <pre class="yaml-editor">{{ deploymentYaml }}</pre>
    </a-modal>
  </div>
</template>

<script lang="ts" setup>
import { ref, computed, onMounted, watch } from 'vue';
import { message } from 'ant-design-vue';
import { useRoute } from 'vue-router';
import {
  getDeployListApi,
  getDeployYamlApi,
  deleteDeployApi,
  restartDeployApi,
  getAllClustersApi,
  getNamespacesByClusterIdApi,
} from '#/api';
import { 
  CloudServerOutlined, 
  TableOutlined, 
  AppstoreOutlined, 
  SearchOutlined,
  ReloadOutlined,
  DeleteOutlined,
  CodeOutlined,
  RocketOutlined,
  ClockCircleOutlined,
  CopyOutlined,
  ClusterOutlined,
  PartitionOutlined,
  SyncOutlined,
  ContainerOutlined
} from '@ant-design/icons-vue';

// 类型定义
interface Container {
  name: string;
  image: string;
}

interface Deployment {
  metadata: {
    name: string;
    namespace: string;
    uid: string;
    creationTimestamp: string;
  };
  spec: {
    replicas: number;
    selector: {
      matchLabels: Record<string, string>;
    };
    template: {
      metadata: {
        labels: Record<string, string>;
      };
      spec: {
        containers: Container[];
      };
    };
  };
  status: {
    replicas: number;
    availableReplicas: number;
    updatedReplicas: number;
  };
}

// 状态变量
const route = useRoute();
const loading = ref(false);
const clustersLoading = ref(false);
const namespacesLoading = ref(false);
const deployments = ref<Deployment[]>([]);
const searchText = ref('');
const selectedRows = ref<Deployment[]>([]);
const namespaces = ref<string[]>(['default']);
const selectedNamespace = ref<string>('default');
const viewYamlModalVisible = ref(false);
const deploymentYaml = ref('');
const clusters = ref<Array<{id: number, name: string}>>([]);
const selectedCluster = ref<number>();
const viewMode = ref<'table' | 'card'>('table');
const currentDeployment = ref<Deployment | null>(null);
const selectedCardIds = ref<string[]>([]);

// 根据卡片选择更新 selectedRows
watch(selectedCardIds, (newValue) => {
  selectedRows.value = deployments.value.filter(deployment => 
    newValue.includes(deployment.metadata.uid)
  );
});

// 表格列配置
const columns = [
  {
    title: 'Deployment 名称',
    dataIndex: ['metadata', 'name'],
    key: 'name',
    slots: { customRender: 'name' },
    width: '20%',
    sorter: (a: Deployment, b: Deployment) => a.metadata.name.localeCompare(b.metadata.name),
  },
  {
    title: '命名空间',
    dataIndex: ['metadata', 'namespace'],
    key: 'namespace',
    width: '12%',
    sorter: (a: Deployment, b: Deployment) => a.metadata.namespace.localeCompare(b.metadata.namespace),
  },
  {
    title: '状态',
    key: 'status',
    width: '15%',
    slots: { customRender: 'status' },
  },
  {
    title: '镜像',
    dataIndex: ['spec', 'template', 'spec', 'containers', 0, 'image'],
    key: 'image',
    width: '25%',
    slots: { customRender: 'image' },
  },
  {
    title: '创建时间',
    dataIndex: ['metadata', 'creationTimestamp'],
    key: 'creationTimestamp',
    width: '15%',
    sorter: (a: Deployment, b: Deployment) => new Date(a.metadata.creationTimestamp).getTime() - new Date(b.metadata.creationTimestamp).getTime(),
    slots: { customRender: 'creationTimestamp' },
  },
  {
    title: '操作',
    key: 'action',
    width: '15%',
    fixed: 'right',
    slots: { customRender: 'action' },
  },
];

// 计算属性：过滤后的Deployment列表
const filteredDeployments = computed(() => {
  const searchValue = searchText.value.toLowerCase().trim();
  if (!searchValue) return deployments.value;
  return deployments.value.filter(deploy => 
    deploy.metadata.name.toLowerCase().includes(searchValue) || 
    (deploy.spec.template.spec.containers[0]?.image || '').toLowerCase().includes(searchValue)
  );
});

// 获取状态颜色
const getStatusColor = (deployment: Deployment) => {
  if (!deployment.status.replicas) return 'default';
  const available = deployment.status.availableReplicas || 0;
  const total = deployment.status.replicas || 0;
  
  if (available === 0) return 'error';
  if (available < total) return 'warning';
  return 'success';
};

// 获取进度条百分比
const getStatusPercent = (deployment: Deployment) => {
  if (!deployment.status.replicas) return 0;
  const available = deployment.status.availableReplicas || 0;
  const total = deployment.status.replicas || 0;
  
  return Math.round((available / total) * 100);
};

// 获取进度条状态
const getProgressStatus = (deployment: Deployment) => {
  if (!deployment.status.replicas) return 'normal';
  const available = deployment.status.availableReplicas || 0;
  const total = deployment.status.replicas || 0;
  
  if (available === 0) return 'exception';
  if (available < total) return 'active';
  return 'success';
};

// 格式化日期
const formatDate = (dateString: string) => {
  const date = new Date(dateString);
  return date.toLocaleDateString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  });
};

// 获取相对时间
const getRelativeTime = (dateString: string) => {
  const now = new Date();
  const past = new Date(dateString);
  const diffInSeconds = Math.floor((now.getTime() - past.getTime()) / 1000);
  
  if (diffInSeconds < 60) return `${diffInSeconds}秒前`;
  if (diffInSeconds < 3600) return `${Math.floor(diffInSeconds / 60)}分钟前`;
  if (diffInSeconds < 86400) return `${Math.floor(diffInSeconds / 3600)}小时前`;
  return `${Math.floor(diffInSeconds / 86400)}天前`;
};

// 表格选择配置
const rowSelection = {
  onChange: (selectedRowKeys: string[], selectedRowsData: Deployment[]) => {
    selectedRows.value = selectedRowsData;
    selectedCardIds.value = selectedRowsData.map(row => row.metadata.uid);
  },
  getCheckboxProps: (record: Deployment) => ({
    disabled: false, // 可以根据条件禁用某些行的选择
  }),
};

// 复制YAML
const copyYaml = async () => {
  try {
    await navigator.clipboard.writeText(deploymentYaml.value);
    message.success('YAML 已复制到剪贴板');
  } catch (err) {
    message.error('复制失败，请手动选择并复制');
  }
};

// 获取集群列表
const getClusters = async () => {
  clustersLoading.value = true;
  try {
    const res = await getAllClustersApi();
    clusters.value = res ?? [];
    if (clusters.value.length > 0 && !selectedCluster.value) {
      selectedCluster.value = clusters.value[0]?.id;
      if (selectedCluster.value) {
        await getNamespaces();
        await getDeployments();
      }
    }
  } catch (error: any) {
    message.error(error.message || '获取集群列表失败');
  } finally {
    clustersLoading.value = false;
  }
};

// 获取命名空间列表
const getNamespaces = async () => {
  if (!selectedCluster.value) {
    message.warning('请先选择集群');
    return;
  }

  namespacesLoading.value = true;
  try {
    const res = await getNamespacesByClusterIdApi(selectedCluster.value);
    namespaces.value = res.map((ns: { name: string }) => ns.name);
    if (namespaces.value.length > 0) {
      selectedNamespace.value = namespaces.value[0] || 'default';
    }
  } catch (error: any) {
    message.error(error.message || '获取命名空间列表失败');
    namespaces.value = ['default'];
    selectedNamespace.value = 'default';
  } finally {
    namespacesLoading.value = false;
  }
};

// 获取Deployment列表
const getDeployments = async () => {
  if (!selectedCluster.value || !selectedNamespace.value) {
    message.warning('请先选择集群和命名空间');
    return;
  }
  
  loading.value = true;
  try {
    const res = await getDeployListApi(selectedCluster.value, selectedNamespace.value);
    deployments.value = res || [];
    selectedRows.value = [];
    selectedCardIds.value = [];
  } catch (error: any) {
    message.error(error.message || '获取Deployment列表失败');
  } finally {
    loading.value = false;
  }
};

// 刷新数据
const refreshData = () => {
  getDeployments();
};

// 搜索
const onSearch = () => {
  // 搜索逻辑已经在计算属性中实现，这里可以添加其他触发行为
};

// 查看Deployment YAML
const viewDeploymentYaml = async (deployment: Deployment) => {
  if (!selectedCluster.value) return;
  try {
    currentDeployment.value = deployment;
    const res = await getDeployYamlApi(selectedCluster.value, deployment.metadata.name, deployment.metadata.namespace);
    deploymentYaml.value = typeof res === 'string' ? res : JSON.stringify(res, null, 2);
    viewYamlModalVisible.value = true;
  } catch (error: any) {
    message.error(error.message || '获取Deployment YAML失败');
  }
};

// 删除Deployment
const handleDelete = async (deployment: Deployment) => {
  if (!selectedCluster.value) return;
  
  try {
    await deleteDeployApi(selectedCluster.value, deployment.metadata.namespace, deployment.metadata.name);
    message.success('Deployment删除成功');
    getDeployments();
  } catch (error: any) {
    message.error(error.message || '删除Deployment失败');
  }
};

// 重启Deployment
const handleRestart = async (deployment: Deployment) => {
  if (!selectedCluster.value) return;
  
  try {
    await restartDeployApi(selectedCluster.value, deployment.metadata.namespace, deployment.metadata.name);
    message.success('Deployment重启成功');
    getDeployments();
  } catch (error: any) {
    message.error(error.message || '重启Deployment失败');
  }
};

// 批量删除Deployment
const handleBatchDelete = async () => {
  if (!selectedRows.value.length || !selectedCluster.value) return;
  
  try {
    loading.value = true;
    const promises = selectedRows.value.map(deployment => 
      deleteDeployApi(selectedCluster.value!, deployment.metadata.namespace, deployment.metadata.name)
    );
    
    await Promise.all(promises);
    message.success(`成功删除 ${selectedRows.value.length} 个部署`);
    selectedRows.value = [];
    selectedCardIds.value = [];
    getDeployments();
  } catch (error: any) {
    message.error(error.message || '批量删除失败');
  } finally {
    loading.value = false;
  }
};

// 切换命名空间
const handleNamespaceChange = () => {
  getDeployments();
};

// 切换集群
const handleClusterChange = () => {
  selectedNamespace.value = 'default';
  deployments.value = [];
  getNamespaces();
  getDeployments();
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

.deployment-manager {
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

.cluster-option, .namespace-option {
  display: flex;
  align-items: center;
  gap: 10px;
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

/* Deployment 表格样式 */
.deployments-table {
  background: white;
  border-radius: 8px;
  overflow: hidden;
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.05);
}

.deployments-table :deep(.ant-table-thead > tr > th) {
  background-color: #f5f7fa;
  font-weight: 600;
  padding: 14px 16px;
}

.deployments-table :deep(.ant-table-tbody > tr > td) {
  padding: 12px 16px;
}

.deployment-name {
  display: flex;
  align-items: center;
  gap: 10px;
  font-weight: 500;
}

.status-wrapper {
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: 100%;
}

.status-tag {
  font-weight: 500;
  padding: 2px 8px;
  border-radius: 4px;
  text-align: center;
}

.status-progress {
  width: 100%;
  margin-top: 4px;
}

.image-tag {
  display: flex;
  align-items: center;
  gap: 10px;
}

.image-text {
  font-family: 'Courier New', monospace;
  font-size: 13px;
  color: #595959;
  word-break: break-all;
}

.timestamp {
  display: flex;
  align-items: center;
  gap: 10px;
  color: #595959;
}

.relative-time {
  font-size: 12px;
  color: #8c8c8c;
  margin-left: 4px;
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

/* 卡片容器布局 - 横向排列 */
.card-checkbox-group {
  display: flex;
  flex-wrap: wrap;
  gap: 30px;
  padding: 10px;
}

/* 卡片样式优化 */
.deployment-card {
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

.deployment-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.12);
}

.card-checkbox {
  position: absolute;
  top: 12px;
  right: 12px;
  z-index: 2;
}

.card-header {
  padding: 16px 20px;
  border-bottom: 1px solid #f0f0f0;
  background-color: #fafafa;
  position: relative;
}

.deployment-title {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-right: 45px;
}

.deployment-title h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: #333;
  word-break: break-word;
  line-height: 1.4;
}

.deployment-icon {
  color: var(--primary-color);
  font-size: 20px;
}

.card-status-tag {
  position: absolute;
  top: 12px;
  right: 50px;
  padding: 2px 10px;
}

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
  word-break: break-all;
}

.card-status {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.card-progress {
  width: 100%;
  margin-top: 8px;
}

.card-footer {
  padding: 16px 20px;
  background-color: #f5f7fa;
  border-top: 1px solid #eeeeee;
  display: flex;
  justify-content: space-between;
  gap: 12px;
}

.card-footer .ant-btn {
  flex: 1;
  border-radius: 4px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.card-footer .ant-btn svg {
  margin-right: 4px;
}

/* YAML模态框样式 */
.yaml-modal :deep(.ant-modal-content) {
  border-radius: 8px;
  overflow: hidden;
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
  font-family: 'JetBrains Mono', 'Courier New', monospace;
  font-size: 14px;
  line-height: 1.5;
  padding: 16px;
  background-color: #fafafa;
  border-radius: 4px;
  border: 1px solid #f0f0f0;
  overflow: auto;
  max-height: 500px;
  margin: 0;
}

/* 响应式调整 */
@media (max-width: 1400px) {
  .card-checkbox-group {
    justify-content: space-around;
  }
  
  .deployment-card {
    width: 320px;
  }
}

@media (max-width: 768px) {
  .dashboard-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 16px;
  }
  
  .control-panel {
    flex-direction: column;
    gap: 16px;
  }
  
  .search-filters {
    flex-direction: column;
    width: 100%;
  }
  
  .control-item {
    width: 100%;
    min-width: auto;
  }
  
  .action-buttons {
    width: 100%;
    justify-content: flex-end;
    margin-left: 0;
  }
  
  .card-checkbox-group {
    flex-direction: column;
    align-items: center;
  }
  
  .deployment-card {
    width: 100%;
    max-width: 450px;
  }
  
  .card-footer {
    flex-wrap: wrap;
  }
}
</style>