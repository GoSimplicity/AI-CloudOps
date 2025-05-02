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
        <a-input v-model:value="searchText" placeholder="请输入Deployment名称" style="width: 200px; margin-right: 16px" />
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

    <!-- Deployment列表 -->
    <a-table
      :columns="columns"
      :data-source="filteredDeployments"
      :row-selection="rowSelection"
      :loading="loading"
      row-key="name"
    >
      <!-- 状态列 -->
      <template #status="{ record }">
        <a-tag :color="record.status.availableReplicas === record.status.replicas ? 'success' : 'warning'">
          {{ record.status.availableReplicas || 0 }}/{{ record.status.replicas || 0 }}
        </a-tag>
      </template>

      <!-- 操作列 -->
      <template #action="{ record }">
        <a-space>
          <a-button type="primary" ghost size="small" @click="viewDeploymentYaml(record)">
            <template #icon><EyeOutlined /></template>
            查看YAML
          </a-button>
          <a-popconfirm
            title="确定要重启该Deployment吗？"
            @confirm="handleRestart(record)"
            ok-text="确定"
            cancel-text="取消"
          >
            <a-button type="primary" ghost size="small">
              <template #icon><ReloadOutlined /></template>
              重启
            </a-button>
          </a-popconfirm>
          <a-popconfirm
            title="确定要删除该Deployment吗？"
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

    <!-- 查看 Deployment YAML 模态框 -->
    <a-modal
      v-model:visible="viewYamlModalVisible"
      title="查看 Deployment YAML"
      width="800px"
      :footer="null"
    >
      <pre class="yaml-editor">{{ deploymentYaml }}</pre>
    </a-modal>
  </div>
</template>

<script lang="ts" setup>
import { ref, computed, onMounted } from 'vue';
import { message } from 'ant-design-vue';
import {
  getDeployListApi,
  getDeployYamlApi,
  deleteDeployApi,
  restartDeployApi,
  getAllClustersApi,
  getNamespacesByClusterIdApi,
} from '#/api';

// 类型定义
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
        containers: Array<{
          name: string;
          image: string;
        }>;
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
const loading = ref(false);
const deployments = ref<Deployment[]>([]);
const searchText = ref('');
const selectedRows = ref<Deployment[]>([]);
const namespaces = ref<string[]>(['default']);
const selectedNamespace = ref<string>('default');
const viewYamlModalVisible = ref(false);
const deploymentYaml = ref('');
const clusters = ref<Array<{id: number, name: string}>>([]);
const selectedCluster = ref<number>();

// 表格列配置
const columns = [
  {
    title: 'Deployment名称',
    dataIndex: ['metadata', 'name'],
    key: 'name',
  },
  {
    title: '命名空间',
    dataIndex: ['metadata', 'namespace'],
    key: 'namespace',
  },
  {
    title: '状态',
    key: 'status',
    slots: { customRender: 'status' },
  },
  {
    title: '镜像',
    dataIndex: ['spec', 'template', 'spec', 'containers', 0, 'image'],
    key: 'image',
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

// 计算属性：过滤后的Deployment列表
const filteredDeployments = computed(() => {
  const searchValue = searchText.value.toLowerCase().trim();
  return deployments.value.filter(deploy => deploy.metadata.name.toLowerCase().includes(searchValue));
});

// 表格选择配置
const rowSelection = {
  onChange: (selectedRowKeys: string[], selectedRowsData: Deployment[]) => {
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
    if (namespaces.value.length > 0) {
      selectedNamespace.value = namespaces.value[0] || 'default';
    }
  } catch (error: any) {
    message.error(error.message || '获取命名空间列表失败');
    namespaces.value = ['default'];
    selectedNamespace.value = 'default';
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
  } catch (error: any) {
    message.error(error.message || '获取Deployment列表失败');
  } finally {
    loading.value = false;
  }
};

// 查看Deployment YAML
const viewDeploymentYaml = async (deployment: Deployment) => {
  if (!selectedCluster.value) return;
  try {
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
    const promises = selectedRows.value.map(deployment => 
      deleteDeployApi(selectedCluster.value!, deployment.metadata.namespace, deployment.metadata.name)
    );
    
    await Promise.all(promises);
    message.success('批量删除成功');
    selectedRows.value = [];
    getDeployments();
  } catch (error: any) {
    message.error(error.message || '批量删除失败');
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
