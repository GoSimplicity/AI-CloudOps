<template>
    <div>
      <!-- 查询和操作 -->
      <div class="custom-toolbar">
        <!-- 查询功能 -->
        <div class="search-filters">
          <!-- 搜索输入框 -->
          <a-input
            v-model="searchText"
            placeholder="请输入采集任务名称"
            style="width: 200px; margin-right: 16px;"
          />
        </div>
        <!-- 操作按钮 -->
        <div class="action-buttons">
          <a-button type="primary" @click="handleAdd">新增采集任务</a-button>
        </div>
      </div>
  
      <!-- 采集任务列表表格 -->
      <a-table :columns="columns" :data-source="filteredData" row-key="key">
        <!-- 服务发现类型列 -->
        <template #serviceDiscoveryType="{ record }">
          {{ record.serviceDiscoveryType === 'k8s' ? 'Kubernetes' : 'HTTP' }}
        </template>
        <!-- 关联采集池列 -->
        <template #poolName="{ record }">
          {{ record.poolName }}
        </template>
        <!-- 操作列 -->
        <template #action="{ record }">
          <a-space>
            <a-button type="link" @click="handleEdit(record)">编辑采集任务</a-button>
            <a-button type="link" danger @click="handleDelete(record)">删除采集任务</a-button>
          </a-space>
        </template>
      </a-table>
    </div>
  </template>
  
  <script lang="ts" setup>
  import { computed, reactive, ref } from 'vue';
  import { message, Modal } from 'ant-design-vue';
  
  // 定义数据类型
  interface ScrapeJob {
    key: string; // 唯一标识符，用于区分不同的采集任务
    name: string; // 采集任务名称
    serviceDiscoveryType: string; // 服务发现类型，可能为 "k8s" 或 "http"
    metricsPath: string; // 监控采集的路径
    scheme: string; // 监控采集的协议方案（如 http 或 https）
    scrapeInterval: number; // 采集间隔时间，单位为秒
    scrapeTimeout: number; // 采集超时时间，单位为秒
    poolName: string; // 关联的采集池名称
    createUserName: string; // 创建该采集任务的用户名称
    createTime: string; // 采集任务的创建时间
  }
  
  // 示例数据
  const data = reactive<ScrapeJob[]>([
    {
      key: '1',
      name: '默认采集任务',
      serviceDiscoveryType: 'k8s',
      metricsPath: '/metrics',
      scheme: 'https',
      scrapeInterval: 30,
      scrapeTimeout: 10,
      poolName: '默认采集池',
      createUserName: '管理员',
      createTime: '2023-10-01 10:00:00',
    },
    // 可添加更多示例数据
  ]);
  
  // 搜索文本
  const searchText = ref('');
  // 过滤后的数据，通过 computed 属性动态计算
  const filteredData = computed(() => {
    const searchValue = searchText.value.trim().toLowerCase();
    return data.filter(item => item.name.toLowerCase().includes(searchValue));
  });
  
  // 表格列配置
  const columns = [
    {
      title: '采集任务名称',
      dataIndex: 'name',
      key: 'name',
    },
    {
      title: '服务发现类型',
      dataIndex: 'serviceDiscoveryType',
      key: 'serviceDiscoveryType',
      slots: { customRender: 'serviceDiscoveryType' }, // 使用自定义插槽来渲染服务发现类型
    },
    {
      title: '监控采集路径',
      dataIndex: 'metricsPath',
      key: 'metricsPath',
    },
    {
      title: '协议方案',
      dataIndex: 'scheme',
      key: 'scheme',
    },
    {
      title: '采集间隔（秒）',
      dataIndex: 'scrapeInterval',
      key: 'scrapeInterval',
    },
    {
      title: '采集超时（秒）',
      dataIndex: 'scrapeTimeout',
      key: 'scrapeTimeout',
    },
    {
      title: '关联采集池',
      dataIndex: 'poolName',
      key: 'poolName',
      slots: { customRender: 'poolName' }, // 使用自定义插槽来渲染关联的采集池名称
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
      slots: { customRender: 'action' }, // 使用自定义插槽来渲染操作按钮
    },
  ];
  
  // 处理新增采集任务
  const handleAdd = () => {
    // 这里可以打开一个对话框，填写新采集任务的信息
    message.info('点击了新增采集任务按钮');
  };
  
  // 处理编辑采集任务
  const handleEdit = (record: ScrapeJob) => {
    // 这里可以打开一个对话框，编辑采集任务的信息
    message.info(`编辑采集任务 "${record.name}"`);
  };
  
  // 处理删除采集任务
  const handleDelete = (record: ScrapeJob) => {
    Modal.confirm({
      title: '确认删除',
      content: `您确定要删除采集任务 "${record.name}" 吗？`,
      onOk: () => {
        // 查找要删除的数据索引
        const index = data.findIndex(item => item.key === record.key);
        if (index !== -1) {
          // 删除指定索引的数据
          data.splice(index, 1);
          message.success(`采集任务 "${record.name}" 已删除`);
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
  