<template>
  <div>
    <!-- 查询和操作 -->
    <div class="custom-toolbar">
      <!-- 查询功能 -->
      <div class="search-filters">
        <!-- 搜索输入框 -->
        <a-input
          v-model="searchText"
          placeholder="请输入权限名称"
          style="width: 200px; margin-right: 16px;"
        />
        <!-- 状态过滤 -->
        <a-select
          v-model="selectedStatus"
          placeholder="请选择状态"
          style="width: 150px; margin-right: 16px;"
          allowClear
        >
          <a-select-option value="true">启用</a-select-option>
          <a-select-option value="false">禁用</a-select-option>
        </a-select>
        <!-- 搜索按钮 -->
        <a-button type="primary" @click="handleSearch">搜索</a-button>
      </div>
      <!-- 操作按钮 -->
      <div class="action-buttons">
        <a-button type="primary" @click="handleAdd">新增权限</a-button>
      </div>
    </div>

    <!-- 权限列表表格 -->
    <a-table :columns="columns" :data-source="filteredData" row-key="id">
      <!-- 操作列 -->
      <template #action="{ record }">
        <a-space>
          <a-button type="link" @click="handleEdit(record)">编辑</a-button>
          <a-button type="link" danger @click="handleDelete(record)">删除</a-button>
        </a-space>
      </template>

      <!-- 状态列 -->
      <template #status="{ record }">
        <a-switch
          :checked="record.status"
          @change="handleStatusChange(record)"
        />
      </template>
    </a-table>
  </div>
</template>

<script lang="ts" setup>
import { reactive, ref } from 'vue';
import { message } from 'ant-design-vue';

interface Permission {
  id: number;
  name: string;
  description: string;
  status: boolean;
  createTime: string;
}

const data = reactive<Permission[]>([
  {
    id: 1,
    name: '用户查看',
    description: '允许查看用户列表',
    status: true,
    createTime: '2024-9-14 10:00:00',
  },
  {
    id: 2,
    name: '用户编辑',
    description: '允许编辑用户信息',
    status: false,
    createTime: '2024-9-14 14:30:00',
  },
  // 可添加更多示例数据
]);

// 搜索文本
const searchText = ref('');
// 状态过滤
const selectedStatus = ref<string | null>(null);

// 过滤后的数据
const filteredData = ref<Permission[]>([]);

// 初始时显示所有数据
filteredData.value = data;

// 处理搜索
const handleSearch = () => {
  filteredData.value = data.filter(item => {
    const nameMatch = item.name
      .toLowerCase()
      .includes(searchText.value.trim().toLowerCase());
    const statusMatch =
      selectedStatus.value === null ||
      item.status.toString() === selectedStatus.value;
    return nameMatch && statusMatch;
  });
};

// 表格列配置
const columns = [
  {
    title: 'ID',
    dataIndex: 'id',
    key: 'id',
  },
  {
    title: '权限名称',
    dataIndex: 'name',
    key: 'name',
  },
  {
    title: '描述',
    dataIndex: 'description',
    key: 'description',
  },
  {
    title: '创建时间',
    dataIndex: 'createTime',
    key: 'createTime',
  },
  {
    title: '状态',
    dataIndex: 'status',
    key: 'status',
    slots: { customRender: 'status' },
  },
  {
    title: '操作',
    key: 'action',
    slots: { customRender: 'action' },
  },
];

// 处理修改状态
const handleStatusChange = (record: Permission) => {
  record.status = !record.status;
  message.success(`权限 "${record.name}" 的状态已修改`);
};

// 处理新增权限
const handleAdd = () => {
  // 这里可以打开一个对话框，填写新权限的信息
  message.info('点击了新增权限按钮');
};

// 处理编辑权限
const handleEdit = (record: Permission) => {
  // 这里可以打开一个对话框，编辑权限的信息
  message.info(`编辑权限 "${record.name}"`);
};

// 处理删除权限
const handleDelete = (record: Permission) => {
  const index = data.findIndex(item => item.id === record.id);
  if (index !== -1) {
    data.splice(index, 1);
    // 更新过滤后的数据
    handleSearch();
    message.success(`权限 "${record.name}" 已删除`);
  }
};
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
</style>
