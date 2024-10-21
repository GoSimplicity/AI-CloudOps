<template>
    <div>
      <!-- 操作工具栏 -->
      <div class="toolbar">
        <div class="search-area">
          <a-input
            v-model="searchText"
            placeholder="请输入资源名称"
            style="width: 200px; margin-right: 16px;"
          />
          <a-button type="primary" @click="handleSearch">搜索</a-button>
        </div>
        <div class="action-buttons">
          <a-button type="primary" @click="handleAddResource">新增资源</a-button>
        </div>
      </div>
  
      <!-- 资源列表 -->
      <a-table :columns="columns" :data-source="filteredData" row-key="id">
        <!-- 操作列 -->
        <template #action="{ record }">
          <a-space>
            <a-button type="link" @click="handleEditResource(record)">编辑</a-button>
            <a-button type="link" danger @click="handleDeleteResource(record)">删除</a-button>
            <a-button type="link" @click="handleBindToNode(record)">绑定到服务树</a-button>
            <a-button type="link" @click="handleUnbindFromNode(record)">解绑服务树</a-button>
          </a-space>
        </template>
      </a-table>
    </div>
  </template>
  
  <script lang="ts" setup>
  import { reactive, ref } from 'vue';
  import { message } from 'ant-design-vue';
  
  // 资源数据类型
  interface Resource {
    id: string;
    name: string;
    type: string;
    status: string;
    description: string;
  }
  
  // 示例资源数据
  const data = reactive<Resource[]>([
    {
      id: '1',
      name: 'ECS 资源 1',
      type: 'ECS',
      status: 'Running',
      description: 'ECS 资源描述 1',
    },
    {
      id: '2',
      name: 'ELB 资源 1',
      type: 'ELB',
      status: 'Stopped',
      description: 'ELB 资源描述 1',
    },
    {
      id: '3',
      name: 'RDS 资源 1',
      type: 'RDS',
      status: 'Running',
      description: 'RDS 资源描述 1',
    },
  ]);
  
  // 搜索文本
  const searchText = ref('');
  // 过滤后的数据
  const filteredData = ref<Resource[]>(data);
  
  // 表格列配置
  const columns = [
    {
      title: '资源名称',
      dataIndex: 'name',
      key: 'name',
    },
    {
      title: '资源类型',
      dataIndex: 'type',
      key: 'type',
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
    },
    {
      title: '操作',
      key: 'action',
      slots: { customRender: 'action' },
    },
  ];
  
  // 处理搜索
  const handleSearch = () => {
    const searchValue = searchText.value.trim().toLowerCase();
    filteredData.value = data.filter(item => item.name.toLowerCase().includes(searchValue));
  };
  
  // 处理新增资源
  const handleAddResource = () => {
    // 在这里可以打开对话框，输入新资源的信息
    message.info('点击了新增资源按钮');
  };
  
  // 处理编辑资源
  const handleEditResource = (record: Resource) => {
    // 在这里可以打开对话框，编辑资源的信息
    message.info(`编辑资源 "${record.name}"`);
  };
  
  // 处理删除资源
  const handleDeleteResource = (record: Resource) => {
    const index = data.findIndex(item => item.id === record.id);
    if (index !== -1) {
      data.splice(index, 1);
      handleSearch(); // 更新过滤后的数据
      message.success(`资源 "${record.name}" 已删除`);
    }
  };
  
  // 处理绑定到服务树
  const handleBindToNode = (record: Resource) => {
    // 这里可以打开对话框选择要绑定的服务树节点
    message.info(`绑定资源 "${record.name}" 到服务树节点`);
  };
  
  // 处理解绑服务树
  const handleUnbindFromNode = (record: Resource) => {
    // 这里可以执行解绑逻辑
    message.info(`解绑资源 "${record.name}" 从服务树节点`);
  };
  </script>
  
  <style scoped>
  .toolbar {
    padding: 8px;
    display: flex;
    justify-content: space-between;
    align-items: center;
  }
  
  .search-area {
    display: flex;
    align-items: center;
  }
  </style>
  