<template>
  <div>
    <div class="search-filters">
          <!-- 搜索输入框 -->
          <a-input
            v-model:value="searchText"
            placeholder="请输入用户名或昵称"
            style="width: 200px; margin-right: 16px;"
          />
          <!-- 搜索按钮 -->
          <a-button type="primary" @click="handleSearch">搜索</a-button>
    </div>
    <a-table :columns="columns" :data-source="filteredData">
      <template #headerCell="{ column }">
        <template v-if="column.key === 'name'">
          <span>
            <smile-outlined />
            Name
          </span>
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
            <a-tag
              v-for="tag in record.tags"
              :key="tag"
              :color="tag === 'loser' ? 'volcano' : tag.length > 5 ? 'geekblue' : 'green'"
            >
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
import { SmileOutlined } from '@ant-design/icons-vue';
import { ref, onMounted } from 'vue';
import { message } from 'ant-design-vue';
interface DataItem {
  name: string;
  age: number;
  address: string;
  tags: string[];
}

// 搜索文本
const searchText = ref('');
const filteredData = ref<DataItem[]>([]);

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

const columns = [
  {
    title: 'Name',
    dataIndex: 'name',
    key: 'name',
  },
  {
    title: 'Age',
    dataIndex: 'age',
    key: 'age',
  },
  {
    title: 'Address',
    dataIndex: 'address',
    key: 'address',
  },
  {
    title: 'Tags',
    key: 'tags',
    dataIndex: 'tags',
  },
  {
    title: 'Action',
    key: 'action',
  },
];

const data: DataItem[] = [
  {
    name: 'John Brown',
    age: 32,
    address: 'New York No. 1 Lake Park',
    tags: ['nice', 'developer'],
  },
  {
    name: 'Jim Green',
    age: 42,
    address: 'London No. 1 Lake Park',
    tags: ['loser'],
  },
  {
    name: 'Joe Black',
    age: 32,
    address: 'Sidney No. 1 Lake Park',
    tags: ['cool', 'teacher'],
  },
];

onMounted(() => {
  filteredData.value = data;
});

</script>


<style scoped>  
.search-filters {
  padding: 12px; /* 搜索框和按钮之间的间距 */
  display: flex;
  align-items: center;
}
</style>
