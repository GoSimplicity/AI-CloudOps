<template>
    <div>
      <!-- 操作工具栏 -->
      <div class="toolbar">
        <div class="search-area">
          <a-input
            v-model="searchText"
            placeholder="请输入节点名称"
            style="width: 200px; margin-right: 16px;"
          />
          <a-button type="primary" @click="handleSearch">搜索</a-button>
        </div>
        <div class="action-buttons">
          <a-button type="primary" @click="handleAddNode">新增节点</a-button>
        </div>
      </div>
  
      <!-- 节点列表 -->
      <a-table :columns="columns" :data-source="filteredData" row-key="key">
        <!-- 操作列 -->
        <template #action="{ record }">
          <a-space>
            <a-button type="link" @click="handleEditNode(record)">编辑</a-button>
            <a-button type="link" danger @click="handleDeleteNode(record)">删除</a-button>
          </a-space>
        </template>
      </a-table>
    </div>
  </template>
  
  <script lang="ts" setup>
  import { reactive, ref } from 'vue';
  import { message } from 'ant-design-vue';
  
  // 节点数据类型
  interface TreeNode {
    key: string;
    title: string;
    level: number;
    isLeaf: boolean;
    desc: string;
    ecsCount: number;
    elbCount: number;
    rdsCount: number;
  }
  
  // 示例节点数据
  const data = reactive<TreeNode[]>([
    {
      key: '1',
      title: '根节点',
      level: 0,
      isLeaf: false,
      desc: '这是根节点',
      ecsCount: 10,
      elbCount: 2,
      rdsCount: 1,
    },
    {
      key: '1-1',
      title: '子节点1',
      level: 1,
      isLeaf: true,
      desc: '这是子节点1',
      ecsCount: 5,
      elbCount: 1,
      rdsCount: 0,
    },
    {
      key: '1-2',
      title: '子节点2',
      level: 1,
      isLeaf: true,
      desc: '这是子节点2',
      ecsCount: 3,
      elbCount: 0,
      rdsCount: 2,
    },
  ]);
  
  // 搜索文本
  const searchText = ref('');
  // 过滤后的数据
  const filteredData = ref<TreeNode[]>(data);
  
  // 表格列配置
  const columns = [
    {
      title: '节点名称',
      dataIndex: 'title',
      key: 'title',
    },
    {
      title: '层级',
      dataIndex: 'level',
      key: 'level',
    },
    {
      title: '描述',
      dataIndex: 'desc',
      key: 'desc',
    },
    {
      title: '是否为叶子节点',
      dataIndex: 'isLeaf',
      key: 'isLeaf',
      customRender: ({ isLeaf }) => (isLeaf ? '是' : '否'),
    },
    {
      title: 'ECS 数量',
      dataIndex: 'ecsCount',
      key: 'ecsCount',
    },
    {
      title: 'ELB 数量',
      dataIndex: 'elbCount',
      key: 'elbCount',
    },
    {
      title: 'RDS 数量',
      dataIndex: 'rdsCount',
      key: 'rdsCount',
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
    filteredData.value = data.filter(item => item.title.toLowerCase().includes(searchValue));
  };
  
  // 处理新增节点
  const handleAddNode = () => {
    // 在这里可以打开对话框，输入新节点的信息
    message.info('点击了新增节点按钮');
  };
  
  // 处理编辑节点
  const handleEditNode = (record: TreeNode) => {
    // 在这里可以打开对话框，编辑节点的信息
    message.info(`编辑节点 "${record.title}"`);
  };
  
  // 处理删除节点
  const handleDeleteNode = (record: TreeNode) => {
    const index = data.findIndex(item => item.key === record.key);
    if (index !== -1) {
      data.splice(index, 1);
      handleSearch(); // 更新过滤后的数据
      message.success(`节点 "${record.title}" 已删除`);
    }
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
  