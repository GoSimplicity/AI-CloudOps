<template>
  <div>
    <!-- 查询和操作工具栏 -->
    <div class="custom-toolbar">
      <div class="search-filters">
        <a-select
          v-model:value="selectedCluster"
          placeholder="请选择集群"
          style="width: 200px; margin-right: 16px"
          @change="handleClusterChange"
        >
          <a-select-option v-for="cluster in clusters" :key="cluster.id" :value="cluster.id">
            {{ cluster.name }}
          </a-select-option>
        </a-select>
        <a-input v-model:value="searchText" placeholder="请输入ConfigMap名称" style="width: 200px; margin-right: 16px" />
        <a-select
          v-model:value="selectedNamespace"
          placeholder="请选择命名空间"
          style="width: 200px"
          @change="handleNamespaceChange"
        >
          <a-select-option v-for="ns in namespaces" :key="ns" :value="ns">
            {{ ns }}
          </a-select-option>
        </a-select>
      </div>
      <div class="action-buttons">
        <a-button type="primary" danger @click="handleBatchDelete" :disabled="!selectedRows.length">
          批量删除
        </a-button>
      </div>
    </div>

    <!-- ConfigMap列表 -->
    <a-table
      :columns="columns"
      :data-source="filteredConfigMaps"
      :row-selection="rowSelection"
      :loading="loading"
      row-key="name"
    >
      <!-- 操作列 -->
      <template #action="{ record }">
        <a-space>
          <a-button type="primary" ghost size="small" @click="viewConfigMapYaml(record)">
            <template #icon><EyeOutlined /></template>
            查看YAML
          </a-button>
          <a-popconfirm
            title="确定要删除该ConfigMap吗？"
            @confirm="handleDelete(record)"
            ok-text="确定" 
            cancel-text="取消"
          >
            <a-button type="primary" danger ghost size="small">
              <template #icon><DeleteOutlined /></template>
              删除
            </a-button>
          </a-popconfirm>
        </a-space>
      </template>
    </a-table>

    <!-- 查看 ConfigMap YAML 模态框 -->
    <a-modal
      v-model:visible="viewYamlModalVisible"
      title="查看 ConfigMap YAML"
      width="800px"
      :footer="null"
    >
      <pre class="yaml-editor">{{ configMapYaml }}</pre>
    </a-modal>
  </div>
</template>

<script lang="ts" setup>
import { ref, computed, onMounted } from 'vue';
import { message } from 'ant-design-vue';
import {
  getConfigMapListApi,
  getConfigMapYamlApi,
  deleteConfigMapApi,
  getAllClustersApi,
  getNamespacesByClusterIdApi,
} from '#/api';

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
const configMaps = ref<ConfigMap[]>([]);
const searchText = ref('');
const selectedRows = ref<ConfigMap[]>([]);
const namespaces = ref<string[]>(['default']);
const selectedNamespace = ref<string>('default');
const viewYamlModalVisible = ref(false);
const configMapYaml = ref('');
const clusters = ref<Array<{id: number, name: string}>>([]);
const selectedCluster = ref<number>();

// 表格列配置
const columns = [
  {
    title: 'ConfigMap名称',
    dataIndex: ['metadata', 'name'],
    key: 'name',
  },
  {
    title: '命名空间',
    dataIndex: ['metadata', 'namespace'],
    key: 'namespace',
  },
  {
    title: '数据条目数',
    key: 'dataCount',
    customRender: ({ record }: { record: ConfigMap }) => {
      return Object.keys(record.data || {}).length;
    },
  },
  {
    title: '创建时间',
    dataIndex: ['metadata', 'creationTimestamp'],
    key: 'creationTimestamp',
  },
  {
    title: '操作',
    key: 'action',
    slots: { customRender: 'action' },
  },
];

// 计算属性：过滤后的ConfigMap列表
const filteredConfigMaps = computed(() => {
  const searchValue = searchText.value.toLowerCase().trim();
  return configMaps.value.filter(cm => cm.metadata.name.toLowerCase().includes(searchValue));
});

// 表格选择配置
const rowSelection = {
  onChange: (_: string[], selectedRowsData: ConfigMap[]) => {
    selectedRows.value = selectedRowsData;
  },
};

// 获取集群列表
const getClusters = async () => {
  try {
    const res = await getAllClustersApi();
    clusters.value = res || [];
  } catch (error: any) {
    message.error(error.message || '获取集群列表失败');
  }
};

// 获取命名空间列表
const getNamespaces = async () => {
  if (!selectedCluster.value) {
    message.warning('请先选择集群');
    return;
  }

  try {
    const res = await getNamespacesByClusterIdApi(selectedCluster.value);
    namespaces.value = res.map((ns: { name: string }) => ns.name);
    selectedNamespace.value = (namespaces.value[0] || 'default') as string;
  } catch (error: any) {
    message.error(error.message || '获取命名空间列表失败');
    namespaces.value = ['default'];
    selectedNamespace.value = 'default';
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
  } catch (error: any) {
    message.error(error.message || '获取ConfigMap列表失败');
  } finally {
    loading.value = false;
  }
};

// 查看ConfigMap YAML
const viewConfigMapYaml = async (configMap: ConfigMap) => {
  if (!selectedCluster.value) return;
  try {
    const res = await getConfigMapYamlApi(selectedCluster.value, configMap.metadata.name, configMap.metadata.namespace);
    configMapYaml.value = typeof res === 'string' ? res : JSON.stringify(res, null, 2);
    viewYamlModalVisible.value = true;
  } catch (error: any) {
    message.error(error.message || '获取ConfigMap YAML失败');
  }
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
    const promises = selectedRows.value.map(configMap => 
      deleteConfigMapApi(selectedCluster.value!, configMap.metadata.namespace, configMap.metadata.name)
    );
    
    await Promise.all(promises);
    message.success('批量删除成功');
    selectedRows.value = [];
    getConfigMaps();
  } catch (error: any) {
    message.error(error.message || '批量删除失败');
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

<style scoped>
.custom-toolbar {
  display: flex;
  justify-content: space-between;
  margin-bottom: 16px;
}

.search-filters {
  display: flex;
  gap: 16px;
}

.action-buttons {
  display: flex;
  gap: 8px;
}

.yaml-editor {
  font-family: monospace;
  white-space: pre-wrap;
  word-wrap: break-word;
  padding: 12px;
  margin: 0;
  border-radius: 4px;
  max-height: 600px;
  overflow-y: auto;
}
</style>
