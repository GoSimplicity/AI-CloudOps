<template>
    <div>
      <!-- 查询和操作 -->
      <div class="custom-toolbar">
        <!-- 查询功能 -->
        <div class="search-filters">
          <!-- 搜索输入框 -->
          <a-input
            v-model="searchText"
            placeholder="请输入告警规则名称"
            style="width: 200px; margin-right: 16px;"
          />
        </div>
        <!-- 操作按钮 -->
        <div class="action-buttons">
          <a-button type="primary" @click="handleAdd">新增告警规则</a-button>
        </div>
      </div>
  
      <!-- 告警规则列表表格 -->
      <a-table :columns="columns" :data-source="filteredData" row-key="key">
        <!-- 标签组列 -->
        <template #labels="{ record }">
          <a-tag v-for="label in record.labels" :key="label">{{ label }}</a-tag>
        </template>
        <!-- 注解列 -->
        <template #annotations="{ record }">
          <a-tag v-for="annotation in record.annotations" :key="annotation">{{ annotation }}</a-tag>
        </template>
        <!-- 操作列 -->
        <template #action="{ record }">
          <a-space>
            <a-button type="link" @click="handleEdit(record)">编辑告警规则</a-button>
            <a-button type="link" danger @click="handleDelete(record)">删除告警规则</a-button>
          </a-space>
        </template>
      </a-table>
    </div>
  </template>
  
  <script lang="ts" setup>
  import { computed, reactive, ref } from 'vue';
  import { message, Modal } from 'ant-design-vue';
  
  // 定义数据类型
  interface AlertRule {
    key: string; // 唯一标识符，用于区分不同的告警规则
    name: string; // 告警规则名称
    poolName: string; // 关联的 Prometheus 实例池名称
    sendGroupName: string; // 关联的发送组名称
    treeNodeId: number; // 绑定的树节点ID
    enable: number; // 是否启用告警规则：1 启用，2 禁用
    expr: string; // 告警规则表达式
    severity: string; // 告警级别，如 critical、warning
    grafanaLink: string; // Grafana 大盘链接
    forTime: string; // 持续时间，达到此时间才触发告警
    labels: string[]; // 标签组，格式为 key=v
    annotations: string[]; // 注解，格式为 key=v
    createUserName: string; // 创建者用户名
    createTime: string; // 创建时间
  }
  
  // 示例数据
  const data = reactive<AlertRule[]>([
    {
      key: '1',
      name: 'CPU 使用率过高',
      poolName: '默认实例池',
      sendGroupName: '默认发送组',
      treeNodeId: 101,
      enable: 1,
      expr: 'node_cpu_seconds_total{mode="idle"} < 20',
      severity: 'critical',
      grafanaLink: 'http://grafana.example.com',
      forTime: '5m',
      labels: ['job=node', 'instance=server1'],
      annotations: ['summary=CPU 使用率高于 80%', 'description=请检查服务器负载'],
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
      title: '告警规则名称',
      dataIndex: 'name',
      key: 'name',
    },
    {
      title: '关联 Prometheus 实例池',
      dataIndex: 'poolName',
      key: 'poolName',
    },
    {
      title: '关联发送组',
      dataIndex: 'sendGroupName',
      key: 'sendGroupName',
    },
    {
      title: '绑定服务树节点',
      dataIndex: 'treeNodeId',
      key: 'treeNodeId',
    },
    {
      title: '是否启用',
      dataIndex: 'enable',
      key: 'enable',
      customRender: ({ text }: { text: number }) => (text === 1 ? '启用' : '禁用'),
    },
    {
      title: '告警级别',
      dataIndex: 'severity',
      key: 'severity',
    },
    {
      title: 'Grafana 大盘链接',
      dataIndex: 'grafanaLink',
      key: 'grafanaLink',
    },
    {
      title: '持续时间',
      dataIndex: 'forTime',
      key: 'forTime',
    },
    {
      title: '标签组',
      dataIndex: 'labels',
      key: 'labels',
      slots: { customRender: 'labels' }, // 使用自定义插槽来渲染标签组
    },
    {
      title: '注解',
      dataIndex: 'annotations',
      key: 'annotations',
      slots: { customRender: 'annotations' }, // 使用自定义插槽来渲染注解
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
  
  // 处理新增告警规则
  const handleAdd = () => {
    // 这里可以打开一个对话框，填写新告警规则的信息
    message.info('点击了新增告警规则按钮');
  };
  
  // 处理编辑告警规则
  const handleEdit = (record: AlertRule) => {
    // 这里可以打开一个对话框，编辑告警规则的信息
    message.info(`编辑告警规则 "${record.name}"`);
  };
  
  // 处理删除告警规则
  const handleDelete = (record: AlertRule) => {
    Modal.confirm({
      title: '确认删除',
      content: `您确定要删除告警规则 "${record.name}" 吗？`,
      onOk: () => {
        // 查找要删除的数据索引
        const index = data.findIndex(item => item.key === record.key);
        if (index !== -1) {
          // 删除指定索引的数据
          data.splice(index, 1);
          message.success(`告警规则 "${record.name}" 已删除`);
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
  