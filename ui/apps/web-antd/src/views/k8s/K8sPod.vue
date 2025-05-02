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
        <a-input v-model:value="searchText" placeholder="请输入Pod名称" style="width: 200px; margin-right: 16px" />
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

    <!-- Pod列表 -->
    <a-table
      :columns="columns"
      :data-source="filteredPods"
      :row-selection="rowSelection"
      :loading="loading"
      row-key="name"
    >
      <!-- Pod状态列 -->
      <template #status="{ text }">
        <a-tag :color="getPodStatusColor(text)">{{ text }}</a-tag>
      </template>

      <!-- 操作列 -->
      <template #action="{ record }">
        <a-space>
          <a-button type="primary" ghost size="small" @click="viewPodYaml(record)">
            <template #icon><EyeOutlined /></template>
            查看YAML
          </a-button>
          <a-button type="primary" ghost size="small" @click="viewPodLogs(record)">
            <template #icon><EyeOutlined /></template>
            查看日志
          </a-button>
          <a-popconfirm
            title="确定要删除该Pod吗？"
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

    <!-- Pod YAML查看模态框 -->
    <a-modal
      v-model:visible="yamlModalVisible"
      title="Pod YAML"
      width="800px"
      :footer="null"
    >
      <pre>{{ podYaml }}</pre>
    </a-modal>

    <!-- Pod日志查看模态框 -->
    <a-modal
      v-model:visible="logModalVisible" 
      title="Pod日志"
      width="800px"
      :footer="null"
    >
      <div class="log-container">
        <a-select
          v-model:value="selectedContainer"
          style="width: 200px; margin-bottom: 16px"
          placeholder="请选择容器"
          @change="handleContainerChange"
        >
          <a-select-option v-for="container in containers" :key="container" :value="container">
            {{ container }}
          </a-select-option>
        </a-select>
        <a-button type="primary" @click="fetchPodLogs" :disabled="!selectedContainer">
          查看日志
        </a-button>
        <div v-if="podLogs" class="pod-logs">
          <div v-for="(line, index) in podLogs.split('\n')" :key="index" class="log-line">
            {{ line }}
          </div>
        </div>
        <div v-else class="empty-logs">请选择容器并点击查看日志</div>
      </div>
    </a-modal>
  </div>
</template>

<script lang="ts" setup>
import { ref, computed, onMounted } from 'vue';
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

// 表格列配置
const columns = [
  {
    title: 'Pod名称',
    dataIndex: 'name',
    key: 'name',
  },
  {
    title: '命名空间',
    dataIndex: 'namespace',
    key: 'namespace',
  },
  {
    title: '状态',
    dataIndex: 'status',
    key: 'status',
    slots: { customRender: 'status' },
  },
  {
    title: 'IP地址',
    dataIndex: 'ip',
    key: 'ip',
  },
  {
    title: '创建时间',
    dataIndex: 'age',
    key: 'age',
  },
  {
    title: '操作',
    key: 'action',
    slots: { customRender: 'action' },
  },
];

// 计算属性：过滤后的Pod列表
const filteredPods = computed(() => {
  const searchValue = searchText.value.toLowerCase().trim();
  return pods.value.filter(pod => pod.name.toLowerCase().includes(searchValue));
});

// 表格选择配置
const rowSelection = {
  onChange: (selectedRowKeys: string[], selectedRowsData: Pod[]) => {
    selectedRows.value = selectedRowsData;
  },
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

  try {
    const res = await getNamespacesByClusterIdApi(selectedCluster.value);
    if (!res) {
      throw new Error('获取命名空间数据为空');
    }

    // 只获取name字段组成新数组
    namespaces.value = res.map((ns: { name: string }) => ns.name);

    // 如果没有选中的命名空间,默认选择第一个
    if (namespaces.value.length > 0) {
      selectedNamespace.value = selectedNamespace.value || namespaces.value[0];
    }
  } catch (error: any) {
    message.error(error.message || '获取命名空间列表失败');
    namespaces.value = ['default'];
    selectedNamespace.value = 'default';
  }
};

const getClusters = async () => {
  try {
    const res = await getAllClustersApi();
    clusters.value = res || [];
  } catch (error: any) {
    message.error(error.message || '获取集群列表失败');
    clusters.value = [];
  }
};

// 查看Pod YAML
const viewPodYaml = async (pod: Pod) => {
  if (!selectedCluster.value) return;
  try {
    const res = await getPodYamlApi(selectedCluster.value, pod.name, pod.namespace);
    podYaml.value = res;
    yamlModalVisible.value = true;
  } catch (error: any) {
    message.error(error.message || '获取Pod YAML失败');
  }
};

// 查看Pod日志
const viewPodLogs = async (pod: Pod) => {
  if (!selectedCluster.value) return;
  selectedPod.value = pod;
  logModalVisible.value = true;
  podLogs.value = '';
  
  try {
    // 获取容器列表
    const res = await getContainersByPodNameApi(selectedCluster.value, pod.name, pod.namespace);
    if (res) {
      containers.value = res.map((container: { name: string }) => container.name);
    }
  } catch (error: any) {
    message.error(error.message || '获取容器列表失败');
  }
};

// 获取Pod日志
const fetchPodLogs = async () => {
  if (!selectedPod.value || !selectedContainer.value || !selectedCluster.value) return;
  
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
    message.success('Pod删除成功');
    await getPods(); // 删除成功后立即刷新数据
  } catch (error: any) {
    message.error(error.message || '删除Pod失败');
  }
};

// 批量删除Pod
const handleBatchDelete = async () => {
  if (!selectedRows.value.length || !selectedCluster.value) return;
  
  try {
    const promises = selectedRows.value.map(pod => 
      deletePodApi(selectedCluster.value!, pod.name, pod.namespace)
    );
    
    await Promise.all(promises);
    message.success('批量删除成功');
    selectedRows.value = [];
    await getPods(); // 删除成功后立即刷新数据
  } catch (error: any) {
    message.error(error.message || '批量删除失败');
  }
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

.pod-logs {
  max-height: 500px;
  overflow-y: auto;
  background: #1e1e1e;
  padding: 16px;
  border-radius: 4px;
  font-family: 'Courier New', Courier, monospace;
  color: #fff;
}

.log-line {
  padding: 2px 0;
  border-bottom: 1px solid #333;
}

.log-line:hover {
  background: #2a2a2a;
}

.empty-logs {
  text-align: center;
  padding: 20px;
  color: #999;
  border-radius: 4px;
}

pre {
  white-space: pre-wrap;
  word-wrap: break-word;
}

.custom-toolbar {
  padding: 6px;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.search-filters {
  display: flex;
  align-items: center;
}

.action-buttons {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-left: 16px;
}
</style>
