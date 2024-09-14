<template>
  <div>
    <!-- 查询和操作 -->
    <div class="custom-toolbar">
      <!-- 查询功能 -->
      <div class="search-filters">
        <!-- 搜索输入框 -->
        <a-input
          v-model="searchText"
          placeholder="请输入用户名或昵称"
          style="width: 200px; margin-right: 16px;"
        />
        <!-- 搜索按钮 -->
        <a-button type="primary" @click="handleSearch">搜索</a-button>
      </div>
      <!-- 操作按钮 -->
      <div class="action-buttons">
        <a-button type="primary" @click="handleAdd">新增账号</a-button>
      </div>
    </div>

    <!-- 用户列表表格 -->
    <a-table :columns="columns" :data-source="filteredData" row-key="userId">
      <!-- 角色列表列 -->
      <template #roles="{ record }">
        <a-tag v-for="role in record.roles" :key="role">{{ role }}</a-tag>
      </template>

      <!-- 操作列 -->
      <template #action="{ record }">
        <a-space>
          <a-button type="link" @click="handleEdit(record)">编辑用户</a-button>
          <a-button type="link" danger @click="handleDelete(record)">删除用户</a-button>
        </a-space>
      </template>
    </a-table>
  </div>
</template>

<script lang="ts" setup>
import { reactive, ref } from 'vue';
import { message } from 'ant-design-vue';

// 定义数据类型
interface User {
  userId: number;
  username: string;
  nickname: string;
  createTime: string;
  roles: string[];
  remark: string;
}

// 示例数据
const data = reactive<User[]>([
  {
    userId: 1,
    username: 'admin',
    nickname: '超级管理员',
    createTime: '2023-10-01 10:00:00',
    roles: ['管理员', '编辑'],
    remark: '系统超级管理员',
  },
  {
    userId: 2,
    username: 'user1',
    nickname: '普通用户1',
    createTime: '2023-10-05 14:30:00',
    roles: ['用户'],
    remark: '普通用户',
  },
  // 可添加更多示例数据
]);

// 搜索文本
const searchText = ref('');
// 过滤后的数据
const filteredData = ref<User[]>(data);

// 处理搜索
const handleSearch = () => {
  const searchValue = searchText.value.trim().toLowerCase();
  filteredData.value = data.filter(item => {
    return (
      item.username.toLowerCase().includes(searchValue) ||
      item.nickname.toLowerCase().includes(searchValue)
    );
  });
};

// 表格列配置
const columns = [
  {
    title: '用户ID',
    dataIndex: 'userId',
    key: 'userId',
  },
  {
    title: '用户名',
    dataIndex: 'username',
    key: 'username',
  },
  {
    title: '昵称',
    dataIndex: 'nickname',
    key: 'nickname',
  },
  {
    title: '创建时间',
    dataIndex: 'createTime',
    key: 'createTime',
  },
  {
    title: '角色列表',
    dataIndex: 'roles',
    key: 'roles',
    slots: { customRender: 'roles' },
  },
  {
    title: '备注',
    dataIndex: 'remark',
    key: 'remark',
  },
  {
    title: '操作',
    key: 'action',
    slots: { customRender: 'action' },
  },
];

// 处理新增账号
const handleAdd = () => {
  // 这里可以打开一个对话框，填写新用户的信息
  message.info('点击了新增账号按钮');
};

// 处理编辑用户
const handleEdit = (record: User) => {
  // 这里可以打开一个对话框，编辑用户的信息
  message.info(`编辑用户 "${record.username}"`);
};

// 处理删除用户
const handleDelete = (record: User) => {
  // 这里可以添加删除逻辑
  const index = data.findIndex(item => item.userId === record.userId);
  if (index !== -1) {
    data.splice(index, 1);
    handleSearch(); // 更新过滤后的数据
    message.success(`用户 "${record.username}" 已删除`);
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
