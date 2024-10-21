<template>
    <div>
      <!-- 查询和操作 -->
      <div class="custom-toolbar">
        <!-- 查询功能 -->
        <div class="search-filters">
          <!-- 搜索输入框 -->
          <a-input
            v-model="searchText"
            placeholder="请输入告警事件名称"
            style="width: 200px; margin-right: 16px;"
          />
        </div>
        <!-- 操作按钮 -->
        <div class="action-buttons">
          <a-button type="primary" @click="handleBatchSilence">批量屏蔽告警</a-button>
        </div>
      </div>
  
      <!-- 告警事件列表表格 -->
      <a-table :columns="columns" :data-source="filteredData" row-key="key">
        <!-- 标签组列 -->
        <template #labels="{ record }">
          <a-tag v-for="label in record.labels" :key="label">{{ label }}</a-tag>
        </template>
        <!-- 操作列 -->
        <template #action="{ record }">
          <a-space>
            <a-button type="link" @click="handleSilence(record)">屏蔽告警</a-button>
            <a-button type="link" @click="handleClaim(record)">认领告警</a-button>
          </a-space>
        </template>
      </a-table>
    </div>
  </template>
  
  <script lang="ts" setup>
  import { computed, reactive, ref } from 'vue';
  import { message, Modal } from 'ant-design-vue';
  
  // 定义数据类型
  interface AlertEvent {
    key: string; // 唯一标识符，用于区分不同的告警事件
    alertName: string; // 告警名称
    fingerprint: string; // 告警唯一ID
    status: string; // 告警状态，如 "告警中"、"已屏蔽"、"已认领"、"已恢复"
    alertRuleName: string; // 关联的告警规则名称
    sendGroupName: string; // 关联的发送组名称
    eventTimes: number; // 触发次数
    silenceId: string; // AlertManager 返回的静默ID
    renLingUserName: string; // 认领告警的用户名
    labels: string[]; // 标签组，格式为 key=v
    createTime: string; // 创建时间
    firstAlertTime: string; // 第一次告警时间
    lastUpdateTime: string; // 最近更新时间
  }
  
  // 示例数据
  const data = reactive<AlertEvent[]>([
    {
      key: '1',
      alertName: 'CPU 使用率过高',
      fingerprint: 'abcd1234',
      status: '告警中',
      alertRuleName: 'CPU 使用率规则',
      sendGroupName: '默认发送组',
      eventTimes: 3,
      silenceId: 'silence123',
      renLingUserName: '管理员',
      labels: ['job=node', 'instance=server1'],
      createTime: '2023-10-01 10:00:00',
      firstAlertTime: '2023-10-01 09:00:00',
      lastUpdateTime: '2023-10-01 10:00:00',
    },
    // 可添加更多示例数据
  ]);
  
  // 搜索文本
  const searchText = ref('');
  // 过滤后的数据，通过 computed 属性动态计算
  const filteredData = computed(() => {
    const searchValue = searchText.value.trim().toLowerCase();
    return data.filter(item => item.alertName.toLowerCase().includes(searchValue));
  });
  
  // 表格列配置
  const columns = [
    {
      title: '告警名称',
      dataIndex: 'alertName',
      key: 'alertName',
    },
    {
      title: '告警状态',
      dataIndex: 'status',
      key: 'status',
    },
    {
      title: '关联告警规则',
      dataIndex: 'alertRuleName',
      key: 'alertRuleName',
    },
    {
      title: '关联发送组',
      dataIndex: 'sendGroupName',
      key: 'sendGroupName',
    },
    {
      title: '触发次数',
      dataIndex: 'eventTimes',
      key: 'eventTimes',
    },
    {
      title: '静默ID',
      dataIndex: 'silenceId',
      key: 'silenceId',
    },
    {
      title: '认领用户',
      dataIndex: 'renLingUserName',
      key: 'renLingUserName',
    },
    {
      title: '第一次告警时间',
      dataIndex: 'firstAlertTime',
      key: 'firstAlertTime',
    },
    {
      title: '最近更新时间',
      dataIndex: 'lastUpdateTime',
      key: 'lastUpdateTime',
    },
    {
      title: '标签组',
      dataIndex: 'labels',
      key: 'labels',
      slots: { customRender: 'labels' }, // 使用自定义插槽来渲染标签组
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
  
  // 处理批量屏蔽告警
  const handleBatchSilence = () => {
    // 这里可以处理批量屏蔽告警事件
    message.info('点击了批量屏蔽告警按钮');
  };
  
  // 处理屏蔽告警
  const handleSilence = (record: AlertEvent) => {
    // 这里可以处理屏蔽告警事件
    message.info(`屏蔽告警 "${record.alertName}"`);
  };
  
  // 处理认领告警
  const handleClaim = (record: AlertEvent) => {
    // 这里可以处理认领告警事件
    message.info(`认领告警 "${record.alertName}"`);
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
  