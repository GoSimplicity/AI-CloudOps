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
        <a-input v-model:value="searchText" placeholder="请输入Service名称" style="width: 200px; margin-right: 16px" />
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

    <!-- Service列表 -->
    <a-table
      :columns="columns"
      :data-source="filteredServices"
      :row-selection="rowSelection"
      :loading="loading"
      row-key="name"
    >
      <!-- Service类型列 -->
      <template #type="{ text }">
        <a-tag>{{ text }}</a-tag>
      </template>

      <!-- 操作列 -->
      <template #action="{ record }">
        <a-space>
          <a-button type="primary" ghost size="small" @click="viewServiceYaml(record)">
            <template #icon><EyeOutlined /></template>
            查看YAML
          </a-button>
          <a-popconfirm
            title="确定要删除该Service吗？"
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

    <!-- 查看 Service YAML 模态框 -->
    <a-modal
      v-model:visible="viewYamlModalVisible"
      title="查看 Service YAML"
      width="800px"
      :footer="null"
    >
      <pre class="yaml-editor">{{ serviceYaml }}</pre>
    </a-modal>
  </div>
</template>

<script lang="ts" setup>
import { ref, computed, onMounted } from 'vue';
import { message } from 'ant-design-vue';
import { useRoute } from 'vue-router';
import {
  getServiceListApi,
  getServiceYamlApi,
  deleteServiceApi,
  getAllClustersApi,
  getNamespacesByClusterIdApi,
} from '#/api';

// 类型定义
interface ServicePort {
  name: string;
  protocol: string;
  port: number;
  targetPort: number;
}

interface Service {
  metadata: {
    name: string;
    namespace: string;
    uid: string;
    creationTimestamp: string;
  };
  spec: {
    type: string;
    clusterIP: string;
    ports: ServicePort[];
    selector: Record<string, string>;
  };
  status: {
    loadBalancer: Record<string, any>;
  };
}

// 状态变量
const route = useRoute();
const loading = ref(false);
const services = ref<Service[]>([]);
const searchText = ref('');
const selectedRows = ref<Service[]>([]);
const namespaces = ref<string[]>(['default']);
const selectedNamespace = ref<string>('default');
const viewYamlModalVisible = ref(false);
const serviceYaml = ref('');
const clusters = ref<Array<{id: number, name: string}>>([]);
const selectedCluster = ref<number>();

// 表格列配置
const columns = [
  {
    title: 'Service名称',
    dataIndex: ['metadata', 'name'],
    key: 'name',
  },
  {
    title: '命名空间',
    dataIndex: ['metadata', 'namespace'],
    key: 'namespace',
  },
  {
    title: '类型',
    dataIndex: ['spec', 'type'],
    key: 'type',
    slots: { customRender: 'type' },
  },
  {
    title: 'Cluster IP',
    dataIndex: ['spec', 'clusterIP'],
    key: 'clusterIP',
  },
  {
    title: 'Ports',
    dataIndex: ['spec', 'ports'],
    key: 'ports',
    customRender: ({ text }: { text: ServicePort[] }) => {
      return text.map(port => `${port.port}:${port.targetPort}/${port.protocol}`).join(', ');
    }
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

// 计算属性：过滤后的Service列表
const filteredServices = computed(() => {
  const searchValue = searchText.value.toLowerCase().trim();
  return services.value.filter(svc => svc.metadata.name.toLowerCase().includes(searchValue));
});

// 表格选择配置
const rowSelection = {
  onChange: (selectedRowKeys: string[], selectedRowsData: Service[]) => {
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

// 获取Service列表
const getServices = async () => {
  if (!selectedCluster.value || !selectedNamespace.value) {
    message.warning('请先选择集群和命名空间');
    return;
  }
  
  try {
    const res = await getServiceListApi(selectedCluster.value, selectedNamespace.value);
    services.value = res || [];
  } catch (error: any) {
    message.error(error.message || '获取Service列表失败');
  } finally {
    loading.value = false;
  }
};

// 查看Service YAML
const viewServiceYaml = async (service: Service) => {
  if (!selectedCluster.value) return;
  try {
    const res = await getServiceYamlApi(selectedCluster.value, service.metadata.name, service.metadata.namespace);
    serviceYaml.value = typeof res === 'string' ? res : JSON.stringify(res, null, 2);
    viewYamlModalVisible.value = true;
  } catch (error: any) {
    message.error(error.message || '获取Service YAML失败');
  }
};

// 删除Service
const handleDelete = async (service: Service) => {
  if (!selectedCluster.value) return;
  
  try {
    await deleteServiceApi(selectedCluster.value, service.metadata.namespace, service.metadata.name);
    message.success('Service删除成功');
    getServices();
  } catch (error: any) {
    message.error(error.message || '删除Service失败');
  }
};

// 批量删除Service
const handleBatchDelete = async () => {
  if (!selectedRows.value.length || !selectedCluster.value) return;
  
  try {
    const promises = selectedRows.value.map(service => 
      deleteServiceApi(selectedCluster.value!, service.metadata.namespace, service.metadata.name)
    );
    
    await Promise.all(promises);
    message.success('批量删除成功');
    selectedRows.value = [];
    getServices();
  } catch (error: any) {
    message.error(error.message || '批量删除失败');
  }
};

// 切换命名空间
const handleNamespaceChange = () => {
  getServices();
};

// 切换集群
const handleClusterChange = () => {
  selectedNamespace.value = 'default';
  services.value = [];
  getNamespaces();
  getServices();
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
