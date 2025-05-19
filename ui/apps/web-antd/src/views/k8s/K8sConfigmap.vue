<template>
  <div class="configmap-manager">
    <!-- 仪表板标题 -->
    <div class="dashboard-header">
      <h2 class="dashboard-title">ConfigMap 资源管理器</h2>
      <div class="dashboard-stats">
        <div class="stat-item">
          <div class="stat-value">{{ filteredConfigMaps.length }}</div>
          <div class="stat-label">配置</div>
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
          placeholder="搜索 ConfigMap 名称"
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

    <!-- ConfigMap 卡片/表格切换视图 -->
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
      :data-source="filteredConfigMaps"
      :row-selection="rowSelection"
      :loading="loading"
      row-key="uid"
      :pagination="{ 
        pageSize: 10, 
        showSizeChanger: true, 
        showQuickJumper: true,
        showTotal: (total: number) => `共 ${total} 条数据`
      }"
      class="configmaps-table"
    >
      <!-- ConfigMap名称列 -->
      <template #name="{ text, record }">
        <div class="configmap-name">
          <ProfileOutlined />
          <span>{{ text }}</span>
        </div>
      </template>
      
      <!-- 数据条目列 -->
      <template #dataCount="{ record }">
        <a-badge 
          :count="Object.keys(record.data || {}).length" 
          :number-style="{ backgroundColor: '#1890ff' }"
        />
      </template>

      <!-- 配置项预览列 -->
      <template #dataPreview="{ record }">
        <div class="data-preview">
          <a-tag 
            v-for="(item, index) in Object.entries(record.data || {}).slice(0, 3)" 
            :key="index" 
            color="blue"
            class="data-key-tag"
          >
            {{ item[0] }}
          </a-tag>
          <a-tag v-if="Object.keys(record.data || {}).length > 3" color="default">
            +{{ Object.keys(record.data || {}).length - 3 }}
          </a-tag>
          <span v-if="!record.data || Object.keys(record.data).length === 0" class="empty-data">
            无数据项
          </span>
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
            <a-button type="primary" ghost shape="circle" @click="viewConfigMapYaml(record)">
              <template #icon><CodeOutlined /></template>
            </a-button>
          </a-tooltip>
          
          <a-tooltip title="查看配置详情">
            <a-button type="primary" ghost shape="circle" @click="viewConfigDetail(record)">
              <template #icon><EyeOutlined /></template>
            </a-button>
          </a-tooltip>
          
          <a-tooltip title="删除配置">
            <a-popconfirm
              title="确定要删除该 ConfigMap 吗?"
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
        <a-empty v-if="filteredConfigMaps.length === 0" description="暂无配置数据" />
        <div v-else class="configmap-cards">
          <a-checkbox-group v-model:value="selectedCardIds" class="card-checkbox-group">
            <div v-for="configmap in filteredConfigMaps" :key="configmap.metadata.uid" class="configmap-card">
              <div class="card-header">
                <a-checkbox :value="configmap.metadata.uid" class="card-checkbox" />
                <div class="configmap-title">
                  <ProfileOutlined class="configmap-icon" />
                  <h3>{{ configmap.metadata.name }}</h3>
                </div>
                <a-badge 
                  :count="Object.keys(configmap.data || {}).length" 
                  :number-style="{ backgroundColor: '#1890ff' }"
                  class="data-count-badge"
                />
              </div>
              
              <div class="card-content">
                <div class="card-detail">
                  <span class="detail-label">命名空间:</span>
                  <span class="detail-value">{{ configmap.metadata.namespace }}</span>
                </div>
                <div class="card-detail">
                  <span class="detail-label">创建时间:</span>
                  <span class="detail-value">{{ formatDate(configmap.metadata.creationTimestamp) }}</span>
                </div>
                <div class="card-detail">
                  <span class="detail-label">配置项:</span>
                  <div class="data-keys">
                    <a-tag 
                      v-for="(_, key, index) in configmap.data" 
                      :key="index" 
                      color="blue"
                      class="data-key-tag"
                      v-show="index < 5"
                    >
                      {{ key }}
                    </a-tag>
                    <a-tag v-if="Object.keys(configmap.data || {}).length > 5" color="default">
                      +{{ Object.keys(configmap.data || {}).length - 5 }}
                    </a-tag>
                    <span v-if="!configmap.data || Object.keys(configmap.data).length === 0" class="empty-data">
                      无数据项
                    </span>
                  </div>
                </div>
              </div>
              
              <div class="card-footer">
                <a-button type="primary" ghost size="small" @click="viewConfigMapYaml(configmap)">
                  <template #icon><CodeOutlined /></template>
                  YAML
                </a-button>
                <a-button type="primary" ghost size="small" @click="viewConfigDetail(configmap)">
                  <template #icon><EyeOutlined /></template>
                  查看配置
                </a-button>
                <a-popconfirm
                  title="确定要删除该 ConfigMap 吗?"
                  @confirm="handleDelete(configmap)"
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

    <!-- 查看 ConfigMap YAML 模态框 -->
    <a-modal
      v-model:visible="viewYamlModalVisible"
      title="ConfigMap YAML 配置"
      width="800px"
      class="yaml-modal"
      :footer="null"
    >
      <a-alert v-if="currentConfigMap" class="yaml-info" type="info" show-icon>
        <template #message>
          <span>{{ currentConfigMap.metadata.name }} ({{ currentConfigMap.metadata.namespace }})</span>
        </template>
        <template #description>
          <div>配置项数量: {{ Object.keys(currentConfigMap.data || {}).length }} | 创建于: {{ formatDate(currentConfigMap.metadata.creationTimestamp) }}</div>
        </template>
      </a-alert>
      <div class="yaml-actions">
        <a-button type="primary" size="small" @click="copyYaml">
          <template #icon><CopyOutlined /></template>
          复制
        </a-button>
      </div>
      <pre class="yaml-editor">{{ configMapYaml }}</pre>
    </a-modal>

    <!-- 查看配置详情模态框 -->
    <a-modal
      v-model:visible="configDetailModalVisible"
      title="ConfigMap 配置详情"
      width="800px"
      class="config-detail-modal"
      :footer="null"
    >
      <a-alert v-if="currentConfigMap" class="yaml-info" type="info" show-icon>
        <template #message>
          <span>{{ currentConfigMap.metadata.name }} ({{ currentConfigMap.metadata.namespace }})</span>
        </template>
        <template #description>
          <div>配置项数量: {{ Object.keys(currentConfigMap.data || {}).length }} | 创建于: {{ formatDate(currentConfigMap.metadata.creationTimestamp) }}</div>
        </template>
      </a-alert>
      
      <a-tabs v-if="currentConfigMap && currentConfigMap.data">
        <a-tab-pane 
          v-for="(value, key, index) in currentConfigMap.data" 
          :key="index" 
          :tab="key"
        >
          <div class="config-detail-content">
            <div class="config-actions">
              <a-button type="primary" size="small" @click="copyConfigValue(value)">
                <template #icon><CopyOutlined /></template>
                复制内容
              </a-button>
            </div>
            <pre class="config-value-editor">{{ value }}</pre>
          </div>
        </a-tab-pane>
      </a-tabs>
      
      <a-empty v-else description="无配置数据" />
    </a-modal>
  </div>
</template>

<script lang="ts" setup>
import { ref, computed, onMounted, watch } from 'vue';
import { message } from 'ant-design-vue';
import {
  getConfigMapListApi,
  getConfigMapYamlApi,
  deleteConfigMapApi,
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
  EyeOutlined, 
  CodeOutlined,
  ProfileOutlined,
  ClockCircleOutlined,
  CopyOutlined,
  ClusterOutlined,
  PartitionOutlined
} from '@ant-design/icons-vue';

// 类型定义
interface ConfigMap {
  metadata: {
    name: string;
    namespace: string;
    uid: string;
    creationTimestamp: string;
  };
  data: Record<string, string>;
}

// 状态变量
const loading = ref(false);
const clustersLoading = ref(false);
const namespacesLoading = ref(false);
const configMaps = ref<ConfigMap[]>([]);
const searchText = ref('');
const selectedRows = ref<ConfigMap[]>([]);
const namespaces = ref<string[]>(['default']);
const selectedNamespace = ref<string>('default');
const viewYamlModalVisible = ref(false);
const configDetailModalVisible = ref(false);
const configMapYaml = ref('');
const clusters = ref<Array<{id: number, name: string}>>([]);
const selectedCluster = ref<number>();
const viewMode = ref<'table' | 'card'>('table');
const currentConfigMap = ref<ConfigMap | null>(null);
const selectedCardIds = ref<string[]>([]);

// 根据卡片选择更新 selectedRows
watch(selectedCardIds, (newValue) => {
  selectedRows.value = configMaps.value.filter(configmap => 
    newValue.includes(configmap.metadata.uid)
  );
});

// 表格列配置
const columns = [
  {
    title: 'ConfigMap 名称',
    dataIndex: ['metadata', 'name'],
    key: 'name',
    slots: { customRender: 'name' },
    width: '20%',
    sorter: (a: ConfigMap, b: ConfigMap) => a.metadata.name.localeCompare(b.metadata.name),
  },
  {
    title: '命名空间',
    dataIndex: ['metadata', 'namespace'],
    key: 'namespace',
    width: '15%',
    sorter: (a: ConfigMap, b: ConfigMap) => a.metadata.namespace.localeCompare(b.metadata.namespace),
  },
  {
    title: '配置项数量',
    key: 'dataCount',
    width: '10%',
    slots: { customRender: 'dataCount' },
    sorter: (a: ConfigMap, b: ConfigMap) => 
      Object.keys(a.data || {}).length - Object.keys(b.data || {}).length,
  },
  {
    title: '配置项预览',
    key: 'dataPreview',
    width: '20%',
    slots: { customRender: 'dataPreview' },
  },
  {
    title: '创建时间',
    dataIndex: ['metadata', 'creationTimestamp'],
    key: 'creationTimestamp',
    width: '15%',
    sorter: (a: ConfigMap, b: ConfigMap) => new Date(a.metadata.creationTimestamp).getTime() - new Date(b.metadata.creationTimestamp).getTime(),
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

// 计算属性：过滤后的ConfigMap列表
const filteredConfigMaps = computed(() => {
  const searchValue = searchText.value.toLowerCase().trim();
  if (!searchValue) return configMaps.value;
  return configMaps.value.filter(cm => 
    cm.metadata.name.toLowerCase().includes(searchValue) ||
    cm.metadata.namespace.toLowerCase().includes(searchValue)
  );
});

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
  onChange: (selectedRowKeys: string[], selectedRowsData: ConfigMap[]) => {
    selectedRows.value = selectedRowsData;
    selectedCardIds.value = selectedRowsData.map(row => row.metadata.uid);
  },
};

// 复制YAML
const copyYaml = async () => {
  try {
    await navigator.clipboard.writeText(configMapYaml.value);
    message.success('YAML 已复制到剪贴板');
  } catch (err) {
    message.error('复制失败，请手动选择并复制');
  }
};

// 复制配置值
const copyConfigValue = async (value: string) => {
  try {
    await navigator.clipboard.writeText(value);
    message.success('配置内容已复制到剪贴板');
  } catch (err) {
    message.error('复制失败，请手动选择并复制');
  }
};

// 获取集群列表
const getClusters = async () => {
  clustersLoading.value = true;
  try {
    const res = await getAllClustersApi();
    clusters.value = res || [];
    if (clusters.value.length > 0 && clusters.value[0]?.id) {
      selectedCluster.value = clusters.value[0].id;
      await getNamespaces();
      await getConfigMaps();
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

// 获取ConfigMap列表
const getConfigMaps = async () => {
  if (!selectedCluster.value || !selectedNamespace.value) {
    message.warning('请先选择集群和命名空间');
    return;
  }
  
  loading.value = true;
  try {
    const res = await getConfigMapListApi(selectedCluster.value, selectedNamespace.value);
    configMaps.value = res || [];
    selectedRows.value = [];
    selectedCardIds.value = [];
  } catch (error: any) {
    message.error(error.message || '获取ConfigMap列表失败');
  } finally {
    loading.value = false;
  }
};

// 刷新数据
const refreshData = () => {
  getConfigMaps();
};

// 搜索
const onSearch = () => {
  // 搜索逻辑已经在计算属性中实现，这里可以添加其他触发行为
};

// 查看ConfigMap YAML
const viewConfigMapYaml = async (configMap: ConfigMap) => {
  if (!selectedCluster.value) return;
  try {
    currentConfigMap.value = configMap;
    const res = await getConfigMapYamlApi(selectedCluster.value, configMap.metadata.name, configMap.metadata.namespace);
    configMapYaml.value = typeof res === 'string' ? res : JSON.stringify(res, null, 2);
    viewYamlModalVisible.value = true;
  } catch (error: any) {
    message.error(error.message || '获取ConfigMap YAML失败');
  }
};

// 查看配置详情
const viewConfigDetail = (configMap: ConfigMap) => {
  currentConfigMap.value = configMap;
  configDetailModalVisible.value = true;
};

// 删除ConfigMap
const handleDelete = async (configMap: ConfigMap) => {
  if (!selectedCluster.value) return;
  
  try {
    await deleteConfigMapApi(selectedCluster.value, configMap.metadata.namespace, configMap.metadata.name);
    message.success('ConfigMap删除成功');
    getConfigMaps();
  } catch (error: any) {
    message.error(error.message || '删除ConfigMap失败');
  }
};

// 批量删除ConfigMap
const handleBatchDelete = async () => {
  if (!selectedRows.value.length || !selectedCluster.value) return;
  
  try {
    loading.value = true;
    const promises = selectedRows.value.map(configMap => 
      deleteConfigMapApi(selectedCluster.value!, configMap.metadata.namespace, configMap.metadata.name)
    );
    
    await Promise.all(promises);
    message.success(`成功删除 ${selectedRows.value.length} 个配置`);
    selectedRows.value = [];
    selectedCardIds.value = [];
    getConfigMaps();
  } catch (error: any) {
    message.error(error.message || '批量删除失败');
  } finally {
    loading.value = false;
  }
};

// 切换命名空间
const handleNamespaceChange = () => {
  getConfigMaps();
};

// 切换集群
const handleClusterChange = () => {
  selectedNamespace.value = 'default';
  configMaps.value = [];
  getNamespaces();
  getConfigMaps();
};

// 页面加载时获取数据
onMounted(() => {
  getClusters();
});
</script>

<style>
/* 基础样式继承自 Service 组件 */
/* 仅添加/修改与 ConfigMap 相关的特定样式 */

.configmap-manager {
  background-color: #f0f2f5;
  border-radius: 8px;
  padding: 24px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
}

.configmap-name {
  display: flex;
  align-items: center;
  gap: 10px;
  font-weight: 500;
}

.configmap-icon {
  color: var(--primary-color);
  font-size: 20px;
}

.data-preview {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  max-width: 300px;
}

.data-key-tag {
  margin: 0 !important;
  font-family: 'Courier New', monospace;
  font-size: 12px;
  max-width: 120px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.empty-data {
  color: #999;
  font-style: italic;
  font-size: 12px;
}

.data-count-badge {
  position: absolute;
  top: 12px;
  right: 50px;
}

.configmaps-table {
  background: white;
  border-radius: 8px;
  overflow: hidden;
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.05);
}

/* 卡片视图相关样式 */
.configmap-cards {
  display: flex;
  flex-wrap: wrap;
  gap: 30px;
  padding: 10px;
}

.configmap-card {
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

.configmap-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.12);
}

.configmap-title {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-right: 45px;
}

.data-keys {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 8px;
}

/* 配置详情模态框样式 */
.config-detail-content {
  padding: 10px 0;
}

.config-actions {
  display: flex;
  justify-content: flex-end;
  margin-bottom: 10px;
}

.config-value-editor {
  font-family: 'JetBrains Mono', 'Courier New', monospace;
  font-size: 14px;
  line-height: 1.5;
  padding: 16px;
  background-color: #fafafa;
  border-radius: 4px;
  border: 1px solid #f0f0f0;
  overflow: auto;
  max-height: 400px;
  margin: 0;
  white-space: pre-wrap;
  word-break: break-all;
}

/* 以下样式继承自 Service 组件 */
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

/* 时间戳样式 */
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

/* 操作列样式 */
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

/* 卡片视图样式 */
.card-view {
  background: white;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.05);
}

/* 卡片容器布局 */
.card-checkbox-group {
  display: flex;
  flex-wrap: wrap;
  gap: 30px;
  padding: 10px;
}

/* 卡片样式 */
.card-header {
  padding: 16px 20px;
  border-bottom: 1px solid #f0f0f0;
  background-color: #fafafa;
  position: relative;
}

.card-checkbox {
  position: absolute;
  top: 12px;
  right: 12px;
  z-index: 2;
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
  align-items: baseline;
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

.card-footer {
  padding: 16px 20px;
  background-color: #f5f7fa;
  border-top: 1px solid #eeeeee;
  display: flex;
  justify-content: space-between;
  gap: 16px;
}

.card-footer .ant-btn {
  flex: 1;
  border-radius: 4px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.card-footer .ant-btn svg {
  margin-right: 8px;
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
  
  .configmap-card {
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
  
  .configmap-card {
    width: 100%;
    max-width: 450px;
  }
  
  .card-footer {
    flex-wrap: wrap;
  }
}
</style>