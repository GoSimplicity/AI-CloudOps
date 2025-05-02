<template>
  <div>
    <div class="custom-toolbar">
      <div class="search-filters">
        <!-- 搜索输入框 -->
        <a-input v-model:value="searchText" placeholder="请输入用户名或昵称" style="width: 200px; margin-right: 16px;" />
        <!-- 搜索按钮 -->
        <a-button type="primary" @click="handleSearch">搜索</a-button>
      </div>
      <div>
        <a-button type="primary" @click="handleAdd">新增</a-button>
      </div>
    </div>
    <a-table :columns="columns" :data-source="filteredData">
      <template #headerCell="{ column }">
        <template v-if="column.key === 'name'">
        </template>
      </template>

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
  cluster: string;
  app: string;
  namespace: string;
  status: string;
  image: string;
  replicas: number;
  tags: string[];
}

// 搜索文本
const searchText = ref('');
const filteredData = ref<DataItem[]>([]);

const columns = [
  {
    title: '名称',
    dataIndex: 'name',
    key: 'name',
  },
  {
    title: '所属集群',
    dataIndex: 'cluster',
    key: 'cluster',
  },
  {
    title: '所属应用',
    dataIndex: 'app',
    key: 'app',
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
    title: '镜像',
    key: 'image',
    dataIndex: 'image',
  },
  {
    title: '副本数量',
    key: 'replicas',
    dataIndex: 'replicas',
  },
  {
    title: '操作',
    key: 'action',
    dataIndex: 'action',
  }
];

const data: DataItem[] = [
  {
    name: 'k8s-instance-1',
    cluster: 'k8s-cluster-1',
    app: 'k8s-app-1',
    namespace: 'k8s-namespace-1',
    status: 'running',
    image: 'k8s-image-1',
    replicas: 1,
    tags: ['nice', 'developer'],
  },
  {
    name: 'k8s-instance-2',
    cluster: 'k8s-cluster-2',
    app: 'k8s-app-2',
    namespace: 'k8s-namespace-2',
    status: 'running',
    image: 'k8s-image-2',
    replicas: 2,
    tags: ['loser'],
  },
  {
    name: 'k8s-instance-3',
    cluster: 'k8s-cluster-3',
    app: 'k8s-app-3',
    namespace: 'k8s-namespace-3',
    status: 'running',
    image: 'k8s-image-3',
    replicas: 3,
    tags: ['cool', 'teacher'],
  },
];

// 搜索按钮
const handleSearch = () => {
  if (searchText.value.trim() === '') {
    filteredData.value = data; // 如果搜索框为空，显示所有数据
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
