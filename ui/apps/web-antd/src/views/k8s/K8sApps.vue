<template>
  <div>
    <div class="custom-toolbar">
      <div class="search-filters">
        <!-- 搜索输入框 -->
        <a-input v-model:value="searchText" placeholder="请输入应用名称" style="width: 200px; margin-right: 16px;" />
        <!-- 搜索按钮 -->
        <a-button type="primary" @click="handleSearch">搜索</a-button>
      </div>
      <div>
        <a-button type="primary" @click="handleAdd">新增</a-button>
      </div>
    </div>
    <a-table :columns="columns" :data-source="filteredData">
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'name'">
          <a>
            {{ record.name }}
          </a>
        </template>
        <template v-else-if="column.key === 'tags'">
          <span>
            <a-tag v-for="tag in record.tags" :key="tag"
              :color="tag === 'loser' ? 'volcano' : tag.length > 5 ? 'geekblue' : 'green'">
              {{ tag.toUpperCase() }}
            </a-tag>
          </span>
        </template>
        <template v-else-if="column.key === 'action'">
          <span>
            <a-button type="link" size="small" @click="() => handleEdit(record)">编辑</a-button>
            <a-button type="link" size="small" danger @click="() => handleDelete(record)">删除</a-button>
          </span>
        </template>
      </template>
    </a-table>
  </div>
</template>

<script lang="ts" setup>
import { ref, onMounted } from 'vue';
import { message } from 'ant-design-vue';

interface DataItem {
  name: string;
  description: string;
  cluster: string;
  namespace: string;
  status: string;
  tags: string[];
}

// 搜索文本
const searchText = ref('');
const filteredData = ref<DataItem[]>([]);

const columns = [
  {
    title: '应用名称',
    dataIndex: 'name',
    key: 'name',
  },
  {
    title: '描述',
    dataIndex: 'description',
    key: 'description',
  },
  {
    title: '所属集群',
    dataIndex: 'cluster',
    key: 'cluster',
  },
  {
    title: '命名空间',
    dataIndex: 'namespace',
    key: 'namespace',
  },
  {
    title: '状态',
    key: 'status',
    dataIndex: 'status',
  },
  {
    title: '标签',
    key: 'tags',
    dataIndex: 'tags',
  },
  {
    title: '操作',
    key: 'action',
  }
];

const data: DataItem[] = [
  {
    name: 'app-service-1',
    description: '用户服务应用',
    cluster: 'k8s-cluster-1',
    namespace: 'default',
    status: 'running',
    tags: ['prod', 'service'],
  },
  {
    name: 'app-web-1',
    description: '前端应用',
    cluster: 'k8s-cluster-2',
    namespace: 'web',
    status: 'running',
    tags: ['frontend'],
  },
  {
    name: 'app-job-1',
    description: '定时任务应用',
    cluster: 'k8s-cluster-1',
    namespace: 'job',
    status: 'stopped',
    tags: ['job', 'batch'],
  }
];

// 搜索按钮
const handleSearch = () => {
  if (searchText.value.trim() === '') {
    filteredData.value = data;
  } else {
    filteredData.value = data.filter(item => item.name.includes(searchText.value));
  }
};

// 编辑和删除操作
const handleEdit = (_: DataItem) => {
  message.success('编辑成功');
};

const handleDelete = (_: DataItem) => {
  message.success('删除成功');
};

const handleAdd = () => {
  message.success('新增成功');
};

onMounted(() => {
  filteredData.value = data;
});

</script>

<style scoped>
.custom-toolbar {
  padding: 8px;
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
