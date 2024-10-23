<template>
    <div>
      <!-- 查询和操作 -->
      <div class="custom-toolbar">
        <!-- 查询功能 -->
        <div class="search-filters">
          <!-- 搜索输入框 -->
          <a-input
            v-model="searchText"
            placeholder="请输入采集池名称"
            style="width: 200px; margin-right: 16px;"
          />
        </div>
        <!-- 操作按钮 -->
        <div class="action-buttons">
          <a-button type="primary" @click="handleAdd">新增采集池</a-button>
        </div>
      </div>
  
      <!-- 采集池列表表格 -->
      <a-table :columns="columns" :data-source="filteredData" row-key="key">
        <!-- Prometheus实例列 -->
        <template #prometheusInstances="{ record }">
          <a-tag v-for="instance in record.prometheusInstances" :key="instance">{{ instance }}</a-tag>
        </template>
        <!-- AlertManager实例列 -->
        <template #alertManagerInstances="{ record }">
          <a-tag v-for="instance in record.alertManagerInstances" :key="instance">{{ instance }}</a-tag>
        </template>
        <!-- 操作列 -->
        <template #action="{ record }">
          <a-space>
            <a-button type="link" @click="handleEdit(record)">编辑采集池</a-button>
            <a-button type="link" danger @click="handleDelete(record)">删除采集池</a-button>
          </a-space>
        </template>
      </a-table>
    </div>
  </template>
  
  <script lang="ts" setup>
  import { computed, reactive, ref } from 'vue';
  import { message, Modal } from 'ant-design-vue';
  
  // 定义数据类型
  interface ScrapePool {
    key: string;
    name: string;
    prometheusInstances: string[];
    alertManagerInstances: string[];
    externalLabels: string;
    createUserName: string;
    createTime: string;
  }
  
  // 示例数据
  const data = reactive<ScrapePool[]>([
    {
      key: '1',
      name: '默认采集池',
      prometheusInstances: ['Prometheus实例1', 'Prometheus实例2'],
      alertManagerInstances: ['AlertManager实例1'],
      externalLabels: 'scrape_ip=1.1.1.1',
      createUserName: '管理员',
      createTime: '2023-10-01 10:00:00',
    },
    // 可添加更多示例数据
  ]);
  
  // 搜索文本
  const searchText = ref('');
  // 过滤后的数据
  const filteredData = computed(() => {
    const searchValue = searchText.value.trim().toLowerCase();
    return data.filter(item => item.name.toLowerCase().includes(searchValue));
  });
  
  // 表格列配置
  const columns = [
    {
      title: '采集池名称',
      dataIndex: 'name',
      key: 'name',
    },
    {
      title: 'Prometheus实例',
      dataIndex: 'prometheusInstances',
      key: 'prometheusInstances',
      slots: { customRender: 'prometheusInstances' },
    },
    {
      title: 'AlertManager实例',
      dataIndex: 'alertManagerInstances',
      key: 'alertManagerInstances',
      slots: { customRender: 'alertManagerInstances' },
    },
    {
      title: '采集池IP标签',
      dataIndex: 'externalLabels',
      key: 'externalLabels',
    },
    {
      title: '创建者',
      dataIndex: 'createUserName',
      key: 'createUserName',
    },
    {
      title: '创建时间',
      dataIndex: 'createTime',
      key: 'createTime',
    },
    {
      title: '操作',
      key: 'action',
      slots: { customRender: 'action' },
    },
  ];
  
  // 处理新增采集池
  const handleAdd = () => {
    // 这里可以打开一个对话框，填写新采集池的信息
    message.info('点击了新增采集池按钮');
  };
  
  // 处理编辑采集池
  const handleEdit = (record: ScrapePool) => {
    // 这里可以打开一个对话框，编辑采集池的信息
    message.info(`编辑采集池 "${record.name}"`);
  };
  
  // 处理删除采集池
  const handleDelete = (record: ScrapePool) => {
    Modal.confirm({
      title: '确认删除',
      content: `您确定要删除采集池 "${record.name}" 吗？`,
      onOk: () => {
        const index = data.findIndex(item => item.key === record.key);
        if (index !== -1) {
          data.splice(index, 1);
          message.success(`采集池 "${record.name}" 已删除`);
        }
      },
    });
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
  