<template>
    <div>
      <!-- 查询和操作 -->
      <div class="custom-toolbar">
        <!-- 查询功能 -->
        <div class="search-filters">
          <!-- 搜索输入框 -->
          <a-input
            v-model="searchText"
            placeholder="请输入AlertManager实例名称"
            style="width: 200px; margin-right: 16px;"
          />
        </div>
        <!-- 操作按钮 -->
        <div class="action-buttons">
          <a-button type="primary" @click="handleAdd">新增AlertManager实例池</a-button>
        </div>
      </div>
  
      <!-- AlertManager 实例池列表表格 -->
      <a-table :columns="columns" :data-source="filteredData" row-key="key">
        <!-- AlertManager实例列 -->
        <template #alertManagerInstances="{ record }">
          <a-tag v-for="instance in record.alertManagerInstances" :key="instance">{{ instance }}</a-tag>
        </template>
        <!-- 分组标签列 -->
        <template #groupBy="{ record }">
          <a-tag v-for="group in record.groupBy" :key="group">{{ group }}</a-tag>
        </template>
        <!-- 告警规则列 -->
        <template #alertRules="{ record }">
          <a-tag v-for="rule in record.alertRules" :key="rule">{{ rule }}</a-tag>
        </template>
        <!-- 操作列 -->
        <template #action="{ record }">
          <a-space>
            <a-button type="link" @click="handleEdit(record)">编辑实例池</a-button>
            <a-button type="link" danger @click="handleDelete(record)">删除实例池</a-button>
          </a-space>
        </template>
      </a-table>
    </div>
  </template>
  
  <script lang="ts" setup>
  import { computed, reactive, ref } from 'vue';
  import { message, Modal } from 'ant-design-vue';
  
  // 定义数据类型
  interface AlertManagerPool {
    key: string; // 唯一标识符，用于区分不同的实例池
    name: string; // AlertManager 实例池名称
    alertManagerInstances: string[]; // 包含的 AlertManager 实例
    alertRules: string[]; // 包含的告警规则
    resolveTimeout: string; // 默认恢复时间
    groupWait: string; // 默认分组第一次等待时间
    groupInterval: string; // 默认分组等待间隔
    repeatInterval: string; // 默认重复发送时间
    groupBy: string[]; // 分组的标签
    receiver: string; // 兜底接收者
    createUserName: string; // 创建该实例池的用户名称
    createTime: string; // 创建时间
  }
  
  // 示例数据
  const data = reactive<AlertManagerPool[]>([
    {
      key: '1',
      name: '默认实例池',
      alertManagerInstances: ['AlertManager实例1', 'AlertManager实例2'],
      alertRules: ['告警规则1', '告警规则2'],
      resolveTimeout: '5m',
      groupWait: '30s',
      groupInterval: '5m',
      repeatInterval: '1h',
      groupBy: ['job', 'instance'],
      receiver: 'default-receiver',
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
      title: '实例池名称',
      dataIndex: 'name',
      key: 'name',
    },
    {
      title: 'AlertManager实例',
      dataIndex: 'alertManagerInstances',
      key: 'alertManagerInstances',
      slots: { customRender: 'alertManagerInstances' }, // 使用自定义插槽来渲染 AlertManager 实例
    },
    {
      title: '默认恢复时间',
      dataIndex: 'resolveTimeout',
      key: 'resolveTimeout',
    },
    {
      title: '默认分组第一次等待时间',
      dataIndex: 'groupWait',
      key: 'groupWait',
    },
    {
      title: '默认分组等待间隔',
      dataIndex: 'groupInterval',
      key: 'groupInterval',
    },
    {
      title: '默认重复发送时间',
      dataIndex: 'repeatInterval',
      key: 'repeatInterval',
    },
    {
      title: '分组标签',
      dataIndex: 'groupBy',
      key: 'groupBy',
      slots: { customRender: 'groupBy' }, // 使用自定义插槽来渲染分组标签
    },
    {
      title: '告警规则',
      dataIndex: 'alertRules',
      key: 'alertRules',
      slots: { customRender: 'alertRules' }, // 使用自定义插槽来渲染告警规则
    },
    {
      title: '兜底接收者',
      dataIndex: 'receiver',
      key: 'receiver',
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
  
  // 处理新增 AlertManager 实例池
  const handleAdd = () => {
    // 这里可以打开一个对话框，填写新实例池的信息
    message.info('点击了新增 AlertManager 实例池按钮');
  };
  
  // 处理编辑实例池
  const handleEdit = (record: AlertManagerPool) => {
    // 这里可以打开一个对话框，编辑实例池的信息
    message.info(`编辑实例池 "${record.name}"`);
  };
  
  // 处理删除实例池
  const handleDelete = (record: AlertManagerPool) => {
    Modal.confirm({
      title: '确认删除',
      content: `您确定要删除实例池 "${record.name}" 吗？`,
      onOk: () => {
        // 查找要删除的数据索引
        const index = data.findIndex(item => item.key === record.key);
        if (index !== -1) {
          // 删除指定索引的数据
          data.splice(index, 1);
          message.success(`实例池 "${record.name}" 已删除`);
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
  