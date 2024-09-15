<template>
  <div>
    <!-- 查询和操作 -->
    <div class="custom-toolbar">
      <!-- 查询功能 -->
      <div class="search-filters">
        <!-- 搜索输入框 -->
        <a-input
          v-model="searchText"
          placeholder="请输入菜单名称"
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
        <a-button type="primary" @click="handleAdd">创建菜单</a-button>
      </div>
    </div>

    <!-- 菜单列表表格 -->
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

      <!-- 图标列 -->
      <template #icon="{ record }">
        <a-icon :type="record.icon" />
      </template>
    </a-table>
  </div>
</template>

<script lang="ts" setup>
import { reactive, ref } from 'vue';
import { message } from 'ant-design-vue';

interface Menu {
  id: number;
  chineseName: string;
  englishName: string;
  icon: string;
  permission: string;
  component: string;
  status: boolean;
  createTime: string;
}

// 示例数据（假数据）
const data = reactive<Menu[]>([
  {
    id: 1,
    chineseName: '首页',
    englishName: 'Home',
    icon: 'home',
    permission: 'menu:home',
    component: 'HomeComponent',
    status: true,
    createTime: '2024-9-14 10:00:00',
  },
  {
    id: 2,
    chineseName: '用户管理',
    englishName: 'User Management',
    icon: 'user',
    permission: 'menu:user',
    component: 'UserComponent',
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
const filteredData = ref<Menu[]>([]);

// 初始时显示所有数据
filteredData.value = data;

// 处理搜索
const handleSearch = () => {
  filteredData.value = data.filter(item => {
    const nameMatch = item.chineseName
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
    title: '中文名称',
    dataIndex: 'chineseName',
    key: 'chineseName',
  },
  {
    title: '英文名称',
    dataIndex: 'englishName',
    key: 'englishName',
  },
  {
    title: '图标',
    dataIndex: 'icon',
    key: 'icon',
    slots: { customRender: 'icon' },
  },
  {
    title: '权限标识',
    dataIndex: 'permission',
    key: 'permission',
  },
  {
    title: '组件',
    dataIndex: 'component',
    key: 'component',
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
const handleStatusChange = (record: Menu) => {
  record.status = !record.status;
  message.success(`菜单 "${record.chineseName}" 的状态已修改`);
};

// 处理创建菜单
const handleAdd = () => {
  // 这里可以打开一个对话框，填写新菜单的信息
  message.info('点击了创建菜单按钮');
};

// 处理编辑菜单
const handleEdit = (record: Menu) => {
  // 这里可以打开一个对话框，编辑菜单的信息
  message.info(`编辑菜单 "${record.chineseName}"`);
};

// 处理删除菜单
const handleDelete = (record: Menu) => {
  const index = data.findIndex(item => item.id === record.id);
  if (index !== -1) {
    data.splice(index, 1);
    // 更新过滤后的数据
    handleSearch();
    message.success(`菜单 "${record.chineseName}" 已删除`);
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

.action-buttons {
  /* 保持原有样式 */
}
</style>
